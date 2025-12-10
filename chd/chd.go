// Copyright (c) 2025 Niema Moshiri and The Zaparoo Project.
// SPDX-License-Identifier: GPL-3.0-or-later
//
// This file is part of go-gameid.
//
// go-gameid is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-gameid is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-gameid.  If not, see <https://www.gnu.org/licenses/>.

// Package chd provides parsing for CHD (Compressed Hunks of Data) disc images.
// CHD is MAME's compressed disc image format, widely used by RetroArch and other emulators.
package chd

import (
	"fmt"
	"io"
	"os"
)

// CHD represents a CHD (Compressed Hunks of Data) disc image.
type CHD struct {
	file    *os.File
	header  *Header
	hunkMap *HunkMap
	tracks  []Track
}

// Open opens a CHD file and parses its header and metadata.
func Open(path string) (*CHD, error) {
	file, err := os.Open(path) //nolint:gosec // Path from user input is expected
	if err != nil {
		return nil, fmt.Errorf("open CHD file: %w", err)
	}

	chd := &CHD{file: file}

	if err := chd.init(); err != nil {
		_ = file.Close()
		return nil, err
	}

	return chd, nil
}

// init initializes the CHD by parsing header, hunk map, and metadata.
func (c *CHD) init() error {
	// Parse header
	header, err := parseHeader(c.file)
	if err != nil {
		return fmt.Errorf("parse header: %w", err)
	}
	c.header = header

	// Create hunk map
	hunkMap, err := NewHunkMap(c.file, header)
	if err != nil {
		return fmt.Errorf("create hunk map: %w", err)
	}
	c.hunkMap = hunkMap

	// Parse metadata for track information
	if header.MetaOffset > 0 {
		entries, parseErr := parseMetadata(c.file, header.MetaOffset)
		if parseErr != nil {
			// Metadata parsing failure is not fatal, continue without track info
			c.tracks = nil
			return nil //nolint:nilerr // Intentional: metadata parsing failure is non-fatal
		}

		tracks, trackErr := parseTracks(entries)
		if trackErr != nil {
			// Track parsing failure is not fatal, continue without track info
			c.tracks = nil
			return nil //nolint:nilerr // Intentional: track parsing failure is non-fatal
		}
		c.tracks = tracks
	}

	return nil
}

// Close closes the CHD file.
func (c *CHD) Close() error {
	if c.file != nil {
		if err := c.file.Close(); err != nil {
			return fmt.Errorf("close CHD file: %w", err)
		}
	}
	return nil
}

// Header returns the parsed CHD header.
func (c *CHD) Header() *Header {
	return c.header
}

// Tracks returns the parsed track information.
func (c *CHD) Tracks() []Track {
	return c.tracks
}

// Size returns the total logical size (uncompressed) of the CHD data.
func (c *CHD) Size() int64 {
	return int64(c.header.LogicalBytes) //nolint:gosec // LogicalBytes is bounded by file size
}

// SectorReader returns an io.ReaderAt that provides access to decompressed
// sector data with 2048-byte logical sectors (Mode1/Mode2 data portion only).
// This is suitable for ISO9660 filesystem parsing.
// Note: For multi-track CDs with audio tracks first, use DataTrackSectorReader() instead.
func (c *CHD) SectorReader() io.ReaderAt {
	return &sectorReader{
		chd:        c,
		sectorSize: 2048,
		rawMode:    false,
	}
}

// DataTrackSectorReader returns an io.ReaderAt for the first data track,
// providing 2048-byte logical sectors. This is essential for discs like
// Neo Geo CD that have audio tracks before the data track.
func (c *CHD) DataTrackSectorReader() io.ReaderAt {
	return &sectorReader{
		chd:            c,
		sectorSize:     2048,
		rawMode:        false,
		dataTrackStart: c.firstDataTrackSector(),
	}
}

// DataTrackSize returns the logical size of the first data track in bytes.
// For ISO9660 parsing, this is the size in 2048-byte sectors.
func (c *CHD) DataTrackSize() int64 {
	for _, track := range c.tracks {
		if track.IsDataTrack() {
			return int64(track.Frames) * 2048
		}
	}
	// No data track found, return full size
	return int64(c.header.LogicalBytes) //nolint:gosec // LogicalBytes is bounded by CHD format
}

// firstDataTrackSector returns the sector number where the first data track starts.
// If metadata indicates the data starts at frame 0 but the first hunks contain audio
// (zeros from FLAC fallback), we search for the actual ISO9660 PVD location.
func (c *CHD) firstDataTrackSector() int64 {
	// First, check track metadata
	if start := c.dataTrackStartFromMetadata(); start > 0 {
		return start
	}

	// Metadata says data starts at frame 0, search for PVD to verify
	return c.searchForPVD()
}

// dataTrackStartFromMetadata returns the data track start from track metadata, or 0 if unknown.
func (c *CHD) dataTrackStartFromMetadata() int64 {
	for _, track := range c.tracks {
		if track.IsDataTrack() {
			metaStart := int64(track.StartFrame + track.Pregap)
			if metaStart > 0 {
				return metaStart
			}
			break // Data track found but starts at 0, need to search for PVD
		}
	}
	return 0
}

// searchForPVD searches for an ISO9660 Primary Volume Descriptor in the first hunks.
// Returns the calculated data track start sector, or 0 if not found.
func (c *CHD) searchForPVD() int64 {
	unitBytes := int64(c.header.UnitBytes)
	if unitBytes == 0 {
		unitBytes = 2448
	}
	sectorsPerHunk := int64(c.header.HunkBytes) / unitBytes
	maxHunks := c.calculateMaxHunksToSearch(sectorsPerHunk)

	for hunkIdx := range maxHunks {
		hunkData, err := c.hunkMap.ReadHunk(hunkIdx)
		if err != nil {
			continue
		}
		if sector := c.findPVDInHunk(hunkData, hunkIdx, sectorsPerHunk, unitBytes); sector >= 0 {
			return sector
		}
	}
	return 0
}

// calculateMaxHunksToSearch determines how many hunks to search for PVD.
func (c *CHD) calculateMaxHunksToSearch(sectorsPerHunk int64) uint32 {
	// Check first few hunks (up to ~100 sectors worth)
	maxHunks := uint32(100 / sectorsPerHunk) //nolint:gosec // sectorsPerHunk is small and positive
	if maxHunks < 5 {
		maxHunks = 5
	}
	if maxHunks > c.hunkMap.NumHunks() {
		maxHunks = c.hunkMap.NumHunks()
	}
	return maxHunks
}

// pvdMagic is the ISO9660 Primary Volume Descriptor signature.
var pvdMagic = []byte{0x01, 'C', 'D', '0', '0', '1'}

// findPVDInHunk searches for the PVD signature within a single hunk.
// Returns the data track start sector if found, or -1 if not found.
func (*CHD) findPVDInHunk(hunkData []byte, hunkIdx uint32, sectorsPerHunk, unitBytes int64) int64 {
	for sectorInHunk := range sectorsPerHunk {
		offset := sectorInHunk * unitBytes
		if offset+6 > int64(len(hunkData)) {
			break
		}
		if matchesPVD(hunkData, offset) {
			// Found PVD - sector 16 of the ISO, so data track starts 16 sectors before
			absoluteSector := int64(hunkIdx)*sectorsPerHunk + sectorInHunk
			dataTrackStart := absoluteSector - 16
			if dataTrackStart < 0 {
				dataTrackStart = 0
			}
			return dataTrackStart
		}
	}
	return -1
}

// matchesPVD checks if the data at offset matches the PVD magic bytes.
func matchesPVD(data []byte, offset int64) bool {
	if len(data) <= int(offset)+len(pvdMagic) {
		return false
	}
	for i, b := range pvdMagic {
		if data[offset+int64(i)] != b {
			return false
		}
	}
	return true
}

// RawSectorReader returns an io.ReaderAt that provides access to raw
// 2352-byte sectors. This is useful for reading disc headers that may
// be at the start of raw sector data.
func (c *CHD) RawSectorReader() io.ReaderAt {
	return &sectorReader{
		chd:        c,
		sectorSize: 2352,
		rawMode:    true,
	}
}

// sectorReader implements io.ReaderAt for CHD sector data.
type sectorReader struct {
	chd            *CHD
	sectorSize     int
	rawMode        bool  // If true, read raw 2352-byte sectors; if false, extract 2048-byte data
	dataTrackStart int64 // Sector offset to the first data track (for multi-track CDs)
}

// sectorLocation holds the computed location of a sector within CHD hunks.
type sectorLocation struct {
	hunkIdx        uint32
	sectorInHunk   int64
	offsetInSector int64
}

// rawSectorSize is the size of raw CD sector data (without subchannel).
const rawSectorSize = 2352

// computeSectorLocation calculates which hunk and sector contains the given offset.
func (sr *sectorReader) computeSectorLocation(offset, hunkBytes, unitBytes int64) sectorLocation {
	sectorsPerHunk := hunkBytes / unitBytes

	if sr.rawMode {
		sector := offset / rawSectorSize
		return sectorLocation{
			hunkIdx:        uint32(sector / sectorsPerHunk), //nolint:gosec // Sector index bounded by file size
			sectorInHunk:   sector % sectorsPerHunk,
			offsetInSector: offset % rawSectorSize,
		}
	}

	// ISO mode: offset is in terms of 2048-byte logical sectors
	// Apply data track offset for multi-track CDs
	logicalSector := offset/2048 + sr.dataTrackStart
	return sectorLocation{
		hunkIdx:        uint32(logicalSector / sectorsPerHunk), //nolint:gosec // Sector index bounded by file size
		sectorInHunk:   logicalSector % sectorsPerHunk,
		offsetInSector: offset % 2048,
	}
}

// extractSectorData extracts data from a hunk at the given sector location.
func (sr *sectorReader) extractSectorData(hunkData []byte, loc sectorLocation, unitBytes int64) (start, length int64) {
	sectorOffset := loc.sectorInHunk * unitBytes

	if sr.rawMode {
		return sectorOffset + loc.offsetInSector, rawSectorSize - loc.offsetInSector
	}

	// For CD CHD files, the codec returns data at a consistent offset within each unit.
	// Check if this looks like raw sector data (starts with sync header) or user data.
	dataOffset := int64(0)
	if sectorOffset+12 <= int64(len(hunkData)) {
		// Check for CD sync header pattern: 00 FF FF FF FF FF FF FF FF FF FF 00
		hasSyncHeader := hunkData[sectorOffset] == 0x00 &&
			hunkData[sectorOffset+1] == 0xFF &&
			hunkData[sectorOffset+11] == 0x00

		if hasSyncHeader {
			// Raw sector with sync header - user data at offset 16 (Mode1) or 24 (Mode2)
			dataOffset = 16
			if sectorOffset+15 < int64(len(hunkData)) && hunkData[sectorOffset+15] == 2 {
				dataOffset = 24
			}
		}
		// Otherwise: CD codec returned pre-extracted user data, no offset needed
	}

	return sectorOffset + dataOffset + loc.offsetInSector, 2048 - loc.offsetInSector
}

// clampDataLength bounds the data length to available data and sector limits.
func (sr *sectorReader) clampDataLength(dataStart, dataLen int64, hunkLen int, loc sectorLocation) int64 {
	if dataStart+dataLen > int64(hunkLen) {
		dataLen = int64(hunkLen) - dataStart
	}
	if sr.rawMode && dataLen > rawSectorSize-loc.offsetInSector {
		dataLen = rawSectorSize - loc.offsetInSector
	}
	return dataLen
}

// ReadAt reads sector data at the given offset.
// For ISO9660, this provides virtual 2048-byte sectors extracted from the
// CHD's raw sector storage.
func (sr *sectorReader) ReadAt(dest []byte, off int64) (int, error) {
	if len(dest) == 0 {
		return 0, nil
	}

	hunkBytes := int64(sr.chd.hunkMap.HunkBytes())
	unitBytes := int64(sr.chd.header.UnitBytes)
	if unitBytes == 0 {
		unitBytes = 2448 // Default CD sector + subchannel
	}

	totalRead := 0
	remaining := len(dest)
	currentOff := off

	for remaining > 0 {
		loc := sr.computeSectorLocation(currentOff, hunkBytes, unitBytes)

		hunkData, err := sr.chd.hunkMap.ReadHunk(loc.hunkIdx)
		if err != nil {
			if totalRead > 0 {
				return totalRead, nil
			}
			return 0, fmt.Errorf("read hunk %d: %w", loc.hunkIdx, err)
		}

		dataStart, dataLen := sr.extractSectorData(hunkData, loc, unitBytes)
		if dataStart >= int64(len(hunkData)) {
			break
		}

		dataLen = sr.clampDataLength(dataStart, dataLen, len(hunkData), loc)
		toCopy := min(int(dataLen), remaining)

		copy(dest[totalRead:], hunkData[dataStart:dataStart+int64(toCopy)])
		totalRead += toCopy
		remaining -= toCopy
		currentOff += int64(toCopy)
	}

	if totalRead == 0 {
		return 0, io.EOF
	}

	return totalRead, nil
}

// FirstDataTrackOffset returns the byte offset to the first data track.
// This is useful for reading disc headers for Sega Saturn/CD identification.
func (c *CHD) FirstDataTrackOffset() int64 {
	for _, track := range c.tracks {
		if track.IsDataTrack() {
			// Return offset including pregap
			unitBytes := int64(c.header.UnitBytes)
			if unitBytes == 0 {
				unitBytes = 2448
			}
			return int64(track.StartFrame) * unitBytes
		}
	}
	return 0
}

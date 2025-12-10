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

package chd

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Metadata tag constants (as 4-byte big-endian integers)
const (
	// MetaTagCHT2 is the CD Track v2 metadata tag ("CHT2")
	MetaTagCHT2 = 0x43485432

	// MetaTagCHCD is the CD metadata tag ("CHCD")
	MetaTagCHCD = 0x43484344

	// MetaTagCHTR is the CD Track v1 metadata tag ("CHTR")
	MetaTagCHTR = 0x43485452

	// MetaTagGDTR is the GD-ROM track metadata tag ("CHGD")
	MetaTagGDTR = 0x43484744
)

// Track represents a CD track in the CHD file.
type Track struct {
	Type       string
	SubType    string
	Number     int
	Frames     int
	Pregap     int
	Postgap    int
	DataSize   int
	SubSize    int
	StartFrame int
}

// metadataEntry represents a raw metadata entry from the CHD file.
type metadataEntry struct {
	Data  []byte
	Next  uint64
	Tag   uint32
	Flags uint8
}

// parseMetadata reads all metadata entries from the CHD file.
func parseMetadata(reader io.ReaderAt, offset uint64) ([]metadataEntry, error) {
	entries := make([]metadataEntry, 0, 8) // Pre-allocate for typical CHD track count
	visited := make(map[uint64]bool)       // Track visited offsets to detect loops

	for offset != 0 {
		// Detect circular references
		if visited[offset] {
			return entries, fmt.Errorf("%w: circular metadata chain at offset %d", ErrInvalidMetadata, offset)
		}
		visited[offset] = true

		// Limit total entries to prevent memory exhaustion
		if len(entries) >= MaxMetadataEntries {
			return entries, fmt.Errorf("%w: too many metadata entries (%d)", ErrInvalidMetadata, len(entries))
		}

		entry, err := readMetadataEntry(reader, offset)
		if err != nil {
			return entries, fmt.Errorf("read metadata at %d: %w", offset, err)
		}

		entries = append(entries, entry)
		offset = entry.Next
	}

	return entries, nil
}

// readMetadataEntry reads a single metadata entry at the given offset.
// Metadata entry format:
//
//	Offset 0: Tag (4 bytes, big-endian)
//	Offset 4: Flags (1 byte)
//	Offset 5: Length (3 bytes, big-endian)
//	Offset 8: Next offset (8 bytes, big-endian)
//	Offset 16: Data (length bytes)
func readMetadataEntry(reader io.ReaderAt, offset uint64) (metadataEntry, error) {
	headerBuf := make([]byte, 16)
	//nolint:gosec // Safe: offset from metadata chain, validated by CHD file structure
	if _, err := reader.ReadAt(headerBuf, int64(offset)); err != nil {
		return metadataEntry{}, fmt.Errorf("read metadata header: %w", err)
	}

	entry := metadataEntry{
		Tag:   binary.BigEndian.Uint32(headerBuf[0:4]),
		Flags: headerBuf[4],
	}

	// Length is 3 bytes big-endian (bytes 5-7)
	length := uint32(headerBuf[5])<<16 | uint32(headerBuf[6])<<8 | uint32(headerBuf[7])

	// Next offset (8 bytes)
	entry.Next = binary.BigEndian.Uint64(headerBuf[8:16])

	// Read data
	if length > MaxMetadataLen {
		return metadataEntry{}, fmt.Errorf("%w: metadata entry too large (%d > %d)",
			ErrInvalidMetadata, length, MaxMetadataLen)
	}
	if length > 0 {
		entry.Data = make([]byte, length)
		//nolint:gosec // Safe: offset from metadata chain, validated by CHD file structure
		if _, err := reader.ReadAt(entry.Data, int64(offset)+16); err != nil {
			return metadataEntry{}, fmt.Errorf("read metadata data: %w", err)
		}
	}

	return entry, nil
}

// parseTracks extracts track information from metadata entries.
func parseTracks(entries []metadataEntry) ([]Track, error) {
	var tracks []Track

	for _, entry := range entries {
		switch entry.Tag {
		case MetaTagCHT2:
			track, err := parseCHT2(entry.Data)
			if err != nil {
				return nil, fmt.Errorf("parse CHT2: %w", err)
			}
			tracks = append(tracks, track)

		case MetaTagCHTR:
			track, err := parseCHTR(entry.Data)
			if err != nil {
				return nil, fmt.Errorf("parse CHTR: %w", err)
			}
			tracks = append(tracks, track)

		case MetaTagCHCD:
			// Binary CD metadata - contains all tracks at once
			parsed, err := parseCHCD(entry.Data)
			if err != nil {
				return nil, fmt.Errorf("parse CHCD: %w", err)
			}
			tracks = append(tracks, parsed...)
		}
	}

	// Calculate start frames for each track
	startFrame := 0
	for i := range tracks {
		tracks[i].StartFrame = startFrame
		startFrame += tracks[i].Pregap + tracks[i].Frames + tracks[i].Postgap
	}

	return tracks, nil
}

// parseCHT2 parses CHT2 (CD Track v2) metadata.
// Format: ASCII key:value pairs
// Example: "TRACK:1 TYPE:MODE2_RAW SUBTYPE:NONE FRAMES:1234 PREGAP:150 PGTYPE:MODE2_RAW PGSUB:RW POSTGAP:0"
//
//nolint:gocognit,revive // CHT2 parsing requires handling many metadata fields
func parseCHT2(data []byte) (Track, error) {
	var track Track

	// Trim null bytes and whitespace from the metadata string
	str := strings.TrimRight(string(data), "\x00 \t\r\n")
	str = strings.TrimSpace(str)
	fields := strings.Fields(str)

	for _, field := range fields {
		parts := strings.SplitN(field, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToUpper(parts[0])
		value := parts[1]

		switch key {
		case "TRACK":
			num, err := strconv.Atoi(value)
			if err != nil {
				return track, fmt.Errorf("invalid track number %q: %w", value, err)
			}
			track.Number = num

		case "TYPE":
			track.Type = value
			track.DataSize = trackTypeToDataSize(value)

		case "SUBTYPE":
			track.SubType = value
			track.SubSize = subTypeToSize(value)

		case "FRAMES":
			frames, err := strconv.Atoi(value)
			if err != nil {
				return track, fmt.Errorf("invalid frames %q: %w", value, err)
			}
			track.Frames = frames

		case "PREGAP":
			pregap, err := strconv.Atoi(value)
			if err != nil {
				return track, fmt.Errorf("invalid pregap %q: %w", value, err)
			}
			track.Pregap = pregap

		case "POSTGAP":
			postgap, err := strconv.Atoi(value)
			if err != nil {
				return track, fmt.Errorf("invalid postgap %q: %w", value, err)
			}
			track.Postgap = postgap
		}
	}

	return track, nil
}

// parseCHTR parses CHTR (CD Track v1) metadata.
// Format: ASCII, simpler format than CHT2
// Example: "TRACK:1 TYPE:MODE1 SUBTYPE:NONE FRAMES:1234"
func parseCHTR(data []byte) (Track, error) {
	// V1 format is similar to V2, just with fewer fields
	return parseCHT2(data)
}

// parseCHCD parses CHCD (binary CD metadata).
// Format:
//
//	Offset 0: Number of tracks (4 bytes, big-endian)
//	Offset 4: Track entries (24 bytes each)
//
// Track entry format:
//
//	Offset 0: Type (4 bytes)
//	Offset 4: Subtype (4 bytes)
//	Offset 8: Data size (4 bytes)
//	Offset 12: Sub size (4 bytes)
//	Offset 16: Frames (4 bytes)
//	Offset 20: Pad frames (4 bytes)
func parseCHCD(data []byte) ([]Track, error) {
	if len(data) < 4 {
		return nil, ErrInvalidMetadata
	}

	numTracks := binary.BigEndian.Uint32(data[0:4])
	if numTracks > MaxNumTracks {
		return nil, fmt.Errorf("%w: too many tracks (%d > %d)", ErrInvalidMetadata, numTracks, MaxNumTracks)
	}
	if len(data) < int(4+numTracks*24) {
		return nil, ErrInvalidMetadata
	}

	tracks := make([]Track, numTracks)
	offset := 4

	for i := range numTracks {
		trackType := binary.BigEndian.Uint32(data[offset : offset+4])
		subType := binary.BigEndian.Uint32(data[offset+4 : offset+8])
		dataSize := binary.BigEndian.Uint32(data[offset+8 : offset+12])
		subSize := binary.BigEndian.Uint32(data[offset+12 : offset+16])
		frames := binary.BigEndian.Uint32(data[offset+16 : offset+20])
		// Pad frames at offset+20 is just for alignment

		tracks[i] = Track{
			Number:   int(i + 1),
			Type:     cdTypeToString(trackType),
			SubType:  cdSubTypeToString(subType),
			DataSize: int(dataSize),
			SubSize:  int(subSize),
			Frames:   int(frames),
		}

		offset += 24
	}

	return tracks, nil
}

// trackTypeToDataSize returns the data size for a track type string.
func trackTypeToDataSize(trackType string) int {
	switch strings.ToUpper(trackType) {
	case "MODE1/2048", "MODE2_FORM1":
		return 2048
	case "MODE1/2352", "MODE1_RAW":
		return 2352
	case "MODE2/2336", "MODE2_FORM_MIX":
		return 2336
	case "MODE2/2048":
		return 2048
	case "MODE2/2352", "MODE2_RAW":
		return 2352
	case "AUDIO":
		return 2352
	default:
		return 2352 // Default to raw
	}
}

// subTypeToSize returns the subchannel size for a subtype string.
func subTypeToSize(subType string) int {
	switch strings.ToUpper(subType) {
	case "NONE":
		return 0
	case "RW", "RW_RAW":
		return 96
	default:
		return 0
	}
}

// cdTypeToString converts a binary CD type to a string.
func cdTypeToString(cdType uint32) string {
	switch cdType {
	case 0:
		return "MODE1/2048"
	case 1:
		return "MODE1/2352"
	case 2:
		return "MODE2/2048"
	case 3:
		return "MODE2/2336"
	case 4:
		return "MODE2/2352"
	case 5:
		return "AUDIO"
	default:
		return "UNKNOWN"
	}
}

// cdSubTypeToString converts a binary CD subtype to a string.
func cdSubTypeToString(subType uint32) string {
	switch subType {
	case 0:
		return "RW"
	case 1:
		return "RW_RAW"
	case 2:
		return "NONE"
	default:
		return "NONE"
	}
}

// IsDataTrack returns true if this is a data track (not audio).
func (t *Track) IsDataTrack() bool {
	return !strings.EqualFold(t.Type, "AUDIO")
}

// SectorSize returns the total size of each sector including subchannel data.
func (t *Track) SectorSize() int {
	if t.DataSize == 0 {
		return 2352 + t.SubSize
	}
	return t.DataSize + t.SubSize
}

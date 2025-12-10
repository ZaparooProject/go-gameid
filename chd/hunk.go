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
	"sync"
)

// Hunk compression types (V5 map entry types).
const (
	HunkCompTypeCodec0   = 0  // Compressed with compressor 0
	HunkCompTypeCodec1   = 1  // Compressed with compressor 1
	HunkCompTypeCodec2   = 2  // Compressed with compressor 2
	HunkCompTypeCodec3   = 3  // Compressed with compressor 3
	HunkCompTypeNone     = 4  // Uncompressed
	HunkCompTypeSelf     = 5  // Reference to another hunk in this CHD
	HunkCompTypeParent   = 6  // Reference to parent CHD
	HunkCompTypeRLESmall = 7  // RLE: repeat last compression type (small count)
	HunkCompTypeRLELarge = 8  // RLE: repeat last compression type (large count)
	HunkCompTypeSelf0    = 9  // Self reference to same hunk as last
	HunkCompTypeSelf1    = 10 // Self reference to last+1
	HunkCompTypeParSelf  = 11 // Parent reference to self
	HunkCompTypePar0     = 12 // Parent reference same as last
	HunkCompTypePar1     = 13 // Parent reference last+1
)

// HunkMapEntry represents a single entry in the V5 hunk map.
type HunkMapEntry struct {
	Offset     uint64
	CompLength uint32
	CRC16      uint16
	CompType   uint8
}

// HunkMap manages the hunk map and caching for a CHD file.
type HunkMap struct {
	reader    io.ReaderAt
	header    *Header
	cache     map[uint32][]byte
	entries   []HunkMapEntry
	codecs    []Codec
	cacheSize int
	maxCache  int
	cacheMu   sync.RWMutex
}

// NewHunkMap creates a new hunk map from the CHD header and reader.
func NewHunkMap(reader io.ReaderAt, header *Header) (*HunkMap, error) {
	hm := &HunkMap{
		reader:   reader,
		header:   header,
		cache:    make(map[uint32][]byte),
		maxCache: 16, // Cache up to 16 hunks
	}

	// Initialize codecs for V5
	if header.Version == 5 {
		for _, tag := range header.Compressors {
			if tag == 0 {
				hm.codecs = append(hm.codecs, nil)
				continue
			}
			codec, err := GetCodec(tag)
			if err != nil {
				// Codec not available - continue without it. If a hunk actually
				// needs this codec, decompressWithCodec will return a clear error.
				hm.codecs = append(hm.codecs, nil)
				continue
			}
			hm.codecs = append(hm.codecs, codec)
		}
	}

	// Parse hunk map
	if err := hm.parseMap(); err != nil {
		return nil, fmt.Errorf("parse hunk map: %w", err)
	}

	return hm, nil
}

// parseMap parses the hunk map from the CHD file.
func (hm *HunkMap) parseMap() error {
	numHunks := hm.header.NumHunks()
	if numHunks > MaxNumHunks {
		return fmt.Errorf("%w: too many hunks (%d > %d)", ErrInvalidHeader, numHunks, MaxNumHunks)
	}
	hm.entries = make([]HunkMapEntry, numHunks)

	switch hm.header.Version {
	case 5:
		return hm.parseMapV5()
	case 4, 3:
		return hm.parseMapV4()
	default:
		return fmt.Errorf("%w: version %d", ErrUnsupportedVersion, hm.header.Version)
	}
}

// parseMapV5 parses a V5 compressed hunk map.
// V5 map header (16 bytes):
//
//	Offset 0: Compressed map length (4 bytes)
//	Offset 4: First block offset (6 bytes, 48-bit)
//	Offset 10: CRC16 (2 bytes)
//	Offset 12: Bits for length (1 byte)
//	Offset 13: Bits for self-ref (1 byte)
//	Offset 14: Bits for parent-ref (1 byte)
//	Offset 15: Reserved (1 byte)
//
//nolint:gosec,gocyclo,cyclop,funlen,revive // Safe: MapOffset validated; complexity needed for CHD format
func (hm *HunkMap) parseMapV5() error {
	// Read map header
	mapHeader := make([]byte, 16)
	if _, err := hm.reader.ReadAt(mapHeader, int64(hm.header.MapOffset)); err != nil {
		return fmt.Errorf("read map header: %w", err)
	}

	compMapLen := binary.BigEndian.Uint32(mapHeader[0:4])
	if compMapLen > MaxCompMapLen {
		return fmt.Errorf("%w: compressed map too large (%d > %d)", ErrInvalidHeader, compMapLen, MaxCompMapLen)
	}
	firstOffs := uint64(mapHeader[4])<<40 | uint64(mapHeader[5])<<32 |
		uint64(mapHeader[6])<<24 | uint64(mapHeader[7])<<16 |
		uint64(mapHeader[8])<<8 | uint64(mapHeader[9])
	lengthBits := int(mapHeader[12])
	selfBits := int(mapHeader[13])
	parentBits := int(mapHeader[14])

	// Read compressed map data
	compMap := make([]byte, compMapLen)
	if _, err := hm.reader.ReadAt(compMap, int64(hm.header.MapOffset)+16); err != nil {
		return fmt.Errorf("read compressed map: %w", err)
	}

	// Create bit reader and Huffman decoder
	br := newBitReader(compMap)
	decoder := newHuffmanDecoder(16, 8) // 16 codes, 8-bit max

	if err := decoder.importTreeRLE(br); err != nil {
		return fmt.Errorf("import huffman tree: %w", err)
	}

	// Phase 1: Decode compression types with RLE
	numHunks := hm.header.NumHunks()
	compTypes := make([]uint8, numHunks)
	var lastComp uint8
	var repCount int

	for hunkNum := range numHunks {
		if repCount > 0 {
			compTypes[hunkNum] = lastComp
			repCount--
			continue
		}

		val := decoder.decode(br)
		switch val {
		case HunkCompTypeRLESmall:
			compTypes[hunkNum] = lastComp
			repCount = 2 + int(decoder.decode(br))
		case HunkCompTypeRLELarge:
			compTypes[hunkNum] = lastComp
			repCount = 2 + 16 + (int(decoder.decode(br)) << 4)
			repCount += int(decoder.decode(br))
		default:
			compTypes[hunkNum] = val
			lastComp = val
		}
	}

	// Phase 2: Read offsets/lengths based on compression type
	curOffset := firstOffs
	var lastSelf uint32
	var lastParent uint64

	for hunkNum := range numHunks {
		compType := compTypes[hunkNum]
		var length uint32
		var offset uint64

		switch compType {
		case HunkCompTypeCodec0, HunkCompTypeCodec1, HunkCompTypeCodec2, HunkCompTypeCodec3:
			length = br.read(lengthBits)
			offset = curOffset
			curOffset += uint64(length)
			br.read(16) // CRC16
		case HunkCompTypeNone:
			length = hm.header.HunkBytes
			offset = curOffset
			curOffset += uint64(length)
			br.read(16) // CRC16
		case HunkCompTypeSelf:
			lastSelf = br.read(selfBits)
			offset = uint64(lastSelf)
		case HunkCompTypeParent:
			lastParent = uint64(br.read(parentBits))
			offset = lastParent
		case HunkCompTypeSelf0:
			offset = uint64(lastSelf)
			compType = HunkCompTypeSelf
		case HunkCompTypeSelf1:
			lastSelf++
			offset = uint64(lastSelf)
			compType = HunkCompTypeSelf
		case HunkCompTypeParSelf:
			offset = uint64(hunkNum) * uint64(hm.header.HunkBytes) / uint64(hm.header.UnitBytes)
			lastParent = offset
			compType = HunkCompTypeParent
		case HunkCompTypePar0:
			offset = lastParent
			compType = HunkCompTypeParent
		case HunkCompTypePar1:
			lastParent += uint64(hm.header.HunkBytes) / uint64(hm.header.UnitBytes)
			offset = lastParent
			compType = HunkCompTypeParent
		}

		hm.entries[hunkNum] = HunkMapEntry{
			CompType:   compType,
			CompLength: length,
			Offset:     offset,
		}
	}

	return nil
}

// parseMapV4 parses a V3/V4 hunk map.
// V4 map is uncompressed, 16 bytes per entry:
//
//	Offset 0: Offset (8 bytes)
//	Offset 8: CRC32 (4 bytes)
//	Offset 12: Length (2 bytes) + Flags (2 bytes)
func (hm *HunkMap) parseMapV4() error {
	numHunks := hm.header.NumHunks()
	entrySize := 16
	mapData := make([]byte, int(numHunks)*entrySize)

	//nolint:gosec // Safe: MapOffset validated during header parsing, int64 conversion safe for valid CHD files
	if _, err := hm.reader.ReadAt(mapData, int64(hm.header.MapOffset)); err != nil {
		return fmt.Errorf("read V4 map: %w", err)
	}

	for i := range numHunks {
		offset := int(i) * entrySize

		entryOffset := binary.BigEndian.Uint64(mapData[offset : offset+8])
		// CRC32 at offset+8 (skipped)
		length := binary.BigEndian.Uint16(mapData[offset+12 : offset+14])
		flags := binary.BigEndian.Uint16(mapData[offset+14 : offset+16])

		compType := uint8(HunkCompTypeNone)
		if flags&1 != 0 {
			compType = HunkCompTypeCodec0 // Compressed
		}

		hm.entries[i] = HunkMapEntry{
			CompType:   compType,
			CompLength: uint32(length),
			Offset:     entryOffset,
		}
	}

	return nil
}

// ReadHunk reads and decompresses a hunk by index.
func (hm *HunkMap) ReadHunk(index uint32) ([]byte, error) {
	//nolint:gosec // Safe: len(entries) bounded by NumHunks which fits in uint32
	if index >= uint32(len(hm.entries)) {
		return nil, fmt.Errorf("%w: %d >= %d", ErrInvalidHunk, index, len(hm.entries))
	}

	// Check cache
	hm.cacheMu.RLock()
	if data, ok := hm.cache[index]; ok {
		hm.cacheMu.RUnlock()
		return data, nil
	}
	hm.cacheMu.RUnlock()

	// Read and decompress
	entry := hm.entries[index]
	data, err := hm.decompressHunk(entry)
	if err != nil {
		return nil, fmt.Errorf("decompress hunk %d: %w", index, err)
	}

	// Update cache
	hm.cacheMu.Lock()
	if hm.cacheSize >= hm.maxCache {
		// Simple cache eviction: clear all
		hm.cache = make(map[uint32][]byte)
		hm.cacheSize = 0
	}
	hm.cache[index] = data
	hm.cacheSize++
	hm.cacheMu.Unlock()

	return data, nil
}

// decompressHunk decompresses a single hunk.
func (hm *HunkMap) decompressHunk(entry HunkMapEntry) ([]byte, error) {
	hunkSize := int(hm.header.HunkBytes)
	dst := make([]byte, hunkSize)

	switch entry.CompType {
	case HunkCompTypeNone:
		return hm.readUncompressedHunk(dst, entry)
	case HunkCompTypeCodec0, HunkCompTypeCodec1, HunkCompTypeCodec2, HunkCompTypeCodec3:
		return hm.decompressWithCodec(dst, entry, hunkSize)
	case HunkCompTypeSelf:
		return hm.readSelfRefHunk(entry)
	default:
		return nil, fmt.Errorf("%w: compression type %d", ErrUnsupportedCodec, entry.CompType)
	}
}

// readUncompressedHunk reads an uncompressed hunk directly.
func (hm *HunkMap) readUncompressedHunk(dst []byte, entry HunkMapEntry) ([]byte, error) {
	//nolint:gosec // Safe: entry.Offset from validated hunk map
	if _, err := hm.reader.ReadAt(dst, int64(entry.Offset)); err != nil {
		return nil, fmt.Errorf("read uncompressed: %w", err)
	}
	return dst, nil
}

// decompressWithCodec decompresses a hunk using one of the registered codecs.
func (hm *HunkMap) decompressWithCodec(dst []byte, entry HunkMapEntry, hunkSize int) ([]byte, error) {
	codecIdx := int(entry.CompType)
	if codecIdx >= len(hm.codecs) || hm.codecs[codecIdx] == nil {
		return nil, fmt.Errorf("%w: codec %d not available", ErrUnsupportedCodec, codecIdx)
	}

	compData := make([]byte, entry.CompLength)
	//nolint:gosec // Safe: entry.Offset from validated hunk map
	if _, err := hm.reader.ReadAt(compData, int64(entry.Offset)); err != nil {
		return nil, fmt.Errorf("read compressed: %w", err)
	}

	codec := hm.codecs[codecIdx]

	if cdCodec, ok := codec.(CDCodec); ok {
		unitBytes := int(hm.header.UnitBytes)
		if unitBytes == 0 {
			unitBytes = 2448
		}
		frames := hunkSize / unitBytes

		decompN, err := cdCodec.DecompressCD(dst, compData, hunkSize, frames)
		if err != nil {
			return nil, fmt.Errorf("decompress CD: %w", err)
		}
		return dst[:decompN], nil
	}

	decompN, err := codec.Decompress(dst, compData)
	if err != nil {
		return nil, fmt.Errorf("decompress: %w", err)
	}
	return dst[:decompN], nil
}

// readSelfRefHunk reads a hunk that references another hunk.
func (hm *HunkMap) readSelfRefHunk(entry HunkMapEntry) ([]byte, error) {
	//nolint:gosec // Safe: entry.Offset used as hunk index, validated below
	refHunk := uint32(entry.Offset)
	//nolint:gosec // Safe: len(entries) bounded by NumHunks
	if refHunk >= uint32(len(hm.entries)) {
		return nil, fmt.Errorf("%w: self-ref %d", ErrInvalidHunk, refHunk)
	}
	return hm.ReadHunk(refHunk)
}

// NumHunks returns the total number of hunks.
func (hm *HunkMap) NumHunks() uint32 {
	//nolint:gosec // Safe: len(entries) bounded by NumHunks which fits in uint32
	return uint32(len(hm.entries))
}

// HunkBytes returns the size of each hunk in bytes.
func (hm *HunkMap) HunkBytes() uint32 {
	return hm.header.HunkBytes
}

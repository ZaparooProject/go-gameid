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
package chd

import (
	"encoding/binary"
	"fmt"
	"io"
)

// CHD format magic word
var chdMagic = [8]byte{'M', 'C', 'o', 'm', 'p', 'r', 'H', 'D'}

// Header sizes for different CHD versions
const (
	headerSizeV3 = 120
	headerSizeV4 = 108
	headerSizeV5 = 124
)

// Header represents a CHD file header.
// This struct supports V5 format (current standard) with fields for earlier versions.
type Header struct {
	Magic        [8]byte   // "MComprHD"
	HeaderSize   uint32    // Header length in bytes
	Version      uint32    // CHD version (3, 4, or 5)
	Compressors  [4]uint32 // Compression codec tags (V5)
	LogicalBytes uint64    // Total uncompressed size
	MapOffset    uint64    // Offset to hunk map
	MetaOffset   uint64    // Offset to metadata
	HunkBytes    uint32    // Bytes per hunk
	UnitBytes    uint32    // Bytes per unit (sector size)
	RawSHA1      [20]byte  // SHA1 of raw data
	SHA1         [20]byte  // SHA1 of raw + metadata
	ParentSHA1   [20]byte  // Parent SHA1 (for delta CHDs)

	// V3/V4 specific fields
	Flags       uint32 // V3/V4 flags
	Compression uint32 // V3/V4 compression type
	TotalHunks  uint32 // V3/V4 total number of hunks
}

// parseHeader reads and parses a CHD header from the given reader.
func parseHeader(reader io.Reader) (*Header, error) {
	// Read magic and header size first
	magicBuf := make([]byte, 12)
	if _, err := io.ReadFull(reader, magicBuf); err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}

	var header Header
	copy(header.Magic[:], magicBuf[:8])

	// Verify magic
	if header.Magic != chdMagic {
		return nil, ErrInvalidMagic
	}

	header.HeaderSize = binary.BigEndian.Uint32(magicBuf[8:12])

	// Read rest of header based on size
	remaining := int(header.HeaderSize) - 12
	if remaining <= 0 {
		return nil, fmt.Errorf("%w: header size %d", ErrInvalidHeader, header.HeaderSize)
	}

	headerBuf := make([]byte, remaining)
	if _, err := io.ReadFull(reader, headerBuf); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	// Parse version
	header.Version = binary.BigEndian.Uint32(headerBuf[0:4])

	switch header.Version {
	case 5:
		if err := parseHeaderV5(&header, headerBuf); err != nil {
			return nil, err
		}
	case 4:
		if err := parseHeaderV4(&header, headerBuf); err != nil {
			return nil, err
		}
	case 3:
		if err := parseHeaderV3(&header, headerBuf); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w: version %d", ErrUnsupportedVersion, header.Version)
	}

	return &header, nil
}

// parseHeaderV5 parses a V5 CHD header.
// V5 header layout (after magic + size + version, total 124 bytes):
//
//	Offset 0x00: Magic (8 bytes)
//	Offset 0x08: Header size (4 bytes)
//	Offset 0x0C: Version (4 bytes)
//	Offset 0x10: Compressor 0 (4 bytes)
//	Offset 0x14: Compressor 1 (4 bytes)
//	Offset 0x18: Compressor 2 (4 bytes)
//	Offset 0x1C: Compressor 3 (4 bytes)
//	Offset 0x20: Logical bytes (8 bytes)
//	Offset 0x28: Map offset (8 bytes)
//	Offset 0x30: Meta offset (8 bytes)
//	Offset 0x38: Hunk bytes (4 bytes)
//	Offset 0x3C: Unit bytes (4 bytes)
//	Offset 0x40: Raw SHA1 (20 bytes)
//	Offset 0x54: SHA1 (20 bytes)
//	Offset 0x68: Parent SHA1 (20 bytes)
func parseHeaderV5(header *Header, buf []byte) error {
	if len(buf) < headerSizeV5-12 {
		return fmt.Errorf("%w: buffer too small for V5", ErrInvalidHeader)
	}

	// Compressors (4 x 4 bytes starting at offset 4 in buf, which is offset 0x10 in file)
	header.Compressors[0] = binary.BigEndian.Uint32(buf[4:8])
	header.Compressors[1] = binary.BigEndian.Uint32(buf[8:12])
	header.Compressors[2] = binary.BigEndian.Uint32(buf[12:16])
	header.Compressors[3] = binary.BigEndian.Uint32(buf[16:20])

	// Logical bytes (8 bytes at offset 20 in buf)
	header.LogicalBytes = binary.BigEndian.Uint64(buf[20:28])

	// Map offset (8 bytes at offset 28 in buf)
	header.MapOffset = binary.BigEndian.Uint64(buf[28:36])

	// Meta offset (8 bytes at offset 36 in buf)
	header.MetaOffset = binary.BigEndian.Uint64(buf[36:44])

	// Hunk bytes (4 bytes at offset 44 in buf)
	header.HunkBytes = binary.BigEndian.Uint32(buf[44:48])

	// Unit bytes (4 bytes at offset 48 in buf)
	header.UnitBytes = binary.BigEndian.Uint32(buf[48:52])

	// Raw SHA1 (20 bytes at offset 52 in buf)
	copy(header.RawSHA1[:], buf[52:72])

	// SHA1 (20 bytes at offset 72 in buf)
	copy(header.SHA1[:], buf[72:92])

	// Parent SHA1 (20 bytes at offset 92 in buf)
	copy(header.ParentSHA1[:], buf[92:112])

	return nil
}

// parseHeaderV4 parses a V4 CHD header.
// V4 header layout (108 bytes total):
//
//	Offset 0x00: Magic (8 bytes)
//	Offset 0x08: Header size (4 bytes)
//	Offset 0x0C: Version (4 bytes)
//	Offset 0x10: Flags (4 bytes)
//	Offset 0x14: Compression (4 bytes)
//	Offset 0x18: Total hunks (4 bytes)
//	Offset 0x1C: Logical bytes (8 bytes)
//	Offset 0x24: Meta offset (8 bytes)
//	Offset 0x2C: Hunk bytes (4 bytes)
//	Offset 0x30: SHA1 (20 bytes)
//	Offset 0x44: Parent SHA1 (20 bytes)
//	Offset 0x58: Raw SHA1 (20 bytes)
func parseHeaderV4(header *Header, buf []byte) error {
	if len(buf) < headerSizeV4-12 {
		return fmt.Errorf("%w: buffer too small for V4", ErrInvalidHeader)
	}

	// Flags (4 bytes at offset 4 in buf)
	header.Flags = binary.BigEndian.Uint32(buf[4:8])

	// Compression (4 bytes at offset 8 in buf)
	header.Compression = binary.BigEndian.Uint32(buf[8:12])

	// Total hunks (4 bytes at offset 12 in buf)
	header.TotalHunks = binary.BigEndian.Uint32(buf[12:16])

	// Logical bytes (8 bytes at offset 16 in buf)
	header.LogicalBytes = binary.BigEndian.Uint64(buf[16:24])

	// Meta offset (8 bytes at offset 24 in buf)
	header.MetaOffset = binary.BigEndian.Uint64(buf[24:32])

	// Hunk bytes (4 bytes at offset 32 in buf)
	header.HunkBytes = binary.BigEndian.Uint32(buf[32:36])

	// SHA1 (20 bytes at offset 36 in buf)
	copy(header.SHA1[:], buf[36:56])

	// Parent SHA1 (20 bytes at offset 56 in buf)
	copy(header.ParentSHA1[:], buf[56:76])

	// Raw SHA1 (20 bytes at offset 76 in buf)
	copy(header.RawSHA1[:], buf[76:96])

	// V4 doesn't have unit bytes - calculate from typical CD sector size
	header.UnitBytes = 2448 // Default for CD-ROM

	// Map offset for V4 is right after header
	header.MapOffset = uint64(header.HeaderSize)

	return nil
}

// parseHeaderV3 parses a V3 CHD header.
// V3 header layout (120 bytes total):
//
//	Offset 0x00: Magic (8 bytes)
//	Offset 0x08: Header size (4 bytes)
//	Offset 0x0C: Version (4 bytes)
//	Offset 0x10: Flags (4 bytes)
//	Offset 0x14: Compression (4 bytes)
//	Offset 0x18: Total hunks (4 bytes)
//	Offset 0x1C: Logical bytes (8 bytes)
//	Offset 0x24: Meta offset (8 bytes)
//	Offset 0x2C: MD5 (16 bytes)
//	Offset 0x3C: Parent MD5 (16 bytes)
//	Offset 0x4C: Hunk bytes (4 bytes)
//	Offset 0x50: SHA1 (20 bytes)
//	Offset 0x64: Parent SHA1 (20 bytes)
func parseHeaderV3(header *Header, buf []byte) error {
	if len(buf) < headerSizeV3-12 {
		return fmt.Errorf("%w: buffer too small for V3", ErrInvalidHeader)
	}

	// Flags (4 bytes at offset 4 in buf)
	header.Flags = binary.BigEndian.Uint32(buf[4:8])

	// Compression (4 bytes at offset 8 in buf)
	header.Compression = binary.BigEndian.Uint32(buf[8:12])

	// Total hunks (4 bytes at offset 12 in buf)
	header.TotalHunks = binary.BigEndian.Uint32(buf[12:16])

	// Logical bytes (8 bytes at offset 16 in buf)
	header.LogicalBytes = binary.BigEndian.Uint64(buf[16:24])

	// Meta offset (8 bytes at offset 24 in buf)
	header.MetaOffset = binary.BigEndian.Uint64(buf[24:32])

	// MD5 hashes skipped (16 + 16 = 32 bytes at offset 32)

	// Hunk bytes (4 bytes at offset 64 in buf)
	header.HunkBytes = binary.BigEndian.Uint32(buf[64:68])

	// SHA1 (20 bytes at offset 68 in buf)
	copy(header.SHA1[:], buf[68:88])

	// Parent SHA1 (20 bytes at offset 88 in buf)
	copy(header.ParentSHA1[:], buf[88:108])

	// V3 doesn't have unit bytes - calculate from typical CD sector size
	header.UnitBytes = 2448 // Default for CD-ROM

	// Map offset for V3 is right after header
	header.MapOffset = uint64(header.HeaderSize)

	return nil
}

// NumHunks returns the total number of hunks in the CHD file.
func (h *Header) NumHunks() uint32 {
	if h.TotalHunks > 0 {
		return h.TotalHunks
	}
	if h.HunkBytes == 0 {
		return 0
	}
	//nolint:gosec // Safe: result bounded by file size, will not overflow for valid CHD files
	return uint32((h.LogicalBytes + uint64(h.HunkBytes) - 1) / uint64(h.HunkBytes))
}

// IsCompressed returns true if the CHD uses compression.
func (h *Header) IsCompressed() bool {
	if h.Version == 5 {
		return h.Compressors[0] != 0
	}
	return h.Compression != 0
}

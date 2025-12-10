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
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/ulikunitz/xz/lzma"
)

func init() {
	RegisterCodec(CodecLZMA, func() Codec { return &lzmaCodec{} })
	RegisterCodec(CodecCDLZMA, func() Codec { return &cdLZMACodec{} })
}

// lzmaCodec implements LZMA decompression for CHD hunks.
// CHD LZMA uses raw LZMA data with NO header - properties are computed from hunkbytes.
type lzmaCodec struct {
	hunkBytes uint32 // Set by the hunk map when initializing
}

// computeLZMAProps computes LZMA properties matching MAME's configure_properties.
// MAME uses level=8 and reduceSize=hunkbytes, then normalizes to get dictSize.
// Default properties: lc=3, lp=0, pb=2 (encoded as 0x5D).
func computeLZMADictSize(hunkBytes uint32) uint32 {
	// For level 8, initial dictSize would be 1<<26, but reduced based on hunkbytes.
	// From LzmaEncProps_Normalize: find smallest 2<<i or 3<<i >= hunkBytes
	reduceSize := hunkBytes
	for i := uint32(11); i <= 30; i++ {
		if reduceSize <= (2 << i) {
			return 2 << i
		}
		if reduceSize <= (3 << i) {
			return 3 << i
		}
	}
	return 1 << 26 // fallback to level 8 default
}

// Decompress decompresses LZMA compressed data.
// CHD LZMA format: raw LZMA stream with NO header.
// Properties are computed from the decompressed size (dst length).
func (c *lzmaCodec) Decompress(dst, src []byte) (int, error) {
	if len(src) == 0 {
		return 0, fmt.Errorf("%w: lzma: empty source", ErrDecompressFailed)
	}

	// Compute properties like MAME does
	// MAME's configure_properties uses level=8, reduceSize=hunkbytes
	// After normalization: lc=3, lp=0, pb=2, dictSize computed from reduceSize
	hunkBytes := c.hunkBytes
	if hunkBytes == 0 {
		//nolint:gosec // Safe: len(dst) is hunk size, bounded by uint32
		hunkBytes = uint32(len(dst))
	}
	dictSize := computeLZMADictSize(hunkBytes)

	// Properties byte: lc + lp*9 + pb*45 = 3 + 0 + 90 = 93 = 0x5D
	const propsLcLpPb = 0x5D

	// Construct a full 13-byte LZMA header for the library:
	// Byte 0: Properties (lc=3, lp=0, pb=2 encoded as 0x5D)
	// Bytes 1-4: Dictionary size (little-endian)
	// Bytes 5-12: Uncompressed size (little-endian)
	header := make([]byte, 13)
	header[0] = propsLcLpPb
	binary.LittleEndian.PutUint32(header[1:5], dictSize)
	binary.LittleEndian.PutUint64(header[5:13], uint64(len(dst)))

	// Combine header with compressed data
	fullStream := make([]byte, 13+len(src))
	copy(fullStream[0:13], header)
	copy(fullStream[13:], src)

	reader, err := lzma.NewReader(bytes.NewReader(fullStream))
	if err != nil {
		return 0, fmt.Errorf("%w: lzma init: %w", ErrDecompressFailed, err)
	}

	n, err := io.ReadFull(reader, dst)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return n, fmt.Errorf("%w: lzma read: %w", ErrDecompressFailed, err)
	}

	return n, nil
}

// cdLZMACodec implements CD-ROM LZMA decompression.
// CD LZMA compresses sector data with LZMA and subchannel data with zlib.
type cdLZMACodec struct{}

// Decompress implements basic decompression.
func (c *cdLZMACodec) Decompress(dst, src []byte) (int, error) {
	return c.DecompressCD(dst, src, len(dst), len(dst)/2448)
}

// DecompressCD decompresses CD-ROM data with LZMA for sectors and zlib for subchannel.
// CD codec format (from MAME chdcodec.cpp):
//   - ECC bitmap: (frames + 7) / 8 bytes - indicates which frames have ECC data cleared
//   - Compressed length: 2 bytes (if destlen < 65536) or 3 bytes
//   - Base compressed data (LZMA)
//   - Subcode compressed data (zlib)
//
//nolint:gocognit,gocyclo,cyclop,revive // CD LZMA decompression requires complex sector/subchannel interleaving
func (*cdLZMACodec) DecompressCD(dst, src []byte, destLen, frames int) (int, error) {
	// Calculate header sizes (matching MAME's chd_cd_decompressor)
	compLenBytes := 2
	if destLen >= 65536 {
		compLenBytes = 3
	}
	eccBytes := (frames + 7) / 8
	headerBytes := eccBytes + compLenBytes

	if len(src) < headerBytes {
		return 0, fmt.Errorf("%w: cdlz: source too small for header", ErrDecompressFailed)
	}

	// Extract ECC bitmap (for later reconstruction)
	eccBitmap := src[:eccBytes]

	// Extract compressed base length
	var compLenBase int
	if compLenBytes > 2 {
		//nolint:gosec // G602: bounds checked via headerBytes = eccBytes + compLenBytes check above
		compLenBase = int(src[eccBytes])<<16 | int(src[eccBytes+1])<<8 | int(src[eccBytes+2])
	} else {
		compLenBase = int(binary.BigEndian.Uint16(src[eccBytes : eccBytes+2]))
	}

	if headerBytes+compLenBase > len(src) {
		return 0, fmt.Errorf("%w: cdlz: invalid base length %d", ErrDecompressFailed, compLenBase)
	}

	baseData := src[headerBytes : headerBytes+compLenBase]
	subData := src[headerBytes+compLenBase:]

	// Calculate expected sizes
	sectorSize := 2352
	subSize := 96
	totalSectorBytes := frames * sectorSize
	totalSubBytes := frames * subSize

	// Decompress sector data with LZMA
	// Note: For CD codecs, the LZMA properties are computed from totalSectorBytes
	sectorDst := make([]byte, totalSectorBytes)
	//nolint:gosec // Safe: totalSectorBytes = frames * 2352, bounded by hunk size
	lzmaCodec := &lzmaCodec{hunkBytes: uint32(totalSectorBytes)}
	sectorN, err := lzmaCodec.Decompress(sectorDst, baseData)
	if err != nil {
		return 0, fmt.Errorf("%w: cdlz sector: %w", ErrDecompressFailed, err)
	}

	// Decompress subchannel data with zlib if present
	var subDst []byte
	if len(subData) > 0 && totalSubBytes > 0 {
		subDst = make([]byte, totalSubBytes)
		reader := flate.NewReader(bytes.NewReader(subData))
		_, err = io.ReadFull(reader, subDst)
		_ = reader.Close()
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			// Subchannel decompression failure is not fatal
			subDst = make([]byte, totalSubBytes)
		}
	} else {
		subDst = make([]byte, totalSubBytes)
	}

	// Reassemble the data with ECC reconstruction
	dstOffset := 0
	for i := range frames {
		srcSectorOffset := i * sectorSize
		if srcSectorOffset+sectorSize <= sectorN {
			copy(dst[dstOffset:], sectorDst[srcSectorOffset:srcSectorOffset+sectorSize])
		}

		// Reconstitute ECC data and sync header if bit is set
		if (eccBitmap[i/8] & (1 << (i % 8))) != 0 {
			// Copy sync header
			copy(dst[dstOffset:], cdSyncHeader[:])
			// ECC generation would go here but we skip it for identification purposes
		}

		dstOffset += sectorSize

		if subSize > 0 {
			srcSubOffset := i * subSize
			if srcSubOffset+subSize <= len(subDst) {
				copy(dst[dstOffset:], subDst[srcSubOffset:srcSubOffset+subSize])
			}
			dstOffset += subSize
		}
	}

	return dstOffset, nil
}

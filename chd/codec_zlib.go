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
)

func init() {
	RegisterCodec(CodecZlib, func() Codec { return &zlibCodec{} })
	RegisterCodec(CodecCDZlib, func() Codec { return &cdZlibCodec{} })
}

// zlibCodec implements zlib decompression for CHD hunks.
// Note: CHD uses raw deflate (RFC 1951), not zlib wrapper.
type zlibCodec struct{}

// Decompress decompresses zlib/deflate compressed data.
func (*zlibCodec) Decompress(dst, src []byte) (int, error) {
	reader := flate.NewReader(bytes.NewReader(src))
	defer func() { _ = reader.Close() }()

	n, err := io.ReadFull(reader, dst)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return n, fmt.Errorf("%w: zlib: %w", ErrDecompressFailed, err)
	}

	return n, nil
}

// cdZlibCodec implements CD-ROM zlib decompression.
// CD zlib compresses sector data with deflate and subchannel data separately.
type cdZlibCodec struct{}

// Decompress implements basic decompression (delegates to DecompressCD with defaults).
func (c *cdZlibCodec) Decompress(dst, src []byte) (int, error) {
	// For generic decompression, assume standard CD sector size
	// This is a fallback; normally DecompressCD should be called
	return c.DecompressCD(dst, src, len(dst), len(dst)/2448)
}

// DecompressCD decompresses CD-ROM data with sector/subchannel handling.
// CD codec format (from MAME chdcodec.cpp):
//   - ECC bitmap: (frames + 7) / 8 bytes - indicates which frames have ECC data cleared
//   - Compressed length: 2 bytes (if destlen < 65536) or 3 bytes
//   - Base compressed data (deflate)
//   - Subcode compressed data (deflate)
//
//nolint:gocognit,gocyclo,cyclop,revive // CD zlib decompression requires complex sector/subchannel interleaving
func (*cdZlibCodec) DecompressCD(dst, src []byte, destLen, frames int) (int, error) {
	// Calculate header sizes (matching MAME's chd_cd_decompressor)
	compLenBytes := 2
	if destLen >= 65536 {
		compLenBytes = 3
	}
	eccBytes := (frames + 7) / 8
	headerBytes := eccBytes + compLenBytes

	if len(src) < headerBytes {
		return 0, fmt.Errorf("%w: cdzl: source too small for header", ErrDecompressFailed)
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
		return 0, fmt.Errorf("%w: cdzl: invalid base length %d", ErrDecompressFailed, compLenBase)
	}

	baseData := src[headerBytes : headerBytes+compLenBase]
	subData := src[headerBytes+compLenBase:]

	// Calculate expected sizes
	sectorSize := 2352
	subSize := 96
	totalSectorBytes := frames * sectorSize
	totalSubBytes := frames * subSize

	// Decompress sector data
	sectorDst := make([]byte, totalSectorBytes)
	reader := flate.NewReader(bytes.NewReader(baseData))
	sectorN, err := io.ReadFull(reader, sectorDst)
	_ = reader.Close()
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return 0, fmt.Errorf("%w: cdzl sector: %w", ErrDecompressFailed, err)
	}

	// Decompress subchannel data if present
	var subDst []byte
	if len(subData) > 0 && totalSubBytes > 0 {
		subDst = make([]byte, totalSubBytes)
		reader = flate.NewReader(bytes.NewReader(subData))
		_, err = io.ReadFull(reader, subDst)
		_ = reader.Close()
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			// Subchannel decompression failure is not fatal - may be zero-filled
			subDst = make([]byte, totalSubBytes)
		}
	} else {
		subDst = make([]byte, totalSubBytes)
	}

	// Reassemble the data with ECC reconstruction
	dstOffset := 0
	for i := range frames {
		// Copy sector data
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

		// Copy subchannel data
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

// cdSyncHeader is the standard CD-ROM sync header.
var cdSyncHeader = [12]byte{0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00}

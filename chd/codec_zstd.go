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

	"github.com/klauspost/compress/zstd"
)

func init() {
	RegisterCodec(CodecZstd, func() Codec { return &zstdCodec{} })
	RegisterCodec(CodecCDZstd, func() Codec { return &cdZstdCodec{} })
}

// zstdCodec implements Zstandard decompression for CHD hunks.
type zstdCodec struct {
	decoder *zstd.Decoder
}

// Decompress decompresses Zstandard compressed data.
func (z *zstdCodec) Decompress(dst, src []byte) (int, error) {
	if z.decoder == nil {
		decoder, err := zstd.NewReader(nil)
		if err != nil {
			return 0, fmt.Errorf("%w: zstd init: %w", ErrDecompressFailed, err)
		}
		z.decoder = decoder
	}

	result, err := z.decoder.DecodeAll(src, dst[:0])
	if err != nil {
		return 0, fmt.Errorf("%w: zstd: %w", ErrDecompressFailed, err)
	}

	// Copy result to dst if needed
	if len(result) > len(dst) {
		return 0, fmt.Errorf("%w: zstd: output too large", ErrDecompressFailed)
	}
	if &result[0] != &dst[0] {
		copy(dst, result)
	}

	return len(result), nil
}

// cdZstdCodec implements CD-ROM Zstandard decompression.
// CD Zstd compresses sector data with Zstandard and subchannel data with zlib.
type cdZstdCodec struct {
	decoder *zstd.Decoder
}

// Decompress implements basic decompression.
func (c *cdZstdCodec) Decompress(dst, src []byte) (int, error) {
	return c.DecompressCD(dst, src, len(dst), len(dst)/2448)
}

// DecompressCD decompresses CD-ROM data with Zstandard for sectors and zlib for subchannel.
// CD Zstd format:
//   - First 4 bytes: compressed sector data length (big-endian)
//   - Next N bytes: Zstd-compressed sector data
//   - Remaining bytes: zlib-compressed subchannel data
//
//nolint:gocognit,revive // CD Zstd decompression requires complex sector/subchannel interleaving
func (c *cdZstdCodec) DecompressCD(dst, src []byte, _, frames int) (int, error) {
	if len(src) < 4 {
		return 0, fmt.Errorf("%w: cdzs: source too small", ErrDecompressFailed)
	}

	// Read compressed sector data length
	sectorCompLen := binary.BigEndian.Uint32(src[0:4])
	if int(sectorCompLen) > len(src)-4 {
		return 0, fmt.Errorf("%w: cdzs: invalid sector length %d", ErrDecompressFailed, sectorCompLen)
	}

	sectorData := src[4 : 4+sectorCompLen]
	subData := src[4+sectorCompLen:]

	// Calculate expected sizes
	sectorSize := 2352
	subSize := 96
	totalSectorBytes := frames * sectorSize
	totalSubBytes := frames * subSize

	// Initialize decoder if needed
	if c.decoder == nil {
		decoder, err := zstd.NewReader(nil)
		if err != nil {
			return 0, fmt.Errorf("%w: cdzs init: %w", ErrDecompressFailed, err)
		}
		c.decoder = decoder
	}

	// Decompress sector data with Zstandard
	sectorDst, err := c.decoder.DecodeAll(sectorData, make([]byte, 0, totalSectorBytes))
	if err != nil {
		return 0, fmt.Errorf("%w: cdzs sector: %w", ErrDecompressFailed, err)
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

	// Interleave sector and subchannel data
	dstOffset := 0
	for i := range frames {
		srcSectorOffset := i * sectorSize
		if srcSectorOffset+sectorSize <= len(sectorDst) {
			copy(dst[dstOffset:], sectorDst[srcSectorOffset:srcSectorOffset+sectorSize])
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

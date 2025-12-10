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
	"fmt"
	"sync"
)

// Codec tag constants (as 4-byte big-endian integers representing ASCII strings).
// CD-ROM specific codecs handle both data and subchannel compression.
const (
	// CodecNone indicates uncompressed data.
	CodecNone uint32 = 0x00000000

	// CodecZlib is the standard zlib codec ("zlib").
	CodecZlib uint32 = 0x7a6c6962

	// CodecLZMA is the LZMA codec ("lzma").
	CodecLZMA uint32 = 0x6c7a6d61

	// CodecHuff is the CHD Huffman codec ("huff").
	CodecHuff uint32 = 0x68756666

	// CodecFLAC is the FLAC audio codec ("flac").
	CodecFLAC uint32 = 0x666c6163

	// CodecZstd is the Zstandard codec ("zstd").
	CodecZstd uint32 = 0x7a737464

	// CodecCDZlib is the CD zlib codec ("cdzl").
	// Compresses CD data sectors with zlib, subchannel with zlib.
	CodecCDZlib uint32 = 0x63647a6c

	// CodecCDLZMA is the CD LZMA codec ("cdlz").
	// Compresses CD data sectors with LZMA, subchannel with zlib.
	CodecCDLZMA uint32 = 0x63646c7a

	// CodecCDFLAC is the CD FLAC codec ("cdfl").
	// Compresses CD audio sectors with FLAC, subchannel with zlib.
	CodecCDFLAC uint32 = 0x6364666c

	// CodecCDZstd is the CD Zstandard codec ("cdzs").
	// Compresses CD data sectors with Zstandard, subchannel with zlib.
	CodecCDZstd uint32 = 0x63647a73
)

// Codec decompresses CHD hunk data.
type Codec interface {
	// Decompress decompresses src into dst.
	// dst must be pre-allocated to the expected decompressed size.
	// Returns the number of bytes written to dst.
	Decompress(dst, src []byte) (int, error)
}

// CDCodec decompresses CD-ROM specific hunk data.
// CD codecs handle the separation of sector data and subchannel data.
type CDCodec interface {
	Codec

	// DecompressCD decompresses CD-ROM data with sector/subchannel handling.
	// hunkBytes is the total size of a decompressed hunk.
	// frames is the number of CD frames (sectors) in the hunk.
	DecompressCD(dst, src []byte, hunkBytes, frames int) (int, error)
}

// codecRegistry holds registered codecs.
var (
	codecRegistry   = make(map[uint32]func() Codec)
	codecRegistryMu sync.RWMutex
)

// RegisterCodec registers a codec factory for the given tag.
func RegisterCodec(tag uint32, factory func() Codec) {
	codecRegistryMu.Lock()
	defer codecRegistryMu.Unlock()
	codecRegistry[tag] = factory
}

// GetCodec returns a codec instance for the given tag.
func GetCodec(tag uint32) (Codec, error) {
	codecRegistryMu.RLock()
	factory, ok := codecRegistry[tag]
	codecRegistryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: 0x%08x (%s)", ErrUnsupportedCodec, tag, codecTagToString(tag))
	}

	return factory(), nil
}

// codecTagToString converts a codec tag to its ASCII representation.
func codecTagToString(tag uint32) string {
	if tag == 0 {
		return "none"
	}
	tagBytes := []byte{
		byte(tag >> 24),
		byte(tag >> 16),
		byte(tag >> 8),
		byte(tag),
	}
	return string(tagBytes)
}

// IsCDCodec returns true if the codec tag is a CD-ROM specific codec.
func IsCDCodec(tag uint32) bool {
	switch tag {
	case CodecCDZlib, CodecCDLZMA, CodecCDFLAC, CodecCDZstd:
		return true
	default:
		return false
	}
}

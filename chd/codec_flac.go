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
	"errors"
	"fmt"
	"io"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
)

func init() {
	RegisterCodec(CodecFLAC, func() Codec { return &flacCodec{} })
	RegisterCodec(CodecCDFLAC, func() Codec { return &cdFLACCodec{} })
}

// flacCodec implements FLAC decompression for CHD hunks.
type flacCodec struct{}

// Decompress decompresses FLAC compressed data.
func (*flacCodec) Decompress(dst, src []byte) (int, error) {
	stream, err := flac.New(bytes.NewReader(src))
	if err != nil {
		return 0, fmt.Errorf("%w: flac init: %w", ErrDecompressFailed, err)
	}
	defer func() { _ = stream.Close() }()

	return decodeFLACFrames(stream, dst)
}

// decodeFLACFrames decodes all FLAC frames into the destination buffer.
func decodeFLACFrames(stream *flac.Stream, dst []byte) (int, error) {
	offset := 0
	for {
		audioFrame, err := stream.ParseNext()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return offset, fmt.Errorf("%w: flac frame: %w", ErrDecompressFailed, err)
		}

		offset = writeFLACFrameSamples(audioFrame, dst, offset)
	}
	return offset, nil
}

// writeFLACFrameSamples writes samples from a FLAC frame to the destination buffer.
func writeFLACFrameSamples(audioFrame *frame.Frame, dst []byte, offset int) int {
	if len(audioFrame.Subframes) == 0 {
		return offset
	}

	numChannels := min(len(audioFrame.Subframes), 2)
	for i := range audioFrame.Subframes[0].NSamples {
		for ch := range numChannels {
			sample := audioFrame.Subframes[ch].Samples[i]
			if offset+2 <= len(dst) {
				dst[offset] = byte(sample >> 8)
				dst[offset+1] = byte(sample)
				offset += 2
			}
		}
	}
	return offset
}

// cdFLACCodec implements CD-ROM FLAC decompression.
// CD FLAC compresses audio sectors with FLAC and subchannel data with zlib.
type cdFLACCodec struct{}

// Decompress implements basic decompression.
func (c *cdFLACCodec) Decompress(dst, src []byte) (int, error) {
	return c.DecompressCD(dst, src, len(dst), len(dst)/2448)
}

// CD audio constants.
const (
	cdSectorSize = 2352
	cdSubSize    = 96
)

// DecompressCD decompresses CD audio data with FLAC and subchannel with zlib.
// CD FLAC format (from MAME chdcodec.cpp):
//   - FLAC stream starts directly at offset 0 (NO length header)
//   - FLAC decoder determines where the stream ends
//   - Remaining bytes after FLAC: zlib-compressed subchannel data
//
// Note: FLAC decompression may fail for headerless streams that the Go library
// cannot parse. In that case, we return zeros for the audio data since game
// identification only needs data tracks, not audio tracks.
func (*cdFLACCodec) DecompressCD(dst, src []byte, _, frames int) (int, error) {
	if len(src) == 0 {
		return 0, fmt.Errorf("%w: cdfl: empty source", ErrDecompressFailed)
	}

	totalSectorBytes := frames * cdSectorSize
	totalSubBytes := frames * cdSubSize

	// Decompress FLAC audio - returns both data and bytes consumed
	sectorDst, flacBytesConsumed, err := decompressCDFLACAudioWithOffset(src, totalSectorBytes)
	if err != nil {
		// FLAC decompression failed - this is likely an audio track.
		// Return zeros for the audio data since we only need data tracks for identification.
		sectorDst = make([]byte, totalSectorBytes)
		flacBytesConsumed = len(src) // Assume all data is FLAC, no subchannel
	}

	// Subchannel data starts after FLAC data
	var subDst []byte
	if flacBytesConsumed < len(src) {
		subData := src[flacBytesConsumed:]
		subDst = decompressCDSubchannel(subData, totalSubBytes)
	} else {
		subDst = make([]byte, totalSubBytes)
	}

	return interleaveCDData(dst, sectorDst, subDst, frames), nil
}

// countingReader wraps a reader and tracks bytes read from the original data.
type countingReader struct {
	header        []byte
	data          []byte
	headerPos     int
	dataPos       int
	bytesFromData int
}

func (cr *countingReader) Read(buf []byte) (int, error) {
	totalRead := 0

	// First read from synthetic header
	if cr.headerPos < len(cr.header) {
		n := copy(buf, cr.header[cr.headerPos:])
		cr.headerPos += n
		totalRead += n
		buf = buf[n:]
	}

	// Then read from actual data
	if len(buf) > 0 && cr.dataPos < len(cr.data) {
		n := copy(buf, cr.data[cr.dataPos:])
		cr.dataPos += n
		cr.bytesFromData += n
		totalRead += n
	}

	if totalRead == 0 {
		return 0, io.EOF
	}
	return totalRead, nil
}

// flacHeaderTemplate is the synthetic FLAC header used by MAME for CHD.
// This is a minimal valid FLAC stream header with STREAMINFO metadata.
// From MAME's src/lib/util/flac.cpp s_header_template.
//
//nolint:gochecknoglobals // Template constant for FLAC header generation
var flacHeaderTemplate = []byte{
	0x66, 0x4C, 0x61, 0x43, // "fLaC" magic
	0x80, 0x00, 0x00, 0x22, // STREAMINFO block header (last=1, type=0, length=34)
	0x00, 0x00, // min block size (will be patched)
	0x00, 0x00, // max block size (will be patched)
	0x00, 0x00, 0x00, // min frame size
	0x00, 0x00, 0x00, // max frame size
	0x00, 0x00, 0x0A, 0xC4, 0x42, 0xF0, // sample rate, channels, bits (will be patched)
	0x00, 0x00, 0x00, 0x00, // total samples (upper)
	0x00, 0x00, 0x00, 0x00, // total samples (lower)
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // MD5 signature
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // MD5 signature continued
}

// buildFLACHeader creates a synthetic FLAC header for CD audio.
// Parameters match MAME's flac_decoder::reset(sample_rate, num_channels, block_size, ...).
func buildFLACHeader(sampleRate uint32, numChannels uint8, blockSize uint16) []byte {
	header := make([]byte, len(flacHeaderTemplate))
	copy(header, flacHeaderTemplate)

	// Patch block sizes at offsets 0x08 and 0x0A (big-endian 16-bit)
	header[0x08] = byte(blockSize >> 8)
	header[0x09] = byte(blockSize)
	header[0x0A] = byte(blockSize >> 8)
	header[0x0B] = byte(blockSize)

	// Patch sample rate, channels, bits at offset 0x12 (big-endian 24-bit)
	// Format: (sample_rate << 4) | ((num_channels - 1) << 1) | (bits_per_sample - 1 >> 4)
	// For 16-bit audio: bits_per_sample = 16, so (16-1) >> 4 = 0
	val := (sampleRate << 4) | (uint32(numChannels-1) << 1)
	header[0x12] = byte(val >> 16)
	header[0x13] = byte(val >> 8)
	header[0x14] = byte(val)

	return header
}

// cdFLACBlockSize calculates the FLAC block size for CD audio.
// From MAME's chd_cd_flac_compressor::blocksize().
func cdFLACBlockSize(totalBytes int) uint16 {
	// MAME: blocksize = bytes / 4; while (blocksize > MAX_SECTOR_DATA) blocksize /= 2;
	// MAX_SECTOR_DATA = 2352
	blocksize := totalBytes / 4
	for blocksize > 2352 {
		blocksize /= 2
	}
	//nolint:gosec // Safe: blocksize bounded to <= 2352
	return uint16(blocksize)
}

// decompressCDFLACAudioWithOffset decompresses FLAC audio and returns bytes consumed.
func decompressCDFLACAudioWithOffset(audioData []byte, totalBytes int) (decoded []byte, bytesConsumed int, err error) {
	sectorDst := make([]byte, totalBytes)

	// Build synthetic FLAC header (CD audio: 44100 Hz, stereo, 16-bit)
	blockSize := cdFLACBlockSize(totalBytes)
	header := buildFLACHeader(44100, 2, blockSize)

	cr := &countingReader{
		header: header,
		data:   audioData,
	}

	stream, err := flac.New(cr)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: cdfl flac init: %w", ErrDecompressFailed, err)
	}
	defer func() { _ = stream.Close() }()

	_, err = decodeFLACFrames(stream, sectorDst)
	if err != nil {
		return nil, 0, err
	}

	return sectorDst, cr.bytesFromData, nil
}

// decompressCDSubchannel decompresses zlib-compressed subchannel data.
func decompressCDSubchannel(subData []byte, totalBytes int) []byte {
	if len(subData) == 0 || totalBytes == 0 {
		return make([]byte, totalBytes)
	}

	subDst := make([]byte, totalBytes)
	reader := flate.NewReader(bytes.NewReader(subData))
	_, err := io.ReadFull(reader, subDst)
	_ = reader.Close()

	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return make([]byte, totalBytes)
	}
	return subDst
}

// interleaveCDData interleaves sector and subchannel data into the destination.
func interleaveCDData(dst, sectorDst, subDst []byte, frames int) int {
	dstOffset := 0
	for i := range frames {
		srcSectorOffset := i * cdSectorSize
		if srcSectorOffset+cdSectorSize <= len(sectorDst) {
			copy(dst[dstOffset:], sectorDst[srcSectorOffset:srcSectorOffset+cdSectorSize])
		}
		dstOffset += cdSectorSize

		srcSubOffset := i * cdSubSize
		if srcSubOffset+cdSubSize <= len(subDst) {
			copy(dst[dstOffset:], subDst[srcSubOffset:srcSubOffset+cdSubSize])
		}
		dstOffset += cdSubSize
	}
	return dstOffset
}

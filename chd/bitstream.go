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

// bitReader reads bits from a byte slice.
type bitReader struct {
	data   []byte
	offset int  // bit offset
	bits   uint // accumulated bits
	avail  int  // bits available in accumulator
}

// newBitReader creates a new bit reader.
func newBitReader(data []byte) *bitReader {
	return &bitReader{data: data}
}

// read reads count bits from the stream.
func (br *bitReader) read(count int) uint32 {
	// Fill accumulator as needed
	for br.avail < count {
		byteOff := br.offset / 8
		if byteOff >= len(br.data) {
			br.bits <<= 8
			br.avail += 8
			continue
		}
		br.bits = (br.bits << 8) | uint(br.data[byteOff])
		br.avail += 8
		br.offset += 8
	}

	// Extract the bits
	br.avail -= count
	//nolint:gosec // Safe: bits accumulator is bounded by count which is at most 32
	result := uint32((br.bits >> br.avail) & ((1 << count) - 1))
	return result
}

// huffmanDecoder decodes Huffman-encoded data for CHD V5 maps.
type huffmanDecoder struct {
	lookup   []uint32
	nodeBits []uint8
	numCodes int
	maxBits  int
}

// newHuffmanDecoder creates a Huffman decoder for the given parameters.
func newHuffmanDecoder(numCodes, maxBits int) *huffmanDecoder {
	return &huffmanDecoder{
		numCodes: numCodes,
		maxBits:  maxBits,
		nodeBits: make([]uint8, numCodes),
		lookup:   make([]uint32, 1<<maxBits),
	}
}

// importTreeRLE imports a Huffman tree encoded with RLE.
func (hd *huffmanDecoder) importTreeRLE(br *bitReader) error {
	// Determine number of bits to read for each node
	var numBits int
	switch {
	case hd.maxBits >= 16:
		numBits = 5
	case hd.maxBits >= 8:
		numBits = 4
	default:
		numBits = 3
	}

	// Read the tree with RLE decoding
	for curNode := 0; curNode < hd.numCodes; {
		nodeBits := br.read(numBits)
		if nodeBits != 1 {
			//nolint:gosec // Safe: nodeBits from Huffman tree is bounded to 0-32
			hd.nodeBits[curNode] = uint8(nodeBits)
			curNode++
			continue
		}
		// RLE encoding: read actual value
		nodeBits = br.read(numBits)
		if nodeBits == 1 {
			// Literal 1
			hd.nodeBits[curNode] = 1
			curNode++
			continue
		}
		// Repeat count follows
		repCount := int(br.read(numBits)) + 3
		//nolint:gosec // Safe: nodeBits from Huffman tree is bounded to 0-32
		curNode = hd.fillNodeBits(curNode, uint8(nodeBits), repCount)
	}

	// Build lookup table
	return hd.buildLookup()
}

// fillNodeBits fills nodeBits with a repeated value, returning the new curNode.
func (hd *huffmanDecoder) fillNodeBits(curNode int, value uint8, repCount int) int {
	for i := 0; i < repCount && curNode < hd.numCodes; i++ {
		hd.nodeBits[curNode] = value
		curNode++
	}
	return curNode
}

// buildLookup builds the lookup table from node bits.
// This follows MAME's canonical code assignment which processes from highest to lowest bit length.
func (hd *huffmanDecoder) buildLookup() error {
	// Build histogram of bit lengths
	bithisto := make([]uint32, 33)
	for i := range hd.numCodes {
		if hd.nodeBits[i] <= 32 {
			bithisto[hd.nodeBits[i]]++
		}
	}

	// For each code length, determine the starting code number
	// Process from highest to lowest bit length (MAME convention)
	var curstart uint32
	for codelen := 32; codelen > 0; codelen-- {
		nextstart := (curstart + bithisto[codelen]) >> 1
		bithisto[codelen] = curstart
		curstart = nextstart
	}

	// Now assign canonical codes and build lookup table
	// nodeBits stores the assigned code for each symbol
	nodeCodes := make([]uint32, hd.numCodes)
	for i := range hd.numCodes {
		bits := hd.nodeBits[i]
		if bits > 0 {
			nodeCodes[i] = bithisto[bits]
			bithisto[bits]++
		}
	}

	// Build lookup table
	for i := range hd.numCodes {
		bits := int(hd.nodeBits[i])
		if bits > 0 {
			// Set up the entry: (symbol << 5) | numbits
			//nolint:gosec // Safe: i bounded by numCodes (16), bits bounded by maxBits (8)
			value := uint32((i << 5) | bits)

			// Fill all matching entries
			shift := hd.maxBits - bits
			base := int(nodeCodes[i]) << shift
			end := int(nodeCodes[i]+1)<<shift - 1
			for j := base; j <= end; j++ {
				hd.lookup[j] = value
			}
		}
	}

	return nil
}

// decode decodes a single symbol from the bit stream.
func (hd *huffmanDecoder) decode(br *bitReader) uint8 {
	// Peek maxBits bits
	peek := br.read(hd.maxBits)
	entry := hd.lookup[peek]
	//nolint:gosec // Safe: entry stores symbol in upper bits, bounded by numCodes (16)
	symbol := uint8(entry >> 5)
	bits := int(entry & 0x1f)

	// Put back unused bits by adjusting the bit reader
	if bits < hd.maxBits {
		br.avail += hd.maxBits - bits
	}

	return symbol
}

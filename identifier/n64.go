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

package identifier

import (
	"fmt"
	"io"

	"github.com/ZaparooProject/go-gameid/internal/binary"
)

// N64 header offsets
const (
	n64HeaderSize         = 0x40
	n64FirstWordOffset    = 0x00
	n64InternalNameOffset = 0x20
	n64InternalNameSize   = 20 // 0x20-0x34
	n64CartridgeIDOffset  = 0x3C
	n64CartridgeIDSize    = 2
	n64CountryCodeOffset  = 0x3E
	n64VersionOffset      = 0x3F
)

// N64 first word magic - indicates big-endian format
var n64FirstWord = []byte{0x80, 0x37, 0x12, 0x40}

// N64Identifier identifies Nintendo 64 games.
type N64Identifier struct{}

// NewN64Identifier creates a new N64 identifier.
func NewN64Identifier() *N64Identifier {
	return &N64Identifier{}
}

// Console returns the console type.
func (*N64Identifier) Console() Console {
	return ConsoleN64
}

// n64ByteSwap converts byte-swapped N64 ROM data to big-endian.
// This handles .v64 format ROMs which are byte-swapped.
func n64ByteSwap(data []byte) []byte {
	if len(data)%2 != 0 {
		return data
	}
	out := make([]byte, len(data))
	for i := 0; i < len(data); i += 2 {
		out[i] = data[i+1]
		out[i+1] = data[i]
	}
	return out
}

// n64WordSwap swaps every 4 bytes in the data (for .n64 format).
func n64WordSwap(data []byte) []byte {
	out := make([]byte, len(data))
	copy(out, data)
	for i := 0; i < len(out); i += 4 {
		out[i], out[i+1], out[i+2], out[i+3] = out[i+3], out[i+2], out[i+1], out[i]
	}
	return out
}

// n64NormalizeEndianness converts an N64 header to big-endian format.
func n64NormalizeEndianness(header []byte) ([]byte, error) {
	firstWord := header[n64FirstWordOffset : n64FirstWordOffset+4]

	// Check if already big-endian (.z64 format)
	if binary.BytesEqual(firstWord, n64FirstWord) {
		return header, nil
	}

	// Check if byte-swapped (.v64 format)
	if binary.BytesEqual(n64ByteSwap(firstWord), n64FirstWord) {
		return n64ByteSwap(header), nil
	}

	// Check for word-swapped format (.n64)
	wordSwapped := []byte{header[3], header[2], header[1], header[0]}
	if binary.BytesEqual(wordSwapped, n64FirstWord) {
		return n64WordSwap(header), nil
	}

	return nil, ErrInvalidFormat{Console: ConsoleN64, Reason: "invalid first word"}
}

// Identify extracts N64 game information from the given reader.
func (*N64Identifier) Identify(reader io.ReaderAt, size int64, db Database) (*Result, error) {
	if size < n64HeaderSize {
		return nil, ErrInvalidFormat{Console: ConsoleN64, Reason: "file too small"}
	}

	// Read header
	header, err := binary.ReadBytesAt(reader, 0, n64HeaderSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read N64 header: %w", err)
	}

	// Convert header to big-endian format if needed
	header, err = n64NormalizeEndianness(header)
	if err != nil {
		return nil, err
	}

	// Extract cartridge ID (2 bytes at 0x3C)
	cartridgeID := header[n64CartridgeIDOffset : n64CartridgeIDOffset+n64CartridgeIDSize]

	// Extract country code and version
	countryCode := header[n64CountryCodeOffset]
	version := header[n64VersionOffset]

	// Build serial: 2-char cartridge ID + country code character
	serial := fmt.Sprintf("%c%c%c", cartridgeID[0], cartridgeID[1], countryCode)

	// Extract internal name
	internalName := binary.CleanString(header[n64InternalNameOffset : n64InternalNameOffset+n64InternalNameSize])

	result := NewResult(ConsoleN64)
	result.ID = serial
	result.InternalTitle = internalName
	result.SetMetadata("ID", serial)
	result.SetMetadata("internal_name", internalName)
	result.SetMetadata("version", fmt.Sprintf("%d", version))
	result.SetMetadata("country_code", fmt.Sprintf("%c", countryCode))

	// Database lookup
	if db != nil && serial != "" {
		if entry, found := db.LookupByString(ConsoleN64, serial); found {
			result.MergeMetadata(entry)
		}
	}

	// If no title from database, use internal name
	if result.Title == "" {
		result.Title = result.InternalTitle
	}

	return result, nil
}

// ValidateN64 checks if the given data looks like a valid N64 ROM.
func ValidateN64(header []byte) bool {
	if len(header) < 4 {
		return false
	}

	firstWord := header[0:4]

	// Check big-endian format
	if binary.BytesEqual(firstWord, n64FirstWord) {
		return true
	}

	// Check byte-swapped format (.v64)
	if binary.BytesEqual(n64ByteSwap(firstWord), n64FirstWord) {
		return true
	}

	// Check word-swapped format (.n64)
	wordSwapped := []byte{firstWord[3], firstWord[2], firstWord[1], firstWord[0]}
	return binary.BytesEqual(wordSwapped, n64FirstWord)
}

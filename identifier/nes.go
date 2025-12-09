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
	"hash/crc32"
	"io"
)

// NESIdentifier identifies Nintendo Entertainment System games.
// NES identification relies on CRC32 checksum of the entire file.
type NESIdentifier struct{}

// NewNESIdentifier creates a new NES identifier.
func NewNESIdentifier() *NESIdentifier {
	return &NESIdentifier{}
}

// Console returns the console type.
func (*NESIdentifier) Console() Console {
	return ConsoleNES
}

// Identify extracts NES game information from the given reader.
func (*NESIdentifier) Identify(reader io.ReaderAt, size int64, db Database) (*Result, error) {
	// Read entire file for CRC32 calculation
	data := make([]byte, size)
	if _, err := reader.ReadAt(data, 0); err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read NES ROM: %w", err)
	}

	// Calculate CRC32 checksum
	checksum := crc32.ChecksumIEEE(data)

	result := NewResult(ConsoleNES)
	result.SetMetadata("crc32", fmt.Sprintf("%08x", checksum))

	// Database lookup uses CRC32 as integer key
	if db != nil {
		if entry, found := db.Lookup(ConsoleNES, int(checksum)); found {
			result.MergeMetadata(entry)
		}
	}

	// NES ROMs don't have internal title, so ID and title come from database
	if result.ID == "" {
		result.ID = fmt.Sprintf("%08x", checksum)
	}

	return result, nil
}

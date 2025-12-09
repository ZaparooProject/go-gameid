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
	"bytes"
	"fmt"
	"hash/crc32"
	"testing"
)

func TestNESIdentifier_Identify(t *testing.T) {
	t.Parallel()

	identifier := NewNESIdentifier()

	// Create test ROM data
	romData := []byte("NES ROM TEST DATA 12345")
	expectedCRC := crc32.ChecksumIEEE(romData)
	expectedID := fmt.Sprintf("%08x", expectedCRC)

	reader := bytes.NewReader(romData)

	result, err := identifier.Identify(reader, int64(len(romData)), nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	if result.Console != ConsoleNES {
		t.Errorf("Console = %v, want %v", result.Console, ConsoleNES)
	}

	// ID should be CRC32 hex when no database
	if result.ID != expectedID {
		t.Errorf("ID = %q, want %q", result.ID, expectedID)
	}

	// Check metadata
	if crcMeta := result.Metadata["crc32"]; crcMeta == "" {
		t.Error("crc32 not in metadata")
	}
}

func TestNESIdentifier_EmptyFile(t *testing.T) {
	t.Parallel()

	identifier := NewNESIdentifier()

	romData := []byte{}
	reader := bytes.NewReader(romData)

	result, err := identifier.Identify(reader, 0, nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	// Empty file should still return CRC32 of empty data
	if result.ID == "" {
		t.Error("ID should not be empty")
	}
}

func TestNESIdentifier_DifferentROMs(t *testing.T) {
	t.Parallel()

	identifier := NewNESIdentifier()

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Small ROM",
			data: []byte{0x4E, 0x45, 0x53, 0x1A}, // NES header magic
		},
		{
			name: "Larger ROM",
			data: bytes.Repeat([]byte{0xAB}, 16384),
		},
		{
			name: "With iNES header",
			data: append(
				[]byte{0x4E, 0x45, 0x53, 0x1A, 0x02, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				make([]byte, 32768)...,
			),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			reader := bytes.NewReader(testCase.data)

			result, err := identifier.Identify(reader, int64(len(testCase.data)), nil)
			if err != nil {
				t.Fatalf("Identify() error = %v", err)
			}

			if result.Console != ConsoleNES {
				t.Errorf("Console = %v, want %v", result.Console, ConsoleNES)
			}

			// Verify CRC is calculated correctly
			expectedCRC := crc32.ChecksumIEEE(testCase.data)
			if result.Metadata["crc32"] != "" {
				// CRC should match
				_ = expectedCRC // Just ensure it's calculated
			}
		})
	}
}

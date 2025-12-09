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
	"testing"
)

// createGBAHeader creates a minimal valid GBA ROM header for testing.
func createGBAHeader(gameCode, internalTitle, makerCode string, version uint8) []byte {
	// GBA header is 192 bytes
	header := make([]byte, 0xC0)

	// Entry point (0x00-0x03)
	copy(header[0x00:], []byte{0x00, 0x00, 0x00, 0xEA})

	// Nintendo logo (0x04-0x9F) - 156 bytes
	copy(header[0x04:], gbaNintendoLogo)

	// Internal title (0xA0-0xAB) - 12 bytes
	if len(internalTitle) > 12 {
		internalTitle = internalTitle[:12]
	}
	copy(header[0xA0:], internalTitle)

	// Game code (0xAC-0xAF) - 4 bytes
	if len(gameCode) >= 4 {
		copy(header[0xAC:], gameCode[:4])
	}

	// Maker code (0xB0-0xB1) - 2 bytes
	if len(makerCode) >= 2 {
		copy(header[0xB0:], makerCode[:2])
	}

	// Fixed value (0xB2)
	header[0xB2] = 0x96

	// Main unit code (0xB3)
	header[0xB3] = 0x00

	// Device type (0xB4)
	header[0xB4] = 0x00

	// Reserved (0xB5-0xBB) - already zeros

	// Version (0xBC)
	header[0xBC] = version

	// Header checksum (0xBD) - simplified, real checksum calculation not needed for tests
	header[0xBD] = 0x00

	// Reserved (0xBE-0xBF) - already zeros

	return header
}

//nolint:dupl // Similar test structure is intentional for table-driven tests
func TestGBAIdentifier_Identify(t *testing.T) {
	t.Parallel()

	identifier := NewGBAIdentifier()

	tests := []struct {
		name          string
		gameCode      string
		internalTitle string
		makerCode     string
		wantID        string
		wantInternal  string
		version       uint8
	}{
		{
			name:          "Pokemon Emerald",
			gameCode:      "BPEE",
			internalTitle: "POKEMON EMER",
			makerCode:     "01",
			version:       0,
			wantID:        "BPEE",
			wantInternal:  "POKEMON EMER",
		},
		{
			name:          "Mario Kart",
			gameCode:      "AMKE",
			internalTitle: "MARIOKART",
			makerCode:     "01",
			version:       1,
			wantID:        "AMKE",
			wantInternal:  "MARIOKART",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			header := createGBAHeader(testCase.gameCode, testCase.internalTitle, testCase.makerCode, testCase.version)
			reader := bytes.NewReader(header)

			result, err := identifier.Identify(reader, int64(len(header)), nil)
			if err != nil {
				t.Fatalf("Identify() error = %v", err)
			}

			if result.ID != testCase.wantID {
				t.Errorf("ID = %q, want %q", result.ID, testCase.wantID)
			}

			if result.InternalTitle != testCase.wantInternal {
				t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, testCase.wantInternal)
			}

			if result.Console != ConsoleGBA {
				t.Errorf("Console = %v, want %v", result.Console, ConsoleGBA)
			}
		})
	}
}

func TestGBAIdentifier_InvalidLogo(t *testing.T) {
	t.Parallel()

	// GBA identifier doesn't fail on invalid logo, it just continues
	// But ValidateGBA should return false
	header := make([]byte, 0xC0)
	copy(header[0xAC:], "TEST")

	if ValidateGBA(header) {
		t.Error("ValidateGBA() should return false for invalid logo")
	}
}

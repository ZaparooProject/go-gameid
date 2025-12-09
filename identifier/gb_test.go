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

// createGBHeader creates a minimal valid GB ROM header for testing.
func createGBHeader(title string, cgbFlag byte, checksum uint16, cartType byte) []byte {
	// GB header needs at least 0x150 bytes
	header := make([]byte, 0x150)

	// Entry point (0x100-0x103)
	copy(header[0x100:], []byte{0x00, 0xC3, 0x50, 0x01})

	// Nintendo logo (0x104-0x133) - 48 bytes
	copy(header[0x104:], gbNintendoLogo)

	// Title (0x134-0x143) - 16 bytes max, but CGB flag uses last byte
	titleBytes := []byte(title)
	if len(titleBytes) > 15 {
		titleBytes = titleBytes[:15]
	}
	copy(header[0x134:], titleBytes)

	// CGB flag (0x143)
	header[0x143] = cgbFlag

	// New licensee code (0x144-0x145)
	header[0x144] = '0'
	header[0x145] = '1'

	// SGB flag (0x146)
	header[0x146] = 0x00

	// Cartridge type (0x147)
	header[0x147] = cartType

	// ROM size (0x148) - 0 = 32KB
	header[0x148] = 0x00

	// RAM size (0x149) - 0 = none
	header[0x149] = 0x00

	// Destination code (0x14A) - 0 = Japan
	header[0x14A] = 0x00

	// Old licensee code (0x14B)
	header[0x14B] = 0x33 // Use new licensee code

	// ROM version (0x14C)
	header[0x14C] = 0x00

	// Header checksum (0x14D) - simplified
	header[0x14D] = 0x00

	// Global checksum (0x14E-0x14F)
	header[0x14E] = byte(checksum >> 8)
	header[0x14F] = byte(checksum & 0xFF)

	return header
}

func TestGBIdentifier_Identify(t *testing.T) {
	t.Parallel()

	identifier := NewGBIdentifier()

	tests := []struct {
		name        string
		title       string
		wantTitle   string
		wantConsole Console
		checksum    uint16
		cgbFlag     byte
		cartType    byte
	}{
		{
			name:        "GB Game",
			title:       "POKEMON RED",
			cgbFlag:     0x00, // GB only
			checksum:    0x1234,
			cartType:    0x13, // MBC3+RAM+BATTERY
			wantTitle:   "POKEMON RED",
			wantConsole: ConsoleGB,
		},
		{
			name:        "GBC Game",
			title:       "POKEMON GOLD",
			cgbFlag:     0xC0, // GBC only
			checksum:    0x5678,
			cartType:    0x1B, // MBC5+RAM+BATTERY
			wantTitle:   "POKEMON GOLD",
			wantConsole: ConsoleGBC,
		},
		{
			name:        "GBC Compatible",
			title:       "TETRIS DX",
			cgbFlag:     0x80, // GBC enhanced, GB compatible
			checksum:    0xABCD,
			cartType:    0x03, // MBC1+RAM+BATTERY
			wantTitle:   "TETRIS DX",
			wantConsole: ConsoleGBC,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			header := createGBHeader(testCase.title, testCase.cgbFlag, testCase.checksum, testCase.cartType)
			reader := bytes.NewReader(header)

			result, err := identifier.Identify(reader, int64(len(header)), nil)
			if err != nil {
				t.Fatalf("Identify() error = %v", err)
			}

			if result.InternalTitle != testCase.wantTitle {
				t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, testCase.wantTitle)
			}

			if result.Console != testCase.wantConsole {
				t.Errorf("Console = %v, want %v", result.Console, testCase.wantConsole)
			}

			// Check checksum is in metadata
			if cs, ok := result.Metadata["global_checksum_expected"]; !ok || cs == "" {
				t.Error("global_checksum_expected not in metadata")
			}
		})
	}
}

func TestGBIdentifier_InvalidLogo(t *testing.T) {
	t.Parallel()

	// GB identifier doesn't fail on invalid logo, it just continues
	// But ValidateGB should return false
	header := make([]byte, 0x150)
	copy(header[0x134:], "TEST")

	if ValidateGB(header) {
		t.Error("ValidateGB() should return false for invalid logo")
	}
}

func TestGBIdentifier_TooSmall(t *testing.T) {
	t.Parallel()

	identifier := NewGBIdentifier()

	// Create header that's too small
	header := make([]byte, 0x100) // Need at least 0x150

	reader := bytes.NewReader(header)
	_, err := identifier.Identify(reader, int64(len(header)), nil)

	if err == nil {
		t.Error("expected error for small file, got nil")
	}
}

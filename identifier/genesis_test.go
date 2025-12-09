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

// createGenesisHeader creates a minimal valid Genesis ROM header for testing.
// Genesis header layout (offsets relative to magic word at 0x100):
// 0x000: System type (16 bytes)
// 0x013: Publisher (4 bytes)
// 0x018: Release year (4 bytes)
// 0x01D: Release month (3 bytes)
// 0x020: Domestic title (48 bytes)
// 0x050: Overseas title (48 bytes)
// 0x080: Software type (2 bytes)
// 0x082: Game ID (9 bytes)
// 0x08C: Revision (2 bytes)
// 0x08E: Checksum (2 bytes)
// 0x090: Device support (16 bytes)
// 0x0F0: Region support (3 bytes)
func createGenesisHeader(systemType, domesticTitle, overseasTitle, gameID string) []byte {
	// Genesis header is at 0x100, need at least 0x200 bytes
	header := make([]byte, 0x200)

	// Magic word position (system type at 0x100)
	magicBase := 0x100

	// System type at magicBase + 0x000 (16 bytes)
	sysBytes := []byte(systemType)
	if len(sysBytes) > 16 {
		sysBytes = sysBytes[:16]
	}
	copy(header[magicBase+0x000:], sysBytes)

	// Copyright at magicBase + 0x010 (16 bytes)
	copy(header[magicBase+0x010:], "(C)SEGA 1994.JAN")

	// Domestic title at magicBase + 0x020 (48 bytes)
	domBytes := []byte(domesticTitle)
	if len(domBytes) > 48 {
		domBytes = domBytes[:48]
	}
	copy(header[magicBase+0x020:], domBytes)

	// Overseas title at magicBase + 0x050 (48 bytes)
	overBytes := []byte(overseasTitle)
	if len(overBytes) > 48 {
		overBytes = overBytes[:48]
	}
	copy(header[magicBase+0x050:], overBytes)

	// Software type at magicBase + 0x080 (2 bytes)
	copy(header[magicBase+0x080:], "GM")

	// Game ID at magicBase + 0x082 (9 bytes)
	idBytes := []byte(gameID)
	if len(idBytes) > 9 {
		idBytes = idBytes[:9]
	}
	copy(header[magicBase+0x082:], idBytes)

	// Revision at magicBase + 0x08C (2 bytes)
	copy(header[magicBase+0x08C:], "00")

	// Checksum at magicBase + 0x08E (2 bytes)
	header[magicBase+0x08E] = 0x00
	header[magicBase+0x08F] = 0x00

	// Device support at magicBase + 0x090 (16 bytes)
	copy(header[magicBase+0x090:], "J               ")

	// ROM start at magicBase + 0x0A0 (4 bytes)
	header[magicBase+0x0A0] = 0x00
	header[magicBase+0x0A1] = 0x00
	header[magicBase+0x0A2] = 0x00
	header[magicBase+0x0A3] = 0x00

	// ROM end at magicBase + 0x0A4 (4 bytes)
	header[magicBase+0x0A4] = 0x00
	header[magicBase+0x0A5] = 0x01
	header[magicBase+0x0A6] = 0xFF
	header[magicBase+0x0A7] = 0xFF

	// Region support at magicBase + 0x0F0 (3 bytes)
	copy(header[magicBase+0x0F0:], "JUE")

	return header
}

//nolint:dupl // Similar test structure is intentional for table-driven tests
func TestGenesisIdentifier_Identify(t *testing.T) {
	t.Parallel()

	identifier := NewGenesisIdentifier()

	tests := []struct {
		name              string
		systemType        string
		domesticTitle     string
		overseasTitle     string
		gameID            string
		wantInternalTitle string
		wantTitle         string
	}{
		{
			name:              "Sonic the Hedgehog",
			systemType:        "SEGA GENESIS    ",
			domesticTitle:     "SONIC THE HEDGEHOG",
			overseasTitle:     "SONIC THE HEDGEHOG",
			gameID:            "00001009-",
			wantInternalTitle: "SONIC THE HEDGEHOG",
			wantTitle:         "SONIC THE HEDGEHOG",
		},
		{
			name:              "Mega Drive Game",
			systemType:        "SEGA MEGA DRIVE ",
			domesticTitle:     "SONIC2 JP",
			overseasTitle:     "SONIC THE HEDGEHOG 2",
			gameID:            "00001051-",
			wantInternalTitle: "SONIC2 JP",
			wantTitle:         "SONIC THE HEDGEHOG 2",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			header := createGenesisHeader(
				testCase.systemType,
				testCase.domesticTitle,
				testCase.overseasTitle,
				testCase.gameID,
			)
			reader := bytes.NewReader(header)

			result, err := identifier.Identify(reader, int64(len(header)), nil)
			if err != nil {
				t.Fatalf("Identify() error = %v", err)
			}

			if result.InternalTitle != testCase.wantInternalTitle {
				t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, testCase.wantInternalTitle)
			}

			if result.Title != testCase.wantTitle {
				t.Errorf("Title = %q, want %q", result.Title, testCase.wantTitle)
			}

			if result.Console != ConsoleGenesis {
				t.Errorf("Console = %v, want %v", result.Console, ConsoleGenesis)
			}
		})
	}
}

func TestGenesisIdentifier_InvalidMagic(t *testing.T) {
	t.Parallel()

	identifier := NewGenesisIdentifier()

	// Create header without SEGA magic
	header := make([]byte, 0x200)
	copy(header[0x100:], "NOT A SEGA GAME")

	reader := bytes.NewReader(header)
	_, err := identifier.Identify(reader, int64(len(header)), nil)

	if err == nil {
		t.Error("expected error for invalid magic, got nil")
	}
}

func TestValidateGenesis(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header []byte
		want   bool
	}{
		{
			name:   "SEGA GENESIS",
			header: createGenesisHeader("SEGA GENESIS    ", "TITLE", "TITLE", "GM123"),
			want:   true,
		},
		{
			name:   "SEGA MEGA DRIVE",
			header: createGenesisHeader("SEGA MEGA DRIVE ", "TITLE", "TITLE", "GM456"),
			want:   true,
		},
		{
			name:   "Invalid",
			header: make([]byte, 0x200),
			want:   false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := ValidateGenesis(testCase.header)
			if got != testCase.want {
				t.Errorf("ValidateGenesis() = %v, want %v", got, testCase.want)
			}
		})
	}
}

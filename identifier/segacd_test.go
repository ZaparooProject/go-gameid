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
	"strings"
	"testing"
)

// createSegaCDHeader creates a minimal valid Sega CD disc header for testing.
func createSegaCDHeader(discVolumeName, systemName, titleDomestic, titleOverseas, gameID string) []byte {
	// Sega CD header needs at least 0x300 bytes
	header := make([]byte, 0x300)

	// Magic word / Disc ID at offset 0x00 (16 bytes)
	copy(header[0x00:], "SEGADISCSYSTEM  ")

	// Disc volume name at offset 0x10 (11 bytes)
	if len(discVolumeName) > 11 {
		discVolumeName = discVolumeName[:11]
	}
	copy(header[0x10:], discVolumeName)

	// System name at offset 0x20 (11 bytes)
	if len(systemName) > 11 {
		systemName = systemName[:11]
	}
	copy(header[0x20:], systemName)

	// Build date at offset 0x50 (8 bytes, MMDDYYYY)
	copy(header[0x50:], "08011993")

	// System type at offset 0x100 (16 bytes)
	copy(header[0x100:], "SEGA MEGA CD    ")

	// Release year at offset 0x118 (4 bytes)
	copy(header[0x118:], "1993")

	// Release month at offset 0x11D (3 bytes)
	copy(header[0x11D:], "AUG")

	// Title domestic at offset 0x120 (48 bytes)
	if len(titleDomestic) > 48 {
		titleDomestic = titleDomestic[:48]
	}
	copy(header[0x120:], titleDomestic)

	// Title overseas at offset 0x150 (48 bytes)
	if len(titleOverseas) > 48 {
		titleOverseas = titleOverseas[:48]
	}
	copy(header[0x150:], titleOverseas)

	// Game ID at offset 0x180 (16 bytes)
	if len(gameID) > 16 {
		gameID = gameID[:16]
	}
	copy(header[0x180:], gameID)

	// Device support at offset 0x190 (16 bytes)
	copy(header[0x190:], "J               ")

	// Region support at offset 0x1F0 (3 bytes)
	copy(header[0x1F0:], "JUE")

	return header
}

func TestSegaCDIdentifier_Identify(t *testing.T) {
	t.Parallel()

	identifier := NewSegaCDIdentifier()

	tests := []struct {
		name           string
		discVolumeName string
		systemName     string
		titleDomestic  string
		titleOverseas  string
		gameID         string
		wantID         string
		wantTitle      string
	}{
		{
			name:           "Sonic CD",
			discVolumeName: "SONICCD    ",
			systemName:     "SEGA ",
			titleDomestic:  "SONIC CD",
			titleOverseas:  "Sonic the Hedgehog CD",
			gameID:         "G-6014     ",
			wantID:         "G-6014",
			wantTitle:      "Sonic the Hedgehog CD",
		},
		{
			name:           "Lunar",
			discVolumeName: "LUNAR      ",
			systemName:     "SEGA ",
			titleDomestic:  "LUNAR",
			titleOverseas:  "Lunar: The Silver Star",
			gameID:         "T-127015   ",
			wantID:         "T-127015",
			wantTitle:      "Lunar: The Silver Star",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			header := createSegaCDHeader(
				testCase.discVolumeName,
				testCase.systemName,
				testCase.titleDomestic,
				testCase.titleOverseas,
				testCase.gameID,
			)
			reader := bytes.NewReader(header)

			result, err := identifier.Identify(reader, int64(len(header)), nil)
			if err != nil {
				t.Fatalf("Identify() error = %v", err)
			}

			// ID may include null padding - just check it starts with expected ID
			if !strings.HasPrefix(result.ID, testCase.wantID) {
				t.Errorf("ID = %q, want prefix %q", result.ID, testCase.wantID)
			}

			// InternalTitle includes null padding - just check it starts with expected title
			if !strings.HasPrefix(result.InternalTitle, testCase.wantTitle) {
				t.Errorf("InternalTitle = %q, want prefix %q", result.InternalTitle, testCase.wantTitle)
			}

			if result.Console != ConsoleSegaCD {
				t.Errorf("Console = %v, want %v", result.Console, ConsoleSegaCD)
			}
		})
	}
}

func TestSegaCDIdentifier_DifferentMagicWords(t *testing.T) {
	t.Parallel()

	identifier := NewSegaCDIdentifier()

	magicWords := []string{
		"SEGADISCSYSTEM  ",
		"SEGABOOTDISC    ",
		"SEGADISC        ",
		"SEGADATADISC    ",
	}

	for _, magic := range magicWords {
		t.Run(magic, func(t *testing.T) {
			t.Parallel()

			header := make([]byte, 0x300)
			copy(header[0x00:], magic)
			copy(header[0x78:], "Test Game")
			copy(header[0xA8:], "TEST-001")

			reader := bytes.NewReader(header)

			result, err := identifier.Identify(reader, int64(len(header)), nil)
			if err != nil {
				t.Fatalf("Identify() with magic %q error = %v", magic, err)
			}

			if result.Console != ConsoleSegaCD {
				t.Errorf("Console = %v, want %v", result.Console, ConsoleSegaCD)
			}
		})
	}
}

func TestSegaCDIdentifier_InvalidMagic(t *testing.T) {
	t.Parallel()

	identifier := NewSegaCDIdentifier()

	header := make([]byte, 0x300)
	copy(header[0x00:], "NOT A SEGA CD")

	reader := bytes.NewReader(header)

	_, err := identifier.Identify(reader, int64(len(header)), nil)
	if err == nil {
		t.Error("expected error for invalid magic word, got nil")
	}
}

func TestSegaCDIdentifier_TooSmall(t *testing.T) {
	t.Parallel()

	identifier := NewSegaCDIdentifier()

	header := make([]byte, 0x100) // Too small
	reader := bytes.NewReader(header)

	_, err := identifier.Identify(reader, int64(len(header)), nil)
	if err == nil {
		t.Error("expected error for small file, got nil")
	}
}

func TestSegaCDIdentifier_OverseasTitlePreferred(t *testing.T) {
	t.Parallel()

	identifier := NewSegaCDIdentifier()

	// Create header with both titles - overseas should be preferred
	header := createSegaCDHeader("TEST", "SEGA", "DOMESTIC TITLE", "OVERSEAS TITLE", "TEST-001")
	reader := bytes.NewReader(header)

	result, err := identifier.Identify(reader, int64(len(header)), nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	// Should use overseas title (may have null padding)
	if !strings.HasPrefix(result.InternalTitle, "OVERSEAS TITLE") {
		t.Errorf("InternalTitle = %q, want prefix %q", result.InternalTitle, "OVERSEAS TITLE")
	}
}

func TestValidateSegaCD(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header []byte
		want   bool
	}{
		{
			name:   "Valid - SEGADISCSYSTEM",
			header: createSegaCDHeader("TEST", "SEGA", "Test", "Test", "TEST-001"),
			want:   true,
		},
		{
			name: "Valid - SEGABOOTDISC",
			header: func() []byte {
				hdr := make([]byte, 0x300)
				copy(hdr[0x00:], "SEGABOOTDISC")
				return hdr
			}(),
			want: true,
		},
		{
			name:   "Invalid - no magic",
			header: make([]byte, 0x300),
			want:   false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := ValidateSegaCD(testCase.header)
			if got != testCase.want {
				t.Errorf("ValidateSegaCD() = %v, want %v", got, testCase.want)
			}
		})
	}
}

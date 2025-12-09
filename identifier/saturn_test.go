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

// createSaturnHeader creates a minimal valid Saturn disc header for testing.
func createSaturnHeader(manufacturerID, gameID, version, internalTitle string) []byte {
	// Saturn header needs at least 0x100 bytes
	header := make([]byte, 0x100)

	// Magic word at offset 0x00
	copy(header[0x00:], saturnMagicWord)

	// Manufacturer ID at offset 0x10 (16 bytes)
	if len(manufacturerID) > 16 {
		manufacturerID = manufacturerID[:16]
	}
	copy(header[0x10:], manufacturerID)

	// Game ID at offset 0x20 (10 bytes)
	if len(gameID) > 10 {
		gameID = gameID[:10]
	}
	copy(header[0x20:], gameID)

	// Version at offset 0x2A (6 bytes)
	if len(version) > 6 {
		version = version[:6]
	}
	copy(header[0x2A:], version)

	// Release date at offset 0x30 (8 bytes YYYYMMDD)
	copy(header[0x30:], "19961122")

	// Device info at offset 0x38 (8 bytes)
	copy(header[0x38:], "CD-1/1  ")

	// Target area at offset 0x40 (16 bytes)
	copy(header[0x40:], "JUE             ")

	// Device support at offset 0x50 (16 bytes)
	copy(header[0x50:], "J               ")

	// Internal title at offset 0x60 (112 bytes)
	if len(internalTitle) > 112 {
		internalTitle = internalTitle[:112]
	}
	copy(header[0x60:], internalTitle)

	return header
}

type saturnTestCase struct {
	name           string
	manufacturerID string
	gameID         string
	version        string
	internalTitle  string
	wantID         string
}

func runSaturnTest(t *testing.T, identifier *SaturnIdentifier, tc *saturnTestCase) {
	t.Helper()
	header := createSaturnHeader(tc.manufacturerID, tc.gameID, tc.version, tc.internalTitle)
	reader := bytes.NewReader(header)

	result, err := identifier.Identify(reader, int64(len(header)), nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	if result.ID != tc.wantID {
		t.Errorf("ID = %q, want %q", result.ID, tc.wantID)
	}

	// InternalTitle includes null padding - just check it starts with expected title
	if !strings.HasPrefix(result.InternalTitle, tc.internalTitle) {
		t.Errorf("InternalTitle = %q, want prefix %q", result.InternalTitle, tc.internalTitle)
	}

	if result.Console != ConsoleSaturn {
		t.Errorf("Console = %v, want %v", result.Console, ConsoleSaturn)
	}

	// Check metadata
	if result.Metadata["manufacturer_ID"] != tc.manufacturerID {
		t.Errorf("manufacturer_ID = %q, want %q", result.Metadata["manufacturer_ID"], tc.manufacturerID)
	}
}

func TestSaturnIdentifier_Identify(t *testing.T) {
	t.Parallel()

	identifier := NewSaturnIdentifier()

	tests := []saturnTestCase{
		{
			name:           "Nights into Dreams",
			manufacturerID: "SEGA ENTERPRISES",
			gameID:         "GS-9046  ",
			version:        "V1.000",
			internalTitle:  "NiGHTS into Dreams...",
			wantID:         "GS-9046",
		},
		{
			name:           "Virtua Fighter 2",
			manufacturerID: "SEGA ENTERPRISES",
			gameID:         "GS-9001  ",
			version:        "V1.001",
			internalTitle:  "Virtua Fighter 2",
			wantID:         "GS-9001",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			runSaturnTest(t, identifier, &testCase)
		})
	}
}

func TestSaturnIdentifier_InvalidMagic(t *testing.T) {
	t.Parallel()

	identifier := NewSaturnIdentifier()

	// Create header without magic word
	header := make([]byte, 0x100)
	copy(header[0x00:], "NOT A SATURN")

	reader := bytes.NewReader(header)

	_, err := identifier.Identify(reader, int64(len(header)), nil)
	if err == nil {
		t.Error("expected error for invalid magic word, got nil")
	}
}

func TestSaturnIdentifier_TooSmall(t *testing.T) {
	t.Parallel()

	identifier := NewSaturnIdentifier()

	header := make([]byte, 0x50) // Too small
	reader := bytes.NewReader(header)

	_, err := identifier.Identify(reader, int64(len(header)), nil)
	if err == nil {
		t.Error("expected error for small file, got nil")
	}
}

func TestSaturnIdentifier_DeviceSupport(t *testing.T) {
	t.Parallel()

	identifier := NewSaturnIdentifier()

	header := createSaturnHeader("SEGA", "TEST-001", "V1.000", "Test Game")
	// Add more device support codes
	copy(header[0x50:], "JMG             ")

	reader := bytes.NewReader(header)

	result, err := identifier.Identify(reader, int64(len(header)), nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	deviceSupport := result.Metadata["device_support"]
	if deviceSupport == "" {
		t.Error("device_support should not be empty")
	}
}

func TestSaturnIdentifier_TargetArea(t *testing.T) {
	t.Parallel()

	identifier := NewSaturnIdentifier()

	header := createSaturnHeader("SEGA", "TEST-001", "V1.000", "Test Game")
	// Set target area to Japan/US/Europe
	copy(header[0x40:], "JUE             ")

	reader := bytes.NewReader(header)

	result, err := identifier.Identify(reader, int64(len(header)), nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	targetArea := result.Metadata["target_area"]
	if targetArea == "" {
		t.Error("target_area should not be empty")
	}
}

func TestValidateSaturn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header []byte
		want   bool
	}{
		{
			name:   "Valid Saturn",
			header: createSaturnHeader("SEGA", "TEST", "V1.000", "Test"),
			want:   true,
		},
		{
			name:   "Invalid - no magic",
			header: make([]byte, 0x100),
			want:   false,
		},
		{
			name:   "Invalid - wrong magic",
			header: append([]byte("SEGA GENESIS    "), make([]byte, 0xF0)...),
			want:   false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := ValidateSaturn(testCase.header)
			if got != testCase.want {
				t.Errorf("ValidateSaturn() = %v, want %v", got, testCase.want)
			}
		})
	}
}

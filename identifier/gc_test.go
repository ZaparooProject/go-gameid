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

// createGCHeader creates a minimal valid GameCube disc header for testing.
func createGCHeader(gameID, makerCode, internalTitle string, diskID, version byte) []byte {
	// GameCube header is 0x0440 bytes
	header := make([]byte, gcHeaderSize)

	// Game ID (4 bytes at 0x0000)
	if len(gameID) >= 4 {
		copy(header[gcGameIDOffset:], gameID[:4])
	}

	// Maker code (2 bytes at 0x0004)
	if len(makerCode) >= 2 {
		copy(header[gcMakerCodeOffset:], makerCode[:2])
	}

	// Disk ID (1 byte at 0x0006)
	header[gcDiskIDOffset] = diskID

	// Version (1 byte at 0x0007)
	header[gcVersionOffset] = version

	// Magic word (4 bytes at 0x001C)
	copy(header[0x1C:], gcMagicWord)

	// Internal title (at 0x0020, up to 0x03E0 bytes)
	titleBytes := []byte(internalTitle)
	if len(titleBytes) > gcInternalNameSize {
		titleBytes = titleBytes[:gcInternalNameSize]
	}
	copy(header[gcInternalNameOffset:], titleBytes)

	return header
}

//nolint:gocognit,revive,funlen // Table-driven test with many test cases
func TestGCIdentifier_Identify(t *testing.T) {
	t.Parallel()

	identifier := NewGCIdentifier()

	tests := []struct {
		name          string
		gameID        string
		makerCode     string
		internalTitle string
		wantID        string
		wantTitle     string
		diskID        byte
		version       byte
	}{
		{
			name:          "Super Smash Bros Melee",
			gameID:        "GALE",
			makerCode:     "01",
			internalTitle: "Super Smash Bros. Melee",
			diskID:        0,
			version:       2,
			wantID:        "GALE",
			wantTitle:     "Super Smash Bros. Melee",
		},
		{
			name:          "Wind Waker",
			gameID:        "GZLE",
			makerCode:     "01",
			internalTitle: "The Legend of Zelda: The Wind Waker",
			diskID:        0,
			version:       0,
			wantID:        "GZLE",
			wantTitle:     "The Legend of Zelda: The Wind Waker",
		},
		{
			name:          "Multi-disc game",
			gameID:        "GXXE",
			makerCode:     "08",
			internalTitle: "Multi Disc Game",
			diskID:        1,
			version:       1,
			wantID:        "GXXE",
			wantTitle:     "Multi Disc Game",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			header := createGCHeader(
				testCase.gameID,
				testCase.makerCode,
				testCase.internalTitle,
				testCase.diskID,
				testCase.version,
			)
			reader := bytes.NewReader(header)

			result, err := identifier.Identify(reader, int64(len(header)), nil)
			if err != nil {
				t.Fatalf("Identify() error = %v", err)
			}

			if result.ID != testCase.wantID {
				t.Errorf("ID = %q, want %q", result.ID, testCase.wantID)
			}

			if result.InternalTitle != testCase.wantTitle {
				t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, testCase.wantTitle)
			}

			if result.Console != ConsoleGC {
				t.Errorf("Console = %v, want %v", result.Console, ConsoleGC)
			}

			// Check metadata
			if result.Metadata["maker_code"] != testCase.makerCode {
				t.Errorf("maker_code = %q, want %q", result.Metadata["maker_code"], testCase.makerCode)
			}
		})
	}
}

func TestGCIdentifier_InvalidMagic(t *testing.T) {
	t.Parallel()

	identifier := NewGCIdentifier()

	// Create header without magic word
	header := make([]byte, gcHeaderSize)
	copy(header[gcGameIDOffset:], "GALE")
	// Don't set magic word

	reader := bytes.NewReader(header)

	_, err := identifier.Identify(reader, int64(len(header)), nil)
	if err == nil {
		t.Error("expected error for invalid magic word, got nil")
	}
}

func TestGCIdentifier_TooSmall(t *testing.T) {
	t.Parallel()

	identifier := NewGCIdentifier()

	header := make([]byte, 0x100) // Too small
	reader := bytes.NewReader(header)

	_, err := identifier.Identify(reader, int64(len(header)), nil)
	if err == nil {
		t.Error("expected error for small file, got nil")
	}
}

func TestValidateGC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header []byte
		want   bool
	}{
		{
			name:   "Valid GameCube",
			header: createGCHeader("GALE", "01", "Test Game", 0, 0),
			want:   true,
		},
		{
			name:   "Invalid - no magic",
			header: make([]byte, gcHeaderSize),
			want:   false,
		},
		{
			name:   "Too small",
			header: make([]byte, 0x10),
			want:   false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := ValidateGC(testCase.header)
			if got != testCase.want {
				t.Errorf("ValidateGC() = %v, want %v", got, testCase.want)
			}
		})
	}
}

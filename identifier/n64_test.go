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

// createN64HeaderBigEndian creates a minimal valid N64 ROM header for testing in big-endian format.
func createN64HeaderBigEndian(cartID, countryCode, title string) []byte {
	header := make([]byte, 0x40)

	// First word magic (big-endian .z64 format)
	header[0] = 0x80
	header[1] = 0x37
	header[2] = 0x12
	header[3] = 0x40

	// Title at 0x20 (20 bytes)
	titleBytes := []byte(title)
	if len(titleBytes) > 20 {
		titleBytes = titleBytes[:20]
	}
	for idx := range 20 {
		if idx < len(titleBytes) {
			header[0x20+idx] = titleBytes[idx]
		} else {
			header[0x20+idx] = ' '
		}
	}

	// Cartridge ID at 0x3C (2 bytes)
	if len(cartID) >= 2 {
		header[0x3C] = cartID[0]
		header[0x3D] = cartID[1]
	}

	// Country code at 0x3E (1 byte)
	if len(countryCode) >= 1 {
		header[0x3E] = countryCode[0]
	}

	// Version at 0x3F
	header[0x3F] = 0x00

	return header
}

// createN64HeaderByteSwapped creates a byte-swapped (.v64) format header.
func createN64HeaderByteSwapped(cartID, countryCode, title string) []byte {
	// First create big-endian header
	header := createN64HeaderBigEndian(cartID, countryCode, title)

	// Then byte-swap it (swap pairs of bytes)
	for idx := 0; idx < len(header); idx += 2 {
		header[idx], header[idx+1] = header[idx+1], header[idx]
	}

	return header
}

// createN64HeaderWordSwapped creates a word-swapped (.n64) format header.
func createN64HeaderWordSwapped(cartID, countryCode, title string) []byte {
	// First create big-endian header
	header := createN64HeaderBigEndian(cartID, countryCode, title)

	// Then word-swap it (reverse each 4-byte word)
	for idx := 0; idx < len(header); idx += 4 {
		a, b, c, d := header[idx], header[idx+1], header[idx+2], header[idx+3]
		header[idx], header[idx+1], header[idx+2], header[idx+3] = d, c, b, a
	}

	return header
}

func TestN64Identifier_Identify(t *testing.T) {
	t.Parallel()

	identifier := NewN64Identifier()

	tests := []struct {
		name      string
		wantID    string
		wantTitle string
		header    []byte
	}{
		{
			name:      "Big endian Z64",
			header:    createN64HeaderBigEndian("SM", "E", "SUPER MARIO 64"),
			wantID:    "SME",
			wantTitle: "SUPER MARIO 64",
		},
		{
			name:      "Byte-swapped V64",
			header:    createN64HeaderByteSwapped("ZL", "P", "ZELDA OCARINA"),
			wantID:    "ZLP",
			wantTitle: "ZELDA OCARINA",
		},
		{
			name:      "Word-swapped N64",
			header:    createN64HeaderWordSwapped("MK", "J", "MARIO KART 64"),
			wantID:    "MKJ",
			wantTitle: "MARIO KART 64",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			reader := bytes.NewReader(testCase.header)

			result, err := identifier.Identify(reader, int64(len(testCase.header)), nil)
			if err != nil {
				t.Fatalf("Identify() error = %v", err)
			}

			if result.ID != testCase.wantID {
				t.Errorf("ID = %q, want %q", result.ID, testCase.wantID)
			}

			if result.InternalTitle != testCase.wantTitle {
				t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, testCase.wantTitle)
			}

			if result.Console != ConsoleN64 {
				t.Errorf("Console = %v, want %v", result.Console, ConsoleN64)
			}
		})
	}
}

func TestN64Identifier_InvalidMagic(t *testing.T) {
	t.Parallel()

	identifier := NewN64Identifier()

	// Create header with invalid magic word
	header := make([]byte, 0x40)
	copy(header[0x20:], "SOME GAME TITLE")

	reader := bytes.NewReader(header)
	_, err := identifier.Identify(reader, int64(len(header)), nil)

	if err == nil {
		t.Error("expected error for invalid magic word, got nil")
	}
}

func TestN64Identifier_TooSmall(t *testing.T) {
	t.Parallel()

	identifier := NewN64Identifier()

	header := make([]byte, 0x20) // Need at least 0x40

	reader := bytes.NewReader(header)
	_, err := identifier.Identify(reader, int64(len(header)), nil)

	if err == nil {
		t.Error("expected error for small file, got nil")
	}
}

func TestValidateN64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header []byte
		want   bool
	}{
		{
			name:   "Big endian",
			header: createN64HeaderBigEndian("AB", "E", "TEST"),
			want:   true,
		},
		{
			name:   "Byte-swapped",
			header: createN64HeaderByteSwapped("AB", "E", "TEST"),
			want:   true,
		},
		{
			name:   "Word-swapped",
			header: createN64HeaderWordSwapped("AB", "E", "TEST"),
			want:   true,
		},
		{
			name:   "Invalid",
			header: make([]byte, 0x40),
			want:   false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := ValidateN64(testCase.header)
			if got != testCase.want {
				t.Errorf("ValidateN64() = %v, want %v", got, testCase.want)
			}
		})
	}
}

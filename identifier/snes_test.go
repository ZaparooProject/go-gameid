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

// createSNESHeader creates a minimal valid SNES ROM with LoROM header for testing.
// The header is at 0x7FC0 for LoROM.
//
//nolint:dupl // Similar header creation function is intentional for test clarity
func createSNESHeader(internalName string, developerID, romVersion byte, checksum uint16) []byte {
	// Need at least 0x8000 bytes for LoROM header
	rom := make([]byte, 0x8000)

	headerStart := snesLoROMHeaderStart

	// Internal name (21 bytes at 0x00)
	nameBytes := []byte(internalName)
	if len(nameBytes) > 21 {
		nameBytes = nameBytes[:21]
	}
	copy(rom[headerStart+snesInternalNameOffset:], nameBytes)

	// Map mode (0x15) - LoROM, SlowROM
	rom[headerStart+snesMapModeOffset] = 0x20 // LoROM

	// ROM type (0x16)
	rom[headerStart+snesROMTypeOffset] = 0x00 // ROM only

	// Developer ID (0x1A)
	rom[headerStart+snesDeveloperIDOffset] = developerID

	// ROM version (0x1B)
	rom[headerStart+snesROMVersionOffset] = romVersion

	// Checksum complement (0x1C-0x1D) - checksum + complement = 0xFFFF
	complement := 0xFFFF - checksum
	rom[headerStart+snesChecksumComplementOffset] = byte(complement & 0xFF)
	rom[headerStart+snesChecksumComplementOffset+1] = byte(complement >> 8)

	// Checksum (0x1E-0x1F)
	rom[headerStart+snesChecksumOffset] = byte(checksum & 0xFF)
	rom[headerStart+snesChecksumOffset+1] = byte(checksum >> 8)

	return rom
}

// createSNESHeaderHiROM creates a SNES ROM with HiROM header.
//
//nolint:dupl // Similar header creation function is intentional for test clarity
func createSNESHeaderHiROM(internalName string, developerID, romVersion byte, checksum uint16) []byte {
	// Need at least 0x10000 bytes for HiROM header at 0xFFC0
	rom := make([]byte, 0x10000)

	headerStart := snesHiROMHeaderStart

	// Internal name (21 bytes at 0x00)
	nameBytes := []byte(internalName)
	if len(nameBytes) > 21 {
		nameBytes = nameBytes[:21]
	}
	copy(rom[headerStart+snesInternalNameOffset:], nameBytes)

	// Map mode (0x15) - HiROM, SlowROM
	rom[headerStart+snesMapModeOffset] = 0x21 // HiROM

	// ROM type (0x16)
	rom[headerStart+snesROMTypeOffset] = 0x02 // ROM + RAM + Battery

	// Developer ID (0x1A)
	rom[headerStart+snesDeveloperIDOffset] = developerID

	// ROM version (0x1B)
	rom[headerStart+snesROMVersionOffset] = romVersion

	// Checksum complement (0x1C-0x1D)
	complement := 0xFFFF - checksum
	rom[headerStart+snesChecksumComplementOffset] = byte(complement & 0xFF)
	rom[headerStart+snesChecksumComplementOffset+1] = byte(complement >> 8)

	// Checksum (0x1E-0x1F)
	rom[headerStart+snesChecksumOffset] = byte(checksum & 0xFF)
	rom[headerStart+snesChecksumOffset+1] = byte(checksum >> 8)

	return rom
}

func TestSNESIdentifier_Identify(t *testing.T) {
	t.Parallel()

	identifier := NewSNESIdentifier()

	tests := []struct {
		name         string
		wantTitle    string
		wantROMType  string
		wantFastSlow string
		rom          []byte
	}{
		{
			name:         "LoROM Game",
			rom:          createSNESHeader("SUPER MARIO WORLD", 0x01, 0, 0x1234),
			wantTitle:    "SUPER MARIO WORLD",
			wantROMType:  "LoROM",
			wantFastSlow: "SlowROM",
		},
		{
			name:         "HiROM Game",
			rom:          createSNESHeaderHiROM("ZELDA3", 0x01, 1, 0xABCD),
			wantTitle:    "ZELDA3",
			wantROMType:  "HiROM",
			wantFastSlow: "SlowROM",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			reader := bytes.NewReader(testCase.rom)

			result, err := identifier.Identify(reader, int64(len(testCase.rom)), nil)
			if err != nil {
				t.Fatalf("Identify() error = %v", err)
			}

			verifySNESResult(t, result, testCase.wantTitle, testCase.wantROMType, testCase.wantFastSlow)
		})
	}
}

func verifySNESResult(t *testing.T, result *Result, wantTitle, wantROMType, wantFastSlow string) {
	t.Helper()

	if result.InternalTitle != wantTitle {
		t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, wantTitle)
	}

	if result.Console != ConsoleSNES {
		t.Errorf("Console = %v, want %v", result.Console, ConsoleSNES)
	}

	if romType := result.Metadata["rom_type"]; romType != wantROMType {
		t.Errorf("rom_type = %q, want %q", romType, wantROMType)
	}

	if fastSlow := result.Metadata["fast_slow_rom"]; fastSlow != wantFastSlow {
		t.Errorf("fast_slow_rom = %q, want %q", fastSlow, wantFastSlow)
	}
}

func TestSNESIdentifier_SMCHeader(t *testing.T) {
	t.Parallel()

	identifier := NewSNESIdentifier()

	// Create a ROM with 512-byte SMC header
	baseROM := createSNESHeader("SMC TEST GAME", 0x02, 0, 0x5678)
	smcHeader := make([]byte, 512)
	romWithSMC := make([]byte, 0, len(smcHeader)+len(baseROM))
	romWithSMC = append(romWithSMC, smcHeader...)
	romWithSMC = append(romWithSMC, baseROM...)

	reader := bytes.NewReader(romWithSMC)

	result, err := identifier.Identify(reader, int64(len(romWithSMC)), nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	if result.InternalTitle != "SMC TEST GAME" {
		t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, "SMC TEST GAME")
	}
}

func TestSNESIdentifier_InvalidChecksum(t *testing.T) {
	t.Parallel()

	identifier := NewSNESIdentifier()

	// Create a ROM with invalid checksum (doesn't sum to 0xFFFF)
	rom := make([]byte, 0x8000)
	headerStart := snesLoROMHeaderStart
	copy(rom[headerStart:], "INVALID")
	// Don't set valid checksum+complement

	reader := bytes.NewReader(rom)

	_, err := identifier.Identify(reader, int64(len(rom)), nil)
	if err == nil {
		t.Error("expected error for invalid checksum, got nil")
	}
}

func TestValidateSNES(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rom  []byte
		want bool
	}{
		{
			name: "Valid LoROM",
			rom:  createSNESHeader("TEST", 0x01, 0, 0x1234),
			want: true,
		},
		{
			name: "Valid HiROM",
			rom:  createSNESHeaderHiROM("TEST", 0x01, 0, 0x1234),
			want: true,
		},
		{
			name: "Invalid",
			rom:  make([]byte, 0x8000),
			want: false,
		},
		{
			name: "With SMC header",
			rom:  append(make([]byte, 512), createSNESHeader("TEST", 0x01, 0, 0x1234)...),
			want: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := ValidateSNES(testCase.rom)
			if got != testCase.want {
				t.Errorf("ValidateSNES() = %v, want %v", got, testCase.want)
			}
		})
	}
}

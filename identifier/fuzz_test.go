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

// createMinimalSNESROMLoROM creates a minimal SNES LoROM with valid checksum for fuzzing.
func createMinimalSNESROMLoROM() []byte {
	return createMinimalSNESROMWithHeader(0x7FC0, 0x10000)
}

// createMinimalSNESROMHiROM creates a minimal SNES HiROM with valid checksum for fuzzing.
func createMinimalSNESROMHiROM() []byte {
	return createMinimalSNESROMWithHeader(0xFFC0, 0x20000)
}

func createMinimalSNESROMWithHeader(headerStart, size int) []byte {
	data := make([]byte, size)

	// Fill internal name with printable chars
	copy(data[headerStart:], "FUZZ TEST ROM        ")

	// Set checksum and complement to be valid (sum = 0xFFFF)
	checksum := uint16(0x1234)
	complement := 0xFFFF - checksum
	data[headerStart+0x1E] = byte(checksum & 0xFF)
	data[headerStart+0x1F] = byte(checksum >> 8)
	data[headerStart+0x1C] = byte(complement & 0xFF)
	data[headerStart+0x1D] = byte(complement >> 8)

	return data
}

// createMinimalN64ROM creates a minimal N64 ROM header for fuzzing.
func createMinimalN64ROM() []byte {
	data := make([]byte, 0x40) // 64 bytes header

	// Big-endian magic word
	copy(data[0:4], []byte{0x80, 0x37, 0x12, 0x40})

	// Internal name at 0x20
	copy(data[0x20:], "FUZZ TEST       ")

	// Cartridge ID at 0x3C
	data[0x3C] = 'N'
	data[0x3D] = 'F'
	data[0x3E] = 'E' // Country code
	data[0x3F] = 0   // Version

	return data
}

// createMinimalGenesisROM creates a minimal Genesis ROM for fuzzing.
func createMinimalGenesisROM() []byte {
	data := make([]byte, 0x200)

	// Magic word at 0x100
	copy(data[0x100:], "SEGA GENESIS    ")

	// Titles at standard offsets
	copy(data[0x120:], "DOMESTIC TITLE                          ")
	copy(data[0x150:], "OVERSEAS TITLE                          ")

	// Software type
	copy(data[0x180:], "GM")

	// Game ID
	copy(data[0x182:], "T-123456-00")

	return data
}

// createMinimalGBROM creates a minimal Game Boy ROM for fuzzing.
func createMinimalGBROM() []byte {
	data := make([]byte, 0x150)

	// Nintendo logo at 0x104 (48 bytes)
	nintendoLogo := []byte{
		0xCE, 0xED, 0x66, 0x66, 0xCC, 0x0D, 0x00, 0x0B,
		0x03, 0x73, 0x00, 0x83, 0x00, 0x0C, 0x00, 0x0D,
		0x00, 0x08, 0x11, 0x1F, 0x88, 0x89, 0x00, 0x0E,
		0xDC, 0xCC, 0x6E, 0xE6, 0xDD, 0xDD, 0xD9, 0x99,
		0xBB, 0xBB, 0x67, 0x63, 0x6E, 0x0E, 0xEC, 0xCC,
		0xDD, 0xDC, 0x99, 0x9F, 0xBB, 0xB9, 0x33, 0x3E,
	}
	copy(data[0x104:], nintendoLogo)

	// Title at 0x134 (16 bytes)
	copy(data[0x134:], "FUZZ TEST   ")

	// CGB flag
	data[0x143] = 0x00 // GB only

	// SGB flag
	data[0x146] = 0x00

	// Cartridge type
	data[0x147] = 0x00 // ROM only

	// ROM size
	data[0x148] = 0x00 // 32KB

	// RAM size
	data[0x149] = 0x00 // None

	// Destination
	data[0x14A] = 0x01 // Non-Japanese

	// Old licensee code
	data[0x14B] = 0x01

	// Calculate header checksum (0x134-0x14C)
	var checksum byte
	for i := 0x134; i <= 0x14C; i++ {
		checksum = checksum - data[i] - 1
	}
	data[0x14D] = checksum

	return data
}

// FuzzSNESIdentify fuzzes SNES ROM identification.
// Tests snesFindHeader, snesGetCoprocessor, snesGetExtendedCoprocessor.
func FuzzSNESIdentify(f *testing.F) {
	// Add corpus seeds
	f.Add(createMinimalSNESROMLoROM()) // LoROM
	f.Add(createMinimalSNESROMHiROM()) // HiROM
	f.Add(make([]byte, 0x8000))        // Too small for LoROM
	f.Add(make([]byte, 0x10000))       // Minimum size
	f.Add([]byte{})                    // Empty

	// ROM with SMC header (512 byte prefix)
	smcROM := make([]byte, 512+len(createMinimalSNESROMLoROM()))
	copy(smcROM[512:], createMinimalSNESROMLoROM())
	f.Add(smcROM)

	f.Fuzz(func(_ *testing.T, data []byte) {
		// Limit size to prevent memory issues
		if len(data) > 16*1024*1024 { // 16MB
			return
		}

		reader := bytes.NewReader(data)
		identifier := NewSNESIdentifier()

		// Should not panic regardless of input
		result, err := identifier.Identify(reader, int64(len(data)), nil)
		if err != nil {
			// Expected for invalid data
			return
		}

		// If identification succeeded, verify result is usable
		_ = result.ID
		_ = result.Title
		_ = result.InternalTitle
		_ = result.Console
	})
}

// FuzzN64Identify fuzzes N64 ROM identification.
// Tests n64NormalizeEndianness, n64ByteSwap, n64WordSwap.
func FuzzN64Identify(f *testing.F) {
	// Add corpus seeds
	f.Add(createMinimalN64ROM())

	// Byte-swapped (.v64) format
	v64 := createMinimalN64ROM()
	for i := 0; i < len(v64); i += 2 {
		v64[i], v64[i+1] = v64[i+1], v64[i]
	}
	f.Add(v64)

	// Word-swapped (.n64) format
	n64 := createMinimalN64ROM()
	for i := 0; i < len(n64); i += 4 {
		n64[i], n64[i+1], n64[i+2], n64[i+3] = n64[i+3], n64[i+2], n64[i+1], n64[i]
	}
	f.Add(n64)

	f.Add(make([]byte, 64))   // Zeros (invalid magic)
	f.Add(make([]byte, 63))   // Too small
	f.Add([]byte{})           // Empty
	f.Add(make([]byte, 65))   // Odd length (edge case for byte swap)
	f.Add([]byte{0x80, 0x37}) // Partial magic

	f.Fuzz(func(_ *testing.T, data []byte) {
		// Limit size
		if len(data) > 64*1024*1024 { // 64MB
			return
		}

		reader := bytes.NewReader(data)
		identifier := NewN64Identifier()

		// Should not panic regardless of input
		result, err := identifier.Identify(reader, int64(len(data)), nil)
		if err != nil {
			return
		}

		_ = result.ID
		_ = result.Title
	})
}

// FuzzGenesisIdentify fuzzes Genesis ROM identification.
// Tests findGenesisMagicWord, genesisParseHeader.
func FuzzGenesisIdentify(f *testing.F) {
	// Add corpus seeds
	f.Add(createMinimalGenesisROM())
	f.Add(make([]byte, 0x200)) // Zeros (no magic word)
	f.Add(make([]byte, 0x100)) // Too small
	f.Add([]byte{})            // Empty

	// ROM with different magic words
	megaDrive := createMinimalGenesisROM()
	copy(megaDrive[0x100:], "SEGA MEGA DRIVE ")
	f.Add(megaDrive)

	f.Fuzz(func(_ *testing.T, data []byte) {
		// Limit size
		if len(data) > 16*1024*1024 { // 16MB
			return
		}

		reader := bytes.NewReader(data)
		identifier := NewGenesisIdentifier()

		// Should not panic regardless of input
		result, err := identifier.Identify(reader, int64(len(data)), nil)
		if err != nil {
			return
		}

		_ = result.ID
		_ = result.Title
	})
}

// FuzzGBIdentify fuzzes Game Boy ROM identification.
// Tests header checksum calculation, licensee code lookup.
func FuzzGBIdentify(f *testing.F) {
	// Add corpus seeds
	f.Add(createMinimalGBROM())
	f.Add(make([]byte, 0x150)) // Zeros
	f.Add(make([]byte, 0x100)) // Too small
	f.Add([]byte{})            // Empty

	// GBC-only ROM
	gbcROM := createMinimalGBROM()
	gbcROM[0x143] = 0xC0 // GBC only flag
	f.Add(gbcROM)

	// GBC-compatible ROM
	gbcCompat := createMinimalGBROM()
	gbcCompat[0x143] = 0x80 // GBC compatible flag
	f.Add(gbcCompat)

	f.Fuzz(func(_ *testing.T, data []byte) {
		// Limit size
		if len(data) > 16*1024*1024 { // 16MB
			return
		}

		reader := bytes.NewReader(data)
		identifier := NewGBIdentifier()

		// Should not panic regardless of input
		result, err := identifier.Identify(reader, int64(len(data)), nil)
		if err != nil {
			return
		}

		_ = result.ID
		_ = result.Title
	})
}

// FuzzValidateSNES fuzzes SNES ROM validation.
func FuzzValidateSNES(f *testing.F) {
	f.Add(createMinimalSNESROMLoROM())
	f.Add(createMinimalSNESROMHiROM())
	f.Add(make([]byte, 0x10000))
	f.Add([]byte{})

	f.Fuzz(func(_ *testing.T, data []byte) {
		// Should not panic
		_ = ValidateSNES(data)
	})
}

// FuzzValidateN64 fuzzes N64 ROM validation.
func FuzzValidateN64(f *testing.F) {
	f.Add(createMinimalN64ROM())
	f.Add(make([]byte, 64))
	f.Add([]byte{})

	f.Fuzz(func(_ *testing.T, data []byte) {
		// Should not panic
		_ = ValidateN64(data)
	})
}

// FuzzValidateGenesis fuzzes Genesis ROM validation.
func FuzzValidateGenesis(f *testing.F) {
	f.Add(createMinimalGenesisROM())
	f.Add(make([]byte, 0x200))
	f.Add([]byte{})

	f.Fuzz(func(_ *testing.T, data []byte) {
		// Should not panic
		_ = ValidateGenesis(data)
	})
}

// FuzzValidateGB fuzzes Game Boy ROM validation.
func FuzzValidateGB(f *testing.F) {
	f.Add(createMinimalGBROM())
	f.Add(make([]byte, 0x150))
	f.Add([]byte{})

	f.Fuzz(func(_ *testing.T, data []byte) {
		// Should not panic
		_ = ValidateGB(data)
	})
}

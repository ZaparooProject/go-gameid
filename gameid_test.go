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

package gameid

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ZaparooProject/go-gameid/identifier"
)

func TestParseConsole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    Console
		wantErr bool
	}{
		{"GB lowercase", "gb", ConsoleGB, false},
		{"GB uppercase", "GB", ConsoleGB, false},
		{"GameBoy", "gameboy", ConsoleGB, false},
		{"GBC", "gbc", ConsoleGBC, false},
		{"GBA", "gba", ConsoleGBA, false},
		{"GameCube", "gamecube", ConsoleGC, false},
		{"GC", "gc", ConsoleGC, false},
		{"NGC", "ngc", ConsoleGC, false},
		{"Genesis", "genesis", ConsoleGenesis, false},
		{"MegaDrive", "megadrive", ConsoleGenesis, false},
		{"MD", "md", ConsoleGenesis, false},
		{"N64", "n64", ConsoleN64, false},
		{"Nintendo64", "nintendo64", ConsoleN64, false},
		{"NES", "nes", ConsoleNES, false},
		{"Famicom", "famicom", ConsoleNES, false},
		{"SNES", "snes", ConsoleSNES, false},
		{"SuperFamicom", "superfamicom", ConsoleSNES, false},
		{"PSX", "psx", ConsolePSX, false},
		{"PS1", "ps1", ConsolePSX, false},
		{"PlayStation", "playstation", ConsolePSX, false},
		{"PS2", "ps2", ConsolePS2, false},
		{"PlayStation2", "playstation2", ConsolePS2, false},
		{"PSP", "psp", ConsolePSP, false},
		{"Saturn", "saturn", ConsoleSaturn, false},
		{"SegaSaturn", "segasaturn", ConsoleSaturn, false},
		{"SegaCD", "segacd", ConsoleSegaCD, false},
		{"MegaCD", "megacd", ConsoleSegaCD, false},
		{"NeoGeoCD", "neogeocd", ConsoleNeoGeoCD, false},
		{"Unknown", "xbox", "", true},
		{"Empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseConsole(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConsole(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseConsole(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSupportedConsoles(t *testing.T) {
	t.Parallel()

	consoles := SupportedConsoles()
	if len(consoles) != len(AllConsoles) {
		t.Errorf("SupportedConsoles() returned %d consoles, want %d", len(consoles), len(AllConsoles))
	}

	// Check that all expected consoles are present
	expected := map[string]bool{
		"GB": true, "GBC": true, "GBA": true, "GC": true,
		"Genesis": true, "N64": true, "NeoGeoCD": true, "NES": true,
		"PSP": true, "PSX": true, "PS2": true, "Saturn": true,
		"SegaCD": true, "SNES": true,
	}

	for _, c := range consoles {
		if !expected[c] {
			t.Errorf("Unexpected console: %s", c)
		}
	}
}

func TestIsDiscBased(t *testing.T) {
	t.Parallel()

	discBased := []Console{ConsoleGC, ConsoleNeoGeoCD, ConsolePSP, ConsolePSX, ConsolePS2, ConsoleSaturn, ConsoleSegaCD}
	cartBased := []Console{ConsoleGB, ConsoleGBC, ConsoleGBA, ConsoleGenesis, ConsoleN64, ConsoleNES, ConsoleSNES}

	for _, c := range discBased {
		if !IsDiscBased(c) {
			t.Errorf("IsDiscBased(%s) = false, want true", c)
		}
		if IsCartridgeBased(c) {
			t.Errorf("IsCartridgeBased(%s) = true, want false", c)
		}
	}

	for _, c := range cartBased {
		if IsDiscBased(c) {
			t.Errorf("IsDiscBased(%s) = true, want false", c)
		}
		if !IsCartridgeBased(c) {
			t.Errorf("IsCartridgeBased(%s) = false, want true", c)
		}
	}
}

// createTestGBAFile creates a minimal valid GBA ROM for testing
func createTestGBAFile(t *testing.T, tmpDir string) string {
	// GBA header is 192 bytes
	header := make([]byte, 0xC0)

	// Entry point
	copy(header[0x00:], []byte{0x00, 0x00, 0x00, 0xEA})

	// Nintendo logo (156 bytes at 0x04) - use the actual logo
	nintendoLogo := []byte{
		0x24, 0xFF, 0xAE, 0x51, 0x69, 0x9A, 0xA2, 0x21, 0x3D, 0x84, 0x82, 0x0A,
		0x84, 0xE4, 0x09, 0xAD, 0x11, 0x24, 0x8B, 0x98, 0xC0, 0x81, 0x7F, 0x21,
		0xA3, 0x52, 0xBE, 0x19, 0x93, 0x09, 0xCE, 0x20, 0x10, 0x46, 0x4A, 0x4A,
		0xF8, 0x27, 0x31, 0xEC, 0x58, 0xC7, 0xE8, 0x33, 0x82, 0xE3, 0xCE, 0xBF,
		0x85, 0xF4, 0xDF, 0x94, 0xCE, 0x4B, 0x09, 0xC1, 0x94, 0x56, 0x8A, 0xC0,
		0x13, 0x72, 0xA7, 0xFC, 0x9F, 0x84, 0x4D, 0x73, 0xA3, 0xCA, 0x9A, 0x61,
		0x58, 0x97, 0xA3, 0x27, 0xFC, 0x03, 0x98, 0x76, 0x23, 0x1D, 0xC7, 0x61,
		0x03, 0x04, 0xAE, 0x56, 0xBF, 0x38, 0x84, 0x00, 0x40, 0xA7, 0x0E, 0xFD,
		0xFF, 0x52, 0xFE, 0x03, 0x6F, 0x95, 0x30, 0xF1, 0x97, 0xFB, 0xC0, 0x85,
		0x60, 0xD6, 0x80, 0x25, 0xA9, 0x63, 0xBE, 0x03, 0x01, 0x4E, 0x38, 0xE2,
		0xF9, 0xA2, 0x34, 0xFF, 0xBB, 0x3E, 0x03, 0x44, 0x78, 0x00, 0x90, 0xCB,
		0x88, 0x11, 0x3A, 0x94, 0x65, 0xC0, 0x7C, 0x63, 0x87, 0xF0, 0x3C, 0xAF,
		0xD6, 0x25, 0xE4, 0x8B, 0x38, 0x0A, 0xAC, 0x72, 0x21, 0xD4, 0xF8, 0x07,
	}
	copy(header[0x04:], nintendoLogo)

	// Internal title (12 bytes at 0xA0)
	copy(header[0xA0:], "TESTGAME    ")

	// Game code (4 bytes at 0xAC)
	copy(header[0xAC:], "ATST")

	// Maker code (2 bytes at 0xB0)
	copy(header[0xB0:], "01")

	// Fixed value (0xB2)
	header[0xB2] = 0x96

	path := filepath.Join(tmpDir, "test.gba")
	if err := os.WriteFile(path, header, 0o600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	return path
}

func TestIdentifyWithConsole(t *testing.T) {
	t.Parallel()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gameid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a GBA test file
	gbaPath := createTestGBAFile(t, tmpDir)

	// Test identification
	result, err := IdentifyWithConsole(gbaPath, ConsoleGBA, nil)
	if err != nil {
		t.Fatalf("IdentifyWithConsole() error = %v", err)
	}

	if result.Console != identifier.ConsoleGBA {
		t.Errorf("Console = %v, want %v", result.Console, identifier.ConsoleGBA)
	}

	if result.ID != "ATST" {
		t.Errorf("ID = %q, want %q", result.ID, "ATST")
	}

	if result.InternalTitle != "TESTGAME" {
		t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, "TESTGAME")
	}
}

func TestIdentify(t *testing.T) {
	t.Parallel()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gameid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a GBA test file
	gbaPath := createTestGBAFile(t, tmpDir)

	// Test auto-detection and identification
	result, err := Identify(gbaPath, nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	if result.Console != identifier.ConsoleGBA {
		t.Errorf("Console = %v, want %v", result.Console, identifier.ConsoleGBA)
	}

	if result.ID != "ATST" {
		t.Errorf("ID = %q, want %q", result.ID, "ATST")
	}
}

func TestIdentify_NonExistent(t *testing.T) {
	t.Parallel()

	_, err := Identify("/nonexistent/path/game.gba", nil)
	if err == nil {
		t.Error("Identify() should error for non-existent file")
	}
}

func TestIdentify_UnsupportedFormat(t *testing.T) {
	t.Parallel()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gameid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a file with unsupported extension
	path := filepath.Join(tmpDir, "game.xyz")
	if writeErr := os.WriteFile(path, []byte("test"), 0o600); writeErr != nil {
		t.Fatalf("Failed to write test file: %v", writeErr)
	}

	_, err = Identify(path, nil)
	if err == nil {
		t.Error("Identify() should error for unsupported format")
	}
}

func TestIsBlockDevice(t *testing.T) {
	t.Parallel()

	// Regular files should not be detected as block devices
	tmpDir, err := os.MkdirTemp("", "gameid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	regularFile := filepath.Join(tmpDir, "test.iso")
	if err := os.WriteFile(regularFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	if isBlockDevice(regularFile) {
		t.Error("isBlockDevice() should return false for regular file")
	}

	// Non-/dev/ paths should return false
	if isBlockDevice("/tmp/not-a-device") {
		t.Error("isBlockDevice() should return false for non-/dev/ paths")
	}

	// Non-existent paths should return false
	if isBlockDevice("/dev/nonexistent123456789") {
		t.Error("isBlockDevice() should return false for non-existent device")
	}
}

// TestIdentifyFromReader verifies IdentifyFromReader with a GBA ROM.
func TestIdentifyFromReader(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	gbaPath := createTestGBAFile(t, tmpDir)

	// Open the file and use IdentifyFromReader
	//nolint:gosec // G304: test file path constructed from t.TempDir
	file, err := os.Open(gbaPath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	result, err := IdentifyFromReader(file, stat.Size(), ConsoleGBA, nil)
	if err != nil {
		t.Fatalf("IdentifyFromReader() error = %v", err)
	}

	if result.Console != identifier.ConsoleGBA {
		t.Errorf("Console = %v, want %v", result.Console, identifier.ConsoleGBA)
	}

	if result.ID != "ATST" {
		t.Errorf("ID = %q, want %q", result.ID, "ATST")
	}
}

// TestIdentifyFromReader_UnsupportedConsole verifies error for unsupported console.
func TestIdentifyFromReader_UnsupportedConsole(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	gbaPath := createTestGBAFile(t, tmpDir)

	//nolint:gosec // G304: test file path constructed from t.TempDir
	file, err := os.Open(gbaPath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	_, err = IdentifyFromReader(file, stat.Size(), "Xbox", nil)
	if err == nil {
		t.Error("IdentifyFromReader() should error for unsupported console")
	}
}

// TestIdentifyWithConsole_UnsupportedConsole verifies error for unsupported console.
func TestIdentifyWithConsole_UnsupportedConsole(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	gbaPath := createTestGBAFile(t, tmpDir)

	_, err := IdentifyWithConsole(gbaPath, "Xbox", nil)
	if err == nil {
		t.Error("IdentifyWithConsole() should error for unsupported console")
	}
}

// TestIdentifyFromDirectory_PSP verifies mounted PSP directory identification.
func TestIdentifyFromDirectory_PSP(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create UMD_DATA.BIN marker file (identifies as PSP)
	markerPath := filepath.Join(tmpDir, "UMD_DATA.BIN")
	if err := os.WriteFile(markerPath, []byte("ULJM12345|0000000001|0001"), 0o600); err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}

	// First detect the console
	console, err := DetectConsole(tmpDir)
	if err != nil {
		t.Fatalf("DetectConsole() error = %v", err)
	}
	if console != ConsolePSP {
		t.Fatalf("DetectConsole() = %v, want PSP", console)
	}

	// Now identify - note this will error because we don't have a full PSP filesystem
	// but it will exercise the identifyFromDirectory path
	_, err = IdentifyWithConsole(tmpDir, console, nil)
	// Error is expected since we don't have PARAM.SFO
	if err == nil {
		t.Log("IdentifyWithConsole() succeeded unexpectedly for minimal PSP dir")
	}
}

// TestIdentifyFromDirectory_CartridgeConsole verifies error when using directory with cartridge console.
func TestIdentifyFromDirectory_CartridgeConsole(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Try to identify a directory as a GBA game (cartridge-based)
	// This should fail since directories aren't supported for cartridge consoles
	_, err := identifyFromDirectory(tmpDir, ConsoleGBA, nil)
	if err == nil {
		t.Error("identifyFromDirectory() should error for cartridge-based console")
	}
}

// TestIdentifyFromDirectory_UnsupportedConsole verifies error for unsupported console.
func TestIdentifyFromDirectory_UnsupportedConsole(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	_, err := identifyFromDirectory(tmpDir, "Xbox", nil)
	if err == nil {
		t.Error("identifyFromDirectory() should error for unsupported console")
	}
}

// TestIdentifyFromArchive_ZIP verifies identification from ZIP archives.
func TestIdentifyFromArchive_ZIP(t *testing.T) {
	t.Parallel()

	result, err := Identify("testdata/archive/snes.zip", nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	if result.Console != identifier.ConsoleSNES {
		t.Errorf("Console = %v, want %v", result.Console, identifier.ConsoleSNES)
	}

	if result.InternalTitle != "240P TEST SUITE SNES" {
		t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, "240P TEST SUITE SNES")
	}
}

// TestIdentifyFromArchive_7z verifies identification from 7z archives.
func TestIdentifyFromArchive_7z(t *testing.T) {
	t.Parallel()

	result, err := Identify("testdata/archive/snes.7z", nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	if result.Console != identifier.ConsoleSNES {
		t.Errorf("Console = %v, want %v", result.Console, identifier.ConsoleSNES)
	}

	if result.InternalTitle != "240P TEST SUITE SNES" {
		t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, "240P TEST SUITE SNES")
	}
}

// TestIdentifyFromArchive_RAR verifies identification from RAR archives.
func TestIdentifyFromArchive_RAR(t *testing.T) {
	t.Parallel()

	result, err := Identify("testdata/archive/snes.rar", nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	if result.Console != identifier.ConsoleSNES {
		t.Errorf("Console = %v, want %v", result.Console, identifier.ConsoleSNES)
	}

	if result.InternalTitle != "240P TEST SUITE SNES" {
		t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, "240P TEST SUITE SNES")
	}
}

// TestIdentifyFromArchive_Genesis verifies identification of Genesis ROMs from archives.
func TestIdentifyFromArchive_Genesis(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{"ZIP", "testdata/archive/genesis.zip"},
		{"7z", "testdata/archive/genesis.7z"},
		{"RAR", "testdata/archive/genesis.rar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := Identify(tt.path, nil)
			if err != nil {
				t.Fatalf("Identify() error = %v", err)
			}

			if result.Console != identifier.ConsoleGenesis {
				t.Errorf("Console = %v, want %v", result.Console, identifier.ConsoleGenesis)
			}

			if result.InternalTitle != "240P TEST SUITE" {
				t.Errorf("InternalTitle = %q, want %q", result.InternalTitle, "240P TEST SUITE")
			}
		})
	}
}

// TestIdentifyFromArchive_WithInternalPath verifies MiSTer-style paths work.
func TestIdentifyFromArchive_WithInternalPath(t *testing.T) {
	t.Parallel()

	// Test explicit internal path
	result, err := Identify("testdata/archive/snes.zip/240pSuite.sfc", nil)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	if result.Console != identifier.ConsoleSNES {
		t.Errorf("Console = %v, want %v", result.Console, identifier.ConsoleSNES)
	}
}

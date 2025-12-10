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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ZaparooProject/go-gameid/identifier"
)

//nolint:funlen // Table-driven test with many test cases
func TestDetectConsole(t *testing.T) {
	t.Parallel()

	// Create temp directory for test files
	tmpDir, err := os.MkdirTemp("", "gameid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	tests := []struct {
		name     string
		filename string
		want     identifier.Console
		content  []byte
		wantErr  bool
	}{
		{
			name:     "GBA by extension",
			filename: "game.gba",
			content:  make([]byte, 0xC0),
			want:     identifier.ConsoleGBA,
		},
		{
			name:     "GB by extension",
			filename: "game.gb",
			content:  make([]byte, 0x150),
			want:     identifier.ConsoleGB,
		},
		{
			name:     "GBC by extension",
			filename: "game.gbc",
			content:  make([]byte, 0x150),
			want:     identifier.ConsoleGBC,
		},
		{
			name:     "N64 z64 extension",
			filename: "game.z64",
			content:  make([]byte, 0x40),
			want:     identifier.ConsoleN64,
		},
		{
			name:     "N64 v64 extension",
			filename: "game.v64",
			content:  make([]byte, 0x40),
			want:     identifier.ConsoleN64,
		},
		{
			name:     "N64 n64 extension",
			filename: "game.n64",
			content:  make([]byte, 0x40),
			want:     identifier.ConsoleN64,
		},
		{
			name:     "NES by extension",
			filename: "game.nes",
			content:  make([]byte, 0x100),
			want:     identifier.ConsoleNES,
		},
		{
			name:     "SNES by extension",
			filename: "game.sfc",
			content:  make([]byte, 0x8000),
			want:     identifier.ConsoleSNES,
		},
		{
			name:     "Genesis gen extension",
			filename: "game.gen",
			content:  make([]byte, 0x200),
			want:     identifier.ConsoleGenesis,
		},
		{
			name:     "Genesis md extension",
			filename: "game.md",
			content:  make([]byte, 0x200),
			want:     identifier.ConsoleGenesis,
		},
		{
			name:     "GameCube gcm extension",
			filename: "game.gcm",
			content:  make([]byte, 0x100),
			want:     identifier.ConsoleGC,
		},
		{
			name:     "Unsupported extension",
			filename: "game.xyz",
			content:  make([]byte, 0x100),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create test file
			path := filepath.Join(tmpDir, tt.filename)
			if err := os.WriteFile(path, tt.content, 0o600); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			got, err := DetectConsole(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectConsole() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("DetectConsole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectConsole_GzSuffix(t *testing.T) {
	t.Parallel()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gameid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a .gba.gz file
	path := filepath.Join(tmpDir, "game.gba.gz")
	if writeErr := os.WriteFile(path, make([]byte, 0x100), 0o600); writeErr != nil {
		t.Fatalf("Failed to write test file: %v", writeErr)
	}

	got, detectErr := DetectConsole(path)
	if detectErr != nil {
		t.Errorf("DetectConsole() error = %v", detectErr)
		return
	}
	if got != identifier.ConsoleGBA {
		t.Errorf("DetectConsole() = %v, want %v", got, identifier.ConsoleGBA)
	}
}

func TestDetectConsole_NonExistent(t *testing.T) {
	t.Parallel()

	_, err := DetectConsole("/nonexistent/path/game.gba")
	if err == nil {
		t.Error("DetectConsole() should error for non-existent file")
	}
}

func TestDetectConsoleFromHeader_GameCube(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "game.bin")

	// GameCube magic is 0xC2339F3D at offset 0x1C
	header := make([]byte, 0x100)
	header[0x1C] = 0xC2
	header[0x1D] = 0x33
	header[0x1E] = 0x9F
	header[0x1F] = 0x3D

	if err := os.WriteFile(path, header, 0o600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	console, err := DetectConsole(path)
	if err != nil {
		t.Fatalf("DetectConsole() error = %v", err)
	}
	if console != identifier.ConsoleGC {
		t.Errorf("DetectConsole() = %v, want %v", console, identifier.ConsoleGC)
	}
}

func TestDetectConsoleFromHeader_Saturn(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "game.bin")

	// Saturn magic is "SEGA SEGASATURN"
	header := make([]byte, 0x100)
	copy(header, "SEGA SEGASATURN")

	if err := os.WriteFile(path, header, 0o600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	console, err := DetectConsole(path)
	if err != nil {
		t.Fatalf("DetectConsole() error = %v", err)
	}
	if console != identifier.ConsoleSaturn {
		t.Errorf("DetectConsole() = %v, want %v", console, identifier.ConsoleSaturn)
	}
}

func TestDetectConsoleFromHeader_SegaCD(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		magic string
	}{
		{"SEGADISCSYSTEM", "SEGADISCSYSTEM"},
		{"SEGABOOTDISC", "SEGABOOTDISC"},
		{"SEGADISC", "SEGADISC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "game.bin")

			header := make([]byte, 0x100)
			copy(header, tt.magic)

			if err := os.WriteFile(path, header, 0o600); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			console, err := DetectConsole(path)
			if err != nil {
				t.Fatalf("DetectConsole() error = %v", err)
			}
			if console != identifier.ConsoleSegaCD {
				t.Errorf("DetectConsole() = %v, want %v", console, identifier.ConsoleSegaCD)
			}
		})
	}
}

func TestDetectConsoleFromHeader_Genesis(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		magic string
	}{
		{"SEGA GENESIS", "SEGA GENESIS"},
		{"SEGA MEGA DRIVE", "SEGA MEGA DRIVE"},
		{"SEGA 32X", "SEGA 32X"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "game.bin")

			// Genesis magic is at offset 0x100-0x200
			header := make([]byte, 0x200)
			copy(header[0x100:], tt.magic)

			if err := os.WriteFile(path, header, 0o600); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			console, err := DetectConsole(path)
			if err != nil {
				t.Fatalf("DetectConsole() error = %v", err)
			}
			if console != identifier.ConsoleGenesis {
				t.Errorf("DetectConsole() = %v, want %v", console, identifier.ConsoleGenesis)
			}
		})
	}
}

func TestDetectConsoleFromDirectory_PSP(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create UMD_DATA.BIN marker file
	if err := os.WriteFile(filepath.Join(tmpDir, "UMD_DATA.BIN"), []byte("test"), 0o600); err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}

	console, err := DetectConsole(tmpDir)
	if err != nil {
		t.Fatalf("DetectConsole() error = %v", err)
	}
	if console != identifier.ConsolePSP {
		t.Errorf("DetectConsole() = %v, want %v", console, identifier.ConsolePSP)
	}
}

func TestDetectConsoleFromDirectory_NeoGeoCD(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create IPL.TXT marker file
	if err := os.WriteFile(filepath.Join(tmpDir, "IPL.TXT"), []byte("test"), 0o600); err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}

	console, err := DetectConsole(tmpDir)
	if err != nil {
		t.Fatalf("DetectConsole() error = %v", err)
	}
	if console != identifier.ConsoleNeoGeoCD {
		t.Errorf("DetectConsole() = %v, want %v", console, identifier.ConsoleNeoGeoCD)
	}
}

func TestDetectConsoleFromDirectory_PSX(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create SYSTEM.CNF with BOOT (PSX marker)
	if err := os.WriteFile(filepath.Join(tmpDir, "SYSTEM.CNF"), []byte("BOOT=cdrom:\\GAME.EXE"), 0o600); err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}

	console, err := DetectConsole(tmpDir)
	if err != nil {
		t.Fatalf("DetectConsole() error = %v", err)
	}
	if console != identifier.ConsolePSX {
		t.Errorf("DetectConsole() = %v, want %v", console, identifier.ConsolePSX)
	}
}

func TestDetectConsoleFromDirectory_PS2(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create SYSTEM.CNF with BOOT2 (PS2 marker)
	if err := os.WriteFile(filepath.Join(tmpDir, "SYSTEM.CNF"), []byte("BOOT2=cdrom0:\\GAME.ELF"), 0o600); err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}

	console, err := DetectConsole(tmpDir)
	if err != nil {
		t.Fatalf("DetectConsole() error = %v", err)
	}
	if console != identifier.ConsolePS2 {
		t.Errorf("DetectConsole() = %v, want %v", console, identifier.ConsolePS2)
	}
}

func TestDetectConsoleFromDirectory_Unsupported(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Empty directory should fail
	_, err := DetectConsole(tmpDir)
	if err == nil {
		t.Error("DetectConsole() should error for unsupported directory")
	}

	var notSupported identifier.ErrNotSupported
	if !errors.As(err, &notSupported) {
		t.Errorf("Expected ErrNotSupported, got %T: %v", err, err)
	}
}

func TestDetectConsoleFromHeader_AmbiguousISO(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "game.iso")

	// File with no recognizable magic bytes and not valid ISO
	header := make([]byte, 0x1000)
	if err := os.WriteFile(path, header, 0o600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := DetectConsole(path)
	// Should error because it's not a valid ISO and no magic detected
	if err == nil {
		t.Error("DetectConsole() should error for invalid ISO without magic")
	}
}

func TestFileExists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Test existing file
	existingPath := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(existingPath, []byte("test"), 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if !fileExists(existingPath) {
		t.Error("fileExists() = false for existing file")
	}

	// Test non-existent file
	if fileExists(filepath.Join(tmpDir, "nonexistent.txt")) {
		t.Error("fileExists() = true for non-existent file")
	}

	// Test directory (should return false - not a file)
	if fileExists(tmpDir) {
		t.Error("fileExists() = true for directory")
	}
}

// TestDetectConsoleFromCHD_SegaCD verifies CHD detection for Sega CD.
func TestDetectConsoleFromCHD_SegaCD(t *testing.T) {
	t.Parallel()

	console, err := DetectConsole("testdata/SegaCD/240pSuite_USA.chd")
	if err != nil {
		t.Fatalf("DetectConsole() error = %v", err)
	}
	if console != identifier.ConsoleSegaCD {
		t.Errorf("DetectConsole() = %v, want %v", console, identifier.ConsoleSegaCD)
	}
}

// TestDetectConsoleFromCHD_NeoGeoCD verifies CHD detection for Neo Geo CD.
func TestDetectConsoleFromCHD_NeoGeoCD(t *testing.T) {
	t.Parallel()

	console, err := DetectConsole("testdata/NeoGeoCD/240pTestSuite.chd")
	if err != nil {
		t.Fatalf("DetectConsole() error = %v", err)
	}
	if console != identifier.ConsoleNeoGeoCD {
		t.Errorf("DetectConsole() = %v, want %v", console, identifier.ConsoleNeoGeoCD)
	}
}

// TestDetectConsoleFromCHD_GameCube verifies CHD detection for GameCube.
func TestDetectConsoleFromCHD_GameCube(t *testing.T) {
	t.Parallel()

	console, err := DetectConsole("testdata/GC/GameCube-240pSuite-1.17.chd")
	if err != nil {
		t.Fatalf("DetectConsole() error = %v", err)
	}
	if console != identifier.ConsoleGC {
		t.Errorf("DetectConsole() = %v, want %v", console, identifier.ConsoleGC)
	}
}

// TestDetectConsoleFromCHD_NonExistent verifies error for missing CHD.
func TestDetectConsoleFromCHD_NonExistent(t *testing.T) {
	t.Parallel()

	_, err := DetectConsole("/nonexistent/path/game.chd")
	if err == nil {
		t.Error("DetectConsole() should fail for non-existent CHD")
	}
}

// TestDetectConsoleFromCue_MagicBased verifies CUE detection for consoles with magic headers.
func TestDetectConsoleFromCue_MagicBased(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		magic string
		want  identifier.Console
	}{
		{"Saturn", "SEGA SEGASATURN", identifier.ConsoleSaturn},
		{"SegaCD", "SEGADISCSYSTEM", identifier.ConsoleSegaCD},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()

			// Create BIN file with magic
			binPath := filepath.Join(tmpDir, "game.bin")
			binData := make([]byte, 0x100)
			copy(binData, tt.magic)
			if err := os.WriteFile(binPath, binData, 0o600); err != nil {
				t.Fatalf("Failed to write BIN file: %v", err)
			}

			// Create CUE file
			cuePath := filepath.Join(tmpDir, "game.cue")
			cueContent := `FILE "game.bin" BINARY
  TRACK 01 MODE1/2352
    INDEX 01 00:00:00
`
			if err := os.WriteFile(cuePath, []byte(cueContent), 0o600); err != nil {
				t.Fatalf("Failed to write CUE file: %v", err)
			}

			console, err := DetectConsole(cuePath)
			if err != nil {
				t.Fatalf("DetectConsole() error = %v", err)
			}
			if console != tt.want {
				t.Errorf("DetectConsole() = %v, want %v", console, tt.want)
			}
		})
	}
}

// TestDetectConsoleFromCue_EmptyCue verifies error for empty CUE.
func TestDetectConsoleFromCue_EmptyCue(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create CUE file with no BIN files
	cuePath := filepath.Join(tmpDir, "game.cue")
	cueContent := "REM Empty CUE\n"
	if err := os.WriteFile(cuePath, []byte(cueContent), 0o600); err != nil {
		t.Fatalf("Failed to write CUE file: %v", err)
	}

	_, err := DetectConsole(cuePath)
	if err == nil {
		t.Error("DetectConsole() should fail for empty CUE")
	}
}

// TestDetectConsoleFromCue_MissingBin verifies error for CUE with missing BIN.
func TestDetectConsoleFromCue_MissingBin(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create CUE file referencing missing BIN
	cuePath := filepath.Join(tmpDir, "game.cue")
	cueContent := `FILE "nonexistent.bin" BINARY
  TRACK 01 MODE1/2352
    INDEX 01 00:00:00
`
	if err := os.WriteFile(cuePath, []byte(cueContent), 0o600); err != nil {
		t.Fatalf("Failed to write CUE file: %v", err)
	}

	_, err := DetectConsole(cuePath)
	if err == nil {
		t.Error("DetectConsole() should fail for CUE with missing BIN")
	}
}

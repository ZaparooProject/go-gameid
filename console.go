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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ZaparooProject/go-gameid/chd"
	"github.com/ZaparooProject/go-gameid/identifier"
	"github.com/ZaparooProject/go-gameid/iso9660"
)

// Extension to console mapping
// Only includes unambiguous mappings (single console per extension)
var extToConsole = map[string]identifier.Console{
	// Game Boy / Game Boy Color
	".gb":  identifier.ConsoleGB,
	".gbc": identifier.ConsoleGBC,

	// Game Boy Advance
	".gba": identifier.ConsoleGBA,
	".srl": identifier.ConsoleGBA,

	// Nintendo 64
	".n64": identifier.ConsoleN64,
	".z64": identifier.ConsoleN64,
	".v64": identifier.ConsoleN64,
	".ndd": identifier.ConsoleN64,

	// NES
	".nes": identifier.ConsoleNES,
	".fds": identifier.ConsoleNES,
	".unf": identifier.ConsoleNES,
	".nez": identifier.ConsoleNES,

	// SNES
	".sfc": identifier.ConsoleSNES,
	".smc": identifier.ConsoleSNES,
	".swc": identifier.ConsoleSNES,

	// Genesis / Mega Drive
	".gen": identifier.ConsoleGenesis,
	".md":  identifier.ConsoleGenesis,
	".smd": identifier.ConsoleGenesis,

	// GameCube
	".gcm": identifier.ConsoleGC,
	".gcz": identifier.ConsoleGC,
	".rvz": identifier.ConsoleGC,
}

// Ambiguous extensions that need header analysis
var ambiguousExts = map[string]bool{
	".bin": true,
	".iso": true,
	".cue": true,
	".chd": true,
	".cso": true,
	".ecm": true,
}

// DetectConsole attempts to detect the console type for a given file.
// Returns the detected console or an error if detection fails.
func DetectConsole(path string) (identifier.Console, error) {
	// Check if it's a block device (physical disc)
	if isBlockDevice(path) {
		return detectConsoleFromBlockDevice(path)
	}

	// Check if it's a directory (mounted disc)
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat path: %w", err)
	}
	if info.IsDir() {
		return detectConsoleFromDirectory(path)
	}

	// Get extension
	ext := strings.ToLower(filepath.Ext(path))

	// Strip .gz suffix
	if ext == ".gz" {
		path = strings.TrimSuffix(path, ext)
		ext = strings.ToLower(filepath.Ext(path))
	}

	// Check for unambiguous extension
	if console, ok := extToConsole[ext]; ok {
		return console, nil
	}

	// For ambiguous extensions, read header and analyze
	if ambiguousExts[ext] {
		return detectConsoleFromHeader(path, ext)
	}

	return "", identifier.ErrNotSupported{Format: ext}
}

// DetectConsoleFromExtension detects the console type based purely on file extension.
// Unlike DetectConsole, this does not read file headers or check file existence.
// It returns an error for ambiguous extensions (like .bin, .iso) that require header analysis.
func DetectConsoleFromExtension(path string) (identifier.Console, error) {
	ext := strings.ToLower(filepath.Ext(path))

	// Strip .gz suffix
	if ext == ".gz" {
		path = strings.TrimSuffix(path, ext)
		ext = strings.ToLower(filepath.Ext(path))
	}

	// Check for unambiguous extension
	if console, ok := extToConsole[ext]; ok {
		return console, nil
	}

	// Ambiguous extensions cannot be detected without header analysis
	if ambiguousExts[ext] {
		return "", identifier.ErrNotSupported{
			Format: fmt.Sprintf("ambiguous extension %s requires header analysis", ext),
		}
	}

	return "", identifier.ErrNotSupported{Format: ext}
}

// detectConsoleFromDirectory detects console from a mounted disc directory
func detectConsoleFromDirectory(path string) (identifier.Console, error) {
	// Check for PSP (UMD_DATA.BIN)
	if fileExists(filepath.Join(path, "UMD_DATA.BIN")) {
		return identifier.ConsolePSP, nil
	}

	// Check for NeoGeoCD (IPL.TXT)
	if fileExists(filepath.Join(path, "IPL.TXT")) {
		return identifier.ConsoleNeoGeoCD, nil
	}

	// Check for PS2/PSX (SYSTEM.CNF)
	systemCnfPath := filepath.Join(path, "SYSTEM.CNF")
	//nolint:nestif // PS2/PSX detection requires checking file content
	if fileExists(systemCnfPath) {
		data, err := os.ReadFile(systemCnfPath) //nolint:gosec // Path constructed from validated directory
		if err == nil {
			content := strings.ToUpper(string(data))
			if strings.Contains(content, "BOOT2") {
				return identifier.ConsolePS2, nil
			}
			if strings.Contains(content, "BOOT") {
				return identifier.ConsolePSX, nil
			}
		}
	}

	return "", identifier.ErrNotSupported{Format: "directory"}
}

// detectConsoleFromHeader reads the file header to determine console type
func detectConsoleFromHeader(path, ext string) (identifier.Console, error) {
	// Handle CUE files specially
	if ext == ".cue" {
		return detectConsoleFromCue(path)
	}

	// Handle CHD files specially
	if ext == ".chd" {
		return detectConsoleFromCHD(path)
	}

	// Read header for analysis
	file, err := os.Open(path) //nolint:gosec // Path from user input is expected
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	header := make([]byte, 0x1000)
	bytesRead, err := file.Read(header)
	if err != nil {
		return "", fmt.Errorf("read header: %w", err)
	}
	header = header[:bytesRead]

	// Try various magic word checks

	// GameCube magic at 0x1C
	if len(header) > 0x20 && identifier.ValidateGC(header) {
		return identifier.ConsoleGC, nil
	}

	// Saturn magic
	if identifier.ValidateSaturn(header) {
		return identifier.ConsoleSaturn, nil
	}

	// Sega CD magic
	if identifier.ValidateSegaCD(header) {
		return identifier.ConsoleSegaCD, nil
	}

	// Genesis magic (check before trying as ISO)
	if identifier.ValidateGenesis(header) {
		return identifier.ConsoleGenesis, nil
	}

	// Try parsing as ISO9660
	iso, err := iso9660.Open(path)
	if err == nil {
		defer func() { _ = iso.Close() }()

		return detectConsoleFromISO(iso)
	}

	return "", identifier.ErrNotSupported{Format: ext}
}

// detectConsoleFromCHD handles CHD disc image detection.
func detectConsoleFromCHD(path string) (identifier.Console, error) {
	chdFile, err := chd.Open(path)
	if err != nil {
		return "", fmt.Errorf("open CHD: %w", err)
	}
	defer func() { _ = chdFile.Close() }()

	// Read first sectors for magic word detection
	reader := chdFile.RawSectorReader()
	header := make([]byte, 0x1000)
	if _, readErr := reader.ReadAt(header, 0); readErr != nil {
		return "", fmt.Errorf("read CHD header: %w", readErr)
	}

	// Check for Sega consoles first (they have magic words in raw sector data)
	if identifier.ValidateSaturn(header) {
		return identifier.ConsoleSaturn, nil
	}
	if identifier.ValidateSegaCD(header) {
		return identifier.ConsoleSegaCD, nil
	}

	// Check for GameCube (non-ISO9660 proprietary format)
	if identifier.ValidateGC(header) {
		return identifier.ConsoleGC, nil
	}

	// Try parsing as ISO9660 for PSX/PS2/PSP/NeoGeoCD
	iso, err := iso9660.OpenCHD(path)
	if err != nil {
		return "", fmt.Errorf("open CHD as ISO: %w", err)
	}
	defer func() { _ = iso.Close() }()

	return detectConsoleFromISO(iso)
}

// detectConsoleFromCue handles CUE sheet detection
func detectConsoleFromCue(path string) (identifier.Console, error) {
	cue, err := iso9660.ParseCue(path)
	if err != nil {
		return "", fmt.Errorf("parse CUE: %w", err)
	}

	if len(cue.BinFiles) == 0 {
		return "", identifier.ErrNotSupported{Format: "empty CUE"}
	}

	// Read header from first BIN file
	binFile, err := os.Open(cue.BinFiles[0])
	if err != nil {
		return "", fmt.Errorf("open BIN file: %w", err)
	}
	defer func() { _ = binFile.Close() }()

	header := make([]byte, 0x1000)
	bytesRead, _ := binFile.Read(header)
	header = header[:bytesRead]

	// Check for Sega consoles first (they have magic words in header)
	if identifier.ValidateSaturn(header) {
		return identifier.ConsoleSaturn, nil
	}
	if identifier.ValidateSegaCD(header) {
		return identifier.ConsoleSegaCD, nil
	}

	// Try as ISO
	iso, err := iso9660.OpenCue(path)
	if err != nil {
		return "", fmt.Errorf("open CUE as ISO: %w", err)
	}
	defer func() { _ = iso.Close() }()

	return detectConsoleFromISO(iso)
}

// detectConsoleFromISO detects console from ISO9660 filesystem.
//
//nolint:gocognit,revive // Console detection requires checking many conditions
func detectConsoleFromISO(iso *iso9660.ISO9660) (identifier.Console, error) {
	files, err := iso.IterFiles(true)
	if err != nil {
		return "", fmt.Errorf("iterate files: %w", err)
	}

	// Build list of root file names (uppercase)
	rootFiles := make([]string, 0, len(files))
	for _, fileInfo := range files {
		name := strings.ToUpper(filepath.Base(fileInfo.Path))
		// Remove version suffix
		if idx := strings.Index(name, ";"); idx != -1 {
			name = name[:idx]
		}
		rootFiles = append(rootFiles, name)
	}

	// Check for specific files
	for _, fileName := range rootFiles {
		switch fileName {
		case "UMD_DATA.BIN":
			return identifier.ConsolePSP, nil
		case "IPL.TXT":
			return identifier.ConsoleNeoGeoCD, nil
		case "SYSTEM.CNF":
			data, err := iso.ReadFileByPath("/SYSTEM.CNF")
			if err == nil {
				content := strings.ToUpper(string(data))
				if strings.Contains(content, "BOOT2") {
					return identifier.ConsolePS2, nil
				}
				if strings.Contains(content, "BOOT") {
					return identifier.ConsolePSX, nil
				}
			}
		}
	}

	// Default to PSX for ISO files without clear markers
	return identifier.ConsolePSX, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// detectConsoleFromBlockDevice detects the console type from a block device (physical disc).
//
//nolint:revive // Cognitive complexity required for block device detection logic
func detectConsoleFromBlockDevice(path string) (identifier.Console, error) {
	// Open the block device
	blockDev, err := os.Open(path) //nolint:gosec // Path from user input is expected for block device
	if err != nil {
		return "", fmt.Errorf("open block device: %w", err)
	}
	defer func() { _ = blockDev.Close() }()

	// Read header for initial checks (raw disc check for Sega consoles)
	header := make([]byte, 0x1000)
	bytesRead, err := blockDev.Read(header)
	if err != nil {
		return "", fmt.Errorf("read block device header: %w", err)
	}
	header = header[:bytesRead]

	// Check for Sega consoles (Saturn, SegaCD have magic words at start)
	if identifier.ValidateSaturn(header) {
		return identifier.ConsoleSaturn, nil
	}
	if identifier.ValidateSegaCD(header) {
		return identifier.ConsoleSegaCD, nil
	}

	// Try parsing as ISO9660 disc
	iso, err := iso9660.Open(path)
	if err == nil {
		defer func() { _ = iso.Close() }()

		return detectConsoleFromISO(iso)
	}

	// If ISO parsing fails, try at 2352 byte block offset (Mode2 raw sectors)
	// This is common for PSX/PS2 discs
	_, seekErr := blockDev.Seek(16, 0) // Skip sync pattern
	if seekErr == nil {
		bytesRead, readErr := blockDev.Read(header)
		if readErr == nil && bytesRead > 0 {
			iso, openErr := iso9660.Open(path)
			if openErr == nil {
				defer func() { _ = iso.Close() }()

				return detectConsoleFromISO(iso)
			}
		}
	}

	return "", identifier.ErrNotSupported{Format: "block device"}
}

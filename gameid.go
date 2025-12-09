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

// Package gameid provides game identification for various video game consoles.
// It can detect the console type from file extensions and headers, then extract
// game metadata from ROM/disc images.
package gameid

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/ZaparooProject/go-gameid/identifier"
)

// Result is an alias for identifier.Result for convenience.
type Result = identifier.Result

// Console is an alias for identifier.Console for convenience.
type Console = identifier.Console

// Re-export console constants for convenience.
const (
	ConsoleGB       = identifier.ConsoleGB
	ConsoleGBC      = identifier.ConsoleGBC
	ConsoleGBA      = identifier.ConsoleGBA
	ConsoleGC       = identifier.ConsoleGC
	ConsoleGenesis  = identifier.ConsoleGenesis
	ConsoleN64      = identifier.ConsoleN64
	ConsoleNeoGeoCD = identifier.ConsoleNeoGeoCD
	ConsoleNES      = identifier.ConsoleNES
	ConsolePSP      = identifier.ConsolePSP
	ConsolePSX      = identifier.ConsolePSX
	ConsolePS2      = identifier.ConsolePS2
	ConsoleSaturn   = identifier.ConsoleSaturn
	ConsoleSegaCD   = identifier.ConsoleSegaCD
	ConsoleSNES     = identifier.ConsoleSNES
)

// AllConsoles is a list of all supported consoles.
var AllConsoles = identifier.AllConsoles

// identifiers maps console types to their identifier implementations.
var identifiers = map[identifier.Console]identifier.Identifier{
	identifier.ConsoleGB:       identifier.NewGBIdentifier(),
	identifier.ConsoleGBC:      identifier.NewGBIdentifier(), // Same as GB
	identifier.ConsoleGBA:      identifier.NewGBAIdentifier(),
	identifier.ConsoleGC:       identifier.NewGCIdentifier(),
	identifier.ConsoleGenesis:  identifier.NewGenesisIdentifier(),
	identifier.ConsoleN64:      identifier.NewN64Identifier(),
	identifier.ConsoleNES:      identifier.NewNESIdentifier(),
	identifier.ConsoleSNES:     identifier.NewSNESIdentifier(),
	identifier.ConsolePSP:      identifier.NewPSPIdentifier(),
	identifier.ConsolePSX:      identifier.NewPSXIdentifier(),
	identifier.ConsolePS2:      identifier.NewPS2Identifier(),
	identifier.ConsoleSaturn:   identifier.NewSaturnIdentifier(),
	identifier.ConsoleSegaCD:   identifier.NewSegaCDIdentifier(),
	identifier.ConsoleNeoGeoCD: identifier.NewNeoGeoCDIdentifier(),
}

// pathIdentifiers are identifiers that need the file path rather than just a reader.
type pathIdentifier interface {
	IdentifyFromPath(path string, db identifier.Database) (*identifier.Result, error)
}

// Identify detects the console type and identifies the game at the given path.
// It returns the identification result or an error if identification fails.
// If db is nil, no database lookup is performed.
func Identify(path string, db *GameDatabase) (*Result, error) {
	console, err := DetectConsole(path)
	if err != nil {
		return nil, fmt.Errorf("failed to detect console: %w", err)
	}

	return IdentifyWithConsole(path, console, db)
}

// IdentifyWithConsole identifies the game at the given path using the specified console type.
// This is useful when the console is already known or when auto-detection fails.
func IdentifyWithConsole(path string, console Console, db *GameDatabase) (*Result, error) {
	id, ok := identifiers[console]
	if !ok {
		return nil, identifier.ErrNotSupported{Format: string(console)}
	}

	// Convert database to interface (nil-safe)
	var dbInterface identifier.Database
	if db != nil {
		dbInterface = db
	}

	// Check if it's a block device (physical disc)
	if isBlockDevice(path) {
		return identifyFromBlockDevice(path, console, id, dbInterface)
	}

	// Check if it's a directory (mounted disc)
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		return identifyFromDirectory(path, console, dbInterface)
	}

	// Check if this identifier needs the file path (disc-based games)
	if pid, ok := id.(pathIdentifier); ok {
		result, pathErr := pid.IdentifyFromPath(path, dbInterface)
		if pathErr != nil {
			return nil, fmt.Errorf("identify from path: %w", pathErr)
		}
		return result, nil
	}

	// Open file and identify using reader
	file, openErr := os.Open(path) //nolint:gosec // Path from user input is expected
	if openErr != nil {
		return nil, fmt.Errorf("failed to open file: %w", openErr)
	}
	defer func() { _ = file.Close() }()

	stat, statErr := file.Stat()
	if statErr != nil {
		return nil, fmt.Errorf("failed to stat file: %w", statErr)
	}

	result, idErr := id.Identify(file, stat.Size(), dbInterface)
	if idErr != nil {
		return nil, fmt.Errorf("identify: %w", idErr)
	}
	return result, nil
}

// identifyFromDirectory identifies a game from a mounted disc directory.
func identifyFromDirectory(path string, console Console, database identifier.Database) (*Result, error) {
	id, ok := identifiers[console]
	if !ok {
		return nil, identifier.ErrNotSupported{Format: string(console)}
	}

	// Check if identifier supports IdentifyFromPath (disc-based games)
	if pid, ok := id.(pathIdentifier); ok {
		result, err := pid.IdentifyFromPath(path, database)
		if err != nil {
			return nil, fmt.Errorf("identify from path: %w", err)
		}
		return result, nil
	}

	// Cartridge-based consoles don't support directories
	return nil, identifier.ErrNotSupported{Format: "mounted directory for " + string(console)}
}

// IdentifyFromReader identifies a game from an io.ReaderAt.
// This is useful when the file is already open or when reading from non-file sources.
// size is the total size of the data.
func IdentifyFromReader(
	reader interface {
		ReadAt([]byte, int64) (int, error)
	},
	size int64,
	console Console,
	database *GameDatabase,
) (*Result, error) {
	id, ok := identifiers[console]
	if !ok {
		return nil, identifier.ErrNotSupported{Format: string(console)}
	}

	var dbInterface identifier.Database
	if database != nil {
		dbInterface = database
	}

	result, err := id.Identify(reader, size, dbInterface)
	if err != nil {
		return nil, fmt.Errorf("identify: %w", err)
	}
	return result, nil
}

// ParseConsole parses a console name string into a Console type.
// It is case-insensitive and accepts various common names.
func ParseConsole(name string) (Console, error) {
	name = strings.ToUpper(strings.TrimSpace(name))

	// Direct matches
	switch name {
	case "GB", "GAMEBOY":
		return ConsoleGB, nil
	case "GBC", "GAMEBOYCOLOR":
		return ConsoleGBC, nil
	case "GBA", "GAMEBOYADVANCE":
		return ConsoleGBA, nil
	case "GC", "GAMECUBE", "NGC":
		return ConsoleGC, nil
	case "GENESIS", "MEGADRIVE", "MD":
		return ConsoleGenesis, nil
	case "N64", "NINTENDO64":
		return ConsoleN64, nil
	case "NEOGEOCD", "NEOCD", "NGCD":
		return ConsoleNeoGeoCD, nil
	case "NES", "FAMICOM", "FC":
		return ConsoleNES, nil
	case "PSP", "PLAYSTATIONPORTABLE":
		return ConsolePSP, nil
	case "PSX", "PS1", "PLAYSTATION", "PLAYSTATION1":
		return ConsolePSX, nil
	case "PS2", "PLAYSTATION2":
		return ConsolePS2, nil
	case "SATURN", "SEGASATURN", "SS":
		return ConsoleSaturn, nil
	case "SEGACD", "MEGACD", "SCD", "MCD":
		return ConsoleSegaCD, nil
	case "SNES", "SUPERFAMICOM", "SFC":
		return ConsoleSNES, nil
	}

	return "", identifier.ErrNotSupported{Format: name}
}

// SupportedConsoles returns a list of all supported console names.
func SupportedConsoles() []string {
	result := make([]string, len(AllConsoles))
	for i, c := range AllConsoles {
		result[i] = string(c)
	}
	return result
}

// IsDiscBased returns true if the console uses disc-based media.
func IsDiscBased(console Console) bool {
	switch console {
	case ConsoleGC, ConsoleNeoGeoCD, ConsolePSP, ConsolePSX, ConsolePS2, ConsoleSaturn, ConsoleSegaCD:
		return true
	default:
		return false
	}
}

// IsCartridgeBased returns true if the console uses cartridge-based media.
func IsCartridgeBased(console Console) bool {
	return !IsDiscBased(console)
}

// isBlockDevice checks if the given path is a block device (e.g., /dev/sr0).
func isBlockDevice(path string) bool {
	// On Linux, block devices are typically in /dev/
	if !strings.HasPrefix(path, "/dev/") {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	// Check if it's a block device using syscall
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	// S_IFBLK = block device (0x6000 = 0o60000)
	return stat.Mode&syscall.S_IFMT == syscall.S_IFBLK
}

// identifyFromBlockDevice identifies a game from a physical disc (block device).
//
//nolint:revive // Line length acceptable for function signature with ignored parameter
func identifyFromBlockDevice(path string, _ Console, ident identifier.Identifier, database identifier.Database) (*Result, error) {
	// For disc-based consoles, use IdentifyFromPath which handles block devices
	if pid, ok := ident.(pathIdentifier); ok {
		result, err := pid.IdentifyFromPath(path, database)
		if err != nil {
			return nil, fmt.Errorf("identify from path: %w", err)
		}
		return result, nil
	}

	// Open block device directly
	blockDev, err := os.Open(path) //nolint:gosec // Path from user input is expected for block device
	if err != nil {
		return nil, fmt.Errorf("failed to open block device: %w", err)
	}
	defer func() { _ = blockDev.Close() }()

	// Get device size (for block devices, we need to use ioctl or read to end)
	// For now, use a reasonable default size for disc identification
	// Most identifiers only need the first few KB
	size := int64(700 * 1024 * 1024) // 700MB typical CD size

	result, err := ident.Identify(blockDev, size, database)
	if err != nil {
		return nil, fmt.Errorf("identify: %w", err)
	}
	return result, nil
}

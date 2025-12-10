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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ZaparooProject/go-gameid/chd"
	"github.com/ZaparooProject/go-gameid/internal/binary"
	"github.com/ZaparooProject/go-gameid/iso9660"
)

// Saturn magic word
var saturnMagicWord = []byte("SEGA SEGASATURN")

// Saturn device support codes
var saturnDeviceSupport = map[byte]string{
	'J': "Joypad",
	'M': "Mouse",
	'G': "Gun",
	'W': "RAM Cart",
	'S': "Steering Wheel",
	'A': "Virtua Stick or Analog Controller",
	'E': "Analog Controller (3D-pad)",
	'T': "Multi-Tap",
	'C': "Link Cable",
	'D': "Link Cable (Direct Link)",
	'X': "X-Band or Netlink Modem",
	'K': "Keyboard",
	'Q': "Pachinko Controller",
	'F': "Floppy Disk Drive",
	'R': "ROM Cart",
	'P': "Video CD Card (MPEG Movie Card)",
}

// Saturn target area codes
var saturnTargetAreas = map[byte]string{
	'J': "Japan",
	'T': "Asia NTSC (Taiwan, Philippines)",
	'U': "North America (USA, Canada)",
	'B': "Central and South America NTSC (Brazil)",
	'K': "Korea",
	'A': "East Asia PAL (China, Middle and Near East)",
	'E': "Europe PAL",
	'L': "Central and South America PAL",
}

// SaturnIdentifier identifies Sega Saturn games.
type SaturnIdentifier struct{}

// NewSaturnIdentifier creates a new Saturn identifier.
func NewSaturnIdentifier() *SaturnIdentifier {
	return &SaturnIdentifier{}
}

// Console returns the console type.
func (*SaturnIdentifier) Console() Console {
	return ConsoleSaturn
}

// Identify extracts Saturn game information from the given reader.
func (s *SaturnIdentifier) Identify(reader io.ReaderAt, size int64, db Database) (*Result, error) {
	if size < 0x100 {
		return nil, ErrInvalidFormat{Console: ConsoleSaturn, Reason: "file too small"}
	}

	// Read header
	header, err := binary.ReadBytesAt(reader, 0, 0x100)
	if err != nil {
		return nil, fmt.Errorf("failed to read Saturn header: %w", err)
	}

	return s.identifyFromHeader(header, db)
}

// IdentifyFromPath identifies a Saturn game from a file path.
//
//nolint:gocognit,revive // CUE/CHD/ISO handling requires separate branches
func (s *SaturnIdentifier) IdentifyFromPath(path string, database Database) (*Result, error) {
	ext := strings.ToLower(filepath.Ext(path))

	var header []byte

	switch ext {
	case ".cue":
		cue, err := iso9660.ParseCue(path)
		if err != nil {
			return nil, fmt.Errorf("parse CUE: %w", err)
		}
		if len(cue.BinFiles) == 0 {
			return nil, ErrInvalidFormat{Console: ConsoleSaturn, Reason: "no BIN files in CUE"}
		}
		binFile, err := os.Open(cue.BinFiles[0])
		if err != nil {
			return nil, fmt.Errorf("open BIN file: %w", err)
		}
		defer func() { _ = binFile.Close() }()
		header = make([]byte, 0x100)
		if _, err := binFile.Read(header); err != nil {
			return nil, fmt.Errorf("read BIN header: %w", err)
		}

	case ".chd":
		chdFile, err := chd.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open CHD: %w", err)
		}
		defer func() { _ = chdFile.Close() }()
		header = make([]byte, 0x100)
		reader := chdFile.RawSectorReader()
		if _, err := reader.ReadAt(header, 0); err != nil {
			return nil, fmt.Errorf("read CHD header: %w", err)
		}

	default:
		isoFile, err := os.Open(path) //nolint:gosec // Path from user input is expected
		if err != nil {
			return nil, fmt.Errorf("open ISO file: %w", err)
		}
		defer func() { _ = isoFile.Close() }()
		header = make([]byte, 0x100)
		if _, err := isoFile.Read(header); err != nil {
			return nil, fmt.Errorf("read ISO header: %w", err)
		}
	}

	return s.identifyFromHeader(header, database)
}

func (*SaturnIdentifier) identifyFromHeader(header []byte, db Database) (*Result, error) {
	// Find magic word
	magicIdx := binary.FindBytes(header, saturnMagicWord)
	if magicIdx == -1 {
		return nil, ErrInvalidFormat{Console: ConsoleSaturn, Reason: "magic word not found"}
	}

	// Extract fields relative to magic word position
	extractString := func(offset, length int) string {
		start := magicIdx + offset
		end := start + length
		if end > len(header) {
			return ""
		}
		return strings.TrimSpace(string(header[start:end]))
	}

	manufacturerID := extractString(0x10, 0x10)
	gameID := extractString(0x20, 0x0A)
	// Split on space and take first part
	if idx := strings.Index(gameID, " "); idx != -1 {
		gameID = gameID[:idx]
	}
	version := extractString(0x2A, 0x06)
	deviceInfo := extractString(0x38, 0x08)
	internalTitle := extractString(0x60, 0x70)

	// Release date (YYYYMMDD at offset 0x30)
	releaseDate := parseSaturnReleaseDate(extractString(0x30, 0x08))

	// Device support (offset 0x50, 16 bytes)
	deviceSupport := parseSaturnDeviceSupport(header, magicIdx)

	// Target area (offset 0x40, 16 bytes)
	targetArea := parseSaturnTargetArea(header, magicIdx)

	// Normalize serial for database lookup
	serial := strings.ReplaceAll(gameID, "-", "")
	serial = strings.ReplaceAll(serial, " ", "")
	serial = strings.TrimSpace(serial)

	result := NewResult(ConsoleSaturn)
	result.ID = gameID
	result.InternalTitle = internalTitle
	result.SetMetadata("manufacturer_ID", manufacturerID)
	result.SetMetadata("ID", gameID)
	result.SetMetadata("version", version)
	result.SetMetadata("device_info", deviceInfo)
	result.SetMetadata("internal_title", internalTitle)

	if releaseDate != "" {
		result.SetMetadata("release_date", releaseDate)
	}

	if len(deviceSupport) > 0 {
		result.SetMetadata("device_support", strings.Join(deviceSupport, " / "))
	}

	if len(targetArea) > 0 {
		result.SetMetadata("target_area", strings.Join(targetArea, " / "))
	}

	// Database lookup
	if db != nil && serial != "" {
		if entry, found := db.LookupByString(ConsoleSaturn, serial); found {
			result.MergeMetadata(entry)
		}
	}

	// If no title from database, use internal title
	if result.Title == "" {
		result.Title = result.InternalTitle
	}

	return result, nil
}

// parseSaturnReleaseDate parses YYYYMMDD format to YYYY-MM-DD.
func parseSaturnReleaseDate(raw string) string {
	if len(raw) == 8 {
		return fmt.Sprintf("%s-%s-%s", raw[0:4], raw[4:6], raw[6:8])
	}
	return ""
}

// parseSaturnDeviceSupport extracts device support codes from header.
func parseSaturnDeviceSupport(header []byte, magicIdx int) []string {
	var deviceSupport []string
	if magicIdx+0x60 > len(header) {
		return deviceSupport
	}
	for _, b := range header[magicIdx+0x50 : magicIdx+0x60] {
		if b == 0 || b == ' ' {
			continue
		}
		if dev, ok := saturnDeviceSupport[b]; ok {
			deviceSupport = append(deviceSupport, dev)
		}
	}
	return deviceSupport
}

// parseSaturnTargetArea extracts target area codes from header.
func parseSaturnTargetArea(header []byte, magicIdx int) []string {
	var targetArea []string
	if magicIdx+0x50 > len(header) {
		return targetArea
	}
	for _, b := range header[magicIdx+0x40 : magicIdx+0x50] {
		if b == 0 || b == ' ' {
			continue
		}
		if area, ok := saturnTargetAreas[b]; ok {
			targetArea = append(targetArea, area)
		}
	}
	return targetArea
}

// ValidateSaturn checks if the given data looks like a valid Saturn disc.
func ValidateSaturn(header []byte) bool {
	return binary.FindBytes(header, saturnMagicWord) != -1
}

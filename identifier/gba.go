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

	"github.com/ZaparooProject/go-gameid/internal/binary"
)

// GBA header offsets
const (
	gbaHeaderSize         = 192
	gbaNintendoLogoOffset = 0x04
	gbaNintendoLogoSize   = 156
	gbaTitleOffset        = 0xA0
	gbaTitleSize          = 12
	gbaGameCodeOffset     = 0xAC
	gbaGameCodeSize       = 4
	gbaMakerCodeOffset    = 0xB0
	gbaMakerCodeSize      = 2
	gbaMainUnitCodeOffset = 0xB3
	gbaDeviceTypeOffset   = 0xB4
	gbaSoftwareVerOffset  = 0xBC
)

// GBA Nintendo logo - used to validate GBA ROMs
var gbaNintendoLogo = []byte{
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

// GBAIdentifier identifies Game Boy Advance games.
type GBAIdentifier struct{}

// NewGBAIdentifier creates a new GBA identifier.
func NewGBAIdentifier() *GBAIdentifier {
	return &GBAIdentifier{}
}

// Console returns the console type.
func (*GBAIdentifier) Console() Console {
	return ConsoleGBA
}

// Identify extracts GBA game information from the given reader.
func (*GBAIdentifier) Identify(reader io.ReaderAt, size int64, db Database) (*Result, error) {
	if size < gbaHeaderSize {
		return nil, ErrInvalidFormat{Console: ConsoleGBA, Reason: "file too small"}
	}

	// Read header
	header, err := binary.ReadBytesAt(reader, 0, gbaHeaderSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read GBA header: %w", err)
	}

	// Validate Nintendo logo (optional - some homebrew may not have it)
	// Not a fatal error if invalid, just note it - some valid GBA ROMs may have modified logos
	// logo := header[gbaNintendoLogoOffset : gbaNintendoLogoOffset+gbaNintendoLogoSize]
	// _ = binary.BytesEqual(logo, gbaNintendoLogo)

	// Extract title (12 bytes at 0xA0)
	title := binary.ExtractPrintable(header[gbaTitleOffset : gbaTitleOffset+gbaTitleSize])

	// Extract game code (4 bytes at 0xAC)
	gameCode := binary.ExtractPrintable(header[gbaGameCodeOffset : gbaGameCodeOffset+gbaGameCodeSize])

	// Extract maker code (2 bytes at 0xB0)
	makerCode := binary.ExtractPrintable(header[gbaMakerCodeOffset : gbaMakerCodeOffset+gbaMakerCodeSize])

	// Extract other fields
	mainUnitCode := header[gbaMainUnitCodeOffset]
	deviceType := header[gbaDeviceTypeOffset]
	softwareVersion := header[gbaSoftwareVerOffset]

	result := NewResult(ConsoleGBA)
	result.ID = gameCode
	result.InternalTitle = title
	result.SetMetadata("ID", gameCode)
	result.SetMetadata("internal_title", title)
	result.SetMetadata("maker_code", makerCode)
	result.SetMetadata("main_unit_code", fmt.Sprintf("0x%02x", mainUnitCode))
	result.SetMetadata("device_type", fmt.Sprintf("0x%02x", deviceType))
	result.SetMetadata("software_version", fmt.Sprintf("%d", softwareVersion))

	// Database lookup
	if db != nil && gameCode != "" {
		if entry, found := db.LookupByString(ConsoleGBA, gameCode); found {
			result.MergeMetadata(entry)
		}
	}

	// If no title from database, use internal title
	if result.Title == "" {
		result.Title = result.InternalTitle
	}

	return result, nil
}

// ValidateGBA checks if the given data looks like a valid GBA ROM.
func ValidateGBA(header []byte) bool {
	if len(header) < gbaHeaderSize {
		return false
	}
	logo := header[gbaNintendoLogoOffset : gbaNintendoLogoOffset+gbaNintendoLogoSize]
	return binary.BytesEqual(logo, gbaNintendoLogo)
}

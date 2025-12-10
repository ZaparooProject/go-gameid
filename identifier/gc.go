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
	"path/filepath"
	"strings"

	"github.com/ZaparooProject/go-gameid/chd"
	"github.com/ZaparooProject/go-gameid/internal/binary"
)

// GameCube header offsets
const (
	gcHeaderSize         = 0x0440
	gcGameIDOffset       = 0x0000
	gcGameIDSize         = 4
	gcMakerCodeOffset    = 0x0004
	gcMakerCodeSize      = 2
	gcDiskIDOffset       = 0x0006
	gcVersionOffset      = 0x0007
	gcInternalNameOffset = 0x0020
	gcInternalNameSize   = 0x03E0 // 0x0020 to 0x0400
)

// GameCube magic word at offset 0x1C
var gcMagicWord = []byte{0xC2, 0x33, 0x9F, 0x3D}

// GCIdentifier identifies GameCube games.
type GCIdentifier struct{}

// NewGCIdentifier creates a new GameCube identifier.
func NewGCIdentifier() *GCIdentifier {
	return &GCIdentifier{}
}

// Console returns the console type.
func (*GCIdentifier) Console() Console {
	return ConsoleGC
}

// Identify extracts GameCube game information from the given reader.
func (*GCIdentifier) Identify(reader io.ReaderAt, size int64, db Database) (*Result, error) {
	if size < gcHeaderSize {
		return nil, ErrInvalidFormat{Console: ConsoleGC, Reason: "file too small"}
	}

	// Read header
	header, err := binary.ReadBytesAt(reader, 0, gcHeaderSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read GameCube header: %w", err)
	}

	// Validate magic word (at offset 0x1C)
	magic := header[0x1C : 0x1C+4]
	if !binary.BytesEqual(magic, gcMagicWord) {
		return nil, ErrInvalidFormat{Console: ConsoleGC, Reason: "invalid magic word"}
	}

	// Extract game ID (4 bytes at 0x0000)
	gameID := binary.CleanString(header[gcGameIDOffset : gcGameIDOffset+gcGameIDSize])

	// Extract maker code (2 bytes at 0x0004)
	makerCode := binary.CleanString(header[gcMakerCodeOffset : gcMakerCodeOffset+gcMakerCodeSize])

	// Extract disk ID and version
	diskID := header[gcDiskIDOffset]
	version := header[gcVersionOffset]

	// Extract internal title (0x0020 to 0x0400)
	internalTitle := binary.CleanString(header[gcInternalNameOffset : gcInternalNameOffset+gcInternalNameSize])

	result := NewResult(ConsoleGC)
	result.ID = gameID
	result.InternalTitle = internalTitle
	result.SetMetadata("ID", gameID)
	result.SetMetadata("maker_code", makerCode)
	result.SetMetadata("disk_ID", fmt.Sprintf("%d", diskID))
	result.SetMetadata("version", fmt.Sprintf("%d", version))
	result.SetMetadata("internal_title", internalTitle)

	// Database lookup
	if db != nil && gameID != "" {
		if entry, found := db.LookupByString(ConsoleGC, gameID); found {
			result.MergeMetadata(entry)
		}
	}

	// If no title from database, use internal title
	if result.Title == "" {
		result.Title = result.InternalTitle
	}

	return result, nil
}

// ValidateGC checks if the given data looks like a valid GameCube disc.
func ValidateGC(header []byte) bool {
	if len(header) < 0x20 {
		return false
	}
	magic := header[0x1C : 0x1C+4]
	return binary.BytesEqual(magic, gcMagicWord)
}

// IdentifyFromPath handles path-based identification for GameCube discs.
// This is needed for CHD files which require special handling.
func (g *GCIdentifier) IdentifyFromPath(path string, db Database) (*Result, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".chd" {
		return g.identifyFromCHD(path, db)
	}

	// For non-CHD files, fall back to standard file reading
	return nil, ErrNotSupported{Format: "use standard Identify for non-CHD files"}
}

// identifyFromCHD reads GameCube disc data from a CHD file.
func (g *GCIdentifier) identifyFromCHD(path string, db Database) (*Result, error) {
	chdFile, err := chd.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open CHD: %w", err)
	}
	defer func() { _ = chdFile.Close() }()

	// GameCube discs don't use ISO9660 - read raw sector data
	reader := chdFile.RawSectorReader()
	size := chdFile.Size()

	return g.Identify(reader, size, db)
}

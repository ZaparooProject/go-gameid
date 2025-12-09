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

	"github.com/ZaparooProject/go-gameid/internal/binary"
	"github.com/ZaparooProject/go-gameid/iso9660"
)

// SegaCD magic words
var segaCDMagicWords = [][]byte{
	[]byte("SEGADISCSYSTEM"),
	[]byte("SEGABOOTDISC"),
	[]byte("SEGADISC"),
	[]byte("SEGADATADISC"),
}

// SegaCDIdentifier identifies Sega CD games.
type SegaCDIdentifier struct{}

// NewSegaCDIdentifier creates a new Sega CD identifier.
func NewSegaCDIdentifier() *SegaCDIdentifier {
	return &SegaCDIdentifier{}
}

// Console returns the console type.
func (*SegaCDIdentifier) Console() Console {
	return ConsoleSegaCD
}

// Identify extracts Sega CD game information from the given reader.
func (s *SegaCDIdentifier) Identify(reader io.ReaderAt, size int64, db Database) (*Result, error) {
	if size < 0x300 {
		return nil, ErrInvalidFormat{Console: ConsoleSegaCD, Reason: "file too small"}
	}

	// Read header
	header, err := binary.ReadBytesAt(reader, 0, 0x300)
	if err != nil {
		return nil, fmt.Errorf("failed to read Sega CD header: %w", err)
	}

	return s.identifyFromHeader(header, db, nil)
}

// IdentifyFromPath identifies a Sega CD game from a file path.
func (s *SegaCDIdentifier) IdentifyFromPath(path string, database Database) (*Result, error) {
	ext := strings.ToLower(filepath.Ext(path))

	if ext == ".cue" {
		return s.identifyFromCue(path, database)
	}
	return s.identifyFromISO(path, database)
}

func (s *SegaCDIdentifier) identifyFromCue(path string, database Database) (*Result, error) {
	cue, err := iso9660.ParseCue(path)
	if err != nil {
		return nil, fmt.Errorf("parse CUE: %w", err)
	}
	if len(cue.BinFiles) == 0 {
		return nil, ErrInvalidFormat{Console: ConsoleSegaCD, Reason: "no BIN files in CUE"}
	}
	binFile, err := os.Open(cue.BinFiles[0])
	if err != nil {
		return nil, fmt.Errorf("open BIN file: %w", err)
	}
	defer func() { _ = binFile.Close() }()

	header := make([]byte, 0x300)
	if _, err := binFile.Read(header); err != nil {
		return nil, fmt.Errorf("read BIN header: %w", err)
	}

	iso, _ := iso9660.OpenCue(path)
	if iso != nil {
		defer func() { _ = iso.Close() }()
	}

	return s.identifyFromHeader(header, database, iso)
}

func (s *SegaCDIdentifier) identifyFromISO(path string, database Database) (*Result, error) {
	isoFile, err := os.Open(path) //nolint:gosec // Path from user input is expected
	if err != nil {
		return nil, fmt.Errorf("open ISO file: %w", err)
	}
	defer func() { _ = isoFile.Close() }()

	header := make([]byte, 0x300)
	if _, err := isoFile.Read(header); err != nil {
		return nil, fmt.Errorf("read ISO header: %w", err)
	}

	iso, _ := iso9660.Open(path)
	if iso != nil {
		defer func() { _ = iso.Close() }()
	}

	return s.identifyFromHeader(header, database, iso)
}

//nolint:funlen,revive // Header parsing requires many field extractions
func (s *SegaCDIdentifier) identifyFromHeader(header []byte, db Database, iso *iso9660.ISO9660) (*Result, error) {
	// Find magic word
	magicIdx := findSegaCDMagicWord(header)
	if magicIdx == -1 {
		return nil, ErrInvalidFormat{Console: ConsoleSegaCD, Reason: "magic word not found"}
	}

	// Extract fields relative to magic word position (same as Genesis layout)
	extractString := func(offset, length int) string {
		start := magicIdx + offset
		end := start + length
		if end > len(header) {
			return ""
		}
		return strings.TrimSpace(string(header[start:end]))
	}

	discID := extractString(0x000, 0x10)
	discVolumeName := extractString(0x010, 0x0B)
	systemName := extractString(0x020, 0x0B)

	// Build date at 0x50 (MMDDYYYY format)
	buildDate := parseSegaCDBuildDate(extractString(0x050, 0x08))

	// System type and release info (at 0x100+ like Genesis)
	systemType := extractString(0x100, 0x10)
	releaseYear := extractString(0x118, 0x04)
	releaseMonth := extractString(0x11D, 0x03)

	// Titles
	titleDomestic := extractString(0x120, 0x30)
	titleOverseas := extractString(0x150, 0x30)

	// Software type and ID
	gameID := extractString(0x180, 0x10)

	// Device support
	deviceSupport := parseSegaCDDeviceSupport(header, magicIdx)

	// Region support (at 0x1F0 from magic word for Genesis layout)
	regionSupport := parseSegaCDRegionSupport(header, magicIdx)

	// Normalize serial for database lookup
	serial := strings.ReplaceAll(gameID, "#", "")
	serial = strings.ReplaceAll(serial, "-", "")
	serial = strings.ReplaceAll(serial, " ", "")
	serial = strings.TrimSpace(serial)

	result := NewResult(ConsoleSegaCD)
	result.ID = gameID
	result.InternalTitle = titleOverseas
	if result.InternalTitle == "" {
		result.InternalTitle = titleDomestic
	}

	result.SetMetadata("disc_ID", discID)
	result.SetMetadata("disc_volume_name", discVolumeName)
	result.SetMetadata("system_name", systemName)
	result.SetMetadata("build_date", buildDate)
	result.SetMetadata("system_type", systemType)
	result.SetMetadata("release_year", releaseYear)
	result.SetMetadata("release_month", releaseMonth)
	result.SetMetadata("title_domestic", titleDomestic)
	result.SetMetadata("title_overseas", titleOverseas)
	result.SetMetadata("ID", gameID)

	if len(deviceSupport) > 0 {
		result.SetMetadata("device_support", strings.Join(deviceSupport, " / "))
	}

	if len(regionSupport) > 0 {
		result.SetMetadata("region_support", strings.Join(regionSupport, " / "))
	}

	// Add ISO metadata if available
	if iso != nil {
		result.SetMetadata("uuid", iso.GetUUID())
		result.SetMetadata("volume_ID", iso.GetVolumeID())
	}

	// Database lookup
	if db != nil && serial != "" {
		if entry, found := db.LookupByString(ConsoleSegaCD, serial); found {
			result.MergeMetadata(entry)
		}
	}

	// If no title from database, use overseas title
	if result.Title == "" {
		result.Title = result.InternalTitle
	}

	return result, nil
}

// findSegaCDMagicWord searches for a Sega CD magic word in the header.
func findSegaCDMagicWord(header []byte) int {
	for _, magic := range segaCDMagicWords {
		idx := binary.FindBytes(header, magic)
		if idx != -1 {
			return idx
		}
	}
	return -1
}

// parseSegaCDBuildDate parses MMDDYYYY format to YYYY-MM-DD.
func parseSegaCDBuildDate(raw string) string {
	if len(raw) == 8 {
		return raw[4:8] + "-" + raw[0:2] + "-" + raw[2:4]
	}
	return raw
}

// parseSegaCDDeviceSupport extracts device support codes from header.
func parseSegaCDDeviceSupport(header []byte, magicIdx int) []string {
	var deviceSupport []string
	deviceSupportBytes := header[magicIdx+0x190 : magicIdx+0x1A0]
	for _, b := range deviceSupportBytes {
		if b == 0 || b == ' ' {
			continue
		}
		if dev, ok := genesisDeviceSupport[b]; ok {
			deviceSupport = append(deviceSupport, dev)
		}
	}
	return deviceSupport
}

// parseSegaCDRegionSupport extracts region support codes from header.
func parseSegaCDRegionSupport(header []byte, magicIdx int) []string {
	var regionSupport []string
	if magicIdx+0x1F3 > len(header) {
		return regionSupport
	}
	for _, b := range header[magicIdx+0x1F0 : magicIdx+0x1F3] {
		if b < '!' || b > '~' {
			continue
		}
		if reg, ok := genesisRegionSupport[b]; ok {
			regionSupport = append(regionSupport, reg)
		}
	}
	return regionSupport
}

// ValidateSegaCD checks if the given data looks like a valid Sega CD disc.
func ValidateSegaCD(header []byte) bool {
	for _, magic := range segaCDMagicWords {
		if binary.FindBytes(header, magic) != -1 {
			return true
		}
	}
	return false
}

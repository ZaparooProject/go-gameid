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

	"github.com/ZaparooProject/go-gameid/iso9660"
)

// playstationISO is the interface for PlayStation disc images.
type playstationISO interface {
	GetUUID() string
	GetVolumeID() string
	IterFiles(onlyRootDir bool) ([]iso9660.FileInfo, error)
	Close() error
}

// openPlayStationISO opens an ISO from a path, handling CUE and CHD files.
func openPlayStationISO(path string) (playstationISO, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".cue":
		iso, err := iso9660.OpenCue(path)
		if err != nil {
			return nil, fmt.Errorf("open CUE: %w", err)
		}
		return iso, nil

	case ".chd":
		iso, err := iso9660.OpenCHD(path)
		if err != nil {
			return nil, fmt.Errorf("open CHD: %w", err)
		}
		return iso, nil

	default:
		iso, err := iso9660.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open ISO: %w", err)
		}
		return iso, nil
	}
}

// identifyPlayStation identifies a PlayStation game from an ISO.
func identifyPlayStation(
	iso playstationISO,
	console Console,
	database Database,
	sourcePath string,
) (*Result, error) {
	result := NewResult(console)

	// Get root files
	files, err := iso.IterFiles(true)
	if err != nil {
		return nil, fmt.Errorf("iterate files: %w", err)
	}

	// Build list of root filenames
	rootFiles := make([]string, 0, len(files))
	for _, f := range files {
		name := strings.TrimPrefix(f.Path, "/")
		// Remove version suffix (;1)
		if idx := strings.Index(name, ";"); idx != -1 {
			name = name[:idx]
		}
		rootFiles = append(rootFiles, name)
	}

	// Try to find serial from root files using ID prefixes
	serial := findPlayStationSerial(rootFiles, console, database)

	// Fallback to volume ID
	if serial == "" {
		serial = serialFromVolumeID(iso.GetVolumeID())
	}

	// Fallback to filename
	if serial == "" && sourcePath != "" {
		serial = serialFromFilename(sourcePath)
	}

	result.ID = strings.ReplaceAll(serial, "_", "-")
	result.SetMetadata("ID", result.ID)
	result.SetMetadata("uuid", iso.GetUUID())
	result.SetMetadata("volume_ID", iso.GetVolumeID())
	result.SetMetadata("root_files", strings.Join(rootFiles, " / "))

	// Database lookup
	if database != nil && serial != "" {
		if entry, found := database.LookupByString(console, serial); found {
			result.MergeMetadata(entry)
		}
	}

	return result, nil
}

// findPlayStationSerial searches for serial in root files using ID prefixes.
func findPlayStationSerial(rootFiles []string, console Console, database Database) string {
	if database == nil {
		return ""
	}

	prefixes := database.GetIDPrefixes(console)
	for _, prefix := range prefixes {
		if serial := findSerialWithPrefix(rootFiles, prefix, console, database); serial != "" {
			return serial
		}
	}
	return ""
}

// findSerialWithPrefix searches for a serial matching the given prefix.
func findSerialWithPrefix(rootFiles []string, prefix string, console Console, database Database) string {
	for _, fileName := range rootFiles {
		fnUpper := strings.ToUpper(fileName)
		if !strings.HasPrefix(fnUpper, prefix) {
			continue
		}

		if serial := trySerialLookup(fnUpper, prefix, console, database); serial != "" {
			return serial
		}
	}
	return ""
}

// trySerialLookup attempts to look up a serial in the database.
func trySerialLookup(fnUpper, prefix string, console Console, database Database) string {
	// Normalize: remove dots, replace dashes with underscores
	serial := strings.ReplaceAll(fnUpper, ".", "")
	serial = strings.ReplaceAll(serial, "-", "_")

	// Try lookup
	if _, found := database.LookupByString(console, serial); found {
		return serial
	}

	// Try with underscore after prefix
	if len(serial) > len(prefix) {
		altSerial := serial[:len(prefix)] + "_" + serial[len(prefix)+1:]
		if _, found := database.LookupByString(console, altSerial); found {
			return altSerial
		}
	}

	return ""
}

// serialFromVolumeID extracts serial from volume ID.
func serialFromVolumeID(volumeID string) string {
	if volumeID == "" {
		return ""
	}
	serial := strings.ReplaceAll(volumeID, "-", "_")
	// If there are 2 underscores, keep only first 2 parts
	parts := strings.Split(serial, "_")
	if len(parts) > 2 {
		serial = strings.Join(parts[:2], "_")
	}
	return serial
}

// serialFromFilename extracts serial from filename.
func serialFromFilename(sourcePath string) string {
	fileName := filepath.Base(sourcePath)
	fileName = strings.TrimSuffix(fileName, filepath.Ext(fileName))
	fileName = strings.TrimSuffix(fileName, ".gz")
	return fileName
}

// PSXIdentifier identifies PlayStation games.
type PSXIdentifier struct{}

// NewPSXIdentifier creates a new PSX identifier.
func NewPSXIdentifier() *PSXIdentifier {
	return &PSXIdentifier{}
}

// Console returns the console type.
func (*PSXIdentifier) Console() Console {
	return ConsolePSX
}

// Identify extracts PSX game information from the given reader.
// For disc-based games, use IdentifyFromPath instead.
func (*PSXIdentifier) Identify(_ io.ReaderAt, _ int64, _ Database) (*Result, error) {
	return nil, ErrNotSupported{Format: "raw reader for PSX"}
}

// IdentifyFromPath identifies a PSX game from a file path.
func (*PSXIdentifier) IdentifyFromPath(path string, database Database) (*Result, error) {
	iso, err := openPlayStationISO(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = iso.Close() }()

	return identifyPlayStation(iso, ConsolePSX, database, path)
}

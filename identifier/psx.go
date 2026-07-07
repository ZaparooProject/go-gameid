// Copyright (c) 2026 Niema Moshiri and The Zaparoo Project.
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
	"unicode"

	"github.com/ZaparooProject/go-gameid/iso9660"
)

var playStationSerialPrefixes = map[string]struct{}{
	"PAPX": {},
	"PBPX": {},
	"PCPX": {},
	"SCES": {},
	"SCPS": {},
	"SCUS": {},
	"SLES": {},
	"SLKA": {},
	"SLPM": {},
	"SLPS": {},
	"SLUS": {},
	"TCPS": {},
}

// playstationISO is the interface for PlayStation disc images.
type playstationISO interface {
	GetUUID() string
	GetVolumeID() string
	IterFiles(onlyRootDir bool) ([]iso9660.FileInfo, error)
	Close() error
}

type rootFileWalker interface {
	WalkFiles(onlyRootDir bool, fn func(iso9660.FileInfo) bool) error
}

type isoFileReader interface {
	ReadFile(info iso9660.FileInfo) ([]byte, error)
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

	rootFiles, serial, err := playStationRootInfo(iso, console, database)
	if err != nil {
		return nil, err
	}

	// Fallback to volume ID. Unlike upstream GameID this is accepted without
	// a database match: the volume ID is read from the disc itself, so an
	// image and the physical disc it was dumped from still agree on it.
	if serial == "" {
		serial = serialFromVolumeID(iso.GetVolumeID())
	}

	// Fallback to filename, only when the database confirms it is a real
	// serial. A bare filename (or "sr0" for a block device) is not an
	// identifier and would produce junk IDs.
	if serial == "" && sourcePath != "" && database != nil {
		candidate := strings.ReplaceAll(serialFromFilename(sourcePath), "-", "_")
		if _, found := database.LookupByString(console, candidate); found {
			serial = candidate
		}
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

func playStationRootInfo(iso playstationISO, console Console, database Database) (
	rootFiles []string,
	serial string,
	err error,
) {
	if walker, ok := iso.(rootFileWalker); ok {
		return playStationRootInfoWalk(iso, walker, console, database)
	}

	files, iterErr := iso.IterFiles(true)
	if iterErr != nil {
		return nil, "", fmt.Errorf("iterate files: %w", iterErr)
	}
	rootFiles = make([]string, 0, len(files))
	for _, f := range files {
		name := cleanISOFileName(f.Path)
		rootFiles = append(rootFiles, name)
		if serial == "" {
			serial = serialFromRootFile(name)
		}
	}
	if serial == "" {
		serial = findPlayStationSerial(rootFiles, console, database)
	}
	return rootFiles, serial, nil
}

func playStationRootInfoWalk(
	iso playstationISO,
	walker rootFileWalker,
	console Console,
	database Database,
) (rootFiles []string, serial string, err error) {
	rootFiles = make([]string, 0)
	err = walker.WalkFiles(true, func(file iso9660.FileInfo) bool {
		name := cleanISOFileName(file.Path)
		rootFiles = append(rootFiles, name)
		if serial == "" && strings.EqualFold(name, "SYSTEM.CNF") {
			serial = serialFromSystemCNFFile(iso, file)
		}
		if serial == "" {
			serial = serialFromRootFile(name)
		}
		return serial == ""
	})
	if err != nil {
		return nil, "", fmt.Errorf("iterate files: %w", err)
	}
	if serial == "" {
		serial = findPlayStationSerial(rootFiles, console, database)
	}
	return rootFiles, serial, nil
}

func cleanISOFileName(path string) string {
	name := strings.TrimPrefix(path, "/")
	if idx := strings.Index(name, ";"); idx != -1 {
		name = name[:idx]
	}
	return name
}

func serialFromSystemCNFFile(iso playstationISO, file iso9660.FileInfo) string {
	reader, ok := iso.(isoFileReader)
	if !ok {
		return ""
	}
	data, err := reader.ReadFile(file)
	if err != nil {
		return ""
	}
	return serialFromSystemCNF(string(data))
}

func serialFromSystemCNF(content string) string {
	fields := strings.FieldsFunc(strings.ToUpper(content), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' && r != '.'
	})
	for _, field := range fields {
		if serial := serialFromRootFile(field); serial != "" {
			return serial
		}
	}
	return ""
}

// findPlayStationSerial searches for serial in root files.
func findPlayStationSerial(rootFiles []string, console Console, database Database) string {
	for _, fileName := range rootFiles {
		if serial := serialFromRootFile(fileName); serial != "" {
			return serial
		}
	}

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

// serialFromRootFile extracts PlayStation serials from executable names in the disc root.
func serialFromRootFile(fileName string) string {
	name := strings.ToUpper(filepath.Base(fileName))
	if idx := strings.Index(name, ";"); idx != -1 {
		name = name[:idx]
	}
	name = strings.TrimSpace(name)
	if len(name) < 9 {
		return ""
	}

	prefix := name[:4]
	if _, ok := playStationSerialPrefixes[prefix]; !ok {
		return ""
	}

	pos := 4
	if pos < len(name) && isSerialSeparator(rune(name[pos])) {
		pos++
	}

	firstDigits, firstOK := consumeDigits(name, pos, 3)
	if !firstOK {
		return ""
	}
	pos += len(firstDigits)

	if pos < len(name) && isSerialSeparator(rune(name[pos])) {
		pos++
	}

	secondDigits, secondOK := consumeDigits(name, pos, 2)
	if !secondOK {
		return ""
	}

	return prefix + "_" + firstDigits + secondDigits
}

func consumeDigits(value string, start, count int) (string, bool) {
	if start+count > len(value) {
		return "", false
	}
	for pos := start; pos < start+count; pos++ {
		if !unicode.IsDigit(rune(value[pos])) {
			return "", false
		}
	}
	return value[start : start+count], true
}

func isSerialSeparator(value rune) bool {
	return value == '_' || value == '-' || value == '.' || unicode.IsSpace(value)
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

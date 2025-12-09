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

package iso9660

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CueSheet represents a parsed CUE sheet file.
type CueSheet struct {
	Path     string   // Path to the CUE file
	BinFiles []string // Paths to BIN files (absolute)
}

// ParseCue parses a CUE sheet file and returns the BIN file paths.
func ParseCue(cuePath string) (*CueSheet, error) {
	cueFile, err := os.Open(cuePath) //nolint:gosec // Path from user input is expected
	if err != nil {
		return nil, fmt.Errorf("open CUE file: %w", err)
	}
	defer func() { _ = cueFile.Close() }()

	cueDir := filepath.Dir(cuePath)
	cue := &CueSheet{
		Path: cuePath,
	}

	scanner := bufio.NewScanner(cueFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineLower := strings.ToLower(line)

		// Look for FILE "filename" BINARY lines
		if !strings.HasPrefix(lineLower, "file") {
			continue
		}
		// Extract filename between quotes
		parts := strings.Split(line, "\"")
		if len(parts) < 2 {
			continue
		}
		binFile := strings.TrimSpace(parts[1])
		// Make absolute path
		if !filepath.IsAbs(binFile) {
			binFile = filepath.Join(cueDir, binFile)
		}
		cue.BinFiles = append(cue.BinFiles, binFile)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return cue, nil
}

// OpenCue opens an ISO9660 disc image from a CUE sheet.
// It uses the first BIN file referenced in the CUE sheet.
func OpenCue(cuePath string) (*ISO9660, error) {
	cue, err := ParseCue(cuePath)
	if err != nil {
		return nil, err
	}

	if len(cue.BinFiles) == 0 {
		return nil, ErrInvalidISO
	}

	// Open the first BIN file
	return Open(cue.BinFiles[0])
}

// IsCueFile checks if the given path is a CUE file.
func IsCueFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".cue"
}

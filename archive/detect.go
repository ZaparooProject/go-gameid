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

package archive

import (
	"fmt"
	"path/filepath"
	"strings"
)

// gameExtensions are file extensions that indicate cartridge-based game files.
// This only includes unambiguous extensions that can be identified without header analysis.
var gameExtensions = map[string]bool{
	// Game Boy / Game Boy Color
	".gb":  true,
	".gbc": true,

	// Game Boy Advance
	".gba": true,
	".srl": true,

	// Nintendo 64
	".n64": true,
	".z64": true,
	".v64": true,
	".ndd": true,

	// NES
	".nes": true,
	".fds": true,
	".unf": true,
	".nez": true,

	// SNES
	".sfc": true,
	".smc": true,
	".swc": true,

	// Genesis / Mega Drive
	".gen": true,
	".md":  true,
	".smd": true,
}

// IsGameFile checks if a filename has a recognized game file extension.
// This only returns true for cartridge-based game extensions.
func IsGameFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return gameExtensions[ext]
}

// DetectGameFile finds the first game file in an archive.
// It scans the archive's file list and returns the path to the first file
// that has a recognized game extension.
func DetectGameFile(arc Archive) (string, error) {
	files, err := arc.List()
	if err != nil {
		return "", fmt.Errorf("list archive files: %w", err)
	}

	for _, file := range files {
		if IsGameFile(file.Name) {
			return file.Name, nil
		}
	}

	return "", NoGameFilesError{Archive: "archive"}
}

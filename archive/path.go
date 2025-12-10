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
	"os"
	"path/filepath"
	"strings"
)

// Path represents a parsed archive path with optional internal path.
type Path struct {
	ArchivePath  string // Path to the archive file
	InternalPath string // Path inside the archive (empty means auto-detect)
}

// archiveExtensions are the supported archive extensions.
var archiveExtensions = []string{".zip", ".7z", ".rar"}

// ParsePath parses a path that may reference a file inside an archive.
// It supports MiSTer-style paths like "/path/to/archive.zip/folder/game.gba".
//
// Returns:
//   - (*Path, nil) if the path contains an archive reference
//   - (nil, nil) if the path is not an archive reference
//   - (nil, error) if there was an error checking the path
//
//nolint:gocognit,nilnil,revive // Complex path parsing logic requires branching; nil,nil is documented API behavior
func ParsePath(path string) (*Path, error) {
	// Normalize path separators
	normalizedPath := filepath.ToSlash(path)

	// Search for archive extensions followed by a path separator
	for _, ext := range archiveExtensions {
		// Look for pattern like ".zip/" in the path
		pattern := ext + "/"
		idx := strings.Index(strings.ToLower(normalizedPath), pattern)

		if idx != -1 {
			archivePath := path[:idx+len(ext)]
			internalPath := path[idx+len(ext)+1:]

			// Verify the archive file exists
			if _, err := os.Stat(archivePath); err != nil {
				if os.IsNotExist(err) {
					// Archive doesn't exist, this might not be an archive path
					continue
				}
				return nil, fmt.Errorf("stat archive %s: %w", archivePath, err)
			}

			return &Path{
				ArchivePath:  archivePath,
				InternalPath: internalPath,
			}, nil
		}
	}

	// Check if the path itself is an archive (for auto-detection)
	ext := strings.ToLower(filepath.Ext(path))
	if IsArchiveExtension(ext) {
		// Verify the archive file exists
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return nil, nil // Not an archive path
			}
			return nil, fmt.Errorf("stat archive %s: %w", path, err)
		}

		return &Path{
			ArchivePath:  path,
			InternalPath: "", // Auto-detect
		}, nil
	}

	return nil, nil // Not an archive path
}

// IsArchivePath checks if a path references an archive.
// This is a quick check that doesn't verify file existence.
func IsArchivePath(path string) bool {
	normalizedPath := filepath.ToSlash(path)

	// Check for archive extension followed by separator
	for _, ext := range archiveExtensions {
		if strings.Contains(strings.ToLower(normalizedPath), ext+"/") {
			return true
		}
	}

	// Check if path ends with archive extension
	ext := strings.ToLower(filepath.Ext(path))
	return IsArchiveExtension(ext)
}

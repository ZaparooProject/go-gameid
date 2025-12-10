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

//nolint:dupl // Archive implementations are intentionally similar but use different types
package archive

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/bodgit/sevenzip"
)

// SevenZipArchive provides access to files in a 7z archive.
type SevenZipArchive struct {
	reader *sevenzip.ReadCloser
	path   string
}

// OpenSevenZip opens a 7z archive for reading.
func OpenSevenZip(path string) (*SevenZipArchive, error) {
	reader, err := sevenzip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open 7z archive: %w", err)
	}

	return &SevenZipArchive{
		reader: reader,
		path:   path,
	}, nil
}

// List returns all files in the 7z archive.
func (sza *SevenZipArchive) List() ([]FileInfo, error) {
	files := make([]FileInfo, 0, len(sza.reader.File))

	for _, file := range sza.reader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		files = append(files, FileInfo{
			Name: file.Name,
			Size: int64(file.UncompressedSize), //nolint:gosec // Safe: file sizes don't exceed int64
		})
	}

	return files, nil
}

// Open opens a file within the 7z archive.
func (sza *SevenZipArchive) Open(internalPath string) (io.ReadCloser, int64, error) {
	// Normalize path separators
	internalPath = filepath.ToSlash(internalPath)

	for _, file := range sza.reader.File {
		if strings.EqualFold(file.Name, internalPath) {
			reader, err := file.Open()
			if err != nil {
				return nil, 0, fmt.Errorf("open file in 7z: %w", err)
			}
			//nolint:gosec // Safe: file sizes don't exceed int64
			return reader, int64(file.UncompressedSize), nil
		}
	}

	return nil, 0, FileNotFoundError{
		Archive:      sza.path,
		InternalPath: internalPath,
	}
}

// OpenReaderAt opens a file and returns an io.ReaderAt interface.
// The file contents are buffered in memory.
//
//nolint:revive // 4 return values is necessary for this interface pattern
func (sza *SevenZipArchive) OpenReaderAt(internalPath string) (io.ReaderAt, int64, io.Closer, error) {
	return bufferFile(sza, internalPath)
}

// Close closes the 7z archive.
func (sza *SevenZipArchive) Close() error {
	return sza.reader.Close() //nolint:wrapcheck // Close error passthrough is intentional
}

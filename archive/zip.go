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
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// ZIPArchive provides access to files in a ZIP archive.
type ZIPArchive struct {
	reader *zip.ReadCloser
	path   string
}

// OpenZIP opens a ZIP archive for reading.
func OpenZIP(path string) (*ZIPArchive, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open ZIP archive: %w", err)
	}

	return &ZIPArchive{
		reader: reader,
		path:   path,
	}, nil
}

// List returns all files in the ZIP archive.
func (za *ZIPArchive) List() ([]FileInfo, error) {
	files := make([]FileInfo, 0, len(za.reader.File))

	for _, file := range za.reader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		files = append(files, FileInfo{
			Name: file.Name,
			Size: int64(file.UncompressedSize64), //nolint:gosec // Safe: file sizes don't exceed int64
		})
	}

	return files, nil
}

// Open opens a file within the ZIP archive.
func (za *ZIPArchive) Open(internalPath string) (io.ReadCloser, int64, error) {
	// Normalize path separators
	internalPath = filepath.ToSlash(internalPath)

	for _, file := range za.reader.File {
		if strings.EqualFold(file.Name, internalPath) {
			reader, err := file.Open()
			if err != nil {
				return nil, 0, fmt.Errorf("open file in ZIP: %w", err)
			}
			//nolint:gosec // Safe: file sizes don't exceed int64
			return reader, int64(file.UncompressedSize64), nil
		}
	}

	return nil, 0, FileNotFoundError{
		Archive:      za.path,
		InternalPath: internalPath,
	}
}

// OpenReaderAt opens a file and returns an io.ReaderAt interface.
// The file contents are buffered in memory.
//
//nolint:revive // 4 return values is necessary for this interface pattern
func (za *ZIPArchive) OpenReaderAt(internalPath string) (io.ReaderAt, int64, io.Closer, error) {
	return bufferFile(za, internalPath)
}

// Close closes the ZIP archive.
func (za *ZIPArchive) Close() error {
	return za.reader.Close() //nolint:wrapcheck // Close error passthrough is intentional
}

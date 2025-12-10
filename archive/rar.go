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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nwaples/rardecode/v2"
)

// RARArchive provides access to files in a RAR archive.
type RARArchive struct {
	file *os.File
	path string
}

// OpenRAR opens a RAR archive for reading.
func OpenRAR(path string) (*RARArchive, error) {
	file, err := os.Open(path) //nolint:gosec // User-provided path is expected
	if err != nil {
		return nil, fmt.Errorf("open RAR archive: %w", err)
	}

	return &RARArchive{
		file: file,
		path: path,
	}, nil
}

// List returns all files in the RAR archive.
func (ra *RARArchive) List() ([]FileInfo, error) {
	// Seek to beginning
	if _, err := ra.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek RAR archive: %w", err)
	}

	reader, err := rardecode.NewReader(ra.file)
	if err != nil {
		return nil, fmt.Errorf("create RAR reader: %w", err)
	}

	var files []FileInfo //nolint:prealloc // RAR file count unknown until full scan
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read RAR header: %w", err)
		}

		// Skip directories
		if header.IsDir {
			continue
		}

		files = append(files, FileInfo{
			Name: header.Name,
			Size: header.UnPackedSize,
		})
	}

	return files, nil
}

// Open opens a file within the RAR archive.
// Note: RAR archives require sequential reading, so this seeks through the archive.
func (ra *RARArchive) Open(internalPath string) (io.ReadCloser, int64, error) {
	// Normalize path separators
	internalPath = filepath.ToSlash(internalPath)

	// Seek to beginning
	if _, err := ra.file.Seek(0, io.SeekStart); err != nil {
		return nil, 0, fmt.Errorf("seek RAR archive: %w", err)
	}

	reader, err := rardecode.NewReader(ra.file)
	if err != nil {
		return nil, 0, fmt.Errorf("create RAR reader: %w", err)
	}

	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("read RAR header: %w", err)
		}

		if strings.EqualFold(header.Name, internalPath) {
			// Wrap the reader since rardecode doesn't provide a closer
			return &rarFileReader{reader: reader}, header.UnPackedSize, nil
		}
	}

	return nil, 0, FileNotFoundError{
		Archive:      ra.path,
		InternalPath: internalPath,
	}
}

// OpenReaderAt opens a file and returns an io.ReaderAt interface.
// The file contents are buffered in memory.
//
//nolint:revive // 4 return values is necessary for this interface pattern
func (ra *RARArchive) OpenReaderAt(internalPath string) (io.ReaderAt, int64, io.Closer, error) {
	return bufferFile(ra, internalPath)
}

// Close closes the RAR archive.
func (ra *RARArchive) Close() error {
	return ra.file.Close() //nolint:wrapcheck // Close error passthrough is intentional
}

// rarFileReader wraps a rardecode reader to provide io.ReadCloser.
type rarFileReader struct {
	reader *rardecode.Reader
}

func (rfr *rarFileReader) Read(p []byte) (int, error) {
	return rfr.reader.Read(p) //nolint:wrapcheck // Read error passthrough is intentional
}

func (*rarFileReader) Close() error {
	// rardecode doesn't have a close method, nothing to do
	return nil
}

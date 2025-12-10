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

// Package archive provides support for reading game files from archives.
// It supports ZIP, 7z, and RAR formats.
package archive

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// FileInfo contains information about a file in an archive.
type FileInfo struct {
	Name string // Full path within archive
	Size int64  // Uncompressed size
}

// Archive provides read access to files within an archive.
type Archive interface {
	// List returns all files in the archive.
	List() ([]FileInfo, error)

	// Open opens a file within the archive for reading.
	// Returns the reader, uncompressed size, and any error.
	Open(internalPath string) (io.ReadCloser, int64, error)

	// OpenReaderAt opens a file and returns an io.ReaderAt interface.
	// The file contents are buffered in memory to support random access.
	// The returned Closer must be called to release resources.
	OpenReaderAt(internalPath string) (io.ReaderAt, int64, io.Closer, error)

	// Close closes the archive.
	Close() error
}

// Open opens an archive file based on its extension.
// Supported formats: .zip, .7z, .rar
func Open(path string) (Archive, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".zip":
		return OpenZIP(path)
	case ".7z":
		return OpenSevenZip(path)
	case ".rar":
		return OpenRAR(path)
	default:
		return nil, FormatError{Format: ext}
	}
}

// IsArchiveExtension checks if an extension is a supported archive format.
func IsArchiveExtension(ext string) bool {
	ext = strings.ToLower(ext)
	switch ext {
	case ".zip", ".7z", ".rar":
		return true
	default:
		return false
	}
}

// nopCloser wraps a value that doesn't need closing.
type nopCloser struct{}

func (nopCloser) Close() error { return nil }

// bufferFile reads the entire file into memory and returns a ReaderAt.
//
//nolint:revive // 4 return values is necessary for this interface pattern
func bufferFile(arc Archive, internalPath string) (io.ReaderAt, int64, io.Closer, error) {
	reader, size, err := arc.Open(internalPath)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("open file in archive: %w", err)
	}
	defer func() { _ = reader.Close() }()

	data := make([]byte, size)
	bytesRead, err := io.ReadFull(reader, data)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("read file from archive: %w", err)
	}

	return &byteReaderAt{data: data}, int64(bytesRead), nopCloser{}, nil
}

// byteReaderAt implements io.ReaderAt for a byte slice.
type byteReaderAt struct {
	data []byte
}

func (br *byteReaderAt) ReadAt(buf []byte, off int64) (int, error) {
	if off < 0 {
		return 0, fmt.Errorf("negative offset: %d", off)
	}
	if off >= int64(len(br.data)) {
		return 0, io.EOF
	}

	bytesRead := copy(buf, br.data[off:])
	if bytesRead < len(buf) {
		return bytesRead, io.EOF
	}
	return bytesRead, nil
}

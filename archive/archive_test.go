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

package archive_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ZaparooProject/go-gameid/archive"
)

// createTestZIP creates a ZIP archive in tmpDir with the given files.
//
//nolint:gosec // Test helper creates files in test temp directory
func createTestZIP(t *testing.T, tmpDir, name string, files map[string][]byte) string {
	t.Helper()

	zipPath := filepath.Join(tmpDir, name)
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip file: %v", err)
	}
	defer func() { _ = file.Close() }()

	writer := zip.NewWriter(file)

	for filename, content := range files {
		fileWriter, err := writer.Create(filename)
		if err != nil {
			t.Fatalf("create file in zip: %v", err)
		}
		if _, err := fileWriter.Write(content); err != nil {
			t.Fatalf("write file content: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	return zipPath
}

func TestOpen(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a test ZIP
	testContent := []byte("test content")
	zipPath := createTestZIP(t, tmpDir, "test.zip", map[string][]byte{
		"test.txt": testContent,
	})

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "ZIP archive",
			path:    zipPath,
			wantErr: false,
		},
		{
			name:    "non-existent file",
			path:    filepath.Join(tmpDir, "nonexistent.zip"),
			wantErr: true,
		},
		{
			name:    "unsupported format",
			path:    filepath.Join(tmpDir, "test.tar"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			arc, err := archive.Open(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			_ = arc.Close()
		})
	}
}

func TestIsArchiveExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ext  string
		want bool
	}{
		{".zip", true},
		{".ZIP", true},
		{".7z", true},
		{".rar", true},
		{".tar", false},
		{".gz", false},
		{".txt", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			t.Parallel()

			got := archive.IsArchiveExtension(tt.ext)
			if got != tt.want {
				t.Errorf("IsArchiveExtension(%q) = %v, want %v", tt.ext, got, tt.want)
			}
		})
	}
}

func TestZIPArchive_List(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	files := map[string][]byte{
		"game.gba":      make([]byte, 100),
		"readme.txt":    []byte("readme"),
		"folder/file.x": []byte("nested"),
	}
	zipPath := createTestZIP(t, tmpDir, "list.zip", files)

	arc, err := archive.Open(zipPath)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer func() { _ = arc.Close() }()

	fileList, err := arc.List()
	if err != nil {
		t.Fatalf("list files: %v", err)
	}

	if len(fileList) != len(files) {
		t.Errorf("got %d files, want %d", len(fileList), len(files))
	}

	fileMap := make(map[string]int64)
	for _, file := range fileList {
		fileMap[file.Name] = file.Size
	}

	for name, content := range files {
		size, ok := fileMap[name]
		if !ok {
			t.Errorf("missing file: %s", name)
			continue
		}
		if size != int64(len(content)) {
			t.Errorf("file %s: got size %d, want %d", name, size, len(content))
		}
	}
}

func TestZIPArchive_Open_ExistingFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testContent := []byte("test game content")
	files := map[string][]byte{"game.gba": testContent}
	zipPath := createTestZIP(t, tmpDir, "open.zip", files)

	arc, err := archive.Open(zipPath)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer func() { _ = arc.Close() }()

	reader, size, err := arc.Open("game.gba")
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer func() { _ = reader.Close() }()

	if size != int64(len(testContent)) {
		t.Errorf("got size %d, want %d", size, len(testContent))
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	if !bytes.Equal(data, testContent) {
		t.Error("content mismatch")
	}
}

func TestZIPArchive_Open_NonExistent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testContent := []byte("test game content")
	files := map[string][]byte{"game.gba": testContent}
	zipPath := createTestZIP(t, tmpDir, "open2.zip", files)

	arc, err := archive.Open(zipPath)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer func() { _ = arc.Close() }()

	_, _, err = arc.Open("nonexistent.gba")
	if err == nil {
		t.Error("expected error for non-existent file")
	}

	var notFoundErr archive.FileNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("expected FileNotFoundError, got %T", err)
	}
}

func TestZIPArchive_Open_CaseInsensitive(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testContent := []byte("test game content")
	files := map[string][]byte{"game.gba": testContent}
	zipPath := createTestZIP(t, tmpDir, "open3.zip", files)

	arc, err := archive.Open(zipPath)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer func() { _ = arc.Close() }()

	reader, _, err := arc.Open("GAME.GBA")
	if err != nil {
		t.Fatalf("open file case-insensitive: %v", err)
	}
	_ = reader.Close()
}

func TestZIPArchive_OpenReaderAt(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	testContent := []byte("test game content for random access")
	files := map[string][]byte{
		"game.gba": testContent,
	}
	zipPath := createTestZIP(t, tmpDir, "readerAt.zip", files)

	arc, err := archive.Open(zipPath)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer func() { _ = arc.Close() }()

	readerAt, size, closer, err := arc.OpenReaderAt("game.gba")
	if err != nil {
		t.Fatalf("open reader at: %v", err)
	}
	defer func() { _ = closer.Close() }()

	if size != int64(len(testContent)) {
		t.Errorf("got size %d, want %d", size, len(testContent))
	}

	// Test random access
	buf := make([]byte, 4)
	bytesRead, err := readerAt.ReadAt(buf, 5)
	if err != nil {
		t.Fatalf("read at: %v", err)
	}
	if bytesRead != 4 {
		t.Errorf("got %d bytes, want 4", bytesRead)
	}
	if !bytes.Equal(buf, testContent[5:9]) {
		t.Error("content mismatch at offset 5")
	}
}

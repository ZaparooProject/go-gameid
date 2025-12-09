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
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenMounted_ValidDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	disc, err := OpenMounted(tmpDir, "test-uuid", "TEST_VOLUME")
	if err != nil {
		t.Fatalf("OpenMounted() error = %v", err)
	}
	defer func() { _ = disc.Close() }()

	if disc.GetVolumeID() != "TEST_VOLUME" {
		t.Errorf("GetVolumeID() = %q, want %q", disc.GetVolumeID(), "TEST_VOLUME")
	}

	if disc.GetUUID() != "test-uuid" {
		t.Errorf("GetUUID() = %q, want %q", disc.GetUUID(), "test-uuid")
	}
}

func TestOpenMounted_DefaultVolumeID(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dirName := filepath.Base(tmpDir)

	disc, err := OpenMounted(tmpDir, "", "")
	if err != nil {
		t.Fatalf("OpenMounted() error = %v", err)
	}
	defer func() { _ = disc.Close() }()

	// When volumeID is empty, it should use the directory name
	if disc.GetVolumeID() != dirName {
		t.Errorf("GetVolumeID() = %q, want %q (directory name)", disc.GetVolumeID(), dirName)
	}
}

func TestOpenMounted_NonExistentPath(t *testing.T) {
	t.Parallel()

	_, err := OpenMounted("/nonexistent/path/that/does/not/exist", "", "")
	if err == nil {
		t.Error("OpenMounted() should error for non-existent path")
	}
}

func TestOpenMounted_FileNotDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")

	if err := os.WriteFile(filePath, []byte("test"), 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := OpenMounted(filePath, "", "")
	if !errors.Is(err, ErrInvalidISO) {
		t.Errorf("OpenMounted() error = %v, want %v", err, ErrInvalidISO)
	}
}

func TestMountedDisc_Close(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	disc, err := OpenMounted(tmpDir, "", "")
	if err != nil {
		t.Fatalf("OpenMounted() error = %v", err)
	}

	// Close should return nil (no-op)
	if err := disc.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestMountedDisc_GetSystemID(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	disc, err := OpenMounted(tmpDir, "", "")
	if err != nil {
		t.Fatalf("OpenMounted() error = %v", err)
	}
	defer func() { _ = disc.Close() }()

	// GetSystemID should return empty string for mounted discs
	if disc.GetSystemID() != "" {
		t.Errorf("GetSystemID() = %q, want empty string", disc.GetSystemID())
	}
}

func TestMountedDisc_GetPublisherID(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	disc, err := OpenMounted(tmpDir, "", "")
	if err != nil {
		t.Fatalf("OpenMounted() error = %v", err)
	}
	defer func() { _ = disc.Close() }()

	// GetPublisherID should return empty string for mounted discs
	if disc.GetPublisherID() != "" {
		t.Errorf("GetPublisherID() = %q, want empty string", disc.GetPublisherID())
	}
}

func TestMountedDisc_GetDataPreparerID(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	disc, err := OpenMounted(tmpDir, "", "")
	if err != nil {
		t.Fatalf("OpenMounted() error = %v", err)
	}
	defer func() { _ = disc.Close() }()

	// GetDataPreparerID should return empty string for mounted discs
	if disc.GetDataPreparerID() != "" {
		t.Errorf("GetDataPreparerID() = %q, want empty string", disc.GetDataPreparerID())
	}
}

func TestMountedDisc_IterFiles_RootOnly(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create test files
	if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0o600); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file2.bin"), []byte("content2"), 0o600); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Create subdirectory with file
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0o750); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested"), 0o600); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	disc, err := OpenMounted(tmpDir, "", "")
	if err != nil {
		t.Fatalf("OpenMounted() error = %v", err)
	}
	defer func() { _ = disc.Close() }()

	files, err := disc.IterFiles(true)
	if err != nil {
		t.Fatalf("IterFiles(true) error = %v", err)
	}

	// Should only have root files, not nested
	if len(files) != 2 {
		t.Errorf("IterFiles(true) returned %d files, want 2", len(files))
	}

	// Check paths have leading slash
	for _, file := range files {
		if file.Path[0] != '/' {
			t.Errorf("File path %q should start with /", file.Path)
		}
	}
}

//nolint:gocognit,revive // Test verifies multiple file conditions
func TestMountedDisc_IterFiles_AllFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create test files
	if err := os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte("root"), 0o600); err != nil {
		t.Fatalf("Failed to create root file: %v", err)
	}

	// Create subdirectory with file
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0o750); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested content"), 0o600); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	disc, err := OpenMounted(tmpDir, "", "")
	if err != nil {
		t.Fatalf("OpenMounted() error = %v", err)
	}
	defer func() { _ = disc.Close() }()

	files, err := disc.IterFiles(false)
	if err != nil {
		t.Fatalf("IterFiles(false) error = %v", err)
	}

	// Should have both root and nested files
	if len(files) != 2 {
		t.Errorf("IterFiles(false) returned %d files, want 2", len(files))
	}

	// Verify we have both files
	foundRoot, foundNested := false, false
	for _, file := range files {
		if file.Path == "/root.txt" {
			foundRoot = true
		}
		if file.Path == "/subdir/nested.txt" {
			foundNested = true
			// Check size
			const nestedContentLen = 14
			if file.Size != nestedContentLen {
				t.Errorf("Nested file size = %d, want 14", file.Size)
			}
		}
	}

	if !foundRoot {
		t.Error("IterFiles(false) did not find /root.txt")
	}
	if !foundNested {
		t.Error("IterFiles(false) did not find /subdir/nested.txt")
	}
}

func TestMountedDisc_IterFiles_EmptyDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	disc, err := OpenMounted(tmpDir, "", "")
	if err != nil {
		t.Fatalf("OpenMounted() error = %v", err)
	}
	defer func() { _ = disc.Close() }()

	files, err := disc.IterFiles(true)
	if err != nil {
		t.Fatalf("IterFiles() error = %v", err)
	}

	if len(files) != 0 {
		t.Errorf("IterFiles() on empty dir returned %d files, want 0", len(files))
	}
}

func TestMountedDisc_ReadFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	content := []byte("test file content")

	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), content, 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	disc, err := OpenMounted(tmpDir, "", "")
	if err != nil {
		t.Fatalf("OpenMounted() error = %v", err)
	}
	defer func() { _ = disc.Close() }()

	files, err := disc.IterFiles(true)
	if err != nil {
		t.Fatalf("IterFiles() error = %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(files))
	}

	data, err := disc.ReadFile(files[0])
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !bytes.Equal(data, content) {
		t.Errorf("ReadFile() = %q, want %q", data, content)
	}
}

func TestMountedDisc_ReadFile_Nested(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	content := []byte("nested file content")

	// Create nested structure
	subDir := filepath.Join(tmpDir, "dir1", "dir2")
	if err := os.MkdirAll(subDir, 0o750); err != nil {
		t.Fatalf("Failed to create nested dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "deep.txt"), content, 0o600); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	disc, err := OpenMounted(tmpDir, "", "")
	if err != nil {
		t.Fatalf("OpenMounted() error = %v", err)
	}
	defer func() { _ = disc.Close() }()

	files, err := disc.IterFiles(false)
	if err != nil {
		t.Fatalf("IterFiles() error = %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(files))
	}

	data, err := disc.ReadFile(files[0])
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !bytes.Equal(data, content) {
		t.Errorf("ReadFile() = %q, want %q", data, content)
	}
}

func TestMountedDisc_ReadFileByPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	content := []byte("bypath content")

	if err := os.WriteFile(filepath.Join(tmpDir, "bypath.txt"), content, 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	disc, err := OpenMounted(tmpDir, "", "")
	if err != nil {
		t.Fatalf("OpenMounted() error = %v", err)
	}
	defer func() { _ = disc.Close() }()

	// Test with leading slash
	data, err := disc.ReadFileByPath("/bypath.txt")
	if err != nil {
		t.Fatalf("ReadFileByPath() error = %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("ReadFileByPath(/bypath.txt) = %q, want %q", data, content)
	}

	// Test without leading slash
	data, err = disc.ReadFileByPath("bypath.txt")
	if err != nil {
		t.Fatalf("ReadFileByPath() error = %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("ReadFileByPath(bypath.txt) = %q, want %q", data, content)
	}
}

func TestMountedDisc_ReadFileByPath_NotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	disc, err := OpenMounted(tmpDir, "", "")
	if err != nil {
		t.Fatalf("OpenMounted() error = %v", err)
	}
	defer func() { _ = disc.Close() }()

	_, err = disc.ReadFileByPath("/nonexistent.txt")
	if err == nil {
		t.Error("ReadFileByPath() should error for non-existent file")
	}
}

func TestMountedDisc_FileExists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "exists.txt"), []byte("test"), 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a subdirectory
	if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0o750); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	disc, err := OpenMounted(tmpDir, "", "")
	if err != nil {
		t.Fatalf("OpenMounted() error = %v", err)
	}
	defer func() { _ = disc.Close() }()

	// Test existing file
	if !disc.FileExists("/exists.txt") {
		t.Error("FileExists(/exists.txt) = false, want true")
	}

	// Test non-existent file
	if disc.FileExists("/nonexistent.txt") {
		t.Error("FileExists(/nonexistent.txt) = true, want false")
	}

	// Test directory path (should return false - not a file)
	if disc.FileExists("/subdir") {
		t.Error("FileExists(/subdir) = true, want false (directories are not files)")
	}
}

func TestMountedDisc_FileExists_WithoutSlash(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	disc, err := OpenMounted(tmpDir, "", "")
	if err != nil {
		t.Fatalf("OpenMounted() error = %v", err)
	}
	defer func() { _ = disc.Close() }()

	// Test without leading slash
	if !disc.FileExists("test.txt") {
		t.Error("FileExists(test.txt) = false, want true")
	}
}

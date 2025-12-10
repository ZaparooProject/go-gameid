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
	"os"
	"path/filepath"
	"testing"

	"github.com/ZaparooProject/go-gameid/archive"
)

//nolint:gosec // Test helper creates files in test temp directory
func createSimpleTestZIP(t *testing.T, zipPath string) {
	t.Helper()

	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}

	writer := zip.NewWriter(zipFile)
	fileWriter, err := writer.Create("game.gba")
	if err != nil {
		t.Fatalf("create file in zip: %v", err)
	}
	if _, err := fileWriter.Write([]byte("test")); err != nil {
		t.Fatalf("write to zip: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := zipFile.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}
}

func TestParsePath_ArchiveWithInternalPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "games.zip")
	createSimpleTestZIP(t, zipPath)

	result, err := archive.ParsePath(zipPath + "/folder/game.gba")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ArchivePath != zipPath {
		t.Errorf("ArchivePath = %q, want %q", result.ArchivePath, zipPath)
	}
	if result.InternalPath != "folder/game.gba" {
		t.Errorf("InternalPath = %q, want %q", result.InternalPath, "folder/game.gba")
	}
}

func TestParsePath_ArchiveOnly(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "games.zip")
	createSimpleTestZIP(t, zipPath)

	result, err := archive.ParsePath(zipPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ArchivePath != zipPath {
		t.Errorf("ArchivePath = %q, want %q", result.ArchivePath, zipPath)
	}
	if result.InternalPath != "" {
		t.Errorf("InternalPath = %q, want empty", result.InternalPath)
	}
}

func TestParsePath_NonArchive(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	result, err := archive.ParsePath(filepath.Join(tmpDir, "regular.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

func TestParsePath_NonExistentArchive(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	// Use string concatenation instead of filepath.Join to include path separator
	fakePath := tmpDir + "/nonexistent.zip/game.gba"

	result, err := archive.ParsePath(fakePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

func TestIsArchivePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{"/games/roms.zip/game.gba", true},
		{"/games/roms.7z/folder/game.nes", true},
		{"/games/roms.rar/game.sfc", true},
		{"/games/roms.zip", true},
		{"/games/roms.7z", true},
		{"/games/roms.rar", true},
		{"/games/game.gba", false},
		{"/games/roms.tar/game.gba", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			got := archive.IsArchivePath(tt.path)
			if got != tt.want {
				t.Errorf("IsArchivePath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

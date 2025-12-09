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
	"os"
	"path/filepath"
	"testing"
)

type cueTestCase struct {
	name       string
	cueContent string
	wantFiles  []string
}

func runCueTest(t *testing.T, tmpDir string, tc cueTestCase) {
	t.Helper()

	// Write CUE file
	cuePath := filepath.Join(tmpDir, tc.name+".cue")
	if err := os.WriteFile(cuePath, []byte(tc.cueContent), 0o600); err != nil {
		t.Fatalf("Failed to write CUE file: %v", err)
	}

	cue, err := ParseCue(cuePath)
	if err != nil {
		t.Fatalf("ParseCue() error = %v", err)
	}

	if len(cue.BinFiles) != len(tc.wantFiles) {
		t.Errorf("Got %d BIN files, want %d", len(cue.BinFiles), len(tc.wantFiles))
		return
	}

	for i, want := range tc.wantFiles {
		gotBase := filepath.Base(cue.BinFiles[i])
		if gotBase != want {
			t.Errorf("BinFiles[%d] = %q, want %q", i, gotBase, want)
		}

		// Verify path is absolute
		if !filepath.IsAbs(cue.BinFiles[i]) {
			t.Errorf("BinFiles[%d] = %q is not absolute", i, cue.BinFiles[i])
		}
	}
}

func TestParseCue(t *testing.T) {
	t.Parallel()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "iso9660-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	tests := []cueTestCase{
		{
			name: "Single file",
			cueContent: `FILE "game.bin" BINARY
TRACK 01 MODE1/2352
  INDEX 01 00:00:00`,
			wantFiles: []string{"game.bin"},
		},
		{
			name: "Multiple files",
			cueContent: `FILE "track01.bin" BINARY
TRACK 01 MODE1/2352
  INDEX 01 00:00:00
FILE "track02.bin" BINARY
TRACK 02 AUDIO
  INDEX 00 00:00:00
  INDEX 01 00:02:00`,
			wantFiles: []string{"track01.bin", "track02.bin"},
		},
		{
			name: "Mixed case",
			cueContent: `File "Game.BIN" Binary
Track 01 Mode1/2352
  Index 01 00:00:00`,
			wantFiles: []string{"Game.BIN"},
		},
		{
			name:       "No files",
			cueContent: `REM This is a comment`,
			wantFiles:  nil,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			runCueTest(t, tmpDir, testCase)
		})
	}
}

func TestParseCue_NonExistent(t *testing.T) {
	t.Parallel()

	_, err := ParseCue("/nonexistent/path/game.cue")
	if err == nil {
		t.Error("ParseCue() should error for non-existent file")
	}
}

func TestIsCueFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{"game.cue", true},
		{"game.CUE", true},
		{"game.Cue", true},
		{"game.bin", false},
		{"game.iso", false},
		{"game", false},
		{"/path/to/game.cue", true},
		{"/path/to/game.bin", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			got := IsCueFile(tt.path)
			if got != tt.want {
				t.Errorf("IsCueFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

//nolint:gosec // G306 permissions ok for tests
func TestParseCue_AbsolutePaths(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "iso9660-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a platform-appropriate absolute path for testing
	// On Windows, use the temp directory as the base for an absolute path
	// On Unix, use a typical Unix absolute path
	var absPath string
	if filepath.Separator == '\\' {
		// Windows: use tmpDir as an absolute path base
		absPath = filepath.Join(tmpDir, "absolute", "path", "game.bin")
	} else {
		// Unix: use a Unix-style absolute path
		absPath = "/absolute/path/game.bin"
	}

	// CUE with absolute path
	cueContent := `FILE "` + absPath + `" BINARY
TRACK 01 MODE1/2352
  INDEX 01 00:00:00`

	cuePath := filepath.Join(tmpDir, "game.cue")
	if writeErr := os.WriteFile(cuePath, []byte(cueContent), 0o644); writeErr != nil {
		t.Fatalf("Failed to write CUE file: %v", writeErr)
	}

	cue, err := ParseCue(cuePath)
	if err != nil {
		t.Fatalf("ParseCue() error = %v", err)
	}

	if len(cue.BinFiles) != 1 {
		t.Fatalf("Expected 1 BIN file, got %d", len(cue.BinFiles))
	}

	// Absolute paths should be preserved
	if cue.BinFiles[0] != absPath {
		t.Errorf("BinFiles[0] = %q, want %q", cue.BinFiles[0], absPath)
	}
}

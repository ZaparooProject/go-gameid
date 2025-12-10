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
	"errors"
	"testing"

	"github.com/ZaparooProject/go-gameid/archive"
)

func TestIsGameFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		filename string
		want     bool
	}{
		// Game Boy / Game Boy Color
		{"game.gb", true},
		{"GAME.GB", true},
		{"game.gbc", true},

		// Game Boy Advance
		{"game.gba", true},
		{"game.srl", true},

		// Nintendo 64
		{"game.n64", true},
		{"game.z64", true},
		{"game.v64", true},
		{"game.ndd", true},

		// NES
		{"game.nes", true},
		{"game.fds", true},
		{"game.unf", true},
		{"game.nez", true},

		// SNES
		{"game.sfc", true},
		{"game.smc", true},
		{"game.swc", true},

		// Genesis
		{"game.gen", true},
		{"game.md", true},
		{"game.smd", true},

		// Non-game files
		{"game.iso", false},
		{"game.bin", false},
		{"game.cue", false},
		{"readme.txt", false},
		{"game.zip", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			t.Parallel()

			got := archive.IsGameFile(tt.filename)
			if got != tt.want {
				t.Errorf("IsGameFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestDetectGameFile_FindsGame(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	files := map[string][]byte{
		"readme.txt": []byte("readme"),
		"game.gba":   make([]byte, 100),
		"notes.doc":  []byte("notes"),
	}
	zipPath := createTestZIP(t, tmpDir, "games.zip", files)

	arc, err := archive.Open(zipPath)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer func() { _ = arc.Close() }()

	gamePath, err := archive.DetectGameFile(arc)
	if err != nil {
		t.Fatalf("detect game file: %v", err)
	}

	if gamePath != "game.gba" {
		t.Errorf("got %q, want %q", gamePath, "game.gba")
	}
}

func TestDetectGameFile_NoGames(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	files := map[string][]byte{
		"readme.txt": []byte("readme"),
		"notes.doc":  []byte("notes"),
	}
	zipPath := createTestZIP(t, tmpDir, "nogames.zip", files)

	arc, err := archive.Open(zipPath)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer func() { _ = arc.Close() }()

	_, err = archive.DetectGameFile(arc)
	if err == nil {
		t.Error("expected error for archive with no games")
	}

	var noGamesErr archive.NoGameFilesError
	if !errors.As(err, &noGamesErr) {
		t.Errorf("expected NoGameFilesError, got %T", err)
	}
}

func TestDetectGameFile_MultipleGames(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// ZIP iteration order may vary, but we want to ensure at least one is returned
	files := map[string][]byte{
		"game1.gba": make([]byte, 100),
		"game2.sfc": make([]byte, 200),
	}
	zipPath := createTestZIP(t, tmpDir, "multigames.zip", files)

	arc, err := archive.Open(zipPath)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer func() { _ = arc.Close() }()

	gamePath, err := archive.DetectGameFile(arc)
	if err != nil {
		t.Fatalf("detect game file: %v", err)
	}

	if !archive.IsGameFile(gamePath) {
		t.Errorf("returned path %q is not a game file", gamePath)
	}
}

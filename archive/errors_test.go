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
	"strings"
	"testing"

	"github.com/ZaparooProject/go-gameid/archive"
)

func TestFormatError(t *testing.T) {
	t.Parallel()

	err := archive.FormatError{Format: ".tar", Reason: "not supported"}

	msg := err.Error()
	if !strings.Contains(msg, ".tar") {
		t.Errorf("error message should contain format: %s", msg)
	}
	if !strings.Contains(msg, "not supported") {
		t.Errorf("error message should contain reason: %s", msg)
	}
}

func TestFormatError_NoReason(t *testing.T) {
	t.Parallel()

	err := archive.FormatError{Format: ".tar"}

	msg := err.Error()
	if !strings.Contains(msg, ".tar") {
		t.Errorf("error message should contain format: %s", msg)
	}
}

func TestFileNotFoundError(t *testing.T) {
	t.Parallel()

	err := archive.FileNotFoundError{
		Archive:      "/path/to/archive.zip",
		InternalPath: "folder/game.gba",
	}

	msg := err.Error()
	if !strings.Contains(msg, "archive.zip") {
		t.Errorf("error message should contain archive: %s", msg)
	}
	if !strings.Contains(msg, "folder/game.gba") {
		t.Errorf("error message should contain internal path: %s", msg)
	}
}

func TestNoGameFilesError(t *testing.T) {
	t.Parallel()

	err := archive.NoGameFilesError{Archive: "/path/to/archive.zip"}

	msg := err.Error()
	if !strings.Contains(msg, "archive.zip") {
		t.Errorf("error message should contain archive: %s", msg)
	}
	if !strings.Contains(msg, "game") {
		t.Errorf("error message should mention games: %s", msg)
	}
}

func TestDiscNotSupportedError(t *testing.T) {
	t.Parallel()

	err := archive.DiscNotSupportedError{Console: "PSX"}

	msg := err.Error()
	if !strings.Contains(msg, "PSX") {
		t.Errorf("error message should contain console: %s", msg)
	}
	if !strings.Contains(msg, "disc") {
		t.Errorf("error message should mention disc: %s", msg)
	}
}

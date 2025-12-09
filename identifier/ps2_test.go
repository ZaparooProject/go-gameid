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

//nolint:dupl // PS2/PSP tests have similar structure but test different identifiers
package identifier

import (
	"errors"
	"testing"
)

func TestPS2Identifier_Console(t *testing.T) {
	t.Parallel()

	id := NewPS2Identifier()
	if id.Console() != ConsolePS2 {
		t.Errorf("Console() = %v, want %v", id.Console(), ConsolePS2)
	}
}

func TestPS2Identifier_Identify_ReturnsNotSupported(t *testing.T) {
	t.Parallel()

	id := NewPS2Identifier()
	_, err := id.Identify(nil, 0, nil)

	var notSupported ErrNotSupported
	if !errors.As(err, &notSupported) {
		t.Errorf("Identify() error = %v, want ErrNotSupported", err)
	}
}

func TestPS2Identifier_IdentifyFromPath_NonExistent(t *testing.T) {
	t.Parallel()

	id := NewPS2Identifier()
	_, err := id.IdentifyFromPath("/nonexistent/path/game.iso", nil)
	if err == nil {
		t.Error("IdentifyFromPath() should error for non-existent file")
	}
}

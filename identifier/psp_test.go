// Copyright (c) 2026 Niema Moshiri and The Zaparoo Project.
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

package identifier

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ZaparooProject/go-gameid/internal/testiso"
)

func TestPSPIdentifier_Console(t *testing.T) {
	t.Parallel()

	id := NewPSPIdentifier()
	if id.Console() != ConsolePSP {
		t.Errorf("Console() = %v, want %v", id.Console(), ConsolePSP)
	}
}

func TestPSPIdentifier_Identify_ReturnsNotSupported(t *testing.T) {
	t.Parallel()

	id := NewPSPIdentifier()
	_, err := id.Identify(nil, 0, nil)

	var notSupported ErrNotSupported
	if !errors.As(err, &notSupported) {
		t.Errorf("Identify() error = %v, want ErrNotSupported", err)
	}
}

func TestPSPIdentifier_IdentifyFromPath_NonExistent(t *testing.T) {
	t.Parallel()

	id := NewPSPIdentifier()
	_, err := id.IdentifyFromPath("/nonexistent/path/game.iso", nil)
	if err == nil {
		t.Error("IdentifyFromPath() should error for non-existent file")
	}
}

func TestPSPIdentifier_IdentifyFromPath_UMDData(t *testing.T) {
	t.Parallel()

	isoData := testiso.CreateMinimal(t, "PSPTEST", "PLAYSTATION", "", []testiso.File{
		{Name: "UMD_DATA.BIN;1", Data: []byte("UCUS-98765|Example Game")},
	})
	isoPath := filepath.Join(t.TempDir(), "game.iso")
	if err := os.WriteFile(isoPath, isoData, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	id := NewPSPIdentifier()
	result, err := id.IdentifyFromPath(isoPath, nil)
	if err != nil {
		t.Fatalf("IdentifyFromPath() error = %v", err)
	}

	if result.ID != "UCUS-98765" {
		t.Errorf("result.ID = %q, want %q", result.ID, "UCUS-98765")
	}
	if !strings.HasPrefix(result.Metadata["volume_ID"], "PSPTEST") {
		t.Errorf("volume_ID metadata = %q, want prefix %q", result.Metadata["volume_ID"], "PSPTEST")
	}
}

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

package identifier

import (
	"io"
)

// PS2Identifier identifies PlayStation 2 games.
type PS2Identifier struct{}

// NewPS2Identifier creates a new PS2 identifier.
func NewPS2Identifier() *PS2Identifier {
	return &PS2Identifier{}
}

// Console returns the console type.
func (*PS2Identifier) Console() Console {
	return ConsolePS2
}

// Identify extracts PS2 game information from the given reader.
// For disc-based games, use IdentifyFromPath instead.
func (*PS2Identifier) Identify(_ io.ReaderAt, _ int64, _ Database) (*Result, error) {
	return nil, ErrNotSupported{Format: "raw reader for PS2"}
}

// IdentifyFromPath identifies a PS2 game from a file path.
func (*PS2Identifier) IdentifyFromPath(path string, database Database) (*Result, error) {
	iso, err := openPlayStationISO(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = iso.Close() }()

	return identifyPlayStation(iso, ConsolePS2, database, path)
}

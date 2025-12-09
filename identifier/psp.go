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
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/ZaparooProject/go-gameid/iso9660"
)

// PSPIdentifier identifies PlayStation Portable games.
type PSPIdentifier struct{}

// NewPSPIdentifier creates a new PSP identifier.
func NewPSPIdentifier() *PSPIdentifier {
	return &PSPIdentifier{}
}

// Console returns the console type.
func (*PSPIdentifier) Console() Console {
	return ConsolePSP
}

// Identify extracts PSP game information from the given reader.
// For disc-based games, use IdentifyFromPath instead.
func (*PSPIdentifier) Identify(_ io.ReaderAt, _ int64, _ Database) (*Result, error) {
	return nil, ErrNotSupported{Format: "raw reader for PSP"}
}

// IdentifyFromPath identifies a PSP game from a file path.
func (*PSPIdentifier) IdentifyFromPath(path string, database Database) (*Result, error) {
	iso, err := iso9660.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open ISO: %w", err)
	}
	defer func() { _ = iso.Close() }()

	return identifyPSPFromISO(iso, database)
}

func identifyPSPFromISO(iso *iso9660.ISO9660, database Database) (*Result, error) {
	result := NewResult(ConsolePSP)

	// Look for UMD_DATA.BIN in root
	files, err := iso.IterFiles(true)
	if err != nil {
		return nil, fmt.Errorf("iterate files: %w", err)
	}

	var umdDataInfo *iso9660.FileInfo
	for _, f := range files {
		if strings.ToUpper(filepath.Base(f.Path)) == "UMD_DATA.BIN" {
			umdDataInfo = &f
			break
		}
	}

	if umdDataInfo == nil {
		return nil, ErrInvalidFormat{Console: ConsolePSP, Reason: "UMD_DATA.BIN not found"}
	}

	// Read UMD_DATA.BIN
	data, err := iso.ReadFile(*umdDataInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to read UMD_DATA.BIN: %w", err)
	}

	// Extract serial (until first '|' character)
	var serial string
	for _, b := range data {
		if b == '|' {
			break
		}
		serial += string(b)
	}
	serial = strings.TrimSpace(serial)

	result.ID = serial
	result.SetMetadata("ID", serial)
	result.SetMetadata("uuid", iso.GetUUID())
	result.SetMetadata("volume_ID", iso.GetVolumeID())

	// Database lookup
	if database != nil && serial != "" {
		if entry, found := database.LookupByString(ConsolePSP, serial); found {
			result.MergeMetadata(entry)
		}
	}

	return result, nil
}

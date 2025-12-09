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

// NeoGeoCDIdentifier identifies Neo Geo CD games.
type NeoGeoCDIdentifier struct{}

// NewNeoGeoCDIdentifier creates a new Neo Geo CD identifier.
func NewNeoGeoCDIdentifier() *NeoGeoCDIdentifier {
	return &NeoGeoCDIdentifier{}
}

// Console returns the console type.
func (*NeoGeoCDIdentifier) Console() Console {
	return ConsoleNeoGeoCD
}

// Identify extracts Neo Geo CD game information from the given reader.
// For disc-based games, use IdentifyFromPath instead.
func (*NeoGeoCDIdentifier) Identify(_ io.ReaderAt, _ int64, _ Database) (*Result, error) {
	return nil, ErrNotSupported{Format: "raw reader for NeoGeoCD"}
}

// IdentifyFromPath identifies a Neo Geo CD game from a file path.
func (n *NeoGeoCDIdentifier) IdentifyFromPath(path string, database Database) (*Result, error) {
	var iso interface {
		GetUUID() string
		GetVolumeID() string
		Close() error
	}

	ext := strings.ToLower(filepath.Ext(path))

	if ext == ".cue" {
		isoFile, err := iso9660.OpenCue(path)
		if err != nil {
			return nil, fmt.Errorf("open CUE: %w", err)
		}
		iso = isoFile
	} else {
		isoFile, err := iso9660.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open ISO: %w", err)
		}
		iso = isoFile
	}
	defer func() { _ = iso.Close() }()

	return n.identifyFromISO(iso, database)
}

func (*NeoGeoCDIdentifier) identifyFromISO(iso interface {
	GetUUID() string
	GetVolumeID() string
}, db Database,
) (*Result, error) {
	result := NewResult(ConsoleNeoGeoCD)

	uuid := iso.GetUUID()
	volumeID := iso.GetVolumeID()

	result.SetMetadata("uuid", uuid)
	result.SetMetadata("volume_ID", volumeID)

	// NeoGeoCD uses (uuid, volume_ID) tuple as primary key, with volume_ID as fallback
	if db != nil {
		// Try (uuid, volume_ID) tuple first
		type neogeoCDKey struct {
			uuid     string
			volumeID string
		}
		key := neogeoCDKey{uuid: uuid, volumeID: volumeID}
		entry, found := db.Lookup(ConsoleNeoGeoCD, key)
		if !found && volumeID != "" {
			// Fallback to just volume_ID
			entry, found = db.LookupByString(ConsoleNeoGeoCD, volumeID)
		}
		if found {
			result.MergeMetadata(entry)
		}
	}

	// Set ID from volume_ID if not set by database
	if result.ID == "" {
		result.ID = volumeID
	}

	return result, nil
}

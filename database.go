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

package gameid

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io"
	"os"

	"github.com/ZaparooProject/go-gameid/identifier"
)

// GameDatabase holds the game metadata database.
type GameDatabase struct {
	// Console-specific databases
	// Key format varies by console (see identifier package)
	GB       map[gbKey]map[string]string
	GBA      map[string]map[string]string
	GC       map[string]map[string]string
	Genesis  map[string]map[string]string
	N64      map[string]map[string]string
	NES      map[int]map[string]string
	PSP      map[string]map[string]string
	PSX      map[string]map[string]string
	PS2      map[string]map[string]string
	Saturn   map[string]map[string]string
	SegaCD   map[string]map[string]string
	SNES     map[snesKey]map[string]string
	NeoGeoCD map[neogeoCDKey]map[string]string

	// ID prefixes for disc-based consoles
	IDPrefixes map[identifier.Console][]string
}

// gbKey is the lookup key for GB/GBC games: (internal_title, global_checksum)
type gbKey struct {
	Title    string
	Checksum uint16
}

// snesKey is the lookup key for SNES games: (developer_id, internal_name_hex, rom_version, checksum)
type snesKey struct {
	InternalName string
	DeveloperID  int
	ROMVersion   int
	Checksum     int
}

// neogeoCDKey is the lookup key for NeoGeoCD games: (uuid, volume_id)
type neogeoCDKey struct {
	UUID     string
	VolumeID string
}

// NewDatabase creates an empty database.
func NewDatabase() *GameDatabase {
	return &GameDatabase{
		GB:         make(map[gbKey]map[string]string),
		GBA:        make(map[string]map[string]string),
		GC:         make(map[string]map[string]string),
		Genesis:    make(map[string]map[string]string),
		N64:        make(map[string]map[string]string),
		NES:        make(map[int]map[string]string),
		PSP:        make(map[string]map[string]string),
		PSX:        make(map[string]map[string]string),
		PS2:        make(map[string]map[string]string),
		Saturn:     make(map[string]map[string]string),
		SegaCD:     make(map[string]map[string]string),
		SNES:       make(map[snesKey]map[string]string),
		NeoGeoCD:   make(map[neogeoCDKey]map[string]string),
		IDPrefixes: make(map[identifier.Console][]string),
	}
}

// LoadDatabase loads a database from a gob.gz file.
func LoadDatabase(path string) (*GameDatabase, error) {
	dbFile, err := os.Open(path) //nolint:gosec // Path from user input is expected
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = dbFile.Close() }()

	return LoadDatabaseFromReader(dbFile)
}

// LoadDatabaseFromReader loads a database from a gzip-compressed gob reader.
func LoadDatabaseFromReader(r io.Reader) (*GameDatabase, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gz.Close() }()

	db := NewDatabase()
	dec := gob.NewDecoder(gz)
	if err := dec.Decode(db); err != nil {
		return nil, fmt.Errorf("failed to decode database: %w", err)
	}

	return db, nil
}

// SaveDatabase saves the database to a gob.gz file.
func (db *GameDatabase) SaveDatabase(path string) error {
	file, err := os.Create(path) //nolint:gosec // Path from user input is expected
	if err != nil {
		return fmt.Errorf("failed to create database file: %w", err)
	}
	defer func() { _ = file.Close() }()

	gz := gzip.NewWriter(file)
	defer func() { _ = gz.Close() }()

	enc := gob.NewEncoder(gz)
	if err := enc.Encode(db); err != nil {
		return fmt.Errorf("failed to encode database: %w", err)
	}

	return nil
}

// Lookup retrieves metadata for a game by console and key.
//
//nolint:exhaustive // Only some consoles use complex keys; others use LookupByString
func (db *GameDatabase) Lookup(console identifier.Console, key any) (map[string]string, bool) {
	switch console {
	case identifier.ConsoleGB, identifier.ConsoleGBC:
		if k, ok := key.(struct {
			title    string
			checksum uint16
		}); ok {
			entry, found := db.GB[gbKey{Title: k.title, Checksum: k.checksum}]
			return entry, found
		}
		// Also try direct gbKey type
		if k, ok := key.(gbKey); ok {
			entry, found := db.GB[k]
			return entry, found
		}
	case identifier.ConsoleSNES:
		if k, ok := key.(struct {
			internalName string
			developerID  int
			romVersion   int
			checksum     int
		}); ok {
			entry, found := db.SNES[snesKey{
				DeveloperID:  k.developerID,
				InternalName: k.internalName,
				ROMVersion:   k.romVersion,
				Checksum:     k.checksum,
			}]
			return entry, found
		}
		if k, ok := key.(snesKey); ok {
			entry, found := db.SNES[k]
			return entry, found
		}
	case identifier.ConsoleNES:
		if k, ok := key.(int); ok {
			entry, found := db.NES[k]
			return entry, found
		}
	case identifier.ConsoleNeoGeoCD:
		if k, ok := key.(struct {
			uuid     string
			volumeID string
		}); ok {
			entry, found := db.NeoGeoCD[neogeoCDKey{UUID: k.uuid, VolumeID: k.volumeID}]
			return entry, found
		}
		if k, ok := key.(neogeoCDKey); ok {
			entry, found := db.NeoGeoCD[k]
			return entry, found
		}
	}

	return nil, false
}

// LookupByString retrieves metadata using a string key.
//
//nolint:exhaustive // GB, GBC, NES, SNES use Lookup with complex keys, not LookupByString
func (db *GameDatabase) LookupByString(console identifier.Console, key string) (map[string]string, bool) {
	switch console {
	case identifier.ConsoleGBA:
		entry, found := db.GBA[key]
		return entry, found
	case identifier.ConsoleGC:
		entry, found := db.GC[key]
		return entry, found
	case identifier.ConsoleGenesis:
		entry, found := db.Genesis[key]
		return entry, found
	case identifier.ConsoleN64:
		entry, found := db.N64[key]
		return entry, found
	case identifier.ConsolePSP:
		entry, found := db.PSP[key]
		return entry, found
	case identifier.ConsolePSX:
		entry, found := db.PSX[key]
		return entry, found
	case identifier.ConsolePS2:
		entry, found := db.PS2[key]
		return entry, found
	case identifier.ConsoleSaturn:
		entry, found := db.Saturn[key]
		return entry, found
	case identifier.ConsoleSegaCD:
		entry, found := db.SegaCD[key]
		return entry, found
	case identifier.ConsoleNeoGeoCD:
		// Try volume_ID as fallback for NeoGeoCD
		for k, v := range db.NeoGeoCD {
			if k.VolumeID == key {
				return v, true
			}
		}
	}

	return nil, false
}

// GetIDPrefixes returns the ID prefixes for disc-based consoles.
func (db *GameDatabase) GetIDPrefixes(console identifier.Console) []string {
	return db.IDPrefixes[console]
}

// Ensure GameDatabase implements identifier.Database
var _ identifier.Database = (*GameDatabase)(nil)

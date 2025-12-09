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
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"os"
	"path/filepath"
	"testing"

	"github.com/ZaparooProject/go-gameid/identifier"
)

//nolint:gocyclo,revive,cyclop // Verifying all database fields are initialized
func TestNewDatabase(t *testing.T) {
	t.Parallel()

	db := NewDatabase()

	if db == nil {
		t.Fatal("NewDatabase() returned nil")
	}

	// Check all maps are initialized
	if db.GB == nil {
		t.Error("GB map is nil")
	}
	if db.GBA == nil {
		t.Error("GBA map is nil")
	}
	if db.GC == nil {
		t.Error("GC map is nil")
	}
	if db.Genesis == nil {
		t.Error("Genesis map is nil")
	}
	if db.N64 == nil {
		t.Error("N64 map is nil")
	}
	if db.NES == nil {
		t.Error("NES map is nil")
	}
	if db.PSP == nil {
		t.Error("PSP map is nil")
	}
	if db.PSX == nil {
		t.Error("PSX map is nil")
	}
	if db.PS2 == nil {
		t.Error("PS2 map is nil")
	}
	if db.Saturn == nil {
		t.Error("Saturn map is nil")
	}
	if db.SegaCD == nil {
		t.Error("SegaCD map is nil")
	}
	if db.SNES == nil {
		t.Error("SNES map is nil")
	}
	if db.NeoGeoCD == nil {
		t.Error("NeoGeoCD map is nil")
	}
	if db.IDPrefixes == nil {
		t.Error("IDPrefixes map is nil")
	}
}

func TestDatabase_SaveAndLoad(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "gameid-db-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a database with test data
	db := NewDatabase()
	db.GBA["BPEE"] = map[string]string{"title": "Pokemon Emerald", "region": "USA"}
	db.N64["SM64"] = map[string]string{"title": "Super Mario 64"}
	db.PSX["SLUS_00123"] = map[string]string{"title": "Test Game"}
	db.NES[0x12345678] = map[string]string{"title": "NES Game"}
	db.IDPrefixes[identifier.ConsolePSX] = []string{"SLUS", "SCUS", "SLPM"}

	// Save database
	dbPath := filepath.Join(tmpDir, "test.gob.gz")
	if saveErr := db.SaveDatabase(dbPath); saveErr != nil {
		t.Fatalf("SaveDatabase() error = %v", saveErr)
	}

	// Verify file exists
	if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
		t.Fatal("Database file was not created")
	}

	// Load database
	loadedDB, loadErr := LoadDatabase(dbPath)
	if loadErr != nil {
		t.Fatalf("LoadDatabase() error = %v", loadErr)
	}

	// Verify data
	if entry, found := loadedDB.GBA["BPEE"]; !found {
		t.Error("GBA entry not found")
	} else if entry["title"] != "Pokemon Emerald" {
		t.Errorf("GBA title = %q, want %q", entry["title"], "Pokemon Emerald")
	}

	if entry, found := loadedDB.N64["SM64"]; !found {
		t.Error("N64 entry not found")
	} else if entry["title"] != "Super Mario 64" {
		t.Errorf("N64 title = %q, want %q", entry["title"], "Super Mario 64")
	}

	if entry, found := loadedDB.NES[0x12345678]; !found {
		t.Error("NES entry not found")
	} else if entry["title"] != "NES Game" {
		t.Errorf("NES title = %q, want %q", entry["title"], "NES Game")
	}

	prefixes := loadedDB.GetIDPrefixes(identifier.ConsolePSX)
	if len(prefixes) != 3 {
		t.Errorf("IDPrefixes count = %d, want 3", len(prefixes))
	}
}

func TestDatabase_LookupByString(t *testing.T) {
	t.Parallel()

	db := NewDatabase()
	db.GBA["BPEE"] = map[string]string{"title": "Pokemon Emerald"}
	db.GC["GALE"] = map[string]string{"title": "Super Smash Bros Melee"}
	db.Genesis["G-1234"] = map[string]string{"title": "Sonic"}
	db.N64["SM64"] = map[string]string{"title": "Super Mario 64"}
	db.PSP["ULUS12345"] = map[string]string{"title": "PSP Game"}
	db.PSX["SLUS_00123"] = map[string]string{"title": "PS1 Game"}
	db.PS2["SLUS_20123"] = map[string]string{"title": "PS2 Game"}
	db.Saturn["GS9046"] = map[string]string{"title": "Nights"}
	db.SegaCD["G6014"] = map[string]string{"title": "Sonic CD"}

	tests := []struct {
		console  identifier.Console
		key      string
		wantFind bool
	}{
		{identifier.ConsoleGBA, "BPEE", true},
		{identifier.ConsoleGBA, "XXXX", false},
		{identifier.ConsoleGC, "GALE", true},
		{identifier.ConsoleGenesis, "G-1234", true},
		{identifier.ConsoleN64, "SM64", true},
		{identifier.ConsolePSP, "ULUS12345", true},
		{identifier.ConsolePSX, "SLUS_00123", true},
		{identifier.ConsolePS2, "SLUS_20123", true},
		{identifier.ConsoleSaturn, "GS9046", true},
		{identifier.ConsoleSegaCD, "G6014", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.console)+"/"+tt.key, func(t *testing.T) {
			t.Parallel()

			entry, found := db.LookupByString(tt.console, tt.key)
			if found != tt.wantFind {
				t.Errorf("LookupByString(%v, %q) found = %v, want %v", tt.console, tt.key, found, tt.wantFind)
			}
			if found && entry == nil {
				t.Error("Entry is nil when found = true")
			}
		})
	}
}

func TestDatabase_Lookup_NES(t *testing.T) {
	t.Parallel()

	db := NewDatabase()
	db.NES[0xDEADBEEF] = map[string]string{"title": "Test NES Game"}

	entry, found := db.Lookup(identifier.ConsoleNES, 0xDEADBEEF)
	if !found {
		t.Error("Lookup() did not find NES entry")
	}
	if entry["title"] != "Test NES Game" {
		t.Errorf("title = %q, want %q", entry["title"], "Test NES Game")
	}

	// Test not found
	_, found = db.Lookup(identifier.ConsoleNES, 0x00000000)
	if found {
		t.Error("Lookup() should not find non-existent entry")
	}
}

func TestDatabase_GetIDPrefixes(t *testing.T) {
	t.Parallel()

	db := NewDatabase()
	db.IDPrefixes[identifier.ConsolePSX] = []string{"SLUS", "SCUS"}
	db.IDPrefixes[identifier.ConsolePS2] = []string{"SLUS", "SCUS", "SLPM"}

	prefixes := db.GetIDPrefixes(identifier.ConsolePSX)
	if len(prefixes) != 2 {
		t.Errorf("GetIDPrefixes(PSX) = %d prefixes, want 2", len(prefixes))
	}

	prefixes = db.GetIDPrefixes(identifier.ConsolePS2)
	if len(prefixes) != 3 {
		t.Errorf("GetIDPrefixes(PS2) = %d prefixes, want 3", len(prefixes))
	}

	// Console without prefixes
	prefixes = db.GetIDPrefixes(identifier.ConsoleGBA)
	if len(prefixes) != 0 {
		t.Errorf("GetIDPrefixes(GBA) = %d prefixes, want 0", len(prefixes))
	}
}

func TestLoadDatabase_NonExistent(t *testing.T) {
	t.Parallel()

	_, err := LoadDatabase("/nonexistent/path/db.gob.gz")
	if err == nil {
		t.Error("LoadDatabase() should error for non-existent file")
	}
}

func TestLoadDatabaseFromReader(t *testing.T) {
	t.Parallel()

	// Create a database and encode it to a buffer
	db := NewDatabase()
	db.GBA["TEST"] = map[string]string{"title": "Test Game"}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	enc := gob.NewEncoder(gz)
	if err := enc.Encode(db); err != nil {
		t.Fatalf("Failed to encode database: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}

	// Load from buffer
	loadedDB, err := LoadDatabaseFromReader(&buf)
	if err != nil {
		t.Fatalf("LoadDatabaseFromReader() error = %v", err)
	}

	if entry, found := loadedDB.GBA["TEST"]; !found {
		t.Error("GBA entry not found")
	} else if entry["title"] != "Test Game" {
		t.Errorf("title = %q, want %q", entry["title"], "Test Game")
	}
}

func TestLoadDatabaseFromReader_InvalidGzip(t *testing.T) {
	t.Parallel()

	// Not valid gzip data
	buf := bytes.NewReader([]byte("not gzip data"))
	_, err := LoadDatabaseFromReader(buf)
	if err == nil {
		t.Error("LoadDatabaseFromReader() should error for invalid gzip data")
	}
}

//nolint:paralleltest // Interface verification test doesn't need parallel
func TestDatabase_ImplementsInterface(_ *testing.T) {
	// This test just verifies the interface is implemented correctly
	// The actual implementation is tested in other tests
	var _ identifier.Database = (*GameDatabase)(nil)
}

func TestDatabase_Lookup_GB(t *testing.T) {
	t.Parallel()

	db := NewDatabase()
	// Add a GB entry with the gbKey struct
	db.GB[gbKey{Title: "POKEMON RED", Checksum: 0x1234}] = map[string]string{
		"title":  "Pokemon Red",
		"region": "USA",
	}

	// Test lookup with anonymous struct (how identifiers call it)
	key := struct {
		title    string
		checksum uint16
	}{
		title:    "POKEMON RED",
		checksum: 0x1234,
	}

	entry, found := db.Lookup(identifier.ConsoleGB, key)
	if !found {
		t.Error("Lookup() did not find GB entry with struct key")
	}
	if entry["title"] != "Pokemon Red" {
		t.Errorf("title = %q, want %q", entry["title"], "Pokemon Red")
	}

	// Test with GBC (should use same DB)
	_, found = db.Lookup(identifier.ConsoleGBC, key)
	if !found {
		t.Error("Lookup() did not find GBC entry with struct key")
	}

	// Test not found
	notFoundKey := struct {
		title    string
		checksum uint16
	}{
		title:    "UNKNOWN GAME",
		checksum: 0xFFFF,
	}
	_, found = db.Lookup(identifier.ConsoleGB, notFoundKey)
	if found {
		t.Error("Lookup() should not find non-existent GB entry")
	}
}

func TestDatabase_Lookup_SNES(t *testing.T) {
	t.Parallel()

	db := NewDatabase()
	// Add a SNES entry with the snesKey struct
	db.SNES[snesKey{
		InternalName: "SUPER MARIO WORLD",
		DeveloperID:  0x01,
		ROMVersion:   0x00,
		Checksum:     0xA0DA,
	}] = map[string]string{
		"title":  "Super Mario World",
		"region": "USA",
	}

	// Test lookup with anonymous struct (how identifiers call it)
	key := struct {
		internalName string
		developerID  int
		romVersion   int
		checksum     int
	}{
		internalName: "SUPER MARIO WORLD",
		developerID:  0x01,
		romVersion:   0x00,
		checksum:     0xA0DA,
	}

	entry, found := db.Lookup(identifier.ConsoleSNES, key)
	if !found {
		t.Error("Lookup() did not find SNES entry with struct key")
	}
	if entry["title"] != "Super Mario World" {
		t.Errorf("title = %q, want %q", entry["title"], "Super Mario World")
	}

	// Test not found
	notFoundKey := struct {
		internalName string
		developerID  int
		romVersion   int
		checksum     int
	}{
		internalName: "UNKNOWN",
		developerID:  0xFF,
		romVersion:   0x00,
		checksum:     0x0000,
	}
	_, found = db.Lookup(identifier.ConsoleSNES, notFoundKey)
	if found {
		t.Error("Lookup() should not find non-existent SNES entry")
	}
}

func TestDatabase_Lookup_NeoGeoCD(t *testing.T) {
	t.Parallel()

	db := NewDatabase()
	// Add a NeoGeoCD entry with the neogeoCDKey struct
	db.NeoGeoCD[neogeoCDKey{
		UUID:     "2024-01-01-00-00-00-00",
		VolumeID: "BLAZING_STAR",
	}] = map[string]string{
		"title": "Blazing Star",
	}

	// Test lookup with anonymous struct (how identifiers call it)
	key := struct {
		uuid     string
		volumeID string
	}{
		uuid:     "2024-01-01-00-00-00-00",
		volumeID: "BLAZING_STAR",
	}

	entry, found := db.Lookup(identifier.ConsoleNeoGeoCD, key)
	if !found {
		t.Error("Lookup() did not find NeoGeoCD entry with struct key")
	}
	if entry["title"] != "Blazing Star" {
		t.Errorf("title = %q, want %q", entry["title"], "Blazing Star")
	}

	// Test not found
	notFoundKey := struct {
		uuid     string
		volumeID string
	}{
		uuid:     "unknown",
		volumeID: "unknown",
	}
	_, found = db.Lookup(identifier.ConsoleNeoGeoCD, notFoundKey)
	if found {
		t.Error("Lookup() should not find non-existent NeoGeoCD entry")
	}
}

func TestDatabase_LookupByString_NeoGeoCD_VolumeIDFallback(t *testing.T) {
	t.Parallel()

	db := NewDatabase()
	// Add a NeoGeoCD entry
	db.NeoGeoCD[neogeoCDKey{
		UUID:     "2024-01-01-00-00-00-00",
		VolumeID: "METAL_SLUG",
	}] = map[string]string{
		"title": "Metal Slug",
	}

	// LookupByString should find by VolumeID alone
	entry, found := db.LookupByString(identifier.ConsoleNeoGeoCD, "METAL_SLUG")
	if !found {
		t.Error("LookupByString() did not find NeoGeoCD entry by volumeID")
	}
	if entry["title"] != "Metal Slug" {
		t.Errorf("title = %q, want %q", entry["title"], "Metal Slug")
	}

	// Test not found
	_, found = db.LookupByString(identifier.ConsoleNeoGeoCD, "UNKNOWN_VOL")
	if found {
		t.Error("LookupByString() should not find non-existent NeoGeoCD entry")
	}
}

func TestDatabase_Lookup_InvalidKeyTypes(t *testing.T) {
	t.Parallel()

	db := NewDatabase()

	// Test with wrong key type for each console
	tests := []struct {
		key     any
		name    string
		console identifier.Console
	}{
		{name: "GB with string key", console: identifier.ConsoleGB, key: "invalid"},
		{name: "GBC with int key", console: identifier.ConsoleGBC, key: 12345},
		{name: "SNES with string key", console: identifier.ConsoleSNES, key: "invalid"},
		{name: "NES with string key", console: identifier.ConsoleNES, key: "invalid"},
		{name: "NeoGeoCD with int key", console: identifier.ConsoleNeoGeoCD, key: 12345},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, found := db.Lookup(tt.console, tt.key)
			if found {
				t.Error("Lookup() should return false for invalid key type")
			}
		})
	}
}

func TestDatabase_Lookup_UnsupportedConsole(t *testing.T) {
	t.Parallel()

	db := NewDatabase()

	// Consoles that use LookupByString, not Lookup
	unsupportedConsoles := []identifier.Console{
		identifier.ConsoleGBA,
		identifier.ConsoleGC,
		identifier.ConsoleGenesis,
		identifier.ConsoleN64,
		identifier.ConsolePSP,
		identifier.ConsolePSX,
		identifier.ConsolePS2,
		identifier.ConsoleSaturn,
		identifier.ConsoleSegaCD,
	}

	for _, console := range unsupportedConsoles {
		t.Run(string(console), func(t *testing.T) {
			t.Parallel()

			_, found := db.Lookup(console, "any_key")
			if found {
				t.Errorf("Lookup() should return false for %s (uses LookupByString)", console)
			}
		})
	}
}

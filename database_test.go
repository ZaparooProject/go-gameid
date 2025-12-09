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

func TestNewDatabase(t *testing.T) {
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
	tmpDir, err := os.MkdirTemp("", "gameid-db-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a database with test data
	db := NewDatabase()
	db.GBA["BPEE"] = map[string]string{"title": "Pokemon Emerald", "region": "USA"}
	db.N64["SM64"] = map[string]string{"title": "Super Mario 64"}
	db.PSX["SLUS_00123"] = map[string]string{"title": "Test Game"}
	db.NES[0x12345678] = map[string]string{"title": "NES Game"}
	db.IDPrefixes[identifier.ConsolePSX] = []string{"SLUS", "SCUS", "SLPM"}

	// Save database
	dbPath := filepath.Join(tmpDir, "test.gob.gz")
	if err := db.SaveDatabase(dbPath); err != nil {
		t.Fatalf("SaveDatabase() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("Database file was not created")
	}

	// Load database
	loadedDB, err := LoadDatabase(dbPath)
	if err != nil {
		t.Fatalf("LoadDatabase() error = %v", err)
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
	_, err := LoadDatabase("/nonexistent/path/db.gob.gz")
	if err == nil {
		t.Error("LoadDatabase() should error for non-existent file")
	}
}

func TestLoadDatabaseFromReader(t *testing.T) {
	// Create a database and encode it to a buffer
	db := NewDatabase()
	db.GBA["TEST"] = map[string]string{"title": "Test Game"}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	enc := gob.NewEncoder(gz)
	if err := enc.Encode(db); err != nil {
		t.Fatalf("Failed to encode database: %v", err)
	}
	gz.Close()

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
	// Not valid gzip data
	buf := bytes.NewReader([]byte("not gzip data"))
	_, err := LoadDatabaseFromReader(buf)
	if err == nil {
		t.Error("LoadDatabaseFromReader() should error for invalid gzip data")
	}
}

func TestDatabase_ImplementsInterface(t *testing.T) {
	// This test just verifies the interface is implemented correctly
	// The actual implementation is tested in other tests
	var _ identifier.Database = (*GameDatabase)(nil)
}

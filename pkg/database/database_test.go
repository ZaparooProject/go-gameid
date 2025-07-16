package database

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Test data structures matching Python GameID database format
func TestLoadDatabase(t *testing.T) {
	// Create a temporary test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_db.json")

	// Expected database structure from Python
	testDB := map[string]map[string]map[string]string{
		"GBA": {
			"AGBE": {
				"title":       "Golden Sun",
				"region":      "USA",
				"developer":   "Camelot",
				"publisher":   "Nintendo",
				"release_date": "2001-11-11",
			},
			"BPRE": {
				"title":       "Pokemon Ruby",
				"region":      "USA",
				"developer":   "Game Freak",
				"publisher":   "Nintendo",
				"release_date": "2003-03-19",
			},
		},
		"GB_GBC": {
			"POKEMON RED,0x91": { // title,checksum format for GB/GBC
				"title":     "Pokemon Red",
				"region":    "USA",
				"developer": "Game Freak",
				"publisher": "Nintendo",
			},
		},
	}

	// Write test database
	data, err := json.MarshalIndent(testDB, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}
	if err := os.WriteFile(dbPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test database: %v", err)
	}

	// Test loading the database
	db, err := LoadDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to load database: %v", err)
	}

	// Verify structure
	if db == nil {
		t.Fatal("Database is nil")
	}

	// Check GBA games
	gbaGames, ok := db.Systems["GBA"]
	if !ok {
		t.Fatal("GBA system not found in database")
	}

	goldenSun, ok := gbaGames["AGBE"]
	if !ok {
		t.Fatal("Golden Sun (AGBE) not found in GBA games")
	}

	if goldenSun["title"] != "Golden Sun" {
		t.Errorf("Expected title 'Golden Sun', got '%s'", goldenSun["title"])
	}

	// Check GB/GBC games with composite key
	gbGames, ok := db.Systems["GB_GBC"]
	if !ok {
		t.Fatal("GB_GBC system not found in database")
	}

	pokemonRed, ok := gbGames["POKEMON RED,0x91"]
	if !ok {
		t.Fatal("Pokemon Red not found in GB_GBC games")
	}

	if pokemonRed["title"] != "Pokemon Red" {
		t.Errorf("Expected title 'Pokemon Red', got '%s'", pokemonRed["title"])
	}
}

func TestLoadDatabase_FileNotFound(t *testing.T) {
	_, err := LoadDatabase("/nonexistent/path/db.json")
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}
}

func TestLoadDatabase_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON
	if err := os.WriteFile(dbPath, []byte("invalid json content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := LoadDatabase(dbPath)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}

func TestLookupGame(t *testing.T) {
	// Create test database in memory
	db := &GameDatabase{
		Systems: map[string]SystemDatabase{
			"GBA": {
				"AGBE": GameMetadata{
					"ID":    "AGBE",
					"title": "Golden Sun",
				},
			},
		},
	}

	tests := []struct {
		name      string
		system    string
		gameID    string
		wantFound bool
		wantTitle string
	}{
		{"Valid GBA game", "GBA", "AGBE", true, "Golden Sun"},
		{"Invalid game ID", "GBA", "XXXX", false, ""},
		{"Invalid system", "XXX", "AGBE", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game, found := db.LookupGame(tt.system, tt.gameID)
			if found != tt.wantFound {
				t.Errorf("LookupGame() found = %v, want %v", found, tt.wantFound)
			}
			if found && game["title"] != tt.wantTitle {
				t.Errorf("LookupGame() title = %v, want %v", game["title"], tt.wantTitle)
			}
		})
	}
}

func TestLoadDatabase_FromURL(t *testing.T) {
	t.Skip("Skipping URL loading test - implement when needed")
	
	// This would test loading from the Python GameID database URL
	// For now, we'll focus on local file loading
}
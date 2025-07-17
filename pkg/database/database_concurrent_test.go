package database

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestConcurrentDatabaseAccess tests concurrent access to database operations
func TestConcurrentDatabaseAccess(t *testing.T) {
	// create test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "concurrent_test.json")

	// create a reasonably sized database
	testDB := make(map[string]map[string]map[string]string)
	systems := []string{"GBA", "GB_GBC", "N64", "SNES", "Genesis"}

	for _, system := range systems {
		testDB[system] = make(map[string]map[string]string)
		for i := 0; i < 100; i++ {
			gameID := fmt.Sprintf("GAME%03d", i)
			testDB[system][gameID] = map[string]string{
				"ID":     gameID,
				"title":  fmt.Sprintf("Test Game %d", i),
				"region": "USA",
			}
		}
	}

	// write database
	data, err := json.MarshalIndent(testDB, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test database: %v", err)
	}
	if err := os.WriteFile(dbPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test database: %v", err)
	}

	// test concurrent database loading
	t.Run("Concurrent Loading", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				db, err := LoadDatabase(dbPath)
				if err != nil {
					errors <- err
					return
				}

				// verify database structure
				if len(db.Systems) != len(systems) {
					errors <- fmt.Errorf("expected %d systems, got %d", len(systems), len(db.Systems))
				}
			}()
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent loading error: %v", err)
		}
	})

	// load database for lookup tests
	db, err := LoadDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to load database: %v", err)
	}

	// test concurrent lookups
	t.Run("Concurrent Lookups", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 100)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// random system and game
				system := systems[rand.Intn(len(systems))]
				gameID := fmt.Sprintf("GAME%03d", rand.Intn(100))

				game, found := db.LookupGame(system, gameID)
				if !found {
					errors <- fmt.Errorf("game not found: %s/%s", system, gameID)
					return
				}

				if game["ID"] != gameID {
					errors <- fmt.Errorf("wrong game ID: got %s, want %s", game["ID"], gameID)
				}
			}()
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent lookup error: %v", err)
		}
	})
}

// TestConcurrentDatabaseModification tests concurrent read/write scenarios
func TestConcurrentDatabaseModification(t *testing.T) {
	// create initial database
	db := &GameDatabase{
		Systems: make(map[string]SystemDatabase),
	}

	// initialize with some systems
	systems := []string{"GBA", "GB_GBC", "N64"}
	for _, system := range systems {
		db.Systems[system] = make(SystemDatabase)
		for i := 0; i < 10; i++ {
			gameID := fmt.Sprintf("INIT%03d", i)
			db.Systems[system][gameID] = GameMetadata{
				"ID":    gameID,
				"title": fmt.Sprintf("Initial Game %d", i),
			}
		}
	}

	var wg sync.WaitGroup
	errors := make(chan error, 50)

	// concurrent readers
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < 10; j++ {
				system := systems[rand.Intn(len(systems))]
				gameID := fmt.Sprintf("INIT%03d", rand.Intn(10))

				game, found := db.LookupGame(system, gameID)
				if !found {
					// it's okay if not found due to race condition
					continue
				}

				if game["ID"] != gameID {
					errors <- fmt.Errorf("reader %d: wrong game ID: got %s, want %s", id, game["ID"], gameID)
				}

				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	// concurrent writers (simulating database updates)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < 5; j++ {
				system := systems[rand.Intn(len(systems))]
				gameID := fmt.Sprintf("NEW%03d_%03d", id, j)

				// simulate adding new game (not thread-safe in current implementation)
				// this will likely cause race conditions
				if db.Systems[system] == nil {
					db.Systems[system] = make(SystemDatabase)
				}

				db.Systems[system][gameID] = GameMetadata{
					"ID":    gameID,
					"title": fmt.Sprintf("New Game %d-%d", id, j),
				}

				time.Sleep(2 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// expect some errors due to race conditions
	errorCount := 0
	for err := range errors {
		errorCount++
		t.Logf("Race condition error (expected): %v", err)
	}

	t.Logf("Total race condition errors: %d", errorCount)

	// verify final state
	totalGames := 0
	for system, games := range db.Systems {
		totalGames += len(games)
		t.Logf("System %s has %d games", system, len(games))
	}
	t.Logf("Total games in database: %d", totalGames)
}

// TestLargeDatabasePerformance tests performance with large databases
func TestLargeDatabasePerformance(t *testing.T) {
	// create large database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "large_test.json")

	testDB := make(map[string]map[string]map[string]string)
	systems := []string{"GBA", "GB_GBC", "N64", "SNES", "Genesis", "PSX", "PS2", "Saturn", "PSP", "GC"}

	for _, system := range systems {
		testDB[system] = make(map[string]map[string]string)
		for i := 0; i < 1000; i++ {
			gameID := fmt.Sprintf("%s_%04d", system, i)
			testDB[system][gameID] = map[string]string{
				"ID":          gameID,
				"title":       fmt.Sprintf("Game %d for %s", i, system),
				"region":      "USA",
				"developer":   "Test Developer",
				"publisher":   "Test Publisher",
				"genre":       "Action",
				"description": "A test game with a longer description to make the database larger",
			}
		}
	}

	// write database
	data, err := json.MarshalIndent(testDB, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal large database: %v", err)
	}
	if err := os.WriteFile(dbPath, data, 0644); err != nil {
		t.Fatalf("Failed to write large database: %v", err)
	}

	// test loading performance
	start := time.Now()
	db, err := LoadDatabase(dbPath)
	loadTime := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to load large database: %v", err)
	}

	t.Logf("Large database loading time: %v", loadTime)

	// test lookup performance
	start = time.Now()
	for i := 0; i < 1000; i++ {
		system := systems[rand.Intn(len(systems))]
		gameID := fmt.Sprintf("%s_%04d", system, rand.Intn(1000))

		game, found := db.LookupGame(system, gameID)
		if !found {
			t.Errorf("Game not found: %s/%s", system, gameID)
		}
		if found && game["ID"] != gameID {
			t.Errorf("Wrong game ID: got %s, want %s", game["ID"], gameID)
		}
	}
	lookupTime := time.Since(start)

	t.Logf("1000 random lookups time: %v", lookupTime)
	t.Logf("Average lookup time: %v", lookupTime/1000)
}

// TestDatabaseMemoryUsage tests memory usage patterns
func TestDatabaseMemoryUsage(t *testing.T) {
	// create database with varying sized entries
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "memory_test.json")

	testDB := make(map[string]map[string]map[string]string)
	system := "TEST"
	testDB[system] = make(map[string]map[string]string)

	// create entries with different sizes
	for i := 0; i < 100; i++ {
		gameID := fmt.Sprintf("GAME%03d", i)

		// create progressively larger descriptions
		descSize := i * 10
		if descSize > 1000 {
			descSize = 1000
		}

		testDB[system][gameID] = map[string]string{
			"ID":          gameID,
			"title":       fmt.Sprintf("Game %d", i),
			"description": fmt.Sprintf("Description %s", string(make([]byte, descSize))),
		}
	}

	// write and load database
	data, err := json.MarshalIndent(testDB, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal database: %v", err)
	}
	if err := os.WriteFile(dbPath, data, 0644); err != nil {
		t.Fatalf("Failed to write database: %v", err)
	}

	db, err := LoadDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to load database: %v", err)
	}

	// verify all entries are accessible
	for i := 0; i < 100; i++ {
		gameID := fmt.Sprintf("GAME%03d", i)
		game, found := db.LookupGame(system, gameID)
		if !found {
			t.Errorf("Game not found: %s", gameID)
		}
		if found && game["ID"] != gameID {
			t.Errorf("Wrong game ID: got %s, want %s", game["ID"], gameID)
		}
	}
}

// TestDatabaseCorruption tests handling of corrupted database files
func TestDatabaseCorruption(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "Empty file",
			content: "",
			wantErr: true,
		},
		{
			name:    "Invalid JSON",
			content: "not json",
			wantErr: true,
		},
		{
			name:    "Partial JSON",
			content: `{"GBA": {"GAME001": {"ID": "GAME001"`,
			wantErr: true,
		},
		{
			name:    "Wrong structure",
			content: `["array", "instead", "of", "object"]`,
			wantErr: true,
		},
		{
			name:    "Mixed types",
			content: `{"GBA": "string_instead_of_object"}`,
			wantErr: true, // will fail to parse
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := filepath.Join(tmpDir, tt.name+".json")
			os.WriteFile(dbPath, []byte(tt.content), 0644)

			_, err := LoadDatabase(dbPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadDatabase() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// BenchmarkDatabaseLookup benchmarks database lookup performance
func BenchmarkDatabaseLookup(b *testing.B) {
	// create test database
	db := &GameDatabase{
		Systems: map[string]SystemDatabase{
			"GBA": make(SystemDatabase),
		},
	}

	// populate with test data
	for i := 0; i < 1000; i++ {
		gameID := fmt.Sprintf("GAME%04d", i)
		db.Systems["GBA"][gameID] = GameMetadata{
			"ID":    gameID,
			"title": fmt.Sprintf("Game %d", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gameID := fmt.Sprintf("GAME%04d", i%1000)
		db.LookupGame("GBA", gameID)
	}
}

// BenchmarkDatabaseLoad benchmarks database loading performance
func BenchmarkDatabaseLoad(b *testing.B) {
	// create test database file
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.json")

	testDB := map[string]map[string]map[string]string{
		"GBA": make(map[string]map[string]string),
	}

	for i := 0; i < 100; i++ {
		gameID := fmt.Sprintf("GAME%03d", i)
		testDB["GBA"][gameID] = map[string]string{
			"ID":    gameID,
			"title": fmt.Sprintf("Game %d", i),
		}
	}

	data, _ := json.MarshalIndent(testDB, "", "  ")
	os.WriteFile(dbPath, data, 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db, err := LoadDatabase(dbPath)
		if err != nil {
			b.Fatal(err)
		}
		_ = db
	}
}

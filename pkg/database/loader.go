package database

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadDatabase loads a game database from a JSON file
func LoadDatabase(path string) (*GameDatabase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read database file: %w", err)
	}

	// Parse JSON into temporary structure
	var rawDB map[string]map[string]map[string]string
	if err := json.Unmarshal(data, &rawDB); err != nil {
		return nil, fmt.Errorf("failed to parse database JSON: %w", err)
	}

	// Convert to GameDatabase structure
	db := &GameDatabase{
		Systems: make(map[string]SystemDatabase),
	}

	for system, games := range rawDB {
		systemDB := make(SystemDatabase)
		for gameID, metadata := range games {
			gameMetadata := make(GameMetadata)
			for key, value := range metadata {
				gameMetadata[key] = value
			}
			// Ensure ID is in metadata
			if _, hasID := gameMetadata["ID"]; !hasID && system != "GB_GBC" {
				gameMetadata["ID"] = gameID
			}
			systemDB[gameID] = gameMetadata
		}
		db.Systems[system] = systemDB
	}

	return db, nil
}

// LoadDatabaseFromURL loads a game database from a URL (placeholder for now)
func LoadDatabaseFromURL(url string, timeout int) (*GameDatabase, error) {
	// TODO: Implement URL loading with gzip decompression
	return nil, fmt.Errorf("URL loading not yet implemented")
}

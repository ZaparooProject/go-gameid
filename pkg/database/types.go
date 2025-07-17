package database

// GameMetadata represents all metadata for a single game
type GameMetadata map[string]string

// SystemDatabase represents all games for a single system
type SystemDatabase map[string]GameMetadata

// GameDatabase represents the entire game database
type GameDatabase struct {
	Systems map[string]SystemDatabase
}

// LookupGame finds a game in the database by system and ID
func (db *GameDatabase) LookupGame(system, gameID string) (GameMetadata, bool) {
	if db == nil || db.Systems == nil {
		return nil, false
	}

	systemDB, ok := db.Systems[system]
	if !ok {
		return nil, false
	}

	game, ok := systemDB[gameID]
	return game, ok
}

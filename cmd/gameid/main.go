package main

import (
	"fmt"
	"log"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
	"github.com/wizzomafizzo/go-gameid/pkg/identifiers"
)

func main() {
	// Load database
	db, err := database.LoadDatabase("dbs/gameid_db.json")
	if err != nil {
		log.Fatalf("Failed to load database: %v", err)
	}

	// Test GBA identifier with a real game
	gbaIdentifier := identifiers.NewGBAIdentifier(db)
	gamePath := "/Volumes/MiSTer/games/GBA/1 USA - D-H/Golden Sun (USA, Europe).gba"
	
	result, err := gbaIdentifier.Identify(gamePath)
	if err != nil {
		log.Fatalf("Failed to identify game: %v", err)
	}

	// Print results
	for key, value := range result {
		fmt.Printf("%s\t%s\n", key, value)
	}
}

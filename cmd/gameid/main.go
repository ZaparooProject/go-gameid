// Command gameid identifies video game files and returns metadata.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ZaparooProject/go-gameid"
)

var (
	inputFile    = flag.String("i", "", "input file path (required)")
	console      = flag.String("c", "", "console type (auto-detect if omitted)")
	dbPath       = flag.String("db", "", "path to game database (gob.gz file)")
	jsonOutput   = flag.Bool("json", false, "output as JSON")
	listConsoles = flag.Bool("list-consoles", false, "list supported consoles and exit")
	version      = flag.Bool("version", false, "print version and exit")
)

const appVersion = "0.1.0"

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -i <file> [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Identifies video game files and returns metadata.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -i game.gba\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i game.iso -c PSX\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i game.n64 -db gamedb.gob.gz -json\n", os.Args[0])
	}
	flag.Parse()

	if *version {
		fmt.Printf("gameid version %s\n", appVersion)
		os.Exit(0)
	}

	if *listConsoles {
		fmt.Println("Supported consoles:")
		for _, c := range gameid.AllConsoles {
			fmt.Printf("  %s\n", c)
		}
		os.Exit(0)
	}

	if *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: input file required (-i)\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load database if specified
	var db *gameid.GameDatabase
	if *dbPath != "" {
		var err error
		db, err = gameid.LoadDatabase(*dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading database: %v\n", err)
			os.Exit(1)
		}
	}

	// Identify the game
	var result *gameid.Result
	var err error

	if *console != "" {
		// Console specified, use it directly
		c, err := gameid.ParseConsole(*console)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: unknown console '%s'\n", *console)
			fmt.Fprintf(os.Stderr, "Use -list-consoles to see supported consoles\n")
			os.Exit(1)
		}
		result, err = gameid.IdentifyWithConsole(*inputFile, c, db)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error identifying game: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Auto-detect console
		result, err = gameid.Identify(*inputFile, db)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error identifying game: %v\n", err)
			os.Exit(1)
		}
	}

	// Output results
	if *jsonOutput {
		outputJSON(result)
	} else {
		outputText(result)
	}
}

func outputJSON(result *gameid.Result) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func outputText(result *gameid.Result) {
	fmt.Printf("Console: %s\n", result.Console)
	if result.ID != "" {
		fmt.Printf("ID: %s\n", result.ID)
	}
	if result.Title != "" {
		fmt.Printf("Title: %s\n", result.Title)
	}
	if result.InternalTitle != "" && result.InternalTitle != result.Title {
		fmt.Printf("Internal Title: %s\n", result.InternalTitle)
	}
	if result.Region != "" {
		fmt.Printf("Region: %s\n", result.Region)
	}

	// Print other metadata (skip those already printed)
	skipKeys := map[string]bool{
		"ID": true, "title": true, "internal_title": true, "region": true,
	}

	if len(result.Metadata) > 0 {
		var otherKeys []string
		for k := range result.Metadata {
			if !skipKeys[k] {
				otherKeys = append(otherKeys, k)
			}
		}

		if len(otherKeys) > 0 {
			fmt.Println("\nMetadata:")
			for _, k := range otherKeys {
				// Format key for display
				displayKey := strings.ReplaceAll(k, "_", " ")
				displayKey = strings.Title(displayKey)
				fmt.Printf("  %s: %s\n", displayKey, result.Metadata[k])
			}
		}
	}
}

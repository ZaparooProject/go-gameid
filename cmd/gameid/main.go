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

//nolint:gocognit,revive // Main entry point handles all CLI logic
func main() {
	flag.Usage = func() {
		_, _ = fmt.Fprint(os.Stderr, "Usage: "+os.Args[0]+" -i <file> [options]\n\n")
		_, _ = fmt.Fprint(os.Stderr, "Identifies video game files and returns metadata.\n\n")
		_, _ = fmt.Fprint(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		_, _ = fmt.Fprint(os.Stderr, "\nExamples:\n")
		_, _ = fmt.Fprint(os.Stderr, "  "+os.Args[0]+" -i game.gba\n")
		_, _ = fmt.Fprint(os.Stderr, "  "+os.Args[0]+" -i game.iso -c PSX\n")
		_, _ = fmt.Fprint(os.Stderr, "  "+os.Args[0]+" -i game.n64 -db gamedb.gob.gz -json\n")
	}
	flag.Parse()

	if *version {
		fmt.Println("gameid version " + appVersion) //nolint:revive // Output for CLI
		os.Exit(0)
	}

	if *listConsoles {
		fmt.Println("Supported consoles:") //nolint:revive // Output for CLI
		for _, c := range gameid.AllConsoles {
			fmt.Println("  " + string(c)) //nolint:revive // Output for CLI
		}
		os.Exit(0)
	}

	if *inputFile == "" {
		_, _ = fmt.Fprint(os.Stderr, "Error: input file required (-i)\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load database if specified
	var db *gameid.GameDatabase
	if *dbPath != "" {
		var loadErr error
		db, loadErr = gameid.LoadDatabase(*dbPath)
		if loadErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error loading database: %v\n", loadErr)
			os.Exit(1)
		}
	}

	// Identify the game
	var result *gameid.Result
	var err error

	//nolint:nestif // Console identification requires conditional branching
	if *console != "" {
		// Console specified, use it directly
		parsedConsole, parseErr := gameid.ParseConsole(*console)
		if parseErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: unknown console '%s'\n", *console)
			_, _ = fmt.Fprintf(os.Stderr, "Use -list-consoles to see supported consoles\n")
			os.Exit(1)
		}
		result, err = gameid.IdentifyWithConsole(*inputFile, parsedConsole, db)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error identifying game: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Auto-detect console
		result, err = gameid.Identify(*inputFile, db)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error identifying game: %v\n", err)
			os.Exit(1)
		}
	}

	// Output results
	if *jsonOutput {
		if jsonErr := outputJSON(result); jsonErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", jsonErr)
			os.Exit(1)
		}
	} else {
		outputText(result)
	}
}

func outputJSON(result *gameid.Result) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	return nil
}

//nolint:gocognit,revive // Output formatting requires many conditional checks
func outputText(result *gameid.Result) {
	fmt.Println("Console: " + string(result.Console)) //nolint:revive // Output for CLI
	if result.ID != "" {
		fmt.Println("ID: " + result.ID) //nolint:revive // Output for CLI
	}
	if result.Title != "" {
		fmt.Println("Title: " + result.Title) //nolint:revive // Output for CLI
	}
	if result.InternalTitle != "" && result.InternalTitle != result.Title {
		fmt.Println("Internal Title: " + result.InternalTitle) //nolint:revive // Output for CLI
	}
	if result.Region != "" {
		fmt.Println("Region: " + result.Region) //nolint:revive // Output for CLI
	}

	// Print other metadata (skip those already printed)
	skipKeys := map[string]bool{
		"ID": true, "title": true, "internal_title": true, "region": true,
	}

	if len(result.Metadata) > 0 {
		var otherKeys []string
		for key := range result.Metadata {
			if !skipKeys[key] {
				otherKeys = append(otherKeys, key)
			}
		}

		if len(otherKeys) > 0 {
			fmt.Println("\nMetadata:") //nolint:revive // Output for CLI
			for _, key := range otherKeys {
				// Format key for display
				displayKey := strings.ReplaceAll(key, "_", " ")
				//nolint:staticcheck // strings.Title is fine for simple ASCII
				displayKey = strings.Title(displayKey)
				//nolint:revive // Output for CLI
				fmt.Println("  " + displayKey + ": " + result.Metadata[key])
			}
		}
	}
}

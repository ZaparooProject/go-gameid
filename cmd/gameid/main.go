package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
	"github.com/wizzomafizzo/go-gameid/pkg/identifiers"
)

const version = "1.0.0"

var supportedConsoles = []string{
	"GBA", "GB", "GBC", "N64", "SNES", "Genesis",
	"PSX", "PS2", "GC", "Saturn", "SegaCD", "PSP",
}

func printError(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func printLog(msg string) {
	fmt.Fprint(os.Stderr, msg)
}

func getIdentifier(console string, db *database.GameDatabase) identifiers.Identifier {
	switch strings.ToUpper(console) {
	case "GBA":
		return identifiers.NewGBAIdentifier(db)
	case "GB", "GBC":
		return identifiers.NewGBIdentifier(db)
	case "N64":
		return identifiers.NewN64Identifier(db)
	case "SNES":
		return identifiers.NewSNESIdentifier(db)
	case "GENESIS":
		return identifiers.NewGenesisIdentifier(db)
	case "PSX":
		return identifiers.NewPSXIdentifier(db)
	case "PS2":
		return identifiers.NewPS2Identifier(db)
	case "GC":
		return identifiers.NewGameCubeIdentifier(db)
	case "SATURN":
		return identifiers.NewSaturnIdentifier(db)
	case "SEGACD":
		return identifiers.NewSegaCDIdentifier(db)
	case "PSP":
		return identifiers.NewPSPIdentifier(db)
	default:
		return nil
	}
}

func interactiveMode() (string, string) {
	reader := bufio.NewReader(os.Stdin)

	printLog(fmt.Sprintf("=== GameID v%s ===\n", version))

	// get game filename
	var inputFile string
	for inputFile == "" {
		printLog("Enter game filename (no quotes): ")
		input, _ := reader.ReadString('\n')
		inputFile = strings.TrimSpace(input)

		if inputFile != "" && !fileExists(inputFile) && !strings.HasPrefix(strings.ToLower(inputFile), "/dev/") {
			printLog(fmt.Sprintf("ERROR: File/folder not found: %s\n\n", inputFile))
			inputFile = ""
		}
	}

	// get console
	var console string
	for console == "" {
		printLog(fmt.Sprintf("Enter console (options: %s): ", strings.Join(supportedConsoles, ", ")))
		input, _ := reader.ReadString('\n')
		console = strings.TrimSpace(input)

		if getIdentifier(console, nil) == nil {
			printLog(fmt.Sprintf("ERROR: Invalid console: %s\n\n", console))
			console = ""
		}
	}

	return inputFile, console
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func main() {
	// define command line flags
	var (
		inputFile   = flag.String("i", "", "Input Game File")
		input       = flag.String("input", "", "Input Game File")
		console     = flag.String("c", "", "Console (options: "+strings.Join(supportedConsoles, ", ")+")")
		consoleL    = flag.String("console", "", "Console")
		databaseF   = flag.String("d", "", "GameID Database (db.pkl.gz)")
		databaseL   = flag.String("database", "", "GameID Database")
		outputFile  = flag.String("o", "stdout", "Output File")
		outputL     = flag.String("output", "stdout", "Output File")
		discUUID    = flag.String("disc_uuid", "", "Disc UUID (if already known)")
		discLabel   = flag.String("disc_label", "", "Disc Label / Volume ID (if already known)")
		delimiter   = flag.String("delimiter", "\t", "Delimiter")
		preferDB    = flag.Bool("prefer_gamedb", false, "Prefer Metadata in GameDB (rather than metadata loaded from game)")
		versionFlag = flag.Bool("version", false, fmt.Sprintf("Print GameID Version (%s)", version))
	)

	flag.Parse()

	// handle version flag
	if *versionFlag {
		fmt.Printf("GameID v%s\n", version)
		os.Exit(0)
	}

	// merge short and long flags
	if *inputFile == "" && *input != "" {
		*inputFile = *input
	}
	if *console == "" && *consoleL != "" {
		*console = *consoleL
	}
	if *databaseF == "" && *databaseL != "" {
		*databaseF = *databaseL
	}
	if *outputFile == "stdout" && *outputL != "stdout" {
		*outputFile = *outputL
	}

	// interactive mode if no args
	if *inputFile == "" || *console == "" {
		if len(os.Args) == 1 {
			*inputFile, *console = interactiveMode()
		} else {
			flag.Usage()
			os.Exit(1)
		}
	}

	// validate required args
	if *inputFile == "" {
		printError("Input file is required")
	}
	if *console == "" {
		printError("Console is required")
	}

	// check if input file exists
	if !fileExists(*inputFile) && !strings.HasPrefix(strings.ToLower(*inputFile), "/dev/") {
		printError(fmt.Sprintf("File/folder not found: %s", *inputFile))
	}

	// determine database path
	dbPath := *databaseF
	if dbPath == "" {
		// try to find database in standard locations
		possiblePaths := []string{
			"dbs/gameid_db.json",
			filepath.Join(filepath.Dir(os.Args[0]), "dbs/gameid_db.json"),
			filepath.Join(filepath.Dir(os.Args[0]), "../dbs/gameid_db.json"),
		}
		for _, path := range possiblePaths {
			if fileExists(path) {
				dbPath = path
				break
			}
		}
	}

	// load database
	var db *database.GameDatabase
	if dbPath != "" {
		var err error
		db, err = database.LoadDatabase(dbPath)
		if err != nil {
			// database is optional, so just warn
			fmt.Fprintf(os.Stderr, "Warning: Failed to load database: %v\n", err)
		}
	}

	// get identifier
	identifier := getIdentifier(*console, db)
	if identifier == nil {
		printError(fmt.Sprintf("Unknown console: %s", *console))
	}

	// identify game
	var result map[string]string
	var err error

	// Try to use IdentifierWithOptions if available
	if identWithOpts, ok := identifier.(identifiers.IdentifierWithOptions); ok {
		result, err = identWithOpts.IdentifyWithOptions(*inputFile, *discUUID, *discLabel, *preferDB)
	} else {
		result, err = identifier.Identify(*inputFile)
	}
	if err != nil {
		printError(fmt.Sprintf("Failed to identify game: %v", err))
	}

	if result == nil || len(result) == 0 {
		printError(fmt.Sprintf("%s game not found: %s", *console, *inputFile))
	}

	// replace empty string values with 'None'
	for k, v := range result {
		if strings.TrimSpace(v) == "" {
			result[k] = "None"
		}
	}

	// prepare output
	var outputWriter *os.File
	if *outputFile == "stdout" {
		outputWriter = os.Stdout
	} else {
		var err error
		outputWriter, err = os.Create(*outputFile)
		if err != nil {
			printError(fmt.Sprintf("Failed to create output file: %v", err))
		}
		defer outputWriter.Close()
	}

	// sort keys for consistent output
	keys := make([]string, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// write output
	for _, k := range keys {
		fmt.Fprintf(outputWriter, "%s%s%s\n", k, *delimiter, result[k])
	}
}

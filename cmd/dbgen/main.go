// Command dbgen downloads GameDB TSV files and generates the Go game database.
package main

import (
	"bufio"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/ZaparooProject/go-gameid/identifier"
)

// GameDB TSV URL template
const gameDBURLTemplate = "https://github.com/niemasd/GameDB-%s/releases/latest/download/%s.data.tsv"

// Consoles to download
var consoles = []string{
	"GB", "GBA", "GBC", "GC", "Genesis", "N64", "NeoGeoCD", "NES", "PSP", "PSX", "PS2", "Saturn", "SegaCD", "SNES",
}

// gbKey is the lookup key for GB/GBC games
type gbKey struct {
	Title    string
	Checksum uint16
}

// snesKey is the lookup key for SNES games
type snesKey struct {
	DeveloperID  int
	InternalName string
	ROMVersion   int
	Checksum     int
}

// neogeoCDKey is the lookup key for NeoGeoCD games
type neogeoCDKey struct {
	UUID     string
	VolumeID string
}

// Database structure matching the main package
type Database struct {
	GB         map[gbKey]map[string]string
	GBA        map[string]map[string]string
	GC         map[string]map[string]string
	Genesis    map[string]map[string]string
	N64        map[string]map[string]string
	NES        map[int]map[string]string
	PSP        map[string]map[string]string
	PSX        map[string]map[string]string
	PS2        map[string]map[string]string
	Saturn     map[string]map[string]string
	SegaCD     map[string]map[string]string
	SNES       map[snesKey]map[string]string
	NeoGeoCD   map[neogeoCDKey]map[string]string
	IDPrefixes map[identifier.Console][]string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <output.gob.gz>\n", os.Args[0])
		os.Exit(1)
	}

	outputPath := os.Args[1]

	db := &Database{
		GB:         make(map[gbKey]map[string]string),
		GBA:        make(map[string]map[string]string),
		GC:         make(map[string]map[string]string),
		Genesis:    make(map[string]map[string]string),
		N64:        make(map[string]map[string]string),
		NES:        make(map[int]map[string]string),
		PSP:        make(map[string]map[string]string),
		PSX:        make(map[string]map[string]string),
		PS2:        make(map[string]map[string]string),
		Saturn:     make(map[string]map[string]string),
		SegaCD:     make(map[string]map[string]string),
		SNES:       make(map[snesKey]map[string]string),
		NeoGeoCD:   make(map[neogeoCDKey]map[string]string),
		IDPrefixes: make(map[identifier.Console][]string),
	}

	for _, console := range consoles {
		fmt.Printf("Loading GameDB-%s...\n", console)
		if err := loadConsole(db, console); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load %s: %v\n", console, err)
		}
	}

	// Apply fixups
	applyFixups(db)

	// Save database
	fmt.Printf("Writing database to %s...\n", outputPath)
	if err := saveDatabase(db, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done!")
}

func loadConsole(db *Database, console string) error {
	url := fmt.Sprintf(gameDBURLTemplate, console, console)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)

	// Read header
	if !scanner.Scan() {
		return fmt.Errorf("empty TSV")
	}
	header := strings.Split(scanner.Text(), "\t")
	fieldIndex := make(map[string]int)
	for i, field := range header {
		fieldIndex[strings.TrimSpace(field)] = i
	}

	// Read data rows
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")

		// Build metadata map (all fields except ID)
		metadata := make(map[string]string)
		for field, idx := range fieldIndex {
			if idx < len(fields) && field != "ID" {
				value := strings.TrimSpace(fields[idx])
				if value != "" {
					metadata[field] = value
				}
			}
		}

		// Get ID
		idIdx, hasID := fieldIndex["ID"]
		if !hasID || idIdx >= len(fields) {
			continue
		}
		id := strings.TrimSpace(fields[idIdx])
		if id == "" {
			continue
		}

		// Add to appropriate database based on console
		switch console {
		case "GB", "GBC":
			addGB(db, id, metadata)
		case "GBA":
			db.GBA[id] = metadata
		case "GC":
			addGC(db, id, metadata)
		case "Genesis":
			addGenesis(db, id, metadata)
		case "N64":
			addN64(db, id, metadata)
		case "NeoGeoCD":
			addNeoGeoCD(db, metadata)
		case "NES":
			addNES(db, id, metadata)
		case "PSP":
			db.PSP[id] = metadata
		case "PSX":
			addPSXPS2(db, id, metadata, identifier.ConsolePSX)
		case "PS2":
			addPSXPS2(db, id, metadata, identifier.ConsolePS2)
		case "Saturn":
			addSaturn(db, id, metadata)
		case "SegaCD":
			addSegaCD(db, id, metadata)
		case "SNES":
			addSNES(db, id, metadata)
		}
	}

	return scanner.Err()
}

func addGB(db *Database, id string, metadata map[string]string) {
	// Key is (internal_title, global_checksum)
	title := metadata["internal_title"]
	checksumStr := metadata["global_checksum_expected"]
	if title == "" || checksumStr == "" {
		return
	}
	checksum, err := strconv.ParseUint(strings.TrimPrefix(checksumStr, "0x"), 16, 16)
	if err != nil {
		checksum64, err2 := strconv.ParseUint(checksumStr, 0, 16)
		if err2 != nil {
			return
		}
		checksum = checksum64
	}
	key := gbKey{Title: title, Checksum: uint16(checksum)}
	db.GB[key] = metadata
}

func addGC(db *Database, id string, metadata map[string]string) {
	// Extract serial from ID (format: "XXX-YYYY" -> "YYYY")
	parts := strings.Split(id, "-")
	if len(parts) >= 2 {
		id = strings.TrimSpace(parts[1])
	}
	db.GC[id] = metadata
}

func addGenesis(db *Database, id string, metadata map[string]string) {
	// Normalize: take first part, remove dashes and spaces
	parts := strings.Split(strings.TrimSpace(id), " ")
	id = strings.ReplaceAll(parts[0], "-", "")
	id = strings.ReplaceAll(id, " ", "")
	id = strings.TrimSpace(id)
	db.Genesis[id] = metadata
}

func addN64(db *Database, id string, metadata map[string]string) {
	// Extract from ID (format: "XXX-YYYY" -> "YYY" where Y is 2 chars + country)
	parts := strings.Split(id, "-")
	if len(parts) >= 2 {
		cartID := parts[1]
		if len(cartID) >= 3 {
			id = cartID[1:4] // Skip first char, take next 3
		}
	}
	db.N64[id] = metadata
}

func addNeoGeoCD(db *Database, metadata map[string]string) {
	uuid := metadata["uuid"]
	volumeID := metadata["volume_ID"]
	if volumeID == "" {
		return
	}

	// Add with (uuid, volume_ID) key
	if uuid != "" {
		key := neogeoCDKey{UUID: uuid, VolumeID: volumeID}
		db.NeoGeoCD[key] = metadata
	}
	// Also add with just volume_ID
	key := neogeoCDKey{UUID: "", VolumeID: volumeID}
	db.NeoGeoCD[key] = metadata
}

func addNES(db *Database, id string, metadata map[string]string) {
	// ID is CRC32 in hex
	crc, err := strconv.ParseUint(strings.TrimPrefix(id, "0x"), 16, 32)
	if err != nil {
		crc64, err2 := strconv.ParseUint(id, 16, 32)
		if err2 != nil {
			return
		}
		crc = crc64
	}
	db.NES[int(crc)] = metadata
}

func addPSXPS2(db *Database, id string, metadata map[string]string, console identifier.Console) {
	// Normalize: replace dashes with underscores
	id = strings.ReplaceAll(id, "-", "_")

	if console == identifier.ConsolePSX {
		db.PSX[id] = metadata
		// Also add by redump_name if present
		if redumpName, ok := metadata["redump_name"]; ok && redumpName != "" {
			db.PSX[redumpName] = metadata
		}
	} else {
		db.PS2[id] = metadata
		if redumpName, ok := metadata["redump_name"]; ok && redumpName != "" {
			db.PS2[redumpName] = metadata
		}
	}
}

func addSaturn(db *Database, id string, metadata map[string]string) {
	// Normalize: take first part, remove dashes and spaces
	parts := strings.Split(strings.TrimSpace(id), " ")
	id = strings.ReplaceAll(parts[0], "-", "")
	id = strings.ReplaceAll(id, " ", "")
	id = strings.TrimSpace(id)
	db.Saturn[id] = metadata
}

func addSegaCD(db *Database, id string, metadata map[string]string) {
	// Normalize: remove dashes and spaces
	id = strings.ReplaceAll(id, "-", "")
	id = strings.ReplaceAll(id, " ", "")
	id = strings.TrimSpace(id)
	db.SegaCD[id] = metadata
}

func addSNES(db *Database, id string, metadata map[string]string) {
	// Key is (developer_id, internal_name_hex, rom_version, checksum)
	developerIDStr := metadata["developer_ID"]
	internalName := metadata["internal_title"]
	romVersionStr := metadata["rom_version"]
	checksumStr := metadata["checksum"]

	if internalName == "" || checksumStr == "" {
		return
	}

	developerID, _ := strconv.ParseInt(strings.TrimPrefix(developerIDStr, "0x"), 16, 32)
	romVersion, _ := strconv.ParseInt(strings.TrimPrefix(romVersionStr, "0x"), 0, 32)
	checksum, _ := strconv.ParseInt(strings.TrimPrefix(checksumStr, "0x"), 16, 32)

	key := snesKey{
		DeveloperID:  int(developerID),
		InternalName: internalName,
		ROMVersion:   int(romVersion),
		Checksum:     int(checksum),
	}
	db.SNES[key] = metadata
}

func applyFixups(db *Database) {
	// Build ID prefixes for PSX and PS2
	psxPrefixes := buildIDPrefixes(db.PSX)
	ps2Prefixes := buildIDPrefixes(db.PS2)

	db.IDPrefixes[identifier.ConsolePSX] = psxPrefixes
	db.IDPrefixes[identifier.ConsolePS2] = ps2Prefixes
}

func buildIDPrefixes(games map[string]map[string]string) []string {
	counts := make(map[string]int)
	for id := range games {
		parts := strings.Split(id, "_")
		if len(parts) > 0 {
			prefix := strings.TrimSpace(parts[0])
			counts[prefix]++
		}
	}

	// Sort by count (descending)
	type prefixCount struct {
		prefix string
		count  int
	}
	var prefixes []prefixCount
	for p, c := range counts {
		prefixes = append(prefixes, prefixCount{p, c})
	}

	// Simple sort by count
	for i := 0; i < len(prefixes); i++ {
		for j := i + 1; j < len(prefixes); j++ {
			if prefixes[j].count > prefixes[i].count {
				prefixes[i], prefixes[j] = prefixes[j], prefixes[i]
			}
		}
	}

	result := make([]string, len(prefixes))
	for i, p := range prefixes {
		result[i] = p.prefix
	}
	return result
}

func saveDatabase(db *Database, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	enc := gob.NewEncoder(gz)
	return enc.Encode(db)
}

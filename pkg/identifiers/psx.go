package identifiers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
	"github.com/wizzomafizzo/go-gameid/pkg/iso9660"
)

// Common PSX/PS2 serial prefixes
var psxPrefixes = []string{
	"SLUS", "SLES", "SCUS", "SLPM", "SCES", "SIPS", "SLPS", "SCPS",
	"SLED", "SLKA", "SLAJ", "PCPX", "PAPX", "SCZS", "PSRM",
}

var ps2Prefixes = []string{
	"SLUS", "SLES", "SCUS", "SLPM", "SCES", "SIPS", "SLPS", "SCPS",
	"SLED", "SLKA", "SLAJ", "PCPX", "PAPX", "SCZS", "PSRM",
	"SCAJ", "SCKA", "SLBA", "SCCS",
}

// PSXIdentifier implements game identification for PlayStation
type PSXIdentifier struct {
	db *database.GameDatabase
}

// NewPSXIdentifier creates a new PSX identifier
func NewPSXIdentifier(db *database.GameDatabase) *PSXIdentifier {
	return &PSXIdentifier{db: db}
}

// Console returns the console name
func (p *PSXIdentifier) Console() string {
	return "PSX"
}

// Identify identifies a PSX game and returns its metadata
func (p *PSXIdentifier) Identify(path string) (map[string]string, error) {
	return p.IdentifyWithOptions(path, "", "", false)
}

// IdentifyWithOptions identifies a PSX game with additional parameters
func (p *PSXIdentifier) IdentifyWithOptions(path, discUUID, discLabel string, preferDB bool) (map[string]string, error) {
	// Open ISO file or directory
	disc, err := iso9660.OpenImage(path, discUUID, discLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to open disc: %w", err)
	}
	defer disc.Close()

	return p.identifyPSXPS2(disc, "PSX", psxPrefixes, discUUID, discLabel, preferDB)
}

// PS2Identifier implements game identification for PlayStation 2
type PS2Identifier struct {
	db *database.GameDatabase
}

// NewPS2Identifier creates a new PS2 identifier
func NewPS2Identifier(db *database.GameDatabase) *PS2Identifier {
	return &PS2Identifier{db: db}
}

// Console returns the console name
func (p *PS2Identifier) Console() string {
	return "PS2"
}

// Identify identifies a PS2 game and returns its metadata
func (p *PS2Identifier) Identify(path string) (map[string]string, error) {
	return p.IdentifyWithOptions(path, "", "", false)
}

// IdentifyWithOptions identifies a PS2 game with additional parameters
func (p *PS2Identifier) IdentifyWithOptions(path, discUUID, discLabel string, preferDB bool) (map[string]string, error) {
	// Open ISO file or directory
	disc, err := iso9660.OpenImage(path, discUUID, discLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to open disc: %w", err)
	}
	defer disc.Close()

	identifier := &PSXIdentifier{db: p.db}
	return identifier.identifyPSXPS2(disc, "PS2", ps2Prefixes, discUUID, discLabel, preferDB)
}

// identifyPSXPS2 is the shared implementation for PSX and PS2
func (p *PSXIdentifier) identifyPSXPS2(disc iso9660.DiscImage, console string, prefixes []string, discUUID, discLabel string, preferDB bool) (map[string]string, error) {
	result := make(map[string]string)
	var serial string
	pvd := disc.GetPVD()

	// Add disc UUID and label if provided
	if discUUID != "" {
		result["disc_uuid"] = discUUID
	}
	if discLabel != "" {
		result["disc_label"] = discLabel
	}

	// Get list of files in root directory
	files, err := disc.ListFiles(true)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	// Collect root file names for output
	var rootFiles []string
	for _, file := range files {
		filename := strings.TrimPrefix(file.Name, "/")
		rootFiles = append(rootFiles, filename)
	}

	// Try to find serial from filename pattern (SXXX_XXX.XX)
serialFoundLoop:
	for _, file := range files {
		filename := strings.ToUpper(strings.TrimPrefix(file.Name, "/"))

		// Check each prefix
		for _, prefix := range prefixes {
			if strings.HasPrefix(filename, prefix) {
				serial = extractPSXSerial(filename)
				if serial != "" {
					result["ID"] = strings.ReplaceAll(serial, "_", "-")
					// Try database lookup
					if p.db != nil {
						// First try direct lookup
						if gameData, found := p.db.LookupGame(console, serial); found {
							for key, value := range gameData {
								// Override existing data if preferDB is set, otherwise only add new
								_, exists := result[key]
								if preferDB || !exists {
									result[key] = value
								}
							}
							// Python doesn't output ID field
							break serialFoundLoop
						}

						// If that fails and we have more than just the prefix, try with underscore after prefix
						if len(serial) > len(prefix)+1 && !strings.Contains(serial[len(prefix):len(prefix)+1], "_") {
							altSerial := serial[:len(prefix)] + "_" + serial[len(prefix)+1:]
							if gameData, found := p.db.LookupGame(console, altSerial); found {
								for key, value := range gameData {
									// Override existing data if preferDB is set, otherwise only add new
									_, exists := result[key]
									if preferDB || !exists {
										result[key] = value
									}
								}
								serial = altSerial
								result["ID"] = strings.ReplaceAll(serial, "_", "-")
								// Python doesn't output ID field
								break serialFoundLoop
							}
						}
					}
					break serialFoundLoop
				}
			}
		}
	}

	// If no serial found from files, try volume ID
	if serial == "" && pvd != nil {
		volumeID := pvd.VolumeID
		if volumeID != "" {
			// Convert volume ID to serial format
			serial = strings.ReplaceAll(volumeID, "-", "_")

			// Handle case where there might be extra underscores
			parts := strings.Split(serial, "_")
			if len(parts) >= 2 {
				serial = parts[0] + "_" + parts[1]
			}

			// Try database lookup with volume ID serial
			if p.db != nil {
				if gameData, found := p.db.LookupGame(console, serial); found {
					for key, value := range gameData {
						// Override existing data if preferDB is set, otherwise only add new
						_, exists := result[key]
						if preferDB || !exists {
							result[key] = value
						}
					}
					result["ID"] = strings.ReplaceAll(serial, "_", "-")
				}
			}
		}
	}

	// Add ISO metadata
	if pvd != nil {
		if pvd.CreationDateTime != "" && result["uuid"] == "" {
			result["uuid"] = pvd.CreationDateTime
		}
		if pvd.VolumeID != "" && result["volume_ID"] == "" {
			result["volume_ID"] = pvd.VolumeID
		}
	}

	// Add root files list
	sort.Strings(rootFiles)
	result["root_files"] = strings.Join(rootFiles, " / ")

	// Python doesn't output ID field

	return result, nil
}

// extractPSXSerial extracts a serial number from a PSX/PS2 filename
func extractPSXSerial(filename string) string {
	// Remove extension if it exists
	if idx := strings.LastIndex(filename, "."); idx > 0 {
		// Check what comes after the dot
		afterDot := filename[idx+1:]

		// If it's exactly 2 digits and looks like part of a serial (e.g., SLUS_012.34)
		// AND the character before the dot is a digit, keep it
		if len(afterDot) == 2 && isNumeric(afterDot) && idx > 0 && isNumeric(string(filename[idx-1])) {
			// This is part of the serial, don't remove it
			filename = strings.ReplaceAll(filename, ".", "")
		} else {
			// This is a file extension or version number, remove it
			filename = filename[:idx]
		}
	}

	// Replace common delimiters with underscore
	serial := strings.ReplaceAll(filename, "-", "_")
	serial = strings.ReplaceAll(serial, " ", "_")

	// Clean up multiple underscores
	for strings.Contains(serial, "__") {
		serial = strings.ReplaceAll(serial, "__", "_")
	}

	return serial
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

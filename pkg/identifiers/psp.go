package identifiers

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
	"github.com/wizzomafizzo/go-gameid/pkg/iso9660"
)

// PSPIdentifier implements game identification for PlayStation Portable
type PSPIdentifier struct {
	db *database.GameDatabase
}

// NewPSPIdentifier creates a new PSP identifier
func NewPSPIdentifier(db *database.GameDatabase) *PSPIdentifier {
	return &PSPIdentifier{db: db}
}

// Console returns the console name
func (p *PSPIdentifier) Console() string {
	return "PSP"
}

// Identify identifies a PSP game and returns its metadata
func (p *PSPIdentifier) Identify(path string) (map[string]string, error) {
	return p.IdentifyWithOptions(path, "", "", false)
}

// IdentifyWithOptions identifies a PSP game with additional parameters
func (p *PSPIdentifier) IdentifyWithOptions(path, discUUID, discLabel string, preferDB bool) (map[string]string, error) {
	// Open ISO file
	iso, err := iso9660.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open ISO: %w", err)
	}
	defer iso.Close()

	// Look for UMD_DATA.BIN file
	files, err := iso.ListFiles(true)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	var umdFile *iso9660.FileEntry
	for _, file := range files {
		if strings.ToUpper(strings.TrimPrefix(file.Name, "/")) == "UMD_DATA.BIN" {
			umdFile = &file
			break
		}
	}

	if umdFile == nil {
		return nil, fmt.Errorf("UMD_DATA.BIN not found")
	}

	// Read UMD_DATA.BIN
	umdData, err := iso.ReadFile(umdFile.LBA, umdFile.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to read UMD_DATA.BIN: %w", err)
	}

	// Extract serial from UMD data
	// Read until we hit a '|' character or end of data
	serial := ""
	pipeIndex := bytes.IndexByte(umdData, '|')
	if pipeIndex > 0 {
		serial = string(umdData[:pipeIndex])
	} else if pipeIndex == -1 && len(umdData) > 0 {
		// No pipe found, use entire content
		serial = string(umdData)
	}

	serial = strings.TrimSpace(serial)
	if serial == "" {
		return nil, fmt.Errorf("empty serial in UMD_DATA.BIN")
	}

	// Build result - always return basic metadata
	result := make(map[string]string)
	result["ID"] = serial
	result["title"] = serial // Default title is the serial

	// Add disc UUID and label if provided
	if discUUID != "" {
		result["disc_uuid"] = discUUID
	}
	if discLabel != "" {
		result["disc_label"] = discLabel
	}

	// Try to look up game in database to enhance with additional metadata
	if p.db != nil {
		if gameData, found := p.db.LookupGame("PSP", serial); found {
			// Add database fields to result
			for key, value := range gameData {
				// Override existing data if preferDB is set, otherwise only add new
				_, exists := result[key]
				if preferDB || !exists {
					result[key] = value
				}
			}
			// Always keep the serial as ID
			result["ID"] = serial
		}
	}

	return result, nil
}

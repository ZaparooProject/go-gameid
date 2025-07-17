package identifiers

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
	"github.com/wizzomafizzo/go-gameid/pkg/iso9660"
)

var (
	// SegaCD magic words
	segaCDMagicWords = [][]byte{
		[]byte("SEGADISCSYSTEM"),
		[]byte("SEGABOOTDISC"),
		[]byte("SEGADISC"),
		[]byte("SEGADATADISC"),
	}
)

// SegaCDIdentifier implements game identification for Sega CD/Mega CD
type SegaCDIdentifier struct {
	db *database.GameDatabase
}

// NewSegaCDIdentifier creates a new SegaCD identifier
func NewSegaCDIdentifier(db *database.GameDatabase) *SegaCDIdentifier {
	return &SegaCDIdentifier{db: db}
}

// Console returns the console name
func (s *SegaCDIdentifier) Console() string {
	return "SegaCD"
}

// Identify identifies a SegaCD game and returns its metadata
func (s *SegaCDIdentifier) Identify(path string) (map[string]string, error) {
	return s.IdentifyWithOptions(path, "", "", false)
}

// IdentifyWithOptions identifies a SegaCD game with additional parameters
func (s *SegaCDIdentifier) IdentifyWithOptions(path, discUUID, discLabel string, preferDB bool) (map[string]string, error) {
	// Open disc image (ISO, CUE/BIN, or mounted directory)
	disc, err := iso9660.OpenImage(path, discUUID, discLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to open disc: %w", err)
	}
	defer disc.Close()

	// Read header (0x300 bytes) from the disc image
	initialHeaderBytes, err := disc.ReadFile(0, 0x300)
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}
	if len(initialHeaderBytes) < 0x210 { // Minimum size needed for header fields
		return nil, fmt.Errorf("file too small")
	}

	// Search for magic word
	magicOffset := -1
	for _, magicWord := range segaCDMagicWords {
		for i := 0; i <= len(initialHeaderBytes)-len(magicWord); i++ {
			if bytes.Equal(initialHeaderBytes[i:i+len(magicWord)], magicWord) {
				magicOffset = i
				break
			}
		}
		if magicOffset != -1 {
			break
		}
	}

	if magicOffset == -1 {
		return nil, fmt.Errorf("SegaCD magic word not found")
	}

	// Ensure we have enough data after magic word for full header
	requiredSize := magicOffset + 0x1A0
	var header []byte
	if uint32(len(initialHeaderBytes)) < uint32(requiredSize) {
		fullHeader, err := disc.ReadFile(0, uint32(requiredSize))
		if err != nil {
			return nil, fmt.Errorf("failed to read full header: %w", err)
		}
		header = fullHeader
	} else {
		header = initialHeaderBytes
	}

	// Extract fields
	result := make(map[string]string)

	// Helper function to decode bytes to string
	decodeString := func(data []byte) string {
		// Python preserves raw bytes including nulls
		return string(data)
	}

	// Extract all fields
	result["disc_ID"] = decodeString(header[magicOffset+0x000 : magicOffset+0x010])
	result["volume_ID"] = decodeString(header[magicOffset+0x010 : magicOffset+0x01B])
	result["system_name"] = decodeString(header[magicOffset+0x020 : magicOffset+0x02B])

	// Build date at magic + 0x050 (8 bytes) - MMDDYYYY format
	buildDateRaw := decodeString(header[magicOffset+0x050 : magicOffset+0x058])
	if len(buildDateRaw) == 8 {
		result["build_date"] = fmt.Sprintf("%s-%s-%s",
			buildDateRaw[4:8], buildDateRaw[0:2], buildDateRaw[2:4])
	} else {
		result["build_date"] = buildDateRaw
	}

	result["system_type"] = decodeString(header[magicOffset+0x100 : magicOffset+0x110])

	// Release year
	releaseYearBytes := header[magicOffset+0x118 : magicOffset+0x11C]
	releaseYear := decodeString(releaseYearBytes)
	// Try to parse as int
	if yearInt, err := strconv.Atoi(releaseYear); err == nil {
		result["release_year"] = strconv.Itoa(yearInt)
	} else {
		result["release_year"] = releaseYear
	}

	result["release_month"] = decodeString(header[magicOffset+0x11D : magicOffset+0x120])
	result["title_domestic"] = decodeString(header[magicOffset+0x120 : magicOffset+0x150])
	result["title_overseas"] = decodeString(header[magicOffset+0x150 : magicOffset+0x180])

	// ID field (needs special parsing)
	idRaw := decodeString(header[magicOffset+0x180 : magicOffset+0x190])

	// Device support field - process character by character like Python
	deviceBytes := header[magicOffset+0x190 : magicOffset+0x1A0]
	deviceSupport := []string{}
	for _, b := range deviceBytes {
		c := string(b)
		if val, ok := GenesisDeviceSupport[b]; ok {
			deviceSupport = append(deviceSupport, val)
		} else {
			// Python includes the actual character, including spaces
			deviceSupport = append(deviceSupport, c)
		}
	}
	if len(deviceSupport) > 0 {
		// Sort and join like Python
		sort.Strings(deviceSupport)
		result["device_support"] = strings.Join(deviceSupport, " / ")
	} else {
		result["device_support"] = decodeString(deviceBytes)
	}

	// Region support field at offset 0x1F0
	if magicOffset+0x1F3 <= len(header) {
		regionBytes := header[magicOffset+0x1F0 : magicOffset+0x1F3]
		regionSupport := []string{}
		for _, b := range regionBytes {
			if b >= '!' && b <= '~' {
				c := string(b)
				if val, ok := GenesisRegionSupport[b]; ok {
					regionSupport = append(regionSupport, val)
				} else {
					regionSupport = append(regionSupport, c)
				}
			}
		}
		if len(regionSupport) > 0 {
			result["region_support"] = strings.Join(regionSupport, " / ")
		} else {
			// Python sets empty region_support when no valid characters found
			result["region_support"] = ""
		}
	} else {
		// Set empty region_support when offset is out of bounds
		result["region_support"] = ""
	}

	// Parse ID field
	if idRaw != "" {
		parts := strings.Fields(idRaw)
		if len(parts) == 3 {
			// Format: "GM MK-4402 -00"
			result["disc_kind"] = strings.Trim(parts[0], "-")
			result["ID"] = strings.Trim(parts[1], "-")
			result["version"] = strings.Trim(parts[2], "-")
		} else if len(parts) == 2 {
			// Format: "GM MK-4407-00-01" or similar
			result["disc_kind"] = parts[0]
			// The second part contains ID and version
			if strings.Count(parts[1], "-") >= 2 {
				// Find the last dash to separate ID from version
				lastDash := strings.LastIndex(parts[1], "-")
				if lastDash > 0 && lastDash < len(parts[1])-1 {
					result["ID"] = parts[1][:lastDash]
					result["version"] = parts[1][lastDash+1:]
				} else {
					result["ID"] = parts[1]
				}
			} else {
				result["ID"] = strings.Trim(parts[1], "-")
			}
		} else if len(parts) == 1 {
			// Just ID
			result["ID"] = idRaw
		}
	}

	// Try to look up game in database
	serial := result["ID"]
	if s.db != nil && serial != "" {
		if gameData, found := s.db.LookupGame("SegaCD", serial); found {
			// Add database fields to result
			for key, value := range gameData {
				// Override existing data if preferDB is set, otherwise only add new
				_, exists := result[key]
				if preferDB || !exists {
					result[key] = value
				}
			}
		}
	}

	// If no title from database, use overseas title (matching Python behavior)
	if result["title"] == "" {
		result["title"] = result["title_overseas"]
	}

	return result, nil
}

package identifiers

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
	"github.com/wizzomafizzo/go-gameid/pkg/iso9660"
)

var (
	// Saturn magic word
	saturnMagicWord = []byte("SEGA SEGASATURN")

	// Saturn device support mapping
	saturnDeviceSupport = map[byte]string{
		'J': "Joypad",
		'M': "Mouse",
		'G': "Gun",
		'W': "RAM Cart",
		'S': "Steering Wheel",
		'A': "Virtua Stick or Analog Controller",
		'E': "Analog Controller (3D-pad)",
		'T': "Multi-Tap",
		'C': "Link Cable",
		'D': "Link Cable (Direct Link)",
		'X': "X-Band or Netlink Modem",
		'K': "Keyboard",
		'Q': "Pachinko Controller",
		'F': "Floppy Disk Drive",
		'R': "ROM Cart",
		'P': "Video CD Card (MPEG Movie Card)",
	}

	// Saturn target area mapping
	saturnTargetAreas = map[byte]string{
		'J': "Japan",
		'T': "Asia NTSC (Taiwan, Philippines)",
		'U': "North America (USA, Canada)",
		'B': "Central and South America NTSC (Brazil)",
		'K': "Korea",
		'A': "East Asia PAL (China, Middle and Near East)",
		'E': "Europe PAL",
		'L': "Central and South America PAL",
	}
)

// SaturnIdentifier implements game identification for Sega Saturn
type SaturnIdentifier struct {
	db *database.GameDatabase
}

// NewSaturnIdentifier creates a new Saturn identifier
func NewSaturnIdentifier(db *database.GameDatabase) *SaturnIdentifier {
	return &SaturnIdentifier{db: db}
}

// Console returns the console name
func (s *SaturnIdentifier) Console() string {
	return "Saturn"
}

// Identify identifies a Saturn game and returns its metadata
func (s *SaturnIdentifier) Identify(path string) (map[string]string, error) {
	return s.IdentifyWithOptions(path, "", "", false)
}

// IdentifyWithOptions identifies a Saturn game with additional parameters
func (s *SaturnIdentifier) IdentifyWithOptions(path, discUUID, discLabel string, preferDB bool) (map[string]string, error) {
	// Open disc image (ISO, CUE/BIN, or mounted directory)
	disc, err := iso9660.OpenImage(path, discUUID, discLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to open disc: %w", err)
	}
	defer disc.Close()

	// Read header (0x100 bytes should be enough to find magic word) from the disc image
	initialHeaderBytes, err := disc.ReadFile(0, 0x100)
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}
	if len(initialHeaderBytes) < len(saturnMagicWord) {
		return nil, fmt.Errorf("file too small")
	}

	// Search for magic word
	magicOffset := -1
	for i := 0; i <= len(initialHeaderBytes)-len(saturnMagicWord); i++ {
		if bytes.Equal(initialHeaderBytes[i:i+len(saturnMagicWord)], saturnMagicWord) {
			magicOffset = i
			break
		}
	}

	if magicOffset == -1 {
		return nil, fmt.Errorf("Saturn magic word not found")
	}

	// Ensure we have enough data after magic word for full header
	requiredSize := magicOffset + 0xD0
	var header []byte
	if uint32(len(initialHeaderBytes)) < uint32(requiredSize) {
		// Read more data if needed
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

	// Manufacturer ID at magic + 0x10 (16 bytes)
	mfgID := strings.TrimSpace(string(header[magicOffset+0x10 : magicOffset+0x20]))
	result["manufacturer_ID"] = mfgID

	// Product ID at magic + 0x20 (10 bytes)
	productID := strings.TrimSpace(string(header[magicOffset+0x20 : magicOffset+0x2A]))
	// Extract just the first part before any spaces
	parts := strings.Fields(productID)
	if len(parts) > 0 {
		result["ID"] = parts[0]
	} else {
		result["ID"] = productID
	}

	// Version at magic + 0x2A (6 bytes)
	version := strings.TrimSpace(string(header[magicOffset+0x2A : magicOffset+0x30]))
	result["version"] = version

	// Release date at magic + 0x30 (8 bytes) - format YYYYMMDD
	releaseDateRaw := strings.TrimSpace(string(header[magicOffset+0x30 : magicOffset+0x38]))
	if len(releaseDateRaw) == 8 {
		result["release_date"] = fmt.Sprintf("%s-%s-%s",
			releaseDateRaw[0:4], releaseDateRaw[4:6], releaseDateRaw[6:8])
	} else {
		result["release_date"] = releaseDateRaw
	}

	// Device info at magic + 0x38 (8 bytes)
	deviceInfo := strings.TrimSpace(string(header[magicOffset+0x38 : magicOffset+0x40]))
	result["device_info"] = deviceInfo

	// Target area at magic + 0x40 (16 bytes)
	targetAreaBytes := header[magicOffset+0x40 : magicOffset+0x50]
	targetAreas := []string{}
	for _, b := range targetAreaBytes {
		if b == 0 || b == ' ' {
			continue
		}
		if area, ok := saturnTargetAreas[b]; ok {
			targetAreas = append(targetAreas, area)
		} else if b >= 32 && b <= 126 { // Printable ASCII
			targetAreas = append(targetAreas, string(b))
		}
	}
	result["target_area"] = strings.Join(targetAreas, " / ")

	// Device support at magic + 0x50 (16 bytes)
	deviceSupportBytes := header[magicOffset+0x50 : magicOffset+0x60]
	deviceSupports := []string{}
	for _, b := range deviceSupportBytes {
		if b == 0 || b == ' ' {
			continue
		}
		if device, ok := saturnDeviceSupport[b]; ok {
			deviceSupports = append(deviceSupports, device)
		} else if b >= 32 && b <= 126 { // Printable ASCII
			deviceSupports = append(deviceSupports, string(b))
		}
	}
	result["device_support"] = strings.Join(deviceSupports, " / ")

	// Internal title at magic + 0x60 (112 bytes, up to 0xD0)
	titleBytes := header[magicOffset+0x60 : magicOffset+0xD0]
	// Try to decode as string
	internalTitle := ""
	titleEnd := len(titleBytes)
	for i, b := range titleBytes {
		if b == 0 {
			titleEnd = i
			break
		}
	}
	internalTitle = strings.TrimSpace(string(titleBytes[:titleEnd]))
	result["internal_title"] = internalTitle

	// Create serial for database lookup (remove dashes and spaces)
	serial := strings.ReplaceAll(result["ID"], "-", "")
	serial = strings.ReplaceAll(serial, " ", "")
	serial = strings.TrimSpace(serial)

	// Try to look up game in database
	if s.db != nil && serial != "" {
		if gameData, found := s.db.LookupGame("Saturn", serial); found {
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

	// If no title from database, use internal title
	if result["title"] == "" {
		result["title"] = internalTitle
	}

	return result, nil
}

package identifiers

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
	"github.com/wizzomafizzo/go-gameid/pkg/fileio"
)

// Genesis magic words that identify a valid ROM
var genesisMagicWords = []string{
	"SEGA GENESIS",
	"SEGA MEGA DRIVE",
	"SEGA 32X",
	"SEGA EVERDRIVE",
	"SEGA SSF",
	"SEGA MEGAWIFI",
	"SEGA PICO",
	"SEGA TERA68K",
	"SEGA TERA286",
}

// GenesisDeviceSupport maps device support codes to their descriptions
var GenesisDeviceSupport = map[byte]string{
	'J': "3-button Controller",
	'6': "6-button Controller",
	'0': "Master System Controller",
	'A': "Analog Joystick",
	'4': "Multitap",
	'G': "Lightgun",
	'L': "Activator",
	'M': "Mouse",
	'B': "Trackball",
	'T': "Tablet",
	'V': "Paddle",
	'K': "Keyboard or Keypad",
	'R': "RS-232",
	'P': "Printer",
	'C': "CD-ROM (Sega CD)",
	'F': "Floppy Drive",
	'D': "Download",
}

// GenesisRegionSupport maps region codes to their descriptions
var GenesisRegionSupport = map[byte]string{
	'J': "Japan",
	'U': "Americas",
	'E': "Europe",
}

// Software type codes
var genesisSoftwareTypes = map[string]string{
	"GM": "Game",
	"AI": "Aid",
	"OS": "Boot ROM (TMSS)",
	"BR": "Boot ROM (Sega CD)",
}

// Month abbreviations
var monthMap = map[string]string{
	"JAN": "January",
	"FEB": "February",
	"MAR": "March",
	"APR": "April",
	"MAY": "May",
	"JUN": "June",
	"JUL": "July",
	"AUG": "August",
	"SEP": "September",
	"OCT": "October",
	"NOV": "November",
	"DEC": "December",
}

type GenesisIdentifier struct {
	db *database.GameDatabase
}

func NewGenesisIdentifier(db *database.GameDatabase) *GenesisIdentifier {
	return &GenesisIdentifier{db: db}
}

func (g *GenesisIdentifier) Console() string {
	return "Genesis"
}

func (g *GenesisIdentifier) Identify(path string) (map[string]string, error) {
	return g.IdentifyWithOptions(path, "", "", false)
}

func (g *GenesisIdentifier) IdentifyWithOptions(path, discUUID, discLabel string, preferDB bool) (map[string]string, error) {
	reader, err := fileio.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer reader.Close()

	// Read first 0x200 bytes for header detection
	data := make([]byte, 0x200)
	n, err := reader.Read(data)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read ROM data: %w", err)
	}
	if n < 0x200 {
		return nil, fmt.Errorf("ROM too small: got %d bytes, need at least 512", n)
	}

	// Find magic word
	offset := findGenesisMagicWord(data)
	if offset == -1 {
		return nil, fmt.Errorf("not a valid Genesis ROM: magic word not found")
	}

	// Parse header
	result := parseGenesisHeader(data, offset)

	// Database lookup if available
	if g.db != nil && result["ID"] != "" {
		// Clean up ID for database lookup
		serial := strings.ReplaceAll(result["ID"], "-", "")
		serial = strings.ReplaceAll(serial, " ", "_")

		if gameData, found := g.db.LookupGame("Genesis", serial); found {
			for key, value := range gameData {
				// Override existing data if preferDB is set, otherwise only add new
				_, exists := result[key]
				if preferDB || !exists {
					result[key] = value
				}
			}
		}
	}

	// Default title to overseas title if not in database
	if result["title"] == "" && result["title_overseas"] != "" {
		result["title"] = result["title_overseas"]
	}

	return result, nil
}

// findGenesisMagicWord searches for Genesis magic words in the ROM data
func findGenesisMagicWord(data []byte) int {
	for _, magic := range genesisMagicWords {
		magicBytes := []byte(magic)
		// Search from 0x100 to 0x200
		for i := 0x100; i < 0x200 && i+len(magicBytes) <= len(data); i++ {
			match := true
			for j := 0; j < len(magicBytes); j++ {
				if data[i+j] != magicBytes[j] {
					match = false
					break
				}
			}
			if match {
				return i
			}
		}
	}
	return -1
}

// parseGenesisHeader extracts information from the Genesis header
func parseGenesisHeader(data []byte, offset int) map[string]string {
	result := make(map[string]string)

	// Extract raw fields
	result["system_type"] = cleanString(data[offset+0x000 : offset+0x010])
	result["publisher"] = rawString(data[offset+0x013 : offset+0x017])
	result["release_year"] = cleanString(data[offset+0x018 : offset+0x01C])
	result["release_month"] = cleanString(data[offset+0x01D : offset+0x020])
	result["title_domestic"] = cleanString(data[offset+0x020 : offset+0x050])
	result["title_overseas"] = cleanString(data[offset+0x050 : offset+0x080])
	result["software_type"] = cleanString(data[offset+0x080 : offset+0x082])
	result["ID"] = cleanString(data[offset+0x082 : offset+0x08B])
	result["revision"] = rawString(data[offset+0x08C : offset+0x08E])

	// Checksum (big-endian)
	if offset+0x090 <= len(data) {
		checksum := binary.BigEndian.Uint16(data[offset+0x08E : offset+0x090])
		result["checksum"] = fmt.Sprintf("0x%X", checksum)
	}

	// Device support
	if offset+0x0A0 <= len(data) {
		deviceBytes := data[offset+0x090 : offset+0x0A0]
		result["device_support"] = parseDeviceSupport(deviceBytes)
	}

	// ROM/RAM ranges (big-endian)
	if offset+0x0B0 <= len(data) {
		romStart := binary.BigEndian.Uint32(data[offset+0x0A0 : offset+0x0A4])
		romEnd := binary.BigEndian.Uint32(data[offset+0x0A4 : offset+0x0A8])
		ramStart := binary.BigEndian.Uint32(data[offset+0x0A8 : offset+0x0AC])
		ramEnd := binary.BigEndian.Uint32(data[offset+0x0AC : offset+0x0B0])

		result["rom_start"] = fmt.Sprintf("0x%x", romStart)
		result["rom_end"] = fmt.Sprintf("0x%x", romEnd)
		result["ram_start"] = fmt.Sprintf("0x%x", ramStart)
		result["ram_end"] = fmt.Sprintf("0x%x", ramEnd)
	}

	// Modem support
	if offset+0x0C8 <= len(data) {
		result["modem_support"] = rawString(data[offset+0x0BC : offset+0x0C8])
	}

	// Region support
	if offset+0x0F3 <= len(data) {
		regionBytes := data[offset+0x0F0 : offset+0x0F3]
		result["region_support"] = parseRegionSupport(regionBytes)
	}

	// Process month
	if month, ok := monthMap[result["release_month"]]; ok {
		result["release_month"] = month
	}

	// Process software type
	if swType, ok := genesisSoftwareTypes[result["software_type"]]; ok {
		result["software_type"] = swType
	}

	return result
}

// parseDeviceSupport converts device support bytes to human-readable format
func parseDeviceSupport(data []byte) string {
	devices := []string{}
	seen := make(map[string]bool)

	for _, b := range data {
		if b == 0 || b == ' ' {
			continue
		}
		if device, ok := GenesisDeviceSupport[b]; ok {
			if !seen[device] {
				devices = append(devices, device)
				seen[device] = true
			}
		} else if b >= 32 && b <= 126 {
			// Add unknown printable character
			device := string(b)
			if !seen[device] {
				devices = append(devices, device)
				seen[device] = true
			}
		}
	}

	sort.Strings(devices)
	return strings.Join(devices, " / ")
}

// parseRegionSupport converts region support bytes to human-readable format
func parseRegionSupport(data []byte) string {
	regions := []string{}
	seen := make(map[string]bool)

	for _, b := range data {
		if b == 0 || b == ' ' {
			continue
		}
		if region, ok := GenesisRegionSupport[b]; ok {
			if !seen[region] {
				regions = append(regions, region)
				seen[region] = true
			}
		} else if b >= 32 && b <= 126 {
			// Add unknown printable character
			region := string(b)
			if !seen[region] {
				regions = append(regions, region)
				seen[region] = true
			}
		}
	}

	sort.Strings(regions)
	return strings.Join(regions, " / ")
}

// cleanString removes null bytes and trims whitespace
func cleanString(data []byte) string {
	// Find first null byte
	nullIndex := bytes.IndexByte(data, 0)
	if nullIndex >= 0 {
		data = data[:nullIndex]
	}

	result := string(data)
	trimmed := strings.TrimSpace(result)

	// Python main() converts empty strings to 'None', but only for actual empty strings,
	// not for strings containing null bytes
	if trimmed == "" && len(result) > 0 {
		// Return spaces if the string was only spaces
		return result
	}

	return trimmed
}

// rawString returns the string representation including null bytes to match Python
func rawString(data []byte) string {
	// Python just returns the raw bytes as-is
	return string(data)
}

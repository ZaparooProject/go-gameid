package identifier

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	bin "github.com/ZaparooProject/go-gameid/internal/binary"
)

// Genesis magic words to search for in ROM
var genesisMagicWords = [][]byte{
	[]byte("SEGA GENESIS"),
	[]byte("SEGA MEGA DRIVE"),
	[]byte("SEGA 32X"),
	[]byte("SEGA EVERDRIVE"),
	[]byte("SEGA SSF"),
	[]byte("SEGA MEGAWIFI"),
	[]byte("SEGA PICO"),
	[]byte("SEGA TERA68K"),
	[]byte("SEGA TERA286"),
}

// Genesis device support codes
var genesisDeviceSupport = map[byte]string{
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

// Genesis region support codes
var genesisRegionSupport = map[byte]string{
	'J': "Japan",
	'U': "Americas",
	'E': "Europe",
}

// Genesis software types
var genesisSoftwareTypes = map[string]string{
	"GM": "Game",
	"AI": "Aid",
	"OS": "Boot ROM (TMSS)",
	"BR": "Boot ROM (Sega CD)",
}

// GenesisIdentifier identifies Sega Genesis / Mega Drive games.
type GenesisIdentifier struct{}

// NewGenesisIdentifier creates a new Genesis identifier.
func NewGenesisIdentifier() *GenesisIdentifier {
	return &GenesisIdentifier{}
}

// Console returns the console type.
func (g *GenesisIdentifier) Console() Console {
	return ConsoleGenesis
}

// Identify extracts Genesis game information from the given reader.
func (g *GenesisIdentifier) Identify(r io.ReaderAt, size int64, db Database) (*Result, error) {
	// Read enough data to search for magic word and header
	searchSize := int64(0x200)
	if size < searchSize {
		searchSize = size
	}

	data := make([]byte, searchSize)
	if _, err := r.ReadAt(data, 0); err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read Genesis ROM: %w", err)
	}

	// Search for magic word in range 0x100-0x200
	var magicWordInd int = -1
	for _, magicWord := range genesisMagicWords {
		for i := 0x100; i <= 0x200-len(magicWord); i++ {
			if bin.BytesEqual(data[i:i+len(magicWord)], magicWord) {
				magicWordInd = i
				break
			}
		}
		if magicWordInd != -1 {
			break
		}
	}

	if magicWordInd == -1 {
		return nil, ErrInvalidFormat{Console: ConsoleGenesis, Reason: "magic word not found"}
	}

	// Need to read more data for full header
	headerEnd := magicWordInd + 0x100
	if int64(headerEnd) > size {
		headerEnd = int(size)
	}

	if headerEnd > len(data) {
		fullData := make([]byte, headerEnd)
		if _, err := r.ReadAt(fullData, 0); err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read Genesis header: %w", err)
		}
		data = fullData
	}

	// Extract fields relative to magic word position
	extractString := func(offset, length int) string {
		start := magicWordInd + offset
		end := start + length
		if end > len(data) {
			return ""
		}
		return bin.CleanString(data[start:end])
	}

	extractBytes := func(offset, length int) []byte {
		start := magicWordInd + offset
		end := start + length
		if end > len(data) {
			return nil
		}
		return data[start:end]
	}

	systemType := extractString(0x000, 0x010)
	publisher := extractString(0x013, 0x004)
	releaseYear := extractString(0x018, 0x004)
	releaseMonth := extractString(0x01D, 0x003)
	titleDomestic := extractString(0x020, 0x030)
	titleOverseas := extractString(0x050, 0x030)
	softwareType := extractString(0x080, 0x002)
	gameID := extractString(0x082, 0x009)
	revision := extractString(0x08C, 0x002)

	// Checksum is big-endian uint16
	checksumBytes := extractBytes(0x08E, 2)
	var checksum uint16
	if len(checksumBytes) == 2 {
		checksum = binary.BigEndian.Uint16(checksumBytes)
	}

	// Device support
	deviceSupportBytes := extractBytes(0x090, 0x010)
	var deviceSupport []string
	for _, b := range deviceSupportBytes {
		if b == 0 || b == ' ' {
			continue
		}
		if dev, ok := genesisDeviceSupport[b]; ok {
			deviceSupport = append(deviceSupport, dev)
		} else if b >= 0x20 && b <= 0x7E {
			deviceSupport = append(deviceSupport, string(b))
		}
	}

	// ROM/RAM addresses
	romStartBytes := extractBytes(0x0A0, 4)
	romEndBytes := extractBytes(0x0A4, 4)
	ramStartBytes := extractBytes(0x0A8, 4)
	ramEndBytes := extractBytes(0x0AC, 4)

	var romStart, romEnd, ramStart, ramEnd uint32
	if len(romStartBytes) == 4 {
		romStart = binary.BigEndian.Uint32(romStartBytes)
	}
	if len(romEndBytes) == 4 {
		romEnd = binary.BigEndian.Uint32(romEndBytes)
	}
	if len(ramStartBytes) == 4 {
		ramStart = binary.BigEndian.Uint32(ramStartBytes)
	}
	if len(ramEndBytes) == 4 {
		ramEnd = binary.BigEndian.Uint32(ramEndBytes)
	}

	// Region support
	regionSupportBytes := extractBytes(0x0F0, 0x003)
	var regionSupport []string
	for _, b := range regionSupportBytes {
		if b == 0 || b == ' ' {
			continue
		}
		if reg, ok := genesisRegionSupport[b]; ok {
			regionSupport = append(regionSupport, reg)
		} else if b >= 0x20 && b <= 0x7E {
			regionSupport = append(regionSupport, string(b))
		}
	}

	// Normalize serial for database lookup (remove dashes and spaces)
	serial := strings.ReplaceAll(gameID, "-", "")
	serial = strings.ReplaceAll(serial, " ", "")
	serial = strings.TrimSpace(serial)

	result := NewResult(ConsoleGenesis)
	result.ID = gameID
	result.InternalTitle = titleDomestic
	result.SetMetadata("system_type", systemType)
	result.SetMetadata("publisher", publisher)
	result.SetMetadata("release_year", releaseYear)
	result.SetMetadata("release_month", releaseMonth)
	result.SetMetadata("title_domestic", titleDomestic)
	result.SetMetadata("title_overseas", titleOverseas)
	result.SetMetadata("ID", gameID)
	result.SetMetadata("revision", revision)
	result.SetMetadata("checksum", fmt.Sprintf("0x%04x", checksum))
	result.SetMetadata("rom_start", fmt.Sprintf("0x%08x", romStart))
	result.SetMetadata("rom_end", fmt.Sprintf("0x%08x", romEnd))
	result.SetMetadata("ram_start", fmt.Sprintf("0x%08x", ramStart))
	result.SetMetadata("ram_end", fmt.Sprintf("0x%08x", ramEnd))

	if softwareType != "" {
		if st, ok := genesisSoftwareTypes[softwareType]; ok {
			result.SetMetadata("software_type", st)
		} else {
			result.SetMetadata("software_type", softwareType)
		}
	}

	if len(deviceSupport) > 0 {
		result.SetMetadata("device_support", strings.Join(deviceSupport, " / "))
	}

	if len(regionSupport) > 0 {
		result.SetMetadata("region_support", strings.Join(regionSupport, " / "))
	}

	// Database lookup
	if db != nil && serial != "" {
		if entry, found := db.LookupByString(ConsoleGenesis, serial); found {
			result.MergeMetadata(entry, false)
		}
	}

	// If no title from database, use domestic title
	if result.Title == "" {
		if titleOverseas != "" {
			result.Title = titleOverseas
		} else {
			result.Title = titleDomestic
		}
	}

	return result, nil
}

// ValidateGenesis checks if the given data looks like a valid Genesis ROM.
func ValidateGenesis(data []byte) bool {
	if len(data) < 0x200 {
		return false
	}

	for _, magicWord := range genesisMagicWords {
		for i := 0x100; i <= 0x200-len(magicWord); i++ {
			if bin.BytesEqual(data[i:i+len(magicWord)], magicWord) {
				return true
			}
		}
	}

	return false
}

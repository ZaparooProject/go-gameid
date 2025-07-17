package identifiers

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/wizzomafizzo/go-gameid/pkg/binary"
	"github.com/wizzomafizzo/go-gameid/pkg/database"
	"github.com/wizzomafizzo/go-gameid/pkg/fileio"
)

// Identifier interface for game identification
type Identifier interface {
	// Identify identifies a game file and returns its metadata
	Identify(path string) (map[string]string, error)

	// Console returns the console/system name
	Console() string
}

// IdentifierWithOptions extends Identifier with additional options
type IdentifierWithOptions interface {
	Identifier
	// IdentifyWithOptions identifies a game with additional parameters
	IdentifyWithOptions(path, discUUID, discLabel string, preferDB bool) (map[string]string, error)
}

// gbaLogo is the Nintendo logo that must be present in GBA ROMs
var gbaLogo = []byte{
	0x24, 0xFF, 0xAE, 0x51, 0x69, 0x9A, 0xA2, 0x21, 0x3D, 0x84, 0x82, 0x0A,
	0x84, 0xE4, 0x09, 0xAD, 0x11, 0x24, 0x8B, 0x98, 0xC0, 0x81, 0x7F, 0x21,
	0xA3, 0x52, 0xBE, 0x19, 0x93, 0x09, 0xCE, 0x20, 0x10, 0x46, 0x4A, 0x4A,
	0xF8, 0x27, 0x31, 0xEC, 0x58, 0xC7, 0xE8, 0x33, 0x82, 0xE3, 0xCE, 0xBF,
	0x85, 0xF4, 0xDF, 0x94, 0xCE, 0x4B, 0x09, 0xC1, 0x94, 0x56, 0x8A, 0xC0,
	0x13, 0x72, 0xA7, 0xFC, 0x9F, 0x84, 0x4D, 0x73, 0xA3, 0xCA, 0x9A, 0x61,
	0x58, 0x97, 0xA3, 0x27, 0xFC, 0x03, 0x98, 0x76, 0x23, 0x1D, 0xC7, 0x61,
	0x03, 0x04, 0xAE, 0x56, 0xBF, 0x38, 0x84, 0x00, 0x40, 0xA7, 0x0E, 0xFD,
	0xFF, 0x52, 0xFE, 0x03, 0x6F, 0x95, 0x30, 0xF1, 0x97, 0xFB, 0xC0, 0x85,
	0x60, 0xD6, 0x80, 0x25, 0xA9, 0x63, 0xBE, 0x03, 0x01, 0x4E, 0x38, 0xE2,
	0xF9, 0xA2, 0x34, 0xFF, 0xBB, 0x3E, 0x03, 0x44, 0x78, 0x00, 0x90, 0xCB,
	0x88, 0x11, 0x3A, 0x94, 0x65, 0xC0, 0x7C, 0x63, 0x87, 0xF0, 0x3C, 0xAF,
	0xD6, 0x25, 0xE4, 0x8B, 0x38, 0x0A, 0xAC, 0x72, 0x21, 0xD4, 0xF8, 0x07,
}

// GBAIdentifier implements game identification for Game Boy Advance
type GBAIdentifier struct {
	db *database.GameDatabase
}

// gbLogo is the Nintendo logo that must be present in GB/GBC ROMs
var gbLogo = []byte{
	0xCE, 0xED, 0x66, 0x66, 0xCC, 0x0D, 0x00, 0x0B, 0x03, 0x73, 0x00, 0x83,
	0x00, 0x0C, 0x00, 0x0D, 0x00, 0x08, 0x11, 0x1F, 0x88, 0x89, 0x00, 0x0E,
	0xDC, 0xCC, 0x6E, 0xE6, 0xDD, 0xDD, 0xD9, 0x99, 0xBB, 0xBB, 0x67, 0x63,
	0x6E, 0x0E, 0xEC, 0xCC, 0xDD, 0xDC, 0x99, 0x9F, 0xBB, 0xB9, 0x33, 0x3E,
}

// GBIdentifier implements game identification for Game Boy and Game Boy Color
type GBIdentifier struct {
	db *database.GameDatabase
}

// NewGBIdentifier creates a new GB identifier
func NewGBIdentifier(db *database.GameDatabase) *GBIdentifier {
	return &GBIdentifier{db: db}
}

// Console returns the console name
func (g *GBIdentifier) Console() string {
	return "GB"
}

// Identify identifies a GB/GBC game and returns its metadata
func (g *GBIdentifier) Identify(path string) (map[string]string, error) {
	// Check file exists
	if err := fileio.CheckExists(path); err != nil {
		return nil, err
	}

	// Open file
	file, err := fileio.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open GB file: %w", err)
	}
	defer file.Close()

	// Read header (minimum 0x150 bytes for GB header)
	data, err := fileio.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read GB file: %w", err)
	}

	if len(data) < 0x150 {
		return nil, fmt.Errorf("invalid GB ROM: file too small (got %d bytes, need at least %d)", len(data), 0x150)
	}

	// Validate Nintendo logo (optional - some ROMs might have modified logos)
	logoData := data[0x104:0x134]
	if !bytes.Equal(logoData, gbLogo) {
		// Log warning but don't fail
	}

	// Parse header fields
	internalTitle := binary.CleanString(data[0x134:0x144])

	// CGB flag (0x143)
	cgbFlag := data[0x143]
	var cgbMode string
	if cgbFlag == 0x80 {
		cgbMode = "GBC"
	} else if cgbFlag == 0xC0 {
		cgbMode = "GBC"
	} else {
		cgbMode = "GB"
	}

	// SGB flag (0x146)
	sgbSupport := data[0x146] == 0x03

	// Cartridge type (0x147)
	cartridgeType := getCartridgeType(data[0x147])

	// ROM size (0x148)
	romSize, romBanks := getROMInfo(data[0x148])

	// RAM size (0x149)
	ramSize, ramBanks := getRAMInfo(data[0x149])

	// Licensee (0x14B for old, 0x144-0x145 for new)
	licensee := getLicensee(data[0x14B], data[0x144:0x146])

	// ROM version (0x14C)
	romVersion := data[0x14C]

	// Header checksum (0x14D)
	headerChecksumExpected := data[0x14D]
	headerChecksumActual := calculateHeaderChecksum(data[0x134:0x14D])

	// Global checksum (0x14E-0x14F)
	globalChecksumExpected := uint16(data[0x14E])<<8 | uint16(data[0x14F])
	globalChecksumActual := calculateGlobalChecksum(data)

	// Build result matching Python output format
	result := map[string]string{
		"internal_title": internalTitle,
		"cgb_mode":       cgbMode,
		"sgb_support": func() string {
			if sgbSupport {
				return "True"
			}
			return "False"
		}(),
		"cartridge_type":           cartridgeType,
		"rom_size":                 fmt.Sprintf("%d", romSize),
		"rom_banks":                fmt.Sprintf("%d", romBanks),
		"ram_size":                 fmt.Sprintf("%d", ramSize),
		"ram_banks":                fmt.Sprintf("%d", ramBanks),
		"licensee":                 licensee,
		"rom_version":              fmt.Sprintf("%d", romVersion),
		"header_checksum_expected": fmt.Sprintf("0x%02x", headerChecksumExpected),
		"header_checksum_actual":   fmt.Sprintf("0x%02x", headerChecksumActual),
		"global_checksum_expected": fmt.Sprintf("0x%04x", globalChecksumExpected),
		"global_checksum_actual":   fmt.Sprintf("0x%04x", globalChecksumActual),
	}

	// Look up in database for title
	if g.db != nil {
		// Try GB system first
		if gameData, found := g.db.LookupGame("GB", internalTitle); found {
			for key, value := range gameData {
				if key == "title" {
					result[key] = value
				}
			}
		}
	}

	// If no title found in database, use internal title as-is (Python compatibility)
	if result["title"] == "" {
		result["title"] = internalTitle
	}

	return result, nil
}

// Helper functions for GB parsing
func getCartridgeType(typeCode byte) string {
	cartridgeTypes := map[byte]string{
		0x00: "ROM",
		0x01: "MBC1",
		0x02: "MBC1+RAM",
		0x03: "MBC1+RAM+BATTERY",
		0x05: "MBC2",
		0x06: "MBC2+BATTERY",
		0x08: "ROM+RAM",
		0x09: "ROM+RAM+BATTERY",
		0x0B: "MMM01",
		0x0C: "MMM01+RAM",
		0x0D: "MMM01+RAM+BATTERY",
		0x0F: "MBC3+TIMER+BATTERY",
		0x10: "MBC3+TIMER+RAM+BATTERY",
		0x11: "MBC3",
		0x12: "MBC3+RAM",
		0x13: "MBC3+RAM+BATTERY",
		0x19: "MBC5",
		0x1A: "MBC5+RAM",
		0x1B: "MBC5+RAM+BATTERY",
		0x1C: "MBC5+RUMBLE",
		0x1D: "MBC5+RUMBLE+RAM",
		0x1E: "MBC5+RUMBLE+RAM+BATTERY",
		0x20: "MBC6",
		0x22: "MBC7+SENSOR+RUMBLE+RAM+BATTERY",
		0xFC: "POCKET CAMERA",
		0xFD: "BANDAI TAMA5",
		0xFE: "HuC3",
		0xFF: "HuC1+RAM+BATTERY",
	}

	if name, exists := cartridgeTypes[typeCode]; exists {
		return name
	}
	return fmt.Sprintf("UNKNOWN(0x%02x)", typeCode)
}

func getROMInfo(sizeCode byte) (int, int) {
	switch sizeCode {
	case 0x00:
		return 32768, 2 // 32KB, 2 banks
	case 0x01:
		return 65536, 4 // 64KB, 4 banks
	case 0x02:
		return 131072, 8 // 128KB, 8 banks
	case 0x03:
		return 262144, 16 // 256KB, 16 banks
	case 0x04:
		return 524288, 32 // 512KB, 32 banks
	case 0x05:
		return 1048576, 64 // 1MB, 64 banks
	case 0x06:
		return 2097152, 128 // 2MB, 128 banks
	case 0x07:
		return 4194304, 256 // 4MB, 256 banks
	case 0x08:
		return 8388608, 512 // 8MB, 512 banks
	case 0x52:
		return 1179648, 72 // 1.1MB, 72 banks
	case 0x53:
		return 1310720, 80 // 1.2MB, 80 banks
	case 0x54:
		return 1572864, 96 // 1.5MB, 96 banks
	default:
		return 0, 0
	}
}

func getRAMInfo(sizeCode byte) (int, int) {
	switch sizeCode {
	case 0x00:
		return 0, 0 // No RAM
	case 0x01:
		return 2048, 1 // 2KB, unused
	case 0x02:
		return 8192, 1 // 8KB, 1 bank
	case 0x03:
		return 32768, 4 // 32KB, 4 banks
	case 0x04:
		return 131072, 16 // 128KB, 16 banks
	case 0x05:
		return 65536, 8 // 64KB, 8 banks
	default:
		return 0, 0
	}
}

func getLicensee(oldCode byte, newCode []byte) string {
	// Use new licensee code if old code is 0x33
	if oldCode == 0x33 && len(newCode) >= 2 {
		code := string(newCode)
		licensees := map[string]string{
			"01": "Nintendo R&D1",
			"08": "Capcom",
			"09": "Hot-B",
			"0A": "Jaleco",
			"0B": "Coconuts Japan",
			"0C": "Elite Systems",
			"13": "EA",
			"18": "Hudson Soft",
			"19": "ITC Entertainment",
			"1A": "Yanoman",
			"1D": "Clary",
			"1F": "Virgin Interactive",
			"24": "PCM Complete",
			"25": "San-X",
			"28": "Kemco",
			"29": "SETA Corporation",
			"30": "Infogrames",
			"31": "Nintendo",
			"32": "Bandai",
			"33": "Ocean Interactive",
			"34": "Konami",
			"35": "HectorSoft",
			"38": "Capcom",
			"39": "Banpresto",
			"3C": "Entertainment i",
			"3E": "Gremlin",
			"41": "Ubi Soft",
			"42": "Atlus",
			"44": "Malibu Interactive",
			"46": "Angel",
			"47": "Spectrum Holobyte",
			"49": "Irem",
			"4A": "Virgin Interactive",
			"4D": "Malibu Interactive",
			"4F": "U.S. Gold",
			"50": "Absolute",
			"51": "Acclaim Entertainment",
			"52": "Activision",
			"53": "American Sammy",
			"54": "GameTek",
			"55": "Hi Tech Expressions",
			"56": "LJN",
			"57": "Matchbox",
			"59": "Milton Bradley",
			"5A": "Mindscape",
			"5B": "Romstar",
			"5C": "Naxat Soft",
			"5D": "Tradewest",
			"60": "Titus Interactive",
			"61": "Virgin Interactive",
			"67": "Ocean Interactive",
			"69": "EA",
			"6E": "Elite Systems",
			"6F": "Electro Brain",
			"70": "Infogrames",
			"71": "Interplay",
			"72": "Broderbund",
			"73": "Sculptered Soft",
			"75": "The Sales Curve",
			"78": "t.hq",
			"79": "Accolade",
			"7A": "Triffix Entertainment",
			"7C": "Microprose",
			"7F": "Kemco",
			"80": "Misawa Entertainment",
			"83": "Lozc",
			"86": "Tokuma Shoten",
			"8B": "Bullet-Proof Software",
			"8C": "Vic Tokai",
			"8E": "Ape",
			"8F": "I'Max",
			"91": "Chunsoft Co.",
			"92": "Video System",
			"93": "Tsubaraya Productions",
			"95": "Varie",
			"96": "Yonezawa/S'Pal",
			"97": "Kaneko",
			"99": "Arc",
			"9A": "Nihon Bussan",
			"9B": "Tecmo",
			"9C": "Imagineer",
			"9D": "Banpresto",
			"9F": "Nova",
			"A1": "Hori Electric",
			"A2": "Bandai",
			"A4": "Konami",
			"A6": "Kawada",
			"A7": "Takara",
			"A9": "Technos Japan",
			"AA": "Broderbund",
			"AC": "Toei Animation",
			"AD": "Toho",
			"AF": "Namco",
			"B0": "Acclaim Entertainment",
			"B1": "ASCII or Nexoft",
			"B2": "Bandai",
			"B4": "Square Enix",
			"B6": "HAL Laboratory",
			"B7": "SNK",
			"B9": "Pony Canyon",
			"BA": "Culture Brain",
			"BB": "Sunsoft",
			"BD": "Sony Imagesoft",
			"BF": "Sammy",
			"C0": "Taito",
			"C2": "Kemco",
			"C3": "Squaresoft",
			"C4": "Tokuma Shoten",
			"C5": "Data East",
			"C6": "Tonkinhouse",
			"C8": "Koei",
			"C9": "UFL",
			"CA": "Ultra",
			"CB": "Vap",
			"CC": "Use Corporation",
			"CD": "Meldac",
			"CE": "Pony Canyon",
			"CF": "Angel",
			"D0": "Taito",
			"D1": "Sofel",
			"D2": "Quest",
			"D3": "Sigma Enterprises",
			"D4": "ASK Kodansha Co.",
			"D6": "Naxat Soft",
			"D7": "Copya System",
			"D9": "Banpresto",
			"DA": "Tomy",
			"DB": "LJN",
			"DD": "NCS",
			"DE": "Human",
			"DF": "Altron",
			"E0": "Jaleco",
			"E1": "Towa Chiki",
			"E2": "Yutaka",
			"E3": "Varie",
			"E5": "Epoch",
			"E7": "Athena",
			"E8": "Asmik Ace Entertainment",
			"E9": "Natsume",
			"EA": "King Records",
			"EB": "Atlus",
			"EC": "Epic/Sony Records",
			"EE": "IGS",
			"F0": "A Wave",
			"F3": "Extreme Entertainment",
			"FF": "LJN",
		}

		if name, exists := licensees[code]; exists {
			return name
		}
		return fmt.Sprintf("Unknown (%s)", code)
	}

	// Old licensee codes
	oldLicensees := map[byte]string{
		0x00: "None",
		0x01: "Nintendo R&D1",
		0x08: "Capcom",
		0x09: "Hot-B",
		0x0A: "Jaleco",
		0x0B: "Coconuts Japan",
		0x0C: "Elite Systems",
		0x13: "EA",
		0x18: "Hudson Soft",
		0x19: "ITC Entertainment",
		0x1A: "Yanoman",
		0x1D: "Clary",
		0x1F: "Virgin Interactive",
		0x24: "PCM Complete",
		0x25: "San-X",
		0x28: "Kemco",
		0x29: "SETA Corporation",
		0x30: "Infogrames",
		0x31: "Nintendo",
		0x32: "Bandai",
		0x34: "Konami",
		0x35: "HectorSoft",
		0x38: "Capcom",
		0x39: "Banpresto",
		0x3C: "Entertainment i",
		0x3E: "Gremlin",
		0x41: "Ubi Soft",
		0x42: "Atlus",
		0x44: "Malibu Interactive",
		0x46: "Angel",
		0x47: "Spectrum Holobyte",
		0x49: "Irem",
		0x4A: "Virgin Interactive",
		0x4D: "Malibu Interactive",
		0x4F: "U.S. Gold",
		0x50: "Absolute",
		0x51: "Acclaim Entertainment",
		0x52: "Activision",
		0x53: "American Sammy",
		0x54: "GameTek",
		0x55: "Hi Tech Expressions",
		0x56: "LJN",
		0x57: "Matchbox",
		0x59: "Milton Bradley",
		0x5A: "Mindscape",
		0x5B: "Romstar",
		0x5C: "Naxat Soft",
		0x5D: "Tradewest",
		0x60: "Titus Interactive",
		0x61: "Virgin Interactive",
		0x67: "Ocean Interactive",
		0x69: "EA",
		0x6E: "Elite Systems",
		0x6F: "Electro Brain",
		0x70: "Infogrames",
		0x71: "Interplay",
		0x72: "Broderbund",
		0x73: "Sculptered Soft",
		0x75: "The Sales Curve",
		0x78: "t.hq",
		0x79: "Accolade",
		0x7A: "Triffix Entertainment",
		0x7C: "Microprose",
		0x7F: "Kemco",
		0x80: "Misawa Entertainment",
		0x83: "Lozc",
		0x86: "Tokuma Shoten",
		0x8B: "Bullet-Proof Software",
		0x8C: "Vic Tokai",
		0x8E: "Ape",
		0x8F: "I'Max",
		0x91: "Chunsoft Co.",
		0x92: "Video System",
		0x93: "Tsubaraya Productions",
		0x95: "Varie",
		0x96: "Yonezawa/S'Pal",
		0x97: "Kaneko",
		0x99: "Arc",
		0x9A: "Nihon Bussan",
		0x9B: "Tecmo",
		0x9C: "Imagineer",
		0x9D: "Banpresto",
		0x9F: "Nova",
		0xA1: "Hori Electric",
		0xA2: "Bandai",
		0xA4: "Konami",
		0xA6: "Kawada",
		0xA7: "Takara",
		0xA9: "Technos Japan",
		0xAA: "Broderbund",
		0xAC: "Toei Animation",
		0xAD: "Toho",
		0xAF: "Namco",
		0xB0: "Acclaim Entertainment",
		0xB1: "ASCII or Nexoft",
		0xB2: "Bandai",
		0xB4: "Square Enix",
		0xB6: "HAL Laboratory",
		0xB7: "SNK",
		0xB9: "Pony Canyon",
		0xBA: "Culture Brain",
		0xBB: "Sunsoft",
		0xBD: "Sony Imagesoft",
		0xBF: "Sammy",
		0xC0: "Taito",
		0xC2: "Kemco",
		0xC3: "Squaresoft",
		0xC4: "Tokuma Shoten",
		0xC5: "Data East",
		0xC6: "Tonkinhouse",
		0xC8: "Koei",
		0xC9: "UFL",
		0xCA: "Ultra",
		0xCB: "Vap",
		0xCC: "Use Corporation",
		0xCD: "Meldac",
		0xCE: "Pony Canyon",
		0xCF: "Angel",
		0xD0: "Taito",
		0xD1: "Sofel",
		0xD2: "Quest",
		0xD3: "Sigma Enterprises",
		0xD4: "ASK Kodansha Co.",
		0xD6: "Naxat Soft",
		0xD7: "Copya System",
		0xD9: "Banpresto",
		0xDA: "Tomy",
		0xDB: "LJN",
		0xDD: "NCS",
		0xDE: "Human",
		0xDF: "Altron",
		0xE0: "Jaleco",
		0xE1: "Towa Chiki",
		0xE2: "Yutaka",
		0xE3: "Varie",
		0xE5: "Epoch",
		0xE7: "Athena",
		0xE8: "Asmik Ace Entertainment",
		0xE9: "Natsume",
		0xEA: "King Records",
		0xEB: "Atlus",
		0xEC: "Epic/Sony Records",
		0xEE: "IGS",
		0xF0: "A Wave",
		0xF3: "Extreme Entertainment",
		0xFF: "LJN",
	}

	if name, exists := oldLicensees[oldCode]; exists {
		return name
	}
	return fmt.Sprintf("Unknown (0x%02x)", oldCode)
}

func calculateHeaderChecksum(header []byte) byte {
	var checksum byte
	for _, b := range header {
		checksum = checksum - b - 1
	}
	return checksum
}

func calculateGlobalChecksum(data []byte) uint16 {
	var checksum uint16
	for i, b := range data {
		// Skip the global checksum bytes themselves (0x14E-0x14F)
		if i != 0x14E && i != 0x14F {
			checksum += uint16(b)
		}
	}
	return checksum
}

func cleanGBTitle(title string) string {
	// Handle specific known games first
	switch title {
	case "FUNPAK 4IN1 - V2":
		return "4-in-1 Fun Pak Volume II"
	case "ADDAMS FAMILY 2":
		return "Addams Family, The: Pugsley's Scavenger Hunt"
	case "AMAZING-TATER":
		return "A-mazing Tater"
	case "ADDAMS FAMILY":
		return "Addams Family, The"
	case "4 IN 1 FUN PAK":
		return "4-in-1 Fun Pak"
	}

	// Basic title cleaning - remove underscores, fix common patterns
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")

	// Convert to title case
	words := strings.Fields(title)
	for i, word := range words {
		words[i] = strings.Title(strings.ToLower(word))
	}
	title = strings.Join(words, " ")

	// Fix common patterns
	title = strings.ReplaceAll(title, " Gb", " GB")
	title = strings.ReplaceAll(title, " Gbc", " GBC")
	title = strings.ReplaceAll(title, "4 In 1", "4-in-1")

	return title
}

// n64FirstWord is the magic number that identifies N64 ROMs
var n64FirstWord = []byte{0x80, 0x37, 0x12, 0x40}

// N64Identifier implements game identification for Nintendo 64
type N64Identifier struct {
	db *database.GameDatabase
}

// NewN64Identifier creates a new N64 identifier
func NewN64Identifier(db *database.GameDatabase) *N64Identifier {
	return &N64Identifier{db: db}
}

// Console returns the console name
func (n *N64Identifier) Console() string {
	return "N64"
}

// Identify identifies an N64 game and returns its metadata
func (n *N64Identifier) Identify(path string) (map[string]string, error) {
	// Check file exists
	if err := fileio.CheckExists(path); err != nil {
		return nil, err
	}

	// Open file
	file, err := fileio.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open N64 file: %w", err)
	}
	defer file.Close()

	// Read header (64 bytes - stop before boot code)
	data, err := fileio.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read N64 file: %w", err)
	}

	if len(data) < 0x40 {
		return nil, fmt.Errorf("invalid N64 ROM: file too small (got %d bytes, need at least %d)", len(data), 0x40)
	}

	header := data[:0x40]

	// Determine endianness from first word
	firstWord := header[0:4]
	if bytes.Equal(binary.N64EndianSwap(firstWord), n64FirstWord) {
		// Little-endian, need to convert to big-endian
		header = binary.N64EndianSwap(header)
	} else if !bytes.Equal(firstWord, n64FirstWord) {
		// Doesn't match either endianness
		return nil, fmt.Errorf("invalid N64 ROM: invalid magic number")
	}

	// Parse N64 ROM header
	cartridgeID := header[0x3C:0x3E]
	countryCode := header[0x3E]

	// Build serial number
	serial := fmt.Sprintf("%c%c%c", cartridgeID[0], cartridgeID[1], countryCode)

	// Build result - always return basic metadata
	result := make(map[string]string)
	result["ID"] = serial

	// Extract internal title from ROM
	internalTitle := strings.TrimSpace(binary.CleanString(header[0x20:0x34]))
	if internalTitle != "" {
		result["internal_title"] = internalTitle
		result["title"] = internalTitle // Use internal title as default title
	}

	// Look up in database to enhance with additional metadata
	if n.db != nil {
		if gameData, found := n.db.LookupGame("N64", serial); found {
			// Copy all database fields
			for key, value := range gameData {
				result[key] = value
			}
			// Always keep the serial as ID
			result["ID"] = serial
		}
	}

	return result, nil
}

// SNESIdentifier implements game identification for Super Nintendo
type SNESIdentifier struct {
	db *database.GameDatabase
}

// NewSNESIdentifier creates a new SNES identifier
func NewSNESIdentifier(db *database.GameDatabase) *SNESIdentifier {
	return &SNESIdentifier{db: db}
}

// Console returns the console name
func (s *SNESIdentifier) Console() string {
	return "SNES"
}

// Identify identifies a SNES game and returns its metadata
func (s *SNESIdentifier) Identify(path string) (map[string]string, error) {
	// Check file exists
	if err := fileio.CheckExists(path); err != nil {
		return nil, err
	}

	// Open file
	file, err := fileio.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open SNES file: %w", err)
	}
	defer file.Close()

	// Read entire ROM
	data, err := fileio.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read SNES file: %w", err)
	}

	// Remove optional 512-byte copier header if present
	if (len(data) % 1024) == 512 {
		data = data[512:]
	}

	// Try to find header (LoROM at 0x7FC0 or HiROM at 0xFFC0)
	var headerStart int
	var romType string

	// Check if we have enough data for headers
	if len(data) < 0x8000 {
		return nil, fmt.Errorf("invalid SNES ROM: file too small (got %d bytes after header removal)", len(data))
	}

	// Try to detect header location by validating checksum
	checksumValid := false
	for _, offset := range []int{0x7FC0, 0xFFC0} {
		if offset+32 > len(data) {
			continue
		}

		// Read checksum and complement
		checksum := uint16(data[offset+0x1F])<<8 | uint16(data[offset+0x1E])
		complement := uint16(data[offset+0x1D])<<8 | uint16(data[offset+0x1C])

		// Valid if checksum + complement = 0xFFFF
		if checksum+complement == 0xFFFF {
			headerStart = offset
			checksumValid = true
			break
		}
	}

	// If checksum validation failed, try to detect by other means
	if !checksumValid {
		// Default to LoROM if we have enough data
		if len(data) >= 0x8000 {
			headerStart = 0x7FC0
		} else {
			return nil, fmt.Errorf("invalid SNES ROM: file too small")
		}
	}

	// Parse header
	header := data[headerStart:]

	// Internal title (21 bytes)
	internalTitle := header[0:21]
	internalTitleHex := "0x"
	for _, b := range internalTitle {
		internalTitleHex += fmt.Sprintf("%02x", b)
	}

	// For database lookup, trim trailing spaces from internal title
	trimmedTitle := bytes.TrimRight(internalTitle, " \x00")
	trimmedTitleHex := "0x"
	for _, b := range trimmedTitle {
		trimmedTitleHex += fmt.Sprintf("%02x", b)
	}

	// ROM makeup byte
	romMakeup := header[0x15]

	// Fast/Slow ROM
	fastSlowROM := "SlowROM"
	if (romMakeup & 0x10) != 0 {
		fastSlowROM = "FastROM"
	}

	// ROM type based on makeup byte
	if (romMakeup & 0x01) == 0 {
		romType = "LoROM"
	} else {
		romType = "HiROM"
	}
	if (romMakeup & 0x04) != 0 {
		romType = "Ex" + romType
	}

	// Cartridge type (hardware)
	cartridgeType := header[0x16]
	hardware := getHardwareType(cartridgeType, data, headerStart)

	// ROM size
	// romSize := header[0x17]

	// RAM size
	// ramSize := header[0x18]

	// Country code
	// countryCode := header[0x19]

	// Developer ID
	developerID := header[0x1A]

	// Version
	romVersion := header[0x1B]

	// Checksum complement
	// checksumComplement := uint16(header[0x1D])<<8 | uint16(header[0x1C])

	// Checksum
	checksum := uint16(header[0x1F])<<8 | uint16(header[0x1E])

	// Build result
	result := map[string]string{
		"internal_title": internalTitleHex,
		"fast_slow_rom":  fastSlowROM,
		"rom_type":       romType,
		"developer_ID":   fmt.Sprintf("0x%02x", developerID),
		"rom_version":    fmt.Sprintf("%d", romVersion),
		"checksum":       fmt.Sprintf("0x%04x", checksum),
		"hardware":       hardware,
	}

	// Generate gamedb ID using full title for database lookup (matches Python version)
	gamedbID := fmt.Sprintf("%d,%s,%d,%d", developerID, internalTitleHex, romVersion, checksum)

	// Look up in database
	if s.db != nil {
		if gameData, found := s.db.LookupGame("SNES", gamedbID); found {

			// Merge database metadata
			for key, value := range gameData {
				if key != "internal_title" { // Keep our parsed internal title
					result[key] = value
				}
			}
		}
	}

	// If no title found in database, try to clean internal title
	if result["title"] == "" {
		// Convert internal title bytes to string, replacing non-printable with spaces
		titleStr := ""
		for _, b := range internalTitle {
			if b >= 0x20 && b <= 0x7E {
				titleStr += string(b)
			} else {
				titleStr += " "
			}
		}
		result["title"] = strings.TrimSpace(titleStr)
	}

	return result, nil
}

// getHardwareType returns the hardware type string based on cartridge type byte
func getHardwareType(cartridgeType byte, data []byte, headerStart int) string {
	if cartridgeType <= 2 {
		return []string{"ROM", "ROM + RAM", "ROM + RAM + Battery"}[cartridgeType]
	}

	// Check for coprocessor
	hardware := ""
	lastDigit := cartridgeType & 0x0F
	if lastDigit >= 3 && lastDigit <= 6 {
		hardware = []string{"ROM + Coprocessor", "ROM + Coprocessor + RAM", "ROM + Coprocessor + RAM + Battery", "ROM + Coprocessor + Battery"}[lastDigit-3]
	}

	// Determine coprocessor type
	upperDigit := (cartridgeType >> 4) & 0x0F
	// coprocessor := ""

	switch upperDigit {
	case 0:
		// coprocessor = "DSP"
	case 1:
		// coprocessor = "GSU / SuperFX"
	case 2:
		// coprocessor = "OBC1"
	case 3:
		// coprocessor = "SA-1"
	case 4:
		// coprocessor = "S-DD1"
	case 5:
		// coprocessor = "S-RTC"
	case 0xE:
		// coprocessor = "Super Game Boy / Satellaview"
	case 0xF:
		// Check $FFBF equivalent position
		if headerStart >= 1 && headerStart-1 < len(data) {
			ffbf := data[headerStart-1]
			if (ffbf>>4) == 0 && (ffbf&0x0F) <= 3 {
				// coprocessor = []string{"SPC7110", "ST010 / ST011", "ST018", "CX4"}[ffbf&0x0F]
			}
		}
	}

	if hardware == "" {
		// Fallback for unknown hardware types
		hardware = "ROM"
	}

	return hardware
}

// NewGBAIdentifier creates a new GBA identifier
func NewGBAIdentifier(db *database.GameDatabase) *GBAIdentifier {
	return &GBAIdentifier{db: db}
}

// Console returns the console name
func (g *GBAIdentifier) Console() string {
	return "GBA"
}

// Identify identifies a GBA game and returns its metadata
func (g *GBAIdentifier) Identify(path string) (map[string]string, error) {
	// Check file exists
	if err := fileio.CheckExists(path); err != nil {
		return nil, err
	}

	// Open file
	file, err := fileio.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open GBA file: %w", err)
	}
	defer file.Close()

	// Read header (192 bytes minimum)
	header := make([]byte, 192)
	n, err := fileio.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read GBA file: %w", err)
	}

	if len(n) < 192 {
		return nil, fmt.Errorf("invalid GBA ROM: file too small (got %d bytes, need at least 192)", len(n))
	}

	copy(header, n[:192])

	// Validate Nintendo logo (optional validation - Python script doesn't fail on this)
	logoData := header[0x04:0xA0]
	if !bytes.Equal(logoData, gbaLogo) {
		// Log warning but don't fail - some ROMs might have modified logos
		// In production, this could be logged
	}

	// Parse header fields
	internalTitle := binary.CleanString(header[0xA0:0xAC])
	gameCode := binary.CleanString(header[0xAC:0xB0])
	makerCode := binary.CleanString(header[0xB0:0xB2])
	mainUnitCode := header[0xB3]
	deviceType := header[0xB4]
	softwareVersion := header[0xBC]

	// Build result
	result := map[string]string{
		"ID":               gameCode,
		"internal_title":   internalTitle,
		"maker_code":       makerCode,
		"main_unit_code":   fmt.Sprintf("0x%02x", mainUnitCode),
		"device_type":      fmt.Sprintf("0x%02x", deviceType),
		"software_version": fmt.Sprintf("%d", softwareVersion),
	}

	// Look up in database
	if g.db != nil {
		if gameData, found := g.db.LookupGame("GBA", gameCode); found {
			// Merge database metadata, giving preference to database unless prefer_gamedb is false
			for key, value := range gameData {
				if key != "ID" { // Always preserve the ID from the ROM
					result[key] = value
				}
			}
		}
	}

	// If no title found in database, use internal title
	if result["title"] == "" {
		result["title"] = internalTitle
	}

	return result, nil
}

package identifier

import (
	"fmt"
	"io"

	"github.com/ZaparooProject/go-gameid/internal/binary"
)

// GB/GBC header offsets
const (
	gbHeaderSize           = 0x0150
	gbNintendoLogoOffset   = 0x0104
	gbNintendoLogoSize     = 48
	gbTitleOffset          = 0x0134
	gbTitleSize            = 11 // or 16 if no manufacturer code
	gbManufacturerOffset   = 0x013F
	gbManufacturerSize     = 4
	gbCGBFlagOffset        = 0x0143
	gbNewLicenseeOffset    = 0x0144
	gbNewLicenseeSize      = 2
	gbSGBFlagOffset        = 0x0146
	gbCartridgeTypeOffset  = 0x0147
	gbROMSizeOffset        = 0x0148
	gbRAMSizeOffset        = 0x0149
	gbDestinationOffset    = 0x014A
	gbOldLicenseeOffset    = 0x014B
	gbROMVersionOffset     = 0x014C
	gbHeaderChecksumOffset = 0x014D
	gbGlobalChecksumOffset = 0x014E
)

// GB Nintendo logo - used to validate GB/GBC ROMs
var gbNintendoLogo = []byte{
	0xCE, 0xED, 0x66, 0x66, 0xCC, 0x0D, 0x00, 0x0B,
	0x03, 0x73, 0x00, 0x83, 0x00, 0x0C, 0x00, 0x0D,
	0x00, 0x08, 0x11, 0x1F, 0x88, 0x89, 0x00, 0x0E,
	0xDC, 0xCC, 0x6E, 0xE6, 0xDD, 0xDD, 0xD9, 0x99,
	0xBB, 0xBB, 0x67, 0x63, 0x6E, 0x0E, 0xEC, 0xCC,
	0xDD, 0xDC, 0x99, 0x9F, 0xBB, 0xB9, 0x33, 0x3E,
}

// GB cartridge types lookup table
var gbCartridgeTypes = map[byte]string{
	0x00: "ROM",
	0x01: "MBC1",
	0x02: "MBC1 + RAM",
	0x03: "MBC1 + RAM + Battery",
	0x05: "MBC2",
	0x06: "MBC2 + Battery",
	0x08: "ROM + RAM",
	0x09: "ROM + RAM + Battery",
	0x0B: "MMM01",
	0x0C: "MMM01 + RAM",
	0x0D: "MMM01 + RAM + Battery",
	0x0F: "MBC3 + Timer + Battery",
	0x10: "MBC3 + Timer + RAM + Battery",
	0x11: "MBC3",
	0x12: "MBC3 + RAM",
	0x13: "MBC3 + RAM + Battery",
	0x19: "MBC5",
	0x1A: "MBC5 + RAM",
	0x1B: "MBC5 + RAM + Battery",
	0x1C: "MBC5 + Rumble",
	0x1D: "MBC5 + Rumble + RAM",
	0x1E: "MBC5 + Rumble + RAM + Battery",
	0x20: "MBC6",
	0x22: "MBC7 + Sensor + Rumble + RAM + Battery",
	0xFC: "Pocket Camera",
	0xFD: "Bandai TAMA5",
	0xFE: "HuC3",
	0xFF: "HuC1 + RAM + Battery",
}

// GB ROM size and bank count lookup table
var gbROMSizeBanks = map[byte]struct {
	size  int
	banks int
}{
	0x00: {32768, 2},
	0x01: {65536, 4},
	0x02: {131072, 8},
	0x03: {262144, 16},
	0x04: {524288, 32},
	0x05: {1048576, 64},
	0x06: {2097152, 128},
	0x07: {4194304, 256},
	0x08: {8388608, 512},
	0x52: {1179648, 72},
	0x53: {1310720, 80},
	0x54: {1572864, 96},
}

// GB RAM size and bank count lookup table
var gbRAMSizeBanks = map[byte]struct {
	size  int
	banks int
}{
	0x00: {0, 0},
	0x01: {2048, 1},
	0x02: {8192, 1},
	0x03: {32768, 4},
	0x04: {131072, 16},
	0x05: {65536, 8},
}

// GB new licensee codes (for 0x014B == 0x33)
var gbLicenseeNewCodes = map[string]string{
	"00": "None",
	"01": "Nintendo R&D1",
	"08": "Capcom",
	"13": "Electronic Arts",
	"18": "Hudson Soft",
	"19": "b-ai",
	"20": "kss",
	"22": "pow",
	"24": "PCM Complete",
	"25": "san-x",
	"28": "Kemco Japan",
	"29": "seta",
	"30": "Viacom",
	"31": "Nintendo",
	"32": "Bandai",
	"33": "Ocean/Acclaim",
	"34": "Konami",
	"35": "Hector",
	"37": "Taito",
	"38": "Hudson",
	"39": "Banpresto",
	"41": "Ubi Soft",
	"42": "Atlus",
	"44": "Malibu",
	"46": "angel",
	"47": "Bullet-Proof",
	"49": "irem",
	"50": "Absolute",
	"51": "Acclaim",
	"52": "Activision",
	"53": "American sammy",
	"54": "Konami",
	"55": "Hi tech entertainment",
	"56": "LJN",
	"57": "Matchbox",
	"58": "Mattel",
	"59": "Milton Bradley",
	"60": "Titus",
	"61": "Virgin",
	"64": "LucasArts",
	"67": "Ocean",
	"69": "Electronic Arts",
	"70": "Infogrames",
	"71": "Interplay",
	"72": "Broderbund",
	"73": "sculptured",
	"75": "sci",
	"78": "THQ",
	"79": "Accolade",
	"80": "misawa",
	"83": "lozc",
	"86": "Tokuma Shoten Intermedia",
	"87": "Tsukuda Original",
	"91": "Chunsoft",
	"92": "Video system",
	"93": "Ocean/Acclaim",
	"95": "Varie",
	"96": "Yonezawa/s'pal",
	"97": "Kaneko",
	"99": "Pack in soft",
	"A4": "Konami (Yu-Gi-Oh!)",
}

// GB old licensee codes
var gbLicenseeOldCodes = map[byte]string{
	0x00: "None",
	0x01: "Nintendo",
	0x08: "Capcom",
	0x09: "Hot-B",
	0x0A: "Jaleco",
	0x0B: "Coconuts Japan",
	0x0C: "Elite Systems",
	0x13: "EA (Electronic Arts)",
	0x18: "Hudsonsoft",
	0x19: "ITC Entertainment",
	0x1A: "Yanoman",
	0x1D: "Japan Clary",
	0x1F: "Virgin Interactive",
	0x24: "PCM Complete",
	0x25: "San-X",
	0x28: "Kotobuki Systems",
	0x29: "Seta",
	0x30: "Infogrames",
	0x31: "Nintendo",
	0x32: "Bandai",
	0x34: "Konami",
	0x35: "HectorSoft",
	0x38: "Capcom",
	0x39: "Banpresto",
	0x3C: ".Entertainment i",
	0x3E: "Gremlin",
	0x41: "Ubisoft",
	0x42: "Atlus",
	0x44: "Malibu",
	0x46: "Angel",
	0x47: "Spectrum Holoby",
	0x49: "Irem",
	0x4A: "Virgin Interactive",
	0x4D: "Malibu",
	0x4F: "U.S. Gold",
	0x50: "Absolute",
	0x51: "Acclaim",
	0x52: "Activision",
	0x53: "American Sammy",
	0x54: "GameTek",
	0x55: "Park Place",
	0x56: "LJN",
	0x57: "Matchbox",
	0x59: "Milton Bradley",
	0x5A: "Mindscape",
	0x5B: "Romstar",
	0x5C: "Naxat Soft",
	0x5D: "Tradewest",
	0x60: "Titus",
	0x61: "Virgin Interactive",
	0x67: "Ocean Interactive",
	0x69: "EA (Electronic Arts)",
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
	0x86: "Tokuma Shoten Intermedia",
	0x8B: "Bullet-Proof Software",
	0x8C: "Vic Tokai",
	0x8E: "Ape",
	0x8F: "I'Max",
	0x91: "Chunsoft Co.",
	0x92: "Video System",
	0x93: "Tsubaraya Productions Co.",
	0x95: "Varie Corporation",
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
	0xB0: "acclaim",
	0xB1: "ASCII or Nexsoft",
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
	0xC4: "Tokuma Shoten Intermedia",
	0xC5: "Data East",
	0xC6: "Tonkinhouse",
	0xC8: "Koei",
	0xC9: "UFL",
	0xCA: "Ultra",
	0xCB: "Vap",
	0xCC: "Use Corporation",
	0xCD: "Meldac",
	0xCE: ".Pony Canyon or",
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
	0xE5: "Epcoh",
	0xE7: "Athena",
	0xE8: "Asmik ACE Entertainment",
	0xE9: "Natsume",
	0xEA: "King Records",
	0xEB: "Atlus",
	0xEC: "Epic/Sony Records",
	0xEE: "IGS",
	0xF0: "A Wave",
	0xF3: "Extreme Entertainment",
	0xFF: "LJN",
}

// GBIdentifier identifies Game Boy and Game Boy Color games.
type GBIdentifier struct {
	// ForceGBC forces identification as GBC even when the ROM supports both
	ForceGBC bool
}

// NewGBIdentifier creates a new GB/GBC identifier.
func NewGBIdentifier() *GBIdentifier {
	return &GBIdentifier{}
}

// Console returns the console type.
func (g *GBIdentifier) Console() Console {
	return ConsoleGB
}

// Identify extracts GB/GBC game information from the given reader.
func (g *GBIdentifier) Identify(r io.ReaderAt, size int64, db Database) (*Result, error) {
	if size < gbHeaderSize {
		return nil, ErrInvalidFormat{Console: ConsoleGB, Reason: "file too small"}
	}

	// Read the entire file for checksum calculation
	data := make([]byte, size)
	if _, err := r.ReadAt(data, 0); err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read GB ROM: %w", err)
	}

	// Validate Nintendo logo
	logo := data[gbNintendoLogoOffset : gbNintendoLogoOffset+gbNintendoLogoSize]
	if !binary.BytesEqual(logo, gbNintendoLogo) {
		// Not necessarily fatal, but suspicious
	}

	// Check CGB flag to determine if it's a GBC game
	cgbFlag := data[gbCGBFlagOffset]
	var cgbMode string
	var console Console = ConsoleGB

	switch {
	case cgbFlag == 0x80:
		cgbMode = "GBC (supports GB)"
		console = ConsoleGBC
	case cgbFlag == 0xC0:
		cgbMode = "GBC only"
		console = ConsoleGBC
	case (cgbFlag & 0x0C) != 0:
		cgbMode = "PGB"
	default:
		cgbMode = "GB"
	}

	if g.ForceGBC {
		console = ConsoleGBC
	}

	// Extract manufacturer code and title
	// If bytes 0x013F-0x0142 are all uppercase letters, it's a manufacturer code
	// and the title is only 11 bytes. Otherwise, title is 16 bytes.
	var title string
	var manufacturerCode string

	mfgBytes := data[gbManufacturerOffset : gbManufacturerOffset+gbManufacturerSize]
	isManufacturerCode := true
	for _, b := range mfgBytes {
		if b < 'A' || b > 'Z' {
			isManufacturerCode = false
			break
		}
	}

	if isManufacturerCode {
		manufacturerCode = string(mfgBytes)
		title = binary.ExtractPrintable(data[gbTitleOffset : gbTitleOffset+gbTitleSize])
	} else {
		title = binary.ExtractPrintable(data[gbTitleOffset : gbTitleOffset+16])
	}

	// SGB support
	sgbSupport := data[gbSGBFlagOffset] == 0x03

	// Cartridge type
	cartridgeType := "Unknown"
	if ct, ok := gbCartridgeTypes[data[gbCartridgeTypeOffset]]; ok {
		cartridgeType = ct
	}

	// ROM size and banks
	romSize := "Unknown"
	romBanks := "Unknown"
	if rs, ok := gbROMSizeBanks[data[gbROMSizeOffset]]; ok {
		romSize = fmt.Sprintf("%d", rs.size)
		romBanks = fmt.Sprintf("%d", rs.banks)
	}

	// RAM size and banks
	ramSize := "Unknown"
	ramBanks := "Unknown"
	if rs, ok := gbRAMSizeBanks[data[gbRAMSizeOffset]]; ok {
		ramSize = fmt.Sprintf("%d", rs.size)
		ramBanks = fmt.Sprintf("%d", rs.banks)
	}

	// Licensee
	licensee := "Unknown"
	if data[gbOldLicenseeOffset] == 0x33 {
		// Use new licensee code
		newLicenseeCode := string(data[gbNewLicenseeOffset : gbNewLicenseeOffset+gbNewLicenseeSize])
		if l, ok := gbLicenseeNewCodes[newLicenseeCode]; ok {
			licensee = l
		}
	} else {
		// Use old licensee code
		if l, ok := gbLicenseeOldCodes[data[gbOldLicenseeOffset]]; ok {
			licensee = l
		}
	}

	// ROM version
	romVersion := data[gbROMVersionOffset]

	// Header checksum
	headerChecksumExpected := data[gbHeaderChecksumOffset]
	headerChecksumActual := uint8(0)
	for i := 0x0134; i < 0x014D; i++ {
		headerChecksumActual = headerChecksumActual - data[i] - 1
	}

	// Global checksum
	globalChecksumExpected := uint16(data[gbGlobalChecksumOffset])<<8 | uint16(data[gbGlobalChecksumOffset+1])
	var globalChecksumActual uint16
	for i, b := range data {
		if i != gbGlobalChecksumOffset && i != gbGlobalChecksumOffset+1 {
			globalChecksumActual += uint16(b)
		}
	}

	result := NewResult(console)
	result.InternalTitle = title
	result.SetMetadata("internal_title", title)
	result.SetMetadata("cgb_mode", cgbMode)
	result.SetMetadata("sgb_support", fmt.Sprintf("%t", sgbSupport))
	result.SetMetadata("cartridge_type", cartridgeType)
	result.SetMetadata("rom_size", romSize)
	result.SetMetadata("rom_banks", romBanks)
	result.SetMetadata("ram_size", ramSize)
	result.SetMetadata("ram_banks", ramBanks)
	result.SetMetadata("licensee", licensee)
	result.SetMetadata("rom_version", fmt.Sprintf("%d", romVersion))
	result.SetMetadata("header_checksum_expected", fmt.Sprintf("0x%02x", headerChecksumExpected))
	result.SetMetadata("header_checksum_actual", fmt.Sprintf("0x%02x", headerChecksumActual))
	result.SetMetadata("global_checksum_expected", fmt.Sprintf("0x%04x", globalChecksumExpected))
	result.SetMetadata("global_checksum_actual", fmt.Sprintf("0x%04x", globalChecksumActual))

	if manufacturerCode != "" {
		result.SetMetadata("manufacturer_code", manufacturerCode)
	}

	// Database lookup uses (title, global_checksum) as key
	if db != nil {
		type gbKey struct {
			title    string
			checksum uint16
		}
		key := gbKey{title: title, checksum: globalChecksumExpected}
		if entry, found := db.Lookup(ConsoleGB, key); found {
			result.MergeMetadata(entry, false)
		}
	}

	// If no title from database, use internal title
	if result.Title == "" {
		result.Title = result.InternalTitle
	}

	return result, nil
}

// ValidateGB checks if the given data looks like a valid GB/GBC ROM.
func ValidateGB(header []byte) bool {
	if len(header) < gbHeaderSize {
		return false
	}
	logo := header[gbNintendoLogoOffset : gbNintendoLogoOffset+gbNintendoLogoSize]
	return binary.BytesEqual(logo, gbNintendoLogo)
}

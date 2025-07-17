package identifiers

import (
	"github.com/wizzomafizzo/go-gameid/pkg/database"
	"testing"
)

func TestGenesisIdentifier_Console(t *testing.T) {
	identifier := NewGenesisIdentifier(nil)
	if identifier.Console() != "Genesis" {
		t.Errorf("Expected console 'Genesis', got %s", identifier.Console())
	}
}

func TestGenesisIdentifier_MagicWordDetection(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
		offset   int
	}{
		{
			name:     "SEGA GENESIS magic word",
			data:     make([]byte, 0x200),
			expected: true,
			offset:   0x100,
		},
		{
			name:     "SEGA MEGA DRIVE magic word",
			data:     make([]byte, 0x200),
			expected: true,
			offset:   0x120,
		},
		{
			name:     "No magic word",
			data:     make([]byte, 0x200),
			expected: false,
			offset:   -1,
		},
	}

	// Set up test data
	copy(tests[0].data[0x100:], []byte("SEGA GENESIS    "))
	copy(tests[1].data[0x120:], []byte("SEGA MEGA DRIVE "))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset := findGenesisMagicWord(tt.data)
			if tt.expected {
				if offset != tt.offset {
					t.Errorf("Expected offset %d, got %d", tt.offset, offset)
				}
			} else {
				if offset != -1 {
					t.Errorf("Expected offset -1, got %d", offset)
				}
			}
		})
	}
}

func TestGenesisIdentifier_HeaderParsing(t *testing.T) {
	// Create test ROM data with Genesis header
	data := make([]byte, 0x200)
	headerStart := 0x100

	// Add SEGA GENESIS magic word at offset 0x100
	copy(data[headerStart:], []byte("SEGA GENESIS    "))

	// Add test data at various offsets from magic word
	copy(data[headerStart+0x013:], []byte("SEGA"))                             // publisher
	copy(data[headerStart+0x018:], []byte("1994"))                             // release_year
	copy(data[headerStart+0x01D:], []byte("DEC"))                              // release_month
	copy(data[headerStart+0x020:], []byte("SONIC THE HEDGEHOG              ")) // title_domestic
	copy(data[headerStart+0x050:], []byte("SONIC THE HEDGEHOG              ")) // title_overseas
	copy(data[headerStart+0x080:], []byte("GM"))                               // software_type
	copy(data[headerStart+0x082:], []byte("MK-1563 "))                         // ID
	copy(data[headerStart+0x08C:], []byte("00"))                               // revision

	// Add checksum (big-endian)
	data[headerStart+0x08E] = 0x12
	data[headerStart+0x08F] = 0x34

	// Add device support
	copy(data[headerStart+0x090:], []byte("J6      "))

	// Add ROM/RAM ranges (big-endian)
	copy(data[headerStart+0x0A0:], []byte{0x00, 0x00, 0x00, 0x00}) // rom_start
	copy(data[headerStart+0x0A4:], []byte{0x00, 0x07, 0xFF, 0xFF}) // rom_end
	copy(data[headerStart+0x0A8:], []byte{0xFF, 0xFF, 0x00, 0x00}) // ram_start
	copy(data[headerStart+0x0AC:], []byte{0xFF, 0xFF, 0xFF, 0xFF}) // ram_end

	// Add modem support
	copy(data[headerStart+0x0BC:], []byte("        "))

	// Add region support
	copy(data[headerStart+0x0F0:], []byte("JUE"))

	result := parseGenesisHeader(data, headerStart)

	// Verify parsed fields
	tests := []struct {
		key      string
		expected string
	}{
		{"system_type", "SEGA GENESIS"},
		{"publisher", "SEGA"},
		{"release_year", "1994"},
		{"release_month", "December"},
		{"title_domestic", "SONIC THE HEDGEHOG"},
		{"title_overseas", "SONIC THE HEDGEHOG"},
		{"software_type", "Game"},
		{"ID", "MK-1563"},
		{"revision", "00"},
		{"checksum", "0x1234"},
		{"device_support", "3-button Controller / 6-button Controller"},
		{"rom_start", "0x0"},
		{"rom_end", "0x7ffff"},
		{"ram_start", "0xffff0000"},
		{"ram_end", "0xffffffff"},
		{"region_support", "Americas / Europe / Japan"},
	}

	for _, tt := range tests {
		if result[tt.key] != tt.expected {
			t.Errorf("Expected %s='%s', got '%s'", tt.key, tt.expected, result[tt.key])
		}
	}
}

func TestGenesisIdentifier_DatabaseLookup(t *testing.T) {
	// Create test database
	db := &database.GameDatabase{
		Systems: map[string]database.SystemDatabase{
			"Genesis": {
				"MK_1563": {
					"title": "Sonic the Hedgehog",
					"genre": "Platform",
				},
			},
		},
	}

	identifier := NewGenesisIdentifier(db)

	// Create test ROM data
	data := make([]byte, 0x200)
	headerStart := 0x100

	// Add SEGA GENESIS magic word
	copy(data[headerStart:], []byte("SEGA GENESIS    "))

	// Add ID that should be found in database
	copy(data[headerStart+0x082:], []byte("MK-1563 "))

	// Add title_overseas as fallback
	copy(data[headerStart+0x050:], []byte("SONIC THE HEDGEHOG              "))

	result := parseGenesisHeader(data, headerStart)

	// Test database lookup
	if identifier.db != nil {
		if gameData, found := identifier.db.LookupGame("Genesis", "MK_1563"); found {
			for key, value := range gameData {
				result[key] = value
			}
		}
	}

	// Should have database title
	if result["title"] != "Sonic the Hedgehog" {
		t.Errorf("Expected title 'Sonic the Hedgehog', got '%s'", result["title"])
	}
	if result["genre"] != "Platform" {
		t.Errorf("Expected genre 'Platform', got '%s'", result["genre"])
	}
}

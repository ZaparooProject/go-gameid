package identifier

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ZaparooProject/go-gameid/internal/binary"
	"github.com/ZaparooProject/go-gameid/iso9660"
)

// Saturn magic word
var saturnMagicWord = []byte("SEGA SEGASATURN")

// Saturn device support codes
var saturnDeviceSupport = map[byte]string{
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

// Saturn target area codes
var saturnTargetAreas = map[byte]string{
	'J': "Japan",
	'T': "Asia NTSC (Taiwan, Philippines)",
	'U': "North America (USA, Canada)",
	'B': "Central and South America NTSC (Brazil)",
	'K': "Korea",
	'A': "East Asia PAL (China, Middle and Near East)",
	'E': "Europe PAL",
	'L': "Central and South America PAL",
}

// SaturnIdentifier identifies Sega Saturn games.
type SaturnIdentifier struct{}

// NewSaturnIdentifier creates a new Saturn identifier.
func NewSaturnIdentifier() *SaturnIdentifier {
	return &SaturnIdentifier{}
}

// Console returns the console type.
func (s *SaturnIdentifier) Console() Console {
	return ConsoleSaturn
}

// Identify extracts Saturn game information from the given reader.
func (s *SaturnIdentifier) Identify(r io.ReaderAt, size int64, db Database) (*Result, error) {
	if size < 0x100 {
		return nil, ErrInvalidFormat{Console: ConsoleSaturn, Reason: "file too small"}
	}

	// Read header
	header, err := binary.ReadBytesAt(r, 0, 0x100)
	if err != nil {
		return nil, fmt.Errorf("failed to read Saturn header: %w", err)
	}

	return s.identifyFromHeader(header, db)
}

// IdentifyFromPath identifies a Saturn game from a file path.
func (s *SaturnIdentifier) IdentifyFromPath(path string, db Database) (*Result, error) {
	ext := strings.ToLower(filepath.Ext(path))

	var header []byte

	if ext == ".cue" {
		cue, err := iso9660.ParseCue(path)
		if err != nil {
			return nil, err
		}
		if len(cue.BinFiles) == 0 {
			return nil, ErrInvalidFormat{Console: ConsoleSaturn, Reason: "no BIN files in CUE"}
		}
		f, err := os.Open(cue.BinFiles[0])
		if err != nil {
			return nil, err
		}
		defer f.Close()
		header = make([]byte, 0x100)
		if _, err := f.Read(header); err != nil {
			return nil, err
		}
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		header = make([]byte, 0x100)
		if _, err := f.Read(header); err != nil {
			return nil, err
		}
	}

	return s.identifyFromHeader(header, db)
}

func (s *SaturnIdentifier) identifyFromHeader(header []byte, db Database) (*Result, error) {
	// Find magic word
	magicIdx := binary.FindBytes(header, saturnMagicWord)
	if magicIdx == -1 {
		return nil, ErrInvalidFormat{Console: ConsoleSaturn, Reason: "magic word not found"}
	}

	// Extract fields relative to magic word position
	extractString := func(offset, length int) string {
		start := magicIdx + offset
		end := start + length
		if end > len(header) {
			return ""
		}
		return strings.TrimSpace(string(header[start:end]))
	}

	manufacturerID := extractString(0x10, 0x10)
	gameID := extractString(0x20, 0x0A)
	// Split on space and take first part
	if idx := strings.Index(gameID, " "); idx != -1 {
		gameID = gameID[:idx]
	}
	version := extractString(0x2A, 0x06)
	deviceInfo := extractString(0x38, 0x08)
	internalTitle := extractString(0x60, 0x70)

	// Release date (YYYYMMDD at offset 0x30)
	releaseDateRaw := extractString(0x30, 0x08)
	var releaseDate string
	if len(releaseDateRaw) == 8 {
		releaseDate = fmt.Sprintf("%s-%s-%s", releaseDateRaw[0:4], releaseDateRaw[4:6], releaseDateRaw[6:8])
	}

	// Device support (offset 0x50, 16 bytes)
	var deviceSupport []string
	if magicIdx+0x60 <= len(header) {
		for _, b := range header[magicIdx+0x50 : magicIdx+0x60] {
			if b == 0 || b == ' ' {
				continue
			}
			if dev, ok := saturnDeviceSupport[b]; ok {
				deviceSupport = append(deviceSupport, dev)
			}
		}
	}

	// Target area (offset 0x40, 16 bytes)
	var targetArea []string
	if magicIdx+0x50 <= len(header) {
		for _, b := range header[magicIdx+0x40 : magicIdx+0x50] {
			if b == 0 || b == ' ' {
				continue
			}
			if area, ok := saturnTargetAreas[b]; ok {
				targetArea = append(targetArea, area)
			}
		}
	}

	// Normalize serial for database lookup
	serial := strings.ReplaceAll(gameID, "-", "")
	serial = strings.ReplaceAll(serial, " ", "")
	serial = strings.TrimSpace(serial)

	result := NewResult(ConsoleSaturn)
	result.ID = gameID
	result.InternalTitle = internalTitle
	result.SetMetadata("manufacturer_ID", manufacturerID)
	result.SetMetadata("ID", gameID)
	result.SetMetadata("version", version)
	result.SetMetadata("device_info", deviceInfo)
	result.SetMetadata("internal_title", internalTitle)

	if releaseDate != "" {
		result.SetMetadata("release_date", releaseDate)
	}

	if len(deviceSupport) > 0 {
		result.SetMetadata("device_support", strings.Join(deviceSupport, " / "))
	}

	if len(targetArea) > 0 {
		result.SetMetadata("target_area", strings.Join(targetArea, " / "))
	}

	// Database lookup
	if db != nil && serial != "" {
		if entry, found := db.LookupByString(ConsoleSaturn, serial); found {
			result.MergeMetadata(entry, false)
		}
	}

	// If no title from database, use internal title
	if result.Title == "" {
		result.Title = result.InternalTitle
	}

	return result, nil
}

// ValidateSaturn checks if the given data looks like a valid Saturn disc.
func ValidateSaturn(header []byte) bool {
	return binary.FindBytes(header, saturnMagicWord) != -1
}

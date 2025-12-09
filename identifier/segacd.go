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

// SegaCD magic words
var segaCDMagicWords = [][]byte{
	[]byte("SEGADISCSYSTEM"),
	[]byte("SEGABOOTDISC"),
	[]byte("SEGADISC"),
	[]byte("SEGADATADISC"),
}

// SegaCDIdentifier identifies Sega CD games.
type SegaCDIdentifier struct{}

// NewSegaCDIdentifier creates a new Sega CD identifier.
func NewSegaCDIdentifier() *SegaCDIdentifier {
	return &SegaCDIdentifier{}
}

// Console returns the console type.
func (s *SegaCDIdentifier) Console() Console {
	return ConsoleSegaCD
}

// Identify extracts Sega CD game information from the given reader.
func (s *SegaCDIdentifier) Identify(r io.ReaderAt, size int64, db Database) (*Result, error) {
	if size < 0x300 {
		return nil, ErrInvalidFormat{Console: ConsoleSegaCD, Reason: "file too small"}
	}

	// Read header
	header, err := binary.ReadBytesAt(r, 0, 0x300)
	if err != nil {
		return nil, fmt.Errorf("failed to read Sega CD header: %w", err)
	}

	return s.identifyFromHeader(header, db, nil)
}

// IdentifyFromPath identifies a Sega CD game from a file path.
func (s *SegaCDIdentifier) IdentifyFromPath(path string, db Database) (*Result, error) {
	ext := strings.ToLower(filepath.Ext(path))

	var header []byte
	var iso *iso9660.ISO9660

	if ext == ".cue" {
		cue, err := iso9660.ParseCue(path)
		if err != nil {
			return nil, err
		}
		if len(cue.BinFiles) == 0 {
			return nil, ErrInvalidFormat{Console: ConsoleSegaCD, Reason: "no BIN files in CUE"}
		}
		f, err := os.Open(cue.BinFiles[0])
		if err != nil {
			return nil, err
		}
		defer f.Close()
		header = make([]byte, 0x300)
		if _, err := f.Read(header); err != nil {
			return nil, err
		}
		iso, _ = iso9660.OpenCue(path)
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		header = make([]byte, 0x300)
		if _, err := f.Read(header); err != nil {
			return nil, err
		}
		iso, _ = iso9660.Open(path)
	}

	if iso != nil {
		defer iso.Close()
	}

	return s.identifyFromHeader(header, db, iso)
}

func (s *SegaCDIdentifier) identifyFromHeader(header []byte, db Database, iso *iso9660.ISO9660) (*Result, error) {
	// Find magic word
	var magicIdx int = -1
	for _, magic := range segaCDMagicWords {
		idx := binary.FindBytes(header, magic)
		if idx != -1 {
			magicIdx = idx
			break
		}
	}

	if magicIdx == -1 {
		return nil, ErrInvalidFormat{Console: ConsoleSegaCD, Reason: "magic word not found"}
	}

	// Extract fields relative to magic word position (same as Genesis layout)
	extractString := func(offset, length int) string {
		start := magicIdx + offset
		end := start + length
		if end > len(header) {
			return ""
		}
		return strings.TrimSpace(string(header[start:end]))
	}

	discID := extractString(0x000, 0x10)
	discVolumeName := extractString(0x010, 0x0B)
	systemName := extractString(0x020, 0x0B)

	// Build date at 0x50 (MMDDYYYY format)
	buildDateRaw := extractString(0x050, 0x08)
	var buildDate string
	if len(buildDateRaw) == 8 {
		// Convert MMDDYYYY to YYYY-MM-DD
		buildDate = buildDateRaw[4:8] + "-" + buildDateRaw[0:2] + "-" + buildDateRaw[2:4]
	} else {
		buildDate = buildDateRaw
	}

	// System type and release info (at 0x100+ like Genesis)
	systemType := extractString(0x100, 0x10)
	releaseYear := extractString(0x118, 0x04)
	releaseMonth := extractString(0x11D, 0x03)

	// Titles
	titleDomestic := extractString(0x120, 0x30)
	titleOverseas := extractString(0x150, 0x30)

	// Software type and ID
	gameID := extractString(0x180, 0x10)

	// Device support
	deviceSupportBytes := header[magicIdx+0x190 : magicIdx+0x1A0]
	var deviceSupport []string
	for _, b := range deviceSupportBytes {
		if b == 0 || b == ' ' {
			continue
		}
		if dev, ok := genesisDeviceSupport[b]; ok {
			deviceSupport = append(deviceSupport, dev)
		}
	}

	// Region support (at 0x1F0 from magic word for Genesis layout)
	var regionSupport []string
	if magicIdx+0x1F3 <= len(header) {
		for _, b := range header[magicIdx+0x1F0 : magicIdx+0x1F3] {
			if b < '!' || b > '~' {
				continue
			}
			if reg, ok := genesisRegionSupport[b]; ok {
				regionSupport = append(regionSupport, reg)
			}
		}
	}

	// Normalize serial for database lookup
	serial := strings.ReplaceAll(gameID, "#", "")
	serial = strings.ReplaceAll(serial, "-", "")
	serial = strings.ReplaceAll(serial, " ", "")
	serial = strings.TrimSpace(serial)

	result := NewResult(ConsoleSegaCD)
	result.ID = gameID
	result.InternalTitle = titleOverseas
	if result.InternalTitle == "" {
		result.InternalTitle = titleDomestic
	}

	result.SetMetadata("disc_ID", discID)
	result.SetMetadata("disc_volume_name", discVolumeName)
	result.SetMetadata("system_name", systemName)
	result.SetMetadata("build_date", buildDate)
	result.SetMetadata("system_type", systemType)
	result.SetMetadata("release_year", releaseYear)
	result.SetMetadata("release_month", releaseMonth)
	result.SetMetadata("title_domestic", titleDomestic)
	result.SetMetadata("title_overseas", titleOverseas)
	result.SetMetadata("ID", gameID)

	if len(deviceSupport) > 0 {
		result.SetMetadata("device_support", strings.Join(deviceSupport, " / "))
	}

	if len(regionSupport) > 0 {
		result.SetMetadata("region_support", strings.Join(regionSupport, " / "))
	}

	// Add ISO metadata if available
	if iso != nil {
		result.SetMetadata("uuid", iso.GetUUID())
		result.SetMetadata("volume_ID", iso.GetVolumeID())
	}

	// Database lookup
	if db != nil && serial != "" {
		if entry, found := db.LookupByString(ConsoleSegaCD, serial); found {
			result.MergeMetadata(entry, false)
		}
	}

	// If no title from database, use overseas title
	if result.Title == "" {
		result.Title = result.InternalTitle
	}

	return result, nil
}

// ValidateSegaCD checks if the given data looks like a valid Sega CD disc.
func ValidateSegaCD(header []byte) bool {
	for _, magic := range segaCDMagicWords {
		if binary.FindBytes(header, magic) != -1 {
			return true
		}
	}
	return false
}

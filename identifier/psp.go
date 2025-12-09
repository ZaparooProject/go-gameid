package identifier

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/ZaparooProject/go-gameid/iso9660"
)

// PSPIdentifier identifies PlayStation Portable games.
type PSPIdentifier struct{}

// NewPSPIdentifier creates a new PSP identifier.
func NewPSPIdentifier() *PSPIdentifier {
	return &PSPIdentifier{}
}

// Console returns the console type.
func (p *PSPIdentifier) Console() Console {
	return ConsolePSP
}

// Identify extracts PSP game information from the given reader.
// For disc-based games, use IdentifyFromPath instead.
func (p *PSPIdentifier) Identify(r io.ReaderAt, size int64, db Database) (*Result, error) {
	return nil, ErrNotSupported{Format: "raw reader for PSP"}
}

// IdentifyFromPath identifies a PSP game from a file path.
func (p *PSPIdentifier) IdentifyFromPath(path string, db Database) (*Result, error) {
	iso, err := iso9660.Open(path)
	if err != nil {
		return nil, err
	}
	defer iso.Close()

	return p.identifyFromISO(iso, db)
}

func (p *PSPIdentifier) identifyFromISO(iso *iso9660.ISO9660, db Database) (*Result, error) {
	result := NewResult(ConsolePSP)

	// Look for UMD_DATA.BIN in root
	files, err := iso.IterFiles(true)
	if err != nil {
		return nil, err
	}

	var umdDataInfo *iso9660.FileInfo
	for _, f := range files {
		if strings.ToUpper(filepath.Base(f.Path)) == "UMD_DATA.BIN" {
			umdDataInfo = &f
			break
		}
	}

	if umdDataInfo == nil {
		return nil, ErrInvalidFormat{Console: ConsolePSP, Reason: "UMD_DATA.BIN not found"}
	}

	// Read UMD_DATA.BIN
	data, err := iso.ReadFile(*umdDataInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to read UMD_DATA.BIN: %w", err)
	}

	// Extract serial (until first '|' character)
	var serial string
	for _, b := range data {
		if b == '|' {
			break
		}
		serial += string(b)
	}
	serial = strings.TrimSpace(serial)

	result.ID = serial
	result.SetMetadata("ID", serial)
	result.SetMetadata("uuid", iso.GetUUID())
	result.SetMetadata("volume_ID", iso.GetVolumeID())

	// Database lookup
	if db != nil && serial != "" {
		if entry, found := db.LookupByString(ConsolePSP, serial); found {
			result.MergeMetadata(entry, false)
		}
	}

	return result, nil
}

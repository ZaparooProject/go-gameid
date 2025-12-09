package identifier

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/ZaparooProject/go-gameid/iso9660"
)

// PS2Identifier identifies PlayStation 2 games.
type PS2Identifier struct{}

// NewPS2Identifier creates a new PS2 identifier.
func NewPS2Identifier() *PS2Identifier {
	return &PS2Identifier{}
}

// Console returns the console type.
func (p *PS2Identifier) Console() Console {
	return ConsolePS2
}

// Identify extracts PS2 game information from the given reader.
// For disc-based games, use IdentifyFromPath instead.
func (p *PS2Identifier) Identify(r io.ReaderAt, size int64, db Database) (*Result, error) {
	return nil, ErrNotSupported{Format: "raw reader for PS2"}
}

// IdentifyFromPath identifies a PS2 game from a file path.
func (p *PS2Identifier) IdentifyFromPath(path string, db Database) (*Result, error) {
	var iso interface {
		GetUUID() string
		GetVolumeID() string
		IterFiles(onlyRootDir bool) ([]iso9660.FileInfo, error)
		Close() error
	}

	ext := strings.ToLower(filepath.Ext(path))

	if ext == ".cue" {
		i, err := iso9660.OpenCue(path)
		if err != nil {
			return nil, err
		}
		iso = i
	} else {
		i, err := iso9660.Open(path)
		if err != nil {
			return nil, err
		}
		iso = i
	}
	defer iso.Close()

	return p.identifyFromISO(iso, db, path)
}

func (p *PS2Identifier) identifyFromISO(iso interface {
	GetUUID() string
	GetVolumeID() string
	IterFiles(onlyRootDir bool) ([]iso9660.FileInfo, error)
}, db Database, sourcePath string) (*Result, error) {
	result := NewResult(ConsolePS2)

	// Get root files
	files, err := iso.IterFiles(true)
	if err != nil {
		return nil, err
	}

	// Build list of root filenames
	var rootFiles []string
	for _, f := range files {
		name := strings.TrimPrefix(f.Path, "/")
		// Remove version suffix (;1)
		if idx := strings.Index(name, ";"); idx != -1 {
			name = name[:idx]
		}
		rootFiles = append(rootFiles, name)
	}

	// Try to find serial from root files using ID prefixes
	var serial string
	if db != nil {
		prefixes := db.GetIDPrefixes(ConsolePS2)
		for _, prefix := range prefixes {
			for _, fn := range rootFiles {
				fnUpper := strings.ToUpper(fn)
				if strings.HasPrefix(fnUpper, prefix) {
					// Normalize: remove dots, replace dashes with underscores
					serial = strings.ReplaceAll(fnUpper, ".", "")
					serial = strings.ReplaceAll(serial, "-", "_")

					// Try lookup
					if _, found := db.LookupByString(ConsolePS2, serial); found {
						break
					}

					// Try with underscore after prefix
					if len(serial) > len(prefix) {
						altSerial := serial[:len(prefix)] + "_" + serial[len(prefix)+1:]
						if _, found := db.LookupByString(ConsolePS2, altSerial); found {
							serial = altSerial
							break
						}
					}
				}
			}
			if serial != "" {
				break
			}
		}
	}

	// Fallback to volume ID
	if serial == "" {
		volumeID := iso.GetVolumeID()
		if volumeID != "" {
			serial = strings.ReplaceAll(volumeID, "-", "_")
			// If there are 2 underscores, keep only first 2 parts
			parts := strings.Split(serial, "_")
			if len(parts) > 2 {
				serial = strings.Join(parts[:2], "_")
			}
		}
	}

	// Fallback to filename
	if serial == "" && sourcePath != "" {
		fn := filepath.Base(sourcePath)
		fn = strings.TrimSuffix(fn, filepath.Ext(fn))
		fn = strings.TrimSuffix(fn, ".gz")
		serial = fn
	}

	result.ID = strings.ReplaceAll(serial, "_", "-")
	result.SetMetadata("ID", result.ID)
	result.SetMetadata("uuid", iso.GetUUID())
	result.SetMetadata("volume_ID", iso.GetVolumeID())
	result.SetMetadata("root_files", strings.Join(rootFiles, " / "))

	// Database lookup
	if db != nil && serial != "" {
		if entry, found := db.LookupByString(ConsolePS2, serial); found {
			result.MergeMetadata(entry, false)
		}
	}

	return result, nil
}

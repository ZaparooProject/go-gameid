package identifier

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/ZaparooProject/go-gameid/iso9660"
)

// NeoGeoCDIdentifier identifies Neo Geo CD games.
type NeoGeoCDIdentifier struct{}

// NewNeoGeoCDIdentifier creates a new Neo Geo CD identifier.
func NewNeoGeoCDIdentifier() *NeoGeoCDIdentifier {
	return &NeoGeoCDIdentifier{}
}

// Console returns the console type.
func (n *NeoGeoCDIdentifier) Console() Console {
	return ConsoleNeoGeoCD
}

// Identify extracts Neo Geo CD game information from the given reader.
// For disc-based games, use IdentifyFromPath instead.
func (n *NeoGeoCDIdentifier) Identify(r io.ReaderAt, size int64, db Database) (*Result, error) {
	return nil, ErrNotSupported{Format: "raw reader for NeoGeoCD"}
}

// IdentifyFromPath identifies a Neo Geo CD game from a file path.
func (n *NeoGeoCDIdentifier) IdentifyFromPath(path string, db Database) (*Result, error) {
	var iso interface {
		GetUUID() string
		GetVolumeID() string
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

	return n.identifyFromISO(iso, db)
}

func (n *NeoGeoCDIdentifier) identifyFromISO(iso interface {
	GetUUID() string
	GetVolumeID() string
}, db Database) (*Result, error) {
	result := NewResult(ConsoleNeoGeoCD)

	uuid := iso.GetUUID()
	volumeID := iso.GetVolumeID()

	result.SetMetadata("uuid", uuid)
	result.SetMetadata("volume_ID", volumeID)

	// NeoGeoCD uses (uuid, volume_ID) tuple as primary key, with volume_ID as fallback
	if db != nil {
		// Try (uuid, volume_ID) tuple first
		type neogeoCDKey struct {
			uuid     string
			volumeID string
		}
		key := neogeoCDKey{uuid: uuid, volumeID: volumeID}
		if entry, found := db.Lookup(ConsoleNeoGeoCD, key); found {
			result.MergeMetadata(entry, false)
		} else if volumeID != "" {
			// Fallback to just volume_ID
			if entry, found := db.LookupByString(ConsoleNeoGeoCD, volumeID); found {
				result.MergeMetadata(entry, false)
			}
		}
	}

	// Set ID from volume_ID if not set by database
	if result.ID == "" {
		result.ID = volumeID
	}

	return result, nil
}

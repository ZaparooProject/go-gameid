// Package identifier provides console-specific game identification logic.
package identifier

import (
	"fmt"
	"io"
)

// Console represents a gaming console/platform.
type Console string

// Supported console types.
const (
	ConsoleGB       Console = "GB"
	ConsoleGBC      Console = "GBC"
	ConsoleGBA      Console = "GBA"
	ConsoleGC       Console = "GC"
	ConsoleGenesis  Console = "Genesis"
	ConsoleN64      Console = "N64"
	ConsoleNeoGeoCD Console = "NeoGeoCD"
	ConsoleNES      Console = "NES"
	ConsolePSP      Console = "PSP"
	ConsolePSX      Console = "PSX"
	ConsolePS2      Console = "PS2"
	ConsoleSaturn   Console = "Saturn"
	ConsoleSegaCD   Console = "SegaCD"
	ConsoleSNES     Console = "SNES"
)

// AllConsoles is a list of all supported consoles.
var AllConsoles = []Console{
	ConsoleGB,
	ConsoleGBC,
	ConsoleGBA,
	ConsoleGC,
	ConsoleGenesis,
	ConsoleN64,
	ConsoleNeoGeoCD,
	ConsoleNES,
	ConsolePSP,
	ConsolePSX,
	ConsolePS2,
	ConsoleSaturn,
	ConsoleSegaCD,
	ConsoleSNES,
}

// Result contains the identification results for a game.
type Result struct {
	// ID is the primary game identifier (serial, game code, etc.)
	ID string

	// Title is the official game title from the database
	Title string

	// Console is the detected/specified console type
	Console Console

	// InternalTitle is the title embedded in the ROM/disc
	InternalTitle string

	// Region is the region code if available
	Region string

	// Metadata contains all extracted and database metadata as key-value pairs
	Metadata map[string]string
}

// NewResult creates a new Result with initialized metadata map.
func NewResult(console Console) *Result {
	return &Result{
		Console:  console,
		Metadata: make(map[string]string),
	}
}

// SetMetadata sets a metadata value, also updating the Result fields if applicable.
func (r *Result) SetMetadata(key, value string) {
	if value == "" {
		return
	}
	r.Metadata[key] = value

	// Also set the corresponding Result field
	switch key {
	case "ID":
		if r.ID == "" {
			r.ID = value
		}
	case "title":
		if r.Title == "" {
			r.Title = value
		}
	case "internal_title":
		if r.InternalTitle == "" {
			r.InternalTitle = value
		}
	case "region":
		if r.Region == "" {
			r.Region = value
		}
	}
}

// MergeMetadata merges database metadata into the result.
// If preferDB is true, database values overwrite extracted values.
func (r *Result) MergeMetadata(dbEntry map[string]string, preferDB bool) {
	for k, v := range dbEntry {
		if v == "" {
			continue
		}
		if _, exists := r.Metadata[k]; !exists || preferDB {
			r.SetMetadata(k, v)
		}
	}
	// If no title was set, use internal title
	if r.Title == "" && r.InternalTitle != "" {
		r.Title = r.InternalTitle
	}
}

// Database provides lookup capabilities for game metadata.
type Database interface {
	// Lookup retrieves metadata for a game by console and key.
	// The key format varies by console (see plan for details).
	Lookup(console Console, key interface{}) (map[string]string, bool)

	// LookupByString retrieves metadata using a string key.
	LookupByString(console Console, key string) (map[string]string, bool)

	// GetIDPrefixes returns the ID prefixes for disc-based consoles (PSX, PS2).
	GetIDPrefixes(console Console) []string
}

// Identifier is the interface for console-specific identification.
type Identifier interface {
	// Identify extracts game information from the given reader.
	// The reader should be positioned at the start of the file.
	// size is the total file size.
	// db can be nil if no database lookup is needed.
	Identify(r io.ReaderAt, size int64, db Database) (*Result, error)

	// Console returns the console type this identifier handles.
	Console() Console
}

// DiscIdentifier is an extended interface for disc-based games.
type DiscIdentifier interface {
	Identifier

	// IdentifyFromISO identifies a game from a parsed ISO filesystem.
	// This is used when the ISO has already been parsed.
	IdentifyFromISO(iso ISOReader, db Database) (*Result, error)
}

// ISOReader provides read access to an ISO9660 filesystem.
type ISOReader interface {
	// GetSystemID returns the system identifier from the PVD.
	GetSystemID() string

	// GetVolumeID returns the volume identifier from the PVD.
	GetVolumeID() string

	// GetPublisherID returns the publisher identifier from the PVD.
	GetPublisherID() string

	// GetDataPreparerID returns the data preparer identifier from the PVD.
	GetDataPreparerID() string

	// GetUUID returns a unique identifier derived from disc metadata.
	GetUUID() string

	// IterFiles returns a list of files in the filesystem.
	// If onlyRootDir is true, only files in the root directory are returned.
	IterFiles(onlyRootDir bool) ([]FileInfo, error)

	// ReadFile reads the contents of a file.
	ReadFile(path string) ([]byte, error)

	// FileExists checks if a file exists at the given path.
	FileExists(path string) bool
}

// FileInfo contains information about a file in an ISO filesystem.
type FileInfo struct {
	Path   string
	Offset int64
	Size   int64
}

// ErrNotSupported is returned when a file format is not supported.
type ErrNotSupported struct {
	Format string
}

func (e ErrNotSupported) Error() string {
	return fmt.Sprintf("format not supported: %s", e.Format)
}

// ErrInvalidFormat is returned when a file doesn't match the expected format.
type ErrInvalidFormat struct {
	Console Console
	Reason  string
}

func (e ErrInvalidFormat) Error() string {
	return fmt.Sprintf("invalid %s format: %s", e.Console, e.Reason)
}

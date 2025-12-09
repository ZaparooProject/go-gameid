package iso9660

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// CueSheet represents a parsed CUE sheet file.
type CueSheet struct {
	Path     string   // Path to the CUE file
	BinFiles []string // Paths to BIN files (absolute)
}

// ParseCue parses a CUE sheet file and returns the BIN file paths.
func ParseCue(cuePath string) (*CueSheet, error) {
	f, err := os.Open(cuePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cueDir := filepath.Dir(cuePath)
	cue := &CueSheet{
		Path: cuePath,
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineLower := strings.ToLower(line)

		// Look for FILE "filename" BINARY lines
		if strings.HasPrefix(lineLower, "file") {
			// Extract filename between quotes
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 {
				binFile := strings.TrimSpace(parts[1])
				// Make absolute path
				if !filepath.IsAbs(binFile) {
					binFile = filepath.Join(cueDir, binFile)
				}
				cue.BinFiles = append(cue.BinFiles, binFile)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return cue, nil
}

// OpenCue opens an ISO9660 disc image from a CUE sheet.
// It uses the first BIN file referenced in the CUE sheet.
func OpenCue(cuePath string) (*ISO9660, error) {
	cue, err := ParseCue(cuePath)
	if err != nil {
		return nil, err
	}

	if len(cue.BinFiles) == 0 {
		return nil, ErrInvalidISO
	}

	// Open the first BIN file
	return Open(cue.BinFiles[0])
}

// IsCueFile checks if the given path is a CUE file.
func IsCueFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".cue"
}

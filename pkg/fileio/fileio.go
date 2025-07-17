package fileio

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileReader interface for reading files
type FileReader interface {
	io.Reader
	io.Closer
}

// multiCloser wraps multiple closers
type multiCloser struct {
	closers []io.Closer
	reader  io.Reader
}

func (mc *multiCloser) Read(p []byte) (n int, err error) {
	return mc.reader.Read(p)
}

func (mc *multiCloser) Close() error {
	var err error
	for _, c := range mc.closers {
		if cerr := c.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}

// OpenFile opens a file for reading, automatically handling gzip compression
func OpenFile(path string) (FileReader, error) {
	// Handle special cases
	if path == "stdin" {
		return os.Stdin, nil
	}
	if path == "stdout" {
		return nil, fmt.Errorf("stdout is not readable")
	}

	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}

	// Check if it's a gzip file
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".gz" {
		gr, err := gzip.NewReader(file)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		// Return a reader that closes both gzip reader and file
		return &multiCloser{
			closers: []io.Closer{gr, file},
			reader:  gr,
		}, nil
	}

	return file, nil
}

// ReadAll reads all data from a reader
func ReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

// GetSize returns the size of a file or total size of all files in a directory
func GetSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed to stat %s: %w", path, err)
	}

	if !info.IsDir() {
		return info.Size(), nil
	}

	// For directories, calculate total size recursively
	var totalSize int64
	err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to walk directory %s: %w", path, err)
	}

	return totalSize, nil
}

// GetExtension returns the lowercase extension of a file, stripping .gz if present
func GetExtension(filename string) string {
	filename = strings.ToLower(filename)

	// Strip .gz extension if present
	filename = strings.TrimSuffix(filename, ".gz")

	ext := filepath.Ext(filename)
	if ext == "" {
		return ""
	}

	// Remove the leading dot
	return ext[1:]
}

// BinsFromCue parses a CUE file and returns the paths to BIN files
func BinsFromCue(cuePath string) ([]string, error) {
	if GetExtension(cuePath) != "cue" {
		return nil, fmt.Errorf("not a CUE file: %s", cuePath)
	}

	file, err := OpenFile(cuePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CUE file: %w", err)
	}
	defer file.Close()

	var bins []string
	cueDir := filepath.Dir(cuePath)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(strings.ToUpper(line), "FILE") {
			// Extract filename from FILE line
			// Format: FILE "filename" BINARY
			parts := strings.Split(line, `"`)
			if len(parts) >= 3 {
				binFile := parts[1]
				binPath := filepath.Join(cueDir, binFile)
				bins = append(bins, binPath)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading CUE file: %w", err)
	}

	return bins, nil
}

// CheckExists checks if a file exists and returns an error if it doesn't
func CheckExists(path string) error {
	// Special case for /dev/ paths
	if strings.HasPrefix(strings.ToLower(path), "/dev/") {
		return nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file/folder not found: %s", path)
	}
	return nil
}

// CheckNotExists checks if a file doesn't exist and returns an error if it does
func CheckNotExists(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file/folder exists: %s", path)
	}
	return nil
}

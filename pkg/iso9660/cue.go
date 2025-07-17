package iso9660

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/wizzomafizzo/go-gameid/pkg/fileio"
)

// multiFileReader combines multiple files into a single reader
type multiFileReader struct {
	files   []*os.File
	sizes   []int64
	current int
	offset  int64
}

func newMultiFileReader(paths []string) (*multiFileReader, int64, error) {
	reader := &multiFileReader{
		files: make([]*os.File, 0, len(paths)),
		sizes: make([]int64, 0, len(paths)),
	}

	var totalSize int64

	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			reader.Close()
			return nil, 0, fmt.Errorf("failed to open %s: %w", path, err)
		}

		stat, err := file.Stat()
		if err != nil {
			file.Close()
			reader.Close()
			return nil, 0, fmt.Errorf("failed to stat %s: %w", path, err)
		}

		reader.files = append(reader.files, file)
		reader.sizes = append(reader.sizes, stat.Size())
		totalSize += stat.Size()
	}

	return reader, totalSize, nil
}

func (r *multiFileReader) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, fmt.Errorf("negative offset")
	}

	// Find which file contains this offset
	fileOffset := off
	fileIndex := 0

	for i, size := range r.sizes {
		if fileOffset < size {
			fileIndex = i
			break
		}
		fileOffset -= size
	}

	if fileIndex >= len(r.files) {
		return 0, io.EOF
	}

	// Read from the current file
	bytesRead := 0
	for len(p) > 0 && fileIndex < len(r.files) {
		n, err := r.files[fileIndex].ReadAt(p, fileOffset)
		bytesRead += n

		if err == io.EOF {
			// Move to next file
			fileIndex++
			fileOffset = 0
			p = p[n:]

			if fileIndex >= len(r.files) {
				if bytesRead == 0 {
					return 0, io.EOF
				}
				return bytesRead, nil
			}
		} else if err != nil {
			return bytesRead, err
		} else {
			return bytesRead, nil
		}
	}

	return bytesRead, nil
}

func (r *multiFileReader) Close() error {
	var firstErr error
	for _, f := range r.files {
		if err := f.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// OpenFile opens an ISO 9660 image from a file path
// Supports regular ISO files, gzipped ISOs, and CUE/BIN files
func OpenFile(path string) (*ISO9660, error) {
	// Handle CUE files
	if strings.ToLower(strings.TrimSpace(path)) == "-" {
		return nil, fmt.Errorf("stdin not supported for ISO files")
	}

	if strings.HasSuffix(strings.ToLower(path), ".cue") {
		bins, err := fileio.BinsFromCue(path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse CUE file: %w", err)
		}

		if len(bins) == 0 {
			return nil, fmt.Errorf("no BIN files found in CUE")
		}

		reader, size, err := newMultiFileReader(bins)
		if err != nil {
			return nil, err
		}

		iso, err := Open(reader, size)
		if err != nil {
			reader.Close()
			return nil, err
		}

		// Store the reader so it can be closed later
		iso.closer = reader
		return iso, nil
	}

	// Regular file
	size, err := fileio.GetSize(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get file size: %w", err)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	iso, err := Open(file, size)
	if err != nil {
		file.Close()
		return nil, err
	}

	iso.closer = file
	return iso, nil
}

// Close closes the ISO and any underlying files
func (iso *ISO9660) Close() error {
	if iso.closer != nil {
		return iso.closer.Close()
	}
	return nil
}

package iso9660

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MountedDisc represents a mounted disc directory that provides an ISO9660-like interface
type MountedDisc struct {
	path         string
	uuid         string
	volumeID     string
	PVD          *PrimaryVolumeDescriptor
	SectorSize   int
	SectorOffset int
}

// OpenMountedDisc opens a directory as a mounted disc
func OpenMountedDisc(path, uuid, volumeID string) (*MountedDisc, error) {
	// Verify the path is a directory
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path must be a directory: %s", path)
	}

	// Clean the path
	path = filepath.Clean(path)

	// Create a fake PVD with the provided metadata
	pvd := &PrimaryVolumeDescriptor{
		VolumeID:         volumeID,
		CreationDateTime: uuid,
		// Other fields left empty as they're not available from a directory
	}

	return &MountedDisc{
		path:         path,
		uuid:         uuid,
		volumeID:     volumeID,
		PVD:          pvd,
		SectorSize:   SectorSize2048, // Default for mounted discs
		SectorOffset: 0,
	}, nil
}

// ListFiles returns a list of files in the mounted disc directory
func (m *MountedDisc) ListFiles(onlyRootDir bool) ([]FileEntry, error) {
	var files []FileEntry

	if onlyRootDir {
		// List only files in the root directory
		entries, err := os.ReadDir(m.path)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				info, err := entry.Info()
				if err != nil {
					continue
				}
				files = append(files, FileEntry{
					Name: "/" + entry.Name(),
					LBA:  0, // Not applicable for mounted discs
					Size: uint32(info.Size()),
				})
			}
		}
	} else {
		// Walk the entire directory tree
		err := filepath.Walk(m.path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files with errors
			}
			if !info.IsDir() {
				// Get relative path from the mount point
				relPath, err := filepath.Rel(m.path, path)
				if err != nil {
					return nil
				}
				// Convert to ISO-style path with forward slashes and leading slash
				isoPath := "/" + strings.ReplaceAll(relPath, string(filepath.Separator), "/")
				files = append(files, FileEntry{
					Name: isoPath,
					LBA:  0, // Not applicable for mounted discs
					Size: uint32(info.Size()),
				})
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk directory: %w", err)
		}
	}

	// Sort files by name to match ISO behavior
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return files, nil
}

// ReadFile reads a file from the mounted disc directory
func (m *MountedDisc) ReadFile(lba, size uint32) ([]byte, error) {
	// For mounted discs, we can't use LBA, so this method shouldn't be used
	return nil, fmt.Errorf("ReadFile with LBA not supported for mounted discs")
}

// ReadFileByName reads a file by its path from the mounted disc
func (m *MountedDisc) ReadFileByName(name string) ([]byte, error) {
	// Remove leading slash if present
	name = strings.TrimPrefix(name, "/")

	// Construct full path
	fullPath := filepath.Join(m.path, name)

	// CRITICAL SECURITY FIX: Validate that the resolved path is still within the mounted directory.
	// This prevents path traversal attacks using ".." or absolute paths.
	cleanedFullPath := filepath.Clean(fullPath)
	// Ensure the cleaned path is still a sub-path of the base mounted directory.
	// Using HasPrefix is generally sufficient if m.path is already cleaned and absolute.
	// For robustness, consider checking if the path is canonicalized to be within m.path.
	if !strings.HasPrefix(cleanedFullPath, m.path) {
		return nil, fmt.Errorf("attempted path traversal: %s is outside mounted directory %s", cleanedFullPath, m.path)
	}

	// Read the file
	data, err := os.ReadFile(cleanedFullPath) // Use cleanedFullPath
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", name, err)
	}

	return data, nil
}

// Close is a no-op for mounted discs
func (m *MountedDisc) Close() error {
	return nil
}

// Reader returns a reader for the entire disc (not supported for mounted discs)
func (m *MountedDisc) Reader() io.ReaderAt {
	return nil
}

// ReadFileByEntry reads a file using its FileEntry
func (m *MountedDisc) ReadFileByEntry(entry *FileEntry) ([]byte, error) {
	return m.ReadFileByName(entry.Name)
}

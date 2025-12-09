package iso9660

import (
	"os"
	"path/filepath"
	"strings"
)

// MountedDisc represents a mounted disc directory.
// This allows treating a directory as if it were an ISO9660 filesystem.
type MountedDisc struct {
	path     string
	uuid     string
	volumeID string
}

// OpenMounted creates a MountedDisc from a directory path.
func OpenMounted(path string, uuid, volumeID string) (*MountedDisc, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, ErrInvalidISO
	}

	disc := &MountedDisc{
		path: strings.TrimSuffix(absPath, "/"),
		uuid: uuid,
	}

	if volumeID != "" {
		disc.volumeID = volumeID
	} else {
		// Use directory name as volume ID
		disc.volumeID = filepath.Base(absPath)
	}

	return disc, nil
}

// Close is a no-op for mounted discs.
func (m *MountedDisc) Close() error {
	return nil
}

// GetSystemID returns nil for mounted discs.
func (m *MountedDisc) GetSystemID() string {
	return ""
}

// GetVolumeID returns the volume ID.
func (m *MountedDisc) GetVolumeID() string {
	return m.volumeID
}

// GetPublisherID returns nil for mounted discs.
func (m *MountedDisc) GetPublisherID() string {
	return ""
}

// GetDataPreparerID returns nil for mounted discs.
func (m *MountedDisc) GetDataPreparerID() string {
	return ""
}

// GetUUID returns the UUID if set.
func (m *MountedDisc) GetUUID() string {
	return m.uuid
}

// IterFiles returns a list of files in the mounted directory.
func (m *MountedDisc) IterFiles(onlyRootDir bool) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(m.path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			// If only root dir, don't descend into subdirectories
			if onlyRootDir && path != m.path {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(m.path, path)
		if err != nil {
			return nil
		}

		// Convert to forward slashes and add leading slash
		relPath = "/" + strings.ReplaceAll(relPath, "\\", "/")

		files = append(files, FileInfo{
			Path: relPath,
			Size: uint32(info.Size()),
		})

		return nil
	})

	return files, err
}

// ReadFile reads a file from the mounted directory.
func (m *MountedDisc) ReadFile(info FileInfo) ([]byte, error) {
	// Remove leading slash and join with base path
	relPath := strings.TrimPrefix(info.Path, "/")
	fullPath := filepath.Join(m.path, relPath)
	return os.ReadFile(fullPath)
}

// ReadFileByPath reads a file by its path.
func (m *MountedDisc) ReadFileByPath(path string) ([]byte, error) {
	// Remove leading slash
	relPath := strings.TrimPrefix(path, "/")
	fullPath := filepath.Join(m.path, relPath)
	return os.ReadFile(fullPath)
}

// FileExists checks if a file exists at the given path.
func (m *MountedDisc) FileExists(path string) bool {
	relPath := strings.TrimPrefix(path, "/")
	fullPath := filepath.Join(m.path, relPath)
	info, err := os.Stat(fullPath)
	return err == nil && !info.IsDir()
}

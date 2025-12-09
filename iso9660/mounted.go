// Copyright (c) 2025 Niema Moshiri and The Zaparoo Project.
// SPDX-License-Identifier: GPL-3.0-or-later
//
// This file is part of go-gameid.
//
// go-gameid is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-gameid is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-gameid.  If not, see <https://www.gnu.org/licenses/>.

package iso9660

import (
	"fmt"
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
func OpenMounted(path, uuid, volumeID string) (*MountedDisc, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("get absolute path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
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
func (*MountedDisc) Close() error {
	return nil
}

// GetSystemID returns empty string for mounted discs.
func (*MountedDisc) GetSystemID() string {
	return ""
}

// GetVolumeID returns the volume ID.
func (m *MountedDisc) GetVolumeID() string {
	return m.volumeID
}

// GetPublisherID returns empty string for mounted discs.
func (*MountedDisc) GetPublisherID() string {
	return ""
}

// GetDataPreparerID returns empty string for mounted discs.
func (*MountedDisc) GetDataPreparerID() string {
	return ""
}

// GetUUID returns the UUID if set.
func (m *MountedDisc) GetUUID() string {
	return m.uuid
}

// IterFiles returns a list of files in the mounted directory.
//
//nolint:revive // onlyRootDir flag parameter is intentional API design matching ISO9660 interface
func (m *MountedDisc) IterFiles(onlyRootDir bool) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(m.path, func(filePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil //nolint:nilerr // Intentionally skip errors to continue walking
		}

		if info.IsDir() {
			// If only root dir, don't descend into subdirectories
			if onlyRootDir && filePath != m.path {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path
		relPath, relErr := filepath.Rel(m.path, filePath)
		if relErr != nil {
			return nil //nolint:nilerr // Intentionally skip errors to continue walking
		}

		// Convert to forward slashes and add leading slash
		relPath = "/" + strings.ReplaceAll(relPath, "\\", "/")

		files = append(files, FileInfo{
			Path: relPath,
			Size: uint32(info.Size()), //nolint:gosec // File size overflow unlikely for game files
		})

		return nil
	})
	if err != nil {
		return files, fmt.Errorf("walk directory: %w", err)
	}
	return files, nil
}

// ReadFile reads a file from the mounted directory.
func (m *MountedDisc) ReadFile(info FileInfo) ([]byte, error) {
	// Remove leading slash and join with base path
	relPath := strings.TrimPrefix(info.Path, "/")
	fullPath := filepath.Join(m.path, relPath)
	data, err := os.ReadFile(fullPath) //nolint:gosec // Path constructed from mounted disc path
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", info.Path, err)
	}
	return data, nil
}

// ReadFileByPath reads a file by its path.
func (m *MountedDisc) ReadFileByPath(path string) ([]byte, error) {
	// Remove leading slash
	relPath := strings.TrimPrefix(path, "/")
	fullPath := filepath.Join(m.path, relPath)
	data, err := os.ReadFile(fullPath) //nolint:gosec // Path constructed from mounted disc path
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}
	return data, nil
}

// FileExists checks if a file exists at the given path.
func (m *MountedDisc) FileExists(path string) bool {
	relPath := strings.TrimPrefix(path, "/")
	fullPath := filepath.Join(m.path, relPath)
	info, err := os.Stat(fullPath)
	return err == nil && !info.IsDir()
}

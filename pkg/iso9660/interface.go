package iso9660

import (
	"os"
)

// DiscImage is the common interface for ISO9660 images and mounted disc directories
type DiscImage interface {
	// ListFiles returns a list of files in the disc
	ListFiles(onlyRootDir bool) ([]FileEntry, error)

	// ReadFile reads a file by LBA and size (may not be supported by all implementations)
	ReadFile(lba, size uint32) ([]byte, error)

	// ReadFileByEntry reads a file using its FileEntry
	ReadFileByEntry(entry *FileEntry) ([]byte, error)

	// Close closes any underlying resources
	Close() error

	// GetPVD returns the Primary Volume Descriptor (may be synthetic for mounted discs)
	GetPVD() *PrimaryVolumeDescriptor
}

// Ensure our types implement the interface
var _ DiscImage = (*ISO9660)(nil)
var _ DiscImage = (*MountedDisc)(nil)

// GetPVD returns the Primary Volume Descriptor for ISO9660
func (iso *ISO9660) GetPVD() *PrimaryVolumeDescriptor {
	return iso.PVD
}

// GetPVD returns the Primary Volume Descriptor for MountedDisc
func (m *MountedDisc) GetPVD() *PrimaryVolumeDescriptor {
	return m.PVD
}

// OpenImage opens either an ISO file or a mounted disc directory
func OpenImage(path, discUUID, discLabel string) (DiscImage, error) {
	// Check if path is a directory
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		// It's a directory, open as mounted disc
		return OpenMountedDisc(path, discUUID, discLabel)
	}

	// Otherwise, try to open as ISO file
	iso, err := OpenFile(path)
	if err != nil {
		return nil, err
	}

	// If disc_uuid or disc_label are provided for an ISO file, override the extracted values
	if iso.PVD != nil {
		if discUUID != "" {
			iso.PVD.CreationDateTime = discUUID
		}
		if discLabel != "" {
			iso.PVD.VolumeID = discLabel
		}
	}

	return iso, nil
}

// Copyright (c) 2025 Niema Moshiri and The Zaparoo Project.
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build windows

package gameid

// isBlockDevice checks if the given path is a block device.
// On Windows, block devices (like CD/DVD drives) are accessed differently,
// typically via drive letters (e.g., D:\) rather than /dev/ paths.
// This function returns false on Windows as the /dev/ path check doesn't apply.
func isBlockDevice(_ string) bool {
	// Windows doesn't use /dev/ paths for block devices
	// Physical drives are accessed via \\.\PhysicalDriveN or drive letters
	return false
}

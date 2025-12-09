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

//go:build unix

package gameid

import (
	"os"
	"strings"
	"syscall"
)

// isBlockDevice checks if the given path is a block device (e.g., /dev/sr0).
func isBlockDevice(path string) bool {
	// On Unix, block devices are typically in /dev/
	if !strings.HasPrefix(path, "/dev/") {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	// Check if it's a block device using syscall
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	// S_IFBLK = block device (0x6000 = 0o60000)
	return stat.Mode&syscall.S_IFMT == syscall.S_IFBLK
}

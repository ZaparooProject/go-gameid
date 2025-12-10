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

package archive

import "fmt"

// FormatError indicates an unsupported or invalid archive format.
type FormatError struct {
	Format string
	Reason string
}

func (e FormatError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("unsupported archive format %s: %s", e.Format, e.Reason)
	}
	return fmt.Sprintf("unsupported archive format: %s", e.Format)
}

// FileNotFoundError indicates a file was not found in the archive.
type FileNotFoundError struct {
	Archive      string
	InternalPath string
}

func (e FileNotFoundError) Error() string {
	return fmt.Sprintf("file %q not found in archive %q", e.InternalPath, e.Archive)
}

// NoGameFilesError indicates no game files were found in the archive.
type NoGameFilesError struct {
	Archive string
}

func (e NoGameFilesError) Error() string {
	return fmt.Sprintf("no game files found in archive %q", e.Archive)
}

// DiscNotSupportedError indicates disc-based games in archives are not supported.
type DiscNotSupportedError struct {
	Console string
}

func (e DiscNotSupportedError) Error() string {
	return fmt.Sprintf("disc-based games (%s) in archives are not supported", e.Console)
}

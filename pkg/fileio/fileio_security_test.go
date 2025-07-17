package fileio

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestOpenFile_PathTraversal tests protection against path traversal attacks
func TestOpenFile_PathTraversal(t *testing.T) {
	// create a test file in temp directory
	tmpDir := t.TempDir()
	safeFile := filepath.Join(tmpDir, "safe.txt")
	_ = os.WriteFile(safeFile, []byte("safe content"), 0644)

	// create a sensitive file to ensure we don't access it
	sensitiveDir := filepath.Join(tmpDir, "sensitive")
	_ = os.MkdirAll(sensitiveDir, 0755)
	sensitiveFile := filepath.Join(sensitiveDir, "secret.txt")
	_ = os.WriteFile(sensitiveFile, []byte("secret content"), 0644)

	tests := []struct {
		name      string
		path      string
		shouldErr bool
	}{
		{
			name:      "Absolute path traversal to /etc/passwd",
			path:      "/etc/passwd",
			shouldErr: false, // currently no validation, but should fail in secure version
		},
		{
			name:      "Relative path traversal with ..",
			path:      "../../../../../../../etc/passwd",
			shouldErr: false, // currently no validation
		},
		{
			name:      "Path with embedded null bytes",
			path:      "file\x00.txt",
			shouldErr: true,
		},
		{
			name:      "Windows-style path traversal",
			path:      "..\\..\\..\\windows\\system32\\config\\sam",
			shouldErr: runtime.GOOS != "windows",
		},
		{
			name:      "Double-encoded path traversal",
			path:      "%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
			shouldErr: true, // will be treated as literal filename and fail
		},
		{
			name:      "Unicode path traversal attempt",
			path:      "․․/․․/․․/etc/passwd", // uses unicode dots
			shouldErr: true,                  // treated as literal and fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := OpenFile(tt.path)
			if err != nil && !tt.shouldErr {
				t.Errorf("OpenFile() error = %v, shouldErr = %v", err, tt.shouldErr)
			}
			if err == nil {
				reader.Close()
				if tt.shouldErr {
					t.Errorf("OpenFile() succeeded but should have failed for path: %s", tt.path)
				}
			}
		})
	}
}

// TestOpenFile_SymlinkAttacks tests protection against symlink-based attacks
func TestOpenFile_SymlinkAttacks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink tests on Windows")
	}

	tmpDir := t.TempDir()

	// create a safe file
	safeFile := filepath.Join(tmpDir, "safe.txt")
	_ = os.WriteFile(safeFile, []byte("safe content"), 0644)

	// create symlink to /etc/passwd
	symlinkToSensitive := filepath.Join(tmpDir, "passwd_link")
	_ = os.Symlink("/etc/passwd", symlinkToSensitive)

	// create circular symlink
	circularLink1 := filepath.Join(tmpDir, "link1")
	circularLink2 := filepath.Join(tmpDir, "link2")
	_ = os.Symlink(circularLink2, circularLink1)
	_ = os.Symlink(circularLink1, circularLink2)

	// create symlink chain
	chainStart := filepath.Join(tmpDir, "chain_start")
	chain1 := filepath.Join(tmpDir, "chain1")
	chain2 := filepath.Join(tmpDir, "chain2")
	chainEnd := filepath.Join(tmpDir, "chain_end.txt")
	_ = os.WriteFile(chainEnd, []byte("chain end"), 0644)
	_ = os.Symlink(chain1, chainStart)
	_ = os.Symlink(chain2, chain1)
	_ = os.Symlink(chainEnd, chain2)

	tests := []struct {
		name      string
		path      string
		shouldErr bool
	}{
		{
			name:      "Direct symlink to sensitive file",
			path:      symlinkToSensitive,
			shouldErr: false, // currently follows symlinks
		},
		{
			name:      "Circular symlink",
			path:      circularLink1,
			shouldErr: true, // should error due to loop
		},
		{
			name:      "Long symlink chain",
			path:      chainStart,
			shouldErr: false, // should work if chain depth is reasonable
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := OpenFile(tt.path)
			if err != nil && !tt.shouldErr {
				t.Errorf("OpenFile() error = %v, shouldErr = %v", err, tt.shouldErr)
			}
			if reader != nil {
				reader.Close()
			}
		})
	}
}

// TestReadAll_SizeLimits tests protection against memory exhaustion
func TestReadAll_SizeLimits(t *testing.T) {
	t.Skip("Skipping size limit tests - would require actual implementation of limits")

	// TODO: When size limits are implemented, test:
	// 1. Reading file exactly at limit (should succeed)
	// 2. Reading file 1 byte over limit (should fail)
	// 3. Reading from /dev/zero with limit (should fail after limit)
}

// TestGetSize_ResourceExhaustion tests protection against DoS via deep directory traversal
func TestGetSize_ResourceExhaustion(t *testing.T) {
	tmpDir := t.TempDir()

	// create a deeply nested directory structure
	deepPath := tmpDir
	for i := 0; i < 100; i++ {
		deepPath = filepath.Join(deepPath, "level")
		_ = os.MkdirAll(deepPath, 0755)
	}

	// create large number of files
	manyFilesDir := filepath.Join(tmpDir, "many_files")
	_ = os.MkdirAll(manyFilesDir, 0755)
	for i := 0; i < 1000; i++ {
		filename := filepath.Join(manyFilesDir, fmt.Sprintf("file_%d.txt", i))
		_ = os.WriteFile(filename, []byte("content"), 0644)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Deep directory structure",
			path:    tmpDir,
			wantErr: false, // currently no depth limit
		},
		{
			name:    "Directory with many files",
			path:    manyFilesDir,
			wantErr: false, // currently no file count limit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetSize(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestBinsFromCue_PathInjection tests CUE file path injection vulnerabilities
func TestBinsFromCue_PathInjection(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		cueContent  string
		expectPaths []string
		shouldErr   bool
	}{
		{
			name: "Path traversal in FILE entry",
			cueContent: `FILE "../../../etc/passwd" BINARY
  TRACK 01 MODE2/2352
    INDEX 01 00:00:00`,
			expectPaths: []string{filepath.Join(tmpDir, "../../../etc/passwd")},
			shouldErr:   false, // currently no validation
		},
		{
			name: "Absolute path in FILE entry",
			cueContent: `FILE "/etc/passwd" BINARY
  TRACK 01 MODE2/2352
    INDEX 01 00:00:00`,
			expectPaths: []string{filepath.Join(tmpDir, "/etc/passwd")}, // joined with base dir
			shouldErr:   false,                                          // currently allows absolute paths
		},
		{
			name: "Windows UNC path injection",
			cueContent: `FILE "\\\\evil-server\\share\\malware.bin" BINARY
  TRACK 01 MODE2/2352
    INDEX 01 00:00:00`,
			expectPaths: []string{filepath.Join(tmpDir, "\\\\\\\\evil-server\\\\share\\\\malware.bin")}, // escaped
			shouldErr:   false,
		},
		{
			name: "Path with null bytes",
			cueContent: `FILE "safe.bin\x00../../etc/passwd" BINARY
  TRACK 01 MODE2/2352
    INDEX 01 00:00:00`,
			expectPaths: []string{filepath.Join(tmpDir, "safe.bin\x00../../etc/passwd")},
			shouldErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cueFile := filepath.Join(tmpDir, "test.cue")
			err := os.WriteFile(cueFile, []byte(tt.cueContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write CUE file: %v", err)
			}

			bins, err := BinsFromCue(cueFile)
			if (err != nil) != tt.shouldErr {
				t.Errorf("BinsFromCue() error = %v, shouldErr %v", err, tt.shouldErr)
			}

			if !tt.shouldErr && len(bins) != len(tt.expectPaths) {
				t.Errorf("BinsFromCue() returned %d paths, expected %d", len(bins), len(tt.expectPaths))
			}

			for i, bin := range bins {
				if i < len(tt.expectPaths) && bin != tt.expectPaths[i] {
					t.Errorf("BinsFromCue()[%d] = %q, expected %q", i, bin, tt.expectPaths[i])
				}
			}
		})
	}
}

// TestOpenFile_SpecialFiles tests handling of special files that could cause issues
func TestOpenFile_SpecialFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping special file tests on Windows")
	}

	tests := []struct {
		name      string
		path      string
		shouldErr bool
	}{
		{
			name:      "/dev/null",
			path:      "/dev/null",
			shouldErr: false,
		},
		{
			name:      "/dev/zero",
			path:      "/dev/zero",
			shouldErr: false, // dangerous without size limits
		},
		{
			name:      "/dev/random",
			path:      "/dev/random",
			shouldErr: false,
		},
		{
			name:      "/dev/urandom",
			path:      "/dev/urandom",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// check if special file exists
			if _, err := os.Stat(tt.path); os.IsNotExist(err) {
				t.Skipf("Special file %s does not exist", tt.path)
			}

			reader, err := OpenFile(tt.path)
			if (err != nil) != tt.shouldErr {
				t.Errorf("OpenFile() error = %v, shouldErr %v", err, tt.shouldErr)
			}
			if reader != nil {
				reader.Close()
			}
		})
	}
}

// TestCheckExists_SecurityBoundaries tests CheckExists doesn't leak information
func TestCheckExists_SecurityBoundaries(t *testing.T) {
	// test that errors don't reveal too much about file system structure
	tests := []struct {
		name         string
		path         string
		checkErrText string
	}{
		{
			name:         "Sensitive file path",
			path:         "/etc/shadow",
			checkErrText: "file/folder not found", // should give generic error
		},
		{
			name:         "Path with credentials",
			path:         "/home/user/.ssh/id_rsa",
			checkErrText: "file/folder not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckExists(tt.path)
			if err != nil && !strings.Contains(err.Error(), tt.checkErrText) {
				t.Errorf("CheckExists() error text = %q, should contain %q", err.Error(), tt.checkErrText)
			}
		})
	}
}

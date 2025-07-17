package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLIVersion tests the version flag
func TestCLIVersion(t *testing.T) {
	// build the binary
	binPath := filepath.Join(t.TempDir(), "gameid")
	cmd := exec.Command("go", "build", "-o", binPath, "github.com/wizzomafizzo/go-gameid/cmd/gameid")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// run with --version flag
	cmd = exec.Command(binPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run version command: %v", err)
	}

	// check output contains version
	outputStr := string(output)
	if !strings.Contains(outputStr, "GameID v") {
		t.Errorf("Version output incorrect: %s", outputStr)
	}
}

// TestCLIHelp tests the help output
func TestCLIHelp(t *testing.T) {
	// build the binary
	binPath := filepath.Join(t.TempDir(), "gameid")
	cmd := exec.Command("go", "build", "-o", binPath, "github.com/wizzomafizzo/go-gameid/cmd/gameid")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// run with -h flag
	cmd = exec.Command(binPath, "-h")
	output, err := cmd.CombinedOutput()
	// help returns exit code 2, which is expected
	if err != nil && err.(*exec.ExitError).ExitCode() != 2 {
		t.Fatalf("Failed to run help command: %v", err)
	}

	// check output contains usage information
	outputStr := string(output)
	expectedFlags := []string{"-i", "-input", "-c", "-console", "-d", "-database"}
	for _, flag := range expectedFlags {
		if !strings.Contains(outputStr, flag) {
			t.Errorf("Help output missing flag %s: %s", flag, outputStr)
		}
	}
}

// TestCLIMissingArgs tests error handling for missing arguments
func TestCLIMissingArgs(t *testing.T) {
	// build the binary
	binPath := filepath.Join(t.TempDir(), "gameid")
	cmd := exec.Command("go", "build", "-o", binPath, "github.com/wizzomafizzo/go-gameid/cmd/gameid")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	tests := []struct {
		name string
		args []string
	}{
		{"missing all args", []string{}},
		{"missing console", []string{"-i", "test.gba"}},
		{"missing input", []string{"-c", "GBA"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd = exec.Command(binPath, tt.args...)
			err := cmd.Run()
			if err == nil {
				t.Error("Expected error for missing arguments, got nil")
			}
		})
	}
}

// TestCLIInvalidConsole tests error handling for invalid console
func TestCLIInvalidConsole(t *testing.T) {
	// build the binary
	binPath := filepath.Join(t.TempDir(), "gameid")
	cmd := exec.Command("go", "build", "-o", binPath, "github.com/wizzomafizzo/go-gameid/cmd/gameid")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// create a dummy file
	testFile := filepath.Join(t.TempDir(), "test.rom")
	if err := os.WriteFile(testFile, []byte("dummy"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// run with invalid console
	cmd = exec.Command(binPath, "-i", testFile, "-c", "INVALID")
	err := cmd.Run()
	if err == nil {
		t.Error("Expected error for invalid console, got nil")
	}
}

// TestCLIFileNotFound tests error handling for non-existent file
func TestCLIFileNotFound(t *testing.T) {
	// build the binary
	binPath := filepath.Join(t.TempDir(), "gameid")
	cmd := exec.Command("go", "build", "-o", binPath, "github.com/wizzomafizzo/go-gameid/cmd/gameid")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// run with non-existent file
	cmd = exec.Command(binPath, "-i", "/nonexistent/file.gba", "-c", "GBA")
	err := cmd.Run()
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// TestCLIOutputToFile tests output redirection to file
func TestCLIOutputToFile(t *testing.T) {
	// build the binary
	binPath := filepath.Join(t.TempDir(), "gameid")
	cmd := exec.Command("go", "build", "-o", binPath, "github.com/wizzomafizzo/go-gameid/cmd/gameid")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// create a minimal GBA ROM with valid Nintendo logo
	testFile := filepath.Join(t.TempDir(), "test.gba")
	rom := make([]byte, 192)
	// copy Nintendo logo to correct position (0x04-0x9F)
	logo := []byte{
		0x24, 0xFF, 0xAE, 0x51, 0x69, 0x9A, 0xA2, 0x21, 0x3D, 0x84, 0x82, 0x0A,
		0x84, 0xE4, 0x09, 0xAD, 0x11, 0x24, 0x8B, 0x98, 0xC0, 0x81, 0x7F, 0x21,
		0xA3, 0x52, 0xBE, 0x19, 0x93, 0x09, 0xCE, 0x20, 0x10, 0x46, 0x4A, 0x4A,
		0xF8, 0x27, 0x31, 0xEC, 0x58, 0xC7, 0xE8, 0x33, 0x82, 0xE3, 0xCE, 0xBF,
		0x85, 0xF4, 0xDF, 0x94, 0xCE, 0x4B, 0x09, 0xC1, 0x94, 0x56, 0x8A, 0xC0,
		0x13, 0x72, 0xA7, 0xFC, 0x9F, 0x84, 0x4D, 0x73, 0xA3, 0xCA, 0x9A, 0x61,
		0x58, 0x97, 0xA3, 0x27, 0xFC, 0x03, 0x98, 0x76, 0x23, 0x1D, 0xC7, 0x61,
		0x03, 0x04, 0xAE, 0x56, 0xBF, 0x38, 0x84, 0x00, 0x40, 0xA7, 0x0E, 0xFD,
		0xFF, 0x52, 0xFE, 0x03, 0x6F, 0x95, 0x30, 0xF1, 0x97, 0xFB, 0xC0, 0x85,
		0x60, 0xD6, 0x80, 0x25, 0xA9, 0x63, 0xBE, 0x03, 0x01, 0x4E, 0x38, 0xE2,
		0xF9, 0xA2, 0x34, 0xFF, 0xBB, 0x3E, 0x03, 0x44, 0x78, 0x00, 0x90, 0xCB,
		0x88, 0x11, 0x3A, 0x94, 0x65, 0xC0, 0x7C, 0x63, 0x87, 0xF0, 0x3C, 0xAF,
		0xD6, 0x25, 0xE4, 0x8B, 0x38, 0x0A, 0xAC, 0x72, 0x21, 0xD4, 0xF8, 0x07,
	}
	copy(rom[0x04:], logo)
	// set some header fields
	copy(rom[0xA0:], []byte("TEST GAME   ")) // title
	copy(rom[0xAC:], []byte("TEST"))         // game code
	copy(rom[0xB0:], []byte("01"))           // maker code
	rom[0xBC] = 1                            // software version

	if err := os.WriteFile(testFile, rom, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// run with output file
	outputFile := filepath.Join(t.TempDir(), "output.txt")
	cmd = exec.Command(binPath, "-i", testFile, "-c", "GBA", "-o", outputFile)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to run command: %v", err)
	}

	// check output file exists and contains data
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Output file is empty")
	}

	// check output contains expected fields
	output := string(data)
	expectedFields := []string{"ID", "internal_title", "maker_code"}
	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Output missing field %s: %s", field, output)
		}
	}
}

// TestCLIDelimiter tests custom delimiter option
func TestCLIDelimiter(t *testing.T) {
	// build the binary
	binPath := filepath.Join(t.TempDir(), "gameid")
	cmd := exec.Command("go", "build", "-o", binPath, "github.com/wizzomafizzo/go-gameid/cmd/gameid")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// create a minimal GBA ROM
	testFile := filepath.Join(t.TempDir(), "test.gba")
	rom := make([]byte, 192)
	// copy Nintendo logo
	logo := []byte{
		0x24, 0xFF, 0xAE, 0x51, 0x69, 0x9A, 0xA2, 0x21, 0x3D, 0x84, 0x82, 0x0A,
		0x84, 0xE4, 0x09, 0xAD, 0x11, 0x24, 0x8B, 0x98, 0xC0, 0x81, 0x7F, 0x21,
		0xA3, 0x52, 0xBE, 0x19, 0x93, 0x09, 0xCE, 0x20, 0x10, 0x46, 0x4A, 0x4A,
		0xF8, 0x27, 0x31, 0xEC, 0x58, 0xC7, 0xE8, 0x33, 0x82, 0xE3, 0xCE, 0xBF,
		0x85, 0xF4, 0xDF, 0x94, 0xCE, 0x4B, 0x09, 0xC1, 0x94, 0x56, 0x8A, 0xC0,
		0x13, 0x72, 0xA7, 0xFC, 0x9F, 0x84, 0x4D, 0x73, 0xA3, 0xCA, 0x9A, 0x61,
		0x58, 0x97, 0xA3, 0x27, 0xFC, 0x03, 0x98, 0x76, 0x23, 0x1D, 0xC7, 0x61,
		0x03, 0x04, 0xAE, 0x56, 0xBF, 0x38, 0x84, 0x00, 0x40, 0xA7, 0x0E, 0xFD,
		0xFF, 0x52, 0xFE, 0x03, 0x6F, 0x95, 0x30, 0xF1, 0x97, 0xFB, 0xC0, 0x85,
		0x60, 0xD6, 0x80, 0x25, 0xA9, 0x63, 0xBE, 0x03, 0x01, 0x4E, 0x38, 0xE2,
		0xF9, 0xA2, 0x34, 0xFF, 0xBB, 0x3E, 0x03, 0x44, 0x78, 0x00, 0x90, 0xCB,
		0x88, 0x11, 0x3A, 0x94, 0x65, 0xC0, 0x7C, 0x63, 0x87, 0xF0, 0x3C, 0xAF,
		0xD6, 0x25, 0xE4, 0x8B, 0x38, 0x0A, 0xAC, 0x72, 0x21, 0xD4, 0xF8, 0x07,
	}
	copy(rom[0x04:], logo)
	copy(rom[0xA0:], []byte("TEST"))
	copy(rom[0xAC:], []byte("TEST"))
	copy(rom[0xB0:], []byte("01"))

	if err := os.WriteFile(testFile, rom, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// run with custom delimiter
	cmd = exec.Command(binPath, "-i", testFile, "-c", "GBA", "--delimiter", "|")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run command: %v\nOutput: %s", err, output)
	}

	// check output uses custom delimiter
	outputStr := string(output)
	if !strings.Contains(outputStr, "|") {
		t.Errorf("Output does not use custom delimiter: %s", outputStr)
	}
	if strings.Contains(outputStr, "\t") {
		t.Errorf("Output still contains tab delimiter: %s", outputStr)
	}
}

// TestInteractiveMode tests the interactive mode
func TestInteractiveMode(t *testing.T) {
	// Skip this test in CI environments where stdin interaction is difficult
	if os.Getenv("CI") != "" {
		t.Skip("Skipping interactive test in CI")
	}

	// build the binary
	binPath := filepath.Join(t.TempDir(), "gameid")
	cmd := exec.Command("go", "build", "-o", binPath, "github.com/wizzomafizzo/go-gameid/cmd/gameid")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// create a test file
	testFile := filepath.Join(t.TempDir(), "test.gba")
	if err := os.WriteFile(testFile, []byte("dummy"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// prepare interactive input
	input := testFile + "\n" + "GBA\n"

	// run in interactive mode
	cmd = exec.Command(binPath)
	cmd.Stdin = strings.NewReader(input)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	output := out.String()

	// the command will fail because it's not a valid GBA file, but we're testing interactive mode
	_ = err

	// check for interactive prompts
	if !strings.Contains(output, "Enter game filename") {
		t.Errorf("Missing filename prompt in output: %s", output)
	}
	if !strings.Contains(output, "Enter console") {
		t.Errorf("Missing console prompt in output: %s", output)
	}
}

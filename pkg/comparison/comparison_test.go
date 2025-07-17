package comparison

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/wizzomafizzo/go-gameid/pkg/database"
	"github.com/wizzomafizzo/go-gameid/pkg/identifiers"
)

// testResult represents the output from either implementation
type testResult struct {
	Console   string            `json:"console"`
	Filepath  string            `json:"filepath"`
	Fields    map[string]string `json:"fields"`
	Error     string            `json:"error,omitempty"`
	Duration  time.Duration     `json:"duration"`
	Timestamp time.Time         `json:"timestamp"`
}

// comparisonReport tracks differences between implementations
type comparisonReport struct {
	TestFile        string            `json:"test_file"`
	Console         string            `json:"console"`
	GoResult        *testResult       `json:"go_result"`
	PythonResult    *testResult       `json:"python_result"`
	Differences     []fieldDifference `json:"differences"`
	MissingInGo     []string          `json:"missing_in_go"`
	MissingInPython []string          `json:"missing_in_python"`
	Passed          bool              `json:"passed"`
}

type fieldDifference struct {
	Field       string `json:"field"`
	GoValue     string `json:"go_value"`
	PythonValue string `json:"python_value"`
}

// testRunner manages comparison tests
type testRunner struct {
	db               *database.GameDatabase
	pythonScriptPath string
	testDataDir      string
	outputDir        string
}

// newTestRunner creates a new comparison test runner
func newTestRunner(t *testing.T) *testRunner {
	// find python script
	pythonScript := "scripts/GameID.py"
	if _, err := os.Stat(pythonScript); os.IsNotExist(err) {
		t.Skip("Python GameID script not found")
	}

	// check if testdata directory exists
	testDataDir := "testdata"
	if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
		t.Skip("testdata directory not found")
	}

	// create output directory
	outputDir := "comparison_results"
	os.MkdirAll(outputDir, 0755)

	// try to load database
	var db *database.GameDatabase
	dbPaths := []string{
		"dbs/gameid_db.json",
		"../../dbs/gameid_db.json",
	}

	for _, dbPath := range dbPaths {
		if _, err := os.Stat(dbPath); err == nil {
			db, _ = database.LoadDatabase(dbPath)
			break
		}
	}

	return &testRunner{
		db:               db,
		pythonScriptPath: pythonScript,
		testDataDir:      testDataDir,
		outputDir:        outputDir,
	}
}

// runGoImplementation runs the Go implementation
func (tr *testRunner) runGoImplementation(console, filepath string) *testResult {
	start := time.Now()
	result := &testResult{
		Console:   console,
		Filepath:  filepath,
		Fields:    make(map[string]string),
		Timestamp: start,
	}

	// get identifier
	identifier := getIdentifier(console, tr.db)
	if identifier == nil {
		result.Error = fmt.Sprintf("Unknown console: %s", console)
		result.Duration = time.Since(start)
		return result
	}

	var fields map[string]string
	var err error

	// Use IdentifyWithOptions if available, otherwise fall back to Identify
	if identWithOpts, ok := identifier.(identifiers.IdentifierWithOptions); ok {
		// For comparison tests, we typically don't have discUUID/discLabel/preferDB from Python.
		// Pass empty strings and false, assuming Python's default behavior or that these
		// parameters are not critical for basic comparison. If Python's comparison script
		// uses these, this part needs to be aligned.
		fields, err = identWithOpts.IdentifyWithOptions(filepath, "", "", false)
	} else {
		// Fallback to the basic Identify method if IdentifierWithOptions is not implemented
		fields, err = identifier.Identify(filepath)
	}

	if err != nil {
		result.Error = fmt.Sprintf("Failed to identify game: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	if fields == nil || len(fields) == 0 {
		result.Error = fmt.Sprintf("%s game not found: %s", console, filepath)
		result.Duration = time.Since(start)
		return result
	}

	// replace empty string values with 'None'
	for k, v := range fields {
		if strings.TrimSpace(v) == "" {
			result.Fields[k] = "None"
		} else {
			result.Fields[k] = v
		}
	}

	result.Duration = time.Since(start)
	return result
}

// runPythonImplementation runs the original Python implementation
func (tr *testRunner) runPythonImplementation(console, filepath string) *testResult {
	start := time.Now()
	result := &testResult{
		Console:   console,
		Filepath:  filepath,
		Fields:    make(map[string]string),
		Timestamp: start,
	}

	// run python script
	cmd := exec.Command("python3", tr.pythonScriptPath,
		"--input", filepath,
		"--console", console,
		"--delimiter", "\t")

	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = fmt.Sprintf("Python script failed: %v, output: %s", err, string(output))
		result.Duration = time.Since(start)
		return result
	}

	// parse output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			result.Fields[parts[0]] = parts[1]
		}
	}

	result.Duration = time.Since(start)
	return result
}

// compareResults compares Go and Python results
func (tr *testRunner) compareResults(goResult, pythonResult *testResult) *comparisonReport {
	report := &comparisonReport{
		TestFile:        goResult.Filepath,
		Console:         goResult.Console,
		GoResult:        goResult,
		PythonResult:    pythonResult,
		Differences:     []fieldDifference{},
		MissingInGo:     []string{},
		MissingInPython: []string{},
		Passed:          true,
	}

	// check for errors
	if goResult.Error != "" || pythonResult.Error != "" {
		report.Passed = false
		return report
	}

	// get all field names
	allFields := make(map[string]bool)
	for k := range goResult.Fields {
		allFields[k] = true
	}
	for k := range pythonResult.Fields {
		allFields[k] = true
	}

	// compare fields
	for field := range allFields {
		goValue, goExists := goResult.Fields[field]
		pythonValue, pythonExists := pythonResult.Fields[field]

		if !goExists {
			report.MissingInGo = append(report.MissingInGo, field)
			report.Passed = false
		} else if !pythonExists {
			report.MissingInPython = append(report.MissingInPython, field)
			report.Passed = false
		} else if goValue != pythonValue {
			// normalize values for comparison
			normalizedGo := normalizeValue(goValue)
			normalizedPython := normalizeValue(pythonValue)

			if normalizedGo != normalizedPython {
				report.Differences = append(report.Differences, fieldDifference{
					Field:       field,
					GoValue:     goValue,
					PythonValue: pythonValue,
				})
				report.Passed = false
			}
		}
	}

	// sort slices for consistent output
	sort.Strings(report.MissingInGo)
	sort.Strings(report.MissingInPython)

	return report
}

// normalizeValue normalizes values for comparison
func normalizeValue(value string) string {
	// handle common differences
	value = strings.TrimSpace(value)

	// normalize hex values
	if strings.HasPrefix(value, "0x") {
		value = strings.ToLower(value)
	}

	// normalize None values
	if value == "" || value == "None" || value == "null" {
		return "None"
	}

	return value
}

// getIdentifier returns the appropriate identifier for the console
func getIdentifier(console string, db *database.GameDatabase) identifiers.Identifier {
	switch console {
	case "GB":
		return identifiers.NewGBIdentifier(db)
	case "GBA":
		return identifiers.NewGBAIdentifier(db)
	case "GC":
		return identifiers.NewGameCubeIdentifier(db)
	case "Genesis":
		return identifiers.NewGenesisIdentifier(db)
	case "N64":
		return identifiers.NewN64Identifier(db)
	case "PSP":
		return identifiers.NewPSPIdentifier(db)
	case "PSX":
		return identifiers.NewPSXIdentifier(db)
	case "PS2":
		return identifiers.NewPS2Identifier(db)
	case "Saturn":
		return identifiers.NewSaturnIdentifier(db)
	case "SegaCD":
		return identifiers.NewSegaCDIdentifier(db)
	case "SNES":
		return identifiers.NewSNESIdentifier(db)
	default:
		return nil
	}
}

// findTestFiles finds all test files in the testdata directory
func (tr *testRunner) findTestFiles() map[string][]string {
	files := make(map[string][]string)

	// walk through testdata directory
	filepath.Walk(tr.testDataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// determine console based on directory structure or file extension
		console := detectConsole(path)
		if console != "" {
			files[console] = append(files[console], path)
		}

		return nil
	})

	return files
}

// detectConsole detects the console type from file path or extension
func detectConsole(filePath string) string {
	// map file extensions to consoles
	extMap := map[string]string{
		".gb":  "GB",
		".gbc": "GB",
		".gba": "GBA",
		".iso": "GC", // could be multiple, we'll handle this specially
		".gcm": "GC",
		".md":  "Genesis",
		".gen": "Genesis",
		".n64": "N64",
		".z64": "N64",
		".v64": "N64",
		".cso": "PSP",
		".bin": "PSX", // could be multiple
		".cue": "PSX",
		".sfc": "SNES",
		".smc": "SNES",
	}

	// check directory names first
	dirs := strings.Split(filePath, string(os.PathSeparator))
	for _, dir := range dirs {
		dir = strings.ToLower(dir)
		switch dir {
		case "gb", "gameboy":
			return "GB"
		case "gba", "gameboy_advance":
			return "GBA"
		case "gc", "gamecube":
			return "GC"
		case "genesis", "megadrive":
			return "Genesis"
		case "n64", "nintendo64":
			return "N64"
		case "psp":
			return "PSP"
		case "psx", "playstation":
			return "PSX"
		case "ps2", "playstation2":
			return "PS2"
		case "saturn":
			return "Saturn"
		case "segacd":
			return "SegaCD"
		case "snes", "super_nintendo":
			return "SNES"
		}
	}

	// check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	return extMap[ext]
}

// TestComprehensiveComparison runs comprehensive comparison tests
func TestComprehensiveComparison(t *testing.T) {
	runner := newTestRunner(t)

	// find all test files
	testFiles := runner.findTestFiles()

	if len(testFiles) == 0 {
		t.Skip("No test files found in testdata directory")
	}

	allReports := []comparisonReport{}
	totalTests := 0
	passedTests := 0

	// run tests for each console
	for console, files := range testFiles {
		t.Run(console, func(t *testing.T) {
			for _, filepath := range files {
				t.Run(filepath, func(t *testing.T) {
					totalTests++

					// run both implementations
					goResult := runner.runGoImplementation(console, filepath)
					pythonResult := runner.runPythonImplementation(console, filepath)

					// compare results
					report := runner.compareResults(goResult, pythonResult)
					allReports = append(allReports, *report)

					if report.Passed {
						passedTests++
					} else {
						t.Errorf("Comparison failed for %s:%s", console, filepath)
						if len(report.Differences) > 0 {
							t.Logf("Field differences:")
							for _, diff := range report.Differences {
								t.Logf("  %s: Go='%s', Python='%s'", diff.Field, diff.GoValue, diff.PythonValue)
							}
						}
						if len(report.MissingInGo) > 0 {
							t.Logf("Missing in Go: %v", report.MissingInGo)
						}
						if len(report.MissingInPython) > 0 {
							t.Logf("Missing in Python: %v", report.MissingInPython)
						}
					}
				})
			}
		})
	}

	// write comprehensive report
	reportFile := filepath.Join(runner.outputDir, "comparison_report.json")
	if reportData, err := json.MarshalIndent(allReports, "", "  "); err == nil {
		ioutil.WriteFile(reportFile, reportData, 0644)
	}

	// write summary
	summary := fmt.Sprintf("Comparison Summary:\n")
	summary += fmt.Sprintf("Total Tests: %d\n", totalTests)
	summary += fmt.Sprintf("Passed: %d\n", passedTests)
	summary += fmt.Sprintf("Failed: %d\n", totalTests-passedTests)
	summary += fmt.Sprintf("Success Rate: %.1f%%\n", float64(passedTests)/float64(totalTests)*100)

	summaryFile := filepath.Join(runner.outputDir, "summary.txt")
	ioutil.WriteFile(summaryFile, []byte(summary), 0644)

	t.Logf(summary)

	if passedTests < totalTests {
		t.Errorf("Some comparison tests failed. Check %s for details.", reportFile)
	}
}

// TestManualComparison allows manual testing of specific files
func TestManualComparison(t *testing.T) {
	// skip by default
	if !testing.Short() {
		t.Skip("Manual comparison test - use -test.short=false to run")
	}

	runner := newTestRunner(t)

	// define manual test cases
	testCases := []struct {
		console  string
		filepath string
	}{
		// add specific test cases here
		// {"GBA", "testdata/gba/pokemon_ruby.gba"},
		// {"SNES", "testdata/snes/super_metroid.sfc"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s-%s", tc.console, filepath.Base(tc.filepath)), func(t *testing.T) {
			// check if file exists
			if _, err := os.Stat(tc.filepath); os.IsNotExist(err) {
				t.Skipf("Test file not found: %s", tc.filepath)
			}

			// run both implementations
			goResult := runner.runGoImplementation(tc.console, tc.filepath)
			pythonResult := runner.runPythonImplementation(tc.console, tc.filepath)

			// compare results
			report := runner.compareResults(goResult, pythonResult)

			// detailed logging
			t.Logf("Go result: %+v", goResult)
			t.Logf("Python result: %+v", pythonResult)

			if !report.Passed {
				t.Errorf("Comparison failed")
				for _, diff := range report.Differences {
					t.Logf("Difference in %s: Go='%s', Python='%s'", diff.Field, diff.GoValue, diff.PythonValue)
				}
			}
		})
	}
}

// benchmarkComparison benchmarks both implementations
func BenchmarkComparison(b *testing.B) {
	runner := newTestRunner(&testing.T{})

	// use a sample file for benchmarking
	testFile := "testdata/sample.gba" // adjust path as needed
	console := "GBA"

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skip("Benchmark test file not found")
	}

	b.Run("Go", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runner.runGoImplementation(console, testFile)
		}
	})

	b.Run("Python", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runner.runPythonImplementation(console, testFile)
		}
	})
}

#!/usr/bin/env python3
"""Compare GameID Go implementation output with Python reference implementation."""

import subprocess
import sys
import os
import json
import argparse
from pathlib import Path

def run_python_gameid(file_path, console, db_path=None):
    """Run the Python GameID implementation."""
    script_path = Path(__file__).parent / "GameID.py"
    cmd = [sys.executable, str(script_path), "-i", file_path, "-c", console]
    if db_path:
        cmd.extend(["-d", db_path])
    
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, check=True)
        return parse_output(result.stdout)
    except subprocess.CalledProcessError as e:
        print(f"Python GameID failed: {e.stderr}")
        return None

def run_go_gameid(file_path, console, db_path=None):
    """Run the Go GameID implementation."""
    # Try to find the binary
    binary_paths = [
        "./gameid",
        "./cmd/gameid/gameid",
        "gameid",
    ]
    
    binary = None
    for path in binary_paths:
        if os.path.exists(path):
            binary = path
            break
    
    if not binary:
        # Try to build it
        print("Building Go binary...")
        subprocess.run(["go", "build", "./cmd/gameid"], check=True)
        binary = "./gameid"
    
    cmd = [binary, "-i", file_path, "-c", console]
    if db_path:
        cmd.extend(["-d", db_path])
    
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, check=True)
        return parse_output(result.stdout)
    except subprocess.CalledProcessError as e:
        print(f"Go GameID failed: {e.stderr}")
        return None

def parse_output(output):
    """Parse tab-delimited output into a dictionary."""
    result = {}
    for line in output.strip().split('\n'):
        if '\t' in line:
            key, value = line.split('\t', 1)
            result[key] = value
    return result

def compare_outputs(python_output, go_output):
    """Compare outputs and report differences."""
    if python_output is None or go_output is None:
        return False
    
    all_keys = set(python_output.keys()) | set(go_output.keys())
    differences = []
    
    for key in sorted(all_keys):
        py_val = python_output.get(key, "<missing>")
        go_val = go_output.get(key, "<missing>")
        
        if py_val != go_val:
            differences.append({
                'key': key,
                'python': py_val,
                'go': go_val
            })
    
    if differences:
        print("❌ Outputs differ:")
        for diff in differences:
            print(f"  {diff['key']}:")
            print(f"    Python: {diff['python']}")
            print(f"    Go:     {diff['go']}")
        return False
    else:
        print("✅ Outputs match!")
        return True

def test_file(file_path, console, db_path=None):
    """Test a single file."""
    print(f"\nTesting: {file_path} (Console: {console})")
    print("-" * 60)
    
    python_output = run_python_gameid(file_path, console, db_path)
    go_output = run_go_gameid(file_path, console, db_path)
    
    return compare_outputs(python_output, go_output)

def main():
    parser = argparse.ArgumentParser(description="Compare Go and Python GameID implementations")
    parser.add_argument('-i', '--input', required=True, help='Input game file')
    parser.add_argument('-c', '--console', required=True, help='Console type')
    parser.add_argument('-d', '--database', help='Database file path')
    parser.add_argument('--batch', action='store_true', help='Run batch tests on test_data directory')
    
    args = parser.parse_args()
    
    if args.batch:
        # Run tests on all files in test_data directory
        test_data_dir = Path(__file__).parent.parent / "test_data"
        if not test_data_dir.exists():
            print(f"Test data directory not found: {test_data_dir}")
            return 1
        
        # Define test cases
        test_cases = [
            # Add test cases here as tuples of (file_pattern, console)
            ("*.gba", "GBA"),
            ("*.gb", "GB"),
            ("*.gbc", "GBC"),
            ("*.sfc", "SNES"),
            ("*.smc", "SNES"),
            ("*.n64", "N64"),
            ("*.z64", "N64"),
            ("*.md", "Genesis"),
            ("*.gen", "Genesis"),
            ("*.iso", "PSX"),  # Could also be PS2, GC, etc.
            ("*.cue", "PSX"),
        ]
        
        total_tests = 0
        passed_tests = 0
        
        for pattern, console in test_cases:
            for file_path in test_data_dir.rglob(pattern):
                total_tests += 1
                if test_file(str(file_path), console, args.database):
                    passed_tests += 1
        
        print(f"\n{'='*60}")
        print(f"Total tests: {total_tests}")
        print(f"Passed: {passed_tests}")
        print(f"Failed: {total_tests - passed_tests}")
        
        return 0 if passed_tests == total_tests else 1
    else:
        # Test single file
        success = test_file(args.input, args.console, args.database)
        return 0 if success else 1

if __name__ == "__main__":
    sys.exit(main())
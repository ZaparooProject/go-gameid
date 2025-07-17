#!/usr/bin/env python3
"""
Run comprehensive comparison tests between Go and Python GameID implementations
"""

import json
import os
import subprocess
import sys
import time
from pathlib import Path

def run_command(cmd, cwd=None):
    """Run a command and return output"""
    try:
        result = subprocess.run(
            cmd,
            shell=True,
            capture_output=True,
            text=True,
            cwd=cwd
        )
        return result.returncode, result.stdout, result.stderr
    except Exception as e:
        return 1, "", str(e)

def setup_environment():
    """Set up the test environment"""
    print("Setting up test environment...")
    
    # build the Go binary
    print("Building Go binary...")
    returncode, stdout, stderr = run_command("go build -o gameid ./cmd/gameid")
    if returncode != 0:
        print(f"Failed to build Go binary: {stderr}")
        return False
    
    # check if Python script exists
    if not os.path.exists("scripts/GameID.py"):
        print("Python GameID script not found at scripts/GameID.py")
        return False
    
    # create test samples if they don't exist
    if not os.path.exists("testdata"):
        print("Creating test samples...")
        returncode, stdout, stderr = run_command("python3 scripts/create_test_samples.py")
        if returncode != 0:
            print(f"Failed to create test samples: {stderr}")
            return False
    
    return True

def run_go_gameid(console, filepath):
    """Run Go GameID implementation"""
    cmd = f"./gameid -console {console} -input {filepath}"
    start_time = time.time()
    returncode, stdout, stderr = run_command(cmd)
    duration = time.time() - start_time
    
    if returncode != 0:
        return {
            "error": f"Command failed: {stderr}",
            "duration": duration,
            "fields": {}
        }
    
    # parse output
    fields = {}
    for line in stdout.strip().split('\n'):
        if '\t' in line:
            key, value = line.split('\t', 1)
            fields[key] = value
    
    return {
        "error": None,
        "duration": duration,
        "fields": fields
    }

def run_python_gameid(console, filepath):
    """Run Python GameID implementation"""
    cmd = f"python3 scripts/GameID.py --console {console} --input {filepath}"
    start_time = time.time()
    returncode, stdout, stderr = run_command(cmd)
    duration = time.time() - start_time
    
    if returncode != 0:
        return {
            "error": f"Command failed: {stderr}",
            "duration": duration,
            "fields": {}
        }
    
    # parse output
    fields = {}
    for line in stdout.strip().split('\n'):
        if '\t' in line:
            key, value = line.split('\t', 1)
            fields[key] = value
    
    return {
        "error": None,
        "duration": duration,
        "fields": fields
    }

def compare_results(go_result, python_result):
    """Compare Go and Python results"""
    comparison = {
        "passed": True,
        "differences": [],
        "missing_in_go": [],
        "missing_in_python": [],
        "error_comparison": {
            "go_error": go_result.get("error"),
            "python_error": python_result.get("error")
        }
    }
    
    # if both have errors, compare error messages
    if go_result.get("error") and python_result.get("error"):
        comparison["passed"] = False
        return comparison
    
    # if only one has error, it's a failure
    if go_result.get("error") or python_result.get("error"):
        comparison["passed"] = False
        return comparison
    
    # compare fields
    go_fields = go_result.get("fields", {})
    python_fields = python_result.get("fields", {})
    
    all_fields = set(go_fields.keys()) | set(python_fields.keys())
    
    for field in all_fields:
        go_value = go_fields.get(field, "")
        python_value = python_fields.get(field, "")
        
        if field not in go_fields:
            comparison["missing_in_go"].append(field)
            comparison["passed"] = False
        elif field not in python_fields:
            comparison["missing_in_python"].append(field)
            comparison["passed"] = False
        elif normalize_value(go_value) != normalize_value(python_value):
            comparison["differences"].append({
                "field": field,
                "go_value": go_value,
                "python_value": python_value
            })
            comparison["passed"] = False
    
    return comparison

def normalize_value(value):
    """Normalize values for comparison"""
    if isinstance(value, str):
        value = value.strip()
        if value.lower() in ["", "none", "null"]:
            return "None"
        # normalize hex values
        if value.startswith("0x"):
            return value.lower()
    return value

def find_test_files():
    """Find all test files"""
    test_files = {}
    
    # map directories to consoles
    console_dirs = {
        "gb": "GB",
        "gba": "GBA",
        "gc": "GC",
        "genesis": "Genesis",
        "n64": "N64",
        "psp": "PSP",
        "psx": "PSX",
        "ps2": "PS2",
        "saturn": "Saturn",
        "segacd": "SegaCD",
        "snes": "SNES"
    }
    
    testdata_dir = Path("testdata")
    if not testdata_dir.exists():
        return test_files
    
    for dir_name, console in console_dirs.items():
        console_dir = testdata_dir / dir_name
        if console_dir.exists():
            for file_path in console_dir.glob("*"):
                if file_path.is_file():
                    if console not in test_files:
                        test_files[console] = []
                    test_files[console].append(str(file_path))
    
    return test_files

def run_tests():
    """Run all comparison tests"""
    print("Running comparison tests...")
    
    test_files = find_test_files()
    if not test_files:
        print("No test files found")
        return []
    
    results = []
    total_tests = sum(len(files) for files in test_files.values())
    current_test = 0
    
    for console, files in test_files.items():
        print(f"\nTesting {console} ({len(files)} files)...")
        
        for filepath in files:
            current_test += 1
            print(f"  [{current_test}/{total_tests}] {filepath}")
            
            # run both implementations
            go_result = run_go_gameid(console, filepath)
            python_result = run_python_gameid(console, filepath)
            
            # compare results
            comparison = compare_results(go_result, python_result)
            
            # create test result
            test_result = {
                "console": console,
                "filepath": filepath,
                "go_result": go_result,
                "python_result": python_result,
                "comparison": comparison,
                "timestamp": time.time()
            }
            
            results.append(test_result)
            
            # print immediate feedback
            if comparison["passed"]:
                print(f"    ✓ PASS")
            else:
                print(f"    ✗ FAIL")
                if comparison["differences"]:
                    print(f"      {len(comparison['differences'])} field differences")
                if comparison["missing_in_go"]:
                    print(f"      Missing in Go: {', '.join(comparison['missing_in_go'])}")
                if comparison["missing_in_python"]:
                    print(f"      Missing in Python: {', '.join(comparison['missing_in_python'])}")
    
    return results

def generate_report(results):
    """Generate comprehensive test report"""
    print("\nGenerating report...")
    
    # create results directory
    os.makedirs("comparison_results", exist_ok=True)
    
    # calculate summary statistics
    total_tests = len(results)
    passed_tests = sum(1 for r in results if r["comparison"]["passed"])
    failed_tests = total_tests - passed_tests
    
    # categorize failures
    field_differences = {}
    missing_fields = {"go": {}, "python": {}}
    error_cases = {"go": [], "python": [], "both": []}
    
    for result in results:
        if not result["comparison"]["passed"]:
            comp = result["comparison"]
            
            # track field differences
            for diff in comp["differences"]:
                field = diff["field"]
                if field not in field_differences:
                    field_differences[field] = []
                field_differences[field].append({
                    "console": result["console"],
                    "file": result["filepath"],
                    "go_value": diff["go_value"],
                    "python_value": diff["python_value"]
                })
            
            # track missing fields
            for field in comp["missing_in_go"]:
                if field not in missing_fields["go"]:
                    missing_fields["go"][field] = []
                missing_fields["go"][field].append({
                    "console": result["console"],
                    "file": result["filepath"]
                })
            
            for field in comp["missing_in_python"]:
                if field not in missing_fields["python"]:
                    missing_fields["python"][field] = []
                missing_fields["python"][field].append({
                    "console": result["console"],
                    "file": result["filepath"]
                })
            
            # track error cases
            go_error = result["go_result"].get("error")
            python_error = result["python_result"].get("error")
            
            if go_error and python_error:
                error_cases["both"].append(result)
            elif go_error:
                error_cases["go"].append(result)
            elif python_error:
                error_cases["python"].append(result)
    
    # generate detailed JSON report
    detailed_report = {
        "summary": {
            "total_tests": total_tests,
            "passed_tests": passed_tests,
            "failed_tests": failed_tests,
            "success_rate": (passed_tests / total_tests * 100) if total_tests > 0 else 0
        },
        "field_differences": field_differences,
        "missing_fields": missing_fields,
        "error_cases": error_cases,
        "all_results": results,
        "generated_at": time.time()
    }
    
    with open("comparison_results/detailed_report.json", "w") as f:
        json.dump(detailed_report, f, indent=2)
    
    # generate human-readable summary
    summary_lines = [
        "# GameID Comparison Test Report",
        "",
        f"**Generated:** {time.strftime('%Y-%m-%d %H:%M:%S')}",
        "",
        "## Summary",
        "",
        f"- **Total Tests:** {total_tests}",
        f"- **Passed:** {passed_tests}",
        f"- **Failed:** {failed_tests}",
        f"- **Success Rate:** {passed_tests/total_tests*100:.1f}%",
        "",
    ]
    
    if failed_tests > 0:
        summary_lines.extend([
            "## Issues Found",
            "",
        ])
        
        if field_differences:
            summary_lines.extend([
                "### Field Differences",
                "",
            ])
            for field, cases in field_differences.items():
                summary_lines.append(f"**{field}** ({len(cases)} cases)")
                for case in cases[:3]:  # show first 3 cases
                    summary_lines.append(f"  - {case['console']}: Go='{case['go_value']}', Python='{case['python_value']}'")
                if len(cases) > 3:
                    summary_lines.append(f"  - ... and {len(cases) - 3} more cases")
                summary_lines.append("")
        
        if missing_fields["go"] or missing_fields["python"]:
            summary_lines.extend([
                "### Missing Fields",
                "",
            ])
            
            if missing_fields["go"]:
                summary_lines.append("**Missing in Go:**")
                for field, cases in missing_fields["go"].items():
                    summary_lines.append(f"  - {field} ({len(cases)} cases)")
                summary_lines.append("")
            
            if missing_fields["python"]:
                summary_lines.append("**Missing in Python:**")
                for field, cases in missing_fields["python"].items():
                    summary_lines.append(f"  - {field} ({len(cases)} cases)")
                summary_lines.append("")
        
        if any(error_cases.values()):
            summary_lines.extend([
                "### Error Cases",
                "",
            ])
            
            if error_cases["go"]:
                summary_lines.append(f"**Go errors:** {len(error_cases['go'])} cases")
            if error_cases["python"]:
                summary_lines.append(f"**Python errors:** {len(error_cases['python'])} cases")
            if error_cases["both"]:
                summary_lines.append(f"**Both error:** {len(error_cases['both'])} cases")
            summary_lines.append("")
    
    summary_lines.extend([
        "## Console Breakdown",
        "",
    ])
    
    console_stats = {}
    for result in results:
        console = result["console"]
        if console not in console_stats:
            console_stats[console] = {"total": 0, "passed": 0}
        console_stats[console]["total"] += 1
        if result["comparison"]["passed"]:
            console_stats[console]["passed"] += 1
    
    for console, stats in sorted(console_stats.items()):
        success_rate = (stats["passed"] / stats["total"] * 100) if stats["total"] > 0 else 0
        summary_lines.append(f"- **{console}:** {stats['passed']}/{stats['total']} ({success_rate:.1f}%)")
    
    summary_lines.extend([
        "",
        "## Recommendations",
        "",
    ])
    
    if failed_tests == 0:
        summary_lines.append("✓ All tests passed! The Go implementation appears to be working correctly.")
    else:
        summary_lines.append("⚠ Some tests failed. Priority areas for investigation:")
        
        if field_differences:
            most_common_diff = max(field_differences.items(), key=lambda x: len(x[1]))
            summary_lines.append(f"1. **{most_common_diff[0]}** field differences ({len(most_common_diff[1])} cases)")
        
        if missing_fields["go"]:
            most_common_missing = max(missing_fields["go"].items(), key=lambda x: len(x[1]))
            summary_lines.append(f"2. **{most_common_missing[0]}** missing in Go ({len(most_common_missing[1])} cases)")
        
        if error_cases["go"]:
            summary_lines.append(f"3. Go implementation errors ({len(error_cases['go'])} cases)")
    
    summary_lines.append("")
    summary_lines.append("See `comparison_results/detailed_report.json` for full details.")
    
    with open("comparison_results/summary.md", "w") as f:
        f.write("\n".join(summary_lines))
    
    print("Report generated in comparison_results/")
    print(f"Success rate: {passed_tests}/{total_tests} ({passed_tests/total_tests*100:.1f}%)")
    
    return detailed_report

def main():
    """Main function"""
    print("GameID Comparison Test Runner")
    print("=" * 40)
    
    # setup environment
    if not setup_environment():
        print("Failed to setup environment")
        sys.exit(1)
    
    # run tests
    results = run_tests()
    
    if not results:
        print("No tests were run")
        sys.exit(1)
    
    # generate report
    report = generate_report(results)
    
    # return appropriate exit code
    if report["summary"]["failed_tests"] > 0:
        sys.exit(1)
    else:
        sys.exit(0)

if __name__ == "__main__":
    main()
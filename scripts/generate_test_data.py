#!/usr/bin/env python3
"""
Generate test data by running Python GameID on sample games
"""

import json
import subprocess
import os
import sys
import glob
from pathlib import Path

# Path to Python GameID script
GAMEID_SCRIPT = Path(__file__).parent / "GameID.py"

# Default test games to process
DEFAULT_TEST_GAMES = {
    "GBA": [
        "/Volumes/MiSTer/games/GBA/1 USA - 0-9/007 - Everything or Nothing (USA, Europe) (En,Fr,De).gba",
        "/Volumes/MiSTer/games/GBA/1 USA - D-H/Golden Sun (USA, Europe).gba",
        "/Volumes/MiSTer/games/GBA/1 USA - D-H/Golden Sun - The Lost Age (USA, Europe).gba",
        "/Volumes/MiSTer/games/GBA/1 USA - A-C/Classic NES Series - Super Mario Bros. (USA, Europe).gba",
    ],
}

def run_gameid(game_path, console):
    """Run GameID.py and return the output as a dictionary"""
    cmd = [
        sys.executable,
        str(GAMEID_SCRIPT),
        "-i", game_path,
        "-c", console
    ]
    
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, check=True)
        
        # Parse the tab-delimited output
        output = {}
        for line in result.stdout.strip().split('\n'):
            if '\t' in line:
                key, value = line.split('\t', 1)
                output[key] = value
        
        return output
    except subprocess.CalledProcessError as e:
        print(f"Error running GameID on {game_path}: {e}")
        print(f"stderr: {e.stderr}")
        return None

def find_games_in_directories(console, directories):
    """Find game files in the given directories"""
    extensions = {
        "GB": [".gb", ".gbc"],
        "GBA": [".gba"],
        "N64": [".n64", ".z64", ".v64"],
        "SNES": [".sfc", ".smc"],
        "Genesis": [".md", ".gen", ".smd"]
    }
    
    games = []
    for directory in directories:
        if not os.path.exists(directory):
            print(f"Directory not found: {directory}")
            continue
            
        for ext in extensions.get(console, []):
            pattern = os.path.join(directory, f"*{ext}")
            found_games = glob.glob(pattern)
            games.extend(found_games[:5])  # Limit to 5 games per directory
    
    return games

def generate_test_data(console=None, directories=None):
    """Generate test data for specified console and directories, or use defaults"""
    output_dir = Path(__file__).parent.parent / "test_data" / "reference"
    output_dir.mkdir(parents=True, exist_ok=True)
    
    # Use command line args or defaults
    if console and directories:
        games = find_games_in_directories(console, directories)
        test_games = {console: games}
    else:
        test_games = DEFAULT_TEST_GAMES
    
    all_test_data = {}
    
    for console_name, games in test_games.items():
        console_data = {}
        
        for game_path in games:
            if not os.path.exists(game_path):
                print(f"Skipping {game_path} - file not found")
                continue
            
            print(f"Processing {console_name}: {os.path.basename(game_path)}")
            result = run_gameid(game_path, console_name)
            
            if result:
                game_name = os.path.basename(game_path)
                console_data[game_name] = {
                    "path": game_path,
                    "expected": result
                }
        
        if console_data:
            all_test_data[console_name] = console_data
            
            # Write per-console test data
            console_file = output_dir / f"{console_name.lower()}_test_data.json"
            with open(console_file, 'w') as f:
                json.dump({console_name: console_data}, f, indent=2)
            print(f"Wrote {console_name} test data to {console_file}")
    
    # Write combined test data
    combined_file = output_dir / "all_test_data.json"
    with open(combined_file, 'w') as f:
        json.dump(all_test_data, f, indent=2)
    print(f"Wrote combined test data to {combined_file}")

if __name__ == '__main__':
    if len(sys.argv) >= 3:
        console = sys.argv[1]
        directories = sys.argv[2:]
        generate_test_data(console, directories)
    else:
        generate_test_data()
#!/usr/bin/env python3
"""
Convert GameID pickle database to JSON format for Go
"""

import json
import pickle
import gzip
import sys
import os
from urllib.request import urlopen

DB_URL = 'https://github.com/niemasd/GameID/raw/main/db.pkl.gz'
DEFAULT_TIMEOUT = 10

def download_database(url=DB_URL, timeout=DEFAULT_TIMEOUT):
    """Download the GameID database from GitHub"""
    print(f"Downloading database from {url}...")
    try:
        response = urlopen(url, timeout=timeout)
        data = gzip.decompress(response.read())
        return pickle.loads(data)
    except Exception as e:
        print(f"Error downloading database: {e}")
        return None

def load_local_database(path):
    """Load database from local pickle file"""
    print(f"Loading database from {path}...")
    try:
        with gzip.open(path, 'rb') as f:
            return pickle.loads(f.read())
    except:
        # Try uncompressed pickle
        with open(path, 'rb') as f:
            return pickle.loads(f.read())

def convert_database(db):
    """Convert database to JSON-serializable format"""
    # The database structure is:
    # {
    #   'GAMEID': {...},  # Metadata about the database
    #   'GBA': {game_id: {metadata}},
    #   'GB_GBC': {(title, checksum): {metadata}},
    #   ...
    # }
    
    result = {}
    
    for system, games in db.items():
        if system == 'GAMEID':
            # Skip metadata for now
            continue
            
        result[system] = {}
        
        for game_id, metadata in games.items():
            # Handle composite keys (like GB_GBC with (title, checksum) tuples)
            if isinstance(game_id, tuple):
                # Convert tuple to string key
                if len(game_id) == 2 and isinstance(game_id[1], int):
                    # GB/GBC format: (title, checksum)
                    key = f"{game_id[0]},0x{game_id[1]:x}"
                else:
                    key = ",".join(str(x) for x in game_id)
            else:
                key = str(game_id)
            
            # Convert metadata
            game_data = {}
            for k, v in metadata.items():
                # Convert all values to strings
                if isinstance(v, (list, tuple)):
                    game_data[k] = " / ".join(str(x) for x in v)
                else:
                    game_data[k] = str(v)
            
            result[system][key] = game_data
    
    return result

def main():
    if len(sys.argv) > 1:
        # Load from local file
        db = load_local_database(sys.argv[1])
    else:
        # Download from URL
        db = download_database()
    
    if db is None:
        print("Failed to load database")
        sys.exit(1)
    
    # Convert to JSON format
    json_db = convert_database(db)
    
    # Output paths
    output_dir = os.path.join(os.path.dirname(__file__), '..', 'dbs')
    os.makedirs(output_dir, exist_ok=True)
    
    # Write full database
    output_path = os.path.join(output_dir, 'gameid_db.json')
    with open(output_path, 'w', encoding='utf-8') as f:
        json.dump(json_db, f, indent=2, ensure_ascii=False, sort_keys=True)
    print(f"Wrote full database to {output_path}")
    
    # Write individual system databases for easier testing
    for system, games in json_db.items():
        system_path = os.path.join(output_dir, f'{system.lower()}_db.json')
        with open(system_path, 'w', encoding='utf-8') as f:
            json.dump({system: games}, f, indent=2, ensure_ascii=False, sort_keys=True)
        print(f"Wrote {system} database to {system_path} ({len(games)} games)")

if __name__ == '__main__':
    main()
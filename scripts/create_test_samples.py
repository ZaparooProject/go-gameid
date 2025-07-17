#!/usr/bin/env python3
"""
Create sample ROM/ISO files for testing GameID comparison
"""

import os
import struct
import sys
from pathlib import Path

def create_test_directory():
    """Create testdata directory structure"""
    testdata_dir = Path("testdata")
    testdata_dir.mkdir(exist_ok=True)
    
    # create console-specific directories
    consoles = ["gb", "gba", "gc", "genesis", "n64", "psp", "psx", "ps2", "saturn", "segacd", "snes"]
    for console in consoles:
        (testdata_dir / console).mkdir(exist_ok=True)
    
    return testdata_dir

def create_gba_sample(filepath):
    """Create a minimal GBA ROM sample"""
    # GBA ROM header structure
    data = bytearray(0x200)  # 512 bytes minimum
    
    # Entry point (4 bytes)
    data[0x00:0x04] = b'\x00\x00\x00\xea'  # branch instruction
    
    # Nintendo logo (156 bytes at 0x04)
    nintendo_logo = bytes([0x24, 0xFF, 0xAE, 0x51, 0x69, 0x9A, 0xA2, 0x21, 0x3D, 0x84, 0x82, 0x0A, 0x84, 0xE4, 0x09, 0xAD, 0x11, 0x24, 0x8B, 0x98, 0xC0, 0x81, 0x7F, 0x21, 0xA3, 0x52, 0xBE, 0x19, 0x93, 0x09, 0xCE, 0x20, 0x10, 0x46, 0x4A, 0x4A, 0xF8, 0x27, 0x31, 0xEC, 0x58, 0xC7, 0xE8, 0x33, 0x82, 0xE3, 0xCE, 0xBF, 0x85, 0xF4, 0xDF, 0x94, 0xCE, 0x4B, 0x09, 0xC1, 0x94, 0x56, 0x8A, 0xC0, 0x13, 0x72, 0xA7, 0xFC, 0x9F, 0x84, 0x4D, 0x73, 0xA3, 0xCA, 0x9A, 0x61, 0x58, 0x97, 0xA3, 0x27, 0xFC, 0x03, 0x98, 0x76, 0x23, 0x1D, 0xC7, 0x61, 0x03, 0x04, 0xAE, 0x56, 0xBF, 0x38, 0x84, 0x00, 0x40, 0xA7, 0x0E, 0xFD, 0xFF, 0x52, 0xFE, 0x03, 0x6F, 0x95, 0x30, 0xF1, 0x97, 0xFB, 0xC0, 0x85, 0x60, 0xD6, 0x80, 0x25, 0xA9, 0x63, 0xBE, 0x03, 0x01, 0x4E, 0x38, 0xE2, 0xF9, 0xA2, 0x34, 0xFF, 0xBB, 0x3E, 0x03, 0x44, 0x78, 0x00, 0x90, 0xCB, 0x88, 0x11, 0x3A, 0x94, 0x65, 0xC0, 0x7C, 0x63, 0x87, 0xF0, 0x3C, 0xAF, 0xD6, 0x25, 0xE4, 0x8B, 0x38, 0x0A, 0xAC, 0x72, 0x21, 0xD4, 0xF8, 0x07])
    data[0x04:0x04+len(nintendo_logo)] = nintendo_logo
    
    # Game title (12 bytes at 0xA0)
    title = b"TEST GAME".ljust(12, b'\x00')
    data[0xA0:0xAC] = title
    
    # Game code (4 bytes at 0xAC)
    game_code = b"TEST"
    data[0xAC:0xB0] = game_code
    
    # Maker code (2 bytes at 0xB0)
    maker_code = b"01"
    data[0xB0:0xB2] = maker_code
    
    # Fixed value (1 byte at 0xB2)
    data[0xB2] = 0x96
    
    # Main unit code (1 byte at 0xB3)
    data[0xB3] = 0x00
    
    # Device type (1 byte at 0xB4)
    data[0xB4] = 0x00
    
    # Reserved (7 bytes at 0xB5)
    data[0xB5:0xBC] = b'\x00' * 7
    
    # Software version (1 byte at 0xBC)
    data[0xBC] = 0x00
    
    # Complement check (1 byte at 0xBD)
    data[0xBD] = 0x00
    
    # Reserved (2 bytes at 0xBE)
    data[0xBE:0xC0] = b'\x00' * 2
    
    with open(filepath, 'wb') as f:
        f.write(data)

def create_gb_sample(filepath):
    """Create a minimal Game Boy ROM sample"""
    data = bytearray(0x8000)  # 32KB minimum
    
    # Entry point (4 bytes at 0x100)
    data[0x100:0x104] = b'\x00\xc3\x50\x01'  # NOP; JP $0150
    
    # Nintendo logo (48 bytes at 0x104)
    nintendo_logo = bytes([0xCE, 0xED, 0x66, 0x66, 0xCC, 0x0D, 0x00, 0x0B, 0x03, 0x73, 0x00, 0x83, 0x00, 0x0C, 0x00, 0x0D, 0x00, 0x08, 0x11, 0x1F, 0x88, 0x89, 0x00, 0x0E, 0xDC, 0xCC, 0x6E, 0xE6, 0xDD, 0xDD, 0xD9, 0x99, 0xBB, 0xBB, 0x67, 0x63, 0x6E, 0x0E, 0xEC, 0xCC, 0xDD, 0xDC, 0x99, 0x9F, 0xBB, 0xB9, 0x33, 0x3E])
    data[0x104:0x104+len(nintendo_logo)] = nintendo_logo
    
    # Game title (11 bytes at 0x134)
    title = b"TEST GAME".ljust(11, b'\x00')
    data[0x134:0x13F] = title
    
    # Manufacturer code (4 bytes at 0x13F)
    data[0x13F:0x143] = b'\x00\x00\x00\x00'
    
    # CGB flag (1 byte at 0x143)
    data[0x143] = 0x00  # GB only
    
    # New licensee code (2 bytes at 0x144)
    data[0x144:0x146] = b'01'
    
    # SGB flag (1 byte at 0x146)
    data[0x146] = 0x00
    
    # Cartridge type (1 byte at 0x147)
    data[0x147] = 0x00  # ROM only
    
    # ROM size (1 byte at 0x148)
    data[0x148] = 0x00  # 32KB
    
    # RAM size (1 byte at 0x149)
    data[0x149] = 0x00  # No RAM
    
    # Destination code (1 byte at 0x14A)
    data[0x14A] = 0x01  # Non-Japanese
    
    # Old licensee code (1 byte at 0x14B)
    data[0x14B] = 0x33  # Use new licensee code
    
    # ROM version (1 byte at 0x14C)
    data[0x14C] = 0x00
    
    # Header checksum (1 byte at 0x14D)
    checksum = 0
    for i in range(0x134, 0x14D):
        checksum = (checksum - data[i] - 1) & 0xFF
    data[0x14D] = checksum
    
    # Global checksum (2 bytes at 0x14E)
    global_checksum = 0
    for i, byte in enumerate(data):
        if i not in [0x14E, 0x14F]:
            global_checksum += byte
    global_checksum &= 0xFFFF
    data[0x14E:0x150] = struct.pack('>H', global_checksum)
    
    with open(filepath, 'wb') as f:
        f.write(data)

def create_snes_sample(filepath):
    """Create a minimal SNES ROM sample"""
    # create 1MB ROM
    data = bytearray(0x100000)
    
    # SNES header at 0x7FC0 (LoROM)
    header_start = 0x7FC0
    
    # Game title (21 bytes)
    title = b"TEST GAME".ljust(21, b' ')
    data[header_start:header_start+21] = title
    
    # ROM makeup (1 byte at 0x7FD5)
    data[header_start + 21] = 0x20  # LoROM, FastROM
    
    # Cartridge type (1 byte at 0x7FD6)
    data[header_start + 22] = 0x00  # ROM only
    
    # ROM size (1 byte at 0x7FD7)
    data[header_start + 23] = 0x0A  # 1MB
    
    # RAM size (1 byte at 0x7FD8)
    data[header_start + 24] = 0x00  # No RAM
    
    # Country code (1 byte at 0x7FD9)
    data[header_start + 25] = 0x01  # US
    
    # Developer ID (1 byte at 0x7FDA)
    data[header_start + 26] = 0x00
    
    # ROM version (1 byte at 0x7FDB)
    data[header_start + 27] = 0x00
    
    # Checksum complement (2 bytes at 0x7FDC)
    # Checksum (2 bytes at 0x7FDE)
    checksum = sum(data) & 0xFFFF
    complement = (0xFFFF - checksum) & 0xFFFF
    data[header_start + 28:header_start + 30] = struct.pack('<H', complement)
    data[header_start + 30:header_start + 32] = struct.pack('<H', checksum)
    
    with open(filepath, 'wb') as f:
        f.write(data)

def create_n64_sample(filepath):
    """Create a minimal N64 ROM sample"""
    data = bytearray(0x100000)  # 1MB minimum
    
    # First word (4 bytes at 0x00) - big endian
    data[0x00:0x04] = b'\x80\x37\x12\x40'
    
    # Clock rate (4 bytes at 0x04)
    data[0x04:0x08] = b'\x00\x00\x00\x0F'
    
    # Entry point (4 bytes at 0x08)
    data[0x08:0x0C] = b'\x80\x00\x04\x00'
    
    # Release address (4 bytes at 0x0C)
    data[0x0C:0x10] = b'\x00\x00\x14\x44'
    
    # CRC1 (4 bytes at 0x10)
    data[0x10:0x14] = b'\x00\x00\x00\x00'
    
    # CRC2 (4 bytes at 0x14)
    data[0x14:0x18] = b'\x00\x00\x00\x00'
    
    # Unknown (8 bytes at 0x18)
    data[0x18:0x20] = b'\x00' * 8
    
    # Internal name (20 bytes at 0x20)
    name = b"TEST GAME".ljust(20, b' ')
    data[0x20:0x34] = name
    
    # Unknown (4 bytes at 0x34)
    data[0x34:0x38] = b'\x00' * 4
    
    # Unknown (4 bytes at 0x38)
    data[0x38:0x3C] = b'\x00' * 4
    
    # Cartridge ID (2 bytes at 0x3C)
    data[0x3C:0x3E] = b'NT'
    
    # Country code (1 byte at 0x3E)
    data[0x3E] = ord('E')  # USA
    
    # Version (1 byte at 0x3F)
    data[0x3F] = 0x00
    
    with open(filepath, 'wb') as f:
        f.write(data)

def create_genesis_sample(filepath):
    """Create a minimal Genesis ROM sample"""
    data = bytearray(0x80000)  # 512KB
    
    # Fill with Genesis header starting at 0x100
    header_start = 0x100
    
    # System type (16 bytes at 0x100)
    system_type = b"SEGA GENESIS".ljust(16, b' ')
    data[header_start:header_start+16] = system_type
    
    # Publisher (4 bytes at 0x110)
    data[header_start + 16:header_start + 20] = b"    "
    
    # Release date (8 bytes at 0x118)
    data[header_start + 24:header_start + 32] = b"1990.01 "
    
    # Domestic name (48 bytes at 0x120)
    domestic_name = b"TEST GAME".ljust(48, b' ')
    data[header_start + 32:header_start + 80] = domestic_name
    
    # Overseas name (48 bytes at 0x150)
    overseas_name = b"TEST GAME".ljust(48, b' ')
    data[header_start + 80:header_start + 128] = overseas_name
    
    # Serial number (11 bytes at 0x180)
    serial = b"GM 00000000"
    data[header_start + 128:header_start + 139] = serial
    
    # Checksum (2 bytes at 0x18E)
    data[header_start + 142:header_start + 144] = b'\x00\x00'
    
    # Device support (16 bytes at 0x190)
    device_support = b"J".ljust(16, b' ')
    data[header_start + 144:header_start + 160] = device_support
    
    # ROM start/end (8 bytes at 0x1A0)
    data[header_start + 160:header_start + 168] = b'\x00\x00\x00\x00\x00\x07\xFF\xFF'
    
    # RAM start/end (8 bytes at 0x1A8)
    data[header_start + 168:header_start + 176] = b'\x00\x00\x00\x00\x00\x00\x00\x00'
    
    # Region support (3 bytes at 0x1F0)
    data[header_start + 240:header_start + 243] = b"JUE"
    
    with open(filepath, 'wb') as f:
        f.write(data)

def create_basic_iso(filepath, system_id, volume_id):
    """Create a basic ISO 9660 image"""
    data = bytearray(0x10000)  # 64KB minimum
    
    # Primary Volume Descriptor at sector 16 (0x8000)
    pvd_start = 0x8000
    
    # Volume descriptor type (1 byte)
    data[pvd_start] = 0x01  # Primary Volume Descriptor
    
    # Standard identifier (5 bytes)
    data[pvd_start + 1:pvd_start + 6] = b"CD001"
    
    # Volume descriptor version (1 byte)
    data[pvd_start + 6] = 0x01
    
    # Unused (1 byte)
    data[pvd_start + 7] = 0x00
    
    # System identifier (32 bytes)
    sys_id = system_id.encode('ascii').ljust(32, b' ')
    data[pvd_start + 8:pvd_start + 40] = sys_id
    
    # Volume identifier (32 bytes)
    vol_id = volume_id.encode('ascii').ljust(32, b' ')
    data[pvd_start + 40:pvd_start + 72] = vol_id
    
    # Volume space size (8 bytes)
    data[pvd_start + 80:pvd_start + 88] = b'\x00\x00\x00\x20\x20\x00\x00\x00'
    
    # Volume set size (4 bytes)
    data[pvd_start + 120:pvd_start + 124] = b'\x01\x00\x00\x01'
    
    # Volume sequence number (4 bytes)
    data[pvd_start + 124:pvd_start + 128] = b'\x01\x00\x00\x01'
    
    # Logical block size (4 bytes)
    data[pvd_start + 128:pvd_start + 132] = b'\x00\x08\x08\x00'
    
    # Root directory entry (34 bytes at offset 156)
    root_dir_start = pvd_start + 156
    data[root_dir_start] = 34  # Directory record length
    data[root_dir_start + 1] = 0  # Extended attribute record length
    data[root_dir_start + 2:root_dir_start + 6] = b'\x12\x00\x00\x00'  # LBA of extent
    data[root_dir_start + 6:root_dir_start + 10] = b'\x00\x00\x00\x12'  # LBA of extent (big endian)
    data[root_dir_start + 10:root_dir_start + 14] = b'\x00\x08\x00\x00'  # Data length
    data[root_dir_start + 14:root_dir_start + 18] = b'\x00\x00\x08\x00'  # Data length (big endian)
    
    # Creation timestamp (7 bytes)
    data[root_dir_start + 18:root_dir_start + 25] = b'\x00' * 7
    
    # File flags (1 byte)
    data[root_dir_start + 25] = 0x02  # Directory
    
    # File unit size (1 byte)
    data[root_dir_start + 26] = 0x00
    
    # Interleave gap size (1 byte)
    data[root_dir_start + 27] = 0x00
    
    # Volume sequence number (4 bytes)
    data[root_dir_start + 28:root_dir_start + 32] = b'\x01\x00\x00\x01'
    
    # File identifier length (1 byte)
    data[root_dir_start + 32] = 0x01
    
    # File identifier (1 byte)
    data[root_dir_start + 33] = 0x00
    
    with open(filepath, 'wb') as f:
        f.write(data)

def create_psx_sample(filepath):
    """Create a minimal PSX disc image"""
    create_basic_iso(filepath, "PLAYSTATION", "SLUS_012.34")

def create_psp_sample(filepath):
    """Create a minimal PSP disc image"""
    create_basic_iso(filepath, "PSP GAME", "ULUS10000")

def create_saturn_sample(filepath):
    """Create a minimal Saturn disc image"""
    data = bytearray(0x10000)  # 64KB
    
    # Saturn header starts at 0x0000
    header_start = 0x0000
    
    # Magic word (16 bytes)
    magic = b"SEGA SEGASATURN "
    data[header_start:header_start + 16] = magic
    
    # Hardware identifier (16 bytes)
    data[header_start + 16:header_start + 32] = b"SEGA TP T-000   "
    
    # Product ID (10 bytes)
    data[header_start + 32:header_start + 42] = b"T-000     "
    
    # Version (6 bytes)
    data[header_start + 42:header_start + 48] = b"V1.000"
    
    # Release date (8 bytes)
    data[header_start + 48:header_start + 56] = b"19950101"
    
    # Device information (8 bytes)
    data[header_start + 56:header_start + 64] = b"        "
    
    # Area symbols (16 bytes)
    data[header_start + 64:header_start + 80] = b"J               "
    
    # Peripheral support (16 bytes)
    data[header_start + 80:header_start + 96] = b"J               "
    
    # Game title (112 bytes)
    title = b"TEST GAME".ljust(112, b' ')
    data[header_start + 96:header_start + 208] = title
    
    with open(filepath, 'wb') as f:
        f.write(data)

def create_segacd_sample(filepath):
    """Create a minimal Sega CD disc image"""
    data = bytearray(0x10000)  # 64KB
    
    # Sega CD header starts at 0x0000
    header_start = 0x0000
    
    # Magic word (16 bytes)
    magic = b"SEGADISCSYSTEM  "
    data[header_start:header_start + 16] = magic
    
    # Volume ID (11 bytes)
    data[header_start + 16:header_start + 27] = b"SEGACD     "
    
    # System name (11 bytes)
    data[header_start + 32:header_start + 43] = b"SEGACD     "
    
    # Product ID (16 bytes)
    data[header_start + 384:header_start + 400] = b"GM 00000000-00  "
    
    # Version (16 bytes)
    data[header_start + 400:header_start + 416] = b"V1.00           "
    
    # Release year (4 bytes)
    data[header_start + 408:header_start + 412] = b"1993"
    
    # Release month (3 bytes)
    data[header_start + 413:header_start + 416] = b"JAN"
    
    # Domestic title (48 bytes)
    domestic_title = b"TEST GAME".ljust(48, b' ')
    data[header_start + 416:header_start + 464] = domestic_title
    
    # Overseas title (48 bytes)
    overseas_title = b"TEST GAME".ljust(48, b' ')
    data[header_start + 464:header_start + 512] = overseas_title
    
    with open(filepath, 'wb') as f:
        f.write(data)

def create_gamecube_sample(filepath):
    """Create a minimal GameCube disc image"""
    data = bytearray(0x10000)  # 64KB
    
    # GameCube header starts at 0x0000
    header_start = 0x0000
    
    # Game ID (4 bytes)
    data[header_start:header_start + 4] = b"TEST"
    
    # Company ID (2 bytes)
    data[header_start + 4:header_start + 6] = b"01"
    
    # Disk ID (1 byte)
    data[header_start + 6] = 0x00
    
    # Version (1 byte)
    data[header_start + 7] = 0x00
    
    # Audio streaming (1 byte)
    data[header_start + 8] = 0x00
    
    # Stream buffer size (1 byte)
    data[header_start + 9] = 0x00
    
    # Unused (14 bytes)
    data[header_start + 10:header_start + 24] = b'\x00' * 14
    
    # Magic word (4 bytes)
    data[header_start + 28:header_start + 32] = b'\xc2\x33\x9f\x3d'
    
    # Game name (992 bytes at 0x0020)
    game_name = b"TEST GAME".ljust(992, b'\x00')
    data[header_start + 32:header_start + 1024] = game_name
    
    with open(filepath, 'wb') as f:
        f.write(data)

def main():
    """Create all sample files"""
    testdata_dir = create_test_directory()
    
    # create sample files
    samples = [
        ("gb/test_game.gb", create_gb_sample),
        ("gba/test_game.gba", create_gba_sample),
        ("snes/test_game.sfc", create_snes_sample),
        ("n64/test_game.n64", create_n64_sample),
        ("genesis/test_game.gen", create_genesis_sample),
        ("psx/test_game.bin", create_psx_sample),
        ("psp/test_game.iso", create_psp_sample),
        ("saturn/test_game.iso", create_saturn_sample),
        ("segacd/test_game.iso", create_segacd_sample),
        ("gc/test_game.iso", create_gamecube_sample),
    ]
    
    for relative_path, creator_func in samples:
        filepath = testdata_dir / relative_path
        print(f"Creating {filepath}")
        creator_func(filepath)
    
    print(f"Created {len(samples)} sample files in {testdata_dir}")
    print("Note: These are minimal test samples and may not work with all identifiers.")
    print("For comprehensive testing, use real ROM/ISO files.")

if __name__ == "__main__":
    main()
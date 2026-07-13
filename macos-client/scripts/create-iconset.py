from pathlib import Path
import shutil
import subprocess

from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
SRC = ROOT / "assets" / "icon.png"
ICONSET = ROOT / "assets" / "icon.iconset"
OUT = ROOT / "assets" / "icon.icns"

SIZES = [
    ("icon_16x16.png", 16),
    ("icon_16x16@2x.png", 32),
    ("icon_32x32.png", 32),
    ("icon_32x32@2x.png", 64),
    ("icon_128x128.png", 128),
    ("icon_128x128@2x.png", 256),
    ("icon_256x256.png", 256),
    ("icon_256x256@2x.png", 512),
    ("icon_512x512.png", 512),
    ("icon_512x512@2x.png", 1024),
]

if ICONSET.exists():
    shutil.rmtree(ICONSET)
ICONSET.mkdir(parents=True)

source = Image.open(SRC).convert("RGBA")
for filename, size in SIZES:
    resized = source.resize((size, size), Image.Resampling.LANCZOS)
    resized.save(ICONSET / filename, format="PNG")

subprocess.run(["iconutil", "-c", "icns", str(ICONSET), "-o", str(OUT)], check=True)
shutil.rmtree(ICONSET)

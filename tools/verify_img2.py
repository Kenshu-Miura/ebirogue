"""img-2 の全数・再加工の再現性・上位種のアルファと明度を検証。"""
from pathlib import Path
import hashlib
import re
import subprocess
import sys
from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
paths = [ROOT / p for p in (ROOT / "tools/img2-files.txt").read_text(encoding="utf-8").splitlines()]
assert len(paths) == len(set(paths)) == 41

def hashes():
    return {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in paths}

before = hashes()
subprocess.run([sys.executable, "tools/process_expansion_sprites.py", "--originals", "tools/img2-originals", "--count", "17"], cwd=ROOT, check=True)
subprocess.run(["go", "run", "tools/expansion_enemy_images.go"], cwd=ROOT, check=True)
assert before == hashes(), "rebuild changed sprites"
names = {p.stem for p in paths}
table = (ROOT / "tools/expansion_enemy_images.go").read_text(encoding="utf-8")
pairs = [(a, b) for a, b in re.findall(r'\{"([^"]+)", "([^"]+)", [\d.]+\}', table) if a in names]
assert len(pairs) == 24
for src, dst in pairs:
    a = Image.open(ROOT / "img" / (src + ".png")).convert("RGBA")
    b = Image.open(ROOT / "img" / (dst + ".png")).convert("RGBA")
    assert a.size == b.size == (30, 30)
    assert a.getchannel("A").tobytes() == b.getchannel("A").tobytes(), dst
    for x, y in ((x, y) for y in range(30) for x in range(30)):
        c, d = a.getpixel((x, y)), b.getpixel((x, y))
        if c[3]:
            assert abs(max(c[:3]) + min(c[:3]) - max(d[:3]) - min(d[:3])) <= 2, dst
print("OK: 41 SHA-256 matches; 24 alpha/lightness invariants")

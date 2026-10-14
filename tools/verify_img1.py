"""img-1 の全数・色違い不変条件・保存原画からの再現性を検証する。"""
from pathlib import Path
import hashlib
import re
import subprocess
import sys
from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
paths = [ROOT / p for p in (ROOT / "tools/img1-files.txt").read_text(encoding="utf-8").splitlines()]
assert len(paths) == len(set(paths)) == 60, "expected 60 distinct sprites"


def hashes():
    return {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in paths}


before = hashes()
subprocess.run([sys.executable, str(ROOT / "tools/process_expansion_sprites.py")], cwd=ROOT, check=True)
subprocess.run(["go", "run", "tools/expansion_enemy_images.go"], cwd=ROOT, check=True)
assert before == hashes(), "rebuild changed final sprites"

table = (ROOT / "tools/expansion_enemy_images.go").read_text(encoding="utf-8")
names = {p.stem for p in paths}
pairs = [p for p in re.findall(r'\{"([^"]+)", "([^"]+)", ([\d.]+)\}', table) if p[0] in names]
assert len(pairs) == 16, "expected 16 hue conversions"
for src, dst, _ in pairs:
    a = Image.open(ROOT / "img" / (src + ".png")).convert("RGBA")
    b = Image.open(ROOT / "img" / (dst + ".png")).convert("RGBA")
    assert a.size == b.size == (30, 30)
    assert a.getchannel("A").tobytes() == b.getchannel("A").tobytes(), dst + ": alpha changed"
    for y in range(30):
        for x in range(30):
            c, d = a.getpixel((x, y)), b.getpixel((x, y))
            if not c[3]:
                continue
            # HSL明度=(最大RGB+最小RGB)/2。色相変換の丸めを1階調許容。
            assert abs(max(c[:3]) + min(c[:3]) - max(d[:3]) - min(d[:3])) <= 2, dst + ": lightness changed"
print("OK: 60 SHA-256 matches; 16 alpha/lightness invariants")

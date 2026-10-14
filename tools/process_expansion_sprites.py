"""img-1 原画の加工。Pillow: python tools/process_expansion_sprites.py

tools/img1-originals/<name>.png を保持し、切り抜き・縮小・32色化・
外周のアルファ除去・1px輪郭を決定的に再生成する。
"""
from pathlib import Path
from PIL import Image, ImageFilter

ROOT = Path(__file__).resolve().parent.parent


def process(path):
    src = Image.open(path).convert("RGBA")
    # 生成画像の薄い外側のにじみは採用しない。
    alpha = src.getchannel("A").point(lambda a: 255 if a >= 160 else 0)
    src.putalpha(alpha)
    bounds = alpha.getbbox()
    if not bounds:
        raise ValueError(f"empty original: {path}")
    src = src.crop(bounds)
    scale = 26 / max(src.size)
    size = tuple(max(1, round(v * scale)) for v in src.size)
    src = src.resize(size, Image.Resampling.LANCZOS)
    mask = src.getchannel("A").point(lambda a: 255 if a >= 128 else 0)
    # 透明色を色数に含めず、被写体の色を減色する。
    rgb = src.convert("RGB").quantize(colors=31, method=Image.Quantize.MEDIANCUT).convert("RGB")
    src = rgb.convert("RGBA")
    src.putalpha(mask)
    canvas = Image.new("RGBA", (30, 30))
    canvas.paste(src, ((30-size[0])//2, (30-size[1])//2))
    alpha = canvas.getchannel("A")
    expanded = alpha.filter(ImageFilter.MaxFilter(3))
    out = Image.new("RGBA", (30, 30), (3, 2, 1, 0))
    out.putalpha(expanded)
    out.alpha_composite(canvas)
    out.save(ROOT / "img" / path.name)


if __name__ == "__main__":
    paths = sorted((ROOT / "tools" / "img1-originals").glob("*.png"))
    if len(paths) != 44:
        raise ValueError(f"expected 44 originals, found {len(paths)}")
    for path in paths:
        process(path)
    print(f"processed {len(paths)} originals")

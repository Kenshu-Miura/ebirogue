"""段8画像の規格検査と暗い床・8方向・透明時の比較（実機確認とは別）。"""
from pathlib import Path
from PIL import Image, ImageDraw

ROOT = Path(__file__).resolve().parent.parent
OUT = ROOT / "tools/wave8-originals/review"
NAMES = ("no_healing_trap", "player_nigiri", "enemy_decoy")


def main():
    OUT.mkdir(exist_ok=True)
    for name in NAMES:
        im = Image.open(ROOT / f"img/{name}.png")
        assert im.size == (30, 30) and im.mode == "RGBA", name
        assert all(im.getpixel(p)[3] == 0 for p in ((0, 0), (29, 0), (0, 29), (29, 29))), name
        assert set(im.getchannel("A").tobytes()) <= {0, 255}, name
        assert len(im.getcolors(maxcolors=900)) <= 33, name
        print(f"OK: {name}: 30x30 RGBA, transparent corners, binary alpha")
    sheet = Image.new("RGB", (960, 540), (24, 24, 28))
    draw = ImageDraw.Draw(sheet)
    floor = Image.open(ROOT / "img/tileset.png").convert("RGBA").crop((60, 0, 90, 30))

    def cell(sprite, x, y, label):
        tile = floor.copy()
        tile.alpha_composite(sprite, ((30 - sprite.width) // 2, (30 - sprite.height) // 2))
        sheet.paste(tile.resize((90, 90), Image.Resampling.NEAREST), (x, y))
        sheet.paste(tile, (x + 32, y + 96))
        draw.text((x, y + 130), label, fill="white")

    for i, name in enumerate(("slow_trap", "poison_arrow_trap", "no_healing_trap", "ebi", "enemy_decoy", "nigiri_ebi")):
        cell(Image.open(ROOT / f"img/{name}.png").convert("RGBA"), i * 160, 0, name)
    nigiri = Image.open(ROOT / "img/player_nigiri.png").convert("RGBA")
    for i in range(8):
        for row, alpha in enumerate((1, 0.4)):
            sprite = nigiri.rotate(-45 * i, resample=Image.Resampling.NEAREST, expand=True)
            sprite.putalpha(sprite.getchannel("A").point(lambda a: round(a * alpha)))
            cell(sprite, i * 120, 180 + row * 180, f"{i * 45} deg / {alpha}")
    sheet.save(OUT / "sheet.png")
    print(OUT / "sheet.png")


if __name__ == "__main__":
    main()

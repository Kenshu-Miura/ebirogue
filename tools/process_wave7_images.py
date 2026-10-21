"""wave7-D原画atlasの加工。python tools/process_wave7_images.py ATLAS.png

原画はtools/wave7-originals/に保存しコミットしない。
4列×4行の先頭14セルを切り抜き、既存の加工器で30x30にする。
"""
import sys
from pathlib import Path
from PIL import Image
from process_expansion_sprites import process

NAMES = '''warp_trap pitfall_trap bear_trap wood_arrow_trap
iron_arrow_trap falling_rock_trap large_mine_trap unequip_trap
spoil_trap scatter_trap rotten_sausage boss_crab
boss_octopus boss_dragon'''.split()

if __name__ == '__main__':
    atlas = Image.open(sys.argv[1]).convert('RGBA')
    out = Path('tools/wave7-originals')
    out.mkdir(parents=True, exist_ok=True)
    # 生成atlasのセルにはみ出した冠・触手を隣の画像から分離する。
    boxes = [(30,35,302,295), (340,35,615,295), (650,35,925,295), (975,45,1261,295),
             (30,320,315,595), (330,295,620,595), (650,320,925,595), (970,320,1261,595),
             (30,600,315,895), (330,600,630,895), (635,625,915,895), (915,595,1261,895),
             (0,895,335,1247), (335,895,685,1247)]
    if atlas.size != (1261, 1247):
        raise ValueError('expected the documented 1261x1247 atlas')
    for i, name in enumerate(NAMES):
        cell = atlas.crop(boxes[i])
        path = out / (name + '.png')
        cell.save(path)
        process(path)
    print('processed', len(NAMES), 'sprites')

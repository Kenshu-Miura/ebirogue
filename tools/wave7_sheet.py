"""暗い床2種上の4倍比較。--beforeではorigin/masterの画像を使う。"""
import io
import subprocess
import sys
from PIL import Image, ImageDraw
from process_wave7_images import NAMES

BASES = 'ebi snake mamuru honey harisenbow ishigani bakudan_uni kosodoro_yadokari nigiri_ebi noroi_gani ayatsuri_kurage yurei_ebi hayate_shako warp_kurage irekae_dako wanashi_yadokari mimic_gai sea_dragon fire_squid magic_jellyfish'.split()
SECOND = 'dai_ebi moudoku_hebi anagura_mamuru gunegune_honey harisen_ou ganseki_gani dynamite_uni oodorobou_yadokari nigiri_oyakata_ebi tatari_gani shihai_kurage shinigami_ebi inazuma_shako telepo_kurage tokkae_dako wana_master_yadokari bake_horagai azure_sea_dragon inferno_squid arcane_jellyfish'.split()
THIRD = 'ousama_ebi gekidoku_hebi iwaya_mamuru uneune_honey harisen_koutei iwayama_gani megaton_uni kaitou_yadokari nigiri_meijin_ebi onryou_gani teiou_kurage meifu_ebi raijin_shako jikuu_kurage surikae_dako wana_ou_yadokari bake_oumugai shinen_kairyu rengoku_ika kenja_kurage'.split()

def make(before=False):
    rows = [NAMES[i:i+3] for i in range(0,len(NAMES),3)] + list(zip(BASES, SECOND, THIRD))
    # ID61/62は化けオウム貝の同じ画像を共用。両IDを明示する。
    rows.insert(22, ('mimic_gai','bake_horagai','bake_oumugai'))
    canvas = Image.new('RGB', (1560,len(rows)*150), (24,24,28))
    draw = ImageDraw.Draw(canvas)
    for y,row in enumerate(rows):
        for x,name in enumerate(row):
            path = 'img/'+name+'.png'
            sprite = Image.open(io.BytesIO(subprocess.check_output(['git','show','origin/master:'+path]))) if before else Image.open(path)
            sprite = sprite.convert('RGBA').resize((120,120),Image.Resampling.NEAREST)
            for f,floor in enumerate(((38,25,21),(95,97,102))):
                pos = (x*520+f*125,y*150)
                draw.rectangle((*pos,pos[0]+119,pos[1]+119),fill=floor)
                canvas.paste(sprite,pos,sprite)
            draw.text((x*520,y*150+125),name,fill='white')
    canvas.save('docs/agents/done/wave7-D-'+('before' if before else 'sheet')+'.png')

if __name__ == '__main__':
    make('--before' in sys.argv)

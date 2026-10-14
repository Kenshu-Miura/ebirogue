# 画像の作成: 追加する敵・武器・盾（Codex 用）

計画: `docs/agents/content-expansion-plan.md`。このタスクは**画像ファイルだけ**を作る。ゲームへの組み込み（`main.go` の読み込み、`draw.go` の分岐、テーブルへの追加）は後の実装担当が行うので、Go のゲームコードは変更しない。

最初に `AGENTS.md`（特に「新要素追加チェックリスト」の画像の項と Roadmap acceptance criteria）と `docs/agents/README.md` の「画像」を読むこと。`AGENTS.md` と `IMPLEMENTED_FEATURES.md` は編集しない。

## 2回に分ける

| タスク | ブランチ | 対象 | 枚数 |
|---|---|---|---|
| img-1 | `agent/img-content-1` | 優先 A: 武器 126〜146、盾 148〜162、敵 66〜89 | 21 + 15 + 24 = 60 |
| img-2 | `agent/img-content-2` | 優先 B・C: 武器 147、盾 163〜166、敵 90〜125 | 1 + 4 + 36 = 41 |

カードと指輪は既存の `img/card.png`・`img/ring.png` を共用するので作らない。

## 画像の規格（必須）

- 30x30 ピクセル、RGBA の PNG。四隅の画素は完全に透明（アルファ0）。
- 背景は透明。半透明の画素は輪郭の内側の影だけに使い、外側のにじみ（アルファ1〜254 の外周）は残さない。
- 被写体は 26〜28 ピクセル程度に収め、1 ピクセル以上の透明な余白を残す。
- 暗い床（RGB 38,25,21 前後と 95,97,102 前後。`img/tileset.png` の床タイル）の上で輪郭がわかること。被写体の外周に暗い色（ほぼ黒〜濃い茶）の1ピクセルの輪郭線を付ける。
- ファイル名は下の一覧のとおり（小文字のローマ字とアンダースコア）。既存のファイルを上書きしない。

## 絵柄（既存の画像に合わせる）

既存の `img/` を10枚ほど開いて合わせること（敵: `ebi.png`・`hayate_shako.png`・`fire_squid.png`・`sea_dragon.png`、武器: `weapon.png`・`dragon_killer.png`・`sure_hit_sword.png`、盾: `armor.png`・`counter_shield.png`）。

- 彩度の高いピクセルアート。光源は左上、3〜4段階の陰影。
- 敵: 海の生き物を、正面よりやや斜めから見た全身。顔（目）がわかるように描く。体の形で系統がわかることを優先し、細かい模様は 30x30 で潰れない範囲にとどめる。
- 武器: 左下が柄、右上が刃先の斜め45度。刃・柄・鍔の色を分ける。
- 盾: 正面から見た形。縁と中央の紋様で種類を見分けられるようにする。
- 同じ画面に並ぶ既存の敵と色・形が似すぎないようにする（特に、エビ系は赤橙、イシガニ系は灰茶、クラゲ系は青紫が多い）。

## 作り方

1. 画像生成の機能が使える場合は、それで原画（大きめの正方形）を作り、スクリプトで 30x30 に縮小・減色（32色程度）・輪郭線の付加・外周の半透明の除去を行う。使えない場合は Go か Python のスクリプトでピクセルアートを直接描く。どちらでもよいが、最後の加工はスクリプトで再現できるようにする。
   - **原画は git にコミットしない**（どのブランチにも入れない。ブランチに入れると clone のたびにダウンロードされ、マージすれば履歴からも消せなくなる。img-1 の原画約30MB は履歴に残ってしまった）。原画は `tools/img2-originals/`（`.gitignore` 済み）で作業し、残すのは生成に使ったプロンプト（`tools/img2-prompts.md`）と加工スクリプトだけにする。
   - img-1 の加工スクリプト `tools/process_expansion_sprites.py` は原画のフォルダと枚数を引数で受け取れるようにして使い回す（img-1 の原画は `agent/img-content-1` の `d9dbc42` にある）。
2. **敵の上位種は基本種の色違いで作る**（`AGENTS.md` の規則）。基本種の輪郭・アルファ・明暗を保ち、色相だけを回す。既存の `tools/third_stage_enemy_images.go` と同じ方式で、`tools/expansion_enemy_images.go` に「基本種 → 第2段階・第3段階と回転角」の表を書く。回転角の初期値は第2段階 +60°、第3段階 +150°（第2段階から）。灰色に近い基本種は彩度を少し足す。
   - 上位種が既存の敵や同じ系統の別段階と見分けにくい場合は回転角を変えてよい（理由を完了報告に書く）。
3. 検査と一覧画像のツール `tools/sprite_check.go`（`//go:build ignore`）を作る。指定したファイルについて次を検査し、違反があれば一覧を出して終了コード1で終わる。
   - 30x30、四隅が透明、外周1ピクセルの行・列に不透明な画素が無い
   - 不透明な画素の割合: 敵と盾は 30〜85%、武器は 7〜60%（既存の細い武器は 7〜17%、敵は 40〜70% 程度。種類はファイル名の一覧から判定する）
   - 床タイルの色と、被写体の外周の画素の明度差が十分あること（目安: 外周の平均明度が床より 20 以上低い、または高い）
   - あわせて、全画像を床タイルの上に4倍で並べた一覧画像を出力する（`-sheet <出力先>`）。
4. 一覧画像を自分で開いて目視で確認する。系統の見分けがつかない、輪郭が床に溶ける、30x30 で潰れて何かわからない、のどれかがあれば描き直す。

## ファイル一覧

### 敵（基本種／第2段階／第3段階）

| ID | 系統 | ファイル | 描く内容（基本種） | 段 |
|---|---|---|---|---|
| 66〜68 | ヒバナウミウシ | `hibana_umiushi` / `kaen_umiushi` / `gouka_umiushi` | 背中に炎のような突起のある、赤とオレンジのウミウシ | img-1 |
| 69〜71 | テッポウエビ | `teppou_ebi` / `taihou_ebi` / `gouhou_ebi` | 片方だけ大きな鋏を前に構えたエビ（鋏の先が砲口のよう） | img-1 |
| 72〜74 | ニゲハゼ | `nige_haze` / `tonzura_haze` / `kinpika_haze` | 走って逃げる姿勢の小さなハゼ。第3段階は金色が映える角度にする | img-1 |
| 75〜77 | ヨビコミボラ | `yobikomi_bora` / `yobiatsume_bora` / `daishoukan_bora` | 口から音符か波紋が出ているほら貝（巻き貝） | img-1 |
| 78〜80 | イヤシナマコ | `iyashi_namako` / `jihi_namako` / `seibo_namako` | 丸っこい薄緑のナマコ。頭の上に小さな光の輪かハート | img-1 |
| 81〜83 | ゼニガタアザラシ | `zenigata_azarashi` / `kinka_azarashi` / `senryou_azarashi` | 小銭を口にくわえたアザラシ | img-1 |
| 84〜86 | スナモグリ | `sunamoguri` / `sunakake_moguri` / `sunaarashi_moguri` | 砂から上半身を出し、砂粒を投げる構えのモグリ（甲殻類） | img-1 |
| 87〜89 | ヨロイエビ | `yoroi_ebi` / `kabuto_ebi` / `musha_ebi` | 兜と鎧をまとった武者のようなエビ | img-1 |
| 90〜92 | ウミダンゴ | `umi_dango` / `oo_umi_dango` / `daiou_umi_dango` | 丸まりかけのダンゴムシ型（オオグソクムシ風） | img-2 |
| 93〜95 | カエルアンコウ | `kaeru_ankou` / `oo_kaeru_ankou` / `nushi_kaeru_ankou` | 舌か疑似餌を伸ばすずんぐりした魚 | img-2 |
| 96〜98 | ハジキガニ | `hajiki_gani` / `kengou_gani` / `kensei_gani` | 片方の鋏を刀のように振り上げたカニ | img-2 |
| 99〜101 | サビヒトデ | `sabi_hitode` / `kusare_hitode` / `doro_hitode` | 錆色の斑点のあるヒトデ | img-2 |
| 102〜104 | イタズラカワハギ | `itazura_kawahagi` / `warusa_kawahagi` / `akutare_kawahagi` | 舌を出してにやけたカワハギ | img-2 |
| 105〜107 | オオハサミガニ | `oohasami_gani` / `guillotine_gani` / `danzai_gani` | 体より大きな鋏を両方掲げたカニ | img-2 |
| 108〜110 | フクレフグ | `fukure_fugu` / `bakuhatsu_fugu` / `daibakuhatsu_fugu` | 膨らんでトゲが立ったフグ。導火線のような尾 | img-2 |
| 111〜113 | アナジャコ | `ana_jako` / `oo_ana_jako` / `jigoku_ana_jako` | 前脚で掘る姿勢の細長い甲殻類 | img-2 |
| 114〜116 | シビレエイ | `shibire_ei` / `denki_ei` / `raitei_ei` | 上から見たエイ。体のまわりに小さな稲妻 | img-2 |
| 117〜119 | ミミックダコ | `mimic_dako` / `henge_dako` / `senmen_dako` | しま模様の細いタコ（正体を現したときの姿） | img-2 |
| 120〜122 | コバンザメ | `kobanzame` / `oobanzame` / `senryouzame` | 頭に小判形の吸盤がある細長いサメ | img-2 |
| 123〜125 | ウチカエシエビ | `uchikaeshi_ebi` / `hajikikaeshi_ebi` / `zenbukaeshi_ebi` | バットか櫂を振りかぶるエビ | img-2 |

### 武器

| ID | 名前 | ファイル | 描く内容 | 段 |
|---|---|---|---|---|
| 126 | カタナ | `katana` | 反りのある細身の刀 | img-1 |
| 127 | 剛剣 | `gouken` | 幅広で分厚い大剣 | img-1 |
| 128 | 金の剣 | `gold_sword` | 刀身も金色の剣 | img-1 |
| 129 | 甲殻斬り | `shell_cutter` | 鋸歯のある重い刀 | img-1 |
| 130 | ウニ割り | `uni_breaker` | 三日月形の刀身 | img-1 |
| 131 | 空の刃 | `sky_blade` | 水色に透ける刀身 | img-1 |
| 132 | 連撃刀 | `double_strike_sword` | 刀身が二重に見える刀（残像） | img-1 |
| 133 | 大斧 | `great_axe` | 両刃の大きな斧 | img-1 |
| 134 | 回復の剣 | `healing_sword` | 緑の刀身、鍔に十字 | img-1 |
| 135 | 小判こん棒 | `koban_club` | 小判がはめ込まれたこん棒 | img-1 |
| 136 | 鉄塊の大剣 | `iron_mass_sword` | 黒い鉄の塊のような大剣 | img-1 |
| 137 | 腹ペコ包丁 | `hungry_knife` | 肉切り包丁 | img-1 |
| 138 | 溜め打ちの剣 | `charge_sword` | 刀身に黄色い稲妻の模様 | img-1 |
| 139 | 四ツ叉の銛 | `four_prong_harpoon` | 先が4つに分かれた銛 | img-1 |
| 140 | 七福の熊手 | `lucky_rake` | 飾りの付いた縁起物の熊手 | img-1 |
| 141 | 木づち | `mallet` | 木の槌 | img-1 |
| 142 | 罠探りの棒 | `trap_probe` | 先が二股の細い杖 | img-1 |
| 143 | かつおぶし | `katsuobushi` | 刀の形をした鰹節 | img-1 |
| 144 | 呪いの包丁 | `cursed_knife` | 紫の気が立ちのぼる出刃包丁 | img-1 |
| 145 | 黄金のつるはし | `golden_pickaxe` | 金色のつるはし（既存の `shrimp_pickaxe.png` と形をそろえる） | img-1 |
| 146 | 火迅の太刀 | `kajin_tachi` | 炎をまとった太刀 | img-1 |
| 147 | 巨大ハリセン | `giant_harisen` | 大きな紙のハリセン | img-2 |

### 盾

| ID | 名前 | ファイル | 描く内容 | 段 |
|---|---|---|---|---|
| 148 | 青銅甲の盾 | `bronze_shield` | 青緑がかった青銅の盾 | img-1 |
| 149 | 大海の盾 | `taikai_shield` | 波の紋様の大きな盾 | img-1 |
| 150 | 金の盾 | `gold_shield` | 金色の盾 | img-1 |
| 151 | 竜鱗の盾 | `dragon_scale_shield` | 鱗が並ぶ盾 | img-1 |
| 152 | にぎりよけの盾 | `nigiri_guard_shield` | 中央に×印の付いたおにぎりの紋 | img-1 |
| 153 | 操りよけの盾 | `puppet_guard_shield` | 中央に目と×印 | img-1 |
| 154 | 小判の盾 | `koban_shield` | 小判を並べた盾 | img-1 |
| 155 | 鉄塊の大盾 | `iron_mass_shield` | 黒い鉄板の大盾 | img-1 |
| 156 | 腹ペコの盾 | `hungry_shield` | 皿の形の盾 | img-1 |
| 157 | 根性の盾 | `guts_shield` | 中央に拳か炎の紋 | img-1 |
| 158 | 絶好調の盾 | `peak_shield` | 中央に太陽の紋 | img-1 |
| 159 | 悟りの盾 | `satori_shield` | 白い円相（禅の丸）の紋 | img-1 |
| 160 | 七福の御守 | `lucky_charm` | 赤いお守り袋 | img-1 |
| 161 | 呪いの鍋ぶた | `cursed_pot_lid` | 紫の気が立ちのぼる鍋ぶた | img-1 |
| 162 | カメの甲羅 | `turtle_shell` | ウミガメの甲羅 | img-1 |
| 163 | 痛恨よけの盾 | `critical_guard_shield` | 角の生えた分厚い盾 | img-2 |
| 164 | いたずらよけの盾 | `mischief_guard_shield` | 中央に鍵の紋 | img-2 |
| 165 | 不動の盾 | `immovable_shield` | 地面に杭で打ち込まれたような盾 | img-2 |
| 166 | ざぶとん | `zabuton` | 紫の座布団 | img-2 |

## 完了条件

- 一覧のファイルがすべて `img/` にあり、`go run tools/sprite_check.go <ファイル…>` が違反なしで終わる。
- 一覧画像を `docs/agents/done/img-content-1-sheet.png`（img-2 は `-2-`）に保存し、完了報告から参照する。
- `docs/agents/done/img-content-1.md`（img-2 は `-2`）に、作り方（生成かスクリプトか、加工の手順と再実行のコマンド）、上位種の回転角を初期値から変えたものとその理由、描き直したものとその理由、ゲーム内で確認してほしい点を書く。
- ゲームのコードは変更していないので、`go test -tags test ./...` と `go test ./...` が従来どおり成功することだけ確認する。
- ブランチにコミットしてプッシュする（master へは直接プッシュしない）。

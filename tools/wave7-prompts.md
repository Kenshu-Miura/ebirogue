# wave7-D 原画の生成

画像生成機能で透過背景・4列4行のatlasを生成。原画はコミットしない。

プロンプト: SNES roguelike pixel art, sharp pixels, vibrant limited palette,
dark brown outline, three-tone shading, upper-left light, transparent background.
Exactly four columns and four rows, equal cells, generous margins, no labels.
Row 1: spiral warp rune stone plate; broken pit; steel bear trap; wooden arrow launcher plate.
Row 2: iron arrow launcher plate; falling boulder plate; large spiked landmine; unequip helmet/armor plate.
Row 3: green spoil fumes plate; three scattered bags with arrows plate; moldy sausage with flies; crowned armored king crab.
Row 4: purple armored octopus general with gold samurai helmet; red-gold crowned sea dragon king; empty; empty.
Traps top-down; monsters three-quarter full body, visible eyes, chunky silhouettes suitable for 28px.

加工: `python tools/process_wave7_images.py tools/wave7-originals/atlas.png`。
既存の `process_expansion_sprites.py` を再利用し、アルファ閾値160、
26pxへ縮小、31色に減色、外周の半透明除去、暗色1px輪郭を付加して30x30 RGBAへ。

上位種: `go run tools/third_stage_enemy_images.go`。
オンリョウガニ・ジクウクラゲのみ第2段階から+60度へ変更。
残りは+150度のまま。輪郭・アルファ・HSLの明度は保存。
旧ボス・ダメージ罠・所持品罠スクリプトは誤上書き防止のため実行を停止。

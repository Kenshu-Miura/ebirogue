# img-2: 追加コンテンツ用の画像（後半）

作業日: 2026-10-14。ブランチ: `agent/img-content-2`。
起点: master `9ce37430773946d4d5efa5afb4c260cfc5ca8e8a`。

## 成果物

- 新規41枚: 武器147の1枚、盾163〜166の4枚、敵90〜125の36枚。
- 全数30×30 RGBA PNG、四隅と外周1pxは透明、外側の半透明なし。
- [床上の4倍一覧](img-content-2-sheet.png)。左はRGB(38,25,21)、右はRGB(95,97,102)。
- 全ファイル名: [tools/img2-files.txt](../../../tools/img2-files.txt)。
- ゲームコード・AGENTS.md・IMPLEMENTED_FEATURES.mdは変更なし。既存画像も変更なし。

## 作り方と再実行

既存の ebi、hayate_shako、fire_squid、sea_dragon、weapon、dragon_killer、
sure_hit_sword、armor、counter_shield、shrimp_pickaxe の10枚を開いて参照。
built-in image_genで透明背景の原画17枚を個別生成し、上位種24枚は色相回転で作成した。
[生成プロンプト](../../../tools/img2-prompts.md) に共通文と各原画のSubjectを保存。

原画は `tools/img2-originals/` に保持し、Gitには含めない。
原画の再生成は非決定的であり、別環境で同一の最終画像を原画から再加工するには、
このフォルダをGit以外で引き渡す必要がある。最終PNGはGitで配布される。

img-1の `process_expansion_sprites.py` に `--originals`・`--count` を追加して再利用。
既定値はimg-1のまま。アルファ160未満を除去、切り抜き、最大26pxへLANCZOS縮小、
アルファ128で二値化、31色へ減色、8近傍へRGB(3,2,1)の1px輪郭を追加。
輪郭を含む最大サイズ28pxで中央配置する。

リポジトリ直下で実行（Python + Pillow、Go）:

```powershell
python tools/process_expansion_sprites.py --originals tools/img2-originals --count 17
go run tools/expansion_enemy_images.go
$img2Files = Get-Content tools/img2-files.txt
go run tools/sprite_check.go -sheet docs/agents/done/img-content-2-sheet.png $img2Files
python tools/verify_img2.py
```

色違い生成ツールはimg-1とimg-2の両方を処理する。
既存img-1のPNGが変わらないことをGit差分で確認した。
`verify_img1.py` はimg-1の名前で変換表を絞り、表の追加後も従来の16変換を検証できるようにした。

## 上位種の回転角と目視確認

12系統すべて基本→第2を+60°、第2→第3を+150°とし、初期値からの変更はなし。
HSLの明度とアルファを保持。彩度の低い画素には既存方式の彩度補正を適用。

途中の一覧と最終41枚の一覧を自分で開いた。
2色の床で輪郭、系統ごとの体形、各段階の色の違いを確認した。
武器・盾の紙の折り目、角、鍵、杭、座布団の形も確認。
刀状の片鋏と巨大な両鋏、ひし形のエイ、細長いサメ、しま模様のタコ、
櫂を構えるエビを形で区別できる。描き直した素材はなし。

## 検証

- `sprite_check.go`: 全41枚、違反ゼロ。
- 再加工・上位種再生成後の41枚のSHA-256一致。
- 上位種24枚のアルファ完全一致、HSL明度は丸め誤差の範囲で一致。
- 変更したGoスクリプトをgofmt済み。
- `go test -tags test ./...`: 成功。
- `go test ./...`（TestSimSmoke含む）: 成功。
  初回はサンドボックスのTempDir cleanup・Chdirのアクセス拒否で停止したため、
  通常権限で同じコマンドを再実行して成功を確認した。
- 原画は `.gitignore` の対象であり、コミット対象に含めない。

## 統合時にゲーム内で確認してほしい点

画像の読み込み・描画分岐・テンプレート登録は実装担当の範囲。
実際のゲーム画面での確認は未実施。組み込み後に実寸で、
アンコウの疑似餌、カワハギの舌、フグの導火線状の尾、エイの稲妻、
タコのしま模様、サメの吸盤、エビの櫂が判別できるか確認してほしい。
杭付き盾と床の境界、既存敵と各上位種の色の区別も確認してほしい。

IMPLEMENTED_FEATURES.mdへの追記案: 素材のみのため現時点ではなし。
AGENTS.mdから削除するロードマップ項目: なし。
正式画像として作成。仮画像として差し替えが必要な素材はなし。

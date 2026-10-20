# wave7-D: 仮画像の差し替え（画像生成が使える担当用）

ブランチ: `agent/wave7-D-images`
前提: master の最新。`AGENTS.md`、`docs/agents/README.md` の「画像」、`docs/agents/img-content-expansion.md`（画像の規格・絵柄・作り方。**原画をコミットしない規則**を含む）を先に読む。ゲームのコードには触れない（画像ファイルの上書きだけ）ので、ほかのどのタスクとも並行してよい。

## 差し替える画像（スクリプトで作った単純な図形・色相回転の仮画像）

| 種類 | ファイル | 今の仮画像 | 正式な画像の方針 |
|---|---|---|---|
| 移動罠 | `warp_trap.png`・`pitfall_trap.png`・`bear_trap.png` | 単純図形 | 床に描かれた罠として、既存の罠画像（`img/` の罠）と同じ見下ろしの絵柄 |
| ダメージ罠 | `wood_arrow_trap.png`・`iron_arrow_trap.png`・`falling_rock_trap.png`・`large_mine_trap.png` | 単純図形（`tools/damage_trap_images.go`） | 同上 |
| 装備・所持品の罠 | `unequip_trap.png`・`spoil_trap.png`・`scatter_trap.png` | 単純図形（`tools/item_trap_images.go`） | 同上 |
| 食料 | `rotten_sausage.png` | `sausage.png` の色変え | 腐ったことがわかる（緑がかった色・湯気・ハエ）。`sausage.png` と並べて見分けられる |
| ボス | `boss_crab.png`・`boss_octopus.png`・`boss_dragon.png` | 既存の敵の色相回転（`tools/boss_images.go`） | 30x30 の中で通常の敵より大きく見える（余白を減らし、冠・鎧などの飾りで格上とわかる）。鉄壁ガニ王・深海ダコ将軍・海竜王アカツキ |
| 第3段階の敵 | ID 45〜65 の画像（`tools/third_stage_enemy_images.go` が作るもの） | 第2段階の色相を150度回したもの | **描き直すのは、基本種や他系統と色が似て見分けにくいものだけ**。まず全21枚を基本種・第2段階と並べた比較シートを作り、似ているものを列挙してから直す。上位種は色違いの規則（AGENTS.md）を守る |

## 進め方

1. 差し替え前の比較シート（今の画像を暗い床の上に並べた PNG）を作る。`docs/agents/done/img-content-1-sheet.png` と同じ形式。
2. 画像を作り、同じファイル名で上書きする（ゲームのコードを変えずに差し替わる）。
3. 差し替え後の比較シートを作り、`docs/agents/done/wave7-D-sheet.png` として残す。
4. 色相回転のスクリプト（`tools/boss_images.go` など）が正式な画像を上書きしないよう、スクリプトの先頭に「正式な画像に差し替え済み。再実行しない」と書くか、出力先を変える。

## 完了条件

- 全画像が 30x30・RGBA・四隅が透明（`go test ./...` の画像の検査が通る）。
- `docs/agents/done/wave7-D.md`: 差し替えた画像の一覧、描き直さなかった第3段階の敵とその理由、AGENTS.md の「仮画像を正式な画像へ差し替える」の削除案。実機での視認性の確認は開発者が行う。
- コミットしてブランチをプッシュする（原画はコミットしない）。

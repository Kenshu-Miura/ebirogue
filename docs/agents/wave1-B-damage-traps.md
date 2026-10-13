# wave1-B: ダメージ罠（Codex 用）

ブランチ: `agent/wave1-B-damage-traps`
前提: フェーズ0（`docs/agents/phase0-foundation.md`）が master にマージ済みであること。
最初に `AGENTS.md` と `docs/agents/README.md` を読み、並列作業のルールに従うこと（`AGENTS.md` と `IMPLEMENTED_FEATURES.md` は編集しない）。

## 作るもの

| 罠 ID | 名前 | 効果 |
|---|---|---|
| 8 | 木の矢の罠 | 小ダメージ |
| 9 | 鉄の矢の罠 | 中ダメージ |
| 10 | 落石の罠 | 中ダメージ |
| 11 | 大型地雷 | 既存の地雷（現在 HP の半分）より強いダメージ。周囲8マスの敵にもダメージ |

## 仕様

- 罠の追加手順は `AGENTS.md` の「新要素追加チェックリスト」に従う。
- ダメージ値は既存のバランス（初期 HP 100、地雷 = 現在 HP の半分、毒矢の罠）に合わせて決める。値は `status.go` などの定数にまとめ、根拠を完了報告に書く。
  原作の数値をそのまま使わない。
- 防御力で軽減するかどうかを罠ごとに決め、軽減する場合は `damage.go` の `rollDamage` 系を経由する。
- 大型地雷の周囲の敵へのダメージは `g.applyDamageToEnemy` を使う（撃破と経験値の処理を手書きしない）。
- 罠のダメージは `g.damagePlayer(ダメージ, trapSource(罠ID, 罠名))` を通し、死因に罠 ID を記録する。
- 乱数はすべて `g.randIntn()` / `g.random()` 経由にする（フロア生成中は生成用の乱数源を使う）。
- 罠の抽選一覧（`maptraps.go` の `trapIDs`）に新しい ID を追加する。wave1-A も同じ行を編集するため、既存の要素を並べ替えず末尾に追記する。
- 矢・投擲・爆発で罠を起動する仕組みと、敵が罠を踏む処理は wave2 で追加する。
- 新しい罠の画像は、30x30 の透過 PNG をスクリプトで仮作成してよい。仮画像は完了報告に列挙する。

## 完了条件

- `gofmt -w`、`go test -tags test ./...`、`go test ./...`、`go test -run TestSimSmoke -count=1 .` がすべて成功する。
- 罠ごとのテスト: ダメージ量の範囲、不発、最低ダメージ、死因の記録、大型地雷による周囲の敵の撃破。
- `docs/agents/done/wave1-B.md` に次を書く: 変更概要、ダメージ値とその根拠、IMPLEMENTED_FEATURES.md への追記案、AGENTS.md から削除するロードマップ項目、仮画像の一覧。
- ブランチにコミットしてプッシュする（master へは直接プッシュしない）。

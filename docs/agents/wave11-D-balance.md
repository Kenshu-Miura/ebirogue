# wave11-D: クリア率の調整（初級80%・中級45%・上級5%）

前提: wave11-A・B・C の統合後。**1人で行う**（ゲームの数値を触るのはこの担当だけ）。

最初に `AGENTS.md`、`docs/agents/README.md`、`docs/agents/STATUS.md`、`docs/agents/feature-completion-plan.md`（「段11」）、`docs/agents/done/wave11-0.md`〜`wave11-C.md`、`docs/agents/done/wave9-A.md`（「つまみの一覧と使い方」）、`docs/agents/balance-level99.md` を読むこと。

## 目標（開発者の決定、2026-10-21）

| 難易度 | 目標 | 合格の範囲（500冒険） |
|---|---|---|
| 初級（10階ボス） | 80% | 77〜83% |
| 中級（20階ボス） | 45% | 41〜49% |
| 上級（30階ボス） | 5% | 3〜7% |

- 以前の目標（85%・50%・10%）より**少し難しい調整**にする、という開発者の方針。
- 測る BOT は expert。判定は各難易度500冒険（`-sim.persona=expert -sim.runs=1500 -sim.difficulty=all`）。範囲の端にあるときは `-sim.seedbase` を変えてもう1回測り、2回の平均で決める。

## 進め方

1. 11-A/B/C の統合後の expert のベースライン（`reports/baseline-expert.json`、統合担当が取り直す）から始める。BOT が強くなった分、クリア率は今より上がっている見込み。
2. **難易度ごとのつまみを先に使う**。共通の数値（敵の強さ・全難易度の出現テーブル）を変えると、ほかの難易度のクリア率も動くため。
3. 1回に変えるつまみは1つか2つにし、変えるたびに1500冒険を測って、変えた値・クリア率・平均到達階・死因の上位の変化を完了報告の表に1行ずつ足す。

## つまみ（変えてよい順）

| 順 | つまみ | 場所 | 効き方 |
|---|---|---|---|
| 1 | 難易度ごとの補正（食料・回復薬の倍率 `FoodHealWeight`、強い道具の倍率 `StrongItemWeight`、道具の数 `ItemCount`、罠の数 `TrapCount`） | `item_floor_tables.go` の `difficultySpawnTunings`（今はすべて 1.0） | その難易度だけ。値を入れたら `TestDifficultySpawnTuningNeutral` を「値が範囲内」のテストに直す |
| 2 | モンスターハウスの出現率（4〜9階5%・11階以降8%）と密度（床の1/3・1/5・1/20） | `special_rooms.go` の `monsterHouseChance`・`monsterHouseCounts` | 4階以降。難易度で変えたいなら難易度を引数に足す |
| 3 | 地形の出現率（水路45%・割れた壁50%・隠し通路40%） | `terrain.go` の `terrainRates` | 2階以降 |
| 4 | 店の出現率（10%）・商品の数（5〜12） | `special_rooms.go` の `shopChancePercent` ほか | 3階以降 |
| 5 | 強い武具が出始める階（今は1階から） | `item_floor_tables.go` の `itemFloorRules` の `From` | wave9-A の未決事項。遅らせると中級が大きく下がった |
| 6 | 敵の出現階・上位種の出方 | `monster_spawn.go` など | 最後の手段。全難易度に効く |

- 「階層で敵の能力値は変えない（深い階の強さは上位種の出現で表す）」という確定した仕様は守る。
- 初級の10階までの階は中級・上級でも通るので、初級のために1〜10階の共通の数値を変えると全難易度が動く。初級だけ変えたいときは 1 の難易度ごとの補正で。

## あわせて決めること

- **到着時のレベル**: 目標の曲線は expert の到着時平均で 10階 Lv13〜16・20階 Lv22〜26・30階 Lv30〜35（`balance-level99.md`）だが、以前の計測では 10階 Lv18〜21 と上回っていた（STATUS.md の 5c）。クリア率を合わせた後の値を報告し、曲線の目標を変えるか経験値を見直すかの案を、根拠の数字と一緒に未決事項に書く（**経験値の変更は開発者の決定を待つ**）。
- 調整で変えた数値は `IMPLEMENTED_FEATURES.md` の該当項目の案を完了報告に書く。

## 完了条件

- 3つの難易度がすべて合格の範囲に入る（各500冒険、範囲の端なら2回の平均）。
- gofmt、`go test -tags test ./...`、`go test ./...`（`TestSimSmoke` を含む）が成功。つまみの値を固定していたテストは、値の範囲を確かめるテストに直す。
- 違反・ソフトロック・panic 0、BOT 行き詰まり 0（expert）。
- 最後に全性格の `TestSimBatch`（`-sim.runs=1500 -sim.difficulty=all -sim.writebaseline`）と expert のベースライン（`reports/baseline-expert.json`）を取り直してコミット。
- 完了報告は `docs/agents/done/wave11-D.md`（変えたつまみの履歴の表、最終のクリア率、到着時のレベル、死因の上位、未決事項）。

# wave7-A 完了報告（状態異常の共通化）

ブランチ: `agent/wave7-A-status-foundation`（指示書どおり）。

## 変更概要

新しい状態異常は作っていない。今ある状態異常の付与・経過・解除・表示・保存の経路を1か所にまとめた。

- `status_effects.go`（タグなし・純粋）: 状態の定義表 `statusDefs` と純粋な共通処理。
  - 表の列: 種類 `Kind`、名前 `Name`（「〜を防いだ」に使う）、HUD の表示名 `HUD` と並び順 `HUDOrder`、海老さん側・敵側の持続の型 `Player`・`Enemy`（`StatusNotApplicable` 付かない／`StatusTurns` 毎ターン減る／`StatusPermanent` 減らない／`StatusUntilCleared` bool で解除されるまで）、重ねがけの規則 `Stack`（`StackOverwrite` 上書き／`StackLonger` 長い方／`StackAdd` 加算して `MaxTurns` まで）、悪い状態か `Bad`、海老さんにかかったときのメッセージ `PlayerApply`（統合後に追加）、海老さんが治ったときのメッセージ `PlayerRecover`、治ったときにミニマップを描き直すか `RefreshMiniMap`、ボスの行動時に短縮するか `BossShortened`、項目の読み書き `Turns`・`SetTurns`。
  - 純粋な関数: `statusDefOf`・`statusTurns`・`statusActive`・`stackStatusTurns`・`addStatus`（規則を通してかける）・`setStatusTurns`（規則を通さない。仮眠の -1 用）・`clearStatus`・`hasBadStatus`・`clearBadStatuses`・`markHasteOnWake`・`wakeFromSleep`（`status.go` から移動）・`tickStatuses`（1ターン進めて治った種類を返す）・`formatPlayerStatus`（HUD。`status.go` から移動し表から出す）。
  - 表に載せた種類: トラバサミ・混乱・睡眠・目潰し・鈍足・倍速・口封じ・ゾワゾワ・金縛り・封印。`StatusKind` に `StatusHaste`・`StatusMouthSeal`・`StatusParalysis`・`StatusSeal`・`StatusBackward` を追加。`statusKindLabel` は表の名前を優先する（呪い・さびなど表に無い種類は従来の switch）。
- `status_game.go`（`!test`）: `*Game` を使う共通処理。
  - 海老さん: `g.applyPlayerStatus(kind, turns, StatusSource{...}) bool`（耐性→重ねがけ→メッセージ。戻り値は付いたか）、`g.setPlayerStatus(kind, turns)`（耐性の判定を済ませた後に表の規則でかける）、`g.stackPlayerStatus(kind, turns, rule)`（規則を指定）、`g.guardPlayerStatus(kind, source)`、`g.clearPlayerStatus(kind)`、`g.clearPlayerBadStatuses()`。
  - `StatusSource`: `Shield`（状態異常耐性の盾の50%でも防げるか。false なら装備能力の無効化だけ）、`Stack`（規則の上書き）。
  - **統合後の変更（2026-10-21、開発者の決定）**: メッセージと表示時間を統一した。`StatusSource` の `Message`・`Blocked` は廃止し、かかったときは表の `PlayerApply`、防いだときは「<装備品>が<状態の名前>を防いだ」（`statusResultMessage`）、表示時間はかかった・防いだ・治ったのすべてで `statusMessageDuration`（0.5秒）。原因のメッセージ（「〜を飲んだ」「〜のくねくね踊り」など）は付与元が先に出す。耐性の判定を Enqueue の時点で済ませる付与は `g.enqueuePlayerStatus(kind, turns, rule, guard, blocked)`（表示時にかける）、Execute の中ですぐかけるなら `g.applyPlayerStatusResult(...)`。乱数の消費は変えていない（`TestSimBatch` 1512冒険の集計はベースラインと完全一致）。
  - 敵: `g.applyEnemyStatus(index, kind, turns) bool`（ボスが受け付けない状態なら false）、`applyStatusToEnemy(*Enemy, …)`、`g.clearEnemyStatus(index, kind)`。
  - 経過: `g.decrementStatusAilments()` は表を回す形にし、治ったときのメッセージ・ミニマップの描き直しも表から出す。敵の睡眠のカードの目覚めの倍速（`HasteOnWake`）はそのまま残した。
- `bosses.go`: ボスが受け付けない状態を表 `bossStatusImmunities`（今は最終ボス44の金縛りだけ）、短縮する状態を定義表の `BossShortened`（睡眠・混乱・ゾワゾワ）から引く形にした。判定 `bossStatusImmune`。
- 困った時のカード（`troubleToFix`・`troubleCard`）は表の `Bad` を使う（`hasBadStatus`・`clearPlayerBadStatuses`）。
- 付与・解除の箇所をすべて置き換えた: 罠（睡眠ガス・鈍足・トラバサミ）、薬（飲む・投げる）、カード（睡眠・混乱・目潰し・金縛り・倍速・口封じ・ゾワゾワ・真空斬り・困った時）、杖（封印の杖を振る・投げる）、敵の特殊攻撃（くねくね踊り・砂かけ・深海ダコ将軍の墨）、ダメージで金縛り・睡眠が解ける処理（通常攻撃・敵どうしの攻撃・爆発・遠距離の範囲攻撃・`applyDamageToEnemy`）、仮眠と起床（`map.go`・`monster_spawn.go`）。BOT は状態を読むだけで書き換えていない（テストの準備で直接書く箇所はそのまま）。
- 保存: 構造体 `StatusAilments` の項目名は変えていない（`SavedPlayer`・`SavedEnemy` が構造体ごと保存する）。`saveDataVersion` は上げていない。

### 挙動を変えないための判断

- 既存の付与の多くは「耐性の判定を Enqueue の時点で行い、状態をかけるのは Execute の中」という2段になっている。耐性の判定（盾の50%の乱数）の時点を動かすと乱数の消費の順が変わるため、既存の箇所は判定をそのまま残し、かける部分だけ `setPlayerStatus`／`stackPlayerStatus` にした。`applyPlayerStatus`（判定とかけるのを同時に行う）は新しい付与（wave8 以降）用。
- 重ねがけの規則が同じ状態でも箇所で違うものは、表の既定と違う箇所だけ規則を指定した: くねくね踊りの鈍足（8ターン、長い方）、深海ダコ将軍の墨の目潰し（長い方）。ほかは表の既定（トラバサミだけ長い方、ほかは上書き）。
- ボスの睡眠・混乱・ゾワゾワの短縮は、今までどおりボスの行動時（`normalizeBossStatus`）に行う。付与の時点で短縮するとボスの起きるターンが1ターン早まるため。
- 最終ボスの金縛りは、以前は「かかる→行動時に解ける」だったのを「かからない」にした。その間に金縛りを読む処理は無い（金縛りのカードでターンが終わり、次はボスの行動）ので結果は同じ。

### 残る直接の書き換え

`grep -rn 'StatusAilments\.[A-Za-z]* *[-+]\?=' --include=*.go . | grep -v _test` で `==` 以外の書き換えは0件。項目への代入が残るのは共通処理の内部だけ:

- `status_effects.go` の定義表の `SetTurns`（各項目の書き込み）。
- `status_effects.go` の `clearBadStatuses`・`markHasteOnWake`・`wakeFromSleep` の `HasteOnWake`（状態そのものではなく睡眠に付く印なので表に載せていない）。

## 新しい状態異常の追加手順（wave8 の担当向け）

1. `StatusAilments` に項目を1つ足す。**`main.go` と `game_stub.go` の両方**に足す（タグなしの `status_effects.go` が両方のビルドで使う）。残りターンなら `int`、有無だけなら `bool`。中断セーブは構造体ごと保存されるので、ほかに何もしなくても保存・復元され、古いセーブは0（false）で読める。
2. `equipment_abilities.go` か `status_effects.go` に `StatusKind` の定数を足す（例 `StatusAttackUp StatusKind = "attack_up"`）。
3. `status_effects.go` の `statusDefs` の**末尾**に1行足す。決めること:
   - `Name`（防いだときの「〜を防いだ」）、`HUD` と `HUDOrder`（HUD の並び。今の最大は100）。
   - `Player`・`Enemy`: その側に付かないなら `StatusNotApplicable`（何も書かない）、毎ターン減るなら `StatusTurns`、減らないなら `StatusPermanent`、bool なら `StatusUntilCleared`。
   - `Stack` と、`StackAdd` なら `MaxTurns`。
   - `Bad`: 困った時のカード（と今後の治す薬）で治すなら true。
   - `PlayerApply`: 海老さんにかかったときのメッセージ（「海老さんは〜」）。付与元にかかわらずこれが出る。
   - `PlayerRecover`: 海老さんの状態が治ったときのメッセージ（「海老さんは〜」）。
   - `BossShortened`: ボスにかかったとき短縮するか。ボスが受け付けない状態は `bosses.go` の `bossStatusImmunities` に足す。
   - `Turns`・`SetTurns`: 足した項目を読み書きする関数（bool なら `boolTurns(s.X)` と `s.X = n > 0`）。
4. 状態異常耐性の盾（50%）・よけの指輪で防ぐ対象にするなら、付与のときに `StatusSource{Shield: true}` を渡す。確実に無効にする能力は `statusImmunities` に能力 ID → 種類を足す。
5. 付与は `g.applyPlayerStatus(kind, turns, StatusSource{...})`・`g.applyEnemyStatus(index, kind, turns)` を通し、`StatusAilments` の項目を直接書き換えない。解除は `g.clearPlayerStatus`・`g.clearEnemyStatus`。
6. 状態の効果そのもの（倍速で敵の行動を飛ばす、攻撃力を変えるなど）は、効く場所で `statusActive(s, kind)` か項目を読んで書く。
7. テスト: `status_effects_test.go` の `TestStatusDefsTable`（項目の読み書きが他の項目に触れないこと・名前など）が新しい行も自動で確かめる。付与・経過・HUD の文字列・中断セーブ→再開は担当のテストで確かめる。

これで付与・ターン経過・治ったときのメッセージ・HUD・保存・困った時のカードに反映される。

## IMPLEMENTED_FEATURES.md への追記案

「状態異常と一時効果」の節に追記:

- 状態異常は定義表（`status_effects.go` の `statusDefs`）で管理する: トラバサミ・混乱・睡眠・目潰し・鈍足・倍速（敵）・口封じ・ゾワゾワ（敵）・金縛り・封印。表に名前・HUD 表示・持続の型（ターン／減らない／解除されるまで）・重ねがけの規則（上書き・長い方・加算して上限まで）・治ったときのメッセージ・海老さん／敵に付くか・悪い状態か・ボスで短縮するかを持ち、付与（`g.applyPlayerStatus`／`g.applyEnemyStatus`）・ターン経過・HUD・困った時のカードがこの表を使う。
- 重ねがけ: トラバサミ・くねくね踊りの鈍足・深海ダコ将軍の墨は長い方を残し、ほかは上書き。敵の目潰しはターン経過で治らない。ボスは睡眠・混乱・ゾワゾワが短縮され、最終ボスは金縛りにかからない。

## AGENTS.md の変更案

- Implementation roadmap の「Priority A: traps and status ailments」から次の行を削除する:
  - 「状態異常は残りターン、重複時の規則、解除手段、耐性を共通処理へまとめる。可能な状態はプレイヤーと敵の双方へ適用できるようにする。」
- 「ファイル分担」に追記する案:
  - `status_effects.go`（タグなし・純粋）と `status_game.go`（`!test`）: 状態異常の定義表 `statusDefs` と付与・解除・経過の共通処理。海老さんへの付与は `g.applyPlayerStatus(kind, turns, StatusSource{...})`（耐性・重ねがけ・メッセージ）、敵へは `g.applyEnemyStatus(index, kind, turns)`（ボスの耐性 `bossStatusImmunities`）。**`StatusAilments` の項目を直接書き換えない。** 新しい状態の追加手順は `docs/agents/done/wave7-A.md`。
- 「ビルドタグの仕組み」の表のタグなしの列に `status_effects.go` を追加。
- 「主要データ構造」の `StatusAilments` の説明に「種類ごとの規則は `statusDefs`」を追記。

## 仮画像

なし。

## テスト

- `content_status_foundation_test.go`（通常ビルド）: 置き換えの前の挙動を固定したテスト。付与の入口（罠・薬・カード・敵の特殊攻撃）だけを使うので、共通化の前のコードでもそのまま通ることを確かめた（master の作業ツリーに写して実行し、6件すべて成功）。
  - 状態ごとの「2回かけたとき」（睡眠ガス・鈍足の罠・トラバサミ・くねくね踊り・睡眠薬・混乱薬・目潰し薬・口封じのカード）。
  - 耐性: 盾の50%で防ぐ付与（罠）と、装備能力の無効化だけを見る付与（トラバサミ・飲んだ薬）。
  - 治ったときのメッセージとその順序、ミニマップの描き直し。敵の経過（目潰し・仮眠は減らない、睡眠のカードの目覚めの倍速）。ダメージで金縛り・仮眠が解ける。困った時のカードが悪い状態だけを解く。
  - 新しい共通処理: `applyPlayerStatus`（メッセージ・規則の指定・能力の無効化・盾の判定は `Shield` のときだけ・海老さんに付かない状態）、`applyEnemyStatus`（最終ボスの金縛り、短縮は行動時、範囲外の添字）。
- `status_effects_test.go`（両ビルド）: 表の整合（重複・読み書きの対応・名前）、重ねがけの規則、付与と解除、ターン経過、悪い状態、HUD の並び。

## 検証

- `gofmt -l .` で差分なし。
- `go test -tags test ./...`・`go test ./...`: 成功（クラウドの Linux では `xvfb-run -a` で実行。ディスプレイが無いと Ebiten の初期化で止まる）。
- `TestSimBatch -sim.runs=1500 -sim.difficulty=all`: 1512冒険すべてが共通化の前のコード（master と同じ HEAD `b3224c9`）と冒険ごとに完全一致し、集計も `reports/baseline.json` と完全一致（クリア 155・死亡 1348・BOT 行き詰まり 9、違反・ソフトロック・panic 0）。乱数の消費は変わっていない。
- `TestSimBatch -sim.runs=300 -sim.difficulty=all`: 306冒険でクリア 33・死亡 272・BOT 行き詰まり 1、違反・ソフトロック・panic 0。

## 未決事項

- 既存の付与のメッセージ（「海老さんは眠った」「〜が睡眠ガスを防いだ」など）と表示時間は箇所ごとに違うので、そのまま残した。`applyPlayerStatus` に寄せてメッセージを統一するかは、挙動（表示）が変わるので開発者の判断を待つ。
- 既存の付与では耐性の判定の仕方（盾の50%も見るか、能力の無効化だけか）が付与元で違う（罠・敵は盾も見る、飲んだ薬・投げられた薬・トラバサミは能力だけ）。今の挙動をそのまま残した。
- 敵の目潰しはターン経過で治らない（目潰しのカード10ターン・目潰し薬999ターンのどちらも、以前から減っていなかった）。表では敵側を `StatusPermanent` にして今の挙動を残した。カードの「10ターン」に合わせて減らすかは開発者の判断。
- 海老さんの倍速・ゾワゾワ、敵のトラバサミ・口封じは今は付かない状態として扱った（`StatusNotApplicable`。経過もしない）。HUD は値があれば表示する。海老さんの倍速は wave8-A が `Player: StatusTurns` にする。
- 封印・金縛りの bool の状態は「解除されるまで」。海老さんにかける手段は今は無いが、困った時のカードの対象には入っている（従来どおり）。

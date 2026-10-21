//go:build !test

package main

// wave7-A: 状態異常の共通化の前後で挙動が変わらないことを固定するテスト。
// 付与の入口（罠・薬・カード・敵の特殊攻撃）とターン経過だけを使い、共通化の前のコードでも同じ結果になる。

import (
	"strings"
	"testing"
)

// usePlayerItem は持ち物の先頭に道具 id を入れて使い、メッセージを返す。
func usePlayerItem(g *Game, id int) string {
	queuePlayerItem(g, id)
	return runQueuedActions(g)
}

// queuePlayerItem は持ち物の道具 id を使い、積まれた Action を実行せずに残す。
func queuePlayerItem(g *Game, id int) {
	g.state.Player.Inventory = []Item{buildItemFromTemplate(id, 0, 0)}
	switch item := g.state.Player.Inventory[0].(type) {
	case *Potion:
		item.Use(g)
	case *Card:
		item.Use(g)
	}
}

// TestPlayerStatusStackingRules は状態ごとに「2回かけたとき」の残りターンを固定する。
func TestPlayerStatusStackingRules(t *testing.T) {
	cases := []struct {
		name  string
		apply func(g *Game)
		get   func(s *StatusAilments) *int
		// before: かける前の残りターン → want: かけた後
		before, want []int
	}{
		{"睡眠ガスの罠（上書き）", sleepGasTrapEffect, func(s *StatusAilments) *int { return &s.Sleep }, []int{0, 3, 20}, []int{10, 10, 10}},
		{"鈍足の罠（上書き）", slowTrapEffect, func(s *StatusAilments) *int { return &s.Slow }, []int{0, 3, 20}, []int{10, 10, 10}},
		{"トラバサミ（長い方）", bearTrapEffect, func(s *StatusAilments) *int { return &s.BearTrap }, []int{0, 3, 9}, []int{6, 6, 9}},
		{"くねくね踊り（長い方）", func(g *Game) {
			g.state.Enemies = []Enemy{CreateEnemyByID(3, 5, 4)}
			g.state.Enemies[0].SpecialAttack(&g.state.Enemies[0], g)
		}, func(s *StatusAilments) *int { return &s.Slow }, []int{0, 3, 12}, []int{8, 8, 12}},
		{"睡眠薬を飲む（上書き）", func(g *Game) { usePlayerItem(g, 17) }, func(s *StatusAilments) *int { return &s.Sleep }, []int{0, 20}, []int{10, 10}},
		{"混乱薬を飲む（上書き）", func(g *Game) { usePlayerItem(g, 18) }, func(s *StatusAilments) *int { return &s.Confusion }, []int{0, 20}, []int{10, 10}},
		{"目潰し薬を飲む（上書き）", func(g *Game) { usePlayerItem(g, 19) }, func(s *StatusAilments) *int { return &s.Blind }, []int{0, 50}, []int{30, 30}},
		{"口封じのカード（上書き）", func(g *Game) { usePlayerItem(g, 42) }, func(s *StatusAilments) *int { return &s.MouthSeal }, []int{0, 3, 40}, []int{15, 15, 15}},
	}
	for _, tc := range cases {
		for i, before := range tc.before {
			g := newFoundationTestGame()
			*tc.get(&g.state.Player.StatusAilments) = before
			tc.apply(g)
			runQueuedActions(g)
			if got := *tc.get(&g.state.Player.StatusAilments); got != tc.want[i] {
				t.Errorf("%s: %d ターン残りでかけると %d、want %d", tc.name, before, got, tc.want[i])
			}
		}
	}
}

// TestPlayerStatusShieldGuard は盾の状態異常耐性（50%）で防ぐ付与と、能力の無効化だけを見る付与を固定する。
func TestPlayerStatusShieldGuard(t *testing.T) {
	cases := []struct {
		name   string
		apply  func(g *Game)
		active func(s StatusAilments) bool
		shield bool // 盾の耐性で防げるか
	}{
		{"睡眠ガスの罠", sleepGasTrapEffect, func(s StatusAilments) bool { return s.Sleep > 0 }, true},
		{"鈍足の罠", slowTrapEffect, func(s StatusAilments) bool { return s.Slow > 0 }, true},
		{"トラバサミ", bearTrapEffect, func(s StatusAilments) bool { return s.BearTrap > 0 }, false},
		{"睡眠薬を飲む", func(g *Game) { usePlayerItem(g, 17) }, func(s StatusAilments) bool { return s.Sleep > 0 }, false},
		{"混乱薬を飲む", func(g *Game) { usePlayerItem(g, 18) }, func(s StatusAilments) bool { return s.Confusion > 0 }, false},
		{"目潰し薬を飲む", func(g *Game) { usePlayerItem(g, 19) }, func(s StatusAilments) bool { return s.Blind > 0 }, false},
	}
	for _, tc := range cases {
		g := newFoundationTestGame()
		rolled := 0
		g.randHooks.statusResistance = func(int) int { rolled++; return 0 } // 振れば必ず防ぐ
		equip(g, buildItemFromTemplate(65, 0, 0))                           // 状態異常耐性の盾
		tc.apply(g)
		msg := runQueuedActions(g)
		if tc.shield {
			if tc.active(g.state.Player.StatusAilments) || rolled != 1 || !strings.Contains(msg, "防いだ") {
				t.Errorf("%s: 盾で防ぐはず（rolls %d, %q）", tc.name, rolled, msg)
			}
		} else if !tc.active(g.state.Player.StatusAilments) || rolled != 0 {
			t.Errorf("%s: 盾の判定をしないはず（rolls %d）", tc.name, rolled)
		}
	}
}

// TestStatusRecoveryMessages はターン経過で治ったときのメッセージと順序を固定する。
func TestStatusRecoveryMessages(t *testing.T) {
	g := newFoundationTestGame()
	g.state.Player.StatusAilments = StatusAilments{BearTrap: 1, Confusion: 1, Sleep: 1, Blind: 1, Slow: 1, MouthSeal: 1}
	g.decrementStatusAilments()
	got := runQueuedActions(g)
	want := strings.Join([]string{"海老さんはトラバサミから抜け出した", "海老さんは目を覚ました", "海老さんは目が見えるようになった", "海老さんの体の速さが元に戻った", "海老さんは口が動くようになった"}, "\n")
	if got != want {
		t.Fatalf("messages = %q, want %q", got, want)
	}
	if g.state.Player.StatusAilments != (StatusAilments{}) || !g.miniMapDirty {
		t.Fatalf("status %+v, miniMapDirty %v", g.state.Player.StatusAilments, g.miniMapDirty)
	}

	// 2ターン以上残っていれば1減るだけでメッセージは出ない。
	g.state.Player.StatusAilments = StatusAilments{BearTrap: 2, Sleep: 2, Blind: 2, Slow: 2, MouthSeal: 2, Confusion: 2, Paralysis: true, Seal: true}
	g.decrementStatusAilments()
	if msg := runQueuedActions(g); msg != "" {
		t.Fatalf("unexpected messages %q", msg)
	}
	if s := g.state.Player.StatusAilments; s != (StatusAilments{BearTrap: 1, Sleep: 1, Blind: 1, Slow: 1, MouthSeal: 1, Confusion: 1, Paralysis: true, Seal: true}) {
		t.Fatalf("player status after tick %+v", s)
	}
}

// TestEnemyStatusTicks は敵の経過（目潰しは減らない・仮眠は減らない・目覚めの倍速）を固定する。
func TestEnemyStatusTicks(t *testing.T) {
	g := newFoundationTestGame()
	e := CreateEnemyByID(0, 6, 4)
	e.StatusAilments = StatusAilments{Confusion: 2, Haste: 2, Backward: 2, Sleep: 1, HasteOnWake: true, Blind: 999, Paralysis: true, Seal: true}
	nap := CreateEnemyByID(0, 2, 2)
	nap.StatusAilments = StatusAilments{Sleep: -1, Blind: 10}
	g.state.Enemies = []Enemy{e, nap}
	g.decrementStatusAilments()
	msg := runQueuedActions(g)
	if !strings.Contains(msg, "目を覚まし、倍速で動き出した") {
		t.Fatalf("wake message missing: %q", msg)
	}
	want := StatusAilments{Confusion: 1, Haste: hasteOnWakeTurns, Backward: 1, Blind: 999, Paralysis: true, Seal: true}
	if got := g.state.Enemies[0].StatusAilments; got != want {
		t.Fatalf("enemy status %+v, want %+v", got, want)
	}
	if got := g.state.Enemies[1].StatusAilments; got != (StatusAilments{Sleep: -1, Blind: 10}) {
		t.Fatalf("napping enemy status %+v", got)
	}
}

// TestEnemyStatusCardsAndDamage は敵への付与（カード・薬）と、ダメージで解ける状態を固定する。
func TestEnemyStatusCardsAndDamage(t *testing.T) {
	g := newFoundationTestGame()
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 4)}
	g.state.Enemies[0].StatusAilments.Haste = 3
	usePlayerItem(g, 15) // 金縛りのカード
	if !g.state.Enemies[0].StatusAilments.Paralysis {
		t.Fatal("金縛りのカードで金縛りになるはず")
	}
	g.state.Enemies[0].StatusAilments.Sleep = enemyNapSleep
	g.applyDamageToEnemy(0, 1)
	if s := g.state.Enemies[0].StatusAilments; s.Paralysis || s.Sleep != 0 || s.Haste != 3 {
		t.Fatalf("ダメージで金縛りと仮眠だけが解けるはず: %+v", s)
	}
}

// TestTroubleCardClearsBadStatuses は困った時のカードが悪い状態だけを解くことを固定する。
func TestTroubleCardClearsBadStatuses(t *testing.T) {
	g := newFoundationTestGame()
	g.state.Player.StatusAilments = StatusAilments{BearTrap: 3, Confusion: 3, Sleep: 3, Blind: 3, Slow: 3, MouthSeal: 3, Paralysis: true, Seal: true, HasteOnWake: true, Haste: 4, Backward: 5}
	usePlayerItem(g, 173)
	if got := g.state.Player.StatusAilments; got != (StatusAilments{Haste: 4, Backward: 5}) {
		t.Fatalf("status after trouble card %+v", got)
	}
}

// --- 共通処理（wave8 以降の付与が使う） ---

func TestApplyPlayerStatus(t *testing.T) {
	g := newFoundationTestGame()
	if !g.applyPlayerStatus(StatusConfusion, 10, StatusSource{}) || g.state.Player.StatusAilments.Confusion != 10 {
		t.Fatal("confusion should be applied")
	}
	if msg := runQueuedActions(g); msg != "海老さんは混乱した" {
		t.Fatalf("message %q", msg)
	}
	if !g.applyPlayerStatus(StatusHaste, 5, StatusSource{}) || g.state.Player.StatusAilments.Haste != 5 {
		t.Fatal("haste should be applied to the player")
	}
	if !g.applyPlayerStatus(StatusSleep, 3, StatusSource{Stack: StackLonger}) || !g.applyPlayerStatus(StatusSleep, 2, StatusSource{Stack: StackLonger}) || g.state.Player.StatusAilments.Sleep != 3 {
		t.Fatalf("StackLonger sleep = %d", g.state.Player.StatusAilments.Sleep)
	}

	// 能力による無効化
	withTestImmunity(t, "test_immunity", StatusBlind)
	equip(g, testRing("test_immunity"))
	runQueuedActions(g)
	if g.applyPlayerStatus(StatusBlind, 10, StatusSource{}) || g.state.Player.StatusAilments.Blind != 0 {
		t.Fatal("immune ring should block blind")
	}
	if msg := runQueuedActions(g); msg != "試験の指輪が目潰しを防いだ" {
		t.Fatalf("blocked message %q", msg)
	}

	// 盾の耐性は Shield のときだけ振る
	g = newFoundationTestGame()
	rolled := 0
	g.randHooks.statusResistance = func(int) int { rolled++; return 0 }
	equip(g, buildItemFromTemplate(65, 0, 0))
	if !g.applyPlayerStatus(StatusSlow, 10, StatusSource{}) || rolled != 0 {
		t.Fatalf("without Shield the shield must not roll (rolls %d)", rolled)
	}
	if g.applyPlayerStatus(StatusConfusion, 10, StatusSource{Shield: true}) || rolled != 1 {
		t.Fatalf("with Shield the shield should block (rolls %d)", rolled)
	}
	if msg := runQueuedActions(g); !strings.HasSuffix(msg, "が混乱を防いだ") {
		t.Fatalf("shield blocked message %q", msg)
	}
	g.clearPlayerStatus(StatusSlow)
	if g.state.Player.StatusAilments.Slow != 0 {
		t.Fatal("clearPlayerStatus")
	}
}

func TestApplyEnemyStatusBossImmunity(t *testing.T) {
	g := newFoundationTestGame()
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 4), CreateEnemyByID(44, 6, 4), CreateEnemyByID(42, 2, 2)}
	if !g.applyEnemyStatus(0, StatusParalysis, 1) || !g.state.Enemies[0].StatusAilments.Paralysis {
		t.Fatal("normal enemy should be paralyzed")
	}
	if g.applyEnemyStatus(1, StatusParalysis, 1) || g.state.Enemies[1].StatusAilments.Paralysis {
		t.Fatal("final boss resists paralysis")
	}
	if !g.applyEnemyStatus(2, StatusParalysis, 1) || !g.applyEnemyStatus(2, StatusSleep, 10) {
		t.Fatal("crab boss accepts paralysis and sleep")
	}
	// 短縮はボスの行動時
	if g.state.Enemies[2].StatusAilments.Sleep != 10 {
		t.Fatalf("boss sleep is shortened when it acts, got %d", g.state.Enemies[2].StatusAilments.Sleep)
	}
	normalizeBossStatus(&g.state.Enemies[2])
	if g.state.Enemies[2].StatusAilments.Sleep != bossCrabControlTurns {
		t.Fatalf("boss sleep after normalize %d", g.state.Enemies[2].StatusAilments.Sleep)
	}
	if g.applyEnemyStatus(0, StatusMouthSeal, 5) || g.applyEnemyStatus(9, StatusSleep, 5) {
		t.Fatal("mouth seal is not an enemy status / out of range index")
	}
	g.clearEnemyStatus(0, StatusParalysis)
	if g.state.Enemies[0].StatusAilments.Paralysis {
		t.Fatal("clearEnemyStatus")
	}
}

// --- メッセージと表示時間の統一 ---

// queuedActions はキューを実行しながら、表示された Action（メッセージと表示時間）を集める。
func queuedActions(g *Game) []Action {
	var shown []Action
	for len(g.ActionQueue.Queue) > 0 {
		a := g.ActionQueue.Queue[0]
		g.ActionQueue.Queue = g.ActionQueue.Queue[1:]
		shown = append(shown, a)
		if a.Execute != nil {
			a.Execute(g)
		}
	}
	return shown
}

// TestPlayerStatusMessagesAreUnified は付与元にかかわらず、かかった・防いだのメッセージが定義表の文言で、
// 表示時間が statusMessageDuration にそろっていることを固定する。
func TestPlayerStatusMessagesAreUnified(t *testing.T) {
	kunekune := func(g *Game) {
		g.state.Enemies = []Enemy{CreateEnemyByID(3, 5, 4)}
		g.state.Enemies[0].SpecialAttack(&g.state.Enemies[0], g)
	}
	cases := []struct {
		name   string
		apply  func(g *Game)
		immune []StatusKind // 試験の指輪で防ぐ種類（空なら防がない）
		want   string
	}{
		{"睡眠ガスの罠", sleepGasTrapEffect, nil, "海老さんは眠った"},
		{"睡眠薬を飲む", func(g *Game) { queuePlayerItem(g, 17) }, nil, "海老さんは眠った"},
		{"混乱薬を飲む", func(g *Game) { queuePlayerItem(g, 18) }, nil, "海老さんは混乱した"},
		{"目潰し薬を飲む", func(g *Game) { queuePlayerItem(g, 19) }, nil, "海老さんは目が見えなくなった"},
		{"鈍足の罠", slowTrapEffect, nil, "海老さんは鈍足になった"},
		{"くねくね踊り", kunekune, nil, "海老さんは鈍足になった"},
		{"トラバサミ", bearTrapEffect, nil, "海老さんはトラバサミに足を挟まれた"},
		{"口封じのカード", func(g *Game) { queuePlayerItem(g, 42) }, nil, "海老さんは口を封じられた"},
		{"睡眠ガスの罠を防ぐ", sleepGasTrapEffect, []StatusKind{StatusSleep}, "試験の指輪が睡眠を防いだ"},
		{"睡眠薬を防ぐ", func(g *Game) { queuePlayerItem(g, 17) }, []StatusKind{StatusSleep}, "試験の指輪が睡眠を防いだ"},
		{"くねくね踊りを防ぐ", kunekune, []StatusKind{StatusSlow}, "試験の指輪が鈍足を防いだ"},
		{"トラバサミを防ぐ", bearTrapEffect, []StatusKind{StatusBearTrap}, "試験の指輪がトラバサミを防いだ"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := newFoundationTestGame()
			if len(tc.immune) > 0 {
				withTestImmunity(t, "test_unified", tc.immune...)
				equip(g, testRing("test_unified"))
				runQueuedActions(g)
			}
			tc.apply(g)
			var messages []string
			found := false
			for _, a := range queuedActions(g) {
				messages = append(messages, a.Message)
				if a.Message == tc.want {
					found = true
					if a.Duration != statusMessageDuration {
						t.Fatalf("%q の表示時間 %v、want %v", a.Message, a.Duration, statusMessageDuration)
					}
				}
			}
			if !found {
				t.Fatalf("messages = %q, want %q", messages, tc.want)
			}
		})
	}
}

// TestPlayerStatusRecoveryDuration は治ったときのメッセージも同じ表示時間であることを固定する。
func TestPlayerStatusRecoveryDuration(t *testing.T) {
	g := newFoundationTestGame()
	g.state.Player.StatusAilments = StatusAilments{Sleep: 1, Slow: 1}
	g.decrementStatusAilments()
	for _, a := range queuedActions(g) {
		if a.Duration != statusMessageDuration {
			t.Fatalf("%q の表示時間 %v", a.Message, a.Duration)
		}
	}
}

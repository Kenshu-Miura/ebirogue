//go:build !test

package main

// 段3の共通基盤のテスト: 強化値を下げる・装備を弾き飛ばす・持ち物を投げ捨てる（equipment_harm.go）、
// 被弾時の処理と分裂・直線上の海老さん・ゾワゾワ（enemy_behaviors.go）。

import (
	"encoding/json"
	"strings"
	"testing"
)

// withStatusImmunityRing はテスト中だけ kind を防ぐ指輪を装備させる。
func withStatusImmunityRing(t *testing.T, g *Game, kind StatusKind) {
	t.Helper()
	id := EquipmentAbilityID("test_guard_" + string(kind))
	withTestImmunity(t, id, kind)
	equip(g, testRing(id))
}

// withMonsterOnDamaged はテスト中だけ敵 id の被弾時の処理を差し替える。
func withMonsterOnDamaged(t *testing.T, id int, onDamaged EnemyDamagedFunc) {
	t.Helper()
	previous := MonsterDefinitions[id]
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.OnDamaged = onDamaged }, id)
	t.Cleanup(func() { MonsterDefinitions[id] = previous })
}

// --- 強化値を下げる ---

func TestRustPlayerEquipmentLowersSharpnessAndStats(t *testing.T) {
	g := newFoundationTestGame()
	weapon, armor := testWeapon(), testArmor()
	equip(g, weapon, armor)
	attack, defense := g.state.Player.AttackPower, g.state.Player.DefensePower

	if !g.rustPlayerEquipment(rustWeapon, 1) || weapon.Sharpness != -1 || g.state.Player.AttackPower != attack-1 {
		t.Fatalf("weapon sharpness = %d, attack = %d", weapon.Sharpness, g.state.Player.AttackPower)
	}
	if !g.rustPlayerEquipment(rustArmor, 2) || armor.Sharpness != -2 || g.state.Player.DefensePower != defense-2 {
		t.Fatalf("armor sharpness = %d, defense = %d", armor.Sharpness, g.state.Player.DefensePower)
	}
	if got := runQueuedActions(g); strings.Count(got, "錆びてしまった") != 2 {
		t.Fatalf("messages = %q", got)
	}
}

func TestRustPlayerEquipmentGuards(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, g *Game, weapon *Weapon)
	}{
		{name: "rust proof", setup: func(_ *testing.T, _ *Game, weapon *Weapon) { weapon.RustProof = true }},
		{name: "rust immune item", setup: func(_ *testing.T, _ *Game, weapon *Weapon) {
			weapon.Abilities = []EquipmentAbilityID{AbilityRustImmune}
		}},
		{name: "rust ward", setup: func(t *testing.T, g *Game, _ *Weapon) { withStatusImmunityRing(t, g, StatusRust) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newFoundationTestGame()
			weapon := testWeapon()
			equip(g, weapon)
			tc.setup(t, g, weapon)
			if g.rustPlayerEquipment(rustWeapon, 1) || weapon.Sharpness != 0 {
				t.Fatalf("sharpness = %d, want 0", weapon.Sharpness)
			}
			if got := runQueuedActions(g); !strings.Contains(got, "錆びなかった") {
				t.Fatalf("messages = %q", got)
			}
		})
	}
}

func TestRustPlayerEquipmentWithoutEquipmentDoesNothing(t *testing.T) {
	g := newFoundationTestGame()
	if g.rustPlayerEquipment(rustWeapon, 1) || g.rustPlayerEquipment(rustArmor, 1) {
		t.Fatal("nothing equipped should not rust")
	}
}

// --- 装備を弾き飛ばす ---

func TestKnockOffPlayerEquipmentFlingsShield(t *testing.T) {
	g := newFoundationTestGame()
	armor := testArmor()
	armor.Cursed = true // 呪われていても弾かれる
	equip(g, armor)
	defense := g.state.Player.DefensePower

	if !g.knockOffPlayerEquipment(armor, -1, 0, 5) {
		t.Fatal("shield should be knocked off")
	}

	if g.state.Player.EquippedArmor != nil || g.state.Player.DefensePower != defense-armor.DefensePower {
		t.Fatalf("armor still equipped or defense = %d", g.state.Player.DefensePower)
	}
	if len(g.state.Player.Inventory) != 0 {
		t.Fatalf("inventory = %v, the shield should leave the inventory", g.state.Player.Inventory)
	}
	// (4,4) から左へ5マス。x=0 は壁なので (1,4) で止まる
	if items := floorItemsAt(g, 1, 4); len(items) != 1 || items[0] != armor || !armor.Cursed {
		t.Fatalf("items = %v, want the cursed shield at (1,4)", g.state.Items)
	}
	if got := runQueuedActions(g); !strings.Contains(got, "弾き飛ばされた") {
		t.Fatalf("messages = %q", got)
	}
}

func TestKnockOffPlayerEquipmentFindsFreeTile(t *testing.T) {
	g := newFoundationTestGame()
	armor := testArmor()
	equip(g, armor)
	g.state.Items = []Item{buildItemFromTemplate(1, 1, 4)}

	g.knockOffPlayerEquipment(armor, -1, 0, 5)

	// 止まったマスに道具があれば近い順（左上から）に探す
	if items := floorItemsAt(g, 1, 3); len(items) != 1 || items[0] != armor {
		t.Fatalf("items = %v, want the shield at (1,3)", g.state.Items)
	}
}

func TestKnockOffPlayerEquipmentGuarded(t *testing.T) {
	g := newFoundationTestGame()
	armor := testArmor()
	equip(g, armor)
	withStatusImmunityRing(t, g, StatusKnockOff)

	if g.knockOffPlayerEquipment(armor, -1, 0, 5) || g.state.Player.EquippedArmor != armor {
		t.Fatal("the guard should keep the shield equipped")
	}
	if got := runQueuedActions(g); !strings.Contains(got, "装備の弾き飛ばし") {
		t.Fatalf("messages = %q", got)
	}
}

func TestKnockOffUnequippedItemDoesNothing(t *testing.T) {
	g := newFoundationTestGame()
	armor := testArmor()
	g.state.Player.Inventory = []Item{armor}
	if g.knockOffPlayerEquipment(armor, -1, 0, 5) || len(g.state.Player.Inventory) != 1 {
		t.Fatal("an unequipped shield should not be knocked off")
	}
}

// --- 持ち物を投げ捨てる ---

func TestTossPlayerItemPrefersAndFlies(t *testing.T) {
	g := newFoundationTestGame()
	g.state.Map = makeTestFloorMap(20, 9)
	weapon := testWeapon()
	equip(g, weapon)
	food := buildItemFromTemplate(1, 0, 0)
	cane := buildItemFromTemplate(16, 0, 0)
	g.state.Player.Inventory = append(g.state.Player.Inventory, food, cane)
	right := -1
	for i, d := range getDirections() {
		if d.dx == 1 && d.dy == 0 {
			right = i
		}
	}
	// 1回目（候補の選択）は0、2回目（方向）は右
	rolls := []int{0, right}
	intn := func(int) int {
		r := rolls[0]
		rolls = rolls[1:]
		return r
	}

	item, ok := g.tossPlayerItem(4, 4, func(item Item) bool { _, isCane := item.(*Cane); return isCane }, intn)

	if !ok || item != cane {
		t.Fatalf("tossed %v, want the cane", item)
	}
	if len(g.state.Player.Inventory) != 2 || g.state.Player.EquippedWeapon != weapon {
		t.Fatalf("inventory = %v", g.state.Player.Inventory)
	}
	// (4,4) から右へ10マス
	if items := floorItemsAt(g, 14, 4); len(items) != 1 || items[0] != cane {
		t.Fatalf("items = %v, want the cane at (14,4)", g.state.Items)
	}
	if got := runQueuedActions(g); !strings.Contains(got, "投げ捨てられた") {
		t.Fatalf("messages = %q", got)
	}
}

func TestTossPlayerItemFallsBackAndSkipsEquipped(t *testing.T) {
	g := newFoundationTestGame()
	weapon := testWeapon()
	equip(g, weapon)
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = append(g.state.Player.Inventory, food)

	item, ok := g.tossPlayerItem(4, 4, func(item Item) bool { _, isCane := item.(*Cane); return isCane }, func(int) int { return 0 })

	if !ok || item != food || g.state.Player.EquippedWeapon != weapon {
		t.Fatalf("tossed %v, want the food (no cane, weapon is equipped)", item)
	}
}

func TestTossPlayerItemGuardedOrNothing(t *testing.T) {
	g := newFoundationTestGame()
	if _, ok := g.tossPlayerItem(4, 4, nil, func(int) int { return 0 }); ok {
		t.Fatal("empty inventory should toss nothing")
	}
	g.state.Player.Inventory = []Item{buildItemFromTemplate(1, 0, 0)}
	withStatusImmunityRing(t, g, StatusItemToss)
	if _, ok := g.tossPlayerItem(4, 4, nil, func(int) int { return 0 }); ok {
		t.Fatal("the guard should prevent the toss")
	}
	// 指輪は装備中なので、守られた持ち物と合わせて2つ
	if len(g.state.Player.Inventory) != 2 {
		t.Fatalf("inventory = %v", g.state.Player.Inventory)
	}
}

// --- 被弾時の処理と分裂 ---

func TestOnDamagedSplitsEnemy(t *testing.T) {
	withMonsterOnDamaged(t, 0, func(g *Game, i, _ int) { g.splitEnemy(i) })
	g := newFoundationTestGame()
	index := addTestEnemy(g, 0, 6, 4)
	g.state.Enemies[index].Health = 20

	g.applyDamageToEnemy(index, 5)

	if len(g.state.Enemies) != 2 {
		t.Fatalf("enemies = %d, want a clone", len(g.state.Enemies))
	}
	clone := g.state.Enemies[1]
	if clone.ID != 0 || clone.Health != 15 || !clone.PlayerDiscovered {
		t.Fatalf("clone = %+v", clone)
	}
	if max(abs(clone.X-6), abs(clone.Y-4)) != 1 {
		t.Fatalf("clone at (%d,%d), want next to (6,4)", clone.X, clone.Y)
	}
	if got := runQueuedActions(g); !strings.Contains(got, "分裂した") {
		t.Fatalf("messages = %q", got)
	}
}

func TestOnDamagedNotCalledWhenDefeatedOrSealed(t *testing.T) {
	calls := 0
	withMonsterOnDamaged(t, 0, func(*Game, int, int) { calls++ })
	g := newFoundationTestGame()
	index := addTestEnemy(g, 0, 6, 4)
	g.state.Enemies[index].Health = 5

	g.state.Enemies[index].StatusAilments.Seal = true
	g.applyDamageToEnemy(index, 1)
	g.state.Enemies[index].StatusAilments.Seal = false
	g.applyDamageToEnemy(index, 0)
	g.applyDamageToEnemy(index, 10)

	if calls != 0 {
		t.Fatalf("OnDamaged called %d times, want 0 (sealed, no damage, defeated)", calls)
	}
}

func TestSplitEnemyLimits(t *testing.T) {
	g := newFoundationTestGame()
	index := addTestEnemy(g, 0, 1, 1)
	// 隣の空きは (2,1)・(1,2)・(2,2) だけ。敵で埋めると分裂できない
	addTestEnemy(g, 0, 2, 1)
	addTestEnemy(g, 0, 1, 2)
	addTestEnemy(g, 0, 2, 2)
	if got := g.splitEnemy(index); got != -1 {
		t.Fatalf("split = %d, want -1 with no room", got)
	}

	g = newFoundationTestGame()
	for len(g.state.Enemies) < MaxEnemiesOnFloor {
		addTestEnemy(g, 0, 1, 1)
	}
	if got := g.splitEnemy(0); got != -1 {
		t.Fatalf("split = %d, want -1 at the floor limit", got)
	}

	g = newFoundationTestGame()
	boss := addTestEnemy(g, 42, 6, 4)
	if got := g.splitEnemy(boss); got != -1 {
		t.Fatal("a boss should not split")
	}
}

// --- 直線上の海老さん ---

func TestStraightLineToPlayer(t *testing.T) {
	for _, tc := range []struct {
		name      string
		x, y      int
		max       int
		block     *Coordinate
		blocker   *Coordinate
		wantOK    bool
		wantDX    int
		wantDY    int
		wantSteps int
	}{
		{name: "right 3", x: 1, y: 4, max: 3, wantOK: true, wantDX: 1, wantSteps: 3},
		{name: "too far", x: 1, y: 4, max: 2},
		{name: "diagonal", x: 7, y: 1, max: 3, wantOK: true, wantDX: -1, wantDY: 1, wantSteps: 3},
		{name: "not a line", x: 2, y: 3, max: 3},
		{name: "adjacent", x: 5, y: 4, max: 3, wantOK: true, wantDX: -1, wantSteps: 1},
		{name: "wall between", x: 1, y: 4, max: 3, block: &Coordinate{X: 2, Y: 4}},
		{name: "enemy between", x: 1, y: 4, max: 3, blocker: &Coordinate{X: 3, Y: 4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newFoundationTestGame()
			index := addTestEnemy(g, 0, tc.x, tc.y)
			if tc.block != nil {
				g.state.Map[tc.block.Y][tc.block.X] = Tile{Type: "wall", Blocked: true}
			}
			if tc.blocker != nil {
				addTestEnemy(g, 0, tc.blocker.X, tc.blocker.Y)
			}
			dx, dy, steps, ok := g.straightLineToPlayer(index, tc.max)
			if ok != tc.wantOK || dx != tc.wantDX || dy != tc.wantDY || steps != tc.wantSteps {
				t.Fatalf("got (%d,%d) %d %v", dx, dy, steps, ok)
			}
		})
	}
}

// --- ゾワゾワ ---

func TestBackwardEnemyAttacksTheOtherWay(t *testing.T) {
	g := newFoundationTestGame()
	attacker := addTestEnemy(g, 0, 5, 4)
	behind := addTestEnemy(g, 0, 6, 4)
	g.state.Enemies[behind].Health = 50
	g.state.Enemies[attacker].StatusAilments.Backward = 3

	g.AttackFromEnemy(attacker)
	messages := runQueuedActions(g)

	if g.state.Player.Health != 40 {
		t.Fatalf("player health = %d, the backward enemy should not hit the player", g.state.Player.Health)
	}
	if g.state.Enemies[behind].Health >= 50 {
		t.Fatalf("enemy behind health = %d, want damaged (messages %q)", g.state.Enemies[behind].Health, messages)
	}
}

func TestBackwardEnemySwingsAtNothing(t *testing.T) {
	g := newFoundationTestGame()
	attacker := addTestEnemy(g, 0, 5, 5)
	g.state.Enemies[attacker].StatusAilments.Backward = 3

	g.AttackFromEnemy(attacker)
	messages := runQueuedActions(g)

	if g.state.Player.Health != 40 || !strings.Contains(messages, "反対を向いて攻撃した") {
		t.Fatalf("health = %d, messages = %q", g.state.Player.Health, messages)
	}
	if g.state.Enemies[attacker].AttackDirection != DownRight {
		t.Fatalf("attack direction = %v, want away from the player", g.state.Enemies[attacker].AttackDirection)
	}
}

func TestBackwardWearsOffAndIsSaved(t *testing.T) {
	g := newFoundationTestGame()
	index := addTestEnemy(g, 0, 7, 7)
	g.state.Enemies[index].StatusAilments.Backward = 2

	saved := enemyToSaved(&g.state.Enemies[index])
	data, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	var decoded SavedEnemy
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if restored := savedToEnemy(decoded); restored.StatusAilments.Backward != 2 {
		t.Fatalf("restored backward = %d, want 2", restored.StatusAilments.Backward)
	}

	g.IncrementMoveCount()
	g.IncrementMoveCount()
	if got := g.state.Enemies[index].StatusAilments.Backward; got != 0 {
		t.Fatalf("backward = %d after two turns, want 0", got)
	}
}

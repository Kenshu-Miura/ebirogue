//go:build !test

package main

// 共通基盤 F1〜F5（content-expansion-plan.md）の結合テスト。
// 新しい武器・盾・指輪・敵はまだ無いので、テスト用の仮の能力 ID・仮の敵 ID でフックの正常系と無効化を確かめる。

import (
	"strings"
	"testing"
)

// withTestAbilityHooks はテスト中だけ能力のフックを登録する。
func withTestAbilityHooks(t *testing.T, id EquipmentAbilityID, hooks AbilityHooks) {
	t.Helper()
	previous, existed := registerAbilityHooks(id, hooks)
	t.Cleanup(func() {
		if existed {
			equipmentAbilityHooks[id] = previous
		} else {
			delete(equipmentAbilityHooks, id)
		}
	})
}

// withTestImmunity はテスト中だけ能力に状態異常の無効化を登録する。
func withTestImmunity(t *testing.T, id EquipmentAbilityID, kinds ...StatusKind) {
	t.Helper()
	statusImmunities[id] = kinds
	t.Cleanup(func() { delete(statusImmunities, id) })
}

// withTestMonster はテスト中だけ仮の敵の定義を追加する（基になる敵の定義を写して書き換える）。
func withTestMonster(t *testing.T, id, baseID int, edit func(*MonsterDefinition)) {
	t.Helper()
	definition := MonsterDefinitions[baseID]
	definition.ID = id
	edit(&definition)
	MonsterDefinitions[id] = definition
	t.Cleanup(func() { delete(MonsterDefinitions, id) })
}

func testRing(abilities ...EquipmentAbilityID) *Accessory {
	ring := buildItemFromTemplate(10, 0, 0).(*Accessory)
	ring.Name = "試験の指輪"
	ring.Abilities = abilities
	return ring
}

func testWeapon(abilities ...EquipmentAbilityID) *Weapon {
	weapon := buildItemFromTemplate(20, 0, 0).(*Weapon)
	weapon.Sharpness, weapon.Cursed = 0, false
	weapon.Abilities = abilities
	return weapon
}

func testArmor(abilities ...EquipmentAbilityID) *Armor {
	armor := buildItemFromTemplate(66, 0, 0).(*Armor)
	armor.Sharpness, armor.Cursed = 0, false
	armor.Abilities = abilities
	return armor
}

// newFoundationTestGame は 9x9 の部屋の中央 (4,4) にプレイヤーがいるゲームを作る。
// ダメージの乱数は最大・会心なし、命中判定は必ず命中に固定する。
func newFoundationTestGame() *Game {
	return &Game{
		rng: newTurnRNG(7),
		randHooks: randHooks{
			damage:    func(n int) int { return n - 1 },
			attackHit: func(int) int { return 0 },
		},
		state: GameState{
			Map: makeTestFloorMap(9, 9),
			Player: Player{
				Entity: Entity{X: 4, Y: 4}, Direction: Right,
				Health: 40, MaxHealth: 40, Satiety: 50, MaxSatiety: 100,
				AttackPower: 5, DefensePower: 3, Power: 8, MaxPower: 8, Level: 1,
			},
		},
	}
}

func equip(g *Game, items ...Item) {
	for _, item := range items {
		g.state.Player.Inventory = append(g.state.Player.Inventory, item)
		if equipable, ok := item.(Equipable); ok {
			g.state.Player.EquipItem(equipable)
		}
	}
}

// --- F1: 装備能力のフック ---

func TestPlayerAbilityCountStacksTwoRings(t *testing.T) {
	g := newFoundationTestGame()
	if got := g.playerAbilityCount("test_stack"); got != 0 {
		t.Fatalf("count without rings = %d", got)
	}
	equip(g, testRing("test_stack"))
	if got := g.playerAbilityCount("test_stack"); got != 1 {
		t.Fatalf("count with one ring = %d, want 1", got)
	}
	equip(g, testRing("test_stack"))
	if got := g.playerAbilityCount("test_stack"); got != 2 {
		t.Fatalf("count with two rings = %d, want 2", got)
	}

	var stacks []int
	withTestAbilityHooks(t, "test_stack", AbilityHooks{OnTurn: func(g *Game, n int) { stacks = append(stacks, n) }})
	g.IncrementMoveCount()
	if len(stacks) != 1 || stacks[0] != 2 {
		t.Fatalf("turn hook stacks = %v, want one call with 2", stacks)
	}

	// 外すと重ね数が減り、フックも呼ばれなくなる（無効化）
	for _, ring := range g.state.Player.EquippedAccessories {
		g.state.Player.UnequipItem(ring)
	}
	stacks = nil
	g.IncrementMoveCount()
	if g.playerAbilityCount("test_stack") != 0 || len(stacks) != 0 {
		t.Fatalf("after unequip: count %d, hook calls %v", g.playerAbilityCount("test_stack"), stacks)
	}
}

func TestPowerRingWorksByAbilityIDNotName(t *testing.T) {
	g := newFoundationTestGame()
	ring := buildItemFromTemplate(10, 0, 0).(*Accessory)
	ring.Name = "名前を変えた指輪" // 任意名・別名でも能力 ID で効く
	equip(g, ring)
	if g.state.Player.Power != 11 || g.state.Player.MaxPower != 11 {
		t.Fatalf("power after equip = %d/%d, want 11/11", g.state.Player.Power, g.state.Player.MaxPower)
	}
	g.state.Player.UnequipItem(ring)
	if g.state.Player.Power != 8 || g.state.Player.MaxPower != 8 {
		t.Fatalf("power after unequip = %d/%d, want 8/8", g.state.Player.Power, g.state.Player.MaxPower)
	}

	// 能力の無い指輪は名前が鼓舞の指輪でも効かない
	fake := testRing()
	fake.Name = "鼓舞の指輪"
	equip(g, fake)
	if g.state.Player.Power != 8 {
		t.Fatalf("ring without ability changed power to %d", g.state.Player.Power)
	}

	// セーブから復元しても能力はテンプレートから戻る
	restored, err := savedToItem(itemToSaved(ring))
	if err != nil || !hasEquipmentAbility(restored.(*Accessory).Abilities, AbilityPowerBoost) {
		t.Fatalf("restored ring = %#v, %v", restored, err)
	}
}

func TestWeaponAttackHooks(t *testing.T) {
	var before, hits, defeats, misses int
	withTestAbilityHooks(t, "test_weapon", AbilityHooks{
		BeforePlayerAttack: func(g *Game, a *PlayerAttack, stacks int) {
			before++
			a.Multiplier *= 2
			a.MessagePrefix += "試験！"
		},
		AfterPlayerHit:     func(g *Game, hit PlayerAttackHit, stacks int) { hits += hit.Damage },
		AfterEnemyDefeated: func(g *Game, hit PlayerAttackHit, stacks int) { defeats++ },
		AfterPlayerMiss:    func(g *Game, a *PlayerAttack, stacks int) { misses++ },
	})

	attack := func(weapon *Weapon, enemyHealth int) (*Game, int) {
		g := newFoundationTestGame()
		equip(g, weapon)
		enemy := CreateEnemyByID(0, 5, 4)
		enemy.Health, enemy.MaxHealth = enemyHealth, enemyHealth
		g.state.Enemies = []Enemy{enemy}
		g.CheckForEnemies(1, 0)
		message := runQueuedActions(g)
		damage := enemyHealth
		if len(g.state.Enemies) > 0 {
			damage -= g.state.Enemies[0].Health
		}
		if !strings.Contains(message, "ダメージを与えた") && !strings.Contains(message, "外れた") {
			t.Fatalf("attack messages = %q", message)
		}
		return g, damage
	}

	_, plain := attack(testWeapon(), 1000)
	if before != 0 || hits != 0 {
		t.Fatalf("hooks ran for a weapon without the ability: before %d hits %d", before, hits)
	}
	_, doubled := attack(testWeapon("test_weapon"), 1000)
	if before != 1 || hits != doubled || doubled < plain*2-1 || doubled > plain*2+1 {
		t.Fatalf("doubled damage %d (plain %d), before %d, hit hook damage %d", doubled, plain, before, hits)
	}
	if defeats != 0 {
		t.Fatal("defeat hook should not run when the enemy survives")
	}
	attack(testWeapon("test_weapon"), 1)
	if defeats != 1 {
		t.Fatalf("defeat hook calls = %d, want 1", defeats)
	}

	// 命中率を0にすると外れ、外れたときのフックが呼ばれる
	withTestAbilityHooks(t, "test_blunt", AbilityHooks{
		BeforePlayerAttack: func(g *Game, a *PlayerAttack, stacks int) { a.Accuracy = 0 },
		AfterPlayerMiss:    func(g *Game, a *PlayerAttack, stacks int) { misses++ },
	})
	_, missed := attack(testWeapon("test_blunt"), 1000)
	if missed != 0 || misses != 1 {
		t.Fatalf("zero accuracy: damage %d, miss hook calls %d", missed, misses)
	}

	// 武器以外の装備（指輪）の能力でも攻撃のフックは呼ばれる。持っているだけでは呼ばれない
	g := newFoundationTestGame()
	equip(g, testWeapon(), testRing("test_weapon"))
	g.state.Player.Inventory = append(g.state.Player.Inventory, testWeapon("test_blunt"))
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 4)}
	g.state.Enemies[0].Health = 1000
	before = 0
	g.CheckForEnemies(1, 0)
	if msg := runQueuedActions(g); before != 1 || !strings.Contains(msg, "試験！") {
		t.Fatalf("ring hook before=%d messages=%q", before, msg)
	}
}

func TestForceCriticalAndCriticalDenominatorHooks(t *testing.T) {
	withTestAbilityHooks(t, "test_crit", AbilityHooks{
		BeforePlayerAttack: func(g *Game, a *PlayerAttack, stacks int) { a.ForceCritical = true },
	})
	g := newFoundationTestGame()
	equip(g, testWeapon("test_crit"))
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 4)}
	g.state.Enemies[0].Health = 1000
	g.CheckForEnemies(1, 0)
	if msg := runQueuedActions(g); !strings.Contains(msg, "会心の一撃！") {
		t.Fatalf("forced critical messages = %q", msg)
	}
}

func TestIncomingDamageHook(t *testing.T) {
	var applied int
	withTestAbilityHooks(t, "test_guard", AbilityHooks{
		ModifyIncomingDamage: func(g *Game, in *IncomingDamage, stacks int) {
			if in.Attacker == nil || in.Kind != EnemyDamageNormal || in.Delivery != EnemyAttackMelee {
				t.Errorf("incoming = %+v, want melee from an enemy", in)
			}
			in.Damage /= 2
			in.Effects = append(in.Effects, func(g *Game) { applied++ })
		},
	})
	hit := func(armor *Armor) int {
		g := newFoundationTestGame()
		g.randHooks.shieldDefense = func(int) int { return 99 }
		g.state.Player.Health = 100
		equip(g, armor)
		enemy := CreateEnemyByID(18, 5, 4) // 大エビ（攻撃11）
		g.state.Enemies = []Enemy{enemy}
		g.enqueueEnemyNormalAttack(0)
		runQueuedActions(g)
		return 100 - g.state.Player.Health
	}
	full := hit(testArmor())
	if applied != 0 {
		t.Fatal("effects ran without the ability")
	}
	halved := hit(testArmor("test_guard"))
	if halved != full/2 || applied != 1 {
		t.Fatalf("damage %d → %d, effects applied %d", full, halved, applied)
	}

	// 回避された攻撃ではフックも効果も呼ばれない（盾の回避が先）
	g := newFoundationTestGame()
	g.randHooks.shieldDefense = func(int) int { return 0 }
	equip(g, testArmor(AbilityEvasion, "test_guard"))
	g.state.Enemies = []Enemy{CreateEnemyByID(18, 5, 4)}
	g.enqueueEnemyNormalAttack(0)
	runQueuedActions(g)
	if applied != 1 || g.state.Player.Health != 40 {
		t.Fatalf("evaded attack: effects %d, HP %d", applied, g.state.Player.Health)
	}
}

func TestPreventDeathHook(t *testing.T) {
	calls := 0
	withTestAbilityHooks(t, "test_guts", AbilityHooks{
		PreventDeath: func(g *Game, damage int, source DamageSource, stacks int) bool {
			calls++
			return true
		},
	})
	g := newFoundationTestGame()
	equip(g, testArmor("test_guts"))
	g.damagePlayer(10, DamageSource{Kind: DamageSourceStarvation})
	if calls != 0 || g.state.Player.Health != 30 {
		t.Fatalf("non-lethal damage: calls %d, HP %d", calls, g.state.Player.Health)
	}
	g.damagePlayer(100, DamageSource{Kind: DamageSourceStarvation})
	g.state.Player.checkDeath(g)
	if calls != 1 || g.state.Player.Health != 1 || g.playerDead {
		t.Fatalf("lethal damage with guts: calls %d, HP %d, dead %v", calls, g.state.Player.Health, g.playerDead)
	}

	// 装備していなければ倒れる（無効化）
	g = newFoundationTestGame()
	g.damagePlayer(100, DamageSource{Kind: DamageSourceStarvation})
	if g.state.Player.Health != 0 {
		t.Fatalf("lethal damage without guts: HP %d", g.state.Player.Health)
	}
}

func TestPlayerImmunityBlocksStatusAilments(t *testing.T) {
	withTestImmunity(t, "test_immunity", StatusSleep, StatusSlow, StatusConfusion, StatusBlind, StatusBearTrap, StatusPoison)

	newGame := func(immune bool) *Game {
		g := newFoundationTestGame()
		g.randHooks.statusResistance = func(int) int {
			t.Error("shield resistance should not be rolled")
			return 99
		}
		if immune {
			equip(g, testRing("test_immunity"))
		}
		return g
	}
	cases := []struct {
		name    string
		apply   func(g *Game)
		applied func(g *Game) bool
	}{
		{"sleep gas trap", sleepGasTrapEffect, func(g *Game) bool { return g.state.Player.StatusAilments.Sleep > 0 }},
		{"slow trap", slowTrapEffect, func(g *Game) bool { return g.state.Player.StatusAilments.Slow > 0 }},
		{"bear trap", bearTrapEffect, func(g *Game) bool { return g.state.Player.StatusAilments.BearTrap > 0 }},
		{"honey dance", func(g *Game) {
			g.state.Enemies = []Enemy{CreateEnemyByID(3, 5, 4)}
			g.state.Enemies[0].SpecialAttack(&g.state.Enemies[0], g)
		}, func(g *Game) bool { return g.state.Player.StatusAilments.Slow > 0 }},
		{"confusion potion", func(g *Game) {
			g.state.Player.Inventory = []Item{buildItemFromTemplate(18, 0, 0)}
			g.state.Player.Inventory[0].(*Potion).Use(g)
		}, func(g *Game) bool { return g.state.Player.StatusAilments.Confusion > 0 }},
		{"poison", func(g *Game) { g.drainPlayerPower(1) }, func(g *Game) bool { return g.state.Player.Power < 8 }},
	}
	for _, tc := range cases {
		g := newGame(false)
		tc.apply(g)
		runQueuedActions(g)
		if !tc.applied(g) {
			t.Errorf("%s: should apply without immunity", tc.name)
		}

		g = newGame(true)
		tc.apply(g)
		msg := runQueuedActions(g)
		if tc.applied(g) || !strings.Contains(msg, "試験の指輪が") || !strings.Contains(msg, "防いだ") {
			t.Errorf("%s: immune ring should block it, messages %q", tc.name, msg)
		}
	}

	g := newGame(true)
	if !g.playerImmuneTo(StatusSleep) || g.playerImmuneTo(StatusKind("test_other")) {
		t.Fatal("playerImmuneTo should follow the immunity table")
	}
}

func TestShieldResistanceStillRollsWithoutImmunity(t *testing.T) {
	g := newFoundationTestGame()
	rolled := 0
	g.randHooks.statusResistance = func(int) int { rolled++; return 0 }
	equip(g, buildItemFromTemplate(65, 0, 0)) // 状態異常耐性の盾
	if guard, blocked := g.playerStatusGuard(StatusSleep); !blocked || rolled != 1 || guard != g.state.Player.EquippedArmor.Name {
		t.Fatalf("shield guard = %q/%v, rolls %d", guard, blocked, rolled)
	}
}

// --- F2: 敵の属性 ---

func TestNewEnemyTraitsOnExistingEnemies(t *testing.T) {
	want := map[EnemyTrait][]int{
		EnemyTraitShell:     {5, 23, 50, 7, 25, 52, 9, 27, 54, 15, 33, 60, 16, 34, 61, 17, 35, 62, 42},
		EnemyTraitExplosive: {6, 24, 51},
		EnemyTraitFloating:  {10, 28, 55, 13, 31, 58, 39, 41, 65},
	}
	for trait, ids := range want {
		for _, id := range ids {
			if !hasEnemyTrait(MonsterDefinitions[id].Traits, trait) {
				t.Errorf("%s（%d）should have trait %s", MonsterDefinitions[id].Name, id, trait)
			}
		}
	}
	for _, id := range []int{0, 1, 2, 11, 12, 36, 38, 43, 44} {
		for _, trait := range []EnemyTrait{EnemyTraitShell, EnemyTraitExplosive, EnemyTraitFloating} {
			if hasEnemyTrait(MonsterDefinitions[id].Traits, trait) {
				t.Errorf("%s（%d）should not have trait %s", MonsterDefinitions[id].Name, id, trait)
			}
		}
	}
}

func TestShellSlayerAbilityHitsCrabs(t *testing.T) {
	damage := func(ability EquipmentAbilityID, enemyID int) (int, string) {
		g := newFoundationTestGame()
		equip(g, testWeapon(ability))
		enemy := CreateEnemyByID(enemyID, 5, 4)
		enemy.Health = 1000
		g.state.Enemies = []Enemy{enemy}
		g.CheckForEnemies(1, 0)
		msg := runQueuedActions(g)
		return 1000 - g.state.Enemies[0].Health, msg
	}
	plain, _ := damage(AbilitySureHit, 5)
	slayer, msg := damage(AbilityShellSlayer, 5)
	if slayer <= plain || !strings.Contains(msg, "特効！") {
		t.Fatalf("shell slayer vs crab: %d (plain %d), %q", slayer, plain, msg)
	}
	if other, msg := damage(AbilityShellSlayer, 0); strings.Contains(msg, "特効！") || other <= 0 {
		t.Fatalf("shell slayer vs shrimp should not be effective: %q", msg)
	}
}

// --- F3: 吹き飛ばし・引き寄せ・持ち物を落とす ---

func TestKnockbackPlayer(t *testing.T) {
	source := DamageSource{Kind: DamageSourceStarvation}
	g := newFoundationTestGame()
	result := g.knockbackPlayer(1, 0, 5, 7, source)
	runQueuedActions(g)
	if result.Moved != 3 || result.Stop != DisplacementHitWall || g.state.Player.X != 7 || g.state.Player.Health != 33 {
		t.Fatalf("knockback into wall: %+v, player (%d,%d) HP %d", result, g.state.Player.X, g.state.Player.Y, g.state.Player.Health)
	}

	// 敵に当たっても止まり、ダメージを受ける
	g = newFoundationTestGame()
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 6, 4)}
	if result := g.knockbackPlayer(1, 0, 5, 7, source); result.Moved != 1 || result.Stop != DisplacementHitCharacter || g.state.Player.Health != 33 {
		t.Fatalf("knockback into enemy: %+v, HP %d", result, g.state.Player.Health)
	}

	// 受け身ならぶつかってもダメージを受けない
	g = newFoundationTestGame()
	equip(g, testRing(AbilityBreakFall))
	if result := g.knockbackPlayer(1, 0, 5, 7, source); result.Moved != 3 || result.Damage != 0 || g.state.Player.Health != 40 {
		t.Fatalf("break fall: %+v, HP %d", result, g.state.Player.Health)
	}

	// 不動なら動かない
	g = newFoundationTestGame()
	equip(g, testArmor(AbilityImmovable))
	if result := g.knockbackPlayer(1, 0, 5, 7, source); !result.Prevented || g.state.Player.X != 4 || g.state.Player.Health != 40 {
		t.Fatalf("immovable: %+v, player x %d HP %d", result, g.state.Player.X, g.state.Player.Health)
	}

	// 止まったマスの罠を踏む
	g = newFoundationTestGame()
	stepped := 0
	g.state.MapTraps = []MapTrap{{X: 6, Y: 4, Name: "試験の罠", Effect: func(g *Game) { stepped++ }}}
	g.knockbackPlayer(1, 0, 2, 7, source)
	runQueuedActions(g)
	if stepped != 1 || g.state.Player.Health != 40 {
		t.Fatalf("landing trap steps %d, HP %d", stepped, g.state.Player.Health)
	}
}

func TestKnockbackEnemy(t *testing.T) {
	g := newFoundationTestGame()
	enemy := CreateEnemyByID(0, 5, 4)
	enemy.Health = 100
	g.state.Enemies = []Enemy{enemy}
	result := g.knockbackEnemy(0, 1, 0, 5, 9)
	runQueuedActions(g)
	if result.Moved != 2 || result.Stop != DisplacementHitWall || g.state.Enemies[0].X != 7 || g.state.Enemies[0].Health != 91 {
		t.Fatalf("enemy knockback: %+v, enemy %+v", result, g.state.Enemies[0].Entity)
	}
	// ぶつかって倒れたら通常の撃破処理（経験値）になる
	g.state.Enemies[0].Health = 5
	g.knockbackEnemy(0, 1, 0, 1, 9)
	if len(g.state.Enemies) != 0 || g.state.Player.ExperiencePoints == 0 {
		t.Fatalf("collision kill: enemies %d, exp %d", len(g.state.Enemies), g.state.Player.ExperiencePoints)
	}
	// ボスは吹き飛ばない
	g = newFoundationTestGame()
	g.state.Enemies = []Enemy{CreateEnemyByID(42, 5, 4)}
	if result := g.knockbackEnemy(0, 1, 0, 5, 9); !result.Prevented || g.state.Enemies[0].X != 5 {
		t.Fatalf("boss knockback: %+v", result)
	}
}

func TestPullPlayer(t *testing.T) {
	g := newFoundationTestGame()
	g.state.Player.X = 1
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 6, 4)}
	result := g.pullPlayer(1, 0, 6)
	if result.Moved != 4 || result.Stop != DisplacementHitCharacter || g.state.Player.X != 5 {
		t.Fatalf("pull: %+v, player x %d", result, g.state.Player.X)
	}

	// 途中の罠で止まって踏む
	g = newFoundationTestGame()
	g.state.Player.X = 1
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 6, 4)}
	stepped := 0
	g.state.MapTraps = []MapTrap{{X: 3, Y: 4, Name: "試験の罠", Effect: func(g *Game) { stepped++ }}}
	g.pullPlayer(1, 0, 6)
	runQueuedActions(g)
	if g.state.Player.X != 3 || stepped != 1 {
		t.Fatalf("pull onto trap: player x %d, steps %d", g.state.Player.X, stepped)
	}

	// 不動なら動かない
	g = newFoundationTestGame()
	g.state.Player.X = 1
	equip(g, testRing(AbilityImmovable))
	if result := g.pullPlayer(1, 0, 6); !result.Prevented || g.state.Player.X != 1 {
		t.Fatalf("immovable pull: %+v", result)
	}
}

func TestImmovableBlocksSwap(t *testing.T) {
	newGame := func(immovable bool) *Game {
		g := newFoundationTestGame()
		g.rooms = []Room{{ID: 1, X: 0, Y: 0, Width: 9, Height: 9}}
		g.state.Player.X, g.state.Player.Y = 1, 4
		g.state.Enemies = []Enemy{CreateEnemyByID(14, 4, 4)} // イレカエダコ
		if immovable {
			equip(g, testArmor(AbilityImmovable))
		}
		return g
	}
	always := func(int) int { return 0 }
	if g := newGame(false); !g.trySwapEnemyWithPlayer(0, always) || g.state.Player.X != 4 {
		t.Fatal("swap should work without immovable")
	}
	if g := newGame(true); g.trySwapEnemyWithPlayer(0, always) || g.state.Player.X != 1 {
		t.Fatal("immovable should block swap")
	}
}

func TestDropPlayerItemsForward(t *testing.T) {
	g := newFoundationTestGame()
	weapon := testWeapon()
	equip(g, weapon)
	food := buildItemFromTemplate(1, 0, 0)
	potion := buildItemFromTemplate(2, 0, 0)
	g.state.Player.Inventory = append(g.state.Player.Inventory, food, potion)
	g.randHooks.displacement = func(int) int { return 0 }

	dropped := g.dropPlayerItemsForward(2, 1, 0)
	runQueuedActions(g)
	if len(dropped) != 2 || len(g.state.Player.Inventory) != 1 || g.state.Player.Inventory[0] != weapon {
		t.Fatalf("dropped %v, inventory %v (equipped weapon must stay)", dropped, g.state.Player.Inventory)
	}
	got := map[Coordinate]bool{}
	for _, item := range g.state.Items {
		x, y := item.GetPosition()
		got[Coordinate{X: x, Y: y}] = true
	}
	if !got[Coordinate{X: 5, Y: 4}] || !got[Coordinate{X: 6, Y: 4}] {
		t.Fatalf("dropped positions = %v, want 1 and 2 tiles ahead", got)
	}

	// 落とせる物が無ければ何も起きない
	if dropped := g.dropPlayerItemsForward(2, 1, 0); len(dropped) != 0 {
		t.Fatalf("dropped equipped items: %v", dropped)
	}
}

// --- F4: 敵の痛恨の一撃と回避 ---

func TestEnemyCriticalHit(t *testing.T) {
	const testID = 9901
	withTestMonster(t, testID, 18, func(d *MonsterDefinition) { d.CriticalPercent = 100 })

	hit := func(id int, edit func(g *Game)) (int, string) {
		g := newFoundationTestGame()
		g.state.Player.Health = 100
		g.randHooks.enemyCritical = func(int) int { return 0 }
		g.state.Enemies = []Enemy{CreateEnemyByID(id, 5, 4)}
		if edit != nil {
			edit(g)
		}
		g.enqueueEnemyNormalAttack(0)
		msg := runQueuedActions(g)
		return 100 - g.state.Player.Health, msg
	}

	normal, _ := hit(18, func(g *Game) {
		g.randHooks.enemyCritical = func(int) int {
			t.Error("enemy without critical chance should not roll")
			return 0
		}
	})
	critical, msg := hit(testID, nil)
	if !strings.Contains(msg, "痛恨の一撃！") || critical < normal*3/2-1 || critical > normal*3/2+1 {
		t.Fatalf("critical %d (normal %d), %q", critical, normal, msg)
	}
	// 封印された敵は痛恨を出さない
	if sealed, msg := hit(testID, func(g *Game) { g.state.Enemies[0].StatusAilments.Seal = true }); sealed != normal || strings.Contains(msg, "痛恨") {
		t.Fatalf("sealed critical %d, %q", sealed, msg)
	}

	// 装備のフックで痛恨の確率を上げられ、被ダメージのフックに痛恨かどうかが渡る
	var sawCritical bool
	withTestAbilityHooks(t, "test_critical_ring", AbilityHooks{
		BeforeEnemyAttack:    func(g *Game, a *EnemyAttack, stacks int) { a.CriticalPercent += 100 },
		ModifyIncomingDamage: func(g *Game, in *IncomingDamage, stacks int) { sawCritical = in.Critical },
	})
	ringed, msg := hit(18, func(g *Game) { equip(g, testRing("test_critical_ring")) })
	if ringed != critical || !sawCritical || !strings.Contains(msg, "痛恨") {
		t.Fatalf("critical ring damage %d (want %d), incoming critical %v, %q", ringed, critical, sawCritical, msg)
	}
}

func TestEnemyEvadesNormalAttack(t *testing.T) {
	const testID = 9902
	withTestMonster(t, testID, 0, func(d *MonsterDefinition) { d.EvasionPercent = 100 })
	misses := 0
	withTestAbilityHooks(t, "test_counter", AbilityHooks{AfterPlayerMiss: func(g *Game, a *PlayerAttack, stacks int) { misses++ }})

	attack := func(weapon *Weapon, edit func(e *Enemy)) (int, string) {
		g := newFoundationTestGame()
		g.randHooks.enemyEvasion = func(int) int { return 0 }
		equip(g, weapon)
		enemy := CreateEnemyByID(testID, 5, 4)
		enemy.Health = 1000
		if edit != nil {
			edit(&enemy)
		}
		g.state.Enemies = []Enemy{enemy}
		g.CheckForEnemies(1, 0)
		msg := runQueuedActions(g)
		return 1000 - g.state.Enemies[0].Health, msg
	}

	if damage, msg := attack(testWeapon("test_counter"), nil); damage != 0 || !strings.Contains(msg, "かわした") || misses != 1 {
		t.Fatalf("evasion: damage %d, %q, miss hook %d", damage, msg, misses)
	}
	// 必中は回避を無視する
	if damage, _ := attack(testWeapon(AbilitySureHit), nil); damage == 0 {
		t.Fatal("sure hit should ignore evasion")
	}
	// 眠っている・封印された敵はかわさない
	for name, edit := range map[string]func(*Enemy){
		"sleep": func(e *Enemy) { e.StatusAilments.Sleep = 3 },
		"seal":  func(e *Enemy) { e.StatusAilments.Seal = true },
	} {
		if damage, _ := attack(testWeapon(), edit); damage == 0 {
			t.Errorf("%s enemy should not evade", name)
		}
	}
}

// --- F5: アイテムの床出現のカテゴリ別の重み ---

func TestItemCategorySpawnWeights(t *testing.T) {
	types := map[string]int{}
	for _, template := range itemTemplates {
		types[template.ItemType]++
	}
	// どの階でもすべてのカテゴリに重みがあり、道具が1つ以上出る。
	for floor := 1; floor <= finalFloor; floor++ {
		for itemType := range types {
			if itemCategoryWeight(itemType, floor) <= 0 {
				t.Errorf("floor %d: item type %q has no category spawn weight (its items would never appear on the floor)", floor, itemType)
			}
		}
		if got := len(floorItemCategoriesFor(floor)); got != len(types) {
			t.Errorf("floor %d: %d categories can appear, want %d", floor, got, len(types))
		}
	}
	for name := range itemCategoryBandWeights {
		if types[name] == 0 {
			t.Errorf("category weight %q has no item templates", name)
		}
	}
	// 導入時の割合を全階で維持し、wave4-A で指輪だけ重みを1から3へ増やした（wave9-A の初期値も同じ）。
	want := map[string]int{"Card": 31, "Armor": 14, "Weapon": 12, "Potion": 7, "Food": 4, "Cane": 2, "Accessory": 3, "Arrow": 1, "Money": 1, "Pot": 1, "Trap": 1}
	for _, floor := range []int{1, 5, 10, 15, 20, 25, 30} {
		for _, category := range floorItemCategoriesFor(floor) {
			if category.Weight != want[category.Name] {
				t.Errorf("floor %d: category %s weight = %d, want %d", floor, category.Name, category.Weight, want[category.Name])
			}
		}
	}
	// createFloorItem は各カテゴリから選ぶ（カテゴリの抽選 r を全通り試す。補正なしでは重みは spawnWeightScale 倍）
	total := 0
	for _, w := range want {
		total += w
	}
	counts := map[string]int{}
	for r := 0; r < total; r++ {
		calls := 0
		item := createFloorItem(0, 0, 1, DifficultyAdvanced, func(n int) int {
			calls++
			if calls == 1 {
				return r * spawnWeightScale
			}
			return 0
		})
		counts[itemKind(item)]++
	}
	for name, w := range want {
		if counts[name] != w {
			t.Errorf("createFloorItem category %s = %d of %d, want %d", name, counts[name], total, w)
		}
	}
}

// TestItemFloorRulesSanity は道具ごとの規則の健全性を確かめる（wave9-A）。
func TestItemFloorRulesSanity(t *testing.T) {
	for id, rule := range itemFloorRules {
		_, exists := itemTemplates[id]
		if !exists && (id < 225 || id > 239) {
			t.Errorf("rule for unknown item %d (only wave9-C's 225〜239 may be missing)", id)
		}
		if rule.From < 0 || rule.Until < 0 || rule.Weight < 0 {
			t.Errorf("item %d: negative rule %+v", id, rule)
		}
		if rule.Until > 0 && rule.Until <= max(rule.From, 1) {
			t.Errorf("item %d: stops at %d before it starts at %d", id, rule.Until, rule.From)
		}
		if rule.From > finalFloor {
			t.Errorf("item %d: starts at %d below the final floor", id, rule.From)
		}
		if rule.Heal && rule.Strong {
			t.Errorf("item %d: heal and strong at the same time", id)
		}
		if exists && rule.Heal && itemTemplates[id].ItemType != "Potion" {
			t.Errorf("item %d: heal flag is for potions (food is tuned by category)", id)
		}
		// 強い道具は1階からは出さない。
		if rule.Strong && rule.From <= 1 {
			t.Errorf("strong item %d appears from floor 1", id)
		}
	}
}

// floorTableHas は floor 階の抽選表に id があるか。
func floorTableHas(floor, id int) bool {
	for _, category := range floorItemCategoriesFor(floor) {
		for _, entry := range category.Entries {
			if entry.ID == id {
				return true
			}
		}
	}
	return false
}

// TestItemFloorRulesDelayStrongItems は出始める階より浅い階で抽選に入らないことを確かめる。
func TestItemFloorRulesDelayStrongItems(t *testing.T) {
	for id, rule := range itemFloorRules {
		if _, ok := itemTemplates[id]; !ok {
			if floorTableHas(finalFloor, id) {
				t.Errorf("missing template %d is in the floor table", id)
			}
			continue
		}
		for floor := 1; floor <= finalFloor; floor++ {
			want := rule.appearsOnFloor(floor)
			if got := floorTableHas(floor, id); got != want {
				t.Errorf("item %d on floor %d: in table = %v, want %v (rule %+v)", id, floor, got, want, rule)
			}
		}
	}
	// 具体例: 全滅・ジェノサイドのカードと伝説の剣は9階から、無敵の薬は4階から、火迅の太刀は13階から。
	for _, c := range []struct{ id, from int }{{56, 9}, {54, 9}, {4, 9}, {217, 4}, {146, 13}} {
		if floorTableHas(c.from-1, c.id) || !floorTableHas(c.from, c.id) {
			t.Errorf("item %d should start on floor %d", c.id, c.from)
		}
	}
	// 通常の道具はどの階でも出る。
	for floor := 1; floor <= finalFloor; floor++ {
		if !floorTableHas(floor, 1) || !floorTableHas(floor, 20) || !floorTableHas(floor, 7) {
			t.Fatalf("floor %d: basic items missing", floor)
		}
	}
}

// TestPickFloorItemIDDistribution は抽選の割合が表の重みどおりになることを、カテゴリ内の乱数を全通り回して確かめる。
func TestPickFloorItemIDDistribution(t *testing.T) {
	for _, floor := range []int{1, 9, 30} {
		categories := floorItemCategoriesFor(floor)
		catTotal := 0
		for _, c := range categories {
			catTotal += c.Weight
		}
		offset := 0
		for _, c := range categories {
			entryTotal := 0
			for _, e := range c.Entries {
				entryTotal += e.Weight
			}
			counts := map[int]int{}
			for r := 0; r < entryTotal*spawnWeightScale; r++ {
				calls := 0
				id := pickFloorItemID(floor, DifficultyBeginner, func(n int) int {
					calls++
					if calls == 1 {
						if n != catTotal*spawnWeightScale {
							t.Fatalf("category total %d, want %d", n, catTotal*spawnWeightScale)
						}
						return offset * spawnWeightScale
					}
					return r
				})
				counts[id]++
			}
			for _, e := range c.Entries {
				if counts[e.ID] != e.Weight*spawnWeightScale {
					t.Errorf("floor %d %s item %d drawn %d, want %d", floor, c.Name, e.ID, counts[e.ID], e.Weight*spawnWeightScale)
				}
			}
			offset += c.Weight
		}
	}
}

// TestFloorTrapTableSanity は罠の帯別の重みの健全性を確かめる（wave9-A）。
func TestFloorTrapTableSanity(t *testing.T) {
	for id := range mapTrapTemplates {
		if _, ok := trapBandWeights[id]; !ok {
			t.Errorf("trap %d has no band weights (it would never appear on the floor)", id)
		}
	}
	for id, weights := range trapBandWeights {
		if _, ok := mapTrapTemplates[id]; !ok {
			t.Errorf("band weights for unknown trap %d", id)
		}
		for i, w := range weights {
			if w < 0 {
				t.Errorf("trap %d band %d: negative weight", id, i)
			}
		}
	}
	for floor := 1; floor <= finalFloor; floor++ {
		ids := availableTrapIDs(randomMapTrapIDs, floor)
		total := 0
		for _, id := range ids {
			total += trapFloorWeight(id, floor)
		}
		if total <= 0 {
			t.Fatalf("floor %d has no traps", floor)
		}
		// 抽選を全通り試し、重みどおりの回数と、ボス階・30階で落とし穴が出ないことを確かめる。
		counts := map[int]int{}
		for r := 0; r < total; r++ {
			counts[pickFloorTrapID(floor, func(int) int { return r })]++
		}
		for _, id := range ids {
			if counts[id] != trapFloorWeight(id, floor) {
				t.Errorf("floor %d trap %d drawn %d, want %d", floor, id, counts[id], trapFloorWeight(id, floor))
			}
		}
		if (isBossFloor(floor) || floor >= finalFloor) && counts[pitfallTrapID] > 0 {
			t.Errorf("floor %d draws a pitfall", floor)
		}
	}
	// 浅い階は大型地雷・鉄の矢を出さず、深い階ほど大型地雷・鉄の矢・落石・回復封じが増える。
	if trapFloorWeight(11, 1) != 0 || trapFloorWeight(9, 3) != 0 || trapFloorWeight(11, 4) == 0 {
		t.Error("shallow floors should not have large mines or iron arrows")
	}
	for _, id := range []int{9, 10, 11, 15} {
		if trapFloorWeight(id, 26) <= trapFloorWeight(id, 1) {
			t.Errorf("trap %d should be more common deep", id)
		}
	}
}

// TestDifficultySpawnTuningNeutral は難易度の補正がまだすべて 1.0（補正なし）であることを確かめる。
// wave11 で値を入れたらこのテストを直す。
func TestDifficultySpawnTuningNeutral(t *testing.T) {
	for _, d := range allDifficulties {
		if got := spawnTuningFor(d); got != neutralSpawnTuning {
			t.Errorf("%v tuning = %+v, want neutral", d, got)
		}
		if got := spawnTuningFor(d).floorItemCount(); got != floorItemBaseCount {
			t.Errorf("%v item count = %d, want %d", d, got, floorItemBaseCount)
		}
	}
}

// TestFloorItemsUseFloorTable は生成した浅い階に、まだ出ない道具・罠が置かれないことを確かめる。
func TestFloorItemsUseFloorTable(t *testing.T) {
	for seed := int64(1); seed <= 20; seed++ {
		for _, floor := range []int{1, 2, 3} {
			player := newInitialPlayer()
			_, _, items, _, _, traps := GenerateRandomMapFor(70, 70, floor-1, &player, newFloorRNG(seed, floor), DifficultyBeginner)
			if len(items) != floorItemBaseCount {
				t.Fatalf("seed %d floor %d: %d items", seed, floor, len(items))
			}
			for _, item := range items {
				if !floorTableHas(floor, item.GetID()) {
					t.Errorf("seed %d floor %d: item %d is not in the floor table", seed, floor, item.GetID())
				}
			}
			for _, trap := range traps {
				if trapFloorWeight(trap.TemplateID, floor) == 0 {
					t.Errorf("seed %d floor %d: trap %d on a shallow floor", seed, floor, trap.TemplateID)
				}
			}
		}
	}
}

// --- 段1の準備: 攻撃のたびのフック・総攻撃力の加算・命中時の処理・被弾の回避・追加の無効化 ---

func TestSwingHookAttackBonusAndHitEffects(t *testing.T) {
	var swings []PlayerSwing
	paid := 0
	withTestAbilityHooks(t, "test_swing", AbilityHooks{
		AfterPlayerSwing: func(g *Game, s PlayerSwing, stacks int) { swings = append(swings, s) },
		BeforePlayerAttack: func(g *Game, a *PlayerAttack, stacks int) {
			a.AttackBonus += 10
			a.HitEffects = append(a.HitEffects, func(g *Game) { paid++ })
		},
	})
	attack := func(abilities ...EquipmentAbilityID) (int, *Game) {
		g := newFoundationTestGame()
		equip(g, testWeapon(abilities...))
		g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 4)}
		g.state.Enemies[0].Health = 1000
		g.CheckForEnemies(1, 0)
		runQueuedActions(g)
		return 1000 - g.state.Enemies[0].Health, g
	}
	plain, _ := attack()
	bonus, _ := attack("test_swing")
	if bonus <= plain || paid != 1 {
		t.Fatalf("attack bonus damage %d (plain %d), hit effects %d", bonus, plain, paid)
	}
	if len(swings) != 1 || swings[0] != (PlayerSwing{DX: 1, DY: 0, HitEnemy: true}) {
		t.Fatalf("swings = %+v", swings)
	}

	// 空振りでもフックは呼ばれ、外れたら命中時の処理は呼ばれない
	g := newFoundationTestGame()
	equip(g, testWeapon("test_swing"))
	g.CheckForEnemies(0, -1)
	runQueuedActions(g)
	if len(swings) != 2 || swings[1].HitEnemy || swings[1].DY != -1 {
		t.Fatalf("empty swing = %+v", swings)
	}
	g = newFoundationTestGame()
	g.randHooks.attackHit = func(int) int { return 99 }
	equip(g, testWeapon("test_swing"))
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 4)}
	g.CheckForEnemies(1, 0)
	runQueuedActions(g)
	if paid != 1 {
		t.Fatalf("hit effects ran on a miss: %d", paid)
	}
}

func TestIncomingDamageEvadedByHook(t *testing.T) {
	withTestAbilityHooks(t, "test_dodge", AbilityHooks{
		ModifyIncomingDamage: func(g *Game, in *IncomingDamage, stacks int) {
			in.Evaded, in.EvadedBy = true, "試験の甲羅"
		},
	})
	g := newFoundationTestGame()
	g.randHooks.shieldDefense = func(int) int { return 99 }
	equip(g, testArmor(AbilityCounter, "test_dodge"))
	g.state.Enemies = []Enemy{CreateEnemyByID(18, 5, 4)}
	g.state.Enemies[0].Health = 100
	g.enqueueEnemyNormalAttack(0)
	msg := runQueuedActions(g)
	if g.state.Player.Health != 40 || g.state.Enemies[0].Health != 100 || !strings.Contains(msg, "試験の甲羅で") || !strings.Contains(msg, "かわした") {
		t.Fatalf("evaded: HP %d, enemy HP %d, %q", g.state.Player.Health, g.state.Enemies[0].Health, msg)
	}
}

func TestExtraImmunitiesAndRust(t *testing.T) {
	withTestImmunity(t, "test_ward", StatusCurse, StatusFoodTransform, StatusManipulation, StatusRust)
	newGame := func(warded bool) *Game {
		g := newFoundationTestGame()
		g.state.Player.Inventory = []Item{testWeapon(), buildItemFromTemplate(18, 0, 0)}
		if warded {
			equip(g, testRing("test_ward"))
		}
		g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 4)}
		return g
	}
	attacks := map[string]func(g *Game){
		"curse":      func(g *Game) { enqueueCurseAttack(&g.state.Enemies[0], g, func(int) int { return 0 }) },
		"food":       func(g *Game) { enqueueFoodTransformationAttack(&g.state.Enemies[0], g, func(int) int { return 0 }) },
		"manipulate": func(g *Game) { enqueueManipulationAttack(&g.state.Enemies[0], g, func(int) int { return 0 }) },
	}
	for name, apply := range attacks {
		g := newGame(true)
		apply(g)
		if msg := runQueuedActions(g); !strings.Contains(msg, "防いだ") {
			t.Errorf("%s: warded messages = %q", name, msg)
		}
		g = newGame(false)
		apply(g)
		if msg := runQueuedActions(g); strings.Contains(msg, "防いだ") {
			t.Errorf("%s: unwarded messages = %q", name, msg)
		}
	}

	// さび: 自分だけさびない能力（金の剣など）と、装備全体を守る無効化
	rust := func(weapon *Weapon, ring bool) int {
		g := newFoundationTestGame()
		g.randHooks = randHooks{}
		equip(g, weapon)
		if ring {
			equip(g, testRing("test_ward"))
		}
		rustTrapEffect(g)
		runQueuedActions(g)
		return weapon.Sharpness
	}
	if got := rust(testWeapon(), false); got != -1 {
		t.Fatalf("plain weapon sharpness after rust = %d, want -1", got)
	}
	if got := rust(testWeapon(AbilityRustImmune), false); got != 0 {
		t.Fatalf("rust-immune weapon sharpness = %d, want 0", got)
	}
	if got := rust(testWeapon(), true); got != 0 {
		t.Fatalf("rust-warded weapon sharpness = %d, want 0", got)
	}
}

func TestMergeItemTemplatesRejectsDuplicates(t *testing.T) {
	merged := mergeItemTemplates(map[int]ItemTemplate{1: {ID: 1}}, map[int]ItemTemplate{2: {ID: 2}})
	if len(merged) != 2 {
		t.Fatalf("merged = %v", merged)
	}
	for name, tables := range map[string][]map[int]ItemTemplate{
		"duplicate": {{1: {ID: 1}}, {1: {ID: 1}}},
		"wrong ID":  {{3: {ID: 4}}},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: mergeItemTemplates should panic", name)
				}
			}()
			mergeItemTemplates(tables...)
		}()
	}
}

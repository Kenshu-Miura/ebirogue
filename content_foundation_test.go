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
	for itemType := range types {
		if itemCategorySpawnWeights[itemType] <= 0 {
			t.Errorf("item type %q has no category spawn weight (its items would never appear on the floor)", itemType)
		}
	}
	for name := range itemCategorySpawnWeights {
		if types[name] == 0 {
			t.Errorf("category weight %q has no item templates", name)
		}
	}
	// 導入時の割合（全テンプレートが同じ重み、パワーアップのカードだけ4）と一致する
	want := map[string]int{"Card": 31, "Armor": 14, "Weapon": 12, "Potion": 7, "Food": 4, "Cane": 2, "Accessory": 1, "Arrow": 1, "Money": 1, "Pot": 1, "Trap": 1}
	for _, category := range itemSpawnCategories {
		if category.Weight != want[category.Name] {
			t.Errorf("category %s weight = %d, want %d", category.Name, category.Weight, want[category.Name])
		}
	}
	if len(itemSpawnCategories) != len(want) {
		t.Fatalf("categories = %d, want %d", len(itemSpawnCategories), len(want))
	}
	// createItem は各カテゴリから選ぶ（カテゴリの抽選 r を全通り試す）
	total := 0
	for _, w := range want {
		total += w
	}
	counts := map[string]int{}
	for r := 0; r < total; r++ {
		calls := 0
		item := createItem(0, 0, func(n int) int {
			calls++
			if calls == 1 {
				return r
			}
			return 0
		})
		counts[itemKind(item)]++
	}
	for name, w := range want {
		if counts[name] != w {
			t.Errorf("createItem category %s = %d of %d, want %d", name, counts[name], total, w)
		}
	}
}

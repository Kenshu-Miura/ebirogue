//go:build !test

package main

// 段2（追加計画の敵 A）の共通基盤の結合テスト: 系統の表の登録、敵の行動フック、撃破時のフック、
// ほかの敵の回復、仲間を呼ぶ、固定ダメージの特殊攻撃、遅れて現れる敵。
// 新しい敵はまだ無いので、仮の敵 ID（990〜）の定義を既存の敵から写して確かめる。

import (
	"encoding/json"
	"strings"
	"testing"
)

func addTestEnemy(g *Game, id, x, y int) int {
	enemy := CreateEnemyByID(id, x, y)
	enemy.PlayerDiscovered = true
	g.state.Enemies = append(g.state.Enemies, enemy)
	return len(g.state.Enemies) - 1
}

// --- 系統の表 ---

func testMonsterFamily() monsterFamily {
	base := MonsterDefinitions[0]
	base.ID, base.Type, base.Name, base.Health = 995, "TestFamilyBase", "試験エビ", 10
	base.Behavior = func(*Game, int) bool { return false }
	return monsterFamily{
		Base:      base,
		BaseImage: "img/ebi.png",
		Second:    monsterStage{ID: 996, Type: "TestFamilySecond", Name: "試験大エビ", Health: 20, Attack: 8, Defense: 4, Experience: 10, Image: "img/dai_ebi.png"},
		Third:     monsterStage{ID: 997, Type: "TestFamilyThird", Name: "試験王エビ", Health: 40, Attack: 16, Defense: 6, Experience: 25, Image: "img/ousama_ebi.png"},
		Spawns:    []monsterSpawnRange{{MonsterID: 995, From: 9, To: 9, Weight: 7}},
	}
}

func TestMonsterFamilyDefinitionsInheritBaseAbilities(t *testing.T) {
	definitions := map[int]MonsterDefinition{}
	addMonsterFamilies(definitions, []monsterFamily{testMonsterFamily()})
	base, second, third := definitions[995], definitions[996], definitions[997]
	if base.MaxHealth != 10 || second.MaxHealth != 20 || third.MaxHealth != 40 {
		t.Fatalf("max health = %d/%d/%d, want 10/20/40", base.MaxHealth, second.MaxHealth, third.MaxHealth)
	}
	if second.Behavior == nil || third.Behavior == nil || third.Name != "試験王エビ" || third.AttackPower != 16 {
		t.Fatalf("upper stages did not inherit the base behavior: %+v", third)
	}
}

func TestMonsterFamilySpawnsCopyBandSlices(t *testing.T) {
	shared := []MonsterSpawnEntry{{MonsterID: 45, Weight: 5}}
	tables := map[int][]MonsterSpawnEntry{9: shared, 10: shared}
	applyMonsterFamilySpawns(tables, []monsterFamily{testMonsterFamily()})
	if len(tables[9]) != 2 || tables[9][1] != (MonsterSpawnEntry{MonsterID: 995, Weight: 7}) {
		t.Fatalf("floor 9 = %+v, want the family entry appended", tables[9])
	}
	if len(tables[10]) != 1 || len(shared) != 1 {
		t.Fatalf("floor 10 = %+v, shared = %+v; other floors in the band must not change", tables[10], shared)
	}
}

func TestRegisterMonsterFamiliesAddsLevelUpsAndImages(t *testing.T) {
	family := testMonsterFamily()
	t.Cleanup(func() {
		delete(MonsterLevelUpTable, 995)
		delete(MonsterLevelUpTable, 996)
		for _, kind := range []string{"TestFamilyBase", "TestFamilySecond", "TestFamilyThird"} {
			delete(upperEnemyImagePaths, kind)
		}
	})
	registerMonsterFamilies([]monsterFamily{family})
	if MonsterLevelUpTable[995] != 996 || MonsterLevelUpTable[996] != 997 {
		t.Fatalf("level-up table = %d -> %d -> %d", 995, MonsterLevelUpTable[995], MonsterLevelUpTable[996])
	}
	if upperEnemyImagePaths["TestFamilyBase"] != "img/ebi.png" || upperEnemyImagePaths["TestFamilyThird"] != "img/ousama_ebi.png" {
		t.Fatal("family images were not registered")
	}
}

// --- 行動フック ---

func TestEnemyBehaviorRunsBeforeNormalActionAndRespectsStatus(t *testing.T) {
	calls := 0
	withTestMonster(t, 990, 0, func(d *MonsterDefinition) {
		d.Behavior = func(*Game, int) bool { calls++; return true }
	})
	g := newFoundationTestGame()
	addTestEnemy(g, 990, 5, 4)

	g.actEnemy(0)
	if calls != 1 || len(g.ActionQueue.Queue) != 0 {
		t.Fatalf("behavior calls = %d, queued = %d; want 1 call and no normal attack", calls, len(g.ActionQueue.Queue))
	}

	g.state.Enemies[0].StatusAilments.Sleep = 3
	g.actEnemy(0)
	if calls != 1 {
		t.Fatal("sleeping enemy used its behavior")
	}

	g.state.Enemies[0].StatusAilments.Sleep = 0
	g.state.Enemies[0].StatusAilments.Seal = true
	g.actEnemy(0)
	if calls != 1 || len(g.ActionQueue.Queue) == 0 {
		t.Fatalf("sealed enemy: behavior calls = %d, queued = %d; want the normal attack instead", calls, len(g.ActionQueue.Queue))
	}
}

func TestEnemyBehaviorFalseContinuesNormalAction(t *testing.T) {
	withTestMonster(t, 990, 0, func(d *MonsterDefinition) {
		d.Behavior = func(*Game, int) bool { return false }
	})
	g := newFoundationTestGame()
	g.randHooks.shieldDefense = func(int) int { return 99 }
	addTestEnemy(g, 990, 5, 4)
	g.actEnemy(0)
	if messages := runQueuedActions(g); !strings.Contains(messages, "ダメージ") {
		t.Fatalf("messages = %q, want the normal attack", messages)
	}
}

func TestStepEnemyTowardTarget(t *testing.T) {
	g := newFoundationTestGame()
	g.state.Player.X, g.state.Player.Y = 7, 1
	addTestEnemy(g, 0, 1, 1)
	addTestEnemy(g, 2, 4, 4)
	if !g.stepEnemyToward(0, 4, 4) || g.state.Enemies[0].X != 2 || g.state.Enemies[0].Y != 2 {
		t.Fatalf("enemy at (%d,%d), want (2,2)", g.state.Enemies[0].X, g.state.Enemies[0].Y)
	}
	g.state.Enemies[0].X, g.state.Enemies[0].Y = 3, 3
	g.stepEnemyToward(0, 4, 4)
	if g.state.Enemies[0].X == 4 && g.state.Enemies[0].Y == 4 {
		t.Fatal("enemy stepped onto the occupied target")
	}
}

// --- ほかの敵の回復 ---

func TestInjuredAllyNearAndHealEnemy(t *testing.T) {
	g := newFoundationTestGame()
	g.state.Player.X, g.state.Player.Y = 7, 1
	healer := addTestEnemy(g, 2, 2, 2)
	near := addTestEnemy(g, 0, 3, 2)
	g.state.Enemies[near].Health -= 5
	most := addTestEnemy(g, 0, 2, 4)
	g.state.Enemies[most].Health -= 10
	far := addTestEnemy(g, 0, 7, 7)
	g.state.Enemies[far].Health -= 15
	boss := addTestEnemy(g, 42, 3, 3)
	g.state.Enemies[boss].Health -= 50
	g.state.Enemies[healer].Health = 1

	if got := g.injuredAllyNear(healer, 3); got != most {
		t.Fatalf("injured ally = %d, want %d (most HP missing, not self, not boss, within range)", got, most)
	}
	if healed := g.healEnemy(most, 100); healed != 10 || g.state.Enemies[most].Health != g.state.Enemies[most].MaxHealth {
		t.Fatalf("healed = %d, health = %d/%d", healed, g.state.Enemies[most].Health, g.state.Enemies[most].MaxHealth)
	}
	if got := g.injuredAllyNear(healer, 3); got != near {
		t.Fatalf("injured ally after healing = %d, want %d", got, near)
	}
}

// --- 仲間を呼ぶ ---

func TestSummonMonstersAround(t *testing.T) {
	g := newFoundationTestGame()
	g.Floor = 1 // エビ・マムル・毒ヘビ
	g.genocidedMonsterIDs = map[int]bool{1: true}
	addTestEnemy(g, 0, 2, 2)

	if got := g.summonMonstersAround(2, 2, 1, 3, []int{0}); got != 3 {
		t.Fatalf("summoned = %d, want 3", got)
	}
	for _, enemy := range g.state.Enemies[1:] {
		if enemy.ID != 2 || max(abs(enemy.X-2), abs(enemy.Y-2)) != 1 || !enemy.PlayerDiscovered {
			t.Fatalf("summoned enemy = %+v; want a discovered マムル next to the summoner", enemy)
		}
	}

	if got := g.summonMonstersAround(2, 2, 1, 3, []int{0, 2}); got != 0 {
		t.Fatalf("summoned = %d with every candidate excluded, want 0", got)
	}

	for len(g.state.Enemies) < MaxEnemiesOnFloor-1 {
		g.state.Enemies = append(g.state.Enemies, CreateEnemyByID(2, 0, 0))
	}
	if got := g.summonMonstersAround(6, 6, 2, 3, []int{0}); got != 1 || len(g.state.Enemies) != MaxEnemiesOnFloor {
		t.Fatalf("summoned = %d, enemies = %d; want to stop at the floor limit", got, len(g.state.Enemies))
	}
}

// --- 固定ダメージの特殊攻撃 ---

func TestEnemyFixedAttackIgnoresDefenseAndUsesShields(t *testing.T) {
	setup := func() (*Game, *int) {
		g := newFoundationTestGame()
		g.randHooks.shieldDefense = func(int) int { return 99 }
		g.state.Player.DefensePower = 50
		addTestEnemy(g, 38, 5, 4)
		hits := 0
		return g, &hits
	}

	g, hits := setup()
	g.enqueueEnemyFixedAttack(0, 10, EnemyDamageFire, EnemyAttackRanged, "火を吹いた。", "炎", func(*Game) { *hits++ })
	runQueuedActions(g)
	if g.state.Player.Health != 30 || *hits != 1 {
		t.Fatalf("health = %d, hits = %d; want 30 and 1 (defense does not reduce fixed damage)", g.state.Player.Health, *hits)
	}

	g, hits = setup()
	equip(g, testArmor(AbilityFireResistance))
	g.state.Player.DefensePower = 50
	g.enqueueEnemyFixedAttack(0, 10, EnemyDamageFire, EnemyAttackRanged, "火を吹いた。", "炎", func(*Game) { *hits++ })
	runQueuedActions(g)
	if g.state.Player.Health != 35 {
		t.Fatalf("health with fire resistance = %d, want 35", g.state.Player.Health)
	}

	g, hits = setup()
	withTestAbilityHooks(t, "test_evade_all", AbilityHooks{ModifyIncomingDamage: func(_ *Game, incoming *IncomingDamage, _ int) {
		incoming.Evaded = true
	}})
	equip(g, testRing("test_evade_all"))
	g.enqueueEnemyFixedAttack(0, 10, EnemyDamageNormal, EnemyAttackMelee, "砂をかけた。", "砂", func(*Game) { *hits++ })
	runQueuedActions(g)
	if g.state.Player.Health != 40 || *hits != 0 {
		t.Fatalf("evaded: health = %d, hits = %d; want no damage and no follow-up effect", g.state.Player.Health, *hits)
	}
}

// --- 撃破時のフック ---

func TestOnEnemyDefeatedRunsOnEveryDeathPath(t *testing.T) {
	var defeated []Enemy
	withTestMonster(t, 991, 0, func(d *MonsterDefinition) {
		d.OnDefeated = func(_ *Game, enemy Enemy) { defeated = append(defeated, enemy) }
	})

	// プレイヤーの攻撃・アイテムなど（applyDamageToEnemy）
	g := newFoundationTestGame()
	addTestEnemy(g, 991, 2, 2)
	g.applyDamageToEnemy(0, 999)
	if len(defeated) != 1 || defeated[0].X != 2 || len(g.state.Enemies) != 0 {
		t.Fatalf("defeated = %+v, enemies = %d", defeated, len(g.state.Enemies))
	}

	// 封印中は呼ばれない
	addTestEnemy(g, 991, 2, 2)
	g.state.Enemies[0].StatusAilments.Seal = true
	g.applyDamageToEnemy(0, 999)
	if len(defeated) != 1 {
		t.Fatal("sealed enemy ran its defeat hook")
	}

	// 敵に倒された
	addTestEnemy(g, 0, 2, 2)
	addTestEnemy(g, 991, 3, 2)
	g.defeatEnemyByEnemy(0, 1)
	if len(defeated) != 2 {
		t.Fatal("defeat by another enemy did not run the hook")
	}

	// 爆発に巻き込まれた
	g = newFoundationTestGame()
	addTestEnemy(g, 6, 1, 1)
	target := addTestEnemy(g, 991, 4, 4)
	g.state.Enemies[target].Health = 1
	g.enqueueExplosionCollateral(6, 1, 1, 4, 4, RangedAttackDefinition{Kind: RangedAttackExplosion, AttackPower: 100, BlastRadius: 1})
	runQueuedActions(g)
	if len(defeated) != 3 {
		t.Fatal("explosion kill did not run the hook")
	}
}

// --- 遅れて現れる敵 ---

func TestPendingEnemySpawnAppearsAfterTurns(t *testing.T) {
	g := newFoundationTestGame()
	g.scheduleEnemySpawn(11, 2, 2, 3)
	g.processPendingEnemySpawns()
	g.processPendingEnemySpawns()
	if len(g.state.Enemies) != 0 {
		t.Fatal("enemy appeared too early")
	}
	g.processPendingEnemySpawns()
	if len(g.state.Enemies) != 1 || g.state.Enemies[0].ID != 11 || g.state.Enemies[0].X != 2 || !g.state.Enemies[0].PlayerDiscovered {
		t.Fatalf("enemies = %+v, want a discovered ユウレイエビ at (2,2)", g.state.Enemies)
	}
	if messages := runQueuedActions(g); !strings.Contains(messages, "ユウレイエビが現れた") {
		t.Fatalf("messages = %q", messages)
	}
	if len(g.pendingEnemySpawns) != 0 {
		t.Fatal("spawn reservation was not consumed")
	}
}

func TestPendingEnemySpawnFindsNearbyTileAndWaitsWhenFull(t *testing.T) {
	g := newFoundationTestGame()
	g.scheduleEnemySpawn(11, 4, 4, 1) // プレイヤーのいるマス
	for len(g.state.Enemies) < MaxEnemiesOnFloor {
		g.state.Enemies = append(g.state.Enemies, CreateEnemyByID(2, 0, 0))
	}
	g.processPendingEnemySpawns()
	if len(g.pendingEnemySpawns) != 1 {
		t.Fatal("reservation was dropped while the floor was full")
	}
	g.state.Enemies = nil
	g.processPendingEnemySpawns()
	if len(g.state.Enemies) != 1 || g.state.Enemies[0].X != 3 || g.state.Enemies[0].Y != 3 {
		t.Fatalf("enemies = %+v, want the nearest free tile (3,3)", g.state.Enemies)
	}

	g.scheduleEnemySpawn(11, 2, 2, 1)
	g.genocidedMonsterIDs = map[int]bool{11: true}
	g.processPendingEnemySpawns()
	if len(g.state.Enemies) != 1 || len(g.pendingEnemySpawns) != 0 {
		t.Fatal("genocided monster appeared or stayed reserved")
	}
}

func TestPendingEnemySpawnsSaveRoundTripAndFloorChange(t *testing.T) {
	g, _ := newHeadlessGame(77, DifficultyAdvanced)
	g.scheduleEnemySpawn(29, 3, 4, 2)
	data, err := json.Marshal(g.buildSaveData())
	if err != nil {
		t.Fatal(err)
	}
	save, err := decodeSaveData(data)
	if err != nil {
		t.Fatal(err)
	}
	restored := &Game{}
	if err := restored.applySaveData(save); err != nil {
		t.Fatal(err)
	}
	if len(restored.pendingEnemySpawns) != 1 || restored.pendingEnemySpawns[0] != (PendingEnemySpawn{MonsterID: 29, X: 3, Y: 4, Turns: 2}) {
		t.Fatalf("restored reservations = %+v", restored.pendingEnemySpawns)
	}

	g.fadingOut, g.fadeAlpha, g.frameCounter = true, 1.0, 0
	g.handleFadingOut()
	if len(g.pendingEnemySpawns) != 0 {
		t.Fatal("reservations survived the floor change")
	}
}

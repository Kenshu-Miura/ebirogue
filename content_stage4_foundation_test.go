//go:build !test

package main

// 段4の共通基盤の結合テスト: 敵の汎用の状態とセーブ、敵の範囲爆発、壁を掘る・近づくまで追わない、
// 別の敵に化ける敵、打ち返し、すれちがいの指輪（203）。
// 段4の敵はまだ無いので、仮の敵 ID（980〜）の定義を既存の敵から写して確かめる。

import (
	"encoding/json"
	"strings"
	"testing"
)

const (
	testMimicID      = 980
	testMimicUpperID = 981
	testBomberID     = 982
)

// withTestMimic は別の敵に化ける仮の敵（980、上位種 981）を登録する。
func withTestMimic(t *testing.T) {
	t.Helper()
	withTestMonster(t, testMimicID, 0, func(d *MonsterDefinition) {
		d.Type, d.Name, d.Char, d.Disguise = "TestMimic", "試験ダコ", 'm', EnemyDisguiseEnemy
		d.Health, d.MaxHealth = 30, 30
	})
	withTestMonster(t, testMimicUpperID, testMimicID, func(d *MonsterDefinition) {
		d.Type, d.Name = "TestMimicUpper", "試験大ダコ"
	})
	MonsterLevelUpTable[testMimicID] = testMimicUpperID
	t.Cleanup(func() { delete(MonsterLevelUpTable, testMimicID) })
}

// withReturnPercent はテスト中だけ敵 id の打ち返しの確率を変える。
func withReturnPercent(t *testing.T, id, percent int) {
	t.Helper()
	previous := MonsterDefinitions[id]
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.ProjectileReturnPercent = percent }, id)
	t.Cleanup(func() { MonsterDefinitions[id] = previous })
}

// --- 敵の汎用の状態とセーブ ---

func TestEnemyGenericStateSaveRoundTrip(t *testing.T) {
	enemy := CreateEnemyByID(0, 2, 3)
	enemy.Counter, enemy.Timer, enemy.Flag, enemy.Alerted, enemy.DisguiseID = 3, 5, true, true, 7

	restored := savedToEnemy(enemyToSaved(&enemy))

	if restored.Counter != 3 || restored.Timer != 5 || !restored.Flag || !restored.Alerted || restored.DisguiseID != 7 {
		t.Fatalf("restored = %+v", restored)
	}
}

func TestLegacySavedEnemyLoadsWithoutGenericState(t *testing.T) {
	legacy := `{"ID":0,"X":1,"Y":2,"Health":5,"MaxHealth":20,"AttackPower":4,"DefensePower":2,"ExperiencePoints":5,"Direction":0,"PlayerDiscovered":true,"StatusAilments":{},"HeldItem":null,"Fleeing":false,"Revealed":true}`
	var saved SavedEnemy
	if err := json.Unmarshal([]byte(legacy), &saved); err != nil {
		t.Fatal(err)
	}
	enemy := savedToEnemy(saved)
	if enemy.Name != "エビ" || enemy.Health != 5 || enemy.Counter != 0 || enemy.Timer != 0 || enemy.Flag || enemy.Alerted || enemy.DisguiseID != 0 {
		t.Fatalf("enemy = %+v", enemy)
	}
	// 0 の値はセーブに書かない（古い形式のまま）
	data, _ := json.Marshal(enemyToSaved(&enemy))
	for _, key := range []string{"Counter", "Timer", "Flag", "Alerted", "DisguiseID"} {
		if strings.Contains(string(data), key) {
			t.Fatalf("saved = %s, should omit %s", data, key)
		}
	}
}

func TestEnemyGenericStateSurvivesLevelUp(t *testing.T) {
	g := newFoundationTestGame()
	index := addTestEnemy(g, 0, 6, 4)
	g.state.Enemies[index].Counter, g.state.Enemies[index].Flag = 2, true

	if !g.levelUpEnemy(index) {
		t.Fatal("level up failed")
	}
	if e := g.state.Enemies[index]; e.Counter != 2 || !e.Flag {
		t.Fatalf("enemy = %+v", e)
	}
}

// --- 敵の範囲爆発 ---

func newExplosionTestGame() (*Game, int) {
	g := newFoundationTestGame()
	g.state.Map = makeTestFloorMap(12, 9)
	bomber := addTestEnemy(g, 0, 5, 4)
	return g, bomber
}

func TestEnemyExplosionDamagesPlayerEnemiesAndItems(t *testing.T) {
	g, bomber := newExplosionTestGame()
	near := addTestEnemy(g, 1, 6, 5)
	g.state.Enemies[near].Health, g.state.Enemies[near].StatusAilments.Sleep = 30, 5
	far := addTestEnemy(g, 1, 8, 4)
	farHealth := g.state.Enemies[far].Health
	nearItem := buildItemFromTemplate(1, 6, 3)
	farItem := buildItemFromTemplate(1, 8, 5)
	sanctuary := buildItemFromTemplate(55, 4, 5).(*Card)
	sanctuary.Stuck = true
	g.state.Items = []Item{nearItem, farItem, sanctuary}
	exp := g.state.Player.ExperiencePoints

	g.enqueueEnemyExplosion(EnemyExplosion{BomberIndex: bomber, CenterX: 5, CenterY: 4, Radius: 1, Damage: 10})
	messages := runQueuedActions(g)

	if g.state.Player.Health != 30 {
		t.Fatalf("player HP = %d, want 30 (fixed 10 damage)", g.state.Player.Health)
	}
	if g.state.Enemies[near].Health != 20 || g.state.Enemies[near].StatusAilments.Sleep != 0 {
		t.Fatalf("near enemy = %+v, want 10 damage and awake", g.state.Enemies[near])
	}
	if g.state.Enemies[far].Health != farHealth || g.state.Enemies[bomber].Health != g.state.Enemies[bomber].MaxHealth {
		t.Fatal("the far enemy and the bomber should not be damaged")
	}
	if len(g.state.Items) != 2 || g.state.Items[0] != farItem || g.state.Items[1] != sanctuary {
		t.Fatalf("items = %v, want only the near item destroyed", g.state.Items)
	}
	if g.state.Player.ExperiencePoints != exp {
		t.Fatal("an enemy explosion should not give the player experience")
	}
	if g.lastDamageSource.Label() != "エビの爆発" {
		t.Fatalf("death cause = %q", g.lastDamageSource.Label())
	}
	if !strings.Contains(messages, "エビが爆発した") || g.rangedAttackEffect.Kind != RangedAttackExplosion {
		t.Fatalf("messages = %q, effect = %+v", messages, g.rangedAttackEffect)
	}
}

func TestEnemyExplosionHalvedByBlastShield(t *testing.T) {
	g, bomber := newExplosionTestGame()
	equip(g, testArmor(AbilityExplosionResistance))

	g.enqueueEnemyExplosion(EnemyExplosion{BomberIndex: bomber, CenterX: 5, CenterY: 4, Radius: 1, Damage: 10})
	runQueuedActions(g)

	if want := 40 - reducedShieldDamage(10); g.state.Player.Health != want {
		t.Fatalf("player HP = %d, want %d", g.state.Player.Health, want)
	}
}

func TestEnemyExplosionOutOfRangeAndEvasionShield(t *testing.T) {
	g, bomber := newExplosionTestGame()
	equip(g, testArmor(AbilityEvasion, AbilityReflection))
	g.randHooks.shieldDefense = func(int) int { return 0 } // 回避の判定は必ず成功する側

	g.enqueueEnemyExplosion(EnemyExplosion{BomberIndex: bomber, CenterX: 5, CenterY: 4, Radius: 1, Damage: 10})
	runQueuedActions(g)
	if g.state.Player.Health != 30 {
		t.Fatalf("player HP = %d, the explosion cannot be evaded or reflected", g.state.Player.Health)
	}

	g.state.Player.X = 2
	g.enqueueEnemyExplosion(EnemyExplosion{BomberIndex: bomber, CenterX: 5, CenterY: 4, Radius: 1, Damage: 10})
	runQueuedActions(g)
	if g.state.Player.Health != 30 {
		t.Fatalf("player HP = %d, out of range should not be hit", g.state.Player.Health)
	}
}

func TestEnemyExplosionDefeatsEnemiesAndLevelsUpBomber(t *testing.T) {
	g, bomber := newExplosionTestGame()
	victim := addTestEnemy(g, 1, 6, 4)
	g.state.Enemies[victim].Health = 5
	defeated := 0
	previous := MonsterDefinitions[1]
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.OnDefeated = func(*Game, Enemy) { defeated++ } }, 1)
	t.Cleanup(func() { MonsterDefinitions[1] = previous })
	exp := g.state.Player.ExperiencePoints

	g.enqueueEnemyExplosion(EnemyExplosion{BomberIndex: bomber, CenterX: 5, CenterY: 4, Radius: 1, Damage: 10})
	messages := runQueuedActions(g)

	if len(g.state.Enemies) != 1 || defeated != 1 {
		t.Fatalf("enemies = %d, OnDefeated = %d", len(g.state.Enemies), defeated)
	}
	if g.state.Enemies[0].ID != MonsterLevelUpTable[0] {
		t.Fatalf("bomber ID = %d, want leveled up to %d", g.state.Enemies[0].ID, MonsterLevelUpTable[0])
	}
	if g.state.Player.ExperiencePoints != exp || !strings.Contains(messages, "爆発に巻き込まれて倒れた") {
		t.Fatalf("exp = %d, messages = %q", g.state.Player.ExperiencePoints, messages)
	}
}

func TestEnemyExplosionSelfDestruct(t *testing.T) {
	g, _ := newExplosionTestGame()
	defeated := 0
	withTestMonster(t, testBomberID, 0, func(d *MonsterDefinition) {
		d.Name = "試験フグ"
		d.OnDefeated = func(*Game, Enemy) { defeated++ }
	})
	g.state.Enemies = nil
	bomber := addTestEnemy(g, testBomberID, 5, 4)
	other := addTestEnemy(g, 0, 6, 4)
	g.state.Enemies[other].Health = 5
	exp := g.state.Player.ExperiencePoints

	g.enqueueEnemyExplosion(EnemyExplosion{BomberIndex: bomber, CenterX: 5, CenterY: 4, Radius: 1, Damage: 10, SelfDestruct: true, Message: "試験フグが自爆した！"})
	messages := runQueuedActions(g)

	if len(g.state.Enemies) != 0 || defeated != 1 {
		t.Fatalf("enemies = %v, OnDefeated = %d", g.state.Enemies, defeated)
	}
	if g.state.Player.ExperiencePoints != exp || !strings.Contains(messages, "試験フグが自爆した") {
		t.Fatalf("exp = %d, messages = %q", g.state.Player.ExperiencePoints, messages)
	}
	if g.lastDamageSource.Label() != "試験フグの爆発" {
		t.Fatalf("death cause = %q", g.lastDamageSource.Label())
	}
}

func TestEnemyExplosionBossCapAndChain(t *testing.T) {
	g, bomber := newExplosionTestGame()
	boss := addTestEnemy(g, 42, 6, 4)
	g.state.Enemies[boss].Health = 200
	survivor := addTestEnemy(g, 1, 4, 3)
	g.state.Enemies[survivor].Health = 99
	damaged := 0
	withMonsterOnDamaged(t, 1, func(g *Game, i, damage int) { damaged += damage })

	g.enqueueEnemyExplosion(EnemyExplosion{BomberIndex: bomber, CenterX: 5, CenterY: 4, Radius: 1, Damage: 80})
	runQueuedActions(g)

	if g.state.Enemies[boss].Health != 200-enemyExplosionBossDamageCap {
		t.Fatalf("boss HP = %d, want capped damage %d", g.state.Enemies[boss].Health, enemyExplosionBossDamageCap)
	}
	if damaged != 80 {
		t.Fatalf("OnDamaged received %d, want 80 (explosions can chain)", damaged)
	}
}

func TestEnemyExplosionAfterBomberIsGone(t *testing.T) {
	g, bomber := newExplosionTestGame()
	g.enqueueEnemyExplosion(EnemyExplosion{BomberIndex: bomber, CenterX: 5, CenterY: 4, Radius: 1, Damage: 10})
	g.state.Enemies = nil // 爆発の前に倒れていても爆発は起こる

	runQueuedActions(g)

	if g.state.Player.Health != 30 {
		t.Fatalf("player HP = %d", g.state.Player.Health)
	}
}

// --- 壁を掘る・近づくまで追わない ---

func newDigTestGame() *Game {
	g := newFoundationTestGame()
	g.state.Map = makeTestFloorMap(12, 9)
	for y := 1; y < 8; y++ {
		g.state.Map[y][3] = Tile{Type: "wall", Blocked: true, BlockSight: true}
	}
	return g
}

func TestDigWallAtUsesPickaxeRules(t *testing.T) {
	g := newDigTestGame()
	g.state.Player.X, g.state.Player.Y = 8, 4

	if !g.digWallAt(3, 2) {
		t.Fatal("interior wall should be dug")
	}
	if tile := g.state.Map[2][3]; tile.Type != "corridor" || tile.Blocked || tile.BlockSight || tile.Visited {
		t.Fatalf("tile = %+v, want an unseen corridor", tile)
	}
	if g.digWallAt(0, 4) || g.digWallAt(3, 0) || g.digWallAt(5, 4) || g.digWallAt(-1, 4) {
		t.Fatal("outer walls, floors and out of map should not be dug")
	}

	g.state.Player.X, g.state.Player.Y = 4, 4
	if !g.digWallAt(3, 5) || !g.state.Map[5][3].Visited {
		t.Fatal("a wall dug next to the player should become visited")
	}
}

func TestStepEnemyDigging(t *testing.T) {
	g := newDigTestGame()
	g.state.Player.X = 10
	i := addTestEnemy(g, 0, 2, 4)

	if !g.stepEnemyDigging(i, 1, 0) {
		t.Fatal("enemy should dig and move")
	}
	if e := g.state.Enemies[i]; e.X != 3 || e.Y != 4 || g.state.Map[4][3].Blocked {
		t.Fatalf("enemy = (%d,%d), tile blocked = %v", e.X, e.Y, g.state.Map[4][3].Blocked)
	}

	j := addTestEnemy(g, 0, 2, 2)
	if g.stepEnemyDigging(j, 1, 1) || !g.state.Map[3][3].Blocked {
		t.Fatal("diagonal walls should not be dug")
	}
	k := addTestEnemy(g, 0, 1, 6)
	if g.stepEnemyDigging(k, -1, 0) || !g.state.Map[6][0].Blocked {
		t.Fatal("the outer wall should not be dug")
	}
}

func TestStepEnemyTowardDigging(t *testing.T) {
	g := newDigTestGame()
	g.state.Player.X, g.state.Player.Y = 6, 4
	i := addTestEnemy(g, 0, 2, 4)

	if !g.stepEnemyTowardDigging(i, 6, 4) {
		t.Fatal("enemy should dig toward the player")
	}
	if e := g.state.Enemies[i]; e.X != 3 || e.Y != 4 {
		t.Fatalf("enemy = (%d,%d), want (3,4)", e.X, e.Y)
	}
	if !g.stepEnemyTowardDigging(i, 6, 4) || g.state.Enemies[i].X != 4 {
		t.Fatalf("enemy should keep walking on the floor, at %d", g.state.Enemies[i].X)
	}
}

func TestEnemyHoldsUntilNear(t *testing.T) {
	g := newFoundationTestGame()
	g.state.Map = makeTestFloorMap(12, 9)
	g.state.Player.X = 1
	i := addTestEnemy(g, 0, 9, 4)

	if !g.enemyHoldsUntilNear(i, 2) || g.state.Enemies[i].Alerted {
		t.Fatal("far enemy should hold")
	}
	g.state.Player.X = 7
	if g.enemyHoldsUntilNear(i, 2) || !g.state.Enemies[i].Alerted {
		t.Fatal("enemy should start chasing when the player comes near")
	}
	g.state.Player.X = 1
	if g.enemyHoldsUntilNear(i, 2) {
		t.Fatal("once alerted, the enemy keeps chasing")
	}
	if !savedToEnemy(enemyToSaved(&g.state.Enemies[i])).Alerted {
		t.Fatal("alerted state should be saved")
	}

	j := addTestEnemy(g, 0, 9, 6)
	g.state.Enemies[j].Health--
	if g.enemyHoldsUntilNear(j, 2) {
		t.Fatal("a damaged enemy should start chasing")
	}
}

// --- 別の敵に化ける敵 ---

func TestMimicShowsAnotherEnemy(t *testing.T) {
	withTestMimic(t)
	g := newFoundationTestGame()
	i := addTestEnemy(g, testMimicID, 6, 4)
	if g.state.Enemies[i].Name != "試験ダコ" || !g.state.Enemies[i].Revealed {
		t.Fatalf("created mimic = %+v, should be its true form until disguised", g.state.Enemies[i])
	}

	disguiseEnemyAs(&g.state.Enemies[i], 1)
	e := g.state.Enemies[i]
	look := MonsterDefinitions[1]
	if e.Name != look.Name || e.Type != look.Type || e.Char != look.Char || enemyAppearance(e).ID != 1 {
		t.Fatalf("disguised mimic = %+v, want the look of %s", e, look.Name)
	}
	if !enemyMasquerading(e) || isEnemyDisguised(e) {
		t.Fatal("a mimic disguised as an enemy should act as a visible enemy, not a hidden object")
	}
	if e.ID != testMimicID || e.Health != 30 {
		t.Fatalf("mimic keeps its own stats: %+v", e)
	}

	// 普通の敵として動く（化け貝のように止まらない）
	g.state.Enemies[i].X, g.state.Enemies[i].PlayerDiscovered = 7, true
	g.actEnemy(i)
	if g.state.Enemies[i].X != 6 {
		t.Fatalf("mimic x = %d, should approach the player", g.state.Enemies[i].X)
	}
	// 気配察知の指輪に化けた姿のまま映る
	g.state.Enemies[i].X, g.state.Enemies[i].PlayerDiscovered = 1, false
	g.state.Player.X = 7
	equip(g, testRing(AbilitySenseEnemies))
	g.updateEnemyVisibility()
	if !g.state.Enemies[i].ShowOnMiniMap {
		t.Fatal("a masquerading mimic should be sensed like any enemy")
	}
}

func TestMimicRevealsWhenDamaged(t *testing.T) {
	withTestMimic(t)
	g := newFoundationTestGame()
	i := addTestEnemy(g, testMimicID, 5, 4)
	disguiseEnemyAs(&g.state.Enemies[i], 1)
	disguised := g.state.Enemies[i].Name

	g.applyDamageToEnemy(i, 3)
	messages := runQueuedActions(g)

	e := g.state.Enemies[i]
	if !e.Revealed || e.Name != "試験ダコ" || e.Type != "TestMimic" || e.Health != 27 {
		t.Fatalf("mimic = %+v", e)
	}
	if !strings.Contains(messages, disguised+"の正体は試験ダコだった") {
		t.Fatalf("messages = %q", messages)
	}
}

func TestMimicRevealsOnMeleeAndThrow(t *testing.T) {
	withTestMimic(t)
	g := newProjectileTestGame()
	i := addTestEnemy(g, testMimicID, 5, 4)
	g.state.Enemies[i].Health = 99
	disguiseEnemyAs(&g.state.Enemies[i], 1)

	g.CheckForEnemies(1, 0)
	runQueuedActions(g)
	if !g.state.Enemies[i].Revealed {
		t.Fatal("melee attack should reveal the mimic")
	}

	disguiseEnemyAs(&g.state.Enemies[i], 1)
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{food}
	g.throwWithCallbacks(food, 10)
	finishThrow(t, g)
	if !g.state.Enemies[i].Revealed {
		t.Fatal("a thrown item should reveal the mimic")
	}
}

func TestMimicSaveLevelUpSealAndSplit(t *testing.T) {
	withTestMimic(t)
	g := newFoundationTestGame()
	i := addTestEnemy(g, testMimicID, 5, 4)
	disguiseEnemyAs(&g.state.Enemies[i], 1)
	look := MonsterDefinitions[1].Name

	restored := savedToEnemy(enemyToSaved(&g.state.Enemies[i]))
	if !enemyMasquerading(restored) || restored.Name != look || restored.ID != testMimicID {
		t.Fatalf("restored = %+v", restored)
	}

	if !g.levelUpEnemy(i) {
		t.Fatal("level up failed")
	}
	if e := g.state.Enemies[i]; e.ID != testMimicUpperID || e.Name != look || !enemyMasquerading(e) {
		t.Fatalf("leveled mimic = %+v, should keep the disguise", e)
	}

	clone := g.splitEnemy(i)
	if clone < 0 || g.state.Enemies[clone].Name != look || !enemyMasquerading(g.state.Enemies[clone]) {
		t.Fatalf("clone = %d %+v", clone, g.state.Enemies)
	}

	g.state.Enemies[i].StatusAilments.Seal = true
	if enemyMasquerading(g.state.Enemies[i]) || enemyAppearance(g.state.Enemies[i]).Name != "試験大ダコ" {
		t.Fatal("a sealed mimic shows its true form")
	}
}

func TestMimicDeathCauseUsesTrueName(t *testing.T) {
	withTestMimic(t)
	g := newFoundationTestGame()
	i := addTestEnemy(g, testMimicID, 5, 4)
	disguiseEnemyAs(&g.state.Enemies[i], 1)
	e := g.state.Enemies[i]

	g.damagePlayer(1, enemyAttackSource(e.ID, e.Name))
	if g.lastDamageSource.Label() != "試験ダコ" {
		t.Fatalf("death cause = %q, want the true name", g.lastDamageSource.Label())
	}
}

func TestPrepareSpawnedEnemyPicksDisguiseFromFloor(t *testing.T) {
	withTestMimic(t)
	const floor = 5
	allowed := map[int]bool{}
	total := 0
	for _, entry := range FloorSpawnTables[floor] {
		d := MonsterDefinitions[entry.MonsterID]
		if !d.IsBoss && d.Disguise == EnemyDisguiseNone {
			allowed[entry.MonsterID] = true
			total += entry.Weight
		}
	}
	for roll := 0; roll < total; roll++ {
		enemy := CreateEnemyByID(testMimicID, 1, 1)
		prepareSpawnedEnemy(&enemy, floor, func(int) int { return roll })
		if !allowed[enemy.DisguiseID] || !enemyMasquerading(enemy) {
			t.Fatalf("roll %d: disguise = %d (%+v)", roll, enemy.DisguiseID, enemy)
		}
	}

	normal := CreateEnemyByID(0, 1, 1)
	prepareSpawnedEnemy(&normal, floor, func(int) int { t.Fatal("normal enemies should not use the random source"); return 0 })
	clam := CreateEnemyByID(16, 1, 1)
	prepareSpawnedEnemy(&clam, floor, func(int) int { t.Fatal("hidden mimics should not use the random source"); return 0 })

	if id := pickDisguiseID(999, testMimicID, func(int) int { return 0 }); id != -1 {
		t.Fatalf("no table should give -1, got %d", id)
	}
}

// --- 打ち返し ---

func TestReturnedThrowHitsPlayer(t *testing.T) {
	withReturnPercent(t, 0, 100)
	g := newProjectileTestGame()
	index := addTestEnemy(g, 0, 6, 4)
	enemyHealth := g.state.Enemies[index].Health
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{food}
	want := 40 - rollThrownItemDamage(g.damageIntn())

	g.throwWithCallbacks(food, 10)
	messages := finishThrow(t, g)

	if !strings.Contains(messages, "エビはウインナーを打ち返した") || !strings.Contains(messages, "打ち返されたウインナーが飛んできた") {
		t.Fatalf("messages = %q", messages)
	}
	if g.state.Player.Health != want || g.state.Enemies[index].Health != enemyHealth {
		t.Fatalf("player HP = %d (want %d), enemy HP = %d", g.state.Player.Health, want, g.state.Enemies[index].Health)
	}
	if len(g.state.Player.Inventory) != 0 || len(g.state.Items) != 0 {
		t.Fatalf("inventory = %v, floor = %v, the item is used up", g.state.Player.Inventory, g.state.Items)
	}
	if g.lastDamageSource.Label() != "エビの打ち返し" {
		t.Fatalf("death cause = %q", g.lastDamageSource.Label())
	}
}

func TestReturnedArrowUsesShotFormulaWithoutCritical(t *testing.T) {
	withReturnPercent(t, 0, 100)
	g := newProjectileTestGame()
	addTestEnemy(g, 0, 6, 4)
	arrow := buildItemFromTemplate(6, 0, 0).(*Arrow)
	arrow.ShotCount = 3
	equip(g, arrow)
	p := g.state.Player
	want := 40 - rollEnemyAttackDamage(playerAttackTotal(p.AttackPower, p.Power, p.Level)+arrow.AttackPower, p.DefensePower, g.damageIntn())

	g.shootArrow(arrow, 10)
	finishThrow(t, g)

	if g.state.Player.Health != want {
		t.Fatalf("player HP = %d, want %d", g.state.Player.Health, want)
	}
}

func TestSureHitDoesNotPreventReturn(t *testing.T) {
	withReturnPercent(t, 0, 100)
	g := newProjectileTestGame()
	equip(g, testRing(AbilitySureThrow))
	addTestEnemy(g, 0, 6, 4)
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = append(g.state.Player.Inventory, food)

	g.throwWithCallbacks(food, 10)
	finishThrow(t, g)

	if g.state.Player.Health == 40 {
		t.Fatal("returning happens before the hit check, so the sure-hit ring cannot prevent it")
	}
}

func TestReturnChanceAndStatus(t *testing.T) {
	withReturnPercent(t, 0, 50)
	g := newProjectileTestGame()
	index := addTestEnemy(g, 0, 6, 4)
	g.state.Enemies[index].Health = 99
	g.randHooks.projectile = func(int) int { return 50 } // 50 以上は打ち返さない（命中判定は 85% なので当たる）
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{food}
	g.throwWithCallbacks(food, 10)
	finishThrow(t, g)
	if g.state.Player.Health != 40 || g.state.Enemies[index].Health == 99 {
		t.Fatalf("roll 50 should hit the enemy: player HP = %d, enemy HP = %d", g.state.Player.Health, g.state.Enemies[index].Health)
	}

	g.randHooks.projectile = func(int) int { return 49 }
	food = buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{food}
	g.throwWithCallbacks(food, 10)
	finishThrow(t, g)
	if g.state.Player.Health == 40 {
		t.Fatal("roll 49 < 50% should return")
	}

	for _, status := range []func(*Enemy){
		func(e *Enemy) { e.StatusAilments.Seal = true },
		func(e *Enemy) { e.StatusAilments.Sleep = 3 },
		func(e *Enemy) { e.StatusAilments.Paralysis = true },
	} {
		e := CreateEnemyByID(0, 0, 0)
		status(&e)
		if enemyProjectileReturnPercent(e) != 0 {
			t.Fatalf("enemy %+v should not return", e.StatusAilments)
		}
	}
	if enemyProjectileReturnPercent(CreateEnemyByID(1, 0, 0)) != 0 {
		t.Fatal("enemies without the ability should not return")
	}

	returned := Projectile{Returned: true}
	if g.tryReturnProjectile(&returned, index) {
		t.Fatal("a returned projectile is never returned again")
	}
}

func TestReturnedThrowAndPlayerShields(t *testing.T) {
	withReturnPercent(t, 0, 100)

	// 反射の盾は打ち返した敵へダメージを返す
	g := newProjectileTestGame()
	index := addTestEnemy(g, 0, 6, 4)
	g.state.Enemies[index].Health = 99
	equip(g, testArmor(AbilityReflection))
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = append(g.state.Player.Inventory, food)
	damage := rollThrownItemDamage(g.damageIntn())
	g.throwWithCallbacks(food, 10)
	messages := finishThrow(t, g)
	if g.state.Player.Health != 40 || g.state.Enemies[index].Health != 99-damage || !strings.Contains(messages, "反射") {
		t.Fatalf("player HP = %d, enemy HP = %d, messages = %q", g.state.Player.Health, g.state.Enemies[index].Health, messages)
	}

	// 見切りの盾でかわせる
	g = newProjectileTestGame()
	addTestEnemy(g, 0, 6, 4)
	equip(g, testArmor(AbilityEvasion))
	g.randHooks.shieldDefense = func(int) int { return 0 }
	food = buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = append(g.state.Player.Inventory, food)
	g.throwWithCallbacks(food, 10)
	finishThrow(t, g)
	if g.state.Player.Health != 40 {
		t.Fatalf("player HP = %d, the evasion shield should dodge it", g.state.Player.Health)
	}
}

func TestReturnedBoltAndPotion(t *testing.T) {
	withReturnPercent(t, 0, 100)

	g := newProjectileTestGame()
	index := addTestEnemy(g, 0, 6, 4)
	cane := buildItemFromTemplate(16, 0, 0).(*Cane) // 封印の杖
	cane.Uses = 3
	g.state.Player.Inventory = []Item{cane}
	g.useCane(cane)
	messages := finishThrow(t, g)
	if g.state.Enemies[index].StatusAilments.Seal || g.state.Player.Health != 40 || !strings.Contains(messages, "何も起きなかった") {
		t.Fatalf("sealed = %v, HP = %d, messages = %q", g.state.Enemies[index].StatusAilments.Seal, g.state.Player.Health, messages)
	}

	g = newProjectileTestGame()
	addTestEnemy(g, 0, 6, 4)
	potion := buildItemFromTemplate(17, 0, 0) // 睡眠薬
	g.state.Player.Inventory = []Item{potion}
	g.throwWithCallbacks(potion, 10)
	finishThrow(t, g)
	if g.state.Player.StatusAilments.Sleep == 0 {
		t.Fatal("a returned sleeping potion should put the player to sleep")
	}
}

// --- すれちがいの指輪 ---

func newSwapTestGame(t *testing.T) *Game {
	t.Helper()
	g := newFoundationTestGame()
	ring := buildItemFromTemplate(203, 0, 0).(*Accessory)
	ring.Cursed = false
	equip(g, ring)
	return g
}

func TestSwapRingTemplate(t *testing.T) {
	template, ok := itemTemplates[203]
	if !ok || template.Name != "すれちがいの指輪" || template.ItemType != "Accessory" || !hasEquipmentAbility(template.Abilities, AbilitySwapPlaces) {
		t.Fatalf("template = %+v", template)
	}
	if _, ok := ringItemTemplates[203]; !ok {
		t.Fatal("203 should be in the ring table")
	}
}

func TestSwapRingSwapsWithEnemy(t *testing.T) {
	g := newSwapTestGame(t)
	i := addTestEnemy(g, 0, 5, 4)
	g.state.Enemies[i].StatusAilments.Sleep = 5
	g.state.MapTraps = []MapTrap{createMapTrapByID(2, 5, 4)} // 鈍足の罠
	g.state.MapTraps[0].FailRate = 0
	food := buildItemFromTemplate(1, 5, 4)
	g.state.Items = []Item{food}

	if !g.MovePlayer(1, 0) {
		t.Fatal("move should swap with the enemy")
	}
	runQueuedActions(g)

	if g.state.Player.X != 5 || g.state.Enemies[i].X != 4 || !g.isActioned {
		t.Fatalf("player x = %d, enemy x = %d, actioned = %v", g.state.Player.X, g.state.Enemies[i].X, g.isActioned)
	}
	if g.state.Enemies[i].StatusAilments.Sleep != 5 {
		t.Fatal("swapping should not wake the enemy")
	}
	if !g.state.MapTraps[0].Discovered {
		t.Fatal("the player should step on the trap at the new position")
	}
	if len(g.state.Player.Inventory) != 2 {
		t.Fatalf("inventory = %v, the item at the new position should be picked up", g.state.Player.Inventory)
	}
}

func TestSwapRingRefusals(t *testing.T) {
	cases := []struct {
		name  string
		setup func(g *Game) int
	}{
		{"boss", func(g *Game) int { return addTestEnemy(g, 42, 5, 4) }},
		{"enemy in wall", func(g *Game) int {
			g.state.Map[4][5] = Tile{Type: "wall", Blocked: true, BlockSight: true}
			return addTestEnemy(g, 0, 5, 4)
		}},
		{"player on sanctuary", func(g *Game) int {
			g.state.Items = []Item{buildItemFromTemplate(55, 4, 4)}
			return addTestEnemy(g, 0, 5, 4)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := newSwapTestGame(t)
			i := tc.setup(g)
			if g.MovePlayer(1, 0) || g.state.Player.X != 4 || g.state.Enemies[i].X != 5 || g.isActioned {
				t.Fatalf("should not swap: player x = %d, enemy x = %d", g.state.Player.X, g.state.Enemies[i].X)
			}
		})
	}

	g := newFoundationTestGame()
	i := addTestEnemy(g, 0, 5, 4)
	if g.MovePlayer(1, 0) || g.state.Enemies[i].X != 5 {
		t.Fatal("without the ring the player cannot pass")
	}
}

func TestSwapRingAndDisguisedEnemies(t *testing.T) {
	g := newSwapTestGame(t)
	clam := addTestEnemy(g, 16, 5, 4) // 道具に化けた化け貝
	if !g.MovePlayer(1, 0) || g.state.Player.X != 4 || !g.state.Enemies[clam].Revealed {
		t.Fatalf("a hidden mimic is revealed instead of swapped: player x = %d", g.state.Player.X)
	}

	withTestMimic(t)
	g = newSwapTestGame(t)
	mimic := addTestEnemy(g, testMimicID, 5, 4)
	disguiseEnemyAs(&g.state.Enemies[mimic], 1)
	if !g.MovePlayer(1, 0) || g.state.Player.X != 5 || !enemyMasquerading(g.state.Enemies[mimic]) {
		t.Fatalf("a masquerading mimic is swapped without revealing: player x = %d", g.state.Player.X)
	}
}

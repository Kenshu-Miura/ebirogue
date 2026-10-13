//go:build !test

package main

// 敵（特殊移動・上位種・所持品への特殊攻撃・遠距離攻撃）のコンテンツテスト。

import (
	"image/png"
	"os"
	"strings"
	"testing"
)

func TestSpecialMovementEnemyDefinitions(t *testing.T) {
	wantMovement := map[int]SpecialMovement{
		11: SpecialMovementWallPass,
		12: SpecialMovementDoubleSpeed,
		13: SpecialMovementWarp,
		14: SpecialMovementSwap,
		15: SpecialMovementCreateTrap,
	}
	for id, movement := range wantMovement {
		definition, ok := MonsterDefinitions[id]
		if !ok || definition.SpecialMovement != movement {
			t.Fatalf("monster %d = %#v, want movement %d", id, definition, movement)
		}
	}
	if MonsterDefinitions[16].Disguise != EnemyDisguiseItem || MonsterDefinitions[17].Disguise != EnemyDisguiseStairs {
		t.Fatal("item and stairs mimic definitions are missing")
	}
}

// テーブルのキーと各テンプレートの ID が一致し、全テンプレートを生成できることを確認する。
// ID は map のキーなので重複しない。欠番があってもランダム抽選は ID 一覧から行う。

// 各系統は 基本種 → 第2段階 → 第3段階 の3段階で、上位ほど能力値が高く、固有能力を引き継ぐ。
func TestEveryBaseMonsterHasUpperSpecies(t *testing.T) {
	if len(MonsterLevelUpTable) != 2*len(baseMonsterDefinitions) {
		t.Fatalf("level-up mappings = %d, want two for each of %d base monsters", len(MonsterLevelUpTable), len(baseMonsterDefinitions))
	}
	for baseID := range baseMonsterDefinitions {
		secondID, ok := MonsterLevelUpTable[baseID]
		if !ok {
			t.Fatalf("base monster %d has no upper species", baseID)
		}
		thirdID, ok := MonsterLevelUpTable[secondID]
		if !ok {
			t.Fatalf("second-stage monster %d has no third stage", secondID)
		}
		checkUpperSpecies(t, baseID, secondID)
		checkUpperSpecies(t, secondID, thirdID)
		if _, chained := MonsterLevelUpTable[thirdID]; chained {
			t.Fatalf("third-stage monster %d must not level up further", thirdID)
		}
	}
}

func checkUpperSpecies(t *testing.T, lowerID, upperID int) {
	t.Helper()
	base, ok := MonsterDefinitions[lowerID]
	if !ok {
		t.Fatalf("monster %d is missing", lowerID)
	}
	upper, ok := MonsterDefinitions[upperID]
	if !ok {
		t.Fatalf("upper species %d for monster %d is missing", upperID, lowerID)
	}
	if upper.Name == base.Name || upper.Type == base.Type {
		t.Fatalf("upper species %d did not change identity from %d", upperID, lowerID)
	}
	if upper.Health <= base.Health || upper.AttackPower <= base.AttackPower ||
		upper.DefensePower <= base.DefensePower || upper.ExperiencePoints <= base.ExperiencePoints {
		t.Fatalf("upper species %d stats = HP %d, attack %d, defense %d, exp %d; lower %d stats = HP %d, attack %d, defense %d, exp %d",
			upperID, upper.Health, upper.AttackPower, upper.DefensePower, upper.ExperiencePoints,
			lowerID, base.Health, base.AttackPower, base.DefensePower, base.ExperiencePoints)
	}
	if upper.Char != base.Char || (upper.SpecialAttack == nil) != (base.SpecialAttack == nil) ||
		upper.SpecialAttackProbability != base.SpecialAttackProbability || upper.SpecialMovement != base.SpecialMovement ||
		upper.Disguise != base.Disguise || upper.IsBoss {
		t.Fatalf("upper species %d did not inherit monster %d abilities", upperID, lowerID)
	}
	if len(upper.Traits) != len(base.Traits) {
		t.Fatalf("upper species %d did not inherit monster %d traits", upperID, lowerID)
	}
	for _, trait := range base.Traits {
		if !hasEnemyTrait(upper.Traits, trait) {
			t.Fatalf("upper species %d is missing inherited trait %q", upperID, trait)
		}
	}
	if upper.RangedAttack.Kind != base.RangedAttack.Kind || upper.RangedAttack.MinRange != base.RangedAttack.MinRange ||
		upper.RangedAttack.MaxRange != base.RangedAttack.MaxRange || upper.RangedAttack.BlastRadius != base.RangedAttack.BlastRadius {
		t.Fatalf("upper species %d did not inherit monster %d ranged behavior", upperID, lowerID)
	}
	if base.RangedAttack.Kind != RangedAttackNone && upper.RangedAttack.AttackPower <= base.RangedAttack.AttackPower {
		t.Fatalf("upper species %d ranged power = %d, want above %d", upperID, upper.RangedAttack.AttackPower, base.RangedAttack.AttackPower)
	}
}

func TestEveryUpperSpeciesHasImagePath(t *testing.T) {
	for _, upperID := range MonsterLevelUpTable {
		monsterType := MonsterDefinitions[upperID].Type
		path, ok := upperEnemyImagePaths[monsterType]
		if !ok || path == "" {
			t.Fatalf("upper species %d type %q has no image path", upperID, monsterType)
		}
	}
}

func TestWallPassingEnemyAndSeal(t *testing.T) {
	mapGrid := makeTestFloorMap(7, 5)
	mapGrid[2][2] = Tile{Type: "wall", Blocked: true, BlockSight: true}
	enemy := CreateEnemyByID(11, 1, 2)
	enemy.PlayerDiscovered = true
	g := &Game{state: GameState{
		Map:     mapGrid,
		Player:  Player{Entity: Entity{X: 4, Y: 2}},
		Enemies: []Enemy{enemy},
	}}

	g.moveEnemyTowardsPlayer(0)
	if g.state.Enemies[0].X != 2 || g.state.Enemies[0].Y != 2 {
		t.Fatalf("wall-passing enemy moved to (%d, %d), want wall tile (2, 2)", g.state.Enemies[0].X, g.state.Enemies[0].Y)
	}

	g.state.Enemies[0] = CreateEnemyByID(11, 1, 2)
	g.state.Enemies[0].StatusAilments.Seal = true
	g.moveEnemyTowardsPlayer(0)
	if g.state.Enemies[0].X == 2 && g.state.Enemies[0].Y == 2 {
		t.Fatal("sealed wall-passing enemy must not enter a wall")
	}
}

func TestNativeDoubleSpeedAndSeal(t *testing.T) {
	newGame := func(sealed bool) *Game {
		enemy := CreateEnemyByID(12, 2, 2)
		enemy.PlayerDiscovered = true
		enemy.StatusAilments.Seal = sealed
		return &Game{state: GameState{
			Map:     makeTestFloorMap(12, 6),
			Player:  Player{Entity: Entity{X: 9, Y: 2}, Health: 100},
			Enemies: []Enemy{enemy},
		}}
	}

	fast := newGame(false)
	fast.MoveEnemies()
	if fast.state.Enemies[0].X != 4 {
		t.Fatalf("double-speed enemy x = %d, want 4", fast.state.Enemies[0].X)
	}
	sealed := newGame(true)
	sealed.MoveEnemies()
	if sealed.state.Enemies[0].X != 3 {
		t.Fatalf("sealed double-speed enemy x = %d, want 3", sealed.state.Enemies[0].X)
	}
	slowed := newGame(false)
	slowed.state.Enemies[0].StatusAilments.Slow = 3
	slowed.MoveEnemies()
	if slowed.state.Enemies[0].X != 3 {
		t.Fatalf("slowed double-speed enemy x = %d, want 3", slowed.state.Enemies[0].X)
	}
}

func TestWarpEnemyAndSeal(t *testing.T) {
	enemy := CreateEnemyByID(13, 1, 1)
	g := &Game{state: GameState{
		Map:     makeTestFloorMap(10, 10),
		Player:  Player{Entity: Entity{X: 5, Y: 5}},
		Enemies: []Enemy{enemy},
	}}
	if !g.tryWarpEnemy(0, func(int) int { return 0 }) {
		t.Fatal("warp enemy should warp when its roll succeeds")
	}
	if g.state.Enemies[0].X == 1 && g.state.Enemies[0].Y == 1 {
		t.Fatal("warp enemy should change position")
	}

	g.state.Enemies[0] = CreateEnemyByID(13, 1, 1)
	g.state.Enemies[0].StatusAilments.Seal = true
	if g.tryWarpEnemy(0, func(int) int { return 0 }) {
		t.Fatal("sealed warp enemy should not warp")
	}
}

func TestSwapEnemyAndSeal(t *testing.T) {
	enemy := CreateEnemyByID(14, 2, 2)
	g := &Game{
		state: GameState{
			Map:     makeTestFloorMap(10, 10),
			Player:  Player{Entity: Entity{X: 5, Y: 2}},
			Enemies: []Enemy{enemy},
		},
		rooms: []Room{{ID: 1, X: 0, Y: 0, Width: 10, Height: 10}},
	}
	if !g.trySwapEnemyWithPlayer(0, func(int) int { return 0 }) {
		t.Fatal("swap enemy should exchange positions when its roll succeeds")
	}
	if g.state.Player.X != 2 || g.state.Enemies[0].X != 5 {
		t.Fatalf("swapped positions = player %d, enemy %d", g.state.Player.X, g.state.Enemies[0].X)
	}

	g.state.Player.X, g.state.Player.Y = 5, 2
	g.state.Enemies[0] = CreateEnemyByID(14, 2, 2)
	g.state.Enemies[0].StatusAilments.Seal = true
	if g.trySwapEnemyWithPlayer(0, func(int) int { return 0 }) {
		t.Fatal("sealed swap enemy should not exchange positions")
	}
}

func TestTrapCreatingEnemyLeavesHiddenTrapAndRespectsSeal(t *testing.T) {
	enemy := CreateEnemyByID(15, 2, 2)
	g := &Game{state: GameState{
		Map:     makeTestFloorMap(8, 8),
		Player:  Player{Entity: Entity{X: 7, Y: 7}},
		Enemies: []Enemy{enemy},
	}}
	if !g.tryMoveEnemy(0, 1, 0) || !g.createTrapAfterEnemyMove(0, 2, 2, func(int) int { return 0 }) {
		t.Fatal("trap-creating enemy should leave a trap after moving")
	}
	if len(g.state.MapTraps) != 1 || g.state.MapTraps[0].X != 3 || g.state.MapTraps[0].Y != 2 || g.state.MapTraps[0].Discovered {
		t.Fatalf("created trap = %#v, want hidden trap at (3, 2)", g.state.MapTraps)
	}

	g.state.Enemies[0].StatusAilments.Seal = true
	g.tryMoveEnemy(0, 1, 0)
	if g.createTrapAfterEnemyMove(0, 3, 2, func(int) int { return 0 }) || len(g.state.MapTraps) != 1 {
		t.Fatal("sealed trap-creating enemy should not create a trap")
	}
}

func TestMimicRevealAndSaveRoundTrip(t *testing.T) {
	for _, id := range []int{16, 17} {
		enemy := CreateEnemyByID(id, 2, 2)
		if enemy.Revealed {
			t.Fatalf("mimic %d should start disguised", id)
		}
		hiddenRestored := savedToEnemy(enemyToSaved(&enemy))
		if hiddenRestored.Revealed || hiddenRestored.Disguise != enemy.Disguise {
			t.Fatalf("hidden restored mimic %d = %#v", id, hiddenRestored)
		}
		g := &Game{state: GameState{Enemies: []Enemy{enemy}}}
		if !g.revealEnemy(0) || !g.state.Enemies[0].Revealed {
			t.Fatalf("mimic %d should reveal", id)
		}
		restored := savedToEnemy(enemyToSaved(&g.state.Enemies[0]))
		if !restored.Revealed || restored.Disguise != enemy.Disguise {
			t.Fatalf("restored mimic %d = %#v", id, restored)
		}
	}
	sealed := CreateEnemyByID(16, 2, 2)
	sealed.StatusAilments.Seal = true
	if isEnemyDisguised(sealed) {
		t.Fatal("sealed mimic should not keep its disguise ability")
	}
}

func TestSeenMimicStaysOnMiniMapAsDisguise(t *testing.T) {
	enemy := CreateEnemyByID(16, 4, 4)
	enemy.ShowOnMiniMap = false
	g := &Game{
		state: GameState{
			Player:  Player{Entity: Entity{X: 2, Y: 2}},
			Enemies: []Enemy{enemy},
		},
		rooms: []Room{{ID: 1, X: 0, Y: 0, Width: 8, Height: 8}},
	}
	g.updateEnemyVisibility()
	if !g.state.Enemies[0].PlayerDiscovered || !g.state.Enemies[0].ShowOnMiniMap {
		t.Fatal("seen mimic should be remembered on the minimap while disguised")
	}
}

func TestSpecialMovementRespectsCommonActionStatuses(t *testing.T) {
	wallMap := makeTestFloorMap(7, 5)
	wallMap[2][2] = Tile{Type: "wall", Blocked: true, BlockSight: true}
	wallEnemy := CreateEnemyByID(11, 1, 2)
	wallEnemy.PlayerDiscovered = true
	wallEnemy.StatusAilments.Confusion = 3
	wallGame := &Game{state: GameState{
		Map:     wallMap,
		Player:  Player{Entity: Entity{X: 4, Y: 2}},
		Enemies: []Enemy{wallEnemy},
	}}
	wallGame.actEnemy(0)
	if wallGame.state.Map[wallGame.state.Enemies[0].Y][wallGame.state.Enemies[0].X].Blocked {
		t.Fatal("confused wall-passing enemy should use common confused movement")
	}

	warpEnemy := CreateEnemyByID(13, 2, 2)
	warpEnemy.PlayerDiscovered = true
	warpEnemy.StatusAilments.Sleep = 3
	warpGame := &Game{state: GameState{
		Map:     makeTestFloorMap(8, 8),
		Player:  Player{Entity: Entity{X: 6, Y: 6}},
		Enemies: []Enemy{warpEnemy},
	}}
	warpGame.actEnemy(0)
	if warpGame.state.Enemies[0].X != 2 || warpGame.state.Enemies[0].Y != 2 || len(warpGame.ActionQueue.Queue) != 0 {
		t.Fatal("sleeping warp enemy should not move or use its ability")
	}

	trapEnemy := CreateEnemyByID(15, 2, 2)
	trapEnemy.Direction = Right
	trapEnemy.StatusAilments.Blind = 3
	trapGame := &Game{state: GameState{
		Map:     makeTestFloorMap(8, 8),
		Player:  Player{Entity: Entity{X: 6, Y: 6}},
		Enemies: []Enemy{trapEnemy},
	}}
	trapGame.actEnemy(0)
	if len(trapGame.state.MapTraps) != 0 {
		t.Fatal("blind trap-creating enemy should use common blind movement without creating a trap")
	}
}

func TestSpecialMovementEnemiesAppearInFloorSpawnTables(t *testing.T) {
	wantIDs := map[int]bool{11: false, 12: false, 13: false, 14: false, 15: false, 16: false, 17: false}
	for floor := 1; floor <= deepestSpawnFloor; floor++ {
		for _, entry := range FloorSpawnTables[floor] {
			if _, ok := wantIDs[entry.MonsterID]; ok {
				wantIDs[entry.MonsterID] = true
			}
		}
	}
	for id, found := range wantIDs {
		if !found {
			t.Fatalf("special movement enemy %d is missing from floor spawn tables", id)
		}
	}
}

func TestEnemySprites(t *testing.T) {
	files := []string{
		"img/yurei_ebi.png",
		"img/hayate_shako.png",
		"img/warp_kurage.png",
		"img/irekae_dako.png",
		"img/wanashi_yadokari.png",
		"img/mimic_gai.png",
		"img/sea_dragon.png",
		"img/fire_squid.png",
		"img/magic_jellyfish.png",
	}
	for _, path := range specialWeaponImagePaths {
		files = append(files, path)
	}
	for _, path := range specialArmorImagePaths {
		files = append(files, path)
	}
	for _, pair := range upperSpeciesSpritePairs() {
		files = append(files, pair[1])
	}
	for _, filename := range files {
		file, err := os.Open(filename)
		if err != nil {
			t.Fatalf("open %s: %v", filename, err)
		}
		image, err := png.Decode(file)
		file.Close()
		if err != nil {
			t.Fatalf("decode %s: %v", filename, err)
		}
		if image.Bounds().Dx() != 30 || image.Bounds().Dy() != 30 {
			t.Fatalf("%s size = %dx%d, want 30x30", filename, image.Bounds().Dx(), image.Bounds().Dy())
		}
		corners := [][2]int{{0, 0}, {29, 0}, {0, 29}, {29, 29}}
		for _, corner := range corners {
			_, _, _, alpha := image.At(corner[0], corner[1]).RGBA()
			if alpha != 0 {
				t.Fatalf("%s corner (%d, %d) is not transparent", filename, corner[0], corner[1])
			}
		}
	}
}

func upperSpeciesSpritePairs() [][2]string {
	return [][2]string{
		{"img/ebi.png", "img/dai_ebi.png"},
		{"img/mamuru.png", "img/anagura_mamuru.png"},
		{"img/snake.png", "img/moudoku_hebi.png"},
		{"img/honey.png", "img/gunegune_honey.png"},
		{"img/harisenbow.png", "img/harisen_ou.png"},
		{"img/ishigani.png", "img/ganseki_gani.png"},
		{"img/bakudan_uni.png", "img/dynamite_uni.png"},
		{"img/kosodoro_yadokari.png", "img/oodorobou_yadokari.png"},
		{"img/nigiri_ebi.png", "img/nigiri_oyakata_ebi.png"},
		{"img/noroi_gani.png", "img/tatari_gani.png"},
		{"img/ayatsuri_kurage.png", "img/shihai_kurage.png"},
		{"img/yurei_ebi.png", "img/shinigami_ebi.png"},
		{"img/hayate_shako.png", "img/inazuma_shako.png"},
		{"img/warp_kurage.png", "img/telepo_kurage.png"},
		{"img/irekae_dako.png", "img/tokkae_dako.png"},
		{"img/wanashi_yadokari.png", "img/wana_master_yadokari.png"},
		{"img/mimic_gai.png", "img/bake_horagai.png"},
		{"img/sea_dragon.png", "img/azure_sea_dragon.png"},
		{"img/fire_squid.png", "img/inferno_squid.png"},
		{"img/magic_jellyfish.png", "img/arcane_jellyfish.png"},
	}
}

func TestUpperSpeciesSpritesOnlyRecolorOriginal(t *testing.T) {
	for _, pair := range upperSpeciesSpritePairs() {
		originalFile, err := os.Open(pair[0])
		if err != nil {
			t.Fatalf("open %s: %v", pair[0], err)
		}
		original, err := png.Decode(originalFile)
		originalFile.Close()
		if err != nil {
			t.Fatalf("decode %s: %v", pair[0], err)
		}

		variantFile, err := os.Open(pair[1])
		if err != nil {
			t.Fatalf("open %s: %v", pair[1], err)
		}
		variant, err := png.Decode(variantFile)
		variantFile.Close()
		if err != nil {
			t.Fatalf("decode %s: %v", pair[1], err)
		}

		changedPixels := 0
		for y := 0; y < 30; y++ {
			for x := 0; x < 30; x++ {
				originalRed, originalGreen, originalBlue, originalAlpha := original.At(x, y).RGBA()
				variantRed, variantGreen, variantBlue, variantAlpha := variant.At(x, y).RGBA()
				if originalAlpha != variantAlpha {
					t.Fatalf("%s alpha at (%d, %d) = %d, want %d", pair[1], x, y, variantAlpha, originalAlpha)
				}
				if originalAlpha > 0 && (originalRed != variantRed || originalGreen != variantGreen || originalBlue != variantBlue) {
					changedPixels++
				}
			}
		}
		if changedPixels == 0 {
			t.Fatalf("%s does not recolor %s", pair[1], pair[0])
		}
	}
}

func TestEnemyDefeatsMonsterAndLevelsUp(t *testing.T) {
	target := CreateEnemyByID(2, 3, 2)
	target.Health = 1
	attacker := CreateEnemyByID(0, 2, 2)
	attacker.Health = 2
	attacker.StatusAilments.Blind = 4
	g := &Game{state: GameState{
		Map:     makeTestFloorMap(8, 8),
		Player:  Player{Entity: Entity{X: 6, Y: 6}},
		Enemies: []Enemy{target, attacker},
	}}

	g.AttackEnemyFromBlindEnemy(1, 0)
	if len(g.ActionQueue.Queue) != 1 {
		t.Fatalf("queued actions = %d, want 1", len(g.ActionQueue.Queue))
	}
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.state.Enemies) != 1 {
		t.Fatalf("remaining enemies = %d, want 1", len(g.state.Enemies))
	}
	upgraded := g.state.Enemies[0]
	definition := MonsterDefinitions[18]
	if upgraded.ID != 18 || upgraded.Name != definition.Name || upgraded.Type != definition.Type {
		t.Fatalf("upgraded enemy = ID %d, name %q, type %q", upgraded.ID, upgraded.Name, upgraded.Type)
	}
	if upgraded.Health != definition.MaxHealth || upgraded.MaxHealth != definition.MaxHealth {
		t.Fatalf("upgraded health = %d/%d, want %d/%d", upgraded.Health, upgraded.MaxHealth, definition.MaxHealth, definition.MaxHealth)
	}
	if upgraded.StatusAilments.Blind != 4 || upgraded.X != 2 || upgraded.Y != 2 {
		t.Fatalf("upgraded mutable state was not preserved: %#v", upgraded)
	}
	if len(g.ActionQueue.Queue) != 2 || !strings.Contains(g.ActionQueue.Queue[1].Message, "大エビにレベルアップ") {
		t.Fatalf("level-up message queue = %#v", g.ActionQueue.Queue)
	}
}

func TestEnemyWithoutUpperSpeciesDoesNotChange(t *testing.T) {
	attacker := CreateEnemyByID(46, 2, 2) // 第3段階はこれ以上レベルアップしない
	target := CreateEnemyByID(2, 3, 2)
	g := &Game{state: GameState{Enemies: []Enemy{attacker, target}}}

	g.defeatEnemyByEnemy(0, 1)

	if len(g.state.Enemies) != 1 || g.state.Enemies[0].ID != 46 {
		t.Fatalf("remaining enemy = %#v, want unchanged upper species", g.state.Enemies)
	}
	if len(g.ActionQueue.Queue) != 0 {
		t.Fatalf("unexpected level-up action = %#v", g.ActionQueue.Queue)
	}
}

func TestDisguisedEnemyLevelUpPreservesRevealState(t *testing.T) {
	enemy := CreateEnemyByID(16, 2, 2)
	enemy.Revealed = true
	g := &Game{state: GameState{Enemies: []Enemy{enemy}}}

	if !g.levelUpEnemy(0) {
		t.Fatal("revealed mimic should level up")
	}
	upgraded := g.state.Enemies[0]
	if upgraded.ID != 34 || upgraded.Disguise != EnemyDisguiseItem || !upgraded.Revealed {
		t.Fatalf("upgraded mimic state = ID %d, disguise %d, revealed %v", upgraded.ID, upgraded.Disguise, upgraded.Revealed)
	}
}

func TestExplosionKillLevelsUpAttacker(t *testing.T) {
	attacker := CreateEnemyByID(0, 2, 2)
	target := CreateEnemyByID(2, 4, 4)
	target.Health = 1
	g := &Game{state: GameState{Enemies: []Enemy{attacker, target}}}
	attack := RangedAttackDefinition{Kind: RangedAttackExplosion, AttackPower: 100, BlastRadius: 1}

	g.enqueueExplosionCollateral(attacker.ID, attacker.X, attacker.Y, 4, 4, attack)
	if len(g.ActionQueue.Queue) != 1 {
		t.Fatalf("queued actions = %d, want 1", len(g.ActionQueue.Queue))
	}
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.state.Enemies) != 1 || g.state.Enemies[0].ID != 18 {
		t.Fatalf("enemies after explosion = %#v, want upgraded attacker", g.state.Enemies)
	}
}

func TestUpperSpeciesSaveRoundTrip(t *testing.T) {
	for _, upperID := range MonsterLevelUpTable {
		enemy := CreateEnemyByID(upperID, 3, 4)
		restored := savedToEnemy(enemyToSaved(&enemy))
		if restored.ID != enemy.ID || restored.Name != enemy.Name || restored.Type != enemy.Type ||
			restored.SpecialMovement != enemy.SpecialMovement || restored.Disguise != enemy.Disguise ||
			restored.RangedAttack != enemy.RangedAttack || (restored.SpecialAttack == nil) != (enemy.SpecialAttack == nil) {
			t.Fatalf("restored upper-species enemy %d = %#v", upperID, restored)
		}
	}
}

func TestUpperSpeciesAppearInFloorSpawnTables(t *testing.T) {
	wantIDs := make(map[int]bool, len(MonsterLevelUpTable))
	for _, upperID := range MonsterLevelUpTable {
		wantIDs[upperID] = false
	}
	for floor := 1; floor <= deepestSpawnFloor; floor++ {
		for _, entry := range FloorSpawnTables[floor] {
			if _, ok := wantIDs[entry.MonsterID]; ok {
				wantIDs[entry.MonsterID] = true
			}
		}
	}
	for id, found := range wantIDs {
		if !found {
			t.Fatalf("upper-species enemy %d is missing from floor spawn tables", id)
		}
	}
}

func TestThiefStealsFleesAndDropsItem(t *testing.T) {
	stolen := buildItemFromTemplate(1, 0, 0)
	equipped := buildItemFromTemplate(20, 0, 0).(*Weapon)
	player := Player{Inventory: []Item{equipped, stolen}, Level: 1}
	player.EquippedWeapon = equipped
	enemy := CreateEnemyByID(7, 3, 3)
	g := &Game{state: GameState{Player: player, Enemies: []Enemy{enemy}}}

	enqueueStealAttack(&g.state.Enemies[0], g, func(n int) int { return n - 1 })
	if len(g.ActionQueue.Queue) != 1 {
		t.Fatalf("queued actions = %d, want 1", len(g.ActionQueue.Queue))
	}
	g.ActionQueue.Queue[0].Execute(g)
	if len(g.state.Player.Inventory) != 1 || g.state.Player.Inventory[0] != equipped {
		t.Fatal("thief should steal only the unequipped item")
	}
	if g.state.Enemies[0].HeldItem != stolen || !g.state.Enemies[0].Fleeing {
		t.Fatal("thief should carry the stolen item and start fleeing")
	}

	g.defeatEnemy(0)
	if len(g.state.Items) != 1 || g.state.Items[0] != stolen {
		t.Fatal("defeated thief should drop the stolen item")
	}
	x, y := stolen.GetPosition()
	if x != 3 || y != 3 {
		t.Fatalf("dropped item position = (%d, %d), want (3, 3)", x, y)
	}
}

func TestThiefMovesAwayAfterStealing(t *testing.T) {
	enemy := CreateEnemyByID(7, 4, 4)
	enemy.HeldItem = buildItemFromTemplate(1, 0, 0)
	enemy.Fleeing = true
	g := &Game{state: GameState{
		Map:     makeTestFloorMap(10, 10),
		Player:  Player{Entity: Entity{X: 3, Y: 4}},
		Enemies: []Enemy{enemy},
	}}
	before := abs(g.state.Enemies[0].X-g.state.Player.X) + abs(g.state.Enemies[0].Y-g.state.Player.Y)
	if !g.moveEnemyAwayFromPlayer(0, func(n int) int { return 0 }) {
		t.Fatal("fleeing enemy should find a move")
	}
	after := abs(g.state.Enemies[0].X-g.state.Player.X) + abs(g.state.Enemies[0].Y-g.state.Player.Y)
	if after <= before {
		t.Fatalf("distance after fleeing = %d, want greater than %d", after, before)
	}
}

func TestFoodTransformationProtectsEquippedItem(t *testing.T) {
	equipped := buildItemFromTemplate(20, 0, 0).(*Weapon)
	target := buildItemFromTemplate(2, 0, 0)
	player := Player{Inventory: []Item{equipped, target}, EquippedWeapon: equipped}
	enemy := CreateEnemyByID(8, 1, 1)
	g := &Game{state: GameState{Player: player, Enemies: []Enemy{enemy}}}

	enqueueFoodTransformationAttack(&g.state.Enemies[0], g, func(n int) int { return 0 })
	g.ActionQueue.Queue[0].Execute(g)
	if g.state.Player.Inventory[0] != equipped {
		t.Fatal("equipped item should not be transformed")
	}
	food, ok := g.state.Player.Inventory[1].(*Food)
	if !ok || food.ID != 1 || food.Name != "ウインナー" {
		t.Fatalf("transformed item = %#v, want basic sausage", g.state.Player.Inventory[1])
	}
}

func TestCurseAttackHandlesEquippedAccessory(t *testing.T) {
	accessory := buildItemFromTemplate(10, 0, 0).(*Accessory)
	player := Player{Inventory: []Item{accessory}, Power: 11, MaxPower: 11}
	player.EquippedAccessories[0] = accessory
	enemy := CreateEnemyByID(9, 1, 1)
	g := &Game{state: GameState{Player: player, Enemies: []Enemy{enemy}}}

	enqueueCurseAttack(&g.state.Enemies[0], g, func(n int) int { return 0 })
	g.ActionQueue.Queue[0].Execute(g)
	if !accessory.Cursed {
		t.Fatal("curse attack should curse an equipped accessory")
	}
	if g.state.Player.Power != 5 || g.state.Player.MaxPower != 5 {
		t.Fatalf("cursed accessory stats = (%d, %d), want (5, 5)", g.state.Player.Power, g.state.Player.MaxPower)
	}
}

func TestCurseAttackMarksArrowAsCursedEquipment(t *testing.T) {
	arrow := buildItemFromTemplate(6, 0, 0).(*Arrow)
	player := Player{Inventory: []Item{arrow}, EquippedArrow: arrow}
	enemy := CreateEnemyByID(9, 1, 1)
	g := &Game{state: GameState{Player: player, Enemies: []Enemy{enemy}}}

	enqueueCurseAttack(&g.state.Enemies[0], g, func(n int) int { return 0 })
	g.ActionQueue.Queue[0].Execute(g)
	if !arrow.Cursed || !isCursedEquipment(arrow) {
		t.Fatal("curse attack should make an equipped arrow count as cursed equipment")
	}
}

func TestManipulationForcesItemUse(t *testing.T) {
	food := buildItemFromTemplate(1, 0, 0)
	enemy := CreateEnemyByID(10, 2, 2)
	g := &Game{state: GameState{
		Map:     makeTestFloorMap(6, 6),
		Player:  Player{Entity: Entity{X: 3, Y: 2}, Inventory: []Item{food}, Satiety: 10, MaxSatiety: 100},
		Enemies: []Enemy{enemy},
	}}

	enqueueManipulationAttack(&g.state.Enemies[0], g, func(n int) int { return 0 })
	g.ActionQueue.Queue[0].Execute(g)
	if len(g.state.Player.Inventory) != 0 {
		t.Fatal("manipulation should force the selected food to be consumed")
	}
	if len(g.ActionQueue.Queue) < 2 {
		t.Fatal("forced item use should enqueue the item's normal effects")
	}
}

func TestManipulationForcesSafeMove(t *testing.T) {
	enemy := CreateEnemyByID(10, 2, 2)
	g := &Game{state: GameState{
		Map:     makeTestFloorMap(6, 6),
		Player:  Player{Entity: Entity{X: 3, Y: 2}},
		Enemies: []Enemy{enemy},
	}}
	startX, startY := g.state.Player.X, g.state.Player.Y
	intn := func(n int) int {
		if n == 2 {
			return 1
		}
		return 0
	}
	enqueueManipulationAttack(&g.state.Enemies[0], g, intn)
	g.ActionQueue.Queue[0].Execute(g)
	if g.state.Player.X == startX && g.state.Player.Y == startY {
		t.Fatal("manipulation should force the player to move")
	}
	if g.state.Map[g.state.Player.Y][g.state.Player.X].Blocked {
		t.Fatal("forced movement must not enter a blocked tile")
	}
}

func TestSealedInventoryEnemyCannotUseSpecialAttack(t *testing.T) {
	item := buildItemFromTemplate(1, 0, 0)
	enemy := CreateEnemyByID(7, 2, 2)
	enemy.StatusAilments.Seal = true
	enemy.SpecialAttackProbability = 1
	g := &Game{state: GameState{
		Player:  Player{Entity: Entity{X: 3, Y: 2}, Health: 100, Inventory: []Item{item}},
		Enemies: []Enemy{enemy},
	}}

	g.AttackFromEnemy(0)
	g.ActionQueue.Queue[0].Execute(g)
	if len(g.state.Player.Inventory) != 1 || g.state.Enemies[0].HeldItem != nil {
		t.Fatal("sealed inventory enemy should use a normal attack instead of stealing")
	}
}

func TestInventoryEnemySaveRoundTrip(t *testing.T) {
	enemy := CreateEnemyByID(7, 3, 4)
	enemy.HeldItem = buildItemFromTemplate(1, 0, 0)
	enemy.Fleeing = true
	restored := savedToEnemy(enemyToSaved(&enemy))
	if !restored.Fleeing || restored.HeldItem == nil || restored.HeldItem.GetID() != 1 {
		t.Fatalf("restored thief state = %#v", restored)
	}
}

func TestInventoryEnemiesAppearInFloorSpawnTables(t *testing.T) {
	wantIDs := map[int]bool{7: false, 8: false, 9: false, 10: false}
	for floor := 1; floor <= deepestSpawnFloor; floor++ {
		for _, entry := range FloorSpawnTables[floor] {
			if _, ok := wantIDs[entry.MonsterID]; ok {
				wantIDs[entry.MonsterID] = true
			}
		}
	}
	for id, found := range wantIDs {
		if !found {
			t.Fatalf("inventory enemy %d is missing from floor spawn tables", id)
		}
	}
}

func TestRangedEnemyAttackConditions(t *testing.T) {
	tests := []struct {
		name       string
		enemyID    int
		enemyX     int
		enemyY     int
		playerX    int
		playerY    int
		wallX      int
		wallY      int
		wantAttack bool
	}{
		{name: "arrow clear line", enemyID: 4, enemyX: 1, enemyY: 4, playerX: 7, playerY: 4, wantAttack: true},
		{name: "arrow blocked by wall", enemyID: 4, enemyX: 1, enemyY: 4, playerX: 7, playerY: 4, wallX: 4, wallY: 4},
		{name: "arrow requires straight line", enemyID: 4, enemyX: 1, enemyY: 4, playerX: 6, playerY: 6},
		{name: "rock crosses wall", enemyID: 5, enemyX: 1, enemyY: 4, playerX: 5, playerY: 6, wallX: 3, wallY: 5, wantAttack: true},
		{name: "explosion clear line", enemyID: 6, enemyX: 2, enemyY: 2, playerX: 6, playerY: 6, wantAttack: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapState := makeTestFloorMap(10, 10)
			if tt.wallX != 0 || tt.wallY != 0 {
				mapState[tt.wallY][tt.wallX] = Tile{Type: "wall", Blocked: true, BlockSight: true}
			}
			enemy := CreateEnemyByID(tt.enemyID, tt.enemyX, tt.enemyY)
			enemy.PlayerDiscovered = true
			g := &Game{state: GameState{
				Map:     mapState,
				Player:  Player{Entity: Entity{X: tt.playerX, Y: tt.playerY}, Health: 100, DefensePower: 3},
				Enemies: []Enemy{enemy},
			}}

			if got := g.tryEnemyRangedAttack(0); got != tt.wantAttack {
				t.Fatalf("tryEnemyRangedAttack = %v, want %v", got, tt.wantAttack)
			}
			if tt.wantAttack {
				if len(g.ActionQueue.Queue) != 1 {
					t.Fatalf("queued actions = %d, want 1", len(g.ActionQueue.Queue))
				}
				g.ActionQueue.Queue[0].Execute(g)
				if g.state.Player.Health >= 100 {
					t.Fatal("ranged attack should damage the player")
				}
				if g.rangedAttackEffect.Kind != enemy.RangedAttack.Kind {
					t.Fatalf("effect kind = %d, want %d", g.rangedAttackEffect.Kind, enemy.RangedAttack.Kind)
				}
			}
		})
	}
}

func TestSealedRangedEnemyCannotUseAbility(t *testing.T) {
	enemy := CreateEnemyByID(5, 2, 2)
	enemy.PlayerDiscovered = true
	enemy.StatusAilments.Seal = true
	g := &Game{state: GameState{
		Map:     makeTestFloorMap(10, 10),
		Player:  Player{Entity: Entity{X: 6, Y: 5}, Health: 100},
		Enemies: []Enemy{enemy},
	}}

	if g.tryEnemyRangedAttack(0) || len(g.ActionQueue.Queue) != 0 {
		t.Fatal("sealed enemy should not queue a ranged attack")
	}
}

func TestRangedEnemyRespectsCommonActionBlockingStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status StatusAilments
	}{
		{name: "sleep", status: StatusAilments{Sleep: 3}},
		{name: "paralysis", status: StatusAilments{Paralysis: true}},
		{name: "confusion", status: StatusAilments{Confusion: 3}},
		{name: "blind", status: StatusAilments{Blind: 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enemy := CreateEnemyByID(4, 2, 5)
			enemy.PlayerDiscovered = true
			enemy.StatusAilments = tt.status
			g := &Game{state: GameState{
				Map:     makeTestFloorMap(12, 12),
				Player:  Player{Entity: Entity{X: 8, Y: 5}, Health: 100},
				Enemies: []Enemy{enemy},
			}}

			g.actEnemy(0)
			if len(g.ActionQueue.Queue) != 0 {
				t.Fatalf("%s enemy should not queue a ranged attack", tt.name)
			}
		})
	}
}

func TestExplosionRangedAttackDamagesNearbyEnemies(t *testing.T) {
	attacker := CreateEnemyByID(6, 2, 2)
	attacker.PlayerDiscovered = true
	nearby := CreateEnemyByID(0, 7, 5)
	nearby.Health = 100
	distant := CreateEnemyByID(0, 8, 2)
	distant.Health = 100
	g := &Game{state: GameState{
		Map:     makeTestFloorMap(12, 12),
		Player:  Player{Entity: Entity{X: 6, Y: 6}, Health: 100, DefensePower: 3},
		Enemies: []Enemy{attacker, nearby, distant},
	}}

	if !g.tryEnemyRangedAttack(0) {
		t.Fatal("explosion enemy should queue its ranged attack")
	}
	g.ActionQueue.Queue[0].Execute(g)
	if len(g.ActionQueue.Queue) != 2 {
		t.Fatalf("actions after impact = %d, want primary attack and delayed collateral", len(g.ActionQueue.Queue))
	}
	g.ActionQueue.Queue[1].Execute(g)

	if g.state.Enemies[1].Health >= 100 {
		t.Fatal("enemy adjacent to the impact should take blast damage")
	}
	if g.state.Enemies[2].Health != 100 {
		t.Fatal("enemy outside the blast radius should not take damage")
	}
}

func TestRangedEnemySaveRoundTrip(t *testing.T) {
	enemy := CreateEnemyByID(6, 3, 4)
	restored := savedToEnemy(enemyToSaved(&enemy))
	if restored.ID != 6 || restored.RangedAttack.Kind != RangedAttackExplosion || restored.RangedAttack.BlastRadius != 1 {
		t.Fatalf("restored ranged enemy = %#v", restored.RangedAttack)
	}
}

func TestRangedEnemyFloorSpawnTables(t *testing.T) {
	for _, entry := range FloorSpawnTables[1] {
		if entry.MonsterID >= 4 {
			t.Fatalf("ranged enemy %d should not appear on floor 1", entry.MonsterID)
		}
	}
	wantIDs := map[int]bool{4: false, 5: false, 6: false, 38: false, 39: false, 40: false, 41: false}
	for floor := 2; floor <= deepestSpawnFloor; floor++ {
		for _, entry := range FloorSpawnTables[floor] {
			if _, ok := wantIDs[entry.MonsterID]; ok {
				wantIDs[entry.MonsterID] = true
			}
		}
	}
	for id, found := range wantIDs {
		if !found {
			t.Fatalf("ranged enemy %d is missing from floor spawn tables", id)
		}
	}

	lastEntry := FloorSpawnTables[deepestSpawnFloor][len(FloorSpawnTables[deepestSpawnFloor])-1]
	got := selectMonsterForFloor(99, nil, func(n int) int { return n - 1 })
	if got != lastEntry.MonsterID {
		t.Fatalf("deep floor fallback selected %d, want %d", got, lastEntry.MonsterID)
	}
}

// 9〜30階のテーブル: 全階に定義があり、第3段階は9階から出現し8階以前には出ない。
func TestDeepFloorSpawnTables(t *testing.T) {
	thirdStage := map[int]bool{}
	for baseID := range baseMonsterDefinitions {
		thirdStage[MonsterLevelUpTable[MonsterLevelUpTable[baseID]]] = true
	}
	seen := map[int]bool{}
	for floor := 1; floor <= deepestSpawnFloor; floor++ {
		entries := FloorSpawnTables[floor]
		if len(entries) == 0 {
			t.Fatalf("floor %d has no spawn table", floor)
		}
		hasThird := false
		for _, entry := range entries {
			definition, ok := MonsterDefinitions[entry.MonsterID]
			if !ok || definition.IsBoss || entry.Weight <= 0 {
				t.Fatalf("floor %d has invalid entry %+v", floor, entry)
			}
			if thirdStage[entry.MonsterID] {
				hasThird = true
				seen[entry.MonsterID] = true
			}
		}
		if floor <= 8 && hasThird {
			t.Fatalf("third-stage monster appears on floor %d", floor)
		}
		if floor >= 9 && !hasThird {
			t.Fatalf("floor %d has no third-stage monster", floor)
		}
	}
	for id := range thirdStage {
		if !seen[id] {
			t.Fatalf("third-stage monster %d is missing from floor spawn tables", id)
		}
	}
}

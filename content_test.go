//go:build !test

package main

import (
	"fmt"
	"image/png"
	"os"
	"strings"
	"testing"
)

func TestInventoryShootUsesShootingState(t *testing.T) {
	arrow := buildItemFromTemplate(6, 0, 0).(*Arrow)
	arrow.ShotCount = 3
	g := &Game{
		state: GameState{
			Map: makeTestFloorMap(20, 20),
			Player: Player{
				Entity:    Entity{X: 10, Y: 10},
				Direction: Right,
				Inventory: []Item{arrow},
			},
		},
		selectedActionIndex: 1,
	}

	g.executeAction()

	if !g.dPressed {
		t.Fatal("inventory shoot should enable shooting state for rotation and damage")
	}
	if arrow.ShotCount != 2 {
		t.Fatalf("remaining arrows = %d, want 2", arrow.ShotCount)
	}
	if len(g.ActionQueue.Queue) != 1 || !strings.HasSuffix(g.ActionQueue.Queue[0].Message, "を撃った") {
		t.Fatalf("queued action = %#v, want shooting message", g.ActionQueue.Queue)
	}

	g.ActionQueue.Queue[0].Execute(g)
	shotArrow, ok := g.ThrownItem.Item.(*Arrow)
	if !ok {
		t.Fatalf("thrown item = %T, want *Arrow", g.ThrownItem.Item)
	}
	if shotArrow.ShotCount != 1 {
		t.Fatalf("shot arrow count = %d, want 1", shotArrow.ShotCount)
	}
	if g.ThrownItem.DX != 1 || g.ThrownItem.DY != 0 {
		t.Fatalf("shot direction = (%d, %d), want (1, 0)", g.ThrownItem.DX, g.ThrownItem.DY)
	}
}

func TestInventoryShootLastEquippedArrowUnequips(t *testing.T) {
	arrow := buildItemFromTemplate(6, 0, 0).(*Arrow)
	arrow.ShotCount = 1
	g := &Game{
		state: GameState{
			Map: makeTestFloorMap(20, 20),
			Player: Player{
				Entity:        Entity{X: 10, Y: 10},
				Direction:     Down,
				Inventory:     []Item{arrow},
				EquippedArrow: arrow,
			},
		},
		selectedActionIndex: 1,
	}

	g.executeAction()

	if len(g.state.Player.Inventory) != 0 {
		t.Fatalf("inventory length = %d, want 0 after last arrow", len(g.state.Player.Inventory))
	}
	if g.state.Player.EquippedArrow != nil {
		t.Fatal("last equipped arrow should be unequipped")
	}
	if !g.dPressed {
		t.Fatal("last arrow should still be treated as a shot")
	}
}

func TestAddedEquipmentTemplates(t *testing.T) {
	weapons := map[int]int{20: 2, 21: 4, 22: 6}
	for id, wantPower := range weapons {
		weapon, ok := buildItemFromTemplate(id, 0, 0).(*Weapon)
		if !ok || weapon.AttackPower != wantPower {
			t.Fatalf("weapon %d = %#v, want attack power %d", id, weapon, wantPower)
		}
	}

	armors := map[int]int{23: 2, 24: 5, 25: 3}
	for id, wantPower := range armors {
		armor, ok := buildItemFromTemplate(id, 0, 0).(*Armor)
		if !ok || armor.DefensePower != wantPower {
			t.Fatalf("armor %d = %#v, want defense power %d", id, armor, wantPower)
		}
		if id == 25 && !hasEquipmentAbility(armor.Abilities, AbilitySatietyConservation) {
			t.Fatal("皮甲の盾に満腹度消費軽減能力がありません")
		}
	}
}

func TestAddedContentDefinitions(t *testing.T) {
	if MonsterDefinitions[2].Name != "マムル" || MonsterDefinitions[3].Name != "くねくねハニー" {
		t.Fatal("added monster definitions are missing")
	}
	rangedEnemies := map[int]struct {
		name string
		kind RangedAttackKind
	}{
		4: {name: "ハリセンボウ", kind: RangedAttackArrow},
		5: {name: "イシガニ", kind: RangedAttackRock},
		6: {name: "バクダンウニ", kind: RangedAttackExplosion},
	}
	for id, want := range rangedEnemies {
		definition := MonsterDefinitions[id]
		if definition.Name != want.name || definition.RangedAttack.Kind != want.kind {
			t.Fatalf("ranged monster %d = %#v, want %q with kind %d", id, definition, want.name, want.kind)
		}
	}
	inventoryEnemies := map[int]string{
		7:  "コソドロヤドカリ",
		8:  "にぎりエビ",
		9:  "ノロイガニ",
		10: "あやつりクラゲ",
	}
	for id, wantName := range inventoryEnemies {
		definition := MonsterDefinitions[id]
		if definition.Name != wantName || definition.SpecialAttack == nil || definition.SpecialAttackProbability <= 0 {
			t.Fatalf("inventory monster %d = %#v, want %q with a special attack", id, definition, wantName)
		}
	}

	wantTraps := []string{"睡眠ガスの罠", "毒矢の罠", "鈍足の罠", "地雷", "サビの罠"}
	for id, wantName := range wantTraps {
		if got := createMapTrapByID(id, 1, 2); got.Name != wantName {
			t.Fatalf("trap %d = %q, want %q", id, got.Name, wantName)
		}
	}
}

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
	for id := 0; id < len(MonsterDefinitions); id++ {
		if _, ok := MonsterDefinitions[id]; !ok {
			t.Fatalf("monster definition IDs must stay contiguous; missing %d", id)
		}
	}
}

func TestEveryBaseMonsterHasUpperSpecies(t *testing.T) {
	if len(MonsterLevelUpTable) != 18 {
		t.Fatalf("level-up mappings = %d, want 18", len(MonsterLevelUpTable))
	}
	for baseID := 0; baseID < 18; baseID++ {
		upperID, ok := MonsterLevelUpTable[baseID]
		if !ok {
			t.Fatalf("base monster %d has no upper species", baseID)
		}
		base := MonsterDefinitions[baseID]
		upper, ok := MonsterDefinitions[upperID]
		if !ok {
			t.Fatalf("upper species %d for base monster %d is missing", upperID, baseID)
		}
		if upper.Name == base.Name || upper.Type == base.Type {
			t.Fatalf("upper species %d did not change identity from base %d", upperID, baseID)
		}
		if upper.Health <= base.Health || upper.AttackPower <= base.AttackPower ||
			upper.DefensePower <= base.DefensePower || upper.ExperiencePoints <= base.ExperiencePoints {
			t.Fatalf("upper species %d stats = HP %d, attack %d, defense %d, exp %d; base %d stats = HP %d, attack %d, defense %d, exp %d",
				upperID, upper.Health, upper.AttackPower, upper.DefensePower, upper.ExperiencePoints,
				baseID, base.Health, base.AttackPower, base.DefensePower, base.ExperiencePoints)
		}
		if upper.Char != base.Char || (upper.SpecialAttack == nil) != (base.SpecialAttack == nil) ||
			upper.SpecialAttackProbability != base.SpecialAttackProbability || upper.SpecialMovement != base.SpecialMovement ||
			upper.Disguise != base.Disguise {
			t.Fatalf("upper species %d did not inherit base monster %d abilities", upperID, baseID)
		}
		if upper.RangedAttack.Kind != base.RangedAttack.Kind || upper.RangedAttack.MinRange != base.RangedAttack.MinRange ||
			upper.RangedAttack.MaxRange != base.RangedAttack.MaxRange || upper.RangedAttack.BlastRadius != base.RangedAttack.BlastRadius {
			t.Fatalf("upper species %d did not inherit base monster %d ranged behavior", upperID, baseID)
		}
		if base.RangedAttack.Kind != RangedAttackNone && upper.RangedAttack.AttackPower <= base.RangedAttack.AttackPower {
			t.Fatalf("upper species %d ranged power = %d, want above base %d", upperID, upper.RangedAttack.AttackPower, base.RangedAttack.AttackPower)
		}
		if _, chained := MonsterLevelUpTable[upperID]; chained {
			t.Fatalf("upper species %d must not have another level in the current two-stage system", upperID)
		}
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
	attacker := CreateEnemyByID(20, 2, 2)
	target := CreateEnemyByID(2, 3, 2)
	g := &Game{state: GameState{Enemies: []Enemy{attacker, target}}}

	g.defeatEnemyByEnemy(0, 1)

	if len(g.state.Enemies) != 1 || g.state.Enemies[0].ID != 20 {
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
	wantIDs := map[int]bool{4: false, 5: false, 6: false}
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
	got := selectMonsterForFloor(99, func(n int) int { return n - 1 })
	if got != lastEntry.MonsterID {
		t.Fatalf("deep floor fallback selected %d, want %d", got, lastEntry.MonsterID)
	}
}

func TestRecoveryItemTemplates(t *testing.T) {
	foods := map[int]struct {
		name         string
		satiety      int
		fullRecovery bool
	}{
		26: {name: "ジャンボウインナー", satiety: 100},
		27: {name: "海老天むす", fullRecovery: true},
	}
	for id, want := range foods {
		food, ok := buildItemFromTemplate(id, 0, 0).(*Food)
		if !ok || food.Name != want.name || food.Satiety != want.satiety || food.FullRecovery != want.fullRecovery {
			t.Fatalf("food %d = %#v, want %#v", id, food, want)
		}
	}

	potions := map[int]struct {
		name         string
		health       int
		fullRecovery bool
	}{
		28: {name: "大粒ミンティア", health: 60},
		29: {name: "海老印の栄養ドリンク", fullRecovery: true},
	}
	for id, want := range potions {
		potion, ok := buildItemFromTemplate(id, 0, 0).(*Potion)
		if !ok || potion.Name != want.name || potion.Health != want.health || potion.FullRecovery != want.fullRecovery {
			t.Fatalf("potion %d = %#v, want %#v", id, potion, want)
		}
	}
}

func TestAddedCardTemplates(t *testing.T) {
	cards := map[int]string{
		30: "あかりのカード",
		31: "真空斬りのカード",
	}
	for id, wantName := range cards {
		card, ok := buildItemFromTemplate(id, 0, 0).(*Card)
		if !ok || card.Name != wantName {
			t.Fatalf("card %d = %#v, want %q", id, card, wantName)
		}
	}
}

func TestRollVacuumSlashDamage(t *testing.T) {
	if got := rollVacuumSlashDamage(func(int) int { return 0 }); got != 12 {
		t.Fatalf("minimum damage = %d, want 12", got)
	}
	if got := rollVacuumSlashDamage(func(n int) int { return n - 1 }); got != 24 {
		t.Fatalf("maximum damage = %d, want 24", got)
	}
}

func TestRoomWideCardEffectArea(t *testing.T) {
	rooms := []Room{{ID: 1, X: 0, Y: 0, Width: 5, Height: 5}}
	if !isInRoomWideEffect(2, 2, 3, 3, rooms) {
		t.Fatal("enemy in the same room should be affected")
	}
	if isInRoomWideEffect(2, 2, 6, 6, rooms) {
		t.Fatal("enemy outside the room should not be affected")
	}
	if !isInRoomWideEffect(6, 6, 7, 7, rooms) {
		t.Fatal("adjacent enemy in a corridor should be affected")
	}
	if isInRoomWideEffect(6, 6, 8, 6, rooms) {
		t.Fatal("distant enemy in a corridor should not be affected")
	}
}

func TestRevealFloorCard(t *testing.T) {
	card := buildItemFromTemplate(30, 0, 0)
	floorItem := buildItemFromTemplate(1, 4, 4)
	g := &Game{
		state: GameState{
			Map:      [][]Tile{{{Type: "floor"}, {Type: "wall"}}},
			Player:   Player{Inventory: []Item{card}},
			Enemies:  []Enemy{{Entity: Entity{X: 3, Y: 3}}},
			Items:    []Item{floorItem},
			MapTraps: []MapTrap{{X: 1, Y: 1, Discovered: false}},
		},
	}

	card.Use(g)
	if len(g.state.Player.Inventory) != 0 {
		t.Fatal("used card should be removed from inventory")
	}
	g.ActionQueue.Queue[0].Execute(g)

	if !g.state.Map[0][0].Visited || !g.state.Map[0][1].Visited {
		t.Fatal("all map tiles should be revealed")
	}
	if !g.state.Enemies[0].PlayerDiscovered || !g.state.Enemies[0].ShowOnMiniMap {
		t.Fatal("enemy should be revealed on the minimap")
	}
	if !floorItem.GetPlayerDiscovered() || !floorItem.GetShowOnMiniMap() {
		t.Fatal("floor item should be revealed on the minimap")
	}
	if g.state.MapTraps[0].Discovered {
		t.Fatal("あかりのカード should not reveal traps")
	}
}

func TestSleepCardMarksHasteOnWake(t *testing.T) {
	card := buildItemFromTemplate(12, 0, 0)
	g := &Game{
		state: GameState{
			Player: Player{Entity: Entity{X: 2, Y: 2}, Inventory: []Item{card}},
			Enemies: []Enemy{
				{Entity: Entity{X: 3, Y: 3}, Health: 10},
				{Entity: Entity{X: 6, Y: 6}, Health: 10},
			},
		},
		rooms: []Room{{ID: 1, X: 0, Y: 0, Width: 5, Height: 5}},
	}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if got := g.state.Enemies[0].StatusAilments; got.Sleep != 10 || !got.HasteOnWake {
		t.Fatalf("same-room enemy status = %#v, want Sleep 10 with HasteOnWake", got)
	}
	if got := g.state.Enemies[1].StatusAilments; got.Sleep != 0 || got.HasteOnWake {
		t.Fatalf("outside enemy status = %#v, want unaffected", got)
	}
}

func TestEnemyWakesUpHasted(t *testing.T) {
	g := &Game{
		state: GameState{
			Enemies: []Enemy{{Name: "マムル", StatusAilments: StatusAilments{Sleep: 1, HasteOnWake: true}}},
		},
	}

	g.decrementStatusAilments()

	if got := g.state.Enemies[0].StatusAilments; got.Sleep != 0 || got.HasteOnWake || got.Haste != hasteOnWakeTurns {
		t.Fatalf("enemy status after waking = %#v, want Haste %d", got, hasteOnWakeTurns)
	}
	if len(g.ActionQueue.Queue) == 0 {
		t.Fatal("waking up hasted should enqueue a message")
	}

	g.decrementStatusAilments()
	if got := g.state.Enemies[0].StatusAilments.Haste; got != hasteOnWakeTurns-1 {
		t.Fatalf("haste turns = %d, want %d", got, hasteOnWakeTurns-1)
	}
}

func TestPotTemplate(t *testing.T) {
	pot, ok := buildItemFromTemplate(46, 0, 0).(*Pot)
	if !ok || pot.Name != "保存の壺" {
		t.Fatalf("pot template = %#v, want 保存の壺", pot)
	}
	if pot.Capacity < 3 || pot.Capacity > 5 {
		t.Fatalf("pot capacity = %d, want 3-5", pot.Capacity)
	}
	if !pot.Identified {
		t.Fatal("pot should start identified")
	}
	want := fmt.Sprintf("保存の壺[0/%d]", pot.Capacity)
	if got := getItemNameWithSharpness(pot); got != want {
		t.Fatalf("pot display name = %q, want %q", got, want)
	}
}

func TestPotCardTemplates(t *testing.T) {
	cards := map[int]string{
		47: "壺拡大のカード",
		48: "吸い出しのカード",
	}
	for id, wantName := range cards {
		card, ok := buildItemFromTemplate(id, 0, 0).(*Card)
		if !ok || card.Name != wantName {
			t.Fatalf("card %d = %#v, want %q", id, card, wantName)
		}
	}
}

func TestExpandedPotCapacity(t *testing.T) {
	if got := expandedPotCapacity(3); got != 4 {
		t.Fatalf("expandedPotCapacity(3) = %d, want 4", got)
	}
	if got := expandedPotCapacity(maxPotCapacity); got != maxPotCapacity {
		t.Fatalf("expandedPotCapacity(max) = %d, want %d", got, maxPotCapacity)
	}
}

func TestPotInsertAndTakeOut(t *testing.T) {
	pot := buildItemFromTemplate(46, 0, 0).(*Pot)
	food := buildItemFromTemplate(1, 0, 0)
	g := &Game{state: GameState{Player: Player{Inventory: []Item{pot, food}, MaxInventory: 20}}}

	g.potInsertIndex = 0
	g.selectedItemIndex = 1
	g.executePotInsertSelection()
	if len(pot.Contents) != 1 || len(g.state.Player.Inventory) != 1 {
		t.Fatalf("after insert: contents = %d, inventory = %d, want 1 and 1",
			len(pot.Contents), len(g.state.Player.Inventory))
	}

	g.executePotTakeOut(pot)
	if len(pot.Contents) != 0 || len(g.state.Player.Inventory) != 2 {
		t.Fatalf("after take out: contents = %d, inventory = %d, want 0 and 2",
			len(pot.Contents), len(g.state.Player.Inventory))
	}
}

func TestPotRejectsNestedPot(t *testing.T) {
	pot := buildItemFromTemplate(46, 0, 0).(*Pot)
	otherPot := buildItemFromTemplate(46, 0, 0).(*Pot)
	g := &Game{state: GameState{Player: Player{Inventory: []Item{pot, otherPot}, MaxInventory: 20}}}

	g.potInsertIndex = 0
	g.selectedItemIndex = 1
	g.executePotInsertSelection()
	if len(pot.Contents) != 0 || len(g.state.Player.Inventory) != 2 {
		t.Fatalf("pot should not accept another pot: contents = %d, inventory = %d",
			len(pot.Contents), len(g.state.Player.Inventory))
	}
}

func TestExpandPotsCard(t *testing.T) {
	pot := buildItemFromTemplate(46, 0, 0).(*Pot)
	capBefore := pot.Capacity
	card := buildItemFromTemplate(47, 0, 0)
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card, pot}, MaxInventory: 20}}}
	g.selectedItemIndex = 0

	card.Use(g)
	if len(g.state.Player.Inventory) != 1 {
		t.Fatal("used card should be removed from inventory")
	}
	for _, action := range g.ActionQueue.Queue {
		action.Execute(g)
	}
	if pot.Capacity != capBefore+1 {
		t.Fatalf("pot capacity = %d, want %d", pot.Capacity, capBefore+1)
	}
}

func TestSuckOutPotsCard(t *testing.T) {
	pot := buildItemFromTemplate(46, 0, 0).(*Pot)
	pot.Contents = []Item{buildItemFromTemplate(1, 0, 0), buildItemFromTemplate(2, 0, 0)}
	card := buildItemFromTemplate(48, 0, 0)
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card, pot}, MaxInventory: 20}}}
	g.selectedItemIndex = 0

	card.Use(g)
	if len(g.state.Player.Inventory) != 1 {
		t.Fatal("used card should be removed from inventory")
	}
	for _, action := range g.ActionQueue.Queue {
		action.Execute(g)
	}
	if len(pot.Contents) != 0 {
		t.Fatalf("pot contents = %d, want 0", len(pot.Contents))
	}
	// 壺1つ + 取り出した中身2つ
	if len(g.state.Player.Inventory) != 3 {
		t.Fatalf("inventory = %d, want 3", len(g.state.Player.Inventory))
	}
}

func TestSuckOutAllPotsLeftover(t *testing.T) {
	pot := buildItemFromTemplate(46, 0, 0).(*Pot)
	pot.Contents = []Item{buildItemFromTemplate(1, 0, 0), buildItemFromTemplate(2, 0, 0)}
	// 空きが1つしかないインベントリ
	g := &Game{state: GameState{Player: Player{Inventory: []Item{pot}, MaxInventory: 2}}}

	moved, leftover := g.suckOutAllPots()
	if moved != 1 || !leftover {
		t.Fatalf("moved = %d, leftover = %v, want 1 and true", moved, leftover)
	}
	if len(pot.Contents) != 1 {
		t.Fatalf("pot contents = %d, want 1", len(pot.Contents))
	}
}

func TestScatterPotContents(t *testing.T) {
	pot := buildItemFromTemplate(46, 0, 0).(*Pot)
	pot.Contents = []Item{buildItemFromTemplate(1, 0, 0), buildItemFromTemplate(2, 0, 0)}
	tiles := make([][]Tile, 3)
	for y := range tiles {
		tiles[y] = make([]Tile, 3)
		for x := range tiles[y] {
			tiles[y][x] = Tile{Type: "floor"}
		}
	}
	g := &Game{state: GameState{Map: tiles}}

	g.scatterPotContents(pot, 1, 1)
	if len(g.state.Items) != 2 {
		t.Fatalf("scattered items = %d, want 2", len(g.state.Items))
	}
	if len(pot.Contents) != 0 {
		t.Fatal("pot contents should be empty after scattering")
	}
}

func TestPotSaveRoundTrip(t *testing.T) {
	pot := buildItemFromTemplate(46, 3, 4).(*Pot)
	pot.Capacity = 5
	pot.Contents = []Item{buildItemFromTemplate(1, 0, 0)}

	saved := itemToSaved(pot)
	restored, err := savedToItem(saved)
	if err != nil {
		t.Fatalf("savedToItem failed: %v", err)
	}
	restoredPot, ok := restored.(*Pot)
	if !ok {
		t.Fatalf("restored item = %#v, want *Pot", restored)
	}
	if restoredPot.Capacity != 5 || len(restoredPot.Contents) != 1 {
		t.Fatalf("restored pot capacity = %d, contents = %d, want 5 and 1",
			restoredPot.Capacity, len(restoredPot.Contents))
	}
	if restoredPot.Contents[0].GetID() != 1 {
		t.Fatalf("restored content ID = %d, want 1", restoredPot.Contents[0].GetID())
	}
}

func TestRollIdentifyAll(t *testing.T) {
	if !rollIdentifyAll(func(int) int { return 0 }) {
		t.Fatal("最小値では全識別が発動するはず")
	}
	if rollIdentifyAll(func(n int) int { return n - 1 }) {
		t.Fatal("最大値では全識別は発動しないはず")
	}
}

func TestIdentifyAllInventory(t *testing.T) {
	weapon := buildItemFromTemplate(20, 0, 0)
	cane := buildItemFromTemplate(9, 0, 0)
	food := buildItemFromTemplate(1, 0, 0)
	g := &Game{
		state: GameState{
			Player: Player{Inventory: []Item{weapon, cane, food}},
		},
	}

	if got := identifyAllInventory(g); got != 2 {
		t.Fatalf("identified %d items, want 2", got)
	}
	if !weapon.(Identifiable).GetIdentified() || !cane.(Identifiable).GetIdentified() {
		t.Fatal("識別可能なアイテムはすべて識別されるはず")
	}
}

func TestEquipmentCardTemplates(t *testing.T) {
	cards := map[int]string{
		32: "おはらいのカード",
		33: "武器強化のカード",
		34: "盾強化のカード",
	}
	for id, wantName := range cards {
		card, ok := buildItemFromTemplate(id, 0, 0).(*Card)
		if !ok || card.Name != wantName {
			t.Fatalf("card %d = %#v, want %q", id, card, wantName)
		}
	}
}

func TestUncursePlayerBelongings(t *testing.T) {
	weapon := &Weapon{BaseItem: BaseItem{Name: "こん棒", Type: "Weapon"}, AttackPower: 2, Sharpness: -1, Cursed: true}
	accessory := &Accessory{BaseItem: BaseItem{Name: "鼓舞の指輪", Type: "Accessory"}, Cursed: true}
	plainArmor := &Armor{BaseItem: BaseItem{Name: "木甲の盾", Type: "Armor"}, DefensePower: 2}
	player := &Player{Inventory: []Item{weapon, accessory, plainArmor}}
	player.EquipItem(weapon)
	player.EquipItem(accessory)
	attackAfterEquip := player.AttackPower

	uncursed := uncursePlayerBelongings(player)

	if len(uncursed) != 2 {
		t.Fatalf("uncursed = %v, want 2 items", uncursed)
	}
	if weapon.Cursed || accessory.Cursed {
		t.Fatal("装備中の呪いはすべて解けるはず")
	}
	if player.AttackPower != attackAfterEquip {
		t.Fatalf("attack power = %d, want unchanged %d", player.AttackPower, attackAfterEquip)
	}
	// 呪われたアクセサリは補正が反転しているため、解呪後は正の補正へ付け直される
	if player.Power != 3 || player.MaxPower != 3 {
		t.Fatalf("accessory stats = Power %d MaxPower %d, want 3/3", player.Power, player.MaxPower)
	}
}

func TestReinforceWeaponCard(t *testing.T) {
	card := buildItemFromTemplate(33, 0, 0)
	weapon := &Weapon{BaseItem: BaseItem{Name: "こん棒", Type: "Weapon"}, AttackPower: 2, Cursed: true}
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card, weapon}}}}
	g.state.Player.EquipItem(weapon)

	card.Use(g)
	if len(g.state.Player.Inventory) != 1 {
		t.Fatal("used card should be removed from inventory")
	}
	g.ActionQueue.Queue[0].Execute(g)

	if weapon.Sharpness != 1 || weapon.Cursed {
		t.Fatalf("weapon = %#v, want Sharpness 1 and uncursed", weapon)
	}
	if g.state.Player.AttackPower != 3 {
		t.Fatalf("player attack power = %d, want 3", g.state.Player.AttackPower)
	}
}

func TestReinforceArmorCard(t *testing.T) {
	card := buildItemFromTemplate(34, 0, 0)
	armor := &Armor{BaseItem: BaseItem{Name: "木甲の盾", Type: "Armor"}, DefensePower: 2}
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card, armor}}}}
	g.state.Player.EquipItem(armor)

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if armor.Sharpness != 1 {
		t.Fatalf("armor sharpness = %d, want 1", armor.Sharpness)
	}
	if g.state.Player.DefensePower != 3 {
		t.Fatalf("player defense power = %d, want 3", g.state.Player.DefensePower)
	}
}

func TestReinforceArmorCardWithoutShield(t *testing.T) {
	card := buildItemFromTemplate(34, 0, 0)
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card}}}}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if g.state.Player.DefensePower != 0 {
		t.Fatalf("defense power = %d, want unchanged 0", g.state.Player.DefensePower)
	}
	if len(g.ActionQueue.Queue) < 2 || g.ActionQueue.Queue[1].Message != "しかし何も起こらなかった" {
		t.Fatal("盾なしで使用した場合は不発メッセージを表示するはず")
	}
}

func TestFloorCardTemplates(t *testing.T) {
	cards := map[int]string{
		35: "モンスターハウスのカード",
		36: "敵倍速のカード",
		37: "地図忘却のカード",
		38: "拾得禁止のカード",
		39: "大部屋のカード",
		40: "罠のカード",
	}
	for id, wantName := range cards {
		card, ok := buildItemFromTemplate(id, 0, 0).(*Card)
		if !ok || card.Name != wantName {
			t.Fatalf("card %d = %#v, want %q", id, card, wantName)
		}
	}
}

func TestFloorCardRolls(t *testing.T) {
	minIntn := func(int) int { return 0 }
	maxIntn := func(n int) int { return n - 1 }
	if got := rollMonsterHouseEnemyCount(minIntn); got != 5 {
		t.Fatalf("minimum enemy count = %d, want 5", got)
	}
	if got := rollMonsterHouseEnemyCount(maxIntn); got != 8 {
		t.Fatalf("maximum enemy count = %d, want 8", got)
	}
	if got := rollMonsterHouseItemCount(minIntn); got != 2 {
		t.Fatalf("minimum item count = %d, want 2", got)
	}
	if got := rollMonsterHouseItemCount(maxIntn); got != 3 {
		t.Fatalf("maximum item count = %d, want 3", got)
	}
	if got := rollExtraTrapCount(minIntn); got != 5 {
		t.Fatalf("minimum trap count = %d, want 5", got)
	}
	if got := rollExtraTrapCount(maxIntn); got != 8 {
		t.Fatalf("maximum trap count = %d, want 8", got)
	}
}

func TestPickFreeCellsInRoom(t *testing.T) {
	room := Room{X: 0, Y: 0, Width: 5, Height: 5} // 内側は3x3の9マス
	isFree := func(x, y int) bool { return !(x == 1 && y == 1) }
	intn := func(n int) int { return 0 }

	cells := pickFreeCellsInRoom(room, isFree, 20, intn)
	if len(cells) != 8 {
		t.Fatalf("free cells = %d, want 8 (clamped to available)", len(cells))
	}
	for _, cell := range cells {
		if cell.X < 1 || cell.X > 3 || cell.Y < 1 || cell.Y > 3 {
			t.Fatalf("cell %v is outside the room interior", cell)
		}
		if cell.X == 1 && cell.Y == 1 {
			t.Fatal("occupied cell should not be picked")
		}
	}

	if got := pickFreeCellsInRoom(room, isFree, 3, intn); len(got) != 3 {
		t.Fatalf("picked cells = %d, want 3", len(got))
	}
}

// テスト用に外周が壁、内側が床のマップを作る
func makeTestFloorMap(width, height int) [][]Tile {
	mapGrid := make([][]Tile, height)
	for y := range mapGrid {
		mapGrid[y] = make([]Tile, width)
		for x := range mapGrid[y] {
			if x == 0 || x == width-1 || y == 0 || y == height-1 {
				mapGrid[y][x] = Tile{Type: "wall", Blocked: true, BlockSight: true}
			} else {
				mapGrid[y][x] = Tile{Type: "floor"}
			}
		}
	}
	return mapGrid
}

func TestMonsterHouseCard(t *testing.T) {
	card := buildItemFromTemplate(35, 0, 0)
	g := &Game{
		state: GameState{
			Map:    makeTestFloorMap(8, 8),
			Player: Player{Entity: Entity{X: 2, Y: 2}, Inventory: []Item{card}},
		},
		rooms: []Room{{ID: 0, X: 0, Y: 0, Width: 8, Height: 8}},
	}

	card.Use(g)
	if len(g.state.Player.Inventory) != 0 {
		t.Fatal("used card should be removed from inventory")
	}
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.state.Enemies) < 5 || len(g.state.Enemies) > 8 {
		t.Fatalf("spawned enemies = %d, want 5..8", len(g.state.Enemies))
	}
	for _, enemy := range g.state.Enemies {
		if enemy.X < 1 || enemy.X > 6 || enemy.Y < 1 || enemy.Y > 6 {
			t.Fatalf("enemy at (%d, %d) is outside the room interior", enemy.X, enemy.Y)
		}
		if enemy.X == 2 && enemy.Y == 2 {
			t.Fatal("enemy should not spawn on the player")
		}
		if !enemy.PlayerDiscovered || !enemy.ShowOnMiniMap || enemy.StatusAilments.Sleep != 0 {
			t.Fatalf("enemy %#v should be awake and discovered", enemy.StatusAilments)
		}
	}
	if len(g.state.Items) < 2 || len(g.state.Items) > 3 {
		t.Fatalf("spawned items = %d, want 2..3", len(g.state.Items))
	}
}

func TestMonsterHouseCardInCorridor(t *testing.T) {
	card := buildItemFromTemplate(35, 0, 0)
	g := &Game{
		state: GameState{
			Map:    makeTestFloorMap(8, 8),
			Player: Player{Entity: Entity{X: 2, Y: 2}, Inventory: []Item{card}},
		},
		rooms: []Room{}, // プレイヤーはどの部屋にもいない
	}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.state.Enemies) != 0 {
		t.Fatalf("enemies = %d, want 0 in corridor", len(g.state.Enemies))
	}
	if len(g.ActionQueue.Queue) < 2 || g.ActionQueue.Queue[1].Message != "しかし何も起こらなかった" {
		t.Fatal("通路で使用した場合は不発メッセージを表示するはず")
	}
}

func TestHasteAllEnemiesCard(t *testing.T) {
	card := buildItemFromTemplate(36, 0, 0)
	g := &Game{
		state: GameState{
			Player:  Player{Inventory: []Item{card}},
			Enemies: []Enemy{{Entity: Entity{X: 1, Y: 1}}, {Entity: Entity{X: 5, Y: 5}}},
		},
	}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	for i, enemy := range g.state.Enemies {
		if enemy.StatusAilments.Haste != enemyHasteCardTurns {
			t.Fatalf("enemy %d haste = %d, want %d", i, enemy.StatusAilments.Haste, enemyHasteCardTurns)
		}
	}
}

func TestForgetFloorMapCard(t *testing.T) {
	card := buildItemFromTemplate(37, 0, 0)
	floorItem := buildItemFromTemplate(1, 1, 1)
	floorItem.SetPlayerDiscovered(true)
	floorItem.SetShowOnMiniMap(true)
	g := &Game{
		state: GameState{
			Map:     [][]Tile{{{Type: "floor", Visited: true}, {Type: "wall", Visited: true}}},
			Player:  Player{Inventory: []Item{card}},
			Enemies: []Enemy{{PlayerDiscovered: true, ShowOnMiniMap: true}},
			Items:   []Item{floorItem},
		},
	}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if g.state.Map[0][0].Visited || g.state.Map[0][1].Visited {
		t.Fatal("all map tiles should be forgotten")
	}
	if g.state.Enemies[0].PlayerDiscovered || g.state.Enemies[0].ShowOnMiniMap {
		t.Fatal("enemy should be hidden from the minimap")
	}
	if floorItem.GetPlayerDiscovered() || floorItem.GetShowOnMiniMap() {
		t.Fatal("floor item should be hidden from the minimap")
	}
	if !g.miniMapDirty {
		t.Fatal("minimap should be marked dirty")
	}
}

func TestPickupBanCard(t *testing.T) {
	card := buildItemFromTemplate(38, 0, 0)
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card}}}}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if !g.pickupBanned {
		t.Fatal("pickupBanned should be set")
	}
}

func TestPickupBannedBlocksPickup(t *testing.T) {
	floorItem := buildItemFromTemplate(1, 2, 2)
	g := &Game{
		state: GameState{
			Player: Player{Entity: Entity{X: 2, Y: 2}},
			Items:  []Item{floorItem},
		},
		pickupBanned: true,
	}

	g.PickupItem()
	if len(g.ActionQueue.Queue) == 0 {
		t.Fatal("拾得禁止中はメッセージを表示するはず")
	}
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.state.Player.Inventory) != 0 {
		t.Fatal("拾得禁止中はアイテムを拾えないはず")
	}
	if len(g.state.Items) != 1 {
		t.Fatal("床のアイテムはそのまま残るはず")
	}
}

func TestBigRoomCard(t *testing.T) {
	card := buildItemFromTemplate(39, 0, 0)
	mapGrid := make([][]Tile, 6)
	for y := range mapGrid {
		mapGrid[y] = make([]Tile, 6)
		for x := range mapGrid[y] {
			mapGrid[y][x] = Tile{Type: "other", Blocked: true, BlockSight: true}
		}
	}
	mapGrid[3][3] = Tile{Type: "stairs"}
	g := &Game{
		state: GameState{
			Map:    mapGrid,
			Player: Player{Entity: Entity{X: 2, Y: 2}, Inventory: []Item{card}},
		},
		rooms: []Room{{ID: 1, X: 1, Y: 1, Width: 3, Height: 3}},
	}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.rooms) != 1 || g.rooms[0].Width != 6 || g.rooms[0].Height != 6 {
		t.Fatalf("rooms = %#v, want one room covering the map", g.rooms)
	}
	if g.state.Map[0][0].Type != "wall" || !g.state.Map[0][0].Blocked {
		t.Fatal("map boundary should be wall")
	}
	if g.state.Map[2][4].Type != "floor" || g.state.Map[2][4].Blocked {
		t.Fatal("map interior should be floor")
	}
	if g.state.Map[3][3].Type != "stairs" {
		t.Fatal("stairs should be preserved")
	}
}

func TestTrapCardIncreasesTraps(t *testing.T) {
	card := buildItemFromTemplate(40, 0, 0)
	g := &Game{
		state: GameState{
			Map:      makeTestFloorMap(8, 8),
			Player:   Player{Entity: Entity{X: 2, Y: 2}, Inventory: []Item{card}},
			MapTraps: []MapTrap{createMapTrapByID(0, 3, 3)},
		},
		rooms: []Room{{ID: 0, X: 0, Y: 0, Width: 8, Height: 8}},
	}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.state.MapTraps) < 6 || len(g.state.MapTraps) > 9 {
		t.Fatalf("traps = %d, want 6..9", len(g.state.MapTraps))
	}
	seen := map[Coordinate]bool{}
	for _, trap := range g.state.MapTraps {
		pos := Coordinate{X: trap.X, Y: trap.Y}
		if seen[pos] {
			t.Fatalf("duplicate trap at (%d, %d)", trap.X, trap.Y)
		}
		seen[pos] = true
		if trap.X == 2 && trap.Y == 2 {
			t.Fatal("trap should not be placed on the player")
		}
	}
	for _, trap := range g.state.MapTraps[1:] {
		if trap.Discovered {
			t.Fatal("added traps should be undiscovered")
		}
	}
}

func TestVacuumSlashCardDamagesOnlyEffectArea(t *testing.T) {
	card := buildItemFromTemplate(31, 0, 0)
	g := &Game{
		state: GameState{
			Player: Player{Entity: Entity{X: 2, Y: 2}, Inventory: []Item{card}, Level: 1},
			Enemies: []Enemy{
				{Entity: Entity{X: 3, Y: 3}, Health: 100, StatusAilments: StatusAilments{Sleep: 5, Paralysis: true}},
				{Entity: Entity{X: 6, Y: 6}, Health: 100},
			},
		},
		rooms: []Room{{ID: 1, X: 0, Y: 0, Width: 5, Height: 5}},
	}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if got := g.state.Enemies[0].Health; got < 76 || got > 88 {
		t.Fatalf("same-room enemy health = %d, want 76..88", got)
	}
	if g.state.Enemies[0].StatusAilments.Sleep != 0 || g.state.Enemies[0].StatusAilments.Paralysis {
		t.Fatal("damage should wake and release the affected enemy")
	}
	if got := g.state.Enemies[1].Health; got != 100 {
		t.Fatalf("outside enemy health = %d, want 100", got)
	}
}

func TestSelfTargetCardTemplates(t *testing.T) {
	cards := map[int]string{
		41: "自爆のカード",
		42: "口封じのカード",
		43: "パワーアップのカード",
		44: "完全回復のカード",
	}
	for id, wantName := range cards {
		card, ok := buildItemFromTemplate(id, 0, 0).(*Card)
		if !ok || card.Name != wantName {
			t.Fatalf("card %d = %#v, want %q", id, card, wantName)
		}
	}
}

func TestSplitEnemiesByExplosion(t *testing.T) {
	enemies := []Enemy{
		{Entity: Entity{X: 3, Y: 3}, Name: "隣接"},
		{Entity: Entity{X: 1, Y: 1}, Name: "斜め隣接"},
		{Entity: Entity{X: 5, Y: 2}, Name: "遠く"},
	}
	survivors, destroyed := splitEnemiesByExplosion(2, 2, enemies)
	if len(survivors) != 1 || survivors[0].Name != "遠く" {
		t.Fatalf("survivors = %#v, want only 遠く", survivors)
	}
	if len(destroyed) != 2 {
		t.Fatalf("destroyed = %#v, want 2 enemies", destroyed)
	}
}

func TestSelfDestructCard(t *testing.T) {
	card := buildItemFromTemplate(41, 0, 0)
	g := &Game{
		state: GameState{
			Player: Player{Entity: Entity{X: 2, Y: 2}, Health: 100, MaxHealth: 100, Inventory: []Item{card}},
			Enemies: []Enemy{
				{Entity: Entity{X: 3, Y: 2}, Health: 100, ExperiencePoints: 10},
				{Entity: Entity{X: 6, Y: 6}, Health: 100},
			},
		},
	}

	card.Use(g)
	for i := 0; i < len(g.ActionQueue.Queue); i++ {
		g.ActionQueue.Queue[i].Execute(g)
	}

	if g.state.Player.Health != 50 {
		t.Fatalf("player health = %d, want 50", g.state.Player.Health)
	}
	if len(g.state.Enemies) != 1 || g.state.Enemies[0].X != 6 {
		t.Fatalf("enemies = %#v, want only the distant enemy", g.state.Enemies)
	}
	if g.state.Player.ExperiencePoints != 0 {
		t.Fatal("消し飛んだ敵の経験値は入らないはず")
	}
	if len(g.state.Player.Inventory) != 0 {
		t.Fatal("card should be consumed")
	}
}

func TestMouthSealCard(t *testing.T) {
	card := buildItemFromTemplate(42, 0, 0)
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card}}}}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if got := g.state.Player.StatusAilments.MouthSeal; got != mouthSealCardTurns {
		t.Fatalf("mouth seal turns = %d, want %d", got, mouthSealCardTurns)
	}

	g.decrementStatusAilments()
	if got := g.state.Player.StatusAilments.MouthSeal; got != mouthSealCardTurns-1 {
		t.Fatalf("mouth seal turns after a turn = %d, want %d", got, mouthSealCardTurns-1)
	}
}

func TestIsMouthItem(t *testing.T) {
	if !isMouthItem(buildItemFromTemplate(1, 0, 0)) || !isMouthItem(buildItemFromTemplate(2, 0, 0)) || !isMouthItem(buildItemFromTemplate(30, 0, 0)) {
		t.Fatal("食料・薬・カードは口封じの対象のはず")
	}
	if isMouthItem(buildItemFromTemplate(9, 0, 0)) || isMouthItem(buildItemFromTemplate(20, 0, 0)) {
		t.Fatal("杖や武器は口封じの対象外のはず")
	}
}

func TestMouthSealBlocksItemUse(t *testing.T) {
	potion := buildItemFromTemplate(2, 0, 0)
	g := &Game{
		state: GameState{
			Player: Player{
				Inventory:      []Item{potion},
				StatusAilments: StatusAilments{MouthSeal: 5},
			},
		},
	}

	g.executeAction()

	if len(g.state.Player.Inventory) != 1 {
		t.Fatal("口封じ中はアイテムが消費されないはず")
	}
	if len(g.ActionQueue.Queue) != 1 || g.ActionQueue.Queue[0].Message != "口が封じられていて使えない" {
		t.Fatalf("queue = %#v, want blocked message", g.ActionQueue.Queue)
	}
	if g.isActioned {
		t.Fatal("口封じで使えなかった場合はターンを消費しないはず")
	}
}

func TestBoostPower(t *testing.T) {
	tests := []struct {
		power, maxPower    int
		wantPower, wantMax int
	}{
		{power: 5, maxPower: 8, wantPower: 8, wantMax: 8},
		{power: 0, maxPower: 8, wantPower: 3, wantMax: 8},
		{power: 8, maxPower: 8, wantPower: 9, wantMax: 9},
	}
	for _, tt := range tests {
		gotPower, gotMax := boostPower(tt.power, tt.maxPower)
		if gotPower != tt.wantPower || gotMax != tt.wantMax {
			t.Errorf("boostPower(%d, %d) = (%d, %d), want (%d, %d)", tt.power, tt.maxPower, gotPower, gotMax, tt.wantPower, tt.wantMax)
		}
	}
}

func TestPowerUpCard(t *testing.T) {
	card := buildItemFromTemplate(43, 0, 0)
	g := &Game{state: GameState{Player: Player{Power: 4, MaxPower: 8, Inventory: []Item{card}}}}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if g.state.Player.Power != 7 || g.state.Player.MaxPower != 8 {
		t.Fatalf("power = %d/%d, want 7/8", g.state.Player.Power, g.state.Player.MaxPower)
	}
}

func TestFullHealResult(t *testing.T) {
	if health, maxHealth := fullHealResult(40, 100); health != 100 || maxHealth != 100 {
		t.Fatalf("fullHealResult(40, 100) = (%d, %d), want (100, 100)", health, maxHealth)
	}
	if health, maxHealth := fullHealResult(100, 100); health != 105 || maxHealth != 105 {
		t.Fatalf("fullHealResult(100, 100) = (%d, %d), want (105, 105)", health, maxHealth)
	}
}

func TestFullHealCardCuresPoison(t *testing.T) {
	card := buildItemFromTemplate(44, 0, 0)
	g := &Game{
		state: GameState{
			Player: Player{
				Health: 40, MaxHealth: 100,
				Inventory:      []Item{card},
				StatusAilments: StatusAilments{Poison: 4},
			},
		},
	}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if g.state.Player.Health != 100 || g.state.Player.MaxHealth != 100 {
		t.Fatalf("health = %d/%d, want 100/100", g.state.Player.Health, g.state.Player.MaxHealth)
	}
	if g.state.Player.StatusAilments.Poison != 0 {
		t.Fatal("毒も治るはず")
	}
}

func TestRustProofCardTemplate(t *testing.T) {
	card, ok := buildItemFromTemplate(45, 0, 0).(*Card)
	if !ok || card.Name != "さび止めのカード" {
		t.Fatalf("card 45 = %#v, want さび止めのカード", card)
	}
}

func TestRollRustTarget(t *testing.T) {
	intnZero := func(int) int { return 0 }
	intnOne := func(int) int { return 1 }
	if got := rollRustTarget(true, true, intnZero); got != rustWeapon {
		t.Fatalf("both equipped with roll 0 = %d, want rustWeapon", got)
	}
	if got := rollRustTarget(true, true, intnOne); got != rustArmor {
		t.Fatalf("both equipped with roll 1 = %d, want rustArmor", got)
	}
	if got := rollRustTarget(true, false, intnZero); got != rustWeapon {
		t.Fatalf("weapon only = %d, want rustWeapon", got)
	}
	if got := rollRustTarget(false, true, intnZero); got != rustArmor {
		t.Fatalf("armor only = %d, want rustArmor", got)
	}
	if got := rollRustTarget(false, false, intnZero); got != rustNone {
		t.Fatalf("nothing equipped = %d, want rustNone", got)
	}
}

func TestRustTrapRustsWeapon(t *testing.T) {
	weapon := &Weapon{BaseItem: BaseItem{Name: "こん棒", Type: "Weapon"}, AttackPower: 2, Sharpness: 1}
	g := &Game{state: GameState{Player: Player{Inventory: []Item{weapon}}}}
	g.state.Player.EquipItem(weapon)
	attackAfterEquip := g.state.Player.AttackPower

	rustTrapEffect(g)
	g.ActionQueue.Queue[0].Execute(g)

	if weapon.Sharpness != 0 {
		t.Fatalf("weapon sharpness = %d, want 0", weapon.Sharpness)
	}
	if g.state.Player.AttackPower != attackAfterEquip-1 {
		t.Fatalf("attack power = %d, want %d", g.state.Player.AttackPower, attackAfterEquip-1)
	}
}

func TestRustTrapRespectsRustProof(t *testing.T) {
	armor := &Armor{BaseItem: BaseItem{Name: "木甲の盾", Type: "Armor"}, DefensePower: 2, Sharpness: 1, RustProof: true}
	g := &Game{state: GameState{Player: Player{Inventory: []Item{armor}}}}
	g.state.Player.EquipItem(armor)
	defenseAfterEquip := g.state.Player.DefensePower

	rustTrapEffect(g)
	g.ActionQueue.Queue[0].Execute(g)

	if armor.Sharpness != 1 || g.state.Player.DefensePower != defenseAfterEquip {
		t.Fatalf("rust-proof armor changed: sharpness %d, defense %d", armor.Sharpness, g.state.Player.DefensePower)
	}
	if len(g.ActionQueue.Queue) < 2 || g.ActionQueue.Queue[1].Message != "しかし木甲の盾は錆びなかった" {
		t.Fatal("さび止め済みの装備は錆びないメッセージを表示するはず")
	}
}

func TestRustTrapWithoutEquipment(t *testing.T) {
	g := &Game{state: GameState{Player: Player{}}}

	rustTrapEffect(g)
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.ActionQueue.Queue) < 2 || g.ActionQueue.Queue[1].Message != "しかし何も起こらなかった" {
		t.Fatal("装備なしでは不発メッセージを表示するはず")
	}
}

func TestRustProofCard(t *testing.T) {
	card := buildItemFromTemplate(45, 0, 0)
	weapon := &Weapon{BaseItem: BaseItem{Name: "こん棒", Type: "Weapon"}, AttackPower: 2}
	armor := &Armor{BaseItem: BaseItem{Name: "木甲の盾", Type: "Armor"}, DefensePower: 2}
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card, weapon, armor}}}}
	g.state.Player.EquipItem(weapon)
	g.state.Player.EquipItem(armor)

	card.Use(g)
	if len(g.state.Player.Inventory) != 2 {
		t.Fatal("used card should be removed from inventory")
	}
	g.ActionQueue.Queue[0].Execute(g)

	if !weapon.RustProof || !armor.RustProof {
		t.Fatal("装備中の武器と盾が錆びなくなるはず")
	}
}

func TestRustProofCardWithoutEquipment(t *testing.T) {
	card := buildItemFromTemplate(45, 0, 0)
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card}}}}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.ActionQueue.Queue) < 2 || g.ActionQueue.Queue[1].Message != "しかし何も起こらなかった" {
		t.Fatal("装備なしで使用した場合は不発メッセージを表示するはず")
	}
}

func TestRustProofSurvivesSaveRoundTrip(t *testing.T) {
	weapon, ok := buildItemFromTemplate(20, 0, 0).(*Weapon)
	if !ok {
		t.Fatal("template 20 should be a weapon")
	}
	weapon.RustProof = true
	restored, err := savedToItem(itemToSaved(weapon))
	if err != nil {
		t.Fatalf("savedToItem error: %v", err)
	}
	if restoredWeapon, ok := restored.(*Weapon); !ok || !restoredWeapon.RustProof {
		t.Fatal("RustProof should survive a save round trip")
	}

	armor, ok := buildItemFromTemplate(23, 0, 0).(*Armor)
	if !ok {
		t.Fatal("template 23 should be an armor")
	}
	armor.RustProof = true
	restored, err = savedToItem(itemToSaved(armor))
	if err != nil {
		t.Fatalf("savedToItem error: %v", err)
	}
	if restoredArmor, ok := restored.(*Armor); !ok || !restoredArmor.RustProof {
		t.Fatal("RustProof should survive a save round trip")
	}
}

func TestFullHealCardAtFullHealth(t *testing.T) {
	card := buildItemFromTemplate(44, 0, 0)
	g := &Game{state: GameState{Player: Player{Health: 100, MaxHealth: 100, Inventory: []Item{card}}}}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if g.state.Player.Health != 105 || g.state.Player.MaxHealth != 105 {
		t.Fatalf("health = %d/%d, want 105/105", g.state.Player.Health, g.state.Player.MaxHealth)
	}
}

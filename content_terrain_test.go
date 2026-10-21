//go:build !test

package main

import (
	"encoding/json"
	"testing"
)

func TestTerrainWaterMovement(t *testing.T) {
	for _, id := range []int{0, 4, 10, 14, 108, 114, 117} {
		g := newProjectileTestGame()
		g.state.Map[4][6] = Tile{Type: "water", Blocked: true}
		e := CreateEnemyByID(id, 5, 4)
		g.state.Enemies = []Enemy{e}
		want := hasEnemyTrait(e.Traits, EnemyTraitFloating) || hasEnemyTrait(e.Traits, EnemyTraitAquatic)
		if got := g.tryMoveEnemy(0, 1, 0); got != want {
			t.Fatalf("id %d movement=%v want=%v", id, got, want)
		}
		if g.terrainOpenAt(6, 4) {
			t.Fatal("player can enter water")
		}
		g.state.Enemies = nil
		g.state.Player.X = 5
		if g.MovePlayer(1, 0) || g.state.Player.X != 5 {
			t.Fatal("player moved into water")
		}
	}
}

func TestTerrainProjectilesAndWaterLanding(t *testing.T) {
	for _, id := range []int{1, 7, 46} {
		g := newProjectileTestGame()
		item := buildItemFromTemplate(id, 0, 0)
		if item == nil {
			t.Fatalf("missing item %d", id)
		}
		g.state.Player.Inventory = []Item{item}
		g.state.Map[4][5] = Tile{Type: "water", Blocked: true}
		g.throwWithCallbacks(item, 1)
		finishThrow(t, g)
		if len(g.state.Items) != 0 {
			t.Fatalf("id %d survived water: %v", id, g.state.Items)
		}
	}
	g := newProjectileTestGame()
	item := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{item}
	g.state.Map[4][5] = Tile{Type: "water", Blocked: true}
	g.throwWithCallbacks(item, 2)
	finishThrow(t, g)
	if len(floorItemsAt(g, 6, 4)) != 1 {
		t.Fatal("throw did not cross water")
	}
}

func TestTerrainCrackedAttackDigAndExplosion(t *testing.T) {
	for _, method := range []string{"attack", "pickaxe", "explosion", "ring", "card"} {
		g := newProjectileTestGame()
		g.state.Map[4][5] = Tile{Type: "cracked_wall", Blocked: true, BlockSight: true}
		switch method {
		case "attack":
			g.CheckForEnemies(1, 0)
			runQueuedActions(g)
		case "pickaxe":
			if !g.enqueueWallDig(1, 0, []EquipmentAbilityID{AbilityDigWall}) {
				t.Fatal("dig rejected")
			}
			runQueuedActions(g)
		case "explosion":
			g.explodeItemsAround(4, 4, 1)
		case "ring":
			explodeRing(g)
			runQueuedActions(g)
		case "card":
			g.state.Player.Inventory = []Item{buildItemFromTemplate(41, 0, 0)}
			g.state.Player.Inventory[0].Use(g)
			runQueuedActions(g)
		}
		if tile := g.state.Map[4][5]; tile.Type != "corridor" || tile.Blocked || tile.BlockSight {
			t.Fatalf("%s: %+v", method, tile)
		}
	}
}

func TestTerrainHiddenDiscovery(t *testing.T) {
	for _, method := range []string{"search", "light", "ring"} {
		g := newProjectileTestGame()
		g.state.Map[4][5] = Tile{Type: "hidden_corridor", Blocked: true, BlockSight: true}
		if !g.tryHiddenEnemyTest(t) {
			t.Fatal("enemy cannot use hidden corridor")
		}
		g.state.Enemies = nil
		switch method {
		case "search":
			// 固定シードで成功と失敗の両方を確認し、空振りも1ターンを使う。
			for i := 0; i < 40 && g.state.Map[4][5].Blocked; i++ {
				g.CheckForEnemies(1, 0)
				runQueuedActions(g)
			}
		case "light":
			card := buildItemFromTemplate(30, 0, 0)
			// あかりの実際の使用アクションを呼ぶ（名称ではなく既存関数）。
			g.state.Player.Inventory = []Item{card}
			g.selectedItemIndex = 0
			card.Use(g)
			runQueuedActions(g)
		case "ring":
			g.state.Player.EquippedAccessories[0] = buildItemFromTemplate(186, 0, 0).(*Accessory)
			g.updateTileBrightness()
		}
		if tile := g.state.Map[4][5]; tile.Type != "corridor" || tile.Blocked || !tile.Visited {
			t.Fatalf("%s: %+v", method, tile)
		}
	}
}

func TestTerrainSearchSuccessAndFailure(t *testing.T) {
	seen := map[bool]bool{}
	for seed := int64(0); seed < 20; seed++ {
		g := newProjectileTestGame()
		g.rng = newTurnRNG(seed)
		want := newTurnRNG(seed).Intn(2) == 0
		g.state.Map[4][5] = Tile{Type: "hidden_corridor", Blocked: true, BlockSight: true}
		g.CheckForEnemies(1, 0)
		if !g.state.Map[4][5].Blocked {
			t.Fatal("effect ran before Execute")
		}
		runQueuedActions(g)
		got := g.state.Map[4][5].Type == "corridor"
		if got != want || !g.isActioned {
			t.Fatalf("seed=%d got=%v want=%v actioned=%v", seed, got, want, g.isActioned)
		}
		seen[got] = true
	}
	if len(seen) != 2 {
		t.Fatal("missing search outcome")
	}
}

func (g *Game) tryHiddenEnemyTest(t *testing.T) bool {
	t.Helper()
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 6, 4)}
	return g.tryMoveEnemy(0, -1, 0)
}

func TestTerrainSaveCompatibility(t *testing.T) {
	for _, terrain := range []string{"floor", "water", "cracked_wall", "hidden_corridor"} {
		g, _ := newHeadlessGame(123, DifficultyBeginner)
		g.state.Map[2][2] = Tile{Type: terrain, Blocked: terrain != "floor", BlockSight: terrain != "water" && terrain != "floor"}
		data, err := json.Marshal(g.buildSaveData())
		if err != nil {
			t.Fatal(err)
		}
		save, err := decodeSaveData(data)
		if err != nil {
			t.Fatal(err)
		}
		restored := &Game{headless: true, fastForward: true}
		if err := restored.applySaveData(save); err != nil {
			t.Fatal(err)
		}
		if restored.state.Map[2][2] != g.state.Map[2][2] {
			t.Fatalf("lost %s", terrain)
		}
	}
}

func TestTerrainKnockbackStopsBeforeWater(t *testing.T) {
	g := newProjectileTestGame()
	g.state.Map[4][6] = Tile{Type: "water", Blocked: true}
	g.knockbackPlayer(1, 0, 4, 0, DamageSource{})
	if g.state.Player.X != 5 {
		t.Fatalf("x=%d", g.state.Player.X)
	}
}

func TestTerrainArrowAndBoltCrossWater(t *testing.T) {
	for _, kind := range []string{"arrow", "bolt"} {
		g := newProjectileTestGame()
		i := addTestEnemy(g, 0, 6, 4)
		g.state.Enemies[i].Health = 99
		g.state.Map[4][5] = Tile{Type: "water", Blocked: true}
		if kind == "arrow" {
			arrow := buildItemFromTemplate(6, 0, 0).(*Arrow)
			arrow.ShotCount = 3
			equip(g, arrow)
			g.shootArrow(arrow, 10)
		} else {
			cane := buildItemFromTemplate(16, 0, 0).(*Cane)
			cane.Uses = 3
			g.state.Player.Inventory = []Item{cane}
			g.useCane(cane)
		}
		finishThrow(t, g)
		if kind == "arrow" && g.state.Enemies[i].Health == 99 {
			t.Fatal("arrow blocked by water")
		}
		if kind == "bolt" && !g.state.Enemies[i].StatusAilments.Seal {
			t.Fatal("bolt blocked by water")
		}
	}
}

func TestTerrainWaterDestroysPotContentsAndDroppedLoot(t *testing.T) {
	g := newProjectileTestGame()
	pot := buildItemFromTemplate(46, 0, 0).(*Pot)
	pot.Contents = append(pot.Contents, buildItemFromTemplate(1, 0, 0))
	g.state.Player.Inventory = []Item{pot}
	g.state.Map[4][5] = Tile{Type: "water", Blocked: true}
	g.throwWithCallbacks(pot, 1)
	finishThrow(t, g)
	if len(g.state.Items) != 0 {
		t.Fatal("pot contents survived water")
	}
	g.state.Enemies = []Enemy{CreateEnemyByID(4, 5, 4)}
	g.state.Enemies[0].HeldItem = buildItemFromTemplate(1, 0, 0)
	g.dropEnemyHeldItem(0)
	if len(g.state.Items) != 0 || g.state.Enemies[0].HeldItem != nil {
		t.Fatal("loot survived water")
	}
}

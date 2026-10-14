//go:build !test

package main

import (
	"encoding/json"
	"image/png"
	"reflect"
	"testing"
)

func newItemTrapTestGame() *Game {
	return &Game{rng: newTurnRNG(23), state: GameState{Map: makeTestFloorMap(7, 7), Player: Player{Entity: Entity{X: 3, Y: 3}, Health: 40, MaxHealth: 40, Satiety: 50, MaxSatiety: 100}}}
}

func TestItemTrapsActivationAndFailure(t *testing.T) {
	for id := 12; id <= 14; id++ {
		for _, fail := range []bool{false, true} {
			g := newItemTrapTestGame()
			weapon := buildItemFromTemplate(4, 0, 0).(*Weapon)
			food := buildItemFromTemplate(1, 0, 0)
			g.state.Player.Inventory = []Item{weapon, food}
			g.state.Player.EquipItem(weapon)
			trap := createMapTrapByID(id, 3, 3)
			trap.FailRate = 0
			if fail {
				trap.FailRate = 100
			}
			g.state.MapTraps = []MapTrap{trap}
			g.stepOnTrap(0)
			if g.state.Player.EquippedWeapon != weapon || g.state.Player.Inventory[1] != food || len(g.state.Items) != 0 {
				t.Fatal("effect was not deferred")
			}
			executeMoveTrapActions(g)
			if !g.state.MapTraps[0].Discovered {
				t.Fatal("trap not discovered")
			}
			if fail {
				if g.state.Player.EquippedWeapon != weapon || g.state.Player.Inventory[1] != food || len(g.state.Items) != 0 {
					t.Fatalf("failed trap %d changed inventory", id)
				}
				continue
			}
			switch id {
			case 12:
				if g.state.Player.EquippedWeapon != nil || len(g.state.Player.Inventory) != 2 || len(g.state.Items) != 0 {
					t.Fatal("unequip must retain inventory")
				}
			case 13:
				if g.state.Player.Inventory[1].GetID() != 71 {
					t.Fatal("food did not spoil")
				}
			case 14:
				if len(g.state.Items) != 1 || g.state.Items[0] != food || g.state.Player.EquippedWeapon != weapon || len(g.state.Player.Inventory) != 1 {
					t.Fatal("scatter included equipped item")
				}
			}
		}
	}
}

func TestUnequipTrapCursedEquipmentAndStats(t *testing.T) {
	for _, id := range []int{4, 5} {
		g := newItemTrapTestGame()
		item := buildItemFromTemplate(id, 0, 0).(Equipable)
		switch v := item.(type) {
		case *Weapon:
			v.Cursed = true
		case *Armor:
			v.Cursed = true
			v.Abilities = []EquipmentAbilityID{AbilityExplosionResistance}
		default:
			t.Fatal("fixture is not weapon/armor")
		}
		g.state.Player.Inventory = []Item{item}
		attack, defense := g.state.Player.AttackPower, g.state.Player.DefensePower
		g.state.Player.EquipItem(item)
		if _, armor := item.(*Armor); armor && g.resolvePlayerShieldDefense(10, EnemyDamageExplosion, EnemyAttackEnvironment).Damage != 5 {
			t.Fatal("shield ability fixture is ineffective")
		}
		g.state.Map = nil // 外すだけなので置き場所がなくても外れる。
		unequipTrapEffect(g)
		executeMoveTrapActions(g)
		if g.state.Player.IsEquipped(item) || !isCursedEquipment(item) || g.state.Player.Inventory[0] != item || len(g.state.Items) != 0 {
			t.Fatal("cursed equipment must unequip without losing curse/item")
		}
		if g.state.Player.AttackPower != attack || g.state.Player.DefensePower != defense {
			t.Fatal("equipment stats not reverted")
		}
		if g.resolvePlayerShieldDefense(10, EnemyDamageExplosion, EnemyAttackEnvironment).Damage != 10 {
			t.Fatal("shield ability remained after unequip")
		}
	}
}

func TestItemTrapsWithoutTargets(t *testing.T) {
	for id := 12; id <= 14; id++ {
		g := newItemTrapTestGame()
		createMapTrapByID(id, 3, 3).Effect(g)
		executeMoveTrapActions(g)
		if len(g.state.Player.Inventory) != 0 || len(g.state.Items) != 0 {
			t.Fatal("empty inventory changed")
		}
	}
}

func TestSpoilTrapProtectsPotContentsAndRottenFood(t *testing.T) {
	g := newItemTrapTestGame()
	pot := buildItemFromTemplate(46, 0, 0).(*Pot)
	food := buildItemFromTemplate(1, 0, 0)
	pot.Contents = []Item{food}
	rotten := buildItemFromTemplate(71, 0, 0)
	g.state.Player.Inventory = []Item{pot, rotten}
	spoilTrapEffect(g)
	executeMoveTrapActions(g)
	if pot.Contents[0] != food || g.state.Player.Inventory[1] != rotten {
		t.Fatal("protected food changed")
	}
}

func TestScatterTrapPlacementAndLimit(t *testing.T) {
	g := newItemTrapTestGame()
	g.state.Map[2][2].Type = "wall"
	g.state.Map[2][3].Type = "stairs"
	g.state.MapTraps = []MapTrap{createMapTrapByID(14, 4, 2)}
	g.state.Items = []Item{buildItemFromTemplate(1, 2, 3)}
	g.state.Enemies = []Enemy{{Entity: Entity{X: 4, Y: 3}}}
	g.state.Map[4][2].Blocked = true
	for i := 0; i < 5; i++ {
		g.state.Player.Inventory = append(g.state.Player.Inventory, buildItemFromTemplate(1, 0, 0))
	}
	allowed := g.itemTrapDropPositions()
	if len(allowed) != 2 {
		t.Fatalf("safe positions = %v", allowed)
	}
	scatterTrapEffect(g)
	executeMoveTrapActions(g)
	if len(g.state.Player.Inventory) != 3 || len(g.state.Items) != 3 {
		t.Fatal("must scatter only placeable items")
	}
	seen := map[Coordinate]bool{}
	for _, item := range g.state.Items[1:] {
		x, y := item.GetPosition()
		p := Coordinate{X: x, Y: y}
		if seen[p] || (p != allowed[0] && p != allowed[1]) {
			t.Fatalf("invalid placement %v", p)
		}
		seen[p] = true
	}
	g = newItemTrapTestGame()
	for i := 0; i < 5; i++ {
		g.state.Player.Inventory = append(g.state.Player.Inventory, buildItemFromTemplate(1, 0, 0))
	}
	scatterTrapEffect(g)
	executeMoveTrapActions(g)
	if len(g.state.Items) != 3 || len(g.state.Player.Inventory) != 2 {
		t.Fatal("maximum scatter count must be 3")
	}
}

func TestScatterTrapInCorridor(t *testing.T) {
	g := newItemTrapTestGame()
	// 敵が通路に仕掛けた罠でも、周囲の通路へ散らばる（扉・壁には落ちない）。
	for y := range g.state.Map {
		for x := range g.state.Map[y] {
			g.state.Map[y][x].Type = "wall"
		}
	}
	g.state.Map[3][2].Type = "corridor"
	g.state.Map[3][3].Type = "corridor"
	g.state.Map[3][4].Type = "door"
	g.state.Player.Inventory = []Item{buildItemFromTemplate(1, 0, 0), buildItemFromTemplate(1, 0, 0)}
	scatterTrapEffect(g)
	executeMoveTrapActions(g)
	if len(g.state.Items) != 1 || len(g.state.Player.Inventory) != 1 {
		t.Fatalf("items/inventory = %d/%d", len(g.state.Items), len(g.state.Player.Inventory))
	}
	if x, y := g.state.Items[0].GetPosition(); x != 2 || y != 3 {
		t.Fatalf("placed at %d,%d", x, y)
	}
}

func TestScatterTrapNoSpaceAndSpecialItems(t *testing.T) {
	g := newItemTrapTestGame()
	pot := buildItemFromTemplate(46, 0, 0).(*Pot)
	pot.Contents = []Item{buildItemFromTemplate(1, 0, 0)}
	card := buildItemFromTemplate(55, 0, 0).(*Card)
	if !isSanctuaryItem(card) {
		t.Fatal("fixture is not sanctuary")
	}
	g.state.Player.Inventory = []Item{pot, card}
	g.state.Map = nil
	scatterTrapEffect(g)
	executeMoveTrapActions(g)
	if len(g.state.Player.Inventory) != 2 || card.Stuck || len(g.state.Items) != 0 {
		t.Fatal("no space must preserve items")
	}
	g.state.Map = makeTestFloorMap(7, 7)
	scatterTrapEffect(g)
	executeMoveTrapActions(g)
	if len(g.state.Player.Inventory) != 0 || len(g.state.Items) != 2 || len(pot.Contents) != 1 || !card.Stuck {
		t.Fatal("pot contents/sanctuary rule")
	}
	// 既に貼りついたカードが不正に所持品にあっても移動しない。
	g.state.Player.Inventory = []Item{card}
	g.state.Items = nil
	scatterTrapEffect(g)
	executeMoveTrapActions(g)
	if len(g.state.Player.Inventory) != 1 || len(g.state.Items) != 0 {
		t.Fatal("stuck card moved")
	}
}

func TestRottenSausageEffect(t *testing.T) {
	for _, ground := range []bool{false, true} {
		for _, satiety := range []int{50, 95, 100} {
			g := newItemTrapTestGame()
			g.state.Player.Satiety = satiety
			g.state.Player.Power, g.state.Player.MaxPower = 8, 8
			food := buildItemFromTemplate(71, 3, 3)
			if ground {
				g.state.Items = []Item{food}
				g.GroundItemActioned = true
			} else {
				g.state.Player.Inventory = []Item{food}
			}
			food.Use(g)
			if g.state.Player.Satiety != satiety || g.state.Player.Power != 8 {
				t.Fatal("food side effects not deferred")
			}
			executeMoveTrapActions(g)
			if g.state.Player.Satiety != min(100, satiety+10) || g.state.Player.MaxSatiety != 100 || g.state.Player.Power != 7 || len(g.state.Items)+len(g.state.Player.Inventory) != 0 {
				t.Fatal("rotten food recovery/power drain/consumption")
			}
		}
	}
	g := newItemTrapTestGame()
	g.state.Player.Inventory = []Item{buildItemFromTemplate(71, 0, 0)}
	g.state.Player.Power = 0
	eatRottenSausage(g)
	executeMoveTrapActions(g)
	if g.state.Player.Power != 0 || g.state.Player.Satiety != 60 {
		t.Fatalf("power 0: power %d satiety %d, want 0/60", g.state.Player.Power, g.state.Player.Satiety)
	}
}

func TestItemTrapsSaveGenerationAndImages(t *testing.T) {
	g, _ := newHeadlessGame(234, DifficultyAdvanced)
	g.state.MapTraps = nil
	for id := 12; id <= 14; id++ {
		trap := createMapTrapByID(id, 2, id)
		trap.Discovered = true
		g.state.MapTraps = append(g.state.MapTraps, trap)
	}
	g.state.Player.Inventory = []Item{buildItemFromTemplate(71, 0, 0)}
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
	for i, trap := range restored.state.MapTraps {
		if trap.TemplateID != i+12 || !trap.Discovered || trap.Effect == nil {
			t.Fatal("trap save round trip")
		}
	}
	if restored.state.Player.Inventory[0].GetID() != 71 {
		t.Fatal("food save round trip")
	}
	seen := map[int]bool{}
	for seed := int64(0); seed < 100; seed++ {
		for _, trap := range generateMapTraps([]Room{{X: 0, Y: 0, Width: 20, Height: 20}}, newFloorRNG(seed, 1), 1) {
			seen[trap.TemplateID] = true
		}
	}
	for id := 12; id <= 14; id++ {
		if !seen[id] {
			t.Fatalf("trap %d not generated", id)
		}
	}
	for _, name := range []string{"unequip_trap", "spoil_trap", "scatter_trap", "rotten_sausage"} {
		f, err := imgFS.Open("img/" + name + ".png")
		if err != nil {
			t.Fatal(err)
		}
		im, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if im.Bounds().Dx() != 30 || im.Bounds().Dy() != 30 {
			t.Fatal("image size")
		}
		for _, p := range []Coordinate{{X: 0, Y: 0}, {X: 29, Y: 0}, {X: 0, Y: 29}, {X: 29, Y: 29}} {
			_, _, _, a := im.At(p.X, p.Y).RGBA()
			if a != 0 {
				t.Fatalf("opaque corner %s", name)
			}
		}
	}
}

func TestItemTrapReproducibility(t *testing.T) {
	for id := 12; id <= 14; id++ {
		games := []*Game{newItemTrapTestGame(), newItemTrapTestGame()}
		for _, g := range games {
			g.state.Player.Inventory = []Item{buildItemFromTemplate(1, 0, 0), buildItemFromTemplate(1, 0, 0), buildItemFromTemplate(71, 0, 0)}
			createMapTrapByID(id, 3, 3).Effect(g)
			executeMoveTrapActions(g)
		}
		if !reflect.DeepEqual(games[0].buildSaveData(), games[1].buildSaveData()) {
			t.Fatalf("trap %d is not reproducible", id)
		}
	}
}

func TestItemTrapEffectsSaveRoundTrip(t *testing.T) {
	for id := 12; id <= 14; id++ {
		g, _ := newHeadlessGame(321, DifficultyAdvanced)
		g.state.Map = makeTestFloorMap(70, 70)
		g.state.Player.X, g.state.Player.Y = 3, 3
		g.state.Enemies, g.state.Items, g.state.MapTraps = nil, nil, nil
		g.state.Player.EquippedWeapon, g.state.Player.EquippedArmor = nil, nil
		g.state.Player.EquippedArrow = nil
		g.state.Player.EquippedAccessories = [2]*Accessory{}
		g.state.Player.Inventory = nil
		weapon := rollItemFromTemplate(4, 0, 0, g.randIntn).(*Weapon)
		weapon.Cursed = true
		pot := rollItemFromTemplate(46, 0, 0, g.randIntn).(*Pot)
		pot.Contents = []Item{g.createItemByID(71, 0, 0)}
		g.state.Player.Inventory = []Item{weapon, g.createItemByID(1, 0, 0), pot, g.createItemByID(55, 0, 0)}
		g.state.Player.EquipItem(weapon)
		createMapTrapByID(id, 3, 3).Effect(g)
		executeMoveTrapActions(g)
		before := g.buildSaveData()
		data, err := json.Marshal(before)
		if err != nil {
			t.Fatal(err)
		}
		save, err := decodeSaveData(data)
		if err != nil {
			t.Fatal(err)
		}
		resumed, _ := newHeadlessGame(9, DifficultyAdvanced)
		if err := resumed.applySaveData(save); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(before, resumed.buildSaveData()) {
			t.Fatalf("trap %d changed state during resume", id)
		}
		if g.randIntn(1000) != resumed.randIntn(1000) {
			t.Fatal("random state not preserved")
		}
	}
}

func TestRottenSausageMenuConsumesTurn(t *testing.T) {
	g, b := newTurnTestGame(t)
	g.state.Player.Inventory = []Item{buildItemFromTemplate(71, 0, 0)}
	g.state.Player.Satiety = 50
	power := g.state.Player.Power
	expectTurns(t, g, 1, func() error { return useOrFail(b, 0, "使う") })
	if len(g.state.Player.Inventory) != 0 || g.state.Player.Satiety != 60 || g.state.Player.Power != max(power-1, 0) {
		t.Fatal("menu food consumption/turn decrement failed")
	}
}

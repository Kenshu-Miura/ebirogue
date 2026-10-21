//go:build !test

package main

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func insertSynthesisTestItem(g *Game, pot *Pot, item Item) {
	g.state.Player.Inventory = []Item{pot, item}
	g.state.Player.MaxInventory = 20
	g.potInsertIndex, g.selectedItemIndex = 0, 1
	g.executePotInsertSelection()
	drainBossActions(g)
}

func TestSynthesisEquipmentAndSuspend(t *testing.T) {
	for _, kind := range []string{"Weapon", "Armor"} {
		t.Run(kind, func(t *testing.T) {
			g, _ := newHeadlessGame(910, DifficultyBeginner)
			pot := buildItemFromTemplate(225, 0, 0).(*Pot)
			id := 128
			if kind == "Armor" {
				id = 150
			}
			base := buildItemFromTemplate(id, 0, 0)
			donor := buildItemFromTemplate(id, 0, 0)
			abilities := []EquipmentAbilityID{AbilityRustImmune, AbilityEvasion, AbilityReflection, AbilityCounter, AbilityTheftResistance}
			switch b := base.(type) {
			case *Weapon:
				b.Sharpness = 2
				d := donor.(*Weapon)
				d.Sharpness = -1
				d.Cursed = true
				d.Abilities = abilities
			case *Armor:
				b.Sharpness = 2
				d := donor.(*Armor)
				d.Sharpness = -1
				d.Cursed = true
				d.Abilities = abilities
			}
			insertSynthesisTestItem(g, pot, base)
			insertSynthesisTestItem(g, pot, donor)
			if len(pot.Contents) != 1 || pot.Contents[0] != base {
				t.Fatal("first item must remain the base")
			}
			saved := itemToSaved(base)
			if saved.Sharpness != 1 || !saved.Cursed {
				t.Fatalf("enhancement/curse: %+v", saved)
			}
			var actual []EquipmentAbilityID
			switch b := base.(type) {
			case *Weapon:
				actual = b.Abilities
			case *Armor:
				actual = b.Abilities
			}
			if len(actual) != 5 {
				t.Fatalf("marks = %v", actual)
			}
			for i, a := range actual {
				if hasEquipmentAbility(actual[:i], a) {
					t.Fatal("duplicate mark")
				}
			}
			if !strings.Contains(base.GetDescription(), "印：") {
				t.Fatal("missing marks in description")
			}
			data, err := json.Marshal(g.buildSaveData())
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := decodeSaveData(data)
			if err != nil {
				t.Fatal(err)
			}
			resumed, _ := newHeadlessGame(2, DifficultyBeginner)
			if err := resumed.applySaveData(decoded); err != nil {
				t.Fatal(err)
			}
			got := resumed.state.Player.Inventory[0].(*Pot).Contents[0]
			if !reflect.DeepEqual(itemToSaved(got), saved) {
				t.Fatal("synthesis lost on suspend")
			}
			old := saved
			old.ExtraAbilities = nil
			legacy, err := savedToItem(old)
			if err != nil {
				t.Fatal(err)
			}
			if len(itemToSaved(legacy).ExtraAbilities) != 0 {
				t.Fatal("old saves must retain template abilities only")
			}
		})
	}
}

func TestSynthesisKindsAndCanes(t *testing.T) {
	g, _ := newHeadlessGame(91, DifficultyBeginner)
	pot := buildItemFromTemplate(225, 0, 0).(*Pot)
	pot.Capacity = 8
	weapon := buildItemFromTemplate(128, 0, 0)
	armor := buildItemFromTemplate(150, 0, 0)
	// Use real cane templates, independent of their numeric assignment.
	var ids []int
	for _, id := range randomItemTemplateIDs {
		if itemTemplates[id].ItemType == "Cane" {
			ids = append(ids, id)
		}
	}
	cane := buildItemFromTemplate(ids[0], 0, 0)
	donor := buildItemFromTemplate(ids[0], 0, 0).(*Cane)
	donor.Uses = 3
	other := buildItemFromTemplate(ids[1], 0, 0)
	for _, item := range []Item{weapon, armor, cane, donor, other} {
		insertSynthesisTestItem(g, pot, item)
	}
	if len(pot.Contents) != 4 || cane.(*Cane).Uses != 8 {
		t.Fatal("must merge only same equipment category / cane ID")
	}
	before := append([]EquipmentAbilityID(nil), weapon.(*Weapon).Abilities...)
	duplicate := buildItemFromTemplate(128, 0, 0)
	insertSynthesisTestItem(g, pot, duplicate)
	if !reflect.DeepEqual(before, weapon.(*Weapon).Abilities) {
		t.Fatal("same abilities stacked")
	}
}

func TestPotInsertionDefersEffectAndChangeIsDeterministic(t *testing.T) {
	var results []SavedItem
	for run := 0; run < 2; run++ {
		g, _ := newHeadlessGame(951, DifficultyBeginner)
		pot := buildItemFromTemplate(228, 0, 0).(*Pot)
		weapon := buildItemFromTemplate(128, 0, 0)
		g.state.Player.Inventory = []Item{pot, weapon}
		g.potInsertIndex, g.selectedItemIndex = 0, 1
		g.executePotInsertSelection()
		if len(pot.Contents) != 0 {
			t.Fatal("effect ran before insertion Action")
		}
		drainBossActions(g)
		results = append(results, itemToSaved(pot.Contents[0]))
	}
	if !reflect.DeepEqual(results[0], results[1]) {
		t.Fatal("same seed must produce the same change")
	}
}

func TestSynthesisTransferredAbilityWorksAndTemplatesRemainUnchanged(t *testing.T) {
	g, _ := newHeadlessGame(952, DifficultyBeginner)
	pot := buildItemFromTemplate(225, 0, 0).(*Pot)
	base := buildItemFromTemplate(126, 0, 0).(*Weapon)
	donor := buildItemFromTemplate(145, 0, 0)
	before := append([]EquipmentAbilityID(nil), itemTemplates[126].Abilities...)
	insertSynthesisTestItem(g, pot, base)
	insertSynthesisTestItem(g, pot, donor)
	g.executePotTakeOut(pot)
	if _, err := g.state.Player.EquipItem(base); err != nil {
		t.Fatal(err)
	}
	if !g.playerHasAbility(AbilityDigWall) || !g.playerHasAbility(AbilityRustImmune) {
		t.Fatal("transferred abilities not active")
	}
	if !slices.Equal(itemTemplates[126].Abilities, before) {
		t.Fatal("synthesis modified global templates")
	}
	for _, template := range itemTemplates {
		if template.ItemType != "Weapon" && template.ItemType != "Armor" {
			continue
		}
		for _, ability := range template.Abilities {
			if synthesisMarkNames[ability] == "" {
				t.Fatalf("missing mark name: %s", ability)
			}
		}
	}
}

func TestBackPotPreservesTheftAndTossImmunities(t *testing.T) {
	g, _ := newHeadlessGame(953, DifficultyBeginner)
	pot := buildItemFromTemplate(231, 0, 0).(*Pot)
	food := buildItemFromTemplate(1, 0, 0)
	pot.Contents = []Item{food}
	g.state.Player.Inventory = []Item{pot}
	g.state.Player.EquippedArmor = &Armor{BaseItem: BaseItem{Name: "守る盾"}, Abilities: []EquipmentAbilityID{AbilityTheftResistance, AbilityTossWard}}
	enemy := &Enemy{Name: "カワハギ"}
	enqueueStealAttack(enemy, g, func(int) int { return 0 })
	drainBossActions(g)
	if enemy.HeldItem != nil || len(pot.Contents) != 1 {
		t.Fatal("theft bypassed immunity")
	}
	if _, ok := g.tossPlayerItem(0, 0, nil, func(int) int { return 0 }); ok || len(pot.Contents) != 1 {
		t.Fatal("toss bypassed immunity")
	}
}

func TestAdditionalPotEffects(t *testing.T) {
	for id := 225; id <= 232; id++ {
		t.Run(itemTemplates[id].Name, func(t *testing.T) {
			g, _ := newHeadlessGame(92, DifficultyBeginner)
			pot := rollItemFromTemplate(id, 0, 0, g.randIntn).(*Pot)
			if pot.Capacity < 3 || pot.Capacity > 5 || pot.Identified {
				t.Fatal("capacity / identification")
			}
			if name, identified := displayItemName(pot); identified || name == pot.Name {
				t.Fatal("missing unidentified name")
			}
			item := buildItemFromTemplate(128, 0, 0).(*Weapon)
			item.Sharpness = 2
			item.Cursed = true
			item.Identified = false
			insertSynthesisTestItem(g, pot, item)
			switch id {
			case 226:
				if !item.Identified {
					t.Fatal("not identified")
				}
			case 227:
				if item.Sharpness != 3 || !item.Cursed {
					t.Fatal("enhancement must preserve curse")
				}
			case 228:
				if pot.Contents[0].GetID() == item.ID || itemTemplates[pot.Contents[0].GetID()].ItemType != "Weapon" {
					t.Fatal("wrong transformation")
				}
			case 229:
				if len(pot.Contents) != 0 {
					t.Fatal("bottomless retained item")
				}
			case 230:
				// 換金の額は店の売値（wave10-B）。金の剣 1500円×強化+2で 2100、呪いで半額 1050、売値はその半分
				if len(pot.Contents) != 0 || g.state.Player.Cash != 525 {
					t.Fatal("cash conversion")
				}
			}
			if id != 229 && id != 230 && len(pot.Contents) != 1 {
				t.Fatal("lost contents")
			}
		})
	}
}

func TestAdditionalPotsCardsBreakAndRefrigeration(t *testing.T) {
	for id := 225; id <= 232; id++ {
		g, _ := newHeadlessGame(93, DifficultyBeginner)
		pot := buildItemFromTemplate(id, 0, 0).(*Pot)
		food := buildItemFromTemplate(1, 0, 0)
		pot.Contents = []Item{food}
		g.state.Player.Inventory = []Item{pot}
		spoilTrapEffect(g)
		drainBossActions(g)
		if pot.Contents[0] != food {
			t.Fatal("pot food spoiled")
		}
		card := buildItemFromTemplate(47, 0, 0)
		g.state.Player.Inventory = append(g.state.Player.Inventory, card)
		g.selectedItemIndex = 1
		capacity := pot.Capacity
		card.Use(g)
		drainBossActions(g)
		if pot.Capacity != capacity+1 {
			t.Fatal("expansion did not work")
		}
		card = buildItemFromTemplate(48, 0, 0)
		g.state.Player.Inventory = append(g.state.Player.Inventory, card)
		g.selectedItemIndex = 1
		card.Use(g)
		drainBossActions(g)
		if len(pot.Contents) != 0 || g.state.Player.Inventory[1] != food {
			t.Fatal("suction did not work")
		}
		pot.Contents = []Item{food}
		g.state.Player.Inventory = []Item{pot}
		g.executePotTakeOut(pot)
		drainBossActions(g)
		if (len(pot.Contents) == 0) != potAllowsTakeOut(pot) {
			t.Fatal("take-out restriction")
		}
		pot.Contents = []Item{food}
		g.state.Map = make([][]Tile, 5)
		for y := range g.state.Map {
			g.state.Map[y] = make([]Tile, 5)
			for x := range g.state.Map[y] {
				g.state.Map[y][x].Type = "floor"
			}
		}
		g.state.Items = nil
		g.state.MapTraps = nil
		g.scatterPotContents(pot, 2, 2)
		if id == 229 {
			if len(g.state.MapTraps) == 0 || len(g.state.Items) != 0 {
				t.Fatal("bottomless traps")
			}
		} else if len(g.state.Items) != 1 {
			t.Fatal("contents did not scatter")
		}
	}
}

func TestBackPotStealAndToss(t *testing.T) {
	for _, empty := range []bool{false, true} {
		for _, toss := range []bool{false, true} {
			g, _ := newHeadlessGame(94, DifficultyBeginner)
			pot := buildItemFromTemplate(231, 0, 0).(*Pot)
			food := buildItemFromTemplate(1, 0, 0)
			if !empty {
				pot.Contents = []Item{food}
			}
			g.state.Player.Inventory = []Item{buildItemFromTemplate(128, 0, 0), pot}
			target := Item(pot)
			if !empty {
				target = food
			}
			if toss {
				got, ok := g.tossPlayerItem(g.state.Player.X, g.state.Player.Y, nil, func(int) int { return 0 })
				if !ok || got != target {
					t.Fatal("wrong toss target")
				}
			} else {
				enemy := &Enemy{Name: "カワハギ"}
				enqueueStealAttack(enemy, g, func(int) int { return 0 })
				drainBossActions(g)
				if enemy.HeldItem != target || !enemy.Fleeing {
					t.Fatal("wrong theft target")
				}
			}
			if len(pot.Contents) != 0 {
				t.Fatal("target not removed")
			}
			if _, exists := inventoryItemIndex(&g.state.Player, pot); exists == empty {
				t.Fatal("wrong back pot removal")
			}
		}
	}
}

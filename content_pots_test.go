//go:build !test

package main

// 壺と壺に関わるカードのコンテンツテスト。

import (
	"fmt"
	"testing"
)

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
	if g.stats.PotInsertIDs[46] != 1 { // 壺の ID ごとに入れた回数を数える（wave11-0）
		t.Fatalf("pot insert IDs = %+v", g.stats.PotInsertIDs)
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

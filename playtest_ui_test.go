//go:build !test

package main

import "testing"

func TestPlaytestPreviewActions(t *testing.T) {
	for _, item := range []Item{&Pot{}, &Weapon{Abilities: []EquipmentAbilityID{AbilityEdible}}, &Arrow{}, &Cane{}, &Food{}} {
		g := newFoundationTestGame()
		g.state.Player.Inventory = []Item{item}
		g.showItemActions = true
		for i, action := range g.currentItemActions() {
			g.selectedActionIndex = i
			r, wall, active := g.trajectoryPreviewSpec()
			_, cane := item.(*Cane)
			want := action == "投げる" || action == "撃つ" || (action == "使う" && cane)
			if active != want {
				t.Errorf("%T %s active=%v want %v", item, action, active, want)
			}
			if active && ((cane && action == "使う" && (r != 30 || !wall)) || (!(cane && action == "使う") && (r != 10 || wall))) {
				t.Error("wrong range")
			}
		}
		g.showItemActions = false
		g.ShowGroundItem = true
		g.currentGroundItem = item
		for i, action := range g.currentGroundItemActions() {
			g.selectedGroundActionIndex = i
			_, _, active := g.trajectoryPreviewSpec()
			_, cane := item.(*Cane)
			want := action == "投げる" || (action == "使う" && cane)
			if active != want {
				t.Errorf("ground %T %s active=%v", item, action, active)
			}
		}
	}
}

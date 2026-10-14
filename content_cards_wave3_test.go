//go:build !test

package main

import (
	"reflect"
	"strings"
	"testing"
)

func useWave3TestCard(g *Game, id int) string {
	g.GroundItemActioned = false
	g.state.Player.Inventory = append(g.state.Player.Inventory, buildItemFromTemplate(id, 0, 0))
	g.selectedItemIndex = len(g.state.Player.Inventory) - 1
	g.state.Player.Inventory[g.selectedItemIndex].(*Card).Use(g)
	return runQueuedActions(g)
}

func wave3Press(t *testing.T, g *Game, key Key) {
	t.Helper()
	v := NewVirtualInput()
	g.input = v
	v.Press(key)
	v.Update()
	if err := g.handleInventoryInput(); err != nil {
		t.Fatal(err)
	}
}

func TestWave3CardTemplatesAndSave(t *testing.T) {
	names := []string{"ウインナー化のカード", "呪いのカード", "罠消しのカード", "敵集合のカード", "道具寄せのカード", "バクチのカード", "困った時のカード"}
	roles := []itemRole{roleNone, roleHarmful, roleBuff, roleHarmful, roleBuff, roleNone, roleHeal}
	for i, name := range names {
		id := 167 + i
		template, ok := itemTemplates[id]
		if !ok || template.Name != name || template.ItemType != "Card" || template.Type != "Card" || template.Char != 'C' || template.UseActions["UseCard"] == nil {
			t.Fatalf("template %d: %+v", id, template)
		}
		role, ok := itemRoles[id]
		if !ok || role != roles[i] {
			t.Fatalf("role %d", id)
		}
		card := buildItemFromTemplate(id, 2, 3)
		restored, err := savedToItem(itemToSaved(card))
		if err != nil || restored.GetID() != id || restored.GetName() != name {
			t.Fatalf("restore %d: %v", id, err)
		}
		g := newFoundationTestGame()
		g.customNames = map[int]string{id: "任意名"}
		label, unidentified := g.inventoryItemLabel(card)
		// カードは既存カードと同じく識別対象外。名前入力も既存どおり適用されない。
		if label != name || unidentified {
			t.Fatalf("card label %q %v", label, unidentified)
		}
		if _, ok := card.(Identifiable); ok {
			t.Fatal("card unexpectedly identifiable")
		}
		g.state.Player.Inventory = []Item{card}
		identifyAllInventory(g)
		if label, _ := g.inventoryItemLabel(card); label != name {
			t.Fatal(label)
		}
	}
	// 174（ゾワゾワのカード）は段3の wave5-C で追加済み。内容は content_items_wave5c_test.go が検査する。
	if itemCategorySpawnWeights["Card"] != 31 {
		t.Fatal("category weight changed")
	}
}

func TestWave3SausageSelection(t *testing.T) {
	for _, cardFirst := range []bool{false, true} {
		g := newFoundationTestGame()
		card := buildItemFromTemplate(167, 0, 0)
		target := testWeapon()
		other := buildItemFromTemplate(2, 0, 0)
		g.state.Player.Inventory = []Item{target, card, other}
		cardIndex, targetIndex := 1, 0
		if cardFirst {
			g.state.Player.Inventory = []Item{card, target, other}
			cardIndex, targetIndex = 0, 1
		}
		g.selectedItemIndex = cardIndex
		g.executeAction()
		if g.itemTargetEffect == nil || !g.showInventory || g.isActioned {
			t.Fatal("selection must wait without a turn")
		}
		g.selectedItemIndex = targetIndex
		wave3Press(t, g, KeyZ)
		if g.state.Player.Inventory[0] != target {
			t.Fatal("effect ran before Action")
		}
		runQueuedActions(g)
		if len(g.state.Player.Inventory) != 2 || g.state.Player.Inventory[0].GetID() != 1 || g.state.Player.Inventory[1] != other || !g.isActioned {
			t.Fatal("wrong target or card consumption")
		}
	}
	for _, ground := range []bool{false, true} {
		g := newFoundationTestGame()
		weapon := testWeapon()
		equip(g, weapon)
		card := buildItemFromTemplate(167, 4, 4)
		if ground {
			g.state.Items = []Item{card}
			g.GroundItemActioned = true
			g.selectedGroundActionIndex = 2
			g.executeGroundItemAction()
		} else {
			g.state.Player.Inventory = append(g.state.Player.Inventory, card)
			g.selectedItemIndex = 1
			g.executeAction()
		}
		if g.itemTargetEffect == nil || g.isActioned {
			t.Fatal("ground/inventory selection start")
		}
		g.selectedItemIndex = 0
		wave3Press(t, g, KeyZ)
		if !strings.Contains(runQueuedActions(g), "装備中の道具は変えられない") || g.itemTargetEffect == nil || weapon.Cursed {
			t.Fatal("equipped item must be rejected")
		}
		wave3Press(t, g, KeyS)
		wave3Press(t, g, KeyX)
		if g.itemTargetEffect != nil || g.isActioned || g.GroundItemActioned {
			t.Fatal("cancel consumed a turn")
		}
		if ground && (len(g.state.Items) != 1 || g.state.Items[0] != card) {
			t.Fatal("ground card lost")
		}
		if !ground && (len(g.state.Player.Inventory) != 2 || g.state.Player.Inventory[1] != card) {
			t.Fatal("inventory card lost")
		}
	}
	g := newFoundationTestGame()
	card := buildItemFromTemplate(167, 4, 4)
	pot := &Pot{Contents: []Item{buildItemFromTemplate(1, 0, 0)}}
	g.state.Items = []Item{card}
	g.state.Player.Inventory = []Item{pot}
	g.GroundItemActioned = true
	g.selectedGroundActionIndex = 2
	g.executeGroundItemAction()
	wave3Press(t, g, KeyZ)
	runQueuedActions(g)
	if len(g.state.Items) != 0 || g.state.Player.Inventory[0].GetID() != 1 {
		t.Fatal("ground transformation or pot contents")
	}
	// カード自身は選べず、確定前にCで閉じても消費しない。
	g = newFoundationTestGame()
	g.state.Player.Inventory = []Item{buildItemFromTemplate(167, 0, 0)}
	sausageCard(g)
	wave3Press(t, g, KeyZ)
	runQueuedActions(g)
	if g.itemTargetEffect == nil || len(g.state.Player.Inventory) != 1 {
		t.Fatal("self was transformed")
	}
	wave3Press(t, g, KeyC)
	if g.itemTargetEffect != nil || g.isActioned || len(g.state.Player.Inventory) != 1 {
		t.Fatal("C cancellation")
	}
}

func TestWave3CurseTargets(t *testing.T) {
	for _, id := range []int{20, 66, 5, 10, 1} {
		g := newFoundationTestGame()
		target := buildItemFromTemplate(id, 0, 0)
		if e, ok := target.(Equipable); ok {
			switch v := e.(type) {
			case *Weapon:
				v.Cursed = false
			case *Armor:
				v.Cursed = false
			case *Arrow:
				v.Cursed = false
			case *Accessory:
				v.Cursed = false
			}
		}
		if id == 10 {
			equip(g, target)
			if g.state.Player.Power != 11 {
				t.Fatal("ring bonus")
			}
		} else {
			g.state.Player.Inventory = []Item{target}
		}
		g.state.Player.Inventory = append(g.state.Player.Inventory, buildItemFromTemplate(168, 0, 0))
		g.selectedItemIndex = 1
		curseCard(g)
		g.itemTargetEffect(g, 0)
		if id != 1 && !isCurseableItem(target) {
			t.Fatal("curse applied before Action")
		}
		message := runQueuedActions(g)
		if id == 1 {
			if !strings.Contains(message, "何も起こらなかった") {
				t.Fatal(message)
			}
		} else if isCurseableItem(target) {
			t.Fatal("target not cursed")
		}
		if id == 10 && (g.state.Player.Power != 5 || g.state.Player.MaxPower != 5) {
			t.Fatal("equipped ring bonus not reversed")
		}
	}
	g := newFoundationTestGame()
	withTestImmunity(t, "wave3_curse_guard", StatusCurse)
	equip(g, testRing("wave3_curse_guard"))
	target := testWeapon()
	g.state.Player.Inventory = append(g.state.Player.Inventory, target, buildItemFromTemplate(168, 0, 0))
	g.selectedItemIndex = 2
	curseCard(g)
	g.itemTargetEffect(g, 1)
	if message := runQueuedActions(g); target.Cursed || !strings.Contains(message, "防いだ") {
		t.Fatal(message)
	}
}

func TestWave3EraseTraps(t *testing.T) {
	for _, n := range []int{0, 3} {
		g := newFoundationTestGame()
		g.state.MapTraps = make([]MapTrap, n)
		message := useWave3TestCard(g, 169)
		want := "罠が3個消えた"
		if n == 0 {
			want = "何も起こらなかった"
		}
		if len(g.state.MapTraps) != 0 || !g.miniMapDirty || !strings.Contains(message, want) {
			t.Fatal(message)
		}
	}
}

func TestWave3GatherEnemies(t *testing.T) {
	g := newFoundationTestGame()
	g.state.Map[3][3] = Tile{Type: "wall", Blocked: true}
	g.state.Map[3][4] = Tile{Type: "stairs"}
	far := CreateEnemyByID(0, 8, 8)
	near := CreateEnemyByID(0, 6, 4)
	boss := CreateEnemyByID(42, 1, 1)
	far.Animating, far.dx, far.dy = true, 1, 1
	near.Animating, near.dx, near.dy = true, 1, 1
	g.state.Enemies = []Enemy{far, near, boss}
	useWave3TestCard(g, 170)
	if g.state.Enemies[1].X != 5 || g.state.Enemies[1].Y != 3 || g.state.Enemies[0].X != 3 || g.state.Enemies[0].Y != 4 {
		t.Fatalf("wrong order: %+v", g.state.Enemies)
	}
	if g.state.Enemies[2].X != boss.X || g.state.Enemies[2].Y != boss.Y || g.state.Enemies[2].ID != boss.ID {
		t.Fatal("boss moved")
	}
	for _, e := range g.state.Enemies[:2] {
		if e.Animating || e.dx != 0 || e.dy != 0 {
			t.Fatal("animation not reset")
		}
	}
	// 空き床が1つしかなければ、近い敵だけ動く。元の敵の位置は空き床に含めない。
	g = newFoundationTestGame()
	for y := range g.state.Map {
		for x := range g.state.Map[y] {
			g.state.Map[y][x] = Tile{Type: "wall", Blocked: true}
		}
	}
	g.state.Map[3][3] = Tile{Type: "corridor"}
	g.state.Enemies = []Enemy{far, near}
	useWave3TestCard(g, 170)
	if g.state.Enemies[1].X != 3 || g.state.Enemies[1].Y != 3 || g.state.Enemies[0].X != far.X || g.state.Enemies[0].Y != far.Y {
		t.Fatal("insufficient space")
	}
}

func TestWave3GatherItems(t *testing.T) {
	g := newFoundationTestGame()
	far := buildItemFromTemplate(1, 8, 8)
	near := buildItemFromTemplate(2, 6, 4)
	foot := buildItemFromTemplate(3, 4, 4)
	stuck := buildItemFromTemplate(55, 0, 0).(*Card)
	stuck.Stuck = true
	g.state.Items = []Item{far, near, foot, stuck}
	g.state.MapTraps = []MapTrap{{X: 3, Y: 3}}
	useWave3TestCard(g, 171)
	x, y := near.GetPosition()
	if x != 4 || y != 3 {
		t.Fatal("near item destination")
	}
	x, y = far.GetPosition()
	if x != 5 || y != 3 {
		t.Fatal("far item destination")
	}
	x, y = foot.GetPosition()
	if x != 4 || y != 4 {
		t.Fatal("foot item moved")
	}
	x, y = stuck.GetPosition()
	if x != 0 || y != 0 {
		t.Fatal("stuck card moved")
	}
	if !g.miniMapDirty {
		t.Fatal("minimap")
	}
	g = newFoundationTestGame()
	for y := range g.state.Map {
		for x := range g.state.Map[y] {
			g.state.Map[y][x] = Tile{Type: "wall", Blocked: true}
		}
	}
	g.state.Map[3][3] = Tile{Type: "floor"}
	far.SetPosition(8, 8)
	near.SetPosition(6, 4)
	g.state.Items = []Item{far, near}
	useWave3TestCard(g, 171)
	x, y = near.GetPosition()
	if x != 3 || y != 3 {
		t.Fatal("one space: near item")
	}
	x, y = far.GetPosition()
	if x != 8 || y != 8 {
		t.Fatal("one space: far moved")
	}
}

func TestWave3Gamble(t *testing.T) {
	for _, tc := range []struct {
		cash, roll, want int
		won              bool
	}{{50, 0, 200, true}, {50, 1, 0, false}, {0, 0, 100, true}} {
		got, won := rollGamble(tc.cash, func(n int) int {
			if n != 2 {
				t.Fatal(n)
			}
			return tc.roll
		})
		if got != tc.want || won != tc.won {
			t.Fatalf("%d %v", got, won)
		}
	}
	g := newFoundationTestGame()
	g.state.Player.Cash = 50
	before, _ := g.rng.MarshalState()
	message := useWave3TestCard(g, 172)
	after, _ := g.rng.MarshalState()
	if reflect.DeepEqual(after, before) || (!strings.Contains(message, "勝った") && !strings.Contains(message, "負けて")) {
		t.Fatal("gamble Action did not roll")
	}
}

func TestWave3TroublePriority(t *testing.T) {
	g := newFoundationTestGame()
	p := &g.state.Player
	weapon := testWeapon()
	weapon.Cursed = true
	p.Inventory = []Item{weapon}
	p.Health = 1
	p.Satiety = 0
	p.Power = 2
	p.StatusAilments = StatusAilments{Confusion: 3, Sleep: 3, Blind: 3, Slow: 3, MouthSeal: 3, BearTrap: 3, Paralysis: true, Seal: true, Haste: 7, HasteOnWake: true}
	for _, want := range []troubleKind{troubleStatus, troubleHunger, troubleCurse, troublePower, troubleNone} {
		if got := troubleToFix(p.StatusAilments, p.Satiety, playerHasCursedBelongings(p), p.Power, p.MaxPower); got != want {
			t.Fatalf("priority %v want %v", got, want)
		}
		useWave3TestCard(g, 173)
		if p.Health != 40 || p.MaxHealth != 40 || p.StatusAilments.Haste != 7 {
			t.Fatal("health/max/haste")
		}
		switch want {
		case troubleStatus:
			if !reflect.DeepEqual(p.StatusAilments, StatusAilments{Haste: 7}) || p.Satiety != 0 || !weapon.Cursed || p.Power != 2 {
				t.Fatal("status priority")
			}
		case troubleHunger:
			if p.Satiety != 50 || !weapon.Cursed || p.Power != 2 {
				t.Fatal("hunger priority")
			}
		case troubleCurse:
			if weapon.Cursed || p.Power != 2 {
				t.Fatal("curse priority")
			}
		case troublePower:
			if p.Power != p.MaxPower {
				t.Fatal("power priority")
			}
		}
	}
}

func TestWave3BuffConditions(t *testing.T) {
	g := newFoundationTestGame()
	b := &bot{g: g}
	if b.expertBuffUsable(169) || b.expertBuffUsable(171) {
		t.Fatal("empty floor")
	}
	g.state.MapTraps = []MapTrap{{Discovered: false}}
	if b.expertBuffUsable(169) {
		t.Fatal("hidden trap")
	}
	g.state.MapTraps[0].Discovered = true
	if !b.expertBuffUsable(169) {
		t.Fatal("discovered trap")
	}
	g.state.Items = []Item{buildItemFromTemplate(1, 7, 4)}
	if b.expertBuffUsable(171) {
		t.Fatal("distance 3")
	}
	g.state.Items[0].SetPosition(8, 4)
	if !b.expertBuffUsable(171) {
		t.Fatal("distance 4")
	}
}

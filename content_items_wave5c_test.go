//go:build !test

package main

import (
	"reflect"
	"strings"
	"testing"
)

var wave5CIDs = []int{147, 163, 164, 165, 166, 174, 189, 190, 191, 192, 193, 200, 201, 202, 204}

func TestWave5CTemplatesAndSave(t *testing.T) {
	expected := []struct {
		id        int
		name      string
		power     int
		abilities []EquipmentAbilityID
	}{
		{147, "巨大ハリセン", 4, []EquipmentAbilityID{AbilityGiantFan}},
		{163, "痛恨よけの盾", 5, []EquipmentAbilityID{AbilityCriticalGuard}},
		{164, "いたずらよけの盾", 4, []EquipmentAbilityID{AbilityTossWard}},
		{165, "不動の盾", 4, []EquipmentAbilityID{AbilityImmovable}},
		{166, "ざぶとん", 5, []EquipmentAbilityID{AbilityBreakFall}},
		{174, "ゾワゾワのカード", 0, nil},
		{189, "遠投の指輪", 0, []EquipmentAbilityID{AbilityLongThrow}},
		{190, "ヘタ投げの指輪", 0, []EquipmentAbilityID{AbilityPoorThrow, AbilityAlwaysCursed}},
		{191, "百発百中の指輪", 0, []EquipmentAbilityID{AbilitySureThrow}},
		{192, "反射の指輪", 0, []EquipmentAbilityID{AbilityBounceThrow}},
		{193, "連射の指輪", 0, []EquipmentAbilityID{AbilityRapidShot}},
		{200, "弾きよけの指輪", 0, []EquipmentAbilityID{AbilityKnockOffWard}},
		{201, "痛恨の指輪", 0, []EquipmentAbilityID{AbilityCriticalCurse, AbilityAlwaysCursed}},
		{202, "諸刃の指輪", 0, []EquipmentAbilityID{AbilityDoubleEdge}},
		{204, "不動の指輪", 0, []EquipmentAbilityID{AbilityImmovable}},
	}
	for _, tc := range expected {
		t.Run(tc.name, func(t *testing.T) {
			template, ok := itemTemplates[tc.id]
			if !ok || template.Name != tc.name || template.AttackPower+template.DefensePower != tc.power || !reflect.DeepEqual(template.Abilities, tc.abilities) {
				t.Fatalf("template: %+v", template)
			}
			if template.ItemType == "Weapon" || template.ItemType == "Armor" {
				path := specialWeaponImagePaths[template.Type]
				if template.ItemType == "Armor" {
					path = specialArmorImagePaths[template.Type]
				}
				if _, err := imgFS.ReadFile(path); err != nil {
					t.Fatal(err)
				}
			}
			g, _ := newHeadlessGame(7, DifficultyAdvanced)
			item := buildItemFromTemplate(tc.id, 0, 0)
			if ring, ok := item.(*Accessory); ok {
				if ring.Cursed != (tc.id == 190 || tc.id == 201) {
					t.Fatal("initial curse")
				}
			}
			switch v := item.(type) {
			case *Weapon:
				v.Sharpness = 2
			case *Armor:
				v.Sharpness = 2
			}
			equip(g, item)
			addTestEnemy(g, 0, 5, 4)
			g.state.Enemies[len(g.state.Enemies)-1].StatusAilments.Backward = 13
			save := g.buildSaveData()
			restored, _ := newHeadlessGame(8, DifficultyAdvanced)
			if err := restored.applySaveData(save); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(save, restored.buildSaveData()) {
				t.Fatal("save changed after restore")
			}
			found := false
			for _, got := range restored.state.Player.Inventory {
				if got.GetID() == tc.id {
					found = true
					switch v := got.(type) {
					case *Weapon:
						if !reflect.DeepEqual(v.Abilities, tc.abilities) || v.Sharpness != 2 || restored.state.Player.EquippedWeapon != v {
							t.Fatal("weapon restore")
						}
					case *Armor:
						if !reflect.DeepEqual(v.Abilities, tc.abilities) || v.Sharpness != 2 || restored.state.Player.EquippedArmor != v {
							t.Fatal("armor restore")
						}
					case *Accessory:
						if !reflect.DeepEqual(v.Abilities, tc.abilities) {
							t.Fatal("ring abilities")
						}
					}
				}
			}
			if !found {
				t.Fatal("missing item")
			}
		})
	}
	for _, title := range []string{"指輪：飛び道具・痛恨", "道具：追加の盾・武器・カード"} {
		found := false
		for _, page := range helpPages(nil) {
			if page.Title == title {
				found = true
				if len(page.Lines) > helpPageSize {
					t.Fatal("help overflow")
				}
			}
		}
		if !found {
			t.Fatal(title)
		}
	}
	if itemRoles[174] != roleEscape {
		t.Fatal("BOT role")
	}
}

func wave5CEquip(g *Game, ids ...int) {
	for _, id := range ids {
		equip(g, buildItemFromTemplate(id, 0, 0))
	}
}

func TestWave5CProjectileKinds(t *testing.T) {
	for _, id := range []int{189, 190, 191, 192, 193} {
		for _, kind := range []ProjectileKind{ProjectileThrow, ProjectileShot, ProjectileBolt} {
			g := newProjectileTestGame()
			wave5CEquip(g, id, id)
			p := g.newPlayerProjectile(buildItemFromTemplate(1, 0, 0), kind, 10)
			physical := kind != ProjectileBolt
			if p.PierceWalls != (id == 189 && physical) || p.PierceTargets != (id == 189 && physical) || p.SureHit != (id == 191 && physical) {
				t.Fatalf("id %d kind %d: %+v", id, kind, p)
			}
			wantHit := projectileBaseHitPercent(kind)
			if id == 190 && physical {
				wantHit = 0
			}
			if p.HitPercent != wantHit {
				t.Fatal("hit percent")
			}
			wantBounce := 0
			if id == 192 && physical {
				wantBounce = 6
			}
			if p.Bounces != wantBounce {
				t.Fatal("bounce stacks")
			}
			wantVolley := 1
			if id == 193 && kind == ProjectileShot {
				wantVolley = 3
			}
			if p.Volley != wantVolley {
				t.Fatal("volley kinds")
			}
		}
	}
}

func TestWave5CThrowHitAndMiss(t *testing.T) {
	for _, tc := range []struct {
		ids  []int
		roll int
		hit  bool
	}{{nil, 99, false}, {[]int{190}, 0, false}, {[]int{191}, 99, true}, {[]int{190, 191}, 99, true}, {[]int{191, 190}, 99, true}} {
		g := newProjectileTestGame()
		wave5CEquip(g, tc.ids...)
		i := addTestEnemy(g, 0, 6, 4)
		g.state.Enemies[i].Health = 20
		g.randHooks.projectile = func(int) int { return tc.roll }
		food, _ := throwTestFood(t, g)
		if (g.state.Enemies[i].Health < 20) != tc.hit {
			t.Fatalf("ids %v hit mismatch", tc.ids)
		}
		if !tc.hit {
			if items := floorItemsAt(g, 7, 4); len(items) != 1 || items[0] != food {
				t.Fatal("miss landing")
			}
		}
	}
	g := newProjectileTestGame()
	wave5CEquip(g, 191)
	p := g.newPlayerProjectile(buildItemFromTemplate(1, 0, 0), ProjectileThrow, 10)
	p.BeforeHit = append(p.BeforeHit, func(_ *Game, _ *Projectile, h *ProjectileHit) { h.HitPercent = 0 })
	i := addTestEnemy(g, 0, 5, 4)
	if !g.projectileHitsEnemy(&p, i) {
		t.Fatal("sure hit override")
	}
}

func TestWave5CLongThrowAndBounce(t *testing.T) {
	g := newProjectileTestGame()
	g.state.Map = makeTestFloorMap(20, 9)
	g.state.Map[4][6] = Tile{Type: "wall", Blocked: true}
	wave5CEquip(g, 189)
	throwTestFood(t, g)
	if len(floorItemsAt(g, 14, 4)) != 1 {
		t.Fatal("pierce wall landing")
	}
	g = newProjectileTestGame()
	wave5CEquip(g, 189)
	for _, x := range []int{5, 6} {
		i := addTestEnemy(g, 0, x, 4)
		g.state.Enemies[i].Health = 20
	}
	throwTestFood(t, g)
	for _, e := range g.state.Enemies {
		if e.Health != 17 {
			t.Fatal("pierce targets")
		}
	}
	if len(g.state.Items) != 0 {
		t.Fatal("hit item survives")
	}
	g = newProjectileTestGame()
	wave5CEquip(g, 192)
	i := addTestEnemy(g, 0, 2, 4)
	g.state.Enemies[i].Health = 20
	throwTestFood(t, g)
	if g.state.Enemies[i].Health != 17 || g.state.Player.Health != 40 {
		t.Fatal("bounce hit")
	}
	for _, rings := range []int{1, 2} {
		g = newProjectileTestGame()
		for j := 0; j < rings; j++ {
			wave5CEquip(g, 192)
		}
		p := g.newPlayerProjectile(buildItemFromTemplate(1, 0, 0), ProjectileThrow, 100)
		g.flyProjectile(&p)
		if p.Bounces != 0 || p.Traveled() >= 100 || g.state.Player.Health != 40 {
			t.Fatalf("bounce cap: %+v", p)
		}
	}
	g = newProjectileTestGame()
	wave5CEquip(g, 189, 192)
	i = addTestEnemy(g, 0, 2, 4)
	g.state.Enemies[i].Health = 20
	throwTestFood(t, g)
	if g.state.Enemies[i].Health > 17 {
		t.Fatal("bounce before pierce")
	}
}

func TestWave5CShots(t *testing.T) {
	for _, tc := range []struct {
		ids           []int
		arrows, shots int
	}{{nil, 5, 1}, {[]int{193}, 5, 2}, {[]int{193, 193}, 5, 3}, {[]int{193, 193}, 1, 1}, {[]int{190}, 5, 1}} {
		g := newProjectileTestGame()
		wave5CEquip(g, tc.ids...)
		i := addTestEnemy(g, 0, 6, 4)
		g.state.Enemies[i].Health = 999
		arrow := buildItemFromTemplate(6, 0, 0).(*Arrow)
		arrow.ShotCount = tc.arrows
		equip(g, arrow)
		p := g.state.Player
		damage := rollPlayerAttackDamage(playerAttackTotal(p.AttackPower, p.Power, p.Level)+arrow.AttackPower, g.state.Enemies[i].DefensePower, 1, g.damageIntn()).Damage
		g.shootArrow(arrow, 10)
		finishThrow(t, g)
		want := damage * tc.shots
		if len(tc.ids) > 0 && tc.ids[0] == 190 {
			want = 0
		}
		if 999-g.state.Enemies[i].Health != want || arrow.ShotCount != tc.arrows-tc.shots {
			t.Fatal("shot damage/count")
		}
	}
}

func TestWave5CProtection(t *testing.T) {
	for _, id := range []int{0, 164, 200} {
		g := newFoundationTestGame()
		if id != 0 {
			wave5CEquip(g, id)
		}
		armor := testWeapon()
		equip(g, armor)
		g.state.Player.Inventory = append(g.state.Player.Inventory, buildItemFromTemplate(1, 0, 0))
		changed := g.knockOffPlayerEquipment(armor, 1, 0, 3)
		msg := runQueuedActions(g)
		if changed == (id == 200) || (id == 200 && !strings.Contains(msg, "防いだ")) {
			t.Fatal("knock ward")
		}
		_, changed = g.tossPlayerItem(4, 4, nil, func(int) int { return 0 })
		msg = runQueuedActions(g)
		if changed == (id == 164) || (id == 164 && !strings.Contains(msg, "防いだ")) {
			t.Fatal("toss ward")
		}
	}
	for _, id := range []int{0, 165, 166, 204} {
		g := newFoundationTestGame()
		if id != 0 {
			wave5CEquip(g, id)
		}
		g.knockbackPlayer(1, 0, 10, 5, DamageSource{})
		immovable := id == 165 || id == 204
		if (g.state.Player.X == 4) != immovable {
			t.Fatal("immovable position")
		}
		want := 35
		if id != 0 {
			want = 40
		}
		if g.state.Player.Health != want {
			t.Fatal("collision protection")
		}
	}
	for _, critical := range []bool{false, true} {
		for _, damage := range []int{1, 2, 9} {
			g := newFoundationTestGame()
			wave5CEquip(g, 163)
			in := IncomingDamage{Damage: damage, Critical: critical}
			g.runModifyIncomingDamageHooks(&in)
			want := damage
			if critical {
				want = max(1, damage/3)
			}
			if in.Damage != want {
				t.Fatal("critical guard")
			}
		}
	}
}

func TestWave5CCriticalRings(t *testing.T) {
	for _, id := range []int{201, 202} {
		for _, stacks := range []int{1, 2} {
			for _, roll := range []int{10*stacks - 1, 10 * stacks} {
				g := newFoundationTestGame()
				for j := 0; j < stacks; j++ {
					wave5CEquip(g, id)
				}
				g.randHooks.enemyCritical = func(int) int { return roll }
				addTestEnemy(g, 0, 5, 4)
				g.enqueueEnemyNormalAttack(0)
				msg := runQueuedActions(g)
				if strings.Contains(msg, "痛恨の一撃") != (roll < 10*stacks) {
					t.Fatal("critical boundary")
				}
				a := PlayerAttack{CriticalDenominator: 20}
				g.runBeforePlayerAttackHooks(&a)
				want := 20
				if id == 202 {
					want = 8 / stacks
				}
				if a.CriticalDenominator != want {
					t.Fatal("player critical denominator")
				}
			}
		}
	}
}

func TestWave5CGiantFan(t *testing.T) {
	for _, tc := range []struct {
		roll, id                  int
		defeated, equipped, moves bool
	}{{24, 0, false, true, true}, {25, 0, false, true, false}, {24, 0, true, true, false}, {24, 42, false, true, false}, {24, 0, false, false, false}} {
		g := newFoundationTestGame()
		g.state.Map = makeTestFloorMap(20, 9)
		if tc.equipped {
			wave5CEquip(g, 147)
		}
		i := addTestEnemy(g, tc.id, 5, 4)
		g.state.Enemies[i].Health = 100
		g.randHooks.ability = func(int) int { return tc.roll }
		e := g.state.Enemies[i]
		g.runAfterPlayerHitHooks(PlayerAttackHit{EnemyID: e.ID, Enemy: e, Defeated: tc.defeated})
		if (g.state.Enemies[i].X != 5) != tc.moves {
			t.Fatal("fan boundary")
		}
	}
	g := newFoundationTestGame()
	wave5CEquip(g, 147)
	i := addTestEnemy(g, 0, 5, 4)
	g.state.Enemies[i].Health = 100
	g.randHooks.ability = func(int) int { return 24 }
	e := g.state.Enemies[i]
	g.runAfterPlayerHitHooks(PlayerAttackHit{EnemyID: e.ID, Enemy: e})
	if g.state.Enemies[i].Health != 95 {
		t.Fatal("fan collision damage")
	}
}

func TestWave5CBackwardCard(t *testing.T) {
	for _, floor := range []bool{false, true} {
		g := newFoundationTestGame()
		card := buildItemFromTemplate(174, 4, 4)
		g.showInventory = !floor
		g.GroundItemActioned = floor
		if floor {
			g.state.Items = []Item{card}
		} else {
			g.state.Player.Inventory = []Item{card}
		}
		near := addTestEnemy(g, 0, 5, 4)
		far := addTestEnemy(g, 0, 6, 4)
		g.state.Enemies[far].Health = 100
		boss := addTestEnemy(g, 42, 3, 3)
		backwardCard(g)
		if g.state.Enemies[near].StatusAilments.Backward != 0 {
			t.Fatal("effect should be deferred")
		}
		msg := runQueuedActions(g)
		if g.state.Enemies[near].StatusAilments.Backward != 20 || g.state.Enemies[far].StatusAilments.Backward != 0 || g.state.Enemies[boss].StatusAilments.Backward != 20 || !strings.Contains(msg, "ゾワゾワした") {
			t.Fatal("card radius")
		}
		if len(g.state.Items)+len(g.state.Player.Inventory) != 0 {
			t.Fatal("card consumption")
		}
		g.AttackFromEnemy(near)
		runQueuedActions(g)
		if g.state.Player.Health != 40 || g.state.Enemies[far].Health >= 100 {
			t.Fatal("backward attack")
		}
	}
	g := newFoundationTestGame()
	g.showInventory = true
	g.state.Player.Inventory = []Item{buildItemFromTemplate(174, 0, 0)}
	backwardCard(g)
	if !strings.Contains(runQueuedActions(g), "何も起こらなかった") {
		t.Fatal("empty radius")
	}
}

func TestWave5CProjectileRemovalAndBolt(t *testing.T) {
	g := newProjectileTestGame()
	wave5CEquip(g, 191)
	g.state.Player.EquippedAccessories = [2]*Accessory{}
	g.randHooks.projectile = func(int) int { return 99 }
	i := addTestEnemy(g, 0, 6, 4)
	g.state.Enemies[i].Health = 20
	throwTestFood(t, g)
	if g.state.Enemies[i].Health != 20 {
		t.Fatal("removed ring still active")
	}
	g = newProjectileTestGame()
	wave5CEquip(g, 189)
	g.state.Map[4][5] = Tile{Type: "wall", Blocked: true}
	i = addTestEnemy(g, 0, 6, 4)
	cane := buildItemFromTemplate(16, 0, 0).(*Cane)
	cane.Uses = 3
	g.state.Player.Inventory = append(g.state.Player.Inventory, cane)
	g.useCane(cane)
	finishThrow(t, g)
	if g.state.Enemies[i].StatusAilments.Seal || cane.Uses != 2 {
		t.Fatal("long throw must not modify bolt")
	}
	g = newFoundationTestGame()
	wave5CEquip(g, 163)
	g.state.Player.EquippedArmor = nil
	in := IncomingDamage{Damage: 9, Critical: true}
	g.runModifyIncomingDamageHooks(&in)
	if in.Damage != 9 {
		t.Fatal("removed shield")
	}
	g = newFoundationTestGame()
	wave5CEquip(g, 202)
	g.state.Player.EquippedAccessories = [2]*Accessory{}
	a := PlayerAttack{CriticalDenominator: 20}
	g.runBeforePlayerAttackHooks(&a)
	if a.CriticalDenominator != 20 {
		t.Fatal("removed double edge")
	}
}

func TestWave5CGenerationAndIdentification(t *testing.T) {
	for _, id := range wave5CIDs {
		for _, roll := range []int{0, 1, 7, 99} {
			item := rollItemFromTemplate(id, 0, 0, func(n int) int { return roll % n })
			if ring, ok := item.(*Accessory); ok {
				if ring.Cursed != (id == 190 || id == 201) || ring.Identified {
					t.Fatal("ring generation")
				}
			}
		}
		g := newFoundationTestGame()
		item := buildItemFromTemplate(id, 0, 0)
		g.state.Player.Inventory = []Item{item}
		g.customNames = map[int]string{id: "試験名"}
		if identifiable, ok := item.(Identifiable); ok {
			identifiable.SetIdentified(false)
			label, _ := g.inventoryItemLabel(item)
			if !strings.Contains(label, "試験名") {
				t.Fatalf("custom label %d %s", id, label)
			}
		}
		identifyAllInventory(g)
		label, _ := g.inventoryItemLabel(item)
		if !strings.Contains(label, item.GetName()) {
			t.Fatalf("identified label %d %s", id, label)
		}
	}
}

func TestWave5CBackwardAllNeighborsAndBlindness(t *testing.T) {
	g := newFoundationTestGame()
	g.state.Player.StatusAilments.Blind = 10
	g.state.Player.Inventory = []Item{buildItemFromTemplate(174, 0, 0)}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx != 0 || dy != 0 {
				addTestEnemy(g, 0, 4+dx, 4+dy)
			}
		}
	}
	backwardCard(g)
	msg := runQueuedActions(g)
	for _, e := range g.state.Enemies {
		if e.StatusAilments.Backward != 20 {
			t.Fatal("missing neighbor")
		}
		if strings.Contains(msg, e.Name+"は") {
			t.Fatal("blindness reveals enemy")
		}
	}
}

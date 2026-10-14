//go:build !test

package main

import (
	"reflect"
	"strings"
	"testing"
)

func wave4RingGame(ids ...int) *Game {
	g := newFoundationTestGame()
	g.Floor = 1
	g.rooms = []Room{{ID: 0, X: 1, Y: 1, Width: 7, Height: 7}}
	g.randHooks.ability = func(n int) int { return n - 1 }
	g.randHooks.displacement = func(int) int { return 0 }
	for _, id := range ids {
		equip(g, buildItemFromTemplate(id, 0, 0))
	}
	return g
}

func TestWave4RingTemplatesAndSave(t *testing.T) {
	cases := []struct {
		id      int
		name    string
		ability EquipmentAbilityID
		cursed  bool
	}{
		{175, "回復の指輪", "regeneration", false}, {176, "胃拡張の指輪", "stomach_expand", false}, {177, "胃縮小の指輪", "stomach_shrink", true},
		{178, "しあわせの指輪", "happiness", false}, {179, "毒消しの指輪", "poison_ward", false}, {180, "混乱よけの指輪", "confusion_ward", false},
		{181, "睡眠よけの指輪", "sleep_ward", false}, {182, "錆よけの指輪", "rust_ward", false}, {183, "呪いよけの指輪", "curse_ward", false},
		{184, "気配察知の指輪", "sense_enemies", false}, {185, "道具感知の指輪", "sense_items", false}, {186, "透視の指輪", "clairvoyance", false},
		{187, "鑑定師の指輪", "appraiser", false}, {188, "浮遊の指輪", "levitation", false}, {194, "高飛びの指輪", "high_jump", false},
		{195, "爆発の指輪", "explosion_ring", false}, {196, "垂れ流しの指輪", "leak_items", true}, {197, "小銭垂れ流しの指輪", "leak_money", true},
		{198, "魔物呼びの指輪", "monster_call", false}, {199, "罠増しの指輪", "trap_increase", false},
	}
	// 段3（wave5-C）で 189〜193・200〜202・204 が加わったので、数ではなく段2の20種がそろっていることを下で検査する。
	for _, tc := range cases {
		if _, ok := ringItemTemplates[tc.id]; !ok {
			t.Fatal("missing ring", tc.id)
		}
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := []EquipmentAbilityID{tc.ability}
			if tc.cursed {
				want = append(want, AbilityAlwaysCursed)
			}
			template, ok := itemTemplates[tc.id]
			if !ok || template.Name != tc.name || template.Type != "Accessory" || template.ItemType != "Accessory" || template.Char != '!' || !reflect.DeepEqual(template.Abilities, want) || template.UseActions["AccessoryEffect"] == nil {
				t.Fatalf("template %+v", template)
			}
			for _, roll := range []int{0, 1, 7, 99} {
				ring := rollItemFromTemplate(tc.id, 0, 0, func(n int) int { return roll % n }).(*Accessory)
				if ring.Cursed != tc.cursed || ring.Identified {
					t.Fatalf("generated %+v", ring)
				}
			}
			g := wave4RingGame(tc.id)
			save := g.buildSaveData()
			loaded := wave4RingGame()
			if err := loaded.applySaveData(save); err != nil {
				t.Fatal(err)
			}
			ring := loaded.state.Player.EquippedAccessories[0]
			if ring == nil || !reflect.DeepEqual(ring.Abilities, want) || ring.Cursed != tc.cursed || loaded.state.Player.MaxSatiety != g.state.Player.MaxSatiety {
				t.Fatalf("restore %+v", loaded.state.Player)
			}
			if got := loaded.playerAbilityCount(tc.ability); got != 1 {
				t.Fatal("restored ability", got)
			}
		})
	}
}

func TestWave4RingRecoveryAndHappiness(t *testing.T) {
	for _, tc := range []struct {
		ids                                        []int
		satiety, move, health, wantHP, wantSatiety int
	}{
		{nil, 50, 0, 30, 31, 50}, {[]int{175}, 50, 0, 30, 32, 50}, {[]int{175, 175}, 50, 0, 30, 33, 50},
		{[]int{175}, 0, 0, 30, 29, 0}, {[]int{175}, 50, 19, 30, 32, 48}, {[]int{175, 175}, 50, 19, 30, 33, 47},
		{[]int{175}, 1, 19, 30, 31, 0}, {[]int{175}, 50, 0, 40, 40, 50},
	} {
		g := wave4RingGame(tc.ids...)
		g.moveCount = tc.move
		g.state.Player.Health, g.state.Player.Satiety = tc.health, tc.satiety
		g.IncrementMoveCount()
		if g.state.Player.Health != tc.wantHP || g.state.Player.Satiety != tc.wantSatiety {
			t.Fatalf("%+v: hp %d satiety %d", tc, g.state.Player.Health, g.state.Player.Satiety)
		}
	}
	for stacks := 0; stacks <= 2; stacks++ {
		g := wave4RingGame()
		for i := 0; i < stacks; i++ {
			equip(g, buildItemFromTemplate(178, 0, 0))
		}
		g.state.Player.ExperiencePoints = expForLevel(2) - 1
		g.IncrementMoveCount()
		if g.state.Player.ExperiencePoints != expForLevel(2)-1+stacks || (stacks > 0 && g.state.Player.Level != 2) {
			t.Fatalf("happiness %+v", g.state.Player)
		}
	}
	g := wave4RingGame(178)
	g.state.Player.UnequipItem(g.state.Player.EquippedAccessories[0])
	g.IncrementMoveCount()
	if g.state.Player.ExperiencePoints != 0 {
		t.Fatal("removed happiness")
	}
}

func TestWave4StomachEquipment(t *testing.T) {
	for _, id := range []int{176, 177} {
		g := wave4RingGame()
		g.state.Player.Satiety = 100
		ring := buildItemFromTemplate(id, 0, 0).(*Accessory)
		equip(g, ring)
		want := 100 + accessorySatietyBonus(ring.Abilities)
		if g.state.Player.MaxSatiety != want || g.state.Player.Satiety != min(100, want) {
			t.Fatalf("equip %+v", g.state.Player)
		}
		if id == 177 {
			if !isCursedEquipment(ring) {
				t.Fatal("shrink not cursed")
			}
			g.selectedItemIndex, g.selectedActionIndex = 0, 0
			g.executeAction()
			runQueuedActions(g)
			if g.state.Player.EquippedAccessories[0] != ring {
				t.Fatal("removed cursed ring")
			}
			uncursePlayerBelongings(&g.state.Player)
		}
		g.state.Player.UnequipItem(ring)
		if g.state.Player.MaxSatiety != 100 {
			t.Fatal("maximum not restored")
		}
	}
	g := wave4RingGame(176, 176)
	if g.state.Player.MaxSatiety != 150 {
		t.Fatal("expand stacks")
	}
	g = wave4RingGame(177, 177)
	if g.state.Player.MaxSatiety != 50 {
		t.Fatal("shrink stacks")
	}
	g = wave4RingGame(176, 177)
	if g.state.Player.MaxSatiety != 100 {
		t.Fatal("opposite rings")
	}
	g.state.Player.MaxSatiety = 10
	buildItemFromTemplate(177, 0, 0).(*Accessory).UpdatePlayerStats(&g.state.Player, true)
	if g.state.Player.MaxSatiety != 1 {
		t.Fatal("lower bound")
	}
	// 悟りは既存仕様どおり現在満腹度だけを1へ制限する。
	g = wave4RingGame(176)
	equip(g, buildItemFromTemplate(159, 0, 0))
	g.capSatietyForSatori()
	if g.state.Player.Satiety != 1 || g.state.Player.MaxSatiety != 125 {
		t.Fatal("satori interaction")
	}
}

func TestWave4RingImmunities(t *testing.T) {
	for _, tc := range []struct {
		id   int
		kind StatusKind
	}{{179, StatusPoison}, {180, StatusConfusion}, {181, StatusSleep}, {182, StatusRust}, {183, StatusCurse}} {
		g := wave4RingGame(tc.id)
		for _, kind := range []StatusKind{StatusPoison, StatusConfusion, StatusSleep, StatusRust, StatusCurse, StatusBlind} {
			if _, blocked := g.playerStatusGuard(kind); blocked != (kind == tc.kind) {
				t.Fatalf("%d guards %s", tc.id, kind)
			}
		}
		if msg, ok := g.playerImmunityMessage(tc.kind); !ok || !strings.Contains(msg, itemTemplates[tc.id].Name) {
			t.Fatal("immunity message", msg)
		}
		g.state.Player.UnequipItem(g.state.Player.EquippedAccessories[0])
		if g.playerImmuneTo(tc.kind) {
			t.Fatal("unequipped immunity")
		}
	}
	g := wave4RingGame(179)
	if g.drainPlayerPower(1) != 0 || g.state.Player.Power != 8 {
		t.Fatal("poison ward")
	}
	for _, protected := range []bool{false, true} {
		id := 10
		if protected {
			id = 180
		}
		g = wave4RingGame(id)
		g.state.Player.Inventory = append(g.state.Player.Inventory, buildItemFromTemplate(18, 0, 0))
		g.selectedItemIndex = 1
		confusionPotion(g)
		runQueuedActions(g)
		if (g.state.Player.StatusAilments.Confusion == 0) != protected {
			t.Fatal("confusion ward")
		}
		id = 10
		if protected {
			id = 181
		}
		g = wave4RingGame(id)
		sleepGasTrapEffect(g)
		runQueuedActions(g)
		if (g.state.Player.StatusAilments.Sleep == 0) != protected {
			t.Fatal("sleep ward")
		}
		id = 10
		if protected {
			id = 182
		}
		g = wave4RingGame(id)
		weapon := testWeapon()
		equip(g, weapon)
		rustTrapEffect(g)
		runQueuedActions(g)
		if (weapon.Sharpness == 0) != protected {
			t.Fatal("rust ward")
		}
		id = 10
		if protected {
			id = 183
		}
		g = wave4RingGame(id)
		e := CreateEnemyByID(9, 5, 4)
		e.SpecialAttack(&e, g)
		runQueuedActions(g)
		cursed := false
		for _, item := range g.state.Player.Inventory {
			if equipment, ok := item.(Equipable); ok && isCursedEquipment(equipment) {
				cursed = true
			}
		}
		if cursed == protected {
			t.Fatal("curse ward")
		}
		id = 10
		if protected {
			id = 179
		}
		g = wave4RingGame(id)
		e = CreateEnemyByID(1, 5, 4)
		e.SpecialAttack(&e, g)
		runQueuedActions(g)
		if (g.state.Player.Power == 8) != protected {
			t.Fatal("snake poison ward")
		}
		stepOnPoisonArrowTrap(g, 0)
		if (g.state.Player.Power == 8) != protected {
			t.Fatal("arrow poison ward")
		}
	}
}

func TestWave4RingVisibility(t *testing.T) {
	for _, id := range []int{10, 184, 185, 186} {
		g := wave4RingGame(id)
		g.rooms = []Room{{ID: 0, X: 1, Y: 1, Width: 4, Height: 4}, {ID: 1, X: 6, Y: 6, Width: 2, Height: 2}}
		g.state.Enemies = []Enemy{CreateEnemyByID(0, 7, 7), CreateEnemyByID(0, 7, 6)}
		g.state.Enemies[1].Disguise = EnemyDisguiseItem
		g.state.Enemies[1].Revealed = false
		item := buildItemFromTemplate(9, 7, 7)
		g.state.Items = []Item{item}
		g.updateEnemyVisibility()
		g.updateItemVisibility()
		if g.state.Enemies[0].ShowOnMiniMap != (id == 184 || id == 186) || g.state.Enemies[1].ShowOnMiniMap || item.GetShowOnMiniMap() != (id == 185 || id == 186) || item.GetPlayerDiscovered() {
			t.Fatalf("visibility id %d", id)
		}
		g.state.Player.StatusAilments.Blind = 2
		g.updateEnemyVisibility()
		g.updateItemVisibility()
		if g.state.Enemies[0].ShowOnMiniMap || item.GetShowOnMiniMap() {
			t.Fatal("blind")
		}
		g.state.Player.StatusAilments.Blind = 0
		g.updateEnemyVisibility()
		g.updateItemVisibility()
		g.miniMapDirty = false
		g.state.Player.UnequipItem(g.state.Player.EquippedAccessories[0])
		g.updateEnemyVisibility()
		g.updateItemVisibility()
		if g.state.Enemies[0].ShowOnMiniMap || item.GetShowOnMiniMap() {
			t.Fatal("removed sensing")
		}
		if id != 10 && !g.miniMapDirty {
			t.Fatal("removed sensing did not invalidate minimap")
		}
	}
}

func TestWave4RingIdentificationAndUncurseSave(t *testing.T) {
	for _, id := range sortedTemplateIDs(ringItemTemplates, nil) {
		g := wave4RingGame(id)
		ring := g.state.Player.EquippedAccessories[0]
		if _, unknown := g.inventoryItemLabel(ring); !unknown {
			t.Fatal("ring identified by equipping", id)
		}
		g.customNames = map[int]string{id: "試した指輪"}
		if label, unknown := g.inventoryItemLabel(ring); label != "試した指輪" || !unknown {
			t.Fatal("custom name", id)
		}
		identifyAllInventory(g)
		if label, unknown := g.inventoryItemLabel(ring); !strings.Contains(label, itemTemplates[id].Name) || unknown {
			t.Fatal("identify ring", id)
		}
		uncursePlayerBelongings(&g.state.Player)
		loaded := wave4RingGame()
		if err := loaded.applySaveData(g.buildSaveData()); err != nil {
			t.Fatal(err)
		}
		if r := loaded.state.Player.EquippedAccessories[0]; r.Cursed || !r.Identified {
			t.Fatal("uncursed/identified save", id)
		}
	}
	g := wave4RingGame(187)
	pot := buildItemFromTemplate(46, 4, 4).(*Pot)
	pot.SetIdentified(false)
	inside := buildItemFromTemplate(9, 0, 0).(*Cane)
	pot.Contents = []Item{inside}
	g.state.Items = []Item{pot}
	g.PickUpItem(pot, 0)
	if !pot.IsIdentified() || inside.IsIdentified() {
		t.Fatal("appraiser pot contents")
	}
	money := buildItemFromTemplate(0, 4, 4).(*Money)
	money.SetIdentified(false)
	g.state.Items = []Item{money}
	g.PickUpItem(money, 0)
	if money.IsIdentified() {
		t.Fatal("appraiser money")
	}
}

func TestWave4AppraiserAndLevitation(t *testing.T) {
	for _, id := range []int{10, 187} {
		g := wave4RingGame(id)
		cane := buildItemFromTemplate(9, 4, 4).(*Cane)
		g.state.Items = []Item{cane}
		g.PickUpItem(cane, 0)
		if cane.IsIdentified() != (id == 187) {
			t.Fatal("appraiser", id)
		}
		if id == 187 && !strings.Contains(runQueuedActions(g), "だった。") {
			t.Fatal("identify message")
		}
	}
	for _, id := range []int{10, 188} {
		g := wave4RingGame(id)
		triggered := false
		trap := createMapTrapByID(1, 4, 4)
		trap.FailRate = 0
		trap.Effect = func(*Game) { triggered = true }
		g.state.MapTraps = []MapTrap{trap}
		g.checkForTrapAtPosition(4, 4)
		msg := runQueuedActions(g)
		if triggered != (id != 188) || !g.state.MapTraps[0].Discovered {
			t.Fatal("levitation", id)
		}
		if id == 188 {
			if !strings.Contains(msg, "浮いて") {
				t.Fatal("float message")
			}
			g.checkForTrapAtPosition(4, 4)
			if len(g.ActionQueue.Queue) != 0 {
				t.Fatal("known trap message")
			}
			g.state.Player.UnequipItem(g.state.Player.EquippedAccessories[0])
			g.checkForTrapAtPosition(4, 4)
			runQueuedActions(g)
			if !triggered {
				t.Fatal("removed levitation")
			}
		}
	}
}

func TestWave4RingRandomTurns(t *testing.T) {
	for _, id := range []int{194, 195, 196, 197} {
		for _, stacks := range []int{0, 1, 2} {
			threshold, denominator := map[int]int{194: 2, 195: 1, 196: 3, 197: 10}[id], 100
			if id == 195 {
				denominator = 200
			}
			for _, hit := range []bool{false, true} {
				g := wave4RingGame()
				for i := 0; i < stacks; i++ {
					equip(g, buildItemFromTemplate(id, 0, 0))
				}
				roll := threshold * stacks
				if hit && stacks > 0 {
					roll--
				}
				g.randHooks.ability = func(n int) int {
					if n != denominator {
						t.Fatalf("%d probability denominator %d", id, n)
					}
					return roll
				}
				g.state.Player.Cash = 100
				item := buildItemFromTemplate(9, 0, 0)
				g.state.Player.Inventory = append(g.state.Player.Inventory, item)
				g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 4), CreateEnemyByID(0, 6, 4), CreateEnemyByID(42, 3, 4)}
				g.state.Enemies[2].Health, g.state.Enemies[2].MaxHealth = 200, 200
				g.runTurnHooks()
				if g.state.Player.X != 4 || g.state.Player.Health != 40 || len(g.state.Player.Inventory) != stacks+1 || g.state.Player.Cash != 100 {
					t.Fatal("effects not deferred")
				}
				runQueuedActions(g)
				trigger := hit && stacks > 0
				switch id {
				case 194:
					if (g.state.Player.X != 4 || g.state.Player.Y != 4) != trigger {
						t.Fatalf("jump stacks %d hit %v", stacks, hit)
					}
				case 195:
					wantHP, wantEnemies := 40, 3
					if trigger {
						wantHP, wantEnemies = 1, 2
					}
					if g.state.Player.Health != wantHP || len(g.state.Enemies) != wantEnemies {
						t.Fatal("explosion result")
					}
					if trigger && (g.state.Enemies[0].X != 6 || g.state.Enemies[1].Health != 150 || g.state.Player.ExperiencePoints == 0) {
						t.Fatal("explosion range/boss/experience")
					}
				case 196:
					if (len(g.state.Items) == 1) != trigger {
						t.Fatal("leak item result")
					}
					if trigger {
						if g.state.Items[0] != item {
							t.Fatal("equipped item leaked")
						}
						x, y := item.GetPosition()
						if x != 5 || y != 4 {
							t.Fatal("not forward", x, y)
						}
					}
				case 197:
					want := 100
					if trigger {
						want = 98
					}
					if g.state.Player.Cash != want {
						t.Fatal("cash leak")
					}
					if trigger {
						m := g.state.Items[0].(*Money)
						if m.Amount != 2 || m.X != 4 || m.Y != 4 {
							t.Fatal("money position/value")
						}
					}
				}
			}
		}
	}
	g := wave4RingGame(195)
	g.randHooks.ability = func(int) int { return 0 }
	g.state.Player.Health = 1
	g.runTurnHooks()
	runQueuedActions(g)
	if g.state.Player.Health != 1 || g.playerDead {
		t.Fatal("explosion HP1")
	}
	for _, cash := range []int{0, 1} {
		g = wave4RingGame(197)
		g.state.Player.Cash = cash
		g.randHooks.ability = func(int) int { return 0 }
		g.runTurnHooks()
		runQueuedActions(g)
		if g.state.Player.Cash != 0 || len(g.state.Items) != cash {
			t.Fatal("cash boundary")
		}
	}
	g = wave4RingGame(197)
	g.randHooks.ability = func(int) int { return 0 }
	g.state.Player.Cash = 100
	g.state.Items = []Item{buildItemFromTemplate(9, 4, 4)}
	g.runTurnHooks()
	runQueuedActions(g)
	if g.state.Player.Cash != 100 || len(g.state.Items) != 1 {
		t.Fatal("occupied cash drop")
	}
	// 外した指輪のフックは呼ばれない。
	for _, id := range []int{194, 195, 196, 197} {
		g = wave4RingGame(id)
		g.state.Player.Cash = 100
		g.state.Player.UnequipItem(g.state.Player.EquippedAccessories[0])
		g.randHooks.ability = func(int) int { t.Fatal("removed ring rolled"); return 0 }
		g.runTurnHooks()
		if len(g.ActionQueue.Queue) != 0 {
			t.Fatal("removed ring action")
		}
	}
}

func TestWave4MonsterCallAndTrapIncrease(t *testing.T) {
	for stacks := 0; stacks <= 2; stacks++ {
		g, control := wave4RingGame(), wave4RingGame()
		for i := 0; i < stacks; i++ {
			equip(g, buildItemFromTemplate(198, 0, 0))
		}
		g.SetNextSpawnInterval()
		control.SetNextSpawnInterval()
		if g.spawnInterval != ringSpawnInterval(control.spawnInterval, stacks) {
			t.Fatal("spawn interval")
		}
		a, _ := g.random().MarshalState()
		b, _ := control.random().MarshalState()
		if !reflect.DeepEqual(a, b) {
			t.Fatal("changed spawn RNG consumption")
		}
		for i := 0; i < stacks; i++ {
			g.state.Player.UnequipItem(g.state.Player.EquippedAccessories[i])
		}
		g.SetNextSpawnInterval()
		control.SetNextSpawnInterval()
		if g.spawnInterval != control.spawnInterval {
			t.Fatal("removed monster call")
		}
	}
	for _, floor := range []int{1, 10, 30} {
		for _, count := range []int{0, 1, 2} {
			for _, move := range []int{49, 50} {
				g := wave4RingGame()
				g.Floor, g.moveCount = floor, move
				g.randHooks.ability = func(int) int { return 0 }
				g.state.Items = []Item{buildItemFromTemplate(9, 1, 1)}
				g.state.Map[1][2].Type = "stairs"
				for i := 0; i < count; i++ {
					equip(g, buildItemFromTemplate(199, 0, 0))
				}
				g.runTurnHooks()
				if len(g.state.MapTraps) != 0 {
					t.Fatal("trap addition not deferred")
				}
				runQueuedActions(g)
				want := 0
				if move == 50 {
					want = count
				}
				if len(g.state.MapTraps) != want {
					t.Fatalf("trap count %d want %d", len(g.state.MapTraps), want)
				}
				for i, trap := range g.state.MapTraps {
					if trap.Discovered || (trap.X == 4 && trap.Y == 4) || (trap.Y == 1 && (trap.X == 1 || trap.X == 2)) || g.state.Map[trap.Y][trap.X].Type != "floor" || ((floor == 10 || floor == 30) && trap.TemplateID == pitfallTrapID) {
						t.Fatal("trap placement", trap)
					}
					for j := 0; j < i; j++ {
						if trap.X == g.state.MapTraps[j].X && trap.Y == g.state.MapTraps[j].Y {
							t.Fatal("duplicate trap")
						}
					}
				}
			}
		}
	}
	g := wave4RingGame(199)
	g.moveCount = 50
	for y := range g.state.Map {
		for x := range g.state.Map[y] {
			g.state.Map[y][x].Type = "wall"
		}
	}
	g.runTurnHooks()
	runQueuedActions(g)
	if len(g.state.MapTraps) != 0 {
		t.Fatal("full floor")
	}
}

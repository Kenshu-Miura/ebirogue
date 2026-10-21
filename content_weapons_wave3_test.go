//go:build !test

package main

import (
	"encoding/json"
	"image/png"
	"os"
	"reflect"
	"strings"
	"testing"
)

func wave3WeaponGame(id int) (*Game, *Weapon) {
	g := newFoundationTestGame()
	w := buildItemFromTemplate(id, 0, 0).(*Weapon)
	w.Sharpness, w.Cursed = 0, false
	equip(g, w)
	e := CreateEnemyByID(0, 5, 4)
	e.Health, e.MaxHealth, e.DefensePower = 1000, 1000, 0
	g.state.Enemies = []Enemy{e}
	return g, w
}

func wave3Attack(g *Game) (int, string) {
	before := g.state.Enemies[0].Health
	g.CheckForEnemies(1, 0)
	messages := runQueuedActions(g)
	after := 0
	if len(g.state.Enemies) > 0 {
		after = g.state.Enemies[0].Health
	}
	return before - after, messages
}

func TestWave3WeaponTemplates(t *testing.T) {
	cases := []struct {
		id                int
		name, kind, image string
		power             int
		abilities         []EquipmentAbilityID
	}{
		{126, "カタナ", "Katana", "img/katana.png", 5, []EquipmentAbilityID{}},
		{127, "剛剣", "Gouken", "img/gouken.png", 10, []EquipmentAbilityID{}},
		{128, "金の剣", "GoldSword", "img/gold_sword.png", 3, []EquipmentAbilityID{AbilityRustImmune, AbilityKeenEdge}},
		{129, "甲殻斬り", "ShellCutter", "img/shell_cutter.png", 5, []EquipmentAbilityID{AbilityShellSlayer}},
		{130, "ウニ割り", "UniBreaker", "img/uni_breaker.png", 5, []EquipmentAbilityID{AbilityExplosiveSlayer}},
		{131, "空の刃", "SkyBlade", "img/sky_blade.png", 4, []EquipmentAbilityID{AbilityFloatingSlayer}},
		{132, "連撃刀", "DoubleStrikeSword", "img/double_strike_sword.png", 4, []EquipmentAbilityID{AbilityDoubleStrike}},
		{133, "大斧", "GreatAxe", "img/great_axe.png", 6, []EquipmentAbilityID{AbilityCriticalUp}},
		{134, "回復の剣", "HealingSword", "img/healing_sword.png", 2, []EquipmentAbilityID{AbilityLifeSteal}},
		{135, "小判こん棒", "KobanClub", "img/koban_club.png", 6, []EquipmentAbilityID{AbilityMoneyStrike}},
		{136, "鉄塊の大剣", "IronMassSword", "img/iron_mass_sword.png", 7, []EquipmentAbilityID{AbilitySatietyStrike}},
		{137, "腹ペコ包丁", "HungryKnife", "img/hungry_knife.png", 5, []EquipmentAbilityID{AbilityHungerStrike}},
		{138, "溜め打ちの剣", "ChargeSword", "img/charge_sword.png", 3, []EquipmentAbilityID{AbilityChargeStrike}},
		{139, "四ツ叉の銛", "FourProngHarpoon", "img/four_prong_harpoon.png", 5, []EquipmentAbilityID{AbilityComboStrike}},
		{140, "七福の熊手", "LuckyRake", "img/lucky_rake.png", 5, []EquipmentAbilityID{AbilityLuckySeven}},
		{141, "木づち", "Mallet", "img/mallet.png", 5, []EquipmentAbilityID{AbilityMallet}},
		{142, "罠探りの棒", "TrapProbe", "img/trap_probe.png", 3, []EquipmentAbilityID{AbilityTrapProbe}},
		{143, "かつおぶし", "Katsuobushi", "img/katsuobushi.png", 4, []EquipmentAbilityID{AbilityEdible}},
		{144, "呪いの包丁", "CursedKnife", "img/cursed_knife.png", 7, []EquipmentAbilityID{AbilityCursePower}},
		{145, "黄金のつるはし", "GoldenPickaxe", "img/golden_pickaxe.png", 2, []EquipmentAbilityID{AbilityDigWall, AbilityRustImmune}},
		{146, "火迅の太刀", "KajinTachi", "img/kajin_tachi.png", 12, []EquipmentAbilityID{AbilityFlameBlade}},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			template, ok := itemTemplates[tc.id]
			if !ok || template.Name != tc.name || template.Type != tc.kind || template.AttackPower != tc.power || template.ItemType != "Weapon" || !reflect.DeepEqual(template.Abilities, tc.abilities) {
				t.Fatalf("template = %#v", template)
			}
			if specialWeaponImagePaths[tc.kind] != tc.image {
				t.Fatal("image mapping")
			}
			f, err := os.Open(tc.image)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			img, err := png.Decode(f)
			if err != nil {
				t.Fatal(err)
			}
			if img.Bounds().Dx() != 30 || img.Bounds().Dy() != 30 {
				t.Fatal(img.Bounds())
			}
			for _, corner := range [][2]int{{0, 0}, {29, 0}, {0, 29}, {29, 29}} {
				_, _, _, a := img.At(corner[0], corner[1]).RGBA()
				if a != 0 {
					t.Fatal("opaque corner")
				}
			}
			w := buildItemFromTemplate(tc.id, 2, 3).(*Weapon)
			w.AbilityCounter, w.Sharpness, w.Cursed, w.Identified, w.RustProof = 2, 3, true, true, true
			data, err := json.Marshal(itemToSaved(w))
			if err != nil {
				t.Fatal(err)
			}
			var saved SavedItem
			if err := json.Unmarshal(data, &saved); err != nil {
				t.Fatal(err)
			}
			restored, err := savedToItem(saved)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(w, restored) {
				t.Fatalf("restore: %#v != %#v", w, restored)
			}
		})
	}
	for floor := 1; floor <= finalFloor; floor++ {
		if itemCategoryWeight("Weapon", floor) != 12 {
			t.Fatal("weapon category weight changed")
		}
	}
	// 147（巨大ハリセン）は段3の wave5-C で追加済み。内容は content_items_wave5c_test.go が検査する。
}

func TestWave3GoldSwordAndPickaxe(t *testing.T) {
	for _, id := range []int{128, 145} {
		g, w := wave3WeaponGame(id)
		rustTrapEffect(g)
		runQueuedActions(g)
		if w.Sharpness != 0 {
			t.Fatal("rust immunity")
		}
	}
	for _, roll := range []int{94, 95} {
		g, _ := wave3WeaponGame(128)
		g.randHooks.attackHit = func(int) int { return roll }
		damage, _ := wave3Attack(g)
		if (damage > 0) != (roll == 94) {
			t.Fatalf("accuracy roll %d damage %d", roll, damage)
		}
	}
	g, w := wave3WeaponGame(145)
	g.state.Enemies = nil
	g.state.Map[4][5] = Tile{Type: "wall", Blocked: true, BlockSight: true}
	g.CheckForEnemies(1, 0)
	runQueuedActions(g)
	if g.state.Map[4][5].Type != "corridor" || w.Sharpness != 0 {
		t.Fatal("digging")
	}
}

func TestWave3Slayers(t *testing.T) {
	for _, tc := range []struct {
		id    int
		trait EnemyTrait
	}{{129, EnemyTraitShell}, {130, EnemyTraitExplosive}, {131, EnemyTraitFloating}} {
		plain, _ := wave3WeaponGame(tc.id)
		normal, _ := wave3Attack(plain)
		g, _ := wave3WeaponGame(tc.id)
		g.state.Enemies[0].Traits = []EnemyTrait{tc.trait}
		damage, message := wave3Attack(g)
		if damage <= normal || !strings.Contains(message, "特効！") {
			t.Fatalf("slayer %d: %d / %d %s", tc.id, damage, normal, message)
		}
	}
}

func TestWave3DoubleStrike(t *testing.T) {
	for _, tc := range []struct{ roll, hp, hits int }{{24, 1000, 2}, {25, 1000, 1}, {0, 1, 1}} {
		g, _ := wave3WeaponGame(132)
		g.state.Enemies[0].Health = tc.hp
		g.randHooks.ability = func(int) int { return tc.roll }
		_, message := wave3Attack(g)
		if strings.Count(message, "ダメージを与えた") != tc.hits || g.inDoubleStrike {
			t.Fatalf("%+v: %s flag %v", tc, message, g.inDoubleStrike)
		}
	}
	g, _ := wave3WeaponGame(132)
	g.randHooks.attackHit = func(int) int { return 99 }
	_, message := wave3Attack(g)
	if strings.Contains(message, "連撃") {
		t.Fatal(message)
	}
}

func TestWave3CriticalWeapons(t *testing.T) {
	for _, tc := range []struct {
		id, hp int
		force  bool
	}{{133, 40, false}, {140, 17, true}, {140, 16, false}} {
		g, _ := wave3WeaponGame(tc.id)
		g.state.Player.Health = tc.hp
		var denominator int
		g.randHooks.damage = func(n int) int {
			if n != randomFactorSpread {
				denominator = n
			}
			return n - 1
		}
		_, message := wave3Attack(g)
		if tc.id == 133 && denominator != 8 {
			t.Fatal(denominator)
		}
		if strings.Contains(message, "会心の一撃") != tc.force {
			t.Fatal(message)
		}
	}
	// 会心は1/8。乱数の呼び出し範囲と全8結果を確認する。
	for roll := 0; roll < 8; roll++ {
		g, _ := wave3WeaponGame(133)
		g.randHooks.damage = func(n int) int {
			if n == 8 {
				return roll
			}
			return n - 1
		}
		_, message := wave3Attack(g)
		if strings.Contains(message, "会心の一撃") != (roll == 0) {
			t.Fatalf("roll %d: %s", roll, message)
		}
	}
}

func TestWave3LifeSteal(t *testing.T) {
	for _, hp := range []int{10, 39, 40} {
		g, _ := wave3WeaponGame(134)
		g.state.Player.Health = hp
		damage, _ := wave3Attack(g)
		if g.state.Player.Health != min(40, hp+max(1, damage/4)) {
			t.Fatal(g.state.Player.Health, damage)
		}
	}
	g, w := wave3WeaponGame(134)
	g.state.Player.Health = 10
	equipmentAbilityHooks[AbilityLifeSteal].AfterPlayerHit(g, PlayerAttackHit{Weapon: w, Damage: 1}, 1)
	if g.state.Player.Health != 11 {
		t.Fatal("minimum healing")
	}
	g, _ = wave3WeaponGame(134)
	g.state.Player.Health = 10
	g.randHooks.attackHit = func(int) int { return 99 }
	wave3Attack(g)
	if g.state.Player.Health != 10 {
		t.Fatal("miss healed")
	}
}

func TestWave3StrikeCosts(t *testing.T) {
	for _, id := range []int{135, 136} {
		for _, enough := range []bool{true, false} {
			for _, hit := range []bool{true, false} {
				g, w := wave3WeaponGame(id)
				p := &g.state.Player
				p.Cash, p.Satiety = 19, 9
				if enough {
					p.Cash, p.Satiety = 20, 10
				}
				cash, satiety := p.Cash, p.Satiety
				a := &PlayerAttack{Weapon: w, Multiplier: 1, CriticalDenominator: 20}
				g.runBeforePlayerAttackHooks(a)
				want := 1.0
				if enough {
					want = 1.5
				}
				if a.Multiplier != want || p.Cash != cash || p.Satiety != satiety {
					t.Fatal("early effect", a, p)
				}
				if !hit {
					g.randHooks.attackHit = func(int) int { return 99 }
				}
				wave3Attack(g)
				if enough && hit {
					if id == 135 {
						cash -= 20
					} else {
						satiety -= 2
					}
				}
				if p.Cash != cash || p.Satiety != satiety {
					t.Fatalf("id %d enough %v hit %v: cash %d satiety %d", id, enough, hit, p.Cash, p.Satiety)
				}
			}
		}
	}
	for _, satiety := range []int{0, 1} {
		g, w := wave3WeaponGame(137)
		g.state.Player.Satiety = satiety
		a := &PlayerAttack{Weapon: w, Multiplier: 1}
		g.runBeforePlayerAttackHooks(a)
		want := 1.0
		if satiety == 0 {
			want = 2
		}
		if a.Multiplier != want {
			t.Fatal(a)
		}
	}
}

func TestWave3Counters(t *testing.T) {
	g, w := wave3WeaponGame(138)
	g.randHooks.attackHit = func(int) int { return 99 }
	wave3Attack(g)
	wave3Attack(g)
	if w.AbilityCounter != 2 {
		t.Fatal(w.AbilityCounter)
	}
	_, message := wave3Attack(g)
	if w.AbilityCounter != 0 || !strings.Contains(message, "会心") {
		t.Fatal(w.AbilityCounter, message)
	}
	// 敵の回避も溜め打ちの外れに数え、発動時は回避を無視する。
	g, w = wave3WeaponGame(138)
	withTestMonster(t, 999, 0, func(def *MonsterDefinition) { def.EvasionPercent = 100 })
	g.state.Enemies[0].ID = 999
	g.randHooks.enemyEvasion = func(int) int { return 0 }
	wave3Attack(g)
	wave3Attack(g)
	wave3Attack(g)
	if w.AbilityCounter != 0 {
		t.Fatal("evaded charge", w.AbilityCounter)
	}
	g, w = wave3WeaponGame(139)
	wave3Attack(g)
	wave3Attack(g)
	wave3Attack(g)
	wave3Attack(g)
	if w.AbilityCounter != 4 {
		t.Fatal(w.AbilityCounter)
	}
	g.randHooks.attackHit = func(int) int { return 99 }
	_, message = wave3Attack(g)
	if w.AbilityCounter != 0 || !strings.Contains(message, "会心") {
		t.Fatal(w.AbilityCounter, message)
	}
	g.randHooks.attackHit = func(int) int { return 0 }
	wave3Attack(g)
	g.randHooks.attackHit = func(int) int { return 99 }
	wave3Attack(g)
	if w.AbilityCounter != 0 {
		t.Fatal("combo miss", w.AbilityCounter)
	}
	for _, id := range []EquipmentAbilityID{AbilityChargeStrike, AbilityComboStrike} {
		hooks := equipmentAbilityHooks[id]
		hooks.BeforePlayerAttack(g, &PlayerAttack{}, 1)
		hooks.AfterPlayerMiss(g, &PlayerAttack{}, 1)
	}
}

func TestWave3CursePower(t *testing.T) {
	g, w := wave3WeaponGame(144)
	for _, cursed := range []bool{true, false} {
		w.Cursed = cursed
		a := &PlayerAttack{Weapon: w, Multiplier: 1}
		g.runBeforePlayerAttackHooks(a)
		want := 0
		if cursed {
			want = 3
		}
		if a.AttackBonus != want {
			t.Fatal(a)
		}
	}
}

func TestWave3Mallet(t *testing.T) {
	for _, tc := range []struct {
		trap, item, enemy bool
		roll              int
	}{{true, true, false, 24}, {false, true, false, 25}, {true, true, true, 0}, {false, false, false, 0}} {
		g, w := wave3WeaponGame(141)
		w.RustProof = true
		if !tc.enemy {
			g.state.Enemies = nil
		}
		if tc.trap {
			g.state.MapTraps = []MapTrap{createMapTrapByID(0, 5, 4)}
		}
		if tc.item {
			g.state.Items = []Item{buildItemFromTemplate(1, 5, 4)}
		}
		power := g.state.Player.AttackPower
		g.randHooks.ability = func(int) int { return tc.roll }
		g.CheckForEnemies(1, 0)
		if len(g.state.MapTraps) != boolInt(tc.trap) || w.Sharpness != 0 {
			t.Fatal("early mallet effect")
		}
		runQueuedActions(g)
		destroyed := !tc.enemy && (tc.trap || tc.item)
		if destroyed && !g.miniMapDirty {
			t.Fatal("minimap")
		}
		traps, items := boolInt(tc.trap), boolInt(tc.item)
		if destroyed {
			if tc.trap {
				traps--
			} else {
				items--
			}
		}
		if len(g.state.MapTraps) != traps || len(g.state.Items) != items {
			t.Fatal(tc, g.state.MapTraps, g.state.Items)
		}
		wear := 0
		if destroyed && tc.roll < 25 {
			wear = 1
		}
		if w.Sharpness != -wear || g.state.Player.AttackPower != power-wear {
			t.Fatal("wear", w, g.state.Player.AttackPower)
		}
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestWave3TrapProbe(t *testing.T) {
	for _, delta := range []Coordinate{{1, 0}, {0, 1}, {1, 1}, {-1, -1}} {
		g, _ := wave3WeaponGame(142)
		g.state.Enemies = nil
		for distance := 1; distance <= 2; distance++ {
			for side := -1; side <= 1; side++ {
				g.state.MapTraps = append(g.state.MapTraps, createMapTrapByID(0, 4+distance*delta.X-side*delta.Y, 4+distance*delta.Y+side*delta.X))
			}
		}
		g.state.MapTraps = append(g.state.MapTraps, createMapTrapByID(0, 4+3*delta.X, 4+3*delta.Y))
		g.CheckForEnemies(delta.X, delta.Y)
		runQueuedActions(g)
		for i, tr := range g.state.MapTraps {
			if tr.Discovered != (i < 6) {
				t.Fatal(delta, i, tr)
			}
		}
	}
}

func TestWave3Gnaw(t *testing.T) {
	for _, equipped := range []bool{false, true} {
		for _, sharpness := range []int{0, -3} {
			g, w := wave3WeaponGame(143)
			g.state.Enemies = nil
			if !equipped {
				g.state.Player.UnequipItem(w)
			}
			w.Sharpness = sharpness
			// 強化値設定後の装備ステータスを合わせる。
			if equipped {
				g.state.Player.AttackPower += sharpness
			}
			g.state.Player.Satiety = 95
			power := g.state.Player.AttackPower
			wantActions := []string{"装備", "かじる", "投げる", "置く", "説明"}
			if equipped {
				wantActions[0] = "はずす"
			}
			if !reflect.DeepEqual(g.currentItemActions(), wantActions) {
				t.Fatal(g.currentItemActions())
			}
			g.selectedActionIndex = 1
			g.executeAction()
			if g.state.Player.Satiety != 95 {
				t.Fatal("early gnaw")
			}
			runQueuedActions(g)
			if g.state.Player.Satiety != 100 || !g.isActioned {
				t.Fatal(g.state.Player)
			}
			if sharpness == -3 {
				if len(g.state.Player.Inventory) != 0 || g.state.Player.EquippedWeapon != nil {
					t.Fatal("not consumed")
				}
				if equipped && g.state.Player.AttackPower != power-(w.AttackPower+sharpness) {
					t.Fatal("unequip stats")
				}
			} else if w.Sharpness != -1 || (equipped && g.state.Player.AttackPower != power-1) {
				t.Fatal("gnaw wear")
			}
		}
	}
	g, w := wave3WeaponGame(143)
	g.state.Player.StatusAilments.MouthSeal = 1
	g.selectedActionIndex = 1
	g.executeAction()
	runQueuedActions(g)
	if g.isActioned || w.Sharpness != 0 || g.state.Player.Satiety != 50 {
		t.Fatal("mouth seal")
	}
}

func TestWave3MenusAndTurns(t *testing.T) {
	for _, action := range []string{"かじる", "投げる", "置く", "説明"} {
		t.Run(action, func(t *testing.T) {
			g, b := newTurnTestGame(t)
			w := buildItemFromTemplate(143, 0, 0).(*Weapon)
			w.Sharpness, w.Cursed = 0, false
			g.state.Player.Inventory = []Item{w}
			g.state.Player.Satiety = 50
			if action == "説明" {
				g.selectedActionIndex = 4
				g.executeAction()
				if !g.showItemDescription || g.isActioned {
					t.Fatal("description menu")
				}
				return
			}
			expectTurns(t, g, 1, func() error { return useOrFail(b, 0, action) })
			if action == "かじる" && (w.Sharpness != -1 || g.state.Player.Satiety != 65) {
				t.Fatal("gnaw menu")
			}
			if action == "置く" && (len(g.state.Items) != 1 || g.state.Items[0] != w) {
				t.Fatal("place menu")
			}
		})
	}
	for _, id := range []int{132, 141, 142, 146} {
		g, b := newTurnTestGame(t)
		w := buildItemFromTemplate(id, 0, 0).(*Weapon)
		w.Sharpness, w.Cursed = 0, false
		equip(g, w)
		if id == 132 {
			e := CreateEnemyByID(0, 5, 3)
			e.Health, e.MaxHealth = 1000, 1000
			g.state.Enemies = []Enemy{e}
			g.randHooks.ability = func(int) int { return 0 }
		}
		expectTurns(t, g, 1, func() error { return b.tap(KeyZ) })
	}
}

func TestWave3FlameBlade(t *testing.T) {
	for _, tc := range []struct {
		hp     int
		wall   bool
		damage int
	}{{40, false, 20}, {39, false, 0}, {40, true, 0}} {
		g, _ := wave3WeaponGame(146)
		g.state.Player.Health = tc.hp
		g.state.Enemies[0].X = 7
		second := g.state.Enemies[0]
		second.X = 8
		g.state.Enemies = append(g.state.Enemies, second)
		if tc.wall {
			g.state.Map[4][6] = Tile{Type: "wall", Blocked: true}
		}
		g.CheckForEnemies(1, 0)
		// 発動の判定は攻撃時、ダメージはExecute時。
		g.state.Player.Health = 1
		runQueuedActions(g)
		if g.state.Enemies[0].Health != 1000-tc.damage || g.state.Enemies[1].Health != 1000 {
			t.Fatal(tc, g.state.Enemies)
		}
	}
	g, _ := wave3WeaponGame(146)
	normal := CreateEnemyByID(42, 5, 4)
	normal.Health, normal.MaxHealth = 1000, 1000
	g.state.Enemies = []Enemy{normal}
	before := 1000
	_, message := wave3Attack(g)
	if g.state.Enemies[0].Health >= before-20 || !strings.Contains(message, "太刀の炎") {
		t.Fatal("boss flame", message)
	}
}

func TestWave3FlameRangeAndExperience(t *testing.T) {
	for _, distance := range []int{10, 11} {
		g, _ := wave3WeaponGame(146)
		g.state.Map = makeTestFloorMap(20, 9)
		g.state.Enemies[0].X = 4 + distance
		g.CheckForEnemies(1, 0)
		runQueuedActions(g)
		want := 1000
		if distance == 10 {
			want -= 20
		}
		if g.state.Enemies[0].Health != want {
			t.Fatal(distance, g.state.Enemies[0].Health)
		}
	}
	g, _ := wave3WeaponGame(146)
	g.state.Enemies[0].X, g.state.Enemies[0].Health = 7, 20
	exp := g.state.Player.ExperiencePoints
	g.CheckForEnemies(1, 0)
	runQueuedActions(g)
	if len(g.state.Enemies) != 0 || g.state.Player.ExperiencePoints <= exp {
		t.Fatal("flame defeat experience")
	}
}

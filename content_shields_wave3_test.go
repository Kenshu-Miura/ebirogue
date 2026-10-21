//go:build !test

package main

import (
	"image/png"
	"os"
	"reflect"
	"strings"
	"testing"
)

func wave3ShieldGame(id int) *Game {
	g := newFoundationTestGame()
	g.randHooks.shieldDefense = func(int) int { return 99 }
	g.randHooks.ability = func(int) int { return 99 }
	if id != 0 {
		armor := buildItemFromTemplate(id, 0, 0).(*Armor)
		armor.Sharpness, armor.Cursed = 0, false
		equip(g, armor)
	}
	return g
}

func TestWave3ShieldTemplatesAndSave(t *testing.T) {
	cases := []struct {
		id               int
		name, kind, path string
		defense          int
		abilities        []EquipmentAbilityID
	}{
		{148, "青銅甲の盾", "BronzeShield", "img/bronze_shield.png", 4, []EquipmentAbilityID{}},
		{149, "大海の盾", "TaikaiShield", "img/taikai_shield.png", 10, []EquipmentAbilityID{}},
		{150, "金の盾", "GoldShield", "img/gold_shield.png", 3, []EquipmentAbilityID{AbilityRustImmune}},
		{151, "竜鱗の盾", "DragonScaleShield", "img/dragon_scale_shield.png", 5, []EquipmentAbilityID{AbilityDragonGuard}},
		{152, "にぎりよけの盾", "NigiriGuardShield", "img/nigiri_guard_shield.png", 3, []EquipmentAbilityID{AbilityNigiriGuard}},
		{153, "操りよけの盾", "PuppetGuardShield", "img/puppet_guard_shield.png", 3, []EquipmentAbilityID{AbilityPuppetGuard}},
		{154, "小判の盾", "KobanShield", "img/koban_shield.png", 6, []EquipmentAbilityID{AbilityMoneyGuard}},
		{155, "鉄塊の大盾", "IronMassShield", "img/iron_mass_shield.png", 7, []EquipmentAbilityID{AbilitySatietyGuard}},
		{156, "腹ペコの盾", "HungryShield", "img/hungry_shield.png", 5, []EquipmentAbilityID{AbilityHungerGuard}},
		{157, "根性の盾", "GutsShield", "img/guts_shield.png", 6, []EquipmentAbilityID{AbilityGuts}},
		{158, "絶好調の盾", "PeakShield", "img/peak_shield.png", 4, []EquipmentAbilityID{AbilityPeakGuard}},
		{159, "悟りの盾", "SatoriShield", "img/satori_shield.png", 2, []EquipmentAbilityID{AbilitySatori}},
		{160, "七福の御守", "LuckyCharm", "img/lucky_charm.png", 5, []EquipmentAbilityID{AbilityLuckyGuard}},
		{161, "呪いの鍋ぶた", "CursedPotLid", "img/cursed_pot_lid.png", 7, []EquipmentAbilityID{AbilityCurseGuard}},
		{162, "カメの甲羅", "TurtleShell", "img/turtle_shell.png", 3, []EquipmentAbilityID{AbilityShellDodge}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			template, ok := itemTemplates[tc.id]
			if !ok || template.Name != tc.name || template.Type != tc.kind || template.ItemType != "Armor" || template.DefensePower != tc.defense || !reflect.DeepEqual(template.Abilities, tc.abilities) {
				t.Fatalf("template = %+v", template)
			}
			if specialArmorImagePaths[tc.kind] != tc.path {
				t.Fatal("image registration missing")
			}
			f, err := os.Open(tc.path)
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
			for _, xy := range [][2]int{{0, 0}, {29, 0}, {0, 29}, {29, 29}} {
				_, _, _, a := img.At(xy[0], xy[1]).RGBA()
				if a != 0 {
					t.Fatal("opaque corner")
				}
			}
			armor := buildItemFromTemplate(tc.id, 0, 0).(*Armor)
			armor.Sharpness, armor.Cursed, armor.RustProof, armor.Identified = 2, true, true, true
			restored, err := savedToItem(itemToSaved(armor))
			if err != nil {
				t.Fatal(err)
			}
			got := restored.(*Armor)
			if !reflect.DeepEqual(got.Abilities, armor.Abilities) || got.Type != armor.Type || got.Sharpness != 2 || !got.Cursed || !got.RustProof || !got.Identified {
				t.Fatalf("restored = %+v", got)
			}
			found := false
			for _, id := range randomItemTemplateIDs {
				if id == tc.id {
					found = true
				}
			}
			if !found {
				t.Fatal("missing floor spawn candidate")
			}
		})
	}
	for floor := 1; floor <= finalFloor; floor++ {
		if itemCategoryWeight("Armor", floor) != 14 {
			t.Fatal("armor category weight changed")
		}
	}
}

func TestWave3GoldShieldRust(t *testing.T) {
	for _, id := range []int{148, 150} {
		g := wave3ShieldGame(id)
		armor := g.state.Player.EquippedArmor
		before := g.state.Player.DefensePower
		rustTrapEffect(g)
		runQueuedActions(g)
		want := -1
		if id == 150 {
			want = 0
		}
		if armor.Sharpness != want || g.state.Player.DefensePower != before+want {
			t.Fatalf("id %d: sharpness %d defense %d", id, armor.Sharpness, g.state.Player.DefensePower)
		}
	}
}

func TestWave3DragonGuardAttacks(t *testing.T) {
	for _, tc := range []struct {
		name   string
		traits []EnemyTrait
		kind   RangedAttackKind
		half   bool
	}{
		{"dragon melee", []EnemyTrait{EnemyTraitDragon}, RangedAttackNone, true},
		{"ordinary melee", nil, RangedAttackNone, false},
		{"fire", nil, RangedAttackFire, true},
		{"dragon fire once", []EnemyTrait{EnemyTraitDragon}, RangedAttackFire, true},
		{"rock", nil, RangedAttackRock, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := wave3ShieldGame(151)
			enemy := CreateEnemyByID(0, 6, 4)
			enemy.Traits = tc.traits
			enemy.AttackPower = 20
			enemy.PlayerDiscovered = true
			enemy.RangedAttack = RangedAttackDefinition{Kind: tc.kind, AttackPower: 20, MinRange: 1, MaxRange: 8}
			g.state.Enemies = []Enemy{enemy}
			raw := rollEnemyAttackDamage(20, g.state.Player.DefensePower, g.damageIntn())
			want := raw
			if tc.half {
				want = max(1, raw/2)
			}
			if tc.kind == RangedAttackNone {
				g.enqueueEnemyNormalAttack(0)
			} else if !g.tryEnemyRangedAttack(0) {
				t.Fatal("ranged attack unavailable")
			}
			if g.state.Player.Health != 40 {
				t.Fatal("damage applied before Execute")
			}
			runQueuedActions(g)
			if g.state.Player.Health != 40-want {
				t.Fatalf("health %d want %d", g.state.Player.Health, 40-want)
			}
		})
	}
}

func TestWave3ShieldImmunities(t *testing.T) {
	for _, tc := range []struct {
		id    int
		apply func(*Enemy, *Game, func(int) int)
	}{
		{152, enqueueFoodTransformationAttack}, {153, enqueueManipulationAttack},
	} {
		for _, guarded := range []bool{false, true} {
			id := 0
			if guarded {
				id = tc.id
			}
			g := wave3ShieldGame(id)
			food := buildItemFromTemplate(1, 0, 0)
			if tc.id == 152 {
				food = testWeapon()
			}
			g.state.Player.Inventory = append(g.state.Player.Inventory, food)
			g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 4)}
			tc.apply(&g.state.Enemies[0], g, func(int) int { return 0 })
			msg := runQueuedActions(g)
			if guarded {
				if !strings.Contains(msg, itemTemplates[tc.id].Name) || !strings.Contains(msg, "防いだ") {
					t.Fatal(msg)
				}
				if len(g.state.Player.Inventory) != 2 || g.state.Player.Inventory[1] != food {
					t.Fatal("guarded special attack changed inventory")
				}
			} else if strings.Contains(msg, "防いだ") {
				t.Fatal(msg)
			}
			if !guarded && tc.id == 152 && g.state.Player.Inventory[0].GetID() != 1 {
				t.Fatal("food transformation did not occur")
			}
			if !guarded && tc.id == 153 && len(g.state.Player.Inventory) != 0 {
				t.Fatal("manipulation did not consume food")
			}
		}
	}
}

func TestWave3ResourceGuards(t *testing.T) {
	for _, id := range []int{154, 155} {
		for _, tc := range []struct {
			name          string
			cash, satiety int
			delivery      EnemyAttackDelivery
			evaded        bool
			reduced       bool
		}{
			{"threshold", 20, 10, EnemyAttackMelee, false, true},
			{"short", 19, 9, EnemyAttackMelee, false, false},
			{"ranged", 20, 10, EnemyAttackRanged, false, false},
			{"environment", 20, 10, EnemyAttackEnvironment, false, false},
			{"evaded", 20, 10, EnemyAttackMelee, true, false},
		} {
			t.Run(itemTemplates[id].Name+"/"+tc.name, func(t *testing.T) {
				g := wave3ShieldGame(id)
				g.state.Player.Cash, g.state.Player.Satiety = tc.cash, tc.satiety
				if tc.evaded {
					g.state.Player.EquippedArmor.Abilities = append(g.state.Player.EquippedArmor.Abilities, AbilityEvasion)
					g.randHooks.shieldDefense = func(int) int { return 0 }
				}
				result := g.resolvePlayerShieldDefense(12, EnemyDamageNormal, tc.delivery)
				want := 12
				if tc.reduced {
					want = 8
				}
				if tc.evaded {
					want = 0
				}
				if result.Damage != want || result.Evaded != tc.evaded {
					t.Fatalf("result %+v", result)
				}
				if g.state.Player.Cash != tc.cash || g.state.Player.Satiety != tc.satiety {
					t.Fatal("paid before Execute")
				}
				g.applyPlayerShieldDefense(result, DamageSource{}, -1, 0, 0)
				cash, satiety := tc.cash, tc.satiety
				if tc.reduced && id == 154 {
					cash -= 20
				}
				if tc.reduced && id == 155 {
					satiety -= 2
				}
				if g.state.Player.Cash != cash || g.state.Player.Satiety != satiety {
					t.Fatalf("cash/satiety %d/%d", g.state.Player.Cash, g.state.Player.Satiety)
				}
			})
		}
		// 最低1への切り上げで実際に軽減できなければ代価を払わない。
		g := wave3ShieldGame(id)
		g.state.Player.Cash = 20
		g.state.Player.Satiety = 10
		result := g.resolvePlayerShieldDefense(1, EnemyDamageNormal, EnemyAttackMelee)
		g.applyPlayerShieldDefense(result, DamageSource{}, -1, 0, 0)
		if result.Damage != 1 || g.state.Player.Cash != 20 || g.state.Player.Satiety != 10 {
			t.Fatal("paid for no reduction")
		}
	}
}

func TestWave3ConditionalGuards(t *testing.T) {
	for _, tc := range []struct {
		id, health, satiety, want int
		cursed                    bool
	}{
		{156, 40, 0, 6, false}, {156, 40, 1, 12, false},
		{158, 40, 50, 6, false}, {158, 39, 50, 12, false},
		{160, 17, 50, 6, false}, {160, 16, 50, 12, false},
		{161, 40, 50, 9, true}, {161, 40, 50, 12, false},
	} {
		g := wave3ShieldGame(tc.id)
		g.state.Player.Health, g.state.Player.Satiety = tc.health, tc.satiety
		g.state.Player.EquippedArmor.Cursed = tc.cursed
		for _, delivery := range []EnemyAttackDelivery{EnemyAttackMelee, EnemyAttackRanged, EnemyAttackEnvironment} {
			result := g.resolvePlayerShieldDefense(12, EnemyDamageNormal, delivery)
			if result.Damage != tc.want {
				t.Fatalf("id %d: damage %d want %d", tc.id, result.Damage, tc.want)
			}
			if g.resolvePlayerShieldDefense(1, EnemyDamageNormal, delivery).Damage != 1 {
				t.Fatal("minimum damage")
			}
		}
	}
	g := wave3ShieldGame(161)
	g.state.Player.EquippedArmor.Cursed = true
	// 実際のおはらい効果で呪いを解除したあとも、装備し直さず通常の被害に戻る。
	g.state.Player.Inventory = append(g.state.Player.Inventory, buildItemFromTemplate(32, 0, 0))
	g.selectedItemIndex = len(g.state.Player.Inventory) - 1
	removeCurse(g)
	runQueuedActions(g)
	if g.resolvePlayerShieldDefense(12, EnemyDamageNormal, EnemyAttackMelee).Damage != 12 {
		t.Fatal("uncursing did not remove guard")
	}
}

func TestWave3HungerGuardTrapAndStarvation(t *testing.T) {
	g := wave3ShieldGame(156)
	g.state.Player.Satiety = 0
	mineTrapEffect(g)
	runQueuedActions(g)
	if g.state.Player.Health != 30 {
		t.Fatalf("mine health=%d", g.state.Player.Health)
	}
	g.state.Player.Health = 1
	g.IncrementMoveCount()
	if g.state.Player.Health != 0 || g.lastDamageSource.Kind != DamageSourceStarvation {
		t.Fatal("starvation was reduced")
	}
}

func TestWave3Guts(t *testing.T) {
	for _, tc := range []struct {
		roll, damage, want, calls int
		source                    DamageSourceKind
	}{
		{0, 40, 1, 1, DamageSourceEnemyAttack}, {1, 40, 0, 1, DamageSourceEnemyAttack},
		{2, 40, 0, 1, DamageSourceEnemyAttack}, {0, 40, 0, 0, DamageSourceStarvation},
		{0, 39, 1, 0, DamageSourceEnemyAttack}, {0, 0, 40, 0, DamageSourceEnemyAttack},
	} {
		g := wave3ShieldGame(157)
		calls := 0
		g.randHooks.ability = func(n int) int {
			calls++
			if n != 3 {
				t.Fatal(n)
			}
			return tc.roll
		}
		g.damagePlayer(tc.damage, DamageSource{Kind: tc.source})
		msg := runQueuedActions(g)
		if g.state.Player.Health != tc.want || calls != tc.calls {
			t.Fatalf("health/calls %d/%d", g.state.Player.Health, calls)
		}
		if strings.Contains(msg, "根性で踏みとどまった！") != (tc.calls == 1 && tc.roll == 0) {
			t.Fatal(msg)
		}
	}
}

func TestWave3ShellDodgeAttacks(t *testing.T) {
	for _, tc := range []struct {
		name  string
		kind  RangedAttackKind
		roll  int
		dodge bool
	}{
		{"arrow 49", RangedAttackArrow, 49, true}, {"arrow 50", RangedAttackArrow, 50, false},
		{"rock", RangedAttackRock, 0, true}, {"fire", RangedAttackFire, 0, true},
		{"magic", RangedAttackMagic, 0, true}, {"explosion", RangedAttackExplosion, 0, false},
		{"melee", RangedAttackNone, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := wave3ShieldGame(162)
			calls := 0
			g.randHooks.ability = func(int) int { calls++; return tc.roll }
			enemy := CreateEnemyByID(0, 6, 4)
			enemy.PlayerDiscovered = true
			enemy.AttackPower = 10
			enemy.RangedAttack = RangedAttackDefinition{Kind: tc.kind, AttackPower: 10, MinRange: 1, MaxRange: 8, BlastRadius: 1}
			g.state.Enemies = []Enemy{enemy}
			if tc.kind == RangedAttackNone {
				g.enqueueEnemyNormalAttack(0)
			} else if !g.tryEnemyRangedAttack(0) {
				t.Fatal("no attack")
			}
			msg := runQueuedActions(g)
			if (g.state.Player.Health == 40) != tc.dodge {
				t.Fatalf("health=%d msg=%s", g.state.Player.Health, msg)
			}
			if tc.dodge && !strings.Contains(msg, "カメの甲羅で") {
				t.Fatal(msg)
			}
			if (tc.kind == RangedAttackNone || tc.kind == RangedAttackExplosion) && calls != 0 {
				t.Fatal("ineligible dodge consumed random")
			}
		})
	}
}

func TestWave3SatoriEquipmentAndFood(t *testing.T) {
	for _, ground := range []bool{false, true} {
		g := wave3ShieldGame(0)
		armor := buildItemFromTemplate(159, 4, 4).(*Armor)
		armor.Cursed = false
		armor.Sharpness = 0
		if ground {
			g.state.Items = []Item{armor}
			g.selectedGroundItemIndex = 0
			g.selectedGroundActionIndex = 2
			g.executeGroundItemAction()
		} else {
			g.state.Player.Inventory = []Item{armor}
			g.selectedActionIndex = 0
			g.executeAction()
		}
		runQueuedActions(g)
		if g.state.Player.EquippedArmor != armor || g.state.Player.Satiety != 1 || g.state.Player.MaxSatiety != 100 || !g.isActioned {
			t.Fatalf("equip: %+v", g.state.Player)
		}
		for i := 0; i < 30; i++ {
			g.IncrementMoveCount()
		}
		if g.state.Player.Satiety != 1 {
			t.Fatal("satori lost satiety")
		}
		for _, foodID := range []int{1, 26, 27, 71} {
			g.state.Player.Inventory = []Item{armor, buildItemFromTemplate(foodID, 0, 0)}
			g.selectedItemIndex = 1
			g.state.Player.Inventory[1].Use(g)
			runQueuedActions(g)
			if g.state.Player.Satiety != 1 || g.state.Player.MaxSatiety != 100 {
				t.Fatalf("food %d: satiety %d/%d", foodID, g.state.Player.Satiety, g.state.Player.MaxSatiety)
			}
		}
		g.state.Player.UnequipItem(armor)
		g.state.Player.Inventory = []Item{buildItemFromTemplate(26, 0, 0)}
		g.selectedItemIndex = 0
		g.state.Player.Inventory[0].Use(g)
		runQueuedActions(g)
		if g.state.Player.Satiety != 100 || g.state.Player.MaxSatiety != 100 {
			t.Fatal("unequipped cannot eat normally")
		}
	}
}

func TestWave3ResourceGuardAttackQueue(t *testing.T) {
	for _, id := range []int{154, 155} {
		g := wave3ShieldGame(id)
		g.state.Player.Cash, g.state.Player.Satiety = 20, 10
		enemy := CreateEnemyByID(0, 5, 4)
		enemy.AttackPower = 20
		g.state.Enemies = []Enemy{enemy}
		raw := rollEnemyAttackDamage(20, g.state.Player.DefensePower, g.damageIntn())
		g.enqueueEnemyNormalAttack(0)
		if g.state.Player.Health != 40 || g.state.Player.Cash != 20 || g.state.Player.Satiety != 10 {
			t.Fatal("attack paid/applied before Execute")
		}
		runQueuedActions(g)
		cash, satiety := 20, 10
		if id == 154 {
			cash = 0
		} else {
			satiety = 8
		}
		if g.state.Player.Health != 40-max(1, raw*2/3) || g.state.Player.Cash != cash || g.state.Player.Satiety != satiety {
			t.Fatalf("queued resource guard: %+v", g.state.Player)
		}
	}
}
func TestWave3ShieldBotExclusions(t *testing.T) {
	for _, id := range []int{155, 156, 159} {
		g := wave3ShieldGame(0)
		armor := buildItemFromTemplate(id, 0, 0).(*Armor)
		armor.Sharpness = 0
		armor.Cursed = false
		armor.Identified = true
		g.state.Player.Inventory = []Item{armor}
		b := bot{g: g}
		if _, ok := b.bestEquipment(); ok {
			t.Fatalf("bot chose shield %d", id)
		}
	}
}

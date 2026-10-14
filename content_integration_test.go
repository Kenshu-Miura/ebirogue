//go:build !test

package main

import "testing"

// 担当をまたぐ組み合わせのテスト（統合時に追加）。
// 段1（wave3-A/B/C）の統合で接続した、悟りの盾と他担当の満腹度を増やす効果の組み合わせ。
func wave3EquipSatori(t *testing.T, g *Game) {
	t.Helper()
	armor := buildItemFromTemplate(159, 0, 0).(*Armor)
	armor.Cursed, armor.Sharpness = false, 0
	g.state.Player.Inventory = append(g.state.Player.Inventory, armor)
	if _, err := g.state.Player.EquipItem(armor); err != nil {
		t.Fatal(err)
	}
	if !g.playerHasAbility(AbilitySatori) {
		t.Fatal("satori not equipped")
	}
}

func TestWave3SatoriCapsKatsuobushiGnaw(t *testing.T) {
	g, w := wave3WeaponGame(143)
	g.state.Player.Inventory = []Item{w}
	wave3EquipSatori(t, g)
	g.state.Player.Satiety = 1
	g.selectedItemIndex = 0
	g.selectedActionIndex = 1 // 「かじる」
	g.executeAction()
	runQueuedActions(g)
	if g.state.Player.Satiety != 1 {
		t.Fatalf("gnaw with satori: satiety %d", g.state.Player.Satiety)
	}
}

func TestWave3SatoriCapsTroubleCardHunger(t *testing.T) {
	g := newFoundationTestGame()
	wave3EquipSatori(t, g)
	g.state.Player.Satiety = 0
	useWave3TestCard(g, 173)
	if g.state.Player.Satiety != 1 {
		t.Fatalf("trouble card with satori: satiety %d", g.state.Player.Satiety)
	}
}

// 段3の統合: wave5-B のイタズラカワハギ系の投げ捨てを、wave5-C のいたずらよけの盾（164）が防ぐ。
func TestWave5MischiefGuardShieldBlocksToss(t *testing.T) {
	for _, id := range []int{102, 103, 104} {
		g := newFoundationTestGame()
		g.state.Map = makeTestFloorMap(25, 25)
		addTestEnemy(g, id, 5, 4)
		shield := buildItemFromTemplate(164, 0, 0).(*Armor)
		shield.Cursed, shield.Sharpness = false, 0
		food := buildItemFromTemplate(1, 0, 0)
		g.state.Player.Inventory = []Item{food, shield}
		if _, err := g.state.Player.EquipItem(shield); err != nil {
			t.Fatal(err)
		}
		MonsterDefinitions[id].SpecialAttack(&g.state.Enemies[0], g)
		runQueuedActions(g)
		if len(g.state.Player.Inventory) != 2 || g.state.Player.Inventory[0] != food || len(g.state.Items) != 0 {
			t.Fatalf("%s tossed an item through the mischief guard shield", MonsterDefinitions[id].Name)
		}
	}
}

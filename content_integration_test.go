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

// 段4（wave6-A/B/C）の組み合わせ: フグの自爆が化けたミミックダコの正体を現し、シビレエイの帯電を進める。
func TestWave6FuguExplosionRevealsMimicAndChargesRay(t *testing.T) {
	g := newFoundationTestGame()
	g.state.Map = makeTestFloorMap(12, 9)
	fugu := addTestEnemy(g, 108, 5, 4)
	mimic := addTestEnemy(g, 117, 6, 4)
	g.state.Enemies[mimic].Health = 99
	disguiseEnemyAs(&g.state.Enemies[mimic], 0)
	ray := addTestEnemy(g, 114, 5, 5)
	g.state.Enemies[ray].Health = 99

	g.enqueueEnemyExplosion(EnemyExplosion{BomberIndex: fugu, CenterX: 5, CenterY: 4, Radius: 1, Damage: 14, SelfDestruct: true})
	runQueuedActions(g)

	for _, e := range g.state.Enemies {
		switch e.ID {
		case 117:
			if !e.Revealed || e.Name != "ミミックダコ" {
				t.Fatalf("mimic = %+v, the explosion should reveal it", e)
			}
		case 114:
			if e.Counter != 1 {
				t.Fatalf("ray counter = %d, the explosion counts as a hit", e.Counter)
			}
		case 108:
			t.Fatal("the fugu should be gone")
		}
	}
}

// 化けた姿の能力は使わない: フクレフグに化けたミミックダコは弱っても破裂せず、ウチカエシエビに化けても打ち返さない。
func TestWave6MimicDoesNotUseDisguisedAbilities(t *testing.T) {
	g := newFoundationTestGame()
	mimic := addTestEnemy(g, 117, 5, 4)
	disguiseEnemyAs(&g.state.Enemies[mimic], 108)
	g.state.Enemies[mimic].Revealed = false

	if enemyProjectileReturnPercent(g.state.Enemies[mimic]) != 0 {
		t.Fatal("a mimic never returns projectiles")
	}
	g.applyDamageToEnemy(mimic, g.state.Enemies[mimic].Health-1)
	messages := runQueuedActions(g)
	if len(g.state.Enemies) != 1 || g.state.Player.Health != 40 || g.state.Enemies[0].Flag {
		t.Fatalf("mimic should not inflate or explode: %+v, messages = %q", g.state.Enemies, messages)
	}
}

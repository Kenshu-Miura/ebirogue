//go:build !test

package main

import (
	"encoding/json"
	"testing"
)

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

// 段7（wave7-A/B/C/D）の統合で確かめる組み合わせ。

// 状態異常の共通処理（7-A）で進む混乱は、入力の記録（7-B）の開始状態に入り、再生で同じ結果になる。
func TestWave7StatusTicksMatchInReplay(t *testing.T) {
	t.Chdir(t.TempDir())
	g, v := newHeadlessGame(2718, DifficultyIntermediate)
	g.persistent = true
	if err := g.RunUntilInputReady(100); err != nil {
		t.Fatal(err)
	}
	g.setPlayerStatus(StatusConfusion, 10)
	g.setPlayerStatus(StatusSlow, 6)
	replayTestFrames(t, g, v, 450)
	if g.replay == nil || len(g.replay.Segments) == 0 {
		t.Fatal("no replay recorded")
	}
	initial := g.replay.Segments[0].Initial.Player.StatusAilments
	if initial.Confusion != 10 || initial.Slow != 6 {
		t.Fatalf("initial state should carry the statuses: %+v", initial)
	}
	if s := g.state.Player.StatusAilments; s.Confusion >= 10 || s.Slow >= 6 {
		t.Fatalf("statuses should tick down while recording: %+v", s)
	}
	replayRoundTrip(t, g.replay)
}

// 状態異常を起こす薬（7-A の共通処理）に効果音（7-C）を添えても、BOT では音声を作らず、効果も Action の数も変わらない。
func TestWave7StatusPotionSoundKeepsEffectHeadless(t *testing.T) {
	g := newFoundationTestGame()
	g.persistent = true
	g.headless = true
	potion := buildItemFromTemplate(18, 0, 0)
	g.state.Player.Inventory = []Item{potion}
	g.selectedItemIndex = 0
	g.GroundItemActioned = false
	if !g.useConsumable(potion) {
		t.Fatal("potion not used")
	}
	if len(g.ActionQueue.Queue) == 0 {
		t.Fatal("potion should enqueue its effect")
	}
	runQueuedActions(g)
	if g.state.Player.StatusAilments.Confusion != 10 {
		t.Fatalf("confusion = %d, want 10", g.state.Player.StatusAilments.Confusion)
	}
	if g.audioState != nil {
		t.Fatal("headless game must not create audio")
	}
}

// 設定画面の入力の記録（7-B）と音量（7-C）は別の項目で、切り替えが互いに影響せず、保存して読み直しても残る。
func TestWave7SettingsReplayAndVolumesAreIndependent(t *testing.T) {
	g, _ := newHeadlessGame(5, DifficultyBeginner)
	g.headless = false
	g.settings = defaultSettings()
	g.showSettings = true
	v := NewVirtualInput()
	g.input = v
	press := func(index int, key Key) {
		t.Helper()
		g.settingsSelectedIndex = index
		v.Press(key)
		v.Update()
		if err := g.handleSettingsInput(); err != nil {
			t.Fatal(err)
		}
		v.Release(key)
		v.Update()
	}
	press(3, KeyRight)
	if replayEnabled(g.settings) || !g.replaySuppressed {
		t.Fatal("index 3 should turn the input replay off")
	}
	if g.settings.BGMVolume != 6 || g.settings.SEVolume != 6 {
		t.Fatalf("replay toggle changed volumes: %d %d", g.settings.BGMVolume, g.settings.SEVolume)
	}
	press(4, KeyLeft)
	press(5, KeyLeft)
	press(5, KeyLeft)
	if g.settings.BGMVolume != 5 || g.settings.SEVolume != 4 || replayEnabled(g.settings) {
		t.Fatalf("volumes = %d %d, replay = %v", g.settings.BGMVolume, g.settings.SEVolume, replayEnabled(g.settings))
	}
	data, err := json.Marshal(g.settings)
	if err != nil {
		t.Fatal(err)
	}
	again, err := decodeSettings(data)
	if err != nil {
		t.Fatal(err)
	}
	if replayEnabled(again) || again.BGMVolume != 5 || again.SEVolume != 4 {
		t.Fatalf("reloaded settings = %+v", again)
	}
	old, err := decodeSettings([]byte(`{"lastDifficulty":0,"showMinimap":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if !replayEnabled(old) || old.BGMVolume != 6 || old.SEVolume != 6 {
		t.Fatalf("old settings should get defaults: %+v", old)
	}
}

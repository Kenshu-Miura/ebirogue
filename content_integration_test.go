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

// 段8（wave8-A 能力値系・wave8-B 特殊系・wave8-C 罠の相互作用）の組み合わせ。

// 8-A×8-B: にぎりの防御0に、防御力上昇と、敵の攻撃力上昇・暴走の倍率が重なる。
func TestWave8NigiriDefenseAndStatMultipliersStack(t *testing.T) {
	g := newFoundationTestGame()
	g.state.Player.Health, g.state.Player.MaxHealth = 1000, 1000
	g.state.Player.DefensePower = 20
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 4)}
	g.setPlayerStatus(StatusNigiri, 20)
	g.setPlayerStatus(StatusDefenseUp, 30)
	g.applyEnemyStatus(0, StatusAttackUp, 30)
	g.applyEnemyStatus(0, StatusBerserk, 15)
	runQueuedActions(g)
	e, p := g.state.Enemies[0], g.state.Player
	multiplier := statAttackMultiplier(e.StatusAilments) * statDefenseMultiplier(p.StatusAilments) * berserkMultiplier(e.StatusAilments)
	if multiplier == 1 {
		t.Fatal("倍率がかかっていない")
	}
	want := rollEnemyAttackDamageWith(e.AttackPower, 0, multiplier, g.damageIntn())
	g.enqueueEnemyNormalAttack(0)
	runQueuedActions(g)
	if got := 1000 - g.state.Player.Health; got != want {
		t.Fatalf("damage=%d want=%d", got, want)
	}
}

// 8-A×8-B: 回復不能でも、投げ当てられた回復薬でにぎりは解ける（HP は戻らない）。
func TestWave8ThrownHealingCuresNigiriDuringNoHealing(t *testing.T) {
	g := newFoundationTestGame()
	g.state.Player.Health = 10
	g.setPlayerStatus(StatusNigiri, 20)
	g.setPlayerStatus(StatusNoHealing, 30)
	g.onTargetHit(&g.state.Player, buildItemFromTemplate(2, 0, 0), -1)
	runQueuedActions(g)
	if g.state.Player.Health != 10 {
		t.Fatalf("回復不能なのに HP が %d になった", g.state.Player.Health)
	}
	if statusActive(g.state.Player.StatusAilments, StatusNigiri) {
		t.Fatal("にぎりが解けない")
	}
}

// 8-A: 回復封じの罠（15）は海老さんが踏むと回復不能にする（敵は罠を踏まない。TestEnemyDoesNotTriggerTraps）。
func TestWave8NoHealingTrapAffectsPlayer(t *testing.T) {
	g := trapInteractionGame(15)
	trap := g.state.MapTraps[0]
	g.steppedTrapSource = trapSource(trap.TemplateID, trap.Name)
	trap.Effect(g)
	runQueuedActions(g)
	if !statusActive(g.state.Player.StatusAilments, StatusNoHealing) {
		t.Fatal("海老さんが踏んでも回復不能にならない")
	}
}

// 8-B×8-C: 無敵の海老さんは、投げた道具で起こした地雷の誘爆に巻き込まれても HP が減らない。
func TestWave8InvinciblePlayerSurvivesMineChain(t *testing.T) {
	g := newProjectileTestGame()
	g.state.Player.X, g.state.Player.Y = 3, 3
	g.setPlayerStatus(StatusInvincible, 10)
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 3)}
	g.state.Enemies[0].Health = 100
	large, mine := createMapTrapByID(11, 4, 3), createMapTrapByID(3, 4, 4)
	large.FailRate, mine.FailRate = 0, 0
	g.state.MapTraps = []MapTrap{large, mine}
	before := g.state.Player.Health
	g.queueLandedTrap(4, 3, true)
	runQueuedActions(g)
	if g.state.Player.Health != before {
		t.Fatalf("無敵なのに HP が %d→%d", before, g.state.Player.Health)
	}
	if len(g.state.Enemies) == 1 && g.state.Enemies[0].Health >= 100 {
		t.Fatal("周りの敵に爆風が届いていない")
	}
	if len(g.state.MapTraps) != 0 {
		t.Fatalf("誘爆していない: %+v", g.state.MapTraps)
	}
}

// 8-B×8-C: トラバサミにかかった敵は隣の海老さんを殴るが、身代わりの間は何もしない。
func TestWave8DecoyInBearTrapDoesNotAttack(t *testing.T) {
	for _, decoy := range []bool{false, true} {
		g := newFoundationTestGame()
		g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 4)}
		g.applyEnemyStatus(0, StatusBearTrap, 6)
		if decoy {
			g.applyEnemyStatus(0, StatusDecoy, 30)
		}
		runQueuedActions(g)
		hp := g.state.Player.Health
		g.actEnemy(0)
		runQueuedActions(g)
		if attacked := g.state.Player.Health < hp; attacked == decoy {
			t.Fatalf("decoy=%v attacked=%v", decoy, attacked)
		}
		if g.state.Enemies[0].X != 5 || g.state.Enemies[0].Y != 4 {
			t.Fatalf("decoy=%v: トラバサミの敵が動いた", decoy)
		}
	}
}

// 9-A×9-C: 合成・強化の壺は4階から、そのほかの段9の壺は1階から床の抽選表に入る。
func TestWave9NewPotsFollowFloorTable(t *testing.T) {
	for id := 225; id <= 232; id++ {
		if _, ok := itemTemplates[id]; !ok {
			t.Fatalf("pot %d has no template", id)
		}
		delayed := id == 225 || id == 227
		if got := floorTableHas(3, id); got == delayed {
			t.Errorf("pot %d on floor 3: in table = %v", id, got)
		}
		if !floorTableHas(4, id) || !floorTableHas(finalFloor, id) {
			t.Errorf("pot %d missing from deep floors", id)
		}
	}
}

// 9-A×9-B×9-C: 難易度付きのマップ生成でも、地形の上に道具・罠・敵を置かず、
// 3階までに合成・強化の壺を置かない。段9の壺と3種の地形は実際に生成される。
func TestWave9FloorGenerationWithTerrainAndPots(t *testing.T) {
	pots, terrain := map[int]int{}, map[string]int{}
	for seed := 0; seed < 240; seed++ {
		floor := 1 + seed%8
		difficulty := []Difficulty{DifficultyBeginner, DifficultyIntermediate, DifficultyAdvanced}[seed%3]
		player := Player{}
		grid, enemies, items, _, _, traps := GenerateRandomMapFor(70, 70, floor-1, &player, newFloorRNG(int64(seed), floor), difficulty)
		onTerrain := func(x, y int) bool { return terrainBlocksWalking(grid[y][x]) }
		for _, item := range items {
			x, y := item.GetPosition()
			if onTerrain(x, y) {
				t.Fatalf("seed=%d floor=%d: item %d on %s", seed, floor, item.GetID(), grid[y][x].Type)
			}
			if id := item.GetID(); id >= 225 && id <= 232 {
				pots[id]++
				if floor < 4 && (id == 225 || id == 227) {
					t.Fatalf("seed=%d floor=%d: delayed pot %d", seed, floor, id)
				}
			}
		}
		for _, trap := range traps {
			if onTerrain(trap.X, trap.Y) {
				t.Fatalf("seed=%d floor=%d: trap on %s", seed, floor, grid[trap.Y][trap.X].Type)
			}
		}
		for _, e := range enemies {
			if onTerrain(e.X, e.Y) {
				t.Fatalf("seed=%d floor=%d: enemy on %s", seed, floor, grid[e.Y][e.X].Type)
			}
		}
		for _, row := range grid {
			for _, tile := range row {
				terrain[tile.Type]++
			}
		}
	}
	if len(pots) == 0 {
		t.Fatal("no wave9 pots generated")
	}
	for _, kind := range []string{"water", "cracked_wall", "hidden_corridor"} {
		if terrain[kind] == 0 {
			t.Errorf("terrain %s never generated", kind)
		}
	}
}

// 9-A×9-C: 底抜けの壺が割れて作る罠も床の罠の表を使い、ボス階では落とし穴を作らない
// （落とし穴でボスを倒さずに降りられないように）。
func TestWave9BottomlessPotTrapsUseFloorTable(t *testing.T) {
	pitfalls := map[int]int{}
	for _, floor := range []int{9, 10} {
		for seed := int64(0); seed < 80; seed++ {
			g := newFoundationTestGame()
			g.rng = newTurnRNG(seed)
			g.Floor = floor
			g.scatterBottomlessTraps(4, 4)
			if len(g.state.MapTraps) == 0 {
				t.Fatal("no traps")
			}
			for _, trap := range g.state.MapTraps {
				if trap.TemplateID == 6 {
					pitfalls[floor]++
				}
				if !trap.Discovered {
					t.Fatal("bottomless trap must be discovered")
				}
			}
		}
	}
	if pitfalls[10] != 0 {
		t.Fatalf("pitfall on boss floor: %d", pitfalls[10])
	}
	if pitfalls[9] == 0 {
		t.Fatal("pitfall never chosen on floor 9 (test is not meaningful)")
	}
}

// 9-B×9-C: 底抜けの壺は水路・割れた壁・隠し通路の上に罠を作らない。
func TestWave9BottomlessPotAvoidsTerrain(t *testing.T) {
	g := newFoundationTestGame()
	g.Floor = 5
	g.state.Map[4][5] = Tile{Type: "water", Blocked: true}
	g.state.Map[3][4] = Tile{Type: "cracked_wall", Blocked: true, BlockSight: true}
	g.state.Map[5][4] = Tile{Type: "hidden_corridor", Blocked: true, BlockSight: true}
	g.scatterBottomlessTraps(4, 4)
	if len(g.state.MapTraps) != 6 {
		t.Fatalf("traps = %d, want 6 (9 tiles minus 3 terrain)", len(g.state.MapTraps))
	}
	for _, trap := range g.state.MapTraps {
		if terrainBlocksWalking(g.state.Map[trap.Y][trap.X]) {
			t.Fatalf("trap on %s", g.state.Map[trap.Y][trap.X].Type)
		}
	}
}

// 10-A×10-B: 店とモンスターハウスが同じ階に出ても別の部屋になり、互いの部屋を荒らさない。
// 店主は眠らされず、モンスターハウスの道具は商品にならない。
func TestWave10ShopAndMonsterHouseShareFloors(t *testing.T) {
	both := 0
	for seed := int64(1); seed <= 150; seed++ {
		for floor := 4; floor < finalFloor; floor++ {
			if isBossFloor(floor) {
				continue
			}
			player := newInitialPlayer()
			_, enemies, items, _, rooms, _ := GenerateRandomMapFor(70, 70, floor-1, &player, newFloorRNG(seed, floor), DifficultyAdvanced)
			shop, house := -1, -1
			for i, r := range rooms {
				switch r.Role {
				case RoomRoleShop:
					if shop >= 0 {
						t.Fatalf("seed %d floor %d: two shops", seed, floor)
					}
					shop = i
				case RoomRoleMonsterHouse:
					if house >= 0 {
						t.Fatalf("seed %d floor %d: two monster houses", seed, floor)
					}
					house = i
				}
			}
			if shop < 0 || house < 0 {
				continue
			}
			both++
			s, h := rooms[shop], rooms[house]
			for _, e := range enemies {
				inShop, inHouse := roomContainsWithWalls(s, e.X, e.Y), roomContains(h, e.X, e.Y)
				if inShop && inHouse {
					t.Fatalf("seed %d floor %d: rooms overlap at (%d,%d)", seed, floor, e.X, e.Y)
				}
				if inShop && (e.ID != shopkeeperMonsterID || e.StatusAilments.Sleep != 0) {
					t.Fatalf("seed %d floor %d: enemy %d (sleep %d) in the shop", seed, floor, e.ID, e.StatusAilments.Sleep)
				}
				if e.ID == shopkeeperMonsterID && !inShop {
					t.Fatalf("seed %d floor %d: keeper outside the shop", seed, floor)
				}
			}
			for _, item := range items {
				x, y := item.GetPosition()
				if roomContains(h, x, y) && itemShopPrice(item) != 0 {
					t.Fatalf("seed %d floor %d: monster house item %d has a price", seed, floor, item.GetID())
				}
			}
		}
	}
	if both == 0 {
		t.Fatal("no floor had both a shop and a monster house (test is not meaningful)")
	}
}

// 10-A×10-B: 部屋の役割（店・モンスターハウス）と入室済みは同じ階でまとめて中断セーブされる。
func TestWave10ShopAndMonsterHouseSaveTogether(t *testing.T) {
	g := newShopTestGame(t)
	g.rooms[1].Role = RoomRoleMonsterHouse
	g.rooms[1].Entered = true
	pickUpShopProduct(t, g)
	restored, vs := saveRoundTrip(g)
	if len(vs) > 0 {
		t.Fatalf("round trip: %+v", vs)
	}
	if restored.rooms[0].Role != RoomRoleShop || restored.rooms[0].Entered {
		t.Fatalf("shop room = %+v", restored.rooms[0])
	}
	if restored.rooms[1].Role != RoomRoleMonsterHouse || !restored.rooms[1].Entered {
		t.Fatalf("monster house = %+v", restored.rooms[1])
	}
	if !restored.shopOpen() || restored.shopUnpaidTotal() != 200 {
		t.Fatalf("shop open %v unpaid %d", restored.shopOpen(), restored.shopUnpaidTotal())
	}
	// 店へ入り直しても「モンスターハウスだ！」は出ない
	restored.previousPlayerRoomID = -1
	before := len(restored.ActionQueue.Queue)
	restored.CheckSleepingEnemyWakeUp()
	if len(restored.ActionQueue.Queue) != before {
		t.Fatal("entering the shop must not announce a monster house")
	}
}

// 10-B×10-C: 店主・番犬も図鑑の敵の一覧に1回ずつ並ぶ（上位種化の表に無い1段階の敵）。
func TestWave10CodexListsShopMonsters(t *testing.T) {
	order := codexEnemyOrder()
	seen := map[int]int{}
	for _, id := range order {
		seen[id]++
	}
	if len(order) != len(MonsterDefinitions) {
		t.Fatalf("codex lists %d enemies, want %d", len(order), len(MonsterDefinitions))
	}
	for _, id := range []int{shopkeeperMonsterID, guardDogMonsterID, eliteGuardDogMonsterID} {
		if seen[id] != 1 {
			t.Fatalf("enemy %d listed %d times", id, seen[id])
		}
	}
}

// 10-B×10-C: 怒った店主に倒されると図鑑の店主に敗北が1つ記録され、店主を倒すと撃破が記録される。
func TestWave10CodexRecordsShopkeeper(t *testing.T) {
	t.Chdir(t.TempDir())
	g := newShopTestGame(t)
	g.persistent = true
	g.codexLoaded, g.codex = true, newCodexData()
	g.applyDamageToEnemy(g.shopkeeperIndex(), 1) // 怒らせる
	g.state.Player.X, g.state.Player.Y = 7, 3
	g.state.Player.Health = 1
	for i := 0; i < 5 && !g.RunEnded(); i++ {
		g.MoveEnemies()
		if err := g.RunUntilInputReady(600); err != nil {
			t.Fatal(err)
		}
	}
	if !g.RunEnded() {
		t.Fatal("angry keeper must defeat the player")
	}
	if r := g.codex.Enemies[shopkeeperMonsterID]; !r.Seen || r.Deaths != 1 {
		t.Fatalf("keeper record after death = %+v", r)
	}

	h := newShopTestGame(t)
	h.persistent = true
	h.codexLoaded, h.codex = true, newCodexData()
	h.applyDamageToEnemy(h.shopkeeperIndex(), 1)
	h.applyDamageToEnemy(h.shopkeeperIndex(), 10000)
	if err := h.RunUntilInputReady(200); err != nil {
		t.Fatal(err)
	}
	if r := h.codex.Enemies[shopkeeperMonsterID]; r.Defeated != 1 {
		t.Fatalf("keeper record after defeat = %+v", r)
	}
}

// --- 段11（wave11-A/B/C）の統合で足したテスト ---

// 10-B×wave11-C の決定: 持ち物が満杯でも店の中のお金（値段は付かない）を拾え、泥棒にならない。
func TestWave11FullInventoryPicksUpMoneyInShop(t *testing.T) {
	g := newShopTestGame(t)
	for i := 0; i < 20; i++ {
		g.state.Player.Inventory = append(g.state.Player.Inventory, buildItemFromTemplate(1, 0, 0))
	}
	money := newStolenCash(80)
	money.SetPosition(5, 4)
	g.state.Items = append(g.state.Items, money)
	if !g.MovePlayer(1, 0) {
		t.Fatal("移動できない")
	}
	runQueuedActions(g)
	if g.state.Player.Cash != 80 || g.itemOnFloor(money) || len(g.state.Player.Inventory) != 20 {
		t.Fatalf("所持金 %d・床に残った %v・持ち物 %d", g.state.Player.Cash, g.itemOnFloor(money), len(g.state.Player.Inventory))
	}
	if !g.shopOpen() || g.stats.ShopThefts != 0 || g.shopUnpaidTotal() != 0 {
		t.Fatal("店のお金を拾って泥棒・未払いになった")
	}
}

// wave11-B×wave11-C: expert は満杯でも見えているお金へ向かって拾い（道具を手放さない）、足元のお金と交換しない。
func TestWave11ExpertPicksUpMoneyWhenFull(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	for i := 0; i < 20; i++ {
		g.state.Player.Inventory = append(g.state.Player.Inventory, buildItemFromTemplate(2, 0, 0))
	}
	money := newStolenCash(30)
	money.SetPosition(5, 3)
	money.SetShowOnMiniMap(true)
	g.state.Items = []Item{money}
	if !b.worthPicking(money) {
		t.Fatal("満杯でお金を拾う価値が無いと判断した")
	}
	stepUntil(t, b, 12, func() bool { return g.state.Player.Cash > 0 })
	if g.state.Player.Cash != 30 || len(g.state.Player.Inventory) != 20 || countInventoryID(g, 2) != 20 {
		t.Fatalf("所持金 %d・持ち物 %d act=%s", g.state.Player.Cash, len(g.state.Player.Inventory), b.act)
	}
}

// wave11-B×wave11-C: 合成・強化の壺の作業は商売中の店の中では始めず（壺や中身が店の床に出るため）、店の外なら始める。
func TestWave11ExpertNoPotJobInsideShop(t *testing.T) {
	g := newShopTestGame(t)
	player := &g.state.Player
	weapon := buildItemFromTemplate(126, 0, 0).(*Weapon)
	weapon.Sharpness, weapon.Cursed = 1, false
	spare := buildItemFromTemplate(129, 0, 0).(*Weapon)
	spare.Sharpness, spare.Cursed = 2, false
	pot := buildItemFromTemplate(synthesisPotID, 0, 0).(*Pot)
	pot.Capacity, pot.Contents = 3, nil
	player.Inventory = []Item{weapon, spare, pot}
	player.EquippedWeapon = weapon
	b := newBot(g, g.input.(*VirtualInput), personaExpert, 1)
	if !g.inShopInterior(player.X, player.Y) || b.potJobCalm() {
		t.Fatal("店の中で壺の作業を始めてよいと判断した")
	}
	// 店の外（隣の部屋）では落ち着いていれば始める
	player.X, player.Y = 16, 6
	if !b.potJobCalm() {
		t.Fatal("店の外で壺の作業を始めない")
	}
	stepUntil(t, b, 8, func() bool { return b.potJob == nil && hasEquipmentAbility(weapon.Abilities, AbilityShellSlayer) })
	if !hasEquipmentAbility(weapon.Abilities, AbilityShellSlayer) || g.stats.ShopThefts != 0 {
		t.Fatalf("店の外で合成しない: act=%s", b.act)
	}
}

// wave11-A×wave11-B: モンスターハウスで囲まれたときに強化の薬を飲むのは、BOT が通知・混み具合で気づいた
// 部屋だけ（部屋の正体 Room.Role は読まない）。
func TestWave11HouseBuffUsesNoticedHouse(t *testing.T) {
	setup := func() (*Game, *bot) {
		g, b := newHouseTestGame(t, 0)
		g.rooms[0].Role = RoomRoleMonsterHouse
		g.state.Enemies = []Enemy{newActiveEnemy(0, 3, 4), newActiveEnemy(0, 5, 4)}
		g.state.Player.Inventory = []Item{buildItemFromTemplate(potionHasteID, 0, 0)}
		b.newMessages() // ここまでのメッセージは読んだことにする
		return g, b
	}
	g, b := setup()
	b.updateHouse()
	if done, err := b.expertBuffMoments(b.adjacentEnemies()); err != nil || done || countInventoryID(g, potionHasteID) != 1 {
		t.Fatalf("気づいていない部屋で強化の薬を飲んだ: act=%s", b.act)
	}
	g, b = setup()
	g.messageLog.Add(houseNotice)
	b.updateHouse()
	if done, err := b.expertBuffMoments(b.adjacentEnemies()); err != nil || !done {
		t.Fatalf("気づいたモンスターハウスで囲まれても強化の薬を飲まない: act=%s", b.act)
	}
	runQueuedActions(g)
	if countInventoryID(g, potionHasteID) != 0 || g.state.Player.StatusAilments.Haste == 0 {
		t.Fatal("倍速の薬が効いていない")
	}
}

// 段11の統合の TestSimBatch で見つけた泥棒（wave11-C の shopsafe の抜け道）: 食料のために満杯の持ち物を
// 手放すとき、足元がふさがっていると投げて捨てる（bot-food）が、向いている直線に店主がいれば投げない。
func TestWave11FoodDiscardThrowAvoidsShopkeeper(t *testing.T) {
	g := newShopTestGame(t)
	p := &g.state.Player
	p.X, p.Y, p.Direction = 4, 3, Right // 店主 (8,3) の方を向く
	for i := 0; i < 20; i++ {
		p.Inventory = append(p.Inventory, buildItemFromTemplate(35, 0, 0)) // 不利なカード（手放す候補）
	}
	g.state.Items = []Item{buildItemFromTemplate(20, 4, 3), buildItemFromTemplate(1, 16, 6)} // 足元の自分の道具・隣の部屋の食料
	g.state.Items[1].SetShowOnMiniMap(true)
	markAllVisited(g)
	b := newBot(g, g.input.(*VirtualInput), personaExpert, 1)
	if done, err := b.makeRoomForFood(); err != nil || done || len(p.Inventory) != 20 {
		t.Fatalf("店主の方へ道具を投げた: done=%v 持ち物 %d", done, len(p.Inventory))
	}
	p.Direction = Down
	if done, err := b.makeRoomForFood(); err != nil || !done {
		t.Fatalf("店主のいない向きでも手放さない: done=%v", done)
	}
	runQueuedActions(g)
	if g.stats.ShopThefts != 0 || !g.shopOpen() {
		t.Fatal("泥棒になった")
	}
}

// 同じく: 未払いの品を持つ間は、ふさがれた道を越える経路（敵・見つけた罠も通る）でも見つけた罠を踏まない
// （落とし穴で階を去ると泥棒）。払えば今までどおり罠も越える。
func TestWave11UnpaidBotKeepsOffKnownTraps(t *testing.T) {
	g := newShopTestGame(t)
	pit := createMapTrapByID(6, 11, 4) // 店の外の通路
	pit.Discovered = true
	g.state.MapTraps = []MapTrap{pit}
	b := newBot(g, g.input.(*VirtualInput), personaExpert, 1)
	pastPit := func(x, y int) bool { return x == 16 && y == 4 } // 通路の罠の先（隣の部屋）
	if _, _, ok := b.pathStepThrough(pastPit, true); !ok {
		t.Fatal("未払いが無いのに罠を越える経路が無い")
	}
	product := g.state.Items[0] // 店の商品を拾って未払いにする
	g.state.Items = nil
	g.state.Player.Inventory = []Item{product}
	if b.unpaidTotal() == 0 {
		t.Fatal("未払いになっていない")
	}
	if _, _, ok := b.pathStepThrough(pastPit, true); ok {
		t.Fatal("未払いのまま見つけた落とし穴へ向かった")
	}
	// 店を去らせない罠（鈍足の罠）なら未払いでも越える（罠と商品に囲まれて動けなくならないように）
	slow := createMapTrapByID(2, 11, 4)
	slow.Discovered = true
	g.state.MapTraps = []MapTrap{slow}
	if _, _, ok := b.pathStepThrough(pastPit, true); !ok {
		t.Fatal("未払いの間は危なくない罠も越えない")
	}
}

// 同じく: 店の床に貼りついた聖域のカードは拾えないので、買う品に選ばない（乗っては離れるを繰り返していた）。
func TestWave11ExpertDoesNotBuyStuckSanctuary(t *testing.T) {
	g := newShopTestGame(t)
	card := findSanctuaryCardForTest(t)
	card.Stuck = true
	card.SetPosition(2, 6)
	card.SetShowOnMiniMap(true)
	setItemShopPrice(card, itemBuyPrice(card))
	g.state.Items = []Item{card}
	g.state.Player.Cash = 100000
	b := newBot(g, g.input.(*VirtualInput), personaExpert, 1)
	if b.bestPurchase() != nil {
		t.Fatal("貼りついた聖域のカードを買いに行く")
	}
	card.Stuck = false
	if b.buyValue(card) > 0 && b.bestPurchase() != card {
		t.Fatal("貼りついていない聖域のカードは買う候補になる")
	}
}

func findSanctuaryCardForTest(t *testing.T) *Card {
	t.Helper()
	for _, id := range sortedTemplateIDs(itemTemplates, nil) {
		if card, ok := buildItemFromTemplate(id, 0, 0).(*Card); ok && card.Name == sanctuaryCardName {
			return card
		}
	}
	t.Fatal("聖域のカードが無い")
	return nil
}

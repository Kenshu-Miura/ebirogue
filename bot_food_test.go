//go:build !test

package main

import "testing"

// BOT の食料の扱いと段4の敵への対処のテスト（bot_test.go・bot_expert_test.go）。

// newBotFoodTestGame は 9x7 の部屋にプレイヤーだけを置いたヘッドレスゲームと、性格 p の BOT を作る。
func newBotFoodTestGame(t *testing.T, p persona) (*Game, *bot) {
	t.Helper()
	g, in := newHeadlessGame(5, DifficultyAdvanced)
	g.state.Map = makeTestFloorMap(9, 7)
	g.rooms = []Room{{ID: 0, X: 0, Y: 0, Width: 9, Height: 7, Center: Coordinate{X: 4, Y: 3}}}
	g.state.Enemies, g.state.Items, g.state.MapTraps = nil, nil, nil
	g.state.Player.X, g.state.Player.Y, g.state.Player.Direction = 1, 3, Right
	g.state.Player.Inventory = nil
	if err := g.RunUntilInputReady(100); err != nil {
		t.Fatal(err)
	}
	return g, newBot(g, in, p, 5)
}

const testFoodID = 1 // ウインナー

func placeVisibleItem(g *Game, item Item) {
	item.SetShowOnMiniMap(true)
	item.SetPlayerDiscovered(true)
	g.state.Items = append(g.state.Items, item)
}

func countInventoryID(g *Game, id int) int {
	n := 0
	for _, item := range g.state.Player.Inventory {
		if item.GetID() == id {
			n++
		}
	}
	return n
}

func stepUntil(t *testing.T, b *bot, limit int, done func() bool) {
	t.Helper()
	for i := 0; i < limit && !done(); i++ {
		if err := b.step(); err != nil {
			t.Fatal(err)
		}
	}
}

// 所持品が満杯でも、見えている食料のために価値の低い道具（不利なカード）を手放して食料を拾う。
// 回復の薬は手放さず、手放した道具は同じ階で拾い直さない。
func TestBotMakesRoomForVisibleFood(t *testing.T) {
	for _, p := range []persona{personaAggressive, personaCautious, personaItemUser, personaExpert} {
		t.Run(string(p), func(t *testing.T) {
			g, b := newBotFoodTestGame(t, p)
			player := &g.state.Player
			player.Inventory = append(player.Inventory, buildItemFromTemplate(2, 0, 0)) // 回復の薬
			for len(player.Inventory) < player.MaxInventory {
				player.Inventory = append(player.Inventory, buildItemFromTemplate(37, 0, 0)) // 地図忘却（不利）
			}
			placeVisibleItem(g, buildItemFromTemplate(testFoodID, 6, 3))
			stepUntil(t, b, 40, func() bool { return countInventoryID(g, testFoodID) > 0 })
			if countInventoryID(g, testFoodID) != 1 {
				t.Fatalf("food was not picked up; inventory=%d items=%d", len(player.Inventory), len(g.state.Items))
			}
			if countInventoryID(g, 2) != 1 {
				t.Fatal("healing potion was discarded")
			}
			if countInventoryID(g, 37) != player.MaxInventory-2 {
				t.Fatalf("expected exactly one harmful card discarded, have %d", countInventoryID(g, 37))
			}
			// 空きができても、手放したカードは拾い直さない
			player.Inventory = player.Inventory[:len(player.Inventory)-3]
			cards := countInventoryID(g, 37)
			for i := 0; i < 30; i++ {
				if err := b.step(); err != nil {
					t.Fatal(err)
				}
			}
			if countInventoryID(g, 37) != cards {
				t.Fatal("discarded card was picked up again")
			}
		})
	}
}

// 満杯で拾えなかった食料の上で空きができたら、足元のメニューから拾う。
func TestBotPicksUpFoodUnderfoot(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	placeVisibleItem(g, buildItemFromTemplate(testFoodID, 1, 3))
	done, err := b.pickUpHere()
	if err != nil {
		t.Fatal(err)
	}
	if !done || countInventoryID(g, testFoodID) != 1 || len(g.state.Items) != 0 {
		t.Fatalf("food underfoot was not picked up: done=%v inventory=%d items=%d", done, len(g.state.Player.Inventory), len(g.state.Items))
	}
	if b.windowOpen() {
		t.Fatal("menu left open")
	}
}

// 空腹で食料を持っていれば、腐ったウインナーより普通の食料を先に食べる。
func TestBotEatsGoodFoodFirst(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaAggressive)
	player := &g.state.Player
	player.Inventory = []Item{buildItemFromTemplate(rottenSausageItemID, 0, 0), buildItemFromTemplate(testFoodID, 0, 0)}
	player.Satiety = 5
	stepUntil(t, b, 3, func() bool { return countInventoryID(g, testFoodID) == 0 })
	if countInventoryID(g, testFoodID) != 0 || countInventoryID(g, rottenSausageItemID) != 1 {
		t.Fatal("did not eat the good food first")
	}
}

// 満腹度を余計に減らす盾は、盾が無くても装備しない。
func TestBotSkipsSatietyHarmfulEquipment(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaAggressive)
	g.state.Player.Inventory = []Item{buildItemFromTemplate(69, 0, 0)} // 重甲の盾
	if done, err := b.equipUpgrades(); done || err != nil {
		t.Fatalf("equipped satiety-harmful shield: done=%v err=%v", done, err)
	}
}

// 段4の敵の見分けは人に見える姿で行う（化けたミミックダコは化けた姿）。
func TestStage4EnemyAppearanceHelpers(t *testing.T) {
	if !isReflectorEnemy(CreateEnemyByID(123, 0, 0)) || isReflectorEnemy(CreateEnemyByID(0, 0, 0)) {
		t.Fatal("reflector detection")
	}
	mimic := CreateEnemyByID(117, 0, 0)
	disguiseEnemyAs(&mimic, 124)
	if !isReflectorEnemy(mimic) {
		t.Fatal("mimic disguised as a reflector should be treated as one")
	}
	fugu := CreateEnemyByID(108, 0, 0)
	ray := CreateEnemyByID(114, 0, 0)
	if isHazardEnemy(fugu) || isHazardEnemy(ray) {
		t.Fatal("not inflated / not charged should not be hazards")
	}
	fugu.Flag, ray.Flag = true, true
	if !isHazardEnemy(fugu) || !isHazardEnemy(ray) {
		t.Fatal("inflated fugu / charged ray should be hazards")
	}
}

// 膨らんだフクレフグの隣では、一撃で倒せないなら殴らずに離れる。
func TestExpertStepsAwayFromInflatedFugu(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	g.state.Player.X = 4
	fugu := CreateEnemyByID(108, 5, 3)
	fugu.Health, fugu.MaxHealth, fugu.Flag = 500, 1000, true
	g.state.Enemies = []Enemy{fugu}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if g.state.Enemies[0].Health != 500 {
		t.Fatal("attacked the inflated fugu")
	}
	p := g.state.Player
	if max(abs(p.X-g.state.Enemies[0].X), abs(p.Y-g.state.Enemies[0].Y)) <= 1 {
		t.Fatalf("did not step away: player=(%d,%d) act=%s", p.X, p.Y, b.act)
	}
}

// ウチカエシエビ系には矢を撃たない。
func TestExpertDoesNotShootReflector(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	arrow := buildItemFromTemplate(6, 0, 0).(*Arrow)
	arrow.ShotCount = 5
	g.state.Player.Inventory = []Item{arrow}
	g.state.Player.EquippedArrow = arrow
	enemy := CreateEnemyByID(125, 5, 3)
	enemy.StatusAilments.Sleep = 50
	g.state.Enemies = []Enemy{enemy}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if arrow.ShotCount != 5 || b.act == "shoot" {
		t.Fatalf("shot at a reflector: shots=%d act=%s", arrow.ShotCount, b.act)
	}
}

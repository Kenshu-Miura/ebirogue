//go:build !test

package main

import (
	"fmt"
	"testing"
)

// BOT のテストプレイ（TestSimSmoke / TestSimBatch）で見つかった不具合の回帰テスト。

// 所持品が1個だけのときに「使う」と、空になった所持品を参照して panic していた。
func TestUsingOnlyInventoryItemDoesNotPanic(t *testing.T) {
	for _, id := range randomItemTemplateIDs {
		id := id
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			g, _ := newHeadlessGame(1, DifficultyAdvanced)
			item := g.createItemByID(id, 0, 0)
			if _, ok := item.(Equipable); ok {
				return
			}
			if _, ok := item.(*Pot); ok {
				return
			}
			g.state.Player.Inventory = []Item{item}
			g.showInventory, g.showItemActions = true, true
			g.selectedItemIndex, g.selectedActionIndex = 0, 0
			g.executeAction()
			if err := g.RunUntilInputReady(2000); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// 部屋の数が少ないと、敵がプレイヤーと同じマスに配置されることがあった。
func TestGeneratedEnemiesNeverStartOnPlayer(t *testing.T) {
	for seed := int64(0); seed < 200; seed++ {
		for _, floor := range []int{1, 4, 10} {
			player := newInitialPlayer()
			_, enemies, _, _, _, _ := GenerateRandomMap(70, 70, floor-1, &player, newFloorRNG(seed, floor))
			for _, enemy := range enemies {
				if enemy.X == player.X && enemy.Y == player.Y {
					t.Fatalf("seed %d floor %d: %s placed on the player", seed, floor, enemy.Name)
				}
			}
		}
	}
}

// 通路の外側（壁ではない通れないマス）を投擲物が突き抜け、マップ外を参照して panic していた。
func TestThrowStopsAtBlockedTilesAndMapEdge(t *testing.T) {
	g, _ := newHeadlessGame(1, DifficultyAdvanced)
	m := make([][]Tile, 5)
	for y := range m {
		m[y] = make([]Tile, 5)
		for x := range m[y] {
			m[y][x] = Tile{Type: "other", Blocked: true, BlockSight: true}
		}
	}
	m[2][1], m[2][2], m[2][3] = Tile{Type: "corridor"}, Tile{Type: "corridor"}, Tile{Type: "corridor"}
	g.state.Map = m
	g.rooms = nil
	g.state.Enemies, g.state.Items, g.state.MapTraps = nil, nil, nil
	g.state.Player.X, g.state.Player.Y, g.state.Player.Direction = 2, 2, Up
	item := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{item}
	g.selectedItemIndex = 0
	g.throwWithCallbacks(item, 10)
	if err := g.RunUntilInputReady(2000); err != nil {
		t.Fatal(err)
	}
	for _, floorItem := range g.state.Items {
		x, y := floorItem.GetPosition()
		if y < 0 || y >= len(m) || x < 0 || x >= len(m[y]) || m[y][x].Blocked {
			t.Fatalf("thrown item landed at (%d, %d)", x, y)
		}
	}
}

// あやつりクラゲの「勝手に移動」が、予約後に別の敵が入ったマスへプレイヤーを動かしていた。
func TestForcedMoveRechecksDestination(t *testing.T) {
	g, _ := newHeadlessGame(1, DifficultyAdvanced)
	g.state.Map = makeTestFloorMap(5, 5)
	g.rooms = []Room{{ID: 0, X: 0, Y: 0, Width: 5, Height: 5}}
	g.state.Items, g.state.MapTraps = nil, nil
	g.state.Player.X, g.state.Player.Y = 2, 2
	jelly := CreateEnemyByID(10, 1, 1)
	g.state.Enemies = []Enemy{jelly}
	enqueueForcedMove(&g.state.Enemies[0], g, []Coordinate{{X: 3, Y: 2}}, func(int) int { return 0 })
	// 実行前に別の敵が移動先へ入る
	g.state.Enemies = append(g.state.Enemies, CreateEnemyByID(0, 3, 2))
	if err := g.RunUntilInputReady(2000); err != nil {
		t.Fatal(err)
	}
	if g.state.Player.X != 2 || g.state.Player.Y != 2 {
		t.Fatalf("player was forced onto an occupied tile: (%d, %d)", g.state.Player.X, g.state.Player.Y)
	}
}

// 足元のアイテムを投げると、足元ではなく所持品の選択位置のアイテムが消えていた（所持品が空なら panic）。
// FuzzInputSequence が見つけた不具合。
func TestThrowingGroundItemKeepsInventory(t *testing.T) {
	for _, inventorySize := range []int{0, 1} {
		g, _ := newHeadlessGame(1, DifficultyAdvanced)
		g.state.Map = makeTestFloorMap(7, 5)
		g.rooms = []Room{{ID: 0, X: 0, Y: 0, Width: 7, Height: 5}}
		g.state.Enemies, g.state.MapTraps = nil, nil
		g.state.Player.X, g.state.Player.Y, g.state.Player.Direction = 2, 2, Right
		owned := buildItemFromTemplate(2, 0, 0)
		g.state.Player.Inventory = []Item{owned}[:inventorySize]
		ground := buildItemFromTemplate(1, 2, 2)
		g.state.Items = []Item{ground}
		g.currentGroundItem = ground
		g.ShowGroundItem, g.GroundItemActioned = true, true
		g.selectedGroundActionIndex = 3 // 投げる
		g.executeGroundItemAction()
		if err := g.RunUntilInputReady(2000); err != nil {
			t.Fatal(err)
		}
		if len(g.state.Player.Inventory) != inventorySize {
			t.Fatalf("inventory size %d -> %d", inventorySize, len(g.state.Player.Inventory))
		}
		if len(g.state.Items) != 1 {
			t.Fatalf("ground items = %d, want only the thrown item", len(g.state.Items))
		}
		if x, y := g.state.Items[0].GetPosition(); x == 2 && y == 2 {
			t.Fatal("thrown item should have left the player's tile")
		}
	}
}

// 足元のお金を「使う」と、所持品の選択位置をお金として扱い panic していた。FuzzInputSequence が発見。
func TestUsingGroundMoneyAddsCash(t *testing.T) {
	for _, inventorySize := range []int{0, 1} {
		g, _ := newHeadlessGame(1, DifficultyAdvanced)
		g.state.Enemies, g.state.MapTraps = nil, nil
		px, py := g.state.Player.X, g.state.Player.Y
		owned := buildItemFromTemplate(1, 0, 0)
		g.state.Player.Inventory = []Item{owned}[:inventorySize]
		var moneyID int
		for _, id := range randomItemTemplateIDs {
			if _, ok := buildItemFromTemplate(id, 0, 0).(*Money); ok {
				moneyID = id
			}
		}
		cash := buildItemFromTemplate(moneyID, px, py).(*Money)
		cash.Amount = 123
		g.state.Items = []Item{cash}
		g.currentGroundItem = cash
		g.ShowGroundItem, g.GroundItemActioned = true, true
		g.selectedGroundActionIndex = 2 // 使う
		g.executeGroundItemAction()
		if err := g.RunUntilInputReady(2000); err != nil {
			t.Fatal(err)
		}
		if g.state.Player.Cash != 123 || len(g.state.Items) != 0 || len(g.state.Player.Inventory) != inventorySize {
			t.Fatalf("cash %d, ground items %d, inventory %d", g.state.Player.Cash, len(g.state.Items), len(g.state.Player.Inventory))
		}
	}
}

// 目潰し中の敵どうしの攻撃が予約時の添字とポインタを使っていたため、先の攻撃で敵が倒れて並びが変わると、
// ダメージが別の敵に入ったり HP 0 以下の敵が残ったりしていた。TestSimBatch が発見。
func TestBlindEnemyAttacksResolveTargetsAtExecution(t *testing.T) {
	g, _ := newHeadlessGame(1, DifficultyAdvanced)
	g.state.Map = makeTestFloorMap(9, 7)
	g.state.Items, g.state.MapTraps = nil, nil
	g.state.Player.X, g.state.Player.Y = 7, 5
	g.randHooks.damage = func(n int) int { return n - 1 }
	attacker1, victim1 := CreateEnemyByID(5, 1, 1), CreateEnemyByID(0, 2, 1)
	attacker2, victim2 := CreateEnemyByID(5, 1, 3), CreateEnemyByID(0, 2, 3)
	victim1.Health = 1
	victim2.Health, victim2.MaxHealth = 500, 500
	g.state.Enemies = []Enemy{attacker1, victim1, attacker2, victim2}
	g.AttackEnemyFromBlindEnemy(0, 1)
	g.AttackEnemyFromBlindEnemy(2, 3)
	if err := g.RunUntilInputReady(2000); err != nil {
		t.Fatal(err)
	}
	for _, enemy := range g.state.Enemies {
		if enemy.Health <= 0 {
			t.Fatalf("%s remains with HP %d", enemy.Name, enemy.Health)
		}
	}
	if i := g.enemyIndexAt(2, 3); i < 0 || g.state.Enemies[i].Health >= 500 {
		t.Fatal("the second attack should damage its own target")
	}
	if g.enemyIndexAt(2, 1) >= 0 {
		t.Fatal("the first victim should be defeated")
	}
}

// 炸裂装甲のカードの撃破が予約時の添字で行われ、同じ敵ターンに複数の敵が攻撃すると何度も発動して panic していた。
// expert の BOT が発見。罠は1回で消費する。
func TestExplosiveArmorTrapTriggersOnce(t *testing.T) {
	g, _ := newHeadlessGame(1, DifficultyAdvanced)
	g.state.Map = makeTestFloorMap(7, 7)
	g.state.Items, g.state.MapTraps = nil, nil
	g.state.Player.X, g.state.Player.Y = 3, 3
	g.state.Player.SetTrap = buildItemFromTemplate(8, 0, 0)
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 2, 3), CreateEnemyByID(0, 4, 3), CreateEnemyByID(0, 3, 2)}
	for i := range g.state.Enemies {
		g.AttackFromEnemy(i)
	}
	if err := g.RunUntilInputReady(2000); err != nil {
		t.Fatal(err)
	}
	if len(g.state.Enemies) != 2 || g.state.Player.SetTrap != nil {
		t.Fatalf("enemies left %d (want 2), trap %v", len(g.state.Enemies), g.state.Player.SetTrap)
	}
}

// 足元の道具を拾う行動がキューに残ったまま落とし穴で階を移動すると、
// 古い階の添字で新しい階の道具一覧を削除して panic していた（seed=121 itemUser 中級）。
func TestPitfallWaitsForQueuedPickup(t *testing.T) {
	g, _ := newHeadlessGame(121, DifficultyIntermediate)
	px, py := g.state.Player.X, g.state.Player.Y
	// 足元のウインナー（ID 1）を新しい階の道具数より後ろの添字に置く
	for i := 0; i < 40; i++ {
		g.state.Items = append(g.state.Items, g.createItemByID(1, 0, 0))
	}
	item := g.createItemByID(1, px, py)
	g.state.Items = append(g.state.Items, item)
	g.state.Player.Inventory = nil
	floor := g.Floor

	// 実際の順序どおり、罠の発動の後に拾う行動が積まれる
	g.steppedTrapSource = trapSource(pitfallTrapID, "落とし穴")
	pitfallTrapEffect(g)
	g.PickupItem()
	g.isActioned = true
	if err := g.RunUntilInputReady(3000); err != nil {
		t.Fatal(err)
	}
	if g.Floor != floor+1 {
		t.Fatalf("floor = %d, want %d", g.Floor, floor+1)
	}
	if len(g.state.Player.Inventory) != 1 || g.state.Player.Inventory[0] != item {
		t.Fatalf("pickup before falling failed: %v", g.state.Player.Inventory)
	}
}

// tapReleased はキーを1フレーム押し、離したフレームも進めてから入力待ちまで進める（続けて同じキーを押せる）。
func tapReleased(t *testing.T, g *Game, v *VirtualInput, k Key) {
	t.Helper()
	tapHeadless(t, g, v, k)
	if err := g.StepFrame(); err != nil {
		t.Fatal(err)
	}
	if err := g.RunUntilInputReady(600); err != nil {
		t.Fatal(err)
	}
}

// 白紙のカードを持ち物の行動メニューの「使う」で使うと、決定の Z が同じフレームで書き込み画面にも渡り、
// 一覧の先頭のカードが必ず書き込まれていた。expert BOT の改善（bot-tactics）で発見。
func TestBlankCardMenuIgnoresOpeningKey(t *testing.T) {
	g, v := newHeadlessGame(1, DifficultyAdvanced)
	g.state.Enemies, g.state.Items, g.state.MapTraps = nil, nil, nil
	card := buildItemFromTemplate(cardBlankID, 0, 0).(*Card)
	g.state.Player.Inventory = []Item{card}
	if err := g.RunUntilInputReady(600); err != nil {
		t.Fatal(err)
	}
	for _, k := range []Key{KeyC, KeyZ, KeyZ, KeyZ} { // メニュー → 道具 → 行動メニュー → 使う
		tapReleased(t, g, v, k)
	}
	if !g.showBlankCardMenu || card.GetID() != cardBlankID {
		t.Fatalf("menu open=%v, card id=%d; want the menu open and the card still blank", g.showBlankCardMenu, card.GetID())
	}
	tapReleased(t, g, v, KeyDown)
	tapReleased(t, g, v, KeyZ)
	if want := blankCardOptionIDs()[1]; card.GetID() != want || g.showBlankCardMenu {
		t.Fatalf("card id=%d, menu open=%v; want %d written", card.GetID(), g.showBlankCardMenu, want)
	}
}

// 足元の道具を「交換」した後も GroundItemActioned が残り、次に持ち物の道具を使うと床の道具を参照して
// panic していた（床に道具が無いと index out of range）。expert BOT の改善（bot-tactics）で発見。
func TestUsingInventoryItemAfterGroundSwap(t *testing.T) {
	g, v := newHeadlessGame(1, DifficultyAdvanced)
	g.state.Map = makeTestFloorMap(7, 5)
	g.rooms = []Room{{ID: 0, X: 0, Y: 0, Width: 7, Height: 5}}
	g.state.Enemies, g.state.MapTraps = nil, nil
	g.state.Player.X, g.state.Player.Y, g.state.Player.Direction = 2, 2, Right
	g.state.Player.Satiety = 50
	sausage := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{buildItemFromTemplate(7, 0, 0), sausage} // 黒炎弾・ウインナー
	g.state.Items = []Item{buildItemFromTemplate(2, 2, 2)}                     // 足元に回復の薬
	if err := g.RunUntilInputReady(600); err != nil {
		t.Fatal(err)
	}
	// メニュー → 足元 → 行動を開く → 交換 → 持ち物の先頭（黒炎弾）と交換
	for _, k := range []Key{KeyC, KeyRight, KeyZ, KeyDown, KeyZ, KeyZ} {
		tapReleased(t, g, v, k)
	}
	if countInventoryID(g, 2) != 1 || countInventoryID(g, 7) != 0 {
		t.Fatalf("swap did not happen: inventory=%d ground=%d", len(g.state.Player.Inventory), len(g.state.Items))
	}
	if g.GroundItemActioned {
		t.Fatal("GroundItemActioned left true after the swap")
	}
	g.state.Items = nil // 床に道具が無いと、残った状態で panic していた
	// メニュー → 道具 → ウインナー（2番目）→ 行動メニュー → 食べる
	for _, k := range []Key{KeyC, KeyZ, KeyDown, KeyZ, KeyZ} {
		tapReleased(t, g, v, k)
	}
	if countInventoryID(g, 1) != 0 || g.state.Player.Satiety <= 50 {
		t.Fatalf("sausage not eaten: satiety=%d inventory=%d", g.state.Player.Satiety, len(g.state.Player.Inventory))
	}
}

// 真実の眼のカード（11）は、全識別に外れたあと対象選択を取り消すとカードもターンも消費されず、
// 使い直して全識別の判定（1/5）を何度でも振り直せた。判定は対象を確定したときだけ行う。
func newIdentifyCardTestGame(seed int64) (*Game, Item, Item) {
	g, _ := newHeadlessGame(seed, DifficultyAdvanced)
	weapon := g.createItemByID(20, 0, 0)
	cane := g.createItemByID(9, 0, 0)
	g.state.Player.Inventory = []Item{weapon, cane, g.createItemByID(11, 0, 0)}
	return g, weapon, cane
}

func useIdentifyCardFromInventory(g *Game) {
	g.showInventory, g.showItemActions = true, true
	g.selectedItemIndex, g.selectedActionIndex = 2, 0
	g.executeAction()
}

func TestIdentifyCardCancelCannotReroll(t *testing.T) {
	g, weapon, cane := newIdentifyCardTestGame(1)
	before, err := g.rng.MarshalState()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		useIdentifyCardFromInventory(g)
		if !g.useIdentifyItem || g.isActioned || len(g.ActionQueue.Queue) != 0 {
			t.Fatalf("use %d: selection must open without a turn or message", i)
		}
		wave3Press(t, g, KeyX)
		if g.useIdentifyItem || g.isActioned || len(g.state.Player.Inventory) != 3 {
			t.Fatalf("cancel %d consumed the card or a turn", i)
		}
	}
	after, err := g.rng.MarshalState()
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("取り消しても全識別の判定の乱数が進んではいけない")
	}
	if weapon.(Identifiable).GetIdentified() || cane.(Identifiable).GetIdentified() {
		t.Fatal("取り消しで識別されてはいけない")
	}

	// 取り消しを挟んでも、確定したときの結果は取り消さずに使った場合と同じになる。
	useIdentifyCardFromInventory(g)
	g.selectedItemIndex = 0
	wave3Press(t, g, KeyZ)
	ref, refWeapon, refCane := newIdentifyCardTestGame(1)
	useIdentifyCardFromInventory(ref)
	ref.selectedItemIndex = 0
	wave3Press(t, ref, KeyZ)
	if got, want := runQueuedActions(g), runQueuedActions(ref); got != want {
		t.Fatalf("messages = %q, want %q", got, want)
	}
	if weapon.(Identifiable).GetIdentified() != refWeapon.(Identifiable).GetIdentified() ||
		cane.(Identifiable).GetIdentified() != refCane.(Identifiable).GetIdentified() {
		t.Fatal("identification result differs after cancelling")
	}
	if !g.isActioned || g.useIdentifyItem || len(g.state.Player.Inventory) != 2 {
		t.Fatal("confirming must consume the card and a turn")
	}
}

func TestIdentifyCardConfirmRollsOnce(t *testing.T) {
	sawAll, sawOne := false, false
	for seed := int64(1); seed <= 200 && !(sawAll && sawOne); seed++ {
		g, weapon, cane := newIdentifyCardTestGame(seed)
		useIdentifyCardFromInventory(g)
		g.selectedItemIndex = 0
		wave3Press(t, g, KeyZ)
		runQueuedActions(g)
		if !weapon.(Identifiable).GetIdentified() {
			t.Fatalf("seed %d: the chosen item must be identified", seed)
		}
		if cane.(Identifiable).GetIdentified() {
			sawAll = true
		} else {
			sawOne = true
		}
	}
	if !sawAll || !sawOne {
		t.Fatalf("both outcomes must occur: all=%v one=%v", sawAll, sawOne)
	}
}

func TestIdentifyCardFromGround(t *testing.T) {
	g, weapon, cane := newIdentifyCardTestGame(1)
	g.state.Player.Inventory = []Item{weapon, cane}
	card := buildItemFromTemplate(11, g.state.Player.X, g.state.Player.Y)
	g.state.Items = []Item{card}
	g.selectedGroundItemIndex = 0
	g.GroundItemActioned = true
	g.selectedGroundActionIndex = 2
	g.executeGroundItemAction()
	if !g.useIdentifyItem || g.isActioned || !g.GroundItemActioned {
		t.Fatal("ground card must open the selection without a turn")
	}
	wave3Press(t, g, KeyX)
	if g.useIdentifyItem || g.isActioned || g.GroundItemActioned || len(g.state.Items) != 1 {
		t.Fatal("cancel must keep the ground card")
	}

	g.GroundItemActioned = true
	g.selectedGroundActionIndex = 2
	g.executeGroundItemAction()
	g.selectedItemIndex = 1
	wave3Press(t, g, KeyZ)
	runQueuedActions(g)
	if len(g.state.Items) != 0 || len(g.state.Player.Inventory) != 2 || !g.isActioned {
		t.Fatalf("ground card must be consumed instead of an inventory item: items=%d inv=%d actioned=%v", len(g.state.Items), len(g.state.Player.Inventory), g.isActioned)
	}
	if !cane.(Identifiable).GetIdentified() {
		t.Fatal("the chosen inventory item must be identified")
	}
}

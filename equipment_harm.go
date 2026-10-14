//go:build !test

package main

import "fmt"

// 装備・持ち物を損なう共通処理（段3の共通基盤）: 強化値を下げる（さび）・装備を弾き飛ばす・
// 持ち物を投げ捨てる。どれも Action の Execute の中から呼ぶ（その場で状態を変え、メッセージを積む）。
// 防ぐ能力はすべて statusImmunities（g.playerImmunityMessage）を通す。

// itemTossDistance は敵に投げ捨てられた持ち物が飛ぶマス数。
const itemTossDistance = 10

// itemFlingSearchRadius は飛んだ先に置けないとき、置き場所を探す距離。
const itemFlingSearchRadius = 2

// rustPlayerEquipment は装備中の武器（rustWeapon）か盾（rustArmor）の強化値を amount 下げ、
// 能力値へ即時反映する。さびない装備品（さび止め済み・AbilityRustImmune）と、さびを防ぐ能力
// （StatusRust の無効化）なら下げない。下げたら true。メッセージは「〜が錆びてしまった」「しかし〜は錆びなかった」。
func (g *Game) rustPlayerEquipment(target rustTarget, amount int) bool {
	player := &g.state.Player
	var name string
	var sharpness, stat *int
	var resists bool
	switch target {
	case rustWeapon:
		if player.EquippedWeapon == nil {
			return false
		}
		weapon := player.EquippedWeapon
		name, sharpness, stat = weapon.GetName(), &weapon.Sharpness, &player.AttackPower
		resists = itemResistsRust(weapon.RustProof, weapon.Abilities)
	case rustArmor:
		if player.EquippedArmor == nil {
			return false
		}
		armor := player.EquippedArmor
		name, sharpness, stat = armor.GetName(), &armor.Sharpness, &player.DefensePower
		resists = itemResistsRust(armor.RustProof, armor.Abilities)
	default:
		g.EnqueueMessage("しかし何も起こらなかった", 0.4)
		return false
	}
	if resists || g.playerImmuneTo(StatusRust) {
		g.EnqueueMessage(fmt.Sprintf("しかし%sは錆びなかった", name), 0.4)
		return false
	}
	*sharpness -= amount
	// 装備中の強化値の低下を攻撃力・防御力へ即時反映する
	*stat -= amount
	g.EnqueueMessage(fmt.Sprintf("%sが錆びてしまった", name), 0.4)
	return true
}

// flingItem は所持品・装備から外した道具 item を (fromX, fromY) から (dx, dy) 方向へ最大 distance マス飛ばし、
// 落ちた位置を返す。敵や海老さんの上は素通りし、壁の手前で止まる。止まったマスに置けなければ
// 近い順（2マス以内）に空いた床を探す。置ける場所が無ければ消えて false。
func (g *Game) flingItem(item Item, fromX, fromY, dx, dy, distance int) (Coordinate, bool) {
	path, _ := displacementPath(fromX, fromY, dx, dy, distance, g.terrainOpenAt, func(int, int) bool { return false })
	landing := Coordinate{X: fromX, Y: fromY}
	if len(path) > 0 {
		landing = path[len(path)-1]
	}
	for _, offset := range nearestFreeOffsets(itemFlingSearchRadius) {
		x, y := landing.X+offset.X, landing.Y+offset.Y
		if g.itemDropFree(x, y) {
			position := Coordinate{X: x, Y: y}
			g.dropItemFromTrap(item, position)
			return position, true
		}
	}
	g.removeThrownItemFromSource(item)
	return Coordinate{}, false
}

// knockOffPlayerEquipment は装備中の武器か盾 item を外し、(dx, dy) 方向へ distance マス弾き飛ばす。
// 呪われていても外れる（装備はずしの罠と同じ。呪いは道具に残る）。弾かれるのを防ぐ能力
// （StatusKnockOff の無効化）なら何もしない。弾き飛ばしたら true。
func (g *Game) knockOffPlayerEquipment(item Equipable, dx, dy, distance int) bool {
	if guard, immune := g.playerImmunityMessage(StatusKnockOff); immune {
		g.EnqueueMessage(guard, 0.4)
		return false
	}
	player := &g.state.Player
	if !player.IsEquipped(item) {
		return false
	}
	if _, err := player.UnequipItem(item); err != nil {
		return false
	}
	name := item.GetName()
	if _, landed := g.flingItem(item, player.X, player.Y, dx, dy, distance); landed {
		g.EnqueueMessage(fmt.Sprintf("%sが弾き飛ばされた", name), 0.4)
	} else {
		g.EnqueueMessage(fmt.Sprintf("%sが弾き飛ばされ、どこかへ消えた", name), 0.4)
	}
	return true
}

// tossPlayerItem は装備していない持ち物を1つ選び、(fromX, fromY) からランダムな方向へ遠くへ投げ捨てる。
// prefer を満たす持ち物があればその中から選ぶ（nil なら全部から）。床に貼りついた聖域のカードは選ばない。
// 投げ捨てを防ぐ能力（StatusItemToss の無効化）なら何もしない。投げ捨てた道具を返す。
// 対象と方向の抽選は intn（敵の行動なら g.randIntn）。
func (g *Game) tossPlayerItem(fromX, fromY int, prefer func(Item) bool, intn func(int) int) (Item, bool) {
	if guard, immune := g.playerImmunityMessage(StatusItemToss); immune {
		g.EnqueueMessage(guard, 0.4)
		return nil, false
	}
	player := &g.state.Player
	tossable := func(item Item) bool {
		return !isEquippedByPlayer(player, item) && !isStuckSanctuaryItem(item)
	}
	index, ok := -1, false
	if prefer != nil {
		index, ok = chooseInventoryIndex(player, func(item Item) bool { return tossable(item) && prefer(item) }, intn)
	}
	if !ok {
		index, ok = chooseInventoryIndex(player, tossable, intn)
	}
	if !ok {
		return nil, false
	}
	item := player.Inventory[index]
	directions := getDirections()
	direction := directions[intn(len(directions))]
	itemName, _ := displayItemName(item)
	if _, landed := g.flingItem(item, fromX, fromY, direction.dx, direction.dy, itemTossDistance); landed {
		g.EnqueueMessage(fmt.Sprintf("%sは遠くへ投げ捨てられた", itemName), 0.4)
	} else {
		g.EnqueueMessage(fmt.Sprintf("%sは投げ捨てられ、どこかへ消えた", itemName), 0.4)
	}
	return item, true
}

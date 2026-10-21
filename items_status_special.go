//go:build !test

package main

import "fmt"

var specialStatusItemTemplates = map[int]ItemTemplate{
	215: specialPotionTemplate(215, "暴れ薬", "15ターン暴走し、攻撃力が1.5倍になる。敵に投げると仲間どうしで争う。", StatusBerserk, 15),
	216: specialPotionTemplate(216, "透明の薬", "30ターン透明になり、隣接していない敵に追われなくなる。", StatusInvisible, 30),
	217: specialPotionTemplate(217, "無敵の薬", "10ターンすべてのダメージを防ぐ。状態異常は防げない。", StatusInvincible, 10),
	218: specialPotionTemplate(218, "にぎりの薬", "20ターンにぎり寿司になり、道具が使えず防御力が0になる。敵に投げるとウインナーに変える。", StatusNigiri, 20),
	219: specialCaneTemplate(219, "身代わりの杖", "30ターン敵を動かない身代わりにする。ほかの敵の攻撃を引きつける。ボスには効かない。"),
	220: specialCaneTemplate(220, "透明の杖", "30ターン敵を透明にする。気配察知・透視の指輪で見える。"),
	221: specialCaneTemplate(221, "暴走の杖", "15ターン敵を暴走させ、攻撃力を1.5倍にして近くの相手を襲わせる。"),
}

func specialPotionTemplate(id int, name, description string, kind StatusKind, turns int) ItemTemplate {
	return ItemTemplate{ID: id, ItemType: "Potion", Type: "Potion", Name: name, Description: description, Char: '!', UseActions: map[string]UseAction{"RestoreHealth": func(g *Game) {
		if g.rejectSpecialItemUse() {
			return
		}
		item, inventory := determineItemSource(g)
		g.Enqueue(Action{Duration: statusMessageDuration, Message: fmt.Sprintf("%sを飲んだ", item.GetName()), Execute: func(g *Game) { g.applyPlayerStatus(kind, turns, StatusSource{}) }})
		removeUsedItem(g, inventory)
	}}}
}
func specialCaneTemplate(id int, name, description string) ItemTemplate {
	return ItemTemplate{ID: id, ItemType: "Cane", Type: "Cane", Name: name, Description: description, Char: '!', UseActions: map[string]UseAction{"CaneEffect": func(g *Game) { g.queueSpecialCaneStatus(id, g.TargetEnemyIndex) }}}
}

func specialItemStatus(id int) (StatusKind, int) {
	switch id {
	case 215, 221:
		return StatusBerserk, 15
	case 216, 220:
		return StatusInvisible, 30
	case 217:
		return StatusInvincible, 10
	case 218:
		return StatusNigiri, 20
	case 219:
		return StatusDecoy, 30
	}
	return "", 0
}

// 投げる・杖の魔法弾の両方から呼ぶ。遅延実行時にはIDと座標で標的を引き直す。
func (g *Game) specialStatusItemHit(target Character, item Item, index int) bool {
	kind, turns := specialItemStatus(item.GetID())
	if kind == "" {
		return false
	}
	_, player := target.(*Player)
	id, x, y := 0, 0, 0
	if !player {
		if index < 0 || index >= len(g.state.Enemies) {
			return true
		}
		e := g.state.Enemies[index]
		id, x, y = e.ID, e.X, e.Y
	}
	g.Enqueue(Action{Duration: statusMessageDuration, Message: fmt.Sprintf("%sに%sが当たった", target.GetName(), item.GetName()), Execute: func(g *Game) {
		if player {
			if kind != StatusDecoy {
				g.applyPlayerStatus(kind, turns, StatusSource{})
			}
			return
		}
		i := g.findEnemyForDefense(id, x, y)
		if i < 0 {
			return
		}
		if item.GetID() == 217 {
			g.EnqueueMessage("しかし何も起こらなかった", statusMessageDuration)
			return
		}
		if item.GetID() == 218 {
			if MonsterDefinitions[id].IsBoss {
				g.EnqueueMessage("ボスには効かなかった", statusMessageDuration)
				return
			}
			g.dropEnemyHeldItem(i)
			g.state.Enemies = append(g.state.Enemies[:i], g.state.Enemies[i+1:]...)
			food := g.createItemByID(1, x, y)
			g.placeSpecialFood(food, x, y)
			g.miniMapDirty = true
			return
		}
		if g.applyEnemyStatus(i, kind, turns) {
			g.EnqueueMessage(fmt.Sprintf("%sは%s状態になった", g.state.Enemies[i].Name, statusKindLabel(kind)), statusMessageDuration)
		} else {
			g.EnqueueMessage("しかし何も起こらなかった", statusMessageDuration)
		}
		g.miniMapDirty = true
	}})
	return true
}

func init() {
	expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "特殊な状態", Lines: []string{
		"にぎり：20ターン道具が使えず防御力0。",
		"攻撃を受けるたび満腹度が5減る。",
		"回復薬を当てるか困った時のカードで治る。",
		"暴走：15ターン攻撃力1.5倍で自動行動。",
		"敵の暴走は近くの敵も襲う。",
		"身代わり：30ターン動かず敵を引きつける。",
		"透明：30ターン。隣の敵には攻撃される。",
		"透明な敵は気配察知・透視の指輪で見える。",
		"無敵：10ターンダメージ0。状態異常は受ける。",
	}})
}

// 変化した食料は重ならない最寄りの床へ置く。いっぱいなら消える。
func (g *Game) placeSpecialFood(food Item, x, y int) {
	for _, offset := range append([]Coordinate{{}}, nearestFreeOffsets(4)...) {
		tx, ty := x+offset.X, y+offset.Y
		if ty < 0 || ty >= len(g.state.Map) || tx < 0 || tx >= len(g.state.Map[ty]) || g.state.Map[ty][tx].Blocked {
			continue
		}
		occupied := false
		for _, item := range g.state.Items {
			ix, iy := item.GetPosition()
			if ix == tx && iy == ty {
				occupied = true
				break
			}
		}
		if !occupied {
			food.SetPosition(tx, ty)
			g.state.Items = append(g.state.Items, food)
			return
		}
	}
}

// 未識別表示だけを差し替え、テンプレートの正式名は変えない。
func specialUnidentifiedName(item Item) string {
	switch item.GetID() {
	case 219:
		return "木彫りの杖"
	case 220:
		return "水晶の杖"
	case 221:
		return "赤珊瑚の杖"
	}
	return item.GetName()
}

// 杖の効果を直接呼ぶ経路でも、共通の付与と遅延実行を使う。
func (g *Game) queueSpecialCaneStatus(itemID, index int) {
	if index < 0 || index >= len(g.state.Enemies) {
		return
	}
	enemy := g.state.Enemies[index]
	kind, turns := specialItemStatus(itemID)
	g.Enqueue(Action{Duration: statusMessageDuration, Message: fmt.Sprintf("%sに杖の力が届いた", g.enemyDisplayName(enemy.Name)), Execute: func(g *Game) {
		current := g.findEnemyForDefense(enemy.ID, enemy.X, enemy.Y)
		if !g.applyEnemyStatus(current, kind, turns) {
			g.EnqueueMessage("しかし何も起こらなかった", statusMessageDuration)
		}
		g.TargetEnemyIndex = -1
		g.miniMapDirty = true
	}})
}

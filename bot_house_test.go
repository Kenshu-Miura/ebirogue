//go:build !test

package main

// expert のモンスターハウスでの立ち回り（wave11-A。botFeature の名前は "house"）。
// 人に見える情報だけで判断する: 「モンスターハウスだ！」の通知と、部屋に見えている起きている敵の数と位置。
// Room.Role や、まだ入っていない部屋の敵の配置は読まない。
//
//   - 部屋の中で敵が多ければ、部屋の敵をまとめて片づけるカード（全滅・真空斬り・混乱・目潰し・睡眠）を先に使う。
//   - 階段が近く、敵がまだ届かないなら降りる。
//   - 通路の入口へ下がって1対1で戦う（ふだんより遠くまで下がり、長く待つ）。下がる先が無く囲まれたら道具を使い、
//     部屋の隅へ寄る。
//   - HP が危ないときは敵の動きを止める道具で逃げる。
//   - 部屋の道具は、部屋の敵を片づけてから拾う。

const (
	houseNotice       = "モンスターハウスだ！"
	houseCrowdMin     = 5  // 部屋に起きている敵がこれ以上見えたら、通知を見逃してもモンスターハウスとみなす
	houseThreatRange  = 6  // この距離以内の起きている敵も、モンスターハウスの敵として数える
	houseSweepMin     = 3  // 部屋の中の敵がこれ以上なら、まとめて片づける道具を使う
	houseSweepRate    = 50 // 部屋の敵と殴り合うときの被ダメージの見込みが HP のこの割合（%）以上なら使う
	houseVacuumMin    = 4  // 真空斬り・混乱・目潰し・睡眠のカードを使う部屋の敵の数
	houseStairsMax    = 8  // この歩数以内の階段へは、敵が届かなければ向かって降りる
	houseStairsFoes   = 3  // 敵がこれ以上残っていれば、階段の上で降りる
	houseChokeRange   = 10 // 通路の入口へ下がる最大の歩数（隣に敵がいなければ）
	houseChokeAdjMax  = 3  // 隣に敵がいるときに下がる最大の歩数
	houseChokeWaitMax = 24 // 通路の入口で敵を待つ最大のターン数
	houseCornerRange  = 3  // 下がる先が無いとき、隣から攻撃できるマスの少ない場所を探す歩数
	houseDangerHPRate = 50 // HP がこの割合（%）未満なら、逃げる道具を使う
)

// botHouse は expert が今いる階で気づいたモンスターハウス。
type botHouse struct {
	floor int // 気づいた階（0 は気づいていない）
	room  int // 部屋の ID
}

// newMessages は前の呼び出しの後に増えたメッセージを返す（履歴は上限で古いものが消えるので、最後に読んだ文で位置を合わせる）。
func (b *bot) newMessages() []string {
	msgs := b.g.messageLog.Messages
	var out []string
	switch {
	case b.logLen <= len(msgs) && (b.logLen == 0 || msgs[b.logLen-1] == b.logTail):
		out = msgs[b.logLen:]
	default:
		out = msgs
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i] == b.logTail {
				out = msgs[i+1:]
				break
			}
		}
	}
	b.logLen = len(msgs)
	if len(msgs) > 0 {
		b.logTail = msgs[len(msgs)-1]
	}
	return out
}

// roomByID は部屋 id を返す。
func (b *bot) roomByID(id int) (Room, bool) {
	for _, r := range b.g.rooms {
		if r.ID == id {
			return r, true
		}
	}
	return Room{}, false
}

// roomActiveEnemies は部屋 room の中（壁の線を含む）にいる起きている敵。
func (b *bot) roomActiveEnemies(room Room) []Enemy {
	var out []Enemy
	for _, e := range b.g.state.Enemies {
		if activeEnemy(e) && !botIgnoresEnemy(e) && roomContainsWithWalls(room, e.X, e.Y) {
			out = append(out, e)
		}
	}
	return out
}

// updateHouse は通知と部屋の敵の数から、今いる部屋がモンスターハウスかを覚える（expertStep の毎回の行動の前）。
func (b *bot) updateHouse() {
	g := b.g
	notice := false
	for _, m := range b.newMessages() {
		if m == houseNotice {
			notice = true
		}
	}
	if b.house.floor != g.Floor {
		b.house = botHouse{}
	}
	if b.house.floor != 0 || isBossFloor(g.Floor) {
		return // ボス階にモンスターハウスは無い（ボスが呼んだ手下で部屋が混んでも数えない）
	}
	p := g.state.Player
	id := g.roomIDAt(p.X, p.Y)
	if id < 0 {
		return
	}
	room, ok := b.roomByID(id)
	if !ok {
		return
	}
	if notice || len(b.roomActiveEnemies(room)) >= houseCrowdMin {
		b.house = botHouse{floor: g.Floor, room: id}
	}
}

// houseRoom は気づいたモンスターハウスの部屋（この階で気づいていなければ false）。
func (b *bot) houseRoom() (Room, bool) {
	if b.house.floor == 0 || b.house.floor != b.g.Floor {
		return Room{}, false
	}
	return b.roomByID(b.house.room)
}

// inKnownHouse は気づいたモンスターハウスの部屋の床にいるかどうか。
func (b *bot) inKnownHouse() bool {
	room, ok := b.houseRoom()
	p := b.g.state.Player
	return ok && roomContains(room, p.X, p.Y)
}

// houseFoes はモンスターハウスの部屋にいる起きている敵と、houseThreatRange 以内の起きている敵。
func (b *bot) houseFoes() []Enemy {
	room, ok := b.houseRoom()
	if !ok {
		return nil
	}
	p := b.g.state.Player
	var out []Enemy
	for _, e := range b.g.state.Enemies {
		if !activeEnemy(e) || botIgnoresEnemy(e) {
			continue
		}
		if roomContainsWithWalls(room, e.X, e.Y) || max(abs(e.X-p.X), abs(e.Y-p.Y)) <= houseThreatRange {
			out = append(out, e)
		}
	}
	return out
}

// roomEnemies は部屋 room の中（壁の線を含む）にいる敵（眠っている敵を含む）。
func (b *bot) roomEnemies(room Room) []Enemy {
	var out []Enemy
	for _, e := range b.g.state.Enemies {
		if !botIgnoresEnemy(e) && roomContainsWithWalls(room, e.X, e.Y) {
			out = append(out, e)
		}
	}
	return out
}

// houseCrowded はモンスターハウスの部屋に、眠っている敵も含めて houseSweepMin 体以上が残っているかどうか。
func (b *bot) houseCrowded() bool {
	room, ok := b.houseRoom()
	return ok && len(b.roomEnemies(room)) >= houseSweepMin
}

// inHouseFight はモンスターハウスに気づいていて、起きている敵が残っているか、部屋にまだ敵が多く残っているかどうか。
func (b *bot) inHouseFight() bool {
	return botFeature("house") && (len(b.houseFoes()) > 0 || b.houseCrowded())
}

// houseWantsToDescend はモンスターハウスの敵が多く残っているので、階段の上なら降りるかどうか。
func (b *bot) houseWantsToDescend() bool {
	return botFeature("house") && (len(b.houseFoes()) >= houseStairsFoes || b.houseCrowded())
}

// houseAvoidsEnemy は、敵の多く残ったモンスターハウスの部屋にいる敵へ自分から近づかないかどうか。
func (b *bot) houseAvoidsEnemy(e Enemy) bool {
	if !botFeature("house") || !b.houseCrowded() {
		return false
	}
	room, _ := b.houseRoom()
	return roomContainsWithWalls(room, e.X, e.Y)
}

// useFirstID は ids を書いた順に探し、最初に持っている道具を action で使う（useByID は所持品の並び順）。
func (b *bot) useFirstID(action string, ids ...int) (bool, error) {
	for _, id := range ids {
		if used, err := b.useByID(action, id); used || err != nil {
			return used, err
		}
	}
	return false, nil
}

// houseItemBlocked は (x, y) の道具がモンスターハウスの部屋にあり、部屋に起きている敵が残っているかどうか（まだ拾いに行かない）。
func (b *bot) houseItemBlocked(x, y int) bool {
	if !botFeature("house") {
		return false
	}
	room, ok := b.houseRoom()
	if !ok || !roomContainsWithWalls(room, x, y) {
		return false
	}
	return len(b.roomEnemies(room)) > 0
}

// houseStairsStep は階段が houseStairsMax 歩以内にあり、起きている敵が先に届かないとき、階段へ向かう1歩を返す。
func (b *bot) houseStairsStep(foes []Enemy) (int, int, bool) {
	dx, dy, steps, ok := b.pathStepWithin(b.stairsGoal, houseStairsMax)
	if !ok {
		return 0, 0, false
	}
	m := b.g.state.Map
	sx, sy := -1, -1
	for y := range m {
		for x := range m[y] {
			if m[y][x].Type == "stairs" {
				sx, sy = x, y
			}
		}
	}
	for _, e := range foes {
		if max(abs(e.X-sx), abs(e.Y-sy)) <= steps {
			return 0, 0, false
		}
	}
	return dx, dy, true
}

// expertHouse はモンスターハウスでの立ち回り。行動したら true。
func (b *bot) expertHouse(adjacent []Enemy) (bool, error) {
	g := b.g
	p := &g.state.Player
	room, ok := b.houseRoom()
	if !ok {
		return false, nil
	}
	foes := b.houseFoes()
	if len(foes) == 0 {
		b.houseWait = 0
		return b.expertHouseSneak(room)
	}
	hpRate := p.Health * 100 / max(1, p.MaxHealth)

	// 1. 部屋の中で敵が多ければ、部屋の敵をまとめて片づける
	if roomContains(room, p.X, p.Y) {
		roomFoes := b.roomActiveEnemies(room)
		if len(roomFoes) >= houseSweepMin && b.fightDamageEstimate(roomFoes)*100 >= float64(p.Health*houseSweepRate) {
			b.act = "houseSweep"
			if used, err := b.useByID("使う", 56); used || err != nil { // 全滅
				return used, err
			}
			if len(roomFoes) >= houseVacuumMin {
				if used, err := b.useFirstID("使う", 31, 13, 14, 12); used || err != nil { // 真空斬り・混乱・目潰し・睡眠
					return used, err
				}
			}
			b.act = ""
		}
	}

	// 2. 階段が近く、敵がまだ届かないなら降りる
	if g.state.Map[p.Y][p.X].Type == "stairs" && b.houseWantsToDescend() {
		b.act = "descend"
		return true, b.tap(KeyS)
	}
	if len(adjacent) == 0 && len(foes) >= houseStairsFoes {
		if dx, dy, ok := b.houseStairsStep(foes); ok {
			b.act = "houseStairs"
			return true, b.move(dx, dy)
		}
	}

	// 3. HP が危なければ、隣の敵の動きを止める
	if hpRate < houseDangerHPRate && len(adjacent) > 0 && len(foes) >= 2 {
		if done, err := b.expertEscape(adjacent); done || err != nil {
			b.act = "houseEscape"
			return done, err
		}
	}

	threats := b.approachingEnemies(houseThreatRange)
	onChoke := b.isChoke(p.X, p.Y)
	// 4. 通路の入口へ下がる（ふだんより遠くまで）。下がる先が無く囲まれたら、道具を使って部屋の隅へ寄る
	if len(threats) >= 2 && !onChoke && !b.inRangedFire(p.X, p.Y) {
		limit := houseChokeRange
		if len(adjacent) > 0 {
			limit = houseChokeAdjMax
		}
		if dx, dy, _, ok := b.pathStepWithin(b.isChoke, limit); ok {
			b.act = "houseRetreat"
			return true, b.move(dx, dy)
		}
		if len(adjacent) >= 2 {
			b.act = "houseSurrounded"
			if botFeature("zowazowa") {
				if used, err := b.useByID("使う", cardZowazowaID); used || err != nil {
					return used, err
				}
			}
			if used, err := b.useFirstID("使う", 56, 31); used || err != nil {
				return used, err
			}
			if done, err := b.expertEscape(adjacent); done || err != nil {
				return done, err
			}
			b.act = ""
		}
		slots := b.attackSlots(p.X, p.Y)
		corner := func(x, y int) bool { return b.attackSlots(x, y) < slots && b.attackSlots(x, y) <= 3 }
		if dx, dy, _, ok := b.pathStepWithin(corner, houseCornerRange); ok && len(adjacent) <= 1 {
			b.act = "houseCorner"
			return true, b.move(dx, dy)
		}
	}
	if len(adjacent) > 0 || b.inRangedFire(p.X, p.Y) {
		b.houseWait = 0
		return false, nil
	}
	// 5. 通路の入口で、近づいてくる敵を待つ
	if onChoke && len(threats) >= 1 && b.houseWait < houseChokeWaitMax {
		b.houseWait++
		b.act = "houseHold"
		return true, b.stepInPlace()
	}
	if len(threats) == 0 {
		b.houseWait = 0
	}
	return false, nil
}

// expertHouseSneak は、モンスターハウスの部屋の中にいて、部屋に眠っている（動けない）敵が多く残っているとき、
// 殴って起こさずに階段へ向かうか部屋の外へ出る（カードで眠らせた敵は起きると倍速になる）。行動したら true。
func (b *bot) expertHouseSneak(room Room) (bool, error) {
	g := b.g
	p := g.state.Player
	if !b.houseCrowded() || !roomContainsWithWalls(room, p.X, p.Y) {
		return false, nil // 部屋の外では普段どおり（部屋の敵は追わず、部屋の道具も拾いに行かない）
	}
	if g.state.Map[p.Y][p.X].Type == "stairs" {
		b.act = "descend"
		return true, b.tap(KeyS)
	}
	if dx, dy, ok := b.pathStep(b.stairsGoal); ok {
		b.act = "houseSneakStairs"
		return true, b.move(dx, dy)
	}
	if dx, dy, ok := b.pathStep(func(x, y int) bool { return !roomContainsWithWalls(room, x, y) }); ok {
		b.act = "houseSneakOut"
		return true, b.move(dx, dy)
	}
	return false, nil
}

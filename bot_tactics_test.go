//go:build !test

package main

import (
	"flag"
	"sort"
	"strings"
	"sync"
)

// expert の立ち回り（bot_expert_test.go から呼ぶ）。不思議のダンジョンの定石に合わせて、
// 部屋で複数の敵に近づかれたら通路の入口へ下がって1対1で戦い、距離2の敵は足踏みで待って先手を取り、
// 「このまま殴り合うと倒れる」と見込んだら止める・倒す道具を早めに使う。

// botOff は計測用に expert の改善を個別に切る（例: -bot.off=retreat,ambush）。
var botOff = flag.String("bot.off", "", "計測用: 切る expert の改善（カンマ区切り）")

var (
	botOffOnce sync.Once
	botOffSet  map[string]bool
)

// botFeature は expert の改善 name が有効かどうか。
func botFeature(name string) bool {
	botOffOnce.Do(func() {
		botOffSet = map[string]bool{}
		for _, s := range strings.Split(*botOff, ",") {
			if s = strings.TrimSpace(s); s != "" {
				botOffSet[s] = true
			}
		}
	})
	return !botOffSet[name]
}

const (
	expertThreatRange  = 4 // この距離以内の起きている敵を「近づいてくる敵」とみなす
	expertChokeRange   = 5 // 通路の入口へ下がる最大の歩数（隣に敵がいなければ）
	expertChokeAdjMax  = 2 // 隣に敵がいるときに下がる最大の歩数
	expertChokeWaitMax = 8 // 通路の入口で敵を待つ最大のターン数
	expertAmbushMax    = 3 // 距離2の敵を足踏みで待つ最大のターン数
)

// activeEnemy は起きていて動ける（こちらへ来る）敵かどうか。
func activeEnemy(e Enemy) bool {
	return e.StatusAilments.Sleep == 0 && !e.StatusAilments.Paralysis && !isEnemyDisguised(e) && !e.Fleeing
}

// enemiesWithin は距離 r 以内の起きている敵。
func (b *bot) enemiesWithin(r int) []Enemy {
	p := b.g.state.Player
	var out []Enemy
	for _, e := range b.g.state.Enemies {
		if activeEnemy(e) && max(abs(e.X-p.X), abs(e.Y-p.Y)) <= r {
			out = append(out, e)
		}
	}
	return out
}

// attackSlots は (x, y) へ隣から攻撃できるマスの数（部屋の中は最大8、通路は2）。
func (b *bot) attackSlots(x, y int) int {
	n := 0
	for _, s := range botSteps {
		if b.canStep(x, y, s[0], s[1]) {
			n++
		}
	}
	return n
}

// isChoke は (x, y) が1対1で戦える場所（通路・入口）かどうか。
func (b *bot) isChoke(x, y int) bool {
	return b.attackSlots(x, y) <= 2 && b.g.state.Map[y][x].Type != "stairs"
}

// pathStepWithin は pathStep と同じだが maxSteps 歩以内の goal だけを探し、最初の1歩と歩数を返す。
func (b *bot) pathStepWithin(goal func(x, y int) bool, maxSteps int) (int, int, int, bool) {
	g := b.g
	height, width := len(g.state.Map), len(g.state.Map[0])
	blocked := make([]bool, width*height)
	for _, enemy := range g.state.Enemies {
		if enemy.X >= 0 && enemy.Y >= 0 && enemy.X < width && enemy.Y < height {
			blocked[enemy.Y*width+enemy.X] = true
		}
	}
	for _, trap := range g.state.MapTraps {
		if trap.Discovered && trap.X >= 0 && trap.Y >= 0 && trap.X < width && trap.Y < height {
			blocked[trap.Y*width+trap.X] = true
		}
	}
	start := g.state.Player.Y*width + g.state.Player.X
	first := make([]int, width*height)
	dist := make([]int, width*height)
	queue := []int{start}
	first[start] = -1
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if dist[cur] >= maxSteps {
			continue
		}
		x, y := cur%width, cur/width
		for dir, step := range botSteps {
			if !b.canStep(x, y, step[0], step[1]) {
				continue
			}
			nx, ny := x+step[0], y+step[1]
			next := ny*width + nx
			if first[next] != 0 || blocked[next] {
				continue
			}
			dirFromStart := dir + 1
			if cur != start {
				dirFromStart = first[cur]
			}
			first[next] = dirFromStart
			dist[next] = dist[cur] + 1
			if goal(nx, ny) {
				s := botSteps[dirFromStart-1]
				return s[0], s[1], dist[next], true
			}
			queue = append(queue, next)
		}
	}
	return 0, 0, 0, false
}

// expertTactics は立ち回り（道具を早めに使う・通路へ下がる・先手を取る）。行動したら true。
func (b *bot) expertTactics(adjacent []Enemy) (bool, error) {
	g := b.g
	p := &g.state.Player
	if b.bossAlive() || g.playerOnSanctuary() {
		return false, nil
	}
	if botFeature("saveitem") {
		if done, err := b.expertSaveItems(adjacent); done || err != nil {
			return done, err
		}
	}
	threats := b.enemiesWithin(expertThreatRange)
	if botFeature("retreat") && len(threats) >= 2 && b.chokeWait < expertChokeWaitMax && !b.isChoke(p.X, p.Y) && !b.inRangedFire(p.X, p.Y) {
		limit := expertChokeRange
		if len(adjacent) > 0 {
			limit = expertChokeAdjMax
		}
		if dx, dy, _, ok := b.pathStepWithin(b.isChoke, limit); ok {
			b.act = "retreat"
			return true, b.move(dx, dy)
		}
	}
	if len(adjacent) > 0 || b.inRangedFire(p.X, p.Y) {
		b.chokeWait, b.ambushWait = 0, 0
		return false, nil
	}
	// 通路の入口で、近づいてくる敵を待つ（部屋へ出ていかない）
	if botFeature("retreat") && len(threats) >= 2 && b.isChoke(p.X, p.Y) && b.chokeWait < expertChokeWaitMax {
		b.chokeWait++
		b.act = "holdChoke"
		return true, b.stepInPlace()
	}
	// 距離2の敵は足踏みで待ち、隣に来たところを先に殴る（倍速・遠距離攻撃の敵は待たない）
	if botFeature("ambush") && b.ambushWait < expertAmbushMax {
		for _, e := range b.enemiesWithin(2) {
			if max(abs(e.X-p.X), abs(e.Y-p.Y)) != 2 || e.SpecialMovement == SpecialMovementDoubleSpeed ||
				e.StatusAilments.Haste > 0 || e.RangedAttack.Kind != RangedAttackNone || isHazardEnemy(e) {
				continue
			}
			b.ambushWait++
			b.act = "ambush"
			return true, b.stepInPlace()
		}
	}
	if len(b.enemiesWithin(2)) == 0 {
		b.ambushWait = 0
	}
	if len(threats) < 2 {
		b.chokeWait = 0
	}
	return false, nil
}

// fightDamageEstimate は enemies を通常攻撃だけで順に倒すまでに受けるダメージの見込み。
// 倒しやすく痛い敵から倒す。距離2の敵も1ターン後から加わるものとして数える。
func (b *bot) fightDamageEstimate(enemies []Enemy) float64 {
	type foe struct{ turns, dmg float64 }
	foes := make([]foe, 0, len(enemies))
	for _, e := range enemies {
		perHit := b.expectedDamageToEnemy(e) * float64(basePlayerAttackAccuracy) / 100
		if perHit < 1 {
			perHit = 1
		}
		turns := float64(e.Health) / perHit
		if turns < 1 {
			turns = 1
		}
		foes = append(foes, foe{turns, b.expectedDamageToPlayer(e)})
	}
	sort.Slice(foes, func(i, j int) bool { return foes[i].dmg/foes[i].turns > foes[j].dmg/foes[j].turns })
	total, elapsed := 0.0, 0.0
	for _, f := range foes {
		elapsed += f.turns
		total += f.dmg * elapsed
	}
	return total
}

// expertSaveItems は「このまま殴り合うと倒れる」と見込んだら、止める・倒す道具を早めに使う。使ったら true。
func (b *bot) expertSaveItems(adjacent []Enemy) (bool, error) {
	if len(adjacent) == 0 {
		return false, nil
	}
	p := &b.g.state.Player
	foes := b.enemiesWithin(2)
	if len(foes) == 0 {
		return false, nil
	}
	if b.fightDamageEstimate(foes) < float64(p.Health) {
		return false, nil
	}
	b.act = "saveItem"
	if len(adjacent) >= 2 {
		if botFeature("zowazowa") {
			if used, err := b.useByID("使う", cardZowazowaID); used || err != nil { // 隣の敵どうしで殴り合わせる
				return used, err
			}
		}
		if used, err := b.useByID("使う", 56, 31); used || err != nil { // 全滅・真空斬り
			return used, err
		}
	}
	if done, err := b.expertEscape(adjacent); done || err != nil {
		return done, err
	}
	if botFeature("selfdestruct") && len(adjacent) >= 2 && p.Health*100 >= p.MaxHealth*50 {
		if used, err := b.useByID("使う", cardSelfDestructID); used || err != nil { // HP が半分になるが隣の敵を消す
			return used, err
		}
	}
	target := adjacent[0]
	for _, e := range adjacent {
		if b.expectedDamageToPlayer(e) > b.expectedDamageToPlayer(target) {
			target = e
		}
	}
	if i := b.findInventoryID(7); i >= 0 { // 黒炎弾は一番痛い敵へ
		return b.faceAndUse(target.X-p.X, target.Y-p.Y, i, "使う")
	}
	if used, err := b.useByID("使う", 56, 31); used || err != nil {
		return used, err
	}
	if p.Health*100 < p.MaxHealth*60 {
		if used, err := b.useByID("使う", 173, 2, 28, 3, 29, 44); used || err != nil {
			b.act = "heal"
			return used, err
		}
	}
	if botFeature("canes") && p.Health*100 < p.MaxHealth*50 {
		if done, err := b.expertShift(); done || err != nil {
			return done, err
		}
	}
	if botFeature("sanctuary") && p.Health*100 < p.MaxHealth*50 {
		if done, err := b.expertSanctuary(); done || err != nil {
			return done, err
		}
	}
	b.act = ""
	return false, nil
}

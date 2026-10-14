//go:build !test

package main

// expert 性格の BOT。状況に合わせてカード・杖・薬・矢を使い分け、装備を比べて持ち替え、
// フロアを探索してレベルを上げてから階段を降りる。既存の性格は比較用にそのまま残す。

// itemRole はカード・杖・薬の使い道の分類。
type itemRole int

const (
	roleNone    itemRole = iota // 使わない（壺関連・白紙・聖域など、BOT が扱わないもの）
	roleAttack                  // 攻撃: 敵へのダメージや撃破
	roleEscape                  // 逃走: 敵の動きを止める・そらす
	roleBuff                    // 強化: 自分の能力・装備を良くする
	roleHeal                    // 回復
	roleHarmful                 // 不利: 自分に不利な効果。使わない
)

// itemRoles はテンプレート ID ごとの分類表（items.go の itemTemplates に対応）。
var itemRoles = map[int]itemRole{
	// 回復
	2: roleHeal, 3: roleHeal, 28: roleHeal, 29: roleHeal, // ミンティア系・栄養ドリンク
	44: roleHeal, // 完全回復のカード
	// 攻撃
	7:  roleAttack, // 黒炎弾のカード（正面の敵に30ダメージ）
	31: roleAttack, // 真空斬りのカード（部屋の敵全員）
	56: roleAttack, // 全滅のカード（部屋の敵をすべて消す）
	54: roleAttack, // ジェノサイドのカード（投げ当てる）
	// 逃走（敵の動きを止める）
	12: roleEscape, 13: roleEscape, 14: roleEscape, // 睡眠・混乱・目潰しのカード（部屋の敵全員）
	15: roleEscape,                                 // 金縛りのカード（周囲8マス）
	17: roleEscape, 18: roleEscape, 19: roleEscape, // 睡眠薬・混乱薬・目潰し薬（敵に投げる）
	16: roleEscape, // 封印の杖
	9:  roleEscape, // シフトチェンジの杖
	// 強化
	33: roleBuff, // 武器強化
	34: roleBuff, // 盾強化
	43: roleBuff, // パワーアップ
	45: roleBuff, // さび止め
	32: roleBuff, // おはらい
	11: roleBuff, // 真実の眼
	30: roleBuff, // あかり
	8:  roleBuff, // 炸裂装甲のカード（罠をセット）
	// 不利
	35: roleHarmful, // モンスターハウス
	36: roleHarmful, // 敵倍速
	37: roleHarmful, // 地図忘却
	38: roleHarmful, // 拾得禁止
	40: roleHarmful, // 罠
	41: roleHarmful, // 自爆（HP が半分になる）
	42: roleHarmful, // 口封じ

	167: roleNone, 168: roleHarmful, 169: roleBuff, 170: roleHarmful,
	171: roleBuff, 172: roleNone, 173: roleHeal,
}

// 強化のカードのうち「すぐ使う」もの（使える条件は expertBuffUsable で判定する）。
func (b *bot) expertBuffUsable(id int) bool {
	p := &b.g.state.Player
	switch id {
	case 169:
		for _, trap := range b.g.state.MapTraps {
			if trap.Discovered {
				return true
			}
		}
		return false
	case 171:
		for _, item := range b.g.state.Items {
			x, y := item.GetPosition()
			if gatherDistance(Coordinate{X: x, Y: y}, Coordinate{X: p.X, Y: p.Y}) > 3 {
				return true
			}
		}
		return false
	case 33:
		return p.EquippedWeapon != nil
	case 34:
		return p.EquippedArmor != nil
	case 45:
		return (p.EquippedWeapon != nil && !p.EquippedWeapon.RustProof) || (p.EquippedArmor != nil && !p.EquippedArmor.RustProof)
	case 32:
		return b.hasCursedBelongings()
	case 11:
		return b.hasUnidentified()
	case 8:
		return p.SetTrap == nil
	case 43, 30:
		return true
	}
	return false
}

func (b *bot) hasCursedBelongings() bool {
	for _, item := range b.g.state.Player.Inventory {
		if equipable, ok := item.(Equipable); ok && isCursedEquipment(equipable) {
			return true
		}
	}
	return false
}

func (b *bot) hasUnidentified() bool {
	for _, item := range b.g.state.Player.Inventory {
		if identifiable, ok := item.(Identifiable); ok && !identifiable.IsIdentified() {
			return true
		}
	}
	return false
}

func (b *bot) findInventoryID(ids ...int) int {
	for i, item := range b.g.state.Player.Inventory {
		for _, id := range ids {
			if item.GetID() == id {
				return i
			}
		}
	}
	return -1
}

// expectedDamageToPlayer は敵の通常攻撃1回の目安のダメージ。
func (b *bot) expectedDamageToPlayer(enemy Enemy) float64 {
	damage := float64(enemy.AttackPower) * pow1516(b.g.state.Player.DefensePower)
	if enemy.SpecialMovement == SpecialMovementDoubleSpeed || enemy.StatusAilments.Haste > 0 {
		damage *= 2 // 1ターンに2回攻撃してくる
	}
	return damage
}

// expectedDamageToEnemy はプレイヤーの通常攻撃1回の目安のダメージ。
func (b *bot) expectedDamageToEnemy(enemy Enemy) float64 {
	p := &b.g.state.Player
	return float64(playerAttackTotal(p.AttackPower, p.Power, p.Level)) * pow1516(enemy.DefensePower)
}

func pow1516(n int) float64 {
	v := 1.0
	for i := 0; i < n; i++ {
		v *= 15.0 / 16.0
	}
	return v
}

// isStrongEnemy は通常攻撃だけで戦うと危ない敵かどうか。
func (b *bot) isStrongEnemy(enemy Enemy) bool {
	p := &b.g.state.Player
	if MonsterDefinitions[enemy.ID].IsBoss {
		return true
	}
	perHit := b.expectedDamageToEnemy(enemy)
	if perHit < 1 {
		perHit = 1
	}
	hitsToKill := float64(enemy.Health) / perHit
	damageTaken := hitsToKill * b.expectedDamageToPlayer(enemy)
	return damageTaken >= float64(p.Health)*0.6 || b.expectedDamageToPlayer(enemy) >= float64(p.MaxHealth)*0.2
}

func (b *bot) adjacentEnemies() []Enemy {
	p := b.g.state.Player
	var out []Enemy
	for _, enemy := range b.g.state.Enemies {
		dx, dy := enemy.X-p.X, enemy.Y-p.Y
		if abs(dx) <= 1 && abs(dy) <= 1 && (dx != 0 || dy != 0) {
			out = append(out, enemy)
		}
	}
	return out
}

// enemyInLine は向き dx, dy の直線上（壁で遮られない range マス以内）で最初に当たる敵を返す。
func (b *bot) enemyInLine(dx, dy, maxRange int) (Enemy, bool) {
	g := b.g
	x, y := g.state.Player.X, g.state.Player.Y
	for i := 1; i <= maxRange; i++ {
		x, y = x+dx, y+dy
		if y < 0 || y >= len(g.state.Map) || x < 0 || x >= len(g.state.Map[y]) || g.state.Map[y][x].Blocked {
			return Enemy{}, false
		}
		if idx := g.enemyIndexAt(x, y); idx >= 0 {
			return g.state.Enemies[idx], true
		}
	}
	return Enemy{}, false
}

// targetInLine は8方向のうち、直線上に敵がいる方向とその敵を返す（近い敵を優先）。
func (b *bot) targetInLine(maxRange int) (int, int, Enemy, bool) {
	bestDX, bestDY, best, found := 0, 0, Enemy{}, false
	bestDistance := maxRange + 1
	p := b.g.state.Player
	for _, s := range botSteps {
		if enemy, ok := b.enemyInLine(s[0], s[1], maxRange); ok {
			if d := max(abs(enemy.X-p.X), abs(enemy.Y-p.Y)); d < bestDistance {
				bestDX, bestDY, best, found, bestDistance = s[0], s[1], enemy, true, d
			}
		}
	}
	return bestDX, bestDY, best, found
}

// inRangedFire は (x, y) が遠距離攻撃をする敵の射線（同じ直線上で射程内）にあるかどうか。
func (b *bot) inRangedFire(x, y int) bool {
	for _, enemy := range b.g.state.Enemies {
		attack := enemy.RangedAttack
		if attack.Kind == RangedAttackNone || enemy.StatusAilments.Seal {
			continue
		}
		dx, dy := x-enemy.X, y-enemy.Y
		d := max(abs(dx), abs(dy))
		if d <= 1 || d < attack.MinRange || d > attack.MaxRange {
			continue
		}
		// 投石は山なりに投げるので、直線上でなくても壁越しに届く
		if attack.Kind == RangedAttackRock || dx == 0 || dy == 0 || abs(dx) == abs(dy) {
			return true
		}
	}
	return false
}

// expertPathStep は遠距離攻撃をする敵の射線を避けて goal へ向かう1歩を返す。避けられなければ普通に向かう。
func (b *bot) expertPathStep(goal func(x, y int) bool) (int, int, bool) {
	if dx, dy, ok := b.pathStepAvoiding(goal, b.inRangedFire); ok {
		return dx, dy, true
	}
	return b.pathStep(goal)
}

// pathStepAvoiding は avoid を満たすマスを通らない（goal のマスは除く）経路で goal へ向かう1歩を返す。
func (b *bot) pathStepAvoiding(goal func(x, y int) bool, avoid func(x, y int) bool) (int, int, bool) {
	return b.pathStep(func(x, y int) bool {
		return goal(x, y) && !avoid(x, y)
	})
}

func (b *bot) bestEquipment() (int, bool) {
	p := &b.g.state.Player
	weaponScore := func(w *Weapon) int { return w.AttackPower + w.Sharpness }
	armorScore := func(a *Armor) int { return a.DefensePower + a.Sharpness }
	usable := func(item Equipable) bool {
		return !isCursedEquipment(item) && !p.IsEquipped(item)
	}
	for i, item := range p.Inventory {
		switch v := item.(type) {
		case *Weapon:
			// 満腹度を消費・参照する武器は自動で持ち替えない。
			if v.GetID() == 136 || v.GetID() == 137 {
				continue
			}
			if v.Sharpness < 0 || !usable(v) {
				continue
			}
			if p.EquippedWeapon == nil || (!p.EquippedWeapon.Cursed && weaponScore(v) > weaponScore(p.EquippedWeapon)) {
				return i, true
			}
		case *Armor:
			// 満腹度を使う盾・空腹用の盾・悟りの盾は自動で選ばない。
			if v.GetID() == 155 || v.GetID() == 156 || v.GetID() == 159 {
				continue
			}
			// 重甲の盾は満腹度の減りが速くなるため選ばない
			if v.Sharpness < 0 || v.GetID() == 69 || !usable(v) {
				continue
			}
			if p.EquippedArmor == nil || (!p.EquippedArmor.Cursed && armorScore(v) > armorScore(p.EquippedArmor)) {
				return i, true
			}
		case *Arrow:
			if p.EquippedArrow == nil && !v.Cursed && v.ShotCount > 0 {
				return i, true
			}
		}
	}
	return -1, false
}

// expertWantsToDescend は階段を降りるかどうか。探索し尽くすまでは降りず、レベルを上げる。
func (b *bot) expertWantsToDescend() bool {
	g := b.g
	if b.bossAlive() {
		return false
	}
	if g.floorTurns > 700 { // 風（1200ターン）よりかなり前に降りる
		return true
	}
	// 食料の残りから、この階にいられるターン数を見積もる（満腹度は10ターンで1減る）
	food := 0
	for _, item := range g.state.Player.Inventory {
		if f, ok := item.(*Food); ok {
			food += max(f.Satiety, 50)
		}
	}
	if g.state.Player.Satiety+food <= 50 {
		return true // 食料が乏しければ探索を切り上げて先へ急ぐ（次の階で食料を探す）
	}
	if _, _, ok := b.pathStep(b.itemGoal); ok {
		return false
	}
	_, _, unexplored := b.pathStep(b.unexploredGoal)
	return !unexplored
}

// useByID は所持品のうち ids のどれかを action で使う。
func (b *bot) useByID(action string, ids ...int) (bool, error) {
	if i := b.findInventoryID(ids...); i >= 0 {
		return b.useInventoryItem(i, action)
	}
	return false, nil
}

// faceAndUse は (dx, dy) を向いてから所持品 index に action を行う。
func (b *bot) faceAndUse(dx, dy, index int, action string) (bool, error) {
	if err := b.tap(append([]Key{KeyA}, directionKeys(sign(dx), sign(dy))...)...); err != nil {
		return false, err
	}
	return b.useInventoryItem(index, action)
}

func (b *bot) roleItems(role itemRole) []int {
	var out []int
	for i, item := range b.g.state.Player.Inventory {
		if itemRoles[item.GetID()] == role {
			out = append(out, i)
		}
	}
	return out
}

// expertStep は expert 性格の1回分の行動。
func (b *bot) expertStep() error {
	g := b.g
	player := &g.state.Player
	if g.showStairsPrompt {
		if b.expertWantsToDescend() && g.selectedOption == 0 {
			return b.tap(KeyZ)
		}
		return b.tap(KeyX)
	}
	if b.windowOpen() {
		return b.closeWindows()
	}
	if b.idle > 20 && b.idle%5 == 0 {
		s := botSteps[b.rng.IntN(len(botSteps))]
		return b.move(s[0], s[1])
	}
	adjacent := b.adjacentEnemies()
	b.act = ""

	// 1. 回復: HP 40% 未満で回復薬・完全回復のカード
	if player.Health*100 < player.MaxHealth*40 {
		if used, err := b.useByID("使う", 173); used || err != nil {
			b.act = "heal"
			return err
		}
		if used, err := b.useByID("使う", 2, 28, 3, 29, 44); used || err != nil {
			b.act = "heal"
			return err
		}
		// 回復手段が無く敵に囲まれているなら、動きを止める
		if len(adjacent) > 0 {
			if done, err := b.expertEscape(adjacent); done || err != nil {
				b.act = "escape"
				return err
			}
		}
	}

	// 2. 空腹
	if player.Satiety <= 20 {
		if i := b.findInventory(isFoodItem); i >= 0 {
			if used, err := b.useInventoryItem(i, "使う"); used || err != nil {
				b.act = "eat"
				return err
			}
		}
	}

	// 3. 強化のカードはすぐ使う
	for _, i := range b.roleItems(roleBuff) {
		if b.expertBuffUsable(player.Inventory[i].GetID()) {
			if used, err := b.useInventoryItem(i, "使う"); used || err != nil {
				return err
			}
		}
	}

	// 4. 装備の持ち替え
	if i, ok := b.bestEquipment(); ok {
		if used, err := b.useInventoryItem(i, "装備"); used || err != nil {
			return err
		}
	}

	// 5. 戦闘
	if len(adjacent) >= 3 {
		if used, err := b.useByID("使う", 56, 31); used || err != nil { // 囲まれたら全滅・真空斬り
			b.act = "attackItem"
			return err
		}
	}
	if len(adjacent) > 0 {
		target := adjacent[0]
		for _, enemy := range adjacent {
			if b.isStrongEnemy(enemy) {
				target = enemy
			}
		}
		dx, dy := target.X-player.X, target.Y-player.Y
		if b.isStrongEnemy(target) {
			if done, err := b.expertFightStrong(target, dx, dy); done || err != nil {
				b.act = "strongItem"
				return err
			}
		}
		if b.canStep(player.X, player.Y, dx, dy) {
			b.act = "melee"
			return b.attack(dx, dy)
		}
	}
	// 直線上の敵を矢で撃つ・状態異常の薬を投げる
	if dx, dy, enemy, ok := b.targetInLine(8); ok && len(adjacent) == 0 {
		if b.isStrongEnemy(enemy) {
			if i := b.findInventoryID(17, 18, 19, 54); i >= 0 {
				if used, err := b.faceAndUse(dx, dy, i, "投げる"); used || err != nil {
					b.act = "throw"
					return err
				}
			}
		}
		if player.EquippedArrow != nil && player.EquippedArrow.ShotCount > 0 {
			b.act = "shoot"
			if err := b.tap(append([]Key{KeyA}, directionKeys(dx, dy)...)...); err != nil {
				return err
			}
			return b.tap(KeyD)
		}
	}

	// 6. 階段の上で降りたくなったら確認を開き直す
	if g.state.Map[player.Y][player.X].Type == "stairs" && b.expertWantsToDescend() {
		b.act = "descend"
		return b.tap(KeyS)
	}

	// 6b. HP が少ないのに遠距離攻撃の射程にいたら、部屋の敵を止めるカードを使うか射程の外へ下がる
	if player.Health*100 < player.MaxHealth*40 && b.inRangedFire(player.X, player.Y) {
		if used, err := b.useByID("使う", 12, 13, 14); used || err != nil {
			b.act = "escape"
			return err
		}
		if dx, dy, ok := b.pathStep(func(x, y int) bool { return !b.inRangedFire(x, y) }); ok {
			b.act = "leaveFire"
			return b.move(dx, dy)
		}
	}

	// 6c. 周りに敵がいなければ、足踏みで HP の自然回復を待つ
	if b.expertShouldRest() {
		b.act = "rest"
		return b.stepInPlace()
	}

	// 7. 移動: ボス > 経験値になる敵 > 道具 > 未探索 > 階段
	healthy := player.Health*100 >= player.MaxHealth*60
	var goals []func(x, y int) bool
	var labels []string
	addGoal := func(label string, goal func(x, y int) bool) {
		goals, labels = append(goals, goal), append(labels, label)
	}
	if b.bossAlive() && healthy {
		if boss, ok := b.nearestBoss(); ok {
			addGoal("toBoss", b.enemyGoal(boss))
		}
	}
	// 射程内から撃ってくる遠距離の敵（特に障害物越しの投石）は、離れるより詰めて倒す
	if player.Health*100 >= player.MaxHealth*40 {
		if enemy, ok := b.rangedThreat(); ok {
			addGoal("toRanged", b.enemyGoal(enemy))
		}
	}
	if healthy {
		if enemy, ok := b.nearestEnemy(8); ok && !b.isStrongEnemy(enemy) {
			addGoal("toEnemy", b.enemyGoal(enemy))
		}
	}
	addGoal("toItem", b.itemGoal)
	addGoal("explore", b.unexploredGoal)
	if b.expertWantsToDescend() {
		goals = append([]func(x, y int) bool{b.stairsGoal}, goals...)
		labels = append([]string{"toStairs"}, labels...)
	}
	addGoal("toStairs", b.stairsGoal)
	for i, goal := range goals {
		if dx, dy, ok := b.expertPathStep(goal); ok {
			b.act = labels[i]
			return b.move(dx, dy)
		}
	}
	// 通路を敵や罠にふさがれているときは、敵を倒し罠を踏んで進む
	for _, goal := range []func(x, y int) bool{b.unexploredGoal, b.stairsGoal} {
		if dx, dy, ok := b.pathStepThrough(goal, true); ok {
			b.act = "forceThrough"
			if isOccupied(g, player.X+dx, player.Y+dy) {
				return b.attack(dx, dy)
			}
			return b.move(dx, dy)
		}
	}
	b.act = "wait"
	return b.stepInPlace()
}

// expertFightStrong は強敵に対して動きを止める道具や攻撃のカードを使う。使ったら true。
func (b *bot) expertFightStrong(target Enemy, dx, dy int) (bool, error) {
	if !target.StatusAilments.Paralysis && target.StatusAilments.Sleep == 0 {
		if done, err := b.expertEscape([]Enemy{target}); done || err != nil {
			return done, err
		}
	}
	if i := b.findInventoryID(7); i >= 0 { // 黒炎弾は正面の敵へ
		return b.faceAndUse(dx, dy, i, "使う")
	}
	return b.useByID("使う", 31) // 真空斬り（部屋の敵全員）
}

// expertEscape は隣の敵の動きを止める（金縛り・睡眠・混乱・目潰しのカード、状態異常の薬を投げる）。
func (b *bot) expertEscape(adjacent []Enemy) (bool, error) {
	if used, err := b.useByID("使う", 15, 12, 13, 14); used || err != nil {
		return used, err
	}
	p := b.g.state.Player
	for _, enemy := range adjacent {
		if i := b.findInventoryID(17, 18, 19); i >= 0 {
			return b.faceAndUse(enemy.X-p.X, enemy.Y-p.Y, i, "投げる")
		}
	}
	return false, nil
}

// rangedThreat はプレイヤーを射程に収めている遠距離攻撃の敵のうち最も近いものを返す。
func (b *bot) rangedThreat() (Enemy, bool) {
	p := b.g.state.Player
	best, found, bestDistance := Enemy{}, false, 1<<30
	for _, enemy := range b.g.state.Enemies {
		attack := enemy.RangedAttack
		if attack.Kind == RangedAttackNone || enemy.StatusAilments.Seal {
			continue
		}
		d := max(abs(enemy.X-p.X), abs(enemy.Y-p.Y))
		if d > attack.MaxRange+1 || d < 2 {
			continue
		}
		aligned := enemy.X == p.X || enemy.Y == p.Y || abs(enemy.X-p.X) == abs(enemy.Y-p.Y)
		if (attack.Kind == RangedAttackRock || aligned) && d < bestDistance {
			best, found, bestDistance = enemy, true, d
		}
	}
	return best, found
}

// 休憩の判断に使う値。休憩は HP が restStartRate 未満で始め、restStopRate 以上になるか敵が来るまで続ける。
const (
	expertRestStartRate   = 60  // 休憩を始める HP（最大HP比 %）
	expertRestStopRate    = 90  // 休憩をやめる HP（最大HP比 %）
	expertRestEnemyRange  = 6   // この距離以内に敵がいたら休まない
	expertRestFloorTurns  = 900 // この階に長く居たら休まない（1200ターンの風より前に降りる）
	expertRestSatietyNeed = 30  // 満腹度と手持ちの食料の合計がこれ以下なら休まない
)

// expertShouldRest は足踏みで HP の自然回復を待つかどうか。
func (b *bot) expertShouldRest() bool {
	g := b.g
	p := &g.state.Player
	rate := p.Health * 100 / max(1, p.MaxHealth)
	if rate >= expertRestStopRate || (!b.resting && rate >= expertRestStartRate) {
		b.resting = false
		return false
	}
	food := 0
	for _, item := range p.Inventory {
		if f, ok := item.(*Food); ok && item.GetID() != rottenSausageItemID {
			food += f.Satiety
		}
	}
	safe := g.floorTurns < expertRestFloorTurns && p.Satiety+food > expertRestSatietyNeed && p.Satiety > 0 &&
		!b.inRangedFire(p.X, p.Y) && !b.bossAlive()
	if safe {
		if _, ok := b.nearestEnemy(expertRestEnemyRange); ok {
			safe = false
		}
	}
	b.resting = safe
	return safe
}

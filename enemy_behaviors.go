//go:build !test

package main

import "fmt"

// 追加計画の敵の固有行動に使う共通処理（docs/agents/wave4-B-enemies.md・wave4-C-enemies.md）。
// 系統ごとの行動は MonsterDefinition.Behavior / OnDefeated に関数を設定し、この中の処理を組み合わせて書く。

// EnemyBehavior は系統固有の行動。actEnemy が共通の状態処理（睡眠・金縛り・混乱・目潰し・擬態・
// 盗品を持った逃走）と発見状態の更新を終えた後、ボスの特技・ワープ・遠距離攻撃・接近・通常攻撃より前に呼ぶ。
// true を返すとその回の行動を終える（false なら通常の行動を続ける）。封印中は呼ばれない。
// i は g.state.Enemies の添字。行動の中で敵を増やす・消す場合は、呼び出し後に添字がずれないよう
// 自分より後ろへの追加（append）だけにする。
type EnemyBehavior func(g *Game, i int) bool

// EnemyDefeatedFunc は敵が倒れた後（フロアから取り除いた後）に呼ばれる。defeated は倒れた敵の写し。
// 封印中に倒れた場合は呼ばれない。ジェノサイドのカードで消えた場合も呼ばれない。
type EnemyDefeatedFunc func(g *Game, defeated Enemy)

// runEnemyBehavior は敵 i の系統固有の行動を実行し、行動を終えたら true を返す。
func (g *Game) runEnemyBehavior(i int) bool {
	enemy := &g.state.Enemies[i]
	behavior := MonsterDefinitions[enemy.ID].Behavior
	if behavior == nil || enemy.StatusAilments.Seal {
		return false
	}
	return behavior(g, i)
}

// onEnemyDefeated は倒れた敵の撃破時の処理を呼ぶ。敵が倒れて取り除かれるすべての経路から呼ぶ。
func (g *Game) onEnemyDefeated(defeated Enemy) {
	g.playSE("defeat")
	onDefeated := MonsterDefinitions[defeated.ID].OnDefeated
	if onDefeated == nil || defeated.StatusAilments.Seal {
		return
	}
	onDefeated(g, defeated)
}

// --- 移動 ---

// stepEnemyToward は敵 i を (targetX, targetY) へ1マス近づける。動けたら true。
// プレイヤーへの接近（MoveTowardsPlayer）と同じ規則で、直進できなければ代替の方向を順に試す。
// 目標のマスそのもの（プレイヤーや別の敵がいるマス）には入らない。
func (g *Game) stepEnemyToward(enemyIndex, targetX, targetY int) bool {
	enemy := &g.state.Enemies[enemyIndex]
	dx := targetX - enemy.X
	dy := targetY - enemy.Y
	stepX, stepY := sign(dx), sign(dy)

	// 斜め移動が壁に阻まれる場合は縦か横だけの移動へ切り替える
	if dx != 0 && dy != 0 {
		blockUp, blockDown, blockLeft, blockRight := isBlocked(g, enemy.X, enemy.Y)
		blockDiagonal := isDiagonallyBlocked(g, enemy.X+stepX, enemy.Y+stepY)
		if blockDiagonal || ((dx > 0 && dy > 0 && (blockDown || blockRight)) ||
			(dx > 0 && dy < 0 && (blockUp || blockRight)) ||
			(dx < 0 && dy > 0 && (blockDown || blockLeft)) ||
			(dx < 0 && dy < 0 && (blockUp || blockLeft))) {
			if g.randIntn(2) == 0 {
				stepY = 0 // 縦移動をやめる
			} else {
				stepX = 0 // 横移動をやめる
			}
		}
	}

	// まず目標へ直進する
	if g.tryMoveEnemy(enemyIndex, stepX, stepY) {
		return true
	}

	// 直進できない場合は方向別の代替候補を順に試す
	for _, step := range enemyMoveFallbacks(g, enemyIndex, dx, dy) {
		if g.tryMoveEnemy(enemyIndex, step[0], step[1]) {
			return true
		}
	}
	return false
}

// --- ほかの敵の回復 ---

// injuredAllyNear は敵 i からチェビシェフ距離 maxDistance 以内にいる、HP の減ったほかの敵の添字を返す。
// 減った HP が最も多い敵を選び、同じなら添字の小さい敵。ボスと擬態中の敵は対象外。いなければ -1。
func (g *Game) injuredAllyNear(i, maxDistance int) int {
	self := g.state.Enemies[i]
	best, bestMissing := -1, 0
	for j := range g.state.Enemies {
		ally := g.state.Enemies[j]
		if j == i || MonsterDefinitions[ally.ID].IsBoss || isEnemyDisguised(ally) {
			continue
		}
		if max(abs(ally.X-self.X), abs(ally.Y-self.Y)) > maxDistance {
			continue
		}
		if missing := ally.MaxHealth - ally.Health; missing > bestMissing {
			best, bestMissing = j, missing
		}
	}
	return best
}

// healEnemy は敵 index の HP を amount 回復し（最大 HP まで）、実際に回復した量を返す。
func (g *Game) healEnemy(index, amount int) int {
	enemy := &g.state.Enemies[index]
	before := enemy.Health
	enemy.Health = recoveredValue(enemy.Health, enemy.MaxHealth, amount, false)
	return enemy.Health - before
}

// --- 仲間を呼ぶ ---

// summonMonstersAround は (centerX, centerY) からチェビシェフ距離 radius 以内の空いた床へ、
// 今の階の出現テーブルから選んだ敵を最大 count 体呼び出し、呼び出した数を返す。
// ジェノサイドで封じた系統と excludeIDs（呼び出し役自身の系統など）は選ばない。
// フロアの敵の上限（MaxEnemiesOnFloor）を超えない。呼び出された敵は海老さんを発見済みにする。
func (g *Game) summonMonstersAround(centerX, centerY, radius, count int, excludeIDs []int) int {
	banned := make(map[int]bool, len(g.genocidedMonsterIDs)+len(excludeIDs))
	for id := range g.genocidedMonsterIDs {
		banned[id] = true
	}
	for _, id := range excludeIDs {
		banned[id] = true
	}
	var free []Coordinate
	for _, offset := range nearestFreeOffsets(radius) {
		x, y := centerX+offset.X, centerY+offset.Y
		if g.canPlaceEnemyAt(x, y) {
			free = append(free, Coordinate{X: x, Y: y})
		}
	}
	summoned := 0
	for summoned < count && len(free) > 0 && len(g.state.Enemies) < MaxEnemiesOnFloor {
		id := selectMonsterForFloor(g.Floor, banned, g.randIntn)
		if id < 0 {
			break
		}
		k := g.randIntn(len(free))
		position := free[k]
		free = append(free[:k], free[k+1:]...)
		enemy := CreateEnemyByID(id, position.X, position.Y)
		prepareSpawnedEnemy(&enemy, g.Floor, g.randIntn)
		enemy.PlayerDiscovered = true
		g.state.Enemies = append(g.state.Enemies, enemy)
		summoned++
	}
	if summoned > 0 {
		g.miniMapDirty = true
	}
	return summoned
}

// canPlaceEnemyAt は敵を新しく置けるマス（マップ内の通れる床・通路で、階段・聖域・プレイヤー・ほかの敵がいない）かを返す。
func (g *Game) canPlaceEnemyAt(x, y int) bool {
	if !isPositionFree(g, x, y, -1) {
		return false
	}
	return g.state.Map[y][x].Type != "stairs"
}

// --- 固定ダメージの特殊攻撃 ---

// enqueueEnemyFixedAttack は敵 enemyIndex の特殊攻撃として、防御力で減らない damage を与える Action を積む。
// 盾の能力（回避・反射・属性軽減・カウンター）と装備能力のフック（ModifyIncomingDamage）は通常どおり通す。
// kind は属性（炎なら EnemyDamageFire で火喰いの盾が半減）、delivery は攻撃方法（近接なら EnemyAttackMelee）。
// message は攻撃の文（「ヒバナウミウシが火を吹いた。」）で、後ろに被ダメージの文が続く。
// attackName は死因の表示（「ヒバナウミウシの炎」の「炎」）。
// onHit はダメージが届いた（回避・反射されなかった）ときだけ、ダメージの後に Execute 内で呼ぶ（目潰しの付与など）。
// 聖域のカードの上にいるプレイヤーには何もしない。
func (g *Game) enqueueEnemyFixedAttack(enemyIndex, damage int, kind EnemyDamageKind, delivery EnemyAttackDelivery, message, attackName string, onHit func(g *Game)) {
	if g.playerOnSanctuary() {
		return
	}
	enemy := &g.state.Enemies[enemyIndex]
	defense := g.resolvePlayerDefense(damage, kind, delivery, enemy, false)
	enemyName := g.enemyDisplayName(enemy.Name)
	enemyID, enemyX, enemyY := enemy.ID, enemy.X, enemy.Y
	source := enemyRangedSource(enemy.ID, enemy.Name, attackName)
	dx, dy := sign(g.state.Player.X-enemy.X), sign(g.state.Player.Y-enemy.Y)
	g.Enqueue(Action{
		Duration: 0.5,
		Message:  message + g.shieldDefenseMessage(enemyName, defense),
		Execute: func(g *Game) {
			if index := g.findEnemyForDefense(enemyID, enemyX, enemyY); index >= 0 {
				g.state.Enemies[index].AttackTimer = 0.5
				g.state.Enemies[index].AttackDirection = determineDirection(dx, dy)
			}
			g.applyPlayerShieldDefense(defense, source, enemyID, enemyX, enemyY)
			if onHit != nil && !defense.Evaded && !defense.Reflected && !g.playerDead {
				onHit(g)
			}
		},
	})
}

// --- 遅れて現れる敵 ---

// pendingEnemySpawnRadius は予約した敵が現れる位置を、中心からこの距離まで探す。
const pendingEnemySpawnRadius = 2

// scheduleEnemySpawn は turns ターン後に (x, y) の近くへ monsterID の敵が現れるよう予約する。
func (g *Game) scheduleEnemySpawn(monsterID, x, y, turns int) {
	g.pendingEnemySpawns = append(g.pendingEnemySpawns, PendingEnemySpawn{MonsterID: monsterID, X: x, Y: y, Turns: turns})
}

// processPendingEnemySpawns は予約のターンを進め、時間になった敵を現す（AdvanceTurn から毎ターン呼ぶ）。
// 置ける場所が無い・敵が上限のときは次のターンにもう一度試す。ジェノサイドで封じた系統は現れず予約も消える。
func (g *Game) processPendingEnemySpawns() {
	if len(g.pendingEnemySpawns) == 0 {
		return
	}
	due, waiting := tickPendingEnemySpawns(g.pendingEnemySpawns)
	for _, spawn := range due {
		if g.genocidedMonsterIDs[spawn.MonsterID] {
			continue
		}
		if !g.spawnPendingEnemy(spawn) {
			spawn.Turns = 0
			waiting = append(waiting, spawn)
		}
	}
	g.pendingEnemySpawns = waiting
}

func (g *Game) spawnPendingEnemy(spawn PendingEnemySpawn) bool {
	if len(g.state.Enemies) >= MaxEnemiesOnFloor {
		return false
	}
	for _, offset := range nearestFreeOffsets(pendingEnemySpawnRadius) {
		x, y := spawn.X+offset.X, spawn.Y+offset.Y
		if !g.canPlaceEnemyAt(x, y) {
			continue
		}
		enemy := CreateEnemyByID(spawn.MonsterID, x, y)
		prepareSpawnedEnemy(&enemy, g.Floor, g.randIntn)
		enemy.PlayerDiscovered = true
		g.state.Enemies = append(g.state.Enemies, enemy)
		g.miniMapDirty = true
		g.EnqueueMessage(fmt.Sprintf("%sが現れた！", g.enemyDisplayName(enemy.Name)), 0.5)
		return true
	}
	return false
}

// --- 被弾時の処理と分裂 ---

// EnemyDamagedFunc はダメージを受けて倒れなかった敵の処理。i は g.state.Enemies の添字。
// 範囲攻撃のループの中からも呼ばれるので、敵を増やすときは末尾への追加（append）だけにし、取り除かない。
type EnemyDamagedFunc func(g *Game, i int, damage int)

// onEnemyDamaged は敵 index の被弾時の処理を呼ぶ。封印中・ダメージ0なら呼ばない。
// g.applyDamageToEnemy を通るダメージ（海老さんの攻撃・射撃・投擲・杖・カード・罠・吹き飛ばしの衝突・
// 盾の反撃）で呼ばれる。敵どうしの攻撃では呼ばれない。
func (g *Game) onEnemyDamaged(index, damage int) {
	enemy := g.state.Enemies[index]
	onDamaged := MonsterDefinitions[enemy.ID].OnDamaged
	if onDamaged == nil || damage <= 0 || enemy.StatusAilments.Seal {
		return
	}
	onDamaged(g, index, damage)
}

// splitEnemy は敵 index と同じ種類の分身を、隣の空いたマスのどれか（g.randIntn）に1体出す。
// 分身は今の HP を引き継ぎ、状態異常・持ち物は持たず、海老さんを発見済み。出した分身の添字を返す。
// 隣に空きが無い・フロアの敵が上限（MaxEnemiesOnFloor）なら出さずに -1。ボスは分裂しない。
func (g *Game) splitEnemy(index int) int {
	source := g.state.Enemies[index]
	if MonsterDefinitions[source.ID].IsBoss || len(g.state.Enemies) >= MaxEnemiesOnFloor {
		return -1
	}
	var free []Coordinate
	for _, offset := range nearestFreeOffsets(1)[1:] {
		x, y := source.X+offset.X, source.Y+offset.Y
		if g.canPlaceEnemyAt(x, y) {
			free = append(free, Coordinate{X: x, Y: y})
		}
	}
	if len(free) == 0 {
		return -1
	}
	position := free[g.randIntn(len(free))]
	clone := CreateEnemyByID(source.ID, position.X, position.Y)
	clone.Health = source.Health
	clone.PlayerDiscovered = true
	if enemyMasquerading(source) {
		disguiseEnemyAs(&clone, source.DisguiseID) // 化けている敵の分身は同じ姿
	}
	g.state.Enemies = append(g.state.Enemies, clone)
	g.miniMapDirty = true
	g.EnqueueMessage(fmt.Sprintf("%sが分裂した！", g.enemyDisplayName(source.Name)), 0.4)
	return len(g.state.Enemies) - 1
}

// --- 直線上の海老さん ---

// straightLineToPlayer は海老さんが敵 i から縦・横・斜めの一直線上 maxDistance マス以内にいて、
// 間のマスがすべて通れて敵もいないとき、海老さんへの向き (dx, dy) と距離を返す。
// 隣（距離1）も含む。斜めの角の壁は見ない（引き寄せる・弾き飛ばすなど、届くかだけを見る処理用）。
func (g *Game) straightLineToPlayer(i, maxDistance int) (dx, dy, distance int, ok bool) {
	enemy := g.state.Enemies[i]
	offsetX, offsetY := g.state.Player.X-enemy.X, g.state.Player.Y-enemy.Y
	if offsetX == 0 && offsetY == 0 {
		return 0, 0, 0, false
	}
	if offsetX != 0 && offsetY != 0 && abs(offsetX) != abs(offsetY) {
		return 0, 0, 0, false
	}
	dx, dy = sign(offsetX), sign(offsetY)
	distance = max(abs(offsetX), abs(offsetY))
	if distance > maxDistance {
		return 0, 0, 0, false
	}
	for step := 1; step < distance; step++ {
		x, y := enemy.X+dx*step, enemy.Y+dy*step
		if !g.terrainOpenAt(x, y) || g.enemyIndexAt(x, y) >= 0 {
			return 0, 0, 0, false
		}
	}
	return dx, dy, distance, true
}

// --- ゾワゾワ（敵の状態） ---

// enemyAttackBackward はゾワゾワの敵 i が、海老さんと反対の向きの隣のマスへ通常攻撃を出す。
// そこに別の敵がいればその敵を攻撃し（倒すと上位種になる）、いなければ空振りする。
// AttackFromEnemy が StatusAilments.Backward > 0 の敵のときに呼ぶ（特技・痛恨は出さない）。
func (g *Game) enemyAttackBackward(i int) {
	enemy := g.state.Enemies[i]
	dx, dy := sign(enemy.X-g.state.Player.X), sign(enemy.Y-g.state.Player.Y)
	if dx == 0 && dy == 0 {
		return
	}
	if target := g.enemyIndexAt(enemy.X+dx, enemy.Y+dy); target >= 0 {
		g.AttackEnemyFromBlindEnemy(i, target)
		return
	}
	enemyID, enemyX, enemyY := enemy.ID, enemy.X, enemy.Y
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sはゾワゾワして反対を向いて攻撃した", g.enemyDisplayName(enemy.Name)),
		Execute: func(g *Game) {
			if index := g.findEnemyForDefense(enemyID, enemyX, enemyY); index >= 0 {
				g.state.Enemies[index].AttackTimer = 0.5
				g.state.Enemies[index].AttackDirection = determineDirection(dx, dy)
			}
		},
	})
}

//go:build !test

package main

import "fmt"

// 投げた道具・撃った矢・杖の魔法弾の飛び方の共通処理（段3の共通基盤）。
//
// 飛ぶものは Projectile にまとめ、g.ThrowItem(p) が Action を積む。Action の Execute の中で
// flyProjectile が1マスずつ進め、壁・敵・射程の端で止まる。能力で変える項目（貫通・跳ね返り・
// 命中率・本数）は Projectile のフィールドで、装備能力は AbilityHooks.ModifyProjectile で書き換える。
// 1マスごとの処理は OnTile、敵に当たるかどうかの判定は BeforeHit に関数を積む。
// 敵が打ち返す（MonsterDefinition.ProjectileReturnPercent、段4）と、向きを反転して海老さんへ飛んで戻る
// （returnProjectile / hitPlayerWithReturnedProjectile）。

// ProjectileKind は飛ぶものの種類。
type ProjectileKind int

const (
	ProjectileThrow ProjectileKind = iota // 投げた道具
	ProjectileShot                        // 撃った矢（射撃。ダメージはプレイヤーの攻撃力＋矢の攻撃力）
	ProjectileBolt                        // 杖の魔法弾（壁のマスまで届き、床には落ちない）
)

// Projectile は飛んでいる道具1つ。
type Projectile struct {
	Item       Item
	Kind       ProjectileKind
	X, Y       int // 発射地点。飛んでいる間は今いるマス
	DX, DY     int // 進む向き（跳ね返ると反転する）
	Range      int // 進めるマスの数
	FromPlayer bool

	// 以下は能力で変える項目。ModifyProjectile のフックが書き換える。
	PierceWalls   bool // 壁を通り抜ける。射程の端で壁の中なら、最後に通った床に落ちる
	PierceTargets bool // 敵に当たっても止まらず、後ろの敵にも当たる（1体にでも当たれば道具は消える）
	Bounces       int  // 壁で跳ね返る残りの回数（跳ね返りは射程を使わない）
	HitPercent    int  // 敵に当たる確率（%）。100 以上なら乱数を使わない。外れると敵の後ろへ飛び続ける
	SureHit       bool // 必ず当たる（HitPercent と BeforeHit の外れを無視する）
	Volley        int  // 一度に撃つ本数（射撃のみ。shootArrow が矢を1本ずつ消費して撃つ）。1 が通常

	// OnTile は1マス進むたびに、進んだ先のマスで呼ばれる（壁を通り抜けている途中も呼ぶ）。
	// p.Stop() で飛ぶのをやめ、そのマス（壁の中なら最後に通った床）に落ちる。
	OnTile []func(g *Game, p *Projectile, x, y int)
	// BeforeHit は敵に当たる判定の前に呼ばれ、hit.HitPercent を変えられる。
	// 打ち返しはこれより前に判定する（打ち返された道具は当たらない）。
	BeforeHit []func(g *Game, p *Projectile, hit *ProjectileHit)

	// Returned は敵に打ち返されて戻ってくる途中か。打ち返されたものは二度と打ち返されない。
	Returned bool
	// 打ち返した敵（死因・盾の反射の相手）。ID・座標は打ち返した時点のもの
	ReturnerID, ReturnerX, ReturnerY int

	traveled int
	stopped  bool
	lastOpen Coordinate
	hits     int
}

// ProjectileHit は飛んでいる道具が敵のマスに入ったときの命中判定の条件。
type ProjectileHit struct {
	EnemyIndex int // g.state.Enemies の添字
	HitPercent int // 当たる確率（%）。初期値は Projectile.HitPercent
}

// Stop は飛ぶのをやめさせる（OnTile の中から呼ぶ）。
func (p *Projectile) Stop() { p.stopped = true }

// Traveled は発射地点から進んだマスの数（跳ね返りを含む道のり）。
func (p *Projectile) Traveled() int { return p.traveled }

// Hits はここまでに当たった敵の数。
func (p *Projectile) Hits() int { return p.hits }

// newPlayerProjectile は海老さんの向きへ飛ぶ道具を作り、装備能力のフックで能力を反映する。
func (g *Game) newPlayerProjectile(item Item, kind ProjectileKind, throwRange int) Projectile {
	player := &g.state.Player
	dx, dy := directionToDelta(player.GetDirection())
	p := Projectile{
		Item: item, Kind: kind,
		X: player.X, Y: player.Y, DX: dx, DY: dy,
		Range: throwRange, FromPlayer: true,
		HitPercent: projectileBaseHitPercent(kind), Volley: 1,
	}
	g.runModifyProjectileHooks(&p)
	return p
}

// thrownItemHitPercent は投げた道具・撃った矢が敵に当たる基本の確率（%）。杖の魔法弾は必ず当たる。
const thrownItemHitPercent = 85

// projectileBaseHitPercent は種類ごとの基本の命中率を返す。
func projectileBaseHitPercent(kind ProjectileKind) int {
	if kind == ProjectileBolt {
		return 100
	}
	return thrownItemHitPercent
}

// projectileKindFor は海老さんが道具を飛ばしたときの種類を返す（魔法弾・射撃・投擲）。
func (g *Game) projectileKindFor(item Item) ProjectileKind {
	if isCaneBolt(item) {
		return ProjectileBolt
	}
	if g.dPressed {
		return ProjectileShot
	}
	return ProjectileThrow
}

func isCaneBolt(item Item) bool {
	cane, ok := item.(*Cane)
	return ok && cane.BaseItem.Type == "Effect"
}

// projectileItemLabel は飛ばすときのメッセージに使う道具の名前と、識別済みかを返す。
func projectileItemLabel(item Item) (string, bool) {
	identifiable, ok := item.(Identifiable)
	if !ok {
		return item.GetName(), true
	}
	if !identifiable.IsIdentified() {
		return identifiable.GetName(), false
	}
	return getItemNameWithSharpness(item), true
}

// ThrowItem は道具 p を飛ばす Action を積む（「〜を投げた」「〜を撃った」「〜を使った」と表示し、
// 表示の後に飛ぶ）。海老さんが飛ばす道具は g.newPlayerProjectile で作る。
func (g *Game) ThrowItem(p Projectile) {
	itemName, identified := projectileItemLabel(p.Item)
	message := fmt.Sprintf("%sを投げた", itemName)
	switch p.Kind {
	case ProjectileBolt:
		message = fmt.Sprintf("%sを使った", itemName)
	case ProjectileShot:
		message = fmt.Sprintf("%sを撃った", itemName)
	}
	g.Enqueue(Action{
		Duration: 0.5,
		Message:  message,
		ItemName: itemName,
		Execute: func(g *Game) {
			flight := p
			g.flyProjectile(&flight)
		},
		IsIdentified: identified,
	})
}

// flyProjectile は p を1マスずつ進め、敵に当たるか、壁・マップの端・射程の端で落ちるまで処理する。
// 当たった敵への効果は onTargetHit、落ちた道具は着地のアニメーション（UpdateThrownItem）が床に置く。
func (g *Game) flyProjectile(p *Projectile) {
	// 射撃かどうかは描画の向き・ダメージの計算に使う。連射の2本目でも撃った時の状態に戻す
	g.dPressed = p.Kind == ProjectileShot
	g.ThrownItem = ThrownItem{Item: p.Item, Image: g.getItemImage(p.Item), X: p.X, Y: p.Y, DX: p.DX, DY: p.DY}
	p.lastOpen = Coordinate{X: p.X, Y: p.Y}
	removed := false
	removeFromSource := func() {
		if !removed {
			removed = true
			g.removeLaunchedItem(p)
		}
	}

	for p.traveled < p.Range && !p.stopped {
		nextX, nextY := p.X+p.DX, p.Y+p.DY
		// マップの外へは飛ばない（手前のマスに落ちる）
		if !g.inMapBounds(nextX, nextY) {
			break
		}
		// 壁だけでなく通路の外側など通れないマスでも止まる（突き抜けてマップ外へ出ないように）
		if tile := g.state.Map[nextY][nextX]; tile.Type == "wall" || tile.Blocked {
			if p.Bounces > 0 {
				p.Bounces--
				p.DX, p.DY = -p.DX, -p.DY
				// アニメーションは跳ね返った地点から描き直す
				g.ThrownItem.X, g.ThrownItem.Y, g.ThrownItem.DX, g.ThrownItem.DY = p.X, p.Y, p.DX, p.DY
				continue
			}
			if p.Kind == ProjectileBolt {
				// 杖の魔法弾は壁のマス自体に届く
				removeFromSource()
				g.landProjectile(p, Coordinate{X: nextX, Y: nextY})
				return
			}
			if !p.PierceWalls {
				break
			}
		}

		p.X, p.Y = nextX, nextY
		p.traveled++
		if g.projectilePassable(p.X, p.Y) {
			p.lastOpen = Coordinate{X: p.X, Y: p.Y}
		}
		for _, onTile := range p.OnTile {
			onTile(g, p, p.X, p.Y)
		}
		if p.stopped {
			break
		}

		if index := g.enemyIndexAt(p.X, p.Y); index >= 0 {
			if g.tryReturnProjectile(p, index) {
				continue
			}
			if !g.projectileHitsEnemy(p, index) {
				itemName, _ := projectileItemLabel(p.Item)
				g.EnqueueMessage(fmt.Sprintf("%sは%sに当たらなかった", itemName, g.enemyDisplayName(g.state.Enemies[index].Name)), 0.3)
				continue
			}
			removeFromSource()
			p.hits++
			g.hitEnemyWithProjectile(p, index)
			if !p.PierceTargets {
				return
			}
			continue
		}
		// 海老さんが飛ばした道具は、跳ね返って戻ってきても海老さんには当たらない（打ち返されたものは当たる）
		if !p.FromPlayer && p.X == g.state.Player.X && p.Y == g.state.Player.Y {
			removeFromSource()
			g.ThrownItemDestination = Coordinate{X: p.X, Y: p.Y}
			if p.Returned {
				g.TargetEnemy = nil
				g.thrownItemHitPlayer = true
				g.hitPlayerWithReturnedProjectile(p)
				return
			}
			g.onTargetHit(&g.state.Player, p.Item, g.selectedItemIndex)
			return
		}
	}

	removeFromSource()
	if p.hits > 0 {
		// 貫通して敵に当たった道具は、射程の端まで飛んで消える
		g.ThrownItemDestination = p.lastOpen
		return
	}
	g.landProjectile(p, p.lastOpenOr(g))
}

// lastOpenOr は今いるマスが通れればそこを、壁の中なら最後に通った床を返す。
func (p *Projectile) lastOpenOr(g *Game) Coordinate {
	if g.projectilePassable(p.X, p.Y) {
		return Coordinate{X: p.X, Y: p.Y}
	}
	return p.lastOpen
}

func (g *Game) inMapBounds(x, y int) bool {
	return y >= 0 && y < len(g.state.Map) && x >= 0 && x < len(g.state.Map[y])
}

// projectilePassable は道具が落ちられるマス（マップ内の壁でない通れるマス）かを返す。
func (g *Game) projectilePassable(x, y int) bool {
	if !g.inMapBounds(x, y) {
		return false
	}
	tile := g.state.Map[y][x]
	return tile.Type != "wall" && !tile.Blocked
}

// projectileHitsEnemy は p が敵 index に当たるかを判定する。必ず当たるときは乱数を使わない。
func (g *Game) projectileHitsEnemy(p *Projectile, index int) bool {
	hit := ProjectileHit{EnemyIndex: index, HitPercent: p.HitPercent}
	for _, beforeHit := range p.BeforeHit {
		beforeHit(g, p, &hit)
	}
	switch {
	case p.SureHit || hit.HitPercent >= 100:
		return true
	case hit.HitPercent <= 0:
		return false
	default:
		return g.projectileIntn()(100) < hit.HitPercent
	}
}

// hitEnemyWithProjectile は敵 index に道具が当たったときの処理（効果は onTargetHit）。
func (g *Game) hitEnemyWithProjectile(p *Projectile, index int) {
	g.TargetEnemyIndex = index
	g.ThrownItemDestination = Coordinate{X: p.X, Y: p.Y}
	g.TargetEnemy = &g.state.Enemies[index]
	g.onTargetHit(&g.state.Enemies[index], p.Item, index)
	g.showItemActions = false
	g.showInventory = false
	g.selectedItemIndex = 0
	g.selectedActionIndex = 0
}

// landProjectile は道具が position に落ちる（魔法弾は壁に届く）ときの処理。
// 床に置くのは着地のアニメーションの最後（UpdateThrownItem）。
func (g *Game) landProjectile(p *Projectile, position Coordinate) {
	g.TargetEnemy = nil
	p.Item.SetPosition(position.X, position.Y)
	g.ThrownItemDestination = position
	g.showItemActions = false
	g.showInventory = false
	g.isActioned = true
	g.selectedItemIndex = 0
	g.selectedActionIndex = 0
}

// removeLaunchedItem は飛ばした道具を元の場所（所持品または足元）から取り除く。
// 射撃の矢は shootArrow が1本ずつ減らしているので、撃ち尽くした矢の束だけを所持品から消す。
// 杖の魔法弾は杖の写しなので何もしない。
func (g *Game) removeLaunchedItem(p *Projectile) {
	if _, ok := p.Item.(*Arrow); ok && p.Kind == ProjectileShot {
		inventory := g.state.Player.Inventory[:0]
		for _, item := range g.state.Player.Inventory {
			if arrow, ok := item.(*Arrow); ok && arrow.ShotCount == 0 {
				continue
			}
			inventory = append(inventory, item)
		}
		g.state.Player.Inventory = inventory
		return
	}
	if p.Kind == ProjectileBolt {
		return
	}
	g.removeThrownItemFromSource(p.Item)
}

// --- 打ち返し（段4の共通基盤。ウチカエシエビ系） ---

// enemyProjectileReturnPercent は敵が飛んできた道具を打ち返す確率（%）を返す。
// 確率は ID の定義（MonsterDefinition.ProjectileReturnPercent）から引く。封印・睡眠・金縛り中は0。
func enemyProjectileReturnPercent(enemy Enemy) int {
	if enemy.StatusAilments.Seal || enemy.StatusAilments.Sleep != 0 || enemy.StatusAilments.Paralysis || statusActive(enemy.StatusAilments, StatusDecoy) {
		return 0
	}
	return MonsterDefinitions[enemy.ID].ProjectileReturnPercent
}

// tryReturnProjectile は敵 index が p を打ち返すかを判定し、打ち返したら true を返す。
// 打ち返しは命中判定より前に起こるので、百発百中（必ず当たる）・ヘタ投げ（必ず外れる）でも打ち返される。
// 一度打ち返されたものは打ち返さない。乱数は飛ぶ道具の命中判定と同じ randHooks.projectile。
func (g *Game) tryReturnProjectile(p *Projectile, index int) bool {
	if p.Returned {
		return false
	}
	percent := enemyProjectileReturnPercent(g.state.Enemies[index])
	if percent <= 0 || (percent < 100 && g.projectileIntn()(100) >= percent) {
		return false
	}
	g.returnProjectile(p, index)
	return true
}

// returnProjectile は p を敵 index が打ち返したことにし、向きを反転して海老さんの側へ飛ばし直す。
// 戻る道のりは来た道のりと同じマス数。装備能力で付いた貫通・跳ね返り・必中は消え、命中率は基本に戻る
// （戻る途中の敵には基本の命中率で当たる）。海老さんのマスに着くと hitPlayerWithReturnedProjectile。
func (g *Game) returnProjectile(p *Projectile, index int) {
	enemy := &g.state.Enemies[index]
	if enemyMasquerading(*enemy) {
		g.revealEnemy(index)
	}
	itemName, _ := projectileItemLabel(p.Item)
	g.EnqueueMessage(fmt.Sprintf("%sは%sを打ち返した！", g.enemyDisplayName(enemy.Name), itemName), 0.4)
	enemy.AttackTimer = 0.5
	enemy.AttackDirection = determineDirection(-p.DX, -p.DY)
	enemy.Direction = enemy.AttackDirection
	enemy.PlayerDiscovered = true

	p.Returned = true
	p.ReturnerID, p.ReturnerX, p.ReturnerY = enemy.ID, enemy.X, enemy.Y
	p.FromPlayer = false
	p.DX, p.DY = -p.DX, -p.DY
	p.Range = p.traveled * 2
	p.PierceWalls, p.PierceTargets, p.Bounces, p.SureHit = false, false, 0, false
	p.HitPercent = projectileBaseHitPercent(p.Kind)
	// アニメーションは打ち返した地点から描き直す
	g.ThrownItem.X, g.ThrownItem.Y, g.ThrownItem.DX, g.ThrownItem.DY = p.X, p.Y, p.DX, p.DY
}

// hitPlayerWithReturnedProjectile は打ち返された道具が海老さんに当たったときの処理。
//   - 杖の魔法弾: 海老さんには効果が無い。
//   - 薬: 投げ当てたときと同じ効果を海老さんが受ける（状態異常の無効化は効く）。
//   - そのほか（矢・投げた道具）: 打ち返した敵の遠距離攻撃として扱う。ダメージは敵に当てたときと同じ式
//     （射撃は海老さんの総攻撃力＋矢の攻撃力、投擲は固定の範囲）で海老さんの防御力で減り、会心は出ない。
//     盾の回避・反射（打ち返した敵へダメージを返す）・カメの甲羅などの被ダメージのフックが効く。
//     死因は「<打ち返した敵>の打ち返し」。
func (g *Game) hitPlayerWithReturnedProjectile(p *Projectile) {
	itemName, _ := projectileItemLabel(p.Item)
	if p.Kind == ProjectileBolt {
		g.EnqueueMessage(fmt.Sprintf("%sは海老さんに当たったが、何も起きなかった", itemName), 0.4)
		g.isActioned = true
		return
	}
	if _, ok := p.Item.(*Potion); ok {
		g.onTargetHit(&g.state.Player, p.Item, -1)
		return
	}
	player := &g.state.Player
	var damage int
	if p.Kind == ProjectileShot {
		attack := playerAttackTotal(player.AttackPower, player.Power, player.Level)
		if arrow, ok := p.Item.(*Arrow); ok {
			attack += arrow.AttackPower
		}
		damage = rollEnemyAttackDamageWith(attack, g.specialPlayerDefense(), statAttackMultiplier(player.StatusAilments)*statDefenseMultiplier(player.StatusAilments), g.damageIntn())
	} else {
		damage = rollThrownItemDamage(g.damageIntn())
	}
	var returner *Enemy
	returnerName := MonsterDefinitions[p.ReturnerID].Name
	displayName := returnerName
	if index := g.findEnemyForDefense(p.ReturnerID, p.ReturnerX, p.ReturnerY); index >= 0 {
		returner = &g.state.Enemies[index]
		displayName = returner.Name
	}
	displayName = g.enemyDisplayName(displayName)
	defense := g.resolvePlayerDefense(damage, EnemyDamageNormal, EnemyAttackRanged, returner, false)
	source := enemyRangedSource(p.ReturnerID, returnerName, "打ち返し")
	returnerID, returnerX, returnerY := p.ReturnerID, p.ReturnerX, p.ReturnerY
	g.Enqueue(Action{
		Duration: 0.5,
		Message:  fmt.Sprintf("打ち返された%sが飛んできた。", itemName) + g.shieldDefenseMessage(displayName, defense),
		Execute: func(g *Game) {
			g.applyPlayerShieldDefense(defense, source, returnerID, returnerX, returnerY)
			g.isActioned = true
		},
	})
}

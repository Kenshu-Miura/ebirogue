//go:build !test

package main

import "fmt"

// 敵が起こす範囲爆発の共通処理（段4の共通基盤。フクレフグ系の自爆など）。
//
// 規則は既存の爆発に合わせる。
//   - 海老さん: 防御力で減らない Damage を、爆発・環境の攻撃として受ける（爆風よけの盾で半減。
//     盾の回避・反射・カウンターは効かない）。聖域のカードの上でも受ける（敵の攻撃ではなく爆発のため）。
//     死因は「<爆発させた敵>の爆発」。
//   - ほかの敵: 同じ Damage を受ける（睡眠・金縛りは解ける）。ボスは enemyExplosionBossDamageCap が上限
//     （爆発の指輪と同じ）。倒れた敵の経験値は海老さんに入らず（爆発弾の巻き込みと同じ）、生き残った
//     爆発させた敵が倒した数だけ上位種になる。倒れなかった敵は被弾時の処理（OnDamaged）を呼ぶので、
//     爆発が別のフグの自爆を連鎖させることもある。化けている敵は正体を現す。
//   - 床の道具: 範囲内の道具は消し飛ぶ（床に貼りついた聖域のカードは残る）。
//
// 処理は Action の Execute の中で行うので、OnDamaged（範囲攻撃のループの中から呼ばれる）や
// 特殊攻撃の中から g.enqueueEnemyExplosion を呼んでよい。

// enemyExplosionBossDamageCap は敵の爆発がボスへ与えるダメージの上限（爆発の指輪と同じ）。
const enemyExplosionBossDamageCap = 50

// EnemyExplosion は敵が起こす爆発1回。
type EnemyExplosion struct {
	// 爆発させた敵（死因・上位種化・自爆の対象）。ID・座標は enqueue した時点のもの
	BomberIndex int
	// 爆発の中心と半径（チェビシェフ距離。1 なら周囲1マス）
	CenterX, CenterY, Radius int
	// 海老さん・ほかの敵へのダメージ（防御力で減らない）
	Damage int
	// SelfDestruct なら爆発させた敵自身も消える（経験値は入らない。撃破時の処理 OnDefeated は呼ぶ）。
	// false なら爆発させた敵は巻き込まれない
	SelfDestruct bool
	// Message は爆発の文（空なら「〜が爆発した！」）
	Message string
}

// enqueueEnemyExplosion は爆発の Action を積む。爆発させた敵が Execute の時点でいなくなっていても爆発する。
func (g *Game) enqueueEnemyExplosion(explosion EnemyExplosion) {
	if explosion.BomberIndex < 0 || explosion.BomberIndex >= len(g.state.Enemies) {
		return
	}
	bomber := g.state.Enemies[explosion.BomberIndex]
	bomberID, bomberX, bomberY := bomber.ID, bomber.X, bomber.Y
	source := enemyRangedSource(bomber.ID, MonsterDefinitions[bomber.ID].Name, "爆発")
	message := explosion.Message
	if message == "" {
		message = fmt.Sprintf("%sが爆発した！", g.enemyDisplayName(bomber.Name))
	}
	g.Enqueue(Action{
		Duration: 0.5,
		Message:  message,
		Execute: func(g *Game) {
			g.rangedAttackEffect = RangedAttackEffect{
				Kind:    RangedAttackExplosion,
				OriginX: explosion.CenterX, OriginY: explosion.CenterY,
				TargetX: explosion.CenterX, TargetY: explosion.CenterY,
				BlastRadius: explosion.Radius,
				Timer:       rangedAttackEffectDuration,
			}
			if explosion.SelfDestruct {
				if index := g.findEnemyForDefense(bomberID, bomberX, bomberY); index >= 0 {
					defeated := g.state.Enemies[index]
					g.dropEnemyHeldItem(index)
					g.state.Enemies = append(g.state.Enemies[:index], g.state.Enemies[index+1:]...)
					g.onEnemyDefeated(defeated)
				}
			}
			// 海老さんへのダメージを先に積み、誘爆した地雷の割合のダメージはその後の HP から数える。
			hp := g.state.Player.Health
			if withinBlastRadius(explosion.CenterX, explosion.CenterY, g.state.Player.X, g.state.Player.Y, explosion.Radius) {
				hp -= g.explodePlayer(explosion.Damage, source)
			}
			g.explodeTrapsChain(explosion.CenterX, explosion.CenterY, explosion.Radius, false, 0, &hp)
			g.explodeItemsAround(explosion.CenterX, explosion.CenterY, explosion.Radius)
			g.explodeEnemiesAround(explosion, bomberID, bomberX, bomberY)
			g.miniMapDirty = true
		},
	})
}

// explodePlayer は海老さんへ爆発のダメージを与える（爆風よけの盾で半減）。
// explodePlayer は盾の軽減の後に受けるダメージを返す（HP は積んだ Action で減る）。
func (g *Game) explodePlayer(damage int, source DamageSource) int {
	defense := g.resolvePlayerShieldDefense(damage, EnemyDamageExplosion, EnemyAttackEnvironment)
	message := fmt.Sprintf("爆発に巻き込まれ、%dダメージを受けた！", defense.Damage)
	if defense.Resisted && g.state.Player.EquippedArmor != nil {
		message = fmt.Sprintf("爆発に巻き込まれたが、%sが威力を抑えて%dダメージを受けた！", g.state.Player.EquippedArmor.Name, defense.Damage)
	}
	g.Enqueue(Action{
		Duration: 0.5,
		Message:  message,
		Execute: func(g *Game) {
			g.applyPlayerShieldDefense(defense, source, -1, -1, -1)
		},
	})
	return defense.Damage
}

// explodeEnemiesAround は爆発の範囲にいる敵（爆発させた敵を除く）へダメージを与える。
func (g *Game) explodeEnemiesAround(explosion EnemyExplosion, bomberID, bomberX, bomberY int) {
	defeatedCount := 0
	for i := len(g.state.Enemies) - 1; i >= 0; i-- {
		if i >= len(g.state.Enemies) {
			continue
		}
		target := &g.state.Enemies[i]
		if target.ID == bomberID && target.X == bomberX && target.Y == bomberY {
			continue
		}
		if !withinBlastRadius(explosion.CenterX, explosion.CenterY, target.X, target.Y, explosion.Radius) {
			continue
		}
		damage := explosion.Damage
		if MonsterDefinitions[target.ID].IsBoss {
			damage = min(damage, enemyExplosionBossDamageCap)
		}
		if enemyMasquerading(*target) {
			g.revealEnemy(i)
		}
		target.Health -= damage
		clearStatus(&target.StatusAilments, StatusSleep)
		clearStatus(&target.StatusAilments, StatusParalysis)
		if target.Health <= 0 {
			defeated := *target
			g.EnqueueMessage(fmt.Sprintf("%sは爆発に巻き込まれて倒れた", g.enemyDisplayName(defeated.Name)), 0.4)
			g.dropEnemyHeldItem(i)
			g.state.Enemies = append(g.state.Enemies[:i], g.state.Enemies[i+1:]...)
			defeatedCount++
			g.onEnemyDefeated(defeated)
			continue
		}
		g.onEnemyDamaged(i, damage)
	}
	if explosion.SelfDestruct {
		return
	}
	if bomberIndex := g.findEnemyForDefense(bomberID, bomberX, bomberY); bomberIndex >= 0 {
		for range defeatedCount {
			if !g.levelUpEnemy(bomberIndex) {
				break
			}
		}
	}
}

// explodeItemsAround は爆発の範囲の床の道具を消す（床に貼りついた聖域のカードは残る）。
func (g *Game) explodeItemsAround(centerX, centerY, radius int) {
	kept := g.state.Items[:0]
	destroyed := 0
	for _, item := range g.state.Items {
		x, y := item.GetPosition()
		if withinBlastRadius(centerX, centerY, x, y, radius) && !isStuckSanctuaryItem(item) {
			destroyed++
			continue
		}
		kept = append(kept, item)
	}
	g.state.Items = kept
	if destroyed > 0 {
		g.EnqueueMessage("床の道具が爆発で消し飛んだ", 0.4)
	}
}

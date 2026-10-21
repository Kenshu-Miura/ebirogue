//go:build !test

package main

import "time"

// damagePlayer はプレイヤーの HP を damage だけ減らし（0 未満にはしない）、死因として source を記録する。
// プレイヤーへのダメージはすべてこの関数を通す（死亡判定 checkDeath は呼び出し側で行う）。
func (g *Game) damagePlayer(damage int, source DamageSource) {
	if damage <= 0 || statusActive(g.state.Player.StatusAilments, StatusInvincible) {
		return
	}
	g.playSE("hurt")
	if g.headless {
		g.headlessDamageTaken += damage
	}
	minHealth := 0
	if g.immortal {
		minHealth = 1 // 計測用の不死モード（BOT のレベル曲線を深い階まで測る）
	} else if g.state.Player.Health-damage <= 0 && g.runPreventDeathHooks(damage, source) {
		minHealth = 1 // 装備能力のフック PreventDeath（根性の盾など）で倒れる直前に踏みとどまった
	}
	g.state.Player.Health = max(minHealth, g.state.Player.Health-damage)
	g.lastDamageSource = trueEnemySource(source)
}

// trueEnemySource は敵による死因の名前を正体の名前にする（別の敵に化けた敵の攻撃でも、化けた姿の名前を記録しない）。
func trueEnemySource(source DamageSource) DamageSource {
	if source.Kind != DamageSourceEnemyAttack && source.Kind != DamageSourceEnemyRanged {
		return source
	}
	if definition, ok := MonsterDefinitions[source.EnemyID]; ok {
		source.EnemyName = definition.Name
	}
	return source
}

// buildRunResult は現在の状態から冒険の結果を作る。
func (g *Game) buildRunResult(cleared bool) RunResult {
	player := &g.state.Player
	result := RunResult{
		Seed:       g.seed,
		Difficulty: g.difficulty,
		Cleared:    cleared,
		Floor:      g.Floor,
		Turns:      g.turnCount,
		Level:      player.Level,
		Satiety:    player.Satiety,
		Inventory:  []string{},
		Stats:      g.stats,
		Equipment:  []string{},
	}
	if !cleared {
		result.DeathCause = g.lastDamageSource
		result.DeathMessage = deathMessage(g.Floor, g.lastDamageSource)
	}
	for _, item := range player.Inventory {
		result.Inventory = append(result.Inventory, item.GetName())
		result.InventoryIDs = append(result.InventoryIDs, item.GetID())
	}
	if player.EquippedWeapon != nil {
		result.Equipment = append(result.Equipment, player.EquippedWeapon.GetName())
	}
	if player.EquippedArmor != nil {
		result.Equipment = append(result.Equipment, player.EquippedArmor.GetName())
	}
	if player.EquippedArrow != nil {
		result.Equipment = append(result.Equipment, player.EquippedArrow.GetName())
	}
	for _, accessory := range player.EquippedAccessories {
		if accessory != nil {
			result.Equipment = append(result.Equipment, accessory.GetName())
		}
	}
	return result
}

// finishRun は冒険の終了（死亡・クリア）を1回だけ記録する。
// 結果を lastRunResult に残して記録一覧へ保存し、中断データを削除する（復活による再開を防ぐ）。
func (g *Game) finishRun(cleared bool) {
	if g.lastRunResult != nil {
		return
	}
	result := g.buildRunResult(cleared)
	if g.persistent {
		result.EndedAt = time.Now().Format(time.RFC3339)
	}
	g.lastRunResult = &result
	g.deleteSave()
	g.saveRunResult(result)
}

// LastRunResult は直近に終了した冒険の結果を返す（終了していなければ nil）。
func (g *Game) LastRunResult() *RunResult {
	return g.lastRunResult
}

// itemKind は統計に使うアイテムの種類名を返す。
func itemKind(item Item) string {
	switch item.(type) {
	case *Weapon:
		return "Weapon"
	case *Armor:
		return "Armor"
	case *Arrow:
		return "Arrow"
	case *Food:
		return "Food"
	case *Potion:
		return "Potion"
	case *Card:
		return "Card"
	case *Cane:
		return "Cane"
	case *Accessory:
		return "Accessory"
	case *Money:
		return "Money"
	case *Trap:
		return "Trap"
	case *Pot:
		return "Pot"
	default:
		return "Other"
	}
}

// recordItemUse は道具を使ったことを統計に記録する。
func (g *Game) recordItemUse(item Item) {
	g.stats.recordUse(itemKind(item), item.GetID(), g.Floor, g.turnCount)
}

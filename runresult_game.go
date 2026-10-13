//go:build !test

package main

import "time"

// damagePlayer はプレイヤーの HP を damage だけ減らし（0 未満にはしない）、死因として source を記録する。
// プレイヤーへのダメージはすべてこの関数を通す（死亡判定 checkDeath は呼び出し側で行う）。
func (g *Game) damagePlayer(damage int, source DamageSource) {
	if damage <= 0 {
		return
	}
	g.state.Player.Health = max(0, g.state.Player.Health-damage)
	g.lastDamageSource = source
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
		Inventory:  []string{},
		Equipment:  []string{},
	}
	if !cleared {
		result.DeathCause = g.lastDamageSource
		result.DeathMessage = deathMessage(g.Floor, g.lastDamageSource)
	}
	for _, item := range player.Inventory {
		result.Inventory = append(result.Inventory, item.GetName())
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

//go:build !test

package main

import (
	"strings"
	"testing"
)

// newDeathTestGame は 12x7 の部屋に HP の少ないプレイヤーだけを置いたヘッドレスゲームを作る。
func newDeathTestGame(t *testing.T, health int) *Game {
	t.Helper()
	g, _ := newHeadlessGame(5, DifficultyAdvanced)
	g.state.Map = makeTestFloorMap(12, 7)
	g.rooms = []Room{{ID: 0, X: 0, Y: 0, Width: 12, Height: 7}}
	g.state.Player.X, g.state.Player.Y = 2, 3
	g.state.Player.Health = health
	g.state.Enemies = nil
	g.state.Items = nil
	g.state.MapTraps = nil
	g.Floor = 4
	g.randHooks = randHooks{damage: func(n int) int { return n - 1 }, shieldDefense: func(int) int { return 99 }}
	return g
}

// finishDeath は死亡処理を最後まで進め、記録された結果を返す。
func finishDeath(t *testing.T, g *Game) *RunResult {
	t.Helper()
	if err := g.RunUntilInputReady(2000); err != nil {
		t.Fatal(err)
	}
	if !g.RunEnded() {
		t.Fatalf("run should have ended (HP %d)", g.state.Player.Health)
	}
	result := g.LastRunResult()
	if result == nil || result.Cleared {
		t.Fatalf("death result = %+v", result)
	}
	logged := strings.Join(g.messageLog.Messages, "\n")
	if !strings.Contains(logged, result.DeathMessage) {
		t.Fatalf("death message %q was not shown; log:\n%s", result.DeathMessage, logged)
	}
	return result
}

func TestDeathCauseEnemyNormalAttack(t *testing.T) {
	g := newDeathTestGame(t, 1)
	g.state.Enemies = []Enemy{CreateEnemyByID(2, 3, 3)} // マムル
	g.enqueueEnemyNormalAttack(0)
	result := finishDeath(t, g)
	if result.DeathCause != enemyAttackSource(2, "マムル") {
		t.Fatalf("cause = %+v", result.DeathCause)
	}
	if result.DeathMessage != "4階で マムル にやられた" || result.Floor != 4 || result.Seed != 5 {
		t.Fatalf("result = %+v", result)
	}
}

func TestDeathCauseEnemyRangedAttack(t *testing.T) {
	g := newDeathTestGame(t, 1)
	enemy := CreateEnemyByID(38, 6, 3) // 火吹きイカ
	enemy.PlayerDiscovered = true
	g.state.Enemies = []Enemy{enemy}
	if !g.tryEnemyRangedAttack(0) {
		t.Fatal("ranged attack should be available")
	}
	result := finishDeath(t, g)
	if result.DeathCause != enemyRangedSource(38, "火吹きイカ", "炎") {
		t.Fatalf("cause = %+v", result.DeathCause)
	}
	if result.DeathMessage != "4階で 火吹きイカ の炎にやられた" {
		t.Fatalf("message = %q", result.DeathMessage)
	}
}

func TestDeathCauseTrap(t *testing.T) {
	g := newDeathTestGame(t, 3)
	trap := createMapTrapByID(1, 2, 3) // 毒矢の罠（5ダメージ）
	trap.FailRate = 0
	g.state.MapTraps = []MapTrap{trap}
	g.stepOnTrap(0)
	result := finishDeath(t, g)
	if result.DeathCause != trapSource(1, "毒矢の罠") {
		t.Fatalf("cause = %+v", result.DeathCause)
	}
}

func TestDeathCauseStarvationAndPoison(t *testing.T) {
	starving := newDeathTestGame(t, 1)
	starving.state.Player.Satiety = 0
	starving.IncrementMoveCount()
	if result := finishDeath(t, starving); result.DeathCause.Kind != DamageSourceStarvation {
		t.Fatalf("starvation cause = %+v", result.DeathCause)
	}

	poisoned := newDeathTestGame(t, 2)
	poisoned.state.Player.StatusAilments.Poison = 5
	poisoned.IncrementMoveCount()
	if result := finishDeath(t, poisoned); result.DeathCause.Kind != DamageSourcePoison {
		t.Fatalf("poison cause = %+v", result.DeathCause)
	}
}

func TestDamageSourceIsLastHit(t *testing.T) {
	g := newDeathTestGame(t, 100)
	g.damagePlayer(10, DamageSource{Kind: DamageSourcePoison})
	g.damagePlayer(0, DamageSource{Kind: DamageSourceStarvation}) // 0ダメージは原因を上書きしない
	if g.lastDamageSource.Kind != DamageSourcePoison || g.state.Player.Health != 90 {
		t.Fatalf("source %+v, HP %d", g.lastDamageSource, g.state.Player.Health)
	}
	g.damagePlayer(1000, enemyAttackSource(0, "エビ"))
	if g.state.Player.Health != 0 || g.lastDamageSource.EnemyName != "エビ" {
		t.Fatal("damage should clamp HP at 0 and record the last source")
	}
}

func TestRunResultIncludesInventoryAndEquipment(t *testing.T) {
	g := newDeathTestGame(t, 50)
	weapon := buildItemFromTemplate(20, 0, 0).(*Weapon)
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{weapon, food}
	g.state.Player.EquippedWeapon = weapon
	g.turnCount = 123
	result := g.buildRunResult(true)
	if !result.Cleared || result.Turns != 123 || len(result.Inventory) != 2 || len(result.Equipment) != 1 ||
		result.Equipment[0] != weapon.GetName() || result.DeathMessage != "" {
		t.Fatalf("result = %+v", result)
	}
}

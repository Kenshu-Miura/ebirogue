//go:build !test

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// simViolation は BOT のテストプレイで見つかった不変条件違反・異常の1件。
type simViolation struct {
	Kind   string // 種類（"hp_range" "enemy_in_wall" "softlock" "panic" など）
	Detail string
	Floor  int
	Turn   int
}

func newViolation(g *Game, kind, format string, args ...any) simViolation {
	return simViolation{Kind: kind, Detail: fmt.Sprintf(format, args...), Floor: g.Floor, Turn: g.turnCount}
}

// checkInvariants は入力待ちの状態で常に成り立つべき条件を検査し、違反を返す。
func checkInvariants(g *Game) []simViolation {
	var out []simViolation
	if len(g.state.Map) == 0 || len(g.state.Map[0]) == 0 {
		return []simViolation{newViolation(g, "map_empty", "マップが空")}
	}
	height, width := len(g.state.Map), len(g.state.Map[0])
	inBounds := func(x, y int) bool { return x >= 0 && y >= 0 && x < width && y < height }

	player := &g.state.Player
	if !inBounds(player.X, player.Y) {
		out = append(out, newViolation(g, "player_out_of_bounds", "プレイヤー座標 (%d, %d)", player.X, player.Y))
	} else if g.state.Map[player.Y][player.X].Blocked {
		out = append(out, newViolation(g, "player_in_wall", "プレイヤーが壁の中 (%d, %d) %s", player.X, player.Y, g.state.Map[player.Y][player.X].Type))
	}
	if player.MaxHealth <= 0 || player.Health < 0 || player.Health > player.MaxHealth {
		out = append(out, newViolation(g, "hp_range", "HP %d / 最大 %d", player.Health, player.MaxHealth))
	}
	if player.Satiety < 0 || player.Satiety > player.MaxSatiety {
		out = append(out, newViolation(g, "satiety_range", "満腹度 %d / 最大 %d", player.Satiety, player.MaxSatiety))
	}
	if len(player.Inventory) > player.MaxInventory {
		out = append(out, newViolation(g, "inventory_overflow", "所持品 %d / 上限 %d", len(player.Inventory), player.MaxInventory))
	}

	occupied := map[Coordinate]string{}
	for _, enemy := range g.state.Enemies {
		pos := Coordinate{X: enemy.X, Y: enemy.Y}
		if !inBounds(enemy.X, enemy.Y) {
			out = append(out, newViolation(g, "enemy_out_of_bounds", "%s (ID %d) の座標 (%d, %d)", enemy.Name, enemy.ID, enemy.X, enemy.Y))
			continue
		}
		if !enemyTerrainPassable(g.state.Map[enemy.Y][enemy.X], enemy.Traits) && enemy.SpecialMovement != SpecialMovementWallPass {
			out = append(out, newViolation(g, "enemy_in_wall", "%s (ID %d) が壁の中 (%d, %d)", enemy.Name, enemy.ID, enemy.X, enemy.Y))
		}
		if enemy.Health <= 0 {
			out = append(out, newViolation(g, "dead_enemy_remains", "%s (ID %d) の HP が %d のまま残っている", enemy.Name, enemy.ID, enemy.Health))
		}
		if enemy.X == player.X && enemy.Y == player.Y {
			out = append(out, newViolation(g, "enemy_on_player", "%s (ID %d) がプレイヤーと同じマス (%d, %d)", enemy.Name, enemy.ID, enemy.X, enemy.Y))
		}
		if other, dup := occupied[pos]; dup {
			out = append(out, newViolation(g, "enemy_overlap", "%s と %s が同じマス (%d, %d)", other, enemy.Name, enemy.X, enemy.Y))
		}
		occupied[pos] = enemy.Name
	}
	for _, item := range g.state.Items {
		if x, y := item.GetPosition(); !inBounds(x, y) {
			out = append(out, newViolation(g, "item_out_of_bounds", "%s の座標 (%d, %d)", item.GetName(), x, y))
		}
	}
	for _, trap := range g.state.MapTraps {
		if !inBounds(trap.X, trap.Y) {
			out = append(out, newViolation(g, "trap_out_of_bounds", "%s の座標 (%d, %d)", trap.Name, trap.X, trap.Y))
		}
	}
	return out
}

// saveRoundTrip は中断セーブ→再開を行い、再開後のゲームを返す。
// 再開後に作り直したセーブデータが元と一致しなければ違反を返す（その場合も再開後のゲームで続行する）。
func saveRoundTrip(g *Game) (*Game, []simViolation) {
	before, err := json.Marshal(g.buildSaveData())
	if err != nil {
		return g, []simViolation{newViolation(g, "save_encode", "%v", err)}
	}
	save, err := decodeSaveData(before)
	if err != nil {
		return g, []simViolation{newViolation(g, "save_decode", "%v", err)}
	}
	restored := &Game{
		input:                g.input,
		headless:             g.headless,
		fastForward:          g.fastForward,
		immortal:             g.immortal,
		tmpSelectedItemIndex: -1,
		customNames:          map[int]string{},
		settings:             g.settings,
	}
	if err := restored.applySaveData(save); err != nil {
		return g, []simViolation{newViolation(g, "save_apply", "%v", err)}
	}
	restored.fadeInProgress = 1.0
	after, err := json.Marshal(restored.buildSaveData())
	if err != nil {
		return restored, []simViolation{newViolation(g, "save_encode", "%v", err)}
	}
	if !bytes.Equal(before, after) {
		return restored, []simViolation{newViolation(g, "save_mismatch", "再開後のセーブデータが一致しない（%s）", firstDifference(before, after))}
	}
	return restored, nil
}

// firstDifference は2つのバイト列が最初に食い違う付近を短く示す。
func firstDifference(a, b []byte) string {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	clip := func(s []byte) string {
		start, end := max(0, i-40), min(len(s), i+40)
		if start > len(s) {
			return ""
		}
		return string(s[start:end])
	}
	return fmt.Sprintf("offset %d: %q vs %q", i, clip(a), clip(b))
}

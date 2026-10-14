//go:build !test

package main

import (
	"strings"
	"testing"
)

func newPoisonTestGame(power int) *Game {
	g := newItemTrapTestGame()
	g.state.Player.Power, g.state.Player.MaxPower = power, 8
	return g
}

// runQueuedActions はキューを実行しながら表示されたメッセージを集める。
func runQueuedActions(g *Game) string {
	var messages []string
	for len(g.ActionQueue.Queue) > 0 {
		a := g.ActionQueue.Queue[0]
		g.ActionQueue.Queue = g.ActionQueue.Queue[1:]
		messages = append(messages, a.Message)
		if a.Execute != nil {
			a.Execute(g)
		}
	}
	return strings.Join(messages, "\n")
}

func TestDrainPlayerPower(t *testing.T) {
	g := newPoisonTestGame(2)
	if drained := g.drainPlayerPower(1); drained != 1 || g.state.Player.Power != 1 || g.state.Player.MaxPower != 8 {
		t.Fatalf("drained %d, power %d/%d", drained, g.state.Player.Power, g.state.Player.MaxPower)
	}
	if got := runQueuedActions(g); !strings.Contains(got, "パワーが1下がった") {
		t.Fatalf("messages = %q", got)
	}

	g = newPoisonTestGame(0)
	if drained := g.drainPlayerPower(1); drained != 0 || g.state.Player.Power != 0 {
		t.Fatalf("power 0: drained %d, power %d", drained, g.state.Player.Power)
	}
	if got := runQueuedActions(g); !strings.Contains(got, "これ以上下がらない") {
		t.Fatalf("messages = %q", got)
	}
}

func TestPoisonSnakeLineDrainsPower(t *testing.T) {
	for _, id := range []int{1, 20, 46} { // 毒ヘビ・猛毒ヘビ・ゲキドクヘビ
		for _, power := range []int{8, 0} {
			g := newPoisonTestGame(power)
			g.state.Enemies = []Enemy{CreateEnemyByID(id, 2, 3)}
			g.state.Enemies[0].SpecialAttack(&g.state.Enemies[0], g)
			if g.state.Player.Power != power {
				t.Fatalf("enemy %d: drain was not deferred", id)
			}
			runQueuedActions(g)
			if want := max(power-1, 0); g.state.Player.Power != want {
				t.Fatalf("enemy %d: power = %d, want %d", id, g.state.Player.Power, want)
			}
		}
	}
}

func stepOnPoisonArrowTrap(g *Game, failRate int) string {
	trap := createMapTrapByID(1, 3, 3)
	trap.FailRate = failRate
	g.state.MapTraps = []MapTrap{trap}
	g.stepOnTrap(0)
	return runQueuedActions(g)
}

func TestPoisonArrowTrapDamagesAndDrainsPower(t *testing.T) {
	if trap := createMapTrapByID(1, 0, 0); trap.Name != "毒矢の罠" || trap.FailRate != 20 {
		t.Fatalf("trap 1 = %q fail %d, want 毒矢の罠 20", trap.Name, trap.FailRate)
	}

	g := newPoisonTestGame(8)
	stepOnPoisonArrowTrap(g, 0)
	if hp := g.state.Player.Health; hp >= 40 || hp < 40-6 || g.state.Player.Power != 7 {
		t.Fatalf("HP %d power %d, want about 5 damage and power 7", hp, g.state.Player.Power)
	}

	// 不発なら何も起きない
	g = newPoisonTestGame(8)
	stepOnPoisonArrowTrap(g, 100)
	if g.state.Player.Health != 40 || g.state.Player.Power != 8 {
		t.Fatalf("failed trap: HP %d power %d", g.state.Player.Health, g.state.Player.Power)
	}

	// パワー0ではダメージだけ受ける
	g = newPoisonTestGame(0)
	messages := stepOnPoisonArrowTrap(g, 0)
	if g.state.Player.Health >= 40 || g.state.Player.Power != 0 || !strings.Contains(messages, "これ以上下がらない") {
		t.Fatalf("power 0: HP %d power %d messages %q", g.state.Player.Health, g.state.Player.Power, messages)
	}
}

func TestPoisonArrowTrapStatusResistance(t *testing.T) {
	for _, resisted := range []bool{true, false} {
		g := newPoisonTestGame(8)
		g.state.Player.EquippedArmor = buildItemFromTemplate(65, 0, 0).(*Armor)
		roll := 99
		if resisted {
			roll = 0
		}
		g.randHooks.statusResistance = func(int) int { return roll }
		messages := stepOnPoisonArrowTrap(g, 0)
		if g.state.Player.Health >= 40 {
			t.Fatalf("resisted=%v: 耐性があってもダメージは受ける", resisted)
		}
		want := 7
		if resisted {
			want = 8
		}
		if g.state.Player.Power != want {
			t.Fatalf("resisted=%v: power %d, want %d", resisted, g.state.Player.Power, want)
		}
		if resisted != strings.Contains(messages, "毒を防いだ") {
			t.Fatalf("resisted=%v: messages %q", resisted, messages)
		}
	}
}

func TestNaturalRegenByLevelAndHunger(t *testing.T) {
	for _, c := range []struct{ level, want int }{{1, 1}, {10, 2}, {20, 3}} {
		g := newPoisonTestGame(8)
		g.state.Player.Level = c.level
		g.state.Player.Health, g.state.Player.MaxHealth = 10, 300
		g.IncrementMoveCount()
		if got := g.state.Player.Health - 10; got != c.want {
			t.Fatalf("Lv%d regen = %d, want %d", c.level, got, c.want)
		}
	}
	// 最大HPを超えない
	g := newPoisonTestGame(8)
	g.state.Player.Level = 20
	g.state.Player.Health, g.state.Player.MaxHealth = 39, 40
	g.IncrementMoveCount()
	if g.state.Player.Health != 40 {
		t.Fatalf("HP = %d, want 40", g.state.Player.Health)
	}
	// 満腹度0では回復せず1減る
	g = newPoisonTestGame(8)
	g.state.Player.Satiety = 0
	g.state.Player.Health = 20
	g.IncrementMoveCount()
	if g.state.Player.Health != 19 {
		t.Fatalf("starving HP = %d, want 19", g.state.Player.Health)
	}
}

func TestEarlyBalanceValues(t *testing.T) {
	if got := MonsterDefinitions[5].RangedAttack.AttackPower; got != 8 {
		t.Errorf("イシガニの投石 = %d, want 8", got)
	}
	for _, id := range []int{1, 20, 46} {
		if got := MonsterDefinitions[id].SpecialAttackProbability; got != 0.15 {
			t.Errorf("enemy %d special attack probability = %v, want 0.15", id, got)
		}
	}
	count, cards := 0, 0
	for _, category := range itemSpawnCategories {
		if category.Name != "Card" {
			continue
		}
		cards = len(category.Pool)
		for _, id := range category.Pool {
			if id == 43 {
				count++
			}
		}
	}
	if count != powerUpCardSpawnWeight || cards != countTemplatesOfType("Card")+powerUpCardSpawnWeight-1 {
		t.Fatalf("power-up card appears %d times in a card pool of %d", count, cards)
	}
}

func countTemplatesOfType(itemType string) int {
	n := 0
	for _, template := range itemTemplates {
		if template.ItemType == itemType {
			n++
		}
	}
	return n
}

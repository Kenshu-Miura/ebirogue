//go:build !test

package main

import "fmt"

// アナジャコ系は Alerted だけを使う（近接・被弾後の追跡を保持）。
// Counter・Timer・Flag は使わず、コバンザメ系は汎用状態を使わない。
var expansionC2MonsterFamilies = []monsterFamily{
	{Base: MonsterDefinition{ID: 111, Type: "AnaJako", Name: "アナジャコ", Char: 'a', Health: 36, AttackPower: 8, DefensePower: 7, ExperiencePoints: 26},
		BaseImage: "img/ana_jako.png",
		Second:    monsterStage{112, "OoAnaJako", "オオアナジャコ", 63, 14, 10, 65, "img/oo_ana_jako.png"},
		Third:     monsterStage{113, "JigokuAnaJako", "ジゴクアナジャコ", 107, 28, 12, 162, "img/jigoku_ana_jako.png"},
		Spawns:    []monsterSpawnRange{{111, 7, 13, 5}, {112, 14, 21, 5}, {113, 22, 30, 5}}},
	{Base: MonsterDefinition{Traits: []EnemyTrait{EnemyTraitAquatic}, ID: 120, Type: "Kobanzame", Name: "コバンザメ", Char: 'z', Health: 28, AttackPower: 6, DefensePower: 4, ExperiencePoints: 20},
		BaseImage: "img/kobanzame.png",
		Second:    monsterStage{121, "Oobanzame", "オオバンザメ", 50, 11, 7, 50, "img/oobanzame.png"},
		Third:     monsterStage{122, "Senryouzame", "センリョウザメ", 85, 22, 9, 125, "img/senryouzame.png"},
		Spawns:    []monsterSpawnRange{{120, 5, 11, 6}, {121, 12, 19, 6}, {122, 20, 30, 6}}},
}

var kobanzameCoinAmount = map[int]int{120: 10, 121: 24, 122: 50}
var kobanzameCoinDamage = map[int]int{120: 6, 121: 14, 122: 30}

func init() {
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.Behavior = anaJakoBehavior }, 111, 112, 113)
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.Behavior = kobanzameBehavior }, 120, 121, 122)
	expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "新しい敵（6）", Lines: []string{
		"アナジャコ：近づくまで追ってこない。壁を掘りながら歩く。",
		"コバンザメ：離れた所から財布の小銭を投げつける。",
		"守銭の盾で小銭の吸い出しを防げる。",
		"上位種も同じ能力を持つ。",
	}})
}

func anaJakoBehavior(g *Game, i int) bool {
	if g.enemyHoldsUntilNear(i, 2) {
		if g.randIntn(4) == 0 {
			steps := [4][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
			step := steps[g.randIntn(4)]
			if g.stepEnemyDigging(i, step[0], step[1]) {
				return true
			}
		}
		moveRandomly(g, i)
		return true
	}
	e := g.state.Enemies[i]
	if !e.PlayerDiscovered || max(abs(e.X-g.state.Player.X), abs(e.Y-g.state.Player.Y)) <= 1 {
		return false
	}
	return g.stepEnemyTowardDigging(i, g.state.Player.X, g.state.Player.Y)
}

func kobanzameBehavior(g *Game, i int) bool {
	e := g.state.Enemies[i]
	amount := kobanzameCoinAmount[e.ID]
	if !e.PlayerDiscovered || g.playerBlocksTheft() || g.state.Player.Cash < amount {
		return false
	}
	_, _, distance, ok := g.straightLineToPlayer(i, 5)
	if !ok || distance < 2 || g.randIntn(2) != 0 {
		return false
	}
	name := g.enemyDisplayName(e.Name)
	g.Enqueue(Action{Duration: 0.5, Message: fmt.Sprintf("%sが海老さんの小銭を吸い出した！（%d円）", name, amount),
		Execute: func(g *Game) { g.state.Player.Cash = max(0, g.state.Player.Cash-amount) }})
	g.enqueueEnemyFixedAttack(i, kobanzameCoinDamage[e.ID], EnemyDamageNormal, EnemyAttackRanged,
		fmt.Sprintf("%sが小銭を投げつけた！", name), "小銭", nil)
	return true
}

//go:build !test

package main

import "fmt"

// expansionA1MonsterFamilies は追加計画の敵 A 前半（ID 66〜77、docs/agents/wave4-B-enemies.md）。
// 担当はこの表と、同じファイルの特殊攻撃・行動の関数だけを追加する（enemy_expansion.go がまとめて登録する）。
var expansionA1MonsterFamilies = []monsterFamily{
	{
		Base:      MonsterDefinition{Description: "隣から火を吹く。火喰いの盾で半減できる。", ID: 66, Type: "HibanaUmiushi", Name: "ヒバナウミウシ", Char: 'A', Health: 26, AttackPower: 6, DefensePower: 3, ExperiencePoints: 16, SpecialAttackProbability: 0.3},
		BaseImage: "img/hibana_umiushi.png",
		Second:    monsterStage{67, "KaenUmiushi", "カエンウミウシ", 48, 11, 6, 40, "img/kaen_umiushi.png"},
		Third:     monsterStage{68, "GoukaUmiushi", "ゴウカウミウシ", 82, 22, 8, 100, "img/gouka_umiushi.png"},
		Spawns:    []monsterSpawnRange{{66, 3, 4, 5}, {66, 5, 8, 8}, {67, 7, 8, 4}, {67, 9, 16, 7}, {68, 15, 16, 4}, {68, 17, 30, 7}},
	},
	{
		Base:      MonsterDefinition{Description: "2マス先から攻撃してくる。角越しにも届く。", ID: 69, Type: "TeppouEbi", Name: "テッポウエビ", Char: 'P', Health: 30, AttackPower: 7, DefensePower: 4, ExperiencePoints: 18},
		BaseImage: "img/teppou_ebi.png",
		Second:    monsterStage{70, "TaihouEbi", "タイホウエビ", 54, 12, 7, 45, "img/taihou_ebi.png"},
		Third:     monsterStage{71, "GouhouEbi", "ゴウホウエビ", 92, 24, 9, 112, "img/gouhou_ebi.png"},
		Spawns:    []monsterSpawnRange{{69, 4, 5, 5}, {69, 6, 10, 8}, {70, 10, 17, 7}, {71, 18, 30, 7}},
	},
	{
		Base:      MonsterDefinition{Description: "攻撃せず倍速で逃げ回る。経験値が多い。", Traits: []EnemyTrait{EnemyTraitAquatic}, ID: 72, Type: "NigeHaze", Name: "ニゲハゼ", Char: 'Z', Health: 14, AttackPower: 2, DefensePower: 2, ExperiencePoints: 24, SpecialMovement: SpecialMovementDoubleSpeed},
		BaseImage: "img/nige_haze.png",
		Second:    monsterStage{73, "TonzuraHaze", "トンズラハゼ", 26, 4, 5, 60, "img/tonzura_haze.png"},
		Third:     monsterStage{74, "KinpikaHaze", "キンピカハゼ", 44, 8, 7, 150, "img/kinpika_haze.png"},
		Spawns:    []monsterSpawnRange{{72, 2, 9, 4}, {73, 8, 16, 4}, {74, 15, 30, 3}},
	},
	{
		Base:      MonsterDefinition{Description: "同じ部屋で仲間を呼ぶ。", Traits: []EnemyTrait{EnemyTraitAquatic}, ID: 75, Type: "YobikomiBora", Name: "ヨビコミボラ", Char: 'Y', Health: 28, AttackPower: 5, DefensePower: 6, ExperiencePoints: 22},
		BaseImage: "img/yobikomi_bora.png",
		Second:    monsterStage{76, "YobiatsumeBora", "ヨビアツメボラ", 50, 9, 9, 55, "img/yobiatsume_bora.png"},
		Third:     monsterStage{77, "DaishoukanBora", "ダイショウカンボラ", 85, 18, 11, 138, "img/daishoukan_bora.png"},
		Spawns:    []monsterSpawnRange{{75, 5, 11, 5}, {76, 12, 19, 5}, {77, 20, 30, 5}},
	},
}

var hibanaFireDamage = map[int]int{66: 8, 67: 14, 68: 24}

func hibanaFireAttack(e *Enemy, g *Game) {
	name := g.enemyDisplayName(e.Name)
	if g.randIntn(100) < 25 {
		g.EnqueueMessage(fmt.Sprintf("%sは火を吹こうとしたが、しけっていた。", name), 0.5)
		return
	}
	// 特殊攻撃は実際の敵スライスの要素を受け取る。
	for i := range g.state.Enemies {
		if e == &g.state.Enemies[i] {
			g.enqueueEnemyFixedAttack(i, hibanaFireDamage[e.ID], EnemyDamageFire, EnemyAttackRanged, fmt.Sprintf("%sは火を吹いた。", name), "炎", nil)
			return
		}
	}
}

func teppouBehavior(g *Game, i int) bool {
	e := g.state.Enemies[i]
	if !e.PlayerDiscovered {
		return false
	}
	dx, dy := g.state.Player.X-e.X, g.state.Player.Y-e.Y
	if !((abs(dx) == 2 && dy == 0) || (dx == 0 && abs(dy) == 2) || (abs(dx) == 2 && abs(dy) == 2)) {
		return false
	}
	x, y := e.X+sign(dx), e.Y+sign(dy)
	if g.state.Map[y][x].Blocked {
		return false
	}
	for j := range g.state.Enemies {
		if g.state.Enemies[j].X == x && g.state.Enemies[j].Y == y {
			return false
		}
	}
	g.AttackFromEnemy(i)
	return true
}

func nigeHazeBehavior(g *Game, i int) bool {
	if g.state.Enemies[i].PlayerDiscovered {
		g.moveEnemyAwayFromPlayer(i, g.randIntn)
	} else {
		moveRandomly(g, i)
	}
	return true
}

func yobikomiBehavior(g *Game, i int) bool {
	e := g.state.Enemies[i]
	if !e.PlayerDiscovered || !isSameRoom(e.X, e.Y, g.state.Player.X, g.state.Player.Y, g.rooms) {
		return false
	}
	nearby := 0
	for j, other := range g.state.Enemies {
		if j != i && max(abs(other.X-e.X), abs(other.Y-e.Y)) <= 2 {
			nearby++
		}
	}
	if nearby >= 3 || g.randIntn(4) != 0 {
		return false
	}
	count := 1
	if e.ID == 77 {
		count = 2
	}
	// 召喚によるスライスの拡張もメッセージの実行時に行う。
	g.Enqueue(Action{Duration: 0.5, Message: fmt.Sprintf("%sがほら貝を鳴らした！", g.enemyDisplayName(e.Name)), Execute: func(g *Game) {
		if g.summonMonstersAround(e.X, e.Y, 2, count, monsterFamilyIDs(75)) == 0 {
			g.EnqueueMessage("しかし何も来なかった。", 0.5)
		}
	}})
	return true
}

func init() {
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.SpecialAttack = hibanaFireAttack }, 66, 67, 68)
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.Behavior = teppouBehavior }, 69, 70, 71)
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.Behavior = nigeHazeBehavior }, 72, 73, 74)
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.Behavior = yobikomiBehavior }, 75, 76, 77)
	expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "新しい敵（1）", Lines: []string{
		"ヒバナウミウシ：隣から火を吹く。火喰いの盾で半減。",
		"テッポウエビ：2マス先から攻撃してくる。角越しも可。",
		"ニゲハゼ：攻撃せず倍速で逃げ回る。経験値が多い。",
		"ヨビコミボラ：同じ部屋で仲間を呼ぶ。",
		"上位種も同じ能力を持つ。",
	}})
}

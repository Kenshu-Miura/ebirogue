//go:build !test

package main

import "fmt"

// expansionB1MonsterFamilies は追加計画の敵 B 前半（ID 90〜98、docs/agents/wave5-A-enemies.md）。
// 担当はこの表と、同じファイルの特殊攻撃・行動の関数だけを追加する（enemy_expansion.go がまとめて登録する）。
var expansionB1MonsterFamilies = []monsterFamily{
	{Base: MonsterDefinition{Traits: []EnemyTrait{EnemyTraitAquatic}, ID: 90, Type: "UmiDango", Name: "ウミダンゴ", Char: 'd', Health: 34, AttackPower: 6, DefensePower: 6, ExperiencePoints: 20, SpecialAttackProbability: 0.25},
		BaseImage: "img/umi_dango.png",
		Second:    monsterStage{91, "OoUmiDango", "オオウミダンゴ", 60, 11, 9, 50, "img/oo_umi_dango.png"},
		Third:     monsterStage{92, "DaiouUmiDango", "ダイオウウミダンゴ", 102, 22, 11, 125, "img/daiou_umi_dango.png"},
		Spawns:    []monsterSpawnRange{{90, 4, 10, 6}, {91, 11, 18, 6}, {92, 19, 30, 6}}},
	{Base: MonsterDefinition{Traits: []EnemyTrait{EnemyTraitAquatic}, ID: 93, Type: "KaeruAnkou", Name: "カエルアンコウ", Char: 'f', Health: 26, AttackPower: 5, DefensePower: 4, ExperiencePoints: 14},
		BaseImage: "img/kaeru_ankou.png",
		Second:    monsterStage{94, "OoKaeruAnkou", "オオカエルアンコウ", 48, 10, 7, 38, "img/oo_kaeru_ankou.png"},
		Third:     monsterStage{95, "NushiKaeruAnkou", "ヌシカエルアンコウ", 82, 20, 9, 95, "img/nushi_kaeru_ankou.png"},
		Spawns:    []monsterSpawnRange{{93, 3, 9, 6}, {94, 10, 17, 6}, {95, 18, 30, 6}}},
	{Base: MonsterDefinition{ID: 96, Type: "HajikiGani", Name: "ハジキガニ", Char: 'h', Health: 40, AttackPower: 9, DefensePower: 8, ExperiencePoints: 28, SpecialAttackProbability: 0.25, Traits: []EnemyTrait{EnemyTraitShell}},
		BaseImage: "img/hajiki_gani.png",
		Second:    monsterStage{97, "KengouGani", "ケンゴウガニ", 70, 15, 11, 70, "img/kengou_gani.png"},
		Third:     monsterStage{98, "KenseiGani", "ケンセイガニ", 119, 30, 13, 175, "img/kensei_gani.png"},
		Spawns:    []monsterSpawnRange{{96, 7, 13, 6}, {97, 14, 21, 6}, {98, 22, 30, 6}}},
}

var umiDangoRamDamage = map[int]int{90: 6, 91: 11, 92: 20}

func init() {
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.SpecialAttack = umiDangoRamAttack }, 90, 91, 92)
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.Behavior = kaeruAnkouBehavior }, 93, 94, 95)
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.SpecialAttack = hajikiGaniAttack }, 96, 97, 98)
	expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "新しい敵（3）", Lines: []string{
		"ウミダンゴ：体当たりで吹き飛ばし、持ち物を落とさせる。",
		"カエルアンコウ：3マス先から引き寄せる。途中の罠も踏む。",
		"ハジキガニ：盾を弾き飛ばす。弾きよけの指輪で防げる。",
		"上位種も同じ能力を持つ。",
	}})
}

func umiDangoRamAttack(e *Enemy, g *Game) {
	i := g.findEnemyForDefense(e.ID, e.X, e.Y)
	if i < 0 {
		return
	}
	dx, dy := sign(g.state.Player.X-e.X), sign(g.state.Player.Y-e.Y)
	source := enemyRangedSource(e.ID, e.Name, "体当たり")
	g.enqueueEnemyFixedAttack(i, umiDangoRamDamage[e.ID], EnemyDamageNormal, EnemyAttackMelee,
		fmt.Sprintf("%sが丸まって体当たりしてきた。", g.enemyDisplayName(e.Name)), "体当たり", func(g *Game) {
			g.knockbackPlayer(dx, dy, 2, 5, source)
			g.dropPlayerItemsForward(1+g.displacementIntn()(2), dx, dy)
		})
}

func kaeruAnkouBehavior(g *Game, i int) bool {
	e := g.state.Enemies[i]
	if !e.PlayerDiscovered {
		return false
	}
	dx, dy, distance, ok := g.straightLineToPlayer(i, 3)
	if !ok || distance < 2 || g.randIntn(2) != 0 {
		return false
	}
	g.Enqueue(Action{Duration: 0.5, Message: fmt.Sprintf("%sが疑似餌を伸ばした！", g.enemyDisplayName(e.Name)), Execute: func(g *Game) {
		g.pullPlayer(-dx, -dy, distance-1)
	}})
	return true
}

func hajikiGaniAttack(e *Enemy, g *Game) {
	i := g.findEnemyForDefense(e.ID, e.X, e.Y)
	if i < 0 {
		return
	}
	if g.state.Player.EquippedArmor == nil {
		g.enqueueEnemyNormalAttack(i)
		return
	}
	dx, dy := sign(g.state.Player.X-e.X), sign(g.state.Player.Y-e.Y)
	g.Enqueue(Action{Duration: 0.5, Message: fmt.Sprintf("%sが盾を鋏ではじいた！", g.enemyDisplayName(e.Name)), Execute: func(g *Game) {
		if armor := g.state.Player.EquippedArmor; armor != nil {
			g.knockOffPlayerEquipment(armor, dx, dy, 4)
		}
	}})
}

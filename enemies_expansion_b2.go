//go:build !test

package main

import "fmt"

// expansionB2MonsterFamilies は追加計画の敵 B 後半（ID 99〜107、docs/agents/wave5-B-enemies.md）。
// 担当はこの表と、同じファイルの特殊攻撃・行動・被弾時の関数だけを追加する（enemy_expansion.go がまとめて登録する）。
var expansionB2MonsterFamilies = []monsterFamily{
	{Base: MonsterDefinition{ID: 99, Type: "SabiHitode", Name: "サビヒトデ", Char: 's', Health: 30, AttackPower: 6, DefensePower: 5, ExperiencePoints: 18, SpecialAttackProbability: 0.3},
		BaseImage: "img/sabi_hitode.png", Second: monsterStage{100, "KusareHitode", "クサレヒトデ", 55, 11, 8, 45, "img/kusare_hitode.png"}, Third: monsterStage{101, "DoroHitode", "ドロヒトデ", 94, 22, 10, 112, "img/doro_hitode.png"}, Spawns: []monsterSpawnRange{{99, 5, 11, 6}, {100, 12, 19, 6}, {101, 20, 30, 6}}},
	{Base: MonsterDefinition{Traits: []EnemyTrait{EnemyTraitAquatic}, ID: 102, Type: "ItazuraKawahagi", Name: "イタズラカワハギ", Char: 'k', Health: 22, AttackPower: 4, DefensePower: 3, ExperiencePoints: 12, SpecialAttackProbability: 0.3},
		BaseImage: "img/itazura_kawahagi.png", Second: monsterStage{103, "WarusaKawahagi", "ワルサカワハギ", 40, 8, 6, 32, "img/warusa_kawahagi.png"}, Third: monsterStage{104, "AkutareKawahagi", "アクタレカワハギ", 68, 16, 8, 80, "img/akutare_kawahagi.png"}, Spawns: []monsterSpawnRange{{102, 3, 9, 6}, {103, 10, 17, 6}, {104, 18, 30, 6}}},
	{Base: MonsterDefinition{ID: 105, Type: "OohasamiGani", Name: "オオハサミガニ", Char: 'g', Health: 36, AttackPower: 10, DefensePower: 6, ExperiencePoints: 26, CriticalPercent: 15, Traits: []EnemyTrait{EnemyTraitShell}},
		BaseImage: "img/oohasami_gani.png", Second: monsterStage{106, "GuillotineGani", "ギロチンガニ", 64, 17, 9, 65, "img/guillotine_gani.png"}, Third: monsterStage{107, "DanzaiGani", "ダンザイガニ", 109, 34, 11, 162, "img/danzai_gani.png"}, Spawns: []monsterSpawnRange{{105, 6, 12, 6}, {106, 13, 20, 6}, {107, 21, 30, 6}}},
}

func init() {
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.SpecialAttack = sabiHitodeAttack; d.OnDamaged = sabiHitodeDamaged }, 99, 100, 101)
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.SpecialAttack = itazuraKawahagiAttack }, 102, 103, 104)
	expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "新しい敵（4）", Lines: []string{
		"サビヒトデ：武器か盾をさびさせる。攻撃すると分裂することがある。",
		"分身は経験値が無い。さび止め・金の装備・錆よけでさびを防げる。",
		"イタズラカワハギ：持ち物を遠くへ投げ捨てる。杖を狙う。",
		"オオハサミガニ：たまに痛恨の一撃を出す。甲殻斬りが有効。",
		"上位種も同じ能力を持つ。",
	}})
}

func sabiHitodeAttack(e *Enemy, g *Game) {
	i := g.findEnemyForDefense(e.ID, e.X, e.Y)
	if i < 0 {
		return
	}
	if g.state.Player.EquippedWeapon == nil && g.state.Player.EquippedArmor == nil {
		g.enqueueEnemyNormalAttack(i)
		return
	}
	g.Enqueue(Action{Duration: 0.5, Message: fmt.Sprintf("%sがさびた汁をかけてきた！", g.enemyDisplayName(e.Name)), Execute: func(g *Game) {
		p := &g.state.Player
		target := rollRustTarget(p.EquippedWeapon != nil, p.EquippedArmor != nil, g.randIntn)
		g.rustPlayerEquipment(target, 1)
	}})
}

// 分身の経験値は0にし、分裂を繰り返して経験値を稼ぐことを防ぐ。
func sabiHitodeDamaged(g *Game, i, _ int) {
	if g.randIntn(100) < 20 {
		if clone := g.splitEnemy(i); clone >= 0 {
			g.state.Enemies[clone].ExperiencePoints = 0
		}
	}
}

func itazuraKawahagiAttack(e *Enemy, g *Game) {
	x, y := e.X, e.Y
	g.Enqueue(Action{Duration: 0.5, Message: fmt.Sprintf("%sが持ち物を奪った！", g.enemyDisplayName(e.Name)), Execute: func(g *Game) {
		_, tossed := g.tossPlayerItem(x, y, func(item Item) bool { _, cane := item.(*Cane); return cane }, g.randIntn)
		if !tossed && !g.playerImmuneTo(StatusItemToss) {
			g.EnqueueMessage("しかし何も取られなかった。", 0.4)
		}
	}})
}

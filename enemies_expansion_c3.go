//go:build !test

package main

// expansionC3MonsterFamilies は追加計画の敵 C（ミミックダコ系 117〜119・ウチカエシエビ系 123〜125、docs/agents/wave6-C-enemies.md）。
// 担当はこの表と、同じファイルの特殊攻撃・行動・被弾時の関数だけを追加する（enemy_expansion.go がまとめて登録する）。
var expansionC3MonsterFamilies = []monsterFamily{
	{Base: MonsterDefinition{ID: 117, Type: "MimicDako", Name: "ミミックダコ", Char: 'm', Health: 34, AttackPower: 8, DefensePower: 6, ExperiencePoints: 30, Disguise: EnemyDisguiseEnemy},
		BaseImage: "img/mimic_dako.png", Second: monsterStage{118, "HengeDako", "ヘンゲダコ", 60, 14, 9, 75, "img/henge_dako.png"}, Third: monsterStage{119, "SenmenDako", "センメンダコ", 102, 28, 11, 188, "img/senmen_dako.png"}, Spawns: []monsterSpawnRange{{117, 6, 12, 5}, {118, 13, 20, 5}, {119, 21, 30, 5}}},
	{Base: MonsterDefinition{ID: 123, Type: "UchikaeshiEbi", Name: "ウチカエシエビ", Char: 'u', Health: 30, AttackPower: 7, DefensePower: 6, ExperiencePoints: 24, ProjectileReturnPercent: 40},
		BaseImage: "img/uchikaeshi_ebi.png", Second: monsterStage{124, "HajikikaeshiEbi", "ハジキカエシエビ", 54, 12, 9, 60, "img/hajikikaeshi_ebi.png"}, Third: monsterStage{125, "ZenbukaeshiEbi", "ゼンブカエシエビ", 92, 24, 11, 150, "img/zenbukaeshi_ebi.png"}, Spawns: []monsterSpawnRange{{123, 6, 12, 5}, {124, 13, 20, 5}, {125, 21, 30, 5}}},
}

func init() {
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.Behavior = hengeDakoBehavior }, 118, 119)
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.ProjectileReturnPercent = 60 }, 124)
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.ProjectileReturnPercent = 80 }, 125)
	expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "新しい敵（7）", Lines: []string{
		"ミミックダコ：その階に出る別の敵の姿で現れる。",
		"攻撃すると正体を現す。",
		"ヘンゲダコ以上：正体を現してもしばらくすると化け直す。",
		"ウチカエシエビ：投げた物・矢・魔法弾を打ち返す。",
		"上位種ほどよく打ち返す。",
		"上位種も同じ能力を持つ。",
	}})
}

// Timer は正体を現している間に行動した回数。化けている間は増えず、化け直すと0になる。
func hengeDakoBehavior(g *Game, i int) bool {
	e := &g.state.Enemies[i]
	if enemyMasquerading(*e) {
		return false
	}
	e.Timer++
	p := g.state.Player
	if e.Timer < 10 || (abs(e.X-p.X) <= 1 && abs(e.Y-p.Y) <= 1) {
		return false
	}
	id := pickDisguiseID(g.Floor, e.ID, g.randIntn)
	if id < 0 {
		return false
	}
	name := g.enemyDisplayName(e.Name)
	visible := isSameRoom(e.X, e.Y, p.X, p.Y, g.rooms)
	x, y, enemyID := e.X, e.Y, e.ID
	message := ""
	if visible {
		message = name + "はまた姿を変えた！"
	}
	g.Enqueue(Action{Duration: 0.4, Message: message, Execute: func(g *Game) {
		// 行動キューで先行する処理が敵を消しても、別の敵を書き換えない。
		index := g.findEnemyForDefense(enemyID, x, y)
		if index >= 0 {
			disguiseEnemyAs(&g.state.Enemies[index], id)
			g.state.Enemies[index].Timer = 0
		}
	}})
	return true
}

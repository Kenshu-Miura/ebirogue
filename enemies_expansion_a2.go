//go:build !test

package main

import "fmt"

// expansionA2MonsterFamilies は敵 A 後半の4系統。能力は init で設定し初期化循環を避ける。
var expansionA2MonsterFamilies = []monsterFamily{
	{Base: MonsterDefinition{ID: 78, Type: "IyashiNamako", Name: "イヤシナマコ", Char: 'L', Health: 30, AttackPower: 3, DefensePower: 4, ExperiencePoints: 20},
		BaseImage: "img/iyashi_namako.png",
		Second:    monsterStage{79, "JihiNamako", "ジヒナマコ", 54, 6, 7, 50, "img/jihi_namako.png"},
		Third:     monsterStage{80, "SeiboNamako", "セイボナマコ", 92, 12, 9, 125, "img/seibo_namako.png"},
		Spawns:    []monsterSpawnRange{{78, 4, 10, 5}, {79, 11, 18, 5}, {80, 19, 30, 5}}},
	{Base: MonsterDefinition{ID: 81, Type: "ZenigataAzarashi", Name: "ゼニガタアザラシ", Char: 'X', Health: 28, AttackPower: 6, DefensePower: 4, ExperiencePoints: 18, SpecialAttackProbability: 0.35},
		BaseImage: "img/zenigata_azarashi.png",
		Second:    monsterStage{82, "KinkaAzarashi", "キンカアザラシ", 50, 11, 7, 45, "img/kinka_azarashi.png"},
		Third:     monsterStage{83, "SenryouAzarashi", "センリョウアザラシ", 85, 22, 9, 112, "img/senryou_azarashi.png"},
		Spawns:    []monsterSpawnRange{{81, 3, 8, 6}, {82, 9, 16, 6}, {83, 17, 30, 6}}},
	{Base: MonsterDefinition{ID: 84, Type: "Sunamoguri", Name: "スナモグリ", Char: 'K', Health: 32, AttackPower: 7, DefensePower: 5, ExperiencePoints: 22, SpecialAttackProbability: 0.3, EvasionPercent: 15, Traits: []EnemyTrait{EnemyTraitShell}},
		BaseImage: "img/sunamoguri.png",
		Second:    monsterStage{85, "SunakakeMoguri", "スナカケモグリ", 58, 12, 8, 55, "img/sunakake_moguri.png"},
		Third:     monsterStage{86, "SunaarashiMoguri", "スナアラシモグリ", 98, 24, 10, 138, "img/sunaarashi_moguri.png"},
		Spawns:    []monsterSpawnRange{{84, 5, 11, 7}, {85, 12, 19, 7}, {86, 20, 30, 7}}},
	{Base: MonsterDefinition{ID: 87, Type: "YoroiEbi", Name: "ヨロイエビ", Char: 'V', Health: 40, AttackPower: 9, DefensePower: 8, ExperiencePoints: 30, Traits: []EnemyTrait{EnemyTraitShell}},
		BaseImage: "img/yoroi_ebi.png",
		Second:    monsterStage{88, "KabutoEbi", "カブトエビ", 70, 15, 11, 70, "img/kabuto_ebi.png"},
		Third:     monsterStage{89, "MushaEbi", "ムシャエビ", 119, 30, 13, 175, "img/musha_ebi.png"},
		Spawns:    []monsterSpawnRange{{87, 6, 12, 7}, {88, 13, 20, 7}, {89, 21, 30, 7}}},
}

func init() {
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.Behavior = healingNamakoBehavior }, 78, 79, 80)
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.SpecialAttack = sealCashAttack }, 81, 82, 83)
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.SpecialAttack = sandMoguriAttack }, 84, 85, 86)
	updateMonsterDefinitions(func(d *MonsterDefinition) { d.OnDefeated = armoredEbiDefeated }, 87, 88, 89)
	expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "新しい敵（2）", Lines: []string{
		"イヤシナマコ：攻撃せず、傷ついた敵を回復する。",
		"ゼニガタアザラシ：お金を盗んでワープする。守銭の盾で防げる。",
		"倒すと盗まれたお金を取り戻せる。",
		"スナモグリ：砂で目潰しをしてくる。攻撃をかわすことがある。",
		"ヨロイエビ：倒すと3ターン後にユウレイエビ系が現れる。",
		"上位種も同じ能力を持つ。",
	}})
}

// 攻撃をせず、傷ついた仲間へ接近して回復する。対象は実行時に引き直す。
func healingNamakoBehavior(g *Game, i int) bool {
	j := g.injuredAllyNear(i, 5)
	if j < 0 {
		moveRandomly(g, i)
		return true
	}
	self, ally := g.state.Enemies[i], g.state.Enemies[j]
	if max(abs(self.X-ally.X), abs(self.Y-ally.Y)) > 1 {
		g.stepEnemyToward(i, ally.X, ally.Y)
		return true
	}
	amount := []int{15, 30, 60}[self.ID-78]
	g.Enqueue(Action{Duration: 0.5, Message: fmt.Sprintf("%sが%sの傷を癒やした。", g.enemyDisplayName(self.Name), g.enemyDisplayName(ally.Name)), Execute: func(g *Game) {
		if current := g.findEnemyForDefense(ally.ID, ally.X, ally.Y); current >= 0 {
			g.healEnemy(current, amount)
		}
	}})
	return true
}

// 盗品は通常の HeldItem として保存し、撃破時の共通ドロップ処理へ渡す。
func sealCashAttack(enemy *Enemy, g *Game) {
	i := g.findEnemyForDefense(enemy.ID, enemy.X, enemy.Y)
	if i < 0 {
		return
	}
	if g.state.Player.Cash <= 0 || enemy.HeldItem != nil {
		g.enqueueEnemyNormalAttack(i)
		return
	}
	name := g.enemyDisplayName(enemy.Name)
	if g.playerBlocksTheft() {
		g.EnqueueMessage(fmt.Sprintf("%sは盗もうとしたが、%sが守った。", name, g.state.Player.EquippedArmor.Name), 0.5)
		return
	}
	amount := min(g.state.Player.Cash, []int{200, 500, 1000}[enemy.ID-81])
	id, x, y := enemy.ID, enemy.X, enemy.Y
	g.Enqueue(Action{Duration: 0.5, Message: fmt.Sprintf("%sはお金を狙っている。", name), Execute: func(g *Game) {
		current := g.findEnemyForDefense(id, x, y)
		if current < 0 || g.state.Enemies[current].HeldItem != nil {
			return
		}
		stolen := min(amount, g.state.Player.Cash)
		if stolen <= 0 {
			return
		}
		g.state.Player.Cash -= stolen
		g.state.Enemies[current].HeldItem = newStolenCash(stolen)
		g.state.Enemies[current].Fleeing = true
		warped := g.warpEnemyAwayFromPlayer(current)
		message := fmt.Sprintf("%sは %d を盗んだ！", name, stolen)
		if warped {
			message = fmt.Sprintf("%sは %d を盗んでワープした！", name, stolen)
		}
		g.EnqueueMessage(message, 0.5)
	}})
}

// テンプレート表を参照せず、初期化循環を避ける。
func newStolenCash(amount int) *Money {
	return &Money{BaseItem: BaseItem{Entity: Entity{Char: '!'}, ID: 0, Type: "Kane", Name: "小銭", Description: "小銭。それは海老さんが絆と呼ぶもの。", UseActions: map[string]UseAction{"UseMoney": money}}, Amount: amount, Identified: true}
}

// 部屋外の空いた床から選ぶ。候補が無くても盗み自体は成立する。
func (g *Game) warpEnemyAwayFromPlayer(i int) bool {
	var candidates []Coordinate
	for y, row := range g.state.Map {
		for x, tile := range row {
			if tile.Type == "floor" && g.canPlaceEnemyAt(x, y) && !g.IsInPlayerVisibleRoom(x, y) {
				candidates = append(candidates, Coordinate{X: x, Y: y})
			}
		}
	}
	if len(candidates) == 0 {
		return false
	}
	target := candidates[g.randIntn(len(candidates))]
	enemy := &g.state.Enemies[i]
	enemy.X, enemy.Y = target.X, target.Y
	enemy.Animating = false
	enemy.dx, enemy.dy = 0, 0
	g.miniMapDirty = true
	return true
}

func sandMoguriAttack(enemy *Enemy, g *Game) {
	i := g.findEnemyForDefense(enemy.ID, enemy.X, enemy.Y)
	if i < 0 {
		return
	}
	damage := []int{6, 10, 18}[enemy.ID-84]
	g.enqueueEnemyFixedAttack(i, damage, EnemyDamageNormal, EnemyAttackMelee, fmt.Sprintf("%sが砂をかけた。", g.enemyDisplayName(enemy.Name)), "砂かけ", func(g *Game) {
		if g.randIntn(100) >= 30 {
			return
		}
		if guard, blocked := g.playerStatusGuard(StatusBlind); blocked {
			g.EnqueueMessage(fmt.Sprintf("%sが目潰しを防いだ。", guard), 0.5)
			return
		}
		g.state.Player.StatusAilments.Blind = 5
		g.EnqueueMessage("海老さんは目が見えなくなった。", 0.5)
	})
}

func armoredEbiDefeated(g *Game, defeated Enemy) {
	spirit := []int{11, 29, 56}[defeated.ID-87]
	g.EnqueueMessage(fmt.Sprintf("%sの鎧から何かが抜け出した…", g.enemyDisplayName(defeated.Name)), 0.5)
	g.scheduleEnemySpawn(spirit, defeated.X, defeated.Y, 3)
}

//go:build !test

package main

import "fmt"

// フクレフグ系: Counter は自爆予約済みなら1、Flag は膨張中。Timer は使わない。
// シビレエイ系: Counter は非帯電中の被弾数、Timer は帯電の残りターン、Flag は帯電中。
var expansionC1MonsterFamilies = []monsterFamily{
	{Base: MonsterDefinition{ID: 108, Type: "FukureFugu", Name: "フクレフグ", Char: 'p', Health: 32, AttackPower: 7, DefensePower: 5, ExperiencePoints: 24, Traits: []EnemyTrait{EnemyTraitExplosive}},
		BaseImage: "img/fukure_fugu.png",
		Second:    monsterStage{109, "BakuhatsuFugu", "バクハツフグ", 56, 12, 8, 60, "img/bakuhatsu_fugu.png"},
		Third:     monsterStage{110, "DaibakuhatsuFugu", "ダイバクハツフグ", 95, 24, 10, 150, "img/daibakuhatsu_fugu.png"},
		Spawns:    []monsterSpawnRange{{108, 6, 12, 6}, {109, 13, 20, 6}, {110, 21, 30, 6}}},
	{Base: MonsterDefinition{ID: 114, Type: "ShibireEi", Name: "シビレエイ", Char: 'e', Health: 38, AttackPower: 8, DefensePower: 7, ExperiencePoints: 30},
		BaseImage: "img/shibire_ei.png",
		Second:    monsterStage{115, "DenkiEi", "デンキエイ", 66, 14, 10, 75, "img/denki_ei.png"},
		Third:     monsterStage{116, "RaiteiEi", "ライテイエイ", 112, 28, 12, 188, "img/raitei_ei.png"},
		Spawns:    []monsterSpawnRange{{114, 8, 14, 5}, {115, 15, 22, 5}, {116, 23, 30, 5}}},
}

var fukureFuguExplosionDamage = map[int]int{108: 14, 109: 26, 110: 48}
var shibireEiLightningDamage = map[int]int{114: 8, 115: 15, 116: 28}
var shibireEiDischargeDamage = map[int]int{114: 5, 115: 9, 116: 16}

func init() {
	updateMonsterDefinitions(func(d *MonsterDefinition) {
		d.OnDamaged = fukureFuguDamaged
		d.Behavior = fukureFuguBehavior
	}, 108, 109, 110)
	updateMonsterDefinitions(func(d *MonsterDefinition) {
		d.OnDamaged = shibireEiDamaged
		d.Behavior = shibireEiBehavior
	}, 114, 115, 116)
	expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "新しい敵（5）", Lines: []string{
		"フクレフグ：弱ると膨らんで止まり、さらに弱ると",
		"周りを巻き込んで破裂する。",
		"シビレエイ：3回攻撃されると帯電し、同じ部屋に雷を落とす。",
		"帯電中に殴ると放電で反撃する。",
		"上位種も同じ能力を持つ。",
	}})
}

func fukureFuguDamaged(g *Game, i, damage int) {
	e := &g.state.Enemies[i]
	if e.Health <= e.MaxHealth/4 {
		if e.Counter == 0 {
			e.Counter = 1
			g.enqueueEnemyExplosion(EnemyExplosion{BomberIndex: i, CenterX: e.X, CenterY: e.Y, Radius: 1, Damage: fukureFuguExplosionDamage[e.ID], SelfDestruct: true, Message: fmt.Sprintf("%sが破裂した！", g.enemyDisplayName(e.Name))})
		}
		return
	}
	if e.Health <= e.MaxHealth/2 && !e.Flag {
		e.Flag = true
		g.EnqueueMessage(fmt.Sprintf("%sは膨らんだ！", g.enemyDisplayName(e.Name)), 0.5)
	}
}

func fukureFuguBehavior(g *Game, i int) bool { return g.state.Enemies[i].Flag }

func shibireEiDamaged(g *Game, i, damage int) {
	e := &g.state.Enemies[i]
	if !e.Flag {
		e.Counter++
		if e.Counter == 3 {
			e.Flag, e.Timer, e.Counter = true, 10, 0
			g.EnqueueMessage(fmt.Sprintf("%sは帯電した！", g.enemyDisplayName(e.Name)), 0.5)
		}
		return
	}
	if max(abs(e.X-g.state.Player.X), abs(e.Y-g.state.Player.Y)) == 1 {
		g.enqueueEnemyFixedAttack(i, shibireEiDischargeDamage[e.ID], EnemyDamageMagic, EnemyAttackEnvironment, fmt.Sprintf("%sが放電した！", g.enemyDisplayName(e.Name)), "放電", nil)
	}
}

func shibireEiBehavior(g *Game, i int) bool {
	e := &g.state.Enemies[i]
	if !e.Flag {
		return false
	}
	e.Timer--
	if e.Timer <= 0 {
		e.Flag = false
		g.EnqueueMessage(fmt.Sprintf("%sの帯電が消えた", g.enemyDisplayName(e.Name)), 0.5)
		return false
	}
	if e.PlayerDiscovered && isSameRoom(e.X, e.Y, g.state.Player.X, g.state.Player.Y, g.rooms) && g.randIntn(3) == 0 {
		g.enqueueEnemyFixedAttack(i, shibireEiLightningDamage[e.ID], EnemyDamageMagic, EnemyAttackRanged, fmt.Sprintf("%sが雷を落とした！", g.enemyDisplayName(e.Name)), "雷", nil)
		return true
	}
	return false
}

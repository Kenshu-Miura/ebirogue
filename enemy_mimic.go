//go:build !test

package main

import "fmt"

// 別の敵の姿に化ける敵（EnemyDisguiseEnemy、段4のミミックダコ系）の共通処理。
//
// 化けた姿は enemyAppearance が1か所で決める。syncEnemyAppearance がその結果を Enemy の
// Name・Type・Char へ写すので、画像（getEnemyImage は Type で選ぶ）・メッセージ（Name）・
// ミニマップ（敵の印）など、Name と Type を読む既存の処理はすべて化けた姿を表示する。
// 正体の名前は MonsterDefinitions[e.ID].Name で引く。能力値・特殊能力・属性は正体のまま。
// 姿は出現時の準備（prepareSpawnedEnemy）で、その階の出現テーブルから選ぶ。
// 攻撃を受ける（通常攻撃・投擲・射撃・杖・ダメージを受ける処理すべて）と正体を現す（revealEnemy）。
// 封印されている間も正体の姿になる。

// enemyMasquerading は敵が別の敵の姿に化けているかを返す。
func enemyMasquerading(enemy Enemy) bool {
	if enemy.Disguise != EnemyDisguiseEnemy || enemy.Revealed || enemy.StatusAilments.Seal {
		return false
	}
	_, ok := MonsterDefinitions[enemy.DisguiseID]
	return ok
}

// enemyLook は敵の見た目（化けていれば化けた姿）。
type enemyLook struct {
	ID         int
	Name, Type string
	Char       rune
}

// enemyAppearance は敵の見た目を返す。化けた姿を決める唯一の関数。
func enemyAppearance(enemy Enemy) enemyLook {
	id := enemy.ID
	if enemyMasquerading(enemy) {
		id = enemy.DisguiseID
	}
	definition, ok := MonsterDefinitions[id]
	if !ok {
		return enemyLook{ID: enemy.ID, Name: enemy.Name, Type: enemy.Type, Char: enemy.Char}
	}
	return enemyLook{ID: id, Name: definition.Name, Type: definition.Type, Char: definition.Char}
}

// syncEnemyAppearance は enemyAppearance の結果を Name・Type・Char へ写す。
// 化ける・正体を現す・上位種になる・封印される・セーブから戻すたびに呼ぶ。
func syncEnemyAppearance(enemy *Enemy) {
	if enemy.Disguise != EnemyDisguiseEnemy {
		return
	}
	look := enemyAppearance(*enemy)
	enemy.Name, enemy.Type, enemy.Char = look.Name, look.Type, look.Char
}

// disguiseEnemyAs は敵を monsterID の敵の姿に化けさせる。
func disguiseEnemyAs(enemy *Enemy, monsterID int) {
	enemy.DisguiseID = monsterID
	enemy.Revealed = false
	syncEnemyAppearance(enemy)
}

// pickDisguiseID は floor 階の出現テーブルから、化ける姿の敵 ID を重みに従って選ぶ。
// ボス・化ける敵（化け貝系・ミミックダコ系など Disguise を持つ敵）・selfID は選ばない。候補が無ければ -1。
func pickDisguiseID(floor, selfID int, intn func(int) int) int {
	var candidates []MonsterSpawnEntry
	total := 0
	for _, entry := range FloorSpawnTables[floor] {
		definition, ok := MonsterDefinitions[entry.MonsterID]
		if !ok || entry.MonsterID == selfID || definition.IsBoss || definition.Disguise != EnemyDisguiseNone || entry.Weight <= 0 {
			continue
		}
		candidates = append(candidates, entry)
		total += entry.Weight
	}
	if total == 0 {
		return -1
	}
	roll := intn(total)
	for _, entry := range candidates {
		if roll < entry.Weight {
			return entry.MonsterID
		}
		roll -= entry.Weight
	}
	return -1
}

// prepareSpawnedEnemy は floor 階に新しく置く敵の準備をする（フロアの生成・湧き・仲間呼び・遅れて現れる敵・
// モンスターハウスのカードから呼ぶ）。今は、別の敵に化ける敵の姿をその階の出現テーブルから選ぶ。
// intn は呼び出し元の乱数（フロアの生成ならフロア専用の系列）。化ける敵でなければ乱数を使わない。
func prepareSpawnedEnemy(enemy *Enemy, floor int, intn func(int) int) {
	if MonsterDefinitions[enemy.ID].Disguise != EnemyDisguiseEnemy {
		return
	}
	if id := pickDisguiseID(floor, enemy.ID, intn); id >= 0 {
		disguiseEnemyAs(enemy, id)
	}
}

// revealMessage は正体を現したときのメッセージ。
func revealMessage(enemy Enemy) string {
	trueName := MonsterDefinitions[enemy.ID].Name
	switch enemy.Disguise {
	case EnemyDisguiseStairs:
		return fmt.Sprintf("階段の正体は%sだった", trueName)
	case EnemyDisguiseEnemy:
		return fmt.Sprintf("%sの正体は%sだった！", enemyAppearance(enemy).Name, trueName)
	default:
		return fmt.Sprintf("道具の正体は%sだった", trueName)
	}
}

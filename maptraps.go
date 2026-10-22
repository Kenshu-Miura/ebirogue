//go:build !test

package main

import (
	"fmt"
)

// マップ上に配置される罠の構造体
type MapTrap struct {
	X, Y       int
	TemplateID int // mapTrapTemplates の ID（セーブ・復元に使う）
	Name       string
	Discovered bool          // プレイヤーが発見済みかどうか
	Effect     func(g *Game) // 罠の効果を実行する関数
	FailRate   int           // 不発率（0-100のパーセンテージ）
}

// 罠の効果関数
var sleepGasTrapEffect = func(g *Game) {
	guard, resisted := g.playerStatusGuard(StatusSleep)
	g.enqueuePlayerStatus(StatusSleep, 10, StackDefault, guard, resisted) // 10ターン睡眠
}

var poisonArrowTrapEffect = func(g *Game) {
	source := g.steppedTrapSource
	guard, resisted := g.playerStatusGuard(StatusPoison)
	defense := g.resolvePlayerShieldDefense(5, EnemyDamageNormal, EnemyAttackEnvironment)
	message := "毒矢が刺さり、5ダメージを受けた"
	if resisted {
		message = fmt.Sprintf("毒矢で5ダメージを受けたが、%sが毒を防いだ", guard)
	}
	g.Enqueue(Action{
		Duration: 0.5,
		Message:  message,
		Execute: func(g *Game) {
			g.applyPlayerShieldDefense(defense, source, -1, -1, -1)
			if !resisted && g.state.Player.Health > 0 {
				g.drainPlayerPower(poisonPowerDrain)
			}
		},
	})
}

var slowTrapEffect = func(g *Game) {
	guard, resisted := g.playerStatusGuard(StatusSlow)
	g.enqueuePlayerStatus(StatusSlow, 10, StackDefault, guard, resisted)
}

var mineTrapEffect = func(g *Game) {
	source := g.steppedTrapSource
	defense := g.resolvePlayerShieldDefense(mineTrapDamage(g.state.Player.Health), EnemyDamageExplosion, EnemyAttackEnvironment)
	message := fmt.Sprintf("地雷が爆発し、%dダメージを受けた", defense.Damage)
	if defense.Resisted {
		message = fmt.Sprintf("地雷が爆発したが、%sが威力を抑えて%dダメージを受けた", g.state.Player.EquippedArmor.Name, defense.Damage)
	}
	g.Enqueue(Action{
		Duration: 0.5,
		Message:  message,
		Execute: func(g *Game) {
			g.applyPlayerShieldDefense(defense, source, -1, -1, -1)
			g.removeMineAt(g.state.Player.X, g.state.Player.Y)
			g.explodeTraps(g.state.Player.X, g.state.Player.Y, 1, true, 0)
		},
	})
}

// サビの罠が武器と盾のどちらを錆びさせるかを表す
type rustTarget int

const (
	rustNone rustTarget = iota
	rustWeapon
	rustArmor
)

// rollRustTarget は装備状況からサビの罠の対象を決める純粋関数
func rollRustTarget(hasWeapon, hasArmor bool, intn func(int) int) rustTarget {
	switch {
	case hasWeapon && hasArmor:
		if intn(2) == 0 {
			return rustWeapon
		}
		return rustArmor
	case hasWeapon:
		return rustWeapon
	case hasArmor:
		return rustArmor
	default:
		return rustNone
	}
}

var rustTrapEffect = func(g *Game) {
	g.Enqueue(Action{
		Duration: 0.5,
		Message:  "赤茶けた液体が噴き出した",
		Execute: func(g *Game) {
			player := &g.state.Player
			g.rustPlayerEquipment(rollRustTarget(player.EquippedWeapon != nil, player.EquippedArmor != nil, g.randIntn), 1)
		},
	})
}

type mapTrapTemplate struct {
	ID       int
	Name     string
	Effect   func(g *Game)
	FailRate int
}

// mapTrapTemplates は罠テンプレートを明示 ID で引くテーブル。
// ID はセーブデータに保存されるため、既存の ID を変更・再利用しないこと。
var mapTrapTemplates = map[int]mapTrapTemplate{
	0:  {ID: 0, Name: "睡眠ガスの罠", Effect: sleepGasTrapEffect, FailRate: 30},
	1:  {ID: 1, Name: "毒矢の罠", Effect: poisonArrowTrapEffect, FailRate: 20},
	2:  {ID: 2, Name: "鈍足の罠", Effect: slowTrapEffect, FailRate: 20},
	3:  {ID: 3, Name: "地雷", Effect: mineTrapEffect, FailRate: 10},
	4:  {ID: 4, Name: "サビの罠", Effect: rustTrapEffect, FailRate: 20},
	5:  {ID: 5, Name: "ワープの罠", Effect: warpTrapEffect, FailRate: 20},
	6:  {ID: 6, Name: "落とし穴", Effect: pitfallTrapEffect, FailRate: 20},
	7:  {ID: 7, Name: "トラバサミ", Effect: bearTrapEffect, FailRate: 20},
	8:  {ID: 8, Name: "木の矢の罠", Effect: woodArrowTrapEffect, FailRate: 20},
	9:  {ID: 9, Name: "鉄の矢の罠", Effect: ironArrowTrapEffect, FailRate: 20},
	10: {ID: 10, Name: "落石の罠", Effect: fallingRockTrapEffect, FailRate: 20},
	11: {ID: 11, Name: "大型地雷", Effect: largeMineTrapEffect, FailRate: 10},
	12: {ID: 12, Name: "装備はずしの罠", Effect: unequipTrapEffect, FailRate: itemTrapFailRate},
	13: {ID: 13, Name: "腐敗の罠", FailRate: itemTrapFailRate}, // 効果はアイテムテーブルとの初期化循環を避けて init で登録。
	14: {ID: 14, Name: "散乱の罠", Effect: scatterTrapEffect, FailRate: itemTrapFailRate},
	15: {ID: 15, Name: "回復封じの罠", Effect: noHealingTrapEffect, FailRate: 20},
}

// randomMapTrapIDs は敵の罠設置などで一様に抽選する罠 ID の一覧（昇順）。
var randomMapTrapIDs = sortedTemplateIDs(mapTrapTemplates, nil)

// pickFloorTrapID は床生成・ワナのカード・底抜けの壺で置く罠を、floor 階の重み（trapBandWeights、item_floor_tables.go）で選ぶ。
// ボス階と30階では落とし穴を除く（availableTrapIDs）。乱数は1回使う。
func pickFloorTrapID(floor int, intn func(int) int) int {
	return pickWeightedTrapID(availableTrapIDs(randomMapTrapIDs, floor), floor, intn, 0)
}

func createMapTrapByID(id, x, y int) MapTrap {
	template, ok := mapTrapTemplates[id]
	if !ok {
		template = mapTrapTemplates[0]
	}
	return MapTrap{X: x, Y: y, TemplateID: template.ID, Name: template.Name, Effect: template.Effect, FailRate: template.FailRate}
}

// 罠を踏んだ時の処理
func (g *Game) stepOnTrap(trapIndex int) {
	trap := g.state.MapTraps[trapIndex]
	g.recordCodex(2, trap.TemplateID, func(r *CodexRecord) { r.Seen = true; r.Stepped = true })

	// デバッグログを追加
	g.debugf("プレイヤーが罠を踏みました: %s 座標(%d, %d)", trap.Name, trap.X, trap.Y)

	// まず罠を踏んだメッセージを表示
	action := Action{
		Duration: 0.5,
		Message:  trap.Name + "を踏んだ",
		Execute:  func(g *Game) {},
	}
	g.Enqueue(action)

	if g.randIntn(100) < trap.FailRate {
		g.Enqueue(Action{
			Duration: 0.5,
			Message:  "しかし" + trap.Name + "は作動しなかった",
			Execute:  func(g *Game) {},
		})
	} else {
		// 罠の効果関数は死因の記録にこの罠を使う
		g.steppedTrapSource = trapSource(trap.TemplateID, trap.Name)
		start := len(g.ActionQueue.Queue)
		trap.Effect(g)
		g.soundFirstAction(start, "trap")
	}

	// 罠を発見済みにする
	g.state.MapTraps[trapIndex].Discovered = true
}

// 前方に罠があるかチェックし、あれば発見する
func (g *Game) checkTrapInFront() bool {
	playerX, playerY := g.state.Player.GetPosition()
	var frontX, frontY int

	switch g.state.Player.Direction {
	case Up:
		frontX, frontY = playerX, playerY-1
	case Down:
		frontX, frontY = playerX, playerY+1
	case Left:
		frontX, frontY = playerX-1, playerY
	case Right:
		frontX, frontY = playerX+1, playerY
	case UpRight:
		frontX, frontY = playerX+1, playerY-1
	case DownRight:
		frontX, frontY = playerX+1, playerY+1
	case UpLeft:
		frontX, frontY = playerX-1, playerY-1
	case DownLeft:
		frontX, frontY = playerX-1, playerY+1
	}

	// 前方の位置に罠があるかチェック
	for i := range g.state.MapTraps {
		if g.state.MapTraps[i].X == frontX && g.state.MapTraps[i].Y == frontY && !g.state.MapTraps[i].Discovered {
			g.state.MapTraps[i].Discovered = true
			g.debugf("前方の罠を発見しました: %s 座標(%d, %d)", g.state.MapTraps[i].Name, frontX, frontY)
			g.Enqueue(Action{
				Duration: 0.4,
				Message:  g.state.MapTraps[i].Name + "を見つけた",
				Execute:  func(g *Game) {},
			})
			return true
		}
	}
	return false
}

// 罠を生成する関数（難易度の補正なし）
func generateMapTraps(rooms []Room, rng *gameRNG, floor int) []MapTrap {
	return generateMapTrapsWith(rooms, rng, floor, neutralSpawnTuning)
}

// generateMapTrapsWith は難易度の補正 tuning で罠の数を変えて生成する。
func generateMapTrapsWith(rooms []Room, rng *gameRNG, floor int, tuning difficultySpawnTuning) []MapTrap {
	var traps []MapTrap

	// 5-10個の罠をランダムに生成（難易度の補正で増減）
	numTraps := tuning.floorTrapCount(rng.Intn(floorTrapMaxCount-floorTrapMinCount+1) + floorTrapMinCount)

	for i := 0; i < numTraps; i++ {
		// ランダムな部屋を選択
		roomIndex := rng.Intn(len(rooms))
		room := rooms[roomIndex]

		// 部屋内のランダムな位置を選択（部屋の境界を除く）
		x := rng.Intn(room.Width-2) + room.X + 1
		y := rng.Intn(room.Height-2) + room.Y + 1

		// 既存の罠と重複しないかチェック
		duplicate := false
		for _, existingTrap := range traps {
			if existingTrap.X == x && existingTrap.Y == y {
				duplicate = true
				break
			}
		}

		if !duplicate {
			trap := createMapTrapByID(pickFloorTrapID(floor, rng.Intn), x, y)
			traps = append(traps, trap)
		} else {
			// 重複していた場合は再試行
			i--
		}
	}

	return traps
}

// プレイヤーが指定位置に移動した際に罠があるかチェックし、あれば踏む
func (g *Game) checkForTrapAtPosition(x, y int) {
	for i, trap := range g.state.MapTraps {
		if trap.X == x && trap.Y == y {
			if g.playerHasAbility(AbilityLevitation) {
				if !trap.Discovered {
					g.state.MapTraps[i].Discovered = true
					g.miniMapDirty = true
					g.EnqueueMessage(fmt.Sprintf("%sの上を浮いて通り過ぎた。", trap.Name), 0.5)
				}
				return
			}
			// 罠を踏んだ処理を実行
			g.stepOnTrap(i)
			return
		}
	}
}

//go:build !test

package main

import (
	"fmt"
	"log"
	"math/rand"
)

// マップ上に配置される罠の構造体
type MapTrap struct {
	X, Y       int
	Name       string
	Discovered bool          // プレイヤーが発見済みかどうか
	Effect     func(g *Game) // 罠の効果を実行する関数
	FailRate   int           // 不発率（0-100のパーセンテージ）
}

// 罠の効果関数
var sleepGasTrapEffect = func(g *Game) {
	g.Enqueue(Action{
		Duration: 0.5,
		Message:  "海老さんは眠った",
		Execute: func(g *Game) {
			g.state.Player.StatusAilments.Sleep = 10 // 10ターン睡眠
		},
	})
}

var poisonArrowTrapEffect = func(g *Game) {
	g.Enqueue(Action{
		Duration: 0.5,
		Message:  "毒矢が刺さり、毒状態になった",
		Execute: func(g *Game) {
			g.state.Player.Health = max(0, g.state.Player.Health-5)
			g.state.Player.StatusAilments.Poison = 8
			g.state.Player.checkDeath(g)
		},
	})
}

var slowTrapEffect = func(g *Game) {
	g.Enqueue(Action{
		Duration: 0.5,
		Message:  "体が重くなり、鈍足状態になった",
		Execute: func(g *Game) {
			g.state.Player.StatusAilments.Slow = 10
		},
	})
}

var mineTrapEffect = func(g *Game) {
	g.Enqueue(Action{
		Duration: 0.5,
		Message:  "地雷が爆発した",
		Execute: func(g *Game) {
			g.state.Player.Health -= mineTrapDamage(g.state.Player.Health)
			g.state.Player.checkDeath(g)
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
			switch rollRustTarget(player.EquippedWeapon != nil, player.EquippedArmor != nil, rand.Intn) {
			case rustWeapon:
				weapon := player.EquippedWeapon
				if weapon.RustProof {
					g.EnqueueMessage(fmt.Sprintf("しかし%sは錆びなかった", weapon.GetName()), 0.4)
					return
				}
				weapon.Sharpness--
				// 装備中の武器の強化値低下を攻撃力へ即時反映する
				player.AttackPower--
				g.EnqueueMessage(fmt.Sprintf("%sが錆びてしまった", weapon.GetName()), 0.4)
			case rustArmor:
				armor := player.EquippedArmor
				if armor.RustProof {
					g.EnqueueMessage(fmt.Sprintf("しかし%sは錆びなかった", armor.GetName()), 0.4)
					return
				}
				armor.Sharpness--
				// 装備中の盾の強化値低下を防御力へ即時反映する
				player.DefensePower--
				g.EnqueueMessage(fmt.Sprintf("%sが錆びてしまった", armor.GetName()), 0.4)
			default:
				g.EnqueueMessage("しかし何も起こらなかった", 0.4)
			}
		},
	})
}

type mapTrapTemplate struct {
	Name     string
	Effect   func(g *Game)
	FailRate int
}

var mapTrapTemplates = []mapTrapTemplate{
	{Name: "睡眠ガスの罠", Effect: sleepGasTrapEffect, FailRate: 30},
	{Name: "毒矢の罠", Effect: poisonArrowTrapEffect, FailRate: 20},
	{Name: "鈍足の罠", Effect: slowTrapEffect, FailRate: 20},
	{Name: "地雷", Effect: mineTrapEffect, FailRate: 10},
	{Name: "サビの罠", Effect: rustTrapEffect, FailRate: 20},
}

func createMapTrapByID(id, x, y int) MapTrap {
	if id < 0 || id >= len(mapTrapTemplates) {
		id = 0
	}
	template := mapTrapTemplates[id]
	return MapTrap{X: x, Y: y, Name: template.Name, Effect: template.Effect, FailRate: template.FailRate}
}

// 罠を踏んだ時の処理
func (g *Game) stepOnTrap(trapIndex int) {
	trap := g.state.MapTraps[trapIndex]

	// デバッグログを追加
	log.Printf("プレイヤーが罠を踏みました: %s 座標(%d, %d)", trap.Name, trap.X, trap.Y)

	// まず罠を踏んだメッセージを表示
	action := Action{
		Duration: 0.5,
		Message:  trap.Name + "を踏んだ",
		Execute:  func(g *Game) {},
	}
	g.Enqueue(action)

	if rand.Intn(100) < trap.FailRate {
		g.Enqueue(Action{
			Duration: 0.5,
			Message:  "しかし" + trap.Name + "は作動しなかった",
			Execute:  func(g *Game) {},
		})
	} else {
		trap.Effect(g)
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
			log.Printf("前方の罠を発見しました: %s 座標(%d, %d)", g.state.MapTraps[i].Name, frontX, frontY)
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

// 罠を生成する関数
func generateMapTraps(rooms []Room) []MapTrap {
	var traps []MapTrap

	// 5-10個の罠をランダムに生成
	numTraps := rand.Intn(6) + 5 // 5～10個

	for i := 0; i < numTraps; i++ {
		// ランダムな部屋を選択
		roomIndex := rand.Intn(len(rooms))
		room := rooms[roomIndex]

		// 部屋内のランダムな位置を選択（部屋の境界を除く）
		x := rand.Intn(room.Width-2) + room.X + 1
		y := rand.Intn(room.Height-2) + room.Y + 1

		// 既存の罠と重複しないかチェック
		duplicate := false
		for _, existingTrap := range traps {
			if existingTrap.X == x && existingTrap.Y == y {
				duplicate = true
				break
			}
		}

		// 睡眠ガスを多めにしつつ、ほかの罠も混ぜる
		if !duplicate {
			trapIDs := []int{0, 0, 1, 2, 3, 4}
			trap := createMapTrapByID(trapIDs[rand.Intn(len(trapIDs))], x, y)
			traps = append(traps, trap)
			log.Printf("罠を生成しました: %s 座標(%d, %d)", trap.Name, x, y)
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
			// 罠を踏んだ処理を実行
			g.stepOnTrap(i)
			return
		}
	}
}

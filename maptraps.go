//go:build !test

package main

import (
	"log"
	"math/rand"
)

// マップ上に配置される罠の構造体
type MapTrap struct {
	X, Y       int
	Name       string
	Discovered bool // プレイヤーが発見済みかどうか
	Effect     func(g *Game) // 罠の効果を実行する関数
	FailRate   int    // 不発率（0-100のパーセンテージ）
}

// 罠の効果関数

// 睡眠ガスの罠の効果
var sleepGasTrapEffect = func(g *Game) {
	// 不発判定
	if rand.Intn(100) < 30 { // 30%の確率で不発
		action := Action{
			Duration: 0.5,
			Message:  "しかし睡眠ガスの罠は作動しなかった",
			Execute:  func(g *Game) {},
		}
		g.Enqueue(action)
		return
	}
	
	// 睡眠効果を適用
	action := Action{
		Duration: 0.5,
		Message:  "海老さんは眠った",
		Execute: func(g *Game) {
			g.state.Player.StatusAilments.Sleep = 10 // 10ターン睡眠
		},
	}
	g.Enqueue(action)
}

// 睡眠ガスの罠を作成する関数
func createSleepGasTrap(x, y int) MapTrap {
	return MapTrap{
		X:          x,
		Y:          y,
		Name:       "睡眠ガスの罠",
		Discovered: false,
		Effect:     sleepGasTrapEffect,
		FailRate:   30,
	}
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
	
	// 罠の効果を実行
	trap.Effect(g)
	
	// 罠を発見済みにする
	g.state.MapTraps[trapIndex].Discovered = true
}

// 前方に罠があるかチェックし、あれば発見する
func (g *Game) checkTrapInFront() {
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
		}
	}
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
		
		// 重複していなければ睡眠ガスの罠を作成
		if !duplicate {
			trap := createSleepGasTrap(x, y)
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
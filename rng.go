package main

import (
	"math/rand/v2"
	"time"
)

// 乱数系列の種類。同じシードでも系列ごとに独立した乱数列になる。
//   - ターン進行（戦闘・敵 AI・アイテム効果など）は Game.rng の1系列。
//   - フロア生成は「シード＋階層」から毎回作り直すため、プレイ内容が違っても同じ階は同じ地形・配置になる。
const (
	rngStreamTurn      uint64 = 0x7475726e // "turn"
	rngStreamFloorBase uint64 = 0x666c6f6f72000000
)

// gameRNG はセーブ可能な乱数源。PCG の内部状態をそのまま保存・復元できる。
type gameRNG struct {
	src *rand.PCG
	r   *rand.Rand
}

func newGameRNG(seed int64, stream uint64) *gameRNG {
	src := rand.NewPCG(uint64(seed), stream)
	return &gameRNG{src: src, r: rand.New(src)}
}

// newTurnRNG はターン進行用の乱数源を作る。
func newTurnRNG(seed int64) *gameRNG {
	return newGameRNG(seed, rngStreamTurn)
}

// newFloorRNG は指定階層の生成専用の乱数源を作る。
func newFloorRNG(seed int64, floor int) *gameRNG {
	return newGameRNG(seed, rngStreamFloorBase+uint64(floor))
}

// Intn は [0, n) の乱数を返す（n <= 0 なら panic）。math/rand の Intn と同じ契約。
func (r *gameRNG) Intn(n int) int {
	return r.r.IntN(n)
}

// Float64 は [0, 1) の乱数を返す。
func (r *gameRNG) Float64() float64 {
	return r.r.Float64()
}

// MarshalState は乱数源の内部状態をバイト列にする（セーブ用）。
func (r *gameRNG) MarshalState() ([]byte, error) {
	return r.src.MarshalBinary()
}

// UnmarshalState は MarshalState の結果から内部状態を復元する。
func (r *gameRNG) UnmarshalState(state []byte) error {
	return r.src.UnmarshalBinary(state)
}

// newRandomSeed は新しい冒険用のシードを作る。
func newRandomSeed() int64 {
	return time.Now().UnixNano() & 0x7fffffffffff // 表示しやすいよう 47bit に収める
}

// random は Game のターン進行用乱数源を返す。
// テストなどで rng を設定せずに作った Game は seed から遅延生成する。
func (g *Game) random() *gameRNG {
	if g.rng == nil {
		g.rng = newTurnRNG(g.seed)
	}
	return g.rng
}

// randIntn は g.random().Intn の短縮形。
func (g *Game) randIntn(n int) int {
	return g.random().Intn(n)
}

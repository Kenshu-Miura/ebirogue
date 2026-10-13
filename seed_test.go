//go:build !test

package main

import (
	"encoding/json"
	"testing"
)

// floorSnapshot はフロアの状態を比較用の JSON にする（関数フィールドを含まないセーブ形式を使う）。
func floorSnapshot(t *testing.T, g *Game) string {
	t.Helper()
	save := g.buildSaveData()
	save.Messages = nil
	data, err := json.Marshal(save)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSameSeedGeneratesIdenticalFloors(t *testing.T) {
	for _, seed := range []int64{1, 42, 987654321} {
		g1, g2 := &Game{}, &Game{}
		g1.startNewRun(seed)
		g2.startNewRun(seed)
		if floorSnapshot(t, g1) != floorSnapshot(t, g2) {
			t.Fatalf("seed %d: first floors differ", seed)
		}
		if len(g1.state.Enemies) == 0 || len(g1.state.Items) == 0 || len(g1.state.MapTraps) == 0 {
			t.Fatalf("seed %d: floor should contain enemies, items and traps", seed)
		}
	}

	g1, g2 := &Game{}, &Game{}
	g1.startNewRun(1)
	g2.startNewRun(2)
	if floorSnapshot(t, g1) == floorSnapshot(t, g2) {
		t.Fatal("different seeds should generate different floors")
	}
}

func TestFloorGenerationDoesNotDependOnTurnRNG(t *testing.T) {
	const seed = 2024
	g1, g2 := &Game{}, &Game{}
	g1.startNewRun(seed)
	g2.startNewRun(seed)
	// 片方だけターン進行の乱数を消費しても、次の階の生成結果は変わらない
	for i := 0; i < 37; i++ {
		g1.randIntn(100)
	}
	for _, g := range []*Game{g1, g2} {
		player := newInitialPlayer()
		mapGrid, enemies, items, floor, rooms, traps := GenerateRandomMap(70, 70, 1, &player, newFloorRNG(g.seed, 2))
		g.state = GameState{Map: mapGrid, Player: player, Enemies: enemies, Items: items, MapTraps: traps}
		g.rooms = rooms
		g.Floor = floor
		g.rng = newTurnRNG(seed) // RNGState の差は比較対象から外す
	}
	if floorSnapshot(t, g1) != floorSnapshot(t, g2) {
		t.Fatal("floor 2 changed depending on turn RNG consumption")
	}
}

func TestSaveRestoresSeedAndRNGState(t *testing.T) {
	g := &Game{}
	g.startNewRun(555)
	for i := 0; i < 11; i++ {
		g.randIntn(10)
	}
	save := g.buildSaveData()
	restored := &Game{}
	if err := restored.applySaveData(save); err != nil {
		t.Fatal(err)
	}
	if restored.seed != 555 {
		t.Fatalf("restored seed = %d", restored.seed)
	}
	for i := 0; i < 50; i++ {
		if a, b := g.randIntn(1000), restored.randIntn(1000); a != b {
			t.Fatalf("draw %d after restore: %d vs %d", i, a, b)
		}
	}
}

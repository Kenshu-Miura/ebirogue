//go:build !test

package main

import "testing"

// 実際の生成条件（70x70・ランダム部屋）で、リトライ込みの生成が
// 常に「全部屋連結かつ壁が破壊されていない」フロアを作れることを確認する
func TestMapGenStress(t *testing.T) {
	const iterations = 300
	rawFailures := 0
	finalFailures := 0

	for i := 0; i < iterations; i++ {
		ok := false
		for attempt := 0; attempt < maxMapGenAttempts; attempt++ {
			grid := makeBlockedGrid(70, 70)
			rooms := generateRooms(grid, 70, 70, 6)
			if len(rooms) < 2 {
				rawFailures++
				continue
			}
			connectRooms(rooms, grid)
			if floorConnected(grid, rooms) && roomWallsIntact(grid, rooms) {
				ok = true
				break
			}
			rawFailures++
		}
		if !ok {
			finalFailures++
		}
	}

	t.Logf("iterations=%d rawFailures=%d finalFailures=%d", iterations, rawFailures, finalFailures)
	if finalFailures > 0 {
		t.Fatalf("%d/%d generations could not produce a valid floor within %d attempts", finalFailures, iterations, maxMapGenAttempts)
	}
}

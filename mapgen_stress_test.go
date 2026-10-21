//go:build !test

package main

import "testing"

func TestTerrainMapGenStress(t *testing.T) {
	counts := map[string]int{}
	for seed := 0; seed < 2000; seed++ {
		floor := 1 + seed%30
		player := Player{}
		grid, enemies, items, _, rooms, traps := GenerateRandomMap(70, 70, floor-1, &player, newFloorRNG(int64(seed), floor))
		if !floorConnected(grid, rooms) {
			t.Fatalf("seed=%d floor=%d disconnected", seed, floor)
		}
		if grid[player.Y][player.X].Blocked {
			t.Fatalf("seed=%d player on blocked tile", seed)
		}
		for _, e := range enemies {
			if grid[e.Y][e.X].Blocked {
				t.Fatalf("seed=%d enemy on terrain", seed)
			}
		}
		for _, item := range items {
			x, y := item.GetPosition()
			if grid[y][x].Blocked {
				t.Fatalf("seed=%d item on terrain", seed)
			}
		}
		for _, trap := range traps {
			if grid[trap.Y][trap.X].Blocked {
				t.Fatalf("seed=%d trap on terrain", seed)
			}
		}
		for y, row := range grid {
			for x, tile := range row {
				switch tile.Type {
				case "water", "hidden_corridor", "cracked_wall":
					counts[tile.Type]++
					if isBossFloor(floor) || x == 0 || y == 0 || x == 69 || y == 69 {
						t.Fatalf("seed=%d invalid terrain location", seed)
					}
					if tile.Type == "water" && (floor < 4 || tile.BlockSight) {
						t.Fatalf("invalid water: %+v", tile)
					}
					if tile.Type == "cracked_wall" && floor < 2 {
						t.Fatal("cracked wall too early")
					}
					if tile.Type == "hidden_corridor" && floor < 6 {
						t.Fatal("hidden corridor too early")
					}
				}
			}
		}
		for _, room := range rooms {
			water := 0
			for y := room.Y + 1; y < room.Y+room.Height-1; y++ {
				for x := room.X + 1; x < room.X+room.Width-1; x++ {
					if grid[y][x].Type == "water" {
						water++
					}
				}
			}
			if water > (room.Width-2)*(room.Height-2)/4 {
				t.Fatalf("seed=%d room flooded", seed)
			}
		}
	}
	for _, kind := range []string{"water", "cracked_wall", "hidden_corridor"} {
		if counts[kind] == 0 {
			t.Fatalf("never generated %s", kind)
		}
	}
	t.Logf("2000 seeds: %v", counts)
}

// 実際の生成条件（70x70・ランダム部屋）で、リトライ込みの生成が
// 常に「全部屋連結かつ壁が破壊されていない」フロアを作れることを確認する
func TestMapGenStress(t *testing.T) {
	const iterations = 300
	rawFailures := 0
	finalFailures := 0

	for i := 0; i < iterations; i++ {
		rng := newFloorRNG(int64(i), 1)
		ok := false
		for attempt := 0; attempt < maxMapGenAttempts; attempt++ {
			grid := makeBlockedGrid(70, 70)
			rooms := generateRooms(grid, 70, 70, 6, rng)
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

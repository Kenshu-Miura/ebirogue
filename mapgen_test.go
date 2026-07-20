package main

import "testing"

func makeBlockedGrid(width, height int) [][]Tile {
	grid := make([][]Tile, height)
	for y := range grid {
		grid[y] = make([]Tile, width)
		for x := range grid[y] {
			grid[y][x] = Tile{Type: "other", Blocked: true, BlockSight: true}
		}
	}
	return grid
}

func TestPlanCorridorDefaultsToVerticalFirst(t *testing.T) {
	legs := planCorridor(Coordinate{X: 5, Y: 5}, Coordinate{X: 15, Y: 15}, nil)
	if len(legs) != 2 {
		t.Fatalf("legs = %d, want 2", len(legs))
	}
	if legs[0].x1 != legs[0].x2 {
		t.Fatalf("first leg should be vertical: %+v", legs[0])
	}
	if legs[1].y1 != legs[1].y2 {
		t.Fatalf("second leg should be horizontal: %+v", legs[1])
	}
}

func TestPlanCorridorAvoidsWallParallelRun(t *testing.T) {
	// 縦→横の経路（x=5 の縦区間）が左壁 x=5 の部屋と平行に重なるよう配置する
	obstacle := Room{ID: 9, X: 5, Y: 8, Width: 6, Height: 5}
	setRoomCenter(&obstacle)
	rooms := []Room{obstacle}

	legs := planCorridor(Coordinate{X: 5, Y: 5}, Coordinate{X: 15, Y: 15}, rooms)
	if legs[0].y1 != legs[0].y2 {
		t.Fatalf("expected horizontal-first path to avoid running along the wall, got %+v", legs)
	}
}

func TestCarveLegPreservesRoomInterior(t *testing.T) {
	room := Room{ID: 0, X: 3, Y: 3, Width: 7, Height: 7}
	setRoomCenter(&room)
	rooms := []Room{room}

	grid := makeBlockedGrid(20, 20)
	carveRoom(grid, room)
	carveLeg(grid, corridorLeg{x1: 6, y1: 0, x2: 6, y2: 12}, rooms)

	if grid[0][6].Type != "corridor" {
		t.Fatalf("tile outside the room should be corridor, got %q", grid[0][6].Type)
	}
	if grid[3][6].Type != "corridor" || grid[9][6].Type != "corridor" {
		t.Fatalf("wall tiles crossed by the leg should become corridor, got %q and %q", grid[3][6].Type, grid[9][6].Type)
	}
	if grid[5][6].Type != "floor" {
		t.Fatalf("room interior must stay floor, got %q", grid[5][6].Type)
	}
}

func TestSpanningTreeEdgesConnectAllRooms(t *testing.T) {
	rooms := []Room{
		{ID: 0, Center: Coordinate{X: 5, Y: 5}},
		{ID: 1, Center: Coordinate{X: 30, Y: 5}},
		{ID: 2, Center: Coordinate{X: 5, Y: 30}},
		{ID: 3, Center: Coordinate{X: 30, Y: 30}},
		{ID: 4, Center: Coordinate{X: 60, Y: 15}},
	}
	edges := spanningTreeEdges(rooms)
	if len(edges) != len(rooms)-1 {
		t.Fatalf("edges = %d, want %d", len(edges), len(rooms)-1)
	}

	// Union-Find で全部屋の連結を確認する
	parent := make([]int, len(rooms))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	for _, e := range edges {
		parent[find(e[0])] = find(e[1])
	}
	root := find(0)
	for i := range rooms {
		if find(i) != root {
			t.Fatalf("room %d is not connected by the spanning tree", i)
		}
	}
}

func TestConnectRoomsProducesConnectedFloor(t *testing.T) {
	rooms := []Room{
		{ID: 0, X: 2, Y: 2, Width: 8, Height: 8},
		{ID: 1, X: 20, Y: 2, Width: 8, Height: 8},
		{ID: 2, X: 2, Y: 20, Width: 8, Height: 8},
	}
	grid := makeBlockedGrid(40, 40)
	for i := range rooms {
		setRoomCenter(&rooms[i])
		carveRoom(grid, rooms[i])
	}

	connectRooms(rooms, grid)

	if !floorConnected(grid, rooms) {
		t.Fatal("all rooms should be reachable after connectRooms")
	}

	// 通路の出入口以外の壁が保たれていることを確認する（壁の平行破壊の検出）
	for _, room := range rooms {
		openings := 0
		for y := room.Y; y < room.Y+room.Height; y++ {
			for x := room.X; x < room.X+room.Width; x++ {
				if isOnBoundary(x, y, room) && grid[y][x].Type == "corridor" {
					openings++
				}
			}
		}
		if openings == 0 || openings > 4 {
			t.Fatalf("room %d has %d corridor openings, want 1-4", room.ID, openings)
		}
	}
}

func TestFloorConnectedDetectsIsolatedRoom(t *testing.T) {
	rooms := []Room{
		{ID: 0, X: 2, Y: 2, Width: 8, Height: 8},
		{ID: 1, X: 20, Y: 2, Width: 8, Height: 8},
	}
	grid := makeBlockedGrid(40, 40)
	for i := range rooms {
		setRoomCenter(&rooms[i])
		carveRoom(grid, rooms[i])
	}

	if floorConnected(grid, rooms) {
		t.Fatal("rooms without corridors must not be reported as connected")
	}
}

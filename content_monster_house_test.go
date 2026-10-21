//go:build !test

package main

import (
	"encoding/json"
	"testing"
)

func TestMonsterHouseGenerationRates(t *testing.T) {
	for _, floor := range []int{1, 3, 4, 9, 10, 11, 20, 21, 30} {
		houses := 0
		for seed := int64(0); seed < 600; seed++ {
			p := Player{}
			grid, enemies, items, _, rooms, traps := GenerateRandomMapFor(70, 70, floor-1, &p, newFloorRNG(seed, floor), DifficultyAdvanced)
			for _, r := range rooms {
				if r.Role != RoomRoleMonsterHouse {
					continue
				}
				houses++
				if roomContains(r, p.X, p.Y) {
					t.Fatal("house in start room")
				}
				for y := r.Y + 1; y < r.Y+r.Height-1; y++ {
					for x := r.X + 1; x < r.X+r.Width-1; x++ {
						if grid[y][x].Type == "stairs" {
							t.Fatal("house in stairs room")
						}
					}
				}
				for _, e := range enemies {
					if roomContains(r, e.X, e.Y) && (grid[e.Y][e.X].Blocked || e.StatusAilments.Sleep != enemyNapSleep) {
						t.Fatal("invalid sleeping enemy")
					}
				}
				for _, item := range items {
					x, y := item.GetPosition()
					if grid[y][x].Blocked {
						t.Fatal("item on terrain")
					}
				}
				for _, trap := range traps {
					if grid[trap.Y][trap.X].Blocked {
						t.Fatal("trap on terrain")
					}
				}
			}
		}
		chance := monsterHouseChance(floor)
		if chance == 0 && houses != 0 {
			t.Fatalf("floor %d houses=%d", floor, houses)
		}
		if chance > 0 && (houses < chance*3 || houses > chance*9) {
			t.Fatalf("floor %d houses=%d/600", floor, houses)
		}
	}
}

func TestMonsterHouseCountsAndWater(t *testing.T) {
	grid := makeTestFloorMap(12, 12)
	grid[3][3] = Tile{Type: "water", Blocked: true}
	rooms := []Room{{ID: 1, X: 0, Y: 0, Width: 3, Height: 3}, {ID: 2, X: 2, Y: 2, Width: 9, Height: 9}}
	p := Player{Entity: Entity{X: 1, Y: 1}}
	intn := func(int) int { return 0 }
	enemies, items, traps := generateMonsterHouse(grid, rooms, &p, Coordinate{X: 1, Y: 1}, 4, DifficultyAdvanced, intn, nil, nil, nil)
	// 7x7 の床から水路1マスを引く。
	if len(enemies) != 16 || len(items) != 9 || len(traps) != 2 {
		t.Fatalf("counts %d/%d/%d", len(enemies), len(items), len(traps))
	}
	seen := map[Coordinate]bool{}
	check := func(x, y int) {
		c := Coordinate{X: x, Y: y}
		if seen[c] || grid[y][x].Blocked {
			t.Fatal("overlap or water")
		}
		seen[c] = true
	}
	for _, e := range enemies {
		check(e.X, e.Y)
	}
	for _, item := range items {
		x, y := item.GetPosition()
		check(x, y)
	}
	for _, trap := range traps {
		check(trap.X, trap.Y)
	}
	if selectSpecialRoom(rooms, Coordinate{X: 1, Y: 1}, Coordinate{X: 4, Y: 4}, intn) != -1 {
		t.Fatal("selected occupied role/start/stairs")
	}
	for _, n := range []int{0, 1} {
		e, i := monsterHouseCounts(48, true, func(max int) int {
			if n == 0 {
				return 0
			}
			return max - 1
		})
		if e != 5+3*n || i != 2+n {
			t.Fatal("card counts changed")
		}
	}
}

func TestMonsterHouseEntryAndSave(t *testing.T) {
	g, _ := newHeadlessGame(5, DifficultyAdvanced)
	g.rooms = []Room{{ID: 1, X: 0, Y: 0, Width: 9, Height: 9, Role: RoomRoleMonsterHouse}}
	g.state.Map = makeTestFloorMap(12, 12)
	g.state.Player.X = 4
	g.state.Player.Y = 4
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 5), CreateEnemyByID(0, 6, 5), CreateEnemyByID(0, 10, 10)}
	for i := range g.state.Enemies {
		setStatusTurns(&g.state.Enemies[i].StatusAilments, StatusSleep, enemyNapSleep)
	}
	restore := func() *Game {
		t.Helper()
		data, err := json.Marshal(g.buildSaveData())
		if err != nil {
			t.Fatal(err)
		}
		var s SaveData
		if err = json.Unmarshal(data, &s); err != nil {
			t.Fatal(err)
		}
		h, _ := newHeadlessGame(6, DifficultyAdvanced)
		if err = h.applySaveData(&s); err != nil {
			t.Fatal(err)
		}
		return h
	}
	h := restore()
	if h.rooms[0].Entered || h.rooms[0].Role != RoomRoleMonsterHouse {
		t.Fatal("lost unentered room")
	}
	g.previousPlayerRoomID = -1
	g.CheckSleepingEnemyWakeUp()
	if !g.rooms[0].Entered || g.state.Enemies[0].StatusAilments.Sleep != 0 || g.state.Enemies[1].StatusAilments.Sleep != 0 || g.state.Enemies[2].StatusAilments.Sleep != enemyNapSleep {
		t.Fatal("wake scope")
	}
	if len(g.ActionQueue.Queue) == 0 {
		t.Fatal("missing announcement")
	}
	before := len(g.ActionQueue.Queue)
	g.previousPlayerRoomID = -1
	g.CheckSleepingEnemyWakeUp()
	if len(g.ActionQueue.Queue) != before {
		t.Fatal("duplicate announcement")
	}
	h = restore()
	if !h.rooms[0].Entered {
		t.Fatal("lost entered flag")
	}
}

func TestMonsterHouseExpertRetreatsToCorridor(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	m := makeTestFloorMap(12, 7)
	for y := 0; y < 7; y++ {
		for x := 8; x < 12; x++ {
			if y != 3 || x == 11 {
				m[y][x] = Tile{Type: "wall", Blocked: true, BlockSight: true}
			} else {
				m[y][x] = Tile{Type: "corridor"}
			}
		}
	}
	g.state.Map = m
	g.state.Player.X, g.state.Player.Y = 6, 3
	g.state.Enemies = []Enemy{newActiveEnemy(0, 3, 1), newActiveEnemy(0, 3, 5)}
	g.rooms[0].Role = RoomRoleMonsterHouse
	for i := range g.state.Enemies {
		setStatusTurns(&g.state.Enemies[i].StatusAilments, StatusSleep, enemyNapSleep)
	}
	g.previousPlayerRoomID = -1
	g.CheckSleepingEnemyWakeUp()
	if err := g.RunUntilInputReady(100); err != nil {
		t.Fatal(err)
	}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if b.act != "retreat" || g.state.Player.X != 7 {
		t.Fatalf("did not retreat toward the corridor: act=%s player=(%d,%d)", b.act, g.state.Player.X, g.state.Player.Y)
	}
}

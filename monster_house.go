//go:build !test

package main

// 通常の湧き上限は変えず、生成時の群れだけは19体を超えてよい。
func generateMonsterHouse(grid [][]Tile, rooms []Room, player *Player, stairs Coordinate, floor int, difficulty Difficulty, intn func(int) int, enemies []Enemy, items []Item, traps []MapTrap) ([]Enemy, []Item, []MapTrap) {
	chance := monsterHouseChance(floor)
	if chance == 0 || intn(100) >= chance {
		return enemies, items, traps
	}
	i := selectSpecialRoom(rooms, Coordinate{X: player.X, Y: player.Y}, stairs, intn)
	if i < 0 {
		return enemies, items, traps
	}
	room := rooms[i]
	occupied := map[Coordinate]bool{}
	for _, e := range enemies {
		occupied[Coordinate{X: e.X, Y: e.Y}] = true
	}
	for _, item := range items {
		x, y := item.GetPosition()
		occupied[Coordinate{X: x, Y: y}] = true
	}
	for _, trap := range traps {
		occupied[Coordinate{X: trap.X, Y: trap.Y}] = true
	}
	floors := 0
	for y := room.Y + 1; y < room.Y+room.Height-1; y++ {
		for x := room.X + 1; x < room.X+room.Width-1; x++ {
			if !grid[y][x].Blocked && grid[y][x].Type == "floor" {
				floors++
			}
		}
	}
	enemyCount, itemCount := monsterHouseCounts(floors, false, intn)
	trapCount := floors / 20
	cells := pickFreeCellsInRoom(room, func(x, y int) bool {
		return !grid[y][x].Blocked && grid[y][x].Type == "floor" && !occupied[Coordinate{X: x, Y: y}]
	}, enemyCount+itemCount+trapCount, intn)
	rooms[i].Role = RoomRoleMonsterHouse
	for n, cell := range cells {
		switch {
		case n < enemyCount:
			id := selectMonsterForFloor(floor, nil, intn)
			if id < 0 {
				continue
			}
			e := CreateEnemyByID(id, cell.X, cell.Y)
			prepareSpawnedEnemy(&e, floor, intn)
			setStatusTurns(&e.StatusAilments, StatusSleep, enemyNapSleep)
			enemies = append(enemies, e)
		case n < enemyCount+itemCount:
			items = append(items, createFloorItem(cell.X, cell.Y, floor, difficulty, intn))
		default:
			traps = append(traps, createMapTrapByID(pickFloorTrapID(floor, intn), cell.X, cell.Y))
		}
	}
	// もともと部屋にいた敵も含め、入室までは全員仮眠する。
	for j := range enemies {
		if roomContains(room, enemies[j].X, enemies[j].Y) {
			setStatusTurns(&enemies[j].StatusAilments, StatusSleep, enemyNapSleep)
		}
	}
	return enemies, items, traps
}

func (g *Game) enterMonsterHouse(roomID int) bool {
	for i := range g.rooms {
		r := &g.rooms[i]
		if r.ID != roomID || r.Role != RoomRoleMonsterHouse || r.Entered {
			continue
		}
		r.Entered = true
		for j := range g.state.Enemies {
			e := &g.state.Enemies[j]
			if roomContains(*r, e.X, e.Y) {
				g.clearEnemyStatus(j, StatusSleep)
			}
		}
		g.EnqueueMessage("モンスターハウスだ！", 0.5)
		return true
	}
	return false
}

func (g *Game) inMonsterHouse() bool {
	for _, r := range g.rooms {
		if r.Role == RoomRoleMonsterHouse && roomContains(r, g.state.Player.X, g.state.Player.Y) {
			return true
		}
	}
	return false
}

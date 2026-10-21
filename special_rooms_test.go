package main

import "testing"

func TestSpecialRoomSelection(t *testing.T) {
	rooms := []Room{{ID: 1, X: 0, Y: 0, Width: 5, Height: 5}, {ID: 2, X: 6, Y: 0, Width: 5, Height: 5}, {ID: 3, X: 12, Y: 0, Width: 5, Height: 5}, {ID: 4, X: 18, Y: 0, Width: 5, Height: 5}}
	start, stairs := Coordinate{X: 1, Y: 1}, Coordinate{X: 7, Y: 1}
	for n, want := range []int{2, 3} {
		if got := selectSpecialRoom(rooms, start, stairs, func(int) int { return n }); got != want {
			t.Fatalf("got %d want %d", got, want)
		}
	}
	rooms[2].Role = RoomRoleMonsterHouse
	rooms[3].Role = "shop"
	if got := selectSpecialRoom(rooms, start, stairs, func(int) int { t.Fatal("unexpected random draw"); return 0 }); got != -1 {
		t.Fatal("selected reserved room")
	}
}

func TestMonsterHouseCountPolicy(t *testing.T) {
	for _, cells := range []int{0, 1, 5, 30, 99} {
		e, i := monsterHouseCounts(cells, false, func(int) int { t.Fatal("density must be deterministic"); return 0 })
		if e != cells/3 || i != cells/5 {
			t.Fatal("wrong density")
		}
	}
	for floor := 1; floor <= 30; floor++ {
		want := 8
		if floor < 10 {
			want = 5
		}
		if floor < 4 || isBossFloor(floor) {
			want = 0
		}
		if monsterHouseChance(floor) != want {
			t.Fatalf("floor %d", floor)
		}
	}
}

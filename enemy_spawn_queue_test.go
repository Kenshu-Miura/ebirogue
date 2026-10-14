package main

import (
	"reflect"
	"testing"
)

func TestTickPendingEnemySpawns(t *testing.T) {
	spawns := []PendingEnemySpawn{{MonsterID: 11, Turns: 1}, {MonsterID: 29, Turns: 3}, {MonsterID: 56, Turns: 0}}
	due, waiting := tickPendingEnemySpawns(spawns)
	if !reflect.DeepEqual(due, []PendingEnemySpawn{{MonsterID: 11, Turns: 0}, {MonsterID: 56, Turns: -1}}) {
		t.Fatalf("due = %+v", due)
	}
	if !reflect.DeepEqual(waiting, []PendingEnemySpawn{{MonsterID: 29, Turns: 2}}) {
		t.Fatalf("waiting = %+v", waiting)
	}
	if spawns[0].Turns != 1 {
		t.Fatal("input slice was modified")
	}
}

func TestNearestFreeOffsets(t *testing.T) {
	offsets := nearestFreeOffsets(2)
	if len(offsets) != 25 {
		t.Fatalf("offsets = %d, want 25", len(offsets))
	}
	want := []Coordinate{{0, 0}, {-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}, {-2, -2}}
	if !reflect.DeepEqual(offsets[:len(want)], want) {
		t.Fatalf("first offsets = %+v, want %+v", offsets[:len(want)], want)
	}
}

func TestInsertHelpPagesAfter(t *testing.T) {
	pages := []HelpPage{{Title: "A"}, {Title: "特殊な敵"}, {Title: "罠"}}
	got := insertHelpPagesAfter(pages, "特殊な敵", []HelpPage{{Title: "新1"}, {Title: "新2"}})
	titles := []string{}
	for _, page := range got {
		titles = append(titles, page.Title)
	}
	if !reflect.DeepEqual(titles, []string{"A", "特殊な敵", "新1", "新2", "罠"}) {
		t.Fatalf("titles = %v", titles)
	}
	if len(insertHelpPagesAfter(pages, "特殊な敵", nil)) != 3 {
		t.Fatal("empty extra pages changed the list")
	}
}

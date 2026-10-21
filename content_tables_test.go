//go:build !test

package main

// アイテム・敵・罠テーブルの整合性テストと、コンテンツテスト共通のヘルパー。

import "testing"

func TestTemplateTablesHaveConsistentIDs(t *testing.T) {
	g := &Game{}
	for id, template := range itemTemplates {
		if template.ID != id {
			t.Fatalf("itemTemplates[%d].ID = %d", id, template.ID)
		}
		if item := g.createItemByID(id, 1, 2); item == nil || item.GetName() != template.Name {
			t.Fatalf("item %d generated %#v, want %q", id, item, template.Name)
		}
	}
	for id, definition := range MonsterDefinitions {
		if definition.ID != id {
			t.Fatalf("MonsterDefinitions[%d].ID = %d", id, definition.ID)
		}
		if enemy := CreateEnemyByID(id, 1, 2); enemy.ID != id || enemy.Name != definition.Name {
			t.Fatalf("enemy %d generated %#v", id, enemy)
		}
	}
	for id, template := range mapTrapTemplates {
		if template.ID != id {
			t.Fatalf("mapTrapTemplates[%d].ID = %d", id, template.ID)
		}
		trap := createMapTrapByID(id, 1, 2)
		if trap.TemplateID != id || trap.Name != template.Name || trap.Effect == nil {
			t.Fatalf("trap %d generated %#v", id, trap)
		}
		if got := mapTrapTemplateID(trap); got != id {
			t.Fatalf("mapTrapTemplateID(trap %d) = %d", id, got)
		}
	}
	if got := mapTrapTemplateID(MapTrap{Name: "地雷"}); got != 3 {
		t.Fatalf("trap without TemplateID should be resolved by name, got %d", got)
	}
}

func TestRandomIDListsCoverTablesAndExcludeBosses(t *testing.T) {
	if len(randomItemTemplateIDs) != len(itemTemplates) || len(randomMapTrapIDs) != len(mapTrapTemplates) {
		t.Fatalf("random item/trap ID lists = %d/%d, want %d/%d", len(randomItemTemplateIDs), len(randomMapTrapIDs), len(itemTemplates), len(mapTrapTemplates))
	}
	for _, id := range randomMonsterIDs {
		definition, ok := MonsterDefinitions[id]
		if !ok || definition.IsBoss {
			t.Fatalf("random monster ID %d must exist and must not be a boss", id)
		}
	}
	bosses := 0
	for _, definition := range MonsterDefinitions {
		if definition.IsBoss || isShopMonsterID(definition.ID) { // 店主・番犬もランダム出現の対象外（wave10-B）
			bosses++
		}
	}
	if len(randomMonsterIDs)+bosses != len(MonsterDefinitions) {
		t.Fatalf("random monster IDs = %d, want %d non-boss monsters", len(randomMonsterIDs), len(MonsterDefinitions)-bosses)
	}
	// 欠番とボスを含むテーブルでも、一覧にある ID だけが選ばれる。
	gapped := map[int]MonsterDefinition{0: {ID: 0}, 5: {ID: 5}, 9: {ID: 9, IsBoss: true}}
	ids := sortedTemplateIDs(gapped, func(def MonsterDefinition) bool { return !def.IsBoss })
	for i := 0; i < len(ids); i++ {
		if got := pickTemplateID(ids, func(int) int { return i }, -1); got != []int{0, 5}[i] {
			t.Fatalf("gapped pick %d = %d", i, got)
		}
	}
}

func makeTestFloorMap(width, height int) [][]Tile {
	mapGrid := make([][]Tile, height)
	for y := range mapGrid {
		mapGrid[y] = make([]Tile, width)
		for x := range mapGrid[y] {
			if x == 0 || x == width-1 || y == 0 || y == height-1 {
				mapGrid[y][x] = Tile{Type: "wall", Blocked: true, BlockSight: true}
			} else {
				mapGrid[y][x] = Tile{Type: "floor"}
			}
		}
	}
	return mapGrid
}

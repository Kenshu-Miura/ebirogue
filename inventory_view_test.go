package main

import "testing"

func makePresent(categories ...ItemCategory) []bool {
	present := make([]bool, categoryCount)
	for _, c := range categories {
		present[c] = true
	}
	return present
}

func TestNextFilterCyclesPresentCategories(t *testing.T) {
	present := makePresent(CategoryWeapon, CategoryCane, CategoryCard)

	// 全て → 武器 → 杖 → カード → 全て と存在するカテゴリだけを巡回する
	f := nextFilter(CategoryAll, present)
	if f != CategoryWeapon {
		t.Errorf("1回目の切替: got %v, want CategoryWeapon", f)
	}
	f = nextFilter(f, present)
	if f != CategoryCane {
		t.Errorf("2回目の切替: got %v, want CategoryCane", f)
	}
	f = nextFilter(f, present)
	if f != CategoryCard {
		t.Errorf("3回目の切替: got %v, want CategoryCard", f)
	}
	f = nextFilter(f, present)
	if f != CategoryAll {
		t.Errorf("4回目の切替: got %v, want CategoryAll", f)
	}
}

func TestNextFilterEmptyInventory(t *testing.T) {
	present := makePresent()
	if f := nextFilter(CategoryAll, present); f != CategoryAll {
		t.Errorf("空のインベントリでは全て表示のまま: got %v", f)
	}
}

func TestFilteredIndices(t *testing.T) {
	categories := []ItemCategory{CategoryWeapon, CategoryCard, CategoryWeapon, CategoryCane}

	all := filteredIndices(categories, CategoryAll)
	if len(all) != 4 {
		t.Errorf("全て表示のインデックス数: got %d, want 4", len(all))
	}

	weapons := filteredIndices(categories, CategoryWeapon)
	if len(weapons) != 2 || weapons[0] != 0 || weapons[1] != 2 {
		t.Errorf("武器の絞り込みが不正: %v", weapons)
	}

	none := filteredIndices(categories, CategoryMoney)
	if len(none) != 0 {
		t.Errorf("該当なしの絞り込みが不正: %v", none)
	}
}

func TestMoveSelection(t *testing.T) {
	indices := []int{0, 2, 5}

	// リスト内の移動
	if got := moveSelection(indices, 0, 1); got != 2 {
		t.Errorf("下移動: got %d, want 2", got)
	}
	if got := moveSelection(indices, 2, 1); got != 5 {
		t.Errorf("下移動2: got %d, want 5", got)
	}
	if got := moveSelection(indices, 5, -1); got != 2 {
		t.Errorf("上移動: got %d, want 2", got)
	}

	// 端で止まる
	if got := moveSelection(indices, 0, -1); got != 0 {
		t.Errorf("先頭で上移動: got %d, want 0", got)
	}
	if got := moveSelection(indices, 5, 10); got != 5 {
		t.Errorf("末尾を越える移動: got %d, want 5", got)
	}

	// リストにない位置からは先頭へ移動
	if got := moveSelection(indices, 3, 1); got != 0 {
		t.Errorf("リスト外からの移動: got %d, want 0", got)
	}

	// 空のリストでは現在位置を維持
	if got := moveSelection(nil, 3, 1); got != 3 {
		t.Errorf("空リストでの移動: got %d, want 3", got)
	}
}

func TestNameInput(t *testing.T) {
	var n NameInput

	// 最大文字数まで追加できる
	for i := 0; i < maxItemNameLength; i++ {
		if !n.Append('あ') {
			t.Fatalf("%d文字目の追加に失敗", i+1)
		}
	}
	if n.Append('い') {
		t.Error("最大文字数を超えて追加できた")
	}
	if n.String() != "ああああああ" {
		t.Errorf("入力結果が不正: %s", n.String())
	}

	// 削除
	if !n.Backspace() {
		t.Error("削除に失敗")
	}
	if len(n.Runes) != maxItemNameLength-1 {
		t.Errorf("削除後の文字数が不正: %d", len(n.Runes))
	}

	// 空になったら削除は false
	n.Runes = nil
	if n.Backspace() {
		t.Error("空の状態で削除できた")
	}
}

func TestMoveGridCursor(t *testing.T) {
	total := len(kanaChars)

	if got := moveGridCursor(0, 1, total); got != 1 {
		t.Errorf("右移動: got %d, want 1", got)
	}
	if got := moveGridCursor(0, -1, total); got != 0 {
		t.Errorf("先頭で左移動: got %d, want 0", got)
	}
	if got := moveGridCursor(5, kanaGridColumns, total); got != 5+kanaGridColumns {
		t.Errorf("下移動: got %d, want %d", got, 5+kanaGridColumns)
	}
	if got := moveGridCursor(total-1, kanaGridColumns, total); got != total-1 {
		t.Errorf("最終行で下移動: got %d, want %d", got, total-1)
	}
}

func TestCategoryLabel(t *testing.T) {
	if categoryLabel(CategoryAll) != "全て" {
		t.Errorf("CategoryAllのラベルが不正: %s", categoryLabel(CategoryAll))
	}
	if categoryLabel(CategoryWeapon) != "武器" {
		t.Errorf("CategoryWeaponのラベルが不正: %s", categoryLabel(CategoryWeapon))
	}
}

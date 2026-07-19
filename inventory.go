//go:build !test

package main

import "sort"

// itemCategory はアイテムの分類を返す
func itemCategory(item Item) ItemCategory {
	switch item.(type) {
	case *Weapon:
		return CategoryWeapon
	case *Armor:
		return CategoryArmor
	case *Arrow:
		return CategoryArrow
	case *Accessory:
		return CategoryAccessory
	case *Cane:
		return CategoryCane
	case *Pot:
		return CategoryPot
	case *Food:
		return CategoryFood
	case *Potion:
		return CategoryPotion
	case *Card:
		return CategoryCard
	case *Trap:
		return CategoryTrap
	case *Money:
		return CategoryMoney
	default:
		return CategoryOther
	}
}

// presentCategories はインベントリに存在するカテゴリの一覧を返す
func (g *Game) presentCategories() []bool {
	present := make([]bool, categoryCount)
	for _, item := range g.state.Player.Inventory {
		present[itemCategory(item)] = true
	}
	return present
}

// inventoryCategories はインベントリ各アイテムのカテゴリを返す
func (g *Game) inventoryCategories() []ItemCategory {
	categories := make([]ItemCategory, len(g.state.Player.Inventory))
	for i, item := range g.state.Player.Inventory {
		categories[i] = itemCategory(item)
	}
	return categories
}

// normalizeInventoryView は絞り込み後の表示インデックスを返す。
// 絞り込み対象が無くなった場合は全て表示へ戻し、
// 選択位置が表示リストから外れている場合は先頭へ合わせる。
func (g *Game) normalizeInventoryView() []int {
	indices := filteredIndices(g.inventoryCategories(), g.inventoryFilter)
	if len(indices) == 0 && g.inventoryFilter != CategoryAll {
		g.inventoryFilter = CategoryAll
		indices = filteredIndices(g.inventoryCategories(), CategoryAll)
	}
	if len(indices) > 0 {
		found := false
		for _, idx := range indices {
			if idx == g.selectedItemIndex {
				found = true
				break
			}
		}
		if !found {
			g.selectedItemIndex = indices[0]
		}
	}
	return indices
}

// sortInventory はインベントリをカテゴリ順・ID順に整頓し、同じ矢を1つへ統合する
func (g *Game) sortInventory() {
	// カテゴリ順、同カテゴリ内はID順で安定ソート
	sort.SliceStable(g.state.Player.Inventory, func(i, j int) bool {
		itemI, itemJ := g.state.Player.Inventory[i], g.state.Player.Inventory[j]
		catI, catJ := itemCategory(itemI), itemCategory(itemJ)
		if catI != catJ {
			return catI < catJ
		}
		return itemI.GetID() < itemJ.GetID()
	})

	// 同じIDの矢を1つへ統合する
	arrowItemsMap := make(map[int][]*Arrow)
	for _, item := range g.state.Player.Inventory {
		if arrow, ok := item.(*Arrow); ok {
			arrowItemsMap[arrow.GetID()] = append(arrowItemsMap[arrow.GetID()], arrow)
		}
	}

	for _, arrows := range arrowItemsMap {
		if len(arrows) > 1 {
			var totalShotCount int
			var equippedArrow *Arrow
			for _, arrow := range arrows {
				totalShotCount += arrow.ShotCount
				if g.state.Player.EquippedArrow == arrow {
					equippedArrow = arrow
				}
			}

			// 装備中の矢があればそれを統合先にする
			mergedArrow := equippedArrow
			if mergedArrow == nil {
				mergedArrow = arrows[0]
			}
			mergedArrow.ShotCount = totalShotCount

			// 統合先以外の矢をインベントリから取り除く
			newInventory := []Item{}
			for _, item := range g.state.Player.Inventory {
				keep := true
				for _, arrow := range arrows {
					if item == arrow && arrow != mergedArrow {
						keep = false
						break
					}
				}
				if keep {
					newInventory = append(newInventory, item)
				}
			}
			g.state.Player.Inventory = newInventory
		}
	}
}

// inventoryItemLabel はインベントリでの表示名と未識別かどうかを返す。
// 未識別アイテムに任意名が設定されていればそれを表示する。
func (g *Game) inventoryItemLabel(item Item) (string, bool) {
	if identifiableItem, ok := item.(Identifiable); ok && !identifiableItem.IsIdentified() {
		if name, exists := g.customNames[item.GetID()]; exists && name != "" {
			return name, true
		}
		return identifiableItem.GetName(), true
	}
	return getItemNameWithSharpness(item), false
}

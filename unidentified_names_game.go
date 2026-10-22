//go:build !test

package main

import "sort"

// unidentifiedNameTemplateIDs は種類ごとの、未識別で生まれるテンプレート ID の一覧（昇順）。
// 最初から識別済みの道具（保存の壺など）は含めない。unidentifiedNameKinds は ID → 種類。
// 名前の関数はアイテムの表の初期化から参照されるので、変数の初期化式ではなく init で作る（初期化の循環を避ける）。
var (
	unidentifiedNameTemplateIDs map[string][]int
	unidentifiedNameKinds       map[int]string
)

func init() {
	unidentifiedNameTemplateIDs = map[string][]int{}
	unidentifiedNameKinds = map[int]string{}
	for _, id := range sortedTemplateIDs(itemTemplates, nil) {
		kind := itemTemplates[id].ItemType
		if _, ok := unidentifiedNamePools[kind]; !ok {
			continue
		}
		if v, ok := buildItemFromTemplate(id, 0, 0).(Identifiable); ok && !v.IsIdentified() {
			unidentifiedNameTemplateIDs[kind] = append(unidentifiedNameTemplateIDs[kind], id)
			unidentifiedNameKinds[id] = kind
		}
	}
}

// unidentifiedAppearance はテンプレート ID の見た目の番号（名前の候補の番号）を返す。
// 冒険のシードから作り、シードが変わったら作り直す。
func (g *Game) unidentifiedAppearance(id int) (int, bool) {
	if g.unidentifiedAppearances == nil || g.unidentifiedAppearanceSeed != g.seed {
		r := newGameRNG(g.seed, rngStreamUnidentifiedNames)
		g.unidentifiedAppearances = assignUnidentifiedAppearances(unidentifiedNameTemplateIDs, r.Intn)
		g.unidentifiedAppearanceSeed = g.seed
	}
	index, ok := g.unidentifiedAppearances[id]
	return index, ok
}

// unidentifiedName は未識別のときに見せる名前を返す（未識別名の無い道具はテンプレートの名前）。
func (g *Game) unidentifiedName(item Item) string {
	if index, ok := g.unidentifiedAppearance(item.GetID()); ok {
		return unidentifiedNamePools[unidentifiedNameKinds[item.GetID()]][index]
	}
	return item.GetName()
}

// potAppearanceImageID は壺の画像のファイル番号（pot_<番号>.png）を返す。
// 色の名前と画像をそろえるため、識別の前後を問わず見た目の番号で選ぶ。
func (g *Game) potAppearanceImageID(id int) (int, bool) {
	if unidentifiedNameKinds[id] != "Pot" {
		return 0, false
	}
	index, ok := g.unidentifiedAppearance(id)
	if !ok || index >= len(unidentifiedPotColors) {
		return 0, false
	}
	return synthesisPotID + index, true
}

// 識別は種類ごと（杖・壺・指輪）。1つ識別すると、同じテンプレートの道具はこの冒険の中ですべて識別済みになる。
// 武器・盾は強化値が1つずつ違うので対象外（1つずつ識別する）。
// 識別する経路（装備・識別の壺・鑑定師の指輪・カードなど）を個別に直さず、毎フレームの syncIdentifiedTemplates が
// 識別済みの道具を見つけて種類を覚え、同じ種類の道具へ広げる。覚えた種類は中断セーブに含む。

// identifiedTemplateKnown はテンプレートがこの冒険で識別済みの種類かを返す。
func (g *Game) identifiedTemplateKnown(id int) bool {
	return g.identifiedTemplates[id]
}

// forEachHeldOrPlacedItem は持ち物・装備・床・敵が持つ道具（壺の中身を含む）を順に渡す。
func (g *Game) forEachHeldOrPlacedItem(visit func(Item)) {
	var walk func(Item)
	walk = func(item Item) {
		if item == nil {
			return
		}
		visit(item)
		if pot, ok := item.(*Pot); ok {
			for _, child := range pot.Contents {
				walk(child)
			}
		}
	}
	p := &g.state.Player
	for _, item := range p.Inventory {
		walk(item)
	}
	// 装備の欄は型付きのポインタなので、nil を Item に入れずに調べる（武器・盾・矢は種類ごとの識別の対象外）
	for _, accessory := range p.EquippedAccessories {
		if accessory != nil {
			walk(accessory)
		}
	}
	for _, item := range g.state.Items {
		walk(item)
	}
	for _, enemy := range g.state.Enemies {
		walk(enemy.HeldItem)
	}
}

// syncIdentifiedTemplates は識別済みの杖・壺・指輪の種類を覚え、同じ種類の未識別の道具を識別済みにする。
func (g *Game) syncIdentifiedTemplates() {
	g.forEachHeldOrPlacedItem(func(item Item) {
		id := item.GetID()
		if _, ok := unidentifiedNameKinds[id]; !ok || g.identifiedTemplates[id] {
			return
		}
		if v, ok := item.(Identifiable); ok && v.IsIdentified() {
			if g.identifiedTemplates == nil {
				g.identifiedTemplates = map[int]bool{}
			}
			g.identifiedTemplates[id] = true
		}
	})
	if len(g.identifiedTemplates) == 0 {
		return
	}
	g.forEachHeldOrPlacedItem(func(item Item) {
		if !g.identifiedTemplates[item.GetID()] {
			return
		}
		if v, ok := item.(Identifiable); ok && !v.IsIdentified() {
			v.SetIdentified(true)
		}
	})
}

// identifiedTemplateIDs はセーブ用に識別済みの種類を昇順で返す。
func (g *Game) identifiedTemplateIDs() []int {
	ids := []int{}
	for id, known := range g.identifiedTemplates {
		if known {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	return ids
}

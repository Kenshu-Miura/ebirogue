//go:build !test

package main

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

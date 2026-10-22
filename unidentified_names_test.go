//go:build !test

package main

import (
	"strings"
	"testing"
)

// 名前の候補は未識別のテンプレートの数以上あり、重複せず、本当の道具の名前と同じものが無い。
func TestUnidentifiedNamePoolsCoverTemplates(t *testing.T) {
	realNames := map[string]bool{}
	for _, tp := range itemTemplates {
		realNames[tp.Name] = true
	}
	for kind, pool := range unidentifiedNamePools {
		ids := unidentifiedNameTemplateIDs[kind]
		if len(ids) == 0 {
			t.Fatalf("%s: 未識別のテンプレートが無い", kind)
		}
		if len(pool) < len(ids) {
			t.Fatalf("%s: 名前の候補 %d < テンプレート %d。unidentifiedNamePools に足す", kind, len(pool), len(ids))
		}
		seen := map[string]bool{}
		for _, name := range pool {
			if seen[name] || realNames[name] {
				t.Fatalf("%s: 候補 %q が重複しているか本当の名前と同じ", kind, name)
			}
			seen[name] = true
		}
	}
	if len(unidentifiedNamePools["Pot"]) != len(unidentifiedPotColors) {
		t.Fatal("壺の候補と色の画像の数がずれている")
	}
	for _, id := range []int{46} { // 最初から識別済みの保存の壺は対象外
		if _, ok := unidentifiedNameKinds[id]; ok {
			t.Fatalf("ID %d が未識別名の対象になった", id)
		}
	}
}

func unidentifiedNamesFor(g *Game) map[int]string {
	names := map[int]string{}
	for _, ids := range unidentifiedNameTemplateIDs {
		for _, id := range ids {
			names[id] = g.unidentifiedName(buildItemFromTemplate(id, 0, 0))
		}
	}
	return names
}

// 冒険ごと（シードごと）にランダムで、同じ冒険の中では種類ごとに重複せず、中断・再開（同じシード）で変わらない。
func TestUnidentifiedNamesRandomPerRun(t *testing.T) {
	g1, _ := newHeadlessGame(11, DifficultyAdvanced)
	again, _ := newHeadlessGame(11, DifficultyAdvanced)
	names := unidentifiedNamesFor(g1)
	for id, name := range unidentifiedNamesFor(again) {
		if names[id] != name {
			t.Fatalf("同じシードで名前が変わった: ID %d %q / %q", id, names[id], name)
		}
	}
	for kind, ids := range unidentifiedNameTemplateIDs {
		used := map[string]int{}
		for _, id := range ids {
			name := names[id]
			if name == itemTemplates[id].Name {
				t.Fatalf("ID %d: 本当の名前のまま", id)
			}
			if prev, dup := used[name]; dup {
				t.Fatalf("%s: ID %d と %d が同じ名前 %q", kind, prev, id, name)
			}
			used[name] = id
		}
	}
	differs := false
	for seed := int64(12); seed < 20 && !differs; seed++ {
		other, _ := newHeadlessGame(seed, DifficultyAdvanced)
		for id, name := range unidentifiedNamesFor(other) {
			if names[id] != name {
				differs = true
				break
			}
		}
	}
	if !differs {
		t.Fatal("シードを変えても名前が同じ")
	}

	// 中断セーブから再開しても同じ名前
	data := g1.buildSaveData()
	restored, _ := newHeadlessGame(99, DifficultyAdvanced)
	if err := restored.applySaveData(data); err != nil {
		t.Fatal(err)
	}
	for id, name := range unidentifiedNamesFor(restored) {
		if names[id] != name {
			t.Fatalf("再開で名前が変わった: ID %d %q → %q", id, names[id], name)
		}
	}
}

// 未識別なら割り当てた名前、識別すれば本当の名前。壺の画像は色の名前とそろう。
func TestUnidentifiedNameDisplayAndPotImage(t *testing.T) {
	g, _ := newHeadlessGame(5, DifficultyAdvanced)
	for _, id := range []int{synthesisPotID, 9, 175} {
		item := buildItemFromTemplate(id, 0, 0)
		name, identified := g.displayItemName(item)
		if identified || name != g.unidentifiedName(item) || name == item.GetName() {
			t.Fatalf("ID %d: 未識別の表示 %q", id, name)
		}
		if label, _ := g.inventoryItemLabel(item); label != name {
			t.Fatalf("ID %d: 持ち物の表示 %q, want %q", id, label, name)
		}
		item.(Identifiable).SetIdentified(true)
		if name, _ := g.displayItemName(item); !strings.HasPrefix(name, item.GetName()) {
			t.Fatalf("ID %d: 識別後の表示 %q", id, name)
		}
	}
	for _, id := range unidentifiedNameTemplateIDs["Pot"] {
		fileID, ok := g.potAppearanceImageID(id)
		if !ok {
			t.Fatalf("ID %d: 壺の画像が選べない", id)
		}
		color := unidentifiedPotColors[fileID-synthesisPotID]
		if name := g.unidentifiedName(buildItemFromTemplate(id, 0, 0)); name != color+"の壺" {
			t.Fatalf("ID %d: 名前 %q と画像の色 %s がずれている", id, name, color)
		}
	}
}

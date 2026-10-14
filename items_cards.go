//go:build !test

package main

// cardItemTemplates は追加計画（docs/agents/content-expansion-plan.md）のカード（アイテム ID 167〜174）。
// 効果は card_effects_wave3.go に置き、itemTemplates へは mergeItemTemplates でまとめられる。
var cardItemTemplates = map[int]ItemTemplate{
	167: {ID: 167, ItemType: "Card", Type: "Card", Name: "ウインナー化のカード", Description: "選んだ道具をウインナーに変える。装備中の道具は選べず、壺は中身ごと変わる。", Char: 'C', UseActions: map[string]UseAction{"UseCard": sausageCard}},
	168: {ID: 168, ItemType: "Card", Type: "Card", Name: "呪いのカード", Description: "選んだ武器・盾・矢・指輪を呪う。呪いよけの能力で防げる。", Char: 'C', UseActions: map[string]UseAction{"UseCard": curseCard}},
	169: {ID: 169, ItemType: "Card", Type: "Card", Name: "罠消しのカード", Description: "フロアにある罠をすべて消す。", Char: 'C', UseActions: map[string]UseAction{"UseCard": eraseTrapsCard}},
	170: {ID: 170, ItemType: "Card", Type: "Card", Name: "敵集合のカード", Description: "フロアの敵を近い順に自分の近くへ集める。ボスは動かない。", Char: 'C', UseActions: map[string]UseAction{"UseCard": gatherEnemiesCard}},
	171: {ID: 171, ItemType: "Card", Type: "Card", Name: "道具寄せのカード", Description: "床の道具を近い順に自分の周囲へ集める。足元の道具と貼りついた聖域のカードは動かない。", Char: 'C', UseActions: map[string]UseAction{"UseCard": gatherItemsCard}},
	172: {ID: 172, ItemType: "Card", Type: "Card", Name: "バクチのカード", Description: "半分の確率で所持金が2倍と100増える。残りの半分は所持金が0になる。", Char: 'C', UseActions: map[string]UseAction{"UseCard": gambleCard}},
	173: {ID: 173, ItemType: "Card", Type: "Card", Name: "困った時のカード", Description: "HPを全回復し、悪い状態異常・空腹・呪い・パワー低下の順に困りごとを1つ解消する。", Char: 'C', UseActions: map[string]UseAction{"UseCard": troubleCard}},
}

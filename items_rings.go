//go:build !test

package main

// 指輪は既存の未識別表示・ring.png・保存経路を共用する。
var ringItemTemplates = map[int]ItemTemplate{
	175: {ID: 175, ItemType: "Accessory", Type: "Accessory", Name: "回復の指輪", Description: "アクセサリ。満腹度が1以上なら毎ターンHPを追加で1回復。20ターンごとに満腹度が追加で1減る。2つで量が2倍。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityRegeneration}},
	176: {ID: 176, ItemType: "Accessory", Type: "Accessory", Name: "胃拡張の指輪", Description: "アクセサリ。最大満腹度が25増える。2つで50増える。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityStomachExpand}},
	177: {ID: 177, ItemType: "Accessory", Type: "Accessory", Name: "胃縮小の指輪", Description: "アクセサリ。最大満腹度が25減る。2つで50減る。必ず呪われている。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityStomachShrink, AbilityAlwaysCursed}},
	178: {ID: 178, ItemType: "Accessory", Type: "Accessory", Name: "しあわせの指輪", Description: "アクセサリ。毎ターン経験値が1増える。2つで2増える。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityHappiness}},
	179: {ID: 179, ItemType: "Accessory", Type: "Accessory", Name: "毒消しの指輪", Description: "アクセサリ。毒によるパワー低下を防ぐ。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityPoisonWard}},
	180: {ID: 180, ItemType: "Accessory", Type: "Accessory", Name: "混乱よけの指輪", Description: "アクセサリ。混乱を防ぐ。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityConfusionWard}},
	181: {ID: 181, ItemType: "Accessory", Type: "Accessory", Name: "睡眠よけの指輪", Description: "アクセサリ。睡眠を防ぐ。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilitySleepWard}},
	182: {ID: 182, ItemType: "Accessory", Type: "Accessory", Name: "錆よけの指輪", Description: "アクセサリ。武器と盾の強化値がさびで下がらなくなる。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityRustWard}},
	183: {ID: 183, ItemType: "Accessory", Type: "Accessory", Name: "呪いよけの指輪", Description: "アクセサリ。持ち物が呪われるのを防ぐ。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityCurseWard}},
	184: {ID: 184, ItemType: "Accessory", Type: "Accessory", Name: "気配察知の指輪", Description: "アクセサリ。フロアの敵をミニマップに表示する。擬態中や目潰し中は表示しない。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilitySenseEnemies}},
	185: {ID: 185, ItemType: "Accessory", Type: "Accessory", Name: "道具感知の指輪", Description: "アクセサリ。フロアの床の道具をミニマップに表示する。目潰し中は表示しない。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilitySenseItems}},
	186: {ID: 186, ItemType: "Accessory", Type: "Accessory", Name: "透視の指輪", Description: "アクセサリ。フロアの敵と床の道具をミニマップに表示する。擬態中の敵や目潰し中は表示しない。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityClairvoyance}},
	187: {ID: 187, ItemType: "Accessory", Type: "Accessory", Name: "鑑定師の指輪", Description: "アクセサリ。拾った未識別の道具を識別する。壺の中身は識別しない。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityAppraiser}},
	188: {ID: 188, ItemType: "Accessory", Type: "Accessory", Name: "浮遊の指輪", Description: "アクセサリ。罠を踏まずに通り過ぎ、隠れた罠を発見する。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityLevitation}},
	194: {ID: 194, ItemType: "Accessory", Type: "Accessory", Name: "高飛びの指輪", Description: "アクセサリ。毎ターン2%でフロアのどこかへワープする。2つで4%。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityHighJump}},
	195: {ID: 195, ItemType: "Accessory", Type: "Accessory", Name: "爆発の指輪", Description: "アクセサリ。毎ターン1/200で周囲1マスの敵を倒す。ボスには50ダメージ。自分のHPは1になる。2つで1/100。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityExplosionRing}},
	196: {ID: 196, ItemType: "Accessory", Type: "Accessory", Name: "垂れ流しの指輪", Description: "アクセサリ。毎ターン3%で未装備の持ち物を1つ落とす。2つで6%。必ず呪われている。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityLeakItems, AbilityAlwaysCursed}},
	197: {ID: 197, ItemType: "Accessory", Type: "Accessory", Name: "小銭垂れ流しの指輪", Description: "アクセサリ。毎ターン10%で所持金の1/50（最低1）を足元に落とす。2つで20%。必ず呪われている。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityLeakMoney, AbilityAlwaysCursed}},
	198: {ID: 198, ItemType: "Accessory", Type: "Accessory", Name: "魔物呼びの指輪", Description: "アクセサリ。敵の湧き間隔が1/2、2つで1/3になる。最短5ターン。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityMonsterCall}},
	199: {ID: 199, ItemType: "Accessory", Type: "Accessory", Name: "罠増しの指輪", Description: "アクセサリ。50ターンごとに隠れた罠を1つ置く。2つで2個。", Char: '!', UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityTrapIncrease}},
}

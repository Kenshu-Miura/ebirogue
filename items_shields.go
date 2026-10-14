//go:build !test

package main

// shieldItemTemplates は追加計画（docs/agents/content-expansion-plan.md）の盾（アイテム ID 148〜166）。
// 担当（wave3-B）はこの表と同じファイルの効果関数だけを追加する。itemTemplates へは mergeItemTemplates でまとめられる。
var shieldItemTemplates = map[int]ItemTemplate{
	148: {ID: 148, ItemType: "Armor", Type: "BronzeShield", Name: "青銅甲の盾", Description: "防御力が4上昇する。木甲と鉄甲の間を埋める盾。", Char: '!', DefensePower: 4, UseActions: map[string]UseAction{"ArmorEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{}},
	149: {ID: 149, ItemType: "Armor", Type: "TaikaiShield", Name: "大海の盾", Description: "防御力が10上昇する。深層の敵に備える頑丈な盾。", Char: '!', DefensePower: 10, UseActions: map[string]UseAction{"ArmorEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{}},
	150: {ID: 150, ItemType: "Armor", Type: "GoldShield", Name: "金の盾", Description: "防御力が3上昇する。さびによる強化値の低下を防ぐ。", Char: '!', DefensePower: 3, UseActions: map[string]UseAction{"ArmorEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityRustImmune}},
	151: {ID: 151, ItemType: "Armor", Type: "DragonScaleShield", Name: "竜鱗の盾", Description: "防御力が5上昇する。竜の攻撃と炎のダメージを半分にする（最低1）。", Char: '!', DefensePower: 5, UseActions: map[string]UseAction{"ArmorEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityDragonGuard}},
	152: {ID: 152, ItemType: "Armor", Type: "NigiriGuardShield", Name: "にぎりよけの盾", Description: "防御力が3上昇する。持ち物が食料に変えられるのを防ぐ。", Char: '!', DefensePower: 3, UseActions: map[string]UseAction{"ArmorEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityNigiriGuard}},
	153: {ID: 153, ItemType: "Armor", Type: "PuppetGuardShield", Name: "操りよけの盾", Description: "防御力が3上昇する。敵に操られるのを防ぐ。", Char: '!', DefensePower: 3, UseActions: map[string]UseAction{"ArmorEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityPuppetGuard}},
	154: {ID: 154, ItemType: "Armor", Type: "KobanShield", Name: "小判の盾", Description: "防御力が6上昇する。所持金20以上で近接ダメージを2/3にする（最低1）。軽減すると20消費する。", Char: '!', DefensePower: 6, UseActions: map[string]UseAction{"ArmorEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityMoneyGuard}},
	155: {ID: 155, ItemType: "Armor", Type: "IronMassShield", Name: "鉄塊の大盾", Description: "防御力が7上昇する。満腹度10以上で近接ダメージを2/3にする（最低1）。軽減すると満腹度2を消費する。", Char: '!', DefensePower: 7, UseActions: map[string]UseAction{"ArmorEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilitySatietyGuard}},
	156: {ID: 156, ItemType: "Armor", Type: "HungryShield", Name: "腹ペコの盾", Description: "防御力が5上昇する。満腹度0のとき敵と罠のダメージを半分にする（最低1）。餓死は防げない。", Char: '!', DefensePower: 5, UseActions: map[string]UseAction{"ArmorEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityHungerGuard}},
	157: {ID: 157, ItemType: "Armor", Type: "GutsShield", Name: "根性の盾", Description: "防御力が6上昇する。致命傷を1/3の確率でHP1で耐える。餓死は防げない。", Char: '!', DefensePower: 6, UseActions: map[string]UseAction{"ArmorEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityGuts}},
	158: {ID: 158, ItemType: "Armor", Type: "PeakShield", Name: "絶好調の盾", Description: "防御力が4上昇する。HP満タンのとき被ダメージを半分にする（最低1）。", Char: '!', DefensePower: 4, UseActions: map[string]UseAction{"ArmorEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityPeakGuard}},
	159: {ID: 159, ItemType: "Armor", Type: "SatoriShield", Name: "悟りの盾", Description: "防御力が2上昇する。装備中は満腹度が時間で減らないが、食べても1まで。最大満腹度は変わらない。", Char: '!', DefensePower: 2, UseActions: map[string]UseAction{"ArmorEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilitySatori}},
	160: {ID: 160, ItemType: "Armor", Type: "LuckyCharm", Name: "七福の御守", Description: "防御力が5上昇する。HPの1の位が7のとき被ダメージを半分にする（最低1）。", Char: '!', DefensePower: 5, UseActions: map[string]UseAction{"ArmorEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityLuckyGuard}},
	161: {ID: 161, ItemType: "Armor", Type: "CursedPotLid", Name: "呪いの鍋ぶた", Description: "防御力が7上昇する。この盾が呪われている間、被ダメージを4/5にする（最低1）。", Char: '!', DefensePower: 7, UseActions: map[string]UseAction{"ArmorEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityCurseGuard}},
	162: {ID: 162, ItemType: "Armor", Type: "TurtleShell", Name: "カメの甲羅", Description: "防御力が3上昇する。爆発以外の敵の遠距離攻撃を50%でかわす。", Char: '!', DefensePower: 3, UseActions: map[string]UseAction{"ArmorEffect": func(g *Game) {}}, Abilities: []EquipmentAbilityID{AbilityShellDodge}},
}

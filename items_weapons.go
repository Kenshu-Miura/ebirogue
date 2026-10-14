//go:build !test

package main

// weaponItemTemplates は追加計画（docs/agents/content-expansion-plan.md）の武器（アイテム ID 126〜147）。
// 担当（wave3-A）はこの表と同じファイルの効果関数だけを追加する。itemTemplates へは mergeItemTemplates でまとめられる。
var weaponItemTemplates = map[int]ItemTemplate{
	126: {ID: 126, ItemType: "Weapon", Type: "Katana", Name: "カタナ", AttackPower: 5, Char: '!', Description: "中盤向けの刀。攻撃力が5上昇する。", Abilities: []EquipmentAbilityID{}},
	127: {ID: 127, ItemType: "Weapon", Type: "Gouken", Name: "剛剣", AttackPower: 10, Char: '!', Description: "深層向けの大剣。攻撃力が10上昇する。", Abilities: []EquipmentAbilityID{}},
	128: {ID: 128, ItemType: "Weapon", Type: "GoldSword", Name: "金の剣", AttackPower: 3, Char: '!', Description: "さびず、通常攻撃の命中率が95%になる。", Abilities: []EquipmentAbilityID{AbilityRustImmune, AbilityKeenEdge}},
	129: {ID: 129, ItemType: "Weapon", Type: "ShellCutter", Name: "甲殻斬り", AttackPower: 5, Char: '!', Description: "甲殻の敵に1.5倍のダメージを与える。", Abilities: []EquipmentAbilityID{AbilityShellSlayer}},
	130: {ID: 130, ItemType: "Weapon", Type: "UniBreaker", Name: "ウニ割り", AttackPower: 5, Char: '!', Description: "爆発する敵に1.5倍のダメージを与える。", Abilities: []EquipmentAbilityID{AbilityExplosiveSlayer}},
	131: {ID: 131, ItemType: "Weapon", Type: "SkyBlade", Name: "空の刃", AttackPower: 4, Char: '!', Description: "浮遊する敵に1.5倍のダメージを与える。", Abilities: []EquipmentAbilityID{AbilityFloatingSlayer}},
	132: {ID: 132, ItemType: "Weapon", Type: "DoubleStrikeSword", Name: "連撃刀", AttackPower: 4, Char: '!', Description: "命中後、敵が生きていれば25%でもう1回攻撃する。", Abilities: []EquipmentAbilityID{AbilityDoubleStrike}},
	133: {ID: 133, ItemType: "Weapon", Type: "GreatAxe", Name: "大斧", AttackPower: 6, Char: '!', Description: "通常攻撃の会心率が1/8になる。", Abilities: []EquipmentAbilityID{AbilityCriticalUp}},
	134: {ID: 134, ItemType: "Weapon", Type: "HealingSword", Name: "回復の剣", AttackPower: 2, Char: '!', Description: "与えたダメージの1/4（最低1）だけHPを回復する。", Abilities: []EquipmentAbilityID{AbilityLifeSteal}},
	135: {ID: 135, ItemType: "Weapon", Type: "KobanClub", Name: "小判こん棒", AttackPower: 6, Char: '!', Description: "所持金20以上ならダメージ1.5倍。命中時に20消費する。", Abilities: []EquipmentAbilityID{AbilityMoneyStrike}},
	136: {ID: 136, ItemType: "Weapon", Type: "IronMassSword", Name: "鉄塊の大剣", AttackPower: 7, Char: '!', Description: "満腹度10以上ならダメージ1.5倍。命中時に満腹度2消費。", Abilities: []EquipmentAbilityID{AbilitySatietyStrike}},
	137: {ID: 137, ItemType: "Weapon", Type: "HungryKnife", Name: "腹ペコ包丁", AttackPower: 5, Char: '!', Description: "満腹度0のとき、通常攻撃のダメージが2倍になる。", Abilities: []EquipmentAbilityID{AbilityHungerStrike}},
	138: {ID: 138, ItemType: "Weapon", Type: "ChargeSword", Name: "溜め打ちの剣", AttackPower: 3, Char: '!', Description: "2回外れると次の通常攻撃が必中の会心になる。", Abilities: []EquipmentAbilityID{AbilityChargeStrike}},
	139: {ID: 139, ItemType: "Weapon", Type: "FourProngHarpoon", Name: "四ツ叉の銛", AttackPower: 5, Char: '!', Description: "4回連続命中すると次の通常攻撃が必中の会心になる。", Abilities: []EquipmentAbilityID{AbilityComboStrike}},
	140: {ID: 140, ItemType: "Weapon", Type: "LuckyRake", Name: "七福の熊手", AttackPower: 5, Char: '!', Description: "HPの1の位が7のとき、通常攻撃が必ず会心になる。", Abilities: []EquipmentAbilityID{AbilityLuckySeven}},
	141: {ID: 141, ItemType: "Weapon", Type: "Mallet", Name: "木づち", AttackPower: 5, Char: '!', Description: "正面に敵がいなければ罠、なければ道具を壊す。25%で強化値-1。", Abilities: []EquipmentAbilityID{AbilityMallet}},
	142: {ID: 142, ItemType: "Weapon", Type: "TrapProbe", Name: "罠探りの棒", AttackPower: 3, Char: '!', Description: "攻撃すると前方3マス幅、2マス先までの隠れた罠を見つける。", Abilities: []EquipmentAbilityID{AbilityTrapProbe}},
	143: {ID: 143, ItemType: "Weapon", Type: "Katsuobushi", Name: "かつおぶし", AttackPower: 4, Char: '!', Description: "かじると満腹度15回復、強化値-1。強化値-3以下でかじると食べきる。", Abilities: []EquipmentAbilityID{AbilityEdible}},
	144: {ID: 144, ItemType: "Weapon", Type: "CursedKnife", Name: "呪いの包丁", AttackPower: 7, Char: '!', Description: "呪われている間、通常攻撃の総攻撃力が3上がる。", Abilities: []EquipmentAbilityID{AbilityCursePower}},
	145: {ID: 145, ItemType: "Weapon", Type: "GoldenPickaxe", Name: "黄金のつるはし", AttackPower: 2, Char: '!', Description: "壁を掘れる、さびないつるはし。掘っても劣化しない。", Abilities: []EquipmentAbilityID{AbilityDigWall, AbilityRustImmune}},
	146: {ID: 146, ItemType: "Weapon", Type: "KajinTachi", Name: "火迅の太刀", AttackPower: 12, Char: '!', Description: "HP満タンで攻撃すると直線10マスの炎。最初の敵に20ダメージ。", Abilities: []EquipmentAbilityID{AbilityFlameBlade}},
}

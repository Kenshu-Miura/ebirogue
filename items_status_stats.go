//go:build !test

package main

// 既存の薬・杖の画像、未識別表示、回数（5）、カテゴリ内の重み（1）を共用する。
var statItemTemplates = map[int]ItemTemplate{
	205: {ID: 205, ItemType: "Potion", Type: "Potion", Name: "倍速の薬", Char: '!', Description: "飲むと20ターン倍速。敵に投げると敵が倍速になる。鈍足と打ち消し合う。", UseActions: map[string]UseAction{"RestoreHealth": drinkStatPotion}},
	206: {ID: 206, ItemType: "Potion", Type: "Potion", Name: "剛力の薬", Char: '!', Description: "飲むと30ターン攻撃力が1.5倍。投げると敵の攻撃力が上昇。攻撃力低下と打ち消し合う。", UseActions: map[string]UseAction{"RestoreHealth": drinkStatPotion}},
	207: {ID: 207, ItemType: "Potion", Type: "Potion", Name: "堅守の薬", Char: '!', Description: "飲むと30ターン被ダメージが0.67倍。投げると敵の防御を強化。防御力低下と打ち消し合う。", UseActions: map[string]UseAction{"RestoreHealth": drinkStatPotion}},
	208: {ID: 208, ItemType: "Potion", Type: "Potion", Name: "弱気の薬", Char: '!', Description: "飲むと30ターン攻撃力が半分。敵に投げると敵の攻撃力が低下。攻撃力上昇と打ち消し合う。", UseActions: map[string]UseAction{"RestoreHealth": drinkStatPotion}},
	209: {ID: 209, ItemType: "Potion", Type: "Potion", Name: "退化の薬", Char: '!', Description: "飲むとレベルが1低下。投げると敵が同系統の1段下へ退化。基本種とボスには無効。", UseActions: map[string]UseAction{"RestoreHealth": drinkStatPotion}},
	210: {ID: 210, ItemType: "Cane", Type: "Cane", Name: "鈍足の杖", Char: '!', Description: "敵を30ターン鈍足にする。倍速と打ち消し合う。", UseActions: map[string]UseAction{"CaneEffect": statCaneEffect}},
	211: {ID: 211, ItemType: "Cane", Type: "Cane", Name: "弱体の杖", Char: '!', Description: "敵の攻撃力を0.5倍、被ダメージを1.5倍にする。30ターン。上昇と打ち消し合う。", UseActions: map[string]UseAction{"CaneEffect": statCaneEffect}},
	212: {ID: 212, ItemType: "Cane", Type: "Cane", Name: "退化の杖", Char: '!', Description: "敵を同系統の1段下へ退化させる。基本種とボスには無効。", UseActions: map[string]UseAction{"CaneEffect": statCaneEffect}},
	213: {ID: 213, ItemType: "Cane", Type: "Cane", Name: "加速の杖", Char: '!', Description: "敵を20ターン倍速にしてしまう。不利な道具。鈍足と打ち消し合う。", UseActions: map[string]UseAction{"CaneEffect": statCaneEffect}},
}

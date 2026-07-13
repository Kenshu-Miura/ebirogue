//go:build !test

package main

import "math/rand"

type BaseItem struct {
	Entity
	ID               int
	Type             string
	Name             string
	Description      string
	UseActions       map[string]UseAction
	ShowOnMiniMap    bool
	PlayerDiscovered bool // プレイヤーによって発見されたかどうか
}

type Weapon struct {
	BaseItem
	AttackPower int
	Sharpness   int    // 例: 0-100の範囲で切れ味を表現
	Element     string // 例: "Fire", "Ice", "Electric", etc.
	Cursed      bool   // 武器が呪われているかどうか
	Identified  bool   // 武器が識別されているかどうか
}

type Armor struct {
	BaseItem
	DefensePower int
	Sharpness    int
	Element      string
	Cursed       bool
	Identified   bool // 鎧が識別されているかどうか
}

type Arrow struct {
	BaseItem
	ShotCount   int
	AttackPower int
	Cursed      bool
	Identified  bool // 矢が識別されているかどうか
}

type Food struct {
	BaseItem
	Satiety int
}

type Potion struct {
	BaseItem
	Health int
}

type Card struct {
	BaseItem
}

type Money struct {
	BaseItem
	Amount     int  // 金額を保持するフィールド
	Identified bool // お金が識別されているかどうか
}

type Accessory struct {
	BaseItem
	Cursed     bool
	Identified bool // アクセサリが識別されているかどうか
}

type Cane struct {
	BaseItem
	Uses       int  // 回数を保持するフィールド
	Identified bool // 杖が識別されているかどうか
}

type Trap struct {
	BaseItem
}

// アイテムデータテーブル用の構造体
type ItemTemplate struct {
	ID           int
	ItemType     string // "Money", "Food", "Potion", etc.
	Type         string // BaseItem.Type用
	Name         string
	Description  string
	Char         rune
	UseActions   map[string]UseAction
	AttackPower  int // 武器・矢の基礎攻撃力
	DefensePower int // 防具の基礎防御力
}

// アイテムデータテーブル
var itemTemplates = map[int]ItemTemplate{
	0: {
		ID:          0,
		ItemType:    "Money",
		Type:        "Kane",
		Name:        "小銭",
		Description: "小銭。それは海老さんが絆と呼ぶもの。",
		Char:        '!',
		UseActions:  map[string]UseAction{"UseMoney": money},
	},
	1: {
		ID:          1,
		ItemType:    "Food",
		Type:        "Sausage",
		Name:        "ウインナー",
		Description: "海老さんが配信中に食べる食事。満腹度を50回復する。",
		Char:        '!',
		UseActions:  map[string]UseAction{"RestoreSatiety": restoreSatiety50},
	},
	2: {
		ID:          2,
		ItemType:    "Potion",
		Type:        "Mintia",
		Name:        "ミンティア",
		Description: "海老さんを元気にする薬。HPを30回復する。",
		Char:        '!',
		UseActions:  map[string]UseAction{"RestoreHealth": restoreHP30},
	},
	3: {
		ID:          3,
		ItemType:    "Potion",
		Type:        "Mintia",
		Name:        "すごいミンティア",
		Description: "海老さんをすごく元気にする薬。HPを100回復する。",
		Char:        '!',
		UseActions:  map[string]UseAction{"RestoreHealth": restoreHP100},
	},
	4: {
		ID:          4,
		ItemType:    "Weapon",
		Type:        "Weapon",
		Name:        "伝説の剣",
		Description: "伝説の剣。攻撃力が8上昇する。",
		Char:        '!',
		UseActions:  map[string]UseAction{"WeaponEffect": func(g *Game) {}},
		AttackPower: 8,
	},
	5: {
		ID:           5,
		ItemType:     "Armor",
		Type:         "Armor",
		Name:         "光の角",
		Description:  "光の角。防御力が8上昇する。",
		Char:         '!',
		UseActions:   map[string]UseAction{"ArmorEffect": func(g *Game) {}},
		DefensePower: 8,
	},
	6: {
		ID:          6,
		ItemType:    "Arrow",
		Type:        "Arrow",
		Name:        "銀の弓矢",
		Description: "銀の弓矢。攻撃力が5上昇する。",
		Char:        '!',
		UseActions:  map[string]UseAction{"ArrowEffect": func(g *Game) {}},
		AttackPower: 5,
	},
	7: {
		ID:          7,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "黒炎弾のカード",
		Description: "眼の前の敵に30ダメージを与える。",
		Char:        '!',
		UseActions:  map[string]UseAction{"UseCard": damageHP30},
	},
	8: {
		ID:          8,
		ItemType:    "Trap",
		Type:        "Card",
		Name:        "炸裂装甲のカード",
		Description: "セットして使用する罠カード。攻撃を行った敵を破壊する",
		Char:        '!',
		UseActions:  map[string]UseAction{"SetTrap": setTrap},
	},
	9: {
		ID:          9,
		ItemType:    "Cane",
		Type:        "Cane",
		Name:        "シフトチェンジの杖",
		Description: "敵に当たった場合、自分と位置を交換する。",
		Char:        '!',
		UseActions:  map[string]UseAction{"CaneEffect": shiftChange},
	},
	10: {
		ID:          10,
		ItemType:    "Accessory",
		Type:        "Accessory",
		Name:        "鼓舞の指輪",
		Description: "アクセサリ。パワーの最大値が3上昇する。",
		Char:        '!',
		UseActions:  map[string]UseAction{"AccessoryEffect": func(g *Game) {}},
	},
	11: {
		ID:          11,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "真実の眼のカード",
		Description: "所持アイテムを1つ識別する。",
		Char:        '!',
		UseActions:  map[string]UseAction{"UseCard": identifyItem},
	},
	12: {
		ID:          12,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "睡眠のカード",
		Description: "同じ部屋にいる敵を全員睡眠状態にする。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": sleepAllEnemiesInRoom},
	},
	13: {
		ID:          13,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "混乱のカード",
		Description: "同じ部屋にいる敵を全員混乱状態にする。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": confuseAllEnemiesInRoom},
	},
	14: {
		ID:          14,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "目潰しのカード",
		Description: "同じ部屋にいる敵を全員目潰し状態にする。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": blindAllEnemiesInRoom},
	},
	15: {
		ID:          15,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "金縛りのカード",
		Description: "周囲8マスの敵を金縛り状態にする。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": paralyzeAllEnemiesAround},
	},
	16: {
		ID:          16,
		ItemType:    "Cane",
		Type:        "Cane",
		Name:        "封印の杖",
		Description: "敵に当たった場合、その敵を封印状態にする。",
		Char:        '!',
		UseActions:  map[string]UseAction{"CaneEffect": sealEnemy},
	},
	17: {
		ID:          17,
		ItemType:    "Potion",
		Type:        "Potion",
		Name:        "睡眠薬",
		Description: "飲むと10ターン睡眠状態になる。敵に投げることもできる。",
		Char:        '!',
		UseActions:  map[string]UseAction{"RestoreHealth": sleepPotion},
	},
	18: {
		ID:          18,
		ItemType:    "Potion",
		Type:        "Potion",
		Name:        "混乱薬",
		Description: "飲むと10ターン混乱状態になる。敵に投げることもできる。",
		Char:        '!',
		UseActions:  map[string]UseAction{"RestoreHealth": confusionPotion},
	},
	19: {
		ID:          19,
		ItemType:    "Potion",
		Type:        "Potion",
		Name:        "目潰し薬",
		Description: "飲むと30ターン目潰し状態になる。敵に投げると永続的な目潰し状態にする。",
		Char:        '!',
		UseActions:  map[string]UseAction{"RestoreHealth": blindPotion},
	},
	20: {
		ID:          20,
		ItemType:    "Weapon",
		Type:        "Weapon",
		Name:        "こん棒",
		Description: "扱いやすい木の武器。攻撃力が2上昇する。",
		Char:        '!',
		UseActions:  map[string]UseAction{"WeaponEffect": func(g *Game) {}},
		AttackPower: 2,
	},
	21: {
		ID:          21,
		ItemType:    "Weapon",
		Type:        "Weapon",
		Name:        "長巻",
		Description: "長い柄を持つ刀。攻撃力が4上昇する。",
		Char:        '!',
		UseActions:  map[string]UseAction{"WeaponEffect": func(g *Game) {}},
		AttackPower: 4,
	},
	22: {
		ID:          22,
		ItemType:    "Weapon",
		Type:        "Weapon",
		Name:        "どうたぬき",
		Description: "重く頑丈な刀。攻撃力が6上昇する。",
		Char:        '!',
		UseActions:  map[string]UseAction{"WeaponEffect": func(g *Game) {}},
		AttackPower: 6,
	},
	23: {
		ID:           23,
		ItemType:     "Armor",
		Type:         "Armor",
		Name:         "木甲の盾",
		Description:  "木を組んだ軽い盾。防御力が2上昇する。",
		Char:         '!',
		UseActions:   map[string]UseAction{"ArmorEffect": func(g *Game) {}},
		DefensePower: 2,
	},
	24: {
		ID:           24,
		ItemType:     "Armor",
		Type:         "Armor",
		Name:         "鉄甲の盾",
		Description:  "鉄で補強された盾。防御力が5上昇する。",
		Char:         '!',
		UseActions:   map[string]UseAction{"ArmorEffect": func(g *Game) {}},
		DefensePower: 5,
	},
	25: {
		ID:           25,
		ItemType:     "Armor",
		Type:         "Armor",
		Name:         "皮甲の盾",
		Description:  "軽さと守りを両立した盾。防御力が3上昇する。",
		Char:         '!',
		UseActions:   map[string]UseAction{"ArmorEffect": func(g *Game) {}},
		DefensePower: 3,
	},
}

// テーブルからアイテムを生成する共通関数
func buildItemFromTemplate(id, x, y int) Item {
	template, exists := itemTemplates[id]
	if !exists {
		// デフォルトで混乱薬を返す
		template = itemTemplates[18]
	}

	baseItem := BaseItem{
		Entity: Entity{
			X:    x,
			Y:    y,
			Char: template.Char,
		},
		ID:          template.ID,
		Type:        template.Type,
		Name:        template.Name,
		Description: template.Description,
		UseActions:  template.UseActions,
	}

	var item Item
	sharpnessValue := rand.Intn(5) - 1

	switch template.ItemType {
	case "Money":
		item = &Money{
			BaseItem:   baseItem,
			Amount:     rand.Intn(2001),
			Identified: true,
		}
	case "Food":
		satiety := 50 // デフォルト値
		item = &Food{
			BaseItem: baseItem,
			Satiety:  satiety,
		}
	case "Potion":
		var health int
		switch id {
		case 2:
			health = 30
		case 3:
			health = 100
		default:
			health = 0 // 睡眠薬・混乱薬
		}
		item = &Potion{
			BaseItem: baseItem,
			Health:   health,
		}
	case "Weapon":
		item = &Weapon{
			BaseItem:    baseItem,
			AttackPower: template.AttackPower,
			Sharpness:   sharpnessValue,
			Element:     "None",
			Cursed:      sharpnessValue == -1,
		}
	case "Armor":
		item = &Armor{
			BaseItem:     baseItem,
			DefensePower: template.DefensePower,
			Sharpness:    sharpnessValue,
			Element:      "None",
			Cursed:       sharpnessValue == -1,
		}
	case "Arrow":
		item = &Arrow{
			BaseItem:    baseItem,
			ShotCount:   rand.Intn(11) + 5,
			AttackPower: template.AttackPower,
			Cursed:      false,
			Identified:  true,
		}
	case "Card":
		item = &Card{
			BaseItem: baseItem,
		}
	case "Trap":
		item = &Trap{
			BaseItem: baseItem,
		}
	case "Cane":
		item = &Cane{
			BaseItem:   baseItem,
			Uses:       5,
			Identified: false,
		}
	case "Accessory":
		item = &Accessory{
			BaseItem:   baseItem,
			Cursed:     false,
			Identified: false,
		}
	default:
		// デフォルトは混乱薬
		item = &Potion{
			BaseItem: baseItem,
			Health:   0,
		}
	}

	return item
}

func createItem(x, y int) Item {
	randomValue := rand.Intn(len(itemTemplates))
	//randomValue := 9
	return buildItemFromTemplate(randomValue, x, y)
}

// デバッグ用：特定のIDのアイテムを生成する関数
func (g *Game) createItemByID(id int, x, y int) Item {
	return buildItemFromTemplate(id, x, y)
}

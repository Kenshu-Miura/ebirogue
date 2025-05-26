//go:build !test
// +build !test

package main

type BaseItem struct {
	Entity
	ID            int
	Type          string
	Name          string
	Description   string
	UseActions    map[string]UseAction
	ShowOnMiniMap bool
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

func createItem(x, y int) Item {
	var item Item
	randomValue := localRand.Intn(12) // Store the random value to ensure it's only generated once
	//randomValue := 9
	sharpnessValue := localRand.Intn(5) - 1
	//sharpnessValue := -1
	switch randomValue {
	case 0:
		item = &Money{
			BaseItem: BaseItem{
				Entity: Entity{
					X:    x,
					Y:    y,
					Char: '!',
				},
				ID:          0,
				Type:        "Kane",
				Name:        "小銭",
				Description: "小銭。それは海老さんが絆と呼ぶもの。",
				UseActions: map[string]UseAction{
					"UseMoney": money,
				},
			},
			Amount:     localRand.Intn(2001), // Generates a random integer between 0 and 2000
			Identified: true,
		}
	case 1:
		item = &Food{
			BaseItem: BaseItem{
				Entity: Entity{
					X:    x,
					Y:    y,
					Char: '!',
				},
				ID:          1,
				Type:        "Sausage",
				Name:        "ウインナー",
				Description: "海老さんが配信中に食べる食事。満腹度を50回復する。",
				UseActions: map[string]UseAction{
					"RestoreSatiety": restoreSatiety50,
				},
			},
			Satiety: 50,
		}
	case 2:
		item = &Potion{
			BaseItem: BaseItem{
				Entity: Entity{
					X:    x,
					Y:    y,
					Char: '!',
				},
				ID:          2,
				Type:        "Mintia",
				Name:        "ミンティア",
				Description: "海老さんを元気にする薬。HPを30回復する。",
				UseActions: map[string]UseAction{
					"RestoreHealth": restoreHP30,
				},
			},
			Health: 30,
		}
	case 3:
		item = &Potion{
			BaseItem: BaseItem{
				Entity: Entity{
					X:    x,
					Y:    y,
					Char: '!',
				},
				ID:          3,
				Type:        "Mintia",
				Name:        "すごいミンティア",
				Description: "海老さんをすごく元気にする薬。HPを100回復する。",
				UseActions: map[string]UseAction{
					"RestoreHealth": restoreHP100,
				},
			},
			Health: 100,
		}
	case 4:
		item = &Weapon{
			BaseItem: BaseItem{
				Entity: Entity{
					X:    x,
					Y:    y,
					Char: '!',
				},
				ID:          4,
				Type:        "Weapon",
				Name:        "伝説の剣",
				Description: "伝説の剣。攻撃力が8上昇する。",
				UseActions: map[string]UseAction{
					"WeaponEffect": func(g *Game) {
					},
				},
			},
			AttackPower: 8,
			Sharpness:   sharpnessValue,
			Element:     "None",
			Cursed:      sharpnessValue == -1,
		}
	case 5:
		item = &Armor{
			BaseItem: BaseItem{
				Entity: Entity{
					X:    x,
					Y:    y,
					Char: '!',
				},
				ID:          5,
				Type:        "Armor",
				Name:        "光の角",
				Description: "光の角。防御力が8上昇する。",
				UseActions: map[string]UseAction{
					"ArmorEffect": func(g *Game) {
					},
				},
			},
			DefensePower: 8,
			Sharpness:    sharpnessValue,
			Element:      "None",
			Cursed:       sharpnessValue == -1,
		}

	case 6:
		item = &Arrow{
			BaseItem: BaseItem{
				Entity: Entity{
					X:    x,
					Y:    y,
					Char: '!',
				},
				ID:          6,
				Type:        "Arrow",
				Name:        "銀の弓矢",
				Description: "銀の弓矢。攻撃力が5上昇する。",
				UseActions: map[string]UseAction{
					"ArrowEffect": func(g *Game) {
					},
				},
			},
			ShotCount:   localRand.Intn(11) + 5, // Generates a random number between 5 and 15
			AttackPower: 5,
			Cursed:      false,
			Identified:  true,
		}

	case 7:
		item = &Card{
			BaseItem: BaseItem{
				Entity: Entity{
					X:    x,
					Y:    y,
					Char: '!',
				},
				ID:          7,
				Type:        "Card",
				Name:        "黒炎弾のカード",
				Description: "眼の前の敵に30ダメージを与える。",
				UseActions: map[string]UseAction{
					"UseCard": damageHP30,
				},
			},
		}

	case 8:
		item = &Trap{
			BaseItem: BaseItem{
				Entity: Entity{
					X:    x,
					Y:    y,
					Char: '!',
				},
				ID:          8,
				Type:        "Card",
				Name:        "炸裂装甲のカード",
				Description: "セットして使用する罠カード。攻撃を行った敵を破壊する",
				UseActions: map[string]UseAction{
					"SetTrap": setTrap,
				},
			},
		}

	case 9:
		item = &Cane{
			BaseItem: BaseItem{
				Entity: Entity{
					X:    x,
					Y:    y,
					Char: '!',
				},
				ID:          9,
				Type:        "Cane",
				Name:        "シフトチェンジの杖",
				Description: "敵に当たった場合、自分と位置を交換する。",
				UseActions: map[string]UseAction{
					"CaneEffect": shiftChange,
				},
			},
			Uses:       5,
			Identified: false,
		}
	case 10:
		item = &Accessory{
			BaseItem: BaseItem{
				Entity: Entity{
					X:    x,
					Y:    y,
					Char: '!',
				},
				ID:          10,
				Type:        "Accessory",
				Name:        "鼓舞の指輪",
				Description: "アクセサリ。パワーの最大値が3上昇する。",
				UseActions: map[string]UseAction{
					"AccessoryEffect": func(g *Game) {
					},
				},
			},
			Cursed:     false,
			Identified: false,
		}
	case 11:
		item = &Card{
			BaseItem: BaseItem{
				Entity: Entity{
					X:    x,
					Y:    y,
					Char: '!',
				},
				ID:          11,
				Type:        "Card",
				Name:        "真実の眼のカード",
				Description: "所持アイテムを1つ識別する。",
				UseActions: map[string]UseAction{
					"UseCard": identifyItem,
				},
			},
		}
	}
	return item
}

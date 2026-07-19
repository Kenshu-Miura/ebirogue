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
	Sharpness   int                  // 例: 0-100の範囲で切れ味を表現
	Element     string               // 例: "Fire", "Ice", "Electric", etc.
	Cursed      bool                 // 武器が呪われているかどうか
	Identified  bool                 // 武器が識別されているかどうか
	Abilities   []EquipmentAbilityID // 装備名に依存しない能力ID
}

type Armor struct {
	BaseItem
	DefensePower int
	Sharpness    int
	Element      string
	Cursed       bool
	Identified   bool                 // 鎧が識別されているかどうか
	Abilities    []EquipmentAbilityID // 装備名に依存しない能力ID
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
	Satiety      int
	FullRecovery bool
	MaxStatBonus int
}

type Potion struct {
	BaseItem
	Health       int
	FullRecovery bool
	MaxStatBonus int
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
	Abilities    []EquipmentAbilityID
	Recovery     int  // 満腹度またはHPの回復量
	FullRecovery bool // 現在の最大値まで回復するか
	MaxStatBonus int  // 満タン時に使った場合の最大値上昇量
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
		ID:           1,
		ItemType:     "Food",
		Type:         "Sausage",
		Name:         "ウインナー",
		Description:  "海老さんが配信中に食べる食事。満腹度を50回復する。",
		Char:         '!',
		UseActions:   map[string]UseAction{"RestoreSatiety": restoreSatiety},
		Recovery:     50,
		MaxStatBonus: 1,
	},
	2: {
		ID:           2,
		ItemType:     "Potion",
		Type:         "Mintia",
		Name:         "ミンティア",
		Description:  "海老さんを元気にする薬。HPを30回復する。",
		Char:         '!',
		UseActions:   map[string]UseAction{"RestoreHealth": restoreHP},
		Recovery:     30,
		MaxStatBonus: 1,
	},
	3: {
		ID:           3,
		ItemType:     "Potion",
		Type:         "Mintia",
		Name:         "すごいミンティア",
		Description:  "海老さんをすごく元気にする薬。HPを100回復する。",
		Char:         '!',
		UseActions:   map[string]UseAction{"RestoreHealth": restoreHP},
		Recovery:     100,
		MaxStatBonus: 2,
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
		Description: "所持アイテムを1つ識別する。まれに持ち物すべてを識別する。",
		Char:        '!',
		UseActions:  map[string]UseAction{"UseCard": identifyItem},
	},
	12: {
		ID:          12,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "睡眠のカード",
		Description: "同じ部屋にいる敵を全員睡眠状態にする。目を覚ました敵はしばらく倍速で動く。",
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
		Description:  "軽さと守りを両立した盾。防御力が3上昇し、満腹度が減りにくくなる。",
		Char:         '!',
		UseActions:   map[string]UseAction{"ArmorEffect": func(g *Game) {}},
		DefensePower: 3,
		Abilities:    []EquipmentAbilityID{AbilitySatietyConservation},
	},
	26: {
		ID:           26,
		ItemType:     "Food",
		Type:         "Sausage",
		Name:         "ジャンボウインナー",
		Description:  "食べ応えのある大きなウインナー。満腹度を100回復する。",
		Char:         '!',
		UseActions:   map[string]UseAction{"RestoreSatiety": restoreSatiety},
		Recovery:     100,
		MaxStatBonus: 2,
	},
	27: {
		ID:           27,
		ItemType:     "Food",
		Type:         "Sausage",
		Name:         "海老天むす",
		Description:  "海老天がはみ出す特製天むす。満腹度を最大まで回復する。",
		Char:         '!',
		UseActions:   map[string]UseAction{"RestoreSatiety": restoreSatiety},
		FullRecovery: true,
		MaxStatBonus: 3,
	},
	28: {
		ID:           28,
		ItemType:     "Potion",
		Type:         "Mintia",
		Name:         "大粒ミンティア",
		Description:  "大粒で効き目の強いミンティア。HPを60回復する。",
		Char:         '!',
		UseActions:   map[string]UseAction{"RestoreHealth": restoreHP},
		Recovery:     60,
		MaxStatBonus: 1,
	},
	29: {
		ID:           29,
		ItemType:     "Potion",
		Type:         "Mintia",
		Name:         "海老印の栄養ドリンク",
		Description:  "海老印の滋養たっぷりな飲み物。HPを最大まで回復する。",
		Char:         '!',
		UseActions:   map[string]UseAction{"RestoreHealth": restoreHP},
		FullRecovery: true,
		MaxStatBonus: 2,
	},
	30: {
		ID:          30,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "あかりのカード",
		Description: "フロアの地形と敵、アイテムの位置を明らかにする。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": revealFloor},
	},
	31: {
		ID:          31,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "真空斬りのカード",
		Description: "同じ部屋の敵全員に12～24ダメージを与える。通路では周囲1マスに有効。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": vacuumSlash},
	},
	32: {
		ID:          32,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "おはらいのカード",
		Description: "装備品と所持品の呪いをすべて解く。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": removeCurse},
	},
	33: {
		ID:          33,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "武器強化のカード",
		Description: "装備中の武器の強化値を1上げる。呪われていた場合は呪いも解く。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": reinforceWeapon},
	},
	34: {
		ID:          34,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "盾強化のカード",
		Description: "装備中の盾の強化値を1上げる。呪われていた場合は呪いも解く。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": reinforceArmor},
	},
	35: {
		ID:          35,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "モンスターハウスのカード",
		Description: "今いる部屋がモンスターハウスになってしまう。通路では効果がない。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": summonMonsterHouse},
	},
	36: {
		ID:          36,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "敵倍速のカード",
		Description: "フロアにいる敵全員がしばらく倍速で行動するようになってしまう。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": hasteAllEnemies},
	},
	37: {
		ID:          37,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "地図忘却のカード",
		Description: "フロアの地図を忘れてしまい、ミニマップの敵やアイテムの表示も消える。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": forgetFloorMap},
	},
	38: {
		ID:          38,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "拾得禁止のカード",
		Description: "フロアを移るまで、床に落ちているアイテムを拾えなくなってしまう。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": banItemPickup},
	},
	39: {
		ID:          39,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "大部屋のカード",
		Description: "壁が崩れ、フロア全体がひとつの大部屋になる。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": expandFloorToBigRoom},
	},
	40: {
		ID:          40,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "罠のカード",
		Description: "フロアの罠が増えてしまう。増えた罠は見えない。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": increaseFloorTraps},
	},
	41: {
		ID:          41,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "自爆のカード",
		Description: "大爆発が起こり、自分のHPが半分になるが、周囲1マスの敵を消し飛ばす。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": selfDestruct},
	},
	42: {
		ID:          42,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "口封じのカード",
		Description: "しばらくの間口が開かなくなり、カード・薬・食料を使えなくなってしまう。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": sealPlayerMouth},
	},
	43: {
		ID:          43,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "パワーアップのカード",
		Description: "パワーが3上昇する。パワーが満タンのときは最大パワーが1上昇する。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": powerUpCard},
	},
	44: {
		ID:          44,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "完全回復のカード",
		Description: "HPが完全に回復し、毒も治る。HPが満タンのときは最大HPが5上昇する。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": fullHealCard},
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
		item = &Food{
			BaseItem:     baseItem,
			Satiety:      template.Recovery,
			FullRecovery: template.FullRecovery,
			MaxStatBonus: template.MaxStatBonus,
		}
	case "Potion":
		item = &Potion{
			BaseItem:     baseItem,
			Health:       template.Recovery,
			FullRecovery: template.FullRecovery,
			MaxStatBonus: template.MaxStatBonus,
		}
	case "Weapon":
		item = &Weapon{
			BaseItem:    baseItem,
			AttackPower: template.AttackPower,
			Sharpness:   sharpnessValue,
			Element:     "None",
			Cursed:      sharpnessValue == -1,
			Abilities:   append([]EquipmentAbilityID(nil), template.Abilities...),
		}
	case "Armor":
		item = &Armor{
			BaseItem:     baseItem,
			DefensePower: template.DefensePower,
			Sharpness:    sharpnessValue,
			Element:      "None",
			Cursed:       sharpnessValue == -1,
			Abilities:    append([]EquipmentAbilityID(nil), template.Abilities...),
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

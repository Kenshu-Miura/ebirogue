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
	RustProof   bool                 // さび止め済みで強化値が下がらないかどうか
	Abilities   []EquipmentAbilityID // 装備名に依存しない能力ID
}

type Armor struct {
	BaseItem
	DefensePower int
	Sharpness    int
	Element      string
	Cursed       bool
	Identified   bool                 // 鎧が識別されているかどうか
	RustProof    bool                 // さび止め済みで強化値が下がらないかどうか
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
	Stuck bool // 聖域のカードを床に置いて貼りついた状態かどうか
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

type Pot struct {
	BaseItem
	Capacity   int    // 中身を入れられる最大数
	Contents   []Item // 壺の中身
	Identified bool   // 壺が識別されているかどうか
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
	45: {
		ID:          45,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "さび止めのカード",
		Description: "装備中の武器と盾がサビの罠などで錆びなくなり、強化値が下がらなくなる。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": rustProofCard},
	},
	46: {
		ID:          46,
		ItemType:    "Pot",
		Type:        "Pot",
		Name:        "保存の壺",
		Description: "アイテムを入れて持ち運べる壺。中身は自由に出し入れできる。投げると割れて中身が飛び出す。",
		Char:        '!',
		UseActions:  map[string]UseAction{"PotEffect": func(g *Game) {}},
	},
	47: {
		ID:          47,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "壺拡大のカード",
		Description: "持っている壺すべての容量を1増やす。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": expandPotsCard},
	},
	48: {
		ID:          48,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "吸い出しのカード",
		Description: "持っている壺すべての中身を壺を壊さずに取り出す。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": suckOutPotsCard},
	},
	49: {
		ID:          49,
		ItemType:    "Weapon",
		Type:        "DragonKiller",
		Name:        "ドラゴンキラー",
		Description: "竜の鱗を断つ大剣。攻撃力が5上昇し、ドラゴン系への通常攻撃が強くなる。",
		Char:        '!',
		UseActions:  map[string]UseAction{"WeaponEffect": func(g *Game) {}},
		AttackPower: 5,
		Abilities:   []EquipmentAbilityID{AbilityDragonSlayer},
	},
	50: {
		ID:          50,
		ItemType:    "Weapon",
		Type:        "ExorcismSickle",
		Name:        "成仏の鎌",
		Description: "霊を現世から解き放つ鎌。攻撃力が4上昇し、ゴースト系への通常攻撃が強くなる。",
		Char:        '!',
		UseActions:  map[string]UseAction{"WeaponEffect": func(g *Game) {}},
		AttackPower: 4,
		Abilities:   []EquipmentAbilityID{AbilityGhostSlayer},
	},
	51: {
		ID:          51,
		ItemType:    "Weapon",
		Type:        "OneEyeKiller",
		Name:        "一ツ目殺し",
		Description: "一ツ目の急所を狙いやすい刀。攻撃力が4上昇し、一ツ目系への通常攻撃が強くなる。",
		Char:        '!',
		UseActions:  map[string]UseAction{"WeaponEffect": func(g *Game) {}},
		AttackPower: 4,
		Abilities:   []EquipmentAbilityID{AbilityOneEyeSlayer},
	},
	52: {
		ID:          52,
		ItemType:    "Weapon",
		Type:        "DrainBuster",
		Name:        "ドレインバスター",
		Description: "弱体化の力を打ち払う剣。攻撃力が4上昇し、能力低下系への通常攻撃が強くなる。",
		Char:        '!',
		UseActions:  map[string]UseAction{"WeaponEffect": func(g *Game) {}},
		AttackPower: 4,
		Abilities:   []EquipmentAbilityID{AbilityDrainerSlayer},
	},
	53: {
		ID:          53,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "白紙のカード",
		Description: "何も書かれていないカード。使うと好きなカードの効果を書き込める。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": blankCard},
	},
	54: {
		ID:          54,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "ジェノサイドのカード",
		Description: "敵に投げ当てると、その敵と同系統の敵をフロアから消し去り、以後出現しなくなる。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": genocideCardHint},
	},
	55: {
		ID:          55,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "聖域のカード",
		Description: "床に置くと貼りつき、その上にいる間は敵から攻撃されなくなる。貼りつくと拾えない。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": sanctuaryCardHint},
	},
	56: {
		ID:          56,
		ItemType:    "Card",
		Type:        "Card",
		Name:        "全滅のカード",
		Description: "同じ部屋にいる敵をすべて消し去る。消し去った敵の経験値は入らない。",
		Char:        'C',
		UseActions:  map[string]UseAction{"UseCard": annihilateEnemiesInRoom},
	},
	57: {
		ID:          57,
		ItemType:    "Weapon",
		Type:        "SweepingNaginata",
		Name:        "海老薙刀",
		Description: "大きく薙ぎ払える長柄武器。攻撃力が4上昇し、正面と左右斜めの3方向を同時に攻撃する。",
		Char:        '!',
		UseActions:  map[string]UseAction{"WeaponEffect": func(g *Game) {}},
		AttackPower: 4,
		Abilities:   []EquipmentAbilityID{AbilityThreeWayAttack},
	},
	58: {
		ID:          58,
		ItemType:    "Weapon",
		Type:        "SureHitSword",
		Name:        "必中の剣",
		Description: "狙いがぶれない細身の剣。攻撃力が3上昇し、通常攻撃が必ず命中する。",
		Char:        '!',
		UseActions:  map[string]UseAction{"WeaponEffect": func(g *Game) {}},
		AttackPower: 3,
		Abilities:   []EquipmentAbilityID{AbilitySureHit},
	},
	59: {
		ID:          59,
		ItemType:    "Weapon",
		Type:        "ShrimpPickaxe",
		Name:        "海老つるはし",
		Description: "岩盤も砕く頑丈なつるはし。攻撃力が2上昇し、攻撃すると正面の壁を掘れる。",
		Char:        '!',
		UseActions:  map[string]UseAction{"WeaponEffect": func(g *Game) {}},
		AttackPower: 2,
		Abilities:   []EquipmentAbilityID{AbilityDigWall},
	},
	60: {
		ID:          60,
		ItemType:    "Weapon",
		Type:        "DisposableSword",
		Name:        "使い捨ての大剣",
		Description: "攻撃力が12上昇する強力な大剣。攻撃するたびに基礎攻撃力が1低下する。",
		Char:        '!',
		UseActions:  map[string]UseAction{"WeaponEffect": func(g *Game) {}},
		AttackPower: 12,
		Abilities:   []EquipmentAbilityID{AbilityDisposable},
	},
	61: {
		ID:           61,
		ItemType:     "Armor",
		Type:         "BlastGuardShield",
		Name:         "爆風よけの盾",
		Description:  "爆発の衝撃を逃がす盾。防御力が4上昇し、爆発で受けるダメージを半減する。",
		Char:         '!',
		UseActions:   map[string]UseAction{"ArmorEffect": func(g *Game) {}},
		DefensePower: 4,
		Abilities:    []EquipmentAbilityID{AbilityExplosionResistance},
	},
	62: {
		ID:           62,
		ItemType:     "Armor",
		Type:         "FlameGuardShield",
		Name:         "火喰いの盾",
		Description:  "熱を吸い込む耐火盾。防御力が4上昇し、炎で受けるダメージを半減する。",
		Char:         '!',
		UseActions:   map[string]UseAction{"ArmorEffect": func(g *Game) {}},
		DefensePower: 4,
		Abilities:    []EquipmentAbilityID{AbilityFireResistance},
	},
	63: {
		ID:           63,
		ItemType:     "Armor",
		Type:         "MagicGuardShield",
		Name:         "魔封じの盾",
		Description:  "術式を弱める紋章盾。防御力が4上昇し、魔法で受けるダメージを半減する。",
		Char:         '!',
		UseActions:   map[string]UseAction{"ArmorEffect": func(g *Game) {}},
		DefensePower: 4,
		Abilities:    []EquipmentAbilityID{AbilityMagicResistance},
	},
	64: {
		ID:           64,
		ItemType:     "Armor",
		Type:         "TheftGuardShield",
		Name:         "守銭の盾",
		Description:  "持ち物をがっちり守る盾。防御力が3上昇し、敵の盗みを完全に防ぐ。",
		Char:         '!',
		UseActions:   map[string]UseAction{"ArmorEffect": func(g *Game) {}},
		DefensePower: 3,
		Abilities:    []EquipmentAbilityID{AbilityTheftResistance},
	},
	65: {
		ID:           65,
		ItemType:     "Armor",
		Type:         "StatusGuardShield",
		Name:         "清めの盾",
		Description:  "澄んだ潮の力を宿す盾。防御力が3上昇し、敵や罠から受ける状態異常を50%で防ぐ。",
		Char:         '!',
		UseActions:   map[string]UseAction{"ArmorEffect": func(g *Game) {}},
		DefensePower: 3,
		Abilities:    []EquipmentAbilityID{AbilityStatusResistance},
	},
	66: {
		ID:           66,
		ItemType:     "Armor",
		Type:         "EvasionShield",
		Name:         "見切りの盾",
		Description:  "敵の動きを読む軽い盾。防御力が2上昇し、敵の攻撃を25%で回避する。",
		Char:         '!',
		UseActions:   map[string]UseAction{"ArmorEffect": func(g *Game) {}},
		DefensePower: 2,
		Abilities:    []EquipmentAbilityID{AbilityEvasion},
	},
	67: {
		ID:           67,
		ItemType:     "Armor",
		Type:         "ReflectionShield",
		Name:         "跳ね返しの盾",
		Description:  "飛来物を敵へ返す鏡面盾。防御力が3上昇し、矢・石・炎・魔法弾を反射する。爆発は反射できない。",
		Char:         '!',
		UseActions:   map[string]UseAction{"ArmorEffect": func(g *Game) {}},
		DefensePower: 3,
		Abilities:    []EquipmentAbilityID{AbilityReflection},
	},
	68: {
		ID:           68,
		ItemType:     "Armor",
		Type:         "CounterShield",
		Name:         "海老返しの盾",
		Description:  "受けた勢いを返す頑丈な盾。防御力が5上昇し、敵の通常攻撃ダメージの半分を返す。",
		Char:         '!',
		UseActions:   map[string]UseAction{"ArmorEffect": func(g *Game) {}},
		DefensePower: 5,
		Abilities:    []EquipmentAbilityID{AbilityCounter},
	},
	69: {
		ID:           69,
		ItemType:     "Armor",
		Type:         "HeavyShield",
		Name:         "重甲の盾",
		Description:  "圧倒的に重い大盾。防御力が12上昇するが、満腹度が5ターンごとに減る。",
		Char:         '!',
		UseActions:   map[string]UseAction{"ArmorEffect": func(g *Game) {}},
		DefensePower: 12,
		Abilities:    []EquipmentAbilityID{AbilitySatietyHunger},
	},
	70: {
		ID:           70,
		ItemType:     "Armor",
		Type:         "DisposableShield",
		Name:         "使い捨ての盾",
		Description:  "防御力が15上昇する強力な盾。ダメージを受けるたびに強化値と防御力が1低下する。",
		Char:         '!',
		UseActions:   map[string]UseAction{"ArmorEffect": func(g *Game) {}},
		DefensePower: 15,
		Abilities:    []EquipmentAbilityID{AbilityDisposableArmor},
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
	case "Pot":
		item = &Pot{
			BaseItem:   baseItem,
			Capacity:   rand.Intn(3) + 3, // 容量3〜5
			Identified: true,
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

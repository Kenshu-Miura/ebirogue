package main

type Item interface {
	GetType() string
	GetName() string
	GetDescription() string
	GetChar() rune
	GetPosition() (int, int)
	SetPosition(x, y int)
	// 他にも共通のメソッドがあればここに追加します。
}

type BaseItem struct {
	Entity
	ID          int
	Type        string
	Name        string
	Description string
}

func (bi BaseItem) GetID() int {
	return bi.ID
}

func (bi BaseItem) GetType() string {
	return bi.Type
}

func (bi BaseItem) GetName() string {
	return bi.Name
}

func (bi BaseItem) GetDescription() string {
	return bi.Description
}

func (bi BaseItem) GetChar() rune {
	return bi.Char
}

func (bi BaseItem) GetPosition() (int, int) {
	return bi.X, bi.Y
}

func (bi *BaseItem) SetPosition(x, y int) {
	bi.X, bi.Y = x, y
}

type Weapon struct {
	BaseItem
	AttackPower int
}

type Armor struct {
	BaseItem
	DefensePower int
}

type Arrow struct {
	BaseItem
	ShotCount int
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
}

func createItem(x, y int) Item {
	var item Item
	randomValue := localRand.Intn(4) // Store the random value to ensure it's only generated once
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
			},
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
				Description: "海老さんが配信中に食べる食事。",
			},
		}
	case 2:
		item = &Food{
			BaseItem: BaseItem{
				Entity: Entity{
					X:    x,
					Y:    y,
					Char: '!',
				},
				ID:          2,
				Type:        "Mintia",
				Name:        "ミンティア",
				Description: "海老さんを元気にする薬。",
			},
		}
	default:
		item = &Card{
			BaseItem: BaseItem{
				Entity: Entity{
					X:    x,
					Y:    y,
					Char: '!',
				},
				ID:          3,
				Type:        "Card",
				Name:        "カード",
				Description: "遊戯王カード。",
			},
		}
	}

	return item
}

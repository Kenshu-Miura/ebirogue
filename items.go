package main

type UseAction func(g *Game)

type Item interface {
	GetType() string
	GetName() string
	GetDescription() string
	GetChar() rune
	GetPosition() (int, int)
	SetPosition(x, y int)
	Use(g *Game) // Now, Use will call the appropriate function from the UseActions map
	// 他にも共通のメソッドがあればここに追加します。
}

type BaseItem struct {
	Entity
	ID          int
	Type        string
	Name        string
	Description string
	UseActions  map[string]UseAction
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

func (f *Food) Use(g *Game) {
	if action, exists := f.UseActions["RestoreSatiety"]; exists {
		action(g)
	}
}

var restoreSatiety50 = func(g *Game) {
	// logic to restore 50 satiety
}

type Potion struct {
	BaseItem
	Health int
}

func (p *Potion) Use(g *Game) {
	if action, exists := p.UseActions["RestoreHealth"]; exists {
		action(g)
	}
}

var restoreHP30 = func(g *Game) {
	// logic to restore 30 HP
}

var restoreHP100 = func(g *Game) {
	// logic to restore 100 HP
}

type Card struct {
	BaseItem
}

func (c *Card) Use(g *Game) {
	if action, exists := c.UseActions["UseCard"]; exists {
		action(g)
	}
}

var damageHP30 = func(g *Game) {
	// logic to restore 30 HP
}

type Money struct {
	BaseItem
}

func (m *Money) Use(g *Game) {
	if action, exists := m.UseActions["UseMoney"]; exists {
		action(g)
	}
}

var money30 = func(g *Game) {
	// logic to restore 30 HP
}

type Ring struct {
	BaseItem
}

type Cane struct {
	BaseItem
}

func createItem(x, y int) Item {
	var item Item
	randomValue := localRand.Intn(5) // Store the random value to ensure it's only generated once
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
					"UseMoney": money30,
				},
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
	default:
		item = &Card{
			BaseItem: BaseItem{
				Entity: Entity{
					X:    x,
					Y:    y,
					Char: '!',
				},
				ID:          4,
				Type:        "Card",
				Name:        "カード",
				Description: "遊戯王カード。眼の前の敵に30ダメージを与える。",
				UseActions: map[string]UseAction{
					"UseCard": damageHP30,
				},
			},
		}
	}

	return item
}

package main

import "fmt"

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
	Sharpness   int    // 例: 0-100の範囲で切れ味を表現
	Element     string // 例: "Fire", "Ice", "Electric", etc.
	Cursed      bool   // 武器が呪われているかどうか
}

func (c *Weapon) Use(g *Game) {
	if action, exists := c.UseActions["WeaponEffect"]; exists {
		action(g)
	}
}

type Armor struct {
	BaseItem
	DefensePower int
	Sharpness    int
	Element      string
	Cursed       bool
}

func (c *Armor) Use(g *Game) {
	if action, exists := c.UseActions["ArmorEffect"]; exists {
		action(g)
	}
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
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを食べた", g.state.Player.Inventory[g.selectedItemIndex].GetName()),
		Execute: func(g *Game) {
		},
	}
	g.Enqueue(action)
	if g.state.Player.Satiety == g.state.Player.MaxSatiety {
		action := Action{
			Duration: 0.4,
			Message:  "最大満腹度が1上昇した。",
			Execute: func(g *Game) {
				g.state.Player.MaxSatiety++
			},
		}
		g.Enqueue(action)
	} else {
		foodItem := g.state.Player.Inventory[g.selectedItemIndex].(*Food) // Assumes item is of type *Food
		action := Action{
			Duration: 0.4,
			Message:  fmt.Sprintf("満腹度が%d回復した。", foodItem.Satiety),
			Execute: func(g *Game) {
				g.state.Player.Satiety += foodItem.Satiety
				if g.state.Player.Satiety > g.state.Player.MaxSatiety {
					g.state.Player.Satiety = g.state.Player.MaxSatiety
				}
			},
		}
		g.Enqueue(action)
	}
	g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)
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
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを食べた", g.state.Player.Inventory[g.selectedItemIndex].GetName()),
		Execute: func(g *Game) {
		},
	}
	g.Enqueue(action)
	if g.state.Player.Health == g.state.Player.MaxHealth {
		action := Action{
			Duration: 0.4,
			Message:  "最大HPが1上昇した。",
			Execute: func(g *Game) {
				g.state.Player.MaxHealth++
			},
		}
		g.Enqueue(action)
	} else {
		potionItem := g.state.Player.Inventory[g.selectedItemIndex].(*Potion) // Assumes item is of type *Food
		action := Action{
			Duration: 0.4,
			Message:  fmt.Sprintf("HPが%d回復した。", potionItem.Health),
			Execute: func(g *Game) {
				g.state.Player.Health += potionItem.Health
				if g.state.Player.Health > g.state.Player.MaxHealth {
					g.state.Player.Health = g.state.Player.MaxHealth
				}
			},
		}
		g.Enqueue(action)
	}
	g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)
}

var restoreHP100 = func(g *Game) {
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを食べた", g.state.Player.Inventory[g.selectedItemIndex].GetName()),
		Execute: func(g *Game) {
		},
	}
	g.Enqueue(action)
	if g.state.Player.Health == g.state.Player.MaxHealth {
		action := Action{
			Duration: 0.4,
			Message:  "最大HPが2上昇した。",
			Execute: func(g *Game) {
				g.state.Player.MaxHealth += 2
			},
		}
		g.Enqueue(action)
	} else {
		potionItem := g.state.Player.Inventory[g.selectedItemIndex].(*Potion) // Assumes item is of type *Food
		action := Action{
			Duration: 0.4,
			Message:  fmt.Sprintf("HPが%d回復した。", potionItem.Health),
			Execute: func(g *Game) {
				g.state.Player.Health += potionItem.Health
				if g.state.Player.Health > g.state.Player.MaxHealth {
					g.state.Player.Health = g.state.Player.MaxHealth
				}
			},
		}
		g.Enqueue(action)
	}
	g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)
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
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使った。", g.state.Player.Inventory[g.selectedItemIndex].GetName()),
		Execute: func(g *Game) {
		},
	}
	g.Enqueue(action)

	action = Action{
		Duration: 0.4,
		Message:  "",
		Execute: func(g *Game) {
			var targetX, targetY int
			switch g.state.Player.Direction {
			case Up:
				targetX, targetY = g.state.Player.X, g.state.Player.Y-1
			case Down:
				targetX, targetY = g.state.Player.X, g.state.Player.Y+1
			case Left:
				targetX, targetY = g.state.Player.X-1, g.state.Player.Y
			case Right:
				targetX, targetY = g.state.Player.X+1, g.state.Player.Y
			case UpRight:
				targetX, targetY = g.state.Player.X+1, g.state.Player.Y-1
			case DownRight:
				targetX, targetY = g.state.Player.X+1, g.state.Player.Y+1
			case UpLeft:
				targetX, targetY = g.state.Player.X-1, g.state.Player.Y-1
			case DownLeft:
				targetX, targetY = g.state.Player.X-1, g.state.Player.Y+1
			}
			for i, enemy := range g.state.Enemies {
				if enemy.X == targetX && enemy.Y == targetY {
					action := Action{
						Duration: 0.5,
						Message:  fmt.Sprintf("%sに30ダメージを与えた。", g.state.Enemies[i].Name),
						Execute: func(g *Game) {
							g.state.Enemies[i].Health -= 30
							if g.state.Enemies[i].Health <= 0 {
								// 敵のHealthが0以下の場合、敵を配列から削除
								defeatAction := Action{
									Duration: 0.5,
									Message:  fmt.Sprintf("%sを倒した。", g.state.Enemies[i].Name),
									Execute:  func(g *Game) {},
								}
								g.Enqueue(defeatAction)

								g.state.Enemies = append(g.state.Enemies[:i], g.state.Enemies[i+1:]...)

								// 敵の経験値をプレイヤーの所持経験値に加える
								g.state.Player.ExperiencePoints += enemy.ExperiencePoints

								g.state.Player.checkLevelUp() // レベルアップをチェック
							}
						},
					}
					g.Enqueue(action)
					break
				}
			}
		},
	}
	g.Enqueue(action)
	g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)
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
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使った。", g.state.Player.Inventory[g.selectedItemIndex].GetName()),
		Execute: func(g *Game) {
		},
	}
	g.Enqueue(action)
	action = Action{
		Duration: 0.4,
		Message:  "リスナーとの絆が深まった。",
		Execute: func(g *Game) {
		},
	}
	g.Enqueue(action)
	g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)
}

type Ring struct {
	BaseItem
}

type Cane struct {
	BaseItem
}

func createItem(x, y int) Item {
	var item Item
	randomValue := localRand.Intn(6) // Store the random value to ensure it's only generated once
	sharpnessValue := localRand.Intn(5) - 1
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

	default:
		item = &Card{
			BaseItem: BaseItem{
				Entity: Entity{
					X:    x,
					Y:    y,
					Char: '!',
				},
				ID:          6,
				Type:        "Card",
				Name:        "黒炎弾のカード",
				Description: "魔法カード。眼の前の敵に30ダメージを与える。",
				UseActions: map[string]UseAction{
					"UseCard": damageHP30,
				},
			},
		}
	}

	return item
}

package main

type Character interface {
	GetPosition() (int, int)          // X, Y座標を返す
	SetPosition(x, y int)             // X, Y座標を設定する
	GetDirection() Direction          // Directionを返す
	SetDirection(direction Direction) // Directionを設定する
	GetName() string                  // GetName returns the name of the character
	GetHealth() int                   // GetHealth returns the current health of the character
	SetHealth(health int)             // SetHealth sets the current health of the character
	GetMaxHealth() int                // GetMaxHealth returns the maximum health of the character
	GetDefensePower() int             // GetDefensePower returns the defense power of the character
	// 他にも必要なメソッドを定義します（例: GetHealth(), SetHealth(), GetName(), etc.）
}

func (p *Player) GetPosition() (int, int) {
	return p.X, p.Y
}

func (p *Player) SetPosition(x, y int) {
	p.X = x
	p.Y = y
}

func (p *Player) GetDirection() Direction {
	return p.Direction
}

func (p *Player) SetDirection(direction Direction) {
	p.Direction = direction
}

func (p *Player) GetName() string {
	return p.Name
}

func (p *Player) GetHealth() int {
	return p.Health
}

func (p *Player) SetHealth(health int) {
	p.Health = health
}

func (p *Player) GetMaxHealth() int {
	return p.MaxHealth
}

func (p *Player) GetDefensePower() int {
	return p.DefensePower
}

func (e *Enemy) GetPosition() (int, int) {
	return e.X, e.Y
}

func (e *Enemy) SetPosition(x, y int) {
	e.X = x
	e.Y = y
}

func (e *Enemy) GetDirection() Direction {
	return e.Direction
}

func (e *Enemy) SetDirection(direction Direction) {
	e.Direction = direction
}

func (e *Enemy) GetName() string {
	return e.Name
}

func (e *Enemy) GetHealth() int {
	return e.Health
}

func (e *Enemy) SetHealth(health int) {
	e.Health = health
}

func (e *Enemy) GetMaxHealth() int {
	return e.MaxHealth
}

func (e *Enemy) GetDefensePower() int {
	return e.DefensePower
}

// Enemyタイプにこのメソッドを追加
func (e *Enemy) SetShowOnMiniMap(show bool) {
	e.ShowOnMiniMap = show
}

func (e *Enemy) GetShowOnMiniMap() bool {
	return e.ShowOnMiniMap
}

type UseAction func(g *Game)

type Item interface {
	GetType() string
	GetName() string
	GetDescription() string
	GetChar() rune
	GetPosition() (int, int)
	SetPosition(x, y int)
	Use(g *Game) // Now, Use will call the appropriate function from the UseActions map
	GetID() int  // Add this method to get the ID of the item
	SetShowOnMiniMap(show bool)
	GetShowOnMiniMap() bool
	// 他にも共通のメソッドがあればここに追加します。
}

func (item *BaseItem) SetShowOnMiniMap(show bool) {
	item.ShowOnMiniMap = show
}

func (b *BaseItem) GetShowOnMiniMap() bool {
	return b.ShowOnMiniMap
}

// Equipable interface
type Equipable interface {
	Item                                          // Embed the Item interface
	Identifiable                                  // Embed the Identifiable interface
	UpdatePlayerStats(player *Player, equip bool) // Method to update player stats when equipping/unequipping
}

// UpdatePlayerStats is a method to update player stats when equipping/unequipping an item
// This method needs to be implemented by each equipable item type (Weapon, Armor, Arrow, Accessory)
func (w *Weapon) UpdatePlayerStats(player *Player, equip bool) {
	if equip {
		player.AttackPower += w.AttackPower + w.Sharpness
	} else {
		player.AttackPower -= w.AttackPower + w.Sharpness
	}
}

func (a *Armor) UpdatePlayerStats(player *Player, equip bool) {
	if equip {
		player.DefensePower += a.DefensePower + a.Sharpness
	} else {
		player.DefensePower -= a.DefensePower + a.Sharpness
	}
}

func (ar *Arrow) UpdatePlayerStats(player *Player, equip bool) {
	// Arrows might not affect player stats but can affect other stats like ammo count
	// Implement logic accordingly
}

func (ac *Accessory) UpdatePlayerStats(player *Player, equip bool) {
	// Accessories might affect various stats
	// Implement logic accordingly
	// もし鼓舞の指輪を装備した場合は、プレイヤーのパワーとパワーの最大値を3上昇させる
	if ac.Name == "鼓舞の指輪" {
		if equip {
			// もし鼓舞の指輪が呪われている場合は、プレイヤーのパワーとパワーの最大値を3減少させる
			if ac.Cursed {
				player.Power -= 3
				player.MaxPower -= 3
			} else {
				player.Power += 3
				player.MaxPower += 3
			}
		} else {
			// もし鼓舞の指輪が呪われている場合は、プレイヤーのパワーとパワーの最大値を3上昇させる
			if ac.Cursed {
				player.Power += 3
				player.MaxPower += 3
			} else {
				player.Power -= 3
				player.MaxPower -= 3
			}
		}
	}
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

type Identifiable interface {
	IsIdentified() bool
	GetName() string
	SetIdentified(value bool) // Method to set the identified status of the item
	GetIdentified() bool      // Method to get the identified status of the item
}

func (w *Weapon) IsIdentified() bool {
	return w.Identified
}

func (a *Armor) IsIdentified() bool {
	return a.Identified
}

func (a *Arrow) IsIdentified() bool {
	return a.Identified
}

func (ac *Accessory) IsIdentified() bool {
	return ac.Identified
}

func (c *Cane) IsIdentified() bool {
	return c.Identified
}

func (m *Money) IsIdentified() bool {
	return m.Identified
}

func (w *Weapon) SetIdentified(value bool) {
	w.Identified = value
}

func (a *Armor) SetIdentified(value bool) {
	a.Identified = value
}

func (a *Arrow) SetIdentified(value bool) {
	// ArrowのIdentified状態を設定するロジック
	a.Identified = value
}

func (ac *Accessory) SetIdentified(value bool) {
	// AccessoryのIdentified状態を設定するロジック
	ac.Identified = value
}

func (c *Cane) SetIdentified(value bool) {
	// CaneのIdentified状態を設定するロジック
	c.Identified = value
}

func (m *Money) SetIdentified(value bool) {
	m.Identified = value
}

func (w *Weapon) GetIdentified() bool {
	return w.Identified
}

func (a *Armor) GetIdentified() bool {
	return a.Identified
}

func (a *Arrow) GetIdentified() bool {
	// ArrowのIdentified状態を取得するロジック
	return a.Identified
}

func (ac *Accessory) GetIdentified() bool {
	// AccessoryのIdentified状態を取得するロジック
	return ac.Identified
}

func (c *Cane) GetIdentified() bool {
	// CaneのIdentified状態を取得するロジック
	return c.Identified
}

func (m *Money) GetIdentified() bool {
	return m.Identified
}

func (c *Weapon) Use(g *Game) {
	if action, exists := c.UseActions["WeaponEffect"]; exists {
		action(g)
	}
}

func (c *Armor) Use(g *Game) {
	if action, exists := c.UseActions["ArmorEffect"]; exists {
		action(g)
	}
}

func (f *Food) Use(g *Game) {
	if action, exists := f.UseActions["RestoreSatiety"]; exists {
		action(g)
	}
}

func (p *Potion) Use(g *Game) {
	if action, exists := p.UseActions["RestoreHealth"]; exists {
		action(g)
	}
}

func (a *Arrow) Use(g *Game) {
	if action, exists := a.UseActions["ArrowEffect"]; exists {
		action(g)
	}
}

func (c *Card) Use(g *Game) {
	if action, exists := c.UseActions["UseCard"]; exists {
		action(g)
	}
}

func (m *Money) Use(g *Game) {
	if action, exists := m.UseActions["UseMoney"]; exists {
		action(g)
	}
}

func (t *Trap) Use(g *Game) {
	if action, exists := t.UseActions["SetTrap"]; exists {
		action(g)
	}
}

func (c *Cane) Use(g *Game) {
	if action, exists := c.UseActions["CaneEffect"]; exists {
		action(g)
	}
}

func (a *Accessory) Use(g *Game) {
	if action, exists := a.UseActions["AccessoryEffect"]; exists {
		action(g)
	}
}

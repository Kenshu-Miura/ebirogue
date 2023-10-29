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

// Equipable interface
type Equipable interface {
	Item                                          // Embed the Item interface
	UpdatePlayerStats(player *Player, equip bool) // Method to update player stats when equipping/unequipping
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

//go:build test

package main

type Tile struct {
	Type       string
	Blocked    bool
	BlockSight bool
	Visited    bool
	Brightness float64
}

type Entity struct {
	X, Y int
	Char rune
}

type Coordinate struct {
	X, Y int
}

type StatusAilments struct {
	Confusion   int
	Sleep       int
	Blind       int
	Poison      int
	Slow        int
	Haste       int
	Paralysis   bool
	Seal        bool
	HasteOnWake bool
}

type Player struct {
	Entity
	StatusAilments StatusAilments
}

type Enemy struct {
	Entity
}

type Item interface{}

type GameState struct {
	Map     [][]Tile
	Player  Player
	Enemies []Enemy
}

type Game struct {
	state      GameState
	isActioned bool
}

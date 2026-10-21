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
	Nigiri      int `json:",omitempty"`
	Berserk     int `json:",omitempty"`
	Decoy       int `json:",omitempty"`
	Invisible   int `json:",omitempty"`
	Invincible  int `json:",omitempty"`
	BearTrap    int // トラバサミの残りターン数
	Confusion   int
	Sleep       int
	Blind       int
	Slow        int
	Haste       int
	MouthSeal   int
	Paralysis   bool
	Seal        bool
	HasteOnWake bool
	Backward    int
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
	seed       int64
	rng        *gameRNG
}

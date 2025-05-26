//go:build test
// +build test

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

type Player struct {
	Entity
}

type Enemy struct {
	Entity
}

type Item interface{}

type GameState struct {
	Map    [][]Tile
	Player Player
}

type Game struct {
	state      GameState
	isActioned bool
}

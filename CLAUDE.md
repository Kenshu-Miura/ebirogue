# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

### Running the Game
- `go run .` - Run the game locally
- `GOOS=js GOARCH=wasm go build -o ebirogue.wasm` - Build for web (WASM)

### Testing
- `go test` - Run all tests
- `go test -v` - Run tests with verbose output
- `go test ./...` - Run tests in all subdirectories

### Building
- `go build` - Build the executable
- `go mod tidy` - Clean up module dependencies

## Architecture Overview

This is a roguelike game written in Go using the Ebiten game engine. The codebase is organized into focused files handling different game aspects:

### Core Game Loop
- `main.go` - Entry point, initializes Game struct and runs with `ebiten.RunGame`
- `Game` struct holds all game state (player, enemies, items, map, etc.)
- `ActionQueue` system manages turn-based actions and animations via `HandleActionQueue`

### Key Components
- **Map Generation** (`map.go`) - Room generation, connections, lighting, stairs, minimap
- **Input Handling** (`input.go`) - Keyboard input for movement, inventory, item usage
- **Player Movement** (`move.go`) - Player movement, level up/death, game reset
- **Enemy AI** (`enemy_ai.go`) - Enemy movement, chasing, status-ailment behavior
- **Rendering** (`draw.go` world/animation, `draw_hud.go` HUD/minimap, `draw_ui.go` menus/windows)
- **Item System** (`item.go`, `items.go`, `itemeffects.go`) - Item definitions, effects, throwing mechanics
- **Enemy System** (`enemies.go`) - Enemy definitions and spawning
- **Shared Helpers** (`direction.go`, `gamehelpers.go`) - Direction/delta conversion, message actions, damage/defeat, cursed-equipment checks

### Interface System
- `interfaces.go` defines core interfaces: `Item`, `Equipable`, `Identifiable`
- Equipment items implement `Equipable` to modify player stats when equipped
- Items have various effect functions in `itemeffects.go`

### Special Systems
- **Lighting System** - `updateTileBrightness` shows only current room and adjacent tiles
- **Stub System** - `*_stub.go` files provide build stubs for testing without Ebiten dependencies
- **WASM Support** - Can be built for web deployment with appropriate build tags

## Development Notes

- Game uses Japanese comments throughout the codebase
- Turn-based action system queues actions for sequential execution
- Map generation uses room-based algorithm with connecting corridors
- Inventory and equipment system with stat modifications
- Tile-based visibility and exploration tracking
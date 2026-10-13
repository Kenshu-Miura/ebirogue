//go:build !test

package main

import (
	"image/png"
	"os"
	"testing"
)

func executeDamageTrapQueue(g *Game) {
	for len(g.ActionQueue.Queue) > 0 {
		a := g.ActionQueue.Queue[0]
		g.ActionQueue.Queue = g.ActionQueue.Queue[1:]
		if a.Execute != nil {
			a.Execute(g)
		}
	}
}

func TestDamageTrapEffects(t *testing.T) {
	// 物理罠は威力の7/8〜9/8（防御0）、大型地雷は現在HPの3/4。
	physical := func(id, power int) struct{ id, lo, hi int } {
		return struct{ id, lo, hi int }{id, power * 7 / 8, (power*9 + 7) / 8}
	}
	for _, tc := range []struct{ id, lo, hi int }{
		physical(8, woodArrowTrapPower), physical(9, ironArrowTrapPower), physical(10, fallingRockTrapPower), {11, 75, 75},
	} {
		for _, roll := range []int{0, 32} {
			g := &Game{state: GameState{Player: Player{Health: 100}}}
			g.randHooks.damage = func(int) int { return roll }
			trap := createMapTrapByID(tc.id, 2, 2)
			g.steppedTrapSource = trapSource(tc.id, trap.Name)
			trap.Effect(g)
			if g.state.Player.Health != 100 {
				t.Fatal("effect must be deferred")
			}
			executeDamageTrapQueue(g)
			damage := 100 - g.state.Player.Health
			if damage < tc.lo || damage > tc.hi {
				t.Fatalf("id %d damage %d", tc.id, damage)
			}
			if g.lastDamageSource != trapSource(tc.id, trap.Name) {
				t.Fatal("wrong source")
			}
			g.state.Player.Health = 1
			g.state.Player.DefensePower = 10000
			trap.Effect(g)
			executeDamageTrapQueue(g)
			if g.state.Player.Health != 0 || g.buildRunResult(false).DeathCause.TrapID != tc.id {
				t.Fatal("minimum damage/death cause")
			}
		}
	}
}

func TestDamageTrapMisfire(t *testing.T) {
	for id := 8; id <= 11; id++ {
		g := &Game{state: GameState{Player: Player{Health: 100}, MapTraps: []MapTrap{createMapTrapByID(id, 0, 0)}}}
		g.state.MapTraps[0].FailRate = 100
		g.stepOnTrap(0)
		executeDamageTrapQueue(g)
		if g.state.Player.Health != 100 || !g.state.MapTraps[0].Discovered || g.lastDamageSource.Kind != DamageSourceUnknown {
			t.Fatal("misfire changed health/source or failed discovery")
		}
		g.state.MapTraps[0].FailRate = 0
		g.stepOnTrap(0)
		executeDamageTrapQueue(g)
		if g.lastDamageSource.TrapID != id {
			t.Fatal("step source")
		}
	}
}

func TestLargeMineHitsAllEightNeighbors(t *testing.T) {
	g := &Game{state: GameState{Player: Player{Entity: Entity{X: 3, Y: 3}, Health: 100, Level: 1}}}
	g.randHooks.damage = func(int) int { return 0 }
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			e := CreateEnemyByID(0, 3+dx, 3+dy)
			e.Health = 35
			e.ExperiencePoints = 1
			g.state.Enemies = append(g.state.Enemies, e)
		}
	}
	e := CreateEnemyByID(0, 5, 3)
	e.Health = 100
	g.state.Enemies = append(g.state.Enemies, e)
	largeMineTrapEffect(g)
	executeDamageTrapQueue(g)
	if len(g.state.Enemies) != 1 || g.state.Enemies[0].Health != 100 || g.state.Player.ExperiencePoints != 8 {
		t.Fatal("neighbors/experience/outside range")
	}
}

func TestDamageTrapImages(t *testing.T) {
	for _, name := range []string{"wood_arrow_trap", "iron_arrow_trap", "falling_rock_trap", "large_mine_trap"} {
		f, err := os.Open("img/" + name + ".png")
		if err != nil {
			t.Fatal(err)
		}
		im, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if im.Bounds().Dx() != 30 || im.Bounds().Dy() != 30 {
			t.Fatal("image size")
		}
		for _, p := range [][2]int{{0, 0}, {29, 0}, {0, 29}, {29, 29}} {
			_, _, _, a := im.At(p[0], p[1]).RGBA()
			if a != 0 {
				t.Fatal("opaque corner")
			}
		}
	}
}

func TestDamageTrapsSaveAndGeneration(t *testing.T) {
	g, _ := newHeadlessGame(234, DifficultyAdvanced)
	g.state.MapTraps = nil
	for id := 8; id <= 11; id++ {
		trap := createMapTrapByID(id, 2, id)
		trap.Discovered = true
		g.state.MapTraps = append(g.state.MapTraps, trap)
	}
	restored := &Game{}
	if err := restored.applySaveData(g.buildSaveData()); err != nil {
		t.Fatal(err)
	}
	for i, trap := range restored.state.MapTraps {
		if trap.TemplateID != i+8 || !trap.Discovered || trap.Effect == nil || trap.Name != g.state.MapTraps[i].Name {
			t.Fatal("save round trip")
		}
	}
	seen := map[int]bool{}
	for seed := int64(0); seed < 100; seed++ {
		for _, trap := range generateMapTraps([]Room{{X: 0, Y: 0, Width: 20, Height: 20}}, newFloorRNG(seed, 1), 1) {
			seen[trap.TemplateID] = true
		}
	}
	for id := 8; id <= 11; id++ {
		if !seen[id] {
			t.Fatalf("id %d not generated", id)
		}
	}
}

func TestDamageTrapDefense(t *testing.T) {
	for id := 8; id <= 10; id++ {
		g := &Game{state: GameState{Player: Player{Health: 100, DefensePower: 20}}}
		g.randHooks.damage = func(int) int { return 0 }
		createMapTrapByID(id, 0, 0).Effect(g)
		executeDamageTrapQueue(g)
		if g.state.Player.Health <= 90 {
			t.Fatalf("id %d defense ineffective", id)
		}
	}
	g := &Game{state: GameState{Player: Player{Health: 100, EquippedArmor: &Armor{Abilities: []EquipmentAbilityID{AbilityExplosionResistance}}}}}
	largeMineTrapEffect(g)
	executeDamageTrapQueue(g)
	if g.state.Player.Health != 63 {
		t.Fatalf("blast resistance health %d", g.state.Player.Health)
	}
}

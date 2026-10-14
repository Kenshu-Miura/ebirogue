//go:build !test

package main

import (
	"bytes"
	"encoding/json"
	"image/png"
	"os"
	"testing"
)

func TestBossRoomPlacement(t *testing.T) {
	for seed := int64(1); seed <= 30; seed++ {
		for _, floor := range []int{10, 20, 30} {
			player := Player{}
			_, enemies, _, _, rooms, _ := GenerateRandomMap(70, 70, floor-1, &player, newFloorRNG(seed, floor))
			var boss Enemy
			for _, e := range enemies {
				if MonsterDefinitions[e.ID].IsBoss {
					boss = e
				}
			}
			largestArea, bossArea := 0, 0
			for _, room := range rooms {
				largestArea = max(largestArea, room.Width*room.Height)
				if isSameRoom(boss.X, boss.Y, room.Center.X, room.Center.Y, []Room{room}) {
					bossArea = room.Width * room.Height
					if isSameRoom(boss.X, boss.Y, player.X, player.Y, []Room{room}) {
						t.Fatal("player starts in boss room")
					}
				}
			}
			if bossArea != largestArea {
				t.Fatalf("boss room=%d largest=%d", bossArea, largestArea)
			}
		}
	}
}

func TestBossSeedReproducible(t *testing.T) {
	for id := 42; id <= 44; id++ {
		games := make([][]byte, 2)
		for run := range games {
			g, in := bossArena(123, id)
			g.immortal = true
			b := newBot(g, in, personaAggressive, 123)
			for steps := 0; steps < 10 && bossIndex(g) >= 0; steps++ {
				if err := b.step(); err != nil {
					t.Fatal(err)
				}
			}
			games[run], _ = json.Marshal(g.buildSaveData())
		}
		if !bytes.Equal(games[0], games[1]) {
			t.Fatalf("boss %d seed not reproducible", id)
		}
	}
}

func TestBossQueuedAttacksStopAfterDeath(t *testing.T) {
	for _, ranged := range []bool{false, true} {
		g, _ := bossArena(1, 44)
		if ranged {
			g.state.Enemies[0].X = 8
			g.tryBossSpecial(0)
			g.tryEnemyRangedAttack(0)
		} else {
			g.enqueueEnemyNormalAttack(0)
		}
		g.state.Enemies[0].Health = 0
		hp := g.state.Player.Health
		drainBossActions(g)
		if g.state.Player.Health != hp {
			t.Fatal("dead boss still attacked")
		}
	}
}

func bossArena(seed int64, id int) (*Game, *VirtualInput) {
	g, in := newHeadlessGame(seed, DifficultyAdvanced)
	g.Floor = (id - 41) * 10
	g.state.Map = makeTestFloorMap(12, 12)
	g.rooms = []Room{{ID: 1, X: 0, Y: 0, Width: 12, Height: 12}}
	g.state.Player.X, g.state.Player.Y = 4, 5
	g.state.Player.Inventory = nil
	g.state.Player.EquippedWeapon = nil
	g.state.Player.EquippedArmor = nil
	g.state.Player.EquippedArrow = nil
	g.state.Player.EquippedAccessories = [2]*Accessory{}
	g.state.Player.AttackPower, g.state.Player.DefensePower, g.state.Player.Power = 8, 8, 8
	g.state.Player.setLevel(map[int]int{42: 14, 43: 24, 44: 34}[id])
	g.state.Player.Health = g.state.Player.MaxHealth
	g.state.Player.Satiety = 100
	g.state.Player.StatusAilments = StatusAilments{}
	g.state.Enemies = []Enemy{CreateEnemyByID(id, 5, 5)}
	g.state.Enemies[0].PlayerDiscovered = true
	g.state.Items, g.state.MapTraps = nil, nil
	g.ActionQueue.Queue = nil
	g.headlessDamageTaken = 0
	return g, in
}

func drainBossActions(g *Game) {
	for len(g.ActionQueue.Queue) > 0 {
		action := g.ActionQueue.Queue[0]
		g.ActionQueue.Queue = g.ActionQueue.Queue[1:]
		if action.Execute != nil {
			action.Execute(g)
		}
	}
}

func TestBossShellSummonsAndSave(t *testing.T) {
	seenShell, seenSummon := false, false
	for seed := int64(1); seed <= 30; seed++ {
		g, _ := bossArena(seed, 42)
		before := g.state.Enemies[0]
		crabBossSpecial(&g.state.Enemies[0], g)
		if before.DefensePower != g.state.Enemies[0].DefensePower || g.state.Enemies[0].BossShellTurns != 0 || len(g.state.Enemies) != 1 {
			t.Fatal("special executed before its action")
		}
		drainBossActions(g)
		if g.state.Enemies[0].BossShellTurns > 0 {
			seenShell = true
			if g.state.Enemies[0].DefensePower != before.DefensePower+bossShellDefense {
				t.Fatal("shell defense")
			}
			saved := enemyToSaved(&g.state.Enemies[0])
			restored := savedToEnemy(saved)
			if restored.BossShellTurns != bossShellDuration || restored.DefensePower != g.state.Enemies[0].DefensePower {
				t.Fatal("shell save")
			}
			for n := 0; n < bossShellDuration; n++ {
				advanceBossState(&restored)
			}
			if restored.DefensePower != before.DefensePower {
				t.Fatal("shell did not expire")
			}
			restored = savedToEnemy(saved)
			restored.StatusAilments.Seal = true
			advanceBossState(&restored)
			if restored.BossShellTurns != 0 || restored.DefensePower != before.DefensePower {
				t.Fatal("seal did not stop shell")
			}
		} else {
			seenSummon = true
			if len(g.state.Enemies) != 3 {
				t.Fatalf("summoned enemies: %d", len(g.state.Enemies))
			}
			g.summonBossMinions(0)
			if len(g.state.Enemies) != 3 {
				t.Fatal("summon cap")
			}
			if !savedToEnemy(enemyToSaved(&g.state.Enemies[1])).BossMinion {
				t.Fatal("minion save")
			}
			if vs := checkInvariants(g); len(vs) > 0 {
				t.Fatal(vs)
			}
		}
	}
	if !seenShell || !seenSummon {
		t.Fatal("missing special branch")
	}
	g, _ := bossArena(1, 42)
	for len(g.state.Enemies) < MaxEnemiesOnFloor {
		g.state.Enemies = append(g.state.Enemies, CreateEnemyByID(0, 1, 1))
	}
	g.summonBossMinions(0)
	if len(g.state.Enemies) != MaxEnemiesOnFloor {
		t.Fatal("summon exceeds floor cap")
	}
}

func TestBossOctopusInkComboAndSwap(t *testing.T) {
	ink, combo, swap := false, false, false
	for seed := int64(1); seed <= 30; seed++ {
		g, _ := bossArena(seed, 43)
		hp := g.state.Player.Health
		octopusBossSpecial(&g.state.Enemies[0], g)
		if g.state.Player.Health != hp || g.state.Player.StatusAilments.Blind != 0 {
			t.Fatal("immediate special")
		}
		drainBossActions(g)
		if g.state.Player.StatusAilments.Blind > 0 {
			ink = true
			if g.state.Player.StatusAilments.Blind != bossInkDuration {
				t.Fatal("ink duration")
			}
		} else {
			combo = true
			if hp-g.state.Player.Health < 2 {
				t.Fatal("combo damage")
			}
		}
		g, _ = bossArena(seed, 43)
		g.state.Enemies[0].X = 8
		if g.tryBossSpecial(0) {
			swap = true
			if g.state.Player.X != 4 {
				t.Fatal("immediate swap")
			}
			drainBossActions(g)
			if g.state.Player.X != 8 || g.state.Enemies[0].X != 4 {
				t.Fatal("swap positions")
			}
			if vs := checkInvariants(g); len(vs) > 0 {
				t.Fatal(vs)
			}
		}
	}
	if !ink || !combo || !swap {
		t.Fatal("missing octopus branch")
	}
}

func TestBossControlResistanceAndSeal(t *testing.T) {
	for id, limit := range map[int]int{42: 3, 43: 2, 44: 1} {
		g, _ := bossArena(1, id)
		e := &g.state.Enemies[0]
		e.StatusAilments = StatusAilments{Sleep: 10, Confusion: 10, Paralysis: true, Seal: true, Slow: 10}
		normalizeBossStatus(e)
		if e.StatusAilments.Sleep != limit || e.StatusAilments.Confusion != limit || !e.StatusAilments.Seal || e.StatusAilments.Slow != 10 || e.StatusAilments.Paralysis != (id != 44) {
			t.Fatal(e.StatusAilments)
		}
		if g.tryBossSpecial(0) || bossExtraAction(*e) {
			t.Fatal("sealed special")
		}
		for _, status := range []StatusAilments{{Sleep: 1}, {Confusion: 1}, {Blind: 1}, {Paralysis: true}} {
			e.StatusAilments = status
			g.ActionQueue.Queue = nil
			g.actEnemy(0)
			// 最終ボス以外の金縛りと睡眠は完全に行動を止める。
			if (status.Sleep > 0 || (status.Paralysis && id != 44)) && len(g.ActionQueue.Queue) != 0 {
				t.Fatal("blocking status ignored")
			}
		}
	}
}

func TestBossDragonRangedAndRage(t *testing.T) {
	kinds := map[RangedAttackKind]bool{}
	for seed := int64(1); seed <= 30; seed++ {
		g, _ := bossArena(seed, 44)
		g.state.Enemies[0].X = 8
		g.actEnemy(0)
		if len(g.ActionQueue.Queue) == 0 {
			t.Fatal("no ranged attack")
		}
		kinds[g.state.Enemies[0].RangedAttack.Kind] = true
		drainBossActions(g)
		if g.headlessDamageTaken <= 0 {
			t.Fatal("no ranged damage")
		}
		g.state.Enemies[0].Health = g.state.Enemies[0].MaxHealth / 2
		if !bossExtraAction(g.state.Enemies[0]) {
			t.Fatal("rage boundary")
		}
		g.ActionQueue.Queue = nil
		g.MoveEnemies()
		if len(g.ActionQueue.Queue) != 2 {
			t.Fatalf("rage actions %d", len(g.ActionQueue.Queue))
		}
		g.state.Enemies[0].Health++
		if bossExtraAction(g.state.Enemies[0]) {
			t.Fatal("rage too early")
		}
	}
	if !kinds[RangedAttackFire] || !kinds[RangedAttackMagic] {
		t.Fatal("missing ranged kind")
	}
}

func TestBossInstantRemovalResistance(t *testing.T) {
	for id := 42; id <= 44; id++ {
		g, _ := bossArena(1, id)
		if g.applyGenocide(id) != 0 || len(g.genocidedMonsterIDs) != 0 {
			t.Fatal("genocide boss")
		}
		g.genocidedMonsterIDs = map[int]bool{id: true}
		g.removeGenocidedEnemies()
		if len(g.state.Enemies) != 1 {
			t.Fatal("legacy genocide removed boss")
		}
		survivors, destroyed := splitEnemiesByRoomWideEffect(4, 5, g.state.Enemies, g.rooms)
		if len(survivors) != 1 || len(destroyed) != 0 {
			t.Fatal("annihilation removed boss")
		}
		survivors, destroyed = splitEnemiesByExplosion(4, 5, g.state.Enemies)
		if len(survivors) != 1 || len(destroyed) != 0 {
			t.Fatal("self destruct removed boss")
		}
		g.state.Player.SetTrap = &Trap{BaseItem: BaseItem{Name: "炸裂装甲のカード"}}
		g.AttackFromEnemy(0)
		drainBossActions(g)
		if len(g.state.Enemies) != 1 || g.state.Enemies[0].Health >= g.state.Enemies[0].MaxHealth || g.state.Player.SetTrap != nil {
			t.Fatal("explosive armor boss damage")
		}
		// シフトチェンジは変身・消去ではなく場所替え。ボスにも有効。
		g.TargetEnemyIndex = 0
		shiftChange(g)
		drainBossActions(g)
		if len(g.state.Enemies) != 1 || g.state.Enemies[0].ID != id || g.state.Player.X != 5 {
			t.Fatal("shift change")
		}
	}
}

func TestBossAssets(t *testing.T) {
	for _, kind := range []string{"BossCrab", "BossOctopus", "BossDragon"} {
		f, err := os.Open(upperEnemyImagePaths[kind])
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds().Dx() != 30 || img.Bounds().Dy() != 30 {
			t.Fatal(img.Bounds())
		}
		for _, p := range [][2]int{{0, 0}, {29, 0}, {0, 29}, {29, 29}} {
			_, _, _, a := img.At(p[0], p[1]).RGBA()
			if a != 0 {
				t.Fatal("opaque corner")
			}
		}
	}
}

func provisionalBosses() map[int]MonsterDefinition {
	return map[int]MonsterDefinition{
		42: provisionalBossDefinition(MonsterDefinitions[23], 42, "ヌシガンセキガニ", 160, 20, 13, 300),
		43: provisionalBossDefinition(MonsterDefinitions[32], 43, "ヌシトッカエダコ", 240, 25, 15, 500),
		44: provisionalBossDefinition(MonsterDefinitions[37], 44, "ヌシ蒼海竜", 340, 30, 17, 800),
	}
}

// 調整前の仮ボスは比較用テストだけに残す。
func provisionalBossDefinition(base MonsterDefinition, id int, name string, health, attack, defense, experience int) MonsterDefinition {
	boss := upperMonsterDefinition(base, id, base.Type, name, health, attack, defense, experience)
	boss.IsBoss = true
	return boss
}

// 同じレベル・装備込み能力・シードで、仮ボスと新ボスを不死モードで比較する。
func TestSimBossBalance(t *testing.T) {
	if !*simImmortal {
		t.Skip("use -args -sim.immortal")
	}
	original := map[int]MonsterDefinition{}
	for id := 42; id <= 44; id++ {
		original[id] = MonsterDefinitions[id]
	}
	defer func() {
		for id, def := range original {
			MonsterDefinitions[id] = def
		}
	}()
	for _, mode := range []string{"before", "after"} {
		for id := 42; id <= 44; id++ {
			MonsterDefinitions[id] = original[id]
			if mode == "before" {
				MonsterDefinitions[id] = provisionalBosses()[id]
			}
			turns, damage := 0, 0
			for seed := int64(1); seed <= 100; seed++ {
				g, in := bossArena(seed, id)
				g.immortal = *simImmortal
				g.state.Enemies[0].X = 8
				b := newBot(g, in, personaAggressive, seed)
				for steps := 0; bossIndex(g) >= 0 && steps < 1000; steps++ {
					if err := b.step(); err != nil {
						t.Fatalf("%s boss %d seed %d: %v", mode, id, seed, err)
					}
					if vs := checkInvariants(g); len(vs) > 0 {
						t.Fatal(vs)
					}
				}
				if bossIndex(g) >= 0 {
					t.Fatalf("%s boss %d seed %d not defeated", mode, id, seed)
				}
				turns += g.turnCount
				damage += g.headlessDamageTaken
			}
			t.Logf("%s boss=%d runs=100 meanTurns=%.2f meanDamage=%.2f", mode, id, float64(turns)/100, float64(damage)/100)
		}
	}
}

func TestSimBossBaselineBatch(t *testing.T) {
	if *simRuns == 0 {
		t.Skip("use -sim.runs")
	}
	original := map[int]MonsterDefinition{}
	for id, def := range provisionalBosses() {
		original[id] = MonsterDefinitions[id]
		MonsterDefinitions[id] = def
	}
	defer func() {
		for id, def := range original {
			MonsterDefinitions[id] = def
		}
	}()
	TestSimBatch(t)
}

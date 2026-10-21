//go:build !test

package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestStatusStatsPotions(t *testing.T) {
	for id, kind := range map[int]StatusKind{205: StatusHaste, 206: StatusAttackUp, 207: StatusDefenseUp, 208: StatusAttackDown} {
		g := newFoundationTestGame()
		usePlayerItem(g, id)
		if !statusActive(g.state.Player.StatusAilments, kind) {
			t.Fatalf("potion %d: status missing", id)
		}
		g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 4)}
		g.onTargetHit(&g.state.Enemies[0], buildItemFromTemplate(id, 0, 0), 0)
		runQueuedActions(g)
		if !statusActive(g.state.Enemies[0].StatusAilments, kind) {
			t.Fatalf("thrown potion %d: status missing", id)
		}
	}
}

func TestStatusStatsLifecycle(t *testing.T) {
	for _, kind := range []StatusKind{StatusHaste, StatusAttackUp, StatusAttackDown, StatusDefenseUp, StatusDefenseDown, StatusNoHealing} {
		t.Run(string(kind), func(t *testing.T) {
			g := newFoundationTestGame()
			if !g.applyPlayerStatus(kind, 2, StatusSource{}) {
				t.Fatal("not applied")
			}
			g.applyPlayerStatus(kind, 1, StatusSource{})
			if statusTurns(g.state.Player.StatusAilments, kind) != 2 {
				t.Fatal("shorter duration overwrote status")
			}
			def, _ := statusDefOf(kind)
			if !strings.Contains(formatPlayerStatus(g.state.Player.StatusAilments), def.HUD+"(2)") {
				t.Fatal("missing HUD")
			}
			runQueuedActions(g)
			g.decrementStatusAilments()
			if statusTurns(g.state.Player.StatusAilments, kind) != 1 {
				t.Fatal("did not tick")
			}
			g.decrementStatusAilments()
			if statusActive(g.state.Player.StatusAilments, kind) || !strings.Contains(runQueuedActions(g), def.PlayerRecover) {
				t.Fatal("did not recover")
			}
			g.applyPlayerStatus(kind, 3, StatusSource{})
			g.clearPlayerBadStatuses()
			if statusActive(g.state.Player.StatusAilments, kind) == def.Bad {
				t.Fatal("bad status classification")
			}
			if kind != StatusNoHealing {
				g.state.Enemies = []Enemy{CreateEnemyByID(0, 6, 4)}
				g.applyEnemyStatus(0, kind, 2)
				g.decrementStatusAilments()
				if statusTurns(g.state.Enemies[0].StatusAilments, kind) != 1 {
					t.Fatal("enemy did not tick")
				}
			}
		})
	}
	for _, pair := range [][2]StatusKind{{StatusHaste, StatusSlow}, {StatusAttackUp, StatusAttackDown}, {StatusDefenseUp, StatusDefenseDown}} {
		for _, reverse := range []bool{false, true} {
			a, b := pair[0], pair[1]
			if reverse {
				a, b = b, a
			}
			g := newFoundationTestGame()
			g.applyPlayerStatus(a, 30, StatusSource{})
			g.applyPlayerStatus(b, 20, StatusSource{})
			if statusActive(g.state.Player.StatusAilments, a) || statusActive(g.state.Player.StatusAilments, b) {
				t.Fatalf("player did not cancel %s/%s", a, b)
			}
			g.state.Enemies = []Enemy{CreateEnemyByID(0, 6, 4)}
			g.applyEnemyStatus(0, a, 30)
			g.applyEnemyStatus(0, b, 20)
			if statusActive(g.state.Enemies[0].StatusAilments, a) || statusActive(g.state.Enemies[0].StatusAilments, b) {
				t.Fatal("enemy did not cancel")
			}
		}
	}
}

func TestStatusStatsResistance(t *testing.T) {
	for _, kind := range []StatusKind{StatusAttackDown, StatusDefenseDown, StatusNoHealing} {
		g := newFoundationTestGame()
		g.state.Player.EquippedArmor = &Armor{BaseItem: BaseItem{Name: "耐性盾"}, Abilities: []EquipmentAbilityID{AbilityStatusResistance}}
		g.randHooks.statusResistance = func(int) int { return 0 }
		if g.applyPlayerStatus(kind, 30, StatusSource{Shield: true}) {
			t.Fatalf("shield did not block %s", kind)
		}
		if !g.applyPlayerStatus(kind, 30, StatusSource{}) {
			t.Fatal("shield blocked potion")
		}
		withTestImmunity(t, EquipmentAbilityID("stats_"+string(kind)), kind)
		equip(g, testRing(EquipmentAbilityID("stats_"+string(kind))))
		g.clearPlayerStatus(kind)
		if g.applyPlayerStatus(kind, 30, StatusSource{}) {
			t.Fatal("ring did not block")
		}
	}
	g := newFoundationTestGame()
	g.state.Player.EquippedArmor = &Armor{Abilities: []EquipmentAbilityID{AbilityStatusResistance}}
	g.randHooks.statusResistance = func(int) int { t.Fatal("buff used resistance RNG"); return 0 }
	for _, kind := range []StatusKind{StatusHaste, StatusAttackUp, StatusDefenseUp} {
		if !g.applyPlayerStatus(kind, 20, StatusSource{Shield: true}) {
			t.Fatal("buff blocked")
		}
	}
}

func TestStatusStatsHasteEnemyTurns(t *testing.T) {
	for _, slow := range []bool{false, true} {
		g, _ := newTurnTestGame(t)
		g.state.Player.Health, g.state.Player.MaxHealth = 1000, 1000
		g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 3)}
		g.applyPlayerStatus(StatusHaste, 20, StatusSource{})
		if slow {
			g.applyPlayerStatus(StatusSlow, 20, StatusSource{})
		}
		runQueuedActions(g)
		g.isCombatActive = false
		attacks := 0
		for i := 0; i < 4; i++ {
			g.isActioned = true
			g.CheckCombatState()
			for _, a := range queuedActions(g) {
				if strings.Contains(a.Message, "ダメージを受けた") {
					attacks++
				}
			}
			runQueuedActions(g)
			g.isCombatActive = false
		}
		want := 2
		if slow {
			want = 4
		}
		if attacks != want {
			t.Fatalf("slow=%v attacks=%d want=%d", slow, attacks, want)
		}
	}
}

func TestStatusStatsNoHealing(t *testing.T) {
	for _, id := range []int{2, 3, 28, 29, 44} {
		g := newFoundationTestGame()
		g.state.Player.Health = 10
		g.setPlayerStatus(StatusNoHealing, 30)
		usePlayerItem(g, id)
		if g.state.Player.Health != 10 {
			t.Fatalf("item %d healed", id)
		}
		g.IncrementMoveCount()
		runQueuedActions(g)
		if g.state.Player.Health != 10 {
			t.Fatal("natural regen healed")
		}
	}
	for _, id := range []int{2, 44} {
		g := newFoundationTestGame()
		old := g.state.Player.MaxHealth
		g.setPlayerStatus(StatusNoHealing, 30)
		usePlayerItem(g, id)
		if g.state.Player.MaxHealth <= old || g.state.Player.Health != old {
			t.Fatalf("max HP bonus under no healing: %+v", g.state.Player)
		}
	}
	g := newFoundationTestGame()
	noHealingTrapEffect(g)
	runQueuedActions(g)
	if g.state.Player.StatusAilments.NoHealing != 30 {
		t.Fatal("trap failed")
	}
	usePlayerItem(g, 173)
	if statusActive(g.state.Player.StatusAilments, StatusNoHealing) {
		t.Fatal("trouble card did not cure")
	}
}

func TestStatusStatsLevelDown(t *testing.T) {
	g := newFoundationTestGame()
	usePlayerItem(g, 209)
	if g.state.Player.Level != 1 || g.state.Player.MaxHealth != 40 {
		t.Fatal("Lv1 boundary")
	}
	g.state.Player.setLevel(8)
	g.state.Player.ExperiencePoints = expForLevel(9) - 1
	wantMax := g.state.Player.MaxHealth + levelHealthBonus(7) - levelHealthBonus(8)
	usePlayerItem(g, 209)
	if p := g.state.Player; p.Level != 7 || p.ExperiencePoints != expForLevel(7) || p.MaxHealth != wantMax || p.Health > p.MaxHealth {
		t.Fatalf("level down %+v", p)
	}
	g.state.Player.MaxHealth = 1
	g.state.Player.Health = 1
	usePlayerItem(g, 209)
	if g.state.Player.MaxHealth != 1 || g.state.Player.Health != 1 {
		t.Fatal("max HP lower bound")
	}
	for _, id := range []int{0, 42, 43, 44, 118} {
		g.state.Enemies = []Enemy{CreateEnemyByID(id, 5, 4)}
		if id == 118 {
			disguiseEnemyAs(&g.state.Enemies[0], 0)
		}
		g.onTargetHit(&g.state.Enemies[0], buildItemFromTemplate(209, 0, 0), 0)
		runQueuedActions(g)
		want := id
		if id == 118 {
			want = 117
		}
		if e := g.state.Enemies[0]; e.ID != want || (id == 118 && (enemyMasquerading(e) || e.Name != MonsterDefinitions[117].Name)) {
			t.Fatalf("devolution %+v", e)
		}
	}
}

func TestStatusStatsCanes(t *testing.T) {
	for _, id := range []int{210, 211, 212, 213} {
		for _, bolt := range []bool{false, true} {
			g := newProjectileTestGame()
			addTestEnemy(g, 118, 6, 4)
			cane := buildItemFromTemplate(id, 0, 0).(*Cane)
			g.state.Player.Inventory = []Item{cane}
			if bolt {
				if !g.useCane(cane) {
					t.Fatal("cannot use cane")
				}
				finishThrow(t, g)
			} else {
				g.onTargetHit(&g.state.Enemies[0], cane, 0)
				runQueuedActions(g)
			}
			if id == 212 {
				if g.state.Enemies[0].ID != 117 {
					t.Fatal("cane did not devolve")
				}
			} else {
				for _, kind := range statItemKinds[id] {
					if statusTurns(g.state.Enemies[0].StatusAilments, kind) != statItemTurns(kind) {
						t.Fatalf("cane %d missing %s", id, kind)
					}
				}
			}
			if bolt && (cane.Uses != 4 || len(g.state.Player.Inventory) != 1 || len(g.state.Items) != 0) {
				t.Fatal("bolt consumed cane or left floor item")
			}
		}
	}
}

func TestStatusStatsSuspendRoundTrip(t *testing.T) {
	g, _ := newHeadlessGame(77, DifficultyAdvanced)
	g.state.Player.StatusAilments = StatusAilments{Haste: 20, AttackUp: 30, DefenseDown: 25, NoHealing: 12}
	g.state.Enemies = []Enemy{CreateEnemyByID(118, 2, 2)}
	g.state.Enemies[0].StatusAilments = StatusAilments{AttackDown: 27, DefenseUp: 21}
	data, err := json.Marshal(g.buildSaveData())
	if err != nil {
		t.Fatal(err)
	}
	save, err := decodeSaveData(data)
	if err != nil {
		t.Fatal(err)
	}
	restored := &Game{}
	if err = restored.applySaveData(save); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.state.Player.StatusAilments, restored.state.Player.StatusAilments) || !reflect.DeepEqual(g.state.Enemies[0].StatusAilments, restored.state.Enemies[0].StatusAilments) {
		t.Fatal("lost status in save")
	}
	var legacy StatusAilments
	if err = json.Unmarshal([]byte(`{"Sleep":2}`), &legacy); err != nil || legacy.NoHealing != 0 || legacy.AttackDown != 0 {
		t.Fatal("legacy save failed")
	}
}

func TestStatusStatsDamage(t *testing.T) {
	for _, attackKind := range []StatusKind{StatusAttackUp, StatusAttackDown} {
		for _, defenseKind := range []StatusKind{StatusDefenseUp, StatusDefenseDown} {
			g := newFoundationTestGame()
			g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 4)}
			g.state.Enemies[0].Health, g.state.Enemies[0].MaxHealth = 1000, 1000
			g.setPlayerStatus(attackKind, 30)
			g.applyEnemyStatus(0, defenseKind, 30)
			p, e := &g.state.Player, &g.state.Enemies[0]
			want := rollPlayerAttackDamage(playerAttackTotal(p.AttackPower, p.Power, p.Level), e.DefensePower, statAttackMultiplier(p.StatusAilments)*statDefenseMultiplier(e.StatusAilments), g.damageIntn()).Damage
			g.CheckForEnemies(1, 0)
			runQueuedActions(g)
			if 1000-e.Health != want {
				t.Fatalf("player %s/%s damage=%d want=%d", attackKind, defenseKind, 1000-e.Health, want)
			}
			g.clearPlayerStatus(attackKind)
			g.clearEnemyStatus(0, defenseKind)
			g.setPlayerStatus(defenseKind, 30)
			g.applyEnemyStatus(0, attackKind, 30)
			p.Health, p.MaxHealth = 1000, 1000
			want = rollEnemyAttackDamageWith(e.AttackPower, p.DefensePower, statAttackMultiplier(e.StatusAilments)*statDefenseMultiplier(p.StatusAilments), g.damageIntn())
			g.enqueueEnemyNormalAttack(0)
			runQueuedActions(g)
			if 1000-p.Health != want {
				t.Fatalf("enemy %s/%s damage=%d want=%d", attackKind, defenseKind, 1000-p.Health, want)
			}
		}
	}
}

func TestStatusStatsThrownPotionFlight(t *testing.T) {
	for _, id := range []int{205, 206, 207, 208, 209} {
		g := newProjectileTestGame()
		addTestEnemy(g, 118, 6, 4)
		potion := buildItemFromTemplate(id, 0, 0)
		g.state.Player.Inventory = []Item{potion}
		g.throwWithCallbacks(potion, 10)
		finishThrow(t, g)
		if id == 209 {
			if g.state.Enemies[0].ID != 117 {
				t.Fatal("thrown devolution failed")
			}
		} else {
			for _, kind := range statItemKinds[id] {
				if !statusActive(g.state.Enemies[0].StatusAilments, kind) {
					t.Fatalf("thrown %d missing %s", id, kind)
				}
			}
		}
		if len(g.state.Player.Inventory) != 0 || len(g.state.Items) != 0 {
			t.Fatal("hit potion not consumed")
		}
	}
}

func TestStatusStatsEffectIsDeferred(t *testing.T) {
	g := newFoundationTestGame()
	queuePlayerItem(g, 206)
	if statusActive(g.state.Player.StatusAilments, StatusAttackUp) {
		t.Fatal("potion applied before Execute")
	}
	runQueuedActions(g)
	g.state.Enemies = []Enemy{CreateEnemyByID(118, 6, 4)}
	g.onTargetHit(&g.state.Enemies[0], buildItemFromTemplate(212, 0, 0), 0)
	if g.state.Enemies[0].ID != 118 {
		t.Fatal("devolved before Execute")
	}
	runQueuedActions(g)
	if g.state.Enemies[0].ID != 117 {
		t.Fatal("devolution missing")
	}
}

func TestStatusStatsWakeCancelsSlow(t *testing.T) {
	g := newFoundationTestGame()
	e := CreateEnemyByID(0, 6, 4)
	addStatus(&e.StatusAilments, StatusSleep, 1, StackDefault)
	markHasteOnWake(&e.StatusAilments)
	addStatus(&e.StatusAilments, StatusSlow, 10, StackDefault)
	g.state.Enemies = []Enemy{e}
	g.decrementStatusAilments()
	if s := g.state.Enemies[0].StatusAilments; s.Haste != 0 || s.Slow != 0 || s.Sleep != 0 || s.HasteOnWake {
		t.Fatalf("wake did not cancel slow: %+v", s)
	}
}

func TestStatusStatsDevolutionKeepsMutableState(t *testing.T) {
	g := newFoundationTestGame()
	e := CreateEnemyByID(118, 6, 4)
	e.Health = 1
	e.Counter, e.Timer, e.Flag, e.Alerted = 5, 7, true, true
	addStatus(&e.StatusAilments, StatusAttackDown, 11, StackDefault)
	g.state.Enemies = []Enemy{e}
	if !g.levelDownEnemy(0) {
		t.Fatal("did not devolve")
	}
	got := g.state.Enemies[0]
	if got.ID != 117 || got.X != 6 || got.Y != 4 || got.Health != 1 || got.Counter != 5 || got.Timer != 7 || !got.Flag || !got.Alerted || got.StatusAilments != e.StatusAilments {
		t.Fatalf("lost mutable state: %+v", got)
	}
	if g.levelDownEnemy(-1) || g.levelDownEnemy(1) {
		t.Fatal("invalid index accepted")
	}
}

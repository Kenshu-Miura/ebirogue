//go:build !test

package main

import (
	"encoding/json"
	"image/png"
	"os"
	"strings"
	"testing"
)

func TestSpecialStatusLifecycleHUDAndSave(t *testing.T) {
	g, _ := newTurnTestGame(t)
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 6, 3)}
	for _, tc := range []struct {
		k             StatusKind
		n             int
		player, enemy bool
	}{
		{StatusNigiri, 20, true, false}, {StatusBerserk, 15, true, true}, {StatusDecoy, 30, false, true}, {StatusInvisible, 30, true, true}, {StatusInvincible, 10, true, false},
	} {
		t.Run(string(tc.k), func(t *testing.T) {
			g.state.Player.StatusAilments = StatusAilments{}
			g.state.Enemies[0].StatusAilments = StatusAilments{}
			if got := g.applyPlayerStatus(tc.k, tc.n, StatusSource{}); got != tc.player {
				t.Fatalf("player apply=%v", got)
			}
			if got := g.applyEnemyStatus(0, tc.k, tc.n); got != tc.enemy {
				t.Fatalf("enemy apply=%v", got)
			}
			if tc.player && !strings.Contains(formatPlayerStatus(g.state.Player.StatusAilments), statusKindLabel(tc.k)+"(") {
				t.Fatal("missing HUD")
			}
			data, err := json.Marshal(g.buildSaveData())
			if err != nil {
				t.Fatal(err)
			}
			var save SaveData
			if err = json.Unmarshal(data, &save); err != nil {
				t.Fatal(err)
			}
			restored, _ := newHeadlessGame(1, DifficultyAdvanced)
			if err = restored.applySaveData(&save); err != nil {
				t.Fatal(err)
			}
			if restored.state.Player.StatusAilments != g.state.Player.StatusAilments || restored.state.Enemies[0].StatusAilments != g.state.Enemies[0].StatusAilments {
				t.Fatal("status save mismatch")
			}
			runQueuedActions(g)
			g.decrementStatusAilments()
			runQueuedActions(g)
			if tc.player && statusTurns(g.state.Player.StatusAilments, tc.k) != tc.n-1 {
				t.Fatal("player tick")
			}
			if tc.enemy && statusTurns(g.state.Enemies[0].StatusAilments, tc.k) != tc.n-1 {
				t.Fatal("enemy tick")
			}
			for n := 1; n < tc.n; n++ {
				g.decrementStatusAilments()
				runQueuedActions(g)
			}
			if statusActive(g.state.Player.StatusAilments, tc.k) || statusActive(g.state.Enemies[0].StatusAilments, tc.k) {
				t.Fatal("did not expire")
			}
		})
	}
}

func TestSpecialStatusItemsDrinkHitAndBoss(t *testing.T) {
	for id := 215; id <= 221; id++ {
		g, _ := newTurnTestGame(t)
		item := buildItemFromTemplate(id, 0, 0)
		kind, turns := specialItemStatus(id)
		if item == nil || item.GetID() != id {
			t.Fatalf("template %d", id)
		}
		if id <= 218 {
			g.state.Player.Inventory = []Item{item}
			item.Use(g)
			if statusActive(g.state.Player.StatusAilments, kind) {
				t.Fatal("effect ran before Execute")
			}
			runQueuedActions(g)
			if statusTurns(g.state.Player.StatusAilments, kind) != turns || len(g.state.Player.Inventory) != 0 {
				t.Fatalf("drink %d", id)
			}
			g.state.Player.StatusAilments = StatusAilments{}
		} else if c := item.(*Cane); c.Uses != 5 || c.Identified {
			t.Fatalf("cane defaults %d", id)
		}
		for _, boss := range []bool{false, true} {
			enemyID := 0
			if boss {
				enemyID = 42
			}
			g.state.Enemies = []Enemy{CreateEnemyByID(enemyID, 6, 3)}
			g.state.Items = nil
			exp := g.state.Player.ExperiencePoints
			g.onTargetHit(&g.state.Enemies[0], item, 0)
			runQueuedActions(g)
			switch id {
			case 217:
				if statusActive(g.state.Enemies[0].StatusAilments, StatusInvincible) {
					t.Fatal("enemy invincible")
				}
			case 218:
				if boss {
					if len(g.state.Enemies) != 1 {
						t.Fatal("boss transformed")
					}
				} else if len(g.state.Enemies) != 0 || len(g.state.Items) != 1 || g.state.Items[0].GetID() != 1 {
					t.Fatal("food transform")
				}
			default:
				want := !(boss && id == 219)
				if statusActive(g.state.Enemies[0].StatusAilments, kind) != want {
					t.Fatalf("hit %d boss %v", id, boss)
				}
			}
			if g.state.Player.ExperiencePoints != exp {
				t.Fatal("transform awarded experience")
			}
		}
	}
	for id := 222; id <= 224; id++ {
		if _, ok := itemTemplates[id]; ok {
			t.Fatal("reserved ID used")
		}
	}
}

func TestSpecialBerserkAutomaticTurnsAndAttack(t *testing.T) {
	for _, withEnemy := range []bool{false, true} {
		g, b := newTurnTestGame(t)
		if withEnemy {
			e := CreateEnemyByID(0, 6, 3)
			e.Health = 1000
			e.MaxHealth = 1000
			addStatus(&e.StatusAilments, StatusSleep, 50, StackDefault)
			g.state.Enemies = []Enemy{e}
		}
		g.setPlayerStatus(StatusBerserk, 15)
		turn, floor := g.turnCount, g.floorTurns
		// 仮想入力のメニューキーを押しても、自動行動の15ターンが終わる。
		for frames := 0; statusActive(g.state.Player.StatusAilments, StatusBerserk) && frames < 2000; frames++ {
			b.in.Press(KeyC)
			if err := g.StepFrame(); err != nil {
				t.Fatal(err)
			}

		}
		b.in.ReleaseAll()
		if err := g.RunUntilInputReady(2000); err != nil {
			t.Fatal(err)
		}
		if statusActive(g.state.Player.StatusAilments, StatusBerserk) {
			t.Fatal("berserk stuck")
		}
		if g.turnCount-turn != 15 || g.floorTurns-floor != 15 {
			t.Fatalf("turns=%d floor=%d", g.turnCount-turn, g.floorTurns-floor)
		}
		if withEnemy && g.state.Enemies[0].Health == 1000 {
			t.Fatal("did not approach and attack")
		}
		if g.showMenu || g.showInventory {
			t.Fatal("accepted menu input")
		}
	}
}

func TestSpecialEnemyTargetsDecoyBerserkInvisible(t *testing.T) {
	g, _ := newTurnTestGame(t)
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 3), CreateEnemyByID(0, 6, 3), CreateEnemyByID(0, 6, 4)}
	g.applyEnemyStatus(1, StatusDecoy, 30)
	g.applyEnemyStatus(2, StatusDecoy, 30)
	g.state.Enemies[1].Health = 1
	before := g.state.Player.Health
	if target := g.specialEnemyTarget(0); target != 1 {
		t.Fatalf("target=%d", target)
	}
	g.actEnemy(0)
	runQueuedActions(g)
	if g.state.Player.Health != before || len(g.state.Enemies) != 2 {
		t.Fatal("decoy not attacked")
	}
	x, y := g.state.Enemies[1].X, g.state.Enemies[1].Y
	g.actEnemy(1)
	runQueuedActions(g)
	if g.state.Enemies[1].X != x || g.state.Enemies[1].Y != y {
		t.Fatal("decoy moved")
	}
	g.clearEnemyStatus(1, StatusDecoy)
	g.applyEnemyStatus(0, StatusBerserk, 15)
	g.state.Player.X, g.state.Player.Y = 1, 1
	if g.specialEnemyTarget(0) != 1 {
		t.Fatal("berserk ignored closer enemy")
	}
	g.clearEnemyStatus(0, StatusBerserk)
	g.setPlayerStatus(StatusInvisible, 30)
	if g.specialEnemyTarget(0) != -2 {
		t.Fatal("invisible player tracked")
	}
	g.state.Enemies[0].PlayerDiscovered = true
	g.actEnemy(0)
	runQueuedActions(g)
	if g.state.Enemies[0].PlayerDiscovered {
		t.Fatal("retained invisible player position")
	}
	g.state.Player.X, g.state.Player.Y = g.state.Enemies[0].X-1, g.state.Enemies[0].Y
	if g.specialEnemyTarget(0) != -1 {
		t.Fatal("adjacent invisible player ignored")
	}
}

func TestSpecialInvisibleEnemyVisibility(t *testing.T) {
	g, _ := newTurnTestGame(t)
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 3)}
	g.applyEnemyStatus(0, StatusInvisible, 30)
	g.updateEnemyVisibility()
	if g.specialEnemyVisible(g.state.Enemies[0]) || g.state.Enemies[0].ShowOnMiniMap {
		t.Fatal("invisible enemy visible")
	}
	equip(g, testRing(AbilitySenseEnemies))
	g.updateEnemyVisibility()
	if !g.specialEnemyVisible(g.state.Enemies[0]) || !g.state.Enemies[0].ShowOnMiniMap {
		t.Fatal("sense did not reveal")
	}
}

func TestSpecialInvincibleAllDamageAndStatus(t *testing.T) {
	g, _ := newTurnTestGame(t)
	g.setPlayerStatus(StatusInvincible, 10)
	before := g.state.Player.Health
	g.damagePlayer(1000, DamageSource{})
	// 本物の罠・爆発・ターン経過の経路を使う。
	mineTrapEffect(g)
	runQueuedActions(g)
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 3)}
	g.enqueueEnemyExplosion(EnemyExplosion{BomberIndex: 0, CenterX: 4, CenterY: 3, Radius: 1, Damage: 1000})
	runQueuedActions(g)
	g.state.Player.Satiety = 0
	g.IncrementMoveCount()
	runQueuedActions(g)
	if g.state.Player.Health != before {
		t.Fatal("invincible took damage")
	}
	if !g.applyPlayerStatus(StatusSleep, 3, StatusSource{}) {
		t.Fatal("invincible blocked status")
	}
	g.clearPlayerStatus(StatusInvincible)
	g.damagePlayer(1, DamageSource{})
	if g.state.Player.Health != before-1 {
		t.Fatal("invincibility persisted")
	}
}

func TestSpecialNigiriUseDefenseSatietyCure(t *testing.T) {
	g, _ := newTurnTestGame(t)
	g.setPlayerStatus(StatusNigiri, 20)
	g.state.Player.DefensePower = 100
	if g.specialPlayerDefense() != 0 {
		t.Fatal("nigiri defense")
	}
	g.state.Player.Inventory = []Item{buildItemFromTemplate(2, 0, 0)}
	g.selectedItemIndex = 0
	g.selectedActionIndex = 0
	g.executeAction()
	runQueuedActions(g)
	if len(g.state.Player.Inventory) != 1 {
		t.Fatal("used item while nigiri")
	}
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 3)}
	satiety := g.state.Player.Satiety
	g.enqueueEnemyNormalAttack(0)
	runQueuedActions(g)
	if g.state.Player.Satiety != max(0, satiety-5) {
		t.Fatal("nigiri not eaten")
	}
	g.onTargetHit(&g.state.Player, buildItemFromTemplate(2, 0, 0), -1)
	runQueuedActions(g)
	if statusActive(g.state.Player.StatusAilments, StatusNigiri) {
		t.Fatal("medicine did not cure")
	}
	g.setPlayerStatus(StatusNigiri, 20)
	troubleCard(g)
	runQueuedActions(g)
	if statusActive(g.state.Player.StatusAilments, StatusNigiri) {
		t.Fatal("trouble cure")
	}
}

func TestSpecialNigiriPlaceholder(t *testing.T) {
	f, err := os.Open("img/player_nigiri.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 30 || img.Bounds().Dy() != 30 {
		t.Fatal("sprite dimensions")
	}
	for _, p := range [][2]int{{0, 0}, {29, 0}, {0, 29}, {29, 29}} {
		_, _, _, a := img.At(p[0], p[1]).RGBA()
		if a != 0 {
			t.Fatal("opaque corner")
		}
	}
}

func TestSpecialWandAndPotionProjectiles(t *testing.T) {
	for id := 215; id <= 221; id++ {
		g := newProjectileTestGame()
		g.state.Enemies = []Enemy{CreateEnemyByID(0, 6, 4)}
		item := buildItemFromTemplate(id, 0, 0)
		g.state.Player.Inventory = []Item{item}
		if c, ok := item.(*Cane); ok {
			if !g.useCane(c) {
				t.Fatal("wand rejected")
			}
			if c.Uses != 4 {
				t.Fatal("charge not consumed")
			}
		} else {
			g.throwWithCallbacks(item, 10)
		}
		finishThrow(t, g)
		if id == 218 {
			if len(g.state.Enemies) != 0 || len(g.state.Items) != 1 {
				t.Fatal("thrown nigiri")
			}
			continue
		}
		if id != 217 {
			kind, _ := specialItemStatus(id)
			if !statusActive(g.state.Enemies[0].StatusAilments, kind) {
				t.Fatalf("projectile %d failed", id)
			}
		}
	}
}

func TestSpecialStatusImmunityAndNoExperience(t *testing.T) {
	for _, kind := range []StatusKind{StatusNigiri, StatusBerserk, StatusInvisible, StatusInvincible} {
		g := newFoundationTestGame()
		withTestImmunity(t, "special_test_immunity", kind)
		equip(g, testRing("special_test_immunity"))
		if g.applyPlayerStatus(kind, 5, StatusSource{}) {
			t.Fatalf("immunity ignored %s", kind)
		}
	}
	g := newFoundationTestGame()
	equip(g, buildItemFromTemplate(152, 0, 0))
	if g.applyPlayerStatus(StatusNigiri, 20, StatusSource{}) {
		t.Fatal("nigiri shield ignored")
	}
	g = newFoundationTestGame()
	equip(g, buildItemFromTemplate(65, 0, 0))
	rolls := 0
	g.randHooks.statusResistance = func(int) int { rolls++; return 0 }
	if !g.applyPlayerStatus(StatusBerserk, 15, StatusSource{}) || rolls != 0 {
		t.Fatal("potion rolled shield")
	}
	if g.applyPlayerStatus(StatusNigiri, 20, StatusSource{Shield: true}) || rolls != 1 {
		t.Fatal("shield resistance ignored")
	}
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 4)}
	g.applyEnemyStatus(0, StatusDecoy, 30)
	exp := g.state.Player.ExperiencePoints
	g.applyDamageToEnemy(0, 1000)
	runQueuedActions(g)
	if len(g.state.Enemies) != 0 || g.state.Player.ExperiencePoints != exp {
		t.Fatal("decoy experience")
	}
}

func TestSpecialBerserkBotNoSoftlock(t *testing.T) {
	for _, persona := range []persona{personaRandom, personaExpert, personaAggressive} {
		g, b := newTurnTestGame(t)
		b.persona = persona
		g.setPlayerStatus(StatusBerserk, 15)
		for n := 0; n < 40 && statusActive(g.state.Player.StatusAilments, StatusBerserk); n++ {
			if err := b.step(); err != nil {
				t.Fatal(err)
			}
		}
		if statusActive(g.state.Player.StatusAilments, StatusBerserk) {
			t.Fatalf("bot %s stuck", persona)
		}
	}
}

func TestSpecialNigiriBotAndBerserkStairs(t *testing.T) {
	g, b := newTurnTestGame(t)
	b.persona = personaExpert
	g.state.Player.Inventory = []Item{buildItemFromTemplate(2, 0, 0), buildItemFromTemplate(0, 0, 0)}
	g.state.Player.Health = 10
	g.setPlayerStatus(StatusNigiri, 20)
	for n := 0; n < 30 && statusActive(g.state.Player.StatusAilments, StatusNigiri); n++ {
		if err := b.step(); err != nil {
			t.Fatal(err)
		}
	}
	if statusActive(g.state.Player.StatusAilments, StatusNigiri) {
		t.Fatal("nigiri bot stuck")
	}
	g, b = newTurnTestGame(t)
	g.state.Map[3][4].Type = "stairs"
	g.setPlayerStatus(StatusBerserk, 15)
	for n := 0; n < 25 && statusActive(g.state.Player.StatusAilments, StatusBerserk); n++ {
		if err := b.step(); err != nil {
			t.Fatal(err)
		}
	}
	if statusActive(g.state.Player.StatusAilments, StatusBerserk) {
		t.Fatal("berserk stairs stuck")
	}
}

func TestSpecialBerserkDamageMultiplierAndDecoyExpiry(t *testing.T) {
	normal, berserk := newProjectileTestGame(), newProjectileTestGame()
	for _, g := range []*Game{normal, berserk} {
		g.state.Enemies = []Enemy{CreateEnemyByID(0, 5, 4)}
		g.state.Enemies[0].AttackPower = 40
		g.state.Player.Health = 1000
		g.state.Player.MaxHealth = 1000
		g.randHooks.damage = func(int) int { return 0 }
	}
	berserk.applyEnemyStatus(0, StatusBerserk, 15)
	normal.enqueueEnemyNormalAttack(0)
	berserk.enqueueEnemyNormalAttack(0)
	runQueuedActions(normal)
	runQueuedActions(berserk)
	nd, bd := 1000-normal.state.Player.Health, 1000-berserk.state.Player.Health
	if bd < nd*3/2-1 || bd > nd*3/2+1 {
		t.Fatalf("enemy damage normal=%d berserk=%d", nd, bd)
	}
	for _, g := range []*Game{normal, berserk} {
		g.state.Enemies[0].Health = 1000
		g.state.Enemies[0].MaxHealth = 1000
		g.randHooks.attackHit = func(int) int { return 0 }
	}
	berserk.setPlayerStatus(StatusBerserk, 15)
	normal.enqueuePlayerAttack(0, nil)
	berserk.enqueuePlayerAttack(0, nil)
	runQueuedActions(normal)
	runQueuedActions(berserk)
	nd, bd = 1000-normal.state.Enemies[0].Health, 1000-berserk.state.Enemies[0].Health
	if bd < nd*3/2-1 || bd > nd*3/2+1 {
		t.Fatalf("player damage normal=%d berserk=%d", nd, bd)
	}
	g, _ := newTurnTestGame(t)
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 6, 3)}
	g.applyEnemyStatus(0, StatusDecoy, 1)
	g.decrementStatusAilments()
	runQueuedActions(g)
	before := g.state.Enemies[0].X
	g.actEnemy(0)
	runQueuedActions(g)
	if g.state.Enemies[0].X == before {
		t.Fatal("expired decoy did not resume")
	}
}

func TestSpecialUnidentifiedNamesAndRarity(t *testing.T) {
	g, _ := newTurnTestGame(t)
	for id := 219; id <= 221; id++ {
		item := buildItemFromTemplate(id, 0, 0).(*Cane)
		label, unknown := g.inventoryItemLabel(item)
		if !unknown || label == itemTemplates[id].Name {
			t.Fatal("missing unidentified name")
		}
		item.SetIdentified(true)
		label, unknown = g.inventoryItemLabel(item)
		if unknown || !strings.Contains(label, itemTemplates[id].Name) {
			t.Fatal("identified name missing")
		}
	}
	for _, id := range []int{2, 3, 17, 18, 19, 28, 29, 215, 216, 218} {
		if itemSpawnWeights[id] != 4*itemSpawnWeights[217] {
			t.Fatal("invincible is not rarer")
		}
	}
}

func TestSpecialWandDirectEffect(t *testing.T) {
	for id := 219; id <= 221; id++ {
		g, _ := newTurnTestGame(t)
		g.state.Enemies = []Enemy{CreateEnemyByID(0, 6, 3)}
		g.TargetEnemyIndex = 0
		buildItemFromTemplate(id, 0, 0).Use(g)
		runQueuedActions(g)
		kind, _ := specialItemStatus(id)
		if !statusActive(g.state.Enemies[0].StatusAilments, kind) {
			t.Fatalf("direct wand %d", id)
		}
	}
}

func TestSpecialDecoyCannotReturnProjectiles(t *testing.T) {
	g := newProjectileTestGame()
	g.state.Enemies = []Enemy{CreateEnemyByID(123, 6, 4)}
	if enemyProjectileReturnPercent(g.state.Enemies[0]) == 0 {
		t.Fatal("fixture is not a reflector")
	}
	g.applyEnemyStatus(0, StatusDecoy, 30)
	if enemyProjectileReturnPercent(g.state.Enemies[0]) != 0 {
		t.Fatal("decoy reflected")
	}
}

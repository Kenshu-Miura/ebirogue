//go:build !test

package main

import "testing"

// wave11-B の expert の段8の薬・杖と段9の壺の使い方のテスト（bot_expert_wave11_test.go）。

// newLosingFight は隣に強い敵が2体いて、殴り合うと倒れる見積もりの盤面を作る。
func newLosingFight(t *testing.T) (*Game, *bot) {
	t.Helper()
	g, b := newBotFoodTestGame(t, personaExpert)
	player := &g.state.Player
	player.X = 4
	player.Health, player.MaxHealth = 100, 100
	for _, pos := range [][2]int{{5, 3}, {5, 2}} {
		e := newActiveEnemy(0, pos[0], pos[1])
		e.Health, e.MaxHealth, e.AttackPower = 300, 300, 40
		g.state.Enemies = append(g.state.Enemies, e)
	}
	return g, b
}

// 強化の薬は落ち着いているときには飲まず、殴り合うと倒れる見積もりになったら飲む。
func TestExpertKeepsBuffPotionUntilLosing(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	g.state.Player.Inventory = []Item{buildItemFromTemplate(potionHasteID, 0, 0)}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if countInventoryID(g, potionHasteID) != 1 {
		t.Fatalf("drank the haste potion without a reason: act=%s", b.act)
	}

	g, b = newLosingFight(t)
	g.state.Player.Inventory = []Item{buildItemFromTemplate(potionHasteID, 0, 0)}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if countInventoryID(g, potionHasteID) != 0 || g.state.Player.StatusAilments.Haste == 0 {
		t.Fatalf("did not drink the haste potion: act=%s", b.act)
	}
}

// 倍速が効けば、殴り合いの見積もりは軽くなる（statusest）。
func TestFightEstimateCountsStatus(t *testing.T) {
	g, b := newLosingFight(t)
	before := b.fightDamageEstimate(g.state.Enemies)
	g.state.Player.StatusAilments.Haste = 10
	g.state.Enemies[0].StatusAilments.AttackDown = 10
	after := b.fightDamageEstimate(g.state.Enemies)
	if after >= before*0.5 {
		t.Fatalf("estimate did not drop: before=%.1f after=%.1f", before, after)
	}
}

// HP が少なく回復の道具が無ければ、無敵の薬を最後の手段に飲む。
func TestExpertDrinksInvincibleAsLastResort(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	player := &g.state.Player
	player.X = 4
	player.Health, player.MaxHealth = 20, 100
	player.Inventory = []Item{buildItemFromTemplate(potionInvincibleID, 0, 0)}
	g.state.Enemies = []Enemy{newActiveEnemy(0, 5, 3)}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if countInventoryID(g, potionInvincibleID) != 0 || player.StatusAilments.Invincible == 0 {
		t.Fatalf("did not drink the invincible potion: act=%s", b.act)
	}
}

// 一撃で倒せない強い敵には鈍足の杖を振る（効いた後はもう振らない）。
func TestExpertSlowsStrongEnemy(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	player := &g.state.Player
	player.X = 4
	player.Health, player.MaxHealth = 100, 100
	cane := buildItemFromTemplate(caneSlowID, 0, 0).(*Cane)
	player.Inventory = []Item{cane}
	e := newActiveEnemy(0, 5, 3)
	e.Health, e.MaxHealth, e.AttackPower = 300, 300, 40
	g.state.Enemies = []Enemy{e}
	uses := cane.Uses
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if b.act != "debuff" || cane.Uses != uses-1 || g.state.Enemies[0].StatusAilments.Slow == 0 {
		t.Fatalf("did not slow the enemy: act=%s uses=%d slow=%d", b.act, cane.Uses, g.state.Enemies[0].StatusAilments.Slow)
	}
	if done, err := b.expertDebuffAt(1, 0, g.state.Enemies[0]); done || err != nil {
		t.Fatalf("slowed twice: %v %v", done, err)
	}
}

// 退化の杖は基本種には振らない。
func TestExpertDoesNotDevolveBaseEnemy(t *testing.T) {
	base := CreateEnemyByID(0, 0, 0)
	if canDevolveEnemy(base) {
		t.Fatal("base enemy should not devolve")
	}
	upper, ok := MonsterLevelUpTable[0]
	if !ok {
		t.Skip("no upper form")
	}
	if !canDevolveEnemy(CreateEnemyByID(upper, 0, 0)) {
		t.Fatal("upper form should devolve")
	}
}

// 殴り合うと倒れる見積もりなら、身代わりの杖で隣の痛い敵を身代わりにする。
func TestExpertUsesDecoyWhenLosing(t *testing.T) {
	g, b := newLosingFight(t)
	g.state.Player.Inventory = []Item{buildItemFromTemplate(caneDecoyID, 0, 0)}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	decoys := 0
	for _, e := range g.state.Enemies {
		if e.StatusAilments.Decoy > 0 {
			decoys++
		}
	}
	if b.act != "decoy" || decoys != 1 {
		t.Fatalf("did not make a decoy: act=%s decoys=%d", b.act, decoys)
	}
}

// 合成の壺で、装備中の武器に予備の武器の印と強化値を移して装備し直す。
func TestExpertSynthesizesSpareWeapon(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	player := &g.state.Player
	weapon := buildItemFromTemplate(126, 0, 0).(*Weapon) // カタナ
	weapon.Sharpness, weapon.Cursed = 1, false
	spare := buildItemFromTemplate(129, 0, 0).(*Weapon) // 甲殻斬り
	spare.Sharpness, spare.Cursed = 2, false
	pot := buildItemFromTemplate(synthesisPotID, 0, 0).(*Pot)
	pot.Capacity, pot.Contents = 3, nil
	player.Inventory = []Item{weapon, spare, pot}
	player.EquippedWeapon = weapon
	stepUntil(t, b, 8, func() bool { return b.potJob == nil && player.EquippedWeapon == weapon && len(player.Inventory) == 2 })
	if player.EquippedWeapon != weapon || !hasEquipmentAbility(weapon.Abilities, AbilityShellSlayer) || weapon.Sharpness != 3 {
		t.Fatalf("not synthesized: equipped=%v abilities=%v sharpness=%d act=%s", player.EquippedWeapon == weapon, weapon.Abilities, weapon.Sharpness, b.act)
	}
	if countInventoryID(g, 129) != 0 || len(pot.Contents) != 0 || b.windowOpen() {
		t.Fatalf("leftovers: spare=%d pot=%d", countInventoryID(g, 129), len(pot.Contents))
	}
}

// 移すと損な印（使い捨てなど）の予備や呪われた予備は合成しない。
func TestSynthesisSkipsHarmfulDonor(t *testing.T) {
	base := buildItemFromTemplate(126, 0, 0).(*Weapon)
	cursed := buildItemFromTemplate(129, 0, 0).(*Weapon)
	cursed.Cursed = true
	if _, ok := synthesisGain(base, cursed); ok {
		t.Fatal("cursed donor accepted")
	}
	harmful := buildItemFromTemplate(126, 0, 0).(*Weapon)
	harmful.Cursed, harmful.Sharpness = false, 3
	harmful.Abilities = []EquipmentAbilityID{AbilityDisposable}
	if _, ok := synthesisGain(base, harmful); ok {
		t.Fatal("harmful ability accepted")
	}
}

// 強化の壺に装備中の武器を入れ、壁際で壺を割って拾い直し、装備し直す。
func TestExpertStrengthensWeaponWithPot(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	player := &g.state.Player // (1,3)。左は壁
	weapon := buildItemFromTemplate(126, 0, 0).(*Weapon)
	weapon.Sharpness, weapon.Cursed = 1, false
	pot := buildItemFromTemplate(strengthPotID, 0, 0).(*Pot)
	pot.Capacity, pot.Contents = 3, nil
	player.Inventory = []Item{weapon, pot}
	player.EquippedWeapon = weapon
	stepUntil(t, b, 12, func() bool { return player.EquippedWeapon == weapon && countInventoryID(g, strengthPotID) == 0 })
	if player.EquippedWeapon != weapon || weapon.Sharpness != 2 || countInventoryID(g, strengthPotID) != 0 {
		t.Fatalf("not strengthened: equipped=%v sharpness=%d pot=%d act=%s", player.EquippedWeapon == weapon, weapon.Sharpness, countInventoryID(g, strengthPotID), b.act)
	}
}

// 冷蔵の壺にも予備の食料を入れる。
func TestExpertStoresFoodInCoolPot(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	player := &g.state.Player
	pot := buildItemFromTemplate(coolPotID, 0, 0).(*Pot)
	pot.Capacity, pot.Contents = 3, nil
	player.Inventory = []Item{pot, buildItemFromTemplate(testFoodID, 0, 0), buildItemFromTemplate(testFoodID, 0, 0)}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if len(pot.Contents) != 1 || countInventoryID(g, testFoodID) != 1 {
		t.Fatalf("spare food not stored: pot=%d act=%s", len(pot.Contents), b.act)
	}
}

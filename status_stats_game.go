//go:build !test

package main

import "fmt"

func (p *Player) recoveredHP(amount int, full bool) int {
	if statusActive(p.StatusAilments, StatusNoHealing) {
		return p.Health
	}
	return recoveredValue(p.Health, p.MaxHealth, amount, full)
}

func characterStatuses(c Character) StatusAilments {
	switch c := c.(type) {
	case *Player:
		return c.StatusAilments
	case *Enemy:
		return c.StatusAilments
	}
	return StatusAilments{}
}

var statItemKinds = map[int][]StatusKind{
	205: {StatusHaste}, 206: {StatusAttackUp}, 207: {StatusDefenseUp},
	208: {StatusAttackDown}, 210: {StatusSlow},
	211: {StatusAttackDown, StatusDefenseDown}, 213: {StatusHaste},
}

func statItemTurns(kind StatusKind) int {
	if kind == StatusHaste {
		return 20
	}
	return 30
}

// levelDownEnemy は上位種表を逆引きし、位置・状態・盗品などを保ったまま退化させる。
// HP は新しい最大値までに制限し、退化による回復は行わない。
func (g *Game) levelDownEnemy(index int) bool {
	if index < 0 || index >= len(g.state.Enemies) {
		return false
	}
	e := &g.state.Enemies[index]
	if MonsterDefinitions[e.ID].IsBoss {
		return false
	}
	previous := -1
	for id, next := range MonsterLevelUpTable {
		if next == e.ID && (previous < 0 || id < previous) {
			previous = id
		}
	}
	if previous < 0 {
		return false
	}
	fresh := CreateEnemyByID(previous, e.X, e.Y)
	fresh.Health = min(e.Health, fresh.MaxHealth)
	// 種類の能力値・固有能力だけを差し替える（可変状態は維持）。
	e.ID, e.Char, e.Name, e.Type = fresh.ID, fresh.Char, fresh.Name, fresh.Type
	e.Health, e.MaxHealth = fresh.Health, fresh.MaxHealth
	e.AttackPower, e.DefensePower, e.ExperiencePoints = fresh.AttackPower, fresh.DefensePower, fresh.ExperiencePoints
	e.SpecialAttack, e.SpecialAttackProbability = fresh.SpecialAttack, fresh.SpecialAttackProbability
	e.RangedAttack, e.SpecialMovement, e.Disguise, e.Traits = fresh.RangedAttack, fresh.SpecialMovement, fresh.Disguise, fresh.Traits
	syncEnemyAppearance(e)
	return true
}

func (g *Game) applyStatItem(id int, player bool, index int) {
	if id == 209 || id == 212 {
		if player {
			if guard, blocked := g.playerImmunitySource(StatusLevelDown); blocked {
				g.EnqueueMessage(statusBlockedMessage(StatusLevelDown, guard), statusMessageDuration)
				return
			}
			g.state.Player.setLevel(g.state.Player.Level - 1)
			g.state.Player.ExperiencePoints = expForLevel(g.state.Player.Level)
			g.EnqueueMessage(fmt.Sprintf("海老さんのレベルは%dになった", g.state.Player.Level), statusMessageDuration)
		} else if g.levelDownEnemy(index) {
			g.EnqueueMessage("敵は退化した", statusMessageDuration)
		} else {
			g.EnqueueMessage("しかし何も起こらなかった", statusMessageDuration)
		}
		return
	}
	for _, kind := range statItemKinds[id] {
		if player {
			g.applyPlayerStatus(kind, statItemTurns(kind), StatusSource{})
		} else if index >= 0 && index < len(g.state.Enemies) {
			cancelled := statusActive(g.state.Enemies[index].StatusAilments, oppositeStatStatus(kind))
			if g.applyEnemyStatus(index, kind, statItemTurns(kind)) {
				message := fmt.Sprintf("%sは%sになった", g.enemyDisplayName(g.state.Enemies[index].Name), statusKindLabel(kind))
				if cancelled {
					message = fmt.Sprintf("%sの%sが解けた", g.enemyDisplayName(g.state.Enemies[index].Name), statusKindLabel(oppositeStatStatus(kind)))
				}
				g.EnqueueMessage(message, statusMessageDuration)
			}
		}
	}
}

func drinkStatPotion(g *Game) {
	item, inventory := determineItemSource(g)
	id := item.GetID()
	g.Enqueue(Action{Duration: 0.4, Message: item.GetName() + "を飲んだ", Execute: func(g *Game) { g.applyStatItem(id, true, -1) }})
	removeUsedItem(g, inventory)
}

// queueStatItemHit は投げた薬・杖と魔法弾を同じ命中経路で扱う。
func (g *Game) queueStatItemHit(target Character, item Item, index int) bool {
	id := item.GetID()
	if id < 205 || id > 213 {
		return false
	}
	_, player := target.(*Player)
	g.Enqueue(Action{Duration: 0.4, Message: item.GetName() + "が当たった", Execute: func(g *Game) {
		g.applyStatItem(id, player, index)
		g.isActioned = true
	}})
	return true
}

func statCaneEffect(g *Game) {
	// 魔法弾の命中は queueStatItemHit が扱う。UseActions は既存の杖の使用経路用。
}

var noHealingTrapEffect = func(g *Game) {
	guard, blocked := g.playerStatusGuard(StatusNoHealing)
	g.enqueuePlayerStatus(StatusNoHealing, 30, StackDefault, guard, blocked)
}

func init() {
	expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "能力値と倍速", Lines: []string{
		"倍速の薬：２０ターン、２回に１回敵が動く",
		"剛力・堅守の薬：３０ターン攻撃・防御強化",
		"弱気の薬：攻撃低下。敵へ投げると有効",
		"弱体の杖：敵の攻撃と防御を３０ターン低下",
		"鈍足の杖：敵を３０ターン鈍足にする",
		"加速の杖：敵を２０ターン倍速にする",
		"退化の薬・杖：１段下へ。ボスには無効",
		"逆の効果は打ち消し合う",
		"回復封じの罠：３０ターンＨＰ回復不能",
	}})
}

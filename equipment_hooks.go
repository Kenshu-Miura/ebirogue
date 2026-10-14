//go:build !test

package main

import "fmt"

// 装備能力のフック（content-expansion-plan.md の F1）。
//
// 武器・盾・指輪の能力は能力 ID ごとに AbilityHooks を登録し、装備中の全装備品
// （武器・盾・指輪2つ）の能力からフックを呼ぶ。同じ能力を2つの指輪で持つ場合も
// フックは1回だけ呼ばれ、引数 stacks に重ね数（2）が入る。
// フックの中の乱数は g.abilityIntn() を使う（テストで固定できる）。

// PlayerAttack は通常攻撃1回（三方向攻撃なら対象1体ぶん）の命中・ダメージの条件。
// BeforePlayerAttack のフックが書き換える。
type PlayerAttack struct {
	Weapon              *Weapon // 装備中の武器（素手なら nil）
	Target              *Enemy  // 攻撃対象（g.state.Enemies の要素。AfterPlayerMiss では実行時に引き直す）
	Accuracy            int     // 命中率（%）
	SureHit             bool    // 必ず命中し、敵の回避も無視する
	EnemyEvasion        int     // 敵が攻撃をかわす確率（%、F4）。0ならかわさない
	Multiplier          float64 // ダメージ倍率（特効などを乗算で合成する）
	CriticalDenominator int     // 1/N で会心の一撃。小さいほど出やすい
	ForceCritical       bool    // 必ず会心の一撃にする
	MessagePrefix       string  // 命中時のメッセージの先頭に付ける文（「特効！」など）
}

// PlayerAttackHit は命中した通常攻撃の結果。AfterPlayerHit・AfterEnemyDefeated に渡す。
type PlayerAttackHit struct {
	Weapon   *Weapon
	EnemyID  int
	Enemy    Enemy // ダメージを与える直前の敵の写し（倒した後も参照できる）
	Damage   int
	Critical bool
	Defeated bool
}

// EnemyAttack は敵の通常攻撃1回の条件。BeforeEnemyAttack のフックが書き換える。
type EnemyAttack struct {
	Enemy           *Enemy
	CriticalPercent int     // 痛恨の一撃の確率（%、F4）
	Multiplier      float64 // ダメージ倍率
}

// IncomingDamage は盾の回避・反射・軽減の後に残ったプレイヤーへのダメージ。
// ModifyIncomingDamage のフックが Damage を書き換え、実際に受けるときの処理を Effects に積む。
type IncomingDamage struct {
	Damage   int
	Kind     EnemyDamageKind
	Delivery EnemyAttackDelivery
	Attacker *Enemy // 敵以外（罠など）なら nil
	Critical bool   // 敵の痛恨の一撃
	Effects  []func(g *Game)
}

// AbilityHooks は能力1つぶんの処理。使わない項目は nil のままにする。
type AbilityHooks struct {
	// BeforePlayerAttack は命中判定の前に呼ばれ、命中率・倍率・会心を変えられる。
	BeforePlayerAttack func(g *Game, attack *PlayerAttack, stacks int)
	// AfterPlayerMiss は通常攻撃が外れた（敵にかわされた場合を含む）ときに呼ばれる。
	AfterPlayerMiss func(g *Game, attack *PlayerAttack, stacks int)
	// AfterPlayerHit は通常攻撃のダメージを与えた直後（Action の Execute 内）に呼ばれる。
	AfterPlayerHit func(g *Game, hit PlayerAttackHit, stacks int)
	// AfterEnemyDefeated は通常攻撃で敵を倒した直後に呼ばれる（AfterPlayerHit の後）。
	AfterEnemyDefeated func(g *Game, hit PlayerAttackHit, stacks int)
	// BeforeEnemyAttack は敵の通常攻撃のダメージ計算の前に呼ばれる（痛恨の確率・倍率）。
	BeforeEnemyAttack func(g *Game, attack *EnemyAttack, stacks int)
	// ModifyIncomingDamage は被ダメージが確定する前に呼ばれる（盾の軽減・小銭の消費など）。
	ModifyIncomingDamage func(g *Game, incoming *IncomingDamage, stacks int)
	// PreventDeath は HP が0以下になるダメージを受ける直前に呼ばれ、true なら HP1 で耐える。
	PreventDeath func(g *Game, damage int, source DamageSource, stacks int) bool
	// OnTurn はプレイヤーのターン経過（IncrementMoveCount）の最後に毎ターン呼ばれる。
	OnTurn func(g *Game, stacks int)
}

// equipmentAbilityHooks は能力 ID ごとのフック。新しい能力は init() で registerAbilityHooks する。
var equipmentAbilityHooks = map[EquipmentAbilityID]AbilityHooks{}

// registerAbilityHooks は能力のフックを登録し、元の登録を返す（テストで戻すため）。
func registerAbilityHooks(id EquipmentAbilityID, hooks AbilityHooks) (previous AbilityHooks, existed bool) {
	previous, existed = equipmentAbilityHooks[id]
	equipmentAbilityHooks[id] = hooks
	return previous, existed
}

// equippedAbilityLists は装備品ごとの能力一覧を、武器・盾・指輪1・指輪2の順で返す。
func equippedAbilityLists(player *Player) [][]EquipmentAbilityID {
	lists := make([][]EquipmentAbilityID, 0, 4)
	if player.EquippedWeapon != nil {
		lists = append(lists, player.EquippedWeapon.Abilities)
	}
	if player.EquippedArmor != nil {
		lists = append(lists, player.EquippedArmor.Abilities)
	}
	for _, accessory := range player.EquippedAccessories {
		if accessory != nil {
			lists = append(lists, accessory.Abilities)
		}
	}
	return lists
}

// playerAbilityCount は装備中の装備品のうち ability を持つものの数（重ね数）を返す。
func (g *Game) playerAbilityCount(ability EquipmentAbilityID) int {
	return countEquipmentAbility(ability, equippedAbilityLists(&g.state.Player)...)
}

// playerHasAbility は装備中のどれかが ability を持つかを返す。
func (g *Game) playerHasAbility(ability EquipmentAbilityID) bool {
	return g.playerAbilityCount(ability) > 0
}

// forEachEquippedAbilityHook は装備中の能力のうちフックが登録されたものを、重ね数つきで1回ずつ渡す。
func (g *Game) forEachEquippedAbilityHook(fn func(hooks AbilityHooks, stacks int)) {
	if len(equipmentAbilityHooks) == 0 {
		return
	}
	lists := equippedAbilityLists(&g.state.Player)
	for _, ability := range uniqueEquipmentAbilities(lists...) {
		if hooks, ok := equipmentAbilityHooks[ability]; ok {
			fn(hooks, countEquipmentAbility(ability, lists...))
		}
	}
}

func (g *Game) runBeforePlayerAttackHooks(attack *PlayerAttack) {
	g.forEachEquippedAbilityHook(func(h AbilityHooks, stacks int) {
		if h.BeforePlayerAttack != nil {
			h.BeforePlayerAttack(g, attack, stacks)
		}
	})
}

func (g *Game) runAfterPlayerMissHooks(attack *PlayerAttack) {
	g.forEachEquippedAbilityHook(func(h AbilityHooks, stacks int) {
		if h.AfterPlayerMiss != nil {
			h.AfterPlayerMiss(g, attack, stacks)
		}
	})
}

func (g *Game) runAfterPlayerHitHooks(hit PlayerAttackHit) {
	g.forEachEquippedAbilityHook(func(h AbilityHooks, stacks int) {
		if h.AfterPlayerHit != nil {
			h.AfterPlayerHit(g, hit, stacks)
		}
	})
	if !hit.Defeated {
		return
	}
	g.forEachEquippedAbilityHook(func(h AbilityHooks, stacks int) {
		if h.AfterEnemyDefeated != nil {
			h.AfterEnemyDefeated(g, hit, stacks)
		}
	})
}

func (g *Game) runBeforeEnemyAttackHooks(attack *EnemyAttack) {
	g.forEachEquippedAbilityHook(func(h AbilityHooks, stacks int) {
		if h.BeforeEnemyAttack != nil {
			h.BeforeEnemyAttack(g, attack, stacks)
		}
	})
}

func (g *Game) runModifyIncomingDamageHooks(incoming *IncomingDamage) {
	g.forEachEquippedAbilityHook(func(h AbilityHooks, stacks int) {
		if h.ModifyIncomingDamage != nil && incoming.Damage > 0 {
			h.ModifyIncomingDamage(g, incoming, stacks)
		}
	})
	incoming.Damage = max(0, incoming.Damage)
}

// runPreventDeathHooks は致命的なダメージを1つのフックが防げば true を返す（残りのフックは呼ばない）。
func (g *Game) runPreventDeathHooks(damage int, source DamageSource) bool {
	prevented := false
	g.forEachEquippedAbilityHook(func(h AbilityHooks, stacks int) {
		if !prevented && h.PreventDeath != nil {
			prevented = h.PreventDeath(g, damage, source, stacks)
		}
	})
	return prevented
}

func (g *Game) runTurnHooks() {
	g.forEachEquippedAbilityHook(func(h AbilityHooks, stacks int) {
		if h.OnTurn != nil {
			h.OnTurn(g, stacks)
		}
	})
}

// playerImmunitySource は状態異常 kind を確実に無効化する装備品の名前を返す（無ければ ok=false）。
// 能力と無効化する状態異常の対応は statusImmunities（equipment_abilities.go）。
func (g *Game) playerImmunitySource(kind StatusKind) (name string, ok bool) {
	player := &g.state.Player
	if w := player.EquippedWeapon; w != nil && abilitiesGrantImmunity(w.Abilities, kind) {
		return w.Name, true
	}
	if a := player.EquippedArmor; a != nil && abilitiesGrantImmunity(a.Abilities, kind) {
		return a.Name, true
	}
	for _, accessory := range player.EquippedAccessories {
		if accessory != nil && abilitiesGrantImmunity(accessory.Abilities, kind) {
			return accessory.Name, true
		}
	}
	return "", false
}

// playerImmunityMessage は状態異常 kind を装備能力で無効化できるとき、そのメッセージを返す。
func (g *Game) playerImmunityMessage(kind StatusKind) (string, bool) {
	name, ok := g.playerImmunitySource(kind)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%sが%sを防いだ", name, statusKindLabel(kind)), true
}

// playerImmuneTo は装備能力で状態異常 kind を確実に無効化できるかを返す。乱数は使わない。
func (g *Game) playerImmuneTo(kind StatusKind) bool {
	_, ok := g.playerImmunitySource(kind)
	return ok
}

// playerBlocksStatus は playerStatusGuard のうち、防げたかどうかだけを返す。
func (g *Game) playerBlocksStatus(kind StatusKind) bool {
	_, blocked := g.playerStatusGuard(kind)
	return blocked
}

// playerStatusGuard は状態異常 kind を防げるかを判定し、防いだ装備品の名前を返す。
// 能力による無効化（確実）を先に見て、無ければ状態異常耐性の盾（50%）を判定する。
// 以前の playerResistsStatus と同じく、盾の判定をしたときだけ乱数を1回使う。
func (g *Game) playerStatusGuard(kind StatusKind) (guard string, blocked bool) {
	if name, ok := g.playerImmunitySource(kind); ok {
		return name, true
	}
	if g.playerResistsStatus() {
		return g.state.Player.EquippedArmor.Name, true
	}
	return "", false
}

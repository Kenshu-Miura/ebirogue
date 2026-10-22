//go:build !test

package main

// wave11-B: expert の段8の薬・杖と段9の壺の使い方（bot_expert_test.go・bot_tactics_test.go から呼ぶ）。
// 道具の正体は今までの BOT と同じくテンプレート ID で見分ける（識別の有無で扱いを変えない）。
//
// 計測用の botFeature の名前（-bot.off=<名前,...> で個別に切れる）:
//   buffpotion  強化の薬（倍速・剛力・堅守）をすぐ飲まずにとっておき、殴り合うと倒れる見積もり・ボス戦・
//               モンスターハウスで囲まれたときに飲む。無敵の薬はそのときと、回復の道具が無いときの最後の手段に飲む。
//   debuffcane  一撃で倒せない強い敵（ボスを含む）に鈍足・弱体・退化の杖を振り、弱気・退化の薬を投げる。
//   special     身代わりの杖・にぎりの薬（敵に投げる）・暴走の杖・暴れ薬（敵に投げる）・透明の薬（逃げる）。
//   statusest   殴り合いの見積もり（fightDamageEstimate）に倍速・鈍足・攻撃力と防御力の上下・無敵・身代わりを入れる。
//   synthesis   合成の壺で、装備中の武器・盾に予備の印と強化値を移す（食料は入れない）。
//   strengthpot 強化の壺に装備中の武器・盾を入れ、壺を割って取り戻す。
//   coolpot     冷蔵の壺にも予備の食料を入れる（保存の壺と同じ）。

const (
	potionHasteID      = 205 // 倍速の薬
	potionMightID      = 206 // 剛力の薬
	potionGuardID      = 207 // 堅守の薬
	potionTimidID      = 208 // 弱気の薬（投げると敵の攻撃力が下がる）
	potionDevolveID    = 209 // 退化の薬（投げると敵が退化する）
	caneSlowID         = 210 // 鈍足の杖
	caneWeakenID       = 211 // 弱体の杖
	caneDevolveID      = 212 // 退化の杖
	potionBerserkID    = 215 // 暴れ薬（敵に投げると仲間どうしで争う）
	potionInvisibleID  = 216 // 透明の薬
	potionInvincibleID = 217 // 無敵の薬
	potionNigiriID     = 218 // にぎりの薬（敵に投げるとウインナーに変える）
	caneDecoyID        = 219 // 身代わりの杖
	caneBerserkID      = 221 // 暴走の杖
	strengthPotID      = 227 // 強化の壺
	coolPotID          = 232 // 冷蔵の壺
)

// isFoodPot は予備の食料を入れる壺かどうか（保存の壺。合成の壺は synthesis を切ったときだけ、冷蔵の壺は coolpot のとき）。
func isFoodPot(pot *Pot) bool {
	switch pot.GetID() {
	case storagePotID:
		return true
	case synthesisPotID:
		return !botFeature("synthesis")
	case coolPotID:
		return botFeature("coolpot")
	}
	return false
}

// isFoodSourcePot は中の食料を数えて取り出してよい壺（直接取り出せる壺のうち、食料を入れる可能性があるもの）。
func isFoodSourcePot(pot *Pot) bool {
	return pot.GetID() == storagePotID || pot.GetID() == synthesisPotID || pot.GetID() == coolPotID
}

// --- 強化の薬 ---

// expertDrinkBuff は効いていない強化の薬を1つ飲む（無敵 → 倍速 → 堅守 → 剛力）。無敵は withInvincible のときだけ。
func (b *bot) expertDrinkBuff(withInvincible bool) (bool, error) {
	if !botFeature("buffpotion") {
		return false, nil
	}
	s := b.g.state.Player.StatusAilments
	if s.Nigiri > 0 || s.Berserk > 0 || s.Invincible > 0 {
		return false, nil
	}
	if withInvincible {
		if used, err := b.useByID("使う", potionInvincibleID); used || err != nil {
			b.act = "buffPotion"
			return used, err
		}
	}
	for _, c := range []struct {
		id     int
		active bool
	}{{potionHasteID, s.Haste > 0}, {potionGuardID, s.DefenseUp > 0}, {potionMightID, s.AttackUp > 0}} {
		if c.active {
			continue
		}
		if used, err := b.useByID("使う", c.id); used || err != nil {
			b.act = "buffPotion"
			return used, err
		}
	}
	return false, nil
}

// expertBuffMoments はボス戦の開始（ボスが2マス以内）とモンスターハウスで囲まれたとき（隣に2体以上）に強化の薬を飲む。
func (b *bot) expertBuffMoments(adjacent []Enemy) (bool, error) {
	if !botFeature("buffpotion") {
		return false, nil
	}
	p := &b.g.state.Player
	low := p.Health*100 < p.MaxHealth*50
	if b.bossAlive() {
		if boss, ok := b.nearestBoss(); ok && max(abs(boss.X-p.X), abs(boss.Y-p.Y)) <= 2 {
			return b.expertDrinkBuff(low)
		}
	}
	if len(adjacent) >= 2 && b.g.inMonsterHouse() {
		return b.expertDrinkBuff(low)
	}
	return false, nil
}

// expertLastResort は HP が少なく回復の道具が無いときの最後の手段（無敵の薬・透明の薬）。使ったら true。
func (b *bot) expertLastResort(adjacent []Enemy) (bool, error) {
	p := &b.g.state.Player
	if botFeature("buffpotion") && p.StatusAilments.Invincible == 0 && (len(adjacent) > 0 || b.inRangedFire(p.X, p.Y)) {
		if used, err := b.useByID("使う", potionInvincibleID); used || err != nil {
			b.act = "invincible"
			return used, err
		}
	}
	if botFeature("special") && len(adjacent) == 0 && p.StatusAilments.Invisible == 0 && !b.bossAlive() &&
		len(b.enemiesWithin(5)) > 0 {
		if used, err := b.useByID("使う", potionInvisibleID); used || err != nil {
			b.act = "invisible"
			return used, err
		}
	}
	return false, nil
}

// --- 状態込みの殴り合いの見積もり（statusest） ---

// enemyDamageFactor は敵 e から受けるダメージに掛かる状態の倍率。
func (b *bot) enemyDamageFactor(e Enemy) float64 {
	if !botFeature("statusest") {
		return 1
	}
	ps, es := b.g.state.Player.StatusAilments, e.StatusAilments
	if ps.Invincible > 1 || es.Decoy > 0 {
		return 0
	}
	f := 1.0
	if es.AttackUp > 0 {
		f *= 1.5
	}
	if es.AttackDown > 0 {
		f *= 0.5
	}
	if ps.DefenseUp > 0 {
		f *= 0.67
	}
	if ps.DefenseDown > 0 {
		f *= 1.5
	}
	if ps.Haste > 0 {
		f *= 0.5 // 敵の行動が1回おきに飛ぶ
	}
	if es.Slow > 0 && e.SpecialMovement != SpecialMovementDoubleSpeed && es.Haste == 0 {
		f *= 0.5
	}
	return f
}

// playerDamageFactor は敵 e へ与えるダメージに掛かる状態の倍率。
func (b *bot) playerDamageFactor(e Enemy) float64 {
	if !botFeature("statusest") {
		return 1
	}
	ps, es := b.g.state.Player.StatusAilments, e.StatusAilments
	f := 1.0
	if ps.AttackUp > 0 {
		f *= 1.5
	}
	if ps.AttackDown > 0 {
		f *= 0.5
	}
	if es.DefenseDown > 0 {
		f *= 1.5
	}
	if es.DefenseUp > 0 {
		f *= 0.67
	}
	return f
}

// --- 敵を弱める杖・薬（debuffcane） ---

// canDevolveEnemy は退化の杖・薬が効く敵（人に見える姿が基本種・ボスでない）かどうか。
func canDevolveEnemy(e Enemy) bool {
	id := apparentEnemyID(e)
	if MonsterDefinitions[id].IsBoss || MonsterDefinitions[e.ID].IsBoss {
		return false
	}
	for _, next := range MonsterLevelUpTable {
		if next == id {
			return true
		}
	}
	return false
}

// debuffTarget は弱める道具を使う価値のある敵（強く、一撃で倒せず、打ち返さず、動いている）かどうか。
func (b *bot) debuffTarget(e Enemy) bool {
	return activeEnemy(e) && e.StatusAilments.Decoy == 0 && !isReflectorEnemy(e) && !b.oneHitKill(e) && b.isStrongEnemy(e)
}

// expertDebuffAt は (dx, dy) の方向の直線上の敵 e に、まだ効いていない弱める道具を1つ使う。使ったら true。
func (b *bot) expertDebuffAt(dx, dy int, e Enemy) (bool, error) {
	if !botFeature("debuffcane") || !b.debuffTarget(e) {
		return false, nil
	}
	s := e.StatusAilments
	use := func(index int, action string) (bool, error) {
		b.act = "debuff"
		return b.faceAndUse(dx, dy, index, action)
	}
	if canDevolveEnemy(e) {
		if i := b.findCane(caneDevolveID); i >= 0 {
			return use(i, "使う")
		}
		if i := b.findInventoryID(potionDevolveID); i >= 0 {
			return use(i, "投げる")
		}
	}
	if s.Slow == 0 {
		if i := b.findCane(caneSlowID); i >= 0 {
			return use(i, "使う")
		}
	}
	if s.AttackDown == 0 {
		if i := b.findCane(caneWeakenID); i >= 0 {
			return use(i, "使う")
		}
		if i := b.findInventoryID(potionTimidID); i >= 0 {
			return use(i, "投げる")
		}
	}
	return false, nil
}

// expertDebuffInLine は直線上（杖の届く距離）の強い敵に弱める道具を使う。使ったら true。
func (b *bot) expertDebuffInLine() (bool, error) {
	if !botFeature("debuffcane") {
		return false, nil
	}
	dx, dy, e, ok := b.targetInLine(caneRange, b.debuffTarget)
	if !ok {
		return false, nil
	}
	return b.expertDebuffAt(dx, dy, e)
}

// weakenedEnough は弱めた後の強い敵を、道具を使わずに殴り合えるかどうか（状態込みの見積もりが HP の6割未満）。
func (b *bot) weakenedEnough(e Enemy) bool {
	if !botFeature("debuffcane") || !botFeature("statusest") {
		return false
	}
	s := e.StatusAilments
	if s.Slow == 0 && s.AttackDown == 0 && s.DefenseDown == 0 {
		return false
	}
	return b.fightDamageEstimate([]Enemy{e}) < float64(b.g.state.Player.Health)*0.6
}

// --- 特殊な状態の道具（special） ---

// expertDecoy は隣の一番痛い敵（ボス以外）を身代わりの杖で身代わりにする（周りの敵がそちらを襲う）。振ったら true。
func (b *bot) expertDecoy(adjacent []Enemy) (bool, error) {
	if !botFeature("special") {
		return false, nil
	}
	i := b.findCane(caneDecoyID)
	if i < 0 {
		return false, nil
	}
	p := b.g.state.Player
	best, found := Enemy{}, false
	for _, e := range adjacent {
		if MonsterDefinitions[e.ID].IsBoss || isReflectorEnemy(e) || e.StatusAilments.Decoy > 0 ||
			!b.canStep(p.X, p.Y, e.X-p.X, e.Y-p.Y) {
			continue
		}
		if !found || b.expectedDamageToPlayer(e) > b.expectedDamageToPlayer(best) {
			best, found = e, true
		}
	}
	if !found {
		return false, nil
	}
	b.act = "decoy"
	return b.faceAndUse(best.X-p.X, best.Y-p.Y, i, "使う")
}

// expertNigiriThrow は隣の敵 e（ボス以外）へにぎりの薬を投げてウインナーに変える。投げたら true。
func (b *bot) expertNigiriThrow(e Enemy) (bool, error) {
	if !botFeature("special") || MonsterDefinitions[e.ID].IsBoss || isReflectorEnemy(e) {
		return false, nil
	}
	i := b.findInventoryID(potionNigiriID)
	if i < 0 {
		return false, nil
	}
	p := b.g.state.Player
	if !b.canStep(p.X, p.Y, e.X-p.X, e.Y-p.Y) {
		return false, nil
	}
	b.act = "nigiriThrow"
	return b.faceAndUse(e.X-p.X, e.Y-p.Y, i, "投げる")
}

// expertBerserk は、ほかの敵の方が海老さんより近くにいる離れた敵（ボス以外）を暴走させて争わせる。使ったら true。
// 暴走した敵は一番近い相手を襲い、海老さんと同じ距離なら海老さんを襲うので、隣の敵や1対1では使わない。
func (b *bot) expertBerserk() (bool, error) {
	if !botFeature("special") {
		return false, nil
	}
	i, action := b.findCane(caneBerserkID), "使う"
	if i < 0 {
		i, action = b.findInventoryID(potionBerserkID), "投げる"
	}
	if i < 0 {
		return false, nil
	}
	g := b.g
	p := g.state.Player
	accept := func(e Enemy) bool {
		d := max(abs(e.X-p.X), abs(e.Y-p.Y))
		if d < 2 || !activeEnemy(e) || MonsterDefinitions[e.ID].IsBoss || isReflectorEnemy(e) || e.StatusAilments.Berserk > 0 {
			return false
		}
		for _, other := range g.state.Enemies {
			if (other.X != e.X || other.Y != e.Y) && !botIgnoresEnemy(other) && max(abs(other.X-e.X), abs(other.Y-e.Y)) < d {
				return true
			}
		}
		return false
	}
	dx, dy, _, ok := b.targetInLine(caneRange, accept)
	if !ok {
		return false, nil
	}
	b.act = "berserk"
	return b.faceAndUse(dx, dy, i, action)
}

// expertSaveItemsWave11 は「殴り合うと倒れる」見積もりのときの段8の道具（身代わり → にぎり → 弱める → 暴走 → 強化の薬）。
func (b *bot) expertSaveItemsWave11(adjacent []Enemy) (bool, error) {
	p := &b.g.state.Player
	if done, err := b.expertDecoy(adjacent); done || err != nil {
		return done, err
	}
	worst := adjacent[0]
	for _, e := range adjacent {
		if b.expectedDamageToPlayer(e) > b.expectedDamageToPlayer(worst) {
			worst = e
		}
	}
	if done, err := b.expertNigiriThrow(worst); done || err != nil {
		return done, err
	}
	if done, err := b.expertDebuffAt(worst.X-p.X, worst.Y-p.Y, worst); done || err != nil {
		return done, err
	}
	if done, err := b.expertBerserk(); done || err != nil {
		return done, err
	}
	return b.expertDrinkBuff(p.Health*100 < p.MaxHealth*50)
}

// --- 壺（synthesis・strengthpot） ---

// botPotJob は何ターンかに分けて行う壺の作業（装備をはずす → 入れる → 出す・割る → 装備し直す）。
type botPotJob struct {
	pot      *Pot
	targets  []Item // 合成は土台1つ、強化は入れる装備（武器・盾）
	inserted bool   // 土台・装備を壺へ入れた
	thrown   bool   // 強化の壺を割った（散らばった装備を拾い直して装備する）
	steps    int
}

const botPotJobMaxSteps = 16 // これを超えたら作業をやめる（行き詰まらないため）

// abilityValues は合成で移す印の価値（ringValues・itemValues と同じ考え方。大きいほど欲しい、負は移さない）。
// 書いていない印は 1。
var abilityValues = map[EquipmentAbilityID]int{
	// 武器
	AbilityThreeWayAttack: 5, AbilityDoubleStrike: 5, AbilityLifeSteal: 5,
	AbilitySureHit: 4, AbilityKeenEdge: 4, AbilityCriticalUp: 4, AbilityFlameBlade: 4,
	AbilityDragonSlayer: 4, AbilityGhostSlayer: 4, AbilityOneEyeSlayer: 4, AbilityDrainerSlayer: 4,
	AbilityShellSlayer: 4, AbilityExplosiveSlayer: 4, AbilityFloatingSlayer: 4, AbilitySatietyConservation: 4,
	AbilityChargeStrike: 2, AbilityComboStrike: 2, AbilityLuckySeven: 2, AbilityGiantFan: 2, AbilityRustImmune: 2,
	AbilityHungerStrike: 0, AbilityCursePower: 0,
	AbilityDisposable: -10, AbilitySatietyStrike: -10, AbilityCriticalCurse: -10, AbilityDoubleEdge: -10,
	AbilityMallet: -5, AbilityEdible: -5, AbilityMoneyStrike: -2,
	// 盾
	AbilityStatusResistance: 5, AbilityGuts: 5, AbilityEvasion: 4, AbilityReflection: 4, AbilityCounter: 4,
	AbilityExplosionResistance: 3, AbilityFireResistance: 3, AbilityMagicResistance: 3, AbilityTheftResistance: 3,
	AbilityDragonGuard: 3, AbilityPeakGuard: 3, AbilityShellDodge: 3,
	AbilityNigiriGuard: 2, AbilityPuppetGuard: 2, AbilityLuckyGuard: 2, AbilityCriticalGuard: 2,
	AbilityTossWard: 2, AbilityKnockOffWard: 2, AbilityImmovable: 2, AbilityHungerGuard: 0,
	AbilitySatietyHunger: -10, AbilityDisposableArmor: -10, AbilitySatietyGuard: -10, AbilitySatori: -10,
	AbilityMoneyGuard: -2,
}

func abilityValue(a EquipmentAbilityID) int {
	if v, ok := abilityValues[a]; ok {
		return v
	}
	return 1
}

// equipmentParts は武器・盾の能力・強化値・呪い。武器・盾でなければ ok=false。
func equipmentParts(item Item) (kind int, abilities []EquipmentAbilityID, sharpness int, cursed bool, ok bool) {
	switch v := item.(type) {
	case *Weapon:
		return 1, v.Abilities, v.Sharpness, v.Cursed, true
	case *Armor:
		return 2, v.Abilities, v.Sharpness, v.Cursed, true
	}
	return 0, nil, 0, false, false
}

// synthesisGain は base に donor を合成したときの得（移る印の価値＋強化値×2）。移すと損な印があれば ok=false。
func synthesisGain(base, donor Item) (int, bool) {
	bk, babil, _, _, ok1 := equipmentParts(base)
	dk, dabil, dsharp, dcursed, ok2 := equipmentParts(donor)
	if !ok1 || !ok2 || bk != dk || dcursed || satietyHarmfulEquipment(donor.GetID()) || donor.GetID() == disposableShieldID {
		return 0, false
	}
	gain := dsharp * 2
	merged := mergedAbilities(babil, dabil)
	for _, a := range merged[len(babil):] {
		v := abilityValue(a)
		if v < 0 {
			return 0, false
		}
		gain += v
	}
	return gain, gain > 0
}

// synthesisBaseOK は合成・強化の土台にできる装備中の武器・盾かどうか（呪われていればはずせない）。
func synthesisBaseOK(item Item) bool {
	_, _, _, cursed, ok := equipmentParts(item)
	return ok && !cursed && item.GetID() != disposableShieldID && !satietyHarmfulEquipment(item.GetID())
}

// bestSynthesisDonor は base に合成すると得をする予備の武器・盾の添字（無ければ -1）。
func (b *bot) bestSynthesisDonor(base Item) int {
	p := &b.g.state.Player
	best, bestGain := -1, 0
	for i, item := range p.Inventory {
		if item == base {
			continue
		}
		if equipable, ok := item.(Equipable); ok && p.IsEquipped(equipable) {
			continue
		}
		if gain, ok := synthesisGain(base, item); ok && gain > bestGain {
			best, bestGain = i, gain
		}
	}
	return best
}

func (b *bot) inventoryIndex(item Item) int {
	for i, it := range b.g.state.Player.Inventory {
		if it == item {
			return i
		}
	}
	return -1
}

func potContains(pot *Pot, item Item) bool {
	for _, c := range pot.Contents {
		if c == item {
			return true
		}
	}
	return false
}

// potJobCalm は壺の作業を始めてよい落ち着いた状況かどうか。
func (b *bot) potJobCalm() bool {
	p := &b.g.state.Player
	return !b.bossAlive() && !b.inRangedFire(p.X, p.Y) && len(b.enemiesWithin(6)) == 0 && p.Health*100 >= p.MaxHealth*50 &&
		p.StatusAilments.Nigiri == 0 && p.StatusAilments.Berserk == 0
}

// expertPotJob は壺の作業を進める・始める。行動したら true。bestEquipment より前に呼ぶ（作業中は装備を持ち替えない）。
func (b *bot) expertPotJob(adjacent []Enemy) (bool, error) {
	if b.potJob != nil {
		if done, err := b.runPotJob(adjacent); done || err != nil {
			return done, err
		}
		return false, nil
	}
	p := &b.g.state.Player
	for _, item := range p.Inventory {
		pot, ok := item.(*Pot)
		if !ok {
			continue
		}
		switch pot.GetID() {
		case synthesisPotID:
			if !botFeature("synthesis") {
				continue
			}
			// 前の作業の土台が残っていれば取り出す
			for _, c := range pot.Contents {
				if _, _, _, _, ok := equipmentParts(c); ok {
					b.potJob = &botPotJob{pot: pot, targets: []Item{c}, inserted: true}
					return b.runPotJob(adjacent)
				}
			}
			if pot.Capacity-len(pot.Contents) < 2 || !b.potJobCalm() {
				continue
			}
			for _, base := range []Item{p.EquippedWeapon, p.EquippedArmor} {
				if base == nil || isNilItem(base) || !synthesisBaseOK(base) || b.bestSynthesisDonor(base) < 0 {
					continue
				}
				b.potJob = &botPotJob{pot: pot, targets: []Item{base}}
				return b.runPotJob(adjacent)
			}
		case strengthPotID:
			if !botFeature("strengthpot") || len(pot.Contents) > 0 || !b.potJobCalm() || b.potThrowDirection() < 0 {
				continue
			}
			var targets []Item
			for _, item := range []Item{p.EquippedWeapon, p.EquippedArmor} {
				if item != nil && !isNilItem(item) && synthesisBaseOK(item) && len(targets) < pot.Capacity {
					targets = append(targets, item)
				}
			}
			if len(targets) == 0 {
				continue
			}
			b.potJob = &botPotJob{pot: pot, targets: targets}
			return b.runPotJob(adjacent)
		}
	}
	return false, nil
}

// isNilItem は型付きの nil（*Weapon(nil) など）を Item に入れたものかどうか。
func isNilItem(item Item) bool {
	switch v := item.(type) {
	case *Weapon:
		return v == nil
	case *Armor:
		return v == nil
	}
	return item == nil
}

// runPotJob は壺の作業を1手進める。行動したら true。終わった・続けられないときは potJob を消す。
func (b *bot) runPotJob(adjacent []Enemy) (bool, error) {
	j := b.potJob
	p := &b.g.state.Player
	j.steps++
	if j.thrown {
		return b.collectPotJob()
	}
	potIndex := b.inventoryIndex(j.pot)
	if potIndex < 0 || j.steps > botPotJobMaxSteps {
		b.potJob = nil
		return false, nil
	}
	b.act = "potJob"
	if j.pot.GetID() == synthesisPotID {
		base := j.targets[0]
		baseIndex := b.inventoryIndex(base)
		switch {
		case p.IsEquipped(base.(Equipable)):
			if j.inserted {
				b.potJob = nil // 装備し直した
				return false, nil
			}
			return b.potJobAction(b.useInventoryItem(baseIndex, "はずす"))
		case potContains(j.pot, base):
			j.inserted = true
			// 敵が来たら合成を切り上げて取り出す
			if len(adjacent) == 0 && len(j.pot.Contents) < j.pot.Capacity {
				if donor := b.bestSynthesisDonor(base); donor >= 0 {
					return b.potJobAction(b.useInventoryItemOn(potIndex, "入れる", donor))
				}
			}
			return b.potJobAction(b.useInventoryItem(potIndex, "出す"))
		case baseIndex >= 0 && !j.inserted:
			return b.potJobAction(b.useInventoryItemOn(potIndex, "入れる", baseIndex))
		case baseIndex >= 0:
			b.potJob = nil
			return b.useInventoryItem(baseIndex, "装備")
		}
		b.potJob = nil
		return false, nil
	}
	// 強化の壺: 装備をはずして入れ、全部入れたら壺を割る（中身は周りへ散らばるので拾い直す）
	for _, item := range j.targets {
		if potContains(j.pot, item) {
			continue
		}
		index := b.inventoryIndex(item)
		if index < 0 || len(j.pot.Contents) >= j.pot.Capacity || len(adjacent) > 0 {
			break
		}
		if p.IsEquipped(item.(Equipable)) {
			return b.potJobAction(b.useInventoryItem(index, "はずす"))
		}
		return b.potJobAction(b.useInventoryItemOn(potIndex, "入れる", index))
	}
	if len(j.pot.Contents) == 0 {
		b.potJob = nil // 入れられなかった（装備はそのまま）
		return false, nil
	}
	dir := b.potThrowDirection()
	if dir < 0 {
		b.potJob = nil
		return false, nil
	}
	j.thrown = true
	s := botSteps[dir]
	return b.potJobAction(b.faceAndUse(s[0], s[1], potIndex, "投げる"))
}

// collectPotJob は割った強化の壺から散らばった装備を拾い（足元は「拾う」、周りは歩いて拾う）、装備し直す。
func (b *bot) collectPotJob() (bool, error) {
	j := b.potJob
	p := &b.g.state.Player
	if j.steps > botPotJobMaxSteps+8 {
		b.potJob = nil
		return false, nil
	}
	for _, item := range j.targets {
		if index := b.inventoryIndex(item); index >= 0 && !p.IsEquipped(item.(Equipable)) {
			return b.potJobAction(b.useInventoryItem(index, "装備"))
		}
	}
	for _, item := range j.targets {
		onFloor := false
		for _, floor := range b.g.state.Items {
			if floor == item {
				onFloor = true
			}
		}
		if !onFloor || b.inventoryFull() {
			continue
		}
		x, y := item.GetPosition()
		if x == p.X && y == p.Y {
			return b.potJobAction(b.pickUpItemHere())
		}
		if dx, dy, ok := b.pathStep(func(gx, gy int) bool { return gx == x && gy == y }); ok {
			return true, b.move(dx, dy)
		}
	}
	b.potJob = nil
	return false, nil
}

// pickUpItemHere は足元の道具を足元メニューの「拾う」で拾う（pickUpHere の食料以外も拾う版）。
func (b *bot) pickUpItemHere() (bool, error) {
	if b.itemUnderPlayer() == nil || b.g.pickupBanned {
		return false, nil
	}
	if err := b.tap(KeyC); err != nil {
		return false, err
	}
	if !b.g.showMenu {
		return false, b.closeWindows()
	}
	if err := b.tap(KeyRight); err != nil { // 右隣の「足元」
		return false, err
	}
	for _, key := range []Key{KeyZ, KeyZ, KeyZ, KeyZ} { // 足元を開く → 「拾う」を選ぶ → 実行
		if err := b.tap(key); err != nil {
			return false, err
		}
		if !b.g.ShowGroundItem {
			break
		}
	}
	return true, b.closeWindows()
}

// potJobAction は作業の1手の結果。行動できなかったら作業をやめる。
func (b *bot) potJobAction(done bool, err error) (bool, error) {
	if !done && err == nil {
		b.potJob = nil
	}
	return done, err
}

// potThrowDirection は強化の壺を割る向き（botSteps の添字）。すぐ隣が壁で、壺が足元で割れる向きを選ぶ。無ければ -1。
// 割れた中身は着地点と周り8マスの空いた床へ散らばるので、足元と周りに道具・水路が無い向きにする。
func (b *bot) potThrowDirection() int {
	g := b.g
	p := g.state.Player
	free := 0
	for _, s := range append([][2]int{{0, 0}}, botSteps[:]...) {
		x, y := p.X+s[0], p.Y+s[1]
		if !g.inMapBounds(x, y) || g.state.Map[y][x].Blocked {
			continue
		}
		if g.itemDropFree(x, y) && !b.trapAt(x, y) {
			free++
		}
	}
	if free < 3 || b.itemUnderPlayer() != nil {
		return -1
	}
	for i, s := range botSteps {
		x, y := p.X+s[0], p.Y+s[1]
		if s[0] != 0 && s[1] != 0 {
			continue // 斜めは角で止まる位置が分かりにくいので使わない
		}
		if g.inMapBounds(x, y) && g.state.Map[y][x].Blocked && g.state.Map[y][x].Type != "water" {
			return i
		}
	}
	return -1
}

func (b *bot) trapAt(x, y int) bool {
	for _, trap := range b.g.state.MapTraps {
		if trap.Discovered && trap.X == x && trap.Y == y {
			return true
		}
	}
	return false
}

package main

// EquipmentAbilityID identifies an equipment ability independently of its name.
type EquipmentAbilityID string

const (
	// AbilitySatietyConservation slows the player's satiety consumption.
	AbilitySatietyConservation EquipmentAbilityID = "satiety_conservation"
	// 特効能力は、対応する系統の敵への通常攻撃ダメージを増加させる。
	AbilityDragonSlayer  EquipmentAbilityID = "dragon_slayer"
	AbilityGhostSlayer   EquipmentAbilityID = "ghost_slayer"
	AbilityOneEyeSlayer  EquipmentAbilityID = "one_eye_slayer"
	AbilityDrainerSlayer EquipmentAbilityID = "drainer_slayer"
	// 攻撃方法を変える武器能力。
	AbilityThreeWayAttack EquipmentAbilityID = "three_way_attack"
	AbilitySureHit        EquipmentAbilityID = "sure_hit"
	AbilityDigWall        EquipmentAbilityID = "dig_wall"
	AbilityDisposable     EquipmentAbilityID = "disposable"
	// 盾の特殊防御能力。
	AbilityExplosionResistance EquipmentAbilityID = "explosion_resistance"
	AbilityFireResistance      EquipmentAbilityID = "fire_resistance"
	AbilityMagicResistance     EquipmentAbilityID = "magic_resistance"
	AbilityTheftResistance     EquipmentAbilityID = "theft_resistance"
	AbilityStatusResistance    EquipmentAbilityID = "status_resistance"
	AbilityEvasion             EquipmentAbilityID = "evasion"
	AbilityReflection          EquipmentAbilityID = "reflection"
	AbilityCounter             EquipmentAbilityID = "counter"
	AbilitySatietyHunger       EquipmentAbilityID = "satiety_hunger"
	AbilityDisposableArmor     EquipmentAbilityID = "disposable_armor"
	// 敵の属性（F2）への特効。対応する属性は slayerTraits の表で引く。
	AbilityShellSlayer     EquipmentAbilityID = "shell_slayer"
	AbilityExplosiveSlayer EquipmentAbilityID = "explosive_slayer"
	AbilityFloatingSlayer  EquipmentAbilityID = "floating_slayer"
	// 指輪の能力。
	AbilityPowerBoost EquipmentAbilityID = "power_boost" // 鼓舞の指輪: パワーと最大パワーを上げる（呪われていると下げる）
	// 位置を動かす攻撃への耐性（F3）。
	AbilityImmovable EquipmentAbilityID = "immovable"  // 不動: 吹き飛ばし・引き寄せ・場所替えを受けない
	AbilityBreakFall EquipmentAbilityID = "break_fall" // 受け身: 吹き飛ばされてぶつかってもダメージを受けない
	// AbilityRustImmune は、その装備品自身がさびない（金の剣・金の盾）。ほかの装備品は守らない。
	// 装備中の武器・盾をまとめて守る指輪は statusImmunities に StatusRust を登録する。
	AbilityRustImmune EquipmentAbilityID = "rust_immune"
)

// EnemyTrait は、特効などの判定に使う敵の分類タグを表す。
type EnemyTrait string

const (
	EnemyTraitDragon  EnemyTrait = "dragon"
	EnemyTraitGhost   EnemyTrait = "ghost"
	EnemyTraitOneEye  EnemyTrait = "one_eye"
	EnemyTraitDrainer EnemyTrait = "drainer"
	// 甲殻（カニ・ヤドカリ・貝）、爆発（ウニ・フグ）、浮遊（クラゲ）。
	EnemyTraitShell     EnemyTrait = "shell"
	EnemyTraitExplosive EnemyTrait = "explosive"
	EnemyTraitFloating  EnemyTrait = "floating"
	EnemyTraitAquatic   EnemyTrait = "aquatic"
)

// slayerTraits は特効能力と、その能力が効く敵の属性の対応表。
// 新しい特効武器は能力 ID をここへ追加するだけで slayerMultiplier に反映される。
var slayerTraits = map[EquipmentAbilityID]EnemyTrait{
	AbilityDragonSlayer:    EnemyTraitDragon,
	AbilityGhostSlayer:     EnemyTraitGhost,
	AbilityOneEyeSlayer:    EnemyTraitOneEye,
	AbilityDrainerSlayer:   EnemyTraitDrainer,
	AbilityShellSlayer:     EnemyTraitShell,
	AbilityExplosiveSlayer: EnemyTraitExplosive,
	AbilityFloatingSlayer:  EnemyTraitFloating,
}

// powerRingBonus は鼓舞の指輪（AbilityPowerBoost）1つあたりのパワーの増減量。
const powerRingBonus = 3

const (
	baseSatietyLossInterval     = 10
	hungrySatietyLossInterval   = 5
	shieldResistanceNumerator   = 1
	shieldResistanceDenominator = 2
	shieldEvasionPercent        = 25
	shieldCounterNumerator      = 1
	shieldCounterDenominator    = 2
	statusResistancePercent     = 50
)

const basePlayerAttackAccuracy = 90

func hasEquipmentAbility(abilities []EquipmentAbilityID, target EquipmentAbilityID) bool {
	for _, ability := range abilities {
		if ability == target {
			return true
		}
	}
	return false
}

func hasEnemyTrait(traits []EnemyTrait, target EnemyTrait) bool {
	for _, trait := range traits {
		if trait == target {
			return true
		}
	}
	return false
}

// slayerMultiplier は、武器能力と敵の分類が一致したときのダメージ倍率を返す。
// 複数の特効条件が一致しても倍率は重複させず、rollDamage の Multiplier へ乗算で渡す。
func slayerMultiplier(abilities []EquipmentAbilityID, traits []EnemyTrait) (float64, bool) {
	for _, ability := range abilities {
		if trait, ok := slayerTraits[ability]; ok && hasEnemyTrait(traits, trait) {
			return slayerDamageMultiplier, true
		}
	}
	return 1.0, false
}

// countEquipmentAbility は装備品ごとの能力一覧のうち、target を持つものの数（重ね数）を返す。
// 同じ能力の指輪を2つ装備すると2になる。1つの装備品に同じ ID が重複していても1と数える。
func countEquipmentAbility(target EquipmentAbilityID, abilityLists ...[]EquipmentAbilityID) int {
	count := 0
	for _, abilities := range abilityLists {
		if hasEquipmentAbility(abilities, target) {
			count++
		}
	}
	return count
}

// uniqueEquipmentAbilities は装備品ごとの能力一覧を、最初に現れた順に重複なく並べる。
// 装備能力のフックを1つの能力につき1回（重ね数つき）だけ呼ぶために使う。
func uniqueEquipmentAbilities(abilityLists ...[]EquipmentAbilityID) []EquipmentAbilityID {
	var result []EquipmentAbilityID
	for _, abilities := range abilityLists {
		for _, ability := range abilities {
			if !hasEquipmentAbility(result, ability) {
				result = append(result, ability)
			}
		}
	}
	return result
}

// accessoryPowerBonus は指輪を装備したときのパワーと最大パワーの増減量を返す（外すときは符号を反転して使う）。
// 鼓舞の指輪は +3、呪われていると -3。
func accessoryPowerBonus(abilities []EquipmentAbilityID, cursed bool) int {
	if !hasEquipmentAbility(abilities, AbilityPowerBoost) {
		return 0
	}
	if cursed {
		return -powerRingBonus
	}
	return powerRingBonus
}

// StatusKind は装備能力で無効化できる状態異常・悪い効果の種類。
type StatusKind string

const (
	StatusConfusion StatusKind = "confusion"
	StatusSleep     StatusKind = "sleep"
	StatusBlind     StatusKind = "blind"
	StatusSlow      StatusKind = "slow"
	StatusBearTrap  StatusKind = "bear_trap"
	StatusPoison    StatusKind = "poison" // 毒によるパワー低下（g.drainPlayerPower）
	// 状態異常ではないが、同じ問い合わせで防ぐ悪い効果。
	StatusCurse         StatusKind = "curse"          // 持ち物が呪われる（ノロイガニ系）
	StatusRust          StatusKind = "rust"           // 装備中の武器・盾の強化値が下がる（さびの罠）
	StatusFoodTransform StatusKind = "food_transform" // 持ち物をウインナーに変えられる（にぎりエビ系）
	StatusManipulation  StatusKind = "manipulation"   // 道具を使わされる・歩かされる（あやつりクラゲ系）
	StatusKnockOff      StatusKind = "knock_off"      // 装備中の武器・盾を弾き飛ばされる（g.knockOffPlayerEquipment）
	StatusItemToss      StatusKind = "item_toss"      // 持ち物を投げ捨てられる（g.tossPlayerItem）
)

// statusImmunities は能力 ID ごとの、確実に無効化する状態異常の一覧。
// 混乱よけ・睡眠よけ・毒消しなどの指輪は、能力 ID をここへ追加するだけで
// g.playerImmuneTo と各状態異常の付与箇所に反映される。
var statusImmunities = map[EquipmentAbilityID][]StatusKind{}

// statusKindLabel は「〜を防いだ」のメッセージに使う状態異常の名前を返す。
func statusKindLabel(kind StatusKind) string {
	if def, ok := statusDefOf(kind); ok {
		return def.Name
	}
	switch kind {
	case StatusLevelDown:
		return "レベル低下"
	case StatusPoison:
		return "毒"
	case StatusCurse:
		return "呪い"
	case StatusRust:
		return "さび"
	case StatusFoodTransform:
		return "食料への変化"
	case StatusManipulation:
		return "あやつり"
	case StatusKnockOff:
		return "装備の弾き飛ばし"
	case StatusItemToss:
		return "持ち物の投げ捨て"
	default:
		return string(kind)
	}
}

// itemResistsRust は装備品1つがさび（強化値の低下）を受けないかを返す（さび止め済みか、AbilityRustImmune）。
func itemResistsRust(rustProof bool, abilities []EquipmentAbilityID) bool {
	return rustProof || hasEquipmentAbility(abilities, AbilityRustImmune)
}

// abilitiesGrantImmunity は能力一覧のどれかが kind を無効化するかを返す。
func abilitiesGrantImmunity(abilities []EquipmentAbilityID, kind StatusKind) bool {
	for _, ability := range abilities {
		for _, immune := range statusImmunities[ability] {
			if immune == kind {
				return true
			}
		}
	}
	return false
}

func satietyLossInterval(armorAbilities []EquipmentAbilityID) int {
	// AbilitySatori は !test 側で定義されるため、共有コードでは保存された能力 ID で判定する。
	if hasEquipmentAbility(armorAbilities, "satori") {
		return 0 // 時間経過による減少なし
	}
	if hasEquipmentAbility(armorAbilities, AbilitySatietyHunger) {
		return hungrySatietyLossInterval
	}
	if hasEquipmentAbility(armorAbilities, AbilitySatietyConservation) {
		return baseSatietyLossInterval * 2
	}
	return baseSatietyLossInterval
}

// EnemyDamageKind は盾の属性耐性を判定するための敵攻撃属性を表す。
type EnemyDamageKind int

const (
	EnemyDamageNormal EnemyDamageKind = iota
	EnemyDamageExplosion
	EnemyDamageFire
	EnemyDamageMagic
)

// EnemyAttackDelivery は回避・反射・カウンターの対象になる攻撃方法を表す。
type EnemyAttackDelivery int

const (
	EnemyAttackMelee EnemyAttackDelivery = iota
	EnemyAttackRanged
	EnemyAttackEnvironment
)

// ShieldDefenseResult は盾能力を適用した1回の被弾結果。
type ShieldDefenseResult struct {
	Damage        int
	Evaded        bool
	Resisted      bool
	Reflected     bool
	CounterDamage int
	WearArmor     bool
	// Guard は回避などのメッセージに使う装備品の名前（空なら装備中の盾の名前）。
	Guard string
	// Effects は装備能力のフック（ModifyIncomingDamage）が積んだ、ダメージを受けるときの処理。
	// 構造体を == で比べられるようポインタで持つ（無ければ nil）。
	Effects *[]func(g *Game)
}

func hasDamageResistance(abilities []EquipmentAbilityID, kind EnemyDamageKind) bool {
	switch kind {
	case EnemyDamageExplosion:
		return hasEquipmentAbility(abilities, AbilityExplosionResistance)
	case EnemyDamageFire:
		return hasEquipmentAbility(abilities, AbilityFireResistance)
	case EnemyDamageMagic:
		return hasEquipmentAbility(abilities, AbilityMagicResistance)
	default:
		return false
	}
}

func reducedShieldDamage(damage int) int {
	if damage <= 0 {
		return 0
	}
	reduced := damage * shieldResistanceNumerator / shieldResistanceDenominator
	return max(1, reduced)
}

// resolveShieldDefense は回避→反射→属性軽減→カウンターの順で盾能力を解決する。
// 爆発は範囲攻撃のため反射できず、環境ダメージは回避・反射・カウンターの対象外とする。
func resolveShieldDefense(damage int, kind EnemyDamageKind, delivery EnemyAttackDelivery, abilities []EquipmentAbilityID, intn func(int) int) ShieldDefenseResult {
	result := ShieldDefenseResult{Damage: max(0, damage)}
	if result.Damage == 0 {
		return result
	}

	if delivery != EnemyAttackEnvironment && hasEquipmentAbility(abilities, AbilityEvasion) && intn(100) < shieldEvasionPercent {
		result.Damage = 0
		result.Evaded = true
		return result
	}
	if delivery == EnemyAttackRanged && kind != EnemyDamageExplosion && hasEquipmentAbility(abilities, AbilityReflection) {
		result.Damage = 0
		result.Reflected = true
		result.CounterDamage = damage
		return result
	}
	if hasDamageResistance(abilities, kind) {
		result.Damage = reducedShieldDamage(result.Damage)
		result.Resisted = true
	}
	if delivery == EnemyAttackMelee && hasEquipmentAbility(abilities, AbilityCounter) {
		result.CounterDamage = max(1, result.Damage*shieldCounterNumerator/shieldCounterDenominator)
	}
	result.WearArmor = result.Damage > 0 && hasEquipmentAbility(abilities, AbilityDisposableArmor)
	return result
}

func blocksTheft(abilities []EquipmentAbilityID) bool {
	return hasEquipmentAbility(abilities, AbilityTheftResistance)
}

// rollStatusResistance は状態異常耐性盾の50%無効化を判定する。
func rollStatusResistance(abilities []EquipmentAbilityID, intn func(int) int) bool {
	return hasEquipmentAbility(abilities, AbilityStatusResistance) && intn(100) < statusResistancePercent
}

func nextDisposableArmorSharpness(sharpness int, abilities []EquipmentAbilityID) (int, bool) {
	if !hasEquipmentAbility(abilities, AbilityDisposableArmor) {
		return sharpness, false
	}
	return sharpness - 1, true
}

func shouldReduceSatiety(moveCount int, armorAbilities []EquipmentAbilityID) bool {
	interval := satietyLossInterval(armorAbilities)
	return interval > 0 && moveCount > 0 && moveCount%interval == 0
}

// playerAttackDeltas は武器能力に応じた攻撃対象方向を返す。
// 三方向攻撃は入力方向を正面として、左右45度の隣接マスも対象にする。
func playerAttackDeltas(dx, dy int, abilities []EquipmentAbilityID) []Coordinate {
	front := Coordinate{X: sign(dx), Y: sign(dy)}
	if !hasEquipmentAbility(abilities, AbilityThreeWayAttack) {
		return []Coordinate{front}
	}
	left := Coordinate{X: sign(dx + dy), Y: sign(dy - dx)}
	right := Coordinate{X: sign(dx - dy), Y: sign(dy + dx)}
	return []Coordinate{front, left, right}
}

// rollPlayerAttackHit は通常武器の90%命中と必中能力の100%命中を判定する。
func rollPlayerAttackHit(abilities []EquipmentAbilityID, intn func(int) int) bool {
	return rollPlayerAttackHitChance(basePlayerAttackAccuracy, hasEquipmentAbility(abilities, AbilitySureHit), intn)
}

// rollPlayerAttackHitChance は命中率 accuracy（%）で命中を判定する。必中なら乱数を使わず命中する。
func rollPlayerAttackHitChance(accuracy int, sureHit bool, intn func(int) int) bool {
	if sureHit {
		return true
	}
	return intn(100) < accuracy
}

// nextDisposableWeaponAttackPower は攻撃後の使い捨て武器の基礎攻撃力を返す。
func nextDisposableWeaponAttackPower(attackPower int, abilities []EquipmentAbilityID) (int, bool) {
	if !hasEquipmentAbility(abilities, AbilityDisposable) || attackPower <= 0 {
		return attackPower, false
	}
	return attackPower - 1, true
}

// isDiggableTile はつるはしで掘れる未通行タイルかを返す。マップ外への脱出を防ぐため外周は掘れない。
func isDiggableTile(tile Tile, x, y, width, height int) bool {
	if x <= 0 || y <= 0 || x >= width-1 || y >= height-1 {
		return false
	}
	return tile.Blocked && (tile.Type == "wall" || tile.Type == "other" || tile.Type == "cracked_wall" || tile.Type == "hidden_corridor")
}

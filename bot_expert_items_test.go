//go:build !test

package main

// expert の道具の使いどころ（bot_expert_test.go から呼ぶ）: 指輪の装備、封印・シフトチェンジの杖、
// ゾワゾワ・聖域・自爆・ウインナー化・大部屋・白紙のカード、保存の壺。

// ringValues は指輪の価値（テンプレート ID ごと、itemRoles と同じ形）。大きいほど優先して付け、0 以下は付けない。
var ringValues = map[int]int{
	175: 10, // 回復（毎ターン HP+1）
	10:  9,  // 鼓舞（パワー）
	181: 7,  // 睡眠よけ
	180: 6,  // 混乱よけ
	179: 5,  // 毒消し
	176: 5,  // 胃拡張
	188: 5,  // 浮遊（罠を踏まない）
	191: 4,  // 百発百中（矢・投げる道具）
	193: 2,  // 連射（矢を装備しているときは ringValue で上げる）
	200: 4,  // 弾きよけ
	182: 4,  // 錆よけ
	178: 4,  // しあわせ
	186: 3,  // 透視
	184: 3,  // 気配察知
	204: 3,  // 不動
	183: 3,  // 呪いよけ
	185: 2,  // 道具感知
	187: 2,  // 鑑定師
	189: 1,  // 遠投
	192: 1,  // 反射
	// 付けない: 呪われている（胃縮小・ヘタ投げ・垂れ流し・小銭垂れ流し・痛恨）、
	// 害のある（高飛び・魔物呼び・罠増し）、状況次第（爆発・諸刃・すれちがい）
	177: -1, 190: -1, 196: -1, 197: -1, 201: -1,
	194: -1, 198: -1, 199: -1,
	195: -1, 202: -1, 203: -1,
}

// ringValue は所持している指輪の今の価値。
func (b *bot) ringValue(a *Accessory) int {
	v := ringValues[a.GetID()]
	if a.GetID() == 193 && b.g.state.Player.EquippedArrow != nil {
		v = 6
	}
	if a.Cursed {
		v = -1
	}
	return v
}

// expertEquipRing は良い指輪を付ける（空いている枠へ、または価値の低い指輪と替える）。行動したら true。
func (b *bot) expertEquipRing() (bool, error) {
	p := &b.g.state.Player
	best, bestValue := -1, 0
	for i, item := range p.Inventory {
		if a, ok := item.(*Accessory); ok && !p.IsEquipped(a) {
			if v := b.ringValue(a); v > bestValue {
				best, bestValue = i, v
			}
		}
	}
	if best < 0 {
		return false, nil
	}
	slots := p.EquippedAccessories
	if slots[0] == nil || slots[1] == nil {
		return b.useInventoryItem(best, "装備")
	}
	// 両方の枠が埋まっていれば、2つ目の枠（装備すると入れ替わる）が呪われておらず価値が低いときだけ替える。
	// 1つ目の枠の方が価値が低ければ先にはずす（次の行動で空いた枠へ付ける）
	worst := 1
	if !slots[0].Cursed && (slots[1].Cursed || b.ringValue(slots[0]) < b.ringValue(slots[1])) {
		worst = 0
	}
	if slots[worst].Cursed || b.ringValue(slots[worst]) >= bestValue {
		return false, nil
	}
	if worst == 1 {
		return b.useInventoryItem(best, "装備")
	}
	for i, item := range p.Inventory {
		if item == Item(slots[0]) {
			return b.useInventoryItem(i, "はずす")
		}
	}
	return false, nil
}

// --- 杖 ---

const (
	caneSealID  = 16 // 封印の杖
	caneShiftID = 9  // シフトチェンジの杖
	caneRange   = 10 // 杖を振る相手の最大距離（BOT の目安）
)

// findCane は回数の残っている杖 id の所持品の添字。
func (b *bot) findCane(id int) int {
	for i, item := range b.g.state.Player.Inventory {
		if c, ok := item.(*Cane); ok && c.GetID() == id && c.Uses > 0 {
			return i
		}
	}
	return -1
}

// hasSealableAbility は封印で止められる能力（特殊攻撃・遠距離攻撃・固有の行動・被弾時の反応）を持つ敵かどうか。
func hasSealableAbility(e Enemy) bool {
	def := MonsterDefinitions[e.ID]
	return e.SpecialAttack != nil || e.RangedAttack.Kind != RangedAttackNone || def.Behavior != nil || def.OnDamaged != nil
}

// expertSeal は能力を持つ強敵・遠距離攻撃の敵が直線上にいれば封印の杖を振る。振ったら true。
func (b *bot) expertSeal() (bool, error) {
	i := b.findCane(caneSealID)
	if i < 0 {
		return false, nil
	}
	accept := func(e Enemy) bool {
		if e.StatusAilments.Seal || isReflectorEnemy(e) || MonsterDefinitions[e.ID].IsBoss || !hasSealableAbility(e) {
			return false
		}
		return b.isStrongEnemy(e) || e.RangedAttack.Kind != RangedAttackNone || isHazardEnemy(e)
	}
	dx, dy, _, ok := b.targetInLine(caneRange, accept)
	if !ok {
		return false, nil
	}
	b.act = "seal"
	return b.faceAndUse(dx, dy, i, "使う")
}

// expertShift は直線上の離れた敵（距離3以上）とシフトチェンジの杖で入れ替わり、囲みから抜ける。振ったら true。
func (b *bot) expertShift() (bool, error) {
	i := b.findCane(caneShiftID)
	if i < 0 {
		return false, nil
	}
	p := b.g.state.Player
	accept := func(e Enemy) bool {
		return !isReflectorEnemy(e) && !MonsterDefinitions[e.ID].IsBoss && max(abs(e.X-p.X), abs(e.Y-p.Y)) >= 3
	}
	dx, dy, _, ok := b.targetInLine(caneRange, accept)
	if !ok {
		return false, nil
	}
	b.act = "shift"
	return b.faceAndUse(dx, dy, i, "使う")
}

// --- カード ---

const (
	cardZowazowaID     = 174
	cardSanctuaryID    = 55
	cardSelfDestructID = 41
	cardSausageID      = 167
	cardBigRoomID      = 39
	cardBlankID        = 53
)

// expertSanctuary は足元に聖域のカードを置く（足元が空いていて階段でないとき）。置いたら true。
func (b *bot) expertSanctuary() (bool, error) {
	i := b.findInventoryID(cardSanctuaryID)
	p := b.g.state.Player
	if i < 0 || b.itemUnderPlayer() != nil || b.g.state.Map[p.Y][p.X].Type == "stairs" {
		return false, nil
	}
	b.act = "sanctuary"
	return b.useInventoryItem(i, "置く")
}

// onSanctuaryRest は聖域の上で HP が戻るまで休むかどうか。
func (b *bot) onSanctuaryRest() bool {
	g := b.g
	p := &g.state.Player
	return g.playerOnSanctuary() && p.Health*100 < p.MaxHealth*90 && p.Satiety > 0 && g.floorTurns < expertRestFloorTurns &&
		len(b.enemiesWithin(expertRestEnemyRange)) > 0 && len(b.adjacentEnemies()) == 0
}

// expertSausage は空腹で食料が無いとき、価値の一番低い道具をウインナー化のカードでウインナーに変える。使ったら true。
func (b *bot) expertSausage() (bool, error) {
	p := &b.g.state.Player
	card := b.findInventoryID(cardSausageID)
	if card < 0 || p.Satiety > 20 || b.foodCarried() > 0 {
		return false, nil
	}
	target, targetValue := -1, 1<<30
	for i, item := range p.Inventory {
		if i == card {
			continue
		}
		if v, ok := b.itemValue(item); ok && v < targetValue {
			target, targetValue = i, v
		}
	}
	if target < 0 {
		return false, nil
	}
	b.act = "sausage"
	return b.useInventoryItemOn(card, "使う", target)
}

// expertBigRoom は空腹で食料が無く、階段が見つかっていないか遠いとき、近くに敵がいなければ大部屋のカードを読む。
func (b *bot) expertBigRoom() (bool, error) {
	g := b.g
	p := &g.state.Player
	i := b.findInventoryID(cardBigRoomID)
	if i < 0 || p.Satiety > 15 || b.foodCarried() > 0 || isBossFloor(g.Floor) || len(b.enemiesWithin(6)) > 0 {
		return false, nil
	}
	if d := stairsDistance(g); d >= 0 && d <= 20 {
		return false, nil
	}
	b.act = "bigRoom"
	return b.useInventoryItem(i, "使う")
}

// expertWriteBlank は近くに敵がいないとき、白紙のカードに書き込む（回復の道具が無ければ完全回復、あれば全滅）。
func (b *bot) expertWriteBlank() (bool, error) {
	i := b.findInventoryID(cardBlankID)
	if i < 0 || len(b.enemiesWithin(4)) > 0 {
		return false, nil
	}
	want := 56 // 全滅
	if len(b.roleItems(roleHeal)) == 0 {
		want = 44 // 完全回復
	}
	ids := blankCardOptionIDs()
	option := -1
	for j, id := range ids {
		if id == want {
			option = j
		}
	}
	if option < 0 {
		return false, nil
	}
	b.act = "blank"
	if err := b.openItemActions(i); err != nil {
		return false, err
	}
	if ok, err := b.chooseItemAction("使う"); !ok || err != nil {
		return false, err
	}
	for j := 0; j < len(ids) && b.g.showBlankCardMenu && b.g.blankCardIndex != option; j++ {
		key := KeyDown
		if b.g.blankCardIndex > option {
			key = KeyUp
		}
		if err := b.tap(key); err != nil {
			return false, err
		}
	}
	if b.g.showBlankCardMenu && b.g.blankCardIndex == option {
		if err := b.tap(KeyZ); err != nil {
			return false, err
		}
	}
	return true, b.closeWindows()
}

// --- 保存の壺 ---

const storagePotID = 46

// storagePot は所持している保存の壺（入れる余地の多いもの）を返す。
func (b *bot) storagePot() (int, *Pot) {
	best, bestPot := -1, (*Pot)(nil)
	for i, item := range b.g.state.Player.Inventory {
		if pot, ok := item.(*Pot); ok && (pot.GetID() == storagePotID || pot.GetID() == synthesisPotID) {
			if bestPot == nil || pot.Capacity-len(pot.Contents) > bestPot.Capacity-len(bestPot.Contents) {
				best, bestPot = i, pot
			}
		}
	}
	return best, bestPot
}

// potFood は保存の壺に入っている食料の数（腐ったウインナーを除く）。
func (b *bot) potFood() int {
	n := 0
	for _, item := range b.g.state.Player.Inventory {
		if pot, ok := item.(*Pot); ok && (pot.GetID() == storagePotID || pot.GetID() == synthesisPotID) {
			for _, c := range pot.Contents {
				if isFoodItem(c) && c.GetID() != rottenSausageItemID {
					n++
				}
			}
		}
	}
	return n
}

// expertStoreFood は食料を2つ以上持っていれば、1つを残して予備を保存の壺へ入れる。入れたら true。
func (b *bot) expertStoreFood() (bool, error) {
	potIndex, pot := b.storagePot()
	if pot == nil || len(pot.Contents) >= pot.Capacity || len(b.enemiesWithin(4)) > 0 {
		return false, nil
	}
	var foods []int
	for i, item := range b.g.state.Player.Inventory {
		if isFoodItem(item) && item.GetID() != rottenSausageItemID {
			foods = append(foods, i)
		}
	}
	if len(foods) < 2 {
		return false, nil
	}
	b.act = "storeFood"
	return b.useInventoryItemOn(potIndex, "入れる", foods[len(foods)-1])
}

// expertTakeFoodOut は空腹で持ち物に食料が無ければ、保存の壺から中身を出す。出したら true。
func (b *bot) expertTakeFoodOut(limit int) (bool, error) {
	p := &b.g.state.Player
	if p.Satiety > limit || b.findInventory(isFoodItem) >= 0 || b.potFood() == 0 || b.inventoryFull() {
		return false, nil
	}
	for i, item := range p.Inventory {
		if pot, ok := item.(*Pot); ok && (pot.GetID() == storagePotID || pot.GetID() == synthesisPotID) {
			for _, c := range pot.Contents {
				if isFoodItem(c) {
					b.act = "takeFood"
					return b.useInventoryItem(i, "出す")
				}
			}
		}
	}
	return false, nil
}

// useInventoryItemOn は所持品 index に action を行い、続けて開く道具の選択画面で target を選ぶ
// （ウインナー化のカード・壺へ入れる）。target は使う前の所持品の添字。
func (b *bot) useInventoryItemOn(index int, action string, target int) (bool, error) {
	g := b.g
	if err := b.openItemActions(index); err != nil {
		return false, err
	}
	if !g.showItemActions {
		return false, b.closeWindows()
	}
	if ok, err := b.chooseItemAction(action); !ok || err != nil {
		return false, err
	}
	if g.itemTargetEffect == nil && !g.usePotInsert {
		return true, b.closeWindows()
	}
	for i := 0; i < 2*g.state.Player.MaxInventory && g.selectedItemIndex != target; i++ {
		key := KeyDown
		if g.selectedItemIndex > target {
			key = KeyUp
		}
		if err := b.tap(key); err != nil {
			return false, err
		}
	}
	if g.selectedItemIndex == target {
		if err := b.tap(KeyZ); err != nil {
			return false, err
		}
	}
	return true, b.closeWindows()
}

// --- 道具の価値（持ち物の入れ替え・ウインナー化の対象選び） ---

// itemValues はカード・杖・薬の価値の目安（itemRoles の分類から決まらないもの）。
var itemValues = map[int]int{
	225: 35, 226: 25, 227: 30, 228: 10, 229: 0, 230: 10, 231: 25, 232: 25,
	205: 50, 206: 50, 207: 50, 208: 70, 209: 70,
	210: 65, 211: 70, 212: 70, 213: 0,
	215: 5, 216: 6, 217: 10, 218: 6, 219: 7, 220: -1, 221: 5,
	cardBlankID:        75, // 白紙（全滅・完全回復を書ける）
	cardSanctuaryID:    60,
	cardZowazowaID:     55,
	cardSausageID:      45,
	cardSelfDestructID: 40,
	cardBigRoomID:      35,
	47:                 20, // 壺拡大（保存の壺を持っていなければ下げる）
	48:                 2,  // 吸い出し（保存の壺には要らない）
	172:                1,  // バクチ
}

// itemValue は所持品の価値（大きいほど手放さない）。装備中・食料・回復・中身のある壺など、手放さないものは false。
func (b *bot) itemValue(item Item) (int, bool) {
	p := &b.g.state.Player
	if equipable, ok := item.(Equipable); ok && p.IsEquipped(equipable) {
		return 0, false
	}
	if isFoodItem(item) {
		if item.GetID() == rottenSausageItemID && b.foodCarried()-1 >= 2 {
			return 30, true // 普通の食料が2つ以上あれば腐ったウインナーは手放してよい
		}
		return 0, false
	}
	if isHealingItem(item) || itemRoles[item.GetID()] == roleHeal {
		return 0, false
	}
	switch v := item.(type) {
	case *Pot:
		if len(v.Contents) > 0 {
			return 0, false
		}
		if value, ok := itemValues[v.GetID()]; ok {
			return value, true
		}
		if v.GetID() == storagePotID {
			return 25, true
		}
		return 5, true
	case *Weapon, *Armor:
		return b.spareEquipmentValue(item), true
	case *Accessory:
		if r := b.ringValue(v); r > 0 {
			return 15 + r, true
		}
		return 3, true
	case *Arrow:
		return 20, true
	case *Cane:
		if v.Uses <= 0 {
			return 3, true
		}
	}
	if value, ok := itemValues[item.GetID()]; ok {
		if item.GetID() == 47 {
			if _, pot := b.storagePot(); pot == nil {
				return 5, true
			}
		}
		return value, true
	}
	switch itemRoles[item.GetID()] {
	case roleHarmful:
		return 0, true
	case roleAttack:
		return 70, true
	case roleEscape:
		return 65, true
	case roleBuff:
		return 50, true
	}
	return 10, true
}

// spareEquipmentValue は予備の武器・盾の価値。種類ごとに一番良い予備1つ（先に持っていた方）だけを残す価値にする。
func (b *bot) spareEquipmentValue(item Item) int {
	score := func(it Item) (kind, value int) {
		switch v := it.(type) {
		case *Weapon:
			if v.Cursed || satietyHarmfulEquipment(v.GetID()) {
				return 1, -1
			}
			return 1, v.AttackPower + v.Sharpness
		case *Armor:
			if v.Cursed || satietyHarmfulEquipment(v.GetID()) {
				return 2, -1
			}
			return 2, v.DefensePower + v.Sharpness
		}
		return 0, -1
	}
	kind, mine := score(item)
	if mine < 0 {
		return 1
	}
	p := &b.g.state.Player
	before := true // item より前に並んでいる
	for _, other := range p.Inventory {
		if other == item {
			before = false
			continue
		}
		if equipable, ok := other.(Equipable); ok && p.IsEquipped(equipable) {
			continue
		}
		if k, s := score(other); k == kind && (s > mine || (s == mine && before)) {
			return 5 // もっと良い予備がある
		}
	}
	return 25
}

// --- 持ち物の入れ替え ---

const swapValueMargin = 5 // 床の道具の価値が持ち物の最低よりこれ以上高ければ入れ替える

// keepValue は itemValue の「手放さない」道具を高い価値として扱ったもの（床の道具の評価用）。
func (b *bot) keepValue(item Item) int {
	if v, ok := b.itemValue(item); ok {
		return v
	}
	return 100
}

// worstInventoryItem は手放してよい所持品のうち価値の一番低いものの添字と価値。
func (b *bot) worstInventoryItem() (int, int) {
	worst, worstValue := -1, 1<<30
	for i, item := range b.g.state.Player.Inventory {
		if v, ok := b.itemValue(item); ok && v < worstValue {
			worst, worstValue = i, v
		}
	}
	return worst, worstValue
}

// worthPicking は床の道具を拾う（満杯なら入れ替える）価値があるかどうか。不利なカードは拾わない。
func (b *bot) worthPicking(item Item) bool {
	if isStuckSanctuaryItem(item) || b.wasDiscarded(item) || !item.GetShowOnMiniMap() {
		return false
	}
	v := b.keepValue(item)
	if v <= 0 {
		return false
	}
	if !b.inventoryFull() {
		return true
	}
	_, worst := b.worstInventoryItem()
	return v >= worst+swapValueMargin
}

// expertItemGoal は expert が向かう床の道具（満杯なら入れ替える価値のあるもの）。
func (b *bot) expertItemGoal(x, y int) bool {
	for _, item := range b.g.state.Items {
		if ix, iy := item.GetPosition(); ix == x && iy == y && b.worthPicking(item) {
			return true
		}
	}
	return false
}

// expertSwapHere は所持品が満杯で、足元の道具の方が価値が高ければ「交換」で入れ替える。入れ替えたら true。
func (b *bot) expertSwapHere() (bool, error) {
	g := b.g
	item := b.itemUnderPlayer()
	if item == nil || !b.inventoryFull() || g.pickupBanned || !b.worthPicking(item) {
		return false, nil
	}
	worst, _ := b.worstInventoryItem()
	if worst < 0 {
		return false, nil
	}
	if b.discarded == nil {
		b.discarded = map[discardKey]bool{}
	}
	b.discarded[discardKey{g.Floor, g.state.Player.Inventory[worst].GetID()}] = true
	b.act = "swap"
	if err := b.tap(KeyC); err != nil {
		return false, err
	}
	if !g.showMenu {
		return false, b.closeWindows()
	}
	// 足元を開く → 「交換」を選ぶ → 実行すると持ち物の選択になる
	for _, keys := range [][]Key{{KeyRight}, {KeyZ}, {KeyDown}, {KeyZ}} {
		if err := b.tap(keys...); err != nil {
			return false, err
		}
	}
	if !g.showInventory || g.selectedGroundActionIndex != 1 {
		return true, b.closeWindows()
	}
	for i := 0; i < 2*g.state.Player.MaxInventory && g.selectedItemIndex != worst; i++ {
		key := KeyDown
		if g.selectedItemIndex > worst {
			key = KeyUp
		}
		if err := b.tap(key); err != nil {
			return false, err
		}
	}
	if g.selectedItemIndex == worst {
		if err := b.tap(KeyZ); err != nil {
			return false, err
		}
	}
	return true, b.closeWindows()
}

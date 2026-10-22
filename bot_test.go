//go:build !test

package main

import (
	"errors"
	"math/rand/v2"
)

// persona は BOT の性格（行動方針）。
type persona string

const (
	personaAggressive persona = "aggressive" // 見えた敵に向かい、HP が低くても戦う
	personaCautious   persona = "cautious"   // HP 40% で回復し、危険な敵から逃げる
	personaItemUser   persona = "itemUser"   // カード・杖・投擲を積極的に使う
	personaDiver      persona = "diver"      // 階段へ直行する
	personaRandom     persona = "random"     // ランダムに入力する（バグ探し用）
	personaExpert     persona = "expert"     // 道具を使い分け、探索してレベルを上げる（bot_expert_test.go）
)

var allPersonas = []persona{personaAggressive, personaCautious, personaItemUser, personaDiver, personaRandom, personaExpert}

func parsePersona(name string) (persona, bool) {
	for _, p := range allPersonas {
		if string(p) == name {
			return p, true
		}
	}
	return "", false
}

const (
	botMaxFrames      = 3000 // 1回の入力から入力待ちへ戻るまでの上限フレーム（超えたらソフトロック）
	botStuckLimit     = 400  // ターンが進まない行動がこの回数続いたら BOT 側の行き詰まりとして打ち切る
	botFloorTurnLimit = 500  // この階に長く居すぎたら探索をやめて階段へ向かう
)

// errBotStuck は BOT がゲームを進められなくなったこと（ゲームの不具合ではない）を表す。
var errBotStuck = errors.New("BOT が行き詰まった")

// bot はゲーム状態を直接読み（ホワイトボックス）、仮想入力でキーを押して遊ぶ。
type bot struct {
	g        *Game
	in       *VirtualInput
	persona  persona
	rng      *rand.Rand
	idle     int // ターンが進まなかった行動の連続回数
	lastTurn int
	act      string // 直前に選んだ行動の分類（序盤の死亡分析の記録用。expert だけが設定する）
	resting  bool   // expert が自然回復を待っている（回復しきるか敵が来るまで続ける）
	// discarded は食料のために手放した道具（同じ階では拾い直さない）
	discarded map[discardKey]bool
	// hazardAvoided は expert が隣の危ない敵（膨らんだフグ・帯電したエイ）から続けて離れたターン数
	hazardAvoided int
	// chokeWait・ambushWait は expert が通路の入口・距離2の敵を足踏みで待ったターン数（bot_tactics_test.go）
	chokeWait, ambushWait int
	// house は expert がこの階で気づいたモンスターハウス、houseWait はそこで通路の入口で待ったターン数、
	// logLen・logTail はメッセージ履歴の既読位置（bot_house_test.go）
	house     botHouse
	houseWait int
	logLen    int
	logTail   string
	// searched・searchFloor・searchCount は隠し通路を調べた場所と、この階で調べた回数（bot_terrain_test.go）
	searched    map[hiddenSpot]int
	searchFloor int
	searchCount int
	searchRound int
	// potJob は expert が何ターンかに分けて行う壺の作業（bot_expert_wave11_test.go）
	potJob *botPotJob
	// 店（bot_shop_test.go）: buyTarget は expert が買いに行く商品、shopRejected は払えずに返した商品、
	// keeperSearched は店主の隣で罠を調べたマス
	buyTarget      Item
	shopRejected   map[Item]bool
	keeperSearched map[keeperSearchKey]bool
	// stairsRest は expert が階段の上で休んだターン数（bot_explore_test.go）、restFloor はその階、
	// shopFloor は店の記録（buyTarget・shopRejected）の階
	stairsRest, restFloor, shopFloor int
}

func newBot(g *Game, in *VirtualInput, p persona, seed int64) *bot {
	var stream uint64
	for _, r := range string(p) {
		stream = stream*31 + uint64(r)
	}
	return &bot{g: g, in: in, persona: p, rng: rand.New(rand.NewPCG(uint64(seed), stream)), lastTurn: g.turnCount}
}

// tap はキーを1フレーム押して離し、離した状態で1フレーム進めてから入力待ちまで進める
// （離すフレームを挟まないと、続けて同じキーを押したときに「押した瞬間」にならない）。
func (b *bot) tap(keys ...Key) error {
	b.in.Press(keys...)
	if err := b.g.StepFrame(); err != nil {
		return err
	}
	b.in.Release(keys...)
	if err := b.g.StepFrame(); err != nil {
		return err
	}
	return b.g.RunUntilInputReady(botMaxFrames)
}

// directionKeys は移動量を方向キーの組み合わせに変換する（斜めは2キー同時押し）。
func directionKeys(dx, dy int) []Key {
	var keys []Key
	switch {
	case dy < 0:
		keys = append(keys, KeyUp)
	case dy > 0:
		keys = append(keys, KeyDown)
	}
	switch {
	case dx < 0:
		keys = append(keys, KeyLeft)
	case dx > 0:
		keys = append(keys, KeyRight)
	}
	return keys
}

// move は (dx, dy) へ1歩動く。未払いのまま店を出る1歩は踏まず（足踏み）、expert の経路が正面の
// 割れた壁を通るときは殴って壊し（bot_terrain_test.go）、怒っていない店主の隣の調べていないマスへは
// 先に Z で罠を調べる（bot_shop_test.go）。
func (b *bot) move(dx, dy int) error {
	if botFeature("shopsafe") && b.leavesShopUnpaid(dx, dy) {
		b.act = "holdUnpaid"
		return b.stepInPlace()
	}
	if b.breaksCrackedWall(dx, dy) {
		return b.attack(dx, dy)
	}
	if done, err := b.searchBeforeStep(dx, dy); done || err != nil {
		return err
	}
	return b.tap(directionKeys(dx, dy)...)
}

// attack は A＋方向キーで向きを変えてから Z で正面を攻撃する。三方向の武器の横の振りが怒っていない
// 店主に当たるなら振らずに離れる（離れられなければ足踏み）。
func (b *bot) attack(dx, dy int) error {
	if b.swingHitsShopkeeper(dx, dy) {
		b.act = "avoidKeeper"
		if fx, fy, ok := b.fleeStep(); ok && !b.leavesShopUnpaid(fx, fy) {
			return b.tap(directionKeys(fx, fy)...)
		}
		return b.stepInPlace()
	}
	if err := b.tap(append([]Key{KeyA}, directionKeys(dx, dy)...)...); err != nil {
		return err
	}
	return b.tap(KeyZ)
}

// windowOpen は行動入力を受け付けないウィンドウが開いているかどうか。
func (b *bot) windowOpen() bool {
	g := b.g
	if g.itemTargetEffect != nil {
		return true
	}
	return g.showInventory || g.ShowGroundItem || g.showMenu || g.showEmptyInventoryMessage ||
		g.showMessageLog || g.showSettings || g.showSuspendPrompt || g.showHelp || g.showBlankCardMenu ||
		g.showDifficultySelect || g.showClearScreen || g.showRunHistory || g.showNameInput || g.showItemActions || g.showItemDescription || g.useIdentifyItem || g.usePotInsert ||
		g.shopPrompt != nil
}

// closeWindows は開いているウィンドウを閉じる（C で全て閉じ、閉じない種類は X で戻る）。
func (b *bot) closeWindows() error {
	for i := 0; i < 8 && b.windowOpen(); i++ {
		key := KeyC
		if b.g.showNameInput || b.g.showMessageLog || b.g.showHelp || b.g.showSettings || b.g.showSuspendPrompt || b.g.showBlankCardMenu || b.g.shopPrompt != nil || i%2 == 1 {
			key = KeyX
		}
		if b.g.showDifficultySelect || b.g.showClearScreen {
			key = KeyZ
		}
		if b.g.shopPrompt != nil {
			key = b.shopPromptKey()
		}
		if err := b.tap(key); err != nil {
			return err
		}
	}
	return nil
}

// useInventoryItem は本物のメニュー経路（C→道具→アイテム選択→行動選択）で所持品 index に actionName を行う。
// 行動が見つからなければ何もせずメニューを閉じて false を返す。
func (b *bot) useInventoryItem(index int, actionName string) (bool, error) {
	if err := b.openItemActions(index); err != nil {
		return false, err
	}
	if ok, err := b.chooseItemAction(actionName); !ok || err != nil {
		return false, err
	}
	// 真実の眼のカードは識別する道具を選んで確定する（取り消すとカードが残り、同じ判断を繰り返して行き詰まる）
	if b.g.useIdentifyItem && b.g.showInventory {
		if err := b.confirmIdentifyTarget(); err != nil {
			return true, err
		}
	}
	// 壺・白紙のカードなど、ほかに続けて対象を選ぶ画面が開いたら取り消す
	return true, b.closeWindows()
}

// openItemActions は C→道具→所持品 index を選んで行動メニューを開く（開けなければ閉じる）。
func (b *bot) openItemActions(index int) error {
	g := b.g
	if err := b.tap(KeyC); err != nil { // メニューを開く（初期位置は「道具」）
		return err
	}
	if !g.showMenu {
		return b.closeWindows()
	}
	if err := b.tap(KeyZ); err != nil {
		return err
	}
	if !g.showInventory {
		return b.closeWindows()
	}
	for i := 0; i < 2*g.state.Player.MaxInventory && g.selectedItemIndex != index; i++ {
		key := KeyDown
		if g.selectedItemIndex > index {
			key = KeyUp
		}
		if err := b.tap(key); err != nil {
			return err
		}
	}
	if g.selectedItemIndex != index {
		return b.closeWindows()
	}
	return b.tap(KeyZ) // 行動メニュー
}

// chooseItemAction は開いた行動メニューで actionName を選んで実行する。見つからなければ閉じて false。
func (b *bot) chooseItemAction(actionName string) (bool, error) {
	g := b.g
	actionIndex := -1
	for i, name := range g.currentItemActions() {
		if name == actionName {
			actionIndex = i
		}
	}
	if !g.showItemActions || actionIndex < 0 {
		return false, b.closeWindows()
	}
	for i := 0; i < 10 && g.selectedActionIndex != actionIndex; i++ {
		key := KeyDown
		if g.selectedActionIndex > actionIndex {
			key = KeyUp
		}
		if err := b.tap(key); err != nil {
			return false, err
		}
	}
	if err := b.tap(KeyZ); err != nil {
		return false, err
	}
	return true, nil
}

func (b *bot) findInventory(match func(Item) bool) int {
	for i, item := range b.g.state.Player.Inventory {
		if match(item) {
			return i
		}
	}
	return -1
}

func isFoodItem(item Item) bool { _, ok := item.(*Food); return ok }

func isHealingItem(item Item) bool {
	potion, ok := item.(*Potion)
	return ok && (potion.Health > 0 || potion.FullRecovery)
}

func isUsableAttackItem(item Item) bool {
	switch item.(type) {
	case *Card, *Cane:
		return true
	}
	return false
}

// adjacentEnemy は隣のマス（斜めは角を抜けられる場合のみ）にいる敵への方向を返す。
func (b *bot) adjacentEnemy() (int, int, bool) {
	p := b.g.state.Player
	for _, enemy := range b.g.state.Enemies {
		dx, dy := enemy.X-p.X, enemy.Y-p.Y
		if botIgnoresEnemy(enemy) {
			continue
		}
		if abs(dx) <= 1 && abs(dy) <= 1 && (dx != 0 || dy != 0) && b.canStep(p.X, p.Y, dx, dy) {
			return dx, dy, true
		}
	}
	return 0, 0, false
}

func (b *bot) nearestEnemy(maxDistance int) (Enemy, bool) {
	p := b.g.state.Player
	best, found := Enemy{}, false
	bestDistance := maxDistance + 1
	for _, enemy := range b.g.state.Enemies {
		if botIgnoresEnemy(enemy) {
			continue
		}
		if d := max(abs(enemy.X-p.X), abs(enemy.Y-p.Y)); d < bestDistance {
			best, bestDistance, found = enemy, d, true
		}
	}
	return best, found
}

func (b *bot) bossAlive() bool {
	if !isBossFloor(b.g.Floor) || b.g.bossDefeated {
		return false
	}
	for _, enemy := range b.g.state.Enemies {
		if MonsterDefinitions[enemy.ID].IsBoss {
			return true
		}
	}
	return false
}

// canStep はマップ上でその方向へ1歩進めるか（斜めは両隣が壁でないこと。ゲームの移動規則と同じ）。
func (b *bot) canStep(x, y, dx, dy int) bool {
	m := b.g.state.Map
	nx, ny := x+dx, y+dy
	if ny < 0 || ny >= len(m) || nx < 0 || nx >= len(m[0]) || m[ny][nx].Blocked {
		return false
	}
	if dx != 0 && dy != 0 && (m[y][nx].Blocked || m[ny][x].Blocked) {
		return false
	}
	return true
}

var botSteps = [8][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}, {1, -1}, {1, 1}, {-1, 1}, {-1, -1}}

// pathStep は幅優先探索で goal を満たす最寄りのマスへ向かう最初の1歩を返す。
// 敵のいるマスと発見済みの罠は通らない（敵のいるマスが goal なら隣まで進む）。
// unpaidRiskTrap は未払いの品を持って踏むと泥棒になりうる罠か（店の外へ出す落とし穴・ワープ、
// 店主や商品に爆風が届く地雷）。
func unpaidRiskTrap(id int) bool {
	switch id {
	case 3, 5, 6, 11:
		return true
	}
	return false
}

func (b *bot) pathStep(goal func(x, y int) bool) (int, int, bool) {
	return b.pathStepThrough(goal, false)
}

// pathStepThrough は pathStep と同じだが、through が true なら敵と発見済みの罠も通れるものとして探す
// （通路をふさがれて行き先が無いときの最後の手段）。
func (b *bot) pathStepThrough(goal func(x, y int) bool, through bool) (int, int, bool) {
	g := b.g
	height, width := len(g.state.Map), len(g.state.Map[0])
	blocked := make([]bool, width*height)
	for _, enemy := range g.state.Enemies {
		if through {
			break
		}
		if enemy.X >= 0 && enemy.Y >= 0 && enemy.X < width && enemy.Y < height {
			blocked[enemy.Y*width+enemy.X] = true
		}
	}
	b.blockShopItems(blocked, width, height)
	goods := b.shopGoodsMask(width, height)
	// 未払いの品を持つ間は、ふさがれた道を越えるときも、見つけた店を去らせる罠（落とし穴・ワープ）と地雷を踏まない
	// （泥棒になる。段11の統合で見つけた型）。ほかの罠は越えてよい（罠と商品に囲まれて動けなくなるため）
	keepTraps := botFeature("shopsafe") && b.unpaidTotal() > 0
	unpaidRisk := func(x, y int) bool {
		i := g.mapTrapIndexAt(x, y)
		return keepTraps && i >= 0 && g.state.MapTraps[i].Discovered && unpaidRiskTrap(g.state.MapTraps[i].TemplateID)
	}
	if keepTraps {
		// 行き先のマスは通れないマスでも選ばれる（敵を行き先にするため）ので、行き先からも外す
		orig := goal
		goal = func(x, y int) bool { return !unpaidRisk(x, y) && orig(x, y) }
	}
	for _, trap := range g.state.MapTraps {
		if b.mineNearKeeper(trap.X, trap.Y) {
			blocked[trap.Y*width+trap.X] = true // 踏むと爆風が店主に当たる
			if goods != nil {
				goods[trap.Y*width+trap.X] = true
			}
		}
		if through && !unpaidRisk(trap.X, trap.Y) {
			continue
		}
		if trap.Discovered && trap.X >= 0 && trap.Y >= 0 && trap.X < width && trap.Y < height {
			blocked[trap.Y*width+trap.X] = true
		}
	}
	if b.knownCrackedWall() { // 見えている割れた壁は壊して通れるものとして探す（expert）
		return b.pathStepWeighted(goal, blocked)
	}
	start := g.state.Player.Y*width + g.state.Player.X
	first := make([]int, width*height) // 始点から最初に踏み出した方向（+1）。0 は未訪問
	queue := []int{start}
	first[start] = -1
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		x, y := cur%width, cur/width
		for dir, step := range botSteps {
			if !b.canStep(x, y, step[0], step[1]) {
				continue
			}
			nx, ny := x+step[0], y+step[1]
			next := ny*width + nx
			if first[next] != 0 || (goods != nil && goods[next]) {
				continue
			}
			dirFromStart := dir + 1
			if cur != start {
				dirFromStart = first[cur]
			}
			if goal(nx, ny) {
				s := botSteps[dirFromStart-1]
				return s[0], s[1], true
			}
			if blocked[next] {
				continue
			}
			first[next] = dirFromStart
			queue = append(queue, next)
		}
	}
	return 0, 0, false
}

func (b *bot) stairsGoal(x, y int) bool { return b.g.state.Map[y][x].Type == "stairs" }

func (b *bot) unexploredGoal(x, y int) bool {
	tile := b.g.state.Map[y][x]
	return !tile.Visited && !tile.Blocked
}

func (b *bot) itemGoal(x, y int) bool {
	full := len(b.g.state.Player.Inventory) >= b.g.state.Player.MaxInventory
	for _, item := range b.g.state.Items {
		if ix, iy := item.GetPosition(); ix == x && iy == y && !isStuckSanctuaryItem(item) && !botAvoidsItem(item) {
			if _, isMoney := item.(*Money); full && !isMoney {
				continue // 満杯でもお金は拾える
			}
			return true
		}
	}
	return false
}

func (b *bot) enemyGoal(target Enemy) func(x, y int) bool {
	return func(x, y int) bool { return x == target.X && y == target.Y }
}

// wantsToDescend は階段の確認が開いたときに降りるかどうか。
func (b *bot) wantsToDescend() bool {
	if b.persona == personaDiver || b.g.floorTurns > botFloorTurnLimit {
		return true
	}
	if _, _, ok := b.pathStep(b.visibleFoodGoal); ok {
		return false // 見えている食料を拾ってから降りる
	}
	if b.hungry() {
		return true // 食料が無く空腹なら、探索より次の階で食料を探す
	}
	_, _, unexplored := b.pathStep(b.unexploredGoal)
	return !unexplored
}

// step は1回分の行動を決めて入力する。
func (b *bot) step() error {
	g := b.g
	defer func() {
		if g.turnCount != b.lastTurn {
			b.idle = 0
			b.lastTurn = g.turnCount
		} else {
			b.idle++
		}
	}()
	if b.idle > botStuckLimit {
		return errBotStuck
	}
	if g.specialItemUseBlocked() {
		return b.stepInPlace()
	}
	if b.persona == personaRandom {
		return b.randomInput()
	}
	if b.persona == personaExpert {
		return b.expertStep()
	}
	if g.showStairsPrompt {
		if b.wantsToDescend() && g.selectedOption == 0 {
			return b.tap(KeyZ)
		}
		return b.tap(KeyX)
	}
	if b.windowOpen() {
		return b.closeWindows()
	}
	if done, err := b.returnUnpaidItem(); done || err != nil {
		return err
	}
	// 何度も行動が空振りしたら、ランダムな方向へ動いて抜け出す
	if b.idle > 20 && b.idle%5 == 0 {
		s := botSteps[b.rng.IntN(len(botSteps))]
		return b.move(s[0], s[1])
	}

	player := &g.state.Player
	if used, err := b.eatIfStarving(15); used || err != nil {
		return err
	}
	lowHP := player.Health*100 < player.MaxHealth*40
	if b.persona == personaCautious && lowHP {
		if i := b.findInventory(isHealingItem); i >= 0 {
			if used, err := b.useInventoryItem(i, "使う"); used || err != nil {
				return err
			}
		}
	}
	if b.persona != personaRandom {
		if done, err := b.equipUpgrades(); done || err != nil {
			return err
		}
	}

	if dx, dy, ok := b.adjacentEnemy(); ok {
		if b.persona == personaCautious && lowHP && !b.bossAlive() {
			if fx, fy, ok := b.fleeStep(); ok {
				return b.move(fx, fy)
			}
		}
		if b.persona == personaItemUser && b.rng.IntN(3) == 0 {
			if done, err := b.useAttackItem(dx, dy); done || err != nil {
				return err
			}
		}
		return b.attack(dx, dy)
	}

	if b.persona == personaItemUser {
		if enemy, ok := b.nearestEnemy(4); ok && b.rng.IntN(4) == 0 {
			if done, err := b.useAttackItem(enemy.X-player.X, enemy.Y-player.Y); done || err != nil {
				return err
			}
		}
	}

	// 食料: 満杯なら価値の低い道具を手放して空きを作り、足元の食料を拾う
	if done, err := b.makeRoomForFood(); done || err != nil {
		return err
	}
	if done, err := b.pickUpHere(); done || err != nil {
		return err
	}

	// 階段の上で確認を閉じた後でも、降りたくなったら S で確認を開き直す
	if g.state.Map[player.Y][player.X].Type == "stairs" && !b.bossAlive() && b.wantsToDescend() {
		return b.tap(KeyS)
	}

	// 目的地: ボス > 見えている食料 > （長居・空腹なら）階段 > （性格に応じて）敵 > アイテム > 未探索 > 階段
	var goals []func(x, y int) bool
	if b.bossAlive() {
		if boss, ok := b.nearestBoss(); ok {
			goals = append(goals, b.enemyGoal(boss))
		}
	}
	switch b.persona {
	case personaDiver:
		goals = append(goals, b.stairsGoal, b.unexploredGoal)
	default:
		goals = append(goals, b.visibleFoodGoal)
		if g.floorTurns > botFloorTurnLimit || b.hungry() {
			goals = append(goals, b.stairsGoal)
		}
		if b.persona == personaAggressive || b.persona == personaItemUser {
			if enemy, ok := b.nearestEnemy(10); ok {
				goals = append(goals, b.enemyGoal(enemy))
			}
		}
		goals = append(goals, b.pickableItemGoal, b.unexploredGoal, b.stairsGoal)
	}
	for _, goal := range goals {
		if dx, dy, ok := b.pathStep(goal); ok {
			return b.move(dx, dy)
		}
	}
	// 通路を敵や発見済みの罠にふさがれているときは、敵を倒し罠を踏んで進む（同じ階に閉じ込められない）
	for _, goal := range []func(x, y int) bool{b.stairsGoal, b.unexploredGoal} {
		if dx, dy, ok := b.pathStepThrough(goal, true); ok {
			if isOccupied(g, player.X+dx, player.Y+dy) {
				return b.attack(dx, dy)
			}
			return b.move(dx, dy)
		}
	}
	// 行き先が無ければ足踏み（X＋Z）でターンを進める
	return b.stepInPlace()
}

func (b *bot) nearestBoss() (Enemy, bool) {
	for _, enemy := range b.g.state.Enemies {
		if MonsterDefinitions[enemy.ID].IsBoss {
			return enemy, true
		}
	}
	return Enemy{}, false
}

// stepInPlace は X を押したまま Z を押して足踏みする。
func (b *bot) stepInPlace() error {
	return b.tap(KeyX, KeyZ)
}

// fleeStep は隣の敵から最も離れる方向の1歩を返す。
func (b *bot) fleeStep() (int, int, bool) {
	p := b.g.state.Player
	bestDX, bestDY, best := 0, 0, -1
	for _, s := range botSteps {
		if !b.canStep(p.X, p.Y, s[0], s[1]) || isOccupied(b.g, p.X+s[0], p.Y+s[1]) {
			continue
		}
		score := 1 << 30
		for _, enemy := range b.g.state.Enemies {
			if botIgnoresEnemy(enemy) {
				continue
			}
			score = min(score, max(abs(enemy.X-p.X-s[0]), abs(enemy.Y-p.Y-s[1])))
		}
		if score > best {
			bestDX, bestDY, best = s[0], s[1], score
		}
	}
	return bestDX, bestDY, best > 1
}

// equipUpgrades は未装備の武器・盾を見つけたら装備する（装備していない枠だけ）。
func (b *bot) equipUpgrades() (bool, error) {
	player := &b.g.state.Player
	for i, item := range player.Inventory {
		if satietyHarmfulEquipment(item.GetID()) {
			continue
		}
		switch item.(type) {
		case *Weapon:
			if player.EquippedWeapon != nil {
				continue
			}
		case *Armor:
			if player.EquippedArmor != nil {
				continue
			}
		default:
			continue
		}
		return b.useInventoryItem(i, "装備")
	}
	return false, nil
}

// useAttackItem は敵の方向を向いてカード・杖を使うか、要らない所持品を投げる。
func (b *bot) useAttackItem(dx, dy int) (bool, error) {
	dx, dy = sign(dx), sign(dy)
	if b.lineHitsShopkeeper(dx, dy) {
		return false, nil
	}
	if err := b.tap(append([]Key{KeyA}, directionKeys(dx, dy)...)...); err != nil {
		return false, err
	}
	if i := b.findInventory(isUsableAttackItem); i >= 0 {
		return b.useInventoryItem(i, "使う")
	}
	player := &b.g.state.Player
	throwable := func(item Item) bool {
		if equipable, ok := item.(Equipable); ok && player.IsEquipped(equipable) {
			return false
		}
		return !isFoodItem(item) && !isHealingItem(item)
	}
	if i := b.findInventory(throwable); i >= 0 {
		return b.useInventoryItem(i, "投げる")
	}
	return false, nil
}

// randomInputKeys はランダム入力で押すキー（デバッグ用の F1 は除く）。
var randomInputKeys = []Key{KeyUp, KeyDown, KeyLeft, KeyRight, KeyShift, KeyZ, KeyX, KeyA, KeyC, KeyD, KeySpace, KeyF, KeyS, KeyN, KeyH, KeyL, KeyEnter}

// randomInput は1〜2個のキーをランダムに押す。半分は方向キーにして、フロアを歩き回るようにする。
func (b *bot) randomInput() error {
	if b.rng.IntN(2) == 0 {
		s := botSteps[b.rng.IntN(len(botSteps))]
		return b.move(s[0], s[1])
	}
	keys := []Key{randomInputKeys[b.rng.IntN(len(randomInputKeys))]}
	if b.rng.IntN(3) == 0 {
		keys = append(keys, randomInputKeys[b.rng.IntN(len(randomInputKeys))])
	}
	return b.tap(keys...)
}

// --- 食料の扱い（全性格で共通。random は使わない） ---

// botHungrySatiety は満腹度がこれ以下で手持ちの食料が無ければ、探索より食料と階段を優先する目安。
const botHungrySatiety = 30

// discardKey は食料のために手放した道具を、同じ階で拾い直さないための記録。
type discardKey struct{ floor, id int }

func (b *bot) wasDiscarded(item Item) bool {
	return b.discarded[discardKey{b.g.Floor, item.GetID()}]
}

// foodCarried は手持ちの食料の数（腐ったウインナーを含む）。
func (b *bot) foodCarried() int {
	n := 0
	for _, item := range b.g.state.Player.Inventory {
		if isFoodItem(item) {
			n++
		}
	}
	return n
}

// hungry は満腹度が低く、手持ちの食料も無いかどうか。
func (b *bot) hungry() bool {
	return b.g.state.Player.Satiety <= botHungrySatiety && b.foodCarried() == 0
}

// inventoryFull は所持品が満杯かどうか。
func (b *bot) inventoryFull() bool {
	p := &b.g.state.Player
	return len(p.Inventory) >= p.MaxInventory
}

// visibleFoodGoal は (x, y) に見えている（ミニマップに表示されている）食料があるかどうか。
// 自分で手放した食料は拾い直さないので目標にしない（足元で往復しないため）。
// 所持品が満杯でも食料は目標にする（着く前に makeRoomForFood で空きを作る）。
func (b *bot) visibleFoodGoal(x, y int) bool {
	for _, item := range b.g.state.Items {
		if ix, iy := item.GetPosition(); ix == x && iy == y && isFoodItem(item) && item.GetShowOnMiniMap() && !isStuckSanctuaryItem(item) && !b.wasDiscarded(item) && !botAvoidsItem(item) {
			return true
		}
	}
	return false
}

// pickableItemGoal は itemGoal から、食料のために手放した道具を除いたもの。
func (b *bot) pickableItemGoal(x, y int) bool {
	if !b.itemGoal(x, y) {
		return false
	}
	for _, item := range b.g.state.Items {
		if ix, iy := item.GetPosition(); ix == x && iy == y && !b.wasDiscarded(item) {
			return true
		}
	}
	return false
}

// itemUnderPlayer は足元の道具を返す。
func (b *bot) itemUnderPlayer() Item {
	p := b.g.state.Player
	for _, item := range b.g.state.Items {
		if x, y := item.GetPosition(); x == p.X && y == p.Y {
			return item
		}
	}
	return nil
}

// discardPriority は所持品が満杯のとき、食料のために手放す順（小さいほど先に手放す）。手放さないものは false。
func (b *bot) discardPriority(item Item) (int, bool) {
	p := &b.g.state.Player
	if equipable, ok := item.(Equipable); ok && p.IsEquipped(equipable) {
		return 0, false
	}
	if isFoodItem(item) || isHealingItem(item) {
		return 0, false
	}
	if pot, ok := item.(*Pot); ok && len(pot.Contents) > 0 {
		return 0, false // 中身（保存した食料など）ごと手放さない
	}
	switch item.(type) {
	case *Weapon, *Armor: // 予備の武器・盾
		return 1, true
	case *Accessory, *Arrow:
		return 2, true
	case *Card, *Cane, *Potion:
		switch itemRoles[item.GetID()] {
		case roleHarmful:
			return 0, true
		case roleHeal:
			return 0, false
		case roleBuff:
			return 4, true
		case roleAttack, roleEscape:
			return 5, true
		}
	}
	return 3, true
}

// makeRoomForFood は所持品が満杯で、見えている食料へ行けるとき、価値の低い道具を足元に置く
// （足元がふさがっていれば投げる）。手放したら true。
func (b *bot) makeRoomForFood() (bool, error) {
	if !b.inventoryFull() {
		return false, nil
	}
	if _, _, ok := b.pathStep(b.visibleFoodGoal); !ok {
		return false, nil
	}
	best, bestPriority := -1, 1<<30
	for i, item := range b.g.state.Player.Inventory {
		if priority, ok := b.discardPriority(item); ok && priority < bestPriority {
			best, bestPriority = i, priority
		}
	}
	// 食料を2つ以上持っていれば、強化・攻撃・逃走の道具までは手放さない
	if best < 0 || (bestPriority >= 4 && b.foodCarried() >= 2) {
		return false, nil
	}
	p := b.g.state.Player
	action := "置く"
	if b.itemUnderPlayer() != nil || b.g.state.Map[p.Y][p.X].Type == "stairs" {
		action = "投げる"
		// 向いている直線に店主がいれば投げない（当たると泥棒。段11の統合で見つけた型）
		if b.lineHitsShopkeeper(directionToDelta(p.Direction)) {
			return false, nil
		}
	}
	if b.discarded == nil {
		b.discarded = map[discardKey]bool{}
	}
	b.discarded[discardKey{b.g.Floor, p.Inventory[best].GetID()}] = true
	return b.useInventoryItem(best, action)
}

// pickUpHere は足元の食料を本物のメニュー経路（C→足元→拾う）で拾う。拾おうとしたら true。
// 所持品が満杯で拾えなかった食料の上で、空きを作った後に使う。
func (b *bot) pickUpHere() (bool, error) {
	item := b.itemUnderPlayer()
	if item == nil || !isFoodItem(item) || b.inventoryFull() || isStuckSanctuaryItem(item) || b.g.pickupBanned || (botFeature("shopsafe") && botAvoidsItem(item)) {
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

// eatIfStarving は満腹度が limit 以下なら手持ちの食料を食べる。腐ったウインナーは最後に回す。
func (b *bot) eatIfStarving(limit int) (bool, error) {
	if b.g.state.Player.Satiety > limit {
		return false, nil
	}
	i := b.findInventory(func(item Item) bool { return isFoodItem(item) && item.GetID() != rottenSausageItemID })
	if i < 0 {
		i = b.findInventory(isFoodItem)
	}
	if i < 0 {
		return false, nil
	}
	return b.useInventoryItem(i, "使う")
}

// satietyHarmfulEquipment は満腹度を余計に減らす・消費する装備（BOT は装備しない）。
// 136・137 は満腹度を使う武器、155・156・159 は満腹度を使う盾・空腹用の盾・悟りの盾、69 は重甲の盾。
func satietyHarmfulEquipment(id int) bool {
	switch id {
	case 69, 136, 137, 155, 156, 159:
		return true
	}
	return false
}

// confirmIdentifyTarget は識別の対象を選ぶ画面で、使ったカード以外の未識別の道具（無ければ別の道具）を選んで確定する。
func (b *bot) confirmIdentifyTarget() error {
	g := b.g
	target := -1
	for i, item := range g.state.Player.Inventory {
		if i == g.tmpSelectedItemIndex && !g.GroundItemActioned {
			continue
		}
		if identifiable, ok := item.(Identifiable); ok && !identifiable.IsIdentified() {
			target = i
			break
		}
		if target < 0 {
			target = -2 - i // 未識別が無いときの候補（先頭の別の道具）
		}
	}
	if target < -1 {
		target = -2 - target
	}
	if target < 0 {
		return nil
	}
	for i := 0; i < 2*g.state.Player.MaxInventory && g.selectedItemIndex != target; i++ {
		key := KeyDown
		if g.selectedItemIndex > target {
			key = KeyUp
		}
		if err := b.tap(key); err != nil {
			return err
		}
	}
	if g.selectedItemIndex != target {
		return nil
	}
	return b.tap(KeyZ)
}

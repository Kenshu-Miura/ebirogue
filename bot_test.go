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
)

var allPersonas = []persona{personaAggressive, personaCautious, personaItemUser, personaDiver, personaRandom}

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

func (b *bot) move(dx, dy int) error { return b.tap(directionKeys(dx, dy)...) }

// attack は A＋方向キーで向きを変えてから Z で正面を攻撃する。
func (b *bot) attack(dx, dy int) error {
	if err := b.tap(append([]Key{KeyA}, directionKeys(dx, dy)...)...); err != nil {
		return err
	}
	return b.tap(KeyZ)
}

// windowOpen は行動入力を受け付けないウィンドウが開いているかどうか。
func (b *bot) windowOpen() bool {
	g := b.g
	return g.showInventory || g.ShowGroundItem || g.showMenu || g.showEmptyInventoryMessage ||
		g.showMessageLog || g.showSettings || g.showSuspendPrompt || g.showHelp || g.showBlankCardMenu ||
		g.showNameInput || g.showItemActions || g.showItemDescription || g.useIdentifyItem || g.usePotInsert
}

// closeWindows は開いているウィンドウを閉じる（C で全て閉じ、閉じない種類は X で戻る）。
func (b *bot) closeWindows() error {
	for i := 0; i < 8 && b.windowOpen(); i++ {
		key := KeyC
		if b.g.showNameInput || b.g.showMessageLog || b.g.showHelp || b.g.showSettings || b.g.showSuspendPrompt || b.g.showBlankCardMenu || i%2 == 1 {
			key = KeyX
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
	g := b.g
	if err := b.tap(KeyC); err != nil { // メニューを開く（初期位置は「道具」）
		return false, err
	}
	if !g.showMenu {
		return false, b.closeWindows()
	}
	if err := b.tap(KeyZ); err != nil {
		return false, err
	}
	if !g.showInventory {
		return false, b.closeWindows()
	}
	for i := 0; i < 2*g.state.Player.MaxInventory && g.selectedItemIndex != index; i++ {
		key := KeyDown
		if g.selectedItemIndex > index {
			key = KeyUp
		}
		if err := b.tap(key); err != nil {
			return false, err
		}
	}
	if g.selectedItemIndex != index {
		return false, b.closeWindows()
	}
	if err := b.tap(KeyZ); err != nil { // 行動メニュー
		return false, err
	}
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
	// 識別・壺・白紙のカードなど、続けて対象を選ぶ画面が開いたら取り消す
	return true, b.closeWindows()
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
func (b *bot) pathStep(goal func(x, y int) bool) (int, int, bool) {
	g := b.g
	height, width := len(g.state.Map), len(g.state.Map[0])
	blocked := make([]bool, width*height)
	for _, enemy := range g.state.Enemies {
		if enemy.X >= 0 && enemy.Y >= 0 && enemy.X < width && enemy.Y < height {
			blocked[enemy.Y*width+enemy.X] = true
		}
	}
	for _, trap := range g.state.MapTraps {
		if trap.Discovered && trap.X >= 0 && trap.Y >= 0 && trap.X < width && trap.Y < height {
			blocked[trap.Y*width+trap.X] = true
		}
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
			if first[next] != 0 {
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
	if len(b.g.state.Player.Inventory) >= b.g.state.Player.MaxInventory {
		return false
	}
	for _, item := range b.g.state.Items {
		if ix, iy := item.GetPosition(); ix == x && iy == y && !isStuckSanctuaryItem(item) {
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
	if b.persona == personaRandom {
		return b.randomInput()
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
	// 何度も行動が空振りしたら、ランダムな方向へ動いて抜け出す
	if b.idle > 20 && b.idle%5 == 0 {
		s := botSteps[b.rng.IntN(len(botSteps))]
		return b.move(s[0], s[1])
	}

	player := &g.state.Player
	if player.Satiety <= 15 {
		if i := b.findInventory(isFoodItem); i >= 0 {
			if used, err := b.useInventoryItem(i, "使う"); used || err != nil {
				return err
			}
		}
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

	// 階段の上で確認を閉じた後でも、降りたくなったら S で確認を開き直す
	if g.state.Map[player.Y][player.X].Type == "stairs" && !b.bossAlive() && b.wantsToDescend() {
		return b.tap(KeyS)
	}

	// 目的地: ボス > （性格に応じて）敵 > アイテム > 未探索 > 階段
	var goals []func(x, y int) bool
	if b.bossAlive() {
		if boss, ok := b.nearestBoss(); ok {
			goals = append(goals, b.enemyGoal(boss))
		}
	}
	switch b.persona {
	case personaDiver:
		goals = append(goals, b.stairsGoal, b.unexploredGoal)
	case personaAggressive, personaItemUser:
		if enemy, ok := b.nearestEnemy(10); ok {
			goals = append(goals, b.enemyGoal(enemy))
		}
		fallthrough
	default:
		if g.floorTurns > botFloorTurnLimit {
			goals = append(goals, b.stairsGoal)
		}
		goals = append(goals, b.itemGoal, b.unexploredGoal, b.stairsGoal)
	}
	for _, goal := range goals {
		if dx, dy, ok := b.pathStep(goal); ok {
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

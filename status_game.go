//go:build !test

package main

import "fmt"

// 状態異常の付与・解除・ターン経過の共通処理（定義表は status_effects.go）。
// どれも Action の Execute の中（または効果を即時に反映してよい場所）から呼ぶ。

// StatusSource は applyPlayerStatus に渡す付与の出どころ。
type StatusSource struct {
	// Shield は状態異常耐性の盾（50%）でも防げるか。敵・罠からの付与は true、
	// 自分で薬を飲んだときなどは false（装備能力による無効化だけを見る）。
	Shield bool
	// Stack は重ねがけの規則。StackDefault なら表の規則。
	Stack StatusStack
	// Message は付いたときのメッセージ、Blocked は防いだときのメッセージ（%s に防いだ装備品の名前）。
	// 空なら Message は出さず、Blocked は「<装備品>が<状態>を防いだ」になる。
	Message string
	Blocked string
}

// guardPlayerStatus は付与の前の耐性の判定。source.Shield なら盾の判定で乱数を1回使うことがある。
func (g *Game) guardPlayerStatus(kind StatusKind, source StatusSource) (guard string, blocked bool) {
	if source.Shield {
		return g.playerStatusGuard(kind)
	}
	return g.playerImmunitySource(kind)
}

// applyPlayerStatus は耐性・重ねがけの規則・メッセージを通して海老さんに状態をかける。
// 付いたら true。防いだとき・海老さんに付かない状態のときは false。
func (g *Game) applyPlayerStatus(kind StatusKind, turns int, source StatusSource) bool {
	def, ok := statusDefOf(kind)
	if !ok || def.Player == StatusNotApplicable {
		return false
	}
	if guard, blocked := g.guardPlayerStatus(kind, source); blocked {
		message := fmt.Sprintf("%sが%sを防いだ", guard, def.Name)
		if source.Blocked != "" {
			message = fmt.Sprintf(source.Blocked, guard)
		}
		g.EnqueueMessage(message, 0.4)
		return false
	}
	g.stackPlayerStatus(kind, turns, source.Stack)
	if source.Message != "" {
		g.EnqueueMessage(source.Message, 0.4)
	}
	return true
}

// setPlayerStatus は耐性の判定を済ませた後に、表の重ねがけの規則で海老さんに状態をかける。
// 耐性の判定とかける時点が分かれている既存の付与（判定は Enqueue 時、かけるのは Execute 内）が使う。
func (g *Game) setPlayerStatus(kind StatusKind, turns int) {
	g.stackPlayerStatus(kind, turns, StackDefault)
}

// stackPlayerStatus は setPlayerStatus の重ねがけの規則を指定する版。
func (g *Game) stackPlayerStatus(kind StatusKind, turns int, rule StatusStack) {
	addStatus(&g.state.Player.StatusAilments, kind, turns, rule)
}

// clearPlayerStatus は海老さんの状態 kind を解く。
func (g *Game) clearPlayerStatus(kind StatusKind) {
	clearStatus(&g.state.Player.StatusAilments, kind)
	if def, ok := statusDefOf(kind); ok && def.RefreshMiniMap {
		g.miniMapDirty = true
	}
}

// clearPlayerBadStatuses は海老さんの悪い状態（表の Bad）をすべて解く。
func (g *Game) clearPlayerBadStatuses() {
	clearBadStatuses(&g.state.Player.StatusAilments)
}

// applyEnemyStatus は敵 index に状態をかける。ボスが受け付けない状態（bossStatusImmune）なら false。
// ボスの睡眠・混乱などの短縮は、今までどおりボスの行動時（normalizeBossStatus）に行う。
func (g *Game) applyEnemyStatus(index int, kind StatusKind, turns int) bool {
	if index < 0 || index >= len(g.state.Enemies) {
		return false
	}
	return applyStatusToEnemy(&g.state.Enemies[index], kind, turns)
}

// applyStatusToEnemy は applyEnemyStatus の、敵の値を直接受け取る版（添字の無い敵の写しなど）。
func applyStatusToEnemy(e *Enemy, kind StatusKind, turns int) bool {
	def, ok := statusDefOf(kind)
	if !ok || def.Enemy == StatusNotApplicable || bossStatusImmune(*e, kind) {
		return false
	}
	addStatus(&e.StatusAilments, kind, turns, StackDefault)
	return true
}

// clearEnemyStatus は敵 index の状態 kind を解く。
func (g *Game) clearEnemyStatus(index int, kind StatusKind) {
	if index < 0 || index >= len(g.state.Enemies) {
		return
	}
	clearStatus(&g.state.Enemies[index].StatusAilments, kind)
}

// decrementStatusAilments は海老さんと敵の状態を1ターン進め、治ったときのメッセージを出す。
func (g *Game) decrementStatusAilments() {
	for _, kind := range tickStatuses(&g.state.Player.StatusAilments, false) {
		def, _ := statusDefOf(kind)
		if def.RefreshMiniMap {
			g.miniMapDirty = true // ミニマップを更新して敵・アイテム・階段を表示
		}
		if def.PlayerRecover != "" {
			g.EnqueueMessage(def.PlayerRecover, 0.4)
		}
	}

	for i := range g.state.Enemies {
		advanceBossState(&g.state.Enemies[i])
		s := &g.state.Enemies[i].StatusAilments
		for _, kind := range tickStatuses(s, true) {
			// 睡眠のカードで眠らされた敵は目覚めた時に倍速化する（倍速の経過はこのターン済み）
			if kind == StatusSleep && wakeFromSleep(s) {
				g.EnqueueMessage(fmt.Sprintf("%sは目を覚まし、倍速で動き出した", g.state.Enemies[i].Name), 0.4)
			}
		}
	}
}

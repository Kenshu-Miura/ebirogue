package main

import (
	"fmt"
	"sort"
	"strings"
)

// 状態異常の定義表と、付与・経過・解除・表示の純粋な共通処理。
//
// 状態は StatusAilments の項目に残りターン（int）か有無（bool）で入る。項目名は中断セーブの
// 互換のため変えない。新しい状態を足すときは StatusAilments に項目を1つ、statusDefs に
// 行を1つ足すだけで、付与（applyPlayerStatus など）・ターン経過・HUD・保存（StatusAilments ごと
// 保存される）・困った時のカードに反映される。手順は docs/agents/done/wave7-A.md。
//
// *Game を使う付与・解除・経過は status_game.go（!test）。

// 状態異常の種類（StatusConfusion などの既存の種類は equipment_abilities.go）。
const (
	StatusHaste     StatusKind = "haste"      // 倍速（敵。睡眠のカードで眠った敵の目覚め、倍速のカード）
	StatusMouthSeal StatusKind = "mouth_seal" // 口封じ（カード・薬・食料を口にできない）
	StatusParalysis StatusKind = "paralysis"  // 金縛り（ダメージを受けると解ける）
	StatusSeal      StatusKind = "seal"       // 封印（特殊能力が止まる）
	StatusBackward  StatusKind = "backward"   // ゾワゾワ（敵。通常攻撃を反対向きに出す）
)

// StatusDuration は状態の持続の型。
type StatusDuration int

const (
	StatusNotApplicable StatusDuration = iota // その側には付かない（ターン経過もしない）
	StatusTurns                               // 残りターン。毎ターン1減り、0で治る
	StatusPermanent                           // 残りターンを持つが減らない（敵の目潰し）
	StatusUntilCleared                        // 有無だけ（bool）。解除されるまで続く
)

// StatusStack は付いている状態に重ねてかけたときの規則。
type StatusStack int

const (
	StackDefault   StatusStack = iota // 表の規則を使う（付与の関数の引数用）
	StackOverwrite                    // 新しいターン数で上書きする
	StackLonger                       // 長い方を残す
	StackAdd                          // 加算して MaxTurns まで
)

// statusMessageDuration は海老さんの状態異常のメッセージ（かかった・防いだ・治った）の表示時間。
// どの付与元でも同じ時間にそろえる。
const statusMessageDuration = 0.5

// statusDef は状態1つの定義。
type statusDef struct {
	Kind StatusKind
	// Name は「〜を防いだ」などのメッセージに使う名前。
	Name string
	// HUD は HUD の表示名（残りターンがある状態は「名前(n)」）。空なら表示しない。HUDOrder の昇順に並ぶ。
	HUD      string
	HUDOrder int
	// Player・Enemy はそれぞれの側での持続の型（StatusNotApplicable なら付かない）。
	Player StatusDuration
	Enemy  StatusDuration
	// Stack は重ねがけの既定の規則、MaxTurns は StackAdd の上限（0なら上限なし）。
	Stack    StatusStack
	MaxTurns int
	// Bad は困った時のカードなどで治す「悪い状態」か。
	Bad bool
	// PlayerApply は海老さんに状態がかかったときのメッセージ。付与元にかかわらずこれを出す。
	PlayerApply string
	// PlayerRecover は海老さんの状態がターン経過で治ったときのメッセージ（空なら出さない）。
	PlayerRecover string
	// RefreshMiniMap は海老さんの状態が治ったときにミニマップを描き直すか（目潰し）。
	RefreshMiniMap bool
	// BossShortened はボスにかかったとき行動時に短縮される状態か（normalizeBossStatus）。
	BossShortened bool
	// Turns・SetTurns は StatusAilments の項目を読み書きする。bool の状態は有無を 1/0 で表す。
	Turns    func(s *StatusAilments) int
	SetTurns func(s *StatusAilments, n int)
}

// statusDefs は状態の定義表。並び順がターン経過の処理と治ったときのメッセージの順になる。
// 新しい状態は末尾に追記する。
var statusDefs = []statusDef{
	{
		Kind: StatusBearTrap, Name: "トラバサミ", HUD: "トラバサミ", HUDOrder: 10,
		Player: StatusTurns, Stack: StackLonger, Bad: true,
		PlayerApply:   "海老さんはトラバサミに足を挟まれた",
		PlayerRecover: "海老さんはトラバサミから抜け出した",
		Turns:         func(s *StatusAilments) int { return s.BearTrap },
		SetTurns:      func(s *StatusAilments, n int) { s.BearTrap = n },
	},
	{
		Kind: StatusConfusion, Name: "混乱", HUD: "混乱", HUDOrder: 50,
		Player: StatusTurns, Enemy: StatusTurns, Stack: StackOverwrite, Bad: true, BossShortened: true,
		PlayerApply: "海老さんは混乱した",
		Turns:       func(s *StatusAilments) int { return s.Confusion },
		SetTurns:    func(s *StatusAilments, n int) { s.Confusion = n },
	},
	{
		Kind: StatusSleep, Name: "睡眠", HUD: "睡眠", HUDOrder: 40,
		Player: StatusTurns, Enemy: StatusTurns, Stack: StackOverwrite, Bad: true, BossShortened: true,
		PlayerApply:   "海老さんは眠った",
		PlayerRecover: "海老さんは目を覚ました",
		Turns:         func(s *StatusAilments) int { return s.Sleep },
		SetTurns:      func(s *StatusAilments, n int) { s.Sleep = n },
	},
	{
		Kind: StatusBlind, Name: "目潰し", HUD: "目潰し", HUDOrder: 60,
		Player: StatusTurns, Enemy: StatusPermanent, Stack: StackOverwrite, Bad: true,
		PlayerApply:   "海老さんは目が見えなくなった",
		PlayerRecover: "海老さんは目が見えるようになった", RefreshMiniMap: true,
		Turns:    func(s *StatusAilments) int { return s.Blind },
		SetTurns: func(s *StatusAilments, n int) { s.Blind = n },
	},
	{
		Kind: StatusSlow, Name: "鈍足", HUD: "鈍足", HUDOrder: 20,
		Player: StatusTurns, Enemy: StatusTurns, Stack: StackOverwrite, Bad: true,
		PlayerApply:   "海老さんは鈍足になった",
		PlayerRecover: "海老さんの体の速さが元に戻った",
		Turns:         func(s *StatusAilments) int { return s.Slow },
		SetTurns:      func(s *StatusAilments, n int) { s.Slow = n },
	},
	{
		// 海老さんの倍速は wave8-A で追加する（今は敵だけ）。
		Kind: StatusHaste, Name: "倍速", HUD: "倍速", HUDOrder: 30,
		Enemy: StatusTurns, Stack: StackOverwrite,
		Turns:    func(s *StatusAilments) int { return s.Haste },
		SetTurns: func(s *StatusAilments, n int) { s.Haste = n },
	},
	{
		Kind: StatusMouthSeal, Name: "口封じ", HUD: "口封じ", HUDOrder: 70,
		Player: StatusTurns, Stack: StackOverwrite, Bad: true,
		PlayerApply:   "海老さんは口を封じられた",
		PlayerRecover: "海老さんは口が動くようになった",
		Turns:         func(s *StatusAilments) int { return s.MouthSeal },
		SetTurns:      func(s *StatusAilments, n int) { s.MouthSeal = n },
	},
	{
		Kind: StatusBackward, Name: "ゾワゾワ", HUD: "ゾワゾワ", HUDOrder: 100,
		Enemy: StatusTurns, Stack: StackOverwrite, BossShortened: true,
		Turns:    func(s *StatusAilments) int { return s.Backward },
		SetTurns: func(s *StatusAilments, n int) { s.Backward = n },
	},
	{
		Kind: StatusParalysis, Name: "金縛り", HUD: "金縛り", HUDOrder: 80,
		Player: StatusUntilCleared, Enemy: StatusUntilCleared, Stack: StackOverwrite, Bad: true,
		PlayerApply: "海老さんは金縛りにあった",
		Turns:       func(s *StatusAilments) int { return boolTurns(s.Paralysis) },
		SetTurns:    func(s *StatusAilments, n int) { s.Paralysis = n > 0 },
	},
	{
		Kind: StatusSeal, Name: "封印", HUD: "封印", HUDOrder: 90,
		Player: StatusUntilCleared, Enemy: StatusUntilCleared, Stack: StackOverwrite, Bad: true,
		PlayerApply: "海老さんは封印された",
		Turns:       func(s *StatusAilments) int { return boolTurns(s.Seal) },
		SetTurns:    func(s *StatusAilments, n int) { s.Seal = n > 0 },
	},
}

// enemyNapSleep は仮眠（フロア生成時に眠っている敵）の Sleep の値。通常の睡眠と区別し、ターン経過では減らない。
const enemyNapSleep = -1

func boolTurns(b bool) int {
	if b {
		return 1
	}
	return 0
}

// statusDefOf は kind の定義を返す。状態異常の表に無い種類（呪い・さびなど）は false。
func statusDefOf(kind StatusKind) (statusDef, bool) {
	for _, def := range statusDefs {
		if def.Kind == kind {
			return def, true
		}
	}
	return statusDef{}, false
}

// statusAppliedMessage は海老さんに状態 kind がかかったときのメッセージ。
func statusAppliedMessage(kind StatusKind) string {
	if def, ok := statusDefOf(kind); ok && def.PlayerApply != "" {
		return def.PlayerApply
	}
	return "海老さんは" + statusKindLabel(kind) + "状態になった"
}

// statusBlockedMessage は装備品 guard が状態 kind を防いだときのメッセージ。
func statusBlockedMessage(kind StatusKind, guard string) string {
	return fmt.Sprintf("%sが%sを防いだ", guard, statusKindLabel(kind))
}

// statusResultMessage は耐性の判定の結果に応じたメッセージ（防いだ・かかった）。
func statusResultMessage(kind StatusKind, guard string, blocked bool) string {
	if blocked {
		return statusBlockedMessage(kind, guard)
	}
	return statusAppliedMessage(kind)
}

// statusDuration は海老さん（enemy=false）または敵の側での持続の型を返す。
func (d statusDef) duration(enemy bool) StatusDuration {
	if enemy {
		return d.Enemy
	}
	return d.Player
}

// statusTurns は kind の残りターン（bool の状態は 1/0）を返す。表に無い種類は0。
func statusTurns(s StatusAilments, kind StatusKind) int {
	def, ok := statusDefOf(kind)
	if !ok {
		return 0
	}
	return def.Turns(&s)
}

// statusActive は kind にかかっているかを返す（仮眠の -1 は含まない）。
func statusActive(s StatusAilments, kind StatusKind) bool {
	return statusTurns(s, kind) > 0
}

// stackStatusTurns は重ねがけの規則で新しい残りターンを決める。
func stackStatusTurns(rule StatusStack, current, turns, maxTurns int) int {
	switch rule {
	case StackLonger:
		return max(current, turns)
	case StackAdd:
		next := max(current, 0) + turns
		if maxTurns > 0 {
			next = min(next, maxTurns)
		}
		return next
	default:
		return turns
	}
}

// addStatus は s に kind を turns ターン（bool の状態は turns > 0 で有り）かける。
// rule が StackDefault なら表の規則を使う。表に無い種類は何もしない。
func addStatus(s *StatusAilments, kind StatusKind, turns int, rule StatusStack) {
	def, ok := statusDefOf(kind)
	if !ok {
		return
	}
	if rule == StackDefault {
		rule = def.Stack
	}
	def.SetTurns(s, stackStatusTurns(rule, def.Turns(s), turns, def.MaxTurns))
}

// setStatusTurns は残りターンをそのまま書き込む（仮眠の -1 など、重ねがけの規則を通さない値用）。
func setStatusTurns(s *StatusAilments, kind StatusKind, turns int) {
	if def, ok := statusDefOf(kind); ok {
		def.SetTurns(s, turns)
	}
}

// clearStatus は kind を解く（仮眠も解ける）。睡眠を解いても目覚め時の倍速化（HasteOnWake）は残す。
func clearStatus(s *StatusAilments, kind StatusKind) {
	setStatusTurns(s, kind, 0)
}

// hasBadStatus は悪い状態（statusDef.Bad）にかかっているかを返す。
func hasBadStatus(s StatusAilments) bool {
	for _, def := range statusDefs {
		if def.Bad && def.Turns(&s) > 0 {
			return true
		}
	}
	return false
}

// clearBadStatuses は悪い状態をすべて解き、目覚め時の倍速化も取り消す。
func clearBadStatuses(s *StatusAilments) {
	for _, def := range statusDefs {
		if def.Bad {
			def.SetTurns(s, 0)
		}
	}
	s.HasteOnWake = false
}

// markHasteOnWake は睡眠から覚めたとき倍速になる印を付ける（睡眠のカード）。
func markHasteOnWake(s *StatusAilments) {
	s.HasteOnWake = true
}

// 睡眠のカードで眠らされた敵が目覚めた時に倍速化するターン数
const hasteOnWakeTurns = 5

// wakeFromSleep は睡眠状態を解除し、目覚め時倍速化フラグが立っていれば倍速状態にする。
// 倍速化した場合は true を返す。
func wakeFromSleep(status *StatusAilments) bool {
	clearStatus(status, StatusSleep)
	if !status.HasteOnWake {
		return false
	}
	status.HasteOnWake = false
	setStatusTurns(status, StatusHaste, hasteOnWakeTurns)
	return true
}

// tickStatuses は1ターン分の経過を進め、このターンで治った状態を表の順に返す。
// 減るのはその側で StatusTurns の状態だけ（仮眠の -1 や敵の目潰しは減らない）。
func tickStatuses(s *StatusAilments, enemy bool) []StatusKind {
	var ended []StatusKind
	for _, def := range statusDefs {
		if def.duration(enemy) != StatusTurns {
			continue
		}
		turns := def.Turns(s)
		if turns <= 0 {
			continue
		}
		def.SetTurns(s, turns-1)
		if turns == 1 {
			ended = append(ended, def.Kind)
		}
	}
	return ended
}

// formatPlayerStatus は HUD の状態表示（「鈍足(2) 金縛り」など）を返す。
func formatPlayerStatus(status StatusAilments) string {
	defs := make([]statusDef, 0, len(statusDefs))
	for _, def := range statusDefs {
		if def.HUD != "" && def.Turns(&status) > 0 {
			defs = append(defs, def)
		}
	}
	if len(defs) == 0 {
		return "正常"
	}
	sort.SliceStable(defs, func(i, j int) bool { return defs[i].HUDOrder < defs[j].HUDOrder })
	parts := make([]string, len(defs))
	for i, def := range defs {
		if def.Player == StatusUntilCleared || def.Enemy == StatusUntilCleared {
			parts[i] = def.HUD
		} else {
			parts[i] = fmt.Sprintf("%s(%d)", def.HUD, def.Turns(&status))
		}
	}
	return strings.Join(parts, " ")
}

package main

import "sort"

// Key はゲーム内の論理キー。物理キーとの対応は InputSource の実装が決める
// （将来のキー設定はこの対応表を差し替える形で実装する）。
type Key int

const (
	KeyUp Key = iota
	KeyDown
	KeyLeft
	KeyRight
	KeyShift
	KeyZ
	KeyX
	KeyA
	KeyC
	KeyD
	KeySpace
	KeyF
	KeyS
	KeyN
	KeyH
	KeyL
	KeyEnter
	KeyF1
	keyCount // 論理キーの総数（番兵）
)

var keyNames = [keyCount]string{
	"Up", "Down", "Left", "Right", "Shift", "Z", "X", "A", "C", "D",
	"Space", "F", "S", "N", "H", "L", "Enter", "F1",
}

func (k Key) String() string {
	if k < 0 || k >= keyCount {
		return "Unknown"
	}
	return keyNames[k]
}

// InputSource はキー入力の取得元。本番は Ebiten、テストと BOT は仮想入力を使う。
type InputSource interface {
	Pressed(k Key) bool     // 押されているか
	JustPressed(k Key) bool // このフレームで押されたか
	PressDuration(k Key) int
	Update() // フレームの区切り（Game.Update の先頭で1回呼ぶ）
}

// InputEvent はフレーム番号付きのキー押下・解放の記録。
type InputEvent struct {
	Frame int
	Key   Key
	Down  bool
}

// VirtualInput はスクリプトでキーを押下・解放する InputSource。
// Press/Release は次の Update から反映され、押下継続フレーム数は Ebiten と同じ規則で数える
// （押した最初のフレームが 1 で、JustPressed はその1フレームだけ true）。
type VirtualInput struct {
	frame    int
	desired  [keyCount]bool
	duration [keyCount]int
	script   []InputEvent // 再生する予定のイベント（Frame 昇順）
}

func NewVirtualInput() *VirtualInput {
	return &VirtualInput{}
}

// NewReplayInput は記録したイベント列を再生する仮想入力を作る。
func NewReplayInput(events []InputEvent) *VirtualInput {
	v := &VirtualInput{script: append([]InputEvent(nil), events...)}
	sort.SliceStable(v.script, func(i, j int) bool { return v.script[i].Frame < v.script[j].Frame })
	return v
}

// Press は次のフレームからキーを押した状態にする。
func (v *VirtualInput) Press(keys ...Key) {
	for _, k := range keys {
		v.desired[k] = true
	}
}

// Release は次のフレームからキーを離した状態にする。
func (v *VirtualInput) Release(keys ...Key) {
	for _, k := range keys {
		v.desired[k] = false
	}
}

// ReleaseAll はすべてのキーを離す。
func (v *VirtualInput) ReleaseAll() {
	v.desired = [keyCount]bool{}
}

// Frame は直近の Update で始まったフレームの番号（最初の Update で 1）。
func (v *VirtualInput) Frame() int { return v.frame }

// ScriptDone は再生予定のイベントをすべて適用し終えたかどうか。
func (v *VirtualInput) ScriptDone() bool { return len(v.script) == 0 }

func (v *VirtualInput) Update() {
	v.frame++
	for len(v.script) > 0 && v.script[0].Frame <= v.frame {
		e := v.script[0]
		v.script = v.script[1:]
		if e.Key >= 0 && e.Key < keyCount {
			v.desired[e.Key] = e.Down
		}
	}
	for k := range v.desired {
		if v.desired[k] {
			v.duration[k]++
		} else {
			v.duration[k] = 0
		}
	}
}

func (v *VirtualInput) valid(k Key) bool { return k >= 0 && k < keyCount }

func (v *VirtualInput) Pressed(k Key) bool { return v.valid(k) && v.duration[k] > 0 }

func (v *VirtualInput) JustPressed(k Key) bool { return v.valid(k) && v.duration[k] == 1 }

func (v *VirtualInput) PressDuration(k Key) int {
	if !v.valid(k) {
		return 0
	}
	return v.duration[k]
}

// InputRecorder は別の InputSource を包み、キーの押下・解放をフレーム番号付きで記録する。
// 記録した Events を NewReplayInput に渡すと同じ入力を再生できる（リプレイ・不具合再現用）。
type InputRecorder struct {
	inner   InputSource
	frame   int
	pressed [keyCount]bool
	events  []InputEvent
}

func NewInputRecorder(inner InputSource) *InputRecorder {
	return &InputRecorder{inner: inner}
}

// Reset は記録を消してフレーム番号を 0 に戻す（新しい冒険の開始時に呼ぶ）。
// 押しっぱなしのキーは次の Update で改めて押下として記録される。
func (r *InputRecorder) Reset() {
	r.frame = 0
	r.pressed = [keyCount]bool{}
	r.events = nil
}

// Events はこれまでに記録したイベントのコピーを返す。
func (r *InputRecorder) Events() []InputEvent {
	return append([]InputEvent(nil), r.events...)
}

func (r *InputRecorder) Update() {
	r.inner.Update()
	r.frame++
	for k := Key(0); k < keyCount; k++ {
		now := r.inner.Pressed(k)
		if now != r.pressed[k] {
			r.pressed[k] = now
			r.events = append(r.events, InputEvent{Frame: r.frame, Key: k, Down: now})
		}
	}
}

func (r *InputRecorder) Pressed(k Key) bool { return r.inner.Pressed(k) }

func (r *InputRecorder) JustPressed(k Key) bool { return r.inner.JustPressed(k) }

func (r *InputRecorder) PressDuration(k Key) int { return r.inner.PressDuration(k) }

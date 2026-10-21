//go:build !test

package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

// 同じソースのネイティブ・WASM・go test で共通のビルド識別を使う。
//
//go:embed *.go go.mod go.sum
var replaySources embed.FS
var replayBuildOnce sync.Once
var replaySourceID string
var replayBuildID string

func replayBuild() string {
	if replayBuildID != "" {
		return replayBuildID
	}
	replayBuildOnce.Do(func() {
		names, _ := replaySources.ReadDir(".")
		h := sha256.New()
		for _, name := range names {
			b, _ := replaySources.ReadFile(name.Name())
			h.Write([]byte(name.Name()))
			h.Write([]byte{0})
			h.Write(b)
		}
		replaySourceID = hex.EncodeToString(h.Sum(nil))
	})
	return replaySourceID
}

func (g *Game) replayResult() ReplayResult {
	return ReplayResult{g.Floor, g.state.Player.Health, g.turnCount, g.lastDamageSource}
}
func (g *Game) replayCheck() string {
	rng, _ := g.rng.MarshalState()
	save := g.buildSaveData()
	b, _ := json.Marshal([]any{g.replayResult(), rng, save.Player, save.Enemies, save.Items, save.MapTraps, save.Stats, save.GenocidedIDs, save.PendingEnemySpawns, g.floorTurns, g.bossDefeated})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:12])
}
func (g *Game) beginReplayFrame() func() {
	if !replayEnabled(g.settings) {
		g.replaySuppressed = true
	}
	if g.replayPlayback || !g.persistent || !replayEnabled(g.settings) || g.replaySuppressed || g.showDifficultySelect || g.showClearScreen || g.lastRunResult != nil || g.suspendRequested {
		return func() {}
	}
	if g.replay == nil {
		g.replay = &ReplayData{Version: replayVersion, SaveVersion: saveDataVersion, Build: replayBuild(), Seed: g.seed, Difficulty: g.difficulty}
	}
	r := g.replay
	if g.replaySegment == nil {
		g.replay = nil
		initial := g.buildSaveData()
		g.replay = r
		// JSON を介してスライス・マップをゲームの可変状態から切り離す。
		b, _ := json.Marshal(initial)
		var frozen SaveData
		_ = json.Unmarshal(b, &frozen)
		seg := &ReplaySegment{Initial: &frozen, InputFrame: g.inputFrame, PreviousRoom: g.previousPlayerRoomID, Fade: g.fadeInProgress, Fast: g.fastForward}
		for _, a := range g.ActionQueue.Queue {
			if a.Message == "冒険を再開した" {
				seg.ResumeMessage = true
			}
		}
		g.replaySegment = seg
		r.Segments = append(r.Segments, seg)
	}
	seg := g.replaySegment
	g.replayFrameActive = true
	return func() {
		g.replayFrameActive = false
		if g.replay != r || g.replaySegment != seg {
			return
		}
		seg.Checks = append(seg.Checks, g.replayCheck())
		seg.Final = g.replayResult()
		// 毎秒および終了・中断時に保存。I/O は通常プレイだけ。
		if len(seg.Checks) == 1 || len(seg.Checks)%60 == 0 || g.lastRunResult != nil || g.suspendRequested || g.replaySavePending {
			g.writeReplay(g.lastRunResult != nil)
		}
		if g.replaySavePending {
			g.replaySavePending = false
			if g.lastRunResult == nil {
				if err := g.saveSuspendData(); err != nil {
					log.Printf("入力記録付き中断保存: %v", err)
				}
			}
		}
	}
}
func (g *Game) recordReplayKeys() {
	if !g.replayFrameActive || g.replayPlayback || g.replaySegment == nil || !g.persistent || !replayEnabled(g.settings) {
		return
	}
	var ds [keyCount]int
	for k := Key(0); k < keyCount; k++ {
		ds[k] = g.keys().PressDuration(k)
	}
	s := g.replaySegment
	if len(s.Runs) > 0 {
		p := &s.Runs[len(s.Runs)-1]
		same := true
		for k, d := range ds {
			want := p.Durations[k]
			if want > 0 {
				want += p.Frames
			}
			if d != want {
				same = false
				break
			}
		}
		if same {
			p.Frames++
			return
		}
	}
	s.Runs = append(s.Runs, ReplayKeys{Frames: 1, Durations: ds})
}
func (g *Game) writeReplay(ended bool) {
	if !g.persistent || g.replay == nil || !replayEnabled(g.settings) {
		return
	}
	b, err := json.Marshal(g.replay)
	if err == nil {
		err = storageWrite(replayFileName, b)
	}
	if err != nil {
		log.Printf("入力記録の保存: %v", err)
		return
	}
	if ended {
		var history []json.RawMessage
		old, _ := storageRead(replayHistoryFileName)
		_ = json.Unmarshal(old, &history)
		history = append(history, json.RawMessage(b))
		if len(history) > 10 {
			history = history[len(history)-10:]
		}
		b, err = json.Marshal(history)
		if err == nil {
			err = storageWrite(replayHistoryFileName, b)
		}
		if err != nil {
			log.Printf("入力記録一覧の保存: %v", err)
		}
	}
}

// replayFrameInput は記録された論理押下時間を返す。再開直後の押しっぱなしも復元する。
type replayFrameInput struct{ durations [keyCount]int }

func (v *replayFrameInput) Update() {}
func (v *replayFrameInput) PressDuration(k Key) int {
	if k < 0 || k >= keyCount {
		return 0
	}
	return v.durations[k]
}
func (v *replayFrameInput) Pressed(k Key) bool     { return v.PressDuration(k) > 0 }
func (v *replayFrameInput) JustPressed(k Key) bool { return v.PressDuration(k) == 1 }

func playReplay(r *ReplayData) (result ReplayResult, err error) {
	frame := 0
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("フレーム%d: panic: %v", frame, p)
		}
	}()
	if r.Build != replayBuild() {
		return result, fmt.Errorf("フレーム0: ビルドが異なる（記録 %s / 現在 %s）", r.Build, replayBuild())
	}
	for _, s := range r.Segments {
		g, _ := newHeadlessGame(r.Seed, r.Difficulty)
		if err = g.applySaveData(s.Initial); err != nil {
			return result, fmt.Errorf("フレーム%d: %w", frame, err)
		}
		g.replayPlayback = true
		g.fastForward = s.Fast
		g.inputFrame = s.InputFrame
		g.previousPlayerRoomID = s.PreviousRoom
		g.fadeInProgress = s.Fade
		if s.ResumeMessage {
			g.Enqueue(Action{Duration: 1, Message: "冒険を再開した", Execute: func(*Game) {}, NonBlocking: true})
		}
		v := &replayFrameInput{}
		g.input = v
		local := 0
		physicalIndex := 0
		for _, run := range s.Runs {
			for i := 0; i < run.Frames; i++ {
				for k, d := range run.Durations {
					if d > 0 {
						v.durations[k] = d + i
					} else {
						v.durations[k] = 0
					}
				}
				g.replayPhysical = ReplayPhysical{}
				if physicalIndex < len(s.Physical) && s.Physical[physicalIndex].Frame == local {
					g.replayPhysical = s.Physical[physicalIndex]
					physicalIndex++
				}
				frame++
				if err = g.StepFrame(); err != nil {
					return result, fmt.Errorf("フレーム%d: %w", frame, err)
				}
				if g.replayCheck() != s.Checks[local] {
					return g.replayResult(), fmt.Errorf("フレーム%d: 状態が一致しない", frame)
				}
				local++
			}
		}
		result = g.replayResult()
		if result != s.Final {
			return result, fmt.Errorf("フレーム%d: 最終状態が一致しない", frame)
		}
	}
	return result, nil
}

// 物理キーの割り当て画面だけは論理キー以外の入力も必要になる。
func (g *Game) replayPhysicalKey() (string, bool) {
	if g.replayPlayback {
		return g.replayPhysical.Key, g.replayPhysical.Held
	}
	key, held := capturePhysicalKey()
	if g.replayFrameActive && g.replaySegment != nil {
		s := g.replaySegment
		s.Physical = append(s.Physical, ReplayPhysical{Frame: len(s.Checks), Key: key, Held: held})
	}
	return key, held
}

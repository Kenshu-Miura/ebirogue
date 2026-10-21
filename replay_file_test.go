//go:build !test

package main

import (
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"
)

var replayFile = flag.String("replay.file", "", "入力記録JSONのパス")

func TestReplayFile(t *testing.T) {
	if *replayFile == "" {
		t.Skip("-replay.file で入力記録を指定")
	}
	b, err := os.ReadFile(*replayFile)
	if err != nil {
		t.Fatal(err)
	}
	r, err := decodeReplay(b)
	if err != nil {
		t.Fatal(err)
	}
	result, err := playReplay(r)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("階=%d HP=%d 死因=%+v ターン=%d", result.Floor, result.HP, result.Cause, result.Turns)
}
func replayTestFrames(t *testing.T, g *Game, v *VirtualInput, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		v.ReleaseAll()
		switch (i / 15) % 5 {
		case 0:
			v.Press(KeyZ)
		case 1:
			v.Press(KeyX, KeyZ)
		case 2:
			v.Press(KeyUp)
		case 3:
			v.Press(KeyRight)
		}
		if err := g.StepFrame(); err != nil {
			t.Fatal(err)
		}
		if g.lastRunResult != nil {
			break
		}
	}
}
func replayRoundTrip(t *testing.T, r *ReplayData) {
	t.Helper()
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := decodeReplay(b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = playReplay(parsed); err != nil {
		t.Fatal(err)
	}
}
func TestReplayRecordResume(t *testing.T) {
	// persistent の本物の保存経路も隔離した作業ディレクトリで確認する。
	t.Chdir(t.TempDir())
	g, v := newHeadlessGame(417, DifficultyAdvanced)
	g.persistent = true
	replayTestFrames(t, g, v, 360)
	replayRoundTrip(t, g.replay)
	b, err := json.Marshal(g.buildSaveData())
	if err != nil {
		t.Fatal(err)
	}
	save, err := decodeSaveData(b)
	if err != nil {
		t.Fatal(err)
	}
	if err = g.saveSuspendData(); err != nil {
		t.Fatal(err)
	}
	next, nv := newHeadlessGame(1, DifficultyAdvanced)
	if save.Replay == nil {
		t.Fatal("中断に記録がない")
	}
	if !next.tryResumeFromSave() {
		t.Fatal("中断の再開に失敗")
	}
	next.persistent = true
	replayTestFrames(t, next, nv, 300)
	if len(next.replay.Segments) != 2 {
		t.Fatal("再開前の記録が失われた")
	}
	replayRoundTrip(t, next.replay)
	next.writeReplay(true)
	data, err := storageRead(replayFileName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = decodeReplay(data); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 12; i++ {
		next.writeReplay(true)
	}
	data, err = storageRead(replayHistoryFileName)
	if err != nil {
		t.Fatal(err)
	}
	var history []json.RawMessage
	if err = json.Unmarshal(data, &history); err != nil || len(history) != 10 {
		t.Fatalf("一覧: %d %v", len(history), err)
	}
}
func TestReplayNormalTiming(t *testing.T) {
	t.Chdir(t.TempDir())
	g, v := newHeadlessGame(91, DifficultyBeginner)
	g.headless = false
	g.fastForward = false
	g.fadeInProgress = 0
	g.persistent = true
	replayTestFrames(t, g, v, 600)
	replayRoundTrip(t, g.replay)
	r := g.replay
	r.Segments[0].Checks[10] = "broken"
	if _, err := playReplay(r); err == nil || !strings.Contains(err.Error(), "フレーム11") {
		t.Fatalf("ずれの検出: %v", err)
	}
}
func TestReplayRejectsBroken(t *testing.T) {
	for _, b := range []string{"{", "null", `{"Version":-1}`, `{"Version":1,"SaveVersion":3,"Difficulty":0,"Segments":[null]}`} {
		if _, err := decodeReplay([]byte(b)); err == nil {
			t.Fatalf("破損記録を受理: %s", b)
		}
	}
	t.Chdir(t.TempDir())
	g, v := newHeadlessGame(91, DifficultyBeginner)
	g.persistent = true
	replayTestFrames(t, g, v, 2)
	g.replay.Build = "old"
	if _, err := playReplay(g.replay); err == nil {
		t.Fatal("旧ビルドを受理")
	}
}
func TestReplayDisabledAndNonPersistent(t *testing.T) {
	t.Chdir(t.TempDir())
	g, v := newHeadlessGame(91, DifficultyBeginner)
	replayTestFrames(t, g, v, 20)
	if g.replay != nil {
		t.Fatal("BOTで記録")
	}
	off := false
	g.persistent = true
	g.settings.SaveInputReplay = &off
	replayTestFrames(t, g, v, 20)
	if g.replay != nil {
		t.Fatal("OFFで記録")
	}
	if _, err := storageRead(replayFileName); !storageIsNotExist(err) {
		t.Fatal("保存された")
	}
	s, err := decodeSettings([]byte(`{}`))
	if err != nil || !replayEnabled(s) {
		t.Fatal("旧設定の初期値")
	}
}

func TestReplayNaturalDeath(t *testing.T) {
	t.Chdir(t.TempDir())
	g, v := newHeadlessGame(12, DifficultyAdvanced)
	g.persistent = true
	for i := 0; i < 12000 && g.lastRunResult == nil; i++ {
		v.ReleaseAll()
		if i%2 == 0 {
			v.Press(KeyX, KeyZ)
		}
		if err := g.StepFrame(); err != nil {
			t.Fatal(err)
		}
	}
	if g.lastRunResult == nil {
		t.Fatal("冒険が終わらない")
	}
	b, err := storageRead(replayFileName)
	if err != nil {
		t.Fatal(err)
	}
	r, err := decodeReplay(b)
	if err != nil {
		t.Fatal(err)
	}
	replayRoundTrip(t, r)
	n := len(g.replay.Segments[0].Runs)
	for i := 0; i < 30; i++ {
		_ = g.StepFrame()
	}
	if len(g.replay.Segments[0].Runs) != n {
		t.Fatal("終了後に入力を記録")
	}
}

func replayTap(t *testing.T, g *Game, v *VirtualInput, key Key) {
	t.Helper()
	v.ReleaseAll()
	if err := g.StepFrame(); err != nil {
		t.Fatal(err)
	}
	v.Press(key)
	if err := g.StepFrame(); err != nil {
		t.Fatal(err)
	}
}
func TestReplayMenusAndSuspend(t *testing.T) {
	t.Chdir(t.TempDir())
	g, v := newHeadlessGame(417, DifficultyAdvanced)
	g.persistent = true
	if err := g.RunUntilInputReady(100); err != nil {
		t.Fatal(err)
	}
	// 設定・キー設定・記録を経由して、中断確認を本物のメニューから確定する。
	for _, k := range []Key{KeyC, KeyDown, KeyZ, KeyDown, KeyDown, KeyZ, KeyX, KeyC, KeyC, KeyUp, KeyZ, KeyC, KeyC, KeyDown, KeyRight, KeyZ, KeyZ} {
		replayTap(t, g, v, k)
	}
	if _, err := storageRead(saveFileName); err != nil {
		t.Fatal("中断されていない", err)
	}
	data, err := storageRead(saveFileName)
	if err != nil {
		t.Fatal(err)
	}
	save, err := decodeSaveData(data)
	if err != nil {
		t.Fatal(err)
	}
	if save.Replay == nil {
		t.Fatal("中断に記録がない")
	}
	replayRoundTrip(t, save.Replay)
	next, nv := newHeadlessGame(1, DifficultyAdvanced)
	if !next.tryResumeFromSave() {
		t.Fatal("再開失敗")
	}
	next.persistent = true
	replayTestFrames(t, next, nv, 320)
	replayRoundTrip(t, next.replay)
}

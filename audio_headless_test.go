//go:build !test

package main

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

func TestAudioHeadlessDoesNotCreateContextOrChangeGame(t *testing.T) {
	g, _ := newHeadlessGame(123, DifficultyBeginner)
	before := g.buildSaveData()
	context := soundContext
	// persistent が誤って有効でも headless を優先する。
	g.persistent = true
	g.updateAudio()
	g.playSE("hit")
	g.playBGM("boss")
	g.stopBGM()
	g.applyAudioVolumes()
	if soundContext != context || g.audioState != nil {
		t.Fatal("headless created audio")
	}
	if !reflect.DeepEqual(before, g.buildSaveData()) {
		t.Fatal("audio changed game state")
	}
	g.headless = false
	g.persistent = false
	g.updateAudio()
	g.playSE("hit")
	g.playBGM("dungeon")
	g.stopBGM()
	if soundContext != context || g.audioState != nil {
		t.Fatal("nonpersistent game created audio")
	}
}

func TestEmbeddedSoundsDecode(t *testing.T) {
	for _, id := range []string{"hit", "miss", "hurt", "defeat", "level", "pickup", "stairs", "trap", "use", "death", "clear", "confirm", "cancel", "dungeon", "boss", "victory"} {
		data, err := soundFS.ReadFile("sound/" + id + ".wav")
		if err != nil {
			t.Fatal(err)
		}
		stream, err := wav.DecodeWithSampleRate(44100, bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		pcm, err := io.ReadAll(stream)
		if err != nil {
			t.Fatal(err)
		}
		if len(pcm) == 0 || len(pcm)%4 != 0 {
			t.Fatalf("invalid PCM: %s", id)
		}
		if id == "dungeon" || id == "boss" || id == "victory" {
			if len(pcm) != 44100*4*8 {
				t.Fatalf("BGM must be eight bars: %s", id)
			}
		}
	}
}

func TestAudioVolumeMenuKeepsTurnsAndClamps(t *testing.T) {
	g, _ := newHeadlessGame(123, DifficultyBeginner)
	g.headless = false // 通常UIの5項目を検証する。persistentはfalseのまま。
	g.settings = defaultSettings()
	g.showSettings = true
	v := NewVirtualInput()
	g.input = v
	turn := g.turnCount
	for _, tc := range []struct {
		index         int
		key           Key
		initial, want int
	}{
		{3, KeyLeft, 6, 5}, {3, KeyLeft, 0, 0}, {3, KeyRight, 10, 10},
		{4, KeyRight, 6, 7}, {4, KeyLeft, 0, 0}, {4, KeyRight, 10, 10},
	} {
		g.settingsSelectedIndex = tc.index
		if tc.index == 3 {
			g.settings.BGMVolume = tc.initial
		} else {
			g.settings.SEVolume = tc.initial
		}
		v.Press(tc.key)
		v.Update()
		g.handleSettingsInput()
		got := g.settings.BGMVolume
		if tc.index == 4 {
			got = g.settings.SEVolume
		}
		if got != tc.want {
			t.Fatalf("index=%d: got %d want %d", tc.index, got, tc.want)
		}
		v.Release(tc.key)
		v.Update()
	}
	if g.turnCount != turn || len(g.ActionQueue.Queue) != 0 {
		t.Fatal("settings changed turn or queue")
	}
}

func TestSoundFirstActionPreservesExecution(t *testing.T) {
	g := &Game{}
	calls := 0
	g.Enqueue(Action{Duration: .5, Message: "test", Execute: func(*Game) { calls++ }})
	g.soundFirstAction(0, "hit")
	g.soundFirstAction(1, "hit") // 効果がキューを追加しない経路。
	g.processAction(g.ActionQueue.Queue[0])
	if calls != 1 || len(g.ActionQueue.Queue) != 1 || g.ActionDurationCounter != .5 {
		t.Fatal("audio changed action")
	}
}

// ランダム入力BOTの設定メニュー巡回は音声導入前と同じ3項目。
func TestHeadlessSettingsKeepsLegacyNavigation(t *testing.T) {
	g, _ := newHeadlessGame(123, DifficultyBeginner)
	g.showSettings = true
	g.settingsSelectedIndex = 2
	v := NewVirtualInput()
	g.input = v
	v.Press(KeyDown)
	v.Update()
	g.handleSettingsInput()
	if g.settingsSelectedIndex != 0 {
		t.Fatal("headless settings navigation changed")
	}
}

func TestNormalSettingsIncludesVolumeNavigation(t *testing.T) {
	g, _ := newHeadlessGame(123, DifficultyBeginner)
	g.headless = false
	g.showSettings = true
	g.settingsSelectedIndex = 2
	v := NewVirtualInput()
	g.input = v
	for _, want := range []int{3, 4, 0} {
		v.Press(KeyDown)
		v.Update()
		g.handleSettingsInput()
		if g.settingsSelectedIndex != want {
			t.Fatalf("index=%d want %d", g.settingsSelectedIndex, want)
		}
		v.Release(KeyDown)
		v.Update()
	}
}

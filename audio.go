//go:build !test

package main

import (
	"bytes"
	"embed"
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

//go:embed sound/*.wav
var soundFS embed.FS

// Context はプロセス全体で1つだけ。BOT・テストからは生成しない。
var soundContext *audio.Context

type gameAudio struct {
	bgm     *audio.Player
	bgmID   string
	effects []*audio.Player
	pcm     map[string][]byte
}

func (g *Game) audioEnabled() bool { return g.persistent && !g.headless }

// Ebitengine v2.8 の Context には公開 Resume がない。音声ドライバは
// ブラウザのユーザー操作で自動再開される。こちらも最初の論理キー入力まで
// Context の生成・再生を遅延し、ユーザー操作前には音を要求しない。
func (g *Game) updateAudio() {
	if !g.audioEnabled() {
		return
	}
	if g.audioState == nil {
		pressed := false
		for key := Key(0); key < keyCount; key++ {
			if g.keys().JustPressed(key) {
				pressed = true
				break
			}
		}
		if !pressed {
			return
		}
		if soundContext == nil {
			soundContext = audio.NewContext(44100)
		}
		g.audioState = &gameAudio{pcm: make(map[string][]byte)}
	}
	active := g.audioState.effects[:0]
	for _, p := range g.audioState.effects {
		if p.IsPlaying() {
			active = append(active, p)
		} else {
			p.Close()
		}
	}
	g.audioState.effects = active
	if g.playerDead || g.showDifficultySelect || g.suspendRequested || g.fadingOut {
		g.stopBGM()
	} else if g.cleared {
		g.playBGM("victory")
	} else if isBossFloor(g.Floor) && !g.bossDefeated {
		g.playBGM("boss")
	} else {
		g.playBGM("dungeon")
	}
	// メニュー操作はターンもキューも増やさず、同じフレームで鳴らす。
	if g.showMenu || g.showInventory || g.showSettings || g.showKeyConfig || g.showStairsPrompt || g.showDifficultySelect || g.showClearScreen || g.showSuspendPrompt {
		if g.keys().JustPressed(KeyZ) || g.keys().JustPressed(KeyEnter) {
			g.executeAudioCue("confirm")
		} else if g.keys().JustPressed(KeyX) || g.keys().JustPressed(KeyC) {
			g.executeAudioCue("cancel")
		}
	}
}

// UI とフロア演出の即時 Action。ActionQueue やターン進行には触れない。
func (g *Game) executeAudioCue(id string) { Action{Execute: func(g *Game) { g.playSE(id) }}.Execute(g) }

func (g *Game) soundPCM(id string) []byte {
	if data, ok := g.audioState.pcm[id]; ok {
		return data
	}
	data, err := soundFS.ReadFile("sound/" + id + ".wav")
	if err != nil {
		return nil
	}
	stream, err := wav.DecodeWithSampleRate(44100, bytes.NewReader(data))
	if err != nil {
		log.Printf("音源の読み込み失敗: %v", err)
		return nil
	}
	// WAV は生成時から 44100Hz・16bit・ステレオ。ヘッダーではなくデコーダーを通す。
	var decoded bytes.Buffer
	if _, err = decoded.ReadFrom(stream); err != nil {
		log.Printf("音源のデコード失敗: %v", err)
		return nil
	}
	g.audioState.pcm[id] = decoded.Bytes()
	return decoded.Bytes()
}
func (g *Game) playSE(id string) {
	if !g.audioEnabled() || g.audioState == nil || g.settings.SEVolume == 0 {
		return
	}
	data := g.soundPCM(id)
	if len(data) == 0 {
		return
	}
	// 同時発音を制限し、長時間プレイでも Player を無制限に増やさない。
	if len(g.audioState.effects) >= 16 {
		g.audioState.effects[0].Close()
		g.audioState.effects = g.audioState.effects[1:]
	}
	p, err := soundContext.NewPlayer(bytes.NewReader(data))
	if err != nil {
		log.Printf("効果音の再生失敗: %v", err)
		return
	}
	p.SetVolume(float64(clampVolume(g.settings.SEVolume)) / 10)
	g.audioState.effects = append(g.audioState.effects, p)
	p.Play()
}
func (g *Game) playBGM(id string) {
	if !g.audioEnabled() || g.audioState == nil {
		return
	}
	if g.audioState.bgmID == id {
		return
	}
	g.stopBGM()
	data := g.soundPCM(id)
	if len(data) == 0 {
		return
	}
	p, err := soundContext.NewPlayer(audio.NewInfiniteLoop(bytes.NewReader(data), int64(len(data))))
	if err != nil {
		log.Printf("BGM の再生失敗: %v", err)
		return
	}
	g.audioState.bgm = p
	g.audioState.bgmID = id
	g.applyAudioVolumes()
	p.Play()
}
func (g *Game) stopBGM() {
	if !g.audioEnabled() || g.audioState == nil {
		return
	}
	if g.audioState.bgm != nil {
		g.audioState.bgm.Close()
		g.audioState.bgm = nil
	}
	g.audioState.bgmID = ""
}
func (g *Game) applyAudioVolumes() {
	if !g.audioEnabled() || g.audioState == nil {
		return
	}
	if p := g.audioState.bgm; p != nil {
		p.SetVolume(float64(clampVolume(g.settings.BGMVolume)) / 10)
	}
	for _, p := range g.audioState.effects {
		p.SetVolume(float64(clampVolume(g.settings.SEVolume)) / 10)
	}
}

// 既存キューに追加された最初の Action にだけ音を添える。待ち時間や
// Action 数を変えず、既存の演出・効果と同じタイミングで鳴らす。
func (g *Game) soundFirstAction(start int, id string) {
	if start >= len(g.ActionQueue.Queue) {
		return
	}
	action := &g.ActionQueue.Queue[start]
	execute := action.Execute
	action.Execute = func(g *Game) {
		g.playSE(id)
		if execute != nil {
			execute(g)
		}
	}
}

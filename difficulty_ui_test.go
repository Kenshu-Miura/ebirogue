//go:build !test

package main

import (
	"encoding/json"
	"testing"
)

func TestDifficultyUISelectionAndClear(t *testing.T) {
	g, in := newHeadlessGame(123, DifficultyAdvanced)
	if g.showDifficultySelect || g.difficulty != DifficultyAdvanced {
		t.Fatal("headless should start immediately")
	}
	g.openDifficultySelect()
	b := newBot(g, in, personaExpert, 123)
	if err := b.tap(KeyDown); err != nil {
		t.Fatal(err)
	}
	if g.difficultyCursor != 0 {
		t.Fatal("cursor should wrap to beginner")
	}
	if err := b.tap(KeyZ); err != nil {
		t.Fatal(err)
	}
	if g.showDifficultySelect || g.difficulty != DifficultyBeginner || g.turnCount != 0 {
		t.Fatal("selection must start without consuming a turn")
	}
	data, err := json.Marshal(g.settings)
	if err != nil {
		t.Fatal(err)
	}
	settings, err := decodeSettings(data)
	if err != nil || settings.LastDifficulty != DifficultyBeginner {
		t.Fatal("difficulty preference not preserved")
	}
	g.Floor = 10
	g.clearRun()
	// 通常プレイと同じ終了処理を画像なしで検証する。
	g.headless = false
	if err := g.RunUntilInputReady(botMaxFrames); err != nil {
		t.Fatal(err)
	}
	if !g.showClearScreen || g.lastRunResult == nil || !g.lastRunResult.Cleared {
		t.Fatal("clear screen missing")
	}
	turns := g.turnCount
	if err := b.tap(KeyZ); err != nil {
		t.Fatal(err)
	}
	if !g.showDifficultySelect || g.turnCount != turns || g.difficultyCursor != 0 {
		t.Fatal("clear should return to remembered selection")
	}
	if err := b.tap(KeyZ); err != nil {
		t.Fatal(err)
	}
	if g.cleared || g.lastRunResult != nil {
		t.Fatal("new run must reset result")
	}
}

func TestDifficultyUIResume(t *testing.T) {
	source, _ := newHeadlessGame(456, DifficultyIntermediate)
	save := source.buildSaveData()
	target, _ := newHeadlessGame(999, DifficultyAdvanced)
	if err := target.applySaveData(save); err != nil {
		t.Fatal(err)
	}
	if target.showDifficultySelect || target.difficulty != DifficultyIntermediate {
		t.Fatal("resume must preserve difficulty and skip selection")
	}
}

func TestDifficultyUIDeathReset(t *testing.T) {
	g, _ := newHeadlessGame(1, DifficultyIntermediate)
	g.headless = false
	g.settings.LastDifficulty = DifficultyIntermediate
	g.playerDead = true
	g.resetGame()
	if !g.showDifficultySelect || g.difficultyCursor != 1 {
		t.Fatal("death reset should select remembered difficulty")
	}
}

func TestDifficultyUIStorageAndHistory(t *testing.T) {
	t.Chdir(t.TempDir())
	g, in := newHeadlessGame(123, DifficultyAdvanced)
	g.persistent = true
	g.openDifficultySelect()
	b := newBot(g, in, personaExpert, 123)
	if err := b.closeWindows(); err != nil {
		t.Fatal(err)
	}
	if g.showDifficultySelect || loadSettings().LastDifficulty != DifficultyAdvanced {
		t.Fatal("BOT selection and saved settings")
	}
	data, err := json.Marshal(g.buildSaveData())
	if err != nil {
		t.Fatal(err)
	}
	if err := storageWrite(saveFileName, data); err != nil {
		t.Fatal(err)
	}
	resumed, _ := newHeadlessGame(999, DifficultyBeginner)
	if !resumed.tryResumeFromSave() || resumed.showDifficultySelect || resumed.difficulty != DifficultyAdvanced {
		t.Fatal("suspend resume")
	}
	for i := 1; i <= 8; i++ {
		g.saveRunResult(RunResult{Seed: int64(i), Floor: i, Difficulty: DifficultyBeginner})
	}
	g.openRunHistory()
	if len(g.runHistory) != 8 || g.runHistory[0].Seed != 8 {
		t.Fatal("saved history order")
	}
	turns := g.turnCount
	for i := 0; i < 10; i++ {
		if err := b.tap(KeyDown); err != nil {
			t.Fatal(err)
		}
	}
	if g.runHistoryScroll != 3 || g.turnCount != turns {
		t.Fatal("history scroll must clamp without consuming turns")
	}
	if err := b.tap(KeyX); err != nil {
		t.Fatal(err)
	}
	if g.showRunHistory || !g.showMenu {
		t.Fatal("history should return to menu")
	}
}

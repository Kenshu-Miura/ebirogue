//go:build !test

package main

import (
	"os"
	"reflect"
	"testing"
)

func TestCodexPersistenceAndCorruption(t *testing.T) {
	t.Chdir(t.TempDir())
	g, _ := newHeadlessGame(1, DifficultyBeginner)
	g.persistent = true
	g.observeCodexItem(rollItemFromTemplate(4, 0, 0, func(int) int { return 0 }))
	if _, err := os.Stat(codexFileName); !os.IsNotExist(err) {
		t.Fatal("wrote before flush")
	}
	g.saveCodex()
	data, err := storageRead(codexFileName)
	if err != nil {
		t.Fatal(err)
	}
	c, err := decodeCodex(data)
	if err != nil || !c.Items[4].Seen {
		t.Fatal(c, err)
	}
	if err := storageWrite(codexFileName, []byte("broken")); err != nil {
		t.Fatal(err)
	}
	h, _ := newHeadlessGame(2, DifficultyBeginner)
	h.persistent = true
	h.observeCodexItem(rollItemFromTemplate(4, 0, 0, func(int) int { return 0 }))
	h.saveCodex()
	data, _ = storageRead(codexFileName)
	if string(data) != "broken" || h.codexError == "" {
		t.Fatal("corruption overwritten")
	}
}
func TestCodexNonpersistentAndNavigation(t *testing.T) {
	t.Chdir(t.TempDir())
	g, v := newHeadlessGame(3, DifficultyBeginner)
	g.observeCodexInventory()
	g.saveCodex()
	if g.codexLoaded {
		t.Fatal("BOT touched codex")
	}
	g.openCodex()
	turn := g.turnCount
	press := func(k Key) { v.ReleaseAll(); v.Update(); v.Press(k); v.Update(); g.handleCodexInput() }
	press(KeyDown)
	if g.codexCursor != 1 {
		t.Fatal(g.codexCursor)
	}
	press(KeyUp)
	press(KeyUp)
	if n := len(g.codexEntries()); g.codexCursor != n-1 || g.codexScroll != n-codexListRows {
		t.Fatal("wrap to bottom", g.codexCursor, g.codexScroll)
	}
	press(KeyRight)
	if g.codexTab != 1 || g.codexCursor != 0 || g.codexScroll != 0 {
		t.Fatal("tab")
	}
	press(KeyLeft)
	press(KeyLeft)
	if g.codexTab != 2 {
		t.Fatal("tab wrap", g.codexTab)
	}
	// 長押しは端で止まり、回り込まない
	g.setCodexTab(0)
	v.ReleaseAll()
	v.Update()
	v.Press(KeyDown)
	for i := 0; i < 2000; i++ {
		v.Update()
		g.handleCodexInput()
	}
	if n := len(g.codexEntries()); g.codexCursor != n-1 {
		t.Fatal("hold should stop at the end", g.codexCursor, n)
	}
	press(KeyX)
	if g.showCodex || !g.showMenu {
		t.Fatal("back")
	}
	g.openCodex()
	press(KeyC)
	if g.showCodex || g.showMenu || g.turnCount != turn {
		t.Fatal("close or turn")
	}
	if _, err := os.Stat(codexFileName); !os.IsNotExist(err) {
		t.Fatal("BOT wrote codex")
	}
}
func TestCodexTemplatesAndMimic(t *testing.T) {
	g, _ := newHeadlessGame(4, DifficultyBeginner)
	g.persistent = true
	g.codexLoaded = true
	g.codex = newCodexData()
	before := len(codexIDs(1))
	MonsterDefinitions[999] = MonsterDefinition{ID: 999, Name: "将来の敵"}
	defer delete(MonsterDefinitions, 999)
	if len(codexIDs(1)) != before+1 {
		t.Fatal("new enemy omitted")
	}
	itemTemplates[999] = ItemTemplate{ID: 999}
	defer delete(itemTemplates, 999)
	ids := codexIDs(0)
	if ids[len(ids)-1] != 999 {
		t.Fatal("new item omitted")
	}
	mapTrapTemplates[999] = mapTrapTemplate{ID: 999}
	defer delete(mapTrapTemplates, 999)
	ids = codexIDs(2)
	if ids[len(ids)-1] != 999 {
		t.Fatal("new trap omitted")
	}
	e := Enemy{ID: 117, Disguise: EnemyDisguiseEnemy, DisguiseID: 0}
	g.state.Enemies = []Enemy{e}
	g.observeCodexEnemy(e)
	if g.codex.Enemies[117].Seen || !g.codex.Enemies[0].Seen {
		t.Fatal("mimic spoiled")
	}
	g.revealEnemy(0)
	if !g.codex.Enemies[117].Seen {
		t.Fatal("identity missing")
	}
	snapshot, _ := g.rng.MarshalState()
	_ = rollItemFromTemplate(4, 0, 0, func(int) int { return 0 })
	after, _ := g.rng.MarshalState()
	if !reflect.DeepEqual(snapshot, after) {
		t.Fatal("rng changed")
	}
	g.onEnemyDefeated(g.state.Enemies[0])
	if g.codex.Enemies[117].Defeated != 1 {
		t.Fatal("defeat")
	}
}

func TestCodexRunAndTrapCounters(t *testing.T) {
	t.Chdir(t.TempDir())
	g, _ := newHeadlessGame(5, DifficultyBeginner)
	g.persistent = true
	g.state.MapTraps = []MapTrap{createMapTrapByID(0, g.state.Player.X, g.state.Player.Y)}
	g.stepOnTrap(0)
	if !g.codex.Traps[0].Stepped || !g.codex.Traps[0].Seen {
		t.Fatal("trap record")
	}
	g.onEnemyDefeated(Enemy{ID: 0})
	g.lastDamageSource = enemyAttackSource(0, "敵")
	g.finishRun(false)
	g.finishRun(false)
	h, _ := newHeadlessGame(6, DifficultyBeginner)
	h.persistent = true
	h.ensureCodex()
	if h.codex.Enemies[0].Defeated != 1 || h.codex.Enemies[0].Deaths != 1 || !h.codex.Traps[0].Stepped {
		t.Fatal(h.codex)
	}
}
func TestCodexObservationBoundaries(t *testing.T) {
	g, _ := newHeadlessGame(7, DifficultyBeginner)
	g.persistent = true
	g.codexLoaded = true
	g.codex = newCodexData()
	g.observeCodexEnemy(Enemy{ID: 35, Disguise: EnemyDisguiseItem})
	if g.codex.Enemies[35].Seen {
		t.Fatal("hidden mimic revealed")
	}
	g.setPlayerStatus(StatusBlind, 3)
	g.observeCodexEnemy(Enemy{ID: 0})
	if g.codex.Enemies[0].Seen {
		t.Fatal("blind observation")
	}
	if codexOnScreen(30, 30, 0, 0, 640, 480) || !codexOnScreen(0, 0, 0, 0, 640, 480) {
		t.Fatal("viewport")
	}
	item := rollItemFromTemplate(4, 0, 0, func(int) int { return 0 })
	v := item.(Identifiable)
	v.SetIdentified(false)
	g.observeCodexItem(item)
	if g.codex.Items[4].Identified {
		t.Fatal("unidentified item")
	}
	v.SetIdentified(true)
	g.observeCodexItem(item)
	v.SetIdentified(false)
	g.observeCodexItem(item)
	if !g.codex.Items[4].Identified {
		t.Fatal("identification lost")
	}
}

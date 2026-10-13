//go:build !test

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// BOT によるテストプレイ（シミュレーション）。
//
//	go test -run TestSimSmoke -count=1 .                       # 少数のシード（マージ判定用）
//	go test -run TestSimBatch -count=1 -timeout 0 . -args -sim.runs=1000 -sim.difficulty=all -sim.out=reports
//	go test -run TestSimReplay -count=1 . -args -sim.seed=<シード> -sim.persona=<名前> -sim.difficulty=<難易度>
var (
	simRuns       = flag.Int("sim.runs", 0, "TestSimBatch の冒険数（0 なら TestSimBatch を実行しない）")
	simDifficulty = flag.String("sim.difficulty", "all", "難易度: all / beginner / intermediate / advanced")
	simPersona    = flag.String("sim.persona", "all", "BOT の性格: all / "+personaNames())
	simTurns      = flag.Int("sim.turns", 20000, "1回の冒険の最大ターン数")
	simOut        = flag.String("sim.out", "reports", "レポートの出力先ディレクトリ")
	simBaseline   = flag.String("sim.baseline", "reports/baseline.json", "比較するベースライン（無ければ比較しない）")
	simParallel   = flag.Int("sim.parallel", runtime.NumCPU(), "並列に実行する冒険の数")
	simSeed       = flag.Int64("sim.seed", -1, "TestSimReplay で再現するシード")
	simSeedBase   = flag.Int64("sim.seedbase", 1, "TestSimBatch の最初のシード")
	simWriteBase  = flag.Bool("sim.writebaseline", false, "TestSimBatch の集計を -sim.baseline へ書き出す")
)

func personaNames() string {
	names := make([]string, len(allPersonas))
	for i, p := range allPersonas {
		names[i] = string(p)
	}
	return strings.Join(names, " / ")
}

var difficultyFlagNames = map[string]Difficulty{
	"beginner":     DifficultyBeginner,
	"intermediate": DifficultyIntermediate,
	"advanced":     DifficultyAdvanced,
}

func parseDifficulties(name string) ([]Difficulty, error) {
	if name == "all" {
		return allDifficulties, nil
	}
	if d, ok := difficultyFlagNames[name]; ok {
		return []Difficulty{d}, nil
	}
	return nil, fmt.Errorf("不明な難易度 %q", name)
}

func parsePersonas(name string) ([]persona, error) {
	if name == "all" {
		return allPersonas, nil
	}
	if p, ok := parsePersona(name); ok {
		return []persona{p}, nil
	}
	return nil, fmt.Errorf("不明な性格 %q", name)
}

// simCase は1回の冒険の条件。
type simCase struct {
	Seed       int64
	Persona    persona
	Difficulty Difficulty
	MaxTurns   int
}

// simOutcome は1回の冒険の結果。
type simOutcome struct {
	Seed       int64
	Persona    persona
	Difficulty string
	Ended      string // "cleared" / "died" / "turn_limit" / "bot_stuck" / "softlock" / "panic"
	Floor      int
	Turns      int
	Level      int
	DeathCause string         `json:",omitempty"`
	Mistake    string         `json:",omitempty"` // 判断ミス死の種類（mistakeKind 参照）
	Result     *RunResult     `json:",omitempty"`
	Violations []simViolation `json:",omitempty"`
}

const maxViolationsPerRun = 20

// runSimCase は BOT で1回の冒険を遊び、結果と不変条件違反を返す。panic しても recover して記録する。
func runSimCase(c simCase) (out simOutcome) {
	out = simOutcome{Seed: c.Seed, Persona: c.Persona, Difficulty: c.Difficulty.String()}
	g, in := newHeadlessGame(c.Seed, c.Difficulty)
	b := newBot(g, in, c.Persona, c.Seed)
	turns, lastTurn := 0, 0
	seen := map[string]bool{}
	record := func(vs []simViolation) {
		for _, v := range vs {
			key := v.Kind + "|" + v.Detail
			if seen[key] || len(out.Violations) >= maxViolationsPerRun {
				continue
			}
			seen[key] = true
			out.Violations = append(out.Violations, v)
		}
	}
	defer func() {
		if r := recover(); r != nil {
			out.Ended = "panic"
			record([]simViolation{newViolation(b.g, "panic", "%v\n%s", r, debug.Stack())})
		}
		g := b.g
		out.Floor, out.Turns, out.Level = g.Floor, turns, g.state.Player.Level
		if result := g.LastRunResult(); result != nil {
			out.Result = result
			if !result.Cleared {
				out.DeathCause = result.DeathCause.Label()
				out.Mistake = mistakeKind(result)
			}
		}
	}()

	if err := g.RunUntilInputReady(botMaxFrames); err != nil {
		record([]simViolation{newViolation(g, "softlock", "%v", err)})
		out.Ended = "softlock"
		return out
	}
	for {
		g := b.g
		if g.turnCount >= lastTurn {
			turns += g.turnCount - lastTurn
		}
		lastTurn = g.turnCount // 「再挑戦」でターン数が戻っても累計で数える
		if g.RunEnded() {
			out.Ended = "died"
			if g.cleared {
				out.Ended = "cleared"
			}
			return out
		}
		if turns >= c.MaxTurns {
			out.Ended = "turn_limit"
			return out
		}
		record(checkInvariants(g))
		// ランダムなターンで中断セーブ→再開を挟む
		if !b.windowOpen() && b.rng.IntN(150) == 0 {
			restored, vs := saveRoundTrip(g)
			record(vs)
			b.g = restored
		}
		if err := b.step(); err != nil {
			switch {
			case errors.Is(err, errBotStuck):
				out.Ended = "bot_stuck"
			case errors.Is(err, ErrSoftLock):
				out.Ended = "softlock"
				record([]simViolation{newViolation(b.g, "softlock", "%v", err)})
			default:
				out.Ended = "error"
				record([]simViolation{newViolation(b.g, "error", "%v", err)})
			}
			return out
		}
	}
}

// runSimCases は冒険を goroutine で並列に実行する（各冒険は独立した Game と乱数源を持つ）。
func runSimCases(cases []simCase, parallel int) []simOutcome {
	outcomes := make([]simOutcome, len(cases))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < max(1, parallel); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				outcomes[i] = runSimCase(cases[i])
			}
		}()
	}
	for i := range cases {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return outcomes
}

// buildSimCases は難易度×性格×シードの組を作る。性格どうしは同じシード（同じダンジョン）で比べ、
// 難易度ごとには別のシードを使う（ボス階に着くまでは難易度で結果が変わらないため）。
func buildSimCases(seedBase int64, seeds int, personas []persona, difficulties []Difficulty, maxTurns int) []simCase {
	var cases []simCase
	for di, d := range difficulties {
		for _, p := range personas {
			for i := 0; i < seeds; i++ {
				seed := seedBase + int64(di*seeds+i)
				cases = append(cases, simCase{Seed: seed, Persona: p, Difficulty: d, MaxTurns: maxTurns})
			}
		}
	}
	return cases
}

func failOnViolations(t *testing.T, outcomes []simOutcome) {
	t.Helper()
	failures := 0
	for _, o := range outcomes {
		for _, v := range o.Violations {
			failures++
			if failures <= 30 {
				t.Errorf("違反 [%s] seed=%d persona=%s difficulty=%s floor=%d turn=%d: %s",
					v.Kind, o.Seed, o.Persona, o.Difficulty, v.Floor, v.Turn, firstLine(v.Detail))
			}
		}
	}
	if failures > 0 {
		t.Errorf("不変条件違反 %d 件（再現: go test -run TestSimReplay -count=1 . -args -sim.seed=<seed> -sim.persona=<persona> -sim.difficulty=<difficulty>）", failures)
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// TestSimSmoke は少数のシードで各性格を遊ばせ、不変条件違反・ソフトロック・panic が無いことを確認する。
func TestSimSmoke(t *testing.T) {
	seeds, turns := 20, 500
	if testing.Short() {
		seeds = 4
	}
	cases := buildSimCases(1, seeds, allPersonas, []Difficulty{DifficultyAdvanced}, turns)
	start := time.Now()
	outcomes := runSimCases(cases, runtime.NumCPU())
	summary := summarizeOutcomes(outcomes)
	t.Logf("%d 回（%v）: %s", len(outcomes), time.Since(start).Round(time.Millisecond), summary.oneLine())
	failOnViolations(t, outcomes)
}

// TestSimBatch は -sim.runs を指定したときだけ動く大規模版。レポートを -sim.out へ書き出す。
func TestSimBatch(t *testing.T) {
	if *simRuns <= 0 {
		t.Skip("-sim.runs を指定したときだけ実行する")
	}
	difficulties, err := parseDifficulties(*simDifficulty)
	if err != nil {
		t.Fatal(err)
	}
	personas, err := parsePersonas(*simPersona)
	if err != nil {
		t.Fatal(err)
	}
	// 冒険数を難易度×性格の組に均等に割り振る
	perGroup := max(1, (*simRuns+len(difficulties)*len(personas)-1)/(len(difficulties)*len(personas)))
	cases := buildSimCases(*simSeedBase, perGroup, personas, difficulties, *simTurns)
	start := time.Now()
	outcomes := runSimCases(cases, *simParallel)
	elapsed := time.Since(start)

	summary := summarizeOutcomes(outcomes)
	var baseline *simSummary
	if data, err := os.ReadFile(*simBaseline); err == nil {
		var b simBaselineFile
		if err := json.Unmarshal(data, &b); err == nil {
			baseline = &b.Summary
		} else {
			t.Logf("ベースラインを読めない: %v", err)
		}
	}
	jsonPath, mdPath, err := writeSimReport(*simOut, outcomes, summary, baseline, elapsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d 回（%v）: %s", len(outcomes), elapsed.Round(time.Second), summary.oneLine())
	t.Logf("レポート: %s, %s", jsonPath, mdPath)
	if *simWriteBase {
		data, err := json.MarshalIndent(simBaselineFile{GeneratedAt: time.Now().Format(time.RFC3339), Summary: summary}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(*simBaseline, data, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("ベースラインを更新: %s", *simBaseline)
	}
	failOnViolations(t, outcomes)
}

// TestSimReplay は1件の冒険を再現する（-sim.seed 必須）。
func TestSimReplay(t *testing.T) {
	if *simSeed < 0 {
		t.Skip("-sim.seed を指定したときだけ実行する")
	}
	difficulties, err := parseDifficulties(*simDifficulty)
	if err != nil {
		t.Fatal(err)
	}
	personas, err := parsePersonas(*simPersona)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range difficulties {
		for _, p := range personas {
			o := runSimCase(simCase{Seed: *simSeed, Persona: p, Difficulty: d, MaxTurns: *simTurns})
			t.Logf("seed=%d persona=%s difficulty=%s: %s（%d階・%dターン・Lv%d）%s", o.Seed, o.Persona, o.Difficulty, o.Ended, o.Floor, o.Turns, o.Level, o.DeathCause)
			for _, v := range o.Violations {
				t.Errorf("違反 [%s] floor=%d turn=%d: %s", v.Kind, v.Floor, v.Turn, v.Detail)
			}
		}
	}
}

// TestSimDeterministic は同じシード・性格なら同じ結果になることを確認する（再現性の前提）。
func TestSimDeterministic(t *testing.T) {
	for _, p := range allPersonas {
		c := simCase{Seed: 42, Persona: p, Difficulty: DifficultyAdvanced, MaxTurns: 300}
		a, b := runSimCase(c), runSimCase(c)
		aj, _ := json.Marshal(a)
		bj, _ := json.Marshal(b)
		if string(aj) != string(bj) {
			t.Fatalf("%s: same seed gave different outcomes\n%s\n%s", p, aj, bj)
		}
	}
}

// TestInvariantsDetectInjectedBugs は意図的に壊した状態を不変条件が検出することを確認する。
func TestInvariantsDetectInjectedBugs(t *testing.T) {
	inject := map[string]func(g *Game){
		"hp_range":            func(g *Game) { g.state.Player.Health = g.state.Player.MaxHealth + 1 },
		"satiety_range":       func(g *Game) { g.state.Player.Satiety = -1 },
		"player_in_wall":      func(g *Game) { g.state.Map[g.state.Player.Y][g.state.Player.X].Blocked = true },
		"enemy_on_player":     func(g *Game) { g.state.Enemies[0].X, g.state.Enemies[0].Y = g.state.Player.X, g.state.Player.Y },
		"enemy_out_of_bounds": func(g *Game) { g.state.Enemies[0].X = -1 },
		"inventory_overflow": func(g *Game) {
			for len(g.state.Player.Inventory) <= g.state.Player.MaxInventory {
				g.state.Player.Inventory = append(g.state.Player.Inventory, buildItemFromTemplate(1, 0, 0))
			}
		},
		"enemy_overlap": func(g *Game) {
			g.state.Enemies = append(g.state.Enemies, g.state.Enemies[0])
		},
	}
	for kind, bug := range inject {
		g, _ := newHeadlessGame(1, DifficultyAdvanced)
		if vs := checkInvariants(g); len(vs) != 0 {
			t.Fatalf("fresh game has violations: %+v", vs)
		}
		bug(g)
		found := false
		for _, v := range checkInvariants(g) {
			found = found || v.Kind == kind
		}
		if !found {
			t.Errorf("injected %s was not detected", kind)
		}
	}

	// ソフトロック: 入力待ちに戻らない状態を RunUntilInputReady が検出する
	g, _ := newHeadlessGame(1, DifficultyAdvanced)
	g.Animating = true
	g.fastForward = false
	g.AnimationProgressInt = -1 << 30
	if err := g.RunUntilInputReady(100); !errors.Is(err, ErrSoftLock) {
		t.Fatalf("soft lock was not detected: %v", err)
	}
}

// --- 集計とレポート ---

// 判断ミス死の種類。
const (
	mistakeHealUnused  = "回復薬を持ったまま死亡"
	mistakeFoodUnused  = "食料を持ったまま餓死"
	mistakeHarmfulCard = "自分に不利なカードによる被害"
)

// harmfulCardWindow は不利なカードを使ってからこのターン数以内に同じ階で死んだら、その被害とみなす。
const harmfulCardWindow = 100

// mistakeKind は死亡した冒険の結果から判断ミス死の種類を返す（該当しなければ空）。
func mistakeKind(r *RunResult) string {
	if r == nil || r.Cleared {
		return ""
	}
	cause := r.DeathCause.Kind
	if cause == DamageSourceSelfDestruct {
		return mistakeHarmfulCard
	}
	for _, use := range r.Stats.RecentUses {
		if itemRoles[use.TemplateID] == roleHarmful && use.Floor == r.Floor && r.Turns-use.Turn <= harmfulCardWindow {
			return mistakeHarmfulCard
		}
	}
	hasFood, hasHeal := false, false
	for _, id := range r.InventoryIDs {
		if template, ok := itemTemplates[id]; ok && template.ItemType == "Food" {
			hasFood = true
		}
		if itemRoles[id] == roleHeal {
			hasHeal = true
		}
	}
	switch {
	case cause == DamageSourceStarvation && hasFood:
		return mistakeFoodUnused
	case cause != DamageSourceStarvation && cause != DamageSourceWind && hasHeal:
		return mistakeHealUnused
	}
	return ""
}

type simGroupStats struct {
	Runs        int
	Cleared     int
	ClearRate   float64
	AvgFloor    float64
	AvgTurns    float64
	Died        int
	TurnLimited int
	BotStuck    int
	// 判断ミス死と死亡時の状態
	MistakeDeaths   int
	MistakeRate     float64 // 死亡のうち判断ミス死の割合
	AvgDeathLevel   float64
	AvgDeathSatiety float64
	Mistakes        map[string]int              `json:",omitempty"`
	Causes          map[string]int              `json:",omitempty"` // 死因（判断ミス死はその種類で数える）
	Items           map[string]ItemActionCounts `json:",omitempty"` // 種類別の操作回数の合計
}

type simSummary struct {
	Runs        int
	Groups      map[string]*simGroupStats // "難易度/性格"
	FloorDeaths map[int]int               // 階層別の死亡数
	FloorReach  map[int]int               // 階層別の到達数
	DeathCauses map[string]int
	Violations  int
	SoftLocks   int
	Panics      int
}

type simBaselineFile struct {
	GeneratedAt string
	Summary     simSummary
}

func groupKey(o simOutcome) string { return o.Difficulty + "/" + string(o.Persona) }

func summarizeOutcomes(outcomes []simOutcome) simSummary {
	s := simSummary{Groups: map[string]*simGroupStats{}, FloorDeaths: map[int]int{}, FloorReach: map[int]int{}, DeathCauses: map[string]int{}}
	floorSums, turnSums := map[string]int{}, map[string]int{}
	levelSums, satietySums := map[string]int{}, map[string]int{}
	for _, o := range outcomes {
		s.Runs++
		key := groupKey(o)
		g := s.Groups[key]
		if g == nil {
			g = &simGroupStats{}
			s.Groups[key] = g
		}
		g.Runs++
		floorSums[key] += o.Floor
		turnSums[key] += o.Turns
		for f := 1; f <= o.Floor; f++ {
			s.FloorReach[f]++
		}
		switch o.Ended {
		case "cleared":
			g.Cleared++
		case "died":
			g.Died++
			s.FloorDeaths[o.Floor]++
			s.DeathCauses[o.DeathCause]++
			if g.Causes == nil {
				g.Causes, g.Mistakes = map[string]int{}, map[string]int{}
			}
			cause := o.DeathCause
			if o.Mistake != "" {
				g.MistakeDeaths++
				g.Mistakes[o.Mistake]++
				cause = "（判断ミス）" + o.Mistake
			}
			g.Causes[cause]++
			if o.Result != nil {
				levelSums[key] += o.Result.Level
				satietySums[key] += o.Result.Satiety
			}
		case "turn_limit":
			g.TurnLimited++
		case "bot_stuck":
			g.BotStuck++
		}
		if o.Result != nil {
			for kind, c := range o.Result.Stats.Items {
				if g.Items == nil {
					g.Items = map[string]ItemActionCounts{}
				}
				total := g.Items[kind]
				total.Picked += c.Picked
				total.Used += c.Used
				total.Thrown += c.Thrown
				total.Equipped += c.Equipped
				total.Shot += c.Shot
				g.Items[kind] = total
			}
		}
		for _, v := range o.Violations {
			s.Violations++
			switch v.Kind {
			case "softlock":
				s.SoftLocks++
			case "panic":
				s.Panics++
			}
		}
	}
	for key, g := range s.Groups {
		g.ClearRate = float64(g.Cleared) / float64(g.Runs)
		g.AvgFloor = float64(floorSums[key]) / float64(g.Runs)
		g.AvgTurns = float64(turnSums[key]) / float64(g.Runs)
		if g.Died > 0 {
			g.MistakeRate = float64(g.MistakeDeaths) / float64(g.Died)
			g.AvgDeathLevel = float64(levelSums[key]) / float64(g.Died)
			g.AvgDeathSatiety = float64(satietySums[key]) / float64(g.Died)
		}
	}
	return s
}

func (s simSummary) oneLine() string {
	cleared, died, stuck := 0, 0, 0
	for _, g := range s.Groups {
		cleared += g.Cleared
		died += g.Died
		stuck += g.BotStuck
	}
	return fmt.Sprintf("クリア %d / 死亡 %d / BOT行き詰まり %d / 違反 %d（ソフトロック %d・panic %d）", cleared, died, stuck, s.Violations, s.SoftLocks, s.Panics)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// writeSimReport は全結果の JSON と集計 Markdown を書き出す。
func writeSimReport(dir string, outcomes []simOutcome, s simSummary, baseline *simSummary, elapsed time.Duration) (string, string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	stamp := time.Now().Format("20060102-150405")
	jsonPath := filepath.Join(dir, "sim-"+stamp+".json")
	mdPath := filepath.Join(dir, "sim-"+stamp+".md")

	data, err := json.MarshalIndent(struct {
		GeneratedAt string
		Summary     simSummary
		Outcomes    []simOutcome
	}{time.Now().Format(time.RFC3339), s, outcomes}, "", "  ")
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(mdPath, []byte(renderSimMarkdown(outcomes, s, baseline, elapsed)), 0o644); err != nil {
		return "", "", err
	}
	return jsonPath, mdPath, nil
}

func renderSimMarkdown(outcomes []simOutcome, s simSummary, baseline *simSummary, elapsed time.Duration) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# BOT シミュレーションレポート\n\n")
	fmt.Fprintf(&b, "- 生成: %s（所要 %v）\n- 冒険数: %d\n- %s\n\n", time.Now().Format("2006-01-02 15:04"), elapsed.Round(time.Second), s.Runs, s.oneLine())

	fmt.Fprintf(&b, "## 難易度×性格\n\n| 難易度/性格 | 回数 | クリア率 | 平均到達階 | 平均ターン | 死亡 | ターン上限 | BOT行き詰まり |\n|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, key := range sortedKeys(s.Groups) {
		g := s.Groups[key]
		fmt.Fprintf(&b, "| %s | %d | %.1f%% | %.2f | %.0f | %d | %d | %d |\n", key, g.Runs, g.ClearRate*100, g.AvgFloor, g.AvgTurns, g.Died, g.TurnLimited, g.BotStuck)
	}

	fmt.Fprintf(&b, "\n## 判断ミス死と死亡時の状態\n\n| 難易度/性格 | 死亡 | 判断ミス死 | 割合 | %s | %s | %s | 死亡時Lv | 死亡時満腹度 |\n|---|---:|---:|---:|---:|---:|---:|---:|---:|\n", mistakeHealUnused, mistakeFoodUnused, mistakeHarmfulCard)
	for _, key := range sortedKeys(s.Groups) {
		g := s.Groups[key]
		fmt.Fprintf(&b, "| %s | %d | %d | %.1f%% | %d | %d | %d | %.1f | %.1f |\n", key, g.Died, g.MistakeDeaths, g.MistakeRate*100,
			g.Mistakes[mistakeHealUnused], g.Mistakes[mistakeFoodUnused], g.Mistakes[mistakeHarmfulCard], g.AvgDeathLevel, g.AvgDeathSatiety)
	}

	fmt.Fprintf(&b, "\n## 性格別の死因の上位（判断ミス死は種類で数える）\n\n")
	personaCauses := map[string]map[string]int{}
	for key, g := range s.Groups {
		p := key[strings.Index(key, "/")+1:]
		if personaCauses[p] == nil {
			personaCauses[p] = map[string]int{}
		}
		for cause, n := range g.Causes {
			personaCauses[p][cause] += n
		}
	}
	for _, p := range sortedKeys(personaCauses) {
		causes := sortedKeys(personaCauses[p])
		sort.SliceStable(causes, func(i, j int) bool { return personaCauses[p][causes[i]] > personaCauses[p][causes[j]] })
		var parts []string
		for i, cause := range causes {
			if i >= 5 {
				break
			}
			parts = append(parts, fmt.Sprintf("%s %d", cause, personaCauses[p][cause]))
		}
		fmt.Fprintf(&b, "- %s: %s\n", p, strings.Join(parts, "、"))
	}

	fmt.Fprintf(&b, "\n## 道具の操作回数（1冒険あたり、全難易度）\n\n| 性格 | 拾う | 使う | 投げる | 装備 | 矢を撃つ | 使う（カード） | 使う（薬） | 使う（食料） |\n|---|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	type itemRow struct {
		runs  int
		total ItemActionCounts
		kinds map[string]ItemActionCounts
	}
	rows := map[string]*itemRow{}
	for key, g := range s.Groups {
		p := key[strings.Index(key, "/")+1:]
		if rows[p] == nil {
			rows[p] = &itemRow{kinds: map[string]ItemActionCounts{}}
		}
		r := rows[p]
		r.runs += g.Runs
		for kind, c := range g.Items {
			r.total.Picked += c.Picked
			r.total.Used += c.Used
			r.total.Thrown += c.Thrown
			r.total.Equipped += c.Equipped
			r.total.Shot += c.Shot
			k := r.kinds[kind]
			k.Used += c.Used
			r.kinds[kind] = k
		}
	}
	for _, p := range sortedKeys(rows) {
		r := rows[p]
		n := float64(max(1, r.runs))
		fmt.Fprintf(&b, "| %s | %.1f | %.1f | %.1f | %.1f | %.1f | %.1f | %.1f | %.1f |\n", p,
			float64(r.total.Picked)/n, float64(r.total.Used)/n, float64(r.total.Thrown)/n, float64(r.total.Equipped)/n, float64(r.total.Shot)/n,
			float64(r.kinds["Card"].Used)/n, float64(r.kinds["Potion"].Used)/n, float64(r.kinds["Food"].Used)/n)
	}

	fmt.Fprintf(&b, "\n## 階層別の死亡率\n\n| 階 | 到達 | 死亡 | 死亡率 |\n|---:|---:|---:|---:|\n")
	floors := make([]int, 0, len(s.FloorReach))
	for f := range s.FloorReach {
		floors = append(floors, f)
	}
	sort.Ints(floors)
	for _, f := range floors {
		reach, deaths := s.FloorReach[f], s.FloorDeaths[f]
		fmt.Fprintf(&b, "| %d | %d | %d | %.1f%% |\n", f, reach, deaths, float64(deaths)*100/float64(max(1, reach)))
	}

	fmt.Fprintf(&b, "\n## 死因の上位\n\n| 死因 | 回数 |\n|---|---:|\n")
	causes := sortedKeys(s.DeathCauses)
	sort.SliceStable(causes, func(i, j int) bool { return s.DeathCauses[causes[i]] > s.DeathCauses[causes[j]] })
	for i, cause := range causes {
		if i >= 15 {
			break
		}
		fmt.Fprintf(&b, "| %s | %d |\n", cause, s.DeathCauses[cause])
	}

	fmt.Fprintf(&b, "\n## 不変条件違反・ソフトロック・panic\n\n")
	listed := 0
	for _, o := range outcomes {
		for _, v := range o.Violations {
			if listed == 0 {
				fmt.Fprintf(&b, "| 種類 | シード | 性格 | 難易度 | 階 | ターン | 内容 |\n|---|---:|---|---|---:|---:|---|\n")
			}
			listed++
			if listed > 100 {
				continue
			}
			fmt.Fprintf(&b, "| %s | %d | %s | %s | %d | %d | %s |\n", v.Kind, o.Seed, o.Persona, o.Difficulty, v.Floor, v.Turn, strings.ReplaceAll(firstLine(v.Detail), "|", "\\|"))
		}
	}
	if listed == 0 {
		fmt.Fprintf(&b, "なし\n")
	} else {
		fmt.Fprintf(&b, "\n（全 %d 件。再現: `go test -run TestSimReplay -count=1 . -args -sim.seed=<シード> -sim.persona=<性格> -sim.difficulty=<beginner|intermediate|advanced>`）\n", listed)
	}

	fmt.Fprintf(&b, "\n## ベースラインとの差分\n\n")
	if baseline == nil {
		fmt.Fprintf(&b, "ベースラインなし（`reports/baseline.json`）\n")
		return b.String()
	}
	fmt.Fprintf(&b, "| 難易度/性格 | クリア率 | 平均到達階 | 平均ターン | 判断ミス死の割合 |\n|---|---:|---:|---:|---:|\n")
	for _, key := range sortedKeys(s.Groups) {
		cur, base := s.Groups[key], baseline.Groups[key]
		if base == nil {
			fmt.Fprintf(&b, "| %s | 新規 | | | |\n", key)
			continue
		}
		fmt.Fprintf(&b, "| %s | %+.1f pt | %+.2f | %+.0f | %.1f%% → %.1f%% |\n", key, (cur.ClearRate-base.ClearRate)*100, cur.AvgFloor-base.AvgFloor, cur.AvgTurns-base.AvgTurns, base.MistakeRate*100, cur.MistakeRate*100)
	}
	fmt.Fprintf(&b, "\n違反: %d → %d\n", baseline.Violations, s.Violations)
	return b.String()
}

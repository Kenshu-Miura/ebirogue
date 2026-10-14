//go:build !test

package main

// 序盤（1〜5階）の死亡分析。expert BOT の死亡直前の数十ターンを記録し、
// 「BOT の判断ミス」と「適切に動いても避けにくい死」に分類して集計する。
//
//	go test -run TestEarlyDeathAnalysis -count=1 -timeout 0 . -args -sim.early=2000 -sim.out=reports
//
// 分類基準は classifyEarlyDeath と docs/agents/early-game-analysis.md を参照。

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

var (
	simEarly      = flag.Int("sim.early", 0, "TestEarlyDeathAnalysis の冒険数（expert・上級。0 なら実行しない）")
	simEarlyFloor = flag.Int("sim.earlyfloor", 5, "TestEarlyDeathAnalysis で分析する最も深い階")
)

const deathTraceLength = 40 // 死亡直前に残すターン数

// traceSnap は1ターン分の状態の記録。
type traceSnap struct {
	Turn, Floor, FloorTurns  int
	HP, MaxHP, Satiety       int
	Level, Power, Atk, Def   int
	Act                      string `json:",omitempty"` // BOT が選んだ行動（bot.act）
	Adjacent                 []int  `json:",omitempty"` // 隣にいる敵の ID
	Near                     int    // 5マス以内の敵の数
	InFire                   bool   // いずれかの敵の遠距離攻撃が届く（playerUnderRangedFire）
	StairsDist               int    // 発見済みの階段までの歩数（未発見・到達不能は -1）
	Heal, Escape, AttackItem int    // 所持品の分類ごとの数（itemRoles）
	Food                     int
}

// deathTrace は死亡した冒険の直前の記録。
type deathTrace struct {
	Snaps     []traceSnap
	FloorFood []int // 各階に着いたとき床にあった食料の数（i 番目が i+1 階）
	FloorHeal []int // 同じく回復の道具の数
}

type traceRecorder struct {
	ring      []traceSnap
	floorFood []int
	floorHeal []int
	lastFloor int
}

func countInventoryRoles(items []Item) (heal, escape, attack, food int) {
	for _, item := range items {
		switch itemRoles[item.GetID()] {
		case roleHeal:
			heal++
		case roleEscape:
			escape++
		case roleAttack:
			attack++
		}
		if _, ok := item.(*Food); ok && item.GetID() != rottenSausageItemID {
			food++
		}
	}
	return
}

// playerUnderRangedFire はいずれかの敵が今プレイヤーへ遠距離攻撃できるかどうか（ゲーム本体の判定を使う。
// BOT の inRangedFire とは独立させ、BOT の改善前後で同じ基準になるようにする）。
func playerUnderRangedFire(g *Game) bool {
	for i := range g.state.Enemies {
		if g.canEnemyUseRangedAttack(i) {
			return true
		}
	}
	return false
}

// stairsDistance は発見済みの階段までの歩数（敵は無視する）。
func stairsDistance(g *Game) int {
	m := g.state.Map
	height, width := len(m), len(m[0])
	p := g.state.Player
	dist := make([]int, width*height)
	for i := range dist {
		dist[i] = -1
	}
	start := p.Y*width + p.X
	dist[start] = 0
	queue := []int{start}
	b := &bot{g: g}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		x, y := cur%width, cur/width
		if m[y][x].Type == "stairs" && m[y][x].Visited {
			return dist[cur]
		}
		for _, s := range botSteps {
			if !b.canStep(x, y, s[0], s[1]) {
				continue
			}
			next := (y+s[1])*width + x + s[0]
			if dist[next] < 0 {
				dist[next] = dist[cur] + 1
				queue = append(queue, next)
			}
		}
	}
	return -1
}

func (r *traceRecorder) record(b *bot) {
	g := b.g
	p := &g.state.Player
	if g.Floor != r.lastFloor {
		r.lastFloor = g.Floor
		food, heal := 0, 0
		for _, item := range g.state.Items {
			if _, ok := item.(*Food); ok && item.GetID() != rottenSausageItemID {
				food++
			}
			if itemRoles[item.GetID()] == roleHeal {
				heal++
			}
		}
		for len(r.floorFood) < g.Floor {
			r.floorFood = append(r.floorFood, food)
			r.floorHeal = append(r.floorHeal, heal)
		}
	}
	s := traceSnap{
		Turn: g.turnCount, Floor: g.Floor, FloorTurns: g.floorTurns,
		HP: p.Health, MaxHP: p.MaxHealth, Satiety: p.Satiety,
		Level: p.Level, Power: p.Power, Atk: p.AttackPower, Def: p.DefensePower,
		Act: b.act, InFire: playerUnderRangedFire(g), StairsDist: stairsDistance(g),
	}
	for _, enemy := range b.adjacentEnemies() {
		s.Adjacent = append(s.Adjacent, enemy.ID)
	}
	for _, enemy := range g.state.Enemies {
		if max(abs(enemy.X-p.X), abs(enemy.Y-p.Y)) <= 5 {
			s.Near++
		}
	}
	s.Heal, s.Escape, s.AttackItem, s.Food = countInventoryRoles(p.Inventory)
	r.ring = append(r.ring, s)
	if len(r.ring) > deathTraceLength {
		r.ring = r.ring[len(r.ring)-deathTraceLength:]
	}
}

func (r *traceRecorder) trace() *deathTrace {
	return &deathTrace{Snaps: append([]traceSnap(nil), r.ring...), FloorFood: r.floorFood, FloorHeal: r.floorHeal}
}

// --- 分類 ---

// 判断ミス（BOT の改善で防げる）
const (
	earlyFoodUnused   = "a:食料を持ったまま餓死"
	earlyHealUnused   = "a:回復の道具を持ったまま死亡"
	earlyHarmfulCard  = "a:不利なカードの被害"
	earlyEscapeUnused = "a:止める・倒す道具（カード・薬）を持ったまま死亡"
	earlyRestSkipped  = "a:HPが減ったまま探索を続けて交戦"
	earlyStayedInFire = "a:射線に立ち続けた"
	earlyStairsIgnore = "a:階段で逃げられたのに戦った"
)

// 避けにくい死（ゲームの数値の問題）
const (
	earlyStarvedNoFood = "b:食料が無く餓死"
	earlyOutmatched    = "b:満タンでも勝てない強さの敵"
	earlyBurst         = "b:HP半分以上から3ターン以内に倒れた"
	earlySwarmed       = "b:複数の敵に囲まれた"
	earlyTrap          = "b:罠"
	earlyAttrition     = "b:連戦で消耗（道具なし）"
)

// earlySaverItems は持っていればその場で敵を止める・倒すのに使えた道具（判断ミスの判定に使う）。
// 封印の杖・シフトチェンジの杖・ジェノサイドのカードは敵を止めないか、使える場面が限られるので含めない。
var earlySaverItems = map[int]bool{
	7: true, 31: true, 56: true, // 黒炎弾・真空斬り・全滅のカード
	12: true, 13: true, 14: true, 15: true, // 睡眠・混乱・目潰し・金縛りのカード
	17: true, 18: true, 19: true, // 睡眠薬・混乱薬・目潰し薬
}

// killerFightCost は満タンのプレイヤーが敵を通常攻撃だけで倒すまでに受けるダメージの見込み（最大HP比）。
func killerFightCost(enemyID int, s traceSnap) float64 {
	def, ok := MonsterDefinitions[enemyID]
	if !ok || s.MaxHP <= 0 {
		return 0
	}
	perHit := float64(playerAttackTotal(s.Atk, s.Power, s.Level)) * pow1516(def.DefensePower)
	if perHit < 1 {
		perHit = 1
	}
	taken := float64(def.Health) / perHit * float64(def.AttackPower) * pow1516(s.Def)
	return taken / float64(s.MaxHP)
}

func hpRate(s traceSnap) float64 {
	if s.MaxHP <= 0 {
		return 0
	}
	return float64(s.HP) / float64(s.MaxHP)
}

func containsID(ids []int, id int) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// classifyEarlyDeath は死亡を分類する。判断ミスの判定を先に行い、どれにも当たらなければ避けにくい死とする。
func classifyEarlyDeath(o simOutcome) string {
	r, tr := o.Result, o.Death
	if r == nil || tr == nil || len(tr.Snaps) == 0 {
		return "不明"
	}
	snaps := tr.Snaps
	last := snaps[len(snaps)-1]
	cause := r.DeathCause
	byEnemy := cause.Kind == DamageSourceEnemyAttack || cause.Kind == DamageSourceEnemyRanged
	heal, savers, food := 0, 0, 0
	for _, id := range r.InventoryIDs {
		if itemRoles[id] == roleHeal {
			heal++
		}
		if earlySaverItems[id] {
			savers++
		}
		if template, ok := itemTemplates[id]; ok && template.ItemType == "Food" && id != rottenSausageItemID {
			food++
		}
	}
	switch {
	case mistakeKind(r) == mistakeHarmfulCard:
		return earlyHarmfulCard
	case cause.Kind == DamageSourceStarvation && food > 0:
		return earlyFoodUnused
	case cause.Kind != DamageSourceStarvation && cause.Kind != DamageSourceWind && heal > 0:
		return earlyHealUnused
	case byEnemy && savers > 0:
		return earlyEscapeUnused
	}
	window := snaps[max(0, len(snaps)-15):]
	if byEnemy {
		// 交戦の始まり: 倒した敵が最初に隣（遠距離なら射線）に来たターン
		start := -1
		for i, s := range window {
			if containsID(s.Adjacent, cause.EnemyID) || (cause.Kind == DamageSourceEnemyRanged && s.InFire) {
				start = i
				break
			}
		}
		if start >= 0 {
			s := window[start]
			calm := true
			for _, prev := range window[:start] {
				if len(prev.Adjacent) > 0 {
					calm = false
				}
			}
			if calm && start >= 3 && hpRate(s) < 0.5 && s.Satiety >= 20 {
				return earlyRestSkipped
			}
		}
		if cause.Kind == DamageSourceEnemyRanged {
			exposed := 0
			for _, s := range window[max(0, len(window)-10):] {
				if s.InFire && s.Act != "toRanged" && s.Act != "melee" && s.Act != "shoot" && s.Act != "throw" {
					exposed++
				}
			}
			if exposed >= 3 {
				return earlyStayedInFire
			}
		}
		for _, s := range window[max(0, len(window)-10):] {
			if s.StairsDist == 0 && hpRate(s) < 0.5 && len(s.Adjacent) > 0 {
				return earlyStairsIgnore
			}
		}
	}
	switch {
	case cause.Kind == DamageSourceStarvation:
		return earlyStarvedNoFood
	case cause.Kind == DamageSourceTrap:
		return earlyTrap
	case byEnemy && killerFightCost(cause.EnemyID, last) >= 1:
		return earlyOutmatched
	}
	recent := snaps[max(0, len(snaps)-4):]
	if byEnemy && hpRate(recent[0]) >= 0.5 {
		return earlyBurst
	}
	for _, s := range recent {
		if len(s.Adjacent) >= 2 {
			return earlySwarmed
		}
	}
	return earlyAttrition
}

// --- 集計 ---

type earlyFloorStats struct {
	Reached, Died int
	DeathLevel    float64 // 死亡時の平均レベル
	ArrivalLevel  float64 // その階に着いた全冒険の到着時平均レベル
	FoodOnFloor   float64 // 着いたとき床にあった食料の平均
	HealOnFloor   float64
}

type earlyReport struct {
	Runs        int
	Persona     string
	MaxFloor    int
	Floors      map[int]*earlyFloorStats
	Classes     map[string]int            // 分類ごとの数
	ClassFloors map[string]map[int]int    // 分類×階
	Causes      map[string]int            // 死因
	CauseClass  map[string]map[string]int // 死因×分類
	LastActs    map[string]int            // 死亡直前5ターンの行動
	HPBefore    map[int]float64           // 死亡の n ターン前の平均HP率（n=1,5,10,20）
	SatietyAt   float64                   // 死亡時の平均満腹度
	PowerAt     float64                   // 死亡時の平均パワー（初期8）
	ZeroItems   int                       // 死亡時に回復・止める・倒す道具を1つも持っていなかった数
	Examples    map[string][]string       // 分類ごとの再現シード（最大5件）
	Deaths      int
}

func buildEarlyReport(outcomes []simOutcome, maxFloor int) earlyReport {
	rep := earlyReport{
		Runs: len(outcomes), MaxFloor: maxFloor, Floors: map[int]*earlyFloorStats{},
		Classes: map[string]int{}, ClassFloors: map[string]map[int]int{}, Causes: map[string]int{},
		CauseClass: map[string]map[string]int{}, LastActs: map[string]int{}, HPBefore: map[int]float64{},
		Examples: map[string][]string{},
	}
	if len(outcomes) > 0 {
		rep.Persona = string(outcomes[0].Persona)
	}
	for f := 1; f <= maxFloor; f++ {
		rep.Floors[f] = &earlyFloorStats{}
	}
	hpCounts := map[int]int{}
	for _, o := range outcomes {
		for f := 1; f <= maxFloor && f <= len(o.FloorLevels); f++ {
			st := rep.Floors[f]
			st.Reached++
			st.ArrivalLevel += float64(o.FloorLevels[f-1])
			if o.Death != nil && f <= len(o.Death.FloorFood) {
				st.FoodOnFloor += float64(o.Death.FloorFood[f-1])
				st.HealOnFloor += float64(o.Death.FloorHeal[f-1])
			}
		}
		if o.Ended != "died" || o.Floor > maxFloor || o.Death == nil {
			continue
		}
		rep.Deaths++
		st := rep.Floors[o.Floor]
		st.Died++
		st.DeathLevel += float64(o.Level)
		class := classifyEarlyDeath(o)
		rep.Classes[class]++
		if rep.ClassFloors[class] == nil {
			rep.ClassFloors[class] = map[int]int{}
		}
		rep.ClassFloors[class][o.Floor]++
		rep.Causes[o.DeathCause]++
		if rep.CauseClass[o.DeathCause] == nil {
			rep.CauseClass[o.DeathCause] = map[string]int{}
		}
		rep.CauseClass[o.DeathCause][class]++
		if len(rep.Examples[class]) < 5 {
			rep.Examples[class] = append(rep.Examples[class], fmt.Sprintf("%d", o.Seed))
		}
		snaps := o.Death.Snaps
		for _, s := range snaps[max(0, len(snaps)-5):] {
			act := s.Act
			if act == "" {
				act = "(その他)"
			}
			rep.LastActs[act]++
		}
		last := snaps[len(snaps)-1]
		for _, n := range []int{1, 5, 10, 20} {
			for i := len(snaps) - 1; i >= 0; i-- {
				if last.Turn-snaps[i].Turn >= n {
					rep.HPBefore[n] += hpRate(snaps[i])
					hpCounts[n]++
					break
				}
			}
		}
		rep.SatietyAt += float64(o.Result.Satiety)
		rep.PowerAt += float64(last.Power)
		if h, e, a, _ := countInventoryIDs(o.Result.InventoryIDs); h+e+a == 0 {
			rep.ZeroItems++
		}
	}
	for n, c := range hpCounts {
		rep.HPBefore[n] /= float64(c)
	}
	if rep.Deaths > 0 {
		rep.SatietyAt /= float64(rep.Deaths)
		rep.PowerAt /= float64(rep.Deaths)
	}
	for _, st := range rep.Floors {
		if st.Died > 0 {
			st.DeathLevel /= float64(st.Died)
		}
		if st.Reached > 0 {
			st.ArrivalLevel /= float64(st.Reached)
			st.FoodOnFloor /= float64(st.Reached)
			st.HealOnFloor /= float64(st.Reached)
		}
	}
	return rep
}

func countInventoryIDs(ids []int) (heal, escape, attack, food int) {
	for _, id := range ids {
		switch itemRoles[id] {
		case roleHeal:
			heal++
		case roleEscape:
			escape++
		case roleAttack:
			attack++
		}
		if template, ok := itemTemplates[id]; ok && template.ItemType == "Food" {
			food++
		}
	}
	return
}

func sortedByCount(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if m[keys[i]] != m[keys[j]] {
			return m[keys[i]] > m[keys[j]]
		}
		return keys[i] < keys[j]
	})
	return keys
}

func renderEarlyReport(rep earlyReport) string {
	var b strings.Builder
	pct := func(n, d int) string {
		if d == 0 {
			return "-"
		}
		return fmt.Sprintf("%.1f%%", float64(n)*100/float64(d))
	}
	fmt.Fprintf(&b, "# 序盤（1〜%d階）の死亡分析\n\n", rep.MaxFloor)
	fmt.Fprintf(&b, "- 生成: %s、性格 %s、冒険数 %d、1〜%d階での死亡 %d\n\n", time.Now().Format("2006-01-02 15:04"), rep.Persona, rep.Runs, rep.MaxFloor, rep.Deaths)
	b.WriteString("## 階ごと\n\n| 階 | 到達 | 死亡 | 死亡率 | 到着時Lv | 死亡時Lv | 床の食料 | 床の回復 |\n|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for f := 1; f <= rep.MaxFloor; f++ {
		st := rep.Floors[f]
		fmt.Fprintf(&b, "| %d | %d | %d | %s | %.1f | %.1f | %.2f | %.2f |\n", f, st.Reached, st.Died, pct(st.Died, st.Reached), st.ArrivalLevel, st.DeathLevel, st.FoodOnFloor, st.HealOnFloor)
	}
	a, bb := 0, 0
	for class, n := range rep.Classes {
		if strings.HasPrefix(class, "a:") {
			a += n
		} else {
			bb += n
		}
	}
	fmt.Fprintf(&b, "\n## 分類\n\n- (a) 判断ミス %d（%s）、(b) 避けにくい死 %d（%s）\n\n", a, pct(a, rep.Deaths), bb, pct(bb, rep.Deaths))
	b.WriteString("| 分類 | 数 | 割合 |")
	for f := 1; f <= rep.MaxFloor; f++ {
		fmt.Fprintf(&b, " %d階 |", f)
	}
	b.WriteString(" 再現シード |\n|---|---:|---:|")
	for f := 1; f <= rep.MaxFloor; f++ {
		b.WriteString("---:|")
	}
	b.WriteString("---|\n")
	for _, class := range sortedByCount(rep.Classes) {
		fmt.Fprintf(&b, "| %s | %d | %s |", class, rep.Classes[class], pct(rep.Classes[class], rep.Deaths))
		for f := 1; f <= rep.MaxFloor; f++ {
			fmt.Fprintf(&b, " %d |", rep.ClassFloors[class][f])
		}
		fmt.Fprintf(&b, " %s |\n", strings.Join(rep.Examples[class], " "))
	}
	b.WriteString("\n## 死因の上位と分類\n\n| 死因 | 数 | 内訳 |\n|---|---:|---|\n")
	for i, cause := range sortedByCount(rep.Causes) {
		if i >= 15 {
			break
		}
		var parts []string
		for _, class := range sortedByCount(rep.CauseClass[cause]) {
			parts = append(parts, fmt.Sprintf("%s %d", class, rep.CauseClass[cause][class]))
		}
		fmt.Fprintf(&b, "| %s | %d | %s |\n", cause, rep.Causes[cause], strings.Join(parts, "、"))
	}
	b.WriteString("\n## 死亡直前の状態\n\n")
	fmt.Fprintf(&b, "- HP率（最大HP比）: 20ターン前 %.0f%%、10ターン前 %.0f%%、5ターン前 %.0f%%、1ターン前 %.0f%%\n",
		rep.HPBefore[20]*100, rep.HPBefore[10]*100, rep.HPBefore[5]*100, rep.HPBefore[1]*100)
	fmt.Fprintf(&b, "- 死亡時の平均満腹度 %.1f、平均パワー %.1f（初期8）\n", rep.SatietyAt, rep.PowerAt)
	fmt.Fprintf(&b, "- 回復・止める・倒す道具を1つも持たずに死亡: %d（%s）\n", rep.ZeroItems, pct(rep.ZeroItems, rep.Deaths))
	b.WriteString("- 死亡直前5ターンの行動: ")
	var acts []string
	total := 0
	for _, n := range rep.LastActs {
		total += n
	}
	for _, act := range sortedByCount(rep.LastActs) {
		acts = append(acts, fmt.Sprintf("%s %s", act, pct(rep.LastActs[act], total)))
	}
	b.WriteString(strings.Join(acts, "、") + "\n")
	return b.String()
}

// TestEarlyDeathAnalysis は expert BOT（上級）で -sim.early 回遊び、序盤の死亡を分類したレポートを書き出す。
func TestEarlyDeathAnalysis(t *testing.T) {
	if *simEarly <= 0 {
		t.Skip("-sim.early を指定したときだけ実行する")
	}
	p := personaExpert
	if *simPersona != "all" {
		parsed, ok := parsePersona(*simPersona)
		if !ok {
			t.Fatalf("不明な性格 %q", *simPersona)
		}
		p = parsed
	}
	cases := buildSimCases(*simSeedBase, *simEarly, []persona{p}, []Difficulty{DifficultyAdvanced}, *simTurns)
	for i := range cases {
		cases[i].Trace = true
	}
	start := time.Now()
	outcomes := runSimCases(cases, *simParallel)
	rep := buildEarlyReport(outcomes, *simEarlyFloor)
	md := renderEarlyReport(rep)
	stamp := time.Now().Format("20060102-150405")
	if err := os.MkdirAll(*simOut, 0o755); err != nil {
		t.Fatal(err)
	}
	mdPath := filepath.Join(*simOut, "early-"+stamp+".md")
	jsonPath := filepath.Join(*simOut, "early-"+stamp+".json")
	if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d 回（%v）: %s", len(outcomes), time.Since(start).Round(time.Second), summarizeOutcomes(outcomes).oneLine())
	t.Logf("レポート: %s", mdPath)
	failOnViolations(t, outcomes)
}

var simEarlyDump = flag.Int64("sim.earlydump", -1, "TestEarlyDeathDump で死亡直前の記録を表示するシード（expert・上級）")

// TestEarlyDeathDump は1件の冒険の死亡直前の記録を表示する（分類の確認用）。
func TestEarlyDeathDump(t *testing.T) {
	if *simEarlyDump < 0 {
		t.Skip("-sim.earlydump を指定したときだけ実行する")
	}
	o := runSimCase(simCase{Seed: *simEarlyDump, Persona: personaExpert, Difficulty: DifficultyAdvanced, MaxTurns: *simTurns, Trace: true})
	t.Logf("seed=%d ended=%s floor=%d level=%d cause=%s class=%s", o.Seed, o.Ended, o.Floor, o.Level, o.DeathCause, classifyEarlyDeath(o))
	if o.Result != nil {
		t.Logf("所持品: %s", strings.Join(o.Result.Inventory, "、"))
		t.Logf("装備: %s", strings.Join(o.Result.Equipment, "、"))
	}
	if o.Death == nil {
		return
	}
	for _, s := range o.Death.Snaps {
		var adj []string
		for _, id := range s.Adjacent {
			adj = append(adj, MonsterDefinitions[id].Name)
		}
		t.Logf("T%d F%d ft%d HP%d/%d 満%d Lv%d P%d A%d D%d %-12s 隣[%s] 近%d 射線%v 階段%d 回%d 止%d 攻%d 食%d",
			s.Turn, s.Floor, s.FloorTurns, s.HP, s.MaxHP, s.Satiety, s.Level, s.Power, s.Atk, s.Def, s.Act,
			strings.Join(adj, ","), s.Near, s.InFire, s.StairsDist, s.Heal, s.Escape, s.AttackItem, s.Food)
	}
}

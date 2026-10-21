//go:build !test

package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// 段11（wave11-0）の計測: 段7〜10の要素（新しい道具・モンスターハウス・地形・店・探索）を BOT がどう扱っているか。
// ゲームの状態を読むだけで、乱数・判断は変えない。使った・投げた・壺に入れた回数と店の金額はゲームの RunStats（UsedIDs・ThrownIDs・PotInsertIDs など）から読み、
// 手に入れた数は所持品の ID ごとの数の増え方から数える（拾う・足元の交換・壺から出すを区別しない）。

// wave11ItemIDs はレポートに載せる段8の薬・杖と段9の壺の ID。
func wave11ItemIDs() []int {
	var ids []int
	for id := 205; id <= 239; id++ {
		if _, ok := itemTemplates[id]; ok && (id <= 221 || id >= 225) {
			ids = append(ids, id)
		}
	}
	return ids
}

func isWave11ItemID(id int) bool { return (id >= 205 && id <= 221) || (id >= 225 && id <= 239) }

// simHouseEntry は自然生成のモンスターハウスに初めて入ったときの記録。
type simHouseEntry struct {
	HPPct int // 入ったときの HP の割合（%）
	Items int // 入ったときの所持品の数
	Stay  int // 入ってから部屋を出る・階を離れる・冒険が終わるまでのターン
}

// simDescent は階段で降りたときの記録。
type simDescent struct {
	Wait    int // 階段を見つけてから降りるまでのターン（見つけていなかった階は -1）
	HPPct   int // 降りる直前の HP の割合（%）
	Satiety int // 降りる直前の満腹度
}

// simFloorTracker は1回の冒険の、階ごとの状態を追う（runSimCase から毎ステップ呼ぶ）。
type simFloorTracker struct {
	floor       int
	terrain     bool                  // この階に地形（水路・割れた壁・隠し通路）がある
	terrainLeft map[Coordinate]string // まだ元のままの割れた壁・隠し通路
	stairs      Coordinate
	hasStairs   bool
	stairsSeen  int // 階段を見つけた累計ターン（-1 は未発見）
	inHouse     bool
	houseOpen   int // 記録中のモンスターハウスの入室の添字（-1 は無し）
	houseEnter  int
	houseDone   bool // この階のモンスターハウスに入った
	inShop      bool
	shopDone    bool
	// 直前のステップの状態（降りたときの記録に使う）
	prevX, prevY, prevHPPct, prevSatiety, prevTurns int
	held                                            map[int]int // 直前のステップの所持品の ID ごとの数（段8・9の道具だけ）
}

func newSimFloorTracker() *simFloorTracker { return &simFloorTracker{houseOpen: -1} }

func hpPercent(g *Game) int {
	p := g.state.Player
	return p.Health * 100 / max(1, p.MaxHealth)
}

// arrive は新しい階に着いたときに地形・階段・モンスターハウスを調べる。
func (t *simFloorTracker) arrive(g *Game, out *simOutcome) {
	t.floor = g.Floor
	t.terrain, t.hasStairs, t.stairsSeen = false, false, -1
	t.terrainLeft = map[Coordinate]string{}
	t.inHouse, t.houseDone, t.inShop, t.shopDone = false, false, false, false
	for y, row := range g.state.Map {
		for x, tile := range row {
			switch tile.Type {
			case "water":
				t.terrain = true
			case "cracked_wall", "hidden_corridor":
				t.terrain = true
				t.terrainLeft[Coordinate{X: x, Y: y}] = tile.Type
			case "stairs":
				t.stairs, t.hasStairs = Coordinate{X: x, Y: y}, true
			}
		}
	}
	for _, r := range g.rooms {
		if r.Role == RoomRoleMonsterHouse {
			out.HouseFloors++
			break
		}
	}
}

// closeHouse は記録中のモンスターハウスの滞在を閉じる。
func (t *simFloorTracker) closeHouse(turns int, out *simOutcome) {
	if t.houseOpen >= 0 {
		out.HouseEntries[t.houseOpen].Stay = turns - t.houseEnter
		t.houseOpen = -1
	}
}

// step は毎ステップ（行動の前）に呼ぶ。turns は累計ターン。
func (t *simFloorTracker) step(g *Game, turns int, out *simOutcome) {
	if g.Floor != t.floor {
		if t.floor != 0 {
			t.closeHouse(turns, out)
			out.FloorTerrain = append(out.FloorTerrain, t.terrain)
			if g.Floor == t.floor+1 && t.hasStairs && t.prevX == t.stairs.X && t.prevY == t.stairs.Y {
				wait := -1
				if t.stairsSeen >= 0 {
					wait = t.prevTurns - t.stairsSeen
				}
				out.Descents = append(out.Descents, simDescent{Wait: wait, HPPct: t.prevHPPct, Satiety: t.prevSatiety})
			}
		}
		t.arrive(g, out)
	}
	for c, kind := range t.terrainLeft {
		if g.state.Map[c.Y][c.X].Type == kind {
			continue
		}
		delete(t.terrainLeft, c)
		if kind == "cracked_wall" {
			out.CrackedBroken++
		} else {
			out.HiddenFound++
		}
	}
	if t.hasStairs && t.stairsSeen < 0 && g.state.Map[t.stairs.Y][t.stairs.X].Visited {
		t.stairsSeen = turns
	}
	p := g.state.Player
	inHouse := g.inMonsterHouse()
	if inHouse && !t.inHouse && !t.houseDone {
		t.houseDone = true
		out.HouseEntries = append(out.HouseEntries, simHouseEntry{HPPct: hpPercent(g), Items: len(p.Inventory)})
		t.houseOpen, t.houseEnter = len(out.HouseEntries)-1, turns
	}
	if !inHouse && t.inHouse {
		t.closeHouse(turns, out)
	}
	t.inHouse = inHouse
	inShop := g.inShopInterior(p.X, p.Y)
	if inShop && !t.inShop && !t.shopDone {
		t.shopDone = true
		out.ShopEntries++
	}
	t.inShop = inShop
	held := map[int]int{}
	for _, item := range p.Inventory {
		if id := item.GetID(); isWave11ItemID(id) {
			held[id]++
		}
	}
	for id, n := range held {
		if gained := n - t.held[id]; gained > 0 {
			if out.Acquired == nil {
				out.Acquired = map[int]int{}
			}
			out.Acquired[id] += gained
		}
	}
	t.held = held
	t.prevX, t.prevY, t.prevHPPct, t.prevSatiety, t.prevTurns = p.X, p.Y, hpPercent(g), p.Satiety, turns
}

// finish は冒険の終わりに呼ぶ。
func (t *simFloorTracker) finish(g *Game, turns int, out *simOutcome) {
	t.closeHouse(turns, out)
	out.EndTerrain = t.terrain
	out.EndCash = g.state.Player.Cash
}

// simWave11Stats は性格ごとの段7〜10の要素の集計。
type simWave11Stats struct {
	Runs int
	// 道具（ID ごと）: 手に入れた・使った（飲む・読む・振る）・投げた・壺に入れた（壺の ID）・持ったまま死んだ
	Acquired, Used, Thrown, PotInsert, HeldAtDeath map[int]int `json:",omitempty"`
	// モンスターハウス（自然生成）
	HouseFloors, HouseEntries, HouseDeaths     int
	HouseHPPctSum, HouseItemsSum, HouseStaySum int
	// 地形のある階・無い階（滞在は降りた階だけ。突風・餓死は死んだ階）
	TerrainStaySum, TerrainStayCount, PlainStaySum, PlainStayCount int
	TerrainWind, PlainWind, TerrainStarve, PlainStarve             int
	CrackedBroken, HiddenFound                                     int
	// 店
	ShopFloors, ShopEntries, Bought, Sold, Spent, Earned int
	DeathCashMedian                                      int
	// 探索（階段で降りた階）
	Descents, DescentWaitSum, DescentWaitCount, DescentHPPctSum, DescentSatietySum int
}

func (w *simWave11Stats) add(m *map[int]int, id, n int) {
	if n == 0 || !isWave11ItemID(id) {
		return
	}
	if *m == nil {
		*m = map[int]int{}
	}
	(*m)[id] += n
}

// summarizeWave11 は性格ごとに段7〜10の要素を集計する。
func summarizeWave11(outcomes []simOutcome) map[string]*simWave11Stats {
	stats := map[string]*simWave11Stats{}
	cash := map[string][]int{}
	for _, o := range outcomes {
		name := string(o.Persona)
		w := stats[name]
		if w == nil {
			w = &simWave11Stats{}
			stats[name] = w
		}
		w.Runs++
		died := o.Ended == "died"
		var cause DamageSourceKind
		if o.Result != nil {
			rs := o.Result.Stats
			for id, n := range rs.UsedIDs {
				w.add(&w.Used, id, n)
			}
			for id, n := range rs.ThrownIDs {
				w.add(&w.Thrown, id, n)
			}
			for id, n := range rs.PotInsertIDs {
				w.add(&w.PotInsert, id, n)
			}
			w.Bought += rs.ShopBought
			w.Sold += rs.ShopSold
			w.Spent += rs.ShopSpent
			w.Earned += rs.ShopEarned
			if died {
				for _, id := range o.Result.InventoryIDs {
					w.add(&w.HeldAtDeath, id, 1)
				}
				cause = o.Result.DeathCause.Kind
			}
		}
		for id, n := range o.Acquired {
			w.add(&w.Acquired, id, n)
		}
		w.HouseFloors += o.HouseFloors
		for _, e := range o.HouseEntries {
			w.HouseEntries++
			w.HouseHPPctSum += e.HPPct
			w.HouseItemsSum += e.Items
			w.HouseStaySum += e.Stay
		}
		if died && o.MonsterHouseDeath {
			w.HouseDeaths++
		}
		for i, stay := range o.FloorStays {
			if i < len(o.FloorTerrain) && o.FloorTerrain[i] {
				w.TerrainStaySum += stay
				w.TerrainStayCount++
			} else {
				w.PlainStaySum += stay
				w.PlainStayCount++
			}
		}
		if died {
			switch cause {
			case DamageSourceWind:
				if o.EndTerrain {
					w.TerrainWind++
				} else {
					w.PlainWind++
				}
			case DamageSourceStarvation:
				if o.EndTerrain {
					w.TerrainStarve++
				} else {
					w.PlainStarve++
				}
			}
			cash[name] = append(cash[name], o.EndCash)
		}
		w.CrackedBroken += o.CrackedBroken
		w.HiddenFound += o.HiddenFound
		w.ShopFloors += o.ShopFloors
		w.ShopEntries += o.ShopEntries
		for _, d := range o.Descents {
			w.Descents++
			w.DescentHPPctSum += d.HPPct
			w.DescentSatietySum += d.Satiety
			if d.Wait >= 0 {
				w.DescentWaitSum += d.Wait
				w.DescentWaitCount++
			}
		}
	}
	for name, list := range cash {
		sort.Ints(list)
		stats[name].DeathCashMedian = list[len(list)/2]
	}
	return stats
}

func avgOf(sum, n int) float64 { return float64(sum) / float64(max(1, n)) }

// renderWave11Stats は段7〜10の要素の表を書く。
func renderWave11Stats(b *strings.Builder, s simSummary) {
	if len(s.Wave11) == 0 {
		return
	}
	names := sortedKeys(s.Wave11)

	fmt.Fprintf(b, "\n## モンスターハウス（自然生成、性格別、全難易度）\n\n")
	fmt.Fprintf(b, "入室は各階で初めて入ったときだけ数える。滞在は入ってから部屋を出る・階を離れる・冒険が終わるまで。\n\n")
	fmt.Fprintf(b, "| 性格 | ある階に着いた | 入った | 入ったときの HP | 入ったときの所持品 | 平均の滞在ターン | 中で死亡 |\n|---|---:|---:|---:|---:|---:|---:|\n")
	for _, name := range names {
		w := s.Wave11[name]
		fmt.Fprintf(b, "| %s | %d | %d | %.0f%% | %.1f | %.1f | %d |\n", name, w.HouseFloors, w.HouseEntries,
			avgOf(w.HouseHPPctSum, w.HouseEntries), avgOf(w.HouseItemsSum, w.HouseEntries), avgOf(w.HouseStaySum, w.HouseEntries), w.HouseDeaths)
	}

	fmt.Fprintf(b, "\n## 地形（性格別、全難易度）\n\n")
	fmt.Fprintf(b, "地形のある階 = 水路・割れた壁・隠し通路のどれかがある階。滞在は降りた階だけ。割れた壁は BOT 以外（爆発など）で壊れた分、隠し通路はカードなどで見つかった分も含む。\n\n")
	fmt.Fprintf(b, "| 性格 | 地形のある階の滞在 | 無い階の滞在 | 突風（ある／無い） | 餓死（ある／無い） | 割れた壁を壊した | 隠し通路を見つけた |\n|---|---:|---:|---:|---:|---:|---:|\n")
	for _, name := range names {
		w := s.Wave11[name]
		fmt.Fprintf(b, "| %s | %.0f (%d階) | %.0f (%d階) | %d／%d | %d／%d | %d | %d |\n", name,
			avgOf(w.TerrainStaySum, w.TerrainStayCount), w.TerrainStayCount, avgOf(w.PlainStaySum, w.PlainStayCount), w.PlainStayCount,
			w.TerrainWind, w.PlainWind, w.TerrainStarve, w.PlainStarve, w.CrackedBroken, w.HiddenFound)
	}

	fmt.Fprintf(b, "\n## 店の売買（性格別、全難易度）\n\n| 性格 | 店のある階に着いた | 店に入った | 買った品 | 払った額 | 売った回数 | 売った額 | 死亡時の所持金（中央値） |\n|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, name := range names {
		w := s.Wave11[name]
		fmt.Fprintf(b, "| %s | %d | %d | %d | %d | %d | %d | %d |\n", name, w.ShopFloors, w.ShopEntries, w.Bought, w.Spent, w.Sold, w.Earned, w.DeathCashMedian)
	}

	fmt.Fprintf(b, "\n## 探索（階段で降りた階、性格別、全難易度）\n\n| 性格 | 降りた回数 | 階段を見つけてから降りるまで | 降りる直前の HP | 降りる直前の満腹度 |\n|---|---:|---:|---:|---:|\n")
	for _, name := range names {
		w := s.Wave11[name]
		fmt.Fprintf(b, "| %s | %d | %.0f | %.0f%% | %.1f |\n", name, w.Descents, avgOf(w.DescentWaitSum, w.DescentWaitCount),
			avgOf(w.DescentHPPctSum, w.Descents), avgOf(w.DescentSatietySum, w.Descents))
	}

	// 道具の表は expert（バランス評価に使う性格）だけ。expert がいなければ全性格の合計。
	label, w := "expert", s.Wave11["expert"]
	if w == nil {
		label, w = "全性格", &simWave11Stats{}
		for _, name := range names {
			o := s.Wave11[name]
			for _, pair := range []struct {
				dst *map[int]int
				src map[int]int
			}{
				{&w.Acquired, o.Acquired}, {&w.Used, o.Used}, {&w.Thrown, o.Thrown}, {&w.PotInsert, o.PotInsert}, {&w.HeldAtDeath, o.HeldAtDeath},
			} {
				for id, n := range pair.src {
					w.add(pair.dst, id, n)
				}
			}
		}
	}
	ids := wave11ItemIDs()
	sort.SliceStable(ids, func(i, j int) bool {
		if w.Acquired[ids[i]] != w.Acquired[ids[j]] {
			return w.Acquired[ids[i]] > w.Acquired[ids[j]]
		}
		return ids[i] < ids[j]
	})
	fmt.Fprintf(b, "\n## 段8の薬・杖と段9の壺の使い方（%s、全難易度。手に入れた数の多い順）\n\n", label)
	fmt.Fprintf(b, "手に入れた = 所持品に加わった数（拾う・足元の交換・壺から出す）。使った = 飲む・読む・振る（杖は振った回数）。壺は「壺に入れた」が使ったにあたる。持ったまま死んだ = 死亡時の所持品（壺の中は数えない）。\n\n")
	fmt.Fprintf(b, "| ID | 名前 | 手に入れた | 使った | 投げた | 壺に入れた | 持ったまま死んだ |\n|---:|---|---:|---:|---:|---:|---:|\n")
	for _, id := range ids {
		fmt.Fprintf(b, "| %d | %s | %d | %d | %d | %d | %d |\n", id, itemTemplates[id].Name, w.Acquired[id], w.Used[id], w.Thrown[id], w.PotInsert[id], w.HeldAtDeath[id])
	}
}

// renderWave11Diff はベースラインとの差分に段7〜10の主な数を書く（ベースラインに無ければ書かない）。
func renderWave11Diff(b *strings.Builder, s simSummary, baseline *simSummary) {
	if len(s.Wave11) == 0 || len(baseline.Wave11) == 0 {
		return
	}
	newItemUses := func(w *simWave11Stats) int {
		n := 0
		for _, m := range []map[int]int{w.Used, w.Thrown, w.PotInsert} {
			for _, v := range m {
				n += v
			}
		}
		return n
	}
	fmt.Fprintf(b, "\n| 性格 | モンスターハウスで死亡 | 地形のある階の滞在 | 突風 | 新しい道具を使った回数 | 店で買った品 |\n|---|---|---|---|---|---|\n")
	for _, name := range sortedKeys(s.Wave11) {
		cur, base := s.Wave11[name], baseline.Wave11[name]
		if base == nil {
			continue
		}
		fmt.Fprintf(b, "| %s | %d → %d | %.0f → %.0f | %d → %d | %d → %d | %d → %d |\n", name,
			base.HouseDeaths, cur.HouseDeaths, avgOf(base.TerrainStaySum, base.TerrainStayCount), avgOf(cur.TerrainStaySum, cur.TerrainStayCount),
			base.TerrainWind+base.PlainWind, cur.TerrainWind+cur.PlainWind, newItemUses(base), newItemUses(cur), base.Bought, cur.Bought)
	}
}

// simRunInfo は計測した時点のコミットと TestSimBatch のフラグ（ベースラインとレポートに残す）。
func simRunInfo() (commit, flags string) {
	commit = "unknown"
	if out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output(); err == nil {
		commit = strings.TrimSpace(string(out))
		if status, err := exec.Command("git", "status", "--porcelain", "--untracked-files=no").Output(); err == nil && len(strings.TrimSpace(string(status))) > 0 {
			commit += "（未コミットの変更あり）"
		}
	}
	var args []string
	for _, a := range os.Args[1:] {
		if strings.HasPrefix(a, "-sim.") || strings.HasPrefix(a, "-bot.") {
			args = append(args, a)
		}
	}
	return commit, strings.Join(args, " ")
}

// simBaselineLabel はレポートに書く、比較したベースラインの説明（TestSimBatch が読んだときに設定）。
var simBaselineLabel = "ベースライン"

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

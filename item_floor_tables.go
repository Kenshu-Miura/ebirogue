package main

import (
	"fmt"
	"math"
	"sort"
)

// 階層別のアイテム・罠の出現テーブル（wave9-A）。
//
// 床の道具は「カテゴリ → カテゴリ内の道具」の順に重みで抽選する（乱数を2回使う）。
//   - カテゴリの重みは階の帯ごとの表 itemCategoryBandWeights。
//   - 道具ごとの重みと出る階は itemFloorRules（書いていない道具は全階・重み1）。
//   - 罠は帯ごとの重みの表 trapBandWeights。
//   - 難易度ごとの補正は difficultySpawnTunings（すべて 1.0 ＝補正なし。wave11 で値を入れる）。
//
// ここはタグなしの純粋な処理だけを置く。テンプレートの表を引く部分は items.go・maptraps.go。
// つまみの使い方は docs/agents/done/wave9-A.md。

// numSpawnBands は階の帯の数（spawnFloorBands の長さ）。
const numSpawnBands = 7

// spawnFloorBands は出現テーブルの階の帯。敵の deepFloorSpawnBands（9〜12・13〜16・17〜20・21〜25・26〜30）に合わせ、
// 1〜8階は浅い1〜3階と4〜8階に分ける。
var spawnFloorBands = [numSpawnBands]struct{ From, To int }{
	{1, 3}, {4, 8}, {9, 12}, {13, 16}, {17, 20}, {21, 25}, {26, 30},
}

// spawnBandIndex は階の帯の添字を返す。1階より浅い階は最初の帯、30階より深い階は最後の帯として扱う。
func spawnBandIndex(floor int) int {
	for i, band := range spawnFloorBands {
		if floor <= band.To {
			return i
		}
	}
	return numSpawnBands - 1
}

// spawnBandLabel は帯の表示名（例: "1〜3階"）。BOT のレポートで使う。
func spawnBandLabel(floor int) string {
	band := spawnFloorBands[spawnBandIndex(floor)]
	return fmt.Sprintf("%d〜%d階", band.From, band.To)
}

// itemCategoryBandWeights は床に出るアイテムのカテゴリ（ItemTemplate.ItemType）ごとの、帯別の重み。
// 同じ帯の全カテゴリの合計に対する割合がそのカテゴリの出現率になる。
// 初期値は wave9-A の導入前（全階で同じ重み）と同じ。導入前の値は 2026-10-14（アイテム ID 0〜71）に
// 全テンプレートが同じ重みだったときの割合で、カードはパワーアップのカードの重み4を含めて31、指輪は wave4-A で1→3。
// 割合を変えるときは TestSimBatch の「床の道具（階の帯別）」と餓死の数をベースラインと比べる。
var itemCategoryBandWeights = map[string][numSpawnBands]int{
	//            1〜3 4〜8 9〜12 13〜16 17〜20 21〜25 26〜30
	"Card":      {31, 31, 31, 31, 31, 31, 31},
	"Armor":     {14, 14, 14, 14, 14, 14, 14},
	"Weapon":    {12, 12, 12, 12, 12, 12, 12},
	"Potion":    {7, 7, 7, 7, 7, 7, 7},
	"Food":      {4, 4, 4, 4, 4, 4, 4},
	"Cane":      {2, 2, 2, 2, 2, 2, 2},
	"Accessory": {3, 3, 3, 3, 3, 3, 3},
	"Arrow":     {1, 1, 1, 1, 1, 1, 1},
	"Money":     {1, 1, 1, 1, 1, 1, 1},
	"Pot":       {1, 1, 1, 1, 1, 1, 1},
	"Trap":      {1, 1, 1, 1, 1, 1, 1},
}

// itemCategoryWeight は floor 階でのカテゴリの重み（表に無いカテゴリは0＝床に出ない）。
func itemCategoryWeight(category string, floor int) int {
	weights, ok := itemCategoryBandWeights[category]
	if !ok {
		return 0
	}
	return weights[spawnBandIndex(floor)]
}

// itemFloorRule は道具1種類の出現の規則。
type itemFloorRule struct {
	From   int  // 出始める階（0 は1階から）
	Until  int  // 出なくなる階（この階から出ない。0 は最後まで出る）
	Weight int  // カテゴリ内での重み（0 は1）
	Strong bool // 強い道具（difficultySpawnTuning.StrongItemWeight を掛ける）
	Heal   bool // 回復薬（difficultySpawnTuning.FoodHealWeight を掛ける。食料はカテゴリごと掛ける）
}

// powerUpCardSpawnWeight はパワーアップのカード（ID 43）のカード内での出現の重み（ほかのカードは1）。
// 毒（毒ヘビ系・毒矢の罠・腐った食料）で下がったパワーを戻せるよう多めに出す。
// 根拠は docs/agents/early-game-analysis.md の「採用した変更」。
const powerUpCardSpawnWeight = 4

// potionSpawnWeight は薬の通常の重み。無敵の薬だけ1にして出現を1/4に抑える（wave8-B）。
const potionSpawnWeight = 4

// basicCardSpawnWeight はあかり・真空斬りのカードの重み（ほかのカードの2倍）。
// どちらも浅い階から役に立つ基本のカードなので、カードの中で少し多めに出す（wave9-A で決めた値）。
const basicCardSpawnWeight = 2

// itemFloorRules はテンプレート ID ごとの出現の規則。書いていない道具は全階・重み1。
// 強い道具（Strong）は、全滅・ジェノサイド・白紙・聖域のカード、無敵の薬、wave9-C の合成・強化の壺の出始める階を遅らせる。
// 基礎の攻撃力・防御力が高い武具も Strong だが、出始める階は遅らせない（理由は武器の行）。強化値は生成時の抽選で、階では変えない。
// テンプレートが無い ID は読み飛ばす（TestItemFloorRulesSanity が検出する）。
var itemFloorRules = map[int]itemFloorRule{
	// 薬
	2:   {Weight: potionSpawnWeight, Heal: true}, // ミンティア
	3:   {Weight: potionSpawnWeight, Heal: true}, // すごいミンティア
	28:  {Weight: potionSpawnWeight, Heal: true}, // 大粒ミンティア
	17:  {Weight: potionSpawnWeight},
	18:  {Weight: potionSpawnWeight},
	19:  {Weight: potionSpawnWeight},
	29:  {Weight: potionSpawnWeight},
	205: {Weight: potionSpawnWeight}, // 段8の能力値系の薬（wave8-A）
	206: {Weight: potionSpawnWeight},
	207: {Weight: potionSpawnWeight},
	208: {Weight: potionSpawnWeight},
	209: {Weight: potionSpawnWeight},
	215: {Weight: potionSpawnWeight}, // 段8の特殊系の薬（wave8-B）
	216: {Weight: potionSpawnWeight},
	217: {From: 4, Weight: 1, Strong: true}, // 無敵の薬
	218: {Weight: potionSpawnWeight},

	// カード
	30: {Weight: basicCardSpawnWeight},   // あかりのカード
	31: {Weight: basicCardSpawnWeight},   // 真空斬りのカード
	43: {Weight: powerUpCardSpawnWeight}, // パワーアップのカード
	53: {From: 4, Strong: true},          // 白紙のカード（好きなカードを書ける）
	55: {From: 4, Strong: true},          // 聖域のカード
	54: {From: 9, Strong: true},          // ジェノサイドのカード
	56: {From: 9, Strong: true},          // 全滅のカード

	// 武器（基礎の攻撃力が8以上）。出始める階は遅らせない: 9階・13階からにすると expert の中級のクリア率が
	// 45.4%→36.2%（各500冒険）、4階からでも 40.6% に下がった（序盤に拾う強い武具が中盤の生存を支えている）。
	// 強い道具の印だけ付け、難易度の補正（StrongItemWeight）で増減できるようにする。
	4:   {Strong: true}, // 伝説の剣（8）
	127: {Strong: true}, // 剛剣（10）
	146: {Strong: true}, // 火迅の太刀（12）
	60:  {Strong: true}, // 使い捨ての大剣（12。使うたびに弱る）

	// 盾（基礎の防御力が8以上）。武器と同じ理由で出始める階は遅らせない。
	5:   {Strong: true}, // 光の角（8）
	149: {Strong: true}, // 大海の盾（10）
	69:  {Strong: true}, // 重甲の盾（12。満腹度の減りが倍になる代価がある）
	70:  {Strong: true}, // 使い捨ての盾（15。被弾ごとに弱る）

	// 壺（wave9-C。テンプレートが無いうちは読み飛ばす）
	225: {From: 4, Strong: true}, // 合成の壺
	227: {From: 4, Strong: true}, // 強化の壺
}

// itemFloorRuleFor は ID の規則を返す（書いていない ID は全階・重み1）。
func itemFloorRuleFor(id int) itemFloorRule {
	rule := itemFloorRules[id]
	if rule.Weight == 0 {
		rule.Weight = 1
	}
	return rule
}

// appearsOnFloor は規則の道具が floor 階に出るか。
func (r itemFloorRule) appearsOnFloor(floor int) bool {
	if r.From > 0 && floor < r.From {
		return false
	}
	if r.Until > 0 && floor >= r.Until {
		return false
	}
	return true
}

// itemFloorWeight は floor 階でのカテゴリ内の重み（出ない階は0）。難易度の補正は含まない。
func itemFloorWeight(id, floor int) int {
	rule := itemFloorRuleFor(id)
	if !rule.appearsOnFloor(floor) {
		return 0
	}
	return rule.Weight
}

// difficultySpawnTuning は難易度ごとの床の道具・罠の補正（wave11 のクリア率の調整用のつまみ）。
// 倍率はどれも 1.0 で補正なし。
type difficultySpawnTuning struct {
	FoodHealWeight   float64 // 食料のカテゴリの重みと、回復薬（itemFloorRule.Heal）のカテゴリ内の重みの倍率
	StrongItemWeight float64 // 強い道具（itemFloorRule.Strong）のカテゴリ内の重みの倍率
	ItemCount        float64 // 1階あたりの床の道具の数（floorItemBaseCount）の倍率
	TrapCount        float64 // 1階あたりの罠の数（floorTrapMinCount〜floorTrapMaxCount の抽選結果）の倍率
}

// neutralSpawnTuning は補正なし。
var neutralSpawnTuning = difficultySpawnTuning{FoodHealWeight: 1, StrongItemWeight: 1, ItemCount: 1, TrapCount: 1}

// difficultySpawnTunings は難易度ごとの補正。wave9-A ではすべて補正なしで入れる。
var difficultySpawnTunings = map[Difficulty]difficultySpawnTuning{
	DifficultyBeginner:     {FoodHealWeight: 1.0, StrongItemWeight: 1.0, ItemCount: 1.0, TrapCount: 1.0},
	DifficultyIntermediate: {FoodHealWeight: 1.0, StrongItemWeight: 1.0, ItemCount: 1.0, TrapCount: 1.0},
	DifficultyAdvanced:     {FoodHealWeight: 1.0, StrongItemWeight: 1.0, ItemCount: 1.0, TrapCount: 1.0},
}

// spawnTuningFor は難易度の補正を返す（表に無い難易度は補正なし）。
func spawnTuningFor(d Difficulty) difficultySpawnTuning {
	if tuning, ok := difficultySpawnTunings[d]; ok {
		return tuning
	}
	return neutralSpawnTuning
}

// 1階あたりの床の道具と罠の数（難易度の補正の前）。
const (
	floorItemBaseCount = 10
	floorTrapMinCount  = 5
	floorTrapMaxCount  = 10
)

// scaledCount は数に倍率を掛けて四捨五入する（0未満にはしない）。
func scaledCount(n int, multiplier float64) int {
	return max(0, int(math.Round(float64(n)*multiplier)))
}

// floorItemCount は1階あたりの床の道具の数。
func (t difficultySpawnTuning) floorItemCount() int {
	return scaledCount(floorItemBaseCount, t.ItemCount)
}

// floorTrapCount は抽選した罠の数（floorTrapMinCount〜floorTrapMaxCount）に倍率を掛ける。
func (t difficultySpawnTuning) floorTrapCount(rolled int) int {
	return scaledCount(rolled, t.TrapCount)
}

// spawnWeightScale は倍率を整数の重みで表すための拡大率。倍率 1.0 では割合は表の重みと同じ。
const spawnWeightScale = 100

// scaledSpawnWeight は重みに倍率を掛けた整数の重み（spawnWeightScale 倍）。
// 重みが正で倍率が正なら、小さくしすぎても1は残す（倍率で道具が消えないように）。
func scaledSpawnWeight(weight int, multiplier float64) int {
	if weight <= 0 || multiplier <= 0 {
		return 0
	}
	return max(1, int(math.Round(float64(weight)*multiplier*spawnWeightScale)))
}

// floorSpawnEntry はカテゴリ内の道具1つ。
type floorSpawnEntry struct {
	ID     int
	Weight int
	Strong bool
	Heal   bool
}

// floorSpawnCategory は1つの階のカテゴリ1つ。
type floorSpawnCategory struct {
	Name    string
	Weight  int
	Entries []floorSpawnEntry
}

// foodSpawnCategory は FoodHealWeight をカテゴリの重みに掛けるカテゴリ。
const foodSpawnCategory = "Food"

// buildFloorSpawnCategories は floor 階の抽選表を作る。ids は床の抽選の対象（昇順）、categoryOf は ID のカテゴリ。
// 重みが0のカテゴリ・道具と、道具が1つも無いカテゴリは入れない。並びはカテゴリ名の昇順。
func buildFloorSpawnCategories(ids []int, categoryOf func(int) string, floor int) []floorSpawnCategory {
	members := map[string][]floorSpawnEntry{}
	for _, id := range ids {
		weight := itemFloorWeight(id, floor)
		if weight <= 0 {
			continue
		}
		rule := itemFloorRuleFor(id)
		name := categoryOf(id)
		members[name] = append(members[name], floorSpawnEntry{ID: id, Weight: weight, Strong: rule.Strong, Heal: rule.Heal})
	}
	names := make([]string, 0, len(members))
	for name := range members {
		names = append(names, name)
	}
	sort.Strings(names)
	categories := []floorSpawnCategory{}
	for _, name := range names {
		weight := itemCategoryWeight(name, floor)
		if weight <= 0 {
			continue
		}
		categories = append(categories, floorSpawnCategory{Name: name, Weight: weight, Entries: members[name]})
	}
	return categories
}

// categoryWeightWith は補正込みのカテゴリの重み。
func (c floorSpawnCategory) categoryWeightWith(t difficultySpawnTuning) int {
	multiplier := 1.0
	if c.Name == foodSpawnCategory {
		multiplier = t.FoodHealWeight
	}
	return scaledSpawnWeight(c.Weight, multiplier)
}

// entryWeightWith は補正込みの道具の重み。
func (e floorSpawnEntry) entryWeightWith(t difficultySpawnTuning) int {
	multiplier := 1.0
	if e.Strong {
		multiplier *= t.StrongItemWeight
	}
	if e.Heal {
		multiplier *= t.FoodHealWeight
	}
	return scaledSpawnWeight(e.Weight, multiplier)
}

// pickFloorSpawnID はカテゴリを重みで選び、そのカテゴリ内から道具を重みで選ぶ（乱数を2回使う）。
// 選べなければ fallback を返す。
func pickFloorSpawnID(categories []floorSpawnCategory, tuning difficultySpawnTuning, intn func(int) int, fallback int) int {
	total := 0
	for _, category := range categories {
		total += category.categoryWeightWith(tuning)
	}
	if total <= 0 {
		return fallback
	}
	r := intn(total)
	for _, category := range categories {
		w := category.categoryWeightWith(tuning)
		if r >= w {
			r -= w
			continue
		}
		entryTotal := 0
		for _, entry := range category.Entries {
			entryTotal += entry.entryWeightWith(tuning)
		}
		if entryTotal <= 0 {
			return fallback
		}
		r = intn(entryTotal)
		for _, entry := range category.Entries {
			w := entry.entryWeightWith(tuning)
			if r < w {
				return entry.ID
			}
			r -= w
		}
		return fallback
	}
	return fallback
}

// trapBandWeights は罠 ID ごとの帯別の重み（床生成とワナのカードの抽選）。0 の帯には出ない。
// 9〜16階は導入前（睡眠ガスだけ2倍、ほかは同じ）と同じ割合。浅い階はダメージの小さい罠（木の矢）を多く、
// 大型地雷・鉄の矢を出さず、深い階ほど大型地雷・鉄の矢・落石・回復封じが増える。
// ボス階と30階で落とし穴を除く規則（availableTrapIDs）はこの表の後にかける。新しい罠は行を足す。
var trapBandWeights = map[int][numSpawnBands]int{
	//                1〜3 4〜8 9〜12 13〜16 17〜20 21〜25 26〜30
	0:  {4, 4, 4, 4, 4, 4, 4}, // 睡眠ガスの罠
	1:  {2, 2, 2, 2, 2, 2, 2}, // 毒矢の罠
	2:  {2, 2, 2, 2, 2, 2, 2}, // 鈍足の罠
	3:  {2, 2, 2, 2, 2, 2, 2}, // 地雷
	4:  {2, 2, 2, 2, 2, 2, 2}, // サビの罠
	5:  {2, 2, 2, 2, 2, 2, 2}, // ワープの罠
	6:  {2, 2, 2, 2, 2, 2, 2}, // 落とし穴
	7:  {2, 2, 2, 2, 2, 2, 2}, // トラバサミ
	8:  {4, 3, 2, 2, 1, 1, 1}, // 木の矢の罠
	9:  {0, 1, 2, 2, 3, 3, 3}, // 鉄の矢の罠
	10: {1, 1, 2, 2, 3, 3, 3}, // 落石の罠
	11: {0, 1, 2, 2, 2, 3, 3}, // 大型地雷
	12: {2, 2, 2, 2, 2, 2, 2}, // 装備はずしの罠
	13: {2, 2, 2, 2, 2, 2, 2}, // 腐敗の罠
	14: {2, 2, 2, 2, 2, 2, 2}, // 散乱の罠
	15: {1, 1, 2, 2, 3, 3, 3}, // 回復封じの罠
}

// trapFloorWeight は floor 階での罠の重み（表に無い罠は0）。
func trapFloorWeight(id, floor int) int {
	weights, ok := trapBandWeights[id]
	if !ok {
		return 0
	}
	return weights[spawnBandIndex(floor)]
}

// pickWeightedTrapID は ids（昇順）を floor 階の重みで1つ選ぶ（乱数を1回使う）。選べなければ fallback。
func pickWeightedTrapID(ids []int, floor int, intn func(int) int, fallback int) int {
	total := 0
	for _, id := range ids {
		total += trapFloorWeight(id, floor)
	}
	if total <= 0 {
		return fallback
	}
	r := intn(total)
	for _, id := range ids {
		w := trapFloorWeight(id, floor)
		if r < w {
			return id
		}
		r -= w
	}
	return fallback
}

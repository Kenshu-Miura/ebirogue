package main

// インベントリのカテゴリ別ソート・絞り込み・名前入力の純粋ロジック。
// ゲーム本体に依存しないためテストビルドでも利用できる（ビルドタグなし）。

// ItemCategory はインベントリのソート・絞り込みで使うアイテムの分類
type ItemCategory int

const (
	CategoryAll ItemCategory = iota // 絞り込みなし（全て表示）
	CategoryWeapon
	CategoryArmor
	CategoryArrow
	CategoryAccessory
	CategoryCane
	CategoryPot
	CategoryFood
	CategoryPotion
	CategoryCard
	CategoryTrap
	CategoryMoney
	CategoryOther
	categoryCount
)

// categoryLabel はカテゴリの表示名を返す
func categoryLabel(c ItemCategory) string {
	switch c {
	case CategoryAll:
		return "全て"
	case CategoryWeapon:
		return "武器"
	case CategoryArmor:
		return "盾"
	case CategoryArrow:
		return "矢"
	case CategoryAccessory:
		return "アクセサリー"
	case CategoryCane:
		return "杖"
	case CategoryPot:
		return "壺"
	case CategoryFood:
		return "食料"
	case CategoryPotion:
		return "薬"
	case CategoryCard:
		return "カード"
	case CategoryTrap:
		return "罠"
	case CategoryMoney:
		return "お金"
	default:
		return "その他"
	}
}

// nextFilter は絞り込みカテゴリを次へ切り替える。
// present に存在するカテゴリだけを巡回し、最後まで行ったら CategoryAll へ戻る。
func nextFilter(current ItemCategory, present []bool) ItemCategory {
	for c := int(current) + 1; c < len(present) && c < int(categoryCount); c++ {
		if present[c] {
			return ItemCategory(c)
		}
	}
	return CategoryAll
}

// filteredIndices は categories のうち filter に一致するインデックスを返す。
// filter が CategoryAll の場合は全てのインデックスを返す。
func filteredIndices(categories []ItemCategory, filter ItemCategory) []int {
	indices := make([]int, 0, len(categories))
	for i, c := range categories {
		if filter == CategoryAll || c == filter {
			indices = append(indices, i)
		}
	}
	return indices
}

// moveSelection は絞り込み後のリスト内で delta だけ移動した実インデックスを返す。
// current がリストに無い場合は先頭へ移動する。移動先はリストの端で止まる。
func moveSelection(indices []int, current, delta int) int {
	if len(indices) == 0 {
		return current
	}
	pos := -1
	for i, idx := range indices {
		if idx == current {
			pos = i
			break
		}
	}
	if pos == -1 {
		return indices[0]
	}
	pos += delta
	if pos < 0 {
		pos = 0
	}
	if pos >= len(indices) {
		pos = len(indices) - 1
	}
	return indices[pos]
}

// --- 未識別アイテムの名前入力 ---

// maxItemNameLength は任意名の最大文字数
const maxItemNameLength = 6

// kanaGridColumns は名前入力グリッドの1行あたりの文字数
const kanaGridColumns = 10

// kanaChars は名前入力に使える文字の一覧（グリッド表示順）
var kanaChars = []rune("あいうえおかきくけこ" +
	"さしすせそたちつてと" +
	"なにぬねのはひふへほ" +
	"まみむめもやゆよらり" +
	"るれろわをんーっゃゅ" +
	"ょぁぃぅぇぉがぎぐげ" +
	"ござじずぜぞだぢづで" +
	"どばびぶべぼぱぴぷぺ" +
	"ぽ")

// NameInput は入力中の名前を保持する
type NameInput struct {
	Runes []rune
}

// Append は文字を末尾へ追加する。最大文字数を超える場合は追加しない
func (n *NameInput) Append(r rune) bool {
	if len(n.Runes) >= maxItemNameLength {
		return false
	}
	n.Runes = append(n.Runes, r)
	return true
}

// Backspace は末尾の1文字を削除する。空の場合は false を返す
func (n *NameInput) Backspace() bool {
	if len(n.Runes) == 0 {
		return false
	}
	n.Runes = n.Runes[:len(n.Runes)-1]
	return true
}

// String は入力中の名前を文字列で返す
func (n *NameInput) String() string {
	return string(n.Runes)
}

// moveGridCursor はグリッドのカーソルを delta だけ移動する。
// 範囲外へ出る移動は無視して現在位置を返す。
func moveGridCursor(current, delta, total int) int {
	next := current + delta
	if next < 0 || next >= total {
		return current
	}
	return next
}

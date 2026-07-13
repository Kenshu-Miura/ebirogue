package main

// ヘルプウィンドウの1ページに表示する行数
const helpPageSize = 15

// HelpPage はヘルプ画面の1ページ分の内容を保持する
type HelpPage struct {
	Title string
	Lines []string
}

// helpPages はヘルプ画面の全ページ（操作方法・状態異常）を返す
func helpPages() []HelpPage {
	return []HelpPage{
		{
			Title: "操作方法",
			Lines: []string{
				"【移動】",
				" 矢印キー：移動（Shift+矢印：斜め移動）",
				" X+矢印キー：ダッシュ（入口や分岐で停止）",
				" A+矢印キー：その場で向きを変える",
				"【行動】",
				" Z：向いている方向へ攻撃",
				" X+Z：足踏み（その場でターンを進める）",
				" D：装備中の矢を撃つ",
				" スペース：隣接する扉を開く",
				"【ウィンドウ】",
				" C：メニューを開く／全て閉じる",
				" Z：決定  X：キャンセル・戻る",
				" L：メッセージ履歴  H：このヘルプ",
				"【道具ウィンドウ】",
				" F：カテゴリ絞り込み  S：整頓",
				" N：未識別の道具へ名前を付ける",
			},
		},
		{
			Title: "状態異常",
			Lines: []string{
				"毒　　：行動するたびにHPが2ずつ減る",
				"鈍足　：動きが遅くなり、敵が2回行動してくる",
				"倍速　：通常の2倍の速さで行動できる",
				"睡眠　：しばらくの間行動できない",
				"混乱　：移動や攻撃の方向がランダムになる",
				"目潰し：敵や道具、ミニマップが見えなくなる",
				"金縛り：動けなくなる。攻撃を受けると解除される",
				"封印　：特殊能力が使えなくなる",
				"",
				"ターン数のある状態異常は時間経過で回復する。",
				"現在の状態異常は画面左の「状態」欄で確認できる。",
			},
		},
	}
}

// helpMaxScroll は総行数 total と表示行数 count からスクロールできる最大量を返す
func helpMaxScroll(total, count int) int {
	if total <= count {
		return 0
	}
	return total - count
}

// helpVisibleLines はスクロール量 scroll のときに表示する行を返す。
// scroll=0 で先頭行が表示され、範囲外の値は表示可能な範囲へ丸められる。
func helpVisibleLines(lines []string, scroll, count int) []string {
	if scroll < 0 {
		scroll = 0
	}
	if maxScroll := helpMaxScroll(len(lines), count); scroll > maxScroll {
		scroll = maxScroll
	}
	end := scroll + count
	if end > len(lines) {
		end = len(lines)
	}
	return lines[scroll:end]
}

package main

// 未識別の杖・壺・指輪の名前（冒険ごとにランダム）。
// 種類（ItemType）ごとの名前の候補から、冒険のシードで決まる並びで各テンプレートへ1つずつ割り当てる。
// ターン進行の乱数（Game.rng）を使わないので、BOT・入力の記録・中断セーブの結果を変えず、保存も要らない
// （同じシードなら再開後も同じ名前になる）。

// rngStreamUnidentifiedNames は未識別名の割り当て専用の乱数系列。
const rngStreamUnidentifiedNames uint64 = 0x756e6964 // "unid"

// unidentifiedPotColors は壺の色の名前。順番は壺の画像 img/pot_225.png〜pot_232.png の色と対応する
// （見た目の番号 i の壺は「<色>の壺」で pot_(225+i).png を使う）。
var unidentifiedPotColors = []string{"赤色", "金色", "緑色", "青緑色", "水色", "青色", "紫色", "桃色"}

// unidentifiedNamePools は種類ごとの未識別名の候補。その種類の未識別のテンプレートの数以上を用意する
// （足りないと TestUnidentifiedNamePoolsCoverTemplates が落ちる）。
var unidentifiedNamePools = map[string][]string{
	"Pot": func() []string {
		names := make([]string, len(unidentifiedPotColors))
		for i, c := range unidentifiedPotColors {
			names[i] = c + "の壺"
		}
		return names
	}(),
	"Cane": {
		"木彫りの杖", "水晶の杖", "赤珊瑚の杖", "樫の杖", "竹の杖", "流木の杖", "貝殻の杖", "骨の杖",
		"真珠の杖", "黒檀の杖", "銀の杖", "象牙の杖", "藤の杖", "白樺の杖",
	},
	"Accessory": {
		"金の指輪", "銀の指輪", "銅の指輪", "真鍮の指輪", "白金の指輪", "錫の指輪", "真珠の指輪", "珊瑚の指輪",
		"琥珀の指輪", "翡翠の指輪", "瑠璃の指輪", "水晶の指輪", "紅玉の指輪", "青玉の指輪", "緑玉の指輪", "黄玉の指輪",
		"紫水晶の指輪", "柘榴石の指輪", "月長石の指輪", "黒曜石の指輪", "桜貝の指輪", "夜光貝の指輪", "象牙の指輪", "硝子の指輪",
		"蛍石の指輪", "瑪瑙の指輪", "猫目石の指輪", "虎目石の指輪", "孔雀石の指輪", "蛋白石の指輪", "星砂の指輪", "鱗の指輪",
		"鼈甲の指輪", "黒真珠の指輪", "藍玉の指輪", "橄欖石の指輪",
	},
}

// assignUnidentifiedAppearances は種類ごとのテンプレート ID の一覧（昇順）に、名前の候補の番号を重複なく割り当てる。
// 候補の数が ID の数より少ない種類は、余った ID に割り当てない（呼び出し側はテンプレートの名前に戻す）。
func assignUnidentifiedAppearances(idsByKind map[string][]int, intn func(int) int) map[int]int {
	result := map[int]int{}
	for _, kind := range []string{"Pot", "Cane", "Accessory"} { // 乱数を使う順番を固定する
		ids := idsByKind[kind]
		pool := len(unidentifiedNamePools[kind])
		order := make([]int, pool)
		for i := range order {
			order[i] = i
		}
		for i := pool - 1; i > 0; i-- { // Fisher–Yates
			j := intn(i + 1)
			order[i], order[j] = order[j], order[i]
		}
		for i, id := range ids {
			if i < pool {
				result[id] = order[i]
			}
		}
	}
	return result
}

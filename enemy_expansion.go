//go:build !test

package main

// 追加計画（docs/agents/content-expansion-plan.md）の敵は、1系統（基本種・第2段階・第3段階）を
// monsterFamily にまとめ、担当ごとのファイル（enemies_expansion_*.go）の表に書く。
// 並列の担当が enemies.go・monster_spawn.go・main.go の同じ箇所を触らないよう、
// 定義・上位種化・画像・出現階はこのファイルの関数がまとめて登録する。

// monsterFamily は追加する敵1系統。上位種は基本種の固有能力（特殊攻撃・行動・属性など）を引き継ぐ。
type monsterFamily struct {
	Base      MonsterDefinition
	BaseImage string // 基本種の画像（例: "img/hibana_umiushi.png"）
	Second    monsterStage
	Third     monsterStage
	// Spawns は出現階と重み。基本種・第2段階・第3段階それぞれの ID を書く。
	Spawns []monsterSpawnRange
}

// monsterStage は上位種1段階の能力値と表示。
type monsterStage struct {
	ID                                  int
	Type, Name                          string
	Health, Attack, Defense, Experience int
	Image                               string
}

// monsterSpawnRange は From〜To 階の出現テーブルへ MonsterID を Weight の重みで加える。
// 9〜30階の帯（deepFloorSpawnBands）の途中で区切ってもよい（階ごとに写してから加える）。
type monsterSpawnRange struct {
	MonsterID int
	From, To  int
	Weight    int
}

// allMonsterFamilies は担当ごとの表をまとめる。段3以降の担当は新しいファイルの表をここへ1行加える。
func allMonsterFamilies() []monsterFamily {
	var families []monsterFamily
	families = append(families, expansionA1MonsterFamilies...)
	families = append(families, expansionA2MonsterFamilies...)
	families = append(families, expansionB1MonsterFamilies...)
	families = append(families, expansionB2MonsterFamilies...)
	families = append(families, expansionC1MonsterFamilies...)
	families = append(families, expansionC2MonsterFamilies...)
	families = append(families, expansionC3MonsterFamilies...)
	return families
}

// addMonsterFamilies は系統の3段階を定義テーブルへ加える（buildMonsterDefinitions から呼ぶ）。
func addMonsterFamilies(definitions map[int]MonsterDefinition, families []monsterFamily) {
	for _, family := range families {
		base := family.Base
		base.MaxHealth = base.Health
		definitions[base.ID] = base
		second := family.Second.definition(base)
		definitions[second.ID] = second
		definitions[family.Third.ID] = family.Third.definition(second)
	}
}

func (s monsterStage) definition(lower MonsterDefinition) MonsterDefinition {
	return upperMonsterDefinition(lower, s.ID, s.Type, s.Name, s.Health, s.Attack, s.Defense, s.Experience)
}

// expansionBaseMonsterIDs は追加した系統の基本種の ID 一覧を返す。
func expansionBaseMonsterIDs() []int {
	families := allMonsterFamilies()
	ids := make([]int, 0, len(families))
	for _, family := range families {
		ids = append(ids, family.Base.ID)
	}
	return ids
}

// registerMonsterFamilies は上位種化の対応と画像を登録する（定義テーブルの生成後に init から呼ぶ）。
func registerMonsterFamilies(families []monsterFamily) {
	for _, family := range families {
		MonsterLevelUpTable[family.Base.ID] = family.Second.ID
		MonsterLevelUpTable[family.Second.ID] = family.Third.ID
		upperEnemyImagePaths[family.Base.Type] = family.BaseImage
		upperEnemyImagePaths[family.Second.Type] = family.Second.Image
		upperEnemyImagePaths[family.Third.Type] = family.Third.Image
	}
}

// applyMonsterFamilySpawns は出現階の表へ系統の敵を加える。帯で共有しているスライスは階ごとに写してから加える。
func applyMonsterFamilySpawns(tables map[int][]MonsterSpawnEntry, families []monsterFamily) {
	for _, family := range families {
		for _, spawn := range family.Spawns {
			for floor := spawn.From; floor <= spawn.To; floor++ {
				entries := append([]MonsterSpawnEntry(nil), tables[floor]...)
				tables[floor] = append(entries, MonsterSpawnEntry{MonsterID: spawn.MonsterID, Weight: spawn.Weight})
			}
		}
	}
}

// updateMonsterDefinitions は ids の定義を edit で書き換える。
// 特殊攻撃・行動・撃破時の処理が MonsterDefinitions や CreateEnemyByID を（間接的にでも）参照すると
// 定義テーブルの初期化が循環してコンパイルできないので、そのときは init() でこの関数を使って3段階すべてに設定する
// （bosses.go の init と同じ方法）。
func updateMonsterDefinitions(edit func(*MonsterDefinition), ids ...int) {
	for _, id := range ids {
		definition, ok := MonsterDefinitions[id]
		if !ok {
			continue
		}
		edit(&definition)
		MonsterDefinitions[id] = definition
	}
}

func init() {
	registerMonsterFamilies(allMonsterFamilies())
}

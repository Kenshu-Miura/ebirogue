//go:build !test

package main

// expansionC2MonsterFamilies は追加計画の敵 C（アナジャコ系 111〜113・コバンザメ系 120〜122、docs/agents/wave6-B-enemies.md）。
// 担当はこの表と、同じファイルの特殊攻撃・行動・被弾時の関数だけを追加する（enemy_expansion.go がまとめて登録する）。
var expansionC2MonsterFamilies = []monsterFamily{}

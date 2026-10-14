//go:build !test

package main

// weaponItemTemplates は追加計画（docs/agents/content-expansion-plan.md）の武器（アイテム ID 126〜147）。
// 担当（wave3-A）はこの表と同じファイルの効果関数だけを追加する。itemTemplates へは mergeItemTemplates でまとめられる。
var weaponItemTemplates = map[int]ItemTemplate{}

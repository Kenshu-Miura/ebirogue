//go:build !test

package main

// ringItemTemplates は追加計画（docs/agents/content-expansion-plan.md）の指輪（アイテム ID 175〜204）。
// 担当（段2）はこの表と同じファイルの効果関数だけを追加する。itemTemplates へは mergeItemTemplates でまとめられる。
var ringItemTemplates = map[int]ItemTemplate{}

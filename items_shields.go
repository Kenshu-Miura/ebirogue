//go:build !test

package main

// shieldItemTemplates は追加計画（docs/agents/content-expansion-plan.md）の盾（アイテム ID 148〜166）。
// 担当（wave3-B）はこの表と同じファイルの効果関数だけを追加する。itemTemplates へは mergeItemTemplates でまとめられる。
var shieldItemTemplates = map[int]ItemTemplate{}

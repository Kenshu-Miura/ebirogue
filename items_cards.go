//go:build !test

package main

// cardItemTemplates は追加計画（docs/agents/content-expansion-plan.md）のカード（アイテム ID 167〜174）。
// 担当（wave3-C）はこの表と同じファイルの効果関数だけを追加する。itemTemplates へは mergeItemTemplates でまとめられる。
var cardItemTemplates = map[int]ItemTemplate{}

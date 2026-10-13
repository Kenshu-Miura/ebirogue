package main

// Difficulty は冒険の難易度。冒険開始時に選び、途中で変更しない。
type Difficulty int

const (
	DifficultyBeginner     Difficulty = iota // 初級
	DifficultyIntermediate                   // 中級
	DifficultyAdvanced                       // 上級（標準）
)

module github.com/Kenshu-Miura/ebirogue

go 1.21.3

require (
	github.com/hajimehoshi/ebiten/v2 v2.6.2
	golang.org/x/image v0.12.0
)

replace github.com/hajimehoshi/ebiten/v2 => ./ebitenstub

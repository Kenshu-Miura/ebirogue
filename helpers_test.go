package main

import "testing"

func TestMin(t *testing.T) {
	tests := []struct {
		a, b int
		want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{3, 3, 3},
	}
	for _, tt := range tests {
		if got := min(tt.a, tt.b); got != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		a, b int
		want int
	}{
		{2, 1, 2},
		{1, 2, 2},
		{3, 3, 3},
	}
	for _, tt := range tests {
		if got := max(tt.a, tt.b); got != tt.want {
			t.Errorf("max(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		x    int
		want int
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
	}
	for _, tt := range tests {
		if got := abs(tt.x); got != tt.want {
			t.Errorf("abs(%d) = %d, want %d", tt.x, got, tt.want)
		}
	}
}

func TestSign(t *testing.T) {
	tests := []struct {
		x    int
		want int
	}{
		{5, 1},
		{-2, -1},
		{0, 0},
	}
	for _, tt := range tests {
		if got := sign(tt.x); got != tt.want {
			t.Errorf("sign(%d) = %d, want %d", tt.x, got, tt.want)
		}
	}
}

package main

import "testing"

func TestGameRNGStateRoundTrip(t *testing.T) {
	r := newTurnRNG(12345)
	for i := 0; i < 10; i++ {
		r.Intn(100)
	}
	state, err := r.MarshalState()
	if err != nil {
		t.Fatal(err)
	}
	restored := newTurnRNG(0)
	if err := restored.UnmarshalState(state); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		if a, b := r.Intn(1000), restored.Intn(1000); a != b {
			t.Fatalf("draw %d: original %d, restored %d", i, a, b)
		}
	}
}

func TestRNGStreamsAreIndependent(t *testing.T) {
	same := 0
	turn, floor1, floor2 := newTurnRNG(7), newFloorRNG(7, 1), newFloorRNG(7, 2)
	for i := 0; i < 50; i++ {
		a, b, c := turn.Intn(1<<30), floor1.Intn(1<<30), floor2.Intn(1<<30)
		if a == b || b == c {
			same++
		}
	}
	if same > 2 {
		t.Fatalf("streams look correlated: %d equal draws", same)
	}
	if newFloorRNG(7, 3).Intn(1<<30) != newFloorRNG(7, 3).Intn(1<<30) {
		t.Fatal("same seed and floor must give the same sequence")
	}
}

func TestGameRandomLazilyUsesSeed(t *testing.T) {
	g1, g2 := &Game{seed: 99}, &Game{seed: 99}
	for i := 0; i < 20; i++ {
		if a, b := g1.randIntn(1000), g2.randIntn(1000); a != b {
			t.Fatalf("draw %d differs: %d vs %d", i, a, b)
		}
	}
}

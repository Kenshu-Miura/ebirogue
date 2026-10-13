package main

import (
	"reflect"
	"testing"
)

func TestVirtualInputFollowsEbitenPressRules(t *testing.T) {
	v := NewVirtualInput()
	v.Press(KeyZ)
	if v.Pressed(KeyZ) {
		t.Fatal("press must take effect on the next Update")
	}
	v.Update()
	if !v.Pressed(KeyZ) || !v.JustPressed(KeyZ) || v.PressDuration(KeyZ) != 1 {
		t.Fatal("first frame should be just pressed with duration 1")
	}
	v.Update()
	if !v.Pressed(KeyZ) || v.JustPressed(KeyZ) || v.PressDuration(KeyZ) != 2 {
		t.Fatal("held key should not be just pressed again")
	}
	v.Release(KeyZ)
	v.Update()
	if v.Pressed(KeyZ) || v.PressDuration(KeyZ) != 0 {
		t.Fatal("released key should not be pressed")
	}
	if v.Pressed(Key(-1)) || v.JustPressed(keyCount) {
		t.Fatal("out-of-range keys must be ignored")
	}
}

func TestInputRecorderReplaysSameFrames(t *testing.T) {
	live := NewVirtualInput()
	recorder := NewInputRecorder(live)
	type frameState struct {
		pressed, just [keyCount]bool
	}
	var recorded []frameState
	snapshot := func(src InputSource) frameState {
		var s frameState
		for k := Key(0); k < keyCount; k++ {
			s.pressed[k] = src.Pressed(k)
			s.just[k] = src.JustPressed(k)
		}
		return s
	}
	script := map[int]func(){
		1: func() { live.Press(KeyRight) },
		3: func() { live.Release(KeyRight); live.Press(KeyZ, KeyShift) },
		4: func() { live.Release(KeyZ) },
		7: func() { live.ReleaseAll() },
	}
	for frame := 1; frame <= 10; frame++ {
		if f, ok := script[frame]; ok {
			f()
		}
		recorder.Update()
		recorded = append(recorded, snapshot(recorder))
	}

	replay := NewReplayInput(recorder.Events())
	for i, want := range recorded {
		replay.Update()
		if got := snapshot(replay); !reflect.DeepEqual(got, want) {
			t.Fatalf("frame %d differs on replay", i+1)
		}
	}
	if !replay.ScriptDone() {
		t.Fatal("replay should have consumed every event")
	}

	recorder.Reset()
	if len(recorder.Events()) != 0 {
		t.Fatal("Reset should clear events")
	}
}

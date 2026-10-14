package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func assertUniqueBindings(t *testing.T, bindings map[string]string) {
	t.Helper()
	seen := map[string]bool{}
	if len(bindings) != len(keyRoles) {
		t.Fatalf("binding count: %d", len(bindings))
	}
	for logical, physical := range bindings {
		if !validPhysicalKey(physical) || seen[physical] {
			t.Fatalf("invalid/duplicate %s=%s", logical, physical)
		}
		seen[physical] = true
	}
}

func TestKeyBindingDefaultsAndSwap(t *testing.T) {
	defaults := defaultKeyBindings()
	for k := range keyRoles {
		if defaults[Key(k).String()] != Key(k).String() {
			t.Fatal("default changed")
		}
	}
	bindings := bindPhysicalKey(defaults, KeyZ, "X")
	if bindings["Z"] != "X" || bindings["X"] != "Z" {
		t.Fatal(bindings)
	}
	bindings = bindPhysicalKey(bindings, KeyUp, "7")
	if bindings["Up"] != "7" || defaults["Up"] != "Up" {
		t.Fatal("binding mutates input or fails assignment")
	}
	assertUniqueBindings(t, bindings)
	for _, invalid := range []string{"Enter", "Escape", "F1", "F12", "ShiftLeft", "unknown"} {
		if got := bindPhysicalKey(bindings, KeyZ, invalid); !reflect.DeepEqual(got, bindings) {
			t.Fatalf("accepted %s", invalid)
		}
	}
	if got := bindPhysicalKey(bindings, KeyEnter, "Q"); !reflect.DeepEqual(got, bindings) {
		t.Fatal("Enter changed")
	}
}

func TestKeyBindingSettingsCorrection(t *testing.T) {
	cases := []struct {
		name, json string
		want       map[string]string
	}{
		{"old", `{}`, defaultKeyBindings()},
		{"null", `{"keyBindings":null}`, defaultKeyBindings()},
		{"unknown", `{"keyBindings":{"Z":"Escape","Unknown":"Q"}}`, defaultKeyBindings()},
		{"duplicate", `{"keyBindings":{"Z":"Q","X":"Q"}}`, defaultKeyBindings()},
		{"cascade", `{"keyBindings":{"Z":"X","X":"C","C":"unknown"}}`, defaultKeyBindings()},
		{"swap", `{"keyBindings":{"Z":"X","X":"Z"}}`, bindPhysicalKey(nil, KeyZ, "X")},
		{"partial", `{"keyBindings":{"Z":"Q"}}`, bindPhysicalKey(nil, KeyZ, "Q")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			settings, err := decodeSettings([]byte(tc.json))
			if err != nil || !reflect.DeepEqual(settings.KeyBindings, tc.want) {
				t.Fatalf("%v: %v", err, settings.KeyBindings)
			}
			assertUniqueBindings(t, settings.KeyBindings)
			data, err := json.Marshal(settings)
			if err != nil {
				t.Fatal(err)
			}
			reloaded, err := decodeSettings(data)
			if err != nil || !reflect.DeepEqual(reloaded, settings) {
				t.Fatal("roundtrip failed")
			}
		})
	}
}

func TestKeyBindingHelpFollowsSettings(t *testing.T) {
	bindings := bindPhysicalKey(nil, KeyZ, "Q")
	bindings = bindPhysicalKey(bindings, KeyUp, "W")
	bindings = bindPhysicalKey(bindings, KeyX, "Tab")
	pages := helpPages(bindings)
	pageByTitle := func(title string) HelpPage {
		t.Helper()
		for _, page := range pages {
			if page.Title == title {
				return page
			}
		}
		t.Fatalf("help page %q missing", title)
		return HelpPage{}
	}
	if !strings.Contains(strings.Join(pages[0].Lines, "\n"), "Q：決定  Tab：キャンセル") {
		t.Fatal(pages[0])
	}
	if !strings.Contains(pages[0].Lines[1], "W/") {
		t.Fatal("movement guide stale")
	}
	if !strings.Contains(pageByTitle("罠").Lines[12], "Qで正面") {
		t.Fatal("trap guide stale")
	}
	if !strings.Contains(pageByTitle("難易度とクリア条件").Lines[0], "Qで開始") {
		t.Fatal("difficulty guide stale")
	}
	if !strings.Contains(pageByTitle("罠").Lines[11], "HPが1") {
		t.Fatal("non-key text changed")
	}
	if got := keyGuide(bindPhysicalKey(nil, KeyZ, "X"), "Z:決定 X:戻る"); got != "X:決定 Z:戻る" {
		t.Fatal(got)
	}
}

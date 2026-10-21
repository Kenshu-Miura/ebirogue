package main

import (
	"encoding/json"
	"testing"
)

func TestAudioSettingsCompatibility(t *testing.T) {
	for _, tc := range []struct {
		json    string
		bgm, se int
	}{
		{`{}`, 6, 6}, {`{"fullscreen":true,"showMinimap":false}`, 6, 6},
		{`{"bgmVolume":0,"seVolume":0}`, 0, 0},
		{`{"bgmVolume":10,"seVolume":3}`, 10, 3},
		{`{"bgmVolume":-1,"seVolume":100}`, 0, 10},
	} {
		settings, err := decodeSettings([]byte(tc.json))
		if err != nil {
			t.Fatal(err)
		}
		if settings.BGMVolume != tc.bgm || settings.SEVolume != tc.se {
			t.Fatalf("%s: volumes=%d,%d", tc.json, settings.BGMVolume, settings.SEVolume)
		}
		data, err := json.Marshal(settings)
		if err != nil {
			t.Fatal(err)
		}
		again, err := decodeSettings(data)
		if err != nil {
			t.Fatal(err)
		}
		if again.BGMVolume != tc.bgm || again.SEVolume != tc.se {
			t.Fatalf("volume round trip: %s", data)
		}
	}
}

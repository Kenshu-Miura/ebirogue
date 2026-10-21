//go:build ignore

// 仮音源を再生成する: go run tools/generate_sounds.go
// 外部素材・ゲームの乱数源を使わない、決定的な矩形波とノイズ。
package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
)

const rate = 44100

func write(name string, notes []float64, beat float64, noise bool) {
	var pcm bytes.Buffer
	var phase float64
	var state uint32 = 12345
	n := int(beat * rate)
	for _, hz := range notes {
		for i := 0; i < n; i++ {
			phase += hz / rate
			value := 0.0
			if hz > 0 {
				value = .16
				if math.Mod(phase, 1) < .5 {
					value = -.16
				}
			}
			if noise {
				state ^= state << 13
				state ^= state >> 17
				state ^= state << 5
				value += (float64(state%65536)/32768 - 1) * .12
			}
			envelope := math.Min(1, float64(i)/220) * math.Min(1, float64(n-1-i)/440)
			sample := int16(value * envelope * 32767)
			binary.Write(&pcm, binary.LittleEndian, sample)
			binary.Write(&pcm, binary.LittleEndian, sample)
		}
	}
	var out bytes.Buffer
	out.WriteString("RIFF")
	binary.Write(&out, binary.LittleEndian, uint32(36+pcm.Len()))
	out.WriteString("WAVEfmt ")
	for _, v := range []any{uint32(16), uint16(1), uint16(2), uint32(rate), uint32(rate * 4), uint16(4), uint16(16)} {
		binary.Write(&out, binary.LittleEndian, v)
	}
	out.WriteString("data")
	binary.Write(&out, binary.LittleEndian, uint32(pcm.Len()))
	out.Write(pcm.Bytes())
	if err := os.WriteFile("sound/"+name+".wav", out.Bytes(), 0644); err != nil {
		panic(err)
	}
}
func main() {
	if err := os.MkdirAll("sound", 0755); err != nil {
		panic(err)
	}
	for _, s := range []struct {
		name  string
		notes []float64
		noise bool
	}{
		{"hit", []float64{180, 90}, true}, {"miss", []float64{320, 160}, true},
		{"hurt", []float64{130, 65}, true}, {"defeat", []float64{220, 165, 110}, false},
		{"level", []float64{262, 330, 392, 524}, false}, {"pickup", []float64{660, 880}, false},
		{"stairs", []float64{392, 330, 262}, false}, {"trap", []float64{90, 180, 90}, true},
		{"use", []float64{440, 660, 880}, false}, {"death", []float64{330, 262, 196, 131}, false},
		{"clear", []float64{262, 330, 392, 524, 660}, false}, {"confirm", []float64{660, 880}, false},
		{"cancel", []float64{440, 220}, false},
	} {
		write(s.name, s.notes, .065, s.noise)
	}
	// 8小節×4拍。曲頭・曲末もゼロへ減衰し、ループ境界でクリックしない。
	for _, song := range []struct {
		name  string
		notes []float64
	}{
		{"dungeon", []float64{262, 330, 392, 330, 220, 262, 330, 0, 196, 247, 294, 247, 220, 262, 330, 0}},
		{"boss", []float64{131, 196, 156, 196, 147, 220, 175, 220, 131, 196, 156, 233, 175, 156, 147, 196}},
		{"victory", []float64{262, 330, 392, 524, 392, 330, 392, 0, 330, 392, 494, 660, 524, 392, 524, 0}},
	} {
		notes := append(append([]float64{}, song.notes...), song.notes...)
		write(song.name, notes, .25, false)
	}
}

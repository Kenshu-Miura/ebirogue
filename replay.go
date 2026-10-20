package main

import (
	"encoding/json"
	"fmt"
)

const replayFileName = "ebirogue_replay_last.json"
const replayHistoryFileName = "ebirogue_replays.json"
const replayVersion = 1

// ReplayData は論理入力の連続状態と、中断再開ごとの開始状態を保存する。
type ReplayData struct {
	Version     int
	SaveVersion int
	Build       string
	Seed        int64
	Difficulty  Difficulty
	Segments    []*ReplaySegment
}
type ReplayPhysical struct {
	Frame int
	Key   string
	Held  bool
}
type ReplaySegment struct {
	Physical      []ReplayPhysical `json:",omitempty"`
	Initial       *SaveData
	InputFrame    int
	PreviousRoom  int
	Fade          float64
	Fast          bool
	ResumeMessage bool
	Runs          []ReplayKeys
	Checks        []string
	Final         ReplayResult
}
type ReplayKeys struct {
	Frames    int
	Durations [keyCount]int
}
type ReplayResult struct {
	Floor, HP, Turns int
	Cause            DamageSource
}

func replayEnabled(s GameSettings) bool { return s.SaveInputReplay == nil || *s.SaveInputReplay }

func decodeReplay(data []byte) (*ReplayData, error) {
	if len(data) > 64<<20 {
		return nil, fmt.Errorf("フレーム0: 記録が大きすぎる")
	}
	var r ReplayData
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("フレーム0: 記録の解析: %w", err)
	}
	if r.Version != replayVersion || r.SaveVersion != saveDataVersion || !r.Difficulty.valid() || len(r.Segments) == 0 {
		return nil, fmt.Errorf("フレーム0: 記録のバージョンまたはヘッダーが不正")
	}
	total := 0
	for _, s := range r.Segments {
		if s == nil || s.Initial == nil || s.Initial.Replay != nil {
			return nil, fmt.Errorf("フレーム%d: 開始状態が不正", total)
		}
		if s.Initial.Seed != r.Seed || s.Initial.Difficulty != r.Difficulty || s.InputFrame < 0 || s.InputFrame > 1000000000 || s.Fade < 0 || s.Fade > 1 {
			return nil, fmt.Errorf("フレーム%d: 開始状態のメタデータが不正", total)
		}
		if err := validateSaveData(s.Initial); err != nil {
			return nil, fmt.Errorf("フレーム%d: %w", total, err)
		}
		n := 0
		for _, run := range s.Runs {
			if run.Frames <= 0 || run.Frames > 10000000-n {
				return nil, fmt.Errorf("フレーム%d: 入力の長さが不正", total+n)
			}
			for _, d := range run.Durations {
				if d < 0 || d > 10000000 {
					return nil, fmt.Errorf("フレーム%d: 押下時間が不正", total+n)
				}
			}
			n += run.Frames
		}
		if n != len(s.Checks) || total > 10000000-n {
			return nil, fmt.Errorf("フレーム%d: 照合情報の長さが不正", total)
		}
		lastPhysical := -1
		for _, p := range s.Physical {
			if p.Frame < 0 || p.Frame >= n || p.Frame <= lastPhysical {
				return nil, fmt.Errorf("フレーム%d: 物理入力の位置が不正", total)
			}
			lastPhysical = p.Frame
		}
		total += n
	}
	return &r, nil
}

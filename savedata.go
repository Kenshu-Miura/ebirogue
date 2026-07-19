package main

import (
	"encoding/json"
	"fmt"
)

// セーブ・設定ファイルの定義。ゲーム本体に依存しない純粋なデータ構造と
// 検証ロジックのみを置く（テストビルドでも利用できるようにビルドタグなし）。

const (
	saveFileName     = "ebirogue_save.json"
	settingsFileName = "ebirogue_settings.json"
	saveDataVersion  = 1
)

// SavedRoom は部屋情報のセーブ用表現
type SavedRoom struct {
	ID            int
	X, Y          int
	Width, Height int
	CenterX       int
	CenterY       int
}

// SavedItem はアイテム1個のセーブ用表現。
// テンプレートID(ID)から再生成し、可変フィールドだけを上書きする。
type SavedItem struct {
	Kind       string // 具体型名（"Weapon", "Food" など）
	ID         int    // itemTemplatesのID
	X, Y       int
	Sharpness  int
	Cursed     bool
	Identified bool
	RustProof  bool
	ShotCount  int
	Uses       int
	Amount     int
}

// SavedEnemy は敵1体のセーブ用表現。
// MonsterDefinitionsのIDから再生成し、可変フィールドだけを上書きする。
type SavedEnemy struct {
	ID               int
	X, Y             int
	Health           int
	MaxHealth        int
	AttackPower      int
	DefensePower     int
	ExperiencePoints int
	Direction        int
	PlayerDiscovered bool
	StatusAilments   StatusAilments
}

// SavedMapTrap はマップ上の罠のセーブ用表現
type SavedMapTrap struct {
	TemplateID int
	X, Y       int
	Discovered bool
}

// SavedPlayer はプレイヤーのセーブ用表現。
// 装備はインベントリのインデックスで表す（-1は未装備）。
type SavedPlayer struct {
	X, Y                int
	Health              int
	MaxHealth           int
	AttackPower         int
	DefensePower        int
	Power               int
	MaxPower            int
	Satiety             int
	MaxSatiety          int
	ExperiencePoints    int
	Level               int
	Cash                int
	Direction           int
	StatusAilments      StatusAilments
	Inventory           []SavedItem
	EquippedWeapon      int
	EquippedArmor       int
	EquippedArrow       int
	EquippedAccessories [2]int
	SetTrap             *SavedItem
}

// SaveData は中断セーブ全体
type SaveData struct {
	Version           int
	Floor             int
	Map               [][]Tile
	Rooms             []SavedRoom
	Player            SavedPlayer
	Enemies           []SavedEnemy
	Items             []SavedItem
	MapTraps          []SavedMapTrap
	MoveCount         int
	TurnCount         int
	LastSpawnTurn     int
	SpawnInterval     int
	FloorTurns        int
	WindWarning1Shown bool
	WindWarning2Shown bool
	PickupBanned      bool // 拾得禁止のカードの効果中かどうか
	Messages          []string
	CustomNames       map[int]string // 未識別アイテム種別に付けた任意名
}

// decodeSaveData はJSONを解析し、検証済みのセーブデータを返す
func decodeSaveData(data []byte) (*SaveData, error) {
	var save SaveData
	if err := json.Unmarshal(data, &save); err != nil {
		return nil, fmt.Errorf("中断データの解析に失敗: %w", err)
	}
	if err := validateSaveData(&save); err != nil {
		return nil, err
	}
	return &save, nil
}

// validateSaveData は破損したセーブデータを読み込まないための検証を行う
func validateSaveData(save *SaveData) error {
	if save.Version != saveDataVersion {
		return fmt.Errorf("中断データのバージョンが不正: %d", save.Version)
	}
	if len(save.Map) == 0 || len(save.Map[0]) == 0 {
		return fmt.Errorf("マップデータが空")
	}
	width := len(save.Map[0])
	height := len(save.Map)
	for y, row := range save.Map {
		if len(row) != width {
			return fmt.Errorf("マップの%d行目の長さが不揃い", y)
		}
	}
	if !inMapBounds(save.Player.X, save.Player.Y, width, height) {
		return fmt.Errorf("プレイヤー座標がマップ外: (%d, %d)", save.Player.X, save.Player.Y)
	}
	inventoryLen := len(save.Player.Inventory)
	equippedIndices := []int{
		save.Player.EquippedWeapon,
		save.Player.EquippedArmor,
		save.Player.EquippedArrow,
		save.Player.EquippedAccessories[0],
		save.Player.EquippedAccessories[1],
	}
	for _, idx := range equippedIndices {
		if idx < -1 || idx >= inventoryLen {
			return fmt.Errorf("装備インデックスが不正: %d", idx)
		}
	}
	for _, enemy := range save.Enemies {
		if !inMapBounds(enemy.X, enemy.Y, width, height) {
			return fmt.Errorf("敵の座標がマップ外: (%d, %d)", enemy.X, enemy.Y)
		}
	}
	for _, item := range save.Items {
		if !inMapBounds(item.X, item.Y, width, height) {
			return fmt.Errorf("アイテムの座標がマップ外: (%d, %d)", item.X, item.Y)
		}
	}
	for _, trap := range save.MapTraps {
		if !inMapBounds(trap.X, trap.Y, width, height) {
			return fmt.Errorf("罠の座標がマップ外: (%d, %d)", trap.X, trap.Y)
		}
	}
	return nil
}

func inMapBounds(x, y, width, height int) bool {
	return x >= 0 && x < width && y >= 0 && y < height
}

// GameSettings はゲーム設定
type GameSettings struct {
	Fullscreen  bool `json:"fullscreen"`
	ShowMiniMap bool `json:"showMinimap"`
}

// defaultSettings は設定の初期値を返す
func defaultSettings() GameSettings {
	return GameSettings{
		Fullscreen:  false,
		ShowMiniMap: true,
	}
}

// decodeSettings はJSONを解析して設定を返す。
// 解析に失敗した場合は初期値とエラーを返す。
func decodeSettings(data []byte) (GameSettings, error) {
	settings := defaultSettings()
	if err := json.Unmarshal(data, &settings); err != nil {
		return defaultSettings(), fmt.Errorf("設定ファイルの解析に失敗: %w", err)
	}
	return settings, nil
}

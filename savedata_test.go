package main

import (
	"encoding/json"
	"testing"
)

// テスト用の最小限の有効なセーブデータを作る
func makeValidSaveData() *SaveData {
	tiles := make([][]Tile, 3)
	for y := range tiles {
		tiles[y] = make([]Tile, 3)
		for x := range tiles[y] {
			tiles[y][x] = Tile{Type: "floor"}
		}
	}
	return &SaveData{
		Version: saveDataVersion,
		Floor:   2,
		Map:     tiles,
		Player: SavedPlayer{
			X: 1, Y: 1,
			Health: 50, MaxHealth: 100,
			EquippedWeapon:      -1,
			EquippedArmor:       -1,
			EquippedArrow:       -1,
			EquippedAccessories: [2]int{-1, -1},
		},
	}
}

func TestValidateSaveDataValid(t *testing.T) {
	if err := validateSaveData(makeValidSaveData()); err != nil {
		t.Errorf("有効なデータが検証エラーになった: %v", err)
	}
}

func TestValidateSaveDataVersionMismatch(t *testing.T) {
	save := makeValidSaveData()
	save.Version = saveDataVersion + 1
	if err := validateSaveData(save); err == nil {
		t.Error("バージョン不一致が検出されなかった")
	}
}

func TestValidateSaveDataEmptyMap(t *testing.T) {
	save := makeValidSaveData()
	save.Map = nil
	if err := validateSaveData(save); err == nil {
		t.Error("空のマップが検出されなかった")
	}
}

func TestValidateSaveDataRaggedMap(t *testing.T) {
	save := makeValidSaveData()
	save.Map[1] = save.Map[1][:2] // 行の長さを不揃いにする
	if err := validateSaveData(save); err == nil {
		t.Error("不揃いなマップ行が検出されなかった")
	}
}

func TestValidateSaveDataPlayerOutOfBounds(t *testing.T) {
	save := makeValidSaveData()
	save.Player.X = 10
	if err := validateSaveData(save); err == nil {
		t.Error("マップ外のプレイヤー座標が検出されなかった")
	}
}

func TestValidateSaveDataInvalidEquippedIndex(t *testing.T) {
	// インベントリが空なのに装備インデックスが0
	save := makeValidSaveData()
	save.Player.EquippedWeapon = 0
	if err := validateSaveData(save); err == nil {
		t.Error("範囲外の装備インデックスが検出されなかった")
	}

	// -1未満も不正
	save = makeValidSaveData()
	save.Player.EquippedArmor = -2
	if err := validateSaveData(save); err == nil {
		t.Error("-1未満の装備インデックスが検出されなかった")
	}
}

func TestValidateSaveDataEnemyOutOfBounds(t *testing.T) {
	save := makeValidSaveData()
	save.Enemies = []SavedEnemy{{ID: 0, X: -1, Y: 0}}
	if err := validateSaveData(save); err == nil {
		t.Error("マップ外の敵座標が検出されなかった")
	}
}

func TestDecodeSaveDataCorrupted(t *testing.T) {
	if _, err := decodeSaveData([]byte("{broken json")); err == nil {
		t.Error("壊れたJSONがエラーにならなかった")
	}
	if _, err := decodeSaveData([]byte("")); err == nil {
		t.Error("空データがエラーにならなかった")
	}
}

func TestDecodeSaveDataRoundTrip(t *testing.T) {
	original := makeValidSaveData()
	original.Messages = []string{"メッセージ1", "メッセージ2"}
	original.PickupBanned = true
	original.Player.StatusAilments.MouthSeal = 7
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("シリアライズに失敗: %v", err)
	}
	decoded, err := decodeSaveData(data)
	if err != nil {
		t.Fatalf("デコードに失敗: %v", err)
	}
	if decoded.Floor != original.Floor {
		t.Errorf("Floorが一致しない: got %d, want %d", decoded.Floor, original.Floor)
	}
	if len(decoded.Messages) != 2 || decoded.Messages[0] != "メッセージ1" {
		t.Errorf("Messagesが一致しない: %v", decoded.Messages)
	}
	if !decoded.PickupBanned {
		t.Error("PickupBannedが一致しない")
	}
	if decoded.Player.StatusAilments.MouthSeal != 7 {
		t.Error("口封じの残りターン数が一致しない")
	}
}

func TestDecodeSettings(t *testing.T) {
	// 正常な設定
	settings, err := decodeSettings([]byte(`{"fullscreen":true,"showMinimap":false}`))
	if err != nil {
		t.Fatalf("有効な設定の解析に失敗: %v", err)
	}
	if !settings.Fullscreen || settings.ShowMiniMap {
		t.Errorf("設定値が一致しない: %+v", settings)
	}

	// 一部のキーのみの場合、残りは初期値のまま
	settings, err = decodeSettings([]byte(`{"fullscreen":true}`))
	if err != nil {
		t.Fatalf("部分的な設定の解析に失敗: %v", err)
	}
	if !settings.Fullscreen || !settings.ShowMiniMap {
		t.Errorf("部分的な設定で初期値が保持されていない: %+v", settings)
	}

	// 壊れた設定は初期値とエラーを返す
	settings, err = decodeSettings([]byte("{broken"))
	if err == nil {
		t.Error("壊れた設定がエラーにならなかった")
	}
	if settings != defaultSettings() {
		t.Errorf("壊れた設定で初期値が返らなかった: %+v", settings)
	}
}

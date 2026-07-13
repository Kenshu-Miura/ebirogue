//go:build !test

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// --- 設定の読み書き ---

// loadSettings は設定ファイルを読み込む。存在しない・壊れている場合は初期値を返す
func loadSettings() GameSettings {
	data, err := os.ReadFile(settingsFileName)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("設定ファイルの読み込みに失敗: %v", err)
		}
		return defaultSettings()
	}
	settings, err := decodeSettings(data)
	if err != nil {
		log.Printf("設定ファイルが壊れているため初期値を使用します: %v", err)
		return defaultSettings()
	}
	return settings
}

// saveSettings は設定をファイルへ保存する
func saveSettings(settings GameSettings) {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		log.Printf("設定のシリアライズに失敗: %v", err)
		return
	}
	if err := os.WriteFile(settingsFileName, data, 0644); err != nil {
		log.Printf("設定ファイルの保存に失敗: %v", err)
	}
}

// --- アイテム・敵・罠の相互変換 ---

// itemToSaved はアイテムをセーブ用表現へ変換する
func itemToSaved(item Item) SavedItem {
	switch v := item.(type) {
	case *Weapon:
		return SavedItem{Kind: "Weapon", ID: v.ID, X: v.X, Y: v.Y,
			Sharpness: v.Sharpness, Cursed: v.Cursed, Identified: v.Identified}
	case *Armor:
		return SavedItem{Kind: "Armor", ID: v.ID, X: v.X, Y: v.Y,
			Sharpness: v.Sharpness, Cursed: v.Cursed, Identified: v.Identified}
	case *Arrow:
		return SavedItem{Kind: "Arrow", ID: v.ID, X: v.X, Y: v.Y,
			ShotCount: v.ShotCount, Cursed: v.Cursed, Identified: v.Identified}
	case *Food:
		return SavedItem{Kind: "Food", ID: v.ID, X: v.X, Y: v.Y}
	case *Potion:
		return SavedItem{Kind: "Potion", ID: v.ID, X: v.X, Y: v.Y}
	case *Card:
		return SavedItem{Kind: "Card", ID: v.ID, X: v.X, Y: v.Y}
	case *Money:
		return SavedItem{Kind: "Money", ID: v.ID, X: v.X, Y: v.Y,
			Amount: v.Amount, Identified: v.Identified}
	case *Accessory:
		return SavedItem{Kind: "Accessory", ID: v.ID, X: v.X, Y: v.Y,
			Cursed: v.Cursed, Identified: v.Identified}
	case *Cane:
		return SavedItem{Kind: "Cane", ID: v.ID, X: v.X, Y: v.Y,
			Uses: v.Uses, Identified: v.Identified}
	case *Trap:
		return SavedItem{Kind: "Trap", ID: v.ID, X: v.X, Y: v.Y}
	default:
		return SavedItem{Kind: "Unknown"}
	}
}

// savedToItem はセーブ用表現からアイテムを再構築する。
// テンプレートから生成した後、可変フィールドをセーブ値で上書きする。
func savedToItem(s SavedItem) (Item, error) {
	item := buildItemFromTemplate(s.ID, s.X, s.Y)
	switch s.Kind {
	case "Weapon":
		v, ok := item.(*Weapon)
		if !ok {
			return nil, fmt.Errorf("アイテムID %dはWeaponではない", s.ID)
		}
		v.Sharpness = s.Sharpness
		v.Cursed = s.Cursed
		v.Identified = s.Identified
	case "Armor":
		v, ok := item.(*Armor)
		if !ok {
			return nil, fmt.Errorf("アイテムID %dはArmorではない", s.ID)
		}
		v.Sharpness = s.Sharpness
		v.Cursed = s.Cursed
		v.Identified = s.Identified
	case "Arrow":
		v, ok := item.(*Arrow)
		if !ok {
			return nil, fmt.Errorf("アイテムID %dはArrowではない", s.ID)
		}
		v.ShotCount = s.ShotCount
		v.Cursed = s.Cursed
		v.Identified = s.Identified
	case "Food":
		if _, ok := item.(*Food); !ok {
			return nil, fmt.Errorf("アイテムID %dはFoodではない", s.ID)
		}
	case "Potion":
		if _, ok := item.(*Potion); !ok {
			return nil, fmt.Errorf("アイテムID %dはPotionではない", s.ID)
		}
	case "Card":
		if _, ok := item.(*Card); !ok {
			return nil, fmt.Errorf("アイテムID %dはCardではない", s.ID)
		}
	case "Money":
		v, ok := item.(*Money)
		if !ok {
			return nil, fmt.Errorf("アイテムID %dはMoneyではない", s.ID)
		}
		v.Amount = s.Amount
		v.Identified = s.Identified
	case "Accessory":
		v, ok := item.(*Accessory)
		if !ok {
			return nil, fmt.Errorf("アイテムID %dはAccessoryではない", s.ID)
		}
		v.Cursed = s.Cursed
		v.Identified = s.Identified
	case "Cane":
		v, ok := item.(*Cane)
		if !ok {
			return nil, fmt.Errorf("アイテムID %dはCaneではない", s.ID)
		}
		v.Uses = s.Uses
		v.Identified = s.Identified
	case "Trap":
		if _, ok := item.(*Trap); !ok {
			return nil, fmt.Errorf("アイテムID %dはTrapではない", s.ID)
		}
	default:
		return nil, fmt.Errorf("不明なアイテム種別: %q", s.Kind)
	}
	return item, nil
}

// enemyToSaved は敵をセーブ用表現へ変換する
func enemyToSaved(enemy *Enemy) SavedEnemy {
	return SavedEnemy{
		ID:               enemy.ID,
		X:                enemy.X,
		Y:                enemy.Y,
		Health:           enemy.Health,
		MaxHealth:        enemy.MaxHealth,
		AttackPower:      enemy.AttackPower,
		DefensePower:     enemy.DefensePower,
		ExperiencePoints: enemy.ExperiencePoints,
		Direction:        int(enemy.Direction),
		PlayerDiscovered: enemy.PlayerDiscovered,
		StatusAilments:   enemy.StatusAilments,
	}
}

// savedToEnemy はセーブ用表現から敵を再構築する。
// 定義テーブルから生成した後、可変フィールドをセーブ値で上書きする。
func savedToEnemy(s SavedEnemy) Enemy {
	enemy := CreateEnemyByID(s.ID, s.X, s.Y)
	enemy.Health = s.Health
	enemy.MaxHealth = s.MaxHealth
	enemy.AttackPower = s.AttackPower
	enemy.DefensePower = s.DefensePower
	enemy.ExperiencePoints = s.ExperiencePoints
	enemy.Direction = Direction(s.Direction)
	enemy.PlayerDiscovered = s.PlayerDiscovered
	enemy.StatusAilments = s.StatusAilments
	return enemy
}

// mapTrapTemplateID は罠テンプレートのインデックスを名前から取得する
func mapTrapTemplateID(name string) int {
	for i, template := range mapTrapTemplates {
		if template.Name == name {
			return i
		}
	}
	return 0
}

// --- セーブデータの構築と復元 ---

// buildSaveData は現在のゲーム状態からセーブデータを構築する
func (g *Game) buildSaveData() *SaveData {
	player := &g.state.Player

	savedPlayer := SavedPlayer{
		X:                player.X,
		Y:                player.Y,
		Health:           player.Health,
		MaxHealth:        player.MaxHealth,
		AttackPower:      player.AttackPower,
		DefensePower:     player.DefensePower,
		Power:            player.Power,
		MaxPower:         player.MaxPower,
		Satiety:          player.Satiety,
		MaxSatiety:       player.MaxSatiety,
		ExperiencePoints: player.ExperiencePoints,
		Level:            player.Level,
		Cash:             player.Cash,
		Direction:        int(player.Direction),
		StatusAilments:   player.StatusAilments,
		EquippedWeapon:   -1,
		EquippedArmor:    -1,
		EquippedArrow:    -1,
	}
	savedPlayer.EquippedAccessories = [2]int{-1, -1}

	// インベントリを変換しつつ、装備中アイテムのインデックスを記録する
	for i, item := range player.Inventory {
		savedPlayer.Inventory = append(savedPlayer.Inventory, itemToSaved(item))
		switch v := item.(type) {
		case *Weapon:
			if player.EquippedWeapon == v {
				savedPlayer.EquippedWeapon = i
			}
		case *Armor:
			if player.EquippedArmor == v {
				savedPlayer.EquippedArmor = i
			}
		case *Arrow:
			if player.EquippedArrow == v {
				savedPlayer.EquippedArrow = i
			}
		case *Accessory:
			if player.EquippedAccessories[0] == v {
				savedPlayer.EquippedAccessories[0] = i
			} else if player.EquippedAccessories[1] == v {
				savedPlayer.EquippedAccessories[1] = i
			}
		}
	}

	if player.SetTrap != nil {
		savedTrap := itemToSaved(player.SetTrap)
		savedPlayer.SetTrap = &savedTrap
	}

	save := &SaveData{
		Version:           saveDataVersion,
		Floor:             g.Floor,
		Map:               g.state.Map,
		Player:            savedPlayer,
		MoveCount:         g.moveCount,
		TurnCount:         g.turnCount,
		LastSpawnTurn:     g.lastSpawnTurn,
		SpawnInterval:     g.spawnInterval,
		FloorTurns:        g.floorTurns,
		WindWarning1Shown: g.windWarning1Shown,
		WindWarning2Shown: g.windWarning2Shown,
		Messages:          g.messageLog.Messages,
		CustomNames:       g.customNames,
	}

	for _, room := range g.rooms {
		save.Rooms = append(save.Rooms, SavedRoom{
			ID: room.ID, X: room.X, Y: room.Y,
			Width: room.Width, Height: room.Height,
			CenterX: room.Center.X, CenterY: room.Center.Y,
		})
	}
	for i := range g.state.Enemies {
		save.Enemies = append(save.Enemies, enemyToSaved(&g.state.Enemies[i]))
	}
	for _, item := range g.state.Items {
		save.Items = append(save.Items, itemToSaved(item))
	}
	for _, trap := range g.state.MapTraps {
		save.MapTraps = append(save.MapTraps, SavedMapTrap{
			TemplateID: mapTrapTemplateID(trap.Name),
			X:          trap.X,
			Y:          trap.Y,
			Discovered: trap.Discovered,
		})
	}

	return save
}

// applySaveData はセーブデータからゲーム状態を復元する。
// 全ての要素の構築に成功してからゲーム状態へ反映するため、
// 途中でエラーが起きてもゲーム状態は変化しない。
func (g *Game) applySaveData(save *SaveData) error {
	// インベントリの再構築
	inventory := make([]Item, 0, len(save.Player.Inventory))
	for _, s := range save.Player.Inventory {
		item, err := savedToItem(s)
		if err != nil {
			return fmt.Errorf("インベントリの復元に失敗: %w", err)
		}
		inventory = append(inventory, item)
	}

	// 装備ポインタの解決（インデックスは検証済み）
	var equippedWeapon *Weapon
	var equippedArmor *Armor
	var equippedArrow *Arrow
	var equippedAccessories [2]*Accessory
	if idx := save.Player.EquippedWeapon; idx >= 0 {
		v, ok := inventory[idx].(*Weapon)
		if !ok {
			return fmt.Errorf("装備中の武器の復元に失敗")
		}
		equippedWeapon = v
	}
	if idx := save.Player.EquippedArmor; idx >= 0 {
		v, ok := inventory[idx].(*Armor)
		if !ok {
			return fmt.Errorf("装備中の盾の復元に失敗")
		}
		equippedArmor = v
	}
	if idx := save.Player.EquippedArrow; idx >= 0 {
		v, ok := inventory[idx].(*Arrow)
		if !ok {
			return fmt.Errorf("装備中の矢の復元に失敗")
		}
		equippedArrow = v
	}
	for slot := 0; slot < 2; slot++ {
		if idx := save.Player.EquippedAccessories[slot]; idx >= 0 {
			v, ok := inventory[idx].(*Accessory)
			if !ok {
				return fmt.Errorf("装備中のアクセサリーの復元に失敗")
			}
			equippedAccessories[slot] = v
		}
	}

	var setTrap Item
	if save.Player.SetTrap != nil {
		item, err := savedToItem(*save.Player.SetTrap)
		if err != nil {
			return fmt.Errorf("設置罠の復元に失敗: %w", err)
		}
		setTrap = item
	}

	// フロア上のアイテム・敵・罠の再構築
	floorItems := make([]Item, 0, len(save.Items))
	for _, s := range save.Items {
		item, err := savedToItem(s)
		if err != nil {
			return fmt.Errorf("フロアアイテムの復元に失敗: %w", err)
		}
		floorItems = append(floorItems, item)
	}
	enemies := make([]Enemy, 0, len(save.Enemies))
	for _, s := range save.Enemies {
		enemies = append(enemies, savedToEnemy(s))
	}
	traps := make([]MapTrap, 0, len(save.MapTraps))
	for _, s := range save.MapTraps {
		trap := createMapTrapByID(s.TemplateID, s.X, s.Y)
		trap.Discovered = s.Discovered
		traps = append(traps, trap)
	}
	rooms := make([]Room, 0, len(save.Rooms))
	for _, s := range save.Rooms {
		rooms = append(rooms, Room{
			ID: s.ID, X: s.X, Y: s.Y,
			Width: s.Width, Height: s.Height,
			Center: Coordinate{X: s.CenterX, Y: s.CenterY},
		})
	}

	// ここから先はエラーが起きないため、ゲーム状態へ一括反映する
	player := Player{
		Name:                "海老さん",
		Entity:              Entity{X: save.Player.X, Y: save.Player.Y, Char: '@'},
		Health:              save.Player.Health,
		MaxHealth:           save.Player.MaxHealth,
		AttackPower:         save.Player.AttackPower,
		DefensePower:        save.Player.DefensePower,
		Power:               save.Player.Power,
		MaxPower:            save.Player.MaxPower,
		Satiety:             save.Player.Satiety,
		MaxSatiety:          save.Player.MaxSatiety,
		Inventory:           inventory,
		MaxInventory:        20,
		ExperiencePoints:    save.Player.ExperiencePoints,
		Level:               save.Player.Level,
		Cash:                save.Player.Cash,
		Direction:           Direction(save.Player.Direction),
		StatusAilments:      save.Player.StatusAilments,
		EquippedWeapon:      equippedWeapon,
		EquippedArmor:       equippedArmor,
		EquippedArrow:       equippedArrow,
		EquippedAccessories: equippedAccessories,
		SetTrap:             setTrap,
	}

	g.state = GameState{
		Map:      save.Map,
		Player:   player,
		Enemies:  enemies,
		Items:    floorItems,
		MapTraps: traps,
	}
	g.rooms = rooms
	g.Floor = save.Floor
	g.moveCount = save.MoveCount
	g.turnCount = save.TurnCount
	g.lastSpawnTurn = save.LastSpawnTurn
	g.spawnInterval = save.SpawnInterval
	g.floorTurns = save.FloorTurns
	g.windWarning1Shown = save.WindWarning1Shown
	g.windWarning2Shown = save.WindWarning2Shown

	// メッセージ履歴を復元（上限を超える分は古いものから捨てる）
	messages := save.Messages
	if len(messages) > maxLogMessages {
		messages = messages[len(messages)-maxLogMessages:]
	}
	g.messageLog.Messages = messages

	// 未識別アイテムの任意名を復元
	g.customNames = map[int]string{}
	for id, name := range save.CustomNames {
		runes := []rune(name)
		if len(runes) > maxItemNameLength {
			runes = runes[:maxItemNameLength]
		}
		g.customNames[id] = string(runes)
	}

	// ミニマップの再構築を促す
	g.miniMap = nil
	g.miniMapDirty = true

	return nil
}

// --- セーブファイルの読み書き ---

// saveSuspendData は中断データをファイルへ保存する
func (g *Game) saveSuspendData() error {
	data, err := json.Marshal(g.buildSaveData())
	if err != nil {
		return fmt.Errorf("中断データのシリアライズに失敗: %w", err)
	}
	if err := os.WriteFile(saveFileName, data, 0644); err != nil {
		return fmt.Errorf("中断データの保存に失敗: %w", err)
	}
	return nil
}

// autoSave はフロア移動時などに自動でセーブする（失敗してもゲームは続行）
func (g *Game) autoSave() {
	if err := g.saveSuspendData(); err != nil {
		log.Printf("オートセーブに失敗: %v", err)
	}
}

// deleteSaveFile は中断データを削除する
func deleteSaveFile() {
	if err := os.Remove(saveFileName); err != nil && !os.IsNotExist(err) {
		log.Printf("中断データの削除に失敗: %v", err)
	}
}

// tryResumeFromSave は中断データがあれば復元する。
// データが壊れている場合は削除して新しい冒険を開始する（安全なフォールバック）。
func (g *Game) tryResumeFromSave() bool {
	data, err := os.ReadFile(saveFileName)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("中断データの読み込みに失敗: %v", err)
		}
		return false
	}
	save, err := decodeSaveData(data)
	if err != nil {
		log.Printf("中断データが壊れているため新しい冒険を開始します: %v", err)
		deleteSaveFile()
		return false
	}
	if err := g.applySaveData(save); err != nil {
		log.Printf("中断データの復元に失敗したため新しい冒険を開始します: %v", err)
		deleteSaveFile()
		return false
	}
	// 再開したら中断データは削除する（次の中断・オートセーブで再作成される）
	deleteSaveFile()
	g.Enqueue(Action{
		Duration:    1.0,
		Message:     "冒険を再開した",
		Execute:     func(*Game) {},
		NonBlocking: true,
	})
	return true
}

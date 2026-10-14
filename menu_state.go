//go:build !test

package main

// closeAllMenus は C キーの「すべて閉じる」を共通化する。
func (g *Game) closeAllMenus() {
	g.closeGroundItemMenu()
	g.showGroundItemDescription = false
	g.groundItemDescriptionText = ""
	g.currentGroundItem = nil
	g.showInventory = false
	g.showMenu = false
	g.showEmptyInventoryMessage = false
	g.showItemActions = false
	g.showItemDescription = false
	g.itemdescriptionText = ""
	g.returnToMenuFromInventory = false
	g.returnToMenuFromGround = false
	g.useIdentifyItem = false
	g.usePotInsert = false
	g.itemTargetEffect = nil
	g.tmpSelectedItemIndex = -1
	g.selectedItemIndex = 0
	g.selectedActionIndex = 0
	g.showSettings = false
	g.showSuspendPrompt = false
	g.suspendSelectedOption = 0
	g.suspendPromptJustOpened = false
	g.showMessageLog = false
	g.messageLogScroll = 0
	g.showHelp = false
	g.helpScroll = 0
	g.returnToMenuFromHelp = false
	g.showRunHistory = false
	g.showStairsPrompt = false
	g.showNameInput = false
	g.showKeyConfig = false
	g.keyConfigCapturing = false
	g.keyConfigWaitRelease = false
}

func (g *Game) currentGroundItemActions() []string {
	if g.currentGroundItem == nil {
		return nil
	}
	if _, ok := g.currentGroundItem.(Equipable); ok {
		return []string{"拾う", "交換", "装備", "投げる", "説明"}
	}
	return []string{"拾う", "交換", "使う", "投げる", "説明"}
}

func itemActionPreview(item Item, actions []string, index int) (int, bool, bool) {
	if index < 0 || index >= len(actions) {
		return 0, false, false
	}
	switch actions[index] {
	case "投げる", "撃つ":
		return 10, false, true
	case "使う":
		if _, ok := item.(*Cane); ok {
			return 30, true, true
		}
	}
	return 0, false, false
}

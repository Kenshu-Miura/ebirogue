package main

func createItem(x, y int) Item {
	var itemType, itemName, itemChar, itemDescription string
	randomValue := localRand.Intn(4) // Store the random value to ensure it's only generated once
	switch randomValue {
	case 0:
		itemType = "Kane"
		itemName = "小銭"
		itemChar = "!"
		itemDescription = "小銭。それは海老さんが絆と呼ぶもの。"
	case 1:
		itemType = "Sausage"
		itemName = "ウインナー"
		itemChar = "!"
		itemDescription = "海老さんが配信中に食べる食事。"
	case 2:
		itemType = "Mintia"
		itemName = "ミンティア"
		itemChar = "!"
		itemDescription = "海老さんを元気にする薬。"
	default:
		itemType = "Card"
		itemName = "カード"
		itemChar = "!"
		itemDescription = "遊戯王カード。"
	}

	return Item{
		Entity: Entity{
			X:    x,
			Y:    y,
			Char: rune(itemChar[0]),
		},
		Type:        itemType,
		Name:        itemName,
		Description: itemDescription,
	}
}

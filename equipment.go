//go:build !test

package main

import "fmt"

// 新しい装備システムのヘルパー関数

// 装備可能かチェック
func (p *Player) CanEquip(item Equipable) bool {
	switch item.(type) {
	case *Weapon:
		return true // 武器は常に装備可能（既存のものと交換）
	case *Armor:
		return true // 防具は常に装備可能（既存のものと交換）
	case *Arrow:
		return true // 矢は常に装備可能（既存のものと交換）
	case *Accessory:
		// アクセサリーは2個まで装備可能
		return p.EquippedAccessories[0] == nil || p.EquippedAccessories[1] == nil || len(p.GetAccessorySlots()) < 2
	default:
		return false
	}
}

// アクセサリーのスロット取得
func (p *Player) GetAccessorySlots() []*Accessory {
	return p.EquippedAccessories[:]
}

// アイテム装備
func (p *Player) EquipItem(item Equipable) (string, error) {
	switch v := item.(type) {
	case *Weapon:
		// 既存の武器があれば装備解除
		if p.EquippedWeapon != nil {
			p.EquippedWeapon.UpdatePlayerStats(p, false)
		}
		// 新しい武器を装備
		p.EquippedWeapon = v
		v.UpdatePlayerStats(p, true)
		return fmt.Sprintf("%sを装備した。", v.GetName()), nil
		
	case *Armor:
		// 既存の防具があれば装備解除
		if p.EquippedArmor != nil {
			p.EquippedArmor.UpdatePlayerStats(p, false)
		}
		// 新しい防具を装備
		p.EquippedArmor = v
		v.UpdatePlayerStats(p, true)
		return fmt.Sprintf("%sを装備した。", v.GetName()), nil
		
	case *Arrow:
		// 既存の矢があれば装備解除
		if p.EquippedArrow != nil {
			p.EquippedArrow.UpdatePlayerStats(p, false)
		}
		// 新しい矢を装備
		p.EquippedArrow = v
		v.UpdatePlayerStats(p, true)
		return fmt.Sprintf("%sを装備した。", v.GetName()), nil
		
	case *Accessory:
		// 最初の空きスロットに装備
		if p.EquippedAccessories[0] == nil {
			p.EquippedAccessories[0] = v
		} else if p.EquippedAccessories[1] == nil {
			p.EquippedAccessories[1] = v
		} else {
			// 2個目のアクセサリーと交換
			if p.EquippedAccessories[1] != nil {
				p.EquippedAccessories[1].UpdatePlayerStats(p, false)
			}
			p.EquippedAccessories[1] = v
		}
		v.UpdatePlayerStats(p, true)
		return fmt.Sprintf("%sを装備した。", v.GetName()), nil
		
	default:
		return "", fmt.Errorf("装備できないアイテムです")
	}
}

// アイテム装備解除
func (p *Player) UnequipItem(item Equipable) (string, error) {
	switch v := item.(type) {
	case *Weapon:
		if p.EquippedWeapon == v {
			p.EquippedWeapon.UpdatePlayerStats(p, false)
			p.EquippedWeapon = nil
			return fmt.Sprintf("%sをはずした。", v.GetName()), nil
		}
		
	case *Armor:
		if p.EquippedArmor == v {
			p.EquippedArmor.UpdatePlayerStats(p, false)
			p.EquippedArmor = nil
			return fmt.Sprintf("%sをはずした。", v.GetName()), nil
		}
		
	case *Arrow:
		if p.EquippedArrow == v {
			p.EquippedArrow.UpdatePlayerStats(p, false)
			p.EquippedArrow = nil
			return fmt.Sprintf("%sをはずした。", v.GetName()), nil
		}
		
	case *Accessory:
		if p.EquippedAccessories[0] == v {
			p.EquippedAccessories[0].UpdatePlayerStats(p, false)
			p.EquippedAccessories[0] = nil
			return fmt.Sprintf("%sをはずした。", v.GetName()), nil
		}
		if p.EquippedAccessories[1] == v {
			p.EquippedAccessories[1].UpdatePlayerStats(p, false)
			p.EquippedAccessories[1] = nil
			return fmt.Sprintf("%sをはずした。", v.GetName()), nil
		}
	}
	
	return "", fmt.Errorf("装備されていません")
}

// アイテムが装備されているかチェック
func (p *Player) IsEquipped(item Equipable) bool {
	switch v := item.(type) {
	case *Weapon:
		return p.EquippedWeapon == v
	case *Armor:
		return p.EquippedArmor == v
	case *Arrow:
		return p.EquippedArrow == v
	case *Accessory:
		return p.EquippedAccessories[0] == v || p.EquippedAccessories[1] == v
	default:
		return false
	}
}

// 装備中のアイテム一覧を取得
func (p *Player) GetEquippedItems() []Equipable {
	var equipped []Equipable
	
	if p.EquippedWeapon != nil {
		equipped = append(equipped, p.EquippedWeapon)
	}
	if p.EquippedArmor != nil {
		equipped = append(equipped, p.EquippedArmor)
	}
	if p.EquippedArrow != nil {
		equipped = append(equipped, p.EquippedArrow)
	}
	if p.EquippedAccessories[0] != nil {
		equipped = append(equipped, p.EquippedAccessories[0])
	}
	if p.EquippedAccessories[1] != nil {
		equipped = append(equipped, p.EquippedAccessories[1])
	}
	
	return equipped
}
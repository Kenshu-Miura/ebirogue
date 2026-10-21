//go:build !test

package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

const synthesisPotID = 225
const maxSynthesisAbilities = 5

var additionalPotTemplates = makePotTemplates()

func makePotTemplates() map[int]ItemTemplate {
	names := []string{"合成", "識別", "強化", "変化", "底抜け", "換金", "背中", "冷蔵"}
	descriptions := []string{
		"武器・盾は最初の同種の装備へ能力と強化値を合成する。印は5つまで。同じ杖は回数を合算。呪いも移る。",
		"入れた道具を識別する。中身は割るか吸い出しで取り出せる。",
		"入れた武器・盾の強化値を1増やす。呪いは解けない。中身は割るか吸い出しで取り出せる。",
		"入れた道具を同じ種類の別の道具に変える。中身は割るか吸い出しで取り出せる。",
		"入れた道具は消える。割ると周囲に罠ができる。",
		"入れた道具を所持金に変える。額は店の売値（買値の半分）。",
		"盗みと投げ捨ての標的を中身へ引き受ける。空なら壺が狙われる。自由に出し入れできる。",
		"中の食料を腐敗から守る。自由に出し入れできる。",
	}
	result := map[int]ItemTemplate{}
	for i, name := range names {
		id := synthesisPotID + i
		result[id] = ItemTemplate{ID: id, ItemType: "Pot", Type: "Pot", Name: name + "の壺", Description: descriptions[i], Char: '!', UseActions: map[string]UseAction{"PotEffect": func(*Game) {}}}
	}
	return result
}

func init() {
	expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "壺と合成", Lines: []string{
		"合成の壺：最初の武器・盾が土台。",
		"　同じ種類の装備の強化値と能力を移す。",
		"　印は重複せず5つまで。呪いも移る。",
		"　同じ種類の杖は回数を合算する。",
		"識別・強化・変化：入れた時に効果。",
		"底抜け：道具が消え、割ると周囲に罠。",
		"換金：道具が店の売値の所持金になる。",
		"背中：盗みと投げ捨てを中身が引き受ける。",
		"冷蔵：中の食料が腐らない。",
		"保存・合成・背中・冷蔵は直接取り出せる。",
		"ほかの壺は割るか吸い出しで取り出す。",
	}})
}

func mergedAbilities(base, donor []EquipmentAbilityID) []EquipmentAbilityID {
	result := append([]EquipmentAbilityID(nil), base...)
	for _, ability := range donor {
		if len(result) >= maxSynthesisAbilities {
			break
		}
		if !hasEquipmentAbility(result, ability) {
			result = append(result, ability)
		}
	}
	return result
}

// 合成の能力は既存のAbilitiesへ加え、フック・耐性・特効と共通にする。
func synthesizeItem(base, donor Item) bool {
	switch b := base.(type) {
	case *Weapon:
		d, ok := donor.(*Weapon)
		if !ok {
			return false
		}
		b.Abilities = mergedAbilities(b.Abilities, d.Abilities)
		b.Sharpness += d.Sharpness
		b.Cursed = b.Cursed || d.Cursed
	case *Armor:
		d, ok := donor.(*Armor)
		if !ok {
			return false
		}
		b.Abilities = mergedAbilities(b.Abilities, d.Abilities)
		b.Sharpness += d.Sharpness
		b.Cursed = b.Cursed || d.Cursed
	case *Cane:
		d, ok := donor.(*Cane)
		if !ok || b.ID != d.ID {
			return false
		}
		b.Uses += d.Uses
	default:
		return false
	}
	return true
}

// applyPotInsertEffect は入れるActionの実行時にのみ呼ぶ。
func (g *Game) applyPotInsertEffect(pot *Pot, item Item) {
	switch pot.ID {
	case synthesisPotID:
		for _, base := range pot.Contents {
			if synthesizeItem(base, item) {
				return
			}
		}
	case 226:
		if identifiable, ok := item.(Identifiable); ok {
			identifiable.SetIdentified(true)
		}
	case 227:
		switch v := item.(type) {
		case *Weapon:
			v.Sharpness++
		case *Armor:
			v.Sharpness++
		}
	case 228:
		category := itemTemplates[item.GetID()].ItemType
		ids := []int{}
		for _, id := range randomItemTemplateIDs {
			if id != item.GetID() && itemTemplates[id].ItemType == category {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			item = rollItemFromTemplate(ids[g.randIntn(len(ids))], 0, 0, g.randIntn)
		}
	case 229:
		return
	case 230:
		g.state.Player.Cash += itemSellPrice(item) // 店の売値（wave10-B）
		return
	}
	pot.Contents = append(pot.Contents, item)
}

func potAllowsTakeOut(pot *Pot) bool {
	return pot.ID == 46 || pot.ID == synthesisPotID || pot.ID == 231 || pot.ID == 232
}

func extraItemAbilities(id int, abilities []EquipmentAbilityID) []EquipmentAbilityID {
	var extra []EquipmentAbilityID
	for _, ability := range abilities {
		if !hasEquipmentAbility(itemTemplates[id].Abilities, ability) {
			extra = append(extra, ability)
		}
	}
	return extra
}

var synthesisMarkNames = map[EquipmentAbilityID]string{
	AbilitySatietyConservation: "満腹節約", AbilityDragonSlayer: "竜特効", AbilityGhostSlayer: "霊特効",
	AbilityOneEyeSlayer: "一つ目特効", AbilityDrainerSlayer: "吸い取り特効", AbilityThreeWayAttack: "三方向攻撃",
	AbilitySureHit: "必中", AbilityDigWall: "壁掘り", AbilityDisposable: "使い捨て武器",
	AbilityExplosionResistance: "爆風軽減", AbilityFireResistance: "炎軽減", AbilityMagicResistance: "魔法軽減", AbilityTheftResistance: "盗難防止", AbilityStatusResistance: "状態耐性",
	AbilityEvasion: "回避", AbilityReflection: "反射", AbilityCounter: "反撃",
	AbilitySatietyHunger: "腹減り", AbilityDisposableArmor: "使い捨て盾", AbilityShellSlayer: "甲殻特効",
	AbilityExplosiveSlayer: "爆発特効", AbilityFloatingSlayer: "浮遊特効", AbilityRustImmune: "さびない",
	AbilityKeenEdge: "命中強化", AbilityDoubleStrike: "二連撃", AbilityCriticalUp: "会心強化",
	AbilityLifeSteal: "吸血", AbilityMoneyStrike: "小銭攻撃", AbilitySatietyStrike: "満腹攻撃",
	AbilityHungerStrike: "空腹攻撃", AbilityChargeStrike: "溜め打ち", AbilityComboStrike: "連続命中",
	AbilityLuckySeven: "七の幸運", AbilityMallet: "大槌", AbilityTrapProbe: "罠探り", AbilityEdible: "かじる",
	AbilityCursePower: "呪い強化", AbilityFlameBlade: "炎攻撃", AbilityDragonGuard: "竜防御",
	AbilityNigiriGuard: "食料化防止", AbilityPuppetGuard: "あやつり防止", AbilityMoneyGuard: "小銭防御",
	AbilitySatietyGuard: "満腹防御", AbilityHungerGuard: "空腹防御", AbilityGuts: "踏ん張り",
	AbilityPeakGuard: "絶好調防御", AbilitySatori: "悟り", AbilityLuckyGuard: "幸運防御",
	AbilityCurseGuard: "呪い防止", AbilityShellDodge: "殻回避", AbilityGiantFan: "吹き飛ばし",
	AbilityCriticalGuard: "痛恨防止", AbilityTossWard: "投げ捨て防止", AbilityKnockOffWard: "弾き防止",
	AbilityCriticalCurse: "痛恨の呪い", AbilityDoubleEdge: "諸刃",
	AbilityBreakFall: "受け身", AbilityImmovable: "不動",
}

// 土台の印と追加の印を同じ表示名で列挙する。
func equipmentMarks(abilities []EquipmentAbilityID) string {
	if len(abilities) == 0 {
		return "\n印：なし"
	}
	names := []string{}
	for _, ability := range abilities {
		name := synthesisMarkNames[ability]
		if name == "" {
			name = string(ability)
		}
		names = append(names, name)
	}
	return "\n印：" + strings.Join(names, "、")
}

func (w *Weapon) GetDescription() string {
	return wrapPotDescription(w.Description + equipmentMarks(w.Abilities))
}
func (a *Armor) GetDescription() string {
	return wrapPotDescription(a.Description + equipmentMarks(a.Abilities))
}

// 背中の壺を優先し、中身があれば先頭の道具を標的にする。
func (g *Game) backPotTarget() (Item, bool) {
	for _, pot := range g.inventoryPots() {
		if pot.ID == 231 {
			if len(pot.Contents) > 0 {
				return pot.Contents[0], true
			}
			return pot, true
		}
	}
	return nil, false
}

func (g *Game) removeBackPotContent(item Item) bool {
	for _, pot := range g.inventoryPots() {
		if pot.ID != 231 {
			continue
		}
		for i, content := range pot.Contents {
			if content == item {
				pot.Contents = append(pot.Contents[:i], pot.Contents[i+1:]...)
				return true
			}
		}
	}
	return false
}

func (g *Game) scatterBottomlessTraps(x, y int) {
	for _, offset := range nearestFreeOffsets(1) {
		tx, ty := x+offset.X, y+offset.Y
		if !g.itemDropFree(tx, ty) {
			continue
		}
		trap := createMapTrapByID(pickFloorTrapID(g.Floor, g.randIntn), tx, ty)
		trap.Discovered = true
		g.state.MapTraps = append(g.state.MapTraps, trap)
	}
	g.miniMapDirty = true
}

func potUnidentifiedName(id int) string {
	return fmt.Sprintf("%sの壺", []string{"赤色", "金色", "緑色", "青緑色", "水色", "青色", "紫色", "桃色"}[id-synthesisPotID])
}

func loadAdditionalPotImages() map[int]*ebiten.Image {
	result := map[int]*ebiten.Image{}
	for id := 225; id <= 232; id++ {
		result[id] = loadImage(fmt.Sprintf("img/pot_%d.png", id))
	}
	return result
}

// 説明窓の幅に収める。既存の改行は保持する。
func wrapPotDescription(description string) string {
	var result strings.Builder
	for _, line := range strings.Split(description, "\n") {
		if result.Len() > 0 {
			result.WriteByte('\n')
		}
		for i, r := range []rune(line) {
			if i > 0 && i%30 == 0 {
				result.WriteByte('\n')
			}
			result.WriteRune(r)
		}
	}
	return result.String()
}

package main

import "strings"

// keyRoles は論理キー順の役割名。Enter とデバッグキーは変更しない。
var keyRoles = []string{
	"上", "下", "左", "右", "斜め移動", "攻撃・決定", "ダッシュ・キャンセル",
	"向き変え・射線表示", "メニュー", "矢を撃つ", "扉を開く", "絞り込み", "整頓", "名前を付ける", "ヘルプ", "メッセージ履歴",
}

func physicalKeyNames() []string {
	names := []string{"Up", "Down", "Left", "Right", "Space", "Shift", "Tab", "Backspace"}
	for c := 'A'; c <= 'Z'; c++ {
		names = append(names, string(c))
	}
	for c := '0'; c <= '9'; c++ {
		names = append(names, string(c))
	}
	return names
}

func validPhysicalKey(name string) bool {
	if len(name) == 1 {
		return name[0] >= 'A' && name[0] <= 'Z' || name[0] >= '0' && name[0] <= '9'
	}
	switch name {
	case "Up", "Down", "Left", "Right", "Space", "Shift", "Tab", "Backspace":
		return true
	}
	return false
}

func defaultKeyBindings() map[string]string {
	bindings := make(map[string]string, len(keyRoles))
	for k := range keyRoles {
		bindings[Key(k).String()] = Key(k).String()
	}
	return bindings
}

// normalizeKeyBindings は不正な項目と衝突する項目を初期値に戻す。
// 初期値への復帰で生じる衝突も繰り返し解消し、有効な入れ替えは保つ。
func normalizeKeyBindings(raw map[string]string) map[string]string {
	bindings := defaultKeyBindings()
	for logical := range bindings {
		if validPhysicalKey(raw[logical]) {
			bindings[logical] = raw[logical]
		}
	}
	for {
		counts := map[string]int{}
		for _, physical := range bindings {
			counts[physical]++
		}
		changed := false
		for logical, physical := range bindings {
			if counts[physical] > 1 && physical != logical {
				bindings[logical] = logical
				changed = true
			}
		}
		if !changed {
			return bindings
		}
	}
}

func bindPhysicalKey(bindings map[string]string, logical Key, physical string) map[string]string {
	result := normalizeKeyBindings(bindings)
	if logical < 0 || int(logical) >= len(keyRoles) || !validPhysicalKey(physical) {
		return result
	}
	name := logical.String()
	for other, assigned := range result {
		if other != name && assigned == physical {
			result[other] = result[name]
			break
		}
	}
	result[name] = physical
	return result
}

func bindingName(bindings map[string]string, k Key) string {
	if name := bindings[k.String()]; validPhysicalKey(name) && int(k) < len(keyRoles) && k >= 0 {
		return name
	}
	return k.String()
}

func bindingLabel(bindings map[string]string, k Key) string {
	name := bindingName(bindings, k)
	switch name {
	case "Up":
		return "↑"
	case "Down":
		return "↓"
	case "Left":
		return "←"
	case "Right":
		return "→"
	}
	return name
}

// keyGuide は既存の説明文のキー部分だけを現在の割り当てへ置換する。
// 一度の置換なので、Z と X の入れ替えでも再置換しない。
func keyGuide(bindings map[string]string, guide string) string {
	pairs := []string{
		"矢印キー", bindingLabel(bindings, KeyUp) + "/" + bindingLabel(bindings, KeyDown) + "/" + bindingLabel(bindings, KeyLeft) + "/" + bindingLabel(bindings, KeyRight),
		"矢印", bindingLabel(bindings, KeyUp) + "/" + bindingLabel(bindings, KeyDown) + "/" + bindingLabel(bindings, KeyLeft) + "/" + bindingLabel(bindings, KeyRight),
		"上下", bindingLabel(bindings, KeyUp) + "/" + bindingLabel(bindings, KeyDown),
		"↑", bindingLabel(bindings, KeyUp), "↓", bindingLabel(bindings, KeyDown),
		"←", bindingLabel(bindings, KeyLeft), "→", bindingLabel(bindings, KeyRight),
		"スペース", bindingLabel(bindings, KeySpace),
	}
	for _, k := range []Key{KeyShift, KeyZ, KeyX, KeyA, KeyC, KeyD, KeyF, KeyS, KeyN, KeyH, KeyL} {
		pairs = append(pairs, k.String(), bindingLabel(bindings, k))
	}
	return strings.NewReplacer(pairs...).Replace(guide)
}

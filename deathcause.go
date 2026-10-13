package main

import (
	"encoding/json"
	"fmt"
)

// DamageSourceKind はプレイヤーへダメージを与えた原因の種類（レポートで読みやすいよう文字列）。
type DamageSourceKind string

const (
	DamageSourceUnknown      DamageSourceKind = ""
	DamageSourceEnemyAttack  DamageSourceKind = "enemy"         // 敵の通常攻撃
	DamageSourceEnemyRanged  DamageSourceKind = "enemy_ranged"  // 敵の遠距離・特殊攻撃
	DamageSourceTrap         DamageSourceKind = "trap"          // 罠
	DamageSourceStarvation   DamageSourceKind = "starvation"    // 餓死
	DamageSourcePoison       DamageSourceKind = "poison"        // 毒
	DamageSourceSelfDestruct DamageSourceKind = "self_destruct" // 自爆系カード
	DamageSourceWind         DamageSourceKind = "wind"          // フロア滞在時間切れの突風
	DamageSourceOther        DamageSourceKind = "other"         // その他
)

// DamageSource はプレイヤーへ最後にダメージを与えた原因。
type DamageSource struct {
	Kind      DamageSourceKind
	EnemyID   int    `json:",omitempty"`
	EnemyName string `json:",omitempty"`
	Attack    string `json:",omitempty"` // 遠距離・特殊攻撃の種類（"炎" "矢" など）
	TrapID    int    `json:",omitempty"`
	TrapName  string `json:",omitempty"`
	Detail    string `json:",omitempty"` // 自爆系カードの名前など、その他の原因の説明
}

func enemyAttackSource(enemyID int, enemyName string) DamageSource {
	return DamageSource{Kind: DamageSourceEnemyAttack, EnemyID: enemyID, EnemyName: enemyName}
}

func enemyRangedSource(enemyID int, enemyName, attack string) DamageSource {
	return DamageSource{Kind: DamageSourceEnemyRanged, EnemyID: enemyID, EnemyName: enemyName, Attack: attack}
}

func trapSource(trapID int, trapName string) DamageSource {
	return DamageSource{Kind: DamageSourceTrap, TrapID: trapID, TrapName: trapName}
}

// Label は死因の集計に使う短い名前（例: 「火吹きイカの炎」「地雷」「餓死」）。
func (s DamageSource) Label() string {
	switch s.Kind {
	case DamageSourceEnemyAttack:
		return s.EnemyName
	case DamageSourceEnemyRanged:
		if s.Attack == "" {
			return s.EnemyName
		}
		return s.EnemyName + "の" + s.Attack
	case DamageSourceTrap:
		return s.TrapName
	case DamageSourceStarvation:
		return "餓死"
	case DamageSourcePoison:
		return "毒"
	case DamageSourceSelfDestruct:
		return s.Detail + "の爆発"
	case DamageSourceWind:
		return "突風"
	case DamageSourceOther:
		if s.Detail != "" {
			return s.Detail
		}
		return "その他"
	default:
		return "不明"
	}
}

// deathMessage は死亡画面に出す文（例: 「10階で 火吹きイカ の炎にやられた」）を作る。
func deathMessage(floor int, s DamageSource) string {
	switch s.Kind {
	case DamageSourceEnemyAttack:
		return fmt.Sprintf("%d階で %s にやられた", floor, s.EnemyName)
	case DamageSourceEnemyRanged:
		if s.Attack == "" {
			return fmt.Sprintf("%d階で %s にやられた", floor, s.EnemyName)
		}
		return fmt.Sprintf("%d階で %s の%sにやられた", floor, s.EnemyName, s.Attack)
	case DamageSourceTrap:
		return fmt.Sprintf("%d階で %s にかかって倒れた", floor, s.TrapName)
	case DamageSourceStarvation:
		return fmt.Sprintf("%d階で 空腹のあまり倒れた", floor)
	case DamageSourcePoison:
		return fmt.Sprintf("%d階で 毒がまわって倒れた", floor)
	case DamageSourceSelfDestruct:
		return fmt.Sprintf("%d階で %s の爆発に巻き込まれた", floor, s.Detail)
	case DamageSourceWind:
		return fmt.Sprintf("%d階で 突風に吹き飛ばされた", floor)
	case DamageSourceOther:
		if s.Detail != "" {
			return fmt.Sprintf("%d階で %s で倒れた", floor, s.Detail)
		}
	}
	return fmt.Sprintf("%d階で 倒れた", floor)
}

// RunResult は1回の冒険の結果。冒険の記録一覧と BOT のレポートに使う。
type RunResult struct {
	Seed         int64
	Difficulty   Difficulty
	Cleared      bool
	Floor        int // 到達階
	Turns        int
	Level        int
	DeathCause   DamageSource `json:",omitempty"`
	DeathMessage string       `json:",omitempty"`
	Inventory    []string     // 所持品の名前
	Equipment    []string     // 装備品の名前（武器・盾・矢・装身具）
	EndedAt      string       `json:",omitempty"` // 終了日時（RFC3339。BOT では空）
}

const (
	runHistoryFileName = "ebirogue_results.json"
	runHistoryLimit    = 100 // 保存する冒険の記録の上限（古いものから捨てる）
)

// appendRunHistory は保存済みの記録（JSON 配列。空や壊れたデータは無視）へ結果を追加した JSON を返す。
func appendRunHistory(data []byte, result RunResult, limit int) ([]byte, error) {
	var history []RunResult
	if len(data) > 0 {
		if err := json.Unmarshal(data, &history); err != nil {
			history = nil // 壊れた記録は作り直す
		}
	}
	history = append(history, result)
	if limit > 0 && len(history) > limit {
		history = history[len(history)-limit:]
	}
	return json.MarshalIndent(history, "", "  ")
}

// decodeRunHistory は保存済みの記録を読む。
func decodeRunHistory(data []byte) ([]RunResult, error) {
	var history []RunResult
	if len(data) == 0 {
		return nil, nil
	}
	err := json.Unmarshal(data, &history)
	return history, err
}

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
	DamageSourcePoison       DamageSourceKind = "poison"        // 毒（毒の状態異常は廃止済み。過去の冒険の記録を表示するために残す）
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

// ItemActionCounts はアイテムの種類ごとの操作回数。
type ItemActionCounts struct {
	Picked   int `json:",omitempty"` // 拾った
	Used     int `json:",omitempty"` // 使った（食べる・飲む・読む・振る・置く罠）
	Thrown   int `json:",omitempty"` // 投げた
	Equipped int `json:",omitempty"` // 装備した
	Shot     int `json:",omitempty"` // 矢を撃った
}

// ItemUseRecord は道具を使った記録（判断ミスの集計用に直近の分だけ残す）。
type ItemUseRecord struct {
	TemplateID int
	Floor      int
	Turn       int
}

const recentItemUseLimit = 20

// RunStats は冒険中の行動の統計。セーブにも保存し、再開後も数え続ける。
type RunStats struct {
	Items      map[string]*ItemActionCounts `json:",omitempty"` // キーはアイテムの種類（"Card" "Potion" など）
	UsedIDs    map[int]int                  `json:",omitempty"` // テンプレート ID ごとの使用回数
	RecentUses []ItemUseRecord              `json:",omitempty"`
	// 店（wave10-B）: 買った品の数・売った回数・泥棒をした回数
	ShopBought int `json:",omitempty"`
	ShopSold   int `json:",omitempty"`
	ShopThefts int `json:",omitempty"`
}

func (s *RunStats) counts(kind string) *ItemActionCounts {
	if s.Items == nil {
		s.Items = map[string]*ItemActionCounts{}
	}
	if s.Items[kind] == nil {
		s.Items[kind] = &ItemActionCounts{}
	}
	return s.Items[kind]
}

// recordUse は使用回数と直近の使用記録を更新する。
func (s *RunStats) recordUse(kind string, templateID, floor, turn int) {
	s.counts(kind).Used++
	if s.UsedIDs == nil {
		s.UsedIDs = map[int]int{}
	}
	s.UsedIDs[templateID]++
	s.RecentUses = append(s.RecentUses, ItemUseRecord{TemplateID: templateID, Floor: floor, Turn: turn})
	if len(s.RecentUses) > recentItemUseLimit {
		s.RecentUses = s.RecentUses[len(s.RecentUses)-recentItemUseLimit:]
	}
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
	Satiety      int          // 終了時の満腹度
	Inventory    []string     // 所持品の名前
	InventoryIDs []int        `json:",omitempty"` // 所持品のテンプレート ID（Inventory と同じ順）
	Stats        RunStats     `json:",omitempty"`
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

// newestRunHistory は保存の追加順を逆転し、元の配列を変更しない。
func newestRunHistory(history []RunResult) []RunResult {
	results := make([]RunResult, len(history))
	for i, result := range history {
		results[len(history)-1-i] = result
	}
	return results
}

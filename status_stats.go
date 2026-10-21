package main

const (
	StatusAttackUp    StatusKind = "attack_up"
	StatusAttackDown  StatusKind = "attack_down"
	StatusDefenseUp   StatusKind = "defense_up"
	StatusDefenseDown StatusKind = "defense_down"
	StatusNoHealing   StatusKind = "no_healing"
	StatusLevelDown   StatusKind = "level_down" // 即時のレベル低下。持続する状態ではない。
)

func oppositeStatStatus(kind StatusKind) StatusKind {
	switch kind {
	case StatusHaste:
		return StatusSlow
	case StatusSlow:
		return StatusHaste
	case StatusAttackUp:
		return StatusAttackDown
	case StatusAttackDown:
		return StatusAttackUp
	case StatusDefenseUp:
		return StatusDefenseDown
	case StatusDefenseDown:
		return StatusDefenseUp
	}
	return ""
}

func statAttackMultiplier(s StatusAilments) float64 {
	if s.AttackUp > 0 {
		return 1.5
	}
	if s.AttackDown > 0 {
		return 0.5
	}
	return 1
}

func statDefenseMultiplier(s StatusAilments) float64 {
	if s.DefenseUp > 0 {
		return 0.67
	}
	if s.DefenseDown > 0 {
		return 1.5
	}
	return 1
}

package game

// Status IDs - potions, scrolls, enemy effects, fountain.
const (
	StatusStrength      = "strength"
	StatusInvisibility  = "invisibility"
	StatusFireResist    = "fire_resist"
	StatusParalysis     = "paralysis"
	StatusLevitation    = "levitation"
	StatusEnlightenment = "enlightenment"
	StatusConfusion     = "confusion"
	StatusHex           = "hex"
	StatusRend          = "rend"
	StatusBleed         = "bleed"
	StatusEntangle      = "entangle"
	StatusSpore         = "spore"
	StatusPoison        = "poison"
	StatusRegenerate    = "regenerate"
	StatusBless         = "bless"
	StatusCurse         = "curse"
	StatusSummon        = "summon"
	StatusSleep         = "sleep"
	StatusBlind         = "blindness"
	StatusHaste         = "haste"
	StatusSlow          = "slow"
	StatusSilence       = "silence"
	StatusStun          = "stun"
)

// Ensure Statuses maps are initialized.
func (p *Party) ensureStatuses() {
	if p.Statuses == nil {
		p.Statuses = make(map[string]int)
	}
}
func (e *EnemyParty) ensureStatuses() {
	if e.Statuses == nil {
		e.Statuses = make(map[string]int)
	}
}

// HasStatus reports whether id is active (duration >0).
func (p *Party) HasStatus(id string) bool {
	if p == nil || p.Statuses == nil {
		return false
	}
	return p.Statuses[id] > 0
}
func (e *EnemyParty) HasStatus(id string) bool {
	if e == nil || e.Statuses == nil {
		return false
	}
	return e.Statuses[id] > 0
}

// StatusDuration returns remaining turns.
func (p *Party) StatusDuration(id string) int {
	if p == nil || p.Statuses == nil {
		return 0
	}
	return p.Statuses[id]
}
func (e *EnemyParty) StatusDuration(id string) int {
	if e == nil || e.Statuses == nil {
		return 0
	}
	return e.Statuses[id]
}

// ApplyStatus sets duration (refreshes if longer, otherwise overwrites).
func (p *Party) ApplyStatus(id string, dur int) {
	if p == nil {
		return
	}
	p.ensureStatuses()
	if dur <= 0 {
		delete(p.Statuses, id)
		return
	}
	p.Statuses[id] = dur
}
func (e *EnemyParty) ApplyStatus(id string, dur int) {
	if e == nil {
		return
	}
	e.ensureStatuses()
	if dur <= 0 {
		delete(e.Statuses, id)
		return
	}
	e.Statuses[id] = dur
}

// RemoveStatus deletes status.
func (p *Party) RemoveStatus(id string) {
	if p == nil || p.Statuses == nil {
		return
	}
	delete(p.Statuses, id)
}
func (e *EnemyParty) RemoveStatus(id string) {
	if e == nil || e.Statuses == nil {
		return
	}
	delete(e.Statuses, id)
}

// TickStatuses decrements all durations by 1, removes expired, returns expired ids.
func (p *Party) TickStatuses() []string {
	if p == nil || p.Statuses == nil {
		return nil
	}
	var expired []string
	for id, d := range p.Statuses {
		d--
		if d <= 0 {
			delete(p.Statuses, id)
			expired = append(expired, id)
		} else {
			p.Statuses[id] = d
		}
	}
	return expired
}
func (e *EnemyParty) TickStatuses() []string {
	if e == nil || e.Statuses == nil {
		return nil
	}
	var expired []string
	for id, d := range e.Statuses {
		d--
		if d <= 0 {
			delete(e.Statuses, id)
			expired = append(expired, id)
		} else {
			e.Statuses[id] = d
		}
	}
	return expired
}

// Effective DEF modifiers from statuses — data-driven via statuses.json defDelta.
// Example: hex -1, bless +1, curse -1 (durations in statuses.json, deltas here).
func (p *Party) effectiveDEFDelta() int {
	if p == nil || p.Statuses == nil {
		return 0
	}
	return effectiveDEFDeltaData(p.Statuses)
}
func (e *EnemyParty) effectiveDEFDelta() int {
	if e == nil || e.Statuses == nil {
		return 0
	}
	return effectiveDEFDeltaData(e.Statuses)
}

// StatusResistCheck returns true if party resists status application via Iron Will / Ward.
// Chances are data-driven via statuses.json resists (iron_will 0.10, ward 0.05) with fallback.
func (p *Party) resistsStatus(isMagic bool, rng interface{ Float64() float64 }) bool {
	if p == nil {
		return false
	}
	for _, m := range p.Members {
		if m.IsAlive() && m.HasTalent("iron_will") {
			if rng != nil && rng.Float64() < resistsForTalent("iron_will") {
				return true
			}
		}
		if isMagic && m.IsAlive() && m.HasTalent("ward") {
			if rng != nil && rng.Float64() < resistsForTalent("ward") {
				return true
			}
		}
	}
	return false
}

// StatusHolder is any party that can receive statuses.
type StatusHolder interface {
	ApplyStatus(string, int)
}

// applyEffect is the single-source duration table for combat effect switches.
// It handles Party resists via resistsStatus (iron_will 10% / ward 5% magic) once.
// Durations: hex 10, rend/bleed 6, entangle 4, spore 8, blind 20, haste 50, slow 30, silence 6, stun 2 (1t effective), poison 6/4 (vault/pitfall use poison directly).
func applyEffect(target StatusHolder, effect string, rng interface{ Float64() float64 }, isMagic bool) (applied bool, resisted bool) {
	if p, ok := target.(*Party); ok {
		if p.resistsStatus(isMagic, rng) {
			return false, true
		}
	}
	switch effect {
	case "hex":
		target.ApplyStatus(StatusHex, 10)
		return true, false
	case "rend":
		target.ApplyStatus(StatusRend, 6)
		target.ApplyStatus(StatusBleed, 6)
		return true, false
	case "entangle":
		target.ApplyStatus(StatusEntangle, 4)
		return true, false
	case "spore":
		target.ApplyStatus(StatusSpore, 8)
		return true, false
	case "blind", "blindness":
		target.ApplyStatus(StatusBlind, 20)
		return true, false
	case "haste":
		target.ApplyStatus(StatusHaste, 50)
		return true, false
	case "slow":
		target.ApplyStatus(StatusSlow, 30)
		return true, false
	case "silence":
		target.ApplyStatus(StatusSilence, 6)
		return true, false
	case "stun":
		target.ApplyStatus(StatusStun, 2)
		return true, false
	case "poison":
		target.ApplyStatus(StatusPoison, 6)
		return true, false
	case "poison_short":
		target.ApplyStatus(StatusPoison, 4)
		return true, false
	default:
		return false, false
	}
}

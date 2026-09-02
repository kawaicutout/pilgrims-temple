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

// Effective DEF modifiers from statuses.
// hex -1, bless +1, curse -1. Stack not, but apply net.
func (p *Party) effectiveDEFDelta() int {
	delta := 0
	if p.HasStatus(StatusHex) {
		delta -= 1
	}
	if p.HasStatus(StatusBless) {
		delta += 1
	}
	if p.HasStatus(StatusCurse) {
		delta -= 1
	}
	return delta
}
func (e *EnemyParty) effectiveDEFDelta() int {
	delta := 0
	if e.HasStatus(StatusHex) {
		delta -= 1
	}
	if e.HasStatus(StatusBless) {
		delta += 1
	}
	if e.HasStatus(StatusCurse) {
		delta -= 1
	}
	return delta
}

// StatusResistCheck returns true if party resists status application via Iron Will / Ward.
func (p *Party) resistsStatus(isMagic bool, rng interface{ Float64() float64 }) bool {
	if p == nil {
		return false
	}
	// Iron Will: +10% resist vs status application per living member? Check any member has talent.
	for _, m := range p.Members {
		if m.IsAlive() && m.HasTalent("iron_will") {
			if rng != nil && rng.Float64() < 0.10 {
				return true
			}
		}
		if isMagic && m.IsAlive() && m.HasTalent("ward") {
			if rng != nil && rng.Float64() < 0.05 {
				return true
			}
		}
	}
	return false
}

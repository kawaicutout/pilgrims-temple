package game

import "math/rand/v2"

// RollRaw picks uniformly in [min,max] (no DEF).
func RollRaw(rng *rand.Rand, atkMin, atkMax int) int {
	if atkMax < atkMin {
		atkMax = atkMin
	}
	if atkMax > atkMin {
		return atkMin + rng.IntN(atkMax-atkMin+1)
	}
	return atkMin
}

// RollDamage picks uniformly then subtracts DEF, floor 1.
func RollDamage(rng *rand.Rand, atkMin, atkMax, def int) int {
	dmg := RollRaw(rng, atkMin, atkMax) - def
	if dmg < 1 {
		dmg = 1
	}
	return dmg
}

// RollDamageWithDefense picks uniformly then subtracts appropriate defense.
func RollDamageWithDefense(rng *rand.Rand, atkMin, atkMax int, defender *Member, isMagic bool) int {
	def := defender.DEF
	if isMagic {
		def = defender.MDEF
	}
	return RollDamage(rng, atkMin, atkMax, def)
}

// PlayerBumpEnemy handles player party bumping an enemy party.
// Returns damage dealt, index of enemy member hit, and whether that member died.
func PlayerBumpEnemy(rng *rand.Rand, party *Party, enemy *EnemyParty) (dmg int, hitIdx int, killed bool) {
	party.EnsureSelection()
	enemy.EnsureActive()
	atk := party.Members[party.Active]
	// veterans_grip +1 dmg fighter BuffB per bearer with talent
	extraGrip := 0
	if atk.HasTalent("veterans_grip") {
		extraGrip = 1
	}
	// Pick target in enemy party active-weighted
	hitIdx = pickEnemyTarget(rng, enemy)
	if hitIdx < 0 {
		return 0, -1, false
	}
	target := enemy.Members[hitIdx]
	isMagic := atk.DamageType == "magic"
	dmg = RollRaw(rng, atk.ATK[0]+extraGrip, atk.ATK[1]+extraGrip)
	if party.HasStatus(StatusStrength) {
		dmg += 2
	}
	// radiant +50% vs undead (check enemy ID contains undead/skeleton/zombie/ghost)
	isUndead := false
	if target != nil {
		idLower := target.Class
		if containsUndead(idLower) {
			isUndead = true
		}
	}
	if isUndead && atk.HasTalent("radiant") {
		dmg = (dmg * 3) / 2
	}
	// of_wrath sole survivor +2 outgoing when LivingCount==1 and attacker has affix
	if party.LivingCount() == 1 && atk.HasAffix("of_wrath") {
		dmg += 2
	}
	// Apply DEF or MDEF of target, with enemy hex/bless/curse.
	def := target.DEF
	if isMagic {
		def = target.MDEF
	}
	def += enemy.effectiveDEFDelta()
	actual := dmg - def
	if actual < 1 {
		actual = 1
	}
	// Enemy fire resist reduces fire damage (heuristic: magic fire)
	if enemy.HasStatus(StatusFireResist) && isMagic {
		actual = (actual * 7) / 10
		if actual < 1 {
			actual = 1
		}
	}
	target.HP -= actual
	dmg = actual // return actual after DEF for log
	if target.HP <= 0 {
		target.HP = 0
		target.Alive = false
		killed = true
	}
	// deitys_gift heal 2 on attack — blocked by silence
	if !party.HasStatus(StatusSilence) && atk.HasTalent("deitys_gift") && atk.IsAlive() && atk.HP < atk.MaxHP {
		atk.HP += 2
		if atk.HP > atk.MaxHP {
			atk.HP = atk.MaxHP
		}
	}
	// shrug 20% clear one negative status from self on attack — blocked by silence
	if !party.HasStatus(StatusSilence) && atk.HasTalent("shrug") && rng != nil && rng.Float64() < 0.20 {
		for _, sid := range []string{StatusHex, StatusRend, StatusBleed, StatusSpore, StatusPoison, StatusCurse, StatusParalysis, StatusConfusion, StatusEntangle, StatusSleep, StatusBlind, StatusSilence, StatusStun, StatusSlow} {
			if party.HasStatus(sid) {
				party.RemoveStatus(sid)
				break
			}
		}
	}
	// cleave overflow on kill: if HasTalent("cleave") and killed then 2 dmg to every other member on tile
	if killed && atk.HasTalent("cleave") {
		for i, m := range enemy.Members {
			if i != hitIdx && m.IsAlive() {
				m.HP -= 2
				if m.HP <= 0 {
					m.HP = 0
					m.Alive = false
				}
			}
		}
	}
	return dmg, hitIdx, killed
}

func containsUndead(id string) bool {
	low := id
	// simple contains check
	if len(low) >= 6 {
		for _, kw := range []string{"undead", "skeleton", "zombie", "ghost", "ghoul", "wraith", "lich"} {
			if len(kw) > len(low) {
				continue
			}
			for i := 0; i <= len(low)-len(kw); i++ {
				if low[i:i+len(kw)] == kw {
					return true
				}
			}
		}
	}
	return false
}

// pickEnemyTarget selects a living member of enemy party to hit, active-weighted.
func pickEnemyTarget(rng *rand.Rand, e *EnemyParty) int {
	n := e.LivingCount()
	if n == 0 {
		return -1
	}
	if n == 1 {
		for i, m := range e.Members {
			if m.IsAlive() {
				return i
			}
		}
	}
	// Active-weighted via Tuning.Targeting.ActiveWeight (default 0.5).
	weight := GetTuning().Targeting.ActiveWeight
	if weight <= 0 || weight > 1 {
		if weight == 0 {
			weight = 0.5
		} else if weight < 0 {
			weight = 0
		} else {
			weight = 1
		}
	}
	if rng.Float64() < weight {
		if e.Members[e.Active].IsAlive() {
			return e.Active
		}
	}
	// Uniform among living
	var living []int
	for i, m := range e.Members {
		if m.IsAlive() {
			living = append(living, i)
		}
	}
	return living[rng.IntN(len(living))]
}

// EnemyAttack handles one enemy party's turn: pick active member, hit player party.
func EnemyAttack(rng *rand.Rand, enemy *EnemyParty, party *Party) (attackerIdx int, dmgRaw int, hitPlayerIdx int) {
	enemy.EnsureActive()
	party.EnsureSelection()
	atk := enemy.Members[enemy.Active]
	dmgRaw = RollRaw(rng, atk.ATK[0], atk.ATK[1])
	// Apply to player via active-weighted (handled in Party.ApplyDamage, which does DEF/MDEF branching via type)
	// For log we need to know which player member was hit, but ApplyDamage picks internally.
	// We can simulate picking here for log, but ApplyDamage will pick again (double). Instead, we should have ApplyDamage return hit index.
	// For now, just return raw and let caller handle.
	attackerIdx = enemy.Active
	return
}


// DefenderDefense returns DEF or MDEF based on attacker damage type.
func DefenderDefense(attacker *Member, defender *Member) int {
	if attacker != nil && attacker.DamageType == "magic" {
		return defender.MDEF
	}
	return defender.DEF
}

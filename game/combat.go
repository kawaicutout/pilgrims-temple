package game

import (
	"math/rand/v2"
)

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

// PlayerBumpEnemy handles player party bumping an enemy party.
// Returns damage dealt, index of enemy member hit, and whether that member died.
func PlayerBumpEnemy(rng *rand.Rand, party *Party, enemy *EnemyParty) (dmg int, hitIdx int, killed bool) {
	party.EnsureSelection()
	enemy.EnsureActive()
	atk := party.Members[party.Active]
	// Pick target in enemy party active-weighted
	hitIdx = pickEnemyTarget(rng, enemy)
	if hitIdx < 0 {
		return 0, -1, false
	}
	target := enemy.Members[hitIdx]
	dmg = RollRaw(rng, atk.ATK[0], atk.ATK[1])
	// Apply DEF of target
	actual := dmg - target.DEF
	if actual < 1 {
		actual = 1
	}
	target.HP -= actual
	dmg = actual // return actual after DEF for log
	if target.HP <= 0 {
		target.HP = 0
		target.Alive = false
		killed = true
		// EnsureActive will be called next turn
	}
	return dmg, hitIdx, killed
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
	// 50% to active, 50% uniform
	if rng.Float64() < 0.5 {
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
	// Apply to player via active-weighted (handled in Party.ApplyDamage, which does DEF)
	// For log we need to know which player member was hit, but ApplyDamage picks internally.
	// We can simulate picking here for log, but ApplyDamage will pick again (double). Instead, we should have ApplyDamage return hit index.
	// For now, just return raw and let caller handle.
	attackerIdx = enemy.Active
	return
}

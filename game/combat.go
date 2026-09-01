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

// RollDamage picks uniformly then subtracts DEF, floor 1. Kept for direct target (e.g. enemy HP).
func RollDamage(rng *rand.Rand, atkMin, atkMax, def int) int {
	dmg := RollRaw(rng, atkMin, atkMax) - def
	if dmg < 1 {
		dmg = 1
	}
	return dmg
}

// PlayerBumpEnemy handles player bump into enemy.
func PlayerBumpEnemy(rng *rand.Rand, party *Party, enemy *EnemyParty) (dmgDealt, dmgTakenRaw int, killed bool) {
	party.EnsureSelection()
	atk := party.Members[party.Active]
	dmgDealt = RollRaw(rng, atk.ATK[0], atk.ATK[1])
	enemy.HP -= dmgDealt
	if enemy.HP <= 0 {
		enemy.HP = 0
		enemy.Alive = false
		killed = true
		return
	}
	dmgTakenRaw = RollRaw(rng, enemy.ATK[0], enemy.ATK[1])
	return
}

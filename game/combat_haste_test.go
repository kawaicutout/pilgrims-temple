package game

import (
	"math/rand/v2"
	"testing"
)

func strikeParties() (*Party, *EnemyParty) {
	hero := &Member{Name: "Hero", Class: "fighter", HP: 30, MaxHP: 30, ATK: [2]int{10, 10}, Alive: true}
	party := &Party{Members: []*Member{hero}}
	m1 := &Member{Name: "Gob1", Class: "goblin", HP: 100, MaxHP: 100, ATK: [2]int{1, 1}, Alive: true}
	m2 := &Member{Name: "Gob2", Class: "goblin", HP: 100, MaxHP: 100, ATK: [2]int{1, 1}, Alive: true}
	return party, &EnemyParty{Members: []*Member{m1, m2}}
}

// Slow takes a flat quarter off rolled damage: 10 -> 8.
func TestSlowReducesStrike(t *testing.T) {
	party, enemy := strikeParties()
	party.Members[0].ApplyStatus(StatusSlow, 10)
	dmg, _, _, _ := PlayerBumpEnemy(rand.New(rand.NewPCG(1, 1)), party, enemy)
	if dmg != 8 {
		t.Fatalf("slowed strike = %d, want 8", dmg)
	}
}

// Haste fires a 50% second strike on the same target: over many seeds at
// least one double lands for 20 total on the hit member, and doubles
// never spread across members.
func TestHasteSecondStrike(t *testing.T) {
	fired := 0
	for seed := uint64(0); seed < 40; seed++ {
		party, enemy := strikeParties()
		party.Members[0].ApplyStatus(StatusHaste, 50)
		dmg, hitIdx, _, struckTwice := PlayerBumpEnemy(rand.New(rand.NewPCG(seed, seed)), party, enemy)
		if !struckTwice {
			continue
		}
		fired++
		if dmg != 20 {
			t.Fatalf("seed %d: double strike total = %d, want 20", seed, dmg)
		}
		if got := enemy.Members[hitIdx].HP; got != 80 {
			t.Fatalf("seed %d: hit member HP = %d, want 80", seed, got)
		}
		for i, m := range enemy.Members {
			if i != hitIdx && m.HP != 100 {
				t.Fatalf("seed %d: off-target member %d took damage", seed, i)
			}
		}
	}
	if fired == 0 {
		t.Fatal("haste never fired a second strike in 40 seeds")
	}
}

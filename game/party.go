package game

import (
	"fmt"
	"math/rand/v2"
)

// Member is one character in the party.
type Member struct {
	Name   string
	Class  string
	HP     int
	MaxHP  int
	ATK    [2]int // min,max
	DEF    int
	Light  int
	Alive  bool
}

func (m *Member) IsAlive() bool { return m.Alive && m.HP > 0 }

// Party is 1-4 members sharing one tile.
type Party struct {
	Members []*Member
	Pos     Pos
	// Selected is UI cursor (q/w/e/r), free. Active is last actor.
	Selected int
	Active   int
}

func (p *Party) LivingCount() int {
	n := 0
	for _, m := range p.Members {
		if m.IsAlive() {
			n++
		}
	}
	return n
}

func (p *Party) LivingMembers() []*Member {
	var out []*Member
	for _, m := range p.Members {
		if m.IsAlive() {
			out = append(out, m)
		}
	}
	return out
}

func (p *Party) BestLight() int {
	best := 0
	for _, m := range p.Members {
		if m.IsAlive() && m.Light > best {
			best = m.Light
		}
	}
	if best == 0 {
		return 6 // default for M1
	}
	return best
}

func (p *Party) EnsureSelection() {
	if len(p.Members) == 0 {
		return
	}
	if p.Selected < 0 || p.Selected >= len(p.Members) || !p.Members[p.Selected].IsAlive() {
		for i := range p.Members {
			if p.Members[i].IsAlive() {
				p.Selected = i
				break
			}
		}
	}
	if p.Active < 0 || p.Active >= len(p.Members) || !p.Members[p.Active].IsAlive() {
		for i := range p.Members {
			if p.Members[i].IsAlive() {
				p.Active = i
				break
			}
		}
	}
}

func (p *Party) ApplyDamage(rng *rand.Rand, raw int) {
	// Active-weighted targeting (DESIGN 3.4). raw is pre-DEF roll; DEF of the hit member is subtracted here.
	n := p.LivingCount()
	if n == 0 {
		return
	}
	apply := func(m *Member) {
		dmg := raw - m.DEF
		if dmg < 1 {
			dmg = 1
		}
		m.HP -= dmg
		if m.HP <= 0 {
			m.HP = 0
			m.Alive = false
		}
	}
	if n == 1 {
		for _, m := range p.Members {
			if m.IsAlive() {
				apply(m)
				break
			}
		}
		return
	}
	// Build weight: active 0.5 + 0.5/N each.
	// P(active)=0.5+0.5/N, P(other)=0.5/N.
	p.EnsureSelection()
	// Find active member pointer
	activeIdx := p.Active
	// Collect living indices
	var livingIdx []int
	for i, m := range p.Members {
		if m.IsAlive() {
			livingIdx = append(livingIdx, i)
		}
	}
	// Roll
	r := rng.Float64()
	// Threshold for active
	threshold := 0.5 + 0.5/float64(n)
	if r < threshold {
		m := p.Members[activeIdx]
		if !m.IsAlive() {
			// fallback to random living
			m = p.Members[livingIdx[rng.IntN(len(livingIdx))]]
		}
		apply(m)
	} else {
		// Distribute among all living uniformly for remaining 0.5
		idx := livingIdx[rng.IntN(len(livingIdx))]
		m := p.Members[idx]
		apply(m)
	}
}

// GenerateParty creates starting party (2 members per DESIGN 4.2, but M1 solo =1 for now; we support 1-2).
func GenerateParty(rng *rand.Rand, level int) *Party {
	// For M1, solo: 1 fighter-like. For full, 2 random classes soon.
	classes := []string{"fighter", "cleric", "rogue", "wizard", "druid", "bard", "barbarian", "paladin"}
	// M1: single member at level 1
	m := generateMember(rng, classes[rng.IntN(len(classes))], level)
	p := &Party{Members: []*Member{m}, Pos: Pos{0, 0}, Selected: 0, Active: 0}
	m.Alive = true
	return p
}

func generateMember(rng *rand.Rand, class string, level int) *Member {
	// Base stats per class (M1 placeholder)
	hp := 20 + rng.IntN(6) + (level-1)*4
	atkMin := 3 + (level-1)
	atkMax := atkMin + 2 + rng.IntN(2)
	names := []string{"Ari", "Bren", "Cael", "Dara", "Emri", "Fenn", "Garr", "Hale"}
	name := names[rng.IntN(len(names))]
	if rng.IntN(3) == 0 {
		name += fmt.Sprintf("-%d", rng.IntN(100))
	}
	return &Member{
		Name:  name,
		Class: class,
		HP:    hp, MaxHP: hp,
		ATK:   [2]int{atkMin, atkMax},
		DEF:   rng.IntN(2),
		Light: 6 + rng.IntN(2),
		Alive: true,
	}
}

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

func (p *Party) ApplyDamage(rng *rand.Rand, raw int) (hitIdx int, actual int) {
	// Active-weighted targeting (DESIGN 3.4). raw is pre-DEF roll; DEF of the hit member is subtracted here.
	// Returns index of member hit and actual damage after DEF.
	n := p.LivingCount()
	if n == 0 {
		return -1, 0
	}
	var target *Member
	var idx int
	if n == 1 {
		for i, m := range p.Members {
			if m.IsAlive() {
				target = m
				idx = i
				break
			}
		}
	} else {
		p.EnsureSelection()
		activeIdx := p.Active
		var livingIdx []int
		for i, m := range p.Members {
			if m.IsAlive() {
				livingIdx = append(livingIdx, i)
			}
		}
		r := rng.Float64()
		if r < 0.5 {
			if p.Members[activeIdx].IsAlive() {
				target = p.Members[activeIdx]
				idx = activeIdx
			} else {
				idx = livingIdx[rng.IntN(len(livingIdx))]
				target = p.Members[idx]
			}
		} else {
			idx = livingIdx[rng.IntN(len(livingIdx))]
			target = p.Members[idx]
		}
	}
	actual = raw - target.DEF
	if actual < 1 {
		actual = 1
	}
	target.HP -= actual
	if target.HP <= 0 {
		target.HP = 0
		target.Alive = false
	}
	return idx, actual
}

// GenerateParty creates starting party (M1 solo 1, M2+ 2 per DESIGN 4.2).
func GenerateParty(rng *rand.Rand, level int) *Party {
	// For M2, generate 2 random distinct classes
	classes := []string{"fighter", "cleric", "rogue", "wizard", "druid", "bard", "barbarian", "paladin"}
	// Pick 2 distinct
	a := classes[rng.IntN(len(classes))]
	b := a
	for b == a {
		b = classes[rng.IntN(len(classes))]
	}
	return GeneratePartyWithClasses(rng, []string{a, b}, level)
}

// GeneratePartyWithClasses creates a party from explicit class list (for character select).
func GeneratePartyWithClasses(rng *rand.Rand, classes []string, level int) *Party {
	if len(classes) == 0 {
		classes = []string{"fighter"}
	}
	if len(classes) > 4 {
		classes = classes[:4]
	}
	var members []*Member
	for _, c := range classes {
		m := generateMember(rng, c, level)
		m.Alive = true
		members = append(members, m)
	}
	return &Party{Members: members, Pos: Pos{0, 0}, Selected: 0, Active: 0}
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

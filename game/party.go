package game

import (
	"encoding/json"
	"math/rand/v2"
	"strings"
)

// Member is one character in the party.
type Member struct {
	Name         string
	Class        string
	HP           int
	MaxHP        int
	ATK          [2]int // min,max
	DEF          int
	MDEF         int
	Light        int
	Carry        int
	Alive        bool
	Talents      []string
	Affixes      []string
	DamageType   string // physical or magic; empty means physical
	Effect       string
	EffectChance float64
	Regen        bool
	XP           int
	Color        string
}

func (m *Member) HasTalent(id string) bool {
	for _, t := range m.Talents {
		if t == id {
			return true
		}
	}
	return false
}

func (m *Member) HasAffix(id string) bool {
	for _, a := range m.Affixes {
		if a == id {
			return true
		}
	}
	return false
}

func (m *Member) IsAlive() bool { return m.Alive && m.HP > 0 }

// Party is 1-4 members sharing one tile.
type Party struct {
	Members []*Member
	Pos     Pos
	// Selected is UI cursor (q/w/e/r), free. Active is last actor.
	Selected int
	Active   int
	Inventory []GroundItem `json:"inventory"`
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

// HasRogue reports whether party has a living rogue (for vault locks / pitfall detection).
func (p *Party) HasRogue() bool {
	for _, m := range p.Members {
		if m.IsAlive() && m.Class == "rogue" {
			return true
		}
	}
	return false
}

// HasWizard reports whether party has a living wizard (for pitfall detection / vault).
func (p *Party) HasWizard() bool {
	for _, m := range p.Members {
		if m.IsAlive() && m.Class == "wizard" {
			return true
		}
	}
	return false
}

// HasClass reports whether party has a living member of given class.
func (p *Party) HasClass(class string) bool {
	for _, m := range p.Members {
		if m.IsAlive() && m.Class == class {
			return true
		}
	}
	return false
}

func (p *Party) CarryCapacity() int {
	sum := 0
	for _, m := range p.Members {
		if m.IsAlive() {
			c := m.Carry
			if c == 0 {
				c = 5 // fixup for old saves / defaults
			}
			sum += c
		}
	}
	return sum
}
// CarryUsed is sum of inventory weight (1 per potion/scroll).
func (p *Party) CarryUsed() int {
	if p == nil {
		return 0
	}
	return len(p.Inventory)
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
	return p.ApplyDamageWithType(rng, raw, false)
}

// ApplyDamageWithType applies raw damage choosing DEF vs MDEF based on isMagic.
func (p *Party) ApplyDamageWithType(rng *rand.Rand, raw int, isMagic bool) (hitIdx int, actual int) {
	// Active-weighted targeting (DESIGN 3.4). raw is pre-DEF roll; DEF/MDEF of the hit member is subtracted here.
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
	def := target.DEF
	if isMagic {
		def = target.MDEF
	}
	actual = raw - def
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
	used := map[string]bool{}
	var members []*Member
	for _, c := range classes {
		m := generateMember(rng, c, level, used)
		m.Alive = true
		used[m.Name] = true
		members = append(members, m)
	}
	return &Party{Members: members, Pos: Pos{0, 0}, Selected: 0, Active: 0}
}

// classStats holds per-class tuning from classes.json.
type classStats struct {
	HitDice      string `json:"hitDice"`
	Attack       int    `json:"attack"`
	Defense      int    `json:"defense"`
	MagicDefense int    `json:"magicDefense"`
}

var classStatsCache map[string]classStats

func loadClassStats() map[string]classStats {
	if classStatsCache != nil {
		return classStatsCache
	}
	b, err := RawJSON("classes.json")
	if err != nil {
		classStatsCache = map[string]classStats{}
		return classStatsCache
	}
	var raw struct {
		Classes []struct {
			ID           string `json:"id"`
			HitDice      string `json:"hitDice"`
			Attack       int    `json:"attack"`
			Defense      int    `json:"defense"`
			MagicDefense int    `json:"magicDefense"`
		} `json:"classes"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		classStatsCache = map[string]classStats{}
		return classStatsCache
	}
	m := map[string]classStats{}
	for _, c := range raw.Classes {
		m[c.ID] = classStats{HitDice: c.HitDice, Attack: c.Attack, Defense: c.Defense, MagicDefense: c.MagicDefense}
	}
	classStatsCache = m
	return m
}

func hitDiceSides(s string) int {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "d4":
		return 4
	case "d6":
		return 6
	case "d8":
		return 8
	case "d10":
		return 10
	case "d12":
		return 12
	default:
		return 6
	}
}

func rollHD(rng *rand.Rand, sides int) int {
	if rng == nil {
		return (sides + 1) / 2
	}
	return 1 + rng.IntN(sides)
}

func generateMember(rng *rand.Rand, class string, level int, used map[string]bool) *Member {
	if level < 1 {
		level = 1
	}
	stats := loadClassStats()[class]
	// Defaults if class not found
	sides := hitDiceSides(stats.HitDice)
	if stats.HitDice == "" {
		sides = 6
		if class == "fighter" || class == "paladin" || class == "barbarian" {
			sides = 10
		} else if class == "wizard" {
			sides = 4
		} else if class == "cleric" || class == "druid" {
			sides = 8
		} else {
			sides = 6
		}
	}
	baseATK := stats.Attack
	baseDEF := stats.Defense
	baseMDEF := stats.MagicDefense
	// Fallbacks if JSON missing fields
	if stats.HitDice == "" && stats.Attack == 0 && stats.Defense == 0 && stats.MagicDefense == 0 {
		switch class {
		case "fighter":
			baseATK, baseDEF, baseMDEF = 3, 2, 1
		case "paladin":
			baseATK, baseDEF, baseMDEF = 3, 2, 1
		case "barbarian":
			baseATK, baseDEF, baseMDEF = 3, 2, 1
		case "cleric":
			baseATK, baseDEF, baseMDEF = 1, 1, 2
		case "druid":
			baseATK, baseDEF, baseMDEF = 1, 1, 2
		case "rogue":
			baseATK, baseDEF, baseMDEF = 2, 1, 1
		case "bard":
			baseATK, baseDEF, baseMDEF = 1, 1, 2
		case "wizard":
			baseATK, baseDEF, baseMDEF = 1, 0, 3
		default:
			baseATK, baseDEF, baseMDEF = 2, 1, 1
		}
	}
	// HP: 10 + 3*HD at level 1, +1 HD per additional level
	diceCount := 3 + (level - 1)
	hp := 10
	for range diceCount {
		hp += rollHD(rng, sides)
	}
	// Attack range: base + (level-1) with small variance
	atkMin := baseATK + (level - 1)
	atkMax := atkMin + 2 + rng.IntN(2)
	if used == nil {
		used = map[string]bool{}
	}
	name := GenerateName(rng, used)
	return &Member{
		Name:       name,
		Class:      class,
		HP:         hp,
		MaxHP:      hp,
		ATK:        [2]int{atkMin, atkMax},
		DEF:        baseDEF,
		MDEF:       baseMDEF,
		Light:      6 + rng.IntN(2),
		Carry:      5,
		Alive:      true,
		DamageType: "physical",
	}
}

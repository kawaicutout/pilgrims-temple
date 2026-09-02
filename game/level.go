package game

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
)

// Level holds one floor.
type Level struct {
	W, H    int
	Tiles   [][]Tile
	Seen    [][]bool // explored
	Visible [][]bool

	StairsUp   Pos
	StairsDown Pos

	Enemies  []*EnemyParty
	Features []Feature
	Items    []GroundItem `json:"items"`

	// Biome integration
	BiomeID string      `json:"biomeId"`
	Floor   int         `json:"floor"`
	Litter  []LitterObj `json:"litter"`

	// Doors holds door open state: true = open, false/absent = closed.
	Doors map[Pos]bool `json:"doors"`
}

func NewLevel(w, h int) *Level {
	tiles := make([][]Tile, h)
	seen := make([][]bool, h)
	vis := make([][]bool, h)
	for y := range h {
		tiles[y] = make([]Tile, w)
		seen[y] = make([]bool, w)
		vis[y] = make([]bool, w)
		for x := range w {
			tiles[y][x] = TileWall
		}
	}
	return &Level{W: w, H: h, Tiles: tiles, Seen: seen, Visible: vis, Doors: make(map[Pos]bool)}
}

func (l *Level) IsDoor(p Pos) bool { return l.At(p) == TileDoor }

func (l *Level) IsDoorOpen(p Pos) bool {
	if !l.IsDoor(p) {
		return false
	}
	if l.Doors == nil {
		return false
	}
	return l.Doors[p]
}

func (l *Level) IsDoorClosed(p Pos) bool { return l.IsDoor(p) && !l.IsDoorOpen(p) }

func (l *Level) SetDoorOpen(p Pos, open bool) {
	if l.Doors == nil {
		l.Doors = make(map[Pos]bool)
	}
	if !l.IsDoor(p) {
		return
	}
	l.Doors[p] = open
}

func (l *Level) DoorGlyph(p Pos) rune {
	if !l.IsDoor(p) {
		return l.At(p).Glyph()
	}
	if l.IsDoorOpen(p) {
		return '\''
	}
	return '+'
}

func (l *Level) InBounds(p Pos) bool { return p.X >= 0 && p.X < l.W && p.Y >= 0 && p.Y < l.H }
func (l *Level) At(p Pos) Tile {
	if !l.InBounds(p) {
		return TileWall
	}
	return l.Tiles[p.Y][p.X]
}
func (l *Level) Set(p Pos, t Tile) {
	if l.InBounds(p) {
		l.Tiles[p.Y][p.X] = t
	}
}
func (l *Level) Walkable(p Pos) bool {
	t := l.At(p)
	if t == TileDoor {
		if l.IsDoorClosed(p) {
			return false
		}
		// open door is walkable (subject to litter)
	} else if !t.Walkable() {
		return false
	}
	for _, lit := range l.Litter {
		if lit.Pos == p && lit.BlocksMovement {
			return false
		}
	}
	return true
}
func (l *Level) BlocksFOV(p Pos) bool {
	if l.At(p) == TileDoor {
		if l.IsDoorClosed(p) {
			return true
		}
		return false
	}
	if l.At(p).BlocksFOV() {
		return true
	}
	for _, lit := range l.Litter {
		if lit.Pos == p && lit.BlocksFOV {
			return true
		}
	}
	return false
}

// LitterAt returns litter at p if any.
func (l *Level) LitterAt(p Pos) *LitterObj {
	for i := range l.Litter {
		if l.Litter[i].Pos == p {
			return &l.Litter[i]
		}
	}
	return nil
}

// EnemyParty is a party of 1-4 monsters sharing one tile (DESIGN 3.1).
// Members share Pos; Active selects the actor per turn.
type EnemyParty struct {
	Pos     Pos
	Members []*Member
	Active  int
}

func (e *EnemyParty) IsAlive() bool {
	for _, m := range e.Members {
		if m.IsAlive() {
			return true
		}
	}
	return false
}

func (e *EnemyParty) LivingCount() int {
	n := 0
	for _, m := range e.Members {
		if m.IsAlive() {
			n++
		}
	}
	return n
}

func (e *EnemyParty) Color() string {
	for _, m := range e.Members {
		if m.IsAlive() && m.Color != "" {
			return m.Color
		}
	}
	for _, m := range e.Members {
		if m.Color != "" {
			return m.Color
		}
	}
	return "#c96a5a"
}

func (e *EnemyParty) MemberColor(idx int) string {
	if idx >= 0 && idx < len(e.Members) && e.Members[idx].Color != "" {
		return e.Members[idx].Color
	}
	return e.Color()
}

func (e *EnemyParty) Glyph() rune {
	for _, m := range e.Members {
		if m.IsAlive() {
			switch m.Class {
			case "goblin":
				return 'g'
			case "orc":
				return 'o'
			case "kobold":
				return 'k'
			case "rat":
				return 'r'
			case "troll":
				return 'T'
			default:
				if len(m.Name) > 0 {
					return rune(m.Name[0])
				}
				return 'e'
			}
		}
	}
	return 'e'
}

func (e *EnemyParty) DisplayName() string {
	// For single, just name; for party show "goblin x3" etc. Used for bump log when party not yet numbered.
	if len(e.Members) == 1 {
		return e.Members[0].Name
	}
	// Count by class
	counts := map[string]int{}
	for _, m := range e.Members {
		if m.IsAlive() {
			counts[m.Class]++
		}
	}
	if len(counts) == 1 {
		for cls, n := range counts {
			return fmt.Sprintf("%s x%d", cls, n)
		}
	}
	return fmt.Sprintf("%s +%d", e.Members[0].Name, len(e.Members)-1)
}

// MemberDisplayName returns "goblin #1" style for a specific member index, numbered within its duplicate type.
func (e *EnemyParty) MemberDisplayName(idx int) string {
	if idx < 0 || idx >= len(e.Members) {
		return "unknown"
	}
	m := e.Members[idx]
	base := m.Class
	if base == "" {
		base = m.Name
	}
	// Count duplicates of same base up to idx
	num := 1
	total := 0
	for _, o := range e.Members {
		if o.Class == base || o.Name == base {
			total++
		}
	}
	if total == 1 {
		return base
	}
	for i, o := range e.Members {
		if o.Class == base || o.Name == base {
			if i == idx {
				break
			}
			num++
		}
	}
	return fmt.Sprintf("%s #%d", base, num)
}

func (e *EnemyParty) EnsureActive() {
	if len(e.Members) == 0 {
		return
	}
	if e.Active < 0 || e.Active >= len(e.Members) || !e.Members[e.Active].IsAlive() {
		for i, m := range e.Members {
			if m.IsAlive() {
				e.Active = i
				return
			}
		}
	}
}

// RegenTick heals regen members 1/turn if alive.
func (e *EnemyParty) RegenTick() {
	for _, m := range e.Members {
		if m.IsAlive() && m.Regen && m.HP < m.MaxHP {
			m.HP++
			if m.HP > m.MaxHP {
				m.HP = m.MaxHP
			}
		}
	}
}

// enemyEntry mirrors enemies.json.
type enemyEntry struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Glyph        string  `json:"glyph"`
	Color        string  `json:"color"`
	DamageType   string  `json:"damageType"`
	Effect       string  `json:"effect"`
	EffectChance float64 `json:"effectChance"`
	Regen        bool    `json:"regen"`
	XP           int     `json:"xp"`
	TalentChance float64 `json:"talentChance"`
	AffixChance  float64 `json:"affixChance"`
}

type enemiesFile struct {
	Enemies []enemyEntry `json:"enemies"`
}

var enemiesCache []enemyEntry

func loadEnemies() []enemyEntry {
	if enemiesCache != nil {
		return enemiesCache
	}
	b, err := RawJSON("enemies.json")
	if err != nil {
		enemiesCache = []enemyEntry{
			{ID: "goblin", Name: "Goblin", Glyph: "g", Color: "#5a7a5a", DamageType: "physical", Effect: "hex", EffectChance: 0.08, XP: 10, TalentChance: 0.08, AffixChance: 0.04},
			{ID: "orc", Name: "Orc", Glyph: "o", Color: "#8a7a6a", DamageType: "physical", Effect: "rend", EffectChance: 0.10, XP: 15, TalentChance: 0.10, AffixChance: 0.05},
			{ID: "kobold", Name: "Kobold", Glyph: "k", Color: "#6a7a7a", DamageType: "magic", Effect: "hex", EffectChance: 0.20, XP: 12, TalentChance: 0.12, AffixChance: 0.06},
			{ID: "rat", Name: "Rat", Glyph: "r", Color: "#7a7a7a", DamageType: "physical", Effect: "", EffectChance: 0.0, XP: 8, TalentChance: 0.04, AffixChance: 0.02},
			{ID: "troll", Name: "Troll", Glyph: "T", Color: "#6a8a6a", DamageType: "physical", Effect: "regenerate", EffectChance: 0.12, Regen: true, XP: 30, TalentChance: 0.18, AffixChance: 0.09},
		}
		return enemiesCache
	}
	var f enemiesFile
	if err := json.Unmarshal(b, &f); err != nil || len(f.Enemies) == 0 {
		enemiesCache = []enemyEntry{
			{ID: "goblin", Name: "Goblin", Glyph: "g", Color: "#5a7a5a", DamageType: "physical", XP: 10},
			{ID: "orc", Name: "Orc", Glyph: "o", Color: "#8a7a6a", DamageType: "physical", XP: 15},
			{ID: "kobold", Name: "Kobold", Glyph: "k", Color: "#6a7a7a", DamageType: "magic", XP: 12},
			{ID: "rat", Name: "Rat", Glyph: "r", Color: "#7a7a7a", DamageType: "physical", XP: 8},
			{ID: "troll", Name: "Troll", Glyph: "T", Color: "#6a8a6a", DamageType: "physical", Regen: true, XP: 30},
		}
		return enemiesCache
	}
	enemiesCache = f.Enemies
	return enemiesCache
}

// GetEnemyData returns a copy of enemies data for external use.
func GetEnemyData() []enemyEntry {
	src := loadEnemies()
	out := make([]enemyEntry, len(src))
	copy(out, src)
	return out
}
func pickEnemyForFloor(rng *rand.Rand, floor int) enemyEntry {
	entries := loadEnemies()
	var pool []enemyEntry
	for _, e := range entries {
		if e.ID == "troll" && floor < 3 {
			continue
		}
		if e.ID == "vine_horror" || e.ID == "spore_mother" {
			// Special biome enemies only via biome special spawn, not generic pool
			continue
		}
		pool = append(pool, e)
	}
	if len(pool) == 0 {
		pool = entries
	}
	return pool[rng.IntN(len(pool))]
}

// Generate fills a level with rooms+corridors and stairs. Deterministic from rng.
// Delegates to biome-aware generation (rooms vs cavern) and ensures palette/litter/features.
func (l *Level) Generate(rng *rand.Rand, floor int) {
	biome := GetBiomeForFloor(floor)
	l.GenerateWithBiome(rng, floor, biome)
}

// RegenerateEnemies clears current enemies and spawns fresh parties for floor.
// Used after relic pickup to repopulate old levels. Keeps map geometry intact.
func (l *Level) RegenerateEnemies(rng *rand.Rand, floor int) {
	l.Enemies = nil
	// Collect walkable positions not on stairs.
	var candidates []Pos
	for y := range l.H {
		for x := range l.W {
			p := Pos{x, y}
			if p == l.StairsUp || p == l.StairsDown {
				continue
			}
			if !l.Walkable(p) {
				continue
			}
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return
	}
	partyCount := 3 + floor*2 + rng.IntN(3)
	for range partyCount {
		var p Pos
		for range 100 {
			p = candidates[rng.IntN(len(candidates))]
			occupied := false
			for _, e := range l.Enemies {
				if e.Pos == p {
					occupied = true
					break
				}
			}
			if !occupied {
				break
			}
		}
		partySize := 1
		if floor >= 1 && rng.IntN(3) == 0 {
			partySize++
		}
		if floor >= 3 && rng.IntN(2) == 0 {
			partySize++
		}
		if floor >= 5 && rng.IntN(2) == 0 {
			partySize++
		}
		if partySize > 4 {
			partySize = 4
		}
		if floor >= 2 && rng.IntN(4) == 0 {
			partySize = 1 + rng.IntN(2)
		}
		ep := &EnemyParty{Pos: p, Active: 0}
		for range partySize {
			entry := pickEnemyForFloor(rng, floor)
			hp := 6 + floor*2 + rng.IntN(4)
			if entry.Regen {
				hp += 4
			}
			atkMin := 2 + floor
			atkMax := atkMin + 2 + rng.IntN(2)
			if entry.DamageType == "magic" && floor > 2 {
				atkMin++
				atkMax++
			}
			def := 0
			mdef := 0
			if floor >= 2 {
				def = floor / 3
				mdef = floor / 4
			}
			if entry.ID == "orc" {
				def++
			}
			if entry.ID == "kobold" {
				mdef++
			}
			if entry.ID == "troll" {
				def++
				mdef++
			}
			mem := &Member{
				Name: entry.Name, Class: entry.ID,
				HP: hp, MaxHP: hp,
				ATK: [2]int{atkMin, atkMax},
				DEF: def, MDEF: mdef,
				Alive: true, DamageType: entry.DamageType,
				Effect: entry.Effect, EffectChance: entry.EffectChance,
				Regen: entry.Regen, XP: entry.XP, Color: entry.Color,
			}
			if mem.EffectChance < 0 {
				mem.EffectChance = 0
			}
			if mem.EffectChance > 0.3 {
				mem.EffectChance = 0.3
			}
			if floor >= 3 {
				talentChance := entry.TalentChance
				affixChance := entry.AffixChance
				if floor >= 5 {
					talentChance += 0.05
					affixChance += 0.03
				}
				if talentChance > 0 && rng.Float64() < talentChance {
					opts := GetTalentOptions(rng, mem.Class, 1)
					if len(opts) > 0 {
						chosen := opts[0]
						mem.Talents = append(mem.Talents, chosen)
					}
				}
				if affixChance > 0 && rng.Float64() < affixChance {
					aff := GetRandomAffix(rng)
					mem.Affixes = append(mem.Affixes, aff)
				}
			}
			ep.Members = append(ep.Members, mem)
		}
		l.Enemies = append(l.Enemies, ep)
	}
}

// SpawnNewEnemies is an alias for RegenerateEnemies (level respawn hook).
func (l *Level) SpawnNewEnemies(rng *rand.Rand, floor int) { l.RegenerateEnemies(rng, floor) }

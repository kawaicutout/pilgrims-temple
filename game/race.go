package game

import (
	"encoding/json"
	"math/rand/v2"
	"strings"
	"sync"
)

// Buff holds per-stat bonuses.
type Buff struct {
	HP    int `json:"hp"`
	ATK   int `json:"atk"`
	DEF   int `json:"def"`
	MDEF  int `json:"mdef"`
	Light int `json:"light"`
	Carry int `json:"carry"`
}

// SynergyBuff holds synergy bonus; threshold for activation and XP bonus.
type SynergyBuff struct {
	Desc      string  `json:"desc"`
	Threshold int     `json:"threshold"`
	HP        int     `json:"hp"`
	ATK       int     `json:"atk"`
	DEF       int     `json:"def"`
	MDEF      int     `json:"mdef"`
	Light     int     `json:"light"`
	Carry     int     `json:"carry"`
	XPBonus   float64 `json:"xpBonus"`
}

// Race defines a playable race.
type Race struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Desc        string      `json:"desc"`
	CharBuff    Buff        `json:"charBuff"`
	PartyBuff   Buff        `json:"partyBuff"`
	SynergyBuff SynergyBuff `json:"synergyBuff"`
}

type racesFile struct {
	Races []Race `json:"races"`
}

var (
	racesCache map[string]Race
	racesList  []Race
	racesOnce  sync.Once
)

func fallbackRaces() []Race {
	return []Race{
		{ID: "human", Name: "Human", Desc: "+1 HP per level, +5% XP per Human (stacks)", CharBuff: Buff{HP: 1}, PartyBuff: Buff{}, SynergyBuff: SynergyBuff{Desc: "+5% XP per Human", Threshold: 1}},
		{ID: "elf", Name: "Elf", Desc: "+1 DEF, +2 light (non-stacking), Identify every 250 ticks -50 per extra Elf when 2+", CharBuff: Buff{DEF: 1}, PartyBuff: Buff{Light: 2}, SynergyBuff: SynergyBuff{Desc: "Identify interval -50 per extra Elf", Threshold: 2}},
		{ID: "dwarf", Name: "Dwarf", Desc: "+4 HP, +1 DEF, +2 dmg below 50% HP, tremorsense 4 +1 per extra Dwarf", CharBuff: Buff{HP: 4, DEF: 1}, PartyBuff: Buff{}, SynergyBuff: SynergyBuff{Desc: "Tremorsense 4 +1 per extra Dwarf", Threshold: 1}},
		{ID: "halfling", Name: "Halfling", Desc: "+1 HP, 5% avoid damage/negative, 10% extra item on pickup", CharBuff: Buff{HP: 1}, PartyBuff: Buff{}, SynergyBuff: SynergyBuff{Desc: "5% avoid, 10% extra loot", Threshold: 1}},
		{ID: "gnome", Name: "Gnome", Desc: "+2 MDEF, 10% instant identify first of kind, 5% per Gnome to not consume scroll", CharBuff: Buff{MDEF: 2}, PartyBuff: Buff{}, SynergyBuff: SynergyBuff{Desc: "5% per Gnome scroll save", Threshold: 1}},
		{ID: "half_orc", Name: "Half-orc", Desc: "+2 ATK, no ATK reductions, +1 ATK per other Half-orc", CharBuff: Buff{ATK: 2}, PartyBuff: Buff{}, SynergyBuff: SynergyBuff{Desc: "+1 ATK per other Half-orc", Threshold: 2}},
		{ID: "troll", Name: "Troll", Desc: "+4 HP +2 ATK, Regen every 3 ticks, double food cost, counts twice when >50% health", CharBuff: Buff{HP: 4, ATK: 2}, PartyBuff: Buff{}, SynergyBuff: SynergyBuff{Desc: "Regen 3 ticks, double food", Threshold: 1}},
	}
}

func normalizeRaceID(id string) string {
	s := strings.ToLower(strings.TrimSpace(id))
	s = strings.ReplaceAll(s, "-", "_")
	// alias halforc without underscore
	if s == "halforc" {
		s = "half_orc"
	}
	if s == "half_orc" {
		return "half_orc"
	}
	return s
}

func LoadRaces() []Race {
	racesOnce.Do(func() {
		b, err := RawJSON("races.json")
		if err != nil {
			list := fallbackRaces()
			racesCache = make(map[string]Race, len(list))
			for _, r := range list {
				racesCache[r.ID] = r
				// also allow normalized alias
				n := normalizeRaceID(r.ID)
				if n != r.ID {
					racesCache[n] = r
				}
			}
			racesList = list
			return
		}
		var rf racesFile
		if err := json.Unmarshal(b, &rf); err != nil || len(rf.Races) == 0 {
			list := fallbackRaces()
			racesCache = make(map[string]Race, len(list))
			for _, r := range list {
				racesCache[r.ID] = r
				n := normalizeRaceID(r.ID)
				if n != r.ID {
					racesCache[n] = r
				}
			}
			racesList = list
			return
		}
		// normalize defaults
		m := make(map[string]Race, len(rf.Races))
		for i := range rf.Races {
			if rf.Races[i].SynergyBuff.Threshold == 0 {
				rf.Races[i].SynergyBuff.Threshold = 2
			}
			m[rf.Races[i].ID] = rf.Races[i]
			n := normalizeRaceID(rf.Races[i].ID)
			if n != rf.Races[i].ID {
				m[n] = rf.Races[i]
			}
		}
		racesCache = m
		racesList = rf.Races
	})
	out := make([]Race, len(racesList))
	copy(out, racesList)
	return out
}

func GetRace(id string) (Race, bool) {
	LoadRaces()
	n := normalizeRaceID(id)
	r, ok := racesCache[n]
	if ok {
		return r, true
	}
	if r, ok := racesCache[id]; ok {
		return r, true
	}
	// fallback search
	for _, fr := range fallbackRaces() {
		if fr.ID == id || normalizeRaceID(fr.ID) == n {
			return fr, true
		}
	}
	return Race{}, false
}

// HasRace reports whether party has at least one living member of given race.
func (p *Party) HasRace(raceID string) bool {
	if p == nil {
		return false
	}
	n := normalizeRaceID(raceID)
	for _, m := range p.Members {
		if m.IsAlive() && normalizeRaceID(m.Race) == n {
			return true
		}
	}
	return false
}

// raceCount counts living members per race (normalized).
func raceCount(p *Party) map[string]int {
	m := map[string]int{}
	for _, mem := range p.Members {
		if mem.IsAlive() && mem.Race != "" {
			n := normalizeRaceID(mem.Race)
			m[n]++
		}
	}
	return m
}

// RaceCount is exported helper for tests.
func RaceCount(p *Party, raceID string) int {
	if p == nil {
		return 0
	}
	n := normalizeRaceID(raceID)
	c := 0
	for _, mem := range p.Members {
		if mem.IsAlive() && normalizeRaceID(mem.Race) == n {
			c++
		}
	}
	return c
}

// ApplyRaceBuffs applies party (non-stacking) and synergy buffs to party members.
// It is idempotent: repeated calls adjust by delta so stats don't double-stack.
// Character buffs are assumed already applied at generation; this only handles party/synergy.
// Call after GenerateParty, NewGame, and LevelUp.
func ApplyRaceBuffs(party *Party) {
	if party == nil || len(party.Members) == 0 {
		return
	}
	LoadRaces()
	counts := raceCount(party)
	// distinct races present among living
	distinct := map[string]bool{}
	for raceID, cnt := range counts {
		if cnt > 0 {
			distinct[raceID] = true
		}
	}
	// compute desired party bonus aggregate (sum of each distinct race's PartyBuff)
	// For elf light non-stacking, this naturally gives 2 once (distinct)
	var totalParty Buff
	for raceID := range distinct {
		if r, ok := GetRace(raceID); ok {
			totalParty.HP += r.PartyBuff.HP
			totalParty.ATK += r.PartyBuff.ATK
			totalParty.DEF += r.PartyBuff.DEF
			totalParty.MDEF += r.PartyBuff.MDEF
			totalParty.Light += r.PartyBuff.Light
			totalParty.Carry += r.PartyBuff.Carry
		}
	}
	for _, m := range party.Members {
		if !m.IsAlive() {
			continue
		}
		nrace := normalizeRaceID(m.Race)
		// desired synergy for this member
		var desiredSynergy SynergyBuff
		// generic synergy from data if threshold met
		if r, ok := GetRace(nrace); ok {
			if counts[nrace] >= r.SynergyBuff.Threshold && (r.SynergyBuff.HP != 0 || r.SynergyBuff.ATK != 0 || r.SynergyBuff.DEF != 0 || r.SynergyBuff.MDEF != 0 || r.SynergyBuff.Light != 0 || r.SynergyBuff.Carry != 0 || r.SynergyBuff.XPBonus != 0) {
				desiredSynergy = r.SynergyBuff
			}
		}
		// race-specific synergy overrides
		switch nrace {
		case "half_orc":
			cnt := counts[nrace]
			if cnt >= 2 {
				desiredSynergy.ATK = cnt - 1 // +1 per other half-orc
			} else {
				desiredSynergy.ATK = 0
			}
			// keep other synergy fields from generic if any (but half_orc has none)
		case "human", "elf", "dwarf", "halfling", "gnome", "troll":
			// no synergy stat bonuses beyond party; ensure no ATK etc from generic fallback
			// preserve only if explicitly set, but for our fallback they are zero
			// we already set desiredSynergy via generic; for these races we zero out unless needed
			// To avoid accidental HP from generic threshold, clear if race not half_orc
			// Actually human/elf/dwarf etc fallback synergy has no HP/ATK, so it's fine
		}
		// For non-half_orc, ensure ATK synergy not spuriously set via generic when threshold not met? Already handled.
		// Special: if race is half_orc and counts>=2 we want only ATK bonus, not other fields
		if nrace == "half_orc" && counts[nrace] >= 2 {
			// override to only ATK
			desiredSynergy.HP = 0
			desiredSynergy.DEF = 0
			desiredSynergy.MDEF = 0
			desiredSynergy.Light = 0
			desiredSynergy.Carry = 0
			desiredSynergy.XPBonus = 0
		} else if nrace != "half_orc" {
			// for other races, null out synergy ATK unless generic provided? Our fallback generics have 0 ATK, so fine
		}

		// delta for party buff (aggregate is same for all members, but we track per-member AppliedParty)
		deltaParty := Buff{
			HP:    totalParty.HP - m.AppliedParty.HP,
			ATK:   totalParty.ATK - m.AppliedParty.ATK,
			DEF:   totalParty.DEF - m.AppliedParty.DEF,
			MDEF:  totalParty.MDEF - m.AppliedParty.MDEF,
			Light: totalParty.Light - m.AppliedParty.Light,
			Carry: totalParty.Carry - m.AppliedParty.Carry,
		}
		deltaSynergy := Buff{
			HP:    desiredSynergy.HP - m.AppliedSynergy.HP,
			ATK:   desiredSynergy.ATK - m.AppliedSynergy.ATK,
			DEF:   desiredSynergy.DEF - m.AppliedSynergy.DEF,
			MDEF:  desiredSynergy.MDEF - m.AppliedSynergy.MDEF,
			Light: desiredSynergy.Light - m.AppliedSynergy.Light,
			Carry: desiredSynergy.Carry - m.AppliedSynergy.Carry,
		}
		// apply deltas
		if deltaParty.HP != 0 || deltaSynergy.HP != 0 {
			dhp := deltaParty.HP + deltaSynergy.HP
			m.MaxHP += dhp
			m.HP += dhp
			if m.HP > m.MaxHP {
				m.HP = m.MaxHP
			}
			if m.HP < 0 {
				m.HP = 0
			}
		}
		if deltaParty.ATK != 0 || deltaSynergy.ATK != 0 {
			da := deltaParty.ATK + deltaSynergy.ATK
			m.ATK[0] += da
			m.ATK[1] += da
		}
		if deltaParty.DEF != 0 || deltaSynergy.DEF != 0 {
			m.DEF += deltaParty.DEF + deltaSynergy.DEF
		}
		if deltaParty.MDEF != 0 || deltaSynergy.MDEF != 0 {
			m.MDEF += deltaParty.MDEF + deltaSynergy.MDEF
		}
		if deltaParty.Light != 0 || deltaSynergy.Light != 0 {
			m.Light += deltaParty.Light + deltaSynergy.Light
		}
		if deltaParty.Carry != 0 || deltaSynergy.Carry != 0 {
			m.Carry += deltaParty.Carry + deltaSynergy.Carry
		}
		// store applied
		m.AppliedParty = totalParty
		m.AppliedSynergy = Buff{
			HP:    desiredSynergy.HP,
			ATK:   desiredSynergy.ATK,
			DEF:   desiredSynergy.DEF,
			MDEF:  desiredSynergy.MDEF,
			Light: desiredSynergy.Light,
			Carry: desiredSynergy.Carry,
		}
	}
}

// SynergyXPBonus returns total XP multiplier bonus (e.g., 0.1 for 10%).
// Human: +5% per living Human (no synergy, multiplies by count).
func SynergyXPBonus(p *Party) float64 {
	if p == nil {
		return 0
	}
	counts := raceCount(p)
	bonus := 0.0
	if c, ok := counts["human"]; ok && c > 0 {
		bonus += float64(c) * 0.05
	}
	// Legacy: if other races have xpBonus and threshold met, include them (for compat)
	LoadRaces()
	for raceID, cnt := range counts {
		if raceID == "human" {
			continue
		}
		if r, ok := GetRace(raceID); ok {
			if cnt >= r.SynergyBuff.Threshold && r.SynergyBuff.XPBonus > 0 {
				bonus += r.SynergyBuff.XPBonus
			}
		}
	}
	return bonus
}

// SynergyTremorsense returns dwarven tremorsense radius: 4 tiles +1 per extra dwarf when >=1, else 0.
func SynergyTremorsense(p *Party) int {
	if p == nil {
		return 0
	}
	c := RaceCount(p, "dwarf")
	if c == 0 {
		return 0
	}
	return 4 + (c - 1)
}

// ElfIdentifyInterval returns the auto-identify interval for elves.
// No elves => 0 (no tick). 1 elf =>250, 2=>200, 3=>150 etc, minimum 50.
func ElfIdentifyInterval(p *Party) int {
	if p == nil {
		return 0
	}
	c := RaceCount(p, "elf")
	if c == 0 {
		return 0
	}
	iv := 250 - 50*(c-1)
	if iv < 50 {
		iv = 50
	}
	return iv
}

// SynergyIdentifyInterval alias for elf identify.
func SynergyIdentifyInterval(p *Party) int { return ElfIdentifyInterval(p) }

// DwarfDamageBonus returns +2 if member is dwarf below 50% HP.
func DwarfDamageBonus(m *Member) int {
	if m == nil || !m.IsAlive() {
		return 0
	}
	if normalizeRaceID(m.Race) != "dwarf" {
		return 0
	}
	if m.MaxHP > 0 && m.HP*2 < m.MaxHP {
		return 2
	}
	return 0
}

// HalflingAvoidChance returns 0.05 if party has any living halfling, else 0.
func HalflingAvoidChance(p *Party) float64 {
	if p != nil && p.HasRace("halfling") {
		return 0.05
	}
	return 0
}

// ShouldHalflingAvoid rolls whether party avoids damage due to halfling.
func ShouldHalflingAvoid(rng *rand.Rand, p *Party) bool {
	if HalflingAvoidChance(p) <= 0 {
		return false
	}
	if rng == nil {
		return false
	}
	return rng.Float64() < 0.05
}

// GnomeScrollSaveChance returns 5% per gnome.
func GnomeScrollSaveChance(p *Party) float64 {
	if p == nil {
		return 0
	}
	c := RaceCount(p, "gnome")
	if c == 0 {
		return 0
	}
	return float64(c) * 0.05
}

// ShouldGnomeSaveScroll rolls for scroll consumption save.
func ShouldGnomeSaveScroll(rng *rand.Rand, p *Party) bool {
	ch := GnomeScrollSaveChance(p)
	if ch <= 0 {
		return false
	}
	if rng == nil {
		return false
	}
	return rng.Float64() < ch
}

// ShouldGnomeInstantIdentify rolls 10% for first-of-kind instant identify if party has gnome.
func ShouldGnomeInstantIdentify(rng *rand.Rand, p *Party) bool {
	if p == nil || !p.HasRace("gnome") {
		return false
	}
	if rng == nil {
		return false
	}
	return rng.Float64() < 0.10
}

// HalfOrcATKBonus returns bonus ATK for a given member if half-orc: +1 per other half-orc.
func HalfOrcATKBonus(p *Party, m *Member) int {
	if m == nil || normalizeRaceID(m.Race) != "half_orc" {
		return 0
	}
	c := RaceCount(p, "half_orc")
	if c >= 2 {
		return c - 1
	}
	return 0
}

// IsHalfOrcImmuneToATKReduction reports true if member is half-orc and should ignore ATK reductions.
func IsHalfOrcImmuneToATKReduction(m *Member) bool {
	if m == nil {
		return false
	}
	return normalizeRaceID(m.Race) == "half_orc"
}

// TryReduceATK attempts to reduce member ATK by delta (positive = reduction). Returns true if applied, false if immune.
func TryReduceATK(m *Member, delta int) bool {
	if m == nil || delta <= 0 {
		return false
	}
	if IsHalfOrcImmuneToATKReduction(m) {
		return false
	}
	m.ATK[0] -= delta
	m.ATK[1] -= delta
	if m.ATK[0] < 0 {
		m.ATK[0] = 0
	}
	if m.ATK[1] < 0 {
		m.ATK[1] = 0
	}
	return true
}

// Troll helpers
func TrollCount(p *Party) int { return RaceCount(p, "troll") }

// EffectiveLivingCount counts trolls twice when >50% health, otherwise once.
func EffectiveLivingCount(p *Party) int {
	if p == nil {
		return 0
	}
	n := 0
	for _, m := range p.Members {
		if !m.IsAlive() {
			continue
		}
		if normalizeRaceID(m.Race) == "troll" && m.MaxHP > 0 && m.HP*2 > m.MaxHP {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// EffectivePartySize alias
func EffectivePartySize(p *Party) int { return EffectiveLivingCount(p) }

// TrollFoodMultiplier returns effective living count for food tick (troll double when healthy)
func TrollFoodMultiplier(p *Party) int { return EffectiveLivingCount(p) }

// ShouldTrollRegen returns true every 3 ticks
func ShouldTrollRegen(turn int) bool { return turn > 0 && turn%3 == 0 }

// TrollRegenTick heals living trolls 1 HP if alive and not full.
func TrollRegenTick(p *Party) {
	if p == nil {
		return
	}
	for _, m := range p.Members {
		if !m.IsAlive() || normalizeRaceID(m.Race) != "troll" {
			continue
		}
		if m.HP < m.MaxHP {
			m.HP++
			if m.HP > m.MaxHP {
				m.HP = m.MaxHP
			}
		}
	}
}

// randomRace picks a random race id.
func randomRace(rng *rand.Rand) string {
	list := LoadRaces()
	if len(list) == 0 {
		list = fallbackRaces()
	}
	return list[rng.IntN(len(list))].ID
}

// applyCharBuff mutates member stats by race's CharBuff (once at creation).
func applyCharBuff(m *Member, raceID string) {
	if r, ok := GetRace(raceID); ok {
		b := r.CharBuff
		if b.HP != 0 {
			m.MaxHP += b.HP
			m.HP += b.HP
		}
		if b.ATK != 0 {
			m.ATK[0] += b.ATK
			m.ATK[1] += b.ATK
		}
		if b.DEF != 0 {
			m.DEF += b.DEF
		}
		if b.MDEF != 0 {
			m.MDEF += b.MDEF
		}
		if b.Light != 0 {
			m.Light += b.Light
		}
		if b.Carry != 0 {
			m.Carry += b.Carry
		}
	}
}

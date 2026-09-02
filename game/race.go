package game

import (
	"encoding/json"
	"math/rand/v2"
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
		{ID: "human", Name: "Human", Desc: "Versatile and adaptable", CharBuff: Buff{HP: 1, ATK: 1}, PartyBuff: Buff{HP: 1}, SynergyBuff: SynergyBuff{Desc: "+2 HP each when 2+ humans", Threshold: 2, HP: 2}},
		{ID: "elf", Name: "Elf", Desc: "Keen senses and arcane affinity", CharBuff: Buff{MDEF: 1, Light: 1}, PartyBuff: Buff{Light: 1}, SynergyBuff: SynergyBuff{Desc: "+10% XP when 2+ elves", Threshold: 2, XPBonus: 0.1}},
		{ID: "dwarf", Name: "Dwarf", Desc: "Stalwart and resilient", CharBuff: Buff{HP: 2, DEF: 1}, PartyBuff: Buff{DEF: 1}, SynergyBuff: SynergyBuff{Desc: "+2 HP each when 2+ dwarves", Threshold: 2, HP: 2}},
		{ID: "halfling", Name: "Halfling", Desc: "Nimble and light-footed", CharBuff: Buff{Light: 1, Carry: 2}, PartyBuff: Buff{ATK: 1}, SynergyBuff: SynergyBuff{Desc: "+1 carry each when 2+ halflings", Threshold: 2, Carry: 1}},
		{ID: "gnome", Name: "Gnome", Desc: "Inventive and curious", CharBuff: Buff{MDEF: 1, Carry: 1}, PartyBuff: Buff{Light: 1}, SynergyBuff: SynergyBuff{Desc: "+1 MDEF each when 2+ gnomes", Threshold: 2, MDEF: 1}},
	}
}

func LoadRaces() []Race {
	racesOnce.Do(func() {
		b, err := RawJSON("races.json")
		if err != nil {
			list := fallbackRaces()
			racesCache = make(map[string]Race, len(list))
			for _, r := range list {
				racesCache[r.ID] = r
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
	r, ok := racesCache[id]
	if ok {
		return r, true
	}
	// fallback search
	for _, fr := range fallbackRaces() {
		if fr.ID == id {
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
	for _, m := range p.Members {
		if m.IsAlive() && m.Race == raceID {
			return true
		}
	}
	return false
}

// raceCount counts living members per race.
func raceCount(p *Party) map[string]int {
	m := map[string]int{}
	for _, mem := range p.Members {
		if mem.IsAlive() && mem.Race != "" {
			m[mem.Race]++
		}
	}
	return m
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
		// desired synergy for this member
		var desiredSynergy SynergyBuff
		if r, ok := GetRace(m.Race); ok {
			if counts[m.Race] >= r.SynergyBuff.Threshold {
				desiredSynergy = r.SynergyBuff
			}
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

// SynergyXPBonus returns total XP multiplier bonus from synergy (e.g., 0.1 for 10%).
func SynergyXPBonus(p *Party) float64 {
	if p == nil {
		return 0
	}
	LoadRaces()
	counts := raceCount(p)
	bonus := 0.0
	for raceID, cnt := range counts {
		if r, ok := GetRace(raceID); ok {
			if cnt >= r.SynergyBuff.Threshold && r.SynergyBuff.XPBonus > 0 {
				bonus += r.SynergyBuff.XPBonus
			}
		}
	}
	return bonus
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

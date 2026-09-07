package game

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Data-driven consumables — potions/scrolls/statuses.json is authoritative.
// JSON stores visible turns (e.g., 40); code applies visible+1 to account for
// TickStatuses decrement on the same turn the status is created (H19).
// Helper StatusAppliedDuration handles the +1 consistently.
// ---------------------------------------------------------------------------

// StatusAppliedDuration converts visible turns (as stored in JSON) to the
// value passed to ApplyStatus. Example: strength 40 -> 41.
func StatusAppliedDuration(visible int) int {
	if visible <= 0 {
		return 0
	}
	return visible + 1
}

// StatusDef mirrors data/statuses.json entries.
type StatusDef struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Desc      string `json:"desc,omitempty"`
	Duration  int    `json:"duration,omitempty"` // visible turns
	DefDelta  int    `json:"defDelta,omitempty"`
	Atk       int    `json:"atk,omitempty"`
	Dot       int    `json:"dot,omitempty"`
	ResistPct int    `json:"resistPct,omitempty"`
	Radius    int    `json:"radius,omitempty"`
	Cap       int    `json:"cap,omitempty"`
}

type statusFile struct {
	Notes    string             `json:"notes,omitempty"`
	Statuses []StatusDef        `json:"statuses"`
	Resists  map[string]float64 `json:"resists,omitempty"`
	Dots     map[string]int     `json:"dots,omitempty"`
}

var (
	statusDefs  map[string]StatusDef
	statusResists map[string]float64
	statusDots  map[string]int
)

func loadStatusData() {
	if statusDefs != nil {
		return
	}
	statusDefs = map[string]StatusDef{}
	statusResists = map[string]float64{"iron_will": 0.10, "ward": 0.05}
	statusDots = map[string]int{"poison": 1, "spore": 1, "bleed": 2, "rend": 2}
	b, err := dataFS.ReadFile("data/statuses.json")
	if err != nil {
		// fallback: populate from hardcoded known statuses
		for _, s := range []StatusDef{
			{ID: StatusHex, Duration: 10, DefDelta: -1},
			{ID: StatusCurse, Duration: 200, DefDelta: -1},
			{ID: StatusBless, Duration: 100, DefDelta: 1},
			{ID: StatusStrength, Duration: 40, Atk: 2},
			{ID: StatusInvisibility, Duration: 20},
			{ID: StatusFireResist, Duration: 60, ResistPct: 30},
			{ID: StatusParalysis, Duration: 3},
			{ID: StatusLevitation, Duration: 25},
			{ID: StatusEnlightenment, Duration: 15},
			{ID: StatusConfusion, Duration: 8, Radius: 8},
			{ID: StatusSummon, Duration: 15, Cap: 4},
			{ID: StatusRend, Duration: 5, Dot: 2},
			{ID: StatusBleed, Duration: 5, Dot: 2},
			{ID: StatusSpore, Duration: 7, Dot: 1},
			{ID: StatusPoison, Duration: 5, Dot: 1},
			{ID: StatusEntangle, Duration: 3},
			{ID: StatusSleep, Duration: 4},
			{ID: StatusBlind, Duration: 20},
			{ID: StatusHaste, Duration: 50},
			{ID: StatusSlow, Duration: 30},
			{ID: StatusSilence, Duration: 5},
			{ID: StatusStun, Duration: 1},
		} {
			statusDefs[s.ID] = s
		}
		return
	}
	var sf statusFile
	if err := json.Unmarshal(b, &sf); err != nil {
		return
	}
	for _, s := range sf.Statuses {
		statusDefs[s.ID] = s
	}
	if sf.Resists != nil {
		statusResists = sf.Resists
	}
	if sf.Dots != nil {
		statusDots = sf.Dots
	}
}

func statusDefFor(id string) StatusDef {
	loadStatusData()
	if d, ok := statusDefs[id]; ok {
		return d
	}
	return StatusDef{ID: id}
}

func potionEffect(typeID string) ConsumableEffect {
	_, types := loadPotionData()
	for _, t := range types {
		if t.ID == typeID {
			if t.Effect.Kind != "" || t.Effect.Amount != 0 || t.Effect.Duration != 0 {
				return t.Effect
			}
		}
	}
	// fallback
	for _, t := range fallbackPotionTypes {
		if t.ID == typeID {
			return t.Effect
		}
	}
	return ConsumableEffect{}
}

func scrollEffect(typeID string) ConsumableEffect {
	_, types := loadScrollData()
	for _, t := range types {
		if t.ID == typeID {
			if t.Effect.Kind != "" || t.Effect.Damage != 0 || t.Effect.Heal != 0 || t.Effect.Duration != 0 || len(t.Effect.Cleanse) > 0 {
				return t.Effect
			}
		}
	}
	for _, t := range fallbackScrollTypes {
		if t.ID == typeID {
			return t.Effect
		}
	}
	return ConsumableEffect{}
}

// effectiveDEFDeltaData returns DEF delta from statuses.json (data-driven).
func effectiveDEFDeltaData(statuses map[string]int) int {
	loadStatusData()
	delta := 0
	for id, dur := range statuses {
		if dur <= 0 {
			continue
		}
		if d, ok := statusDefs[id]; ok && d.DefDelta != 0 {
			delta += d.DefDelta
		}
	}
	return delta
}

func statusDotDamage(id string) int {
	loadStatusData()
	if v, ok := statusDots[id]; ok {
		return v
	}
	if d, ok := statusDefs[id]; ok && d.Dot != 0 {
		if d.Dot < 0 {
			return 0
		}
		return d.Dot
	}
	// hardcoded fallback
	switch id {
	case StatusPoison, StatusSpore:
		return 1
	case StatusBleed, StatusRend:
		return 2
	}
	return 0
}

func resistsForTalent(talent string) float64 {
	loadStatusData()
	if v, ok := statusResists[talent]; ok {
		return v
	}
	switch talent {
	case "iron_will":
		return 0.10
	case "ward":
		return 0.05
	}
	return 0
}

// ---------------------------------------------------------------------------
// Single apply hubs — collapse 3× duplicate switches.
// ---------------------------------------------------------------------------

func (g *Game) applyPotionEffect(typeID string, isSelf bool, targetEnemy *EnemyParty, member *Member) {
	eff := potionEffect(typeID)
	// Defaults when JSON missing (robust fallback)
	if eff.Kind == "" {
		// infer from id for old data
		switch typeID {
		case "healing":
			eff = ConsumableEffect{Kind: "heal", Amount: 12}
		case "poison":
			eff = ConsumableEffect{Kind: "damage", Amount: 6}
		case "strength", "invisibility", "fire_resist", "paralysis", "levitation", "enlightenment", "regeneration":
			eff = ConsumableEffect{Kind: "status", Status: typeID, Duration: 40}
			switch typeID {
			case "strength":
				eff.Duration = 40
				eff.Atk = 2
			case "invisibility":
				eff.Duration = 20
			case "fire_resist":
				eff.Duration = 60
				eff.ResistPct = 30
			case "paralysis":
				eff.Duration = 3
			case "levitation":
				eff.Duration = 25
			case "enlightenment":
				eff.Duration = 15
			case "regeneration":
				eff.Duration = 20
				eff.Status = "regenerate"
			}
		}
	}

	switch typeID {
	case "healing":
		amt := eff.Amount
		if amt == 0 {
			amt = 12
		}
		amt += lootHealBonus(g.Party)
		if isSelf {
			if member != nil {
				if member.IsAlive() && member.HP < member.MaxHP {
					member.HP += amt
					if member.HP > member.MaxHP {
						member.HP = member.MaxHP
					}
					g.Logf("Healing potion restores %d HP to %s.", amt, member.Name)
				} else {
					g.Logf("Healing potion: %s is already at full health.", member.Name)
				}
			} else {
				healed := 0
				for _, m := range g.Party.Members {
					if m.IsAlive() && m.HP < m.MaxHP {
						m.HP += amt
						if m.HP > m.MaxHP {
							m.HP = m.MaxHP
						}
						healed++
					}
				}
				if healed > 0 {
					g.Logf("Healing potion restores %d HP to %d members.", amt, healed)
				} else {
					g.Logf("Healing potion: already at full health.")
				}
			}
		} else if targetEnemy != nil {
			healed := 0
			for _, m := range targetEnemy.Members {
				if m.IsAlive() && m.HP < m.MaxHP {
					m.HP += amt
					if m.HP > m.MaxHP {
						m.HP = m.MaxHP
					}
					healed++
				}
			}
			if healed > 0 {
				g.Logf("Healing potion restores %d HP to %d enemies (%s).", amt, healed, targetEnemy.DisplayName())
			} else {
				g.Logf("Healing potion splashes on %s with no effect.", targetEnemy.DisplayName())
			}
		} else {
			g.Logf("Healing potion shatters on ground.")
		}
	case "poison":
		amt := eff.Amount
		if amt == 0 {
			amt = 6
		}
		if isSelf {
			if hasThickSkinNegate(g.Party, g.RNG) {
				g.Logf("Thick skin shrugs off poison!")
			} else if hasCounterspellNegate(g.Party, g.RNG) {
				g.Logf("Counterspell negates poison!")
			} else if member != nil {
				member.HP -= amt
				if member.HP <= 0 {
					member.HP = 0
					member.Alive = false
				}
				g.Logf("Poison potion deals %d damage to %s!", amt, member.Name)
				if g.Party.LivingCount() == 0 {
					g.Over = true
					if g.Cause == "" {
						g.Cause = "Poison"
					}
					g.Logf("You have succumbed to poison. Seed %d.", g.Seed)
					g.RecordScore()
				}
			} else {
				_, dmg := g.Party.ApplyDamage(g.RNG, amt)
				g.Logf("Poison potion deals %d damage!", dmg)
				if g.Party.LivingCount() == 0 {
					g.Over = true
					if g.Cause == "" {
						g.Cause = "Poison"
					}
					g.Logf("You have succumbed to poison. Seed %d.", g.Seed)
					g.RecordScore()
				}
			}
		} else if targetEnemy != nil {
			// Thrown as an attack: single target, favoring the active enemy unit.
			idx := pickEnemyTarget(g.RNG, targetEnemy)
			if idx < 0 {
				g.Logf("Poison potion splashes harmlessly.")
			} else {
				m := targetEnemy.Members[idx]
				m.HP -= amt
				if m.HP <= 0 {
					m.HP = 0
					m.Alive = false
				}
				g.Logf("Poison potion deals %d damage to %s!", amt, m.Name)
				if !targetEnemy.IsAlive() {
					g.Logf("%s collapses from poison!", targetEnemy.DisplayName())
					g.AddKill()
					g.rollKillDrop(targetEnemy)
				}
			}
		} else {
			g.Logf("Poison potion shatters on ground.")
		}
	case "strength", "invisibility", "fire_resist", "paralysis", "levitation", "enlightenment", "regeneration":
		statusID := eff.Status
		if statusID == "" {
			statusID = typeID
		}
		durVisible := eff.Duration
		if durVisible == 0 {
			// fallback via statusDefs
			d := statusDefFor(statusID)
			durVisible = d.Duration
			if durVisible == 0 {
				switch typeID {
				case "strength":
					durVisible = 40
				case "invisibility":
					durVisible = 20
				case "fire_resist":
					durVisible = 60
				case "paralysis":
					durVisible = 3
				case "levitation":
					durVisible = 25
				case "enlightenment":
					durVisible = 15
				case "regeneration":
					durVisible = 20
				}
			}
		}
		applied := StatusAppliedDuration(durVisible)
		// paralysis is resistible when self-targeted
		if typeID == "paralysis" && isSelf {
			if hasThickSkinNegate(g.Party, g.RNG) {
				g.Logf("Thick skin shrugs off paralysis!")
				return
			}
			if hasCounterspellNegate(g.Party, g.RNG) {
				g.Logf("Counterspell negates paralysis!")
				return
			}
		}
		if isSelf {
			g.Party.ApplyStatus(statusID, applied)
		} else if targetEnemy != nil {
			targetEnemy.ApplyStatus(statusID, applied)
		} else {
			g.Logf("%s potion shatters on ground.", strings.Title(strings.ReplaceAll(statusID, "_", " ")))
			return
		}
		// logging with visible duration and bonus info
		switch typeID {
		case "strength":
			atk := eff.Atk
			if atk == 0 {
				atk = 2
			}
			if isSelf {
				g.Logf("Strength potion: +%d ATK for %d turns.", atk, durVisible)
			} else {
				g.Logf("Strength potion: %s gains +%d ATK for %d turns.", targetEnemy.DisplayName(), atk, durVisible)
			}
		case "invisibility":
			if isSelf {
				g.Logf("Invisibility potion: you fade from sight for %d turns.", durVisible)
			} else {
				g.Logf("Invisibility potion: %s fades from sight for %d turns.", targetEnemy.DisplayName(), durVisible)
			}
		case "fire_resist":
			pct := eff.ResistPct
			if pct == 0 {
				pct = 30
			}
			if isSelf {
				g.Logf("Fire resistance potion: +%d%% fire resist for %d turns.", pct, durVisible)
			} else {
				g.Logf("Fire resistance potion splashes on %s (+%d%% for %d turns).", targetEnemy.DisplayName(), pct, durVisible)
			}
		case "paralysis":
			if isSelf {
				g.Logf("Paralysis potion: you are paralyzed for %d turns!", durVisible)
			} else {
				g.Logf("Paralysis potion: %s is paralyzed for %d turns!", targetEnemy.DisplayName(), durVisible)
			}
		case "levitation":
			if isSelf {
				g.Logf("Levitation potion: you float above traps for %d turns.", durVisible)
			} else {
				g.Logf("Levitation potion: %s floats above traps for %d turns.", targetEnemy.DisplayName(), durVisible)
			}
		case "enlightenment":
			if isSelf {
				g.Logf("Enlightenment potion: your mind expands for %d turns.", durVisible)
				if lvl := g.CurLevel(); lvl != nil {
					for y := range lvl.H {
						for x := range lvl.W {
							lvl.Seen[y][x] = true
						}
					}
				}
			} else {
				g.Logf("Enlightenment potion splashes on %s.", targetEnemy.DisplayName())
				// apply status already done above; enemy case does not reveal map
			}
		case "regeneration":
			if isSelf {
				g.Logf("Regeneration potion: you regenerate for %d turns.", durVisible)
			} else {
				g.Logf("Regeneration potion splashes on %s.", targetEnemy.DisplayName())
			}
		}
	default:
		// generic fallback from desc
		_, types := loadPotionData()
		name := typeID
		for _, t := range types {
			if t.ID == typeID {
				name = t.Name
				break
			}
		}
		if targetEnemy != nil {
			g.Logf("Potion effect (%s) hits %s.", name, targetEnemy.DisplayName())
		} else if isSelf {
			g.Logf("Potion effect: %s.", name)
		} else {
			g.Logf("Potion shatters on ground.")
		}
		_ = eff
	}
}

func (g *Game) applyScrollEffect(typeID string, isSelf bool, targetEnemy *EnemyParty, target Pos, member *Member) {
	eff := scrollEffect(typeID)
	// defaults when missing
	if eff.Kind == "" {
		eff.Kind = typeID
		switch typeID {
		case "fireball":
			if eff.Damage == 0 {
				eff.Damage = 10
			}
			if eff.Radius == 0 {
				eff.Radius = 2
			}
			if eff.ResistPct == 0 {
				eff.ResistPct = 30
			}
		case "confusion":
			if eff.Duration == 0 {
				eff.Duration = 8
			}
			if eff.Radius == 0 {
				eff.Radius = 8
			}
		case "greater_healing":
			if eff.Heal == 0 {
				eff.Heal = 20
			}
			if len(eff.Cleanse) == 0 {
				eff.Cleanse = []string{"curse", "hex"}
			}
		case "summon":
			if eff.Duration == 0 {
				eff.Duration = 15
			}
			if eff.Cap == 0 {
				eff.Cap = 4
			}
		}
	}

	switch typeID {
	case "identify":
		var revealed []string
		for _, inv := range g.Party.Inventory {
			app := appearanceFromItem(inv)
			if app == "" || IsIdentified(app) {
				continue
			}
			Identify(app, TypeForAppearance(app))
			tid := TypeForAppearance(app)
			revealed = append(revealed, fmt.Sprintf("%s as %s", app, friendlyTypeName(tid, inv.Kind)))
		}
		if len(revealed) > 0 {
			g.Logf("Identify reveals: %s.", strings.Join(revealed, ", "))
		} else {
			g.Logf("Identify reveals: nothing left to identify.")
		}
	case "teleport":
		// candidate helper deduplicates self vs enemy target
		if isSelf || targetEnemy == nil {
			if lvl := g.CurLevel(); lvl != nil && g.RNG != nil {
				cands := teleportCandidates(g, nil, target)
				if len(cands) > 0 {
					g.Party.Pos = cands[g.RNG.IntN(len(cands))]
					g.Logf("Teleport scroll: you vanish to a new location.")
					g.UpdateFOV()
				}
			}
		} else {
			if lvl := g.CurLevel(); lvl != nil && g.RNG != nil {
				cands := teleportCandidates(g, targetEnemy, target)
				if len(cands) > 0 {
					targetEnemy.Pos = cands[g.RNG.IntN(len(cands))]
					g.Logf("Teleport scroll: %s vanishes to a new location!", targetEnemy.DisplayName())
				} else {
					g.Logf("Teleport scroll fizzles.")
				}
			}
		}
	case "fireball":
		g.Logf("Fireball scroll: flames burst!")
		if lvl := g.CurLevel(); lvl != nil {
			center := g.Party.Pos
			if targetEnemy != nil {
				center = targetEnemy.Pos
			} else if !isSelf {
				center = target
			}
			dmgBase := eff.Damage
			if dmgBase == 0 {
				dmgBase = 10
			}
			dmgBase = scaledScroll(dmgBase, g.Party)
			radius := eff.Radius
			if radius == 0 {
				radius = 2
			}
			for _, e := range lvl.Enemies {
				if !e.IsAlive() {
					continue
				}
				if max(abs(e.Pos.X-center.X), abs(e.Pos.Y-center.Y)) <= radius {
					dmg := dmgBase
					if e.HasStatus(StatusFireResist) {
						// data-driven resistPct (30 default)
						pct := eff.ResistPct
						if pct == 0 {
							pct = 30
						}
						dmg = (dmg * (100 - pct)) / 100
						if dmg < 1 {
							dmg = 1
						}
					}
					for _, m := range e.Members {
						if m.IsAlive() {
							m.HP -= dmg
							if m.HP <= 0 {
								m.HP = 0
								m.Alive = false
							}
						}
					}
					if !e.IsAlive() {
						g.Logf("Fireball slays %s!", e.DisplayName())
						g.AddKill()
						g.rollKillDrop(e)
					} else {
						g.Logf("Fireball hits %s for %d fire damage.", e.DisplayName(), dmg)
					}
				}
			}
		}
		// original code also reveals map after fireball (data bug preserved but now explicit)
		if lvl := g.CurLevel(); lvl != nil {
			for y := range lvl.H {
				for x := range lvl.W {
					lvl.Seen[y][x] = true
				}
			}
			g.Logf("Mapping scroll reveals the floor.")
		}
	case "enchant":
		if isSelf || targetEnemy == nil {
			if member != nil && member.IsAlive() {
				affix := GetRandomAffix(g.RNG)
				if affix != "" {
					member.Affixes = append(member.Affixes, affix)
					ApplyAffixMod(member, affix)
					g.Logf("Enchant scroll: %s gains %s.", member.Name, affix)
				} else {
					member.ATK[0]++
					member.ATK[1]++
					g.Logf("Enchant scroll: %s grows stronger (ATK %d-%d).", member.Name, member.ATK[0], member.ATK[1])
				}
				break
			}
			members := g.Party.LivingMembers()
			if len(members) > 0 {
				m := members[g.RNG.IntN(len(members))]
				affix := GetRandomAffix(g.RNG)
				if affix != "" {
					m.Affixes = append(m.Affixes, affix)
					ApplyAffixMod(m, affix)
					g.Logf("Enchant scroll: %s gains %s.", m.Name, affix)
				} else {
					m.ATK[0]++
					m.ATK[1]++
					g.Logf("Enchant scroll: %s grows stronger (ATK %d-%d).", m.Name, m.ATK[0], m.ATK[1])
				}
			} else {
				g.Logf("Enchant scroll fizzles.")
			}
		} else {
			var members []*Member
			for _, m := range targetEnemy.Members {
				if m.IsAlive() {
					members = append(members, m)
				}
			}
			if len(members) > 0 {
				m := members[g.RNG.IntN(len(members))]
				affix := GetRandomAffix(g.RNG)
				if affix != "" {
					m.Affixes = append(m.Affixes, affix)
					ApplyAffixMod(m, affix)
					g.Logf("Enchant scroll: %s gains %s!", targetEnemy.DisplayName(), affix)
				} else {
					m.ATK[0]++
					m.ATK[1]++
					g.Logf("Enchant scroll: %s grows stronger (ATK %d-%d)!", targetEnemy.DisplayName(), m.ATK[0], m.ATK[1])
				}
			} else {
				g.Logf("Enchant scroll fizzles on %s.", targetEnemy.DisplayName())
			}
		}
	case "confusion":
		durVisible := eff.Duration
		if durVisible == 0 {
			durVisible = 8
		}
		applied := StatusAppliedDuration(durVisible)
		radius := eff.Radius
		if radius == 0 {
			radius = 8
		}
		if isSelf || targetEnemy == nil {
			if !isSelf && targetEnemy == nil {
				// empty ground targeted: confuse enemies around target
				affected := 0
				if lvl := g.CurLevel(); lvl != nil {
					for _, e := range lvl.Enemies {
						if e.IsAlive() && max(abs(e.Pos.X-target.X), abs(e.Pos.Y-target.Y)) <= radius {
							e.ApplyStatus(StatusConfusion, applied)
							affected++
						}
					}
				}
				if affected > 0 {
					g.Logf("Confusion scroll: %d enemies are confused for %d turns.", affected, durVisible)
				} else {
					g.Logf("Confusion scroll: no enemies in range.")
				}
			} else {
				affected := 0
				if lvl := g.CurLevel(); lvl != nil {
					for _, e := range lvl.Enemies {
						if e.IsAlive() && max(abs(e.Pos.X-g.Party.Pos.X), abs(e.Pos.Y-g.Party.Pos.Y)) <= radius {
							e.ApplyStatus(StatusConfusion, applied)
							affected++
						}
					}
				}
				if affected > 0 {
					g.Logf("Confusion scroll: %d enemies are confused for %d turns.", affected, durVisible)
				} else {
					g.Logf("Confusion scroll: no enemies in range.")
				}
			}
		} else {
			targetEnemy.ApplyStatus(StatusConfusion, applied)
			g.Logf("Confusion scroll: %s is confused for %d turns!", targetEnemy.DisplayName(), durVisible)
		}
	case "greater_healing":
		healAmt := eff.Heal
		if healAmt == 0 {
			healAmt = 20
		}
		healAmt = scaledScroll(healAmt, g.Party) + lootHealBonus(g.Party)
		cleanse := eff.Cleanse
		if len(cleanse) == 0 {
			cleanse = []string{"curse", "hex"}
		}
		if isSelf || targetEnemy == nil {
			if member != nil && member.IsAlive() {
				member.HP += healAmt
				if member.HP > member.MaxHP {
					member.HP = member.MaxHP
				}
			} else {
				for _, m := range g.Party.Members {
					if m.IsAlive() {
						m.HP += healAmt
						if m.HP > m.MaxHP {
							m.HP = m.MaxHP
						}
					}
				}
			}
			for _, sid := range cleanse {
				if g.Party.HasStatus(sid) {
					if sid == "curse" {
						g.Logf("Greater healing removes curse.")
					}
					g.Party.RemoveStatus(sid)
				}
			}
			if member != nil && member.IsAlive() {
				g.Logf("Greater healing scroll restores %d HP to %s.", healAmt, member.Name)
			} else {
				g.Logf("Greater healing scroll restores %d HP to all members.", healAmt)
			}
		} else {
			for _, m := range targetEnemy.Members {
				if m.IsAlive() {
					m.HP += healAmt
					if m.HP > m.MaxHP {
						m.HP = m.MaxHP
					}
				}
			}
			for _, sid := range cleanse {
				targetEnemy.RemoveStatus(sid)
			}
			g.Logf("Greater healing scroll restores %d HP to %s!", healAmt, targetEnemy.DisplayName())
		}
	case "summon", "summon_aid", "summon aid":
		durVisible := eff.Duration
		if durVisible == 0 {
			durVisible = 15
		}
		applied := StatusAppliedDuration(durVisible)
		capVal := eff.Cap
		if capVal == 0 {
			capVal = 4
		}
		if isSelf || targetEnemy == nil {
			if len(g.Party.Members) >= capVal {
				g.Logf("Summon scroll: party is full, summon fizzles.")
			} else {
				classes := []string{"fighter", "cleric", "rogue", "wizard", "druid", "bard", "barbarian", "paladin"}
				cls := classes[g.RNG.IntN(len(classes))]
				tmp := GeneratePartyWithClasses(g.RNG, []string{cls}, g.Level)
				if tmp != nil && len(tmp.Members) > 0 {
					m := tmp.Members[0]
					m.Name = "Summoned " + m.Name
					g.Party.Members = append(g.Party.Members, m)
					g.Party.ApplyStatus(StatusSummon, applied)
					g.Logf("Summon scroll: %s the %s appears for %d turns.", m.Name, m.Class, durVisible)
				}
			}
		} else {
			if len(targetEnemy.Members) >= capVal {
				g.Logf("Summon scroll: %s party is full, summon fizzles.", targetEnemy.DisplayName())
			} else {
				classes := []string{"fighter", "cleric", "rogue", "wizard", "druid", "bard", "barbarian", "paladin"}
				cls := classes[g.RNG.IntN(len(classes))]
				tmp := GeneratePartyWithClasses(g.RNG, []string{cls}, g.Level)
				if tmp != nil && len(tmp.Members) > 0 {
					m := tmp.Members[0]
					m.Name = "Summoned " + m.Name
					targetEnemy.Members = append(targetEnemy.Members, m)
					targetEnemy.ApplyStatus(StatusSummon, applied)
					g.Logf("Summon scroll: %s summons %s!", targetEnemy.DisplayName(), m.Name)
				}
			}
		}
	case "mapping":
		if lvl := g.CurLevel(); lvl != nil {
			for y := range lvl.H {
				for x := range lvl.W {
					lvl.Seen[y][x] = true
				}
			}
			g.Logf("Mapping scroll reveals the floor.")
		}
	default:
		_, types := loadScrollData()
		name := typeID
		for _, t := range types {
			if t.ID == typeID {
				name = t.Name
				break
			}
		}
		g.Logf("Scroll effect: %s.", name)
	}
}

// teleportCandidates deduplicates teleport candidate generation for party vs enemy.
func teleportCandidates(g *Game, exclude *EnemyParty, _ Pos) []Pos {
	lvl := g.CurLevel()
	if lvl == nil || g.RNG == nil {
		return nil
	}
	reachable := lvl.TeleportReachable()
	var cands []Pos
	for y := range lvl.H {
		for x := range lvl.W {
			p := Pos{x, y}
			if !lvl.InBounds(p) || !lvl.Walkable(p) || !reachable[p] {
				continue
			}
			if lvl.IsDoor(p) && lvl.IsDoorClosed(p) {
				continue
			}
			occupied := false
			if exclude == nil {
				// party teleport: avoid any enemy
				for _, e := range lvl.Enemies {
					if e != nil && e.IsAlive() && e.Pos == p {
						occupied = true
						break
					}
				}
			} else {
				if p == g.Party.Pos {
					occupied = true
				}
				for _, e := range lvl.Enemies {
					if e != nil && e.IsAlive() && e.Pos == p && e != exclude {
						occupied = true
						break
					}
				}
			}
			if occupied {
				continue
			}
			cands = append(cands, p)
		}
	}
	if len(cands) == 0 {
		for y := range lvl.H {
			for x := range lvl.W {
				p := Pos{x, y}
				if lvl.InBounds(p) && lvl.Walkable(p) && reachable[p] {
					cands = append(cands, p)
				}
			}
		}
	}
	return cands
}

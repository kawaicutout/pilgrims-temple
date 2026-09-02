package game

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
)

// GroundItem is a pick-up on the floor. Added in M5 partial; wizard loot uses it for debug.
type GroundItem struct {
	Pos  Pos    `json:"pos"`
	Kind string `json:"kind"` // gold, ration, potion, scroll
	ID   string `json:"id"`   // for potion/scroll: type id (healing, identify, etc); for gold: ""
	Name string `json:"name"`
	Amount int `json:"amount"` // gold amount or stack
}

// Glyph returns map glyph for item.
func (it GroundItem) Glyph() rune {
	switch it.Kind {
	case "gold":
		return '$'
	case "ration":
		return '%'
	case "potion":
		return '!'
	case "scroll":
		return '?'
	default:
		return '*'
	}
}

// Color returns fg token.
func (it GroundItem) Color() string {
	switch it.Kind {
	case "gold":
		return "gold-bright"
	case "ration":
		return "gold"
	case "potion":
		return "slate"
	case "scroll":
		return "slate"
	default:
		return "fg"
	}
}

// ItemAt returns ground item at p if any.
func (l *Level) ItemAt(p Pos) *GroundItem {
	for i := range l.Items {
		if l.Items[i].Pos == p {
			return &l.Items[i]
		}
	}
	return nil
}

// TryPickup picks up all items at party position via 'g'.
// Returns true if something picked.
func (g *Game) TryPickup() bool {
	lvl := g.CurLevel()
	if lvl == nil || g.Party == nil {
		return false
	}
	p := g.Party.Pos
	picked := false
	remaining := lvl.Items[:0]
	for _, it := range lvl.Items {
		if it.Pos != p {
			remaining = append(remaining, it)
			continue
		}
		// Pick it
		switch it.Kind {
		case "gold":
			amt := it.Amount
			if amt <= 0 {
				amt = 10
			}
			g.Gold += amt
			g.Logf("Picked up %d gold.", amt)
		case "ration":
			amt := it.Amount
			if amt <= 0 {
				amt = 1
			}
			f := amt * 50
			g.Food += f
			g.FoodFloat += float64(f)
			g.Logf("Picked up ration (+%d food).", f)
		case "potion":
			if g.Party.CarryUsed() >= g.Party.CarryCapacity() {
				g.Logf("Carry full - cannot pick up potion: %s.", it.Name)
				remaining = append(remaining, it)
				continue
			}
			g.Party.Inventory = append(g.Party.Inventory, it)
			g.Logf("Picked up potion: %s.", it.Name)
			// Gnome 10% instant identify first of kind
			if g.Party.HasRace("gnome") && !IsIdentified(appearanceFromItem(it)) {
				// check if first of kind (no other held with same appearance already identified)
				isFirst := true
				app := appearanceFromItem(it)
				for _, inv := range g.Party.Inventory[:len(g.Party.Inventory)-1] {
					if appearanceFromItem(inv) == app {
						isFirst = false
						break
					}
				}
				if isFirst && g.RNG != nil && g.RNG.Float64() < 0.10 {
					IdentifyOnUse(app)
					g.Logf("Gnomish insight identifies %s as %s!", app, friendlyTypeName(TypeForAppearance(app), it.Kind))
				}
			}
			// Halfling 10% extra item on pickup
			if g.Party.HasRace("halfling") && g.RNG != nil && g.RNG.Float64() < 0.10 {
				if g.Party.CarryUsed() < g.Party.CarryCapacity() {
					dup := it
					g.Party.Inventory = append(g.Party.Inventory, dup)
					g.Logf("Halfling luck: extra %s!", it.Name)
				}
			}
		case "scroll":
			if g.Party.CarryUsed() >= g.Party.CarryCapacity() {
				g.Logf("Carry full - cannot pick up scroll: %s.", it.Name)
				remaining = append(remaining, it)
				continue
			}
			g.Party.Inventory = append(g.Party.Inventory, it)
			g.Logf("Picked up scroll: %s.", it.Name)
			// Gnome 10% instant identify first of kind for scrolls too
			if g.Party.HasRace("gnome") && !IsIdentified(appearanceFromItem(it)) {
				app := appearanceFromItem(it)
				isFirst := true
				for _, inv := range g.Party.Inventory[:len(g.Party.Inventory)-1] {
					if appearanceFromItem(inv) == app {
						isFirst = false
						break
					}
				}
				if isFirst && g.RNG != nil && g.RNG.Float64() < 0.10 {
					IdentifyOnUse(app)
					g.Logf("Gnomish insight identifies %s as %s!", app, friendlyTypeName(TypeForAppearance(app), it.Kind))
				}
			}
			if g.Party.HasRace("halfling") && g.RNG != nil && g.RNG.Float64() < 0.10 {
				if g.Party.CarryUsed() < g.Party.CarryCapacity() {
					dup := it
					g.Party.Inventory = append(g.Party.Inventory, dup)
					g.Logf("Halfling luck: extra %s!", it.Name)
				}
			}
		default:
			g.Logf("Picked up %s.", it.Name)
		}
		picked = true
	}
	lvl.Items = remaining
	if !picked {
		g.Logf("Nothing to pick up.")
	}
	return picked
}

// appearanceFromItem returns the appearance token for a potion/scroll item
// by stripping the kind suffix from its Name.
func appearanceFromItem(it GroundItem) string {
	if it.Kind == "potion" && strings.HasSuffix(it.Name, " potion") {
		return strings.TrimSuffix(it.Name, " potion")
	}
	if it.Kind == "scroll" && strings.HasSuffix(it.Name, " scroll") {
		return strings.TrimSuffix(it.Name, " scroll")
	}
	// Fallback: derive from ID mapping if Name not in expected form
	if it.Kind == "potion" || it.Kind == "scroll" {
		if app := AppearanceForType(it.ID); app != "" {
			return app
		}
	}
	return it.Name
}

// friendlyTypeName returns display name for a typeID.
func friendlyTypeName(typeID, kind string) string {
	if kind == "potion" {
		_, types := loadPotionData()
		for _, t := range types {
			if t.ID == typeID {
				return t.Name
			}
		}
		for _, t := range fallbackPotionTypes {
			if t.ID == typeID {
				return t.Name
			}
		}
	}
	if kind == "scroll" {
		_, types := loadScrollData()
		for _, t := range types {
			if t.ID == typeID {
				return t.Name
			}
		}
		for _, t := range fallbackScrollTypes {
			if t.ID == typeID {
				return t.Name
			}
		}
	}
	if typeID != "" {
		return FriendlyID(typeID)
	}
	return "unknown"
}

// UseEntry represents a grouped inventory entry for the usage menu.
type UseEntry struct {
	Appearance  string
	Kind        string // potion or scroll
	Count       int
	DisplayName string
}

// InventoryUseEntries returns grouped inventory entries sorted for the usage menu.
// Groups by appearance (potions and scrolls), showing identified names when known.
func (g *Game) InventoryUseEntries() []UseEntry {
	if g.Party == nil || len(g.Party.Inventory) == 0 {
		return nil
	}
	m := map[string]*UseEntry{}
	for _, it := range g.Party.Inventory {
		app := appearanceFromItem(it)
		key := it.Kind + "|" + app
		if e, ok := m[key]; ok {
			e.Count++
		} else {
			display := app
			if IsIdentified(app) {
				if tid, ok := Knowledge[app]; ok && tid != "" {
					display = friendlyTypeName(tid, it.Kind)
				} else if tid := TypeForAppearance(app); tid != "" {
					display = friendlyTypeName(tid, it.Kind)
				}
			}
			m[key] = &UseEntry{Appearance: app, Kind: it.Kind, Count: 1, DisplayName: display}
		}
	}
	entries := make([]UseEntry, 0, len(m))
	for _, e := range m {
		entries = append(entries, *e)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind // potion before scroll lexicographically
		}
		return entries[i].Appearance < entries[j].Appearance
	})
	return entries
}
// InventoryPotionEntries returns potion-only grouped entries for the throw menu, sorted by appearance.
func (g *Game) InventoryPotionEntries() []UseEntry {
	entries := g.InventoryUseEntries()
	var out []UseEntry
	for _, e := range entries {
		if e.Kind == "potion" {
			out = append(out, e)
		}
	}
	return out
}


// TryUseAppearance consumes one item of the given appearance, identifies it, applies effect and advances turn.
// Fallback for non-cursor callers (wizard etc) defaults target to Party.Pos.
func (g *Game) TryUseAppearance(appearance string) bool {
	if g.Party == nil {
		return false
	}
	return g.TryUseAppearanceAt(appearance, g.Party.Pos)
}

// TryUseAppearanceAt consumes one item of the given appearance, identifies it, applies effect to target party and advances turn.
// If target == Party.Pos -> apply to Party; else if enemy at target -> apply to that EnemyParty (gamble).
// Unidentified scrolls are still usable on enemy tiles without revealing effect beforehand.
func (g *Game) TryUseAppearanceAt(appearance string, target Pos) bool {
	if g.Party == nil || len(g.Party.Inventory) == 0 {
		g.Logf("No potions or scrolls to use.")
		return false
	}
	idx := -1
	for i, it := range g.Party.Inventory {
		if appearanceFromItem(it) == appearance {
			idx = i
			break
		}
	}
	if idx == -1 {
		g.Logf("No %s to use.", appearance)
		return false
	}
	it := g.Party.Inventory[idx]
	saved := false
	if it.Kind == "scroll" && ShouldGnomeSaveScroll(g.RNG, g.Party) {
		saved = true
		g.Logf("Gnomish thrift: scroll preserved!")
	} else {
		g.Party.Inventory = append(g.Party.Inventory[:idx], g.Party.Inventory[idx+1:]...)
	}
	trueType := TypeForAppearance(appearance)
	if trueType == "" {
		trueType = it.ID
	}
	newlyIdentified := IdentifyOnUse(appearance)
	typeName := friendlyTypeName(trueType, it.Kind)
	if newlyIdentified {
		g.Logf("Used %s - identified as %s!", appearance, typeName)
	} else if IsIdentified(appearance) {
		g.Logf("Used %s (%s).", appearance, typeName)
	} else {
		g.Logf("Used %s.", it.Name)
	}
	_ = saved
	// Resolve target party: self if target == Party.Pos, else enemy at target.
	isSelf := target == g.Party.Pos
	var targetEnemy *EnemyParty
	if !isSelf {
		if lvl := g.CurLevel(); lvl != nil {
			for _, e := range lvl.Enemies {
				if e.IsAlive() && e.Pos == target {
					targetEnemy = e
					break
				}
			}
		}
	}
	switch it.Kind {
	case "potion":
		switch trueType {
		case "healing":
			if isSelf {
				healed := 0
				for _, m := range g.Party.Members {
					if m.IsAlive() && m.HP < m.MaxHP {
						m.HP += 12
						if m.HP > m.MaxHP {
							m.HP = m.MaxHP
						}
						healed++
					}
				}
				if healed > 0 {
					g.Logf("Healing potion restores 12 HP to %d members.", healed)
				} else {
					g.Logf("Healing potion: already at full health.")
				}
			} else if targetEnemy != nil {
				healed := 0
				for _, m := range targetEnemy.Members {
					if m.IsAlive() && m.HP < m.MaxHP {
						m.HP += 12
						if m.HP > m.MaxHP {
							m.HP = m.MaxHP
						}
						healed++
					}
				}
				if healed > 0 {
					g.Logf("Healing potion restores 12 HP to %d enemies (%s).", healed, targetEnemy.DisplayName())
				} else {
					g.Logf("Healing potion splashes on %s with no effect.", targetEnemy.DisplayName())
				}
			} else {
				g.Logf("Healing potion shatters on ground.")
			}
		case "poison":
			if isSelf {
				_, dmg := g.Party.ApplyDamage(g.RNG, 6)
				g.Logf("Poison potion deals %d damage!", dmg)
				if g.Party.LivingCount() == 0 {
					g.Over = true
					g.Logf("You have succumbed to poison. Seed %d.", g.Seed)
				}
			} else if targetEnemy != nil {
				dmgTotal := 0
				for _, m := range targetEnemy.Members {
					if m.IsAlive() {
						m.HP -= 6
						dmgTotal += 6
						if m.HP <= 0 {
							m.HP = 0
							m.Alive = false
						}
					}
				}
				g.Logf("Poison potion deals %d damage to %s!", dmgTotal, targetEnemy.DisplayName())
				if !targetEnemy.IsAlive() {
					g.Logf("%s collapses from poison!", targetEnemy.DisplayName())
					g.AddKill()
					g.Logf("Score %d (Kills %d).", g.CalculateScore(), g.Kills)
				}
			} else {
				g.Logf("Poison potion shatters on ground.")
			}
		case "strength":
			if isSelf {
				g.Party.ApplyStatus(StatusStrength, 41)
				g.Logf("Strength potion: +2 ATK for 40 turns.")
			} else if targetEnemy != nil {
				targetEnemy.ApplyStatus(StatusStrength, 41)
				g.Logf("Strength potion: %s gains +2 ATK for 40 turns.", targetEnemy.DisplayName())
			} else {
				g.Logf("Strength potion shatters on ground.")
			}
		case "invisibility":
			if isSelf {
				g.Party.ApplyStatus(StatusInvisibility, 21)
				g.Logf("Invisibility potion: you fade from sight for 20 turns.")
			} else if targetEnemy != nil {
				targetEnemy.ApplyStatus(StatusInvisibility, 21)
				g.Logf("Invisibility potion: %s fades from sight for 20 turns.", targetEnemy.DisplayName())
			} else {
				g.Logf("Invisibility potion shatters on ground.")
			}
		case "fire_resist":
			if isSelf {
				g.Party.ApplyStatus(StatusFireResist, 61)
				g.Logf("Fire resistance potion: +30%% fire resist for 60 turns.")
			} else if targetEnemy != nil {
				targetEnemy.ApplyStatus(StatusFireResist, 61)
				g.Logf("Fire resistance potion splashes on %s (+30%% for 60 turns).", targetEnemy.DisplayName())
			} else {
				g.Logf("Fire resistance potion shatters on ground.")
			}
		case "paralysis":
			if isSelf {
				g.Party.ApplyStatus(StatusParalysis, 4)
				g.Logf("Paralysis potion: you are paralyzed for 3 turns!")
			} else if targetEnemy != nil {
				targetEnemy.ApplyStatus(StatusParalysis, 4)
				g.Logf("Paralysis potion: %s is paralyzed for 3 turns!", targetEnemy.DisplayName())
			} else {
				g.Logf("Paralysis potion shatters on ground.")
			}
		case "levitation":
			if isSelf {
				g.Party.ApplyStatus(StatusLevitation, 26)
				g.Logf("Levitation potion: you float above traps for 25 turns.")
			} else if targetEnemy != nil {
				targetEnemy.ApplyStatus(StatusLevitation, 26)
				g.Logf("Levitation potion: %s floats above traps for 25 turns.", targetEnemy.DisplayName())
			} else {
				g.Logf("Levitation potion shatters on ground.")
			}
		case "enlightenment":
			if isSelf {
				g.Party.ApplyStatus(StatusEnlightenment, 16)
				g.Logf("Enlightenment potion: your mind expands for 15 turns.")
				if lvl := g.CurLevel(); lvl != nil {
					for y := range lvl.H {
						for x := range lvl.W {
							lvl.Seen[y][x] = true
						}
					}
				}
			} else if targetEnemy != nil {
				targetEnemy.ApplyStatus(StatusEnlightenment, 16)
				g.Logf("Enlightenment potion splashes on %s.", targetEnemy.DisplayName())
			} else {
				g.Logf("Enlightenment potion shatters on ground.")
			}
		default:
			if targetEnemy != nil {
				g.Logf("Potion effect (%s) hits %s.", typeName, targetEnemy.DisplayName())
			} else if isSelf {
				g.Logf("Potion effect: %s.", typeName)
			} else {
				g.Logf("Potion shatters on ground.")
			}
		}
	case "scroll":
		switch trueType {
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
			if isSelf || targetEnemy == nil {
				if lvl := g.CurLevel(); lvl != nil && g.RNG != nil {
					reachable := lvl.TeleportReachable()
					var cands []Pos
					for y := range lvl.H {
						for x := range lvl.W {
							p := Pos{x, y}
							if !lvl.InBounds(p) {
								continue
							}
							if !lvl.Walkable(p) {
								continue
							}
							if !reachable[p] {
								continue
							}
							if lvl.IsDoor(p) && lvl.IsDoorClosed(p) {
								continue
							}
							occupied := false
							for _, e := range lvl.Enemies {
								if e != nil && e.IsAlive() && e.Pos == p {
									occupied = true
									break
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
					if len(cands) > 0 {
						g.Party.Pos = cands[g.RNG.IntN(len(cands))]
						g.Logf("Teleport scroll: you vanish to a new location.")
						g.UpdateFOV()
					}
				}
			} else {
				if lvl := g.CurLevel(); lvl != nil && g.RNG != nil {
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
							if p == g.Party.Pos {
								occupied = true
							}
							for _, e := range lvl.Enemies {
								if e != nil && e.IsAlive() && e.Pos == p && e != targetEnemy {
									occupied = true
									break
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
				for _, e := range lvl.Enemies {
					if !e.IsAlive() {
						continue
					}
					if max(abs(e.Pos.X-center.X), abs(e.Pos.Y-center.Y)) <= 2 {
						dmg := 10
						if e.HasStatus(StatusFireResist) {
							dmg = (dmg * 7) / 10
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
							g.Logf("Score %d (Kills %d).", g.CalculateScore(), g.Kills)
						} else {
							g.Logf("Fireball hits %s for %d fire damage.", e.DisplayName(), dmg)
						}
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
		case "enchant":
			if isSelf || targetEnemy == nil {
				members := g.Party.LivingMembers()
				if len(members) > 0 {
					m := members[g.RNG.IntN(len(members))]
					affix := GetRandomAffix(g.RNG)
					if affix != "" {
						m.Affixes = append(m.Affixes, affix)
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
				// enchant enemy as gamble
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
			if isSelf || targetEnemy == nil {
				// confusion when targeted on self (or empty) = area as before
				if !isSelf && targetEnemy == nil {
					// empty ground targeted: confuse enemies around target
					affected := 0
					if lvl := g.CurLevel(); lvl != nil {
						for _, e := range lvl.Enemies {
							if e.IsAlive() && max(abs(e.Pos.X-target.X), abs(e.Pos.Y-target.Y)) <= 8 {
								e.ApplyStatus(StatusConfusion, 9)
								affected++
							}
						}
					}
					if affected > 0 {
						g.Logf("Confusion scroll: %d enemies are confused for 8 turns.", affected)
					} else {
						g.Logf("Confusion scroll: no enemies in range.")
					}
				} else {
					affected := 0
					if lvl := g.CurLevel(); lvl != nil {
						for _, e := range lvl.Enemies {
							if e.IsAlive() && max(abs(e.Pos.X-g.Party.Pos.X), abs(e.Pos.Y-g.Party.Pos.Y)) <= 8 {
								e.ApplyStatus(StatusConfusion, 9)
								affected++
							}
						}
					}
					if affected > 0 {
						g.Logf("Confusion scroll: %d enemies are confused for 8 turns.", affected)
					} else {
						g.Logf("Confusion scroll: no enemies in range.")
					}
				}
			} else {
				targetEnemy.ApplyStatus(StatusConfusion, 9)
				g.Logf("Confusion scroll: %s is confused for 8 turns!", targetEnemy.DisplayName())
			}
		case "greater_healing":
			if isSelf || targetEnemy == nil {
				for _, m := range g.Party.Members {
					if m.IsAlive() {
						m.HP += 20
						if m.HP > m.MaxHP {
							m.HP = m.MaxHP
						}
					}
				}
				if g.Party.HasStatus(StatusCurse) {
					g.Party.RemoveStatus(StatusCurse)
					g.Logf("Greater healing removes curse.")
				}
				if g.Party.HasStatus(StatusHex) {
					g.Party.RemoveStatus(StatusHex)
				}
				g.Logf("Greater healing scroll restores 20 HP to all members.")
			} else {
				for _, m := range targetEnemy.Members {
					if m.IsAlive() {
						m.HP += 20
						if m.HP > m.MaxHP {
							m.HP = m.MaxHP
						}
					}
				}
				if targetEnemy.HasStatus(StatusCurse) {
					targetEnemy.RemoveStatus(StatusCurse)
				}
				if targetEnemy.HasStatus(StatusHex) {
					targetEnemy.RemoveStatus(StatusHex)
				}
				g.Logf("Greater healing scroll restores 20 HP to %s!", targetEnemy.DisplayName())
			}
		case "summon":
			if isSelf || targetEnemy == nil {
				if len(g.Party.Members) >= 4 {
					g.Logf("Summon scroll: party is full, summon fizzles.")
				} else {
					classes := []string{"fighter", "cleric", "rogue", "wizard", "druid", "bard", "barbarian", "paladin"}
					cls := classes[g.RNG.IntN(len(classes))]
					tmp := GeneratePartyWithClasses(g.RNG, []string{cls}, g.Level)
					if tmp != nil && len(tmp.Members) > 0 {
						m := tmp.Members[0]
						m.Name = "Summoned " + m.Name
						g.Party.Members = append(g.Party.Members, m)
						g.Party.ApplyStatus(StatusSummon, 16)
						g.Logf("Summon scroll: %s the %s appears for 15 turns.", m.Name, m.Class)
					}
				}
			} else {
				if len(targetEnemy.Members) >= 4 {
					g.Logf("Summon scroll: %s party is full, summon fizzles.", targetEnemy.DisplayName())
				} else {
					classes := []string{"fighter", "cleric", "rogue", "wizard", "druid", "bard", "barbarian", "paladin"}
					cls := classes[g.RNG.IntN(len(classes))]
					tmp := GeneratePartyWithClasses(g.RNG, []string{cls}, g.Level)
					if tmp != nil && len(tmp.Members) > 0 {
						m := tmp.Members[0]
						m.Name = "Summoned " + m.Name
						targetEnemy.Members = append(targetEnemy.Members, m)
						targetEnemy.ApplyStatus(StatusSummon, 16)
						g.Logf("Summon scroll: %s summons %s!", targetEnemy.DisplayName(), m.Name)
					}
				}
			}
		default:
			g.Logf("Scroll effect: %s.", typeName)
		}
	default:
		g.Logf("Used %s: %s.", it.Name, typeName)
	}
	g.EndPlayerTurn("")
	return true
}

// TryUseItemAt consumes the grouped entry at index (sorted order) and advances turn.
func (g *Game) TryUseItemAt(index int) bool {
	entries := g.InventoryUseEntries()
	if index < 0 || index >= len(entries) {
		return false
	}
	return g.TryUseAppearance(entries[index].Appearance)
}
// TryThrowItemAt consumes the potion at potion-menu index and throws it at target.
// It identifies the appearance via TryThrowAppearance and advances turn.
func (g *Game) TryThrowItemAt(index int, target Pos) bool {
	entries := g.InventoryPotionEntries()
	if index < 0 || index >= len(entries) {
		return false
	}
	return g.TryThrowAppearance(entries[index].Appearance, target)
}


// TryUseItem consumes the first available potion/scroll in inventory,
// identifies its appearance, applies its effect, and advances a turn.
// Returns true if an item was consumed.
func (g *Game) TryUseItem() bool {
	if g.Party == nil || len(g.Party.Inventory) == 0 {
		g.Logf("No potions or scrolls to use.")
		return false
	}
	// Prefer potions first, else first scroll/other.
	idx := -1
	for i, it := range g.Party.Inventory {
		if it.Kind == "potion" {
			idx = i
			break
		}
	}
	if idx == -1 {
		idx = 0
	}
	it := g.Party.Inventory[idx]
	// Remove from inventory
	g.Party.Inventory = append(g.Party.Inventory[:idx], g.Party.Inventory[idx+1:]...)
	appearance := appearanceFromItem(it)
	trueType := TypeForAppearance(appearance)
	if trueType == "" {
		trueType = it.ID
	}
	newlyIdentified := IdentifyOnUse(appearance)
	typeName := friendlyTypeName(trueType, it.Kind)
	if newlyIdentified {
		g.Logf("Used %s - identified as %s!", appearance, typeName)
	} else if IsIdentified(appearance) {
		g.Logf("Used %s (%s).", appearance, typeName)
	} else {
		g.Logf("Used %s.", it.Name)
	}
	// Apply effect based on true type
	switch it.Kind {
	case "potion":
		switch trueType {
		case "healing":
			healed := 0
			for _, m := range g.Party.Members {
				if m.IsAlive() && m.HP < m.MaxHP {
					m.HP += 12
					if m.HP > m.MaxHP {
						m.HP = m.MaxHP
					}
					healed++
				}
			}
			if healed > 0 {
				g.Logf("Healing potion restores 12 HP to %d members.", healed)
			} else {
				g.Logf("Healing potion: already at full health.")
			}
		case "poison":
			_, dmg := g.Party.ApplyDamage(g.RNG, 6)
			g.Logf("Poison potion deals %d damage!", dmg)
			if g.Party.LivingCount() == 0 {
				g.Over = true
				g.Logf("You have succumbed to poison. Seed %d.", g.Seed)
			}
		case "strength":
			g.Party.ApplyStatus(StatusStrength, 41)
			g.Logf("Strength potion: +2 ATK for 40 turns.")
		case "invisibility":
			g.Party.ApplyStatus(StatusInvisibility, 21)
			g.Logf("Invisibility potion: you fade from sight for 20 turns.")
		case "fire_resist":
			g.Party.ApplyStatus(StatusFireResist, 61)
			g.Logf("Fire resistance potion: +30%% fire resist for 60 turns.")
		case "paralysis":
			g.Party.ApplyStatus(StatusParalysis, 4)
			g.Logf("Paralysis potion: you are paralyzed for 3 turns!")
		case "levitation":
			g.Party.ApplyStatus(StatusLevitation, 26)
			g.Logf("Levitation potion: you float above traps for 25 turns.")
		case "enlightenment":
			g.Party.ApplyStatus(StatusEnlightenment, 16)
			g.Logf("Enlightenment potion: your mind expands for 15 turns.")
			if lvl := g.CurLevel(); lvl != nil {
				for y := range lvl.H {
					for x := range lvl.W {
						lvl.Seen[y][x] = true
					}
				}
			}
		default:
			g.Logf("Potion effect: %s.", typeName)
		}
	case "scroll":
		switch trueType {
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
			if lvl := g.CurLevel(); lvl != nil && g.RNG != nil {
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
						for _, e := range lvl.Enemies {
							if e != nil && e.IsAlive() && e.Pos == p {
								occupied = true
								break
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
				if len(cands) > 0 {
					g.Party.Pos = cands[g.RNG.IntN(len(cands))]
					g.Logf("Teleport scroll: you vanish to a new location.")
					g.UpdateFOV()
				}
			}
		case "fireball":
			g.Logf("Fireball scroll: flames burst around you!")
			if lvl := g.CurLevel(); lvl != nil {
				for _, e := range lvl.Enemies {
					if !e.IsAlive() {
						continue
					}
					if max(abs(e.Pos.X-g.Party.Pos.X), abs(e.Pos.Y-g.Party.Pos.Y)) <= 2 {
						dmg := 10
						if e.HasStatus(StatusFireResist) {
							dmg = (dmg * 7) / 10
							if dmg < 1 {
								dmg = 1
							}
						}
						wasAlive := e.IsAlive()
						for _, m := range e.Members {
							if m.IsAlive() {
								m.HP -= dmg
								if m.HP <= 0 {
									m.HP = 0
									m.Alive = false
								}
							}
						}
						if wasAlive && !e.IsAlive() {
							g.Logf("Fireball slays %s!", e.DisplayName())
							g.AddKill()
							g.Logf("Score %d (Kills %d).", g.CalculateScore(), g.Kills)
						} else if wasAlive {
							g.Logf("Fireball hits %s for %d fire damage.", e.DisplayName(), dmg)
						}
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
		case "enchant":
			members := g.Party.LivingMembers()
			if len(members) > 0 {
				m := members[g.RNG.IntN(len(members))]
				affix := GetRandomAffix(g.RNG)
				if affix != "" {
					m.Affixes = append(m.Affixes, affix)
					g.Logf("Enchant scroll: %s gains %s.", m.Name, affix)
				} else {
					m.ATK[0]++
					m.ATK[1]++
					g.Logf("Enchant scroll: %s grows stronger (ATK %d-%d).", m.Name, m.ATK[0], m.ATK[1])
				}
			} else {
				g.Logf("Enchant scroll fizzles.")
			}
		case "confusion":
			affected := 0
			if lvl := g.CurLevel(); lvl != nil {
				for _, e := range lvl.Enemies {
					if e.IsAlive() && max(abs(e.Pos.X-g.Party.Pos.X), abs(e.Pos.Y-g.Party.Pos.Y)) <= 8 {
						e.ApplyStatus(StatusConfusion, 9)
						affected++
					}
				}
			}
			if affected > 0 {
				g.Logf("Confusion scroll: %d enemies are confused for 8 turns.", affected)
			} else {
				g.Logf("Confusion scroll: no enemies in range.")
			}
		case "greater_healing":
			for _, m := range g.Party.Members {
				if m.IsAlive() {
					m.HP += 20
					if m.HP > m.MaxHP {
						m.HP = m.MaxHP
					}
				}
			}
			if g.Party.HasStatus(StatusCurse) {
				g.Party.RemoveStatus(StatusCurse)
				g.Logf("Greater healing removes curse.")
			}
			if g.Party.HasStatus(StatusHex) {
				g.Party.RemoveStatus(StatusHex)
			}
			g.Logf("Greater healing scroll restores 20 HP to all members.")
		case "summon":
			if len(g.Party.Members) >= 4 {
				g.Logf("Summon scroll: party is full, summon fizzles.")
			} else {
				classes := []string{"fighter", "cleric", "rogue", "wizard", "druid", "bard", "barbarian", "paladin"}
				cls := classes[g.RNG.IntN(len(classes))]
				tmp := GeneratePartyWithClasses(g.RNG, []string{cls}, g.Level)
				if tmp != nil && len(tmp.Members) > 0 {
					m := tmp.Members[0]
					m.Name = "Summoned " + m.Name
					g.Party.Members = append(g.Party.Members, m)
					g.Party.ApplyStatus(StatusSummon, 16)
					g.Logf("Summon scroll: %s the %s appears for 15 turns.", m.Name, m.Class)
				}
			}
		default:
			g.Logf("Scroll effect: %s.", typeName)
		}
	default:
		g.Logf("Used %s: %s.", it.Name, typeName)
	}
	g.EndPlayerTurn("")
	return true
}

// TryThrowPotion consumes the first potion in inventory, identifies it, logs throw, applies effect to target enemy if present, and advances turn.
func (g *Game) TryThrowPotion(dir Dir) bool {
	if g.Party == nil {
		g.Logf("No potions to throw.")
		return false
	}
	idx := -1
	for i, it := range g.Party.Inventory {
		if it.Kind == "potion" {
			idx = i
			break
		}
	}
	if idx == -1 {
		g.Logf("No potions to throw.")
		return false
	}
	it := g.Party.Inventory[idx]
	g.Party.Inventory = append(g.Party.Inventory[:idx], g.Party.Inventory[idx+1:]...)
	appearance := appearanceFromItem(it)
	trueType := TypeForAppearance(appearance)
	if trueType == "" {
		trueType = it.ID
	}
	newlyIdentified := IdentifyOnUse(appearance)
	typeName := friendlyTypeName(trueType, it.Kind)
	target := g.Party.Pos.Add(dir)
	var targetEnemy *EnemyParty
	if lvl := g.CurLevel(); lvl != nil {
		for _, e := range lvl.Enemies {
			if e.IsAlive() && e.Pos == target {
				targetEnemy = e
				break
			}
		}
	}
	dirStr := dirName(dir)
	if newlyIdentified {
		g.Logf("Threw %s potion - identified as %s at %s!", appearance, typeName, dirStr)
	} else if IsIdentified(appearance) {
		g.Logf("Threw %s potion (%s) at %s.", appearance, typeName, dirStr)
	} else {
		g.Logf("Threw %s at %s.", it.Name, dirStr)
	}
	switch trueType {
	case "healing":
		if targetEnemy != nil {
			healed := 0
			for _, m := range targetEnemy.Members {
				if m.IsAlive() && m.HP < m.MaxHP {
					m.HP += 12
					if m.HP > m.MaxHP {
						m.HP = m.MaxHP
					}
					healed++
				}
			}
			if healed > 0 {
				g.Logf("Healing potion restores 12 HP to %d enemies.", healed)
			} else {
				g.Logf("Healing potion splashes on %s with no effect.", targetEnemy.DisplayName())
			}
		} else {
			g.Logf("Potion shatters on ground.")
		}
	case "poison":
		if targetEnemy != nil {
			dmgTotal := 0
			for _, m := range targetEnemy.Members {
				if m.IsAlive() {
					m.HP -= 6
					dmgTotal += 6
					if m.HP <= 0 {
						m.HP = 0
						m.Alive = false
					}
				}
			}
			g.Logf("Poison potion deals %d damage to %s!", dmgTotal, targetEnemy.DisplayName())
			if !targetEnemy.IsAlive() {
				g.Logf("%s collapses from poison!", targetEnemy.DisplayName())
				g.AddKill()
				g.Logf("Score %d (Kills %d).", g.CalculateScore(), g.Kills)
			}
		} else {
			g.Logf("Poison potion shatters on ground.")
		}
	case "strength":
		if targetEnemy != nil {
			targetEnemy.ApplyStatus(StatusStrength, 41)
			g.Logf("Strength potion: %s gains +2 ATK for 40 turns.", targetEnemy.DisplayName())
		} else {
			g.Logf("Potion shatters on ground.")
		}
	case "invisibility":
		if targetEnemy != nil {
			targetEnemy.ApplyStatus(StatusInvisibility, 21)
			g.Logf("Invisibility potion: %s fades from sight for 20 turns.", targetEnemy.DisplayName())
		} else {
			g.Logf("Potion shatters on ground.")
		}
	case "fire_resist":
		if targetEnemy != nil {
			targetEnemy.ApplyStatus(StatusFireResist, 61)
			g.Logf("Fire resistance potion splashes on %s (+30%% for 60 turns).", targetEnemy.DisplayName())
		} else {
			g.Logf("Potion shatters on ground.")
		}
	case "paralysis":
		if targetEnemy != nil {
			targetEnemy.ApplyStatus(StatusParalysis, 4)
			g.Logf("Paralysis potion: %s is paralyzed for 3 turns!", targetEnemy.DisplayName())
		} else {
			g.Logf("Potion shatters on ground.")
		}
	case "levitation":
		if targetEnemy != nil {
			targetEnemy.ApplyStatus(StatusLevitation, 26)
			g.Logf("Levitation potion: %s floats above traps for 25 turns.", targetEnemy.DisplayName())
		} else {
			g.Logf("Potion shatters on ground.")
		}
	case "enlightenment":
		if targetEnemy != nil {
			targetEnemy.ApplyStatus(StatusEnlightenment, 16)
			g.Logf("Enlightenment potion splashes on %s.", targetEnemy.DisplayName())
		} else {
			g.Logf("Potion shatters on ground.")
		}
	default:
		if targetEnemy != nil {
			g.Logf("Potion effect (%s) hits %s.", typeName, targetEnemy.DisplayName())
		} else {
			g.Logf("Potion shatters on ground.")
		}
	}
	g.EndPlayerTurn("")
	return true
}

// pickLootKind selects a loot kind weighted by WorldConfig + floor theme + depth.
func pickLootKind(rng *rand.Rand, floor int, biome *Biome) string {
	wc := LoadWorldConfig()
	kinds := []string{"potion", "scroll", "ration", "gold"}
	weights := make([]float64, len(kinds))
	for i, k := range kinds {
		w := wc.ItemWeight(k, floor)
		if biome != nil {
			// Apply floor theme item weights if available (via GetFloorTheme)
			ft := GetFloorTheme(floor)
			if v, ok := ft.ItemWeights[k]; ok && v > 0 {
				w *= v
			} else if ft.ItemWeight(k) > 0 {
				w *= ft.ItemWeight(k)
			}
		}
		if w < 0 {
			w = 0
		}
		weights[i] = w
	}
	total := 0.0
	for _, v := range weights {
		total += v
	}
	if total <= 0 {
		return kinds[rng.IntN(len(kinds))]
	}
	r := rng.Float64() * total
	sum := 0.0
	for i, w := range weights {
		sum += w
		if r < sum {
			return kinds[i]
		}
	}
	return kinds[len(kinds)-1]
}

// makeRandomItem creates one random GroundItem for floor/biome.
func makeRandomItem(rng *rand.Rand, floor int, biome *Biome) GroundItem {
	kind := pickLootKind(rng, floor, biome)
	switch kind {
	case "gold":
		amt := 5 + rng.IntN(20) + floor*2
		return GroundItem{Kind: "gold", Name: "Gold", Amount: amt}
	case "ration":
		return GroundItem{Kind: "ration", Name: "Ration", Amount: 1}
	case "potion":
		_, types := loadPotionData()
		if len(types) > 0 {
			t := types[rng.IntN(len(types))]
			app := AppearanceForType(t.ID)
			if app == "" {
				app = t.ID
			}
			return GroundItem{Kind: "potion", ID: t.ID, Name: app + " potion", Amount: 1}
		}
		return GroundItem{Kind: "potion", ID: "healing", Name: "Healing potion", Amount: 1}
	case "scroll":
		_, types := loadScrollData()
		if len(types) > 0 {
			t := types[rng.IntN(len(types))]
			app := AppearanceForType(t.ID)
			if app == "" {
				app = t.ID
			}
			return GroundItem{Kind: "scroll", ID: t.ID, Name: app + " scroll", Amount: 1}
		}
		return GroundItem{Kind: "scroll", ID: "identify", Name: "Identify scroll", Amount: 1}
	default:
		return GroundItem{Kind: "gold", Name: "Gold", Amount: 10}
	}
}

// SpawnFloorLoot places initial floor loot during generation (debug: also used by wizard).
func SpawnFloorLoot(lvl *Level, rng *rand.Rand, floor int, biome *Biome) []GroundItem {
	if lvl == nil || rng == nil {
		return nil
	}
	// Count: 2-4 plus depth bonus
	count := 2 + rng.IntN(3) + floor/3
	var candidates []Pos
	for y := range lvl.H {
		for x := range lvl.W {
			p := Pos{x, y}
			if p == lvl.StairsUp || p == lvl.StairsDown {
				continue
			}
			if lvl.At(p) != TileFloor {
				continue
			}
			if !lvl.Walkable(p) {
				continue
			}
			// Avoid feature/enemy tiles already?
			occupied := false
			for _, f := range lvl.Features {
				if f.Pos == p {
					occupied = true
					break
				}
			}
			if occupied {
				continue
			}
			for _, e := range lvl.Enemies {
				if e.Pos == p {
					occupied = true
					break
				}
			}
			if occupied {
				continue
			}
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	// Shuffle
	for i := len(candidates) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}
	if count > len(candidates) {
		count = len(candidates)
	}
	var out []GroundItem
	for i := 0; i < count; i++ {
		it := makeRandomItem(rng, floor, biome)
		it.Pos = candidates[i]
		out = append(out, it)
		lvl.Items = append(lvl.Items, it)
	}
	if len(out) > 0 {
		_ = fmt.Sprintf("loot %d", len(out))
	}
	return out
}

// WizardSpawnLootItems spawns 2-4 random ground items near player using same generator as floor loot.
// This is the debug path - exercises the same loot table as level generation.
func (g *Game) WizardSpawnLootItems() {
	g.SetWizard()
	lvl := g.CurLevel()
	if lvl == nil {
		g.AddGold(20)
		return
	}
	// Find nearby free tiles around party (including current)
	var candidates []Pos
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			p := Pos{g.Party.Pos.X + dx, g.Party.Pos.Y + dy}
			if !lvl.InBounds(p) {
				continue
			}
			if lvl.At(p) != TileFloor && p != g.Party.Pos {
				continue
			}
			if !lvl.Walkable(p) && p != g.Party.Pos {
				// If blocked by litter, still allow if not impassable wall
				if lit := lvl.LitterAt(p); lit == nil || lit.BlocksMovement {
					continue
				}
			}
			occupied := false
			for _, e := range lvl.Enemies {
				if e.Pos == p && e.IsAlive() {
					occupied = true
					break
				}
			}
			if occupied {
				continue
			}
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		candidates = []Pos{g.Party.Pos}
	}
	count := 2 + g.RNG.IntN(3)
	if count > len(candidates) {
		count = len(candidates)
		// If not enough nearby, also spawn at party tile stacked (allow stacking)
		for len(candidates) < count {
			candidates = append(candidates, g.Party.Pos)
		}
	}
	// Shuffle candidates
	for i := len(candidates) - 1; i > 0; i-- {
		j := g.RNG.IntN(i + 1)
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}
	biome := GetBiomeForFloor(g.Floor)
	var spawned []GroundItem
	for i := 0; i < count; i++ {
		it := makeRandomItem(g.RNG, g.Floor, biome)
		it.Pos = candidates[i%len(candidates)]
		lvl.Items = append(lvl.Items, it)
		spawned = append(spawned, it)
	}
	if len(spawned) == 0 {
		g.Logf("Wizard: Spawn Random Loot -- nothing spawns.")
		return
	}
	// Auto-pick up if spawned on party tile for immediate feedback, else log locations
	for _, it := range spawned {
		if it.Pos == g.Party.Pos {
			// Defer pickup to player 'g' - but log that it's underfoot
		}
	}
	g.Logf("Wizard: Spawn Random Loot -- %d items nearby (press g to pick up).", len(spawned))
	for _, it := range spawned {
		g.Logf("  %s at %d,%d", it.Name, it.Pos.X, it.Pos.Y)
	}
}
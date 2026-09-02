package game

import (
	"fmt"
	"math/rand/v2"
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
		case "scroll":
			if g.Party.CarryUsed() >= g.Party.CarryCapacity() {
				g.Logf("Carry full - cannot pick up scroll: %s.", it.Name)
				remaining = append(remaining, it)
				continue
			}
			g.Party.Inventory = append(g.Party.Inventory, it)
			g.Logf("Picked up scroll: %s.", it.Name)
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
		return strings.Title(typeID)
	}
	return "unknown"
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
			members := g.Party.LivingMembers()
			if len(members) > 0 {
				m := members[g.RNG.IntN(len(members))]
				m.ATK[0] += 2
				m.ATK[1] += 2
				g.Logf("Strength potion: %s gains +2 ATK.", m.Name)
			}
		case "invisibility":
			g.Logf("Invisibility potion: you fade from sight briefly.")
		case "fire_resist":
			g.Logf("Fire resistance potion: you feel resistant to flame.")
		case "paralysis":
			g.Logf("Paralysis potion: you are paralyzed briefly!")
		case "levitation":
			g.Logf("Levitation potion: you float above traps.")
		case "enlightenment":
			g.Logf("Enlightenment potion: your mind expands.")
		default:
			g.Logf("Potion effect: %s.", typeName)
		}
	case "scroll":
		switch trueType {
		case "identify":
			// Identify another random unidentified held appearance if any
			found := ""
			for _, inv := range g.Party.Inventory {
				app := appearanceFromItem(inv)
				if !IsIdentified(app) {
					found = app
					break
				}
			}
			if found != "" {
				IdentifyOnUse(found)
				g.Logf("Identify scroll reveals %s as %s.", found, friendlyTypeName(TypeForAppearance(found), "potion"))
			} else {
				g.Logf("Identify scroll: nothing left to identify.")
			}
		case "teleport":
			if lvl := g.CurLevel(); lvl != nil && g.RNG != nil {
				// Find random walkable tile
				var cands []Pos
				for y := range lvl.H {
					for x := range lvl.W {
						p := Pos{x, y}
						if lvl.Walkable(p) {
							cands = append(cands, p)
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
						for _, m := range e.Members {
							if m.IsAlive() {
								m.HP -= 10
								if m.HP <= 0 {
									m.HP = 0
									m.Alive = false
								}
							}
						}
						if !e.IsAlive() {
							g.Logf("Fireball slays %s!", e.DisplayName())
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
		case "greater_healing":
			for _, m := range g.Party.Members {
				if m.IsAlive() {
					m.HP += 20
					if m.HP > m.MaxHP {
						m.HP = m.MaxHP
					}
				}
			}
			g.Logf("Greater healing scroll restores 20 HP to all members.")
		default:
			g.Logf("Scroll effect: %s.", typeName)
		}
	default:
		g.Logf("Used %s: %s.", it.Name, typeName)
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
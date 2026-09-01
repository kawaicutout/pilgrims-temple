package game

// WizardOption is one entry in the wizard menu.
type WizardOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// WizardState holds cursor for the wizard menu.
type WizardState struct {
	Selected int `json:"selected"`
}

// WizardOptions is the data-driven list (7 entries). If game/data/wizard.json
// exists it would override, but we ship a hardcoded fallback so the menu works
// without a file.
var WizardOptions = []WizardOption{
	{ID: "instant_level", Name: "Instant Level", Desc: "Gain one level immediately"},
	{ID: "spawn_loot", Name: "Spawn Random Loot", Desc: "Add 50 gold"},
	{ID: "food_1000", Name: "+1000 Food", Desc: "Add 1000 food"},
	{ID: "full_heal", Name: "Full Heal", Desc: "Restore all living members to max HP"},
	{ID: "add_member", Name: "Add Party Member", Desc: "Add a new pilgrim (max 4)"},
	{ID: "remove_member", Name: "Remove Party Member", Desc: "Remove a living member"},
	{ID: "resurrect", Name: "Resurrect Member", Desc: "Restore a fallen pilgrim"},
	{ID: "reveal_all", Name: "Reveal All Tiles", Desc: "Reveal entire dungeon"},
}

// IsWizard reports whether wizard mode has been used.
func (g *Game) IsWizard() bool { return g.Wizard }

// SetWizard sets the wizard flag and logs. Idempotent.
func (g *Game) SetWizard() {
	if g.Wizard {
		return
	}
	g.Wizard = true
	g.Logf("Wizard mode enabled - scores disabled")
}

// MoveWizardCursor moves selection in wizard menu.
func (ws *WizardState) Move(delta int) {
	n := len(WizardOptions)
	if n == 0 {
		return
	}
	ws.Selected = (ws.Selected + delta) % n
	if ws.Selected < 0 {
		ws.Selected += n
	}
}

// RevealAllTiles marks every tile on every floor as seen and visible,
// revealing all tile contents (enemies, features) for the next render.
func (g *Game) RevealAllTiles() {
	g.SetWizard()
	g.WizardReveal = true
	for _, lvl := range g.Levels {
		if lvl == nil {
			continue
		}
		for y := range lvl.H {
			for x := range lvl.W {
				lvl.Seen[y][x] = true
				lvl.Visible[y][x] = true
			}
		}
	}
	g.Logf("Wizard: Reveal All Tiles")
}

// WizardInstantLevel grants one level (HP/stats and talent picks).
func (g *Game) WizardInstantLevel() {
	g.SetWizard()
	if g.LevelUpPending != nil {
		g.Logf("Wizard: Instant Level - already pending")
		return
	}
	// Grant enough XP to trigger exactly one level up, without going negative
	g.XP = g.XPToNext
	g.LevelUp()
	if g.LevelUpPending == nil {
		g.Logf("Wizard: Instant Level -> level %d", g.Level)
	} else {
		g.Logf("Wizard: Instant Level -> level %d (pick talent)", g.Level)
	}
}

// WizardSpawnLoot gives gold.
func (g *Game) WizardSpawnLoot() {
	g.SetWizard()
	g.AddGold(50)
	g.Logf("Wizard: Spawn Random Loot (+50 gold)")
}

// WizardAddFood adds 1000 food.
func (g *Game) WizardAddFood() {
	g.SetWizard()
	g.Food += 1000
	g.FoodFloat += 1000
	g.Logf("Wizard: +1000 Food (now %d)", g.Food)
}

// WizardFullHeal restores all living members.
func (g *Game) WizardFullHeal() {
	g.SetWizard()
	for _, m := range g.Party.Members {
		if m.IsAlive() {
			m.HP = m.MaxHP
		}
	}
	g.Logf("Wizard: Full Heal")
}

// WizardAddMember adds a random class member if party <4.
// Returns true if added.
func (g *Game) WizardAddMember() bool {
	g.SetWizard()
	if len(g.Party.Members) >= 4 {
		g.Logf("Wizard: Add Party Member - party full (4)")
		return false
	}
	classes, err := LoadClasses()
	pick := "fighter"
	if err == nil && len(classes) > 0 {
		idx := g.RNG.IntN(len(classes))
		pick = classes[idx].ID
	}
	tmp := GeneratePartyWithClasses(g.RNG, []string{pick}, 1)
	if tmp == nil || len(tmp.Members) == 0 {
		g.Logf("Wizard: Add Party Member - failed")
		return false
	}
	m := tmp.Members[0]
	// Bring to current level approximation: give HP scaling roughly
	// GeneratePartyWithClasses already at level 1; bump to g.Level.
	for lvl := 1; lvl < g.Level; lvl++ {
		m.MaxHP += 1 + g.RNG.IntN(2)
		if g.RNG.IntN(2) == 0 {
			m.ATK[0]++
			m.ATK[1]++
		}
		if g.RNG.IntN(4) == 0 {
			m.DEF++
		}
	}
	m.HP = m.MaxHP
	m.Alive = true
	g.Party.Members = append(g.Party.Members, m)
	g.Party.EnsureSelection()
	g.Logf("Wizard: Add Party Member -> %s the %s joined", m.Name, m.Class)
	return true
}

// WizardRemoveMember removes living member at idx.
func (g *Game) WizardRemoveMember(idx int) bool {
	g.SetWizard()
	if idx < 0 || idx >= len(g.Party.Members) {
		g.Logf("Wizard: Remove Party Member - invalid index")
		return false
	}
	if len(g.Party.Members) <= 1 {
		g.Logf("Wizard: Remove Party Member - cannot remove last member")
		return false
	}
	m := g.Party.Members[idx]
	if !m.IsAlive() {
		g.Logf("Wizard: Remove Party Member - %s already fallen", m.Name)
		return false
	}
	name := m.Name
	g.Party.Members = append(g.Party.Members[:idx], g.Party.Members[idx+1:]...)
	g.Party.EnsureSelection()
	g.Logf("Wizard: Remove Party Member -> %s left", name)
	if g.Party.LivingCount() == 0 {
		g.Over = true
		g.Logf("Party has fallen.")
	}
	return true
}

// WizardResurrectMember restores dead member at idx.
func (g *Game) WizardResurrectMember(idx int) bool {
	g.SetWizard()
	if idx < 0 || idx >= len(g.Party.Members) {
		g.Logf("Wizard: Resurrect - invalid index")
		return false
	}
	m := g.Party.Members[idx]
	if m.IsAlive() {
		g.Logf("Wizard: Resurrect - %s is not fallen", m.Name)
		return false
	}
	// Restore as per shrine: re-scale to party level, no missed level-up awards
	// For wizard, just restore to max HP and alive, keeping original talents/affixes
	m.Alive = true
	m.HP = m.MaxHP
	// Ensure HP is at least 1 and scales with level if needed (simple: ensure max HP reflects level)
	// If member was dead for several levels, we could rescale, but for wizard we just restore as is
	g.Party.EnsureSelection()
	g.Logf("Wizard: Resurrect -> %s the %s rises", m.Name, m.Class)
	return true
}
// WizardExecute runs the effect for a wizard option id.
// For add/remove the frontend may want special UI; this handles the simple cases.
func (g *Game) WizardExecute(id string) {
	switch id {
	case "instant_level":
		g.WizardInstantLevel()
	case "spawn_loot":
		g.WizardSpawnLoot()
	case "food_1000":
		g.WizardAddFood()
	case "full_heal":
		g.WizardFullHeal()
	case "add_member":
		g.WizardAddMember()
	case "remove_member":
		// Default: remove selected member
		g.WizardRemoveMember(g.Party.Selected)
	case "resurrect":
		// Try selected if dead, else first dead
		if g.Party.Selected < len(g.Party.Members) && !g.Party.Members[g.Party.Selected].IsAlive() {
			g.WizardResurrectMember(g.Party.Selected)
		} else {
			for i, m := range g.Party.Members {
				if !m.IsAlive() {
					g.WizardResurrectMember(i)
					break
				}
			}
		}
	case "reveal_all":
		g.RevealAllTiles()
	default:
		g.SetWizard()
		g.Logf("Wizard: unknown option %s", id)
	}
}

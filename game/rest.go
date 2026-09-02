package game

// RestBatch performs up to batchTurns wait turns at 1.5x regen.
// Default batchTurns 10, healPerBatch 15 per DESIGN 11.3 (1 HP/turn *1.5).
// Ends early if a hostile becomes visible or HungerState threshold is crossed.
// Returns completed turns and logs "Rested X turns" or "Interrupted by {enemy}".
func RestBatch(g *Game) int {
	if g.Over {
		return 0
	}
	startState := g.HungerState()

	batchTurns := g.Tuning.Rest.BatchTurns
	if batchTurns <= 0 {
		batchTurns = 10
	}
	healPerBatch := g.Tuning.Rest.HealPerBatch
	if healPerBatch <= 0 {
		healPerBatch = 15
	}
	if g.Party.HasTalent("restful") {
		healPerBatch += 3
	}
	if g.Party.HasAffix("of_mending") {
		healPerBatch += 1
	}
	// blessed_hands +1 healing per rest heal batch? Apply as +1 per heal tick
	blessedBonus := 0
	if g.Party.HasTalent("blessed_hands") {
		blessedBonus = 1
	}
	base := healPerBatch / batchTurns
	rem := healPerBatch % batchTurns

	completed := 0
	for range batchTurns {
		if name := visibleHostile(g); name != "" {
			g.Logf("Interrupted by %s.", name)
			return completed
		}
		if g.HungerState() != startState {
			g.Logf("Rested %d turns.", completed)
			return completed
		}
		if g.Over {
			break
		}

		g.Turn++
		g.tickFood()
		// Troll regen every 3 ticks (mirrors EndPlayerTurn)
		if g.Turn%3 == 0 {
			for _, m := range g.Party.Members {
				if m.IsAlive() && normalizeRaceID(m.Race) == "troll" && m.HP < m.MaxHP {
					m.HP++
					if m.HP > m.MaxHP {
						m.HP = m.MaxHP
					}
				}
			}
		}

		heal := base
		if completed < rem {
			heal++
		}
		heal += blessedBonus
		if heal > 0 {
			for _, m := range g.Party.Members {
				if m.IsAlive() && m.HP < m.MaxHP {
					m.HP += heal
					if m.HP > m.MaxHP {
						m.HP = m.MaxHP
					}
				}
			}
		}
		if g.Turn%5 == 0 {
			for _, m := range g.Party.Members {
				if m.IsAlive() && m.HP < m.MaxHP && m.HasTalent("enduring_regen") {
					m.HP++
					if m.HP > m.MaxHP {
						m.HP = m.MaxHP
					}
				}
			}
		}
		// Elf identify ticker during rest as well
		if iv := ElfIdentifyInterval(g.Party); iv > 0 {
			if g.NextElfIdentifyTurn == 0 {
				g.NextElfIdentifyTurn = g.Turn + iv
			}
			if g.Turn >= g.NextElfIdentifyTurn {
				found := ""
				for _, it := range g.Party.Inventory {
					app := appearanceFromItem(it)
					if !IsIdentified(app) {
						found = app
						break
					}
				}
				if found != "" {
					IdentifyOnUse(found)
					g.Logf("Elven keen senses identify %s as %s.", found, friendlyTypeName(TypeForAppearance(found), "potion"))
				}
				g.NextElfIdentifyTurn = g.Turn + iv
			}
		}
		// Cleric healers_grace HoT during rest: same as EndPlayerTurn
		if g.Party != nil && g.Party.HasClass("cleric") {
			hasBard := g.Party.HasBardAlive()
			should := false
			if g.Turn%2 == 0 {
				should = true
			} else if hasBard && g.RNG != nil && g.RNG.Float64() < 0.10 {
				should = true
			}
			if should {
				for _, m := range g.Party.Members {
					if m.IsAlive() && m.HP < m.MaxHP {
						m.HP++
						if m.HP > m.MaxHP {
							m.HP = m.MaxHP
						}
					}
				}
			}
		}
		// Lorekeeper / attuned ticker during rest
		if g.Party != nil && (g.Party.HasTalent("lorekeeper") || g.Party.HasTalent("attuned")) {
			interval := 50
			if g.Party.HasBardAlive() {
				interval = 45
			}
			if g.NextLorekeeperTurn == 0 {
				g.NextLorekeeperTurn = g.Turn + interval
			}
			if g.Turn >= g.NextLorekeeperTurn {
				found := ""
				for _, it := range g.Party.Inventory {
					app := appearanceFromItem(it)
					if !IsIdentified(app) {
						found = app
						break
					}
				}
				if found != "" {
					IdentifyOnUse(found)
					g.Logf("Keen study identifies %s as %s.", found, friendlyTypeName(TypeForAppearance(found), "potion"))
				}
				g.NextLorekeeperTurn = g.Turn + interval
			}
		}

		g.applyStarvation()
		if g.Over {
			completed++
			break
		}
		g.EnemyTurn()
		g.UpdateFOV()

		completed++

		if g.HungerState() != startState {
			if name := visibleHostile(g); name != "" {
				g.Logf("Interrupted by %s.", name)
			} else {
				g.Logf("Rested %d turns.", completed)
			}
			return completed
		}
		if name := visibleHostile(g); name != "" {
			g.Logf("Interrupted by %s.", name)
			return completed
		}
	}

	g.Logf("Rested %d turns.", completed)
	return completed
}

func visibleHostile(g *Game) string {
	lvl := g.CurLevel()
	if lvl == nil {
		return ""
	}
	for _, e := range lvl.Enemies {
		if !e.IsAlive() {
			continue
		}
		if !lvl.InBounds(e.Pos) {
			continue
		}
		if lvl.Visible[e.Pos.Y][e.Pos.X] {
			return e.DisplayName()
		}
	}
	return ""
}

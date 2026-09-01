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

		heal := base
		if completed < rem {
			heal++
		}
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

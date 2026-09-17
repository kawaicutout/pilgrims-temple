package game

// dropToNextFloor moves party one floor down to StairsUp, handles FOV and logging (DUP-02).
func (g *Game) dropToNextFloor(reason string) bool {
	if g.Floor+1 >= g.Tuning.Floors {
		g.Logf("%s has no lower level -- you climb back out.", reason)
		return false
	}
	g.Floor++
	g.Party.Pos = g.CurLevel().StairsUp
	g.Logf("%s drops you to floor %d (one-way).", reason, g.Floor+1)
	g.UpdateFOV()
	return true
}

// tickAfterMove centralizes Turn++ + tickFood + regen + starvation + FOV/EnemyTurn for pitfall path (DUP-02).
func (g *Game) tickAfterMove() {
	g.Turn++
	g.tickFood()
	g.tickRegen()
	g.applyStarvation()
	g.UpdateFOV()
	if !g.Over {
		g.EnemyTurn()
		g.UpdateFOV()
	}
}

func (g *Game) handleFloorArrival() {
	if g.VisitedFloors == nil {
		g.VisitedFloors = make(map[int]bool)
	}
	if g.TransitionFiredForLevel == nil {
		g.TransitionFiredForLevel = make(map[int]bool)
	}
	if !g.VisitedFloors[g.Floor] && !g.TransitionFiredForLevel[g.Floor] {
		g.ApplyFloorTransition()
		g.VisitedFloors[g.Floor] = true
		g.TransitionFiredForLevel[g.Floor] = true
	} else {
		g.VisitedFloors[g.Floor] = true
		g.logBiomeEntry()
	}
	g.UpdateFOV()
}

func (g *Game) moveFloor(spec stairSpec) bool {
	if g.Over {
		return false
	}
	lvl := g.CurLevel()
	if g.Party.Pos != spec.src(lvl) {
		g.Logf("%s", spec.blockedMsg)
		return false
	}
	if spec.delta > 0 && g.Floor+spec.delta >= g.Tuning.Floors {
		g.Logf("The way down is sealed.")
		return false
	}
	if spec.delta < 0 && g.Floor == 0 {
		g.Logf("You are at the entrance.")
		return false
	}
	g.Floor += spec.delta
	g.Party.Pos = spec.dst(g.CurLevel())
	if spec.delta > 0 {
		g.Logf("You descend to floor %d.", g.Floor+1)
	} else {
		g.Logf("You ascend to floor %d.", g.Floor+1)
	}
	g.handleFloorArrival()
	return true
}

func (g *Game) TryStairsDown() {
	if !g.moveFloor(stairSpec{delta: 1, src: func(l *Level) Pos { return l.StairsDown }, dst: func(l *Level) Pos { return l.StairsUp }, blockedMsg: "No stairs down here."}) {
		return
	}
	g.EnemyTurn()
}

func (g *Game) TryStairsUp() {
	if g.Over {
		return
	}
	lvl := g.CurLevel()
	if g.Party.Pos == lvl.StairsUp && g.Floor == 0 && g.RelicCollected {
		g.Escaped = true
		g.Over = true
		g.Won = true
		g.Cause = "Escaped"
		g.Logf("You escape the temple with the relic! Victory - seed %d. Score %d.", g.Seed, g.CalculateScore())
		g.RecordScore()
		if err := DeleteSave(); err != nil {
			g.Logf("Save delete failed: %v.", err)
		}
		return
	}
	if !g.moveFloor(stairSpec{delta: -1, src: func(l *Level) Pos { return l.StairsUp }, dst: func(l *Level) Pos { return l.StairsDown }, blockedMsg: "No stairs up here."}) {
		return
	}
}

func (g *Game) ApplyFloorTransition() {
	// Biome entry feel: log evocative line on floor entry.
	g.logBiomeEntry()
	// Forage (druid): +100 food per bearer. Restoration (cleric): full heal all living members.
	for _, m := range g.Party.Members {
		if !m.IsAlive() {
			continue
		}
		if m.HasTalent("forage") {
			g.AddFood(100)
			g.Logf("%s forages +100 food (now %d).", m.Name, g.Food)
		}
		if m.HasTalent("restoration") {
			healed := 0
			for _, mm := range g.Party.Members {
				if mm.IsAlive() && mm.HP < mm.MaxHP {
					mm.HP = mm.MaxHP
					healed++
				}
			}
			// clear negative conditions from every living member.
			for _, mm := range g.Party.Members {
				if mm.IsAlive() {
					for _, sid := range []string{StatusHex, StatusRend, StatusBleed, StatusSpore, StatusPoison, StatusCurse, StatusSleep, StatusParalysis, StatusConfusion, StatusEntangle} {
						mm.RemoveStatus(sid)
					}
				}
			}
			if healed > 0 {
				g.Logf("%s restores the party to full health and clears afflictions.", m.Name)
			} else {
				g.Logf("%s channels restoration (party already healthy, afflictions cleared).", m.Name)
			}
			// Only one restoration proc per party per transition (avoid duplicate full-heal spam if multiple clerics).
			break
		}
	}
	// Reset second_wind per floor
	for _, m := range g.Party.Members {
		m.SecondWindUsed = false
	}
}

package game

import "fmt"

// handleVault checks locked status via Party.HasRogue (and wizard). Returns true if blocked.
func (g *Game) handleVault(f *Feature) bool {
	if f == nil || !f.IsVault() {
		return false
	}
	if f.Locked && !g.canOpenLocked() {
		g.Logf("Locked vault - need rogue or wizard.")
		return true
	}
	// Allow loot: give treasure gold, handle trap.
	treasure := f.Treasure
	if treasure == 0 {
		treasure = 30
	}
	g.Gold += treasure
	if f.Trapped {
		dmg := 2 + g.RNG.IntN(3) // 2-4
		hitIdx, actual := g.Party.ApplyDamage(g.RNG, dmg)
		g.Logf("Vault treasure +%d gold! Trap springs for %d damage!", treasure, actual)
		if g.RNG.Float64() < 0.20 {
			if hitIdx >= 0 && hitIdx < len(g.Party.Members) {
				g.Party.Members[hitIdx].ApplyStatus(StatusPoison, 6)
			}
			g.Logf("Trap poisons!")
		}
		if g.Party.LivingCount() == 0 {
			g.Over = true
			if g.Cause == "" {
				g.Cause = "Slain by vault trap"
			}
			g.Logf("You have fallen. Seed %d. Score %d.", g.Seed, g.CalculateScore())
			g.RecordScore()
		}
	} else {
		g.Logf("Vault opened +%d gold!", treasure)
	}
	g.removeFeatureAt(f.Pos, FeatureVault)
	return false
}

func (g *Game) handleForge(f *Feature) bool {
	if f == nil || !f.IsForge() {
		return false
	}
	ct := f.CostType
	if ct == "" {
		ct = "gold"
	}
	cost := f.Cost
	if cost == 0 {
		if ct == "food" {
			cost = 50
		} else {
			cost = 25
		}
	}
	if ct == "gold" {
		if g.Gold < cost {
			g.Logf("Forge needs %d gold to improve gear (you have %d).", cost, g.Gold)
			return false
		}
		g.Gold -= cost
		// Improve random living member ATK or DEF.
		members := g.Party.LivingMembers()
		if len(members) == 0 {
			return false
		}
		m := members[g.RNG.IntN(len(members))]
		if g.RNG.IntN(2) == 0 {
			m.ATK[0]++
			m.ATK[1]++
			g.Logf("Forge hammers +%d gold: %s ATK %d-%d.", cost, m.Name, m.ATK[0], m.ATK[1])
		} else {
			m.DEF++
			g.Logf("Forge tempers +%d gold: %s DEF %d.", cost, m.Name, m.DEF)
		}
	} else { // food
		if g.Food < cost {
			g.Logf("Forge needs %d food to stoke (you have %d).", cost, g.Food)
			return false
		}
		g.Food -= cost
		g.FoodFloat -= float64(cost)
		if g.Food < 0 {
			g.Food = 0
		}
		members := g.Party.LivingMembers()
		if len(members) == 0 {
			return false
		}
		m := members[g.RNG.IntN(len(members))]
		if g.RNG.IntN(2) == 0 {
			m.ATK[0]++
			m.ATK[1]++
			g.Logf("Forge stoked %d food: %s ATK %d-%d.", cost, m.Name, m.ATK[0], m.ATK[1])
		} else {
			m.MDEF++
			g.Logf("Forge quenched %d food: %s MDEF %d.", cost, m.Name, m.MDEF)
		}
	}
	g.removeFeatureAt(f.Pos, FeatureForge)
	return true
}

// TryUseForge attempts deliberate use of a forge at the party's current position.
// Returns true if a forge was present and successfully used (cost deducted, stats bumped).
func (g *Game) TryUseForge() bool {
	f := g.featureAt(g.Party.Pos)
	if f == nil || !f.IsForge() {
		return false
	}
	// Copy to avoid alias issues after removal.
	ff := *f
	used := g.handleForge(&ff)
	return used
}

// TryUseFountain attempts deliberate use of a fountain at the party's current position.
// Returns true if a fountain was present and handled (even if stale).
func (g *Game) TryUseFountain() bool {
	f := g.featureAt(g.Party.Pos)
	if f == nil || !f.IsFountain() {
		return false
	}
	ff := *f
	g.handleFountain(&ff)
	return true
}

// StartMerchant opens merchant wares at pos into g.Merchant state.
func (g *Game) StartMerchant(pos Pos) bool {
	f := g.featureAt(pos)
	if f == nil || !f.IsMerchant() {
		return false
	}
	// Use persistent wares from Feature; fallback for old saves.
	if len(f.Wares) == 0 {
		f.Wares = merchantWares(g.RNG)
	}
	if len(f.Wares) == 0 {
		g.Logf("Merchant has nothing to sell right now.")
		g.removeFeatureAt(pos, FeatureMerchant)
		return false
	}
	g.Merchant = MerchantState{Active: true, Pos: pos, Wares: f.Wares, Selected: 0}
	var offerStr string
	for i, w := range f.Wares {
		if i > 0 {
			offerStr += ", "
		}
		offerStr += fmt.Sprintf("%s (%dg)", w.Name, w.Price)
	}
	g.Logf("Merchant wares: %s -- press Enter to buy, Esc to leave.", offerStr)
	return true
}

// CancelMerchant closes merchant menu without purchase.
func (g *Game) CancelMerchant() {
	if g.Merchant.Active {
		g.Logf("You step away from the merchant.")
	}
	g.Merchant = MerchantState{}
}

// BuySelectedMerchant purchases ware at index, applies effect, removes merchant feature and advances turn.
// Returns true if purchase succeeded.
func (g *Game) BuySelectedMerchant(index int) bool {
	if !g.Merchant.Active {
		return false
	}
	if index < 0 || index >= len(g.Merchant.Wares) {
		return false
	}
	w := g.Merchant.Wares[index]
	// Build temporary merchant for BuyWare validation
	m := &Merchant{Pos: g.Merchant.Pos, Wares: g.Merchant.Wares, Scarce: true}
	if err := g.BuyWare(m, w.ID); err != nil {
		g.Logf("Merchant: %v (you have %dg).", err, g.Gold)
		return false
	}
	// Apply ware effect
	switch w.ID {
	case "ration":
		refill := g.Tuning.Food.RationRefill
		if refill <= 0 {
			refill = GetTuning().Food.RationRefill
			if refill <= 0 {
				refill = 50
			}
		}
		if g.Party.HasTalent("hoarder") {
			refill += 25
		}
		g.Food += refill
		g.FoodFloat += float64(refill)
		g.Logf("Merchant sells %s for %dg (+%d food).", w.Name, w.Price, refill)
	case "potion_heal":
		healed := 0
		for _, mem := range g.Party.Members {
			if mem.IsAlive() && mem.HP < mem.MaxHP {
				mem.HP += 10
				if mem.HP > mem.MaxHP {
					mem.HP = mem.MaxHP
				}
				healed++
			}
		}
		if healed > 0 {
			g.Logf("Merchant sells %s for %dg (healed %d members +10 HP).", w.Name, w.Price, healed)
		} else {
			g.Logf("Merchant sells %s for %dg (already at full health).", w.Name, w.Price)
		}
	case "scroll_upgrade":
		members := g.Party.LivingMembers()
		if len(members) > 0 {
			picked := members[g.RNG.IntN(len(members))]
			if g.RNG.IntN(2) == 0 {
				picked.ATK[0]++
				picked.ATK[1]++
				g.Logf("Merchant sells %s for %dg (%s ATK %d-%d).", w.Name, w.Price, picked.Name, picked.ATK[0], picked.ATK[1])
			} else {
				picked.DEF++
				g.Logf("Merchant sells %s for %dg (%s DEF %d).", w.Name, w.Price, picked.Name, picked.DEF)
			}
		} else {
			g.Logf("Merchant sells %s for %dg.", w.Name, w.Price)
		}
	default:
		g.Logf("Merchant sells %s for %dg.", w.Name, w.Price)
	}
	g.removeFeatureAt(g.Merchant.Pos, FeatureMerchant)
	g.Merchant = MerchantState{}
	return true
}

// TryUseMerchant attempts deliberate use of a merchant at the party's current position.
// Opens the merchant menu (StartMerchant) and returns true if a merchant was present.
func (g *Game) TryUseMerchant() bool {
	if g.Party == nil {
		return false
	}
	return g.StartMerchant(g.Party.Pos)
}

// StartShrine opens shrine menu at pos into g.Shrine state.
func (g *Game) StartShrine(pos Pos) bool {
	f := g.featureAt(pos)
	if f == nil || !f.IsShrine() {
		return false
	}
	g.Shrine = ShrineState{Active: true, Pos: pos, Selected: 0}
	g.Logf("Shrine offers: Add member, Resurrect, Level up, Leave -- Enter to choose, Esc to leave.")
	return true
}

// CancelShrine closes shrine menu without using it, keeping the feature.
func (g *Game) CancelShrine() {
	if g.Shrine.Active {
		g.Logf("You step away from the shrine.")
	}
	g.Shrine = ShrineState{}
}

// TryUseShrine attempts deliberate use of a shrine at the party's current position.
// Opens the shrine menu (StartShrine) and returns true if a shrine was present.
func (g *Game) TryUseShrine() bool {
	if g.Party == nil {
		return false
	}
	return g.StartShrine(g.Party.Pos)
}

// closeShrineWithCompanion refreshes buffs and identify tickers after a
// shrine added or restored a member, logs, consumes the shrine.
func (g *Game) closeShrineWithCompanion(msg string, args ...any) bool {
	g.Party.EnsureSelection()
	ApplyRaceBuffs(g.Party)
	if iv := ElfIdentifyInterval(g.Party); iv > 0 {
		g.NextElfIdentifyTurn = g.Turn + iv
	}
	if g.Party.HasTalent("lorekeeper") || g.Party.HasTalent("attuned") {
		interval := 50
		if g.Party.HasBardAlive() {
			interval = 45
		}
		g.NextLorekeeperTurn = g.Turn + interval
	}
	g.Logf(msg, args...)
	g.removeFeatureAt(g.Shrine.Pos, FeatureShrine)
	g.Shrine = ShrineState{}
	return true
}

// ExecuteShrineChoice handles shrine menu selection 0..3:
// 0 Add new party member random (free), 1 Resurrect dead member (free), 2 Gain instant level-up without XP reset, 3 Leave.
// Returns true if choice was handled (even if it logged a failure like party full). Caller may advance turn for 0-2.
func (g *Game) ExecuteShrineChoice(index int) bool {

	if !g.Shrine.Active {
		return false
	}
	if index < 0 || index > 3 {
		return false
	}
	switch index {
	case 0: // Add new party member random
		if len(g.Party.Members) >= 4 {
			g.Logf("Shrine: party already full (4).")
			return false
		}
		classes, err := LoadClasses()
		pick := "fighter"
		if err == nil && len(classes) > 0 && g.RNG != nil {
			pick = classes[g.RNG.IntN(len(classes))].ID
		}
		tmp := GeneratePartyWithClasses(g.RNG, []string{pick}, g.Level)
		if tmp == nil || len(tmp.Members) == 0 {
			g.Logf("Shrine tries to recruit, but none answer.")
			return false
		}
		m := tmp.Members[0]
		m.HP = m.MaxHP
		m.Alive = true
		g.Party.Members = append(g.Party.Members, m)
		return g.closeShrineWithCompanion("Shrine recruits %s the %s! (+)", m.Name, m.Class)
	case 1: // Resurrect dead member (most recent)
		deadIdx := g.Party.LastDead()
		if deadIdx == -1 {
			g.Logf("Shrine: no fallen pilgrims to resurrect.")
			return false
		}
		m := g.Party.Members[deadIdx]
		m.Alive = true
		m.Statuses = nil // restored clean: death ends all conditions
		m.HP = m.MaxHP
		return g.closeShrineWithCompanion("Shrine resurrects %s for free! (+)", m.Name)
	case 2: // Gain instant level-up without XP reset
		if g.LevelUpPending != nil {
			g.Logf("Shrine: level up already pending.")
			return false
		}
		oldLevel := g.Level
		g.Level++
		g.XPToNext = g.xpForNext()
		g.Logf("Shrine grants level %d! (XP %d/%d)", g.Level, g.XP, g.XPToNext)
		for _, m := range g.Party.Members {
			if !m.IsAlive() {
				continue
			}
			hpGain := 1 + g.RNG.IntN(2)
			m.MaxHP += hpGain
			m.HP += hpGain
			if g.RNG.IntN(2) == 0 {
				m.ATK[0]++
				m.ATK[1]++
			}
			if g.RNG.IntN(4) == 0 {
				m.DEF++
			}
			g.Logf("%s gains +%d HP.", m.Name, hpGain)
		}
		var picks []TalentPick
		for i, m := range g.Party.Members {
			if !m.IsAlive() {
				continue
			}
			if g.RNG.Float64() < g.Tuning.LevelUp.TalentChance {
				pick := TalentPick{MemberIdx: i, MemberName: m.Name, Class: m.Class}
				if g.RNG.Float64() < g.Tuning.LevelUp.AffixReplaceChance {
					pick.IsAffix = true
					pick.Options = []string{GetRandomAffix(g.RNG)}
					g.Logf("%s will gain an affix: %s", m.Name, FriendlyID(pick.Options[0]))
				} else {
					pick.IsAffix = false
					pick.Options = GetTalentOptions(g.RNG, m.Class, 3)
					g.Logf("%s may choose a talent.", m.Name)
				}
				picks = append(picks, pick)
			}
		}
		if len(picks) > 0 {
			g.LevelUpPending = &LevelUpState{NewLevel: g.Level, Picks: picks, Current: 0}
			g.Logf("Level up pending: %d talent picks. Press Tab to choose.", len(picks))
		}
		g.Logf("Shrine: old level %d -> %d free blessing.", oldLevel, g.Level)
		ApplyRaceBuffs(g.Party)
		if iv := ElfIdentifyInterval(g.Party); iv > 0 {
			g.NextElfIdentifyTurn = g.Turn + iv
		}
		if g.Party.HasTalent("lorekeeper") || g.Party.HasTalent("attuned") {
			interval := 50
			if g.Party.HasBardAlive() {
				interval = 45
			}
			g.NextLorekeeperTurn = g.Turn + interval
		}
		g.removeFeatureAt(g.Shrine.Pos, FeatureShrine)
		g.Shrine = ShrineState{}
		return true
	case 3: // Leave and come back later
		g.CancelShrine()
		return true
	}
	return false
}

// TryUseFeature checks current tile for deliberate features in priority Fountain -> Merchant -> Forge -> Shrine.
// Returns true if any feature was handled/opened.
func (g *Game) TryUseFeature() bool {
	if g.TryUseFountain() {
		return true
	}
	if g.TryUseMerchant() {
		return true
	}
	if g.TryUseForge() {
		return true
	}
	if g.TryUseShrine() {
		return true
	}
	return false
}

// TryCloseDoor attempts to close an adjacent open door when standing on empty ground.
// Returns true if a door was closed (consumes turn via caller).
func (g *Game) TryCloseDoor() bool {
	lvl := g.CurLevel()
	if lvl == nil || g.Party == nil {
		return false
	}
	pos := g.Party.Pos
	// Check nothing underfoot: no feature, no litter, no enemy at pos
	if g.featureAt(pos) != nil {
		return false
	}
	if lvl.LitterAt(pos) != nil {
		return false
	}
	for _, e := range lvl.Enemies {
		if e.IsAlive() && e.Pos == pos {
			return false
		}
	}
	// Find adjacent open door (cardinal first, then diagonal)
	for _, d := range []Dir{DirN, DirS, DirW, DirE, DirNW, DirNE, DirSW, DirSE} {
		np := pos.Add(d)
		if !lvl.InBounds(np) {
			continue
		}
		if lvl.IsDoor(np) && lvl.IsDoorOpen(np) {
			lvl.SetDoorOpen(np, false)
			g.Logf("You close the door.")
			return true
		}
	}
	return false
}

func (g *Game) handleDen(f *Feature) {
	if f == nil || !f.IsDen() {
		return
	}
	cnt := f.MonsterCount
	if cnt == 0 {
		cnt = 3
	}
	g.Logf("Den ahead -- %d monsters guard this lair!", cnt)
	// Den remains as marker; not removed on warning (spawn handled by TickDens)
}

// TickDens spawns 1-2 monsters from each den when player within radius 3.
// Uses level-appropriate enemy generation (pickEnemyForFloor + buildMemberFromEntry)
// and decrements MonsterCount individually. Den removed when count reaches 0.
func (g *Game) TickDens() {
	lvl := g.CurLevel()
	if lvl == nil || g.Party == nil || g.RNG == nil {
		return
	}
	for i := range lvl.Features {
		f := &lvl.Features[i]
		if f.Type != FeatureDen {
			continue
		}
		if f.MonsterCount <= 0 {
			continue
		}
		dx := g.Party.Pos.X - f.Pos.X
		if dx < 0 {
			dx = -dx
		}
		dy := g.Party.Pos.Y - f.Pos.Y
		if dy < 0 {
			dy = -dy
		}
		if max(dx, dy) > 3 {
			continue
		}
		remaining := f.MonsterCount
		spawnN := 1 + g.RNG.IntN(2)
		if spawnN > remaining {
			spawnN = remaining
		}
		for range spawnN {
			var spawnPos Pos
			found := false
			for range 20 {
				dx2 := g.RNG.IntN(5) - 2
				dy2 := g.RNG.IntN(5) - 2
				cand := Pos{f.Pos.X + dx2, f.Pos.Y + dy2}
				if cand == lvl.StairsUp || cand == lvl.StairsDown || cand == f.Pos || cand == g.Party.Pos {
					continue
				}
				if !lvl.InBounds(cand) || !lvl.Walkable(cand) {
					continue
				}
				occupied := false
				for _, e := range lvl.Enemies {
					if e != nil && e.Pos == cand {
						occupied = true
						break
					}
				}
				if occupied {
					continue
				}
				blocked := false
				for _, feat := range lvl.Features {
					if feat.Pos == cand && feat.Type != FeatureDen {
						blocked = true
						break
					}
				}
				if blocked {
					continue
				}
				spawnPos = cand
				found = true
				break
			}
			if !found {
				for _, d := range AllDirs {
					cand := f.Pos.Add(d)
					if lvl.Walkable(cand) && cand != lvl.StairsUp && cand != lvl.StairsDown && cand != g.Party.Pos {
						occupied := false
						for _, e := range lvl.Enemies {
							if e != nil && e.Pos == cand {
								occupied = true
								break
							}
						}
						if !occupied {
							spawnPos = cand
							found = true
							break
						}
					}
				}
				if !found {
					spawnPos = f.Pos
				}
			}
			entry := pickEnemyForFloor(g.RNG, g.Floor)
			mem := buildMemberFromEntry(entry, g.RNG, g.Floor)
			ep := &EnemyParty{Pos: spawnPos, Members: []*Member{mem}, Active: 0}
			lvl.Enemies = append(lvl.Enemies, ep)
			g.Logf("Den stirs -- %s emerges!", mem.Name)
		}
		f.MonsterCount -= spawnN
		if f.MonsterCount <= 0 {
			g.Logf("Den emptied.")
		} else {
			g.Logf("Den has %d monsters remaining.", f.MonsterCount)
		}
	}
	// Remove empty dens.
	dst := lvl.Features[:0]
	for _, feat := range lvl.Features {
		if feat.Type == FeatureDen && feat.MonsterCount <= 0 {
			continue
		}
		dst = append(dst, feat)
	}
	lvl.Features = dst
}

func (g *Game) handleShrine(f *Feature) {
	if f == nil || !f.IsShrine() {
		return
	}
	cleansed := false
	for _, m := range g.Party.Members {
		if m.HasStatus(StatusCurse) {
			m.RemoveStatus(StatusCurse)
			cleansed = true
		}
	}
	if cleansed {
		g.Logf("Shrine cleanses your curse.")
	}
	// Try resurrection if any dead member, else recruitment if space.
	deadIdx := g.Party.FirstDead()
	hasDead := deadIdx != -1
	canRecruit := len(g.Party.Members) < 4
	// Shrines are free (2026-09-07 decision): no cost data.
	if hasDead {
		m := g.Party.Members[deadIdx]
		m.Alive = true
		m.Statuses = nil // restored clean: death ends all conditions
		m.HP = m.MaxHP
		g.Party.EnsureSelection()
		g.Logf("Shrine resurrects %s for free! (+)", m.Name)
		g.removeFeatureAt(f.Pos, FeatureShrine)
		return
	}
	if canRecruit {
		// Recruit free (2026-09-07 decision): shrines have no costs.
		classes, err := LoadClasses()
		pick := "fighter"
		if err == nil && len(classes) > 0 && g.RNG != nil {
			pick = classes[g.RNG.IntN(len(classes))].ID
		}
		tmp := GeneratePartyWithClasses(g.RNG, []string{pick}, 1)
		if tmp != nil && len(tmp.Members) > 0 {
			m := tmp.Members[0]
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
			g.Logf("Shrine recruits %s the %s! (+)", m.Name, m.Class)
			g.removeFeatureAt(f.Pos, FeatureShrine)
			return
		}
		g.Logf("Shrine tries to recruit, but none answer.")
		return
	}
	g.Logf("Shrine glows faintly -- your party is whole and needs no resurrection.")
}

func (g *Game) handleFountain(f *Feature) {
	if f == nil || !f.IsFountain() {
		return
	}
	outs := GetFountainOutcomes()
	if len(outs) == 0 {
		g.Logf("Fountain water is stale.")
		g.removeFeatureAt(f.Pos, FeatureFountain)
		return
	}
	idx := 0
	if g.RNG != nil {
		idx = g.RNG.IntN(len(outs))
	}
	o := outs[idx]
	dh := o.DeltaHP
	if dh == 0 {
		dh = o.Delta
	}
	// blessed_hands +1 healing
	if dh > 0 && g.Party.HasTalent("blessed_hands") {
		dh++
	}
	if dh > 0 {
		for _, m := range g.Party.Members {
			if m.IsAlive() {
				m.HP += dh
				if m.HP > m.MaxHP {
					m.HP = m.MaxHP
				}
			}
		}
		g.Logf("Fountain %s: %s (+%d HP)", o.Name, o.Desc, dh)
	} else if dh < 0 {
		_, actual := g.Party.ApplyDamage(g.RNG, -dh)
		g.Logf("Fountain %s: %s (%d damage)", o.Name, o.Desc, actual)
		if g.Party.LivingCount() == 0 {
			g.Over = true
			if g.Cause == "" {
				g.Cause = "Poison"
			}
			g.Logf("You have fallen. Seed %d. Score %d.", g.Seed, g.CalculateScore())
			g.RecordScore()
		}
	} else {
		g.Logf("Fountain %s: %s", o.Name, o.Desc)
	}
	drinker := g.Party.observer()
	if o.Effect == "bless" {
		if drinker != nil {
			drinker.ApplyStatus(StatusBless, 101)
		}
		g.Logf("Blessed waters grant +1 DEF for 100 turns.")
	} else if o.Effect == "curse" {
		if drinker != nil {
			drinker.ApplyStatus(StatusCurse, 201)
		}
		g.Logf("Cursed waters weaken you -1 DEF until cured.")
	}
	g.removeFeatureAt(f.Pos, FeatureFountain)
}

func (g *Game) handleMerchant(f *Feature) {
	if f == nil || !f.IsMerchant() {
		return
	}
	if len(f.Wares) == 0 {
		f.Wares = merchantWares(g.RNG)
	}
	if len(f.Wares) == 0 {
		g.Logf("Merchant (M) has nothing to sell right now.")
		g.removeFeatureAt(f.Pos, FeatureMerchant)
		return
	}
	m := &Merchant{Pos: f.Pos, Wares: f.Wares, Scarce: true}
	// Build offer list
	var offerStr string
	for i, w := range f.Wares {
		if i > 0 {
			offerStr += ", "
		}
		offerStr += fmt.Sprintf("%s (%dg)", w.Name, w.Price)
	}
	// Find cheapest affordable ware
	cheapestIdx := -1
	cheapestPrice := 1 << 30
	for i, w := range f.Wares {
		if g.Gold >= w.Price && w.Price < cheapestPrice {
			cheapestPrice = w.Price
			cheapestIdx = i
		}
	}
	if cheapestIdx == -1 {
		g.Logf("Merchant offers: %s -- need more gold (you have %dg).", offerStr, g.Gold)
		return
	}
	w := f.Wares[cheapestIdx]
	if err := g.BuyWare(m, w.ID); err != nil {
		g.Logf("Merchant: %v", err)
		return
	}
	// Apply ware effect
	switch w.ID {
	case "ration":
		refill := g.Tuning.Food.RationRefill
		if refill <= 0 {
			refill = GetTuning().Food.RationRefill
			if refill <= 0 {
				refill = 50
			}
		}
		if g.Party.HasTalent("hoarder") {
			refill += 25
		}
		g.Food += refill
		g.Logf("Merchant sells %s for %dg (+%d food).", w.Name, w.Price, refill)
	case "potion_heal":
		healed := 0
		for _, mem := range g.Party.Members {
			if mem.IsAlive() && mem.HP < mem.MaxHP {
				mem.HP += 10
				if mem.HP > mem.MaxHP {
					mem.HP = mem.MaxHP
				}
				healed++
			}
		}
		if healed > 0 {
			g.Logf("Merchant sells %s for %dg (healed %d members +10 HP).", w.Name, w.Price, healed)
		} else {
			g.Logf("Merchant sells %s for %dg (already at full health).", w.Name, w.Price)
		}
	case "scroll_upgrade":
		members := g.Party.LivingMembers()
		if len(members) > 0 {
			picked := members[g.RNG.IntN(len(members))]
			if g.RNG.IntN(2) == 0 {
				picked.ATK[0]++
				picked.ATK[1]++
				g.Logf("Merchant sells %s for %dg (%s ATK %d-%d).", w.Name, w.Price, picked.Name, picked.ATK[0], picked.ATK[1])
			} else {
				picked.DEF++
				g.Logf("Merchant sells %s for %dg (%s DEF %d).", w.Name, w.Price, picked.Name, picked.DEF)
			}
		} else {
			g.Logf("Merchant sells %s for %dg.", w.Name, w.Price)
		}
	default:
		g.Logf("Merchant sells %s for %dg.", w.Name, w.Price)
	}
	g.removeFeatureAt(f.Pos, FeatureMerchant)
}

// IsTrapAware reports deterministic trap detection for a feature.
func (g *Game) IsTrapAware(f *Feature) bool {
	if f == nil || !f.Hidden {
		return true
	}
	if g.Party.HasRogueOrWizard() || g.Wizard {
		return true
	}
	if r := SynergyTremorsense(g.Party); r > 0 {
		dx := g.Party.Pos.X - f.Pos.X
		if dx < 0 {
			dx = -dx
		}
		dy := g.Party.Pos.Y - f.Pos.Y
		if dy < 0 {
			dy = -dy
		}
		if dx+dy <= r {
			return true
		}
	}
	if g.Party.HasTalent("attuned") || g.Party.HasTalent("attuned_tag") {
		dx := g.Party.Pos.X - f.Pos.X
		if dx < 0 {
			dx = -dx
		}
		dy := g.Party.Pos.Y - f.Pos.Y
		if dy < 0 {
			dy = -dy
		}
		if dx+dy <= 1+1 {
			return true
		}
	}
	return false
}

func (g *Game) handlePitfall(f *Feature) bool {
	if f == nil || !f.IsPitfall() {
		return false
	}
	mover := g.Party.observer()
	if mover != nil && mover.HasStatus(StatusLevitation) {
		g.Logf("You float over the pitfall.")
		return false
	}
	// Detection: class gate, wizard mode, dwarf tremorsense in range,
	// or attuned +1 range. Stochastic sources (steady hands) and
	// avoidance (levitation, ghost step) stay at their call sites:
	// display code must never roll RNG.
	aware := g.IsTrapAware(f)
	if f.Hidden && !aware && g.Party.HasTalent("steady_hands") && g.RNG != nil && g.RNG.Float64() < 0.10 {
		aware = true
	}
	// ghost_step 50% trap ignore on move when hidden & aware (also wander skip elsewhere)
	if f.Hidden && aware && g.Party.HasTalent("ghost_step") && g.RNG != nil && g.RNG.Float64() < 0.50 {
		g.Logf("Ghost step: you slip past the pitfall.")
		return false
	}
	if f.Hidden && !aware {
		dmg := f.Damage
		if dmg == 0 {
			dmg = 2 + g.RNG.IntN(3)
		}
		hitIdx, actual := g.Party.ApplyDamage(g.RNG, dmg)
		g.Logf("Hidden pitfall! You fall -- %d damage!", actual)
		if g.RNG.Float64() < 0.10 {
			if hitIdx >= 0 && hitIdx < len(g.Party.Members) {
				g.Party.Members[hitIdx].ApplyStatus(StatusPoison, 4)
			}
			g.Logf("Trap poisons!")
		}
		if g.Party.LivingCount() == 0 {
			g.Over = true
			if g.Cause == "" {
				g.Cause = "Fell into pit"
			}
			g.Logf("You have fallen. Seed %d. Score %d.", g.Seed, g.CalculateScore())
			g.RecordScore()
			return true
		}
		g.dropToNextFloor("Pitfall")
		g.removeFeatureAt(f.Pos, FeaturePitfall)
		return true
	}
	if f.Hidden && aware {
		g.Logf("You spot a hidden pitfall and step around its edge... but the floor gives way!")
	}
	if !f.Hidden {
		g.Logf("Pitfall ahead -- one-way drop to the next level.")
	}
	g.Party.Pos = f.Pos
	g.dropToNextFloor("Pitfall")
	g.removeFeatureAt(f.Pos, FeaturePitfall)
	return true
}

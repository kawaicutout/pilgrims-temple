package game

import (
	"fmt"
	"math/rand/v2"
	"unicode"
)

// Game holds run state.
type MerchantState struct {
	Active   bool   `json:"active"`
	Pos      Pos    `json:"pos"`
	Wares    []Ware `json:"wares"`
	Selected int    `json:"selected"`
}

type ShrineState struct {
	Active   bool `json:"active"`
	Pos      Pos  `json:"pos"`
	Selected int  `json:"selected"`
}

type Game struct {
	Seed                    int64         `json:"seed"`
	RNG                     *rand.Rand    `json:"-"`
	Tuning                  Tuning        `json:"tuning"`
	Levels                  []*Level      `json:"levels"`
	Floor                   int           `json:"floor"`
	Party                   *Party        `json:"party"`
	Log                     []string      `json:"log"`
	Turn                    int           `json:"turn"`
	Food                    int           `json:"food"`
	FoodFloat               float64       `json:"foodFloat"`
	Level                   int           `json:"level"`
	XP                      int           `json:"xp"`
	XPToNext                int           `json:"xpToNext"`
	LevelUpPending          *LevelUpState `json:"levelUpPending"`
	Gold                    int           `json:"gold"`
	Kills                   int           `json:"kills"`
	Escaped                 bool          `json:"escaped"`
	Over                    bool          `json:"over"`
	Won                     bool          `json:"won"`
	Quit                    bool          `json:"quit"`
	Cause                   string        `json:"cause"`
	Look                    *LookState    `json:"look"`
	ThrowPending            ThrowState    `json:"throwPending"`
	UsePending              UseState      `json:"usePending"`
	Relic                   Pos           `json:"relic"`
	Wizard                  bool          `json:"wizard"`
	WizardReveal            bool          `json:"wizardReveal"`
	HelpActive              bool          `json:"helpActive"`
	VisitedFloors           map[int]bool  `json:"visitedFloors"`
	TransitionFiredForLevel map[int]bool  `json:"transitionFiredForLevel"`
	RelicCollected          bool          `json:"relicCollected"`
	NextAmbienceTurn        int           `json:"nextAmbienceTurn"`
	NextElfIdentifyTurn     int           `json:"nextElfIdentifyTurn"`
	NextLorekeeperTurn      int           `json:"nextLorekeeperTurn"`
	Merchant                MerchantState `json:"merchant"`
	Shrine                  ShrineState   `json:"shrine"`
	WaitRefrainActive       bool          `json:"-"`
}

func NewGame(seed int64, tuning Tuning) *Game {
	SetGlobalTuning(tuning)
	rng := rand.New(rand.NewPCG(uint64(seed), 0x9e3779b97f4a7c15))
	InitIdentificationSeed(seed)
	g := &Game{
		Seed: seed, RNG: rng, Tuning: tuning,
		Food: tuning.Food.StartClock, FoodFloat: float64(tuning.Food.StartClock), Level: 1,
		VisitedFloors: make(map[int]bool), TransitionFiredForLevel: make(map[int]bool),
	}
	g.XPToNext = g.xpForNext()
	g.Levels = make([]*Level, tuning.Floors)
	for i := range tuning.Floors {
		lvl := NewLevel(tuning.Map.Width, tuning.Map.Height)
		lvl.Generate(rng, i)
		g.Levels[i] = lvl
	}
	// Init ambience ticker: first ambience 30-60 turns from start.
	g.NextAmbienceTurn = 30 + rng.IntN(31)
	// Place relic on final floor down stairs
	final := g.Levels[tuning.Floors-1]
	g.Relic = final.StairsDown
	final.Set(g.Relic, TileRelic)
	g.Party = GenerateParty(rng, 1)
	ApplyRaceBuffs(g.Party)
	lootPartyForVerdant = g.Party
	if iv := ElfIdentifyInterval(g.Party); iv > 0 {
		g.NextElfIdentifyTurn = g.Turn + iv
	}
	start := g.Levels[0].StairsUp
	g.Party.Pos = start
	g.Floor = 0
	g.VisitedFloors[0] = true
	g.TransitionFiredForLevel[0] = true
	g.Logf("Seed %d -- Pilgrims' Temple, %d floors.", seed, tuning.Floors)
	g.Logf("You stand at the temple threshold.")
	// Debug log for deeper enemy talents/affixes
	for fi, lvl := range g.Levels {
		if fi < 3 {
			continue
		}
		for _, e := range lvl.Enemies {
			for _, m := range e.Members {
				if len(m.Talents) > 0 {
					for _, tl := range m.Talents {
						g.Logf("enemy gains talent %s (%s floor %d)", FriendlyID(tl), FriendlyID(m.Class), fi+1)
					}
				}
				if len(m.Affixes) > 0 {
					for _, af := range m.Affixes {
						g.Logf("enemy gains affix %s (%s floor %d)", FriendlyID(af), FriendlyID(m.Class), fi+1)
					}
				}
			}
		}
	}
	g.logBiomeEntry()
	g.UpdateFOV()
	return g
}

// ThrowState holds throw cursor state. Distinct from LookState.
type ThrowState struct {
	Active     bool   `json:"active"`
	Appearance string `json:"appearance"`
	Cursor     Pos    `json:"cursor"`
}

// StartThrow begins throw targeting with the given potion appearance.
// Cursor starts at party position.
func (g *Game) StartThrow(appearance string) {
	if g.Party == nil {
		return
	}
	g.ThrowPending = ThrowState{Active: true, Appearance: appearance, Cursor: g.Party.Pos}
	g.Logf("Throw %s: move cursor (hjkl/arrows), Enter to throw, Esc to cancel.", appearance)
}

// CancelThrow clears throw pending state.
func (g *Game) CancelThrow() {
	g.ThrowPending = ThrowState{}
}

// ThrowAt throws the stored appearance at target, consumes the potion, advances turn and clears pending.
func (g *Game) ThrowAt(target Pos) bool {
	if !g.ThrowPending.Active {
		return false
	}
	appearance := g.ThrowPending.Appearance
	g.ThrowPending = ThrowState{}
	return g.TryThrowAppearance(appearance, target)
}

// TryThrowAppearance throws the potion with given appearance at target Pos.
// Consumes the matching potion (not first), applies effect, advances turn.
// This is the cursor-based throw path; TryThrowPotion remains as deprecated Dir wrapper.
func (g *Game) TryThrowAppearance(appearance string, target Pos) bool {
	if g.Party == nil {
		g.Logf("No potions to throw.")
		return false
	}
	idx := -1
	for i, it := range g.Party.Inventory {
		if it.Kind == "potion" && appearanceFromItem(it) == appearance {
			idx = i
			break
		}
	}
	if idx == -1 {
		// fallback: if appearance not matched (e.g. identified name), try by type
		for i, it := range g.Party.Inventory {
			if it.Kind == "potion" {
				if friendlyTypeName(TypeForAppearance(appearance), "potion") == it.Name || it.Name == appearance {
					idx = i
					break
				}
			}
		}
	}
	if idx == -1 {
		g.Logf("No %s potion to throw.", appearance)
		return false
	}
	it := g.Party.Inventory[idx]
	g.Party.Inventory = append(g.Party.Inventory[:idx], g.Party.Inventory[idx+1:]...)
	trueType := TypeForAppearance(appearance)
	if trueType == "" {
		trueType = it.ID
	}
	newlyIdentified := IdentifyOnUse(appearance)
	typeName := friendlyTypeName(trueType, it.Kind)
	var targetEnemy *EnemyParty
	if lvl := g.CurLevel(); lvl != nil {
		for _, e := range lvl.Enemies {
			if e.IsAlive() && e.Pos == target {
				targetEnemy = e
				break
			}
		}
	}
	if newlyIdentified {
		g.Logf("Threw %s potion - identified as %s at (%d,%d)!", appearance, typeName, target.X, target.Y)
	} else if IsIdentified(appearance) {
		g.Logf("Threw %s potion (%s) at (%d,%d).", appearance, typeName, target.X, target.Y)
	} else {
		g.Logf("Threw %s at (%d,%d).", it.Name, target.X, target.Y)
	}
	// Cursor throw — reuse consumable hub (enemy-targeted).
	g.applyPotionEffect(trueType, false, targetEnemy, nil)
	g.EndPlayerTurn("")
	return true
}
// UseState holds use cursor state for targeted potions/scrolls.
type UseState struct {
	Active     bool   `json:"active"`
	Appearance string `json:"appearance"`
	Kind       string `json:"kind"`
	Cursor     Pos    `json:"cursor"`
}

// StartUse begins use targeting with the given appearance.
func (g *Game) StartUse(appearance string, kind string) {
	if g.Party == nil {
		return
	}
	g.UsePending = UseState{Active: true, Appearance: appearance, Kind: kind, Cursor: g.Party.Pos}
	g.Logf("Use %s: move cursor (hjkl/arrows), Enter to use, Esc to cancel.", appearance)
}

// CancelUse clears use pending state.
func (g *Game) CancelUse() {
	g.UsePending = UseState{}
}

// UseAt uses the stored appearance at target, consumes item, advances turn and clears pending.
func (g *Game) UseAt(target Pos) bool {
	if !g.UsePending.Active {
		return false
	}
	appearance := g.UsePending.Appearance
	g.UsePending = UseState{}
	return g.TryUseAppearanceAt(appearance, target)
}


type LevelUpState struct {
	NewLevel int
	Picks    []TalentPick
	Current  int
	Cursor   int
}

type TalentPick struct {
	MemberIdx  int
	MemberName string
	Class      string
	IsAffix    bool
	Options    []string
}

func (g *Game) xpForNext() int {
	base := g.Tuning.LevelUp.XPBase
	if base <= 0 {
		base = GetTuning().LevelUp.XPBase
		if base <= 0 {
			base = 100
		}
	}
	factor := g.Tuning.LevelUp.XPFactor
	if factor <= 0 {
		factor = GetTuning().LevelUp.XPFactor
		if factor <= 0 {
			factor = 1.5
		}
	}
	step := int(float64(base) * (factor - 1))
	if step <= 0 {
		step = 50
	}
	return base + step*(g.Level-1)
}
func (g *Game) GainXP(amount int) {
	if g.Over || g.LevelUpPending != nil {
		return
	}
	if bonus := SynergyXPBonus(g.Party); bonus > 0 {
		extra := int(float64(amount) * bonus)
		if extra > 0 {
			amount += extra
			g.Logf("Synergy bonus +%d XP.", extra)
		}
	}
	if g.Party != nil {
		inspiringCount := 0
		for _, m := range g.Party.Members {
			if m.IsAlive() && m.HasTalent("inspiring") {
				inspiringCount++
			}
		}
		if inspiringCount > 0 {
			extra := int(float64(amount) * 0.05 * float64(inspiringCount))
			if extra > 0 {
				amount += extra
				g.Logf("Inspiring bonus +%d XP.", extra)
			}
		}
	}
	g.XP += amount
	g.Logf("Gained %d XP (total %d/%d).", amount, g.XP, g.XPToNext)
	for g.XP >= g.XPToNext {
		g.LevelUp()
	}
}

func (g *Game) LevelUp() {
	g.XP -= g.XPToNext
	g.Level++
	g.XPToNext = g.xpForNext()
	g.Logf("Level up! Party is now level %d.", g.Level)
	for _, m := range g.Party.Members {
		if !m.IsAlive() {
			continue
		}
		hpGain := 1 + g.RNG.IntN(2)
		// Human +1 HP per level
		if normalizeRaceID(m.Race) == "human" {
			hpGain += 1
		}
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
	ApplyRaceBuffs(g.Party)
	// Refresh elf identify interval after level up (party may have changed)
	if iv := ElfIdentifyInterval(g.Party); iv > 0 {
		if g.NextElfIdentifyTurn == 0 || g.NextElfIdentifyTurn <= g.Turn {
			g.NextElfIdentifyTurn = g.Turn + iv
		}
	} else {
		g.NextElfIdentifyTurn = 0
	}
}

func (g *Game) ApplyTalentPick(pickIdx int, optionIdx int) {
	if g.LevelUpPending == nil || pickIdx < 0 || pickIdx >= len(g.LevelUpPending.Picks) {
		return
	}
	pick := g.LevelUpPending.Picks[pickIdx]
	if pick.IsAffix {
		if len(pick.Options) == 0 {
			g.Logf("Affix pick has no options; skipping.")
		} else {
			m := g.Party.Members[pick.MemberIdx]
			affixID := pick.Options[0]
			m.Affixes = append(m.Affixes, affixID)
			g.Logf("%s gains affix %s.", m.Name, FriendlyID(affixID))
			ApplyAffixMod(m, affixID)
		}
	} else {
		if optionIdx < 0 || optionIdx >= len(pick.Options) {
			return
		}
		talentID := pick.Options[optionIdx]
		m := g.Party.Members[pick.MemberIdx]
		m.Talents = append(m.Talents, talentID)
		g.Logf("%s learns talent %s.", m.Name, FriendlyID(talentID))
		switch talentID {
		case "tough":
			m.MaxHP += 4
			m.HP += 4
		case "keen":
			m.ATK[0]++
			m.ATK[1]++
		case "weapon_master":
			m.ATK[0] += 2
			m.ATK[1] += 2
		case "burden_bearer":
			if m.Carry == 0 {
				m.Carry = 5
			}
			m.Carry += 3
		case "light_bearer":
			m.Light++
		case "enduring_regen":
			// passive: handled in EndPlayerTurn/RestBatch per 5 ticks
		case "hoarder":
			// passive refill bonus handled on ration use; no instant stat
			lootPartyForVerdant = g.Party
		case "enduring":
			m.MaxHP += 4
			m.HP += 4
		case "veterans_grip":
			m.ATK[0]++
			m.ATK[1]++
		case "verdant":
			lootPartyForVerdant = g.Party
		case "evasion", "nimble", "second_wind", "blessed_hands", "tithe":
			fallthrough
		case "lorekeeper", "resonant", "attuned_tag", "cleave", "ghost_step", "restful":
			fallthrough
		case "inspiring", "refrain", "attuned", "counterspell", "shrug", "radiant":
			fallthrough
		case "deitys_gift", "forage", "restoration", "iron_will", "ward", "steady_hands":
			// wired via HasTalent branches elsewhere; no instant stat
		}
	}
	g.LevelUpPending.Current++
	if g.LevelUpPending.Current >= len(g.LevelUpPending.Picks) {
		g.LevelUpPending = nil
		g.Logf("Level up complete.")
	} else {
		g.LevelUpPending.Cursor = 0
	}
}

func (g *Game) MoveLevelUpCursor(delta int) {
	if g.LevelUpPending == nil || len(g.LevelUpPending.Picks) == 0 {
		return
	}
	pick := g.LevelUpPending.Picks[g.LevelUpPending.Current]
	if pick.IsAffix {
		return
	}
	n := len(pick.Options)
	if n == 0 {
		return
	}
	g.LevelUpPending.Cursor = (g.LevelUpPending.Cursor + delta) % n
	if g.LevelUpPending.Cursor < 0 {
		g.LevelUpPending.Cursor += n
	}
}

func (g *Game) CurLevel() *Level { return g.Levels[g.Floor] }

func (g *Game) UpdateFOV() {
	lvl := g.CurLevel()
	var eyes *Member
	if g.Party != nil {
		eyes = g.Party.observer()
	}
	if eyes != nil && eyes.HasStatus(StatusEnlightenment) {
		// Enlightenment reveals entire floor.
		for y := range lvl.H {
			for x := range lvl.W {
				lvl.Seen[y][x] = true
				lvl.Visible[y][x] = true
			}
		}
		return
	}
	if eyes != nil && eyes.HasStatus(StatusBlind) {
		ComputeFOV(lvl, g.Party.Pos, 2)
		if g.WizardReveal {
			for y := range lvl.H {
				for x := range lvl.W {
					lvl.Seen[y][x] = true
					lvl.Visible[y][x] = true
				}
			}
		}
		return
	}
	ComputeFOV(lvl, g.Party.Pos, g.Party.BestLight())
	if g.WizardReveal {
		for y := range lvl.H {
			for x := range lvl.W {
				lvl.Seen[y][x] = true
				lvl.Visible[y][x] = true
			}
		}
	}
}

func (g *Game) HungerState() string {
	if g.Tuning.Food.StartClock <= 0 {
		return "Ok"
	}
	floatFood := g.FoodFloat
	if floatFood == 0 && g.Food != 0 {
		floatFood = float64(g.Food)
	}
	ratio := floatFood / float64(g.Tuning.Food.StartClock)
	if ratio <= g.Tuning.Food.StarvingThreshold {
		return "Starving"
	}
	if ratio <= g.Tuning.Food.HungryThreshold {
		return "Hungry"
	}
	return "Ok"
}

func (g *Game) hasBardAlive() bool {
	for _, m := range g.Party.Members {
		if m.IsAlive() && m.Class == "bard" {
			return true
		}
	}
	return false
}

func (g *Game) frugalBonus() float64 {
	hasBard := g.hasBardAlive()
	bonus := 0.0
	for _, m := range g.Party.Members {
		if !m.IsAlive() || m.Class != "druid" {
			continue
		}
		if hasBard {
			bonus += 0.275
		} else {
			bonus += 0.25
		}
	}
	return bonus
}

func (g *Game) tickFood() {
	if g.Tuning.Food.PerMemberPerTurn <= 0 {
		return
	}
	// Sync if Food was manually set to zero (tests) or diverges.
	if g.Food == 0 && g.FoodFloat != 0 {
		g.FoodFloat = 0
	}
	if g.FoodFloat == 0 && g.Food != 0 {
		g.FoodFloat = float64(g.Food)
	}
	if g.Party.LivingCount() == 0 {
		g.Food = int(g.FoodFloat)
		if g.FoodFloat < 0 {
			g.FoodFloat = 0
			g.Food = 0
		}
		return
	}
	cost := 0.0
	for _, m := range g.Party.Members {
		if !m.IsAlive() {
			continue
		}
		c := float64(g.Tuning.Food.PerMemberPerTurn)
		if normalizeRaceID(m.Race) == "troll" && m.MaxHP > 0 && m.HP*2 > m.MaxHP {
			c *= 2
		}
		if m.HasStatus(StatusHaste) {
			c *= 0.8
		}
		if m.HasStatus(StatusSlow) {
			c *= 1.2
		}
		cost += c
	}
	cost -= g.frugalBonus()
	if cost < 0 {
		cost = 0
	}
	g.FoodFloat -= cost
	if g.FoodFloat < 0 {
		g.FoodFloat = 0
	}
	g.Food = int(g.FoodFloat)
	if g.Food < 0 {
		g.Food = 0
	}
}

func (g *Game) applyStarvation() {
	if g.Food != 0 && g.FoodFloat > 0 {
		return
	}
	// Also consider FoodFloat ==0 or Food==0 as starving.
	// Apply -1 HP to every living member after regen/food.
	any := false
	for _, m := range g.Party.Members {
		if !m.IsAlive() {
			continue
		}
		any = true
		m.HP--
		if m.HP <= 0 {
			m.HP = 0
			m.Alive = false
		}
	}
	if !any {
		return
	}
	// Log starvation drain.
	g.Logf("Starvation drains your party.")
	g.Party.EnsureSelection()
	if g.Party.LivingCount() == 0 {
		g.Over = true
		g.Won = false
		if g.Cause == "" {
			g.Cause = "Starvation"
		}
		g.Logf("You have succumbed to starvation. Seed %d.", g.Seed)
		g.RecordScore()
	}
}

func (g *Game) Logf(fmtStr string, args ...any) {
	s := fmt.Sprintf(fmtStr, args...)
	if len(s) > 0 {
		rs := []rune(s)
		if unicode.IsLower(rs[0]) {
			rs[0] = unicode.ToUpper(rs[0])
			s = string(rs)
		}
	}
	g.Log = append(g.Log, s)
	if len(g.Log) > g.Tuning.Layout.LogLines {
		g.Log = g.Log[len(g.Log)-g.Tuning.Layout.LogLines:]
	}
}

// featureAt returns the feature at pos or nil.
func (g *Game) featureAt(pos Pos) *Feature {
	lvl := g.CurLevel()
	for i := range lvl.Features {
		if lvl.Features[i].Pos == pos {
			return &lvl.Features[i]
		}
	}
	return nil
}

func (g *Game) removeFeatureAt(pos Pos, typ FeatureType) {
	lvl := g.CurLevel()
	dst := lvl.Features[:0]
	for _, f := range lvl.Features {
		if f.Pos == pos && f.Type == typ {
			continue
		}
		dst = append(dst, f)
	}
	lvl.Features = dst
}

// canOpenLocked reports whether party can open a locked vault (rogue/wizard or wizard mode).
func (g *Game) canOpenLocked() bool { return g.Party.HasRogueOrWizard() || g.Wizard }

// AddFood adds n to Food and FoodFloat together (DUP-16).
func (g *Game) AddFood(n int) {
	g.Food += n
	g.FoodFloat += float64(n)
}

// dealPartyDamage applies raw damage (DEF/MDEF) with cause; returns true if any survivor remains. (DUP-13)
func (g *Game) dealPartyDamage(raw int, cause string, isMagic bool) bool {
	_, _ = g.Party.ApplyDamageWithType(g.RNG, raw, isMagic)
	if g.Party.LivingCount() == 0 {
		g.Over = true
		if g.Cause == "" {
			g.Cause = cause
		}
		g.Logf("You have fallen. Seed %d. Score %d.", g.Seed, g.CalculateScore())
		g.RecordScore()
		return false
	}
	return true
}

// tickRegen handles periodic heals via schedule table — troll(3), natural(10), enduring(5) (DUP-07).
func (g *Game) tickRegen() {
	if g.Turn%3 == 0 {
		TrollRegenTick(g.Party)
	}
	if g.Turn%10 == 0 {
		for _, m := range g.Party.Members {
			if m.IsAlive() && m.HP < m.MaxHP {
				m.HP++
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
}



// HelpEntry is one help line for HelpLines table (DUP-11).
type HelpEntry struct {
	Text string
	FG   string
}

// HelpLines returns help overlay lines table used by RenderHelpOverlay and frontends (DUP-11).
func HelpLines() []HelpEntry {
	return []HelpEntry{
		{Text: "PILGRIMS' TEMPLE - HELP", FG: "gold-bright"},
		{Text: "q / w / e / r  - select member 1-4 (free)", FG: "gray-1"},
		{Text: "Move: arrows, numpad 1-9, hjkl + y b n + 9 (NE)", FG: "gray-1"},
		{Text: "5 / . / Space  - wait 1 turn", FG: "gray-1"},
		{Text: "z / Z  - rest: 10-turn batch, 15 HP, ends on hostile/hunger", FG: "gray-1"},
		{Text: "g  - contextual use: pickup, or on fountain/merchant/forge/vault/shrine/pitfall", FG: "gray-1"},
		{Text: "u/U - use menu (potions/scrolls) -> cursor targeting", FG: "gray-1"},
		{Text: "t  - throw potion (menu + cursor)", FG: "gray-1"},
		{Text: "v  - look: move cursor, v/Enter/Esc to examine", FG: "gray-1"},
		{Text: "> / <  - stairs down / up", FG: "gray-1"},
		{Text: "?  - help (this overlay)", FG: "gold"},
		{Text: "Esc - quit to menu", FG: "gray-1"},
	}
}

// CursorState unifies Throw/Use/Look cursor (DUP-14).
type CursorState struct {
	Active     bool
	Cursor     Pos
	Appearance string
}

func (g *Game) handleCursor(state *CursorState, dir Dir) bool {
	if state == nil || !state.Active {
		return false
	}
	// Simple cursor movement placeholder — actual movement handled by existing Throw/Use handlers
	next := state.Cursor.Add(dir)
	if g.CurLevel().InBounds(next) {
		state.Cursor = next
	}
	return true
}

// Frontend dispatch deferred: shared AppState + Dispatch(State,Frame) would live in game/app (DUP-15).
// Low-risk defer: current duplication between cmd/terminal and cmd/wasm is documented here;
// moving would require cross-package import cycle audit, deferred with comment.


// Action results
type ActionResult struct {
	Moved     bool
	Attacked  bool
	Descended bool
	Ascended  bool
}

func (g *Game) TryMove(dir Dir) ActionResult {
	if g.Over {
		return ActionResult{}
	}
	lvl := g.CurLevel()
	mover := g.Party.observer()
	moverParalyzed := mover != nil && (mover.HasStatus(StatusParalysis) || mover.HasStatus(StatusStun))
	if moverParalyzed {
		if mover.HasStatus(StatusStun) {
			g.Logf("You are stunned and cannot move!")
		} else {
			g.Logf("You are paralyzed and cannot move!")
		}
		g.EndPlayerTurn("")
		return ActionResult{}
	}
	if mover != nil && (mover.HasStatus(StatusEntangle) || mover.HasStatus(StatusSleep)) {
		g.Logf("You are rooted and cannot move!")
		g.EndPlayerTurn("")
		return ActionResult{}
	}
	if mover != nil && mover.HasStatus(StatusConfusion) {
		dirs := []Dir{DirN, DirS, DirW, DirE}
		dir = dirs[g.RNG.IntN(len(dirs))]
		g.Logf("You stumble %s in confusion.", dirName(dir))
	}
	next := g.Party.Pos.Add(dir)
	if lvl.IsDoor(next) && lvl.IsDoorClosed(next) {
		if lvl.vaultLockedNear(next, 4) && !g.canOpenLocked() {
			g.Logf("The vault door is locked -- need rogue or wizard.")
			return ActionResult{}
		}
		lvl.SetDoorOpen(next, true)
		g.Logf("You open the door.")
		g.Party.Active = g.Party.Selected
		g.EndPlayerTurn("")
		return ActionResult{Moved: false}
	}
	// Stay in place (wait) if dir none - silent per UI parity
	if dir == DirNone {
		g.Party.Active = g.Party.Selected
		// refrain: on wait, waiter with talent heals others (handled in EndPlayerTurn)
		if g.Party != nil && g.Party.Active >= 0 && g.Party.Active < len(g.Party.Members) {
			waiter := g.Party.Members[g.Party.Active]
			if waiter.IsAlive() && waiter.HasTalent("refrain") {
				g.WaitRefrainActive = true
			}
		}
		g.EndPlayerTurn("")
		return ActionResult{Moved: false}
	}
	if !lvl.InBounds(next) || !lvl.Walkable(next) {
		if lit := lvl.LitterAt(next); lit != nil && lit.BlocksMovement {
			if lit.Category == "impassable" {
				if msg, ok := litterAltBump(lit.Kind); ok {
					g.Logf("%s", msg)
				} else {
					g.Logf("You bump into a %s.", FriendlyID(lit.Kind))
				}
				return ActionResult{}
			}
			if lit.Category == "destructible" {
				// Find mutable reference by position
				idx := -1
				for i := range lvl.Litter {
					if lvl.Litter[i].Pos == next {
						idx = i
						break
					}
				}
				if idx >= 0 {
					obj := &lvl.Litter[idx]
					obj.Hits++
					if obj.Hits == 1 {
						g.Logf("The %s blocks the way.", FriendlyID(obj.Kind))
						return ActionResult{}
					}
					// Second+ bump: attack it; requires value (HP) to break.
					mem := g.Party.Members[g.Party.Selected]
					base := (mem.ATK[0] + mem.ATK[1]) / 2
					if mem.HasStatus(StatusStrength) {
						base += 2
					}
					if base < 2 {
						base = 2
					}
					dmg := base + g.RNG.IntN(3)
					obj.HP -= dmg
					g.Party.Active = g.Party.Selected
					if obj.HP <= 0 {
						g.Logf("You smash the %s for %d damage -- it shatters! (%d/%d)", FriendlyID(obj.Kind), dmg, 0, obj.MaxHP)
						lvl.Litter = append(lvl.Litter[:idx], lvl.Litter[idx+1:]...)
					} else {
						g.Logf("You strike the %s for %d damage (%d/%d HP).", FriendlyID(obj.Kind), dmg, obj.HP, obj.MaxHP)
					}
					g.EndPlayerTurn("")
					return ActionResult{Attacked: true}
				}
				g.Logf("You bump into a %s.", FriendlyID(lit.Kind))
				return ActionResult{}
			}
			g.Logf("You bump into a %s.", FriendlyID(lit.Kind))
		} else {
			g.Logf("You bump the wall.")
		}
		return ActionResult{}
	}
	for _, e := range lvl.Enemies {
		if e.IsAlive() && e.Pos == next {
			g.Party.Active = g.Party.Selected
			attackerMember := g.Party.Members[g.Party.Active]
			attacker := attackerMember.Name
			dmg, hitIdx, killed, struckTwice := PlayerBumpEnemy(g.RNG, g.Party, e)
			// Dwarf +2 dmg when below 50% HP
			if normalizeRaceID(attackerMember.Race) == "dwarf" && attackerMember.MaxHP > 0 && attackerMember.HP*2 < attackerMember.MaxHP {
				// apply extra 2 damage to the hit enemy member if still alive
				if hitIdx >= 0 && hitIdx < len(e.Members) {
					tgt := e.Members[hitIdx]
					if tgt.IsAlive() {
						tgt.HP -= 2
						dmg += 2
						if tgt.HP <= 0 {
							tgt.HP = 0
							tgt.Alive = false
							killed = true
						}
					}
				} else {
					// fallback: apply to first alive
					for _, m := range e.Members {
						if m.IsAlive() {
							m.HP -= 2
							dmg += 2
							if m.HP <= 0 {
								m.HP = 0
								m.Alive = false
							}
							break
						}
					}
				}
			}
			memberName := e.MemberDisplayName(hitIdx)
			if !e.IsAlive() {
				g.Logf("%s hits %s for %d -- party slain!", attacker, e.DisplayName(), dmg)
				g.AddKill()
				g.rollKillDrop(e)
				g.GainXP(20 + g.Floor*10)
			} else if killed {
				g.Logf("%s hits %s for %d -- slain!", attacker, memberName, dmg)
				g.AddKill()
				g.rollKillDrop(e)
				g.GainXP(10 + g.Floor*5)
			} else {
				g.Logf("%s hits %s for %d.", attacker, memberName, dmg)
			}
			if struckTwice {
				g.Logf("%s strikes %s again!", attacker, memberName)
			}
			if attackerMember.EffectChance > 0 && g.RNG.Float64() < attackerMember.EffectChance {
				effect := attackerMember.Effect
				if effect == "" {
					effect = "hex"
				}
				tgt := hitMember(e.Members, e.Active, hitIdx)
				applied, _ := applyEffect(tgt, effect, g.RNG, false)
				if applied {
					switch effect {
					case "hex":
						g.Logf("%s hexes %s (-1 DEF 10t)", attacker, memberName)
					case "rend":
						g.Logf("%s rends %s (bleed 6t)", attacker, memberName)
					case "entangle":
						g.Logf("%s entangles %s (root 4t)", attacker, memberName)
					case "spore":
						g.Logf("%s spores %s (poison 8t)", attacker, memberName)
					case "blind", "blindness":
						g.Logf("%s blinds %s (blind 20t)", attacker, memberName)
					case "haste":
						g.Logf("%s hastes %s (haste 50t)", attacker, memberName)
					case "slow":
						g.Logf("%s slows %s (slow 30t)", attacker, memberName)
					case "silence":
						g.Logf("%s silences %s (silence 6t)", attacker, memberName)
					case "stun":
						g.Logf("%s stuns %s (stun 1t)", attacker, memberName)
					case "poison":
						g.Logf("%s poisons %s (poison 6t)", attacker, memberName)
					default:
						g.Logf("%s tries to %s %s", attacker, effect, memberName)
					}
				} else {
					g.Logf("%s tries to %s %s", attacker, effect, memberName)
				}
			}
		g.EndPlayerTurn("")
		return ActionResult{Attacked: true}
		}
	}
	// Feature checks at target tile before normal move.
	if f := g.featureAt(next); f != nil {
		// Vault locked check — block entry if no rogue/wizard.
		if f.IsVault() && f.Locked && !g.canOpenLocked() {
			g.Logf("Locked vault - need rogue or wizard.")
			return ActionResult{}
		}
		// Pitfall trigger — one-way hidden/obvious, damage 2-4 if hidden & unaware.
		if f.IsPitfall() {
			wasHandled := g.handlePitfall(f)
			if wasHandled {
				if !g.Over {
					g.tickAfterMove()
				}
				return ActionResult{Moved: true, Descended: true}
			}
		}
	}
	// Move - silent (log reserved for combat/stairs/ambience)
	g.Party.Pos = next
	g.Party.Active = g.Party.Selected
	// Litter step ambience: 25% chance on passable ground litter (slate).
	if lit := lvl.LitterAt(next); lit != nil && lit.Category == "passable" {
		if g.RNG != nil && g.RNG.Float64() < 0.25 {
			if line := litterStepAmbience(lit.Kind, lvl.BiomeID); line != "" {
				g.Logf("%s", line)
			}
		}
	}
	// Post-move feature interactions: Vault loot, Den warning are auto.
	// Forge, Fountain, Merchant, Shrine are deliberate-use only (press g on tile to trigger).
	if f := g.featureAt(next); f != nil {
		if f.IsVault() {
			// Vault already passed locked check; claim treasure.
			// Copy before removal.
			vf := *f
			// Clear blocking check duplicate — handleVault does treasure/trap + removal.
			g.handleVault(&vf)
		} else if f.IsDen() {
			g.handleDen(f)
		} else if f.IsShrine() {
			g.Logf("Shrine here (press g)")
		} else if f.IsFountain() {
			g.Logf("A fountain bubbles here (press g to drink).")
		} else if f.IsMerchant() {
			g.Logf("A merchant beckons (press g to browse).")
		} else if f.IsForge() {
			costStr := "gold"
			costVal := f.Cost
			if f.CostType != "" {
				costStr = f.CostType
			}
			if costVal == 0 {
				if costStr == "food" {
					costVal = 50
				} else {
					costVal = 25
				}
			}
			g.Logf("A forge glows here (%d %s to use, press g).", costVal, costStr)
		}
	}
	// Check relic on final floor — claim but do not end run; escape to surface for bonus.
	if g.Floor == g.Tuning.Floors-1 && next == g.Relic && !g.RelicCollected {
		g.RelicCollected = true
		g.Won = false
		g.CurLevel().Set(g.Relic, TileFloor)
		g.Logf("You got the relic. Return to the surface.")
		// Reset transition tracking so old floors feel new again.
		g.VisitedFloors = make(map[int]bool)
		g.TransitionFiredForLevel = make(map[int]bool)
		// Keep current (final) floor marked visited so descending again doesn't re-fire? But we cleared; final stays visited.
		g.VisitedFloors[g.Floor] = true
		g.TransitionFiredForLevel[g.Floor] = true
		// Repopulate old levels with new enemies.
		if g.RNG != nil {
			for i := range g.Tuning.Floors - 1 {
				if g.Levels[i] != nil {
					g.Levels[i].RegenerateEnemies(g.RNG, i)
				}
			}
		}
		g.Logf("The temple stirs: old floors repopulate.")
		g.EndPlayerTurn("")
		return ActionResult{Moved: true}
	}
	g.EndPlayerTurn("")
	return ActionResult{Moved: true}
}

func dirName(d Dir) string {
	switch d {
	case DirN:
		return "north"
	case DirS:
		return "south"
	case DirW:
		return "west"
	case DirE:
		return "east"
	case DirNW:
		return "northwest"
	case DirNE:
		return "northeast"
	case DirSW:
		return "southwest"
	case DirSE:
		return "southeast"
	default:
		return "here"
	}
}


type stairSpec struct {
	delta      int
	src        func(lvl *Level) Pos
	dst        func(lvl *Level) Pos
	blockedMsg string
}

func (g *Game) EndPlayerTurn(msg string) {
	if msg != "" {
		g.Logf("%s", msg)
	}
	g.Turn++
	g.tickFood()
	g.tickRegen()
	// Tick member conditions and apply DoTs per member.
	if g.Party != nil {
		for _, m := range g.Party.Members {
			for _, id := range m.TickStatuses() {
				switch id {
				case StatusStrength:
					g.Logf("Strength fades.")
				case StatusInvisibility:
					g.Logf("Invisibility fades.")
				case StatusFireResist:
					g.Logf("Fire resistance fades.")
				case StatusLevitation:
					g.Logf("Levitation fades.")
				case StatusEnlightenment:
					g.Logf("Enlightenment fades.")
				case StatusParalysis:
					g.Logf("Paralysis wears off.")
				case StatusBlind:
					g.Logf("Blindness lifts.")
				case StatusHaste:
					g.Logf("Haste fades.")
				case StatusSlow:
					g.Logf("Slow fades.")
				case StatusRegenerate:
					g.Logf("Regeneration fades.")
				case StatusStun:
					g.Logf("Stun wears off.")
				case StatusConfusion:
					g.Logf("Confusion clears.")
				case StatusHex:
					g.Logf("Hex fades.")
				case StatusCurse:
					g.Logf("Curse lifts.")
				}
			}
		}
		for _, id := range g.Party.TickStatuses() {
			if id == StatusSummon {
				for i, m := range g.Party.Members {
					if len(m.Name) >= 8 && m.Name[:8] == "Summoned" {
						g.Party.Members = append(g.Party.Members[:i], g.Party.Members[i+1:]...)
						g.Logf("Summoned ally departs.")
						break
					}
				}
			}
		}
		enlightened := false
		for _, m := range g.Party.Members {
			if m.IsAlive() && m.HasStatus(StatusEnlightenment) {
				enlightened = true
				break
			}
		}
		if enlightened {
			if lvl := g.CurLevel(); lvl != nil {
				for y := range lvl.H {
					for x := range lvl.W {
						lvl.Seen[y][x] = true
					}
				}
			}
		}
		// DoT — data-driven via statuses.json dots (poison 1, bleed 2).
		bleedDot := statusDotDamage(StatusBleed)
		if bleedDot == 0 {
			bleedDot = 2
		}
		for _, m := range g.Party.Members {
			if m.IsAlive() && (m.HasStatus(StatusRend) || m.HasStatus(StatusBleed)) {
				m.HP -= bleedDot
				if m.HP <= 0 {
					m.HP = 0
					m.Alive = false
				}
				g.Logf("%s bleeds for %d damage!", m.Name, bleedDot)
			}
		}
		if g.Party.LivingCount() == 0 {
			g.Over = true
			if g.Cause == "" {
				g.Cause = "Bleed"
			}
			g.Logf("You have bled out. Seed %d.", g.Seed)
			g.RecordScore()
		}
		poisonDot := statusDotDamage(StatusPoison)
		if poisonDot == 0 {
			poisonDot = 1
		}
		for _, m := range g.Party.Members {
			if m.IsAlive() && (m.HasStatus(StatusSpore) || m.HasStatus(StatusPoison)) {
				m.HP -= poisonDot
				if m.HP <= 0 {
					m.HP = 0
					m.Alive = false
				}
				g.Logf("%s suffers %d poison damage!", m.Name, poisonDot)
			}
		}
		if g.Party.LivingCount() == 0 {
			g.Over = true
			if g.Cause == "" {
				g.Cause = "Poison"
			}
			g.Logf("You have succumbed to poison. Seed %d.", g.Seed)
			g.RecordScore()
		}
		// Regenerate — duration data-driven via statuses.json (20t); each regenerating member heals 1 HP every 2 ticks.
		if g.Turn%2 == 0 {
			healed := 0
			for _, m := range g.Party.Members {
				if m.IsAlive() && m.HP < m.MaxHP && m.HasStatus(StatusRegenerate) {
					m.HP++
					healed++
				}
			}
			if healed > 0 {
				g.Logf("Regeneration restores 1 HP to %d members.", healed)
			}
		}
	}
	if lvl := g.CurLevel(); lvl != nil {
		for _, e := range lvl.Enemies {
			if e != nil && e.IsAlive() {
				e.TickStatuses() // party-wide timers only (summon)
				for _, m := range e.Members {
					m.TickStatuses()
				}
			}
		}
	}

	// Cleric healers_grace HoT: +0.5 HP/tick -> +1 every 2 ticks (bard-buffed 0.55 via extra 10% chance on odd)
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
	// Refrain on wait: waiter with refrain heals 1 to every other living member
	if g.WaitRefrainActive {
		// active is waiter index; heal non-active others
		activeIdx := g.Party.Active
		for i, m := range g.Party.Members {
			if i != activeIdx && m.IsAlive() && m.HP < m.MaxHP {
				m.HP++
				if m.HP > m.MaxHP {
					m.HP = m.MaxHP
				}
			}
		}
		g.WaitRefrainActive = false
	}
	// Elf identify ticker: every 250 ticks -50 per extra elf when >=2
	if iv := ElfIdentifyInterval(g.Party); iv > 0 {
		if g.NextElfIdentifyTurn == 0 {
			g.NextElfIdentifyTurn = g.Turn + iv
		}
		if g.Turn >= g.NextElfIdentifyTurn {
			// find an unidentified held appearance
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
	} else {
		g.NextElfIdentifyTurn = 0
	}
	// Lorekeeper / attuned ticker: 1 held appearance / 50 turns (45 if bard)
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
	} else {
		g.NextLorekeeperTurn = 0
	}
	g.applyStarvation()
	g.MaybeTickAmbience()
	g.EnemyTurn()
	g.UpdateFOV()
}

func (g *Game) EnemyTurn() {
	if g.Over {
		return
	}
	g.TickDens()
	lvl := g.CurLevel()
	for _, e := range lvl.Enemies {
		if !e.IsAlive() {
			continue
		}
		// Regen tick for troll and similar
		e.RegenTick()
		e.EnsureActive()
		// Enemy DoTs tick per member — data-driven via statuses.json (bleed 2, poison 1).
		dot := statusDotDamage(StatusBleed)
		if dot == 0 {
			dot = 2
		}
		for _, m := range e.Members {
			if m.IsAlive() && (m.HasStatus(StatusRend) || m.HasStatus(StatusBleed)) {
				m.HP -= dot
				if m.HP <= 0 {
					m.HP = 0
					m.Alive = false
				}
			}
		}
		if !e.IsAlive() {
			g.Logf("%s bleeds out!", e.DisplayName())
			g.AddKill()
			g.rollKillDrop(e)
			continue
		}
		pdot := statusDotDamage(StatusPoison)
		if pdot == 0 {
			pdot = 1
		}
		for _, m := range e.Members {
			if m.IsAlive() && (m.HasStatus(StatusSpore) || m.HasStatus(StatusPoison)) {
				m.HP -= pdot
				if m.HP <= 0 {
					m.HP = 0
					m.Alive = false
				}
			}
		}
		if !e.IsAlive() {
			g.Logf("%s succumbs to poison!", e.DisplayName())
			g.AddKill()
			g.rollKillDrop(e)
			continue
		}
		// Status skip: the acting member's paralysis/entangle/sleep/stun prevent action.
		act := enemyActor(e)
		if act != nil && (act.HasStatus(StatusParalysis) || act.HasStatus(StatusEntangle) || act.HasStatus(StatusSleep) || act.HasStatus(StatusStun)) {
			continue
		}
		// Invisibility hides individuals: untargetable only when every living member is unseen.
		if partyUnseen(g.Party) {
			continue
		}
		dx := g.Party.Pos.X - e.Pos.X
		dy := g.Party.Pos.Y - e.Pos.Y
		cheb := max(abs(dx), abs(dy))
		if cheb == 1 {
			atk := e.Members[e.Active]
			bonus := 0
			if act != nil && act.HasStatus(StatusStrength) {
				bonus = 2
			}
			raw := RollRaw(g.RNG, atk.ATK[0]+bonus, atk.ATK[1]+bonus)
			// Slow: flat quarter penalty on the rolled damage (1 stays 1).
			if atk.HasStatus(StatusSlow) {
				raw -= raw / 4
			}
			isMagic := atk.DamageType == "magic"
			hitIdx, actual := g.Party.ApplyDamageWithType(g.RNG, raw, isMagic)
			defender := "you"
			if hitIdx >= 0 && hitIdx < len(g.Party.Members) {
				defender = g.Party.Members[hitIdx].Name
			}
			attackerName := e.MemberDisplayName(e.Active)
			g.Logf("%s hits %s for %d.", attackerName, defender, actual)
			// of_thorns / of_martyr thorns return 1 on hit (attacker takes 1)
			if hitIdx >= 0 && hitIdx < len(g.Party.Members) && actual > 0 {
				defMem := g.Party.Members[hitIdx]
				if defMem.HasAffix("of_thorns") || (defMem.HasAffix("of_martyr") && defMem.Class == "paladin") {
					if e.Members[e.Active].IsAlive() {
						e.Members[e.Active].HP -= 1
						if e.Members[e.Active].HP <= 0 {
							e.Members[e.Active].HP = 0
							e.Members[e.Active].Alive = false
						}
						g.Logf("Thorns return 1 damage to %s!", attackerName)
						if !e.IsAlive() {
							g.Logf("%s collapses from thorns!", e.DisplayName())
							g.AddKill()
							g.rollKillDrop(e)
							continue
						}
					}
				}
			}
			// Haste: 50% second strike on the same defender. Riders fire once.
			if hitIdx >= 0 && hitIdx < len(g.Party.Members) {
				tgt := g.Party.Members[hitIdx]
				if tgt.IsAlive() && atk.IsAlive() && atk.HasStatus(StatusHaste) && g.RNG.Float64() < 0.5 {
					raw2 := RollRaw(g.RNG, atk.ATK[0]+bonus, atk.ATK[1]+bonus)
					if atk.HasStatus(StatusSlow) {
						raw2 -= raw2 / 4
					}
					def2 := tgt.DEF
					if isMagic {
						def2 = tgt.MDEF
					}
					def2 += tgt.effectiveDEFDelta()
					actual2 := raw2 - def2
					if actual2 < 1 {
						actual2 = 1
					}
					if tgt.HasStatus(StatusFireResist) && isMagic {
						actual2 = resistedFire(actual2)
					}
					tgt.HP -= actual2
					if tgt.HP <= 0 {
						tgt.HP = 0
						tgt.Alive = false
					}
					g.Logf("%s strikes %s again for %d!", attackerName, defender, actual2)
				}
			}
			if atk.EffectChance > 0 {
				if g.RNG.Float64() < atk.EffectChance {
					effect := atk.Effect
					if effect == "" {
						effect = "hex"
					}
					isMagicEff := atk.DamageType == "magic"
					tgt := hitMember(g.Party.Members, g.Party.Active, hitIdx)
					applied, resisted := applyEffect(tgt, effect, g.RNG, isMagicEff)
					if resisted {
						g.Logf("%s resists %s!", defender, effect)
					} else if applied {
						switch effect {
						case "hex":
							g.Logf("%s hexes %s (-1 DEF 10t)", attackerName, defender)
						case "rend":
							g.Logf("%s rends %s (bleed 2/turn 6t)", attackerName, defender)
						case "entangle":
							g.Logf("%s entangles %s (root 4t)", attackerName, defender)
						case "spore":
							g.Logf("%s spores %s (poison 1/turn 8t)", attackerName, defender)
						case "blind", "blindness":
							g.Logf("%s blinds %s (blind 20t)", attackerName, defender)
						case "haste":
							g.Logf("%s hastes %s (haste 50t)", attackerName, defender)
						case "slow":
							g.Logf("%s slows %s (slow 30t)", attackerName, defender)
						case "silence":
							g.Logf("%s silences %s (silence 6t)", attackerName, defender)
						case "stun":
							g.Logf("%s stuns %s (stun 1t)", attackerName, defender)
						case "poison":
							g.Logf("%s poisons %s (poison 6t)", attackerName, defender)
						default:
							g.Logf("%s tries to %s %s", attackerName, effect, defender)
						}
					} else {
						g.Logf("%s tries to %s %s", attackerName, effect, defender)
					}
				}
			}
			if g.Party.LivingCount() == 0 {
				g.Over = true
				if g.Cause == "" {
					if attackerName != "" {
						g.Cause = "Slain by " + attackerName
					} else {
						g.Cause = "Slain by enemy"
					}
				}
				g.Logf("You have fallen. Seed %d. Score %d.", g.Seed, g.CalculateScore())
				g.RecordScore()
				return
			}
			continue
		}
		// Confusion: random movement instead of BFS.
		if act != nil && act.HasStatus(StatusConfusion) {
			dirs := []Dir{DirN, DirS, DirW, DirE}
			d := dirs[g.RNG.IntN(len(dirs))]
			nxt := e.Pos.Add(d)
			if lvl.Walkable(nxt) && nxt != g.Party.Pos {
				coll := false
				for _, o := range lvl.Enemies {
					if o != e && o.IsAlive() && o.Pos == nxt {
						coll = true
						break
					}
				}
				if !coll {
					e.Pos = nxt
				}
			}
			continue
		}
		// Door opening: if adjacent closed door and within 3 of player, open it
		if cheb <= 8 {
			for _, d := range []Dir{DirN, DirS, DirW, DirE} {
				np := e.Pos.Add(d)
				if lvl.IsDoor(np) && lvl.IsDoorClosed(np) {
					// vault lock enemy-side deferred — enemies can open non-locked doors; locked vault doors remain blocked (game/features.go:195 Locked)
					distToPlayer := max(abs(g.Party.Pos.X-np.X), abs(g.Party.Pos.Y-np.Y))
					if distToPlayer <= 3 || cheb <= 3 {
						// vault lock check — if locked, defer (still block)
						locked := false
						for _, f := range lvl.Features {
							if f.IsVault() && f.Locked {
								ddx := f.Pos.X - np.X
								if ddx < 0 {
									ddx = -ddx
								}
								ddy := f.Pos.Y - np.Y
								if ddy < 0 {
									ddy = -ddy
								}
								if ddx <= 4 && ddy <= 4 {
									locked = true
									break
								}
							}
						}
						if !locked {
							lvl.SetDoorOpen(np, true)
						}
					}
				}
			}
		}
		// Move toward if within 8 using BFS cardinal
		if cheb <= 8 {
			parent := make(map[Pos]Pos)
			visited := make(map[Pos]bool)
			queue := []Pos{e.Pos}
			visited[e.Pos] = true
			found := false
			for len(queue) > 0 && !found {
				cur := queue[0]
				queue = queue[1:]
				if cur == g.Party.Pos {
					found = true
					break
				}
				for _, d := range []Dir{DirN, DirS, DirW, DirE} {
					np := cur.Add(d)
					if !lvl.InBounds(np) || visited[np] {
						continue
					}
					if !lvl.Walkable(np) && np != g.Party.Pos {
						continue
					}
					// Avoid other enemies
					blocked := false
					for _, o := range lvl.Enemies {
						if o != e && o.IsAlive() && o.Pos == np && np != g.Party.Pos {
							blocked = true
							break
						}
					}
					if blocked {
						continue
					}
					visited[np] = true
					parent[np] = cur
					queue = append(queue, np)
				}
			}
			if found {
				// Reconstruct step from e.Pos toward player
				stepPos := g.Party.Pos
				for {
					p, ok := parent[stepPos]
					if !ok {
						break
					}
					if p == e.Pos {
						// stepPos is next tile
						if lvl.Walkable(stepPos) && stepPos != g.Party.Pos {
							coll := false
							for _, o := range lvl.Enemies {
								if o != e && o.IsAlive() && o.Pos == stepPos {
									coll = true
									break
								}
							}
							if !coll {
								e.Pos = stepPos
							}
						}
						break
					}
					stepPos = p
				}
			} else {
				// No path, try direct cardinal step as fallback
				best := e.Pos
				bestDist := cheb
				for _, d := range []Dir{DirN, DirS, DirW, DirE} {
					np := e.Pos.Add(d)
					if !lvl.Walkable(np) || np == g.Party.Pos {
						continue
					}
					coll := false
					for _, o := range lvl.Enemies {
						if o != e && o.IsAlive() && o.Pos == np {
							coll = true
							break
						}
					}
					if coll {
						continue
					}
					ndx := g.Party.Pos.X - np.X
					if ndx < 0 {
						ndx = -ndx
					}
					ndy := g.Party.Pos.Y - np.Y
					if ndy < 0 {
						ndy = -ndy
					}
					ndist := max(ndx, ndy)
					if ndist < bestDist {
						bestDist = ndist
						best = np
					}
				}
				if best != e.Pos {
					e.Pos = best
				}
			}
		} else {
			// Wander cardinal — ghost_step skips wander roll
			if g.Party.HasTalent("ghost_step") {
				// no wander
			} else {
				dirs := []Dir{DirN, DirS, DirW, DirE}
				d := dirs[g.RNG.IntN(len(dirs))]
				nxt := e.Pos.Add(d)
				if lvl.Walkable(nxt) && nxt != g.Party.Pos {
					coll := false
					for _, o := range lvl.Enemies {
						if o != e && o.IsAlive() && o.Pos == nxt {
							coll = true
							break
						}
					}
					if !coll {
						e.Pos = nxt
					}
				}
			}
		}
	}
}

func sign(a int) int {
	if a < 0 {
		return -1
	}
	if a > 0 {
		return 1
	}
	return 0
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

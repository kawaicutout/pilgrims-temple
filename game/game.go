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
	Merchant                MerchantState `json:"merchant"`
	Shrine                  ShrineState   `json:"shrine"`
}

func NewGame(seed int64, tuning Tuning) *Game {
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
	g.Party = GenerateParty(rng, 1)
	ApplyRaceBuffs(g.Party)
	// Init elf identify ticker: 250 -50 per extra elf when >=2
	if iv := ElfIdentifyInterval(g.Party); iv > 0 {
		g.NextElfIdentifyTurn = g.Turn + iv
	}
	start := g.Levels[0].StairsUp
	g.Party.Pos = start
	g.Floor = 0
	g.VisitedFloors[0] = true
	g.TransitionFiredForLevel[0] = true
	g.Logf("Seed %d -- Pilgrim's Temple, %d floors.", seed, tuning.Floors)
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
	return 100 + 50*(g.Level-1)
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
		m := g.Party.Members[pick.MemberIdx]
		affixID := pick.Options[0]
		m.Affixes = append(m.Affixes, affixID)
		g.Logf("%s gains affix %s.", m.Name, FriendlyID(affixID))
		switch affixID {
		case "veteran":
			m.ATK[0]++
			m.ATK[1]++
		case "hardy":
			m.MaxHP += 3
			m.HP += 3
		case "keen":
			m.ATK[0]++
		case "stout":
			m.DEF++
		case "bright":
			m.Light++
		case "burdened":
			if m.Carry == 0 {
				m.Carry = 5
			}
			m.Carry += 3
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
	if g.Party != nil && g.Party.HasStatus(StatusEnlightenment) {
		// Enlightenment reveals entire floor.
		for y := range lvl.H {
			for x := range lvl.W {
				lvl.Seen[y][x] = true
				lvl.Visible[y][x] = true
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
	living := EffectiveLivingCount(g.Party)
	if living == 0 {
		// fallback to actual living if effective zero (should not happen)
		living = g.Party.LivingCount()
		if living == 0 {
			g.Food = int(g.FoodFloat)
			if g.FoodFloat < 0 {
				g.FoodFloat = 0
				g.Food = 0
			}
			return
		}
	}
	if g.Party.LivingCount() == 0 {
		g.Food = int(g.FoodFloat)
		if g.FoodFloat < 0 {
			g.FoodFloat = 0
			g.Food = 0
		}
		return
	}
	cost := float64(living)*float64(g.Tuning.Food.PerMemberPerTurn) - g.frugalBonus()
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
		g.Logf("You have succumbed to starvation. Seed %d.", g.Seed)
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

// handleVault checks locked status via Party.HasRogue (and wizard). Returns true if blocked.
func (g *Game) handleVault(f *Feature) bool {
	if f == nil || !f.IsVault() {
		return false
	}
	if f.Locked && !g.Party.HasRogue() && !g.Party.HasWizard() && !g.Wizard {
		g.Logf("Locked vault - need rogue.")
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
		_, actual := g.Party.ApplyDamage(g.RNG, dmg)
		g.Logf("Vault treasure +%d gold! Trap springs for %d damage!", treasure, actual)
		if g.Party.LivingCount() == 0 {
			g.Over = true
			g.Logf("You have fallen. Seed %d. Score %d.", g.Seed, g.CalculateScore())
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
		g.Food += 50
		g.FoodFloat += 50
		g.Logf("Merchant sells %s for %dg (+50 food).", w.Name, w.Price)
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
		g.Party.EnsureSelection()
		g.Logf("Shrine recruits %s the %s! (+)", m.Name, m.Class)
		g.removeFeatureAt(g.Shrine.Pos, FeatureShrine)
		g.Shrine = ShrineState{}
		return true
	case 1: // Resurrect dead member (most recent)
		deadIdx := -1
		for i := len(g.Party.Members) - 1; i >= 0; i-- {
			if !g.Party.Members[i].IsAlive() {
				deadIdx = i
				break
			}
		}
		if deadIdx == -1 {
			g.Logf("Shrine: no fallen pilgrims to resurrect.")
			return false
		}
		m := g.Party.Members[deadIdx]
		m.Alive = true
		m.HP = m.MaxHP
		g.Party.EnsureSelection()
		g.Logf("Shrine resurrects %s for free! (+)", m.Name)
		g.removeFeatureAt(g.Shrine.Pos, FeatureShrine)
		g.Shrine = ShrineState{}
		return true
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
	if g.Party.HasStatus(StatusCurse) {
		g.Party.RemoveStatus(StatusCurse)
		g.Logf("Shrine cleanses your curse.")
	}
	// Try resurrection if any dead member, else recruitment if space.
	hasDead := false
	deadIdx := -1
	for i, m := range g.Party.Members {
		if !m.IsAlive() {
			hasDead = true
			deadIdx = i
			break
		}
	}
	canRecruit := len(g.Party.Members) < 4
	// Load costs (first shrine use defines cost; fallback 75 gold/50 food)
	costGold, costFood := 75, 50
	if uses := GetShrineUses(); len(uses) > 0 {
		for _, u := range uses {
			if u.ID == "resurrect" {
				if u.GoldCost > 0 {
					costGold = u.GoldCost
				}
				if u.FoodCost > 0 {
					costFood = u.FoodCost
				}
				break
			}
		}
	}
	if hasDead {
		// Try to pay gold first, else food, else fail.
		if g.Gold >= costGold {
			g.Gold -= costGold
			m := g.Party.Members[deadIdx]
			m.Alive = true
			m.HP = m.MaxHP
			g.Party.EnsureSelection()
			g.Logf("Shrine resurrects %s for %d gold! (+)", m.Name, costGold)
			g.removeFeatureAt(f.Pos, FeatureShrine)
			return
		}
		if g.Food >= costFood {
			g.Food -= costFood
			g.FoodFloat -= float64(costFood)
			m := g.Party.Members[deadIdx]
			m.Alive = true
			m.HP = m.MaxHP
			g.Party.EnsureSelection()
			g.Logf("Shrine resurrects %s for %d food! (+)", m.Name, costFood)
			g.removeFeatureAt(f.Pos, FeatureShrine)
			return
		}
		g.Logf("Shrine resurrection needs %d gold or %d food (you have %d gold, %d food).", costGold, costFood, g.Gold, g.Food)
		return
	}
	if canRecruit {
		// Recruit costs nothing for now (shrine recruit free).
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
			g.Logf("You have fallen. Seed %d. Score %d.", g.Seed, g.CalculateScore())
		}
	} else {
		g.Logf("Fountain %s: %s", o.Name, o.Desc)
	}
	if o.Effect == "bless" {
		g.Party.ApplyStatus(StatusBless, 101)
		g.Logf("Blessed waters grant +1 DEF for 100 turns.")
	} else if o.Effect == "curse" {
		g.Party.ApplyStatus(StatusCurse, 201)
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
		g.Food += 50
		g.FoodFloat += 50
		g.Logf("Merchant sells %s for %dg (+50 food).", w.Name, w.Price)
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
func (g *Game) handlePitfall(f *Feature) bool {
	if f == nil || !f.IsPitfall() {
		return false
	}
	if g.Party.HasStatus(StatusLevitation) {
		g.Logf("You float over the pitfall.")
		return false
	}
	// Detection: rogue/wizard or wizard mode reveal.
	aware := !f.Hidden || g.Party.HasRogue() || g.Party.HasWizard() || g.Wizard
	if f.Hidden && !aware {
		dmg := f.Damage
		if dmg == 0 {
			dmg = 2 + g.RNG.IntN(3)
		}
		_, actual := g.Party.ApplyDamage(g.RNG, dmg)
		g.Logf("Hidden pitfall! You fall -- %d damage!", actual)
		if g.Party.LivingCount() == 0 {
			g.Over = true
			g.Logf("You have fallen. Seed %d. Score %d.", g.Seed, g.CalculateScore())
			return true
		}
		// One-way fall to next level if possible.
		if g.Floor+1 < g.Tuning.Floors {
			g.Floor++
			g.Party.Pos = g.CurLevel().StairsUp
			g.Logf("Pitfall drops you to floor %d (one-way).", g.Floor+1)
			g.UpdateFOV()
		} else {
			g.Logf("Pitfall has no lower level -- you climb back out.")
		}
		g.removeFeatureAt(f.Pos, FeaturePitfall)
		return true
	}
	// Obvious or detected pitfall: still one-way trigger but no surprise damage.
	if f.Hidden && aware {
		g.Logf("You spot a hidden pitfall and step around its edge... but the floor gives way!")
	}
	if !f.Hidden {
		g.Logf("Pitfall ahead -- one-way drop to the next level.")
	}
	// Trigger drop without damage (or minimal).
	if g.Floor+1 < g.Tuning.Floors {
		// Move onto pitfall tile first for position consistency, then drop.
		g.Party.Pos = f.Pos
		g.Floor++
		g.Party.Pos = g.CurLevel().StairsUp
		g.Logf("You drop through the pitfall to floor %d (one-way).", g.Floor+1)
		g.UpdateFOV()
	} else {
		g.Logf("No lower level beneath the pitfall.")
	}
	g.removeFeatureAt(f.Pos, FeaturePitfall)
	return true
}

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
	if g.Party.HasStatus(StatusParalysis) {
		g.Logf("You are paralyzed and cannot move!")
		g.EndPlayerTurn("")
		return ActionResult{}
	}
	if g.Party.HasStatus(StatusEntangle) || g.Party.HasStatus(StatusSleep) {
		g.Logf("You are rooted and cannot move!")
		g.EndPlayerTurn("")
		return ActionResult{}
	}
	if g.Party.HasStatus(StatusConfusion) {
		dirs := []Dir{DirN, DirS, DirW, DirE}
		dir = dirs[g.RNG.IntN(len(dirs))]
		g.Logf("You stumble %s in confusion.", dirName(dir))
	}
	next := g.Party.Pos.Add(dir)
	if lvl.IsDoor(next) && lvl.IsDoorClosed(next) {
		// Check vault lock near door
		locked := false
		for _, f := range lvl.Features {
			if f.IsVault() && f.Locked {
				dx := f.Pos.X - next.X
				if dx < 0 {
					dx = -dx
				}
				dy := f.Pos.Y - next.Y
				if dy < 0 {
					dy = -dy
				}
				if dx <= 4 && dy <= 4 {
					locked = true
					break
				}
			}
		}
		if locked && !g.Party.HasRogue() && !g.Party.HasWizard() && !g.Wizard {
			g.Logf("The vault door is locked -- need rogue.")
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
					if g.Party.HasStatus(StatusStrength) {
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
			dmg, hitIdx, killed := PlayerBumpEnemy(g.RNG, g.Party, e)
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
				g.GainXP(20 + g.Floor*10)
				g.Logf("Score %d (Kills %d).", g.CalculateScore(), g.Kills)
			} else if killed {
				g.Logf("%s hits %s for %d -- slain!", attacker, memberName, dmg)
				g.AddKill()
				g.GainXP(10 + g.Floor*5)
				g.Logf("Score %d (Kills %d).", g.CalculateScore(), g.Kills)
			} else {
				g.Logf("%s hits %s for %d.", attacker, memberName, dmg)
			}
			if attackerMember.EffectChance > 0 && g.RNG.Float64() < attackerMember.EffectChance {
				effect := attackerMember.Effect
				if effect == "" {
					effect = "hex"
				}
				// Apply status to enemy party based on effect.
				switch effect {
				case "hex":
					e.ApplyStatus(StatusHex, 10)
					g.Logf("%s hexes %s (-1 DEF 10t)", attacker, memberName)
				case "rend":
					e.ApplyStatus(StatusRend, 6)
					e.ApplyStatus(StatusBleed, 6)
					g.Logf("%s rends %s (bleed 6t)", attacker, memberName)
				case "entangle":
					e.ApplyStatus(StatusEntangle, 4)
					g.Logf("%s entangles %s (root 4t)", attacker, memberName)
				case "spore":
					e.ApplyStatus(StatusSpore, 8)
					g.Logf("%s spores %s (poison 8t)", attacker, memberName)
				default:
					g.Logf("%s tries to %s %s", attacker, effect, memberName)
				}
			}
			if g.LevelUpPending != nil {
				g.Turn++
				g.tickFood()
				if g.Turn%10 == 0 {
					for _, m := range g.Party.Members {
						if m.IsAlive() && m.HP < m.MaxHP {
							m.HP++
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
				g.applyStarvation()
				g.UpdateFOV()
				return ActionResult{Attacked: true}
			}
			g.EndPlayerTurn("")
			return ActionResult{Attacked: true}
		}
	}
	// Feature checks at target tile before normal move.
	if f := g.featureAt(next); f != nil {
		// Vault locked check — block entry if no rogue/wizard.
		if f.IsVault() && f.Locked && !g.Party.HasRogue() && !g.Party.HasWizard() && !g.Wizard {
			g.Logf("Locked vault - need rogue.")
			return ActionResult{}
		}
		// Pitfall trigger — one-way hidden/obvious, damage 2-4 if hidden & unaware.
		if f.IsPitfall() {
			wasHandled := g.handlePitfall(f)
			if wasHandled {
				// Pitfall already moved floor / applied damage; consume turn.
				// If still alive and not game over, advance turn like EndPlayerTurn but without double FOV?
				if !g.Over {
					g.Turn++
					g.tickFood()
					if g.Turn%10 == 0 {
						for _, m := range g.Party.Members {
							if m.IsAlive() && m.HP < m.MaxHP {
								m.HP++
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
					g.applyStarvation()
					g.UpdateFOV()
					if !g.Over {
						g.EnemyTurn()
						g.UpdateFOV()
					}
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
	// Check relic on final floor
	if g.Floor == g.Tuning.Floors-1 && next == g.Relic {
		g.Over = true
		g.Won = true
		g.RelicCollected = true
		g.Logf("You claim the relic! Victory - seed %d. Score %d.", g.Seed, g.CalculateScore())
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
		g.Logf("The temple stirs: old floors repopulate. Final Score %d.", g.CalculateScore())
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

func (g *Game) TryStairsDown() {
	if g.Over {
		return
	}
	lvl := g.CurLevel()
	if g.Party.Pos != lvl.StairsDown {
		g.Logf("No stairs down here.")
		return
	}
	if g.Floor+1 >= g.Tuning.Floors {
		g.Logf("The way down is sealed.")
		return
	}
	g.Floor++
	g.Party.Pos = g.CurLevel().StairsUp
	g.Logf("You descend to floor %d.", g.Floor+1)
	// On-transition talents fire exactly once per level per run.
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
		// Ensure visited marked even if transition already fired (for tracking).
		g.VisitedFloors[g.Floor] = true
		g.logBiomeEntry()
	}
	g.UpdateFOV()
	g.EnemyTurn() // enemies act after descent?
}

func (g *Game) TryStairsUp() {
	if g.Over {
		return
	}
	lvl := g.CurLevel()
	if g.Party.Pos != lvl.StairsUp {
		g.Logf("No stairs up here.")
		return
	}
	if g.Floor == 0 {
		g.Logf("You are at the entrance.")
		return
	}
	g.Floor--
	g.Party.Pos = g.CurLevel().StairsDown
	g.Logf("You ascend to floor %d.", g.Floor+1)
	// Same visited/transition gating for upward travel (covers post-relic repopulation).
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

func (g *Game) ApplyFloorTransition() {
	// Biome entry feel: log evocative line on floor entry.
	g.logBiomeEntry()
	// On-transition talents: fire once per floor per run, reset on relic.
	// Forage (druid): +100 food per bearer. Restoration (cleric): full heal all living members.
	for _, m := range g.Party.Members {
		if !m.IsAlive() {
			continue
		}
		if m.HasTalent("forage") {
			g.Food += 100
			g.FoodFloat += 100
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
			if healed > 0 {
				g.Logf("%s restores the party to full health.", m.Name)
			} else {
				g.Logf("%s channels restoration (party already healthy).", m.Name)
			}
			// Only one restoration proc per party per transition (avoid duplicate full-heal spam if multiple clerics).
			break
		}
	}
}
func (g *Game) EndPlayerTurn(msg string) {
	if msg != "" {
		g.Logf("%s", msg)
	}
	g.Turn++
	g.tickFood()
	// Troll regen every 3 ticks
	if g.Turn%3 == 0 {
		healed := false
		for _, m := range g.Party.Members {
			if m.IsAlive() && normalizeRaceID(m.Race) == "troll" && m.HP < m.MaxHP {
				m.HP++
				if m.HP > m.MaxHP {
					m.HP = m.MaxHP
				}
				healed = true
			}
		}
		if healed {
			// optional log? keep silent or minimal
		}
	}
	// Tick party statuses and apply DoTs.
	if g.Party != nil {
		expired := g.Party.TickStatuses()
		for _, id := range expired {
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
			case StatusSummon:
				for i, m := range g.Party.Members {
					if len(m.Name) >= 8 && m.Name[:8] == "Summoned" {
						g.Party.Members = append(g.Party.Members[:i], g.Party.Members[i+1:]...)
						g.Logf("Summoned ally departs.")
						break
					}
				}
			}
		}
		if g.Party.HasStatus(StatusEnlightenment) {
			if lvl := g.CurLevel(); lvl != nil {
				for y := range lvl.H {
					for x := range lvl.W {
						lvl.Seen[y][x] = true
					}
				}
			}
		}
		if g.Party.HasStatus(StatusRend) || g.Party.HasStatus(StatusBleed) {
			_, actual := g.Party.ApplyDamage(g.RNG, 2)
			g.Logf("Bleed deals %d damage!", actual)
			if g.Party.LivingCount() == 0 {
				g.Over = true
				g.Logf("You have bled out. Seed %d.", g.Seed)
			}
		}
		if g.Party.HasStatus(StatusSpore) || g.Party.HasStatus(StatusPoison) {
			_, actual := g.Party.ApplyDamage(g.RNG, 1)
			g.Logf("Poison deals %d damage!", actual)
			if g.Party.LivingCount() == 0 {
				g.Over = true
				g.Logf("You have succumbed to poison. Seed %d.", g.Seed)
			}
		}
	}
	if lvl := g.CurLevel(); lvl != nil {
		for _, e := range lvl.Enemies {
			if e != nil && e.IsAlive() {
				_ = e.TickStatuses()
			}
		}
	}
	// Natural regen: 1 HP every 10 ticks per living member
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
	// Endurance talent: +1 HP per 5 ticks per bearer (tuned from 1/tick = ~0.2/tick)
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
		// Enemy DoTs from bleed/rend/spore
		if e.HasStatus(StatusRend) || e.HasStatus(StatusBleed) {
			for _, m := range e.Members {
				if m.IsAlive() {
					m.HP -= 2
					if m.HP <= 0 {
						m.HP = 0
						m.Alive = false
					}
				}
			}
			if !e.IsAlive() {
				g.Logf("%s bleeds out!", e.DisplayName())
				g.AddKill()
				g.Logf("Score %d (Kills %d).", g.CalculateScore(), g.Kills)
				continue
			}
		}
		if e.HasStatus(StatusSpore) || e.HasStatus(StatusPoison) {
			for _, m := range e.Members {
				if m.IsAlive() {
					m.HP -= 1
					if m.HP <= 0 {
						m.HP = 0
						m.Alive = false
					}
				}
			}
			if !e.IsAlive() {
				g.Logf("%s succumbs to poison!", e.DisplayName())
				g.AddKill()
				g.Logf("Score %d (Kills %d).", g.CalculateScore(), g.Kills)
				continue
			}
		}
		// Status skip: paralysis/entangle/sleep prevent action.
		if e.HasStatus(StatusParalysis) || e.HasStatus(StatusEntangle) || e.HasStatus(StatusSleep) {
			continue
		}
		// Invisibility prevents enemy targeting entirely.
		if g.Party.HasStatus(StatusInvisibility) {
			continue
		}
		dx := g.Party.Pos.X - e.Pos.X
		dy := g.Party.Pos.Y - e.Pos.Y
		cheb := max(abs(dx), abs(dy))
		if cheb == 1 {
			atk := e.Members[e.Active]
			bonus := 0
			if e.HasStatus(StatusStrength) {
				bonus = 2
			}
			raw := RollRaw(g.RNG, atk.ATK[0]+bonus, atk.ATK[1]+bonus)
			isMagic := atk.DamageType == "magic"
			hitIdx, actual := g.Party.ApplyDamageWithType(g.RNG, raw, isMagic)
			defender := "you"
			if hitIdx >= 0 && hitIdx < len(g.Party.Members) {
				defender = g.Party.Members[hitIdx].Name
			}
			attackerName := e.MemberDisplayName(e.Active)
			g.Logf("%s hits %s for %d.", attackerName, defender, actual)
			if atk.EffectChance > 0 {
				if g.RNG.Float64() < atk.EffectChance {
					effect := atk.Effect
					if effect == "" {
						effect = "hex"
					}
					isMagicEff := atk.DamageType == "magic"
					if !g.Party.resistsStatus(isMagicEff, g.RNG) {
						switch effect {
						case "hex":
							g.Party.ApplyStatus(StatusHex, 10)
							g.Logf("%s hexes %s (-1 DEF 10t)", attackerName, defender)
						case "rend":
							g.Party.ApplyStatus(StatusRend, 6)
							g.Party.ApplyStatus(StatusBleed, 6)
							g.Logf("%s rends %s (bleed 2/turn 6t)", attackerName, defender)
						case "entangle":
							g.Party.ApplyStatus(StatusEntangle, 4)
							g.Logf("%s entangles %s (root 4t)", attackerName, defender)
						case "spore":
							g.Party.ApplyStatus(StatusSpore, 8)
							g.Logf("%s spores %s (poison 1/turn 8t)", attackerName, defender)
						case "regenerate":
							// no status, regen already via flag
							g.Logf("%s tries to %s %s", attackerName, effect, defender)
						default:
							g.Logf("%s tries to %s %s", attackerName, effect, defender)
						}
					} else {
						g.Logf("%s resists %s!", defender, effect)
					}
				}
			}
			if g.Party.LivingCount() == 0 {
				g.Over = true
				g.Logf("You have fallen. Seed %d. Score %d.", g.Seed, g.CalculateScore())
				return
			}
			continue
		}
		// Confusion: random movement instead of BFS.
		if e.HasStatus(StatusConfusion) {
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
					// Check vault lock near door - enemies can open non-locked doors or locked if they have rogue? Simplify: enemies can open any door if within 3 of player
					distToPlayer := max(abs(g.Party.Pos.X-np.X), abs(g.Party.Pos.Y-np.Y))
					if distToPlayer <= 3 || cheb <= 3 {
						// Check vault lock - if locked, still block unless enemy has rogue? For now allow
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
			// Wander cardinal
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

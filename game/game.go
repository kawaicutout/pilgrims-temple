package game

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"unicode"
)

// Game holds run state.
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
	Relic                   Pos           `json:"relic"`
	Wizard                  bool          `json:"wizard"`
	WizardReveal            bool          `json:"wizardReveal"`
	HelpActive              bool          `json:"helpActive"`
	VisitedFloors           map[int]bool  `json:"visitedFloors"`
	TransitionFiredForLevel map[int]bool  `json:"transitionFiredForLevel"`
	RelicCollected          bool          `json:"relicCollected"`
	NextAmbienceTurn        int           `json:"nextAmbienceTurn"`
}

func NewGame(seed int64, tuning Tuning) *Game {
	rng := rand.New(rand.NewPCG(uint64(seed), 0x9e3779b97f4a7c15))
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
	// Place party at stairs up of floor 0
	g.Party = GenerateParty(rng, 1)
	start := g.Levels[0].StairsUp
	// Find nearby free tile if stairs blocked by enemy (rare)
	g.Party.Pos = start
	g.Floor = 0
	// Mark starting floor visited and transition-fired so re-entry does not re-fire before relic.
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
						g.Logf("enemy gains talent %s (%s floor %d)", tl, m.Class, fi+1)
					}
				}
				if len(m.Affixes) > 0 {
					for _, af := range m.Affixes {
						g.Logf("enemy gains affix %s (%s floor %d)", af, m.Class, fi+1)
					}
				}
			}
		}
	}
	g.UpdateFOV()
	return g
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
	if base == 0 {
		base = 100
	}
	factor := g.Tuning.LevelUp.XPFactor
	if factor == 0 {
		factor = 1.5
	}
	xp := float64(base)
	for i := 1; i < g.Level; i++ {
		xp *= factor
	}
	return int(xp + 0.5)
}

func (g *Game) GainXP(amount int) {
	if g.Over || g.LevelUpPending != nil {
		return
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
				g.Logf("%s will gain an affix: %s", m.Name, pick.Options[0])
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
		g.Logf("%s gains affix %s.", m.Name, affixID)
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
		g.Logf("%s learns talent %s.", m.Name, talentID)
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
	living := g.Party.LivingCount()
	if living == 0 {
		// No consumption if no living, but keep Food sync.
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
			g.Logf("You have fallen. Seed %d.", g.Seed)
		}
	} else {
		g.Logf("Vault opened +%d gold!", treasure)
	}
	g.removeFeatureAt(f.Pos, FeatureVault)
	return false
}

func (g *Game) handleForge(f *Feature) {
	if f == nil || !f.IsForge() {
		return
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
			return
		}
		g.Gold -= cost
		// Improve random living member ATK or DEF.
		members := g.Party.LivingMembers()
		if len(members) == 0 {
			return
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
			return
		}
		g.Food -= cost
		g.FoodFloat -= float64(cost)
		if g.Food < 0 {
			g.Food = 0
		}
		members := g.Party.LivingMembers()
		if len(members) == 0 {
			return
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
	// Den remains as marker; not removed on warning (optional)
}

func (g *Game) handlePitfall(f *Feature) bool {
	if f == nil || !f.IsPitfall() {
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
			g.Logf("You have fallen. Seed %d.", g.Seed)
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
	next := g.Party.Pos.Add(dir)
	// Stay in place (wait) if dir none - silent per UI parity
	if dir == DirNone {
		g.Party.Active = g.Party.Selected
		g.EndPlayerTurn("")
		return ActionResult{Moved: false}
	}
	if !lvl.InBounds(next) || !lvl.Walkable(next) {
		if lit := lvl.LitterAt(next); lit != nil && lit.BlocksMovement {
			switch lit.Category {
			case "impassable":
				g.Logf("You bump into a %s.", strings.ReplaceAll(lit.Kind, "_", " "))
			case "destructible":
				g.Logf("You bump into a %s.", strings.ReplaceAll(lit.Kind, "_", " "))
				// Optionally destroy on bump: remove litter
				// For now, just bump, don't destroy
			default:
				g.Logf("You bump the wall.")
			}
		} else {
			g.Logf("You bump the wall.")
		}
		return ActionResult{}
	}
	// Check enemy at next
	for _, e := range lvl.Enemies {
		if e.IsAlive() && e.Pos == next {
			g.Party.Active = g.Party.Selected
			attackerMember := g.Party.Members[g.Party.Active]
			attacker := attackerMember.Name
			dmg, hitIdx, killed := PlayerBumpEnemy(g.RNG, g.Party, e)
			memberName := e.MemberDisplayName(hitIdx)
			if !e.IsAlive() {
				g.Logf("%s hits %s for %d -- party slain!", attacker, e.DisplayName(), dmg)
				g.GainXP(20 + g.Floor*10)
			} else if killed {
				g.Logf("%s hits %s for %d -- slain!", attacker, memberName, dmg)
				g.GainXP(10 + g.Floor*5)
			} else {
				g.Logf("%s hits %s for %d.", attacker, memberName, dmg)
			}
			// Player effect placeholder
			if attackerMember.EffectChance > 0 && g.RNG.Float64() < attackerMember.EffectChance {
				effect := attackerMember.Effect
				if effect == "" {
					effect = "hex"
				}
				// defender for placeholder is the hit member name
				defenderName := memberName
				g.Logf("%s tries to %s %s", attacker, effect, defenderName)
			}
			// If level up pending, pause before enemy turn (world pauses)
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
	// Post-move feature interactions: Vault loot, Forge cost, Den warning.
	if f := g.featureAt(next); f != nil {
		if f.IsVault() {
			// Vault already passed locked check; claim treasure.
			// Copy before removal.
			vf := *f
			// Clear blocking check duplicate — handleVault does treasure/trap + removal.
			g.handleVault(&vf)
		} else if f.IsForge() {
			ff := *f
			g.handleForge(&ff)
		} else if f.IsDen() {
			g.handleDen(f)
		}
	}
	// Check relic on final floor
	if g.Floor == g.Tuning.Floors-1 && next == g.Relic {
		g.Over = true
		g.Won = true
		g.RelicCollected = true
		g.Logf("You claim the relic! Victory - seed %d.", g.Seed)
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
	}
	g.UpdateFOV()
}

func (g *Game) ApplyFloorTransition() {
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
	g.applyStarvation()
	g.MaybeTickAmbience()
	g.EnemyTurn()
	g.UpdateFOV()
}

func (g *Game) EnemyTurn() {
	if g.Over {
		return
	}
	lvl := g.CurLevel()
	for _, e := range lvl.Enemies {
		if !e.IsAlive() {
			continue
		}
		// Regen tick for troll and similar
		e.RegenTick()
		e.EnsureActive()
		dx := g.Party.Pos.X - e.Pos.X
		dy := g.Party.Pos.Y - e.Pos.Y
		cheb := max(abs(dx), abs(dy))
		if cheb == 1 {
			atk := e.Members[e.Active]
			raw := RollRaw(g.RNG, atk.ATK[0], atk.ATK[1])
			isMagic := atk.DamageType == "magic"
			hitIdx, actual := g.Party.ApplyDamageWithType(g.RNG, raw, isMagic)
			defender := "you"
			if hitIdx >= 0 && hitIdx < len(g.Party.Members) {
				defender = g.Party.Members[hitIdx].Name
			}
			attackerName := e.MemberDisplayName(e.Active)
			g.Logf("%s hits %s for %d.", attackerName, defender, actual)
			// Effect placeholder roll
			if atk.EffectChance > 0 {
				if g.RNG.Float64() < atk.EffectChance {
					effect := atk.Effect
					if effect == "" {
						effect = "hex"
					}
					g.Logf("%s tries to %s %s", attackerName, effect, defender)
				}
			}
			if g.Party.LivingCount() == 0 {
				g.Over = true
				g.Logf("You have fallen. Seed %d.", g.Seed)
				return
			}
			continue
		}
		// Move toward if within 8
		if cheb <= 8 {
			step := Dir{sign(dx), sign(dy)}
			nxt := e.Pos.Add(step)
			if !lvl.InBounds(nxt) || !lvl.Walkable(nxt) {
				// Try cardinal
				if abs(dx) > abs(dy) {
					nxt = e.Pos.Add(Dir{sign(dx), 0})
					if !lvl.Walkable(nxt) {
						nxt = e.Pos.Add(Dir{0, sign(dy)})
					}
				} else {
					nxt = e.Pos.Add(Dir{0, sign(dy)})
					if !lvl.Walkable(nxt) {
						nxt = e.Pos.Add(Dir{sign(dx), 0})
					}
				}
			}
			if lvl.Walkable(nxt) && nxt != g.Party.Pos {
				// Check collision with other enemies
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
		} else {
			// Wander
			dir := AllDirs[g.RNG.IntN(len(AllDirs))]
			nxt := e.Pos.Add(dir)
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

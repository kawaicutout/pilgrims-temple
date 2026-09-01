package game

import (
	"fmt"
	"math/rand/v2"
	"unicode"
)

// Game holds run state.
type Game struct {
	Seed           int64 `json:"seed"`
	RNG            *rand.Rand `json:"-"`
	Tuning         Tuning `json:"tuning"`
	Levels         []*Level `json:"levels"`
	Floor          int `json:"floor"`
	Party          *Party `json:"party"`
	Log            []string `json:"log"`
	Turn           int `json:"turn"`
	Food           int `json:"food"`
	FoodFloat      float64 `json:"foodFloat"`
	Level          int `json:"level"`
	XP             int `json:"xp"`
	XPToNext       int `json:"xpToNext"`
	LevelUpPending *LevelUpState `json:"levelUpPending"`
	Gold           int `json:"gold"`
	Kills          int `json:"kills"`
	Escaped        bool `json:"escaped"`
	Over           bool `json:"over"`
	Won            bool `json:"won"`
	Quit           bool `json:"quit"`
	Look           *LookState `json:"look"`
	Relic          Pos `json:"relic"`
	Wizard         bool `json:"wizard"`
}

func NewGame(seed int64, tuning Tuning) *Game {
	rng := rand.New(rand.NewPCG(uint64(seed), 0x9e3779b97f4a7c15))
	g := &Game{Seed: seed, RNG: rng, Tuning: tuning, Food: tuning.Food.StartClock, FoodFloat: float64(tuning.Food.StartClock), Level: 1}
	g.XPToNext = g.xpForNext()
	g.Levels = make([]*Level, tuning.Floors)
	for i := range tuning.Floors {
		lvl := NewLevel(tuning.Map.Width, tuning.Map.Height)
		lvl.Generate(rng, i)
		g.Levels[i] = lvl
	}
	// Place relic on final floor down stairs
	final := g.Levels[tuning.Floors-1]
	g.Relic = final.StairsDown
	// Place party at stairs up of floor 0
	g.Party = GenerateParty(rng, 1)
	start := g.Levels[0].StairsUp
	// Find nearby free tile if stairs blocked by enemy (rare)
	g.Party.Pos = start
	g.Floor = 0
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
		g.Logf("You bump the wall.")
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
				g.applyStarvation()
				g.UpdateFOV()
				return ActionResult{Attacked: true}
			}
			g.EndPlayerTurn("")
			return ActionResult{Attacked: true}
		}
	}
	// Move - silent (log reserved for combat/stairs/ambience)
	g.Party.Pos = next
	g.Party.Active = g.Party.Selected
	// Check relic on final floor
	if g.Floor == g.Tuning.Floors-1 && next == g.Relic {
		g.Over = true
		g.Won = true
		g.Logf("You claim the relic! Victory - seed %d.", g.Seed)
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
	g.ApplyFloorTransition()
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
	g.UpdateFOV()
}

func (g *Game) ApplyFloorTransition() {
	// DESIGN: cleric Restoration & druid Forage on floor transition (M3), but M1 food tick stub.
	for _, m := range g.Party.Members {
		if !m.IsAlive() {
			continue
		}
		switch m.Class {
		case "druid":
			// Forage stub: would add food; food not yet tracked in M1
		case "cleric":
			// Restoration stub
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
	g.applyStarvation()
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

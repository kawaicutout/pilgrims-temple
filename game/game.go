package game

import (
	"fmt"
	"math/rand/v2"
)

// Game holds run state.
type Game struct {
	Seed    int64
	RNG     *rand.Rand
	Tuning  Tuning
	Levels  []*Level
	Floor   int
	Party   *Party
	Log     []string // 8 lines, oldest dropped
	Turn    int
	Food    int
	Over    bool
	Won     bool
	Quit    bool // ESC quit to main menu, not a death
	Relic   Pos // on final floor
}
func NewGame(seed int64, tuning Tuning) *Game {
	rng := rand.New(rand.NewPCG(uint64(seed), 0x9e3779b97f4a7c15))
	g := &Game{Seed: seed, RNG: rng, Tuning: tuning, Food: tuning.Food.StartClock}
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
	g.UpdateFOV()
	return g
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
	ratio := float64(g.Food) / float64(g.Tuning.Food.StartClock)
	if ratio <= g.Tuning.Food.StarvingThreshold {
		return "Starving"
	}
	if ratio <= g.Tuning.Food.HungryThreshold {
		return "Hungry"
	}
	return "Ok"
}

func (g *Game) Logf(fmtStr string, args ...any) {
	s := fmt.Sprintf(fmtStr, args...)
	g.Log = append(g.Log, s)
	if len(g.Log) > g.Tuning.Layout.LogLines {
		g.Log = g.Log[len(g.Log)-g.Tuning.Layout.LogLines:]
	}
}

// Action results
type ActionResult struct {
	Moved bool
	Attacked bool
	Descended bool
	Ascended bool
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
	// Check enemy at next
	for _, e := range lvl.Enemies {
		if e.IsAlive() && e.Pos == next {
			g.Party.Active = g.Party.Selected
			attacker := g.Party.Members[g.Party.Active].Name
			dmg, hitIdx, killed := PlayerBumpEnemy(g.RNG, g.Party, e)
			memberName := e.MemberDisplayName(hitIdx)
			if !e.IsAlive() {
				g.Logf("%s hits %s for %d -- party slain!", attacker, e.DisplayName(), dmg)
			} else if killed {
				g.Logf("%s hits %s for %d -- slain!", attacker, memberName, dmg)
			} else {
				g.Logf("%s hits %s for %d.", attacker, memberName, dmg)
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
	// Food tick: per living member
	if g.Tuning.Food.PerMemberPerTurn > 0 {
		g.Food -= g.Party.LivingCount() * g.Tuning.Food.PerMemberPerTurn
		if g.Food < 0 {
			g.Food = 0
		}
		if g.Food == 0 {
			g.Logf("You starve.")
			// For now starving just warns; M4 can add damage.
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
		e.EnsureActive()
		dx := g.Party.Pos.X - e.Pos.X
		dy := g.Party.Pos.Y - e.Pos.Y
		cheb := max(abs(dx), abs(dy))
		if cheb == 1 {
			atk := e.Members[e.Active]
			raw := RollRaw(g.RNG, atk.ATK[0], atk.ATK[1])
			hitIdx, actual := g.Party.ApplyDamage(g.RNG, raw)
			defender := "you"
			if hitIdx >= 0 && hitIdx < len(g.Party.Members) {
				defender = g.Party.Members[hitIdx].Name
			}
			attackerName := e.MemberDisplayName(e.Active)
			g.Logf("%s hits %s for %d.", attackerName, defender, actual)
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

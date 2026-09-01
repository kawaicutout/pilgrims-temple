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
	Over    bool
	Won     bool
	Relic   Pos // on final floor
}

func NewGame(seed int64, tuning Tuning) *Game {
	rng := rand.New(rand.NewPCG(uint64(seed), 0x9e3779b97f4a7c15))
	g := &Game{Seed: seed, RNG: rng, Tuning: tuning}
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
	g.Logf("Seed %d — Pilgrim's Temple, %d floors.", seed, tuning.Floors)
	g.Logf("You stand at the temple threshold.")
	g.UpdateFOV()
	return g
}

func (g *Game) CurLevel() *Level { return g.Levels[g.Floor] }

func (g *Game) UpdateFOV() {
	lvl := g.CurLevel()
	ComputeFOV(lvl, g.Party.Pos, g.Party.BestLight())
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
	for _, e := range lvl.Enemies {
		if e.IsAlive() && e.Pos == next {
			// Bump combat
			g.Party.Active = g.Party.Selected
			dmgDealt, dmgTakenRaw, killed := PlayerBumpEnemy(g.RNG, g.Party, e)
			if killed {
				g.Logf("You hit %s for %d — slain!", e.Name, dmgDealt)
			} else {
				g.Logf("You hit %s for %d.", e.Name, dmgDealt)
				g.Party.ApplyDamage(g.RNG, dmgTakenRaw)
				// Log actual post-DEF? We log raw; with DEF variation the displayed number may be 1 higher than actual.
				g.Logf("%s hits you for %d.", e.Name, dmgTakenRaw)
				if g.Party.LivingCount() == 0 {
					g.Over = true
					g.Logf("You have fallen. Seed %d.", g.Seed)
				}
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
	// Natural regen? M1: 1 HP per turn for living members? But DESIGN says 1/turn.
	// For solo, regen quickly would trivialize. Keep but cap.
	// Apply regen after player action
	for _, m := range g.Party.Members {
		if m.IsAlive() && m.HP < m.MaxHP {
			m.HP++
			if m.HP > m.MaxHP {
				m.HP = m.MaxHP
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
		// Simple AI: if adjacent to player, attack; else if player visible, step toward; else wander
		if !lvl.Visible[e.Pos.Y][e.Pos.X] && !g.CurLevel().Visible[e.Pos.Y][e.Pos.X] {
			// Not visible to player, but enemy may still see? Symmetric for M1: use same FOV stub.
			// For M1, enemies act regardless; keep simple distance check
		}
		dx := g.Party.Pos.X - e.Pos.X
		dy := g.Party.Pos.Y - e.Pos.Y
		cheb := max(abs(dx), abs(dy))
		if cheb == 1 {
			// Attack — raw roll; DEF applied inside ApplyDamage to the actual hit member
			raw := RollRaw(g.RNG, e.ATK[0], e.ATK[1])
			g.Party.ApplyDamage(g.RNG, raw)
			g.Logf("%s hits you for %d.", e.Name, raw)
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

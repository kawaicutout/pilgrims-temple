package game

import (
	"fmt"
	"strings"
)

// LookState holds look-mode cursor state.
type LookState struct {
	Cursor Pos
	Active bool
}

// LookMode is alias for LookState (spec compatibility).
type LookMode = LookState

// Examine returns a description of the tile at p.
// It respects FOV: unseen tiles return Unseen/Not visible, otherwise it
// describes the party, enemy party, or terrain.
func Examine(g *Game, p Pos) string {
	if g == nil {
		return "Empty."
	}
	lvl := g.CurLevel()
	if lvl == nil {
		return "Empty."
	}
	if !lvl.InBounds(p) {
		return "Out of bounds."
	}
	// FOV check — must be currently visible.
	if p.Y < 0 || p.Y >= len(lvl.Visible) || p.X < 0 || p.X >= len(lvl.Visible[0]) {
		return "Unseen."
	}
	visible := lvl.Visible[p.Y][p.X]
	if !visible {
		if lvl.Seen[p.Y][p.X] {
			return "Not visible (remembered)."
		}
		return "Unseen."
	}

	// Party on tile.
	if g.Party != nil && g.Party.Pos == p {
		return describeParty(g.Party, "Party")
	}

	// Enemy party on tile.
	for _, e := range lvl.Enemies {
		if e.Pos == p && e.IsAlive() {
			return describeEnemyParty(e)
		}
	}

	// Check relic on final floor.
	if g.Floor == len(g.Levels)-1 && p == g.Relic {
		return "Relic — the pilgrim's goal."
	}

	// Terrain.
	t := lvl.At(p)
	switch t {
	case TileWall:
		return "Wall."
	case TileStairsDown:
		return "Stairs down (>)"
	case TileStairsUp:
		return "Stairs up (<)"
	case TileFloor:
		return "Empty."
	default:
		return "Empty."
	}
}

func describeParty(p *Party, label string) string {
	if p == nil || len(p.Members) == 0 {
		return label + ": empty."
	}
	var parts []string
	for i, m := range p.Members {
		status := ""
		if !m.IsAlive() {
			status = " (dead)"
		}
		parts = append(parts, fmt.Sprintf("%s HP %d/%d ATK %d-%d DEF %d MDEF %d%s", memberLabel(m, i), m.HP, m.MaxHP, m.ATK[0], m.ATK[1], m.DEF, m.MDEF, status))
	}
	count := len(p.Members)
	header := fmt.Sprintf("%s x%d", label, count)
	if count == 1 {
		header = label
	}
	return header + ": " + strings.Join(parts, "; ")
}

func describeEnemyParty(e *EnemyParty) string {
	if e == nil || len(e.Members) == 0 {
		return "Empty."
	}
	header := e.DisplayName()
	var parts []string
	for i, m := range e.Members {
		status := ""
		if !m.IsAlive() {
			status = " (dead)"
		}
		name := e.MemberDisplayName(i)
		parts = append(parts, fmt.Sprintf("%s HP %d/%d ATK %d-%d DEF %d MDEF %d%s", name, m.HP, m.MaxHP, m.ATK[0], m.ATK[1], m.DEF, m.MDEF, status))
	}
	if header == "" {
		header = "Enemy"
	}
	return header + ": " + strings.Join(parts, "; ")
}

func memberLabel(m *Member, idx int) string {
	if m == nil {
		return fmt.Sprintf("member%d", idx+1)
	}
	if m.Name != "" {
		return m.Name
	}
	if m.Class != "" {
		return m.Class
	}
	return fmt.Sprintf("member%d", idx+1)
}

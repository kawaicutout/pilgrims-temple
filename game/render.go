package game

import (
	"fmt"
	"strings"
)

// Frame is the view model both frontends render.
type Frame struct {
	W, H      int
	Cells     [][]Cell // H x W
	Panel     []string // side panel lines (already truncated)
	Status    string
	Log       []string // 8 lines
	Hints     string
	Over      bool
	Won       bool
	Quit      bool
	MinCols   int
	MinRows   int
}

func (g *Game) Render() Frame {
	t := g.Tuning
	lvl := g.CurLevel()
	w, h := t.Map.Width, t.Map.Height
	cells := make([][]Cell, h)
	for y := range h {
		cells[y] = make([]Cell, w)
		for x := range w {
			p := Pos{x, y}
			vis := lvl.Visible[y][x]
			seen := lvl.Seen[y][x]
			var glyph rune
			var fg string
			if !seen {
				// Fog: styled background only (DESIGN 10.4) — space with bg
				cells[y][x] = Cell{Glyph: ' ', FG: "bg", BG: "bg"}
				continue
			}
			if !vis {
				// Seen but not currently visible: dim
				glyph = lvl.At(p).Glyph()
				fg = "gray-3"
			} else {
				glyph = lvl.At(p).Glyph()
				// Check enemy at pos
				for _, e := range lvl.Enemies {
					if e.IsAlive() && e.Pos == p {
						glyph = e.Glyph()
						fg = "enemy"
						break
					}
				}
				if p == g.Party.Pos {
					glyph = '@'
					fg = "player"
				} else if fg == "" {
					// Terrain colors per design-guide
					switch lvl.At(p) {
					case TileWall:
						fg = "wall"
					case TileFloor:
						fg = "floor"
					case TileStairsDown, TileStairsUp:
						fg = "gold"
					default:
						fg = "fg"
					}
				}
			}
			cells[y][x] = Cell{Glyph: glyph, FG: fg, BG: "bg"}
		}
	}
	// Panel
	var panel []string
	for i, m := range g.Party.Members {
		sel := " "
		if i == g.Party.Selected {
			sel = ">"
		}
		affixStr := ""
		if len(m.Affixes) > 0 {
			affixStr = " " + strings.Join(m.Affixes, ",")
		}
		nameLine := fmt.Sprintf("%s %d %s%s", sel, i+1, m.Name, affixStr)
		if len(nameLine) > 30 {
			nameLine = nameLine[:27] + "..."
		}
		if !m.IsAlive() {
			nameLine = fmt.Sprintf("  %d %s (fallen)", i+1, m.Name)
		}
		talentStr := ""
		if len(m.Talents) > 0 {
			talentStr = " " + strings.Join(m.Talents, ",")
			if len(talentStr) > 30 {
				talentStr = talentStr[:27] + "..."
			}
		}
		classLine := fmt.Sprintf("  %s %d/%d%s", strings.Title(m.Class), m.HP, m.MaxHP, talentStr)
		panel = append(panel, nameLine, classLine, "")
	}
	// Pad to 4 slots
	for len(panel) < 12 {
		panel = append(panel, "")
	}
	panel = append(panel, "Potions:", "  (none)", "Scrolls:", "  (none)")
	// Status - FOOD + Level/XP + floor
	floorStr := fmt.Sprintf("Floor %d/%d", g.Floor+1, t.Floors)
	hpStr := ""
	if len(g.Party.Members) > 0 {
		m := g.Party.Members[0]
		hpStr = fmt.Sprintf("HP %d/%d", m.HP, m.MaxHP)
	}
	foodStr := fmt.Sprintf("FOOD %d %s", g.Food, g.HungerState())
	levelStr := fmt.Sprintf("Lvl %d XP %d/%d", g.Level, g.XP, g.XPToNext)
	status := fmt.Sprintf("%s | %s | %s | %s | Turn %d", floorStr, hpStr, foodStr, levelStr, g.Turn)

	// Log padded to 8
	logLines := make([]string, t.Layout.LogLines)
	for i := range logLines {
		logLines[i] = ""
	}
	copy(logLines[max(0, len(logLines)-len(g.Log)):], g.Log)

	hints := "Move: numpad/arrow/hjkl  Wait:5/.  Stairs:>/ <  Quit:Esc  Help:?"
	if g.Quit {
		hints = "Quit to menu. Seed " + fmt.Sprint(g.Seed) + " - Esc again or close window"
	} else if g.Over {
		if g.Won {
			hints = "VICTORY! Press Esc to quit. Seed " + fmt.Sprint(g.Seed)
		} else {
			hints = "YOU DIED. Press Esc to quit. Seed " + fmt.Sprint(g.Seed)
		}
	}
	return Frame{
		W: w, H: h, Cells: cells,
		Panel:   panel,
		Status:  status,
		Log:     logLines,
		Hints:   hints,
		Over:    g.Over,
		Won:     g.Won,
		Quit:    g.Quit,
		MinCols: t.Layout.MinCols,
		MinRows: t.Layout.MinRows,
	}
}

func (g *Game) RenderLevelUp() Frame {
	t := g.Tuning
	w, h := t.Map.Width, t.Map.Height
	cells := make([][]Cell, h)
	for y := range h {
		cells[y] = make([]Cell, w)
		for x := range w {
			cells[y][x] = Cell{Glyph: ' ', FG: "bg", BG: "bg"}
		}
	}
	if g.LevelUpPending == nil || len(g.LevelUpPending.Picks) == 0 {
		drawCentered(cells, w, h/2, "LEVEL UP!", "gold-bright")
		return Frame{W: w, H: h, Cells: cells, Panel: []string{}, Status: fmt.Sprintf("Lvl %d", g.Level), Log: make([]string, t.Layout.LogLines), Hints: "No picks", MinCols: t.Layout.MinCols, MinRows: t.Layout.MinRows}
	}
	pick := g.LevelUpPending.Picks[g.LevelUpPending.Current]
	title := fmt.Sprintf("LEVEL %d - %s CHOOSES", g.LevelUpPending.NewLevel, pick.MemberName)
	drawCentered(cells, w, 2, title, "gold-bright")
	if pick.IsAffix {
		drawCentered(cells, w, 4, "Gain affix:", "gray-1")
		opt := pick.Options[0]
		fg := "gold"
		drawCentered(cells, w, 6, fmt.Sprintf("> %s", opt), fg)
		drawCentered(cells, w, 8, "Press Enter to accept", "gray-2")
	} else {
		drawCentered(cells, w, 4, fmt.Sprintf("%s (%s) choose talent:", pick.MemberName, pick.Class), "gray-1")
		for i, opt := range pick.Options {
			fg := "gray-1"
			prefix := "  "
			if i == 0 {
				prefix = "> "
				fg = "gold-bright"
			}
			line := fmt.Sprintf("%s%d: %s", prefix, i+1, opt)
			drawCentered(cells, w, 6+i*2, line, fg)
		}
		drawCentered(cells, w, 12, "Press 1/2/3 or Enter for first", "gray-2")
	}
	panel := []string{"", fmt.Sprintf("Lvl %d", g.Level), fmt.Sprintf("Pick %d/%d", g.LevelUpPending.Current+1, len(g.LevelUpPending.Picks))}
	for len(panel) < 12 {
		panel = append(panel, "")
	}
	status := fmt.Sprintf("Level up! Choose for %s", pick.MemberName)
	hints := "1/2/3 or Enter: choose  Esc: skip (not recommended)"
	if pick.IsAffix {
		hints = "Enter: accept affix"
	}
	return Frame{W: w, H: h, Cells: cells, Panel: panel, Status: status, Log: make([]string, t.Layout.LogLines), Hints: hints, MinCols: t.Layout.MinCols, MinRows: t.Layout.MinRows}
}


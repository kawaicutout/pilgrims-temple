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
	PanelFG   []string // per-line FG token parallel to Panel (gold-bright, gray-1, red-bright, slate)
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
						fg = e.Color()
						if fg == "" {
							fg = "red-bright"
						}
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
				// Look cursor highlight
				if g.Look != nil && g.Look.Active && p == g.Look.Cursor {
					fg = "gold-bright"
				}
			}
			cells[y][x] = Cell{Glyph: glyph, FG: fg, BG: "bg"}
		}
	}
	// Panel
	var panel []string
	var panelFG []string
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
			if len(nameLine) > 30 {
				nameLine = nameLine[:27] + "..."
			}
		}
		// color decision after truncation
		var fg string
		if !m.IsAlive() {
			fg = "slate"
		} else if i == g.Party.Selected {
			fg = "gold-bright"
		} else if m.MaxHP > 0 && m.HP*4 <= m.MaxHP {
			fg = "red-bright"
		} else {
			fg = "gray-1"
		}
		talentStr := ""
		if len(m.Talents) > 0 {
			talentStr = " " + strings.Join(m.Talents, ",")
			if len(talentStr) > 30 {
				talentStr = talentStr[:27] + "..."
			}
		}
		classLine := fmt.Sprintf("  %s %d/%d%s", strings.Title(m.Class), m.HP, m.MaxHP, talentStr)
		var classFG string
		if fg == "slate" || fg == "red-bright" {
			classFG = fg
		} else {
			classFG = "gray-1"
		}
		statsLine := fmt.Sprintf("  ATK %d-%d DEF %d MDEF %d", m.ATK[0], m.ATK[1], m.DEF, m.MDEF)
		panel = append(panel, nameLine, classLine, statsLine)
		panelFG = append(panelFG, fg, classFG, classFG)
	}
	// Pad to 4 slots (each slot is 3 lines)
	for len(panel) < 12 {
		panel = append(panel, "")
		panelFG = append(panelFG, "gray-1")
	}
	panel = append(panel, "Potions:", "  (none)", "Scrolls:", "  (none)")
	panelFG = append(panelFG, "gray-1", "gray-1", "gray-1", "gray-1")
	// Status - Floor | FOOD | Level/XP | Turn (no HP, no Seed)
	floorStr := fmt.Sprintf("Floor %d/%d", g.Floor+1, t.Floors)
	foodStr := fmt.Sprintf("FOOD %d %s", g.Food, g.HungerState())
	levelStr := fmt.Sprintf("Lvl %d XP %d/%d", g.Level, g.XP, g.XPToNext)
	status := fmt.Sprintf("%s | %s | %s | Turn %d", floorStr, foodStr, levelStr, g.Turn)

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
		PanelFG: panelFG,
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
		return Frame{W: w, H: h, Cells: cells, Panel: []string{}, PanelFG: []string{}, Status: fmt.Sprintf("Lvl %d", g.Level), Log: make([]string, t.Layout.LogLines), Hints: "No picks", MinCols: t.Layout.MinCols, MinRows: t.Layout.MinRows}
	}
	pick := g.LevelUpPending.Picks[g.LevelUpPending.Current]
	title := fmt.Sprintf("LEVEL %d - %s CHOOSES", g.LevelUpPending.NewLevel, pick.MemberName)
	drawCentered(cells, w, 2, title, "gold-bright")
	if pick.IsAffix {
		drawCentered(cells, w, 4, "Gain affix:", "gray-1")
		opt := pick.Options[0]
		desc := GetAffixDesc(opt)
		if desc == opt {
			if alt := GetTalentDesc(opt); alt != opt {
				desc = alt
			}
		}
		text := opt
		if desc != opt {
			text = desc
		}
		line := fmt.Sprintf("> %s", text)
		if len([]rune(line)) > w-2 {
			line = string([]rune(line)[:w-2])
		}
		drawCentered(cells, w, 6, line, "gold")
	} else {
		drawCentered(cells, w, 4, fmt.Sprintf("%s (%s) choose talent:", pick.MemberName, pick.Class), "gray-1")
		cursor := g.LevelUpPending.Cursor
		for i, opt := range pick.Options {
			fg := "gray-1"
			prefix := "  "
			if i == cursor {
				prefix = "> "
				fg = "gold-bright"
			}
			desc := GetTalentDesc(opt)
			line := fmt.Sprintf("%s%d: %s", prefix, i+1, desc)
			if len([]rune(line)) > w-2 {
				line = string([]rune(line)[:w-2])
			}
			drawString(cells, 2, 6+i*2, line, fg)
		}
		drawCentered(cells, w, 12, "Up/Down + Enter: choose  1/2/3: quick pick", "gray-2")
	}
	panel := []string{"", fmt.Sprintf("Lvl %d", g.Level), fmt.Sprintf("Pick %d/%d", g.LevelUpPending.Current+1, len(g.LevelUpPending.Picks))}
	panelFG := []string{"gray-1", "gray-1", "gray-1"}
	for len(panel) < 12 {
		panel = append(panel, "")
		panelFG = append(panelFG, "gray-1")
	}
	status := fmt.Sprintf("Level up! Choose for %s", pick.MemberName)
	hints := "Up/Down: move  Enter: choose  1/2/3: quick pick  Esc: skip"
	if pick.IsAffix {
		hints = "Enter: accept affix"
	}
	return Frame{W: w, H: h, Cells: cells, Panel: panel, PanelFG: panelFG, Status: status, Log: make([]string, t.Layout.LogLines), Hints: hints, MinCols: t.Layout.MinCols, MinRows: t.Layout.MinRows}
}

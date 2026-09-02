package game

import (
	"fmt"
	"sort"
	"strings"
)

// Frame is the view model both frontends render.
type Frame struct {
	W, H    int
	Cells   [][]Cell // H x W
	Panel   []string // side panel lines (already truncated)
	PanelFG []string // per-line FG token parallel to Panel (gold-bright, gray-1, red-bright, slate)
	Status  string
	Log     []string // 8 lines
	Hints   string
	Over    bool
	Won     bool
	Quit    bool
	MinCols int
	MinRows int
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
				if lvl.IsDoor(p) {
					glyph = lvl.DoorGlyph(p)
				} else {
					glyph = lvl.At(p).Glyph()
				}
				fg = "gray-3"
			} else {
				if lvl.IsDoor(p) {
					glyph = lvl.DoorGlyph(p)
				} else {
					glyph = lvl.At(p).Glyph()
				}
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
				// Check feature at pos (if no enemy)
				if fg == "" && p != g.Party.Pos {
					for _, f := range lvl.Features {
						if f.Pos == p {
							// Hidden pitfalls stay hidden until wizard reveal (or triggered).
							if f.Hidden && !g.WizardReveal {
								break
							}
							glyph = f.Glyph()
							fg = f.Color()
							if fg == "" {
								fg = "gold"
							}
							break
						}
					}
				}
				// Check ground item at pos (if no enemy/feature)
				if fg == "" && p != g.Party.Pos {
					if it := lvl.ItemAt(p); it != nil {
						glyph = it.Glyph()
						fg = it.Color()
					}
				}
				// Check litter at pos (if no enemy/feature/item)
				if fg == "" && p != g.Party.Pos {
					if lit := lvl.LitterAt(p); lit != nil {
						glyph = lit.Glyph
						if lit.Color != "" {
							fg = lit.Color
						} else {
							switch lit.Category {
							case "impassable":
								fg = WallColorForLevel(lvl)
							case "destructible":
								fg = "gray-2"
							default:
								fg = FloorColorForLevel(lvl)
							}
						}
					}
				}
				if p == g.Party.Pos {
					glyph = '@'
					fg = "player"
				} else if fg == "" {
					switch lvl.At(p) {
					case TileWall:
						fg = WallColorForLevel(lvl)
					case TileFloor:
						fg = FloorColorForLevel(lvl)
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
				// Throw cursor highlight (gold-bright, distinct from Look)
				if g.ThrowPending.Active && p == g.ThrowPending.Cursor {
					fg = "gold-bright"
				}
			}
			cells[y][x] = Cell{Glyph: glyph, FG: fg, BG: "bg"}
		}
	}
	// Panel - right-side overhaul: 5 lines per member (20 total) + 6 potion/scroll lines = 26
	var panel []string
	var panelFG []string
	const panelWrap = 28
	buildTalentLines := func(talents []string) (string, string) {
		if len(talents) == 0 {
			return "  ", "  "
		}
		friendly := make([]string, len(talents))
		for i, t := range talents {
			friendly[i] = FriendlyID(t)
		}
		lines := []string{"  ", "  "}
		cur := 0
		for _, part := range friendly {
			trimmed := strings.TrimSpace(lines[cur])
			sep := ", "
			if trimmed == "" {
				sep = ""
			}
			needed := len(sep) + len(part)
			if len(lines[cur])+needed <= panelWrap {
				lines[cur] += sep + part
			} else {
				if cur == 0 {
					cur = 1
					trimmed2 := strings.TrimSpace(lines[cur])
					sep2 := ""
					if trimmed2 != "" {
						sep2 = ", "
					}
					if len(lines[cur])+len(sep2)+len(part) <= panelWrap {
						lines[cur] += sep2 + part
					} else {
						if trimmed2 == "" {
							lines[cur] += part
						} else {
							lines[cur] += ", " + part
						}
					}
				} else {
					if strings.TrimSpace(lines[cur]) == "" {
						lines[cur] += part
					} else {
						lines[cur] += ", " + part
					}
				}
			}
		}
		if len(lines[1]) > panelWrap {
			if panelWrap > 3 {
				lines[1] = lines[1][:panelWrap-3] + "..."
			} else {
				lines[1] = lines[1][:panelWrap]
			}
		}
		return lines[0], lines[1]
	}
	for slot := 0; slot < len(g.Party.Members) && slot < 4; slot++ {
		m := g.Party.Members[slot]
		var fg string
		if !m.IsAlive() {
			fg = "slate"
		} else if slot == g.Party.Selected {
			fg = "gold-bright"
		} else if m.MaxHP > 0 && m.HP*4 <= m.MaxHP {
			fg = "red-bright"
		} else {
			fg = "gray-1"
		}
		classFG := fg
		if fg != "slate" && fg != "red-bright" {
			classFG = "gray-1"
		}
		var line1 string
		if !m.IsAlive() {
			line1 = fmt.Sprintf("%d %s (fallen)", slot+1, m.Name)
		} else {
			line1 = fmt.Sprintf("%d %s %d/%d", slot+1, m.Name, m.HP, m.MaxHP)
		}
		classFriendly := FriendlyID(m.Class)
		var affixFriendly []string
		for _, a := range m.Affixes {
			affixFriendly = append(affixFriendly, FriendlyID(a))
		}
		var line2 string
		if len(affixFriendly) > 0 {
			line2 = fmt.Sprintf("  %s %s", classFriendly, strings.Join(affixFriendly, ", "))
		} else {
			line2 = fmt.Sprintf("  %s", classFriendly)
		}
		t1, t2 := buildTalentLines(m.Talents)
		line5 := ""
		panel = append(panel, line1, line2, t1, t2, line5)
		panelFG = append(panelFG, fg, classFG, classFG, classFG, "gray-1")
	}
	potionCounts := map[string]int{}
	scrollCounts := map[string]int{}
	for _, it := range g.Party.Inventory {
		app := appearanceFromItem(it)
		if it.Kind == "potion" {
			potionCounts[app]++
		} else if it.Kind == "scroll" {
			scrollCounts[app]++
		}
	}
	buildInvLines := func(header string, counts map[string]int, kind string) []string {
		if len(counts) == 0 {
			return []string{header + " (none)", "  ", "  "}
		}
		keys := make([]string, 0, len(counts))
		for k := range counts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var parts []string
		for _, app := range keys {
			cnt := counts[app]
			display := app
			if IsIdentified(app) {
				if tid, ok := Knowledge[app]; ok && tid != "" {
					display = friendlyTypeName(tid, kind)
				} else if tid := TypeForAppearance(app); tid != "" {
					display = friendlyTypeName(tid, kind)
				}
			}
			if cnt > 1 {
				parts = append(parts, fmt.Sprintf("%s (x%d)", display, cnt))
			} else {
				parts = append(parts, display)
			}
		}
		lines := []string{header, "  ", "  "}
		cur := 0
		for _, part := range parts {
			var sep string
			if cur == 0 {
				has := len(lines[0]) > len(header)
				if has {
					sep = ", "
				} else {
					sep = " "
				}
			} else {
				if strings.TrimSpace(lines[cur]) == "" {
					sep = ""
				} else {
					sep = ", "
				}
			}
			needed := len(sep) + len(part)
			if len(lines[cur])+needed <= panelWrap {
				lines[cur] += sep + part
			} else {
				if cur < 2 {
					cur++
					var sep2 string
					if strings.TrimSpace(lines[cur]) == "" {
						sep2 = ""
					} else {
						sep2 = ", "
					}
					if len(lines[cur])+len(sep2)+len(part) <= panelWrap {
						lines[cur] += sep2 + part
					} else {
						if cur == 1 && len(lines[cur])+len(sep2)+len(part) > panelWrap {
							if len("  "+part) <= panelWrap {
								cur = 2
								lines[cur] += part
							} else {
								cur = 2
								if strings.TrimSpace(lines[cur]) == "" {
									lines[cur] += part
								} else {
									lines[cur] += ", " + part
								}
							}
						} else {
							if strings.TrimSpace(lines[cur]) == "" {
								lines[cur] += part
							} else {
								lines[cur] += ", " + part
							}
						}
					}
				} else {
					if strings.TrimSpace(lines[cur]) == "" {
						lines[cur] += part
					} else {
						lines[cur] += ", " + part
					}
				}
			}
		}
		if len(lines[2]) > panelWrap {
			if panelWrap > 3 {
				lines[2] = lines[2][:panelWrap-3] + "..."
			} else {
				lines[2] = lines[2][:panelWrap]
			}
		}
		return lines
	}
	potionLines := buildInvLines("Potions:", potionCounts, "potion")
	scrollLines := buildInvLines("Scrolls:", scrollCounts, "scroll")
	panel = append(panel, potionLines...)
	panel = append(panel, scrollLines...)
	panelFG = append(panelFG, "gray-1", "gray-1", "gray-1", "gray-1", "gray-1", "gray-1")
	for len(panel) < 26 {
		panel = append(panel, "")
		panelFG = append(panelFG, "gray-1")
	}
	if len(panel) > 26 {
		panel = panel[:26]
		panelFG = panelFG[:26]
	}
	// Status - Floor | Food | Carry | Level/XP | Gold | Score (no Turn, no HP, no Seed)
	// Panel below map (status bar) shows Carry as "Carry C/M" in gold; Food is capitalized as "Food" not "FOOD".
	floorStr := fmt.Sprintf("Floor %d/%d", g.Floor+1, t.Floors)
	foodStr := fmt.Sprintf("Food %d %s", g.Food, g.HungerState())
	carryMax := g.Party.CarryCapacity()
	carryUsed := g.Party.CarryUsed()
	carryStr := fmt.Sprintf("Carry %d/%d", carryUsed, carryMax)
	levelStr := fmt.Sprintf("Level %d XP %d/%d", g.Level, g.XP, g.XPToNext)
	goldStr := fmt.Sprintf("Gold %d", g.Gold)
	scoreStr := fmt.Sprintf("Score %d", g.CalculateScore())
	status := fmt.Sprintf("%s | %s | %s | %s | %s | %s", floorStr, foodStr, carryStr, levelStr, goldStr, scoreStr)
	logLines := make([]string, t.Layout.LogLines)
	for i := range logLines {
		logLines[i] = ""
	}
	copy(logLines[max(0, len(logLines)-len(g.Log)):], g.Log)

	hints := "Move: numpad/arrow/hjkl  Wait:5/.  Rest:z  Use:u(menu)  Throw:t(menu+cursor)  Stairs:>/ <  Quit:Esc  Help:?"
	if g.ThrowPending.Active {
		hints = "Throw: move cursor, Enter to throw, Esc to cancel"
	} else if g.Merchant.Active {
		hints = "Merchant: Up/Down select, Enter buy, Esc leave"
	} else if g.Quit {
		hints = "Quit to menu. Seed " + fmt.Sprint(g.Seed) + " - Esc again or close window"
	} else if g.Over {
		if g.Won {
			hints = fmt.Sprintf("VICTORY! Score %d. Press Esc to quit. Seed %d", g.CalculateScore(), g.Seed)
		} else {
			hints = fmt.Sprintf("YOU DIED. Score %d. Press Esc to quit. Seed %d", g.CalculateScore(), g.Seed)
		}
	} else if f := g.featureAt(g.Party.Pos); f != nil {
		if f.IsFountain() {
			hints = "Fountain: g to drink  |  Move: numpad/arrow/hjkl  Wait:5/.  Rest:z  Help:?"
		} else if f.IsMerchant() {
			hints = "Merchant: g to browse  |  Move: numpad/arrow/hjkl  Wait:5/.  Help:?"
		} else if f.IsForge() {
			hints = "Forge: g to use (u also)  |  Move: numpad/arrow/hjkl  Wait:5/."
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
func (g *Game) RenderUseMenu(selected int) Frame {
	t := g.Tuning
	w, h := t.Map.Width, t.Map.Height
	cells := make([][]Cell, h)
	for y := range h {
		cells[y] = make([]Cell, w)
		for x := range w {
			cells[y][x] = Cell{Glyph: ' ', FG: "bg", BG: "bg"}
		}
	}
	entries := g.InventoryUseEntries()
	title := "USE ITEM"
	sub := "Select potion/scroll to use"
	if len(entries) == 0 {
		sub = "No potions or scrolls"
	}
	drawCentered(cells, w, 2, title, "gold-bright")
	drawCentered(cells, w, 3, sub, "gray-1")
	if len(entries) == 0 {
		drawCentered(cells, w, 5, "(empty)", "gray-2")
	} else {
		startY := 5
		if selected < 0 {
			selected = 0
		}
		if selected >= len(entries) {
			selected = len(entries) - 1
		}
		maxRows := h - 7
		if maxRows < 1 {
			maxRows = 1
		}
		start := 0
		if len(entries) > maxRows {
			start = selected - maxRows/2
			if start < 0 {
				start = 0
			}
			if start+maxRows > len(entries) {
				start = len(entries) - maxRows
			}
		}
		end := start + maxRows
		if end > len(entries) {
			end = len(entries)
		}
		for i := start; i < end; i++ {
			e := entries[i]
			fg := "gray-1"
			prefix := "  "
			if i == selected {
				prefix = "> "
				fg = "gold-bright"
			}
			var line string
			if e.Count > 1 {
				line = fmt.Sprintf("%s%s (x%d)", prefix, e.DisplayName, e.Count)
			} else {
				line = fmt.Sprintf("%s%s", prefix, e.DisplayName)
			}
			if len(line) > w-4 {
				line = line[:w-7] + "..."
			}
			drawCentered(cells, w, startY+(i-start)*2, line, fg)
			if i == selected && IsIdentified(e.Appearance) && e.DisplayName != e.Appearance {
				detail := fmt.Sprintf("(%s)", e.Appearance)
				drawCentered(cells, w, startY+(i-start)*2+1, detail, "gray-2")
			}
		}
		if len(entries) > maxRows {
			more := fmt.Sprintf("(%d/%d)", selected+1, len(entries))
			drawCentered(cells, w, h-3, more, "gray-2")
		}
	}
	panel := []string{"", "Use Item", "Enter: use", "Esc: cancel", "Up/Down: move"}
	panelFG := []string{"gray-1", "gold-bright", "gray-1", "gray-1", "gray-1"}
	for len(panel) < 12 {
		panel = append(panel, "")
		panelFG = append(panelFG, "gray-1")
	}
	status := "Use Item"
	hints := "Up/Down or k/j: move  Enter: use  Esc: cancel"
	return Frame{W: w, H: h, Cells: cells, Panel: panel, PanelFG: panelFG, Status: status, Log: make([]string, t.Layout.LogLines), Hints: hints, MinCols: t.Layout.MinCols, MinRows: t.Layout.MinRows}
}
func (g *Game) RenderThrowMenu(selected int) Frame {
	t := g.Tuning
	w, h := t.Map.Width, t.Map.Height
	cells := make([][]Cell, h)
	for y := range h {
		cells[y] = make([]Cell, w)
		for x := range w {
			cells[y][x] = Cell{Glyph: ' ', FG: "bg", BG: "bg"}
		}
	}
	entries := g.InventoryPotionEntries()
	title := "THROW POTION"
	sub := "Select potion to throw"
	if len(entries) == 0 {
		sub = "No potions to throw"
	}
	drawCentered(cells, w, 2, title, "gold-bright")
	drawCentered(cells, w, 3, sub, "gray-1")
	if len(entries) == 0 {
		drawCentered(cells, w, 5, "(empty)", "gray-2")
	} else {
		startY := 5
		if selected < 0 {
			selected = 0
		}
		if selected >= len(entries) {
			selected = len(entries) - 1
		}
		maxRows := h - 7
		if maxRows < 1 {
			maxRows = 1
		}
		start := 0
		if len(entries) > maxRows {
			start = selected - maxRows/2
			if start < 0 {
				start = 0
			}
			if start+maxRows > len(entries) {
				start = len(entries) - maxRows
			}
		}
		end := start + maxRows
		if end > len(entries) {
			end = len(entries)
		}
		for i := start; i < end; i++ {
			e := entries[i]
			fg := "gray-1"
			prefix := "  "
			if i == selected {
				prefix = "> "
				fg = "gold-bright"
			}
			var line string
			if e.Count > 1 {
				line = fmt.Sprintf("%s%s (x%d)", prefix, e.DisplayName, e.Count)
			} else {
				line = fmt.Sprintf("%s%s", prefix, e.DisplayName)
			}
			if len(line) > w-4 {
				line = line[:w-7] + "..."
			}
			drawCentered(cells, w, startY+(i-start)*2, line, fg)
			if i == selected && IsIdentified(e.Appearance) && e.DisplayName != e.Appearance {
				detail := fmt.Sprintf("(%s)", e.Appearance)
				drawCentered(cells, w, startY+(i-start)*2+1, detail, "gray-2")
			}
		}
		if len(entries) > maxRows {
			more := fmt.Sprintf("(%d/%d)", selected+1, len(entries))
			drawCentered(cells, w, h-3, more, "gray-2")
		}
	}
	panel := []string{"", "Throw Potion", "Enter: select", "Esc: cancel", "Up/Down: move"}
	panelFG := []string{"gray-1", "gold-bright", "gray-1", "gray-1", "gray-1"}
	for len(panel) < 12 {
		panel = append(panel, "")
		panelFG = append(panelFG, "gray-1")
	}
	status := "Throw Potion"
	hints := "Up/Down or k/j: move  Enter: throw  Esc: cancel"
	return Frame{W: w, H: h, Cells: cells, Panel: panel, PanelFG: panelFG, Status: status, Log: make([]string, t.Layout.LogLines), Hints: hints, MinCols: t.Layout.MinCols, MinRows: t.Layout.MinRows}
}


func (g *Game) RenderMerchantMenu() Frame {
	t := g.Tuning
	w, h := t.Map.Width, t.Map.Height
	cells := make([][]Cell, h)
	for y := range h {
		cells[y] = make([]Cell, w)
		for x := range w {
			cells[y][x] = Cell{Glyph: ' ', FG: "bg", BG: "bg"}
		}
	}
	title := "MERCHANT WARES"
	sub := fmt.Sprintf("Gold: %d -- choose ware to buy", g.Gold)
	if !g.Merchant.Active || len(g.Merchant.Wares) == 0 {
		sub = "No wares"
	}
	drawCentered(cells, w, 2, title, "gold-bright")
	drawCentered(cells, w, 3, sub, "gray-1")
	if !g.Merchant.Active || len(g.Merchant.Wares) == 0 {
		drawCentered(cells, w, 5, "(no wares)", "gray-2")
	} else {
		selected := g.Merchant.Selected
		if selected < 0 {
			selected = 0
		}
		if selected >= len(g.Merchant.Wares) {
			selected = len(g.Merchant.Wares) - 1
		}
		startY := 5
		for i, w2 := range g.Merchant.Wares {
			fg := "gray-1"
			prefix := "  "
			if i == selected {
				prefix = "> "
				fg = "gold-bright"
			}
			affordable := g.Gold >= w2.Price
			priceStr := fmt.Sprintf("%dg", w2.Price)
			if !affordable {
				fg = "slate"
				if i == selected {
					fg = "red-bright"
				}
				priceStr += " (need gold)"
			}
			line := fmt.Sprintf("%s%s -- %s", prefix, w2.Name, priceStr)
			if len(line) > w-4 {
				line = line[:w-7] + "..."
			}
			drawCentered(cells, w, startY+i*2, line, fg)
		}
	}
	panel := []string{"", "Merchant", fmt.Sprintf("Gold %d", g.Gold), "Enter: buy", "Esc: leave", "Up/Down: move"}
	panelFG := []string{"gray-1", "gold-bright", "gray-1", "gray-1", "gray-1", "gray-1"}
	for len(panel) < 12 {
		panel = append(panel, "")
		panelFG = append(panelFG, "gray-1")
	}
	status := "Merchant"
	hints := "Up/Down: move  Enter: buy  Esc: leave"
	return Frame{W: w, H: h, Cells: cells, Panel: panel, PanelFG: panelFG, Status: status, Log: make([]string, t.Layout.LogLines), Hints: hints, MinCols: t.Layout.MinCols, MinRows: t.Layout.MinRows}
}

func (g *Game) RenderShrineMenu() Frame {
	t := g.Tuning
	w, h := t.Map.Width, t.Map.Height
	cells := make([][]Cell, h)
	for y := range h {
		cells[y] = make([]Cell, w)
		for x := range w {
			cells[y][x] = Cell{Glyph: ' ', FG: "bg", BG: "bg"}
		}
	}
	title := "SHRINE"
	sub := "Choose a blessing -- shrine vanishes after use"
	if !g.Shrine.Active {
		sub = "No shrine"
	}
	drawCentered(cells, w, 2, title, "gold-bright")
	drawCentered(cells, w, 3, sub, "gray-1")
	if !g.Shrine.Active {
		drawCentered(cells, w, 5, "(no shrine)", "gray-2")
	} else {
		selected := g.Shrine.Selected
		if selected < 0 {
			selected = 0
		}
		if selected > 3 {
			selected = 3
		}
		opts := []struct {
			Name string
			Desc string
		}{
			{"Add new party member", "Recruit random outsider at current level"},
			{"Resurrect fallen member", "Restore most recent fallen"},
			{"Gain level (XP unchanged)", "Level+1, XP same, talent pick"},
			{"Leave shrine intact", "Step away, shrine remains"},
		}
		hasDead := false
		for _, m := range g.Party.Members {
			if !m.IsAlive() {
				hasDead = true
				break
			}
		}
		canAdd := len(g.Party.Members) < 4
		canLevel := g.LevelUpPending == nil
		startY := 5
		for i, opt := range opts {
			fg := "gray-1"
			prefix := "  "
			if i == selected {
				prefix = "> "
				fg = "gold-bright"
			}
			disabled := false
			switch i {
			case 0:
				if !canAdd {
					disabled = true
				}
			case 1:
				if !hasDead {
					disabled = true
				}
			case 2:
				if !canLevel {
					disabled = true
				}
			}
			if disabled {
				fg = "slate"
				if i == selected {
					fg = "red-bright"
				}
			}
			line := prefix + opt.Name
			if len(line) > w-4 {
				line = line[:w-7] + "..."
			}
			drawCentered(cells, w, startY+i*2, line, fg)
			if i == selected {
				desc := opt.Desc
				if disabled {
					switch i {
					case 0:
						desc = "Party full (4)"
					case 1:
						desc = "No fallen pilgrims"
					case 2:
						desc = "Level up already pending"
					}
				}
				if len(desc) > w-4 {
					desc = desc[:w-7] + "..."
				}
				drawCentered(cells, w, startY+i*2+1, desc, "gray-2")
			}
		}
	}
	panel := []string{"", "Shrine", "Enter: choose", "Esc: leave", "Up/Down: move"}
	panelFG := []string{"gray-1", "gold-bright", "gray-1", "gray-1", "gray-1"}
	for len(panel) < 12 {
		panel = append(panel, "")
		panelFG = append(panelFG, "gray-1")
	}
	status := "Shrine"
	hints := "Up/Down: move  Enter: choose  Esc: leave"
	return Frame{W: w, H: h, Cells: cells, Panel: panel, PanelFG: panelFG, Status: status, Log: make([]string, t.Layout.LogLines), Hints: hints, MinCols: t.Layout.MinCols, MinRows: t.Layout.MinRows}
}

func (g *Game) RenderWizardMenu(tuning Tuning, selected int) Frame {
	w, h := tuning.Map.Width, tuning.Map.Height
	cells := make([][]Cell, h)
	for y := range h {
		cells[y] = make([]Cell, w)
		for x := range w {
			cells[y][x] = Cell{Glyph: ' ', FG: "bg", BG: "bg"}
		}
	}
	title := "WIZARD MODE"
	sub := "Cheats disable scoring"
	drawCentered(cells, w, 2, title, "gold-bright")
	drawCentered(cells, w, 3, sub, "gray-1")
	startY := 5
	for i, opt := range WizardOptions {
		fg := "gray-1"
		prefix := "  "
		if i == selected {
			prefix = "> "
			fg = "gold-bright"
		}
		line := prefix + opt.Name
		if len(line) > w-4 {
			line = line[:w-7] + "..."
		}
		drawCentered(cells, w, startY+i*2, line, fg)
		if i == selected {
			desc := opt.Desc
			if len(desc) > w-4 {
				desc = desc[:w-7] + "..."
			}
			drawCentered(cells, w, startY+i*2+1, desc, "gray-2")
		}
	}
	panel := []string{"", "Wizard Mode", "Scores disabled", "when active"}
	panelFG := []string{"gray-1", "gold-bright", "gray-1", "gray-1"}
	for len(panel) < 12 {
		panel = append(panel, "")
		panelFG = append(panelFG, "gray-1")
	}
	status := "Wizard Mode"
	hints := "Up/Down: move Enter: select Esc: back"
	return Frame{W: w, H: h, Cells: cells, Panel: panel, PanelFG: panelFG, Status: status, Log: make([]string, tuning.Layout.LogLines), Hints: hints, MinCols: tuning.Layout.MinCols, MinRows: tuning.Layout.MinRows}
}
func (g *Game) RenderHelpOverlay() Frame {
	t := g.Tuning
	w, h := t.Map.Width, t.Map.Height
	cells := make([][]Cell, h)
	for y := range h {
		cells[y] = make([]Cell, w)
		for x := range w {
			cells[y][x] = Cell{Glyph: ' ', FG: "bg", BG: "bg"}
		}
	}
	lines := []struct{ text, fg string }{
		{"PILGRIM'S TEMPLE - HELP", "gold-bright"},
		{"", "bg"},
		{"q / w / e / r  - select member 1-4 (free)", "gray-1"},
		{"Move: arrows, numpad 1-9, hjkl + y u b n", "gray-1"},
		{"5 / . / Space  - wait 1 turn", "gray-1"},
		{"z / Z  - rest: 10-turn batch, 15 HP, ends on hostile/hunger", "gray-1"},
		{"g  - contextual use: pickup, or on fountain/merchant/forge (press g)", "gray-1"},
		{"u/U - use menu (potions/scrolls); on forge also g/u", "gray-1"},
		{"t  - throw potion (menu + cursor)", "gray-1"},
		{"v  - look: move cursor, v/Enter/Esc to examine", "gray-1"},
		{"> / <  - stairs down / up", "gray-1"},
		{"?  - help (this overlay)", "gold"},
		{"Esc - quit to menu   ] - wizard menu", "gray-1"},
		{"", "bg"},
		{"Rest: z heals each living member 15 HP over 10 turns;", "gray-2"},
		{"world advances; ends early if foe appears or hunger ticks.", "gray-2"},
		{"", "bg"},
		{fmt.Sprintf("Seed %d | Food %d (%s) | Lvl %d XP %d/%d | Floor %d/%d | Turn %d", g.Seed, g.Food, g.HungerState(), g.Level, g.XP, g.XPToNext, g.Floor+1, t.Floors, g.Turn), "gold"},
	}
	startY := (h - len(lines)) / 2
	if startY < 1 {
		startY = 1
	}
	for i, ln := range lines {
		if ln.text == "" {
			continue
		}
		y := startY + i
		if y >= h-1 {
			break
		}
		drawCentered(cells, w, y, ln.text, ln.fg)
	}
	// Draw border box around lines (optional subtle frame)
	status := fmt.Sprintf("Help | Seed %d | Floor %d/%d", g.Seed, g.Floor+1, t.Floors)
	hints := "Esc / Enter / ? : close help  (no turn consumed)"
	panel := []string{"", "Commands", "q/w/e/r select", "arrows/hjkl move", "5/. wait  z rest", "g contextual (pickup/fountain/merchant/forge)", "u use(menu)  t throw  v look  >/< stairs", "? help  Esc quit", "] wizard"}
	panelFG := []string{"gray-1", "gold-bright", "gray-1", "gray-1", "gray-1", "gray-1", "gray-1", "gray-1"}
	for len(panel) < 12 {
		panel = append(panel, "")
		panelFG = append(panelFG, "gray-1")
	}
	return Frame{W: w, H: h, Cells: cells, Panel: panel, PanelFG: panelFG, Status: status, Log: make([]string, t.Layout.LogLines), Hints: hints, MinCols: t.Layout.MinCols, MinRows: t.Layout.MinRows}
}

// RenderHelpOverlayTuning is a standalone variant used by frontends that hold only tuning (e.g. menu help).
func RenderHelpOverlay(tuning Tuning) Frame {
	// Build a minimal game stub so the overlay can still show tuning info.
	g := &Game{Tuning: tuning, Seed: 0, Food: tuning.Food.StartClock, Level: 1, XPToNext: 100, Floor: 0}
	if g.Food == 0 {
		g.Food = 2000
	}
	return g.RenderHelpOverlay()
}

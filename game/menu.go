package game

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
)

type ClassInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	BuffA struct {
		Name string `json:"name"`
		Desc string `json:"desc"`
	} `json:"buffA"`
}

func LoadClasses() ([]ClassInfo, error) {
	b, err := RawJSON("classes.json")
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Classes []ClassInfo `json:"classes"`
	}
	if err := json.Unmarshal(b, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Classes, nil
}

var MainMenuOptions = []string{"New Game", "Scores", "Exit"}

type MainMenuState struct {
	Selected int
}

func (m *MainMenuState) Move(dir int) {
	n := len(MainMenuOptions)
	m.Selected = (m.Selected + dir + n) % n
}

type CharSelectState struct {
	Classes []ClassInfo
	Cursor  int
	Picks   []string
}

func NewCharSelect() (*CharSelectState, error) {
	cls, err := LoadClasses()
	if err != nil {
		return nil, err
	}
	return &CharSelectState{Classes: cls, Cursor: 0, Picks: []string{}}, nil
}

func (cs *CharSelectState) Move(dir int) {
	n := len(cs.Classes)
	if n == 0 {
		return
	}
	cs.Cursor = (cs.Cursor + dir + n) % n
}

func (cs *CharSelectState) Select() {
	if len(cs.Picks) >= 2 {
		return
	}
	id := cs.Classes[cs.Cursor].ID
	for _, p := range cs.Picks {
		if p == id {
			return
		}
	}
	cs.Picks = append(cs.Picks, id)
}

func (cs *CharSelectState) Back() bool {
	if len(cs.Picks) > 0 {
		cs.Picks = cs.Picks[:len(cs.Picks)-1]
		return false
	}
	return true
}

func (cs *CharSelectState) Done() bool { return len(cs.Picks) == 2 }

// RaceSelectState holds race picks for each class slot.
type RaceSelectState struct {
	Classes []string
	Races   []Race
	Cursor  int
	Picks   []string
}

func NewRaceSelect(classes []string) (*RaceSelectState, error) {
	races := LoadRaces()
	if len(races) == 0 {
		races = fallbackRaces()
	}
	cp := make([]string, len(classes))
	copy(cp, classes)
	return &RaceSelectState{Classes: cp, Races: races, Cursor: 0, Picks: []string{}}, nil
}

func (rs *RaceSelectState) Move(dir int) {
	n := len(rs.Races)
	if n == 0 {
		return
	}
	rs.Cursor = (rs.Cursor + dir + n) % n
}

func (rs *RaceSelectState) Select() {
	if len(rs.Picks) >= len(rs.Classes) {
		return
	}
	if len(rs.Races) == 0 {
		return
	}
	if rs.Cursor < 0 || rs.Cursor >= len(rs.Races) {
		return
	}
	id := rs.Races[rs.Cursor].ID
	rs.Picks = append(rs.Picks, id)
}

func (rs *RaceSelectState) Back() bool {
	if len(rs.Picks) > 0 {
		rs.Picks = rs.Picks[:len(rs.Picks)-1]
		return false
	}
	return true
}

func (rs *RaceSelectState) Done() bool { return len(rs.Classes) > 0 && len(rs.Picks) == len(rs.Classes) }

func NewGameWithClasses(seed int64, tuning Tuning, classes []string) *Game {
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
	final := g.Levels[tuning.Floors-1]
	g.Relic = final.StairsDown
	g.Party = GeneratePartyWithClasses(rng, classes, 1)
	start := g.Levels[0].StairsUp
	g.Party.Pos = start
	g.Floor = 0
	g.VisitedFloors[0] = true
	g.TransitionFiredForLevel[0] = true
	g.Logf("Seed %d -- Pilgrim's Temple, %d floors.", seed, tuning.Floors)
	names := ""
	for i, m := range g.Party.Members {
		if i > 0 {
			names += ", "
		}
		names += fmt.Sprintf("%s (%s)", m.Name, m.Class)
	}
	g.Logf("Party: %s", names)
	g.Logf("You stand at the temple threshold.")
	g.UpdateFOV()
	return g
}

func NewGameWithClassesAndRaces(seed int64, tuning Tuning, classes []string, races []string) *Game {
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
	final := g.Levels[tuning.Floors-1]
	g.Relic = final.StairsDown
	g.Party = GeneratePartyWithClassesAndRaces(rng, classes, races, 1)
	start := g.Levels[0].StairsUp
	g.Party.Pos = start
	g.Floor = 0
	g.VisitedFloors[0] = true
	g.TransitionFiredForLevel[0] = true
	g.Logf("Seed %d -- Pilgrim's Temple, %d floors.", seed, tuning.Floors)
	names := ""
	for i, m := range g.Party.Members {
		if i > 0 {
			names += ", "
		}
		raceName := ""
		if m.Race != "" {
			if r, ok := GetRace(m.Race); ok {
				raceName = r.Name + " "
			} else {
				raceName = FriendlyID(m.Race) + " "
			}
		}
		names += fmt.Sprintf("%s (%s%s)", m.Name, raceName, FriendlyID(m.Class))
	}
	g.Logf("Party: %s", names)
	g.Logf("You stand at the temple threshold.")
	g.UpdateFOV()
	return g
}

func RenderMainMenu(tuning Tuning, selected int) Frame {
	w, h := tuning.Map.Width, tuning.Map.Height
	cells := make([][]Cell, h)
	for y := range h {
		cells[y] = make([]Cell, w)
		for x := range w {
			cells[y][x] = Cell{Glyph: ' ', FG: "bg", BG: "bg"}
		}
	}
	title := "PILGRIM'S TEMPLE"
	sub := "Roguetemple's Fortnight 2"
	drawCentered(cells, w, h/2-4, title, "gold-bright")
	drawCentered(cells, w, h/2-3, sub, "gray-1")
	for i, opt := range MainMenuOptions {
		prefix := "  "
		fg := "gray-1"
		if i == selected {
			prefix = "> "
			fg = "gold-bright"
		}
		line := prefix + opt
		drawCentered(cells, w, h/2+1+i, line, fg)
	}
	panel := []string{"", "Pilgrim's Temple", "Select: Up/Down  Enter", "Esc: Quit"}
	for len(panel) < 12 {
		panel = append(panel, "")
	}
	status := "Main Menu"
	hints := "Up/Down or k/j: move  Enter: select  Esc: quit"
	return Frame{W: w, H: h, Cells: cells, Panel: panel, Status: status, Log: make([]string, tuning.Layout.LogLines), Hints: hints, MinCols: tuning.Layout.MinCols, MinRows: tuning.Layout.MinRows}
}
// RenderMainMenuWithScores loads the Scoreboard via LoadScoreboard (handles missing file/localStorage gracefully)
// and renders recent entries (top 5 by score descending) with columns: Rank, Score, PartyLevel, Gold, Depth, Seed, Victory/Cause, Members summary.
// Keeps existing menu options above scores and uses available map width for scoreboard footer.
// Deprecated: use RenderScoresScreen for the dedicated scores view; main menu now shows no scores.
func RenderMainMenuWithScores(tuning Tuning, selected int) Frame {
	w, h := tuning.Map.Width, tuning.Map.Height
	cells := make([][]Cell, h)
	for y := range h {
		cells[y] = make([]Cell, w)
		for x := range w {
			cells[y][x] = Cell{Glyph: ' ', FG: "bg", BG: "bg"}
		}
	}
	title := "PILGRIM'S TEMPLE"
	sub := "Roguetemple's Fortnight 2"
	drawCentered(cells, w, h/2-4, title, "gold-bright")
	drawCentered(cells, w, h/2-3, sub, "gray-1")
	for i, opt := range MainMenuOptions {
		prefix := "  "
		fg := "gray-1"
		if i == selected {
			prefix = "> "
			fg = "gold-bright"
		}
		line := prefix + opt
		drawCentered(cells, w, h/2+1+i, line, fg)
	}
	// Scoreboard footer below menu choices, using available map width.
	sb, err := LoadScoreboard()
	if err != nil || sb == nil {
		sb = &Scoreboard{}
	}
	entries := sb.GetHighScores(5)
	yStart := h/2 + 1 + len(MainMenuOptions) + 1
	if yStart < h {
		drawCentered(cells, w, yStart, "-- SCOREBOARD --", "gold")
		yStart++
	}
	if len(entries) == 0 {
		if yStart < h {
			msg := "No scores yet \u2014 survive the temple!"
			if len(msg) > w-2 {
				msg = msg[:w-5] + "..."
			}
			drawCentered(cells, w, yStart, msg, "gray-1")
		}
	} else {
		// Header
		if yStart < h {
			header := " # Score Lv Gold Depth Seed       Result     Members"
			if len(header) > w-2 {
				header = header[:w-2]
			}
			drawString(cells, 1, yStart, header, "gray-2")
			yStart++
		}
		for idx, e := range entries {
			y := yStart + idx
			if y >= h {
				break
			}
			result := e.CauseOfDeath
			if e.Victory {
				result = "Victory"
			}
			if result == "" {
				result = "Unknown"
			}
			members := MembersSummary(e)
			line := fmt.Sprintf("%2d. %5d Lv%d G%d D%d S%d %-10s %s", idx+1, e.Score, e.PartyLevel, e.Gold, e.DepthReached, e.Seed, result, members)
			if len(line) > w-2 {
				line = line[:w-5] + "..."
			}
			fg := "gray-1"
			if e.Victory {
				fg = "gold-bright"
			}
			drawString(cells, 1, y, line, fg)
		}
	}
	panel := []string{"", "Pilgrim's Temple", "Select: Up/Down  Enter", "Esc: Quit"}
	// Also show score count in panel if space.
	if len(entries) > 0 {
		panel = append(panel, fmt.Sprintf("Scores: %d", len(sb.Entries)))
	}
	for len(panel) < 12 {
		panel = append(panel, "")
	}
	status := "Main Menu"
	hints := "Up/Down or k/j: move  Enter: select  Esc: quit"
	return Frame{W: w, H: h, Cells: cells, Panel: panel, Status: status, Log: make([]string, tuning.Layout.LogLines), Hints: hints, MinCols: tuning.Layout.MinCols, MinRows: tuning.Layout.MinRows}
}

// RenderScoresScreen renders the scrolling scoreboard for the Scores menu option.
// Shows all entries sorted by score descending, scrollable via selected index.
func RenderScoresScreen(tuning Tuning, selected int) Frame {
	w, h := tuning.Map.Width, tuning.Map.Height
	cells := make([][]Cell, h)
	for y := range h {
		cells[y] = make([]Cell, w)
		for x := range w {
			cells[y][x] = Cell{Glyph: ' ', FG: "bg", BG: "bg"}
		}
	}
	sb, err := LoadScoreboard()
	if err != nil || sb == nil {
		sb = &Scoreboard{}
	}
	entries := sb.GetHighScores(len(sb.Entries))
	if len(entries) == 0 {
		entries = sb.Entries
	}
	drawCentered(cells, w, 2, "SCOREBOARD", "gold-bright")
	if len(entries) == 0 {
		drawCentered(cells, w, h/2, "No scores yet — survive the temple!", "gray-1")
	} else {
		if selected < 0 {
			selected = 0
		}
		if selected >= len(entries) {
			selected = len(entries) - 1
		}
		header := " # Score Lv Gold Depth Seed       Result     Members"
		if len(header) > w-2 {
			header = header[:w-2]
		}
		drawString(cells, 1, 4, header, "gray-2")
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
			result := e.CauseOfDeath
			if e.Victory {
				result = "Victory"
			}
			if result == "" {
				result = "Unknown"
			}
			members := MembersSummary(e)
			line := fmt.Sprintf("%2d. %5d Lv%d G%d D%d S%d %-10s %s", i+1, e.Score, e.PartyLevel, e.Gold, e.DepthReached, e.Seed, result, members)
			if len(line) > w-2 {
				line = line[:w-5] + "..."
			}
			fg := "gray-1"
			if e.Victory {
				fg = "gold-bright"
			}
			if i == selected {
				fg = "gold-bright"
				line = "> " + line[2:]
			}
			drawString(cells, 1, 5+(i-start), line, fg)
		}
		if len(entries) > maxRows {
			more := fmt.Sprintf("(%d/%d)", selected+1, len(entries))
			drawCentered(cells, w, h-3, more, "gray-2")
		}
	}
	panel := []string{"", "Scores", fmt.Sprintf("%d entries", len(entries)), "Enter/Esc: back", "Up/Down: scroll"}
	for len(panel) < 12 {
		panel = append(panel, "")
	}
	status := "Scores"
	hints := "Up/Down or k/j: scroll  Enter/Esc: back to menu"
	return Frame{W: w, H: h, Cells: cells, Panel: panel, Status: status, Log: make([]string, tuning.Layout.LogLines), Hints: hints, MinCols: tuning.Layout.MinCols, MinRows: tuning.Layout.MinRows}
}

func RenderCharSelect(tuning Tuning, cs *CharSelectState) Frame {
	w, h := tuning.Map.Width, tuning.Map.Height
	cells := make([][]Cell, h)
	for y := range h {
		cells[y] = make([]Cell, w)
		for x := range w {
			cells[y][x] = Cell{Glyph: ' ', FG: "bg", BG: "bg"}
		}
	}
	title := "CHOOSE TWO PILGRIMS"
	sub := fmt.Sprintf("Pick %d/2", len(cs.Picks))
	if len(cs.Picks) == 2 {
		sub = "Press Enter to begin"
	}
	drawCentered(cells, w, 2, title, "gold-bright")
	drawCentered(cells, w, 3, sub, "gray-1")
	pickLine := ""
	for i, p := range cs.Picks {
		if i > 0 {
			pickLine += " + "
		}
		pickLine += strings.Title(p)
	}
	if pickLine == "" {
		pickLine = "(none)"
	}
	drawCentered(cells, w, 5, pickLine, "gold")
	for i, ci := range cs.Classes {
		y := 7 + i*2
		if y+1 >= h-1 {
			break
		}
		prefix := "  "
		fg := "gray-1"
		if i == cs.Cursor {
			prefix = "> "
			fg = "gold-bright"
		}
		chosen := ""
		for _, p := range cs.Picks {
			if p == ci.ID {
				chosen = " [x]"
				if i != cs.Cursor {
					fg = "gray-2"
				}
				break
			}
		}
		line := fmt.Sprintf("%s%s", prefix, strings.Title(ci.Name))
		if chosen != "" {
			line += chosen
		}
		line += fmt.Sprintf(" - %s", ci.BuffA.Name)
		if len(line) > w-2 {
			line = line[:w-5] + "..."
		}
		drawString(cells, 2, y, line, fg)
		drawString(cells, 4, y+1, ci.Role, "gray-2")
	}
	panel := []string{"", "Choose 2", fmt.Sprintf("Picked: %d/2", len(cs.Picks))}
	for len(panel) < 12 {
		panel = append(panel, "")
	}
	panel = append(panel, "Enter: pick", "Esc: back")
	status := fmt.Sprintf("Choose pilgrims %d/2", len(cs.Picks))
	hints := "Up/Down: move  Enter: pick  Esc: back  (need 2 to start)"
	if len(cs.Picks) == 2 {
		hints = "Enter: begin  Esc: back"
	}
	return Frame{W: w, H: h, Cells: cells, Panel: panel, Status: status, Log: make([]string, tuning.Layout.LogLines), Hints: hints, MinCols: tuning.Layout.MinCols, MinRows: tuning.Layout.MinRows}
}
func buffSummary(b Buff) string {
	var parts []string
	if b.HP != 0 {
		parts = append(parts, fmt.Sprintf("%+d HP", b.HP))
	}
	if b.ATK != 0 {
		parts = append(parts, fmt.Sprintf("%+d ATK", b.ATK))
	}
	if b.DEF != 0 {
		parts = append(parts, fmt.Sprintf("%+d DEF", b.DEF))
	}
	if b.MDEF != 0 {
		parts = append(parts, fmt.Sprintf("%+d MDEF", b.MDEF))
	}
	if b.Light != 0 {
		parts = append(parts, fmt.Sprintf("%+d Light", b.Light))
	}
	if b.Carry != 0 {
		parts = append(parts, fmt.Sprintf("%+d Carry", b.Carry))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}

func RenderRaceSelect(tuning Tuning, rs *RaceSelectState) Frame {
	w, h := tuning.Map.Width, tuning.Map.Height
	cells := make([][]Cell, h)
	for y := range h {
		cells[y] = make([]Cell, w)
		for x := range w {
			cells[y][x] = Cell{Glyph: ' ', FG: "bg", BG: "bg"}
		}
	}
	total := len(rs.Classes)
	picked := len(rs.Picks)
	title := "CHOOSE RACES"
	var sub string
	if rs.Done() {
		sub = "Press Enter to begin"
	} else if picked < total {
		className := FriendlyID(rs.Classes[picked])
		sub = fmt.Sprintf("Race for %s (%d/%d)", className, picked+1, total)
	} else {
		sub = fmt.Sprintf("Pick %d/%d", picked, total)
	}
	drawCentered(cells, w, 2, title, "gold-bright")
	drawCentered(cells, w, 3, sub, "gray-1")
	pickLine := ""
	for i := range total {
		if i > 0 {
			pickLine += " + "
		}
		className := FriendlyID(rs.Classes[i])
		if i < len(rs.Picks) {
			raceName := FriendlyID(rs.Picks[i])
			if r, ok := GetRace(rs.Picks[i]); ok {
				raceName = r.Name
			}
			pickLine += fmt.Sprintf("%s %s", raceName, className)
		} else if i == picked {
			pickLine += fmt.Sprintf("? %s", className)
		} else {
			pickLine += fmt.Sprintf("? %s", className)
		}
	}
	if pickLine == "" {
		pickLine = "(none)"
	}
	if len(pickLine) > w-2 {
		pickLine = pickLine[:w-5] + "..."
	}
	drawCentered(cells, w, 5, pickLine, "gold")
	for i, r := range rs.Races {
		y := 7 + i*2
		if y+1 >= h-1 {
			break
		}
		prefix := "  "
		fg := "gray-1"
		if i == rs.Cursor {
			prefix = "> "
			fg = "gold-bright"
		}
		line := fmt.Sprintf("%s%s", prefix, r.Name)
		if s := buffSummary(r.CharBuff); s != "" {
			line += fmt.Sprintf(" [%s]", s)
		}
		if r.PartyBuff.Light != 0 || r.PartyBuff.HP != 0 || r.PartyBuff.ATK != 0 || r.PartyBuff.DEF != 0 {
			if ps := buffSummary(r.PartyBuff); ps != "" {
				line += fmt.Sprintf(" Party:%s", ps)
			}
		}
		if r.SynergyBuff.Desc != "" {
			line += fmt.Sprintf(" Syn:%s", r.SynergyBuff.Desc)
		}
		if len(line) > w-2 {
			line = line[:w-5] + "..."
		}
		drawString(cells, 2, y, line, fg)
		desc := r.Desc
		if len(desc) > w-6 {
			desc = desc[:w-9] + "..."
		}
		drawString(cells, 4, y+1, desc, "gray-2")
	}
	panel := []string{"", "Choose race", fmt.Sprintf("Picked: %d/%d", picked, total)}
	for len(panel) < 12 {
		panel = append(panel, "")
	}
	panel = append(panel, "Enter: pick", "Esc: back", "1-7: quick")
	status := fmt.Sprintf("Choose race %d/%d", picked, total)
	if rs.Done() {
		status = "Races chosen - Enter to begin"
	}
	hints := "Up/Down: move  Enter: pick  Esc: back  1-7: quick pick"
	if rs.Done() {
		hints = "Enter: begin  Esc: back"
	}
	return Frame{W: w, H: h, Cells: cells, Panel: panel, Status: status, Log: make([]string, tuning.Layout.LogLines), Hints: hints, MinCols: tuning.Layout.MinCols, MinRows: tuning.Layout.MinRows}
}

func drawCentered(cells [][]Cell, w, y int, s string, fg string) {
	if y < 0 || y >= len(cells) {
		return
	}
	runes := []rune(s)
	x := (w - len(runes)) / 2
	if x < 0 {
		x = 0
	}
	drawString(cells, x, y, s, fg)
}

func drawString(cells [][]Cell, x, y int, s string, fg string) {
	if y < 0 || y >= len(cells) {
		return
	}
	for i, ch := range s {
		if x+i < 0 || x+i >= len(cells[y]) {
			continue
		}
		cells[y][x+i] = Cell{Glyph: ch, FG: fg, BG: "bg"}
	}
}

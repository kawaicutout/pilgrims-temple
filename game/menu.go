package game

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
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

// GetMainMenuOptions returns menu entries, adding "Load game" when a save exists.
func GetMainMenuOptions() []string {
	if HasSave() {
		return []string{"New Game", "Scores", "Exit", "Load game"}
	}
	return MainMenuOptions
}

type MainMenuState struct {
	Selected int
}

func (m *MainMenuState) Move(dir int) {
	opts := GetMainMenuOptions()
	n := len(opts)
	if n == 0 {
		return
	}
	m.Selected = (m.Selected + dir + n) % n
	// Clamp in case HasSave changed between moves.
	if m.Selected >= n {
		m.Selected = n - 1
	}
	if m.Selected < 0 {
		m.Selected = 0
	}
}
// ---------------------------------------------------------------------------
// Animated title banner (game/data/title.txt, two stacked halves).
// ---------------------------------------------------------------------------

var (
	titleOnce         sync.Once
	titleTop          []string
	titleBottom       []string
	titleOK           bool
)

// loadTitleArt reads the wide banner and splits it into two stacked halves
// along the widest all-blank column band, so letterforms stay intact.
func loadTitleArt() (top, bottom []string, ok bool) {
	titleOnce.Do(func() {
		b, err := dataFS.ReadFile("data/title.txt")
		if err != nil {
			return
		}
		lines := strings.Split(string(b), "\n")
		var rows []string
		for _, ln := range lines {
			ln = strings.TrimRight(ln, " ")
			if ln == "" {
				continue
			}
			rows = append(rows, ln)
		}
		if len(rows) == 0 {
			return
		}
		top, bottom, ok = splitTitleArt(rows)
		if ok {
			titleTop, titleBottom, titleOK = top, bottom, true
		}
	})
	return titleTop, titleBottom, titleOK
}

// splitTitleArt cuts rows along the widest all-blank column band of width 2+,
// keeping original column alignment inside each half.
func splitTitleArt(rows []string) (top, bottom []string, ok bool) {
	w := 0
	for _, ln := range rows {
		if n := len([]rune(ln)); n > w {
			w = n
		}
	}
	if w == 0 {
		return nil, nil, false
	}
	grid := make([][]rune, len(rows))
	for i, ln := range rows {
		r := []rune(ln)
		for len(r) < w {
			r = append(r, ' ')
		}
		grid[i] = r
	}
	bestStart, bestEnd := -1, -1
	for c := 0; c < w; {
		if gridBlankCol(grid, c) {
			s := c
			for c < w && gridBlankCol(grid, c) {
				c++
			}
			if c-s >= 2 && c-s > bestEnd-bestStart {
				bestStart, bestEnd = s, c-1
			}
		} else {
			c++
		}
	}
	if bestStart < 0 {
		return nil, nil, false
	}
	for _, r := range grid {
		top = append(top, string(r[:bestStart]))
		bottom = append(bottom, string(r[bestEnd+1:]))
	}
	return top, bottom, true
}

func gridBlankCol(grid [][]rune, c int) bool {
	for _, r := range grid {
		if c < len(r) && r[c] != ' ' {
			return false
		}
	}
	return true
}

func titleWidth(lines []string) int {
	w := 0
	for _, ln := range lines {
		if n := len([]rune(ln)); n > w {
			w = n
		}
	}
	return w
}

// titleBevelColor chisels the banner: lit yellow upper-left edges, deep red
// lower-right edges, brown/gold faces. Static — no phase. Hex colors render
// on both builds (terminal parses #.., web passes through).
func titleBevelColor(ch rune, lit, dark bool) string {
	if lit {
		if ch == '█' {
			return "#f0d080"
		}
		return "#d3ad6b"
	}
	if dark {
		if ch == '█' {
			return "#a8564a"
		}
		return "#7a3a2a"
	}
	if ch == '█' {
		return "#b8975a"
	}
	return "#8a6f42"
}

func drawTitleBlock(cells [][]Cell, w, x0, y0 int, lines []string) {
	h := len(cells)
	grid := make([][]rune, len(lines))
	for i, ln := range lines {
		grid[i] = []rune(ln)
	}
	at := func(r, c int) rune {
		if r < 0 || r >= len(grid) || c < 0 || c >= len(grid[r]) {
			return ' '
		}
		return grid[r][c]
	}
	for r, ln := range lines {
		y := y0 + r
		if y < 0 || y >= h {
			continue
		}
		col := 0
		for _, ch := range ln {
			x := x0 + col
			lit := at(r-1, col) == ' ' || at(r, col-1) == ' '
			dark := at(r+1, col) == ' ' || at(r, col+1) == ' '
			col++
			if ch == ' ' {
				continue
			}
			if x < 0 || x >= w {
				continue
			}
			cells[y][x] = Cell{Glyph: ch, FG: titleBevelColor(ch, lit, dark), BG: "bg"}
		}
	}
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

func RenderMainMenu(tuning Tuning, selected int) Frame {
	w, h := tuning.Layout.MinCols, tuning.Layout.MinRows
	cells := make([][]Cell, h)
	for y := range h {
		cells[y] = make([]Cell, w)
		for x := range w {
			cells[y][x] = Cell{Glyph: ' ', FG: "bg", BG: "bg"}
		}
	}
	optY := h/2 - 1
	if top, bottom, ok := loadTitleArt(); ok && len(top) > 0 && len(bottom) > 0 {
		y0 := 1
		drawTitleBlock(cells, w, (w-titleWidth(top))/2, y0, top)
		drawTitleBlock(cells, w, (w-titleWidth(bottom))/2, y0+len(top)+1, bottom)
		optY = y0 + len(top) + 1 + len(bottom) + 2
	} else {
		title := "PILGRIMS' TEMPLE"
		drawCentered(cells, w, h/2-4, title, "gold-bright")
	}
	for i, opt := range GetMainMenuOptions() {
		prefix := "  "
		fg := "gray-1"
		if i == selected {
			prefix = "> "
			fg = "gold-bright"
		}
		line := prefix + opt
		y := optY + i
		if y < 0 || y >= h-1 {
			continue
		}
		drawCentered(cells, w, y, line, fg)
	}
	if HasModifiedData() {
		drawCentered(cells, w, h-3, "MODDED — scores disabled", "red-bright")
	}
	panel := []string{}
	for len(panel) < 12 {
		panel = append(panel, "")
	}
	status := ""
	hints := ""
	return Frame{W: w, H: h, Cells: cells, Panel: panel, Status: status, Log: make([]string, tuning.Layout.LogLines), Hints: hints, MinCols: tuning.Layout.MinCols, MinRows: tuning.Layout.MinRows}
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

// newGameShell builds the run skeleton both constructors share: tuning,
// RNG, levels, relic. Callers set the party, log the roster, and finish
// with startPrologue.
func newGameShell(seed int64, tuning Tuning) (*Game, *rand.Rand) {
	SetGlobalTuning(tuning)
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
	final.Set(g.Relic, TileRelic)
	return g, rng
}

// startPrologue logs the threshold/tutorial lines and computes FOV.
func (g *Game) startPrologue() {
	g.Logf("You stand at the temple threshold.")
	// First-20-turns micro-tutorial: one-time on floor 0 Turn 0
	if g.Floor == 0 && g.Turn == 0 {
		g.Logf("Move 8/2/4/6 or arrows/hjkl, 5/. or Space to wait, q/w/e/r pick member, g pick up, ? for help.")
	}
	g.UpdateFOV()
}

func NewGameWithClasses(seed int64, tuning Tuning, classes []string) *Game {
	g, rng := newGameShell(seed, tuning)
	g.Party = GeneratePartyWithClasses(rng, classes, 1)
	start := g.Levels[0].StairsUp
	g.Party.Pos = start
	g.Floor = 0
	g.VisitedFloors[0] = true
	g.TransitionFiredForLevel[0] = true
	g.Logf("Seed %d -- Pilgrims' Temple, %d floors.", seed, tuning.Floors)
	names := ""
	for i, m := range g.Party.Members {
		if i > 0 {
			names += ", "
		}
		names += fmt.Sprintf("%s (%s)", m.Name, m.Class)
	}
	g.startPrologue()
	return g
}

func NewGameWithClassesAndRaces(seed int64, tuning Tuning, classes []string, races []string) *Game {
	g, rng := newGameShell(seed, tuning)
	g.Party = GeneratePartyWithClassesAndRaces(rng, classes, races, 1)
	start := g.Levels[0].StairsUp
	g.Party.Pos = start
	g.Floor = 0
	g.VisitedFloors[0] = true
	g.TransitionFiredForLevel[0] = true
	g.Logf("Seed %d -- Pilgrims' Temple, %d floors.", seed, tuning.Floors)
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
	g.startPrologue()
	return g
}

// RenderScoresScreen renders the scrolling scoreboard for the Scores menu option.
// Shows all entries sorted by score descending, scrollable via selected index.
func RenderScoresScreen(tuning Tuning, selected int) Frame {
	w, h := tuning.Layout.MinCols, tuning.Layout.MinRows
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
		// 4 lines per entry: reserve h-6 rows for entries (title at 2, blank at 3).
		maxRows := h - 6
		if maxRows < 4 {
			maxRows = 4
		}
		perPage := maxRows / 4
		if perPage < 1 {
			perPage = 1
		}
		start := 0
		if len(entries) > perPage {
			start = selected - perPage/2
			if start < 0 {
				start = 0
			}
			if start+perPage > len(entries) {
				start = len(entries) - perPage
			}
		}
		end := start + perPage
		if end > len(entries) {
			end = len(entries)
		}
		for i := start; i < end; i++ {
			e := entries[i]
			cause := e.CauseOfDeath
			if e.Victory {
				cause = "Victory"
			}
			if cause == "" {
				cause = "Unknown"
			}
			line1 := fmt.Sprintf("Run %d | Score %d | Lv %d | G%d | D%d", i+1, e.Score, e.PartyLevel, e.Gold, e.DepthReached)
			if len(line1) > w-2 {
				line1 = line1[:w-5] + "..."
			}
			line2 := fmt.Sprintf("Seed %d | %s", e.Seed, cause)
			if len(line2) > w-2 {
				line2 = line2[:w-5] + "..."
			}
			memberStr := func(idx int) string {
				if idx < 0 || idx >= len(e.Members) {
					return ""
				}
				m := e.Members[idx]
				cls := FriendlyID(m.Class)
				race := FriendlyID(m.Race)
				var base string
				if race != "" && cls != "" {
					base = race + " " + cls
				} else if race != "" {
					base = race
				} else if cls != "" {
					base = cls
				} else if m.Name != "" {
					base = m.Name
				} else {
					base = "Unknown"
				}
				if m.Name != "" {
					if base == m.Name {
						return base
					}
					return fmt.Sprintf("%s, %s", m.Name, base)
				}
				return base
			}
			s0 := memberStr(0)
			s1 := memberStr(1)
			s2 := memberStr(2)
			s3 := memberStr(3)
			line3 := ""
			if s0 != "" && s1 != "" {
				line3 = fmt.Sprintf("%s | %s", s0, s1)
			} else if s0 != "" {
				line3 = s0
			} else if s1 != "" {
				line3 = s1
			}
			if len(line3) > w-2 {
				line3 = line3[:w-5] + "..."
			}
			line4 := ""
			if s2 != "" && s3 != "" {
				line4 = fmt.Sprintf("%s | %s", s2, s3)
			} else if s2 != "" {
				line4 = s2
			} else if s3 != "" {
				line4 = s3
			}
			if len(line4) > w-2 {
				line4 = line4[:w-5] + "..."
			}
			fg := "gray-1"
			if e.Victory {
				fg = "gold-bright"
			}
			y := 4 + (i-start)*4
			if y+3 >= h-2 {
				break
			}
			if i == selected {
				fg = "gold-bright"
				line1 = "> " + line1
				if len(line1) > w-2 {
					line1 = line1[:w-5] + "..."
				}
			}
			drawString(cells, 1, y, line1, fg)
			drawString(cells, 1, y+1, line2, fg)
			if line3 != "" {
				drawString(cells, 1, y+2, line3, fg)
			}
			if line4 != "" {
				drawString(cells, 1, y+3, line4, fg)
			}
		}
	}
	panel := []string{}
	for len(panel) < 12 {
		panel = append(panel, "")
	}
	status := ""
	hints := ""
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

// RacePickState picks one race for a roster slot. Single Enter confirms.
type RacePickState struct {
	Races  []Race
	Slot   int
	Drafts int
	Cursor int
}

// NewRacePickState loads races for the given 0-based slot.
func NewRacePickState(slot, drafts int) *RacePickState {
	races := LoadRaces()
	if len(races) == 0 {
		races = fallbackRaces()
	}
	return &RacePickState{Races: races, Slot: slot, Drafts: drafts}
}

// Move steps the cursor with wraparound.
func (s *RacePickState) Move(dir int) {
	if s == nil || len(s.Races) == 0 {
		return
	}
	s.Cursor = (s.Cursor + dir + len(s.Races)) % len(s.Races)
}

// Choice returns the cursor race id, or "" when empty.
func (s *RacePickState) Choice() string {
	if s == nil || len(s.Races) == 0 {
		return ""
	}
	if s.Cursor < 0 || s.Cursor >= len(s.Races) {
		return ""
	}
	return s.Races[s.Cursor].ID
}

// RenderRacePick draws one race slot of the pipeline.
func RenderRacePick(tuning Tuning, s *RacePickState) Frame {
	slot, drafts := 1, 0
	if s != nil {
		slot, drafts = s.Slot+1, s.Drafts
	}
	w, h, cells := renderCreationChrome(tuning, "CHOOSE RACE", fmt.Sprintf("Pilgrim %d (roster %d/3)", slot, drafts))
	if s != nil {
		for i, r := range s.Races {
			y := 5 + i*2
			if y+1 >= h-1 {
				break
			}
			prefix := "  "
			fg := "gray-1"
			if i == s.Cursor {
				prefix = "> "
				fg = "gold-bright"
			}
			line := fmt.Sprintf("%s%s", prefix, r.Name)
			if bs := buffSummary(r.CharBuff); bs != "" {
				line += fmt.Sprintf(" [%s]", bs)
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
	}
	panel, panelFG, status, hints := creationPanel("Race", "Up/Down: move  Enter: pick  Esc: back", tuning)
	return Frame{W: w, H: h, Cells: cells, Panel: panel, PanelFG: panelFG, Status: status, Log: make([]string, tuning.Layout.LogLines), Hints: hints, MinCols: tuning.Layout.MinCols, MinRows: tuning.Layout.MinRows}
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

// ---------------------------------------------------------------------------
// Guided creation (Option A): Seed -> per slot Race -> Class -> Name -> Review.
// ---------------------------------------------------------------------------

// DraftMember is one roster draft: class and race chosen, name raw.
// Blank name means random at start.
type DraftMember struct {
	Class string
	Race  string
	Name  string
}

// SeedEntryState holds raw seed text. Blank means random at start.
type SeedEntryState struct {
	Text string
}

// AppendRune adds a seed digit (or leading minus). Max 20 runes.
func (s *SeedEntryState) AppendRune(r rune) {
	if r < 32 || r == 127 {
		return
	}
	if len([]rune(s.Text)) >= 20 {
		return
	}
	if (r < '0' || r > '9') && !(r == '-' && s.Text == "") {
		return
	}
	s.Text += string(r)
}

// Backspace drops the last seed rune.
func (s *SeedEntryState) Backspace() {
	rs := []rune(s.Text)
	if len(rs) > 0 {
		s.Text = string(rs[:len(rs)-1])
	}
}

// SeedOr resolves the entry, falling back when blank or invalid.
func (s *SeedEntryState) SeedOr(fallback int64) int64 {
	if s == nil {
		return fallback
	}
	if v, err := strconv.ParseInt(strings.TrimSpace(s.Text), 10, 64); err == nil {
		return v
	}
	return fallback
}

// ClassPickState picks one class for a roster slot. Duplicates allowed.
type ClassPickState struct {
	Classes []ClassInfo
	Slot    int
	Drafts  int
	Cursor  int
}

// NewClassPickState loads classes for the given 0-based slot.
func NewClassPickState(slot, drafts int) (*ClassPickState, error) {
	cls, err := LoadClasses()
	if err != nil {
		return nil, err
	}
	return &ClassPickState{Classes: cls, Slot: slot, Drafts: drafts}, nil
}

// Move steps the cursor with wraparound.
func (s *ClassPickState) Move(dir int) {
	if s == nil || len(s.Classes) == 0 {
		return
	}
	s.Cursor = (s.Cursor + dir + len(s.Classes)) % len(s.Classes)
}

// Choice returns the cursor class id, or "" when empty.
func (s *ClassPickState) Choice() string {
	if s == nil || len(s.Classes) == 0 {
		return ""
	}
	if s.Cursor < 0 || s.Cursor >= len(s.Classes) {
		return ""
	}
	return s.Classes[s.Cursor].ID
}

// NameEntryState holds raw name text for one draft. Blank means random.
type NameEntryState struct {
	Class string
	Race  string
	Text  string
}

// AppendRune adds a name rune. Max 12 content runes.
func (s *NameEntryState) AppendRune(r rune) {
	if r < 32 || r == 127 {
		return
	}
	if len([]rune(strings.TrimSpace(s.Text))) >= 12 && r != ' ' {
		return
	}
	if isNameRune(r) {
		s.Text += string(r)
	}
}

// Backspace drops the last name rune.
func (s *NameEntryState) Backspace() {
	rs := []rune(s.Text)
	if len(rs) > 0 {
		s.Text = string(rs[:len(rs)-1])
	}
}

// isNameRune reports whether r may appear in a player-entered name.
// Mirrors the CleanName set: letters, digits, space, apostrophe, hyphen.
func isNameRune(r rune) bool {
	if r == ' ' || r == '\'' || r == '-' {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
		return true
	}
	return r > 127
}

// ReviewRow is one review-screen row: a draft or an action.
type ReviewRow struct {
	Label  string
	Action string // "discard:i", "add", or "begin"
	Index  int    // draft index for discard, else -1
}

// ReviewRows builds review rows: drafts, then Add (if fewer than 3),
// then Begin (if at least 1 draft).
func ReviewRows(drafts []DraftMember) []ReviewRow {
	var rows []ReviewRow
	for i, d := range drafts {
		name := CleanName(d.Name)
		if name == "" {
			name = "(random name)"
		}
		raceName := d.Race
		if r, ok := GetRace(d.Race); ok {
			raceName = r.Name
		}
		rows = append(rows, ReviewRow{
			Label:  fmt.Sprintf("%d. %s — %s %s", i+1, name, raceName, FriendlyID(d.Class)),
			Action: "discard",
			Index:  i,
		})
	}
	if len(drafts) < 3 {
		rows = append(rows, ReviewRow{Label: "[ Add pilgrim ]", Action: "add", Index: -1})
	}
	if len(drafts) > 0 {
		rows = append(rows, ReviewRow{Label: "[ Begin descent ]", Action: "begin", Index: -1})
	}
	return rows
}

// NewGameFull starts a run from a finished roster: 1-3 class/race pairs with
// raw names (blank = random, de-duplicated).
func NewGameFull(seed int64, tuning Tuning, classes, races, names []string) *Game {
	g := NewGameWithClassesAndRaces(seed, tuning, classes, races)
	used := map[string]bool{}
	for i, m := range g.Party.Members {
		if i < len(names) {
			if n := CleanName(names[i]); n != "" {
				m.Name = n
			}
		}
		if used[m.Name] {
			m.Name = GenerateName(g.RNG, used)
		}
		used[m.Name] = true
	}
	return g
}

func renderCreationChrome(tuning Tuning, title, sub string) (w, h int, cells [][]Cell) {
	w, h = tuning.Map.Width, tuning.Map.Height
	cells = make([][]Cell, h)
	for y := range h {
		cells[y] = make([]Cell, w)
		for x := range w {
			cells[y][x] = Cell{Glyph: ' ', FG: "bg", BG: "bg"}
		}
	}
	drawCentered(cells, w, 2, title, "gold-bright")
	drawCentered(cells, w, 3, sub, "gray-1")
	return w, h, cells
}

func creationPanel(status, hints string, tuning Tuning) ([]string, []string, string, string) {
	panel := []string{"", status, "Enter: continue", "Esc: back", "Up/Down: move"}
	panelFG := []string{"gray-1", "gold-bright", "gray-1", "gray-1", "gray-1"}
	for len(panel) < 12 {
		panel = append(panel, "")
		panelFG = append(panelFG, "gray-1")
	}
	return panel, panelFG, status, hints
}

// RenderSeedEntry draws the seed prompt. Blank means random.
func RenderSeedEntry(tuning Tuning, s *SeedEntryState) Frame {
	text := ""
	if s != nil {
		text = s.Text
	}
	w, h, cells := renderCreationChrome(tuning, "NEW EXPEDITION", "Enter seed (blank for random)")
	line := text + "_"
	if line == "_" {
		line = "(random)"
	}
	drawCentered(cells, w, 5, line, "gold")
	panel, panelFG, status, hints := creationPanel("Seed", "Type digits  Enter: continue  Esc: menu", tuning)
	return Frame{W: w, H: h, Cells: cells, Panel: panel, PanelFG: panelFG, Status: status, Log: make([]string, tuning.Layout.LogLines), Hints: hints, MinCols: tuning.Layout.MinCols, MinRows: tuning.Layout.MinRows}
}

// RenderClassPick draws one class slot of the pipeline.
func RenderClassPick(tuning Tuning, s *ClassPickState) Frame {
	slot, drafts := 1, 0
	if s != nil {
		slot, drafts = s.Slot+1, s.Drafts
	}
	w, h, cells := renderCreationChrome(tuning, "CHOOSE PILGRIM", fmt.Sprintf("Pilgrim %d (roster %d/3)", slot, drafts))
	if s != nil {
		for i, ci := range s.Classes {
			y := 5 + i*2
			if y+1 >= h-1 {
				break
			}
			prefix := "  "
			fg := "gray-1"
			if i == s.Cursor {
				prefix = "> "
				fg = "gold-bright"
			}
			line := fmt.Sprintf("%s%s - %s", prefix, strings.Title(ci.Name), ci.BuffA.Name)
			if len(line) > w-2 {
				line = line[:w-5] + "..."
			}
			drawString(cells, 2, y, line, fg)
			drawString(cells, 4, y+1, ci.Role, "gray-2")
		}
	}
	panel, panelFG, status, hints := creationPanel("Class", "Up/Down: move  Enter: pick  Esc: back", tuning)
	return Frame{W: w, H: h, Cells: cells, Panel: panel, PanelFG: panelFG, Status: status, Log: make([]string, tuning.Layout.LogLines), Hints: hints, MinCols: tuning.Layout.MinCols, MinRows: tuning.Layout.MinRows}
}

// RenderNameEntry draws the name prompt for one draft. Blank means random.
func RenderNameEntry(tuning Tuning, class, race, text string) Frame {
	raceName := race
	if r, ok := GetRace(race); ok {
		raceName = r.Name
	}
	sub := fmt.Sprintf("Name %s %s (blank = random)", raceName, FriendlyID(class))
	w, h, cells := renderCreationChrome(tuning, "NAME PILGRIM", sub)
	line := text + "_"
	if text == "" {
		line = "(random)"
	}
	drawCentered(cells, w, 5, line, "gold")
	panel, panelFG, status, hints := creationPanel("Name", "Type name  Enter: keep  Esc: back", tuning)
	return Frame{W: w, H: h, Cells: cells, Panel: panel, PanelFG: panelFG, Status: status, Log: make([]string, tuning.Layout.LogLines), Hints: hints, MinCols: tuning.Layout.MinCols, MinRows: tuning.Layout.MinRows}
}

// RenderReview draws the roster with discard and begin actions.
func RenderReview(tuning Tuning, drafts []DraftMember, cursor int) Frame {
	w, h, cells := renderCreationChrome(tuning, "REVIEW ROSTER", fmt.Sprintf("%d/3 pilgrims — Enter on a pilgrim discards them", len(drafts)))
	rows := ReviewRows(drafts)
	if cursor < 0 {
		cursor = 0
	}
	if len(rows) > 0 && cursor >= len(rows) {
		cursor = len(rows) - 1
	}
	for i, row := range rows {
		y := 5 + i*2
		if y >= h-1 {
			break
		}
		prefix := "  "
		fg := "gray-1"
		if i == cursor {
			prefix = "> "
			fg = "gold-bright"
		}
		line := prefix + row.Label
		if len(line) > w-2 {
			line = line[:w-5] + "..."
		}
		drawCentered(cells, w, y, line, fg)
	}
	panel, panelFG, status, hints := creationPanel("Review", "Up/Down: move  Enter: choose  Esc: back", tuning)
	return Frame{W: w, H: h, Cells: cells, Panel: panel, PanelFG: panelFG, Status: status, Log: make([]string, tuning.Layout.LogLines), Hints: hints, MinCols: tuning.Layout.MinCols, MinRows: tuning.Layout.MinRows}
}

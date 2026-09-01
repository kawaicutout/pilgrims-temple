package game

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
)

type ClassInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
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

func NewGameWithClasses(seed int64, tuning Tuning, classes []string) *Game {
	rng := rand.New(rand.NewPCG(uint64(seed), 0x9e3779b97f4a7c15))
	g := &Game{Seed: seed, RNG: rng, Tuning: tuning, Food: tuning.Food.StartClock, FoodFloat: float64(tuning.Food.StartClock), Level: 1}
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

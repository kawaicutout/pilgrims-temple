package main

import (
	"log"
	"time"

	"github.com/gdamore/tcell/v2"
	"partyrogue/game"
)

func main() {
	tuning, err := game.LoadTuning()
	if err != nil {
		log.Fatal(err)
	}
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatal(err)
	}
	if err := s.Init(); err != nil {
		log.Fatal(err)
	}
	defer s.Fini()

	bg := tcell.NewRGBColor(20, 18, 16)
	fg := tcell.NewRGBColor(230, 224, 216)
	gold := tcell.NewRGBColor(184, 151, 90)
	goldBr := tcell.NewRGBColor(211, 173, 107)
	redBr := tcell.NewRGBColor(201, 106, 90)
	wallCol := tcell.NewRGBColor(107, 100, 92)
	floorCol := tcell.NewRGBColor(74, 70, 66)
	gray1 := tcell.NewRGBColor(181, 174, 165)
	gray2 := tcell.NewRGBColor(138, 133, 126)

	styleBG := tcell.StyleDefault.Foreground(fg).Background(bg)
	styleGold := styleBG.Foreground(gold)
	styleGoldBr := styleBG.Foreground(goldBr)
	styleRedBr := styleBG.Foreground(redBr)
	styleWall := styleBG.Foreground(wallCol)
	styleFloor := styleBG.Foreground(floorCol)
	styleGray1 := styleBG.Foreground(gray1)
	styleGray2 := styleBG.Foreground(gray2)
	_ = styleGold

	drawFrame := func(frame game.Frame) {
		w, h := s.Size()
		minW := frame.MinCols
		minH := frame.MinRows
		for y := range h {
			for x := range w {
				s.SetContent(x, y, ' ', nil, styleBG)
			}
		}
		if w < minW || h < minH {
			msg := " Pilgrim's Temple - resize to " + itoa(minW) + "x" + itoa(minH) + " (you have " + itoa(w) + "x" + itoa(h) + ") "
			for i, ch := range msg {
				if i < w {
					s.SetContent(i, 0, ch, nil, styleGoldBr)
				}
			}
			hint := " Enlarge terminal, or run ./run.sh which requests 110x34. "
			for i, ch := range hint {
				if i < w && 1 < h {
					s.SetContent(i, 1, ch, nil, styleGray2)
				}
			}
			s.Show()
			return
		}
		for y := range frame.H {
			for x := range frame.W {
				if y >= h || x >= w {
					continue
				}
				cell := frame.Cells[y][x]
				var st tcell.Style
				switch cell.FG {
				case "player":
					st = styleGoldBr
				case "enemy":
					st = styleRedBr
				case "wall":
					st = styleWall
				case "floor":
					st = styleFloor
				case "gold":
					st = styleGold
				case "gold-bright":
					st = styleGoldBr
				case "gray-3", "gray-2":
					st = styleGray2
				default:
					if cell.FG == "bg" {
						st = styleBG
					} else {
						st = styleGray1
						if len(frame.Cells[y][x].FG) > 0 && frame.Cells[y][x].FG == "gold-bright" {
							st = styleGoldBr
						}
					}
					// For menu, check FG string
					if cell.FG == "gold-bright" {
						st = styleGoldBr
					} else if cell.FG == "gray-1" {
						st = styleGray1
					} else if cell.FG == "gray-2" {
						st = styleGray2
					} else if cell.FG == "gold" {
						st = styleGold
					}
				}
				s.SetContent(x, y, cell.Glyph, nil, st)
			}
		}
		panelX := frame.W + 1
		if panelX < w {
			for i, line := range frame.Panel {
				y := i
				if y >= frame.H {
					break
				}
				maxLen := w - panelX
				if maxLen <= 0 {
					break
				}
				runes := []rune(line)
				if len(runes) > maxLen {
					if maxLen > 1 {
						runes = runes[:maxLen-1]
						runes = append(runes, '…')
					} else {
						runes = runes[:maxLen]
					}
				}
				style := styleGray1
				if len(line) > 0 && line[0] == '>' {
					style = styleGoldBr
				}
				for j, ch := range runes {
					s.SetContent(panelX+j, y, ch, nil, style)
				}
			}
		}
		statusY := frame.H
		if statusY < h {
			for i, ch := range frame.Status {
				if i >= w {
					break
				}
				s.SetContent(i, statusY, ch, nil, styleGold)
			}
		}
		logY := frame.H + 1
		for i, line := range frame.Log {
			y := logY + i
			if y >= h {
				break
			}
			for j, ch := range line {
				if j >= w {
					break
				}
				s.SetContent(j, y, ch, nil, styleGray1)
			}
		}
		hintY := h - 1
		if hintY > logY+len(frame.Log) {
			for i, ch := range frame.Hints {
				if i >= w {
					break
				}
				s.SetContent(i, hintY, ch, nil, styleGray2)
			}
		}
		s.Show()
	}

	// App states
	type appState int
	const (
		stateMenu appState = iota
		stateCharSelect
		statePlaying
	)

	state := stateMenu
	menu := &game.MainMenuState{Selected: 0}
	var cs *game.CharSelectState
	var g *game.Game

	drawFrame(game.RenderMainMenu(tuning, menu.Selected))
	for {
		ev := s.PollEvent()
		switch e := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
			switch state {
			case stateMenu:
				drawFrame(game.RenderMainMenu(tuning, menu.Selected))
			case stateCharSelect:
				if cs != nil {
					drawFrame(game.RenderCharSelect(tuning, cs))
				}
			case statePlaying:
				if g != nil {
					drawFrame(g.Render())
				}
			}
		case *tcell.EventKey:
			key, code := tcellKeyToRaw(e)
			k := game.NormalizeKey(key, code)
			switch state {
			case stateMenu:
				switch k {
				case game.KeyUp:
					menu.Move(-1)
					drawFrame(game.RenderMainMenu(tuning, menu.Selected))
				case game.KeyDown:
					menu.Move(1)
					drawFrame(game.RenderMainMenu(tuning, menu.Selected))
				case game.KeyEnter:
					switch menu.Selected {
					case 0: // New Game
						var err error
						cs, err = game.NewCharSelect()
						if err != nil {
							cs = &game.CharSelectState{Classes: []game.ClassInfo{{ID: "fighter", Name: "Fighter"}, {ID: "cleric", Name: "Cleric"}}, Picks: []string{}}
						}
						state = stateCharSelect
						drawFrame(game.RenderCharSelect(tuning, cs))
					case 1: // Scores placeholder
						// Show scores as log in menu? For now just stay
						drawFrame(game.RenderMainMenu(tuning, menu.Selected))
					case 2: // Exit
						return
					}
				case game.KeyQuit:
					return
				}
			case stateCharSelect:
				if cs == nil {
					state = stateMenu
					drawFrame(game.RenderMainMenu(tuning, menu.Selected))
					break
				}
				switch k {
				case game.KeyUp:
					cs.Move(-1)
					drawFrame(game.RenderCharSelect(tuning, cs))
				case game.KeyDown:
					cs.Move(1)
					drawFrame(game.RenderCharSelect(tuning, cs))
				case game.KeyEnter:
					if cs.Done() {
						seed := time.Now().UnixNano()
						g = game.NewGameWithClasses(seed, tuning, cs.Picks)
						state = statePlaying
						drawFrame(g.Render())
					} else {
						cs.Select()
						drawFrame(game.RenderCharSelect(tuning, cs))
						if cs.Done() {
							// Stay, waiting for Enter to begin
							drawFrame(game.RenderCharSelect(tuning, cs))
						}
					}
				case game.KeyQuit:
					if cs.Back() {
						state = stateMenu
						drawFrame(game.RenderMainMenu(tuning, menu.Selected))
					} else {
						drawFrame(game.RenderCharSelect(tuning, cs))
					}
				}
			case statePlaying:
				if g == nil {
					state = stateMenu
					drawFrame(game.RenderMainMenu(tuning, menu.Selected))
					break
				}
				g.HandleKey(k)
				drawFrame(g.Render())
				if g.Quit {
					// Return to menu, not a death
					state = stateMenu
					g = nil
					drawFrame(game.RenderMainMenu(tuning, menu.Selected))
				} else if g.Over {
					// Wait for Esc to return to menu
					for {
						ev2 := s.PollEvent()
						if ke, ok := ev2.(*tcell.EventKey); ok {
							key2, code2 := tcellKeyToRaw(ke)
							k2 := game.NormalizeKey(key2, code2)
							if k2 == game.KeyQuit || k2 == game.KeyEnter {
								state = stateMenu
								g = nil
								drawFrame(game.RenderMainMenu(tuning, menu.Selected))
								break
							}
						} else if _, ok := ev2.(*tcell.EventResize); ok {
							s.Sync()
							if g != nil {
								drawFrame(g.Render())
							} else {
								drawFrame(game.RenderMainMenu(tuning, menu.Selected))
							}
						}
					}
				}
			}
		}
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	s := ""
	for v > 0 {
		s = string(rune('0'+v%10)) + s
		v /= 10
	}
	return s
}

func tcellKeyToRaw(e *tcell.EventKey) (key string, code string) {
	switch e.Key() {
	case tcell.KeyUp:
		return "ArrowUp", "ArrowUp"
	case tcell.KeyDown:
		return "ArrowDown", "ArrowDown"
	case tcell.KeyLeft:
		return "ArrowLeft", "ArrowLeft"
	case tcell.KeyRight:
		return "ArrowRight", "ArrowRight"
	case tcell.KeyEscape:
		return "Escape", "Escape"
	case tcell.KeyEnter:
		return "Enter", "Enter"
	}
	if e.Key() == tcell.KeyRune {
		return string(e.Rune()), ""
	}
	return "", ""
}

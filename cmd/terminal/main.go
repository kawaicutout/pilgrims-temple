// Terminal frontend: uses system monospace via tcell; web build imports Libertinus Mono
// via web/tokens.css @import. run.sh passes font override for terminals that support it
// (alacritty -o font.normal.family="Libertinus Mono", kitty -o font_family="Libertinus Mono");
// foot/gnome-terminal use config/gsettings, not flags. Bin builds rely on system fallback.
package main

import (
	"log"
	"strings"
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
	slateCol := tcell.NewRGBColor(110, 143, 181)

	styleBG := tcell.StyleDefault.Foreground(fg).Background(bg)
	styleGold := styleBG.Foreground(gold)
	styleGoldBr := styleBG.Foreground(goldBr)
	styleRedBr := styleBG.Foreground(redBr)
	styleWall := styleBG.Foreground(wallCol)
	styleFloor := styleBG.Foreground(floorCol)
	styleGray1 := styleBG.Foreground(gray1)
	styleGray2 := styleBG.Foreground(gray2)
	styleSlate := styleBG.Foreground(slateCol)
	_ = styleGold
	_ = styleSlate

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
				if strings.HasPrefix(cell.FG, "#") {
					if c, ok := parseHexColor(cell.FG); ok {
						st = styleBG.Foreground(c)
					} else {
						st = styleRedBr
					}
				} else {
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
					case "red-bright":
						st = styleRedBr
					case "slate":
						st = styleSlate
					case "gray-3", "gray-2":
						st = styleGray2
					case "gray-1":
						st = styleGray1
					default:
						if cell.FG == "bg" {
							st = styleBG
						} else {
							st = styleGray1
							if len(frame.Cells[y][x].FG) > 0 && frame.Cells[y][x].FG == "gold-bright" {
								st = styleGoldBr
							}
						}
						if cell.FG == "gold-bright" {
							st = styleGoldBr
						} else if cell.FG == "gray-1" {
							st = styleGray1
						} else if cell.FG == "gray-2" {
							st = styleGray2
						} else if cell.FG == "gold" {
							st = styleGold
						} else if cell.FG == "red-bright" {
							st = styleRedBr
						} else if cell.FG == "slate" {
							st = styleSlate
						}
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
				if i < len(frame.PanelFG) && frame.PanelFG[i] != "" {
					switch frame.PanelFG[i] {
					case "gold-bright":
						style = styleGoldBr
					case "red-bright":
						style = styleRedBr
					case "slate":
						style = styleSlate
					case "gray-1":
						style = styleGray1
					case "gray-2", "gray-3":
						style = styleGray2
					case "gold":
						style = styleGold
					default:
						style = styleGray1
					}
				} else if len(line) > 0 && line[0] == '>' {
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
		stateWizard
		stateWizardAddMember
		stateWizardRemoveMember
	)

	state := stateMenu
	menu := &game.MainMenuState{Selected: 0}
	var cs *game.CharSelectState
	var g *game.Game
	wizardState := &game.WizardState{Selected: 0}
	var wizardAddCS *game.CharSelectState
	wizardRemoveIdx := 0
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
					if g.HelpActive {
						drawFrame(g.RenderHelpOverlay())
					} else if g.LevelUpPending != nil {
						drawFrame(g.RenderLevelUp())
					} else {
						drawFrame(g.Render())
					}
				}
			case stateWizard:
				if g != nil {
					drawFrame(g.RenderWizardMenu(tuning, wizardState.Selected))
				}
			case stateWizardAddMember:
				if wizardAddCS != nil {
					drawFrame(game.RenderCharSelect(tuning, wizardAddCS))
				}
			case stateWizardRemoveMember:
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
				// Help overlay has priority: Esc / Enter / ? exits without consuming turn.
				if g.HelpActive {
					switch k {
					case game.KeyQuit, game.KeyEnter, game.KeyHelp:
						g.HelpActive = false
						drawFrame(g.Render())
						if g.LevelUpPending != nil {
							drawFrame(g.RenderLevelUp())
						}
					default:
						drawFrame(g.RenderHelpOverlay())
					}
					break
				}
				// Level up pending: handle picks (pause world)
				if g.LevelUpPending != nil {
					pick := g.LevelUpPending.Picks[g.LevelUpPending.Current]
					handled := false
					cursorMoved := false
					switch k {
					case game.KeyUp:
						if !pick.IsAffix {
							g.MoveLevelUpCursor(-1)
							cursorMoved = true
						}
					case game.KeyDown:
						if !pick.IsAffix {
							g.MoveLevelUpCursor(1)
							cursorMoved = true
						}
					case game.KeyEnter:
						idx := 0
						if !pick.IsAffix {
							idx = g.LevelUpPending.Cursor
						}
						g.ApplyTalentPick(g.LevelUpPending.Current, idx)
						handled = true
					case game.KeyQuit:
						g.ApplyTalentPick(g.LevelUpPending.Current, 0)
						handled = true
					}
					if !handled && !cursorMoved && !pick.IsAffix {
						switch key {
						case "1":
							g.ApplyTalentPick(g.LevelUpPending.Current, 0)
							handled = true
						case "2":
							if len(pick.Options) > 1 {
								g.ApplyTalentPick(g.LevelUpPending.Current, 1)
								handled = true
							}
						case "3":
							if len(pick.Options) > 2 {
								g.ApplyTalentPick(g.LevelUpPending.Current, 2)
								handled = true
							}
						}
					}
					if handled {
						if g.LevelUpPending == nil {
							drawFrame(g.Render())
						} else {
							drawFrame(g.RenderLevelUp())
						}
					} else {
						drawFrame(g.RenderLevelUp())
					}
					break
				}
				// Wizard key: open wizard menu (not when LevelUpPending or Look active)
				if k == game.KeyWizard && (g.Look == nil || !g.Look.Active) && !g.Over && !g.Quit {
					wizardState.Selected = 0
					state = stateWizard
					drawFrame(g.RenderWizardMenu(tuning, wizardState.Selected))
					break
				}
				g.HandleKey(k)
				if g.HelpActive {
					drawFrame(g.RenderHelpOverlay())
				} else {
					drawFrame(g.Render())
				}
				if g.LevelUpPending != nil {
					drawFrame(g.RenderLevelUp())
				}
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
			case stateWizard:
				switch k {
				case game.KeyUp:
					wizardState.Move(-1)
					drawFrame(g.RenderWizardMenu(tuning, wizardState.Selected))
				case game.KeyDown:
					wizardState.Move(1)
					drawFrame(g.RenderWizardMenu(tuning, wizardState.Selected))
				case game.KeyQuit:
					state = statePlaying
					drawFrame(g.Render())
					if g.LevelUpPending != nil {
						drawFrame(g.RenderLevelUp())
					}
				case game.KeyEnter:
					opt := game.WizardOptions[wizardState.Selected]
					switch opt.ID {
					case "add_member":
						if len(g.Party.Members) >= 4 {
							g.WizardAddMember()
							state = statePlaying
							drawFrame(g.Render())
							break
						}
						var err error
						wizardAddCS, err = game.NewCharSelect()
						if err != nil {
							// fallback: directly add random
							g.WizardAddMember()
							state = statePlaying
							drawFrame(g.Render())
							break
						}
						wizardAddCS.Picks = []string{}
						state = stateWizardAddMember
						drawFrame(game.RenderCharSelect(tuning, wizardAddCS))
					case "remove_member":
						// Enter remove-member selection
						wizardRemoveIdx = g.Party.Selected
						// ensure valid living
						found := -1
						for i, m := range g.Party.Members {
							if m.IsAlive() {
								found = i
								break
							}
						}
						if found >= 0 {
							wizardRemoveIdx = found
						}
						state = stateWizardRemoveMember
						drawFrame(g.Render())
					case "instant_level", "spawn_loot", "food_1000", "full_heal", "reveal_all":
						g.WizardExecute(opt.ID)
						state = statePlaying
						drawFrame(g.Render())
						if g.LevelUpPending != nil {
							drawFrame(g.RenderLevelUp())
						}
					default:
						g.WizardExecute(opt.ID)
						state = statePlaying
						drawFrame(g.Render())
					}
				}
			case stateWizardAddMember:
				if wizardAddCS == nil {
					state = stateWizard
					drawFrame(g.RenderWizardMenu(tuning, wizardState.Selected))
					break
				}
				switch k {
				case game.KeyUp:
					wizardAddCS.Move(-1)
					drawFrame(game.RenderCharSelect(tuning, wizardAddCS))
				case game.KeyDown:
					wizardAddCS.Move(1)
					drawFrame(game.RenderCharSelect(tuning, wizardAddCS))
				case game.KeyEnter:
					// pick one class then add
					wizardAddCS.Select()
					if len(wizardAddCS.Picks) > 0 {
						classID := wizardAddCS.Picks[0]
						g.SetWizard()
						if len(g.Party.Members) < 4 {
							tmp := game.GeneratePartyWithClasses(g.RNG, []string{classID}, 1)
							if tmp != nil && len(tmp.Members) > 0 {
								m := tmp.Members[0]
								for lvl := 1; lvl < g.Level; lvl++ {
									m.MaxHP += 1 + g.RNG.IntN(2)
									if g.RNG.IntN(2) == 0 {
										m.ATK[0]++
										m.ATK[1]++
									}
									if g.RNG.IntN(4) == 0 {
										m.DEF++
									}
								}
								m.HP = m.MaxHP
								m.Alive = true
								g.Party.Members = append(g.Party.Members, m)
								g.Party.EnsureSelection()
								g.Logf("Wizard: Add Party Member -> %s the %s joined", m.Name, m.Class)
							}
						} else {
							g.Logf("Wizard: Add Party Member - party full")
						}
						wizardAddCS = nil
						state = statePlaying
						drawFrame(g.Render())
					} else {
						drawFrame(game.RenderCharSelect(tuning, wizardAddCS))
					}
				case game.KeyQuit:
					if wizardAddCS.Back() {
						// if backs out completely
						wizardAddCS = nil
						state = stateWizard
						drawFrame(g.RenderWizardMenu(tuning, wizardState.Selected))
					} else {
						drawFrame(game.RenderCharSelect(tuning, wizardAddCS))
					}
				}
			case stateWizardRemoveMember:
				switch k {
				case game.KeyUp:
					// move to previous living
					for range g.Party.Members {
						wizardRemoveIdx--
						if wizardRemoveIdx < 0 {
							wizardRemoveIdx = len(g.Party.Members) - 1
						}
						if g.Party.Members[wizardRemoveIdx].IsAlive() {
							break
						}
					}
					g.Party.Selected = wizardRemoveIdx
					drawFrame(g.Render())
				case game.KeyDown:
					for range g.Party.Members {
						wizardRemoveIdx++
						if wizardRemoveIdx >= len(g.Party.Members) {
							wizardRemoveIdx = 0
						}
						if g.Party.Members[wizardRemoveIdx].IsAlive() {
							break
						}
					}
					g.Party.Selected = wizardRemoveIdx
					drawFrame(g.Render())
				case game.KeyEnter:
					g.WizardRemoveMember(wizardRemoveIdx)
					state = statePlaying
					drawFrame(g.Render())
					if g.Over {
						// will handle in next playing loop
					}
				case game.KeyQuit:
					state = stateWizard
					drawFrame(g.RenderWizardMenu(tuning, wizardState.Selected))
				}
			}
		}
	}
}
func parseHexColor(s string) (tcell.Color, bool) {
	if len(s) != 7 || s[0] != '#' {
		return 0, false
	}
	var r, g, b int
	for i := range 3 {
		hi := hexVal(s[1+i*2])
		lo := hexVal(s[2+i*2])
		if hi < 0 || lo < 0 {
			return 0, false
		}
		v := hi*16 + lo
		switch i {
		case 0:
			r = v
		case 1:
			g = v
		case 2:
			b = v
		}
	}
	return tcell.NewRGBColor(int32(r), int32(g), int32(b)), true
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c - 'a' + 10)
	case c >= 'A' && c <= 'F':
		return int(c - 'A' + 10)
	}
	return -1
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
	case tcell.KeyUpLeft:
		return "7", "Numpad7"
	case tcell.KeyUpRight:
		return "9", "Numpad9"
	case tcell.KeyDownLeft:
		return "1", "Numpad1"
	case tcell.KeyDownRight:
		return "3", "Numpad3"
	case tcell.KeyHome:
		return "7", "Numpad7"
	case tcell.KeyPgUp:
		return "9", "Numpad9"
	case tcell.KeyEnd:
		return "1", "Numpad1"
	case tcell.KeyPgDn:
		return "3", "Numpad3"
	case tcell.KeyClear, tcell.KeyCenter:
		return "5", "Numpad5"
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

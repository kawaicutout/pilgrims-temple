//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"
	"time"

	"partyrogue/game"
)

func main() {
	tuning, err := game.LoadTuning()
	if err != nil {
		panic(err)
	}
	doc := js.Global().Get("document")
	gameDiv := doc.Call("getElementById", "game")
	statusDiv := doc.Call("getElementById", "status")
	logDiv := doc.Call("getElementById", "log")
	hintsDiv := doc.Call("getElementById", "hints")
	if gameDiv.IsNull() {
		body := doc.Get("body")
		gameDiv = doc.Call("createElement", "div")
		gameDiv.Set("id", "game")
		body.Call("appendChild", gameDiv)
		statusDiv = doc.Call("createElement", "div")
		statusDiv.Set("id", "status")
		body.Call("appendChild", statusDiv)
		logDiv = doc.Call("createElement", "div")
		logDiv.Set("id", "log")
		body.Call("appendChild", logDiv)
		hintsDiv = doc.Call("createElement", "div")
		hintsDiv.Set("id", "hints")
		body.Call("appendChild", hintsDiv)
	}
	// Helper to render log/hints into separate divs between map and hints, mirroring terminal y=H status.
	renderLogHints := func(frame game.Frame) {
		if logDiv.IsNull() || hintsDiv.IsNull() {
			return
		}
		logDiv.Set("innerHTML", buildLogHTML(frame.Log))
		hintsDiv.Set("textContent", frame.Hints)
	}
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

	renderMenu := func() {
		frame := game.RenderMainMenu(tuning, menu.Selected)
		gameDiv.Set("innerHTML", buildHTML(frame, tuning))
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
	}
	renderCharSelect := func() {
		if cs == nil {
			return
		}
		frame := game.RenderCharSelect(tuning, cs)
		gameDiv.Set("innerHTML", buildHTML(frame, tuning))
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
	}
	renderHelp := func() {
		if g == nil {
			return
		}
		frame := g.RenderHelpOverlay()
		gameDiv.Set("innerHTML", buildHTML(frame, tuning))
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
	}
	renderGame := func() {
		if g == nil {
			return
		}
		if g.HelpActive {
			renderHelp()
			return
		}
		frame := g.Render()
		gameDiv.Set("innerHTML", buildHTML(frame, tuning))
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
		if g.Quit {
			statusDiv.Set("textContent", "Quit to menu. Seed "+itoa(g.Seed)+" - refresh or Esc")
		} else if g.Won {
			statusDiv.Set("textContent", "VICTORY! Seed "+itoa(g.Seed)+" - refresh to play again.")
		} else if g.Over {
			statusDiv.Set("textContent", "YOU DIED. Seed "+itoa(g.Seed)+" - refresh to play again.")
		} else if g.LevelUpPending != nil {
			frame2 := g.RenderLevelUp()
			gameDiv.Set("innerHTML", buildHTML(frame2, tuning))
			statusDiv.Set("textContent", frame2.Status)
			renderLogHints(frame2)
		}
	}
	renderLevelUp := func() {
		if g == nil || g.LevelUpPending == nil {
			return
		}
		frame := g.RenderLevelUp()
		gameDiv.Set("innerHTML", buildHTML(frame, tuning))
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
	}
	renderWizard := func() {
		if g == nil {
			return
		}
		frame := g.RenderWizardMenu(tuning, wizardState.Selected)
		gameDiv.Set("innerHTML", buildHTML(frame, tuning))
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
	}

	renderMenu()

	var keyHandler js.Func
	keyHandler = js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		key := e.Get("key").String()
		code := e.Get("code").String()
		k := game.NormalizeKey(key, code)
		switch k {
		case game.KeyUp, game.KeyDown, game.KeyLeft, game.KeyRight, game.KeyUpLeft, game.KeyUpRight, game.KeyDownLeft, game.KeyDownRight, game.KeyWait:
			e.Call("preventDefault")
		}
		switch state {
		case stateMenu:
			switch k {
			case game.KeyUp:
				menu.Move(-1)
				renderMenu()
			case game.KeyDown:
				menu.Move(1)
				renderMenu()
			case game.KeyEnter:
				switch menu.Selected {
				case 0:
					var err error
					cs, err = game.NewCharSelect()
					if err != nil {
						cs = &game.CharSelectState{Classes: []game.ClassInfo{{ID: "fighter", Name: "Fighter"}, {ID: "cleric", Name: "Cleric"}}}
					}
					state = stateCharSelect
					renderCharSelect()
				case 1:
					// Scores placeholder
					renderMenu()
				case 2:
					// Exit not applicable on web; just stay
				}
			case game.KeyQuit:
				// No exit on web
			}
		case stateCharSelect:
			if cs == nil {
				state = stateMenu
				renderMenu()
				break
			}
			switch k {
			case game.KeyUp:
				cs.Move(-1)
				renderCharSelect()
			case game.KeyDown:
				cs.Move(1)
				renderCharSelect()
			case game.KeyEnter:
				if cs.Done() {
					seed := time.Now().UnixNano()
					g = game.NewGameWithClasses(seed, tuning, cs.Picks)
					state = statePlaying
					renderGame()
				} else {
					cs.Select()
					renderCharSelect()
				}
			case game.KeyQuit:
				if cs.Back() {
					state = stateMenu
					renderMenu()
				} else {
					renderCharSelect()
				}
			}
		case statePlaying:
			if g == nil {
				state = stateMenu
				renderMenu()
				break
			}
			if g.HelpActive {
				switch k {
				case game.KeyQuit, game.KeyEnter, game.KeyHelp:
					g.HelpActive = false
					renderGame()
					if g.LevelUpPending != nil {
						renderLevelUp()
					}
				default:
					renderHelp()
				}
				break
			}
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
						renderGame()
					} else {
						renderLevelUp()
					}
				} else {
					renderLevelUp()
				}
				break
			}
			if k == game.KeyWizard && (g.Look == nil || !g.Look.Active) && !g.Over && !g.Quit {
				wizardState.Selected = 0
				state = stateWizard
				renderWizard()
				break
			}
			g.HandleKey(k)
			if g.HelpActive {
				renderHelp()
			} else {
				renderGame()
			}
			if g.Quit {
				state = stateMenu
				g = nil
				renderMenu()
			} else if g.Over {
				if k == game.KeyQuit || k == game.KeyEnter {
					state = stateMenu
					g = nil
					renderMenu()
				}
			}
		case stateWizard:
			switch k {
			case game.KeyUp:
				wizardState.Move(-1)
				renderWizard()
			case game.KeyDown:
				wizardState.Move(1)
				renderWizard()
			case game.KeyQuit:
				state = statePlaying
				renderGame()
			case game.KeyEnter:
				opt := game.WizardOptions[wizardState.Selected]
				switch opt.ID {
				case "add_member":
					if len(g.Party.Members) >= 4 {
						g.WizardAddMember()
						state = statePlaying
						renderGame()
						break
					}
					var err error
					wizardAddCS, err = game.NewCharSelect()
					if err != nil {
						g.WizardAddMember()
						state = statePlaying
						renderGame()
						break
					}
					wizardAddCS.Picks = []string{}
					state = stateWizardAddMember
					frame := game.RenderCharSelect(tuning, wizardAddCS)
					gameDiv.Set("innerHTML", buildHTML(frame, tuning))
					statusDiv.Set("textContent", frame.Status)
					renderLogHints(frame)
				case "remove_member":
					wizardRemoveIdx = g.Party.Selected
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
					renderGame()
				case "instant_level", "spawn_loot", "food_1000", "full_heal", "reveal_all":
					g.WizardExecute(opt.ID)
					state = statePlaying
					renderGame()
					if g.LevelUpPending != nil {
						renderLevelUp()
					}
				default:
					g.WizardExecute(opt.ID)
					state = statePlaying
					renderGame()
				}
			}
		case stateWizardAddMember:
			if wizardAddCS == nil {
				state = stateWizard
				renderWizard()
				break
			}
			switch k {
			case game.KeyUp:
				wizardAddCS.Move(-1)
				frame := game.RenderCharSelect(tuning, wizardAddCS)
				gameDiv.Set("innerHTML", buildHTML(frame, tuning))
				statusDiv.Set("textContent", frame.Status)
				renderLogHints(frame)
			case game.KeyDown:
				wizardAddCS.Move(1)
				frame := game.RenderCharSelect(tuning, wizardAddCS)
				gameDiv.Set("innerHTML", buildHTML(frame, tuning))
				statusDiv.Set("textContent", frame.Status)
				renderLogHints(frame)
			case game.KeyEnter:
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
					renderGame()
				} else {
					frame := game.RenderCharSelect(tuning, wizardAddCS)
					gameDiv.Set("innerHTML", buildHTML(frame, tuning))
					statusDiv.Set("textContent", frame.Status)
					renderLogHints(frame)
				}
			case game.KeyQuit:
				if wizardAddCS.Back() {
					wizardAddCS = nil
					state = stateWizard
					renderWizard()
				} else {
					frame := game.RenderCharSelect(tuning, wizardAddCS)
					gameDiv.Set("innerHTML", buildHTML(frame, tuning))
					statusDiv.Set("textContent", frame.Status)
					renderLogHints(frame)
				}
			}
		case stateWizardRemoveMember:
			switch k {
			case game.KeyUp:
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
				renderGame()
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
				renderGame()
			case game.KeyEnter:
				g.WizardRemoveMember(wizardRemoveIdx)
				state = statePlaying
				renderGame()
			case game.KeyQuit:
				state = stateWizard
				renderWizard()
			}
		}
		return nil
	})
	js.Global().Get("document").Call("addEventListener", "keydown", keyHandler)
	select {}
}

func itoa(v int64) string {
	s := ""
	if v == 0 {
		return "0"
	}
	neg := false
	if v < 0 {
		neg = true
		v = -v
	}
	for v > 0 {
		s = string(rune('0'+v%10)) + s
		v /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

func buildHTML(frame game.Frame, tuning game.Tuning) string {
	esc := func(s string) string {
		out := ""
		for _, ch := range s {
			switch ch {
			case '&':
				out += "&amp;"
			case '<':
				out += "&lt;"
			case '>':
				out += "&gt;"
			default:
				out += string(ch)
			}
		}
		return out
	}
	panelMax := tuning.Layout.MinCols - tuning.Map.Width - 1
	if panelMax < 10 {
		panelMax = 29
	}
	html := `<div style="display:flex;gap:16px;align-items:flex-start"><div style="font-family:var(--font-monospace);line-height:var(--map-line-height);white-space:pre">`
	for y := range frame.H {
		for x := range frame.W {
			cell := frame.Cells[y][x]
			col := colorForToken(cell.FG)
			ch := string(cell.Glyph)
			html += `<span style="color:` + col + `">` + esc(ch) + `</span>`
		}
		if y < len(frame.Panel) {
			line := frame.Panel[y]
			runes := []rune(line)
			if len(runes) > panelMax {
				if panelMax > 1 {
					runes = runes[:panelMax-1]
					runes = append(runes, '…')
				} else {
					runes = runes[:panelMax]
				}
				line = string(runes)
			}
			var col string
			if y < len(frame.PanelFG) && frame.PanelFG[y] != "" {
				col = colorForToken(frame.PanelFG[y])
			} else if len(line) > 0 && line[0] == '>' {
				col = "var(--gold-bright)"
			} else {
				col = "var(--gray-1)"
			}
			html += `<span style="color:` + col + `"> ` + esc(line) + `</span>`
		}
		html += "\n"
	}
	html += `</div></div>`
	return html
}

func buildLogHTML(lines []string) string {
	esc := func(s string) string {
		out := ""
		for _, ch := range s {
			switch ch {
			case '&':
				out += "&amp;"
			case '<':
				out += "&lt;"
			case '>':
				out += "&gt;"
			default:
				out += string(ch)
			}
		}
		return out
	}
	html := ""
	for _, line := range lines {
		if line == "" {
			html += "<br>"
		} else {
			html += esc(line) + "<br>"
		}
	}
	return html
}

func colorForToken(token string) string {
	if strings.HasPrefix(token, "#") && len(token) == 7 {
		// Validate hex digits; fallback to red-bright if invalid.
		valid := true
		for _, c := range token[1:] {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				valid = false
				break
			}
		}
		if valid {
			return token
		}
		return "var(--red-bright)"
	}
	if strings.HasPrefix(token, "enemy-#") {
		return token[len("enemy-"):]
	}
	switch token {
	case "player", "gold-bright":
		return "var(--gold-bright)"
	case "enemy", "red-bright":
		return "var(--red-bright)"
	case "wall":
		return "var(--wall)"
	case "floor":
		return "var(--floor)"
	case "gold":
		return "var(--gold)"
	case "gray-1":
		return "var(--gray-1)"
	case "gray-2", "gray-3":
		return "var(--gray-2)"
	case "slate":
		return "var(--slate)"
	case "bg":
		return "var(--fg)"
	default:
		return "var(--fg)"
	}
}

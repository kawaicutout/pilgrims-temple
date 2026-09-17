//go:build js && wasm

package main

import (
	"log"
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
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
	// Shell controls: data zip upload + reset + minimal JSON editor (localStorage overlay "data:"+name).
	setupShellControls := func() {
		// Zip upload handler (archive/zip inside WASM)
		if zipInput := doc.Call("getElementById", "dataZip"); !zipInput.IsNull() {
			var onZipChange js.Func
			onZipChange = js.FuncOf(func(this js.Value, args []js.Value) any {
				files := zipInput.Get("files")
				if files.Get("length").Int() == 0 {
					return nil
				}
				file := files.Index(0)
				reader := js.Global().Get("FileReader").New()
				var onLoad js.Func
				onLoad = js.FuncOf(func(this js.Value, args []js.Value) any {
					defer onLoad.Release()
					buf := reader.Get("result")
					u8 := js.Global().Get("Uint8Array").New(buf)
					n := u8.Get("length").Int()
					data := make([]byte, n)
					js.CopyBytesToGo(data, u8)
					r, err := zip.NewReader(bytes.NewReader(data), int64(n))
					shellStatus := doc.Call("getElementById", "shellStatus")
					if err != nil {
						if !shellStatus.IsNull() {
							shellStatus.Set("textContent", "zip error: "+err.Error())
						}
						return nil
					}
					ls := js.Global().Get("localStorage")
					count := 0
					for _, f := range r.File {
						if f.FileInfo().IsDir() {
							continue
						}
						rc, err := f.Open()
						if err != nil {
							continue
						}
						b, err := io.ReadAll(rc)
						rc.Close()
						if err != nil {
							continue
						}
						name := path.Base(f.Name)
						if name == "" || name == "." {
							continue
						}
						ls.Call("setItem", "data:"+name, string(b))
						count++
					}
					if !shellStatus.IsNull() {
						shellStatus.Set("textContent", fmt.Sprintf("Stored %d files — reloading…", count))
					}
					js.Global().Get("location").Call("reload")
					return nil
				})
				reader.Set("onload", onLoad)
				reader.Call("readAsArrayBuffer", file)
				return nil
			})
			zipInput.Call("addEventListener", "change", onZipChange)
		}
		// Reset handler: clears localStorage data:* and reloads embedded defaults
		if resetBtn := doc.Call("getElementById", "resetData"); !resetBtn.IsNull() {
			resetBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
				ls := js.Global().Get("localStorage")
				if !ls.IsNull() && !ls.IsUndefined() {
					lsLen := ls.Get("length").Int()
					toRemove := []string{}
					for i := range lsLen {
						k := ls.Call("key", i)
						if k.IsNull() || k.IsUndefined() {
							continue
						}
						ks := k.String()
						if strings.HasPrefix(ks, "data:") {
							toRemove = append(toRemove, ks)
						}
					}
					for _, k := range toRemove {
						ls.Call("removeItem", k)
					}
				}
				shellStatus := doc.Call("getElementById", "shellStatus")
				if !shellStatus.IsNull() {
					shellStatus.Set("textContent", "Reset — reloading defaults…")
				}
				js.Global().Get("location").Call("reload")
				return nil
			}))
		}
		// Minimal data editor: select file → textarea, Save validates JSON and writes overlay.
		editorEl := doc.Call("getElementById", "editor")
		editSelect := doc.Call("getElementById", "editSelect")
		editArea := doc.Call("getElementById", "editArea")
		editStatus := doc.Call("getElementById", "editStatus")
		editLoad := doc.Call("getElementById", "editLoad")
		editSave := doc.Call("getElementById", "editSave")
		toggleBtn := doc.Call("getElementById", "toggleEditor")
		loadIntoEditor := func() {
			if editSelect.IsNull() || editArea.IsNull() {
				return
			}
			name := editSelect.Get("value").String()
			if name == "" {
				name = "tuning.json"
			}
			b, err := game.RawJSON(name)
			if err != nil {
				if !editStatus.IsNull() {
					editStatus.Set("textContent", "load error: "+err.Error())
				}
				return
			}
			// Pretty-print if JSON
			var m json.RawMessage
			if json.Unmarshal(b, &m) == nil {
				var pretty bytes.Buffer
				if json.Indent(&pretty, b, "", "  ") == nil {
					b = pretty.Bytes()
				}
			}
			editArea.Set("value", string(b))
			if !editStatus.IsNull() {
				editStatus.Set("textContent", "Loaded "+name+" ("+itoa(int64(len(b)))+" bytes)")
			}
		}
		if !editLoad.IsNull() {
			editLoad.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
				loadIntoEditor()
				return nil
			}))
		}
		if !editSelect.IsNull() {
			editSelect.Call("addEventListener", "change", js.FuncOf(func(this js.Value, args []js.Value) any {
				loadIntoEditor()
				return nil
			}))
		}
		if !editSave.IsNull() {
			editSave.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
				if editSelect.IsNull() || editArea.IsNull() {
					return nil
				}
				name := editSelect.Get("value").String()
				if name == "" {
					name = "tuning.json"
				}
				txt := editArea.Get("value").String()
				if !json.Valid([]byte(txt)) {
					if !editStatus.IsNull() {
						editStatus.Set("textContent", "Invalid JSON — not saved")
					}
					return nil
				}
				ls := js.Global().Get("localStorage")
				if !ls.IsNull() && !ls.IsUndefined() {
					ls.Call("setItem", "data:"+name, txt)
				}
				if !editStatus.IsNull() {
					editStatus.Set("textContent", "Saved "+name+" to overlay — reload to apply")
				}
				return nil
			}))
		}
		if !toggleBtn.IsNull() && !editorEl.IsNull() {
			toggleBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
				cls := editorEl.Get("classList")
				if cls.Call("contains", "open").Bool() {
					cls.Call("remove", "open")
				} else {
					cls.Call("add", "open")
					loadIntoEditor()
				}
				return nil
			}))
		}
	}
	setupShellControls()
	// Helper to render log/hints into separate divs between map and hints, mirroring terminal y=H status.
	renderLogHints := func(frame game.Frame) {
		if logDiv.IsNull() || hintsDiv.IsNull() {
			return
		}
		logDiv.Set("innerHTML", buildLogHTML(frame.Log))
		hintsDiv.Set("textContent", frame.Hints)
	}
	state := stateMenu
	menu := &game.MainMenuState{Selected: 0}
	var cs *game.CharSelectState
	var rs *game.RaceSelectState
	var g *game.Game
	wizardState := &game.WizardState{Selected: 0}
	var wizardAddCS *game.CharSelectState
	wizardRemoveIdx := 0
	useSelected := 0
	useMemberSelected := 0
	useMemberAppearance := ""
	useMemberTitle := ""
	useMemberMode := ""
	throwSelected := 0
	seedState := &game.SeedEntryState{}
	var classPick *game.ClassPickState
	var racePick *game.RacePickState
	slotRace := ""
	slotClass := ""
	var nameEntry *game.NameEntryState
	drafts := []game.DraftMember{}
	reviewCursor := 0
	scoresSelected := 0
	renderMenu := func() {
		frame := game.RenderMainMenu(tuning, menu.Selected)
		html := buildHTML(frame, tuning)
		gameDiv.Set("innerHTML", html)
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
	}
	renderScores := func() {
		frame := game.RenderScoresScreen(tuning, scoresSelected)
		html := buildHTML(frame, tuning)
		gameDiv.Set("innerHTML", html)
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
	renderRaceSelect := func() {
		if rs == nil {
			return
		}
		frame := game.RenderRaceSelect(tuning, rs)
		gameDiv.Set("innerHTML", buildHTML(frame, tuning))
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
	}
	renderSeed := func() {
		frame := game.RenderSeedEntry(tuning, seedState)
		gameDiv.Set("innerHTML", buildHTML(frame, tuning))
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
	}
	renderClassPick := func() {
		if classPick == nil {
			return
		}
		frame := game.RenderClassPick(tuning, classPick)
		gameDiv.Set("innerHTML", buildHTML(frame, tuning))
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
	}
	renderCreationRace := func() {
		if racePick == nil {
			return
		}
		frame := game.RenderRacePick(tuning, racePick)
		gameDiv.Set("innerHTML", buildHTML(frame, tuning))
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
	}
	renderNameEntry := func() {
		if nameEntry == nil {
			return
		}
		frame := game.RenderNameEntry(tuning, nameEntry.Class, nameEntry.Race, nameEntry.Text)
		gameDiv.Set("innerHTML", buildHTML(frame, tuning))
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
	}
	renderReview := func() {
		frame := game.RenderReview(tuning, drafts, reviewCursor)
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
				if err := game.DeleteSave(); err != nil {
					log.Printf("save delete failed: %v", err)
				}
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
	renderUseMenu := func() {
		if g == nil {
			return
		}
		frame := g.RenderUseMenu(useSelected)
		gameDiv.Set("innerHTML", buildHTML(frame, tuning))
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
	}
	renderUseMember := func() {
		if g == nil {
			return
		}
		frame := g.RenderUseMemberMenu(useMemberTitle, useMemberMode == "partyChoice", useMemberSelected)
		gameDiv.Set("innerHTML", buildHTML(frame, tuning))
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
	}
	renderThrowMenu := func() {
		if g == nil {
			return
		}
		frame := g.RenderThrowMenu(throwSelected)
		gameDiv.Set("innerHTML", buildHTML(frame, tuning))
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
	}
	renderMerchant := func() {
		if g == nil {
			return
		}
		frame := g.RenderMerchantMenu()
		gameDiv.Set("innerHTML", buildHTML(frame, tuning))
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
	}
	renderShrine := func() {
		if g == nil {
			return
		}
		frame := g.RenderShrineMenu()
		gameDiv.Set("innerHTML", buildHTML(frame, tuning))
		statusDiv.Set("textContent", frame.Status)
		renderLogHints(frame)
	}
	renderMenu()

	js.Global().Get("window").Call("addEventListener", "beforeunload", js.FuncOf(func(this js.Value, args []js.Value) any {
		if g != nil && !g.Over {
			_ = game.Save(g)
		}
		return nil
	}))
	var keyHandler js.Func
	keyHandler = js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		// Allow normal typing when an editor control is focused: do not intercept keys.
		if !doc.IsNull() && !doc.IsUndefined() {
			active := doc.Get("activeElement")
			if !active.IsNull() && !active.IsUndefined() {
				tagVal := active.Get("tagName")
				if !tagVal.IsNull() && !tagVal.IsUndefined() {
					tag := strings.ToUpper(tagVal.String())
					if tag == "INPUT" || tag == "TEXTAREA" || tag == "SELECT" {
						return nil
					}
				}
				ce := active.Get("isContentEditable")
				if !ce.IsNull() && !ce.IsUndefined() && ce.Truthy() {
					return nil
				}
				idVal := active.Get("id")
				if !idVal.IsNull() && !idVal.IsUndefined() {
					id := idVal.String()
					if id == "editArea" || id == "editSelect" || id == "dataZip" {
						return nil
					}
				}
				editorEl := doc.Call("getElementById", "editor")
				if !editorEl.IsNull() && !editorEl.IsUndefined() {
					cls := editorEl.Get("classList")
					if !cls.IsNull() && !cls.IsUndefined() && cls.Call("contains", "open").Bool() {
						containsFn := editorEl.Get("contains")
						if !containsFn.IsUndefined() {
							if editorEl.Call("contains", active).Bool() {
								return nil
							}
						} else {
							closestFn := active.Get("closest")
							if !closestFn.IsUndefined() {
								res := active.Call("closest", "#editor")
								if !res.IsNull() && !res.IsUndefined() {
									return nil
								}
							}
						}
					}
				}
			}
		}
		key := e.Get("key").String()
		code := e.Get("code").String()
		k := game.NormalizeKey(key, code)
		switch k {
		case game.KeyUp, game.KeyDown, game.KeyLeft, game.KeyRight, game.KeyUpLeft, game.KeyUpRight, game.KeyDownLeft, game.KeyDownRight, game.KeyWait, game.KeyRest, game.KeyUse:
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
				opts := game.GetMainMenuOptions() // HasSave gated
				if menu.Selected < 0 || menu.Selected >= len(opts) {
					break
				}
				opt := opts[menu.Selected]
				switch opt {
				case "New Game":
					seedState = &game.SeedEntryState{}
					drafts = []game.DraftMember{}
					reviewCursor = 0
					state = stateSeed
					renderSeed()
				case "Scores":
					scoresSelected = 0
					state = stateScores
					renderScores()
				case "Exit":
					// Exit not applicable on web; just stay
				case "Load game":
					if lg, err := game.Load(); err == nil && lg != nil {
						g = lg
						state = statePlaying
						renderGame()
					}
				}
			case game.KeyQuit:
				// No exit on web
			case game.KeyHelp:
				doc.Call("getElementById", "game").Set("innerHTML", buildHTML(game.RenderHelpOverlayTuning(tuning), tuning))
				statusDiv.Set("innerHTML", "Help | ?/Esc to close")
				hintsDiv.Set("innerHTML", "Esc / Enter / ? : close help")
				state = stateMenuHelp
			}
		case stateMenuHelp:
				switch k {
				case game.KeyQuit, game.KeyEnter, game.KeyHelp:
					state = stateMenu
					renderMenu()
				default:
					doc.Call("getElementById", "game").Set("innerHTML", buildHTML(game.RenderHelpOverlayTuning(tuning), tuning))
				}
			case stateScores:
			switch k {
			case game.KeyUp:
				if scoresSelected > 0 {
					scoresSelected--
				}
				renderScores()
			case game.KeyDown:
				if sb, err := game.LoadScoreboard(); err == nil && sb != nil && scoresSelected < len(sb.Entries)-1 {
					scoresSelected++
				}
				renderScores()
			case game.KeyEnter, game.KeyQuit:
				state = stateMenu
				renderMenu()
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
					var err error
					rs, err = game.NewRaceSelect(cs.Picks)
					if err != nil {
						seed := time.Now().UnixNano()
						g = game.NewGameWithClasses(seed, tuning, cs.Picks)
						state = statePlaying
						renderGame()
						break
					}
					state = stateRaceSelect
					renderRaceSelect()
				} else {
					cs.Select()
					renderCharSelect()
					if cs.Done() {
						// show but wait for Enter to go to race
					}
				}
			case game.KeyQuit:
				if cs.Back() {
					state = stateMenu
					renderMenu()
				} else {
					renderCharSelect()
				}
			default:
				if key >= "1" && key <= "9" {
					idx := int(key[0] - '1')
					if idx >= 0 && idx < len(cs.Classes) {
						cs.Cursor = idx
						cs.Select()
						renderCharSelect()
						if cs.Done() {
							var err error
							rs, err = game.NewRaceSelect(cs.Picks)
							if err == nil {
								state = stateRaceSelect
								renderRaceSelect()
							}
						}
					}
				}
			}
		case stateRaceSelect:
			if rs == nil {
				state = stateCharSelect
				renderCharSelect()
				break
			}
			switch k {
			case game.KeyUp:
				rs.Move(-1)
				renderRaceSelect()
			case game.KeyDown:
				rs.Move(1)
				renderRaceSelect()
			case game.KeyEnter:
				if rs.Done() {
					seed := time.Now().UnixNano()
					g = game.NewGameWithClassesAndRaces(seed, tuning, rs.Classes, rs.Picks)
					state = statePlaying
					renderGame()
				} else {
					rs.Select()
					renderRaceSelect()
				}
			case game.KeyQuit:
				if rs.Back() {
					state = stateCharSelect
					renderCharSelect()
				} else {
					renderRaceSelect()
				}
			default:
				if key >= "1" && key <= "7" {
					idx := int(key[0] - '1')
					if idx >= 0 && idx < len(rs.Races) {
						rs.Cursor = idx
						if !rs.Done() {
							rs.Select()
						}
						renderRaceSelect()
					}
				}
			}
		case stateSeed:
			if key == "Backspace" {
				e.Call("preventDefault")
				if seedState != nil {
					seedState.Backspace()
					renderSeed()
				}
				break
			}
			if len([]rune(key)) == 1 {
				if seedState != nil {
					for _, r := range key {
						seedState.AppendRune(r)
					}
					renderSeed()
				}
				break
			}
			switch k {
			case game.KeyEnter:
				racePick = game.NewRacePickState(len(drafts), len(drafts))
				if racePick == nil {
					renderSeed()
					break
				}
				state = stateCreationRace
				renderCreationRace()
			case game.KeyQuit:
				state = stateMenu
				renderMenu()
			default:
				renderSeed()
			}
		case stateCreationClass:
			if classPick == nil {
				state = stateSeed
				renderSeed()
				break
			}
			switch k {
			case game.KeyUp:
				classPick.Move(-1)
				renderClassPick()
			case game.KeyDown:
				classPick.Move(1)
				renderClassPick()
			case game.KeyEnter:
				slotClass = classPick.Choice()
				if slotClass == "" {
					renderClassPick()
					break
				}
				nameEntry = &game.NameEntryState{Class: slotClass, Race: slotRace}
				state = stateCreationName
				renderNameEntry()
			case game.KeyQuit:
				state = stateCreationRace
				renderCreationRace()
			default:
				renderClassPick()
			}
		case stateCreationRace:
			if racePick == nil {
				state = stateSeed
				renderSeed()
				break
			}
			switch k {
			case game.KeyUp:
				racePick.Move(-1)
				renderCreationRace()
			case game.KeyDown:
				racePick.Move(1)
				renderCreationRace()
			case game.KeyEnter:
				slotRace = racePick.Choice()
				if slotRace == "" {
					renderCreationRace()
					break
				}
				var err error
				classPick, err = game.NewClassPickState(len(drafts), len(drafts))
				if err != nil || classPick == nil {
					renderCreationRace()
					break
				}
				state = stateCreationClass
				renderClassPick()
			case game.KeyQuit:
				state = stateSeed
				renderSeed()
			default:
				renderCreationRace()
			}
		case stateCreationName:
			if nameEntry == nil {
				state = stateCreationClass
				renderClassPick()
				break
			}
			if key == "Backspace" {
				e.Call("preventDefault")
				nameEntry.Backspace()
				renderNameEntry()
				break
			}
			if len([]rune(key)) == 1 {
				for _, r := range key {
					nameEntry.AppendRune(r)
				}
				renderNameEntry()
				break
			}
			switch k {
			case game.KeyEnter:
				drafts = append(drafts, game.DraftMember{Class: nameEntry.Class, Race: nameEntry.Race, Name: nameEntry.Text})
				reviewCursor = 0
				state = stateReview
				renderReview()
			case game.KeyQuit:
				state = stateCreationClass
				renderClassPick()
			default:
				renderNameEntry()
			}
		case stateReview:
			rows := game.ReviewRows(drafts)
			switch k {
			case game.KeyUp:
				if len(rows) > 0 {
					reviewCursor--
					if reviewCursor < 0 {
						reviewCursor = len(rows) - 1
					}
				}
				renderReview()
			case game.KeyDown:
				if len(rows) > 0 {
					reviewCursor++
					if reviewCursor >= len(rows) {
						reviewCursor = 0
					}
				}
				renderReview()
			case game.KeyEnter:
				if len(rows) == 0 || reviewCursor < 0 || reviewCursor >= len(rows) {
					renderReview()
					break
				}
				row := rows[reviewCursor]
				switch row.Action {
				case "discard":
					if row.Index >= 0 && row.Index < len(drafts) {
						drafts = append(drafts[:row.Index], drafts[row.Index+1:]...)
					}
					if reviewCursor >= len(game.ReviewRows(drafts)) {
						reviewCursor = len(game.ReviewRows(drafts)) - 1
					}
					if reviewCursor < 0 {
						reviewCursor = 0
					}
					renderReview()
				case "add":
					racePick = game.NewRacePickState(len(drafts), len(drafts))
					if racePick == nil {
						renderReview()
						break
					}
					state = stateCreationRace
					renderCreationRace()
				case "begin":
					classes := make([]string, len(drafts))
					races := make([]string, len(drafts))
					names := make([]string, len(drafts))
					for i, d := range drafts {
						classes[i], races[i], names[i] = d.Class, d.Race, d.Name
					}
					seed := seedState.SeedOr(time.Now().UnixNano())
					g = game.NewGameFull(seed, tuning, classes, races, names)
					state = statePlaying
					renderGame()
				default:
					renderReview()
				}
			case game.KeyQuit:
				state = stateMenu
				renderMenu()
			default:
				renderReview()
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
			if k == game.KeyUse && (g.Look == nil || !g.Look.Active) && !g.Over && !g.Quit && !g.UsePending.Active && !g.ThrowPending.Active {
				if g.TryUseForge() {
					g.EndPlayerTurn("")
					renderGame()
					if g.LevelUpPending != nil {
						renderLevelUp()
					}
					break
				}
				entries := g.InventoryUseEntries()
				if len(entries) == 0 {
					g.Logf("No potions or scrolls to use.")
					renderGame()
					break
				}
				useSelected = 0
				state = stateUseInventory
				renderUseMenu()
				break
			}
			if k == game.KeyThrow && (g.Look == nil || !g.Look.Active) && !g.ThrowPending.Active && !g.UsePending.Active && !g.Over && !g.Quit {
				entries := g.InventoryEntriesByKind("potion")
				if len(entries) == 0 {
					g.Logf("No potions to throw.")
					renderGame()
					break
				}
				throwSelected = 0
				state = stateThrowMenu
				renderThrowMenu()
				break
			}
			g.HandleKey(k)
			if g.Merchant.Active {
				state = stateMerchant
				renderMerchant()
				break
			}
			if g.Shrine.Active {
				state = stateShrine
				renderShrine()
				break
			}
			if g.HelpActive {
				renderHelp()
			} else {
				renderGame()
			}
			if g.Quit {
				quitToMenu(&g, &state, renderMenu)
			} else if g.Over {
				if err := game.DeleteSave(); err != nil {
					log.Printf("save delete failed: %v", err)
				}
				if k == game.KeyQuit || k == game.KeyEnter {
					state = stateMenu
					g = nil
					renderMenu()
				}
			}
		case stateUseInventory:
			if g == nil {
				state = statePlaying
				renderGame()
				break
			}
			entries := g.InventoryUseEntries()
			switch k {
			case game.KeyUp:
				if len(entries) > 0 {
					useSelected--
					if useSelected < 0 {
						useSelected = len(entries) - 1
					}
				}
				renderUseMenu()
			case game.KeyDown:
				if len(entries) > 0 {
					useSelected++
					if useSelected >= len(entries) {
						useSelected = 0
					}
				}
				renderUseMenu()
			case game.KeyQuit:
				state = statePlaying
				renderGame()
			case game.KeyEnter:
				if len(entries) > 0 && useSelected >= 0 && useSelected < len(entries) {
					e := entries[useSelected]
					useMemberMode = game.UseTargetMode(e.Kind, e.Appearance)
					if useMemberMode == "tile" {
						g.StartUse(e.Appearance, e.Kind)
						state = stateUseTarget
						renderGame()
					} else {
						useMemberAppearance = e.Appearance
						useMemberTitle = e.DisplayName
						useMemberSelected = 0
						state = stateUseMember
						renderUseMember()
					}
				} else {
					state = statePlaying
					renderGame()
				}
			default:
				renderUseMenu()
			}
		case stateUseMember:
			if g == nil {
				state = statePlaying
				renderGame()
				break
			}
			allowParty := useMemberMode == "partyChoice"
			_, useMemberRows := g.UseMemberOptions(allowParty)
			switch k {
			case game.KeyUp:
				if len(useMemberRows) > 0 {
					useMemberSelected--
					if useMemberSelected < 0 {
						useMemberSelected = len(useMemberRows) - 1
					}
				}
				renderUseMember()
			case game.KeyDown:
				if len(useMemberRows) > 0 {
					useMemberSelected++
					if useMemberSelected >= len(useMemberRows) {
						useMemberSelected = 0
					}
				}
				renderUseMember()
			case game.KeyQuit:
				state = stateUseInventory
				renderUseMenu()
			case game.KeyEnter:
				if len(useMemberRows) > 0 && useMemberSelected >= 0 && useMemberSelected < len(useMemberRows) {
					g.TryUseAppearanceOnMember(useMemberAppearance, useMemberRows[useMemberSelected])
					state = statePlaying
					renderGame()
					if g.LevelUpPending != nil {
						renderLevelUp()
					}
					if g.Quit {
						quitToMenu(&g, &state, renderMenu)
					} else if g.Over {
						if err := game.DeleteSave(); err != nil {
							log.Printf("save delete failed: %v", err)
						}
						state = stateMenu
						g = nil
						renderMenu()
					}
				} else {
					state = statePlaying
					renderGame()
				}
			default:
				renderUseMember()
			}
		case stateUseTarget:
			if g == nil {
				state = statePlaying
				renderGame()
				break
			}
			switch k {
			case game.KeyQuit:
				g.CancelUse()
				g.Logf("Cancelled use.")
				state = statePlaying
				renderGame()
			case game.KeyEnter:
				g.UseAt(g.UsePending.Cursor)
				state = statePlaying
				renderGame()
				if g.LevelUpPending != nil {
					renderLevelUp()
				}
				if g.Quit {
					quitToMenu(&g, &state, renderMenu)
				} else if g.Over {
				if err := game.DeleteSave(); err != nil {
					log.Printf("save delete failed: %v", err)
				}
					if k == game.KeyQuit || k == game.KeyEnter {
						state = stateMenu
						g = nil
						renderMenu()
					}
				}
			default:
				handledTurn := g.HandleKey(k)
				renderGame()
				if handledTurn {
					state = statePlaying
					if g.LevelUpPending != nil {
						renderLevelUp()
					}
					if g.Quit {
						quitToMenu(&g, &state, renderMenu)
					} else if g.Over {
				if err := game.DeleteSave(); err != nil {
					log.Printf("save delete failed: %v", err)
				}
						if k == game.KeyQuit || k == game.KeyEnter {
							state = stateMenu
							g = nil
							renderMenu()
						}
					}
				}
			}
		case stateThrowMenu:
			if g == nil {
				state = statePlaying
				renderGame()
				break
			}
			entries := g.InventoryEntriesByKind("potion")
			switch k {
			case game.KeyUp:
				if len(entries) > 0 {
					throwSelected--
					if throwSelected < 0 {
						throwSelected = len(entries) - 1
					}
				}
				renderThrowMenu()
			case game.KeyDown:
				if len(entries) > 0 {
					throwSelected++
					if throwSelected >= len(entries) {
						throwSelected = 0
					}
				}
				renderThrowMenu()
			case game.KeyQuit:
				state = statePlaying
				renderGame()
			case game.KeyEnter:
				if len(entries) > 0 && throwSelected >= 0 && throwSelected < len(entries) {
					appearance := entries[throwSelected].Appearance
					g.StartThrow(appearance)
					state = stateThrowCursor
					renderGame()
				} else {
					state = statePlaying
					renderGame()
				}
			default:
				renderThrowMenu()
			}
		case stateThrowCursor:
			if g == nil {
				state = statePlaying
				renderGame()
				break
			}
			switch k {
			case game.KeyQuit:
				g.CancelThrow()
				g.Logf("Cancelled throw.")
				state = statePlaying
				renderGame()
			case game.KeyEnter:
				g.ThrowAt(g.ThrowPending.Cursor)
				state = statePlaying
				renderGame()
				if g.LevelUpPending != nil {
					renderLevelUp()
				}
				if g.Quit {
					quitToMenu(&g, &state, renderMenu)
				} else if g.Over {
				if err := game.DeleteSave(); err != nil {
					log.Printf("save delete failed: %v", err)
				}
					if k == game.KeyQuit || k == game.KeyEnter {
						state = stateMenu
						g = nil
						renderMenu()
					}
				}
			default:
				handledTurn := g.HandleKey(k)
				renderGame()
				if handledTurn {
					state = statePlaying
					if g.LevelUpPending != nil {
						renderLevelUp()
					}
					if g.Quit {
						quitToMenu(&g, &state, renderMenu)
					} else if g.Over {
				if err := game.DeleteSave(); err != nil {
					log.Printf("save delete failed: %v", err)
				}
						if k == game.KeyQuit || k == game.KeyEnter {
							state = stateMenu
							g = nil
							renderMenu()
						}
					}
				}
			}
		case stateMerchant:
			if g == nil {
				state = statePlaying
				renderGame()
				break
			}
			switch k {
			case game.KeyUp:
				if g.Merchant.Active && len(g.Merchant.Wares) > 0 {
					g.Merchant.Selected--
					if g.Merchant.Selected < 0 {
						g.Merchant.Selected = len(g.Merchant.Wares) - 1
					}
				}
				renderMerchant()
			case game.KeyDown:
				if g.Merchant.Active && len(g.Merchant.Wares) > 0 {
					g.Merchant.Selected++
					if g.Merchant.Selected >= len(g.Merchant.Wares) {
						g.Merchant.Selected = 0
					}
				}
				renderMerchant()
			case game.KeyQuit:
				g.CancelMerchant()
				state = statePlaying
				renderGame()
			case game.KeyEnter:
				if g.Merchant.Active {
					sel := g.Merchant.Selected
					if g.BuySelectedMerchant(sel) {
						g.EndPlayerTurn("")
						state = statePlaying
						renderGame()
						if g.LevelUpPending != nil {
							renderLevelUp()
						}
						if g.Quit {
							quitToMenu(&g, &state, renderMenu)
						} else if g.Over {
				if err := game.DeleteSave(); err != nil {
					log.Printf("save delete failed: %v", err)
				}
							if k == game.KeyQuit || k == game.KeyEnter {
								state = stateMenu
								g = nil
								renderMenu()
							}
						}
					} else {
						renderMerchant()
					}
				} else {
					state = statePlaying
					renderGame()
				}
			default:
				renderMerchant()
			}
		case stateShrine:
			if g == nil {
				state = statePlaying
				renderGame()
				break
			}
			switch k {
			case game.KeyUp:
				if g.Shrine.Active {
					g.Shrine.Selected--
					if g.Shrine.Selected < 0 {
						g.Shrine.Selected = 3
					}
				}
				renderShrine()
			case game.KeyDown:
				if g.Shrine.Active {
					g.Shrine.Selected++
					if g.Shrine.Selected > 3 {
						g.Shrine.Selected = 0
					}
				}
				renderShrine()
			case game.KeyQuit:
				g.CancelShrine()
				state = statePlaying
				renderGame()
			case game.KeyEnter:
				if g.Shrine.Active {
					sel := g.Shrine.Selected
					if sel == 3 {
						g.CancelShrine()
						state = statePlaying
						renderGame()
					} else {
						if g.ExecuteShrineChoice(sel) {
							g.EndPlayerTurn("")
							state = statePlaying
							renderGame()
							if g.LevelUpPending != nil {
								renderLevelUp()
							}
							if g.Quit {
								quitToMenu(&g, &state, renderMenu)
							} else if g.Over {
				if err := game.DeleteSave(); err != nil {
					log.Printf("save delete failed: %v", err)
				}
								if k == game.KeyQuit || k == game.KeyEnter {
									state = stateMenu
									g = nil
									renderMenu()
								}
							}
						} else {
							renderShrine()
						}
					}
				} else {
					state = statePlaying
					renderGame()
				}
			default:
				renderShrine()
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
				case "resurrect":
					wizardRemoveIdx = g.Party.FirstDead()
					if wizardRemoveIdx < 0 {
						g.Logf("Wizard: Resurrect - no fallen pilgrims")
						state = statePlaying
						renderGame()
						break
					}
					state = stateWizardResurrectMember
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
		case stateWizardResurrectMember:
			switch k {
			case game.KeyUp:
				for range g.Party.Members {
					wizardRemoveIdx--
					if wizardRemoveIdx < 0 {
						wizardRemoveIdx = len(g.Party.Members) - 1
					}
					if !g.Party.Members[wizardRemoveIdx].IsAlive() {
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
					if !g.Party.Members[wizardRemoveIdx].IsAlive() {
						break
					}
				}
				g.Party.Selected = wizardRemoveIdx
				renderGame()
			case game.KeyEnter:
				g.WizardResurrectMember(wizardRemoveIdx)
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

type appState int
const (
	stateMenu appState = iota
	stateCharSelect
	stateRaceSelect
	statePlaying
	stateWizard
	stateWizardAddMember
	stateWizardRemoveMember
	stateWizardResurrectMember
	stateUseInventory
	stateUseTarget
	stateUseMember
	stateThrowMenu
	stateThrowCursor
	stateMerchant
	stateShrine
	stateScores
	stateMenuHelp
	stateSeed
	stateCreationClass
	stateCreationRace
	stateCreationName
	stateReview
)

// quitToMenu applies the shared save policy and returns to the main menu,
// logging save errors to the browser console instead of swallowing them.
func quitToMenu(g **game.Game, state *appState, showMenu func()) {
	if *g != nil {
		if _, err := (*g).PostAction(); err != nil {
			log.Printf("menu save failed: %v", err)
		}
	}
	*g = nil
	*state = stateMenu
	showMenu()
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
	html := `<div style="display:flex;gap:16px;align-items:flex-start"><div style="font-family:var(--font-monospace);font-size:var(--map-cell);line-height:var(--map-line-height);white-space:pre">`
	for y := range frame.H {
		html += `<div style="height:var(--map-cell);line-height:var(--map-line-height);overflow:hidden;white-space:pre">`
		for x := range frame.W {
			cell := frame.Cells[y][x]
			col := colorForToken(cell.FG)
			ch := string(cell.Glyph)
			bgStyle := ";display:inline-block;width:1ch;line-height:1;overflow:hidden;vertical-align:top"
			if cell.BG == "cursor" {
				bgStyle += ";background:var(--gold-bright);color:var(--bg);font-weight:bold"
				html += `<span style="color:` + col + bgStyle + `">` + esc(ch) + `</span>`
			} else {
				html += `<span style="color:` + col + bgStyle + `">` + esc(ch) + `</span>`
			}
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
		html += `</div>`
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

func buildScoreboardHTML() string {
	sb, err := game.LoadScoreboard()
	if err != nil || sb == nil || len(sb.Entries) == 0 {
		return `<div style="margin-top:12px;padding:8px 12px;border:1px solid var(--gray-2);color:var(--gray-1);font-family:var(--font-monospace);max-width:800px">No scores yet — survive the temple!</div>`
	}
	entries := sb.GetHighScores(5)
	if len(entries) == 0 {
		return `<div style="margin-top:12px;padding:8px 12px;border:1px solid var(--gray-2);color:var(--gray-1);font-family:var(--font-monospace);max-width:800px">No scores yet — survive the temple!</div>`
	}
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
			case '"':
				out += "&quot;"
			default:
				out += string(ch)
			}
		}
		return out
	}
	html := `<div style="margin-top:12px;max-width:900px;overflow-x:auto"><table style="width:100%;border-collapse:collapse;font-family:var(--font-monospace);font-size:13px">`
	html += `<thead><tr style="color:var(--gold);text-align:left;border-bottom:1px solid var(--gray-2)">`
	html += `<th style="padding:4px 8px">#</th><th style="padding:4px 8px">Score</th><th style="padding:4px 8px">Lv</th><th style="padding:4px 8px">Gold</th><th style="padding:4px 8px">Depth</th><th style="padding:4px 8px">Seed</th><th style="padding:4px 8px">Result</th><th style="padding:4px 8px">Members</th>`
	html += `</tr></thead><tbody>`
	for i, e := range entries {
		result := e.CauseOfDeath
		if e.Victory {
			result = "Victory"
		}
		if result == "" {
			result = "Unknown"
		}
		members := esc(game.MembersSummary(e))
		rowColor := "var(--gray-1)"
		if e.Victory {
			rowColor = "var(--gold-bright)"
		}
		html += fmt.Sprintf(`<tr style="color:%s;border-bottom:1px solid rgba(255,255,255,0.06)"><td style="padding:4px 8px">%d</td><td style="padding:4px 8px">%d</td><td style="padding:4px 8px">%d</td><td style="padding:4px 8px">%d</td><td style="padding:4px 8px">%d</td><td style="padding:4px 8px">%d</td><td style="padding:4px 8px">%s</td><td style="padding:4px 8px">%s</td></tr>`, rowColor, i+1, e.Score, e.PartyLevel, e.Gold, e.DepthReached, e.Seed, esc(result), members)
	}
	html += `</tbody></table></div>`
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

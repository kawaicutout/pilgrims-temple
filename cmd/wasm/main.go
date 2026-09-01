//go:build js && wasm

package main

import (
	"syscall/js"
	"time"

	"partyrogue/game"
)

func main() {
	tuning, err := game.LoadTuning()
	if err != nil {
		panic(err)
	}
	seed := time.Now().UnixNano()
	g := game.NewGame(seed, tuning)

	doc := js.Global().Get("document")
	gameDiv := doc.Call("getElementById", "game")
	statusDiv := doc.Call("getElementById", "status")
	if gameDiv.IsNull() {
		body := doc.Get("body")
		gameDiv = doc.Call("createElement", "div")
		gameDiv.Set("id", "game")
		body.Call("appendChild", gameDiv)
		statusDiv = doc.Call("createElement", "div")
		statusDiv.Set("id", "status")
		body.Call("appendChild", statusDiv)
	}

	render := func() {
		frame := g.Render()
		html := buildHTML(frame, tuning)
		gameDiv.Set("innerHTML", html)
		// Status bar is separate element for parity with terminal's y=H line
		statusDiv.Set("textContent", frame.Status)
		if g.Over {
			if g.Won {
				statusDiv.Set("textContent", "VICTORY! Seed "+itoa(g.Seed)+" - refresh to play again.")
			} else {
				statusDiv.Set("textContent", "YOU DIED. Seed "+itoa(g.Seed)+" - refresh to play again.")
			}
		}
	}
	render()

	var keyHandler js.Func
	keyHandler = js.FuncOf(func(this js.Value, args []js.Value) any {
		if g.Over {
			return nil
		}
		e := args[0]
		key := e.Get("key").String()
		code := e.Get("code").String()
		k := game.NormalizeKey(key, code)
		switch k {
		case game.KeyUp, game.KeyDown, game.KeyLeft, game.KeyRight, game.KeyUpLeft, game.KeyUpRight, game.KeyDownLeft, game.KeyDownRight, game.KeyWait:
			e.Call("preventDefault")
		}
		g.HandleKey(k)
		render()
		return nil
	})
	js.Global().Get("document").Call("addEventListener", "keydown", keyHandler)
	select {}
}

func itoa(v int64) string {
	// avoid fmt import for tiny wasm
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
	// Panel width for parity with terminal: MinCols - MapWidth - 1 gap
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
			// Truncate to panelMax with ellipsis, same as terminal's w - panelX logic
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
			if len(line) > 0 && line[0] == '>' {
				html += `<span style="color:var(--gold-bright)"> ` + esc(line) + `</span>`
			} else {
				html += `<span style="color:var(--gray-1)"> ` + esc(line) + `</span>`
			}
		}
		html += "\n"
	}
	html += `</div></div>`
	// Log (8 lines) - parity with terminal's log at y=H+1
	html += `<div style="color:var(--gray-1);margin-top:4px;font-family:var(--font-monospace);white-space:pre">`
	for _, line := range frame.Log {
		if line == "" {
			html += "<br>"
		} else {
			html += esc(line) + "<br>"
		}
	}
	html += `</div>`
	// Hints - parity with terminal's bottom row
	html += `<div style="color:var(--gray-2);margin-top:8px;font-family:var(--font-monospace)">` + esc(frame.Hints) + `</div>`
	return html
}

func colorForToken(token string) string {
	switch token {
	case "player":
		return "var(--gold-bright)"
	case "enemy":
		return "var(--red-bright)"
	case "wall":
		return "var(--wall)"
	case "floor":
		return "var(--floor)"
	case "gold":
		return "var(--gold)"
	case "gray-3":
		return "var(--gray-2)"
	case "bg":
		return "var(--fg)"
	default:
		return "var(--fg)"
	}
}

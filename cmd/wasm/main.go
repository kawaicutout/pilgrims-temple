//go:build js && wasm

package main

import (
	"fmt"
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
		// Create fallback elements
		body := doc.Get("body")
		gameDiv = doc.Call("createElement", "div")
		gameDiv.Set("id", "game")
		body.Call("appendChild", gameDiv)
		statusDiv = doc.Call("createElement", "div")
		statusDiv.Set("id", "status")
		body.Call("appendChild", statusDiv)
	}

	// Apply tokens-backed styling via CSS already in tokens.css.
	render := func() {
		frame := g.Render()
		// Build HTML grid: map + panel side-by-side, then status/log/hints
		// Use monospaced pre blocks.
		html := buildHTML(frame, tuning)
		gameDiv.Set("innerHTML", html)
		statusDiv.Set("textContent", fmt.Sprintf("Seed %d | Floor %d/%d | Turn %d | %s", g.Seed, g.Floor+1, tuning.Floors, g.Turn, frame.Status))
		if g.Over {
			if g.Won {
				statusDiv.Set("textContent", fmt.Sprintf("VICTORY! Seed %d — refresh to play again.", g.Seed))
			} else {
				statusDiv.Set("textContent", fmt.Sprintf("YOU DIED. Seed %d — refresh to play again.", g.Seed))
			}
		}
		_ = statusDiv
	}

	render()

	// Keyboard handling
	var keyHandler js.Func
	keyHandler = js.FuncOf(func(this js.Value, args []js.Value) any {
		if g.Over {
			return nil
		}
		e := args[0]
		key := e.Get("key").String()
		code := e.Get("code").String()
		// Ignore if modifier held? Still allow.
		k := game.NormalizeKey(key, code)
		// Swallow movement keys to prevent scroll
		switch k {
		case game.KeyUp, game.KeyDown, game.KeyLeft, game.KeyRight, game.KeyUpLeft, game.KeyUpRight, game.KeyDownLeft, game.KeyDownRight, game.KeyWait:
			e.Call("preventDefault")
		}
		g.HandleKey(k)
		render()
		return nil
	})
	js.Global().Get("document").Call("addEventListener", "keydown", keyHandler)

	// Keep alive
	select {}
}

func buildHTML(frame game.Frame, tuning game.Tuning) string {
	// Colors matching tokens.css
	// Use inline spans for per-cell color. Map uses truecolor tokens.
	esc := func(s string) string {
		// minimal escape
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
	_ = esc
	html := `<div style="display:flex;gap:16px;align-items:flex-start"><div style="font-family:var(--font-monospace);line-height:var(--map-line-height);white-space:pre">`
	for y := range frame.H {
		for x := range frame.W {
			cell := frame.Cells[y][x]
			col := colorForToken(cell.FG)
			ch := string(cell.Glyph)
			if ch == " " {
				ch = " "
			}
			// Use span per cell for truecolor; wall/floor etc.
			html += fmt.Sprintf(`<span style="color:%s">%s</span>`, col, esc(ch))
		}
		// Append panel line to the right (inline)
		if y < len(frame.Panel) {
			line := frame.Panel[y]
			if len(line) > 0 && line[0] == '>' {
				html += fmt.Sprintf(`<span style="color:var(--gold-bright)"> %s</span>`, esc(line))
			} else {
				html += fmt.Sprintf(`<span style="color:var(--gray-1)"> %s</span>`, esc(line))
			}
		}
		html += "\n"
	}
	html += `</div></div>`
	// Status + log + hints
	html += fmt.Sprintf(`<div style="color:var(--gold);margin-top:8px">%s</div>`, esc(frame.Status))
	html += `<div style="color:var(--gray-1);margin-top:4px">`
	for _, line := range frame.Log {
		html += esc(line) + "<br>"
	}
	html += `</div>`
	html += fmt.Sprintf(`<div style="color:var(--gray-2);margin-top:8px">%s</div>`, esc(frame.Hints))
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

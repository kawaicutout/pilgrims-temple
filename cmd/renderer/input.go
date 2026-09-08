package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"partyrogue/game"
)

// keyEvent is one discrete key press mapped to game key space.
type keyEvent struct {
	key  string // raw, like tcellKeyToRaw
	code string // code, like browser code
}

// specialKeys mirrors tcellKeyToRaw and the web key codes so game.NormalizeKey
// resolves identically on all three frontends.
var specialKeys = []struct {
	eb   ebiten.Key
	key  string
	code string
}{
	{ebiten.KeyArrowUp, "ArrowUp", "ArrowUp"},
	{ebiten.KeyArrowDown, "ArrowDown", "ArrowDown"},
	{ebiten.KeyArrowLeft, "ArrowLeft", "ArrowLeft"},
	{ebiten.KeyArrowRight, "ArrowRight", "ArrowRight"},
	{ebiten.KeyNumpad7, "7", "Numpad7"},
	{ebiten.KeyNumpad9, "9", "Numpad9"},
	{ebiten.KeyNumpad1, "1", "Numpad1"},
	{ebiten.KeyNumpad3, "3", "Numpad3"},
	{ebiten.KeyNumpad5, "5", "Numpad5"},
	{ebiten.KeyNumpad8, "8", "Numpad8"},
	{ebiten.KeyNumpad4, "4", "Numpad4"},
	{ebiten.KeyNumpad6, "6", "Numpad6"},
	{ebiten.KeyNumpad2, "2", "Numpad2"},
	{ebiten.KeyNumpad0, "0", "Numpad0"},
	{ebiten.KeyEscape, "Escape", "Escape"},
	{ebiten.KeyEnter, "Enter", "Enter"},
	{ebiten.KeyNumpadEnter, "Enter", "NumpadEnter"},
	{ebiten.KeyBackspace, "Backspace", "Backspace"},
}

// repeatKeys get OS-style auto-repeat when held (movement and wait).
var repeatKeys = map[game.Key]bool{
	game.KeyUp: true, game.KeyDown: true, game.KeyLeft: true, game.KeyRight: true,
	game.KeyUpLeft: true, game.KeyUpRight: true, game.KeyDownLeft: true, game.KeyDownRight: true,
	game.KeyWait: true,
}

// pollKeys returns just-pressed special keys plus typed runes for this tick.
// Numpad keys also emit digit runes via InputChars; those echoes are dropped
// when a numpad key fired this tick so one press never acts twice.
func pollKeys() ([]keyEvent, []rune) {
	var evs []keyEvent
	numpad := false
	for _, sk := range specialKeys {
		if inpututil.IsKeyJustPressed(sk.eb) {
			evs = append(evs, keyEvent{key: sk.key, code: sk.code})
			if len(sk.code) >= 6 && sk.code[:6] == "Numpad" {
				numpad = true
			}
		}
	}
	runes := ebiten.InputChars()
	if numpad {
		kept := runes[:0]
		for _, r := range runes {
			if r < '0' || r > '9' {
				kept = append(kept, r)
			}
		}
		runes = kept
	}
	return evs, runes
}

// normalizeEvent resolves one event to a game key.
func normalizeEvent(ev keyEvent) game.Key {
	return game.NormalizeKey(ev.key, ev.code)
}

// heldRepeat re-fires held movement keys: 20-tick delay, then every 8 ticks.
func heldRepeat() []keyEvent {
	var evs []keyEvent
	for _, sk := range specialKeys {
		d := inpututil.KeyPressDuration(sk.eb)
		if d > 20 && d%8 == 0 {
			if repeatKeys[normalizeEvent(keyEvent{key: sk.key, code: sk.code})] {
				evs = append(evs, keyEvent{key: sk.key, code: sk.code})
			}
		}
	}
	return evs
}

//go:build js && wasm

package game

import (
	"encoding/json"
	"fmt"
	"syscall/js"
)

// SaveToStorage marshals the Game to JSON and stores it in localStorage under pilgirms_save (one slot).
func SaveToStorage(g *Game) error {
	if g == nil {
		return fmt.Errorf("nil game")
	}
	slot := SaveSlot{Version: saveVersion, Game: g}
	data, err := json.Marshal(slot)
	if err != nil {
		return fmt.Errorf("marshal save: %w", err)
	}
	ls := js.Global().Get("localStorage")
	if ls.IsNull() || ls.IsUndefined() {
		return fmt.Errorf("localStorage unavailable")
	}
	ls.Call("setItem", storageKey, string(data))
	return nil
}

// LoadFromStorage reads the save slot from localStorage and deletes it (consume-on-load).
func LoadFromStorage() (*Game, error) {
	ls := js.Global().Get("localStorage")
	if ls.IsNull() || ls.IsUndefined() {
		return nil, fmt.Errorf("localStorage unavailable")
	}
	v := ls.Call("getItem", storageKey)
	if v.IsNull() || v.IsUndefined() {
		return nil, fmt.Errorf("no save")
	}
	s := v.String()
	if s == "" {
		return nil, fmt.Errorf("no save")
	}
	var slot SaveSlot
	if err := json.Unmarshal([]byte(s), &slot); err == nil && slot.Game != nil {
		ls.Call("removeItem", storageKey)
		return slot.Game, nil
	}
	var g Game
	if err := json.Unmarshal([]byte(s), &g); err != nil {
		return nil, fmt.Errorf("parse save: %w", err)
	}
	ls.Call("removeItem", storageKey)
	return &g, nil
}

// HasStorage reports whether a save exists in localStorage.
func HasStorage() bool {
	ls := js.Global().Get("localStorage")
	if ls.IsNull() || ls.IsUndefined() {
		return false
	}
	v := ls.Call("getItem", storageKey)
	return !v.IsNull() && !v.IsUndefined() && v.String() != ""
}

// DeleteStorage removes the save from localStorage.
func DeleteStorage() error {
	ls := js.Global().Get("localStorage")
	if ls.IsNull() || ls.IsUndefined() {
		return fmt.Errorf("localStorage unavailable")
	}
	ls.Call("removeItem", storageKey)
	return nil
}

// Save is shorthand for SaveToStorage.
func Save(g *Game) error {
	return SaveToStorage(g)
}

// Load is shorthand for LoadFromStorage (consumed on load).
func Load() (*Game, error) {
	return LoadFromStorage()
}

// HasSave reports whether the default save slot exists.
func HasSave() bool {
	return HasStorage()
}

// DeleteSave removes the default save slot.
func DeleteSave() error {
	return DeleteStorage()
}

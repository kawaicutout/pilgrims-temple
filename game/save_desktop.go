//go:build !js

package game

import (
	"encoding/json"
	"fmt"
	"os"
)

// SaveToFile marshals the Game (including Party, Levels, Food, XP, etc) to
// JSON and writes it to path. If path is empty, save.json is used. One slot.
func SaveToFile(g *Game, path string) error {
	if g == nil {
		return fmt.Errorf("nil game")
	}
	if path == "" {
		path = saveFileName
	}
	slot := SaveSlot{Version: saveVersion, Game: g}
	data, err := json.Marshal(slot)
	if err != nil {
		return fmt.Errorf("marshal save: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write save: %w", err)
	}
	return nil
}

// LoadFromFile reads and unmarshals the save slot, then deletes the file
// (consume-on-load). If path is empty, save.json is used.
func LoadFromFile(path string) (*Game, error) {
	if path == "" {
		path = saveFileName
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read save: %w", err)
	}
	// Try slot wrapper first.
	var slot SaveSlot
	if err := json.Unmarshal(data, &slot); err == nil && slot.Game != nil {
		_ = os.Remove(path)
		return slot.Game, nil
	}
	// Fallback: raw Game JSON (legacy).
	var g Game
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("parse save: %w", err)
	}
	_ = os.Remove(path)
	return &g, nil
}

// HasSaveFile reports whether a save file exists.
func HasSaveFile(path string) bool {
	if path == "" {
		path = saveFileName
	}
	_, err := os.Stat(path)
	return err == nil
}

// DeleteSaveFile removes the save file if present.
func DeleteSaveFile(path string) error {
	if path == "" {
		path = saveFileName
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Save is shorthand for SaveToFile with the default path.
func Save(g *Game) error {
	return SaveToFile(g, saveFileName)
}

// Load is shorthand for LoadFromFile with the default path (consumed on load).
func Load() (*Game, error) {
	return LoadFromFile(saveFileName)
}

// HasSave reports whether the default save slot exists.
func HasSave() bool {
	return HasSaveFile(saveFileName)
}

// DeleteSave removes the default save slot.
func DeleteSave() error {
	return DeleteSaveFile(saveFileName)
}

//go:build js && wasm

package game

import (
	"encoding/json"
	"syscall/js"
)

const scoreboardStorageKey = "pilgrims_scores"

// Scoreboard persists past runs.
type Scoreboard struct {
	Entries []ScoreEntry `json:"entries"`
}

// AddEntry appends an entry.
func (sb *Scoreboard) AddEntry(e ScoreEntry) {
	sb.Entries = append(sb.Entries, e)
}

// LoadScoreboard reads from localStorage key pilgrims_scores.
func LoadScoreboard() (*Scoreboard, error) {
	ls := js.Global().Get("localStorage")
	if ls.IsNull() || ls.IsUndefined() {
		return &Scoreboard{}, nil
	}
	v := ls.Call("getItem", scoreboardStorageKey)
	if v.IsNull() || v.IsUndefined() {
		return &Scoreboard{}, nil
	}
	s := v.String()
	if s == "" {
		return &Scoreboard{}, nil
	}
	var sb Scoreboard
	if err := json.Unmarshal([]byte(s), &sb); err != nil {
		return nil, err
	}
	if sb.Entries == nil {
		sb.Entries = []ScoreEntry{}
	}
	return &sb, nil
}

// SaveScoreboard writes scoreboard to localStorage.
func SaveScoreboard(sb *Scoreboard) error {
	if sb == nil {
		sb = &Scoreboard{}
	}
	data, err := json.Marshal(sb)
	if err != nil {
		return err
	}
	ls := js.Global().Get("localStorage")
	if ls.IsNull() || ls.IsUndefined() {
		return nil
	}
	ls.Call("setItem", scoreboardStorageKey, string(data))
	return nil
}

func scoreboardFilePath() string { return scoreboardStorageKey }

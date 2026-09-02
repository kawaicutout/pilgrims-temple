//go:build !js

package game

import (
	"encoding/json"
	"os"
)

const scoreboardFileName = "scores.json"

// Scoreboard persists past runs.
type Scoreboard struct {
	Entries []ScoreEntry `json:"entries"`
}

// AddEntry appends an entry.
func (sb *Scoreboard) AddEntry(e ScoreEntry) {
	sb.Entries = append(sb.Entries, e)
}

// LoadScoreboard reads scores.json if present, else empty.
func LoadScoreboard() (*Scoreboard, error) {
	data, err := os.ReadFile(scoreboardFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return &Scoreboard{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return &Scoreboard{}, nil
	}
	var sb Scoreboard
	if err := json.Unmarshal(data, &sb); err != nil {
		return nil, err
	}
	if sb.Entries == nil {
		sb.Entries = []ScoreEntry{}
	}
	return &sb, nil
}

// SaveScoreboard writes scoreboard to scores.json.
func SaveScoreboard(sb *Scoreboard) error {
	if sb == nil {
		sb = &Scoreboard{}
	}
	data, err := json.MarshalIndent(sb, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(scoreboardFileName, data, 0644)
}

// scoreboardFilePath is exposed for tests/helpers.
func scoreboardFilePath() string { return scoreboardFileName }

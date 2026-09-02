//go:build js && wasm

package game

import (
	"encoding/json"
	"sort"
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

// loadScoreboardRaw reads from localStorage without sorting.
func loadScoreboardRaw() (*Scoreboard, error) {
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

// LoadScoreboard reads from localStorage key pilgrims_scores. Returns sorted by Score descending.
func LoadScoreboard() (*Scoreboard, error) {
	sb, err := loadScoreboardRaw()
	if err != nil {
		return nil, err
	}
	sort.Slice(sb.Entries, func(i, j int) bool {
		if sb.Entries[i].Score != sb.Entries[j].Score {
			return sb.Entries[i].Score > sb.Entries[j].Score
		}
		if sb.Entries[i].DepthReached != sb.Entries[j].DepthReached {
			return sb.Entries[i].DepthReached > sb.Entries[j].DepthReached
		}
		return sb.Entries[i].Seed < sb.Entries[j].Seed
	})
	return sb, nil
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

// GetHighScores returns top n entries sorted by Score descending.
func (sb *Scoreboard) GetHighScores(n int) []ScoreEntry {
	if sb == nil || len(sb.Entries) == 0 || n <= 0 {
		return nil
	}
	entries := make([]ScoreEntry, len(sb.Entries))
	copy(entries, sb.Entries)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Score != entries[j].Score {
			return entries[i].Score > entries[j].Score
		}
		if entries[i].DepthReached != entries[j].DepthReached {
			return entries[i].DepthReached > entries[j].DepthReached
		}
		return entries[i].Seed < entries[j].Seed
	})
	if n > len(entries) {
		n = len(entries)
	}
	return entries[:n]
}

// GetRecentScores returns last n entries in chronological order (most recent first) based on raw storage order.
func (sb *Scoreboard) GetRecentScores(n int) []ScoreEntry {
	if sb == nil || len(sb.Entries) == 0 || n <= 0 {
		return nil
	}
	if n > len(sb.Entries) {
		n = len(sb.Entries)
	}
	start := len(sb.Entries) - n
	slice := sb.Entries[start:]
	out := make([]ScoreEntry, n)
	for i := range n {
		out[i] = slice[n-1-i]
	}
	return out
}

// GetHighScores loads the scoreboard and returns top n by score descending.
func GetHighScores(n int) ([]ScoreEntry, error) {
	sb, err := LoadScoreboard()
	if err != nil {
		return nil, err
	}
	return sb.GetHighScores(n), nil
}

// GetRecentScores loads the scoreboard and returns last n chronological (most recent first).
func GetRecentScores(n int) ([]ScoreEntry, error) {
	sb, err := loadScoreboardRaw()
	if err != nil {
		return nil, err
	}
	if sb == nil || len(sb.Entries) == 0 || n <= 0 {
		return nil, nil
	}
	if n > len(sb.Entries) {
		n = len(sb.Entries)
	}
	start := len(sb.Entries) - n
	slice := sb.Entries[start:]
	out := make([]ScoreEntry, n)
	for i := range n {
		out[i] = slice[n-1-i]
	}
	return out, nil
}

// MembersSummary builds "Human Fighter+Elf Cleric" style summary for an entry.
func MembersSummary(e ScoreEntry) string {
	if len(e.Members) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(e.Members))
	for _, m := range e.Members {
		race := FriendlyID(m.Race)
		class := FriendlyID(m.Class)
		var s string
		if race != "" && class != "" {
			s = race + " " + class
		} else if race != "" {
			s = race
		} else if class != "" {
			s = class
		} else if m.Name != "" {
			s = m.Name
		} else {
			s = "Unknown"
		}
		parts = append(parts, s)
	}
	return joinPlus(parts)
}

func joinPlus(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += "+" + p
	}
	return out
}

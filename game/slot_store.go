package game

import (
	"encoding/json"
	"fmt"
	"sort"
)

// slotStore abstracts one-slot persistence: a file on desktop,
// localStorage on web. Absent slots read as nil, nil; deletes tolerate
// absence. All consume-on-load, MOD-gating, and sorting live here, once.
type slotStore interface {
	read(slot string) ([]byte, error)
	write(slot string, data []byte) error
	delete(slot string) error
	exists(slot string) bool
}

func saveGameToSlot(st slotStore, slot string, g *Game) error {
	if g == nil {
		return fmt.Errorf("nil game")
	}
	data, err := json.Marshal(SaveSlot{Version: saveVersion, Game: g})
	if err != nil {
		return fmt.Errorf("marshal save: %w", err)
	}
	return st.write(slot, data)
}

func loadGameFromSlot(st slotStore, slot string) (*Game, error) {
	data, err := st.read(slot)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("no save")
	}
	var s SaveSlot
	if err := json.Unmarshal(data, &s); err == nil && s.Game != nil {
		if err := st.delete(slot); err != nil {
			return nil, fmt.Errorf("consume save: %w", err)
		}
		return s.Game, nil
	}
	// Fallback: raw Game JSON (legacy).
	var g Game
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("parse save: %w", err)
	}
	if err := st.delete(slot); err != nil {
		return nil, fmt.Errorf("consume save: %w", err)
	}
	return &g, nil
}

func hasSaveSlot(st slotStore, slot string) bool {
	return st.exists(slot)
}

func deleteSaveSlot(st slotStore, slot string) error {
	return st.delete(slot)
}

// Scoreboard persists past runs.
type Scoreboard struct {
	Entries []ScoreEntry `json:"entries"`
}

// AddEntry appends an entry.
func (sb *Scoreboard) AddEntry(e ScoreEntry) {
	sb.Entries = append(sb.Entries, e)
}

func sortScoreboard(sb *Scoreboard) {
	sort.Slice(sb.Entries, func(i, j int) bool {
		if sb.Entries[i].Score != sb.Entries[j].Score {
			return sb.Entries[i].Score > sb.Entries[j].Score
		}
		if sb.Entries[i].DepthReached != sb.Entries[j].DepthReached {
			return sb.Entries[i].DepthReached > sb.Entries[j].DepthReached
		}
		return sb.Entries[i].Seed < sb.Entries[j].Seed
	})
}

// loadScoreboardRaw reads raw without sorting (internal).
func loadScoreboardRaw() (*Scoreboard, error) {
	data, err := defaultStore().read(scoreSlot)
	if err != nil {
		return nil, err
	}
	sb := &Scoreboard{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, sb); err != nil {
			return nil, err
		}
	}
	if sb.Entries == nil {
		sb.Entries = []ScoreEntry{}
	}
	return sb, nil
}

// LoadScoreboard reads the slot, else empty. Entries sort by Score descending.
func LoadScoreboard() (*Scoreboard, error) {
	sb, err := loadScoreboardRaw()
	if err != nil {
		return nil, err
	}
	sortScoreboard(sb)
	return sb, nil
}

// SaveScoreboard writes the scoreboard unless data is modified.
func SaveScoreboard(sb *Scoreboard) error {
	if HasModifiedData() {
		return nil
	}
	if sb == nil {
		sb = &Scoreboard{}
	}
	data, err := json.MarshalIndent(sb, "", "  ")
	if err != nil {
		return err
	}
	return defaultStore().write(scoreSlot, data)
}

// Save stores the run in the default slot.
func Save(g *Game) error {
	return saveGameToSlot(defaultStore(), saveSlot, g)
}

// GetHighScores returns top n entries sorted by Score descending. Truncated to n.
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

// GetRecentScores returns last n entries in chronological insertion order (most recent first).
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

// Load reads the default slot (consumed on load).
func Load() (*Game, error) {
	return loadGameFromSlot(defaultStore(), saveSlot)
}

// HasSave reports whether the default save slot exists.
func HasSave() bool {
	return hasSaveSlot(defaultStore(), saveSlot)
}

// DeleteSave removes the default save slot.
func DeleteSave() error {
	return deleteSaveSlot(defaultStore(), saveSlot)
}

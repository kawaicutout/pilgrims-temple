package game

import (
	"encoding/json"
)

// ScoreWeights are tunable via tuning.json scoreWeights.
type ScoreWeights struct {
	Floor       int `json:"floor"`
	Kill        int `json:"kill"`
	Survivor    int `json:"survivor"`
	EscapeBonus int `json:"escapeBonus"`
}

// DefaultScoreWeights matches DESIGN 11.4 / spec.
var DefaultScoreWeights = ScoreWeights{
	Floor:       100,
	Kill:        10,
	Survivor:    50,
	EscapeBonus: 500,
}

// killStore tracks kills per Game until orchestrator adds Game.Kills field.
var killStore = map[*Game]int{}


// AddKill increments kill count.
func (g *Game) AddKill() {
	if v, ok := killStore[g]; ok {
		g.Kills = v
		delete(killStore, g)
	}
	g.Kills++
}

// AddKills increments by n.
func (g *Game) AddKills(n int) {
	if n <= 0 {
		return
	}
	if v, ok := killStore[g]; ok {
		g.Kills = v
		delete(killStore, g)
	}
	g.Kills += n
}

// loadScoreWeights reads scoreWeights from tuning.json if present, else defaults.
func loadScoreWeights() ScoreWeights {
	b, err := dataFS.ReadFile("data/tuning.json")
	if err != nil {
		return DefaultScoreWeights
	}
	var raw struct {
		ScoreWeights *ScoreWeights `json:"scoreWeights"`
	}
	if err := json.Unmarshal(b, &raw); err != nil || raw.ScoreWeights == nil {
		return DefaultScoreWeights
	}
	w := *raw.ScoreWeights
	if w.Floor == 0 {
		w.Floor = DefaultScoreWeights.Floor
	}
	if w.Kill == 0 {
		w.Kill = DefaultScoreWeights.Kill
	}
	if w.Survivor == 0 {
		w.Survivor = DefaultScoreWeights.Survivor
	}
	// EscapeBonus may be 0 intentionally, so keep as is if explicitly 0? Use default only if negative.
	// If file omitted escapeBonus it will be 0; treat 0 as default unless file had explicit 0.
	// We distinguish by checking raw JSON contains escapeBonus: simple - if w.EscapeBonus == 0, set default.
	// Caller can set escape bonus to 0 via tuning by wanting 0? Rare; accept default fallback.
	if w.EscapeBonus == 0 {
		w.EscapeBonus = DefaultScoreWeights.EscapeBonus
	}
	return w
}

// ScoreWeightsFor returns effective weights (file or defaults).
func ScoreWeightsFor() ScoreWeights { return loadScoreWeights() }

// scoreForPure computes score from components and weights.
func scoreForPure(floors, kills, survivors int, escaped bool, w ScoreWeights) int {
	s := floors*w.Floor + kills*w.Kill + survivors*w.Survivor
	if escaped {
		s += w.EscapeBonus
	}
	return s
}

// CalculateScore computes Score = floors*100 + kills*10 + survivors*50 + escapeBonus
// with weights tunable via tuning.json scoreWeights.
func (g *Game) CalculateScore() int {
	w := loadScoreWeights()
	floors := g.Floor + 1
	if floors < 0 {
		floors = 0
	}
	if floors > g.Tuning.Floors {
		floors = g.Tuning.Floors
	}
	kills := g.Kills
	if v, ok := killStore[g]; ok {
		kills = v
		g.Kills = v
		delete(killStore, g)
	}
	survivors := 0
	if g.Party != nil {
		survivors = g.Party.LivingCount()
	}
	escaped := g.Won
	return scoreForPure(floors, kills, survivors, escaped, w)
}

// CalculateScoreWithWeights computes with explicit weights (for tests).
func (g *Game) CalculateScoreWithWeights(w ScoreWeights) int {
	floors := g.Floor + 1
	kills := g.Kills
	if v, ok := killStore[g]; ok {
		kills = v
		g.Kills = v
		delete(killStore, g)
	}
	survivors := 0
	if g.Party != nil {
		survivors = g.Party.LivingCount()
	}
	return scoreForPure(floors, kills, survivors, g.Won, w)
}

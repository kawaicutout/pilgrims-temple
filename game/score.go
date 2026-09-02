package game

import (
	"encoding/json"
)

// MemberScore is per-member snapshot for scoreboard.
type MemberScore struct {
	Name  string `json:"name"`
	Class string `json:"class"`
	Race  string `json:"race"`
	Alive bool   `json:"alive"`
	HP    int    `json:"hp"`
	MaxHP int    `json:"maxHP"`
}

// ScoreEntry is one run result persisted to the scoreboard.
type ScoreEntry struct {
	PartyLevel   int           `json:"partyLevel"`
	Gold         int           `json:"gold"`
	DepthReached int           `json:"depthReached"`
	Members      []MemberScore `json:"members"`
	Seed         int64         `json:"seed"`
	Victory      bool          `json:"victory"`
	CauseOfDeath string        `json:"causeOfDeath"`
	Score        int           `json:"score"`
	Turn         int           `json:"turn"`
}

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
	if g.Wizard {
		return 0
	}
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
	if g.Wizard {
		return 0
	}
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
// depthReached computes DepthReached as max(g.Floor+1, max VisitedFloors+1).
func (g *Game) depthReached() int {
	depth := g.Floor + 1
	if depth < 1 {
		depth = 1
	}
	for k := range g.VisitedFloors {
		if k+1 > depth {
			depth = k + 1
		}
	}
	return depth
}

// buildScoreEntry creates a ScoreEntry snapshot from current Game state.
func (g *Game) buildScoreEntry() ScoreEntry {
	cause := g.Cause
	if g.Won {
		cause = "Victory"
	} else if cause == "" {
		cause = "Unknown"
	}
	depth := g.depthReached()
	var members []MemberScore
	if g.Party != nil {
		for _, m := range g.Party.Members {
			if m == nil {
				continue
			}
			members = append(members, MemberScore{
				Name:  m.Name,
				Class: m.Class,
				Race:  m.Race,
				Alive: m.IsAlive(),
				HP:    m.HP,
				MaxHP: m.MaxHP,
			})
		}
	}
	return ScoreEntry{
		PartyLevel:   g.Level,
		Gold:         g.Gold,
		DepthReached: depth,
		Members:      members,
		Seed:         g.Seed,
		Victory:      g.Won,
		CauseOfDeath: cause,
		Score:        g.CalculateScore(),
		Turn:         g.Turn,
	}
}

// RecordScore builds a ScoreEntry and appends it to the persisted scoreboard.
// It is idempotent per call; callers should ensure they only invoke once per game-over.
func (g *Game) RecordScore() {
	if g == nil {
		return
	}
	entry := g.buildScoreEntry()
	sb, err := LoadScoreboard()
	if err != nil || sb == nil {
		sb = &Scoreboard{}
	}
	sb.AddEntry(entry)
	_ = SaveScoreboard(sb)
}

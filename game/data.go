package game

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed data/*.json
var dataFS embed.FS

// Tuning mirrors data/tuning.json.
type Tuning struct {
	Title  string `json:"title"`
	Floors int    `json:"floors"`
	Map    struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"map"`
	Layout struct {
		MinCols  int `json:"minCols"`
		MinRows  int `json:"minRows"`
		LogLines int `json:"logLines"`
	} `json:"layout"`
	Food struct {
		PerMemberPerTurn  int      `json:"perMemberPerTurn"`
		RationRefill      int      `json:"rationRefill"`
		StartClock        int      `json:"startClock"`
		HungryThreshold   float64  `json:"hungryThreshold"`
		StarvingThreshold float64  `json:"starvingThreshold"`
		States            []string `json:"states"`
	} `json:"food"`
	Rest struct {
		BatchTurns          int `json:"batchTurns"`
		HealPerBatch        int `json:"healPerBatch"`
		NaturalRegenPerTurn int `json:"naturalRegenPerTurn"`
	} `json:"rest"`
	LevelUp struct {
		TalentChance       float64 `json:"talentChance"`
		AffixReplaceChance float64 `json:"affixReplaceChance"`
		XPBase             int     `json:"xpBase"`
		XPFactor           float64 `json:"xpFactor"`
	} `json:"levelUp"`
	Targeting struct {
		ActiveWeight float64 `json:"activeWeight"`
	} `json:"targeting"`
	WorldGen struct {
		RecruitmentRate float64 `json:"recruitmentRate"`
		LightRadius     int     `json:"lightRadius"`
	} `json:"worldGen"`
	ScoreWeights struct {
		Floor       int `json:"floor"`
		Kill        int `json:"kill"`
		Survivor    int `json:"survivor"`
		EscapeBonus int `json:"escapeBonus"`
	} `json:"scoreWeights"`
}

func LoadTuning() (Tuning, error) {
	b, err := dataFS.ReadFile("data/tuning.json")
	if err != nil {
		return Tuning{}, fmt.Errorf("read tuning.json: %w", err)
	}
	var t Tuning
	if err := json.Unmarshal(b, &t); err != nil {
		return Tuning{}, fmt.Errorf("parse tuning.json: %w", err)
	}
	return t, nil
}

// globalTuning caches the tuning from the active Game for package accessors
// that lack a Game reference (e.g., combat targeting). Set via NewGame.
var globalTuning *Tuning

// SetGlobalTuning caches t for GetTuning consumers.
func SetGlobalTuning(t Tuning) {
	c := t
	globalTuning = &c
}

// GetTuning returns the cached global tuning if set, otherwise loads from
// data/tuning.json. Defaults ActiveWeight to 0.5 if missing so callers can
// safely use Tuning.Targeting.ActiveWeight.
func GetTuning() Tuning {
	if globalTuning != nil {
		return *globalTuning
	}
	t, err := LoadTuning()
	if err != nil {
		t = Tuning{}
		t.Food.RationRefill = 50
		t.LevelUp.XPBase = 100
		t.LevelUp.XPFactor = 1.5
		t.Targeting.ActiveWeight = 0.5
		return t
	}
	if t.Targeting.ActiveWeight == 0 {
		t.Targeting.ActiveWeight = 0.5
	}
	if t.Food.RationRefill == 0 {
		t.Food.RationRefill = 50
	}
	if t.LevelUp.XPBase == 0 {
		t.LevelUp.XPBase = 100
	}
	if t.LevelUp.XPFactor == 0 {
		t.LevelUp.XPFactor = 1.5
	}
	return t
}

// ActiveWeight returns the current targeting active weight from GetTuning(),
// clamped to [0,1] with 0.5 fallback.
func ActiveWeight() float64 {
	w := GetTuning().Targeting.ActiveWeight
	if w < 0 {
		w = 0
	}
	if w > 1 {
		w = 1
	}
	if w == 0 {
		return 0.5
	}
	return w
}

// RawJSON returns the raw bytes for a data file (for generic consumers / web upload overlay).
func RawJSON(name string) ([]byte, error) {
	return dataFS.ReadFile("data/" + name)
}

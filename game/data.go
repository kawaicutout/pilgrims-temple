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

// RawJSON returns the raw bytes for a data file (for generic consumers / web upload overlay).
func RawJSON(name string) ([]byte, error) {
	return dataFS.ReadFile("data/" + name)
}

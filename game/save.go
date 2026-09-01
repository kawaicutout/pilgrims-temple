package game

import (
	"encoding/json"
	"math/rand/v2"
)

const saveFileName = "save.json"
const storageKey = "pilgrims_save"
const saveVersion = 1

// SaveSlot is the persisted one-slot save. It wraps the full Game snapshot
// with a version for future migration.
type SaveSlot struct {
	Version int   `json:"version"`
	Game    *Game `json:"game"`
}

type gameJSON struct {
	Seed                    int64          `json:"seed"`
	Tuning                  Tuning         `json:"tuning"`
	Levels                  []*Level       `json:"levels"`
	Floor                   int            `json:"floor"`
	Party                   *Party         `json:"party"`
	Log                     []string       `json:"log"`
	Turn                    int            `json:"turn"`
	Food                    int            `json:"food"`
	FoodFloat               float64        `json:"foodFloat"`
	Level                   int            `json:"level"`
	XP                      int            `json:"xp"`
	XPToNext                int            `json:"xpToNext"`
	LevelUpPending          *LevelUpState  `json:"levelUpPending"`
	Gold                    int            `json:"gold"`
	Kills                   int            `json:"kills"`
	Escaped                 bool           `json:"escaped"`
	Over                    bool           `json:"over"`
	Won                     bool           `json:"won"`
	Quit                    bool           `json:"quit"`
	Relic                   Pos            `json:"relic"`
	Wizard                  bool           `json:"wizard"`
	VisitedFloors           map[int]bool   `json:"visitedFloors"`
	TransitionFiredForLevel map[int]bool   `json:"transitionFiredForLevel"`
	RelicCollected          bool           `json:"relicCollected"`
}
func (g *Game) MarshalJSON() ([]byte, error) {
	if g == nil {
		return []byte("null"), nil
	}
	aux := gameJSON{
		Seed:                    g.Seed,
		Tuning:                  g.Tuning,
		Levels:                  g.Levels,
		Floor:                   g.Floor,
		Party:                   g.Party,
		Log:                     g.Log,
		Turn:                    g.Turn,
		Food:                    g.Food,
		FoodFloat:               g.FoodFloat,
		Level:                   g.Level,
		XP:                      g.XP,
		XPToNext:                g.XPToNext,
		LevelUpPending:          g.LevelUpPending,
		Gold:                    g.Gold,
		Kills:                   g.Kills,
		Escaped:                 g.Escaped,
		Over:                    g.Over,
		Won:                     g.Won,
		Quit:                    g.Quit,
		Relic:                   g.Relic,
		Wizard:                  g.Wizard,
		VisitedFloors:           g.VisitedFloors,
		TransitionFiredForLevel: g.TransitionFiredForLevel,
		RelicCollected:          g.RelicCollected,
	}
	return json.Marshal(aux)
}

// UnmarshalJSON reconstructs RNG from Seed.
func (g *Game) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var aux gameJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	g.Seed = aux.Seed
	g.Tuning = aux.Tuning
	g.Levels = aux.Levels
	g.Floor = aux.Floor
	g.Party = aux.Party
	g.Log = aux.Log
	g.Turn = aux.Turn
	g.Food = aux.Food
	g.FoodFloat = aux.FoodFloat
	g.Level = aux.Level
	g.XP = aux.XP
	g.XPToNext = aux.XPToNext
	g.LevelUpPending = aux.LevelUpPending
	g.Gold = aux.Gold
	g.Kills = aux.Kills
	g.Escaped = aux.Escaped
	g.Over = aux.Over
	g.Won = aux.Won
	g.Quit = aux.Quit
	g.Relic = aux.Relic
	g.Wizard = aux.Wizard
	g.VisitedFloors = aux.VisitedFloors
	g.TransitionFiredForLevel = aux.TransitionFiredForLevel
	g.RelicCollected = aux.RelicCollected
	// Recreate RNG deterministically from seed (same PCG as NewGame).
	g.RNG = rand.New(rand.NewPCG(uint64(g.Seed), 0x9e3779b97f4a7c15))
	return nil
}

package game

import (
	"encoding/json"
	"math/rand/v2"
)

// FeatureType classifies a level feature.
type FeatureType string

const (
	FeatureMerchant FeatureType = "merchant"
	FeatureFountain FeatureType = "fountain"
	FeatureShrine   FeatureType = "shrine"
)

// Feature is a scarce, data-driven level feature placed on a floor tile.
type Feature struct {
	Pos  Pos         `json:"pos"`
	Type FeatureType `json:"type"`
}

// IsMerchant reports whether f is a merchant feature.
func (f Feature) IsMerchant() bool { return f.Type == FeatureMerchant }

// IsFountain reports whether f is a fountain feature.
func (f Feature) IsFountain() bool { return f.Type == FeatureFountain }

// IsShrine reports whether f is a shrine feature.
func (f Feature) IsShrine() bool { return f.Type == FeatureShrine }

// Merchant helpers — convenience constructors / checks.

// NewMerchantFeature creates a merchant feature at pos.
func NewMerchantFeature(pos Pos) Feature { return Feature{Pos: pos, Type: FeatureMerchant} }

// NewFountainFeature creates a fountain feature at pos.
func NewFountainFeature(pos Pos) Feature { return Feature{Pos: pos, Type: FeatureFountain} }

// NewShrineFeature creates a shrine feature at pos.
func NewShrineFeature(pos Pos) Feature { return Feature{Pos: pos, Type: FeatureShrine} }

// IsMerchantFeature is a helper for callers holding Feature values.
func IsMerchantFeature(f Feature) bool { return f.IsMerchant() }

// IsFountainFeature helper.
func IsFountainFeature(f Feature) bool { return f.IsFountain() }

// IsShrineFeature helper.
func IsShrineFeature(f Feature) bool { return f.IsShrine() }

// ---------------------------------------------------------------------------
// Data-driven tunable rates via game/data/features.json
// ---------------------------------------------------------------------------

type FeaturesConfig struct {
	Merchants struct {
		Rate   float64 `json:"rate"`
		Scarce bool    `json:"scarce"`
	} `json:"merchants"`
	Fountains struct {
		Rate float64 `json:"rate"`
	} `json:"fountains"`
	Shrines struct {
		Rate float64 `json:"rate"`
	} `json:"shrines"`
}

// featuresConfig is an alias for backward compatibility with unexported name.
type featuresConfig = FeaturesConfig

var featuresCache *FeaturesConfig

func loadFeaturesConfig() FeaturesConfig {
	if featuresCache != nil {
		return *featuresCache
	}
	cfg := FeaturesConfig{}
	// Defaults matching spec: merchants scarce 0.15, fountains 0.2, shrines 0.25
	cfg.Merchants.Rate = 0.15
	cfg.Merchants.Scarce = true
	cfg.Fountains.Rate = 0.2
	cfg.Shrines.Rate = 0.25

	b, err := dataFS.ReadFile("data/features.json")
	if err != nil {
		featuresCache = &cfg
		return cfg
	}
	// Try object form first.
	var raw featuresConfig
	if err := json.Unmarshal(b, &raw); err == nil {
		// Merge non-zero rates; keep defaults for missing/zero where file omits.
		// We detect presence by re-unmarshalling into map.
		var m map[string]json.RawMessage
		if err2 := json.Unmarshal(b, &m); err2 == nil {
			if _, ok := m["merchants"]; ok {
				cfg.Merchants = raw.Merchants
				// If scarce not explicitly set but rate present, keep scarce true for merchants per spec.
				// The JSON object scarce false is intentional if author sets it; so only default when missing.
				// Detect missing scarce key inside merchants object.
				var inner map[string]json.RawMessage
				if json.Unmarshal(m["merchants"], &inner) == nil {
					if _, hasScarce := inner["scarce"]; !hasScarce {
						cfg.Merchants.Scarce = true
					}
				}
			}
			if _, ok := m["fountains"]; ok {
				cfg.Fountains = raw.Fountains
			}
			if _, ok := m["shrines"]; ok {
				cfg.Shrines = raw.Shrines
			}
			// Also support flat rates: "merchantRate", "fountainRate", "shrineRate"
			var flat map[string]float64
			if json.Unmarshal(b, &flat) == nil {
				if v, ok := flat["merchantRate"]; ok {
					cfg.Merchants.Rate = v
				}
				if v, ok := flat["fountainRate"]; ok {
					cfg.Fountains.Rate = v
				}
				if v, ok := flat["shrineRate"]; ok {
					cfg.Shrines.Rate = v
				}
			}
		} else {
			// Fallback: raw had valid shape
			if raw.Merchants.Rate != 0 {
				cfg.Merchants.Rate = raw.Merchants.Rate
			}
			if raw.Fountains.Rate != 0 {
				cfg.Fountains.Rate = raw.Fountains.Rate
			}
			if raw.Shrines.Rate != 0 {
				cfg.Shrines.Rate = raw.Shrines.Rate
			}
		}
		featuresCache = &cfg
		return cfg
	}
	// Fallback: try flat map of rates
	var flat map[string]float64
	if err := json.Unmarshal(b, &flat); err == nil {
		if v, ok := flat["merchantRate"]; ok {
			cfg.Merchants.Rate = v
		}
		if v, ok := flat["fountains"]; ok {
			cfg.Fountains.Rate = v
		}
		if v, ok := flat["shrines"]; ok {
			cfg.Shrines.Rate = v
		}
		if v, ok := flat["merchants"]; ok {
			cfg.Merchants.Rate = v
		}
	}
	featuresCache = &cfg
	return cfg
}

// GetFeaturesConfig returns a copy of the tunable feature rates.
func GetFeaturesConfig() FeaturesConfig { return loadFeaturesConfig() }

// ---------------------------------------------------------------------------
// Fountain / Shrine data (placeholder outcomes)
// ---------------------------------------------------------------------------

// FountainOutcome is one possible fountain result.
type FountainOutcome struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	DeltaHP int    `json:"deltaHP"`
	Delta   int    `json:"delta"`
	Effect  string `json:"effect,omitempty"`
	Desc    string `json:"desc,omitempty"`
}

type fountainsFile struct {
	Outcomes []FountainOutcome `json:"outcomes"`
}

var fountainsCache []FountainOutcome

func loadFountains() []FountainOutcome {
	if fountainsCache != nil {
		return fountainsCache
	}
	b, err := dataFS.ReadFile("data/fountains.json")
	if err != nil {
		fountainsCache = []FountainOutcome{
			{ID: "heal", Name: "Healing Waters", DeltaHP: 10, Delta: 10, Desc: "Refreshing waters restore 10 HP"},
			{ID: "poison", Name: "Tainted Waters", DeltaHP: -5, Delta: -5, Desc: "Foul waters sicken you for 5 damage"},
			{ID: "blessing", Name: "Blessed Spring", DeltaHP: 5, Delta: 5, Effect: "bless", Desc: "Blessed waters restore 5 HP and grant a brief boon"},
			{ID: "curse", Name: "Cursed Pool", DeltaHP: -2, Delta: -2, Effect: "curse", Desc: "Cursed waters drain 2 HP and weaken you"},
		}
		return fountainsCache
	}
	var ff fountainsFile
	if err := json.Unmarshal(b, &ff); err != nil || len(ff.Outcomes) == 0 {
		fountainsCache = []FountainOutcome{
			{ID: "heal", Name: "Healing Waters", DeltaHP: 10, Delta: 10, Desc: "Refreshing waters restore 10 HP"},
			{ID: "poison", Name: "Tainted Waters", DeltaHP: -5, Delta: -5, Desc: "Foul waters sicken you for 5 damage"},
			{ID: "blessing", Name: "Blessed Spring", DeltaHP: 5, Delta: 5, Effect: "bless", Desc: "Blessed waters restore 5 HP"},
			{ID: "curse", Name: "Cursed Pool", DeltaHP: -2, Delta: -2, Effect: "curse", Desc: "Cursed waters drain 2 HP"},
		}
		return fountainsCache
	}
	fountainsCache = ff.Outcomes
	return fountainsCache
}

// GetFountainOutcomes returns a copy of fountain outcome data.
func GetFountainOutcomes() []FountainOutcome {
	src := loadFountains()
	out := make([]FountainOutcome, len(src))
	copy(out, src)
	return out
}

// ShrineUse is one shrine function (recruit vs resurrect).
type ShrineUse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	GoldCost int    `json:"goldCost"`
	FoodCost int    `json:"foodCost"`
	Desc     string `json:"desc,omitempty"`
}

type shrinesFile struct {
	Uses []ShrineUse `json:"uses"`
}

var shrinesCache []ShrineUse

func loadShrines() []ShrineUse {
	if shrinesCache != nil {
		return shrinesCache
	}
	b, err := dataFS.ReadFile("data/shrines.json")
	if err != nil {
		shrinesCache = []ShrineUse{
			{ID: "recruit", Name: "Recruitment", GoldCost: 0, FoodCost: 0, Desc: "Recruit a new member to your party"},
			{ID: "resurrect", Name: "Resurrection", GoldCost: 75, FoodCost: 50, Desc: "Resurrect a fallen member at a cost of gold or food"},
		}
		return shrinesCache
	}
	var sf shrinesFile
	if err := json.Unmarshal(b, &sf); err != nil || len(sf.Uses) == 0 {
		shrinesCache = []ShrineUse{
			{ID: "recruit", Name: "Recruitment", GoldCost: 0, FoodCost: 0, Desc: "Recruit a new member to your party"},
			{ID: "resurrect", Name: "Resurrection", GoldCost: 75, FoodCost: 50, Desc: "Resurrect a fallen member at a cost of gold or food"},
		}
		return shrinesCache
	}
	shrinesCache = sf.Uses
	return shrinesCache
}

// GetShrineUses returns a copy of shrine use data.
func GetShrineUses() []ShrineUse {
	src := loadShrines()
	out := make([]ShrineUse, len(src))
	copy(out, src)
	return out
}

// ---------------------------------------------------------------------------
// Placement: MaybeSpawnFeatures
// ---------------------------------------------------------------------------

// MaybeSpawnFeatures places merchants/fountains/shrines on random floor tiles
// not on stairs or enemy positions, respecting tunable scarce rates.
// Returns the spawned features (0-3 per floor, typically 0-1 of each type).
// Deterministic from rng; floor is used only to allow future scaling.
func MaybeSpawnFeatures(lvl *Level, floor int, rng *rand.Rand) []Feature {
	if lvl == nil || rng == nil {
		return nil
	}
	_ = floor
	cfg := loadFeaturesConfig()

	// Collect walkable candidates avoiding stairs and enemies.
	candidates := make([]Pos, 0, lvl.W*lvl.H/2)
	enemySet := make(map[Pos]bool, len(lvl.Enemies))
	for _, e := range lvl.Enemies {
		if e != nil {
			enemySet[e.Pos] = true
		}
	}
	for y := range lvl.H {
		for x := range lvl.W {
			p := Pos{x, y}
			if p == lvl.StairsUp || p == lvl.StairsDown {
				continue
			}
			if enemySet[p] {
				continue
			}
			if !lvl.Walkable(p) {
				continue
			}
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	// Shuffle candidates for uniform random selection.
	for i := len(candidates) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}

	var out []Feature
	used := make(map[Pos]bool)
	nextPos := func() (Pos, bool) {
		for len(candidates) > 0 {
			p := candidates[0]
			candidates = candidates[1:]
			if used[p] {
				continue
			}
			used[p] = true
			return p, true
		}
		return Pos{}, false
	}

	if rng.Float64() < cfg.Merchants.Rate {
		if p, ok := nextPos(); ok {
			out = append(out, Feature{Pos: p, Type: FeatureMerchant})
		}
	}
	if rng.Float64() < cfg.Fountains.Rate {
		if p, ok := nextPos(); ok {
			out = append(out, Feature{Pos: p, Type: FeatureFountain})
		}
	}
	if rng.Float64() < cfg.Shrines.Rate {
		if p, ok := nextPos(); ok {
			out = append(out, Feature{Pos: p, Type: FeatureShrine})
		}
	}
	return out
}

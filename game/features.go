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
	FeatureVault    FeatureType = "vault"
	FeatureForge    FeatureType = "forge"
	FeatureDen      FeatureType = "den"
	FeaturePitfall  FeatureType = "pitfall"
)

// Feature is a scarce, data-driven level feature placed on a floor tile.
// Vault carries Locked/Treasure/Trapped; Forge carries CostType/Cost;
// Den carries MonsterCount; Pitfall carries Hidden/Damage.
type Feature struct {
	Pos          Pos         `json:"pos"`
	Type         FeatureType `json:"type"`
	Locked       bool        `json:"locked,omitempty"`
	Treasure     int         `json:"treasure,omitempty"`
	Trapped      bool        `json:"trapped,omitempty"`
	CostType     string      `json:"costType,omitempty"`
	Cost         int         `json:"cost,omitempty"`
	MonsterCount int         `json:"monsterCount,omitempty"`
	Hidden       bool        `json:"hidden,omitempty"`
	Damage       int         `json:"damage,omitempty"`
}

// Vault is the structured view of a vault Feature (locked/treasure/traps).
type Vault struct {
	Pos      Pos  `json:"pos"`
	Locked   bool `json:"locked"`
	Treasure int  `json:"treasure"`
	Trapped  bool `json:"trapped"`
}

// Forge is the structured view of a forge Feature (cost type gold/food).
type Forge struct {
	Pos      Pos    `json:"pos"`
	CostType string `json:"costType"`
	Cost     int    `json:"cost"`
}

// Den is the structured view of a den Feature (monster count 3-5).
type Den struct {
	Pos          Pos `json:"pos"`
	MonsterCount int `json:"monsterCount"`
}

// Pitfall is the structured view of a pitfall Feature (obvious/hidden + damage 2-4).
type Pitfall struct {
	Pos    Pos  `json:"pos"`
	Hidden bool `json:"hidden"`
	Damage int  `json:"damage"`
}

// IsMerchant reports whether f is a merchant feature.
func (f Feature) IsMerchant() bool { return f.Type == FeatureMerchant }

// IsFountain reports whether f is a fountain feature.
func (f Feature) IsFountain() bool { return f.Type == FeatureFountain }

// IsShrine reports whether f is a shrine feature.
func (f Feature) IsShrine() bool { return f.Type == FeatureShrine }

func (f Feature) IsVault() bool   { return f.Type == FeatureVault }
func (f Feature) IsForge() bool   { return f.Type == FeatureForge }
func (f Feature) IsDen() bool     { return f.Type == FeatureDen }
func (f Feature) IsPitfall() bool { return f.Type == FeaturePitfall }

func (f Feature) AsVault() Vault {
	return Vault{Pos: f.Pos, Locked: f.Locked, Treasure: f.Treasure, Trapped: f.Trapped}
}
func (f Feature) AsForge() Forge     { return Forge{Pos: f.Pos, CostType: f.CostType, Cost: f.Cost} }
func (f Feature) AsDen() Den         { return Den{Pos: f.Pos, MonsterCount: f.MonsterCount} }
func (f Feature) AsPitfall() Pitfall { return Pitfall{Pos: f.Pos, Hidden: f.Hidden, Damage: f.Damage} }

// Merchant helpers — convenience constructors / checks.

// NewMerchantFeature creates a merchant feature at pos.
func NewMerchantFeature(pos Pos) Feature { return Feature{Pos: pos, Type: FeatureMerchant} }

// NewFountainFeature creates a fountain feature at pos.
func NewFountainFeature(pos Pos) Feature { return Feature{Pos: pos, Type: FeatureFountain} }

// NewShrineFeature creates a shrine feature at pos.
func NewShrineFeature(pos Pos) Feature { return Feature{Pos: pos, Type: FeatureShrine} }

func NewVaultFeature(pos Pos, locked bool, treasure int, trapped bool) Feature {
	return Feature{Pos: pos, Type: FeatureVault, Locked: locked, Treasure: treasure, Trapped: trapped}
}
func NewForgeFeature(pos Pos, costType string, cost int) Feature {
	return Feature{Pos: pos, Type: FeatureForge, CostType: costType, Cost: cost}
}
func NewDenFeature(pos Pos, count int) Feature {
	if count < 3 {
		count = 3
	}
	if count > 5 {
		count = 5
	}
	return Feature{Pos: pos, Type: FeatureDen, MonsterCount: count}
}
func NewPitfallFeature(pos Pos, hidden bool, damage int) Feature {
	if damage < 2 {
		damage = 2
	}
	if damage > 4 {
		damage = 4
	}
	return Feature{Pos: pos, Type: FeaturePitfall, Hidden: hidden, Damage: damage}
}

// IsMerchantFeature is a helper for callers holding Feature values.
func IsMerchantFeature(f Feature) bool { return f.IsMerchant() }

// IsFountainFeature helper.
func IsFountainFeature(f Feature) bool { return f.IsFountain() }

// IsShrineFeature helper.
func IsShrineFeature(f Feature) bool  { return f.IsShrine() }
func IsVaultFeature(f Feature) bool   { return f.IsVault() }
func IsForgeFeature(f Feature) bool   { return f.IsForge() }
func IsDenFeature(f Feature) bool     { return f.IsDen() }
func IsPitfallFeature(f Feature) bool { return f.IsPitfall() }

// Glyph returns the map glyph for the feature type.
func (f Feature) Glyph() rune {
	switch f.Type {
	case FeatureMerchant:
		return 'M'
	case FeatureFountain:
		return '&'
	case FeatureShrine:
		return '+'
	case FeatureVault:
		return '$'
	case FeatureForge:
		return 'F'
	case FeatureDen:
		return 'D'
	case FeaturePitfall:
		return '^'
	default:
		return '?'
	}
}

// Color returns the FG token for the feature.
func (f Feature) Color() string {
	switch f.Type {
	case FeatureMerchant:
		return "gold"
	case FeatureFountain:
		return "slate"
	case FeatureShrine:
		return "gold-bright"
	case FeatureVault:
		return "gold-bright"
	case FeatureForge:
		return "gold"
	case FeatureDen:
		return "red-bright"
	case FeaturePitfall:
		return "gray-1"
	default:
		return "fg"
	}
}

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
	Vaults struct {
		Rate          float64 `json:"rate"`
		Locked        bool    `json:"locked"`
		TreasureMin   int     `json:"treasureMin"`
		TreasureMax   int     `json:"treasureMax"`
		TrappedChance float64 `json:"trappedChance"`
	} `json:"vaults"`
	Forges struct {
		Rate     float64 `json:"rate"`
		CostType string  `json:"costType"`
		GoldCost int     `json:"goldCost"`
		FoodCost int     `json:"foodCost"`
	} `json:"forges"`
	Dens struct {
		Rate       float64 `json:"rate"`
		MonsterMin int     `json:"monsterMin"`
		MonsterMax int     `json:"monsterMax"`
	} `json:"dens"`
	Pitfalls struct {
		Rate         float64 `json:"rate"`
		HiddenChance float64 `json:"hiddenChance"`
		DamageMin    int     `json:"damageMin"`
		DamageMax    int     `json:"damageMax"`
	} `json:"pitfalls"`
	PerBiomeVariants map[string]json.RawMessage `json:"perBiomeVariants,omitempty"`
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
	// New features: vault 0.12 locked, forge 0.1 gold, den 0.12 3-5, pitfall 0.1 hidden 0.5 dmg 2-4
	cfg.Merchants.Rate = 0.15
	cfg.Merchants.Scarce = true
	cfg.Fountains.Rate = 0.2
	cfg.Shrines.Rate = 0.25
	cfg.Vaults.Rate = 0.12
	cfg.Vaults.Locked = true
	cfg.Vaults.TreasureMin = 25
	cfg.Vaults.TreasureMax = 80
	cfg.Vaults.TrappedChance = 0.2
	cfg.Forges.Rate = 0.1
	cfg.Forges.CostType = "gold"
	cfg.Forges.GoldCost = 25
	cfg.Forges.FoodCost = 50
	cfg.Dens.Rate = 0.12
	cfg.Dens.MonsterMin = 3
	cfg.Dens.MonsterMax = 5
	cfg.Pitfalls.Rate = 0.1
	cfg.Pitfalls.HiddenChance = 0.5
	cfg.Pitfalls.DamageMin = 2
	cfg.Pitfalls.DamageMax = 4

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
			if _, ok := m["vaults"]; ok {
				cfg.Vaults = raw.Vaults
				if cfg.Vaults.TreasureMin == 0 {
					cfg.Vaults.TreasureMin = 25
				}
				if cfg.Vaults.TreasureMax == 0 {
					cfg.Vaults.TreasureMax = 80
				}
			}
			if _, ok := m["forges"]; ok {
				cfg.Forges = raw.Forges
				if cfg.Forges.CostType == "" {
					cfg.Forges.CostType = "gold"
				}
				if cfg.Forges.GoldCost == 0 {
					cfg.Forges.GoldCost = 25
				}
				if cfg.Forges.FoodCost == 0 {
					cfg.Forges.FoodCost = 50
				}
			}
			if _, ok := m["dens"]; ok {
				cfg.Dens = raw.Dens
				if cfg.Dens.MonsterMin == 0 {
					cfg.Dens.MonsterMin = 3
				}
				if cfg.Dens.MonsterMax == 0 {
					cfg.Dens.MonsterMax = 5
				}
			}
			if _, ok := m["pitfalls"]; ok {
				cfg.Pitfalls = raw.Pitfalls
				if cfg.Pitfalls.DamageMin == 0 {
					cfg.Pitfalls.DamageMin = 2
				}
				if cfg.Pitfalls.DamageMax == 0 {
					cfg.Pitfalls.DamageMax = 4
				}
			}
			if _, ok := m["perBiomeVariants"]; ok {
				cfg.PerBiomeVariants = raw.PerBiomeVariants
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
			if raw.Vaults.Rate != 0 {
				cfg.Vaults.Rate = raw.Vaults.Rate
			}
			if raw.Forges.Rate != 0 {
				cfg.Forges.Rate = raw.Forges.Rate
			}
			if raw.Dens.Rate != 0 {
				cfg.Dens.Rate = raw.Dens.Rate
			}
			if raw.Pitfalls.Rate != 0 {
				cfg.Pitfalls.Rate = raw.Pitfalls.Rate
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
	// Vault — locked room, treasure, possible trap.
	if rng.Float64() < cfg.Vaults.Rate {
		if p, ok := nextPos(); ok {
			treasureMin := cfg.Vaults.TreasureMin
			if treasureMin == 0 {
				treasureMin = 25
			}
			treasureMax := cfg.Vaults.TreasureMax
			if treasureMax == 0 {
				treasureMax = 80
			}
			if treasureMax < treasureMin {
				treasureMax = treasureMin
			}
			treasure := treasureMin + rng.IntN(treasureMax-treasureMin+1)
			trapped := rng.Float64() < cfg.Vaults.TrappedChance
			locked := cfg.Vaults.Locked
			out = append(out, Feature{Pos: p, Type: FeatureVault, Locked: locked, Treasure: treasure, Trapped: trapped})
		}
	}
	// Forge — improve ATK/DEF for gold/food.
	if rng.Float64() < cfg.Forges.Rate {
		if p, ok := nextPos(); ok {
			ct := cfg.Forges.CostType
			if ct == "" {
				ct = "gold"
			}
			cost := cfg.Forges.GoldCost
			if ct == "food" {
				cost = cfg.Forges.FoodCost
			}
			if cost == 0 {
				if ct == "food" {
					cost = 50
				} else {
					cost = 25
				}
			}
			out = append(out, Feature{Pos: p, Type: FeatureForge, CostType: ct, Cost: cost})
		}
	}
	// Den — 3-5 monsters in one room (feature marks center, enemies spawned nearby).
	if rng.Float64() < cfg.Dens.Rate {
		if p, ok := nextPos(); ok {
			mn := cfg.Dens.MonsterMin
			if mn == 0 {
				mn = 3
			}
			mx := cfg.Dens.MonsterMax
			if mx == 0 {
				mx = 5
			}
			if mx < mn {
				mx = mn
			}
			count := mn + rng.IntN(mx-mn+1)
			out = append(out, Feature{Pos: p, Type: FeatureDen, MonsterCount: count})
			// Spawn den pack: 3-5 enemies clustered around den center.
			// Use existing enemy pool helper if available; otherwise fallback to simple.
			for i := range count {
				// Find nearby free tile near p (within 2 steps).
				var ep Pos
				found := false
				for tries := range 20 {
					dx := rng.IntN(5) - 2
					dy := rng.IntN(5) - 2
					cand := Pos{p.X + dx, p.Y + dy}
					if cand == lvl.StairsUp || cand == lvl.StairsDown || cand == p {
						continue
					}
					if !lvl.InBounds(cand) || !lvl.Walkable(cand) {
						continue
					}
					// Avoid feature positions and existing enemies.
					if used[cand] {
						continue
					}
					occupied := false
					for _, e := range lvl.Enemies {
						if e != nil && e.Pos == cand {
							occupied = true
							break
						}
					}
					if occupied {
						continue
					}
					// also avoid already placed den pack positions in this loop
					coll := false
					for j := range i {
						_ = j
						// linear scan of candidates already used by pack handled via used map
					}
					if tries < 20 {
						ep = cand
						found = true
						break
					}
					_ = coll
				}
				if !found {
					ep = p // fallback: stack on den (will be offset by combat)
					// Try to jitter
					for _, d := range AllDirs {
						cand := p.Add(d)
						if lvl.Walkable(cand) && cand != lvl.StairsUp && cand != lvl.StairsDown {
							ep = cand
							break
						}
					}
				}
				used[ep] = true
				// Create single-member party for den (pack = multiple single parties).
				entry := pickEnemyForFloor(rng, floor)
				hp := 6 + floor*2 + rng.IntN(4)
				if entry.Regen {
					hp += 4
				}
				atkMin := 2 + floor
				atkMax := atkMin + 2 + rng.IntN(2)
				mem := &Member{
					Name: entry.Name, Class: entry.ID,
					HP: hp, MaxHP: hp,
					ATK:   [2]int{atkMin, atkMax},
					Alive: true, DamageType: entry.DamageType,
					Effect: entry.Effect, EffectChance: entry.EffectChance,
					Regen: entry.Regen, XP: entry.XP, Color: entry.Color,
				}
				epParty := &EnemyParty{Pos: ep, Members: []*Member{mem}, Active: 0}
				lvl.Enemies = append(lvl.Enemies, epParty)
			}
		}
	}
	// Pitfall — tile feature, hidden vs obvious, damage 2-4.
	if rng.Float64() < cfg.Pitfalls.Rate {
		if p, ok := nextPos(); ok {
			hidden := rng.Float64() < cfg.Pitfalls.HiddenChance
			dmin := cfg.Pitfalls.DamageMin
			if dmin == 0 {
				dmin = 2
			}
			dmax := cfg.Pitfalls.DamageMax
			if dmax == 0 {
				dmax = 4
			}
			if dmax < dmin {
				dmax = dmin
			}
			dmg := dmin + rng.IntN(dmax-dmin+1)
			out = append(out, Feature{Pos: p, Type: FeaturePitfall, Hidden: hidden, Damage: dmg})
		}
	}
	return out
}

package game

import (
	"encoding/json"
	"sync"
)

// FloorTheme is one themed floor visual + weighting entry.
// Glyph variants keep walls as "#" family; tint is desaturated per theme.
type FloorTheme struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	WallGlyph          string             `json:"wallGlyph"`
	FloorGlyph         string             `json:"floorGlyph"`
	WallGlyphVariants  []string           `json:"wallGlyphVariants"`
	FloorGlyphVariants []string           `json:"floorGlyphVariants"`
	Color              string             `json:"color"`
	Tint               string             `json:"tint"`
	EnemyWeights       map[string]float64 `json:"enemyWeights"`
	ItemWeights        map[string]float64 `json:"itemWeights"`
}

// TintColor returns the theme tint (falls back to Color).
func (ft FloorTheme) TintColor() string {
	if ft.Tint != "" {
		return ft.Tint
	}
	return ft.Color
}

// WallVariants returns wall glyph variants (always non-empty, fallback to WallGlyph).
func (ft FloorTheme) WallVariants() []string {
	if len(ft.WallGlyphVariants) > 0 {
		return ft.WallGlyphVariants
	}
	if ft.WallGlyph != "" {
		return []string{ft.WallGlyph}
	}
	return []string{"#"}
}

// FloorVariants returns floor glyph variants.
func (ft FloorTheme) FloorVariants() []string {
	if len(ft.FloorGlyphVariants) > 0 {
		return ft.FloorGlyphVariants
	}
	if ft.FloorGlyph != "" {
		return []string{ft.FloorGlyph}
	}
	return []string{"."}
}

// EnemyWeight returns weight for an enemy id (1.0 if missing).
func (ft FloorTheme) EnemyWeight(id string) float64 {
	if w, ok := ft.EnemyWeights[id]; ok {
		return w
	}
	return 1.0
}

// ItemWeight returns weight for an item kind (1.0 if missing).
func (ft FloorTheme) ItemWeight(kind string) float64 {
	if w, ok := ft.ItemWeights[kind]; ok {
		return w
	}
	return 1.0
}

// EnemyScaling controls party size distribution per depth.
// Lone is probability of a lone (size 1) party; 0.7 at floor 0 -> 0.2 at floor 7.
// Canonical key is "lone" only — aliases loneByFloor/loneChance were removed per H20; use lone.
type EnemyScaling struct {
	Lone []float64 `json:"lone"`
}

// effectiveLone returns the backing slice (hard error if missing).
func (e EnemyScaling) effectiveLone() []float64 {
	if len(e.Lone) == 0 {
		panic("world.json: enemyScaling.lone missing or empty — single source required")
	}
	return e.Lone
}

// WorldConfig holds pacing tables loaded from world.json.
type WorldConfig struct {
	RecruitmentRate         float64              `json:"recruitmentRate"`
	RecruitmentRatePerFloor []float64            `json:"recruitmentRatePerFloor"`
	EnemyScaling            EnemyScaling         `json:"enemyScaling"`
	ItemWeighting           map[string][]float64 `json:"itemWeighting"`
	ItemWeights             map[string][]float64 `json:"itemWeights"`
	LevelBiomes             [][]string           `json:"levelBiomes"`
}

// RecruitmentChance returns recruitment chance for a floor (0.3 default).
func (w WorldConfig) RecruitmentChance(floor int) float64 {
	if len(w.RecruitmentRatePerFloor) > 0 {
		if floor < 0 {
			floor = 0
		}
		if floor < len(w.RecruitmentRatePerFloor) {
			return w.RecruitmentRatePerFloor[floor]
		}
		return w.RecruitmentRatePerFloor[len(w.RecruitmentRatePerFloor)-1]
	}
	if w.RecruitmentRate != 0 {
		return w.RecruitmentRate
	}
	return 0.3
}

// LoneChance returns lone-enemy probability for floor, clamped.
func (w WorldConfig) LoneChance(floor int) float64 {
	arr := w.EnemyScaling.effectiveLone()
	if floor < 0 {
		floor = 0
	}
	if floor >= len(arr) {
		floor = len(arr) - 1
	}
	return arr[floor]
}

// BiomeOptions returns the candidate biome ids for a floor from the
// levelBiomes table (clamped). Empty when the table is absent.
func (w WorldConfig) BiomeOptions(floor int) []string {
	if len(w.LevelBiomes) == 0 {
		return nil
	}
	if floor < 0 {
		floor = 0
	}
	if floor >= len(w.LevelBiomes) {
		floor = len(w.LevelBiomes) - 1
	}
	return w.LevelBiomes[floor]
}

// ItemWeight returns per-depth item weight (1.0 if missing).
func (w WorldConfig) ItemWeight(kind string, floor int) float64 {
	m := w.ItemWeighting
	if len(m) == 0 {
		m = w.ItemWeights
	}
	arr, ok := m[kind]
	if !ok || len(arr) == 0 {
		return 1.0
	}
	if floor < 0 {
		floor = 0
	}
	if floor >= len(arr) {
		floor = len(arr) - 1
	}
	return arr[floor]
}

type floorThemesFile struct {
	Themes []FloorTheme `json:"themes"`
	Notes  string       `json:"notes"`
}

var (
	floorThemesOnce  sync.Once
	floorThemesCache []FloorTheme
	worldOnce        sync.Once
	worldCache       WorldConfig
	worldHasCache    bool
)

// LoadFloorThemes returns all floor themes (cached via sync.Once, hard error if missing).
// Single source: game/data/floorThemes.json via RawJSON (which may overlay localStorage in wasm).
func LoadFloorThemes() []FloorTheme {
	floorThemesOnce.Do(func() {
		b, err := RawJSON("floorThemes.json")
		if err != nil {
			panic("floorThemes.json missing — single source required: " + err.Error())
		}
		var f floorThemesFile
		if err := json.Unmarshal(b, &f); err == nil && len(f.Themes) == 8 {
			floorThemesCache = f.Themes
			return
		}
		if len(f.Themes) > 0 {
			floorThemesCache = f.Themes
			return
		}
		var arr []FloorTheme
		if err := json.Unmarshal(b, &arr); err == nil && len(arr) > 0 {
			floorThemesCache = arr
			return
		}
		preview := b
		if len(preview) > 200 {
			preview = preview[:200]
		}
		panic("floorThemes.json invalid — single source required: " + string(preview))
	})
	out := make([]FloorTheme, len(floorThemesCache))
	copy(out, floorThemesCache)
	return out
}
func LoadWorldConfig() WorldConfig {
	worldOnce.Do(func() {
		b, err := RawJSON("world.json")
		if err != nil {
			panic("world.json missing — single source required: " + err.Error())
		}
		var w WorldConfig
		if err := json.Unmarshal(b, &w); err != nil {
			panic("world.json invalid — single source required: " + err.Error())
		}
		// Validate: must have lone array and recruitment.
		if len(w.EnemyScaling.Lone) == 0 {
			panic("world.json: enemyScaling.lone missing or empty — single source required")
		}
		if w.RecruitmentRate == 0 && len(w.RecruitmentRatePerFloor) == 0 {
			panic("world.json: recruitmentRate missing — single source required")
		}
		if len(w.ItemWeighting) == 0 && len(w.ItemWeights) == 0 {
			panic("world.json: itemWeighting/itemWeights missing — single source required")
		}
		// Ensure both maps populated for callers using either key.
		if len(w.ItemWeighting) == 0 && len(w.ItemWeights) > 0 {
			w.ItemWeighting = w.ItemWeights
		}
		if len(w.ItemWeights) == 0 && len(w.ItemWeighting) > 0 {
			w.ItemWeights = w.ItemWeighting
		}
		worldCache = w
		worldHasCache = true
	})
	return worldCache
}

// GetFloorTheme returns theme for floor (floor modulo theme count, clamped negative).
// Returns pointer to a copy so callers can mutate safely.
func GetFloorTheme(floor int) *FloorTheme {
	themes := LoadFloorThemes()
	if len(themes) == 0 {
		panic("floorThemes.json empty — single source required")
	}
	if floor < 0 {
		floor = 0
	}
	idx := floor % len(themes)
	t := themes[idx]
	return &t
}

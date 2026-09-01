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
type EnemyScaling struct {
	Lone        []float64 `json:"lone"`
	LoneByFloor []float64 `json:"loneByFloor"`
	LoneChance  []float64 `json:"loneChance"`
}

// effectiveLone returns the backing slice (tries Lone, then aliases, then fallback).
func (e EnemyScaling) effectiveLone() []float64 {
	if len(e.Lone) > 0 {
		return e.Lone
	}
	if len(e.LoneByFloor) > 0 {
		return e.LoneByFloor
	}
	if len(e.LoneChance) > 0 {
		return e.LoneChance
	}
	return fallbackWorldConfig().EnemyScaling.Lone
}

// WorldConfig holds pacing tables loaded from world.json.
type WorldConfig struct {
	RecruitmentRate         float64              `json:"recruitmentRate"`
	RecruitmentRatePerFloor []float64            `json:"recruitmentRatePerFloor"`
	EnemyScaling            EnemyScaling         `json:"enemyScaling"`
	ItemWeighting           map[string][]float64 `json:"itemWeighting"`
	ItemWeights             map[string][]float64 `json:"itemWeights"`
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

func fallbackFloorThemes() []FloorTheme {
	return []FloorTheme{
		{ID: "crypt", Name: "Crypt", WallGlyph: "#", FloorGlyph: ".", WallGlyphVariants: []string{"#", "#", "▓"}, FloorGlyphVariants: []string{".", "·", "·"}, Color: "#6a7a7a", Tint: "#6a7a7a", EnemyWeights: map[string]float64{"goblin": 1.1, "orc": 0.8, "kobold": 0.6, "rat": 1.0, "troll": 0.15}, ItemWeights: map[string]float64{"potion": 1.0, "scroll": 0.9, "ration": 1.2, "gold": 1.0}},
		{ID: "ossuary", Name: "Ossuary", WallGlyph: "#", FloorGlyph: ".", WallGlyphVariants: []string{"#", "#", "▒"}, FloorGlyphVariants: []string{".", "·", "⋅"}, Color: "#8a7a6a", Tint: "#8a7a6a", EnemyWeights: map[string]float64{"goblin": 0.9, "orc": 1.0, "kobold": 0.7, "rat": 0.9, "troll": 0.2}, ItemWeights: map[string]float64{"potion": 1.1, "scroll": 1.0, "ration": 1.0, "gold": 1.1}},
		{ID: "fungal", Name: "Fungal Grove", WallGlyph: "#", FloorGlyph: ".", WallGlyphVariants: []string{"#", "#", "♣"}, FloorGlyphVariants: []string{".", "·", ","}, Color: "#5a7a5a", Tint: "#5a7a5a", EnemyWeights: map[string]float64{"goblin": 0.8, "orc": 0.7, "kobold": 1.2, "rat": 1.3, "troll": 0.4}, ItemWeights: map[string]float64{"potion": 1.2, "scroll": 0.8, "ration": 1.1, "gold": 0.9}},
		{ID: "flooded", Name: "Flooded Vault", WallGlyph: "#", FloorGlyph: ".", WallGlyphVariants: []string{"#", "#", "≈"}, FloorGlyphVariants: []string{".", "·", "~"}, Color: "#5a6a7a", Tint: "#5a6a7a", EnemyWeights: map[string]float64{"goblin": 0.7, "orc": 0.9, "kobold": 1.1, "rat": 0.8, "troll": 0.5}, ItemWeights: map[string]float64{"potion": 1.0, "scroll": 1.1, "ration": 0.9, "gold": 1.0}},
		{ID: "sanctum", Name: "Sanctum", WallGlyph: "#", FloorGlyph: ".", WallGlyphVariants: []string{"#", "#", "▓"}, FloorGlyphVariants: []string{".", "·", "⋅"}, Color: "#7a7a6a", Tint: "#7a7a6a", EnemyWeights: map[string]float64{"goblin": 0.8, "orc": 1.1, "kobold": 0.9, "rat": 0.6, "troll": 0.55}, ItemWeights: map[string]float64{"potion": 0.9, "scroll": 1.2, "ration": 0.9, "gold": 1.2}},
		{ID: "cinder", Name: "Cinder Chapel", WallGlyph: "#", FloorGlyph: ".", WallGlyphVariants: []string{"#", "#", "▒"}, FloorGlyphVariants: []string{".", "·", "`"}, Color: "#7a6a6a", Tint: "#7a6a6a", EnemyWeights: map[string]float64{"goblin": 0.6, "orc": 1.2, "kobold": 1.0, "rat": 0.5, "troll": 0.7}, ItemWeights: map[string]float64{"potion": 0.9, "scroll": 1.0, "ration": 0.8, "gold": 1.3}},
		{ID: "infernal", Name: "Infernal Depths", WallGlyph: "#", FloorGlyph: ".", WallGlyphVariants: []string{"#", "#", "▓"}, FloorGlyphVariants: []string{".", "·", "⋅"}, Color: "#7a5a5a", Tint: "#7a5a5a", EnemyWeights: map[string]float64{"goblin": 0.5, "orc": 1.0, "kobold": 1.1, "rat": 0.4, "troll": 0.85}, ItemWeights: map[string]float64{"potion": 0.8, "scroll": 1.1, "ration": 0.7, "gold": 1.1}},
		{ID: "abyssal", Name: "Abyssal Void", WallGlyph: "#", FloorGlyph: ".", WallGlyphVariants: []string{"#", "#", "█"}, FloorGlyphVariants: []string{".", "·", " "}, Color: "#5a5a6a", Tint: "#5a5a6a", EnemyWeights: map[string]float64{"goblin": 0.4, "orc": 0.8, "kobold": 1.3, "rat": 0.3, "troll": 1.0}, ItemWeights: map[string]float64{"potion": 0.8, "scroll": 1.3, "ration": 0.6, "gold": 1.0}},
	}
}

func fallbackWorldConfig() WorldConfig {
	return WorldConfig{
		RecruitmentRate:         0.3,
		RecruitmentRatePerFloor: []float64{0.3, 0.3, 0.3, 0.3, 0.3, 0.3, 0.3, 0.3},
		EnemyScaling: EnemyScaling{
			Lone: []float64{0.7, 0.6, 0.5, 0.4, 0.32, 0.26, 0.22, 0.2},
		},
		ItemWeighting: map[string][]float64{
			"potion": {1.0, 1.0, 1.1, 1.1, 1.0, 0.9, 0.9, 0.8},
			"scroll": {0.8, 0.9, 1.0, 1.0, 1.1, 1.1, 1.0, 1.0},
			"ration": {1.2, 1.1, 1.0, 1.0, 0.9, 0.9, 0.8, 0.7},
			"gold":   {1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
		},
		ItemWeights: map[string][]float64{
			"potion": {1.0, 1.0, 1.1, 1.1, 1.0, 0.9, 0.9, 0.8},
			"scroll": {0.8, 0.9, 1.0, 1.0, 1.1, 1.1, 1.0, 1.0},
			"ration": {1.2, 1.1, 1.0, 1.0, 0.9, 0.9, 0.8, 0.7},
			"gold":   {1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
		},
	}
}

// LoadFloorThemes returns all floor themes (cached via sync.Once, fallback on error).
func LoadFloorThemes() []FloorTheme {
	floorThemesOnce.Do(func() {
		b, err := RawJSON("floorThemes.json")
		if err != nil {
			floorThemesCache = fallbackFloorThemes()
			return
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
		floorThemesCache = fallbackFloorThemes()
	})
	out := make([]FloorTheme, len(floorThemesCache))
	copy(out, floorThemesCache)
	return out
}

// LoadWorldConfig returns world pacing config (cached via sync.Once, fallback on error).
func LoadWorldConfig() WorldConfig {
	worldOnce.Do(func() {
		b, err := RawJSON("world.json")
		if err != nil {
			worldCache = fallbackWorldConfig()
			worldHasCache = true
			return
		}
		var w WorldConfig
		if err := json.Unmarshal(b, &w); err != nil {
			worldCache = fallbackWorldConfig()
			worldHasCache = true
			return
		}
		// Validate: must have 8-length lone array and recruitment.
		if len(w.EnemyScaling.effectiveLone()) < 8 && len(w.EnemyScaling.Lone) == 0 && len(w.EnemyScaling.LoneByFloor) == 0 && len(w.EnemyScaling.LoneChance) == 0 {
			fb := fallbackWorldConfig()
			w.EnemyScaling = fb.EnemyScaling
		}
		if w.RecruitmentRate == 0 && len(w.RecruitmentRatePerFloor) == 0 {
			fb := fallbackWorldConfig()
			w.RecruitmentRate = fb.RecruitmentRate
			w.RecruitmentRatePerFloor = fb.RecruitmentRatePerFloor
		}
		if len(w.ItemWeighting) == 0 && len(w.ItemWeights) == 0 {
			fb := fallbackWorldConfig()
			w.ItemWeighting = fb.ItemWeighting
			w.ItemWeights = fb.ItemWeights
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
		themes = fallbackFloorThemes()
	}
	if floor < 0 {
		floor = 0
	}
	idx := floor % len(themes)
	t := themes[idx]
	return &t
}

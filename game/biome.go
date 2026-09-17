package game

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"sync"
)

// ColorPalette holds desaturated per-biome tint colors.
type ColorPalette struct {
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
	Accent    string `json:"accent"`
}

// LitterDef holds litter kinds per walkability category.
type LitterDef struct {
	Destructible []string `json:"destructible"`
	Passable     []string `json:"passable"`
	Impassable   []string `json:"impassable"`
}

// Biome is one dungeon biome.
type Biome struct {
	ID                 string       `json:"id"`
	Name               string       `json:"name"`
	EligibleLevelRange [2]int       `json:"eligibleLevelRange"`
	ColorPalette       ColorPalette `json:"colorPalette"`
	// Legacy tint fields mirrored from palette for fallback/themes compatibility.
	Color            string    `json:"color,omitempty"`
	Tint             string    `json:"tint,omitempty"`
	Litter           LitterDef `json:"litter"`
	GenerationMethod string    `json:"generationMethod"`
	SpecialEnemies   []string  `json:"specialEnemies"`
	Ambience         []string  `json:"ambience"`
	// Optional glyph variants (not required but allowed for palette tint).
	WallGlyphVariants  []string `json:"wallGlyphVariants,omitempty"`
	FloorGlyphVariants []string `json:"floorGlyphVariants,omitempty"`
}

// LitterDefData defines a single litter kind data-driven (game/data/litter.json).
// Glyph is a single-character string (e.g. "]", "|", "=") decoded to rune.
// Color is hex or empty for neutral floor. Category is destructible|passable|impassable.
// HP is for destructibles (0 for others). AltBump is custom bump message or empty.
type LitterDefData struct {
	Kind     string `json:"kind"`
	Glyph    string `json:"glyph"`
	Color    string `json:"color"`
	Category string `json:"category"`
	HP       int    `json:"hp"`
	AltBump  string `json:"altBump"`
}

// LitterObj is a placed litter instance on a level.
type LitterObj struct {
	Pos            Pos    `json:"pos"`
	Kind           string `json:"kind"`
	Category       string `json:"category"` // destructible | passable | impassable
	Glyph          rune   `json:"glyph"`
	BlocksMovement bool   `json:"blocksMovement"`
	BlocksFOV      bool   `json:"blocksFOV"`
	HP             int    `json:"hp,omitempty"`
	MaxHP          int    `json:"maxHP,omitempty"`
	Color          string `json:"color,omitempty"` // hex or token for per-biome tint
	Hits           int    `json:"hits,omitempty"`  // bumps into destructible (first is bump, later are attacks)
}
type biomesFile struct {
	Biomes []Biome `json:"biomes"`
	Notes  string  `json:"notes"`
}

var (
	biomesOnce  sync.Once
	biomesCache []Biome
)

var (
	litterOnce  sync.Once
	litterCache map[string]LitterDefData
)

func fallbackLitterDefs() map[string]LitterDefData {
	return map[string]LitterDefData{
		"barrel":        {Kind: "barrel", Glyph: "]", Color: "", Category: "destructible", HP: 8, AltBump: ""},
		"ash_barrel":    {Kind: "ash_barrel", Glyph: "]", Color: "#8a5a3a", Category: "destructible", HP: 8, AltBump: ""},
		"crate":         {Kind: "crate", Glyph: "=", Color: "", Category: "destructible", HP: 10, AltBump: ""},
		"cinder_block":  {Kind: "cinder_block", Glyph: "=", Color: "#8a5a3a", Category: "destructible", HP: 14, AltBump: ""},
		"urn":           {Kind: "urn", Glyph: ")", Color: "#6a7a7a", Category: "destructible", HP: 5, AltBump: ""},
		"bone_pile":     {Kind: "bone_pile", Glyph: ":", Color: "#9a7a5a", Category: "destructible", HP: 10, AltBump: ""},
		"mushroom_cap":  {Kind: "mushroom_cap", Glyph: "%", Color: "#5a9a5a", Category: "destructible", HP: 6, AltBump: ""},
		"spore_pod":     {Kind: "spore_pod", Glyph: "*", Color: "#5a9a5a", Category: "destructible", HP: 5, AltBump: ""},
		"vine_cluster":  {Kind: "vine_cluster", Glyph: "\"", Color: "#4a8a3a", Category: "destructible", HP: 10, AltBump: ""},
		"rubble":        {Kind: "rubble", Glyph: ",", Color: "", Category: "passable", HP: 0, AltBump: ""},
		"rubble_wall":   {Kind: "rubble_wall", Glyph: ",", Color: "#9a7a5a", Category: "impassable", HP: 0, AltBump: "The rubble wall is too unstable to cross."},
		"bone_dust":     {Kind: "bone_dust", Glyph: ",", Color: "#9a7a5a", Category: "passable", HP: 0, AltBump: ""},
		"dust":          {Kind: "dust", Glyph: ".", Color: "", Category: "passable", HP: 0, AltBump: ""},
		"ash":           {Kind: "ash", Glyph: ".", Color: "#8a5a3a", Category: "passable", HP: 0, AltBump: ""},
		"puddle":        {Kind: "puddle", Glyph: "~", Color: "#5a6a7a", Category: "passable", HP: 0, AltBump: ""},
		"slime":         {Kind: "slime", Glyph: "~", Color: "#4a9a4a", Category: "passable", HP: 0, AltBump: ""},
		"moss":          {Kind: "moss", Glyph: "\"", Color: "#4a7a3a", Category: "passable", HP: 0, AltBump: ""},
		"column":        {Kind: "column", Glyph: "|", Color: "#6a7a7a", Category: "impassable", HP: 0, AltBump: "The column is unyielding stone."},
		"altar":         {Kind: "altar", Glyph: "_", Color: "#6a7a7a", Category: "impassable", HP: 0, AltBump: "The altar is immovable, humming faintly."},
		"sarcophagus":   {Kind: "sarcophagus", Glyph: "_", Color: "#6a7a7a", Category: "impassable", HP: 0, AltBump: "The sarcophagus is sealed shut."},
		"bone_column":   {Kind: "bone_column", Glyph: "|", Color: "#9a7a5a", Category: "impassable", HP: 0, AltBump: "The column is unyielding stone."},
		"fungal_column": {Kind: "fungal_column", Glyph: "|", Color: "#5a9a5a", Category: "impassable", HP: 0, AltBump: "The fungal column is rooted deep."},
		"vine_wall":     {Kind: "vine_wall", Glyph: "|", Color: "#4a8a3a", Category: "impassable", HP: 0, AltBump: "Vines block the way, pulsing faintly."},
		"cinder_column": {Kind: "cinder_column", Glyph: "|", Color: "#8a5a3a", Category: "impassable", HP: 0, AltBump: "The column is unyielding stone."},
		"pit":           {Kind: "pit", Glyph: "0", Color: "", Category: "impassable", HP: 0, AltBump: "The darkness extends deep below — you cannot pass."},
		"lava_pit":      {Kind: "lava_pit", Glyph: "0", Color: "#9a4a2a", Category: "impassable", HP: 0, AltBump: "Heat shimmers over the lava pit — the edge holds."},
		"thicket":       {Kind: "thicket", Glyph: "#", Color: "#4a8a3a", Category: "impassable", HP: 0, AltBump: "The thicket is too dense to push through."},
	}
}

func loadLitterData() {
	litterOnce.Do(func() {
		b, err := dataFS.ReadFile("data/litter.json")
		if err != nil {
			litterCache = fallbackLitterDefs()
			return
		}
		var arr []LitterDefData
		if err := json.Unmarshal(b, &arr); err != nil || len(arr) == 0 {
			litterCache = fallbackLitterDefs()
			return
		}
		m := make(map[string]LitterDefData, len(arr))
		for _, d := range arr {
			m[d.Kind] = d
		}
		// Fill any missing kinds from fallback so old saves and partial files still work.
		for k, v := range fallbackLitterDefs() {
			if _, ok := m[k]; !ok {
				m[k] = v
			}
		}
		litterCache = m
	})
}

func fallbackBiomes() []Biome {
	return []Biome{
		{
			ID: "crypt", Name: "Crypt", EligibleLevelRange: [2]int{1, 2},
			ColorPalette: ColorPalette{Primary: "#6a7a7a", Secondary: "#5a6a6a", Accent: "#8a7a7a"}, Color: "#6a7a7a", Tint: "#6a7a7a",
			Litter:           LitterDef{Destructible: []string{"barrel", "crate", "urn"}, Passable: []string{"rubble", "dust", "puddle"}, Impassable: []string{"column", "altar", "sarcophagus"}},
			GenerationMethod: "rooms", SpecialEnemies: []string{"wight", "kobold"},
			Ambience: []string{"Cold drafts curl through the crypt aisles.", "Distant stone lids grind against their sarcophagi.", "A faint incense of old myrrh hangs in the air.", "Footsteps echo too long in the vaulted dark.", "Candle smoke ghosts along the low arches.", "Water drips somewhere beyond the pillars, patient and cold.", "An unlit censer sways faintly, though no hand touched it.", "Mortar dust sifts from the vault above, catching dim light."},
		},
		{
			ID: "ossuary", Name: "Ossuary", EligibleLevelRange: [2]int{2, 3},
			ColorPalette: ColorPalette{Primary: "#7a5a3a", Secondary: "#6a4a32", Accent: "#9a7a5a"}, Color: "#7a5a3a", Tint: "#7a5a3a",
			Litter:           LitterDef{Destructible: []string{"bone_pile", "crate", "barrel"}, Passable: []string{"rubble", "bone_dust", "puddle"}, Impassable: []string{"bone_column", "pit", "rubble_wall"}},
			GenerationMethod: "rooms", SpecialEnemies: []string{"charnel", "orc"},
			Ambience: []string{"Bones whisper as dust shifts across the ossuary.", "A hollow clatter rolls from stacked skulls.", "Chalky air catches in your throat.", "Shadows pool between leaning bone-columns.", "Something small scuttles among the remains.", "A femur settles with a soft, tired crack.", "Dry air tastes of chalk and old marrow.", "Pale skulls watch with empty, patient sockets."},
		},
		{
			ID: "fungal", Name: "Fungal Grove", EligibleLevelRange: [2]int{3, 4},
			ColorPalette: ColorPalette{Primary: "#4a6a4a", Secondary: "#2e4a35", Accent: "#5a8a5a"}, Color: "#4a6a4a", Tint: "#4a6a4a",
			Litter:           LitterDef{Destructible: []string{"mushroom_cap", "spore_pod", "crate"}, Passable: []string{"moss", "slime", "rubble"}, Impassable: []string{"fungal_column", "pit", "thicket"}},
			GenerationMethod: "cavern", SpecialEnemies: []string{"spore_mother", "beetle"},
			Ambience: []string{"Spores drift like pale snow through fungal gloom.", "Mushroom caps pulse with faint light.", "The air is thick, sweet, and slightly sour.", "Soft caps sigh as you brush past.", "Mycelial threads hum beneath the floor.", "A damp, earthy perfume clings to every breath.", "Distant caps release a soft puff of glowing dust.", "The floor gives slightly, spongy with hidden growth."},
		},
		{
			ID: "jungle", Name: "Jungle Overgrowth", EligibleLevelRange: [2]int{4, 5},
			ColorPalette: ColorPalette{Primary: "#3d5a3a", Secondary: "#2a3d2f", Accent: "#4a7a4a"}, Color: "#3d5a3a", Tint: "#3d5a3a",
			Litter:           LitterDef{Destructible: []string{"vine_cluster", "crate", "barrel"}, Passable: []string{"moss", "rubble", "puddle"}, Impassable: []string{"vine_wall", "pit", "column"}},
			GenerationMethod: "cavern", SpecialEnemies: []string{"vine_horror", "mite"},
			Ambience: []string{"Vines tighten overhead with a soft creak.", "Humid air beads on cold stone.", "Leaves rustle where no wind should reach.", "A distant vine snaps taut, then stills.", "Roots have cracked the temple walls below.", "Warm rot and green perfume hang heavy in the air.", "Something unseen pushes through tangled fronds.", "Moss muffles your steps like a living carpet."},
		},
		{
			ID: "cinder", Name: "Cinder Chapel", EligibleLevelRange: [2]int{5, 8},
			ColorPalette: ColorPalette{Primary: "#6a3d2f", Secondary: "#4a2e2a", Accent: "#8a5a45"}, Color: "#6a3d2f", Tint: "#6a3d2f",
			Litter:           LitterDef{Destructible: []string{"ash_barrel", "crate", "cinder_block"}, Passable: []string{"ash", "rubble", "puddle"}, Impassable: []string{"cinder_column", "lava_pit", "rubble_wall"}},
			GenerationMethod: "cavern", SpecialEnemies: []string{"ember", "troll"},
			Ambience: []string{"Ash drifts on heat that has no source.", "Cinder clicks underfoot, cooling and cracking.", "A low draft carries the tang of soot.", "Embers blink in the dark like tired eyes.", "Stone sweats with old, trapped heat.", "Cracked tiles tick as they cool in the dark.", "A faint, acrid haze stings the eyes.", "Distant stone sighs as heat shifts through old flues."},
		},
		{
			ID: "gatehouse", Name: "Gatehouse", EligibleLevelRange: [2]int{1, 2},
			ColorPalette: ColorPalette{Primary: "#6a6a7a", Secondary: "#5a5a6a", Accent: "#8a8a9a"}, Color: "#6a6a7a", Tint: "#6a6a7a",
			Litter:           LitterDef{Destructible: []string{"barrel", "crate", "urn"}, Passable: []string{"rubble", "dust", "puddle"}, Impassable: []string{"column", "rubble_wall", "altar"}},
			GenerationMethod: "rooms", SpecialEnemies: []string{"warden", "goblin"},
			Ambience: []string{"Chains clink somewhere above the gate arch.", "A rusted portcullis hangs crooked in its slot.", "Cold air funnels through the murder holes."},
		},
		{
			ID: "cells", Name: "Cells", EligibleLevelRange: [2]int{1, 3},
			ColorPalette: ColorPalette{Primary: "#5a5a62", Secondary: "#4a4a52", Accent: "#7a7a82"}, Color: "#5a5a62", Tint: "#5a5a62",
			Litter:           LitterDef{Destructible: []string{"barrel", "crate", "bone_pile"}, Passable: []string{"dust", "rubble", "puddle"}, Impassable: []string{"column", "rubble_wall", "pit"}},
			GenerationMethod: "rooms", SpecialEnemies: []string{"jailer", "rat"},
			Ambience: []string{"Empty manacles chime softly in the dark.", "Scratch marks count days on the cell doors.", "A draft moans through the barred slots."},
		},
		{
			ID: "warren", Name: "Warren", EligibleLevelRange: [2]int{2, 2},
			ColorPalette: ColorPalette{Primary: "#6a5a4a", Secondary: "#54463c", Accent: "#8a7a6a"}, Color: "#6a5a4a", Tint: "#6a5a4a",
			Litter:           LitterDef{Destructible: []string{"crate", "barrel", "bone_pile"}, Passable: []string{"dust", "rubble", "moss"}, Impassable: []string{"rubble_wall", "pit", "column"}},
			GenerationMethod: "cavern", SpecialEnemies: []string{"beetle", "rat"},
			Ambience: []string{"Burrow mouths pock the soft walls.", "Something gnaws steadily, out of sight.", "Loose earth sifts from the low ceiling."},
		},
		{
			ID: "sunken", Name: "Sunken Chapel", EligibleLevelRange: [2]int{3, 3},
			ColorPalette: ColorPalette{Primary: "#4a6a72", Secondary: "#3a555c", Accent: "#6a8a92"}, Color: "#4a6a72", Tint: "#4a6a72",
			Litter:           LitterDef{Destructible: []string{"barrel", "crate", "urn"}, Passable: []string{"puddle", "slime", "moss"}, Impassable: []string{"column", "pit", "altar"}},
			GenerationMethod: "rooms", SpecialEnemies: []string{"drowned", "slime"},
			Ambience: []string{"Black water laps at the sunken flagstones.", "Algae slicks the lower steps.", "Bells toll dully from below the waterline."},
		},
		{
			ID: "archive", Name: "Archive", EligibleLevelRange: [2]int{4, 6},
			ColorPalette: ColorPalette{Primary: "#7a6a4a", Secondary: "#62563e", Accent: "#9a8a6a"}, Color: "#7a6a4a", Tint: "#7a6a4a",
			Litter:           LitterDef{Destructible: []string{"crate", "barrel", "urn"}, Passable: []string{"dust", "rubble", "puddle"}, Impassable: []string{"column", "sarcophagus", "rubble_wall"}},
			GenerationMethod: "rooms", SpecialEnemies: []string{"cultist", "mite"},
			Ambience: []string{"Scroll dust hangs thick as curtain gauze.", "A page turns somewhere, with no hand to turn it.", "Whispers file themselves between the stacks."},
		},
		{
			ID: "sanctum", Name: "Sanctum", EligibleLevelRange: [2]int{5, 5},
			ColorPalette: ColorPalette{Primary: "#8a7a5a", Secondary: "#6e6248", Accent: "#aa9a72"}, Color: "#8a7a5a", Tint: "#8a7a5a",
			Litter:           LitterDef{Destructible: []string{"urn", "barrel", "crate"}, Passable: []string{"dust", "rubble", "puddle"}, Impassable: []string{"altar", "column", "sarcophagus"}},
			GenerationMethod: "rooms", SpecialEnemies: []string{"penitent", "cultist"},
			Ambience: []string{"Candle rows burn for no congregation.", "A choir hums one note, then stops.", "The hush here feels listened-to."},
		},
		{
			ID: "infernal", Name: "Infernal Chapel", EligibleLevelRange: [2]int{6, 8},
			ColorPalette: ColorPalette{Primary: "#7a3a2a", Secondary: "#5e2e22", Accent: "#9a5a42"}, Color: "#7a3a2a", Tint: "#7a3a2a",
			Litter:           LitterDef{Destructible: []string{"ash_barrel", "crate", "cinder_block"}, Passable: []string{"ash", "rubble", "puddle"}, Impassable: []string{"cinder_column", "lava_pit", "rubble_wall"}},
			GenerationMethod: "cavern", SpecialEnemies: []string{"hulk", "imp"},
			Ambience: []string{"Brimstone sweat beads on the black stone.", "Scorch marks climb the walls like ivy.", "Far below, something vast turns over."},
		},
		{
			ID: "throne", Name: "Throne Room", EligibleLevelRange: [2]int{7, 8},
			ColorPalette: ColorPalette{Primary: "#7a6a3a", Secondary: "#62562e", Accent: "#9a8a52"}, Color: "#7a6a3a", Tint: "#7a6a3a",
			Litter:           LitterDef{Destructible: []string{"urn", "crate", "barrel"}, Passable: []string{"dust", "rubble", "moss"}, Impassable: []string{"column", "altar", "sarcophagus"}},
			GenerationMethod: "rooms", SpecialEnemies: []string{"wraith", "husk"},
			Ambience: []string{"A throne of fused bone overlooks the hall.", "A crown of black glass sits empty, waiting.", "Courtier ghosts hold their last positions."},
		},
		{
			ID: "abyss", Name: "Abyssal Rift", EligibleLevelRange: [2]int{8, 8},
			ColorPalette: ColorPalette{Primary: "#3a3a52", Secondary: "#2e2e42", Accent: "#565670"}, Color: "#3a3a52", Tint: "#3a3a52",
			Litter:           LitterDef{Destructible: []string{"bone_pile", "crate", "ash_barrel"}, Passable: []string{"ash", "dust", "rubble"}, Impassable: []string{"pit", "rubble_wall", "bone_column"}},
			GenerationMethod: "cavern", SpecialEnemies: []string{"gloom", "bat"},
			Ambience: []string{"The dark below has a texture, like oil.", "Echoes return saying things you did not say.", "Cold rises in slow, deliberate waves."},
		},
	}
}

// LoadBiomes returns all biomes (cached, fallback on error).
func LoadBiomes() []Biome {
	biomesOnce.Do(func() {
		b, err := dataFS.ReadFile("data/biomes.json")
		if err != nil {
			biomesCache = fallbackBiomes()
			return
		}
		var f biomesFile
		if err := json.Unmarshal(b, &f); err == nil && len(f.Biomes) > 0 {
			biomesCache = f.Biomes
			return
		}
		var arr []Biome
		if err := json.Unmarshal(b, &arr); err == nil && len(arr) > 0 {
			biomesCache = arr
			return
		}
		biomesCache = fallbackBiomes()
	})
	out := make([]Biome, len(biomesCache))
	copy(out, biomesCache)
	return out
}

// isEligible reports whether levelNum (1-indexed) falls within br.
func isEligible(levelNum int, br [2]int) bool {
	if br[0] == 0 && br[1] == 0 {
		return true
	}
	mn, mx := br[0], br[1]
	if mn == 0 {
		mn = 1
	}
	if mx == 0 {
		mx = mn
	}
	if mn > mx {
		mn, mx = mx, mn
	}
	return levelNum >= mn && levelNum <= mx
}

// GetBiomeForFloor returns a biome for floor (0-indexed) from the per-level
// world.json table, drawn with the caller's rng so runs differ. Unknown table
// ids are skipped; an empty table or no match falls back to eligible ranges.
func GetBiomeForFloor(floor int, rng *rand.Rand) *Biome {
	biomes := LoadBiomes()
	if len(biomes) == 0 {
		fb := fallbackBiomes()
		biomes = fb
	}
	byID := make(map[string]Biome, len(biomes))
	for _, b := range biomes {
		byID[b.ID] = b
	}
	if ids := LoadWorldConfig().BiomeOptions(floor); len(ids) > 0 {
		var options []Biome
		for _, id := range ids {
			if b, ok := byID[id]; ok {
				options = append(options, b)
			}
		}
		if len(options) == 1 {
			b := options[0]
			return &b
		}
		if len(options) > 1 {
			if rng == nil {
				b := options[0]
				return &b
			}
			b := options[rng.IntN(len(options))]
			return &b
		}
	}
	levelNum := floor + 1 // biomes.json uses 1-indexed ranges per spec
	var eligible []Biome
	for _, b := range biomes {
		if isEligible(levelNum, b.EligibleLevelRange) {
			eligible = append(eligible, b)
		}
	}
	if len(eligible) == 0 {
		// No eligible — fall back to closest by distance to range.
		b := biomes[0]
		return &b
	}
	if len(eligible) == 1 {
		b := eligible[0]
		return &b
	}
	if rng == nil {
		b := eligible[0]
		return &b
	}
	b := eligible[rng.IntN(len(eligible))]
	return &b
}

// TintColor returns desaturated tint for biome (palette primary fallback).
func (b Biome) TintColor() string {
	if b.ColorPalette.Primary != "" {
		return b.ColorPalette.Primary
	}
	if b.Tint != "" {
		return b.Tint
	}
	if b.Color != "" {
		return b.Color
	}
	return "#6a7a7a"
}

// PrimaryColor shortcut.
func (b Biome) PrimaryColor() string { return b.TintColor() }

func (b Biome) FloorColor() string {
	switch b.ID {
	case "crypt":
		return "#4a4642" // neutral stone (base)
	case "ossuary":
		return "#4f3d32" // warm tan bone
	case "fungal":
		return "#2e4a35" // greenish damp
	case "jungle":
		return "#2a3d2f" // deeper green overgrown
	case "cinder":
		return "#4a2e2a" // ashen reddish
	default:
		// fallback: darken primary toward bg #141210
		return "#4a4642"
	}
}

func (b Biome) WallColor() string {
	switch b.ID {
	case "crypt":
		return "#6b645c" // neutral
	case "ossuary":
		return "#7a5a3a" // tanner warm
	case "fungal":
		return "#4a6a4a" // greener
	case "jungle":
		return "#3d5a3a" // deep green
	case "cinder":
		return "#6a3d2f" // redder
	default:
		return "#6b645c"
	}
}

func FloorColorForLevel(lvl *Level) string {
	if lvl == nil || lvl.BiomeID == "" {
		return "#4a4642"
	}
	for _, b := range LoadBiomes() {
		if b.ID == lvl.BiomeID {
			return b.FloorColor()
		}
	}
	return "#4a4642"
}

func WallColorForLevel(lvl *Level) string {
	if lvl == nil || lvl.BiomeID == "" {
		return "#6b645c"
	}
	for _, b := range LoadBiomes() {
		if b.ID == lvl.BiomeID {
			return b.WallColor()
		}
	}
	return "#6b645c"
}

func litterGlyph(kind string) rune {
	loadLitterData()
	if d, ok := litterCache[kind]; ok && d.Glyph != "" {
		for _, r := range d.Glyph {
			return r
		}
	}
	switch kind {
	case "barrel", "ash_barrel":
		return ']'
	case "crate", "cinder_block":
		return '='
	case "urn":
		return ')'
	case "bone_pile":
		return ':'
	case "mushroom_cap":
		return '%'
	case "spore_pod":
		return '*'
	case "vine_cluster":
		return '"'
	case "rubble", "rubble_wall", "bone_dust":
		return ','
	case "dust", "ash":
		return '.'
	case "puddle", "slime":
		return '~'
	case "moss":
		return '"'
	case "column", "cinder_column", "bone_column", "fungal_column", "vine_wall":
		return '|'
	case "altar", "sarcophagus":
		return '_'
	case "pit", "lava_pit":
		return '0'
	case "thicket":
		return '#'
	default:
		return '.'
	}
}

func litterBlocks(kind, category string) (blocksMove, blocksFOV bool) {
	loadLitterData()
	cat := category
	if d, ok := litterCache[kind]; ok && d.Category != "" {
		cat = d.Category
	}
	switch cat {
	case "passable":
		return false, false
	case "destructible":
		// Destructible blocks movement until cleared, but not FOV (barrels are low).
		return true, false
	case "impassable":
		// Columns/walls block both; pits block movement but not FOV.
		if kind == "pit" || kind == "lava_pit" {
			return true, false
		}
		return true, true
	default:
		return false, false
	}
}
func newLitterObj(pos Pos, kind, category string) LitterObj {
	bm, bf := litterBlocks(kind, category)
	obj := LitterObj{Pos: pos, Kind: kind, Category: category, Glyph: litterGlyph(kind), BlocksMovement: bm, BlocksFOV: bf}
	// Assign HP for destructibles (requires value to break).
	if category == "destructible" {
		hp := litterHP(kind)
		obj.HP = hp
		obj.MaxHP = hp
	}
	// Assign per-biome color where appropriate.
	if col := litterColor(kind); col != "" {
		obj.Color = col
	}
	return obj
}

func litterHP(kind string) int {
	loadLitterData()
	if d, ok := litterCache[kind]; ok {
		// If kind exists in data, use its HP even if 0 (passable/impassable have 0).
		// Fallback default for unknown kinds is handled by switch below, but cache covers all known kinds.
		// Only return directly if the kind is known; HP 0 is valid for non-destructibles.
		return d.HP
	}
	switch kind {
	case "urn", "spore_pod":
		return 5
	case "barrel", "ash_barrel":
		return 8
	case "crate", "bone_pile", "vine_cluster":
		return 10
	case "mushroom_cap":
		return 6
	case "cinder_block":
		return 14
	default:
		return 8
	}
}

func litterColor(kind string) string {
	loadLitterData()
	if d, ok := litterCache[kind]; ok {
		return d.Color
	}
	switch kind {
	case "mushroom_cap", "spore_pod", "fungal_column":
		return "#5a9a5a" // fungal green — more saturated to match #4a6a4a walls
	case "vine_cluster", "vine_wall", "thicket":
		return "#4a8a3a" // jungle green — deeper, distinct from fungal
	case "moss":
		return "#4a7a3a" // jungle/fungal moss — ties to jungle wall #3d5a3a
	case "slime":
		return "#4a9a4a" // brighter fungal slime
	case "ash", "ash_barrel", "cinder_block", "cinder_column":
		return "#8a5a3a" // cinder ash — warmer, redder to match #6a3d2f walls
	case "lava_pit":
		return "#9a4a2a" // hot cinder — more saturated reddish
	case "bone_pile", "bone_column", "bone_dust", "rubble_wall":
		return "#9a7a5a" // ossuary bone — warmer tan matching #7a5a3a walls
	case "column", "altar", "sarcophagus", "urn":
		return "#6a7a7a" // crypt stone
	case "rubble", "dust":
		return "" // keep neutral floor
	case "puddle":
		return "#5a6a7a" // damp slate
	default:
		return ""
	}
}

func litterAltBump(kind string) (string, bool) {
	loadLitterData()
	if d, ok := litterCache[kind]; ok {
		if d.AltBump != "" {
			return d.AltBump, true
		}
		return "", false
	}
	switch kind {
	case "pit":
		return "The darkness extends deep below — you cannot pass.", true
	case "lava_pit":
		return "Heat shimmers over the lava pit — the edge holds.", true
	case "thicket":
		return "The thicket is too dense to push through.", true
	case "vine_wall":
		return "Vines block the way, pulsing faintly.", true
	case "fungal_column":
		return "The fungal column is rooted deep.", true
	case "column", "cinder_column", "bone_column":
		return "The column is unyielding stone.", true
	case "altar":
		return "The altar is immovable, humming faintly.", true
	case "sarcophagus":
		return "The sarcophagus is sealed shut.", true
	case "rubble_wall":
		return "The rubble wall is too unstable to cross.", true
	default:
		return "", false
	}
}

// floorForScaling returns floor index from BiomeID lookup via level's BiomeID.
// Stored separately to avoid adding floor param to Level.
func (l *Level) floorForScaling() int {
	return l.Floor
}

// ---------------------------------------------------------------------------
// Generation branching
// ---------------------------------------------------------------------------

// GenerateWithBiome fills level using biome's generationMethod, then spawns
// enemies (including special), and litter without blocking BFS. Deterministic from rng.
func (l *Level) GenerateWithBiome(rng *rand.Rand, floor int, biome *Biome) {
	if rng == nil {
		rng = rand.New(rand.NewPCG(0, 0))
	}
	if biome == nil {
		biome = GetBiomeForFloor(floor, rng)
	}
	l.BiomeID = biome.ID
	l.Floor = floor
	for attempt := 0; attempt < 50; attempt++ {
		if attempt == 0 {
			// Clear any prior state (in case of re-generation).
			l.Enemies = nil
			l.Features = nil
			l.Litter = nil
			l.Items = nil
			if l.Doors == nil {
				l.Doors = make(map[Pos]bool)
			}
		} else {
			// Redo: clear tiles to walls, reset collections, continue rng sequence (do not reseed)
			for y := range l.H {
				for x := range l.W {
					l.Tiles[y][x] = TileWall
				}
			}
			l.Enemies = nil
			l.Features = nil
			l.Litter = nil
			l.Items = nil
			l.Doors = make(map[Pos]bool)
		}
		// Branch generation (deterministic, continues rng sequence)
		switch biome.GenerationMethod {
		case "cavern":
			l.generateCavern(rng, floor)
		default:
			l.generateRooms(rng, floor)
		}
		// Ensure stairs are placed (both generators do, but double-check).
		if !l.InBounds(l.StairsUp) || !l.InBounds(l.StairsDown) {
			var walks []Pos
			for y := range l.H {
				for x := range l.W {
					p := Pos{x, y}
					if l.At(p) == TileFloor {
						walks = append(walks, p)
					}
				}
			}
			if len(walks) >= 2 {
				l.StairsUp = walks[0]
				l.Tiles[l.StairsUp.Y][l.StairsUp.X] = TileStairsUp
				l.StairsDown = walks[len(walks)-1]
				l.Tiles[l.StairsDown.Y][l.StairsDown.X] = TileStairsDown
			}
		}
		// Ensure connectivity: if stairs unreachable via tiles alone, carve emergency corridor.
		if !stairsReachableViaBFS(l) {
			ax, ay := l.StairsUp.X, l.StairsUp.Y
			bx, by := l.StairsDown.X, l.StairsDown.Y
			if rng.IntN(2) == 0 {
				for x := min(ax, bx); x <= max(ax, bx); x++ {
					p := Pos{x, ay}
					if l.InBounds(p) {
						l.Tiles[p.Y][p.X] = TileFloor
					}
				}
				for y := min(ay, by); y <= max(ay, by); y++ {
					p := Pos{bx, y}
					if l.InBounds(p) {
						l.Tiles[p.Y][p.X] = TileFloor
					}
				}
			} else {
				for y := min(ay, by); y <= max(ay, by); y++ {
					p := Pos{ax, y}
					if l.InBounds(p) {
						l.Tiles[p.Y][p.X] = TileFloor
					}
				}
				for x := min(ax, bx); x <= max(ax, bx); x++ {
					p := Pos{x, by}
					if l.InBounds(p) {
						l.Tiles[p.Y][p.X] = TileFloor
					}
				}
			}
			l.Tiles[l.StairsUp.Y][l.StairsUp.X] = TileStairsUp
			l.Tiles[l.StairsDown.Y][l.StairsDown.X] = TileStairsDown
			// Re-enforce vault walls after emergency corridor (which may have carved through vault).
			for _, vf := range l.Features {
				if !vf.IsVault() {
					continue
				}
				c := vf.Pos
				var door Pos
				bestDist := 1000
				foundDoor := false
				for y := 0; y < l.H; y++ {
					for x := 0; x < l.W; x++ {
						p := Pos{x, y}
						if l.At(p) != TileDoor {
							continue
						}
						dx := p.X - c.X
						if dx < 0 {
							dx = -dx
						}
						dy := p.Y - c.Y
						if dy < 0 {
							dy = -dy
						}
						d := dx + dy
						if d < bestDist && d <= 6 {
							bestDist = d
							door = p
							foundDoor = true
						}
					}
				}
				if !foundDoor {
					ow, oh := 7, 7
					ox := c.X - ow/2
					oy := c.Y - oh/2
					if ox < 1 {
						ox = 1
					}
					if oy < 1 {
						oy = 1
					}
					if ox+ow >= l.W {
						ox = l.W - ow - 1
					}
					if oy+oh >= l.H {
						oy = l.H - oh - 1
					}
					door = Pos{ox + ow/2, oy + oh - 1}
					for yy := oy; yy < oy+oh; yy++ {
						for xx := ox; xx < ox+ow; xx++ {
							isPerim := xx == ox || xx == ox+ow-1 || yy == oy || yy == oy+oh-1
							p := Pos{xx, yy}
							if p == door {
								l.Tiles[yy][xx] = TileDoor
								if l.Doors == nil {
									l.Doors = make(map[Pos]bool)
								}
								l.Doors[p] = false
								continue
							}
							if isPerim {
								l.Tiles[yy][xx] = TileWall
								if l.Doors != nil {
									delete(l.Doors, p)
								}
							} else {
								l.Tiles[yy][xx] = TileFloor
							}
						}
					}
					continue
				}
				left := c.X
				for left >= 0 {
					t := l.At(Pos{left, c.Y})
					if t == TileWall || t == TileDoor {
						break
					}
					left--
				}
				right := c.X
				for right < l.W {
					t := l.At(Pos{right, c.Y})
					if t == TileWall || t == TileDoor {
						break
					}
					right++
				}
				top := c.Y
				for top >= 0 {
					t := l.At(Pos{c.X, top})
					if t == TileWall || t == TileDoor {
						break
					}
					top--
				}
				bottom := c.Y
				for bottom < l.H {
					t := l.At(Pos{c.X, bottom})
					if t == TileWall || t == TileDoor {
						break
					}
					bottom++
				}
				outerW := right - left + 1
				outerH := bottom - top + 1
				if outerW < 7 || outerW > 9 || outerH < 7 || outerH > 9 {
					dx := door.X - c.X
					dy := door.Y - c.Y
					ow, oh := 7, 7
					var ox, oy int
					if dy < 0 && -dy > dx && -dy > -dx {
						ox = door.X - ow/2
						oy = door.Y
					} else if dy > 0 && dy > dx && dy > -dx {
						ox = door.X - ow/2
						oy = door.Y - oh + 1
					} else if dx < 0 {
						ox = door.X
						oy = door.Y - oh/2
					} else {
						ox = door.X - ow + 1
						oy = door.Y - oh/2
					}
					if ox < 1 {
						ox = 1
					}
					if oy < 1 {
						oy = 1
					}
					if ox+ow >= l.W {
						ox = l.W - ow - 1
					}
					if oy+oh >= l.H {
						oy = l.H - oh - 1
					}
					left = ox
					right = ox + ow - 1
					top = oy
					bottom = oy + oh - 1
				}
				for yy := top; yy <= bottom; yy++ {
					for xx := left; xx <= right; xx++ {
						isPerim := xx == left || xx == right || yy == top || yy == bottom
						if !isPerim {
							continue
						}
						p := Pos{xx, yy}
						if p == door {
							if l.At(p) != TileDoor {
								l.Tiles[yy][xx] = TileDoor
								if l.Doors == nil {
									l.Doors = make(map[Pos]bool)
								}
								l.Doors[p] = false
							}
							continue
						}
						if l.At(p) != TileWall {
							l.Tiles[yy][xx] = TileWall
							if l.Doors != nil {
								delete(l.Doors, p)
							}
						}
					}
				}
				for yy := top + 1; yy < bottom; yy++ {
					for xx := left + 1; xx < right; xx++ {
						if l.At(Pos{xx, yy}) != TileFloor {
							l.Tiles[yy][xx] = TileFloor
						}
					}
				}
			}
		}
// Vault re-enforce outside emergency corridor as well (always)
	for _, vf := range l.Features {
		if !vf.IsVault() {
			continue
		}
		c := vf.Pos
		var door Pos
		bestDist := 1000
		foundDoor := false
		for y := 0; y < l.H; y++ {
			for x := 0; x < l.W; x++ {
				p := Pos{x, y}
				if l.At(p) != TileDoor {
					continue
				}
				dx := p.X - c.X
				if dx < 0 {
					dx = -dx
				}
				dy := p.Y - c.Y
				if dy < 0 {
					dy = -dy
				}
				d := dx + dy
				if d < bestDist && d <= 6 {
					bestDist = d
					door = p
					foundDoor = true
				}
			}
		}
		if !foundDoor {
			ow, oh := 7, 7
			ox := c.X - ow/2
			oy := c.Y - oh/2
			if ox < 1 {
				ox = 1
			}
			if oy < 1 {
				oy = 1
			}
			if ox+ow >= l.W {
				ox = l.W - ow - 1
			}
			if oy+oh >= l.H {
				oy = l.H - oh - 1
			}
			door = Pos{ox + ow/2, oy + oh - 1}
			for yy := oy; yy < oy+oh; yy++ {
				for xx := ox; xx < ox+ow; xx++ {
					isPerim := xx == ox || xx == ox+ow-1 || yy == oy || yy == oy+oh-1
					p := Pos{xx, yy}
					if p == door {
						l.Tiles[yy][xx] = TileDoor
						if l.Doors == nil {
							l.Doors = make(map[Pos]bool)
						}
						l.Doors[p] = false
						continue
					}
					if isPerim {
						l.Tiles[yy][xx] = TileWall
						if l.Doors != nil {
							delete(l.Doors, p)
						}
					} else {
						l.Tiles[yy][xx] = TileFloor
					}
				}
			}
			continue
		}
		left := c.X
		for left >= 0 {
			t := l.At(Pos{left, c.Y})
			if t == TileWall || t == TileDoor {
				break
			}
			left--
		}
		right := c.X
		for right < l.W {
			t := l.At(Pos{right, c.Y})
			if t == TileWall || t == TileDoor {
				break
			}
			right++
		}
		top := c.Y
		for top >= 0 {
			t := l.At(Pos{c.X, top})
			if t == TileWall || t == TileDoor {
				break
			}
			top--
		}
		bottom := c.Y
		for bottom < l.H {
			t := l.At(Pos{c.X, bottom})
			if t == TileWall || t == TileDoor {
				break
			}
			bottom++
		}
		outerW := right - left + 1
		outerH := bottom - top + 1
		if outerW < 7 || outerW > 9 || outerH < 7 || outerH > 9 {
			dx := door.X - c.X
			dy := door.Y - c.Y
			ow, oh := 7, 7
			var ox, oy int
			if dy < 0 && -dy > dx && -dy > -dx {
				ox = door.X - ow/2
				oy = door.Y
			} else if dy > 0 && dy > dx && dy > -dx {
				ox = door.X - ow/2
				oy = door.Y - oh + 1
			} else if dx < 0 {
				ox = door.X
				oy = door.Y - oh/2
			} else {
				ox = door.X - ow + 1
				oy = door.Y - oh/2
			}
			if ox < 1 {
				ox = 1
			}
			if oy < 1 {
				oy = 1
			}
			if ox+ow >= l.W {
				ox = l.W - ow - 1
			}
			if oy+oh >= l.H {
				oy = l.H - oh - 1
			}
			left = ox
			right = ox + ow - 1
			top = oy
			bottom = oy + oh - 1
		}
		for yy := top; yy <= bottom; yy++ {
			for xx := left; xx <= right; xx++ {
				isPerim := xx == left || xx == right || yy == top || yy == bottom
				if !isPerim {
					continue
				}
				p := Pos{xx, yy}
				if p == door {
					if l.At(p) != TileDoor {
						l.Tiles[yy][xx] = TileDoor
						if l.Doors == nil {
							l.Doors = make(map[Pos]bool)
						}
						l.Doors[p] = false
					}
					continue
				}
				if l.At(p) != TileWall {
					l.Tiles[yy][xx] = TileWall
					if l.Doors != nil {
						delete(l.Doors, p)
					}
				}
			}
		}
		for yy := top + 1; yy < bottom; yy++ {
			for xx := left + 1; xx < right; xx++ {
				if l.At(Pos{xx, yy}) != TileFloor {
					l.Tiles[yy][xx] = TileFloor
				}
			}
		}
	}
	// Final door post-pass: remove any remaining doors without opposite walls or not single-wide.
	// Preserve vault doors (within 6 of a locked vault) even if not single-wide, as they must remain locked.
	for y := range l.H {
		for x := range l.W {
			p := Pos{x, y}
			if l.At(p) != TileDoor {
				continue
			}
			// Check if this is a vault door — skip removal.
			isVaultDoor := false
			for _, vf := range l.Features {
				if !vf.IsVault() {
					continue
				}
				dx := p.X - vf.Pos.X
				if dx < 0 {
					dx = -dx
				}
				dy := p.Y - vf.Pos.Y
				if dy < 0 {
					dy = -dy
				}
				if dx+dy <= 6 {
					isVaultDoor = true
					break
				}
			}
			if isVaultDoor {
				continue
			}
			if !isSingleWideDoor(l, p) {
				l.Tiles[y][x] = TileFloor
				if l.Doors != nil {
					delete(l.Doors, p)
				}
			}
		}
	}
	// Spawn enemies (depth-appropriate plus special) BEFORE litter/features so
	// candidates are all walkable floors (hallways included) not yet blocked
	// by litter impassables or feature placements. Doors are TileDoor and
	// excluded via At check, so closed doors do not reduce floor candidates.
	l.spawnEnemiesWithBiome(rng, floor, biome)
	// Spawn litter with BFS guard.
	spawnLitter(l, rng, biome)
	// Spawn level features (vault/forge/den/pitfall + merchant/fountain/shrine) via features.go.
	// Centralized here so both rooms and cavern paths populate features; Level.Generate delegates here.
	// Preserve special vault/merchant rooms carved in generateRooms (doors + features) and append random features.
	existing := l.Features
	l.Features = append(existing, MaybeSpawnFeatures(l, floor, rng)...)
	// FIX: ensure vault treasure ($) is inside vault interior (TileFloor, not wall/door/outside).
	// GenerateRooms already places vault at interior center with TileFloor; this hardens any
	// vault that ended up on wall/door (e.g., cavern fallback or random spawn) and avoids $ outside.
	for idx := range l.Features {
		f := &l.Features[idx]
		if !f.IsVault() {
			continue
		}
		c := f.Pos
		if l.InBounds(c) && l.At(c) == TileFloor && !l.IsDoor(c) {
			continue
		}
		// Vault on wall/door/outside — relocate to nearest vault door interior if possible.
		if l.InBounds(c) {
			l.Tiles[c.Y][c.X] = TileFloor
		}
		var door Pos
		bestDist := 1000
		foundDoor := false
		for y := range l.H {
			for x := range l.W {
				p := Pos{x, y}
				if l.At(p) != TileDoor {
					continue
				}
				dx := p.X - c.X
				if dx < 0 {
					dx = -dx
				}
				dy := p.Y - c.Y
				if dy < 0 {
					dy = -dy
				}
				d := dx + dy
				if d < bestDist && d <= 6 {
					bestDist = d
					door = p
					foundDoor = true
				}
			}
		}
		if foundDoor {
			// Find interior neighbor bounded by walls and move vault there.
			best := Pos{}
			found := false
			for _, d := range []Dir{DirN, DirS, DirE, DirW} {
				n := door.Add(d)
				if !l.InBounds(n) || l.At(n) != TileFloor {
					continue
				}
				// Prefer side that is enclosed (vault interior has walls close).
				left := n.X
				for left >= 0 {
					t := l.At(Pos{left, n.Y})
					if t == TileWall || t == TileDoor {
						break
					}
					left--
				}
				right := n.X
				for right < l.W {
					t := l.At(Pos{right, n.Y})
					if t == TileWall || t == TileDoor {
						break
					}
					right++
				}
				top := n.Y
				for top >= 0 {
					t := l.At(Pos{n.X, top})
					if t == TileWall || t == TileDoor {
						break
					}
					top--
				}
				bottom := n.Y
				for bottom < l.H {
					t := l.At(Pos{n.X, bottom})
					if t == TileWall || t == TileDoor {
						break
					}
					bottom++
				}
				if right-left+1 >= 7 && bottom-top+1 >= 7 {
					center := Pos{left + 1 + (right-left-1)/2, top + 1 + (bottom-top-1)/2}
					if l.InBounds(center) {
						l.Tiles[center.Y][center.X] = TileFloor
						f.Pos = center
						found = true
						break
					}
				}
				if !found {
					best = n
				}
			}
			if !found && (best != Pos{}) {
				l.Tiles[best.Y][best.X] = TileFloor
				f.Pos = best
			}
		}
	}
	// Spawn floor loot using same weighted table as wizard debug spawns.
	SpawnFloorLoot(l, rng, floor, biome)
	// Guarantee exit: after vault/doors/litter/features/loot, re-verify BFS and carve emergency corridor up to 5 times.
	ensureExitGuarantee(l, rng)
	if !AssertLevelHasExit(l) {
		debugGenLogf("WARN: floor %d exit not reachable after guarantee (biome %s up %v down %v)\n", floor, biome.ID, l.StairsUp, l.StairsDown)
		// Last-ditch: try both orientations once more.
		carveEmergencyCorridor(l, rng, true)
		carveEmergencyCorridor(l, rng, false)
		if !AssertLevelHasExit(l) {
			panic(fmt.Sprintf("GenerateWithBiome floor %d failed to guarantee exit (up %v down %v)", floor, l.StairsUp, l.StairsDown))
		}
	}
	// Final width-aware door cleanup after emergency corridors (which may have widened hallways).
	// Preserve vault doors even if not single-wide.
	for y := range l.H {
		for x := range l.W {
			p := Pos{x, y}
			if l.At(p) != TileDoor {
				continue
			}
			isVaultDoor := false
			for _, vf := range l.Features {
				if !vf.IsVault() {
					continue
				}
				dx := p.X - vf.Pos.X
				if dx < 0 {
					dx = -dx
				}
				dy := p.Y - vf.Pos.Y
				if dy < 0 {
					dy = -dy
				}
				if dx+dy <= 6 {
					isVaultDoor = true
					break
				}
			}
			if isVaultDoor {
				continue
			}
			if !isSingleWideDoor(l, p) {
				l.Tiles[y][x] = TileFloor
				if l.Doors != nil {
					delete(l.Doors, p)
				}
			}
		}
	}
	// Re-enforce vault walls after emergency corridor and door cleanup — ensure vault 7x7 outer intact
	// and treasure ($) at interior center (distance >=2 from walls, TileFloor, not door).
	for _, vf := range l.Features {
		if !vf.IsVault() {
			continue
		}
		c := vf.Pos
		// Find nearest vault door within 6, or fallback to south wall.
		var door Pos
		bestDist := 1000
		foundDoor := false
		for y := range l.H {
			for x := range l.W {
				p := Pos{x, y}
				if l.At(p) != TileDoor {
					continue
				}
				dx := p.X - c.X
				if dx < 0 {
					dx = -dx
				}
				dy := p.Y - c.Y
				if dy < 0 {
					dy = -dy
				}
				d := dx + dy
				if d < bestDist && d <= 6 {
					bestDist = d
					door = p
					foundDoor = true
				}
			}
		}
		var ox, oy, ow, oh int
		if foundDoor {
			// Derive outer from door and center: scan to find outer bounds, or fallback to 7x7 around center.
			left := c.X
			for left >= 0 {
				t := l.At(Pos{left, c.Y})
				if t == TileWall || t == TileDoor {
					break
				}
				left--
			}
			right := c.X
			for right < l.W {
				t := l.At(Pos{right, c.Y})
				if t == TileWall || t == TileDoor {
					break
				}
				right++
			}
			top := c.Y
			for top >= 0 {
				t := l.At(Pos{c.X, top})
				if t == TileWall || t == TileDoor {
					break
				}
				top--
			}
			bottom := c.Y
			for bottom < l.H {
				t := l.At(Pos{c.X, bottom})
				if t == TileWall || t == TileDoor {
					break
				}
				bottom++
			}
			outerW := right - left + 1
			outerH := bottom - top + 1
			if outerW >= 7 && outerW <= 9 && outerH >= 7 && outerH <= 9 {
				ox, oy, ow, oh = left, top, outerW, outerH
			} else {
				// Fallback: 7x7 around door based on door side
				ow, oh = 7, 7
				dx := door.X - c.X
				dy := door.Y - c.Y
				if dy < 0 && -dy > dx && -dy > -dx {
					ox = door.X - ow/2
					oy = door.Y
				} else if dy > 0 && dy > dx && dy > -dx {
					ox = door.X - ow/2
					oy = door.Y - oh + 1
				} else if dx < 0 {
					ox = door.X
					oy = door.Y - oh/2
				} else {
					ox = door.X - ow + 1
					oy = door.Y - oh/2
				}
				if ox < 1 {
					ox = 1
				}
				if oy < 1 {
					oy = 1
				}
				if ox+ow >= l.W {
					ox = l.W - ow - 1
				}
				if oy+oh >= l.H {
					oy = l.H - oh - 1
				}
			}
		} else {
			ow, oh = 7, 7
			ox = c.X - ow/2
			oy = c.Y - oh/2
			if ox < 1 {
				ox = 1
			}
			if oy < 1 {
				oy = 1
			}
			if ox+ow >= l.W {
				ox = l.W - ow - 1
			}
			if oy+oh >= l.H {
				oy = l.H - oh - 1
			}
			door = Pos{ox + ow/2, oy + oh - 1}
		}
		// Rebuild outer walls and ensure interior floor and door closed
		for yy := oy; yy < oy+oh; yy++ {
			for xx := ox; xx < ox+ow; xx++ {
				isPerim := xx == ox || xx == ox+ow-1 || yy == oy || yy == oy+oh-1
				p := Pos{xx, yy}
				if p == door {
					l.Tiles[yy][xx] = TileDoor
					if l.Doors == nil {
						l.Doors = make(map[Pos]bool)
					}
					l.Doors[p] = false
					continue
				}
				if isPerim {
					l.Tiles[yy][xx] = TileWall
					if l.Doors != nil {
						delete(l.Doors, p)
					}
				} else {
					l.Tiles[yy][xx] = TileFloor
				}
			}
		}
		// Ensure vault feature at interior center (distance >=2 from walls)
		center := Pos{ox + 1 + (ow-2)/2, oy + 1 + (oh-2)/2}
		l.Tiles[center.Y][center.X] = TileFloor
		for i := range l.Features {
			if l.Features[i].IsVault() && l.Features[i].Pos == c {
				l.Features[i].Pos = center
				break
			}
		}
	}
		// Final walkability check with deterministic retry
		if AssertLevelHasExit(l) {
			break
		}
		if attempt == 0 {
			debugGenLogf("WARN: floor %d exit not reachable after guarantee (biome %s up %v down %v)\n", floor, biome.ID, l.StairsUp, l.StairsDown)
		}
		dumpLevelGeometry(l)
		if attempt == 49 {
			panic(fmt.Sprintf("GenerateWithBiome floor %d failed to guarantee exit after 50 attempts (up %v down %v) biome %s", floor, l.StairsUp, l.StairsDown, biome.ID))
		}
	}
}




// ---------------------------------------------------------------------------
// Ambience ticker
// ---------------------------------------------------------------------------

// MaybeTickAmbience checks Game.Turn and emits a slate-blue ambience log line
// every 30-60 turns. Call from EndPlayerTurn/Update. Uses Game.RNG and
// NextAmbienceTurn.
func (g *Game) MaybeTickAmbience() {
	if g == nil || g.RNG == nil {
		return
	}
	if g.NextAmbienceTurn == 0 {
		g.NextAmbienceTurn = g.Turn + 30 + g.RNG.IntN(31)
		return
	}
	if g.Turn < g.NextAmbienceTurn {
		return
	}
	lvl := g.CurLevel()
	var biome *Biome
	if lvl != nil && lvl.BiomeID != "" {
		biomes := LoadBiomes()
		for _, b := range biomes {
			if b.ID == lvl.BiomeID {
				bb := b
				biome = &bb
				break
			}
		}
	}
	if biome == nil {
		biome = GetBiomeForFloor(g.Floor, g.RNG)
	}
	if biome == nil || len(biome.Ambience) == 0 {
		g.NextAmbienceTurn = g.Turn + 30 + g.RNG.IntN(31)
		return
	}
	line := biome.Ambience[g.RNG.IntN(len(biome.Ambience))]
	// Slate-blue ambience is rendered via log; text carries meaning first.
	g.Logf("%s", line)
	g.NextAmbienceTurn = g.Turn + 30 + g.RNG.IntN(31)
}

// litterStepAmbience returns a short slate ambience line for stepping onto
// a passable litter tile. kind is the litter kind, biomeID provides
// biome-specific flavour. Empty string means no line.
func litterStepAmbience(kind, biomeID string) string {
	switch kind {
	case "dust":
		return "Dust puffs underfoot."
	case "rubble":
		// Keep distinct but still dusty; acceptance expects a dust-like line for rubble in some biomes.
		if biomeID == "cinder" {
			return "Cinder crunches underfoot."
		}
		return "Rubble shifts underfoot."
	case "puddle":
		if biomeID == "cinder" {
			return "Warm water ripples underfoot."
		}
		if biomeID == "fungal" || biomeID == "jungle" {
			return "Water ripples through moss."
		}
		return "Water ripples underfoot."
	case "moss":
		return "Moss squelches softly."
	case "slime":
		return "Slime squelches underfoot."
	case "ash":
		return "Ash crunches."
	case "bone_dust":
		// Alternate phrasing requested: "Bones clatter."
		if biomeID == "ossuary" {
			return "Bones clatter."
		}
		return "Bone dust puffs underfoot."
	default:
		return ""
	}
}
// ---------------------------------------------------------------------------
// Biome entry feels
// ---------------------------------------------------------------------------

var (
	entryFeelMu  sync.Mutex
	entryFeelRNG *rand.Rand
)

func getEntryFeelRNG() *rand.Rand {
	entryFeelMu.Lock()
	defer entryFeelMu.Unlock()
	if entryFeelRNG == nil {
		entryFeelRNG = rand.New(rand.NewPCG(0x9e3779b97f4a7c15, 0x6a09e667f3bcc908))
	}
	return entryFeelRNG
}

// BiomeEntryFeel returns one of 2-3 evocative entry variants for biome.
// The returned string always contains biome.Name for acceptance checks.
func BiomeEntryFeel(b *Biome) string {
	if b == nil {
		return ""
	}
	var variants []string
	switch b.ID {
	case "crypt":
		variants = []string{
			fmt.Sprintf("You enter the %s — cold drafts curl through low arches.", b.Name),
			fmt.Sprintf("You descend into the %s — the air is still and chill, scented with old myrrh.", b.Name),
			fmt.Sprintf("You step into the %s — vaulted dark presses close, footsteps echoing too long.", b.Name),
		}
	case "ossuary":
		variants = []string{
			fmt.Sprintf("You enter the %s — chalky air catches in your throat.", b.Name),
			fmt.Sprintf("You descend into the %s — bones whisper as dust shifts.", b.Name),
			fmt.Sprintf("You step into the %s — hollow clatter rolls from stacked skulls.", b.Name),
		}
	case "fungal":
		variants = []string{
			fmt.Sprintf("You descend into the %s — the air is thick and sour.", b.Name),
			fmt.Sprintf("You enter the %s — spores drift like pale snow.", b.Name),
			fmt.Sprintf("You step into the %s — mushroom caps pulse with faint, humid light.", b.Name),
		}
	case "jungle":
		variants = []string{
			fmt.Sprintf("You enter the %s — humid air beads on cold stone.", b.Name),
			fmt.Sprintf("You descend into the %s — vines tighten overhead with a soft creak.", b.Name),
			fmt.Sprintf("You push into the %s — roots have cracked the temple walls below.", b.Name),
		}
	case "cinder":
		variants = []string{
			fmt.Sprintf("You enter the %s — ash drifts on heat that has no source.", b.Name),
			fmt.Sprintf("You descend into the %s — stone sweats with old, trapped heat.", b.Name),
			fmt.Sprintf("You step into the %s — embers blink in the dark like tired eyes.", b.Name),
		}
	default:
		variants = []string{
			fmt.Sprintf("You enter the %s — the air shifts around you.", b.Name),
			fmt.Sprintf("You descend into the %s — shadows deepen.", b.Name),
		}
	}
	// Use package RNG for variety; lock ordering: copy RNG pointer under lock then pick.
	rng := getEntryFeelRNG()
	entryFeelMu.Lock()
	idx := rng.IntN(len(variants))
	entryFeelMu.Unlock()
	return variants[idx]
}

// biomeForCurrentFloor resolves biome for g.Floor via Level.BiomeID or GetBiomeForFloor.
func (g *Game) biomeForCurrentFloor() *Biome {
	if g == nil {
		return nil
	}
	lvl := g.CurLevel()
	if lvl != nil && lvl.BiomeID != "" {
		biomes := LoadBiomes()
		for _, b := range biomes {
			if b.ID == lvl.BiomeID {
				bb := b
				return &bb
			}
		}
	}
	return GetBiomeForFloor(g.Floor, g.RNG)
}

// logBiomeEntry logs an entry feel for the current floor's biome.
func (g *Game) logBiomeEntry() {
	b := g.biomeForCurrentFloor()
	if b == nil {
		return
	}
	feel := BiomeEntryFeel(b)
	if feel != "" {
		g.Logf("%s", feel)
	}
}

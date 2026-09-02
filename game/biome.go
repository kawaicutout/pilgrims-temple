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
			ID: "crypt", Name: "Crypt", EligibleLevelRange: [2]int{1, 3},
			ColorPalette: ColorPalette{Primary: "#6a7a7a", Secondary: "#5a6a6a", Accent: "#8a7a7a"}, Color: "#6a7a7a", Tint: "#6a7a7a",
			Litter:           LitterDef{Destructible: []string{"barrel", "crate", "urn"}, Passable: []string{"rubble", "dust", "puddle"}, Impassable: []string{"column", "altar", "sarcophagus"}},
			GenerationMethod: "rooms", SpecialEnemies: []string{"rat", "kobold"},
			Ambience: []string{"Cold drafts curl through the crypt aisles.", "Distant stone lids grind against their sarcophagi.", "A faint incense of old myrrh hangs in the air.", "Footsteps echo too long in the vaulted dark.", "Candle smoke ghosts along the low arches.", "Water drips somewhere beyond the pillars, patient and cold.", "An unlit censer sways faintly, though no hand touched it.", "Mortar dust sifts from the vault above, catching dim light."},
		},
		{
			ID: "ossuary", Name: "Ossuary", EligibleLevelRange: [2]int{2, 4},
			ColorPalette: ColorPalette{Primary: "#7a5a3a", Secondary: "#6a4a32", Accent: "#9a7a5a"}, Color: "#7a5a3a", Tint: "#7a5a3a",
			Litter:           LitterDef{Destructible: []string{"bone_pile", "crate", "barrel"}, Passable: []string{"rubble", "bone_dust", "puddle"}, Impassable: []string{"bone_column", "pit", "rubble_wall"}},
			GenerationMethod: "rooms", SpecialEnemies: []string{"kobold", "orc"},
			Ambience: []string{"Bones whisper as dust shifts across the ossuary.", "A hollow clatter rolls from stacked skulls.", "Chalky air catches in your throat.", "Shadows pool between leaning bone-columns.", "Something small scuttles among the remains.", "A femur settles with a soft, tired crack.", "Dry air tastes of chalk and old marrow.", "Pale skulls watch with empty, patient sockets."},
		},
		{
			ID: "fungal", Name: "Fungal Grove", EligibleLevelRange: [2]int{3, 5},
			ColorPalette: ColorPalette{Primary: "#4a6a4a", Secondary: "#2e4a35", Accent: "#5a8a5a"}, Color: "#4a6a4a", Tint: "#4a6a4a",
			Litter:           LitterDef{Destructible: []string{"mushroom_cap", "spore_pod", "crate"}, Passable: []string{"moss", "slime", "rubble"}, Impassable: []string{"fungal_column", "pit", "thicket"}},
			GenerationMethod: "cavern", SpecialEnemies: []string{"spore_mother", "rat"},
			Ambience: []string{"Spores drift like pale snow through fungal gloom.", "Mushroom caps pulse with faint light.", "The air is thick, sweet, and slightly sour.", "Soft caps sigh as you brush past.", "Mycelial threads hum beneath the floor.", "A damp, earthy perfume clings to every breath.", "Distant caps release a soft puff of glowing dust.", "The floor gives slightly, spongy with hidden growth."},
		},
		{
			ID: "jungle", Name: "Jungle Overgrowth", EligibleLevelRange: [2]int{4, 6},
			ColorPalette: ColorPalette{Primary: "#3d5a3a", Secondary: "#2a3d2f", Accent: "#4a7a4a"}, Color: "#3d5a3a", Tint: "#3d5a3a",
			Litter:           LitterDef{Destructible: []string{"vine_cluster", "crate", "barrel"}, Passable: []string{"moss", "rubble", "puddle"}, Impassable: []string{"vine_wall", "pit", "column"}},
			GenerationMethod: "cavern", SpecialEnemies: []string{"vine_horror", "kobold"},
			Ambience: []string{"Vines tighten overhead with a soft creak.", "Humid air beads on cold stone.", "Leaves rustle where no wind should reach.", "A distant vine snaps taut, then stills.", "Roots have cracked the temple walls below.", "Warm rot and green perfume hang heavy in the air.", "Something unseen pushes through tangled fronds.", "Moss muffles your steps like a living carpet."},
		},
		{
			ID: "cinder", Name: "Cinder Chapel", EligibleLevelRange: [2]int{5, 8},
			ColorPalette: ColorPalette{Primary: "#6a3d2f", Secondary: "#4a2e2a", Accent: "#8a5a45"}, Color: "#6a3d2f", Tint: "#6a3d2f",
			Litter:           LitterDef{Destructible: []string{"ash_barrel", "crate", "cinder_block"}, Passable: []string{"ash", "rubble", "puddle"}, Impassable: []string{"cinder_column", "lava_pit", "rubble_wall"}},
			GenerationMethod: "cavern", SpecialEnemies: []string{"troll", "orc"},
			Ambience: []string{"Ash drifts on heat that has no source.", "Cinder clicks underfoot, cooling and cracking.", "A low draft carries the tang of soot.", "Embers blink in the dark like tired eyes.", "Stone sweats with old, trapped heat.", "Cracked tiles tick as they cool in the dark.", "A faint, acrid haze stings the eyes.", "Distant stone sighs as heat shifts through old flues."},
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

// GetBiomeForFloor returns an eligible biome for floor (0-indexed). Uses floor-based
// deterministic selection among eligible to satisfy acceptance.
func GetBiomeForFloor(floor int) *Biome {
	biomes := LoadBiomes()
	if len(biomes) == 0 {
		fb := fallbackBiomes()
		biomes = fb
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
	// Deterministic pick among eligible: seeded by floor.
	r := rand.New(rand.NewPCG(uint64(floor+1)*0x9e3779b97f4a7c15, 0x6a09e667f3bcc908))
	idx := r.IntN(len(eligible))
	b := eligible[idx]
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
// stairsReachableViaBFS checks StairsUp -> StairsDown over walkable tiles plus litter blocks.
func stairsReachableViaBFS(lvl *Level) bool {
	if lvl == nil {
		return false
	}
	start := lvl.StairsUp
	goal := lvl.StairsDown
	// Floor 0 may have stairs up at same as start? Still need path to down.
	if !lvl.InBounds(start) || !lvl.InBounds(goal) {
		return false
	}
	// BFS
	visited := make(map[Pos]bool, lvl.W*lvl.H)
	queue := []Pos{start}
	visited[start] = true
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == goal {
			return true
		}
		for _, d := range AllDirs {
			np := Pos{cur.X + d.DX, cur.Y + d.DY}
			if !lvl.InBounds(np) {
				continue
			}
			if visited[np] {
				continue
			}
			// Walkable check includes litter via lvl.Walkable
			if !lvl.Walkable(np) {
				continue
			}
			visited[np] = true
			queue = append(queue, np)
		}
	}
	// Also allow case where stairs are same tile (single room) -> visited already
	return visited[goal]
}


// AssertLevelHasExit checks that a level has a reachable exit stairs.
// For non-final floors (Floor != Tuning.Floors-1), both StairsUp and StairsDown
// must be InBounds, on TileStairsUp/TileStairsDown, Walkable (not blocked by
// litter/closed door), not coincident, and BFS-reachable via Walkable tiles.
// For the final floor (relic floor), StairsUp must be present and Walkable and
// reachable to the relic (which sits at StairsDown); if StairsDown is missing,
// StairsUp alone suffices but if present it must also be walkable and reachable.
func AssertLevelHasExit(lvl *Level) bool {
	if lvl == nil {
		return false
	}
	if !lvl.InBounds(lvl.StairsUp) {
		return false
	}
	if lvl.At(lvl.StairsUp) != TileStairsUp {
		return false
	}
	if !lvl.Walkable(lvl.StairsUp) {
		return false
	}
	for _, f := range lvl.Features {
		if f.Pos == lvl.StairsUp {
			return false
		}
	}
	// Determine final floor via tuning; fallback to 8.
	floors := 8
	if t, err := LoadTuning(); err == nil && t.Floors > 0 {
		floors = t.Floors
	}
	isFinal := lvl.Floor == floors-1
	if isFinal {
		if !lvl.InBounds(lvl.StairsDown) {
			// No stairs down required on relic floor; stairs up alone is enough.
			return true
		}
		if lvl.At(lvl.StairsDown) != TileStairsDown {
			return false
		}
		if !lvl.Walkable(lvl.StairsDown) {
			return false
		}
		for _, f := range lvl.Features {
			if f.Pos == lvl.StairsDown {
				return false
			}
		}
		return stairsReachableViaBFS(lvl)
	}
	if !lvl.InBounds(lvl.StairsDown) {
		return false
	}
	if lvl.At(lvl.StairsDown) != TileStairsDown {
		return false
	}
	if !lvl.Walkable(lvl.StairsDown) {
		return false
	}
	if lvl.StairsUp == lvl.StairsDown {
		return false
	}
	for _, f := range lvl.Features {
		if f.Pos == lvl.StairsDown {
			return false
		}
	}
	return stairsReachableViaBFS(lvl)
}

// ensureStairsTiles fixes missing or blocked stairs by relocating them to
// walkable TileFloor positions that are not blocked by litter/features/doors.
func ensureStairsTiles(lvl *Level, rng *rand.Rand) {
	if lvl == nil {
		return
	}
	// Helper to clean litter/door at p for stairs.
	cleanPos := func(p Pos) {
		for i := 0; i < len(lvl.Litter); {
			if lvl.Litter[i].Pos == p {
				lvl.Litter = append(lvl.Litter[:i], lvl.Litter[i+1:]...)
			} else {
				i++
			}
		}
		if lvl.Doors != nil {
			delete(lvl.Doors, p)
		}
	}
	needsUp := !lvl.InBounds(lvl.StairsUp) || lvl.At(lvl.StairsUp) != TileStairsUp || !lvl.Walkable(lvl.StairsUp)
	if !needsUp {
		for _, f := range lvl.Features {
			if f.Pos == lvl.StairsUp {
				needsUp = true
				break
			}
		}
	}
	needsDown := false
	floors := 8
	if t, err := LoadTuning(); err == nil && t.Floors > 0 {
		floors = t.Floors
	}
	isFinal := lvl.Floor == floors-1
	if !isFinal {
		needsDown = !lvl.InBounds(lvl.StairsDown) || lvl.At(lvl.StairsDown) != TileStairsDown || !lvl.Walkable(lvl.StairsDown) || lvl.StairsUp == lvl.StairsDown
		if !needsDown {
			for _, f := range lvl.Features {
				if f.Pos == lvl.StairsDown {
					needsDown = true
					break
				}
			}
		}
	} else {
		if lvl.InBounds(lvl.StairsDown) {
			if lvl.At(lvl.StairsDown) != TileStairsDown || !lvl.Walkable(lvl.StairsDown) {
				needsDown = true
			}
			for _, f := range lvl.Features {
				if f.Pos == lvl.StairsDown {
					needsDown = true
					break
				}
			}
		}
	}
	if !needsUp && !needsDown {
		// Still ensure no blocking litter directly on stairs.
		cleanPos(lvl.StairsUp)
		if lvl.InBounds(lvl.StairsDown) {
			cleanPos(lvl.StairsDown)
		}
		// Re-assert tiles.
		if lvl.InBounds(lvl.StairsUp) {
			lvl.Tiles[lvl.StairsUp.Y][lvl.StairsUp.X] = TileStairsUp
		}
		if lvl.InBounds(lvl.StairsDown) {
			lvl.Tiles[lvl.StairsDown.Y][lvl.StairsDown.X] = TileStairsDown
		}
		return
	}
	// Collect walkable floor candidates not blocked.
	var candidates []Pos
	for y := 0; y < lvl.H; y++ {
		for x := 0; x < lvl.W; x++ {
			p := Pos{x, y}
			if lvl.At(p) != TileFloor {
				continue
			}
			if !lvl.Walkable(p) {
				continue
			}
			blocked := false
			for _, f := range lvl.Features {
				if f.Pos == p {
					blocked = true
					break
				}
			}
			if blocked {
				continue
			}
			if p == lvl.StairsUp || p == lvl.StairsDown {
				continue
			}
			candidates = append(candidates, p)
		}
	}
	// Also include current stairs positions if they are floor-like fallback.
	if len(candidates) == 0 {
		for y := 0; y < lvl.H; y++ {
			for x := 0; x < lvl.W; x++ {
				p := Pos{x, y}
				if lvl.At(p) == TileFloor {
					candidates = append(candidates, p)
				}
			}
		}
	}
	if len(candidates) == 0 {
		return
	}
	// Shuffle with rng if available.
	if rng != nil {
		for i := len(candidates) - 1; i > 0; i-- {
			j := rng.IntN(i + 1)
			candidates[i], candidates[j] = candidates[j], candidates[i]
		}
	}
	if needsUp {
		// Pick far from down if down valid.
		pick := candidates[0]
		if lvl.InBounds(lvl.StairsDown) && len(candidates) > 1 {
			best := pick
			bestDist := abs(best.X-lvl.StairsDown.X) + abs(best.Y-lvl.StairsDown.Y)
			for _, c := range candidates[1:] {
				d := abs(c.X-lvl.StairsDown.X) + abs(c.Y-lvl.StairsDown.Y)
				if d > bestDist {
					bestDist = d
					best = c
				}
			}
			pick = best
		}
		cleanPos(pick)
		// Clear previous stairs tile to floor if it was stairs.
		if lvl.InBounds(lvl.StairsUp) && lvl.At(lvl.StairsUp) == TileStairsUp {
			lvl.Tiles[lvl.StairsUp.Y][lvl.StairsUp.X] = TileFloor
		}
		lvl.StairsUp = pick
		lvl.Tiles[pick.Y][pick.X] = TileStairsUp
	}
	if needsDown {
		// Refresh candidates excluding new up.
		var filtered []Pos
		for _, c := range candidates {
			if c == lvl.StairsUp {
				continue
			}
			filtered = append(filtered, c)
		}
		if len(filtered) > 0 {
			candidates = filtered
		}
		pick := candidates[0]
		best := pick
		bestDist := abs(best.X-lvl.StairsUp.X) + abs(best.Y-lvl.StairsUp.Y)
		for _, c := range candidates[1:] {
			d := abs(c.X-lvl.StairsUp.X) + abs(c.Y-lvl.StairsUp.Y)
			if d > bestDist {
				bestDist = d
				best = c
			}
		}
		pick = best
		cleanPos(pick)
		if lvl.InBounds(lvl.StairsDown) && lvl.At(lvl.StairsDown) == TileStairsDown {
			lvl.Tiles[lvl.StairsDown.Y][lvl.StairsDown.X] = TileFloor
		}
		lvl.StairsDown = pick
		lvl.Tiles[pick.Y][pick.X] = TileStairsDown
	}
	// Final clean after relocation.
	cleanPos(lvl.StairsUp)
	if lvl.InBounds(lvl.StairsDown) {
		cleanPos(lvl.StairsDown)
	}
	lvl.Tiles[lvl.StairsUp.Y][lvl.StairsUp.X] = TileStairsUp
	if lvl.InBounds(lvl.StairsDown) {
		lvl.Tiles[lvl.StairsDown.Y][lvl.StairsDown.X] = TileStairsDown
	}
}

// carveEmergencyCorridor carves an L-shaped corridor between stairs, removes
// blocking litter on the path, and opens any doors encountered.
func carveEmergencyCorridor(lvl *Level, rng *rand.Rand, horizFirst bool) {
	if lvl == nil || !lvl.InBounds(lvl.StairsUp) || !lvl.InBounds(lvl.StairsDown) {
		return
	}
	ax, ay := lvl.StairsUp.X, lvl.StairsUp.Y
	bx, by := lvl.StairsDown.X, lvl.StairsDown.Y
	var path []Pos
	if horizFirst {
		for x := min(ax, bx); x <= max(ax, bx); x++ {
			path = append(path, Pos{x, ay})
		}
		for y := min(ay, by); y <= max(ay, by); y++ {
			path = append(path, Pos{bx, y})
		}
	} else {
		for y := min(ay, by); y <= max(ay, by); y++ {
			path = append(path, Pos{ax, y})
		}
		for x := min(ax, bx); x <= max(ax, bx); x++ {
			path = append(path, Pos{x, by})
		}
	}
	seen := make(map[Pos]bool, len(path))
	var uniq []Pos
	for _, p := range path {
		if !seen[p] {
			seen[p] = true
			uniq = append(uniq, p)
		}
	}
	for _, p := range uniq {
		if !lvl.InBounds(p) {
			continue
		}
		if p == lvl.StairsUp || p == lvl.StairsDown {
			// Ensure stairs remain.
			if p == lvl.StairsUp {
				lvl.Tiles[p.Y][p.X] = TileStairsUp
			} else {
				lvl.Tiles[p.Y][p.X] = TileStairsDown
			}
			// Remove blocking litter at stairs.
			for i := 0; i < len(lvl.Litter); {
				if lvl.Litter[i].Pos == p && lvl.Litter[i].BlocksMovement {
					lvl.Litter = append(lvl.Litter[:i], lvl.Litter[i+1:]...)
				} else {
					i++
				}
			}
			if lvl.Doors != nil {
				delete(lvl.Doors, p)
			}
			continue
		}
		// Remove blocking litter on corridor.
		for i := 0; i < len(lvl.Litter); {
			if lvl.Litter[i].Pos == p && lvl.Litter[i].BlocksMovement {
				lvl.Litter = append(lvl.Litter[:i], lvl.Litter[i+1:]...)
			} else {
				i++
			}
		}
		// Open door if present.
		if lvl.At(p) == TileDoor {
			lvl.SetDoorOpen(p, true)
			continue
		}
		// Carve floor (overwrites wall).
		lvl.Tiles[p.Y][p.X] = TileFloor
		if lvl.Doors != nil {
			delete(lvl.Doors, p)
		}
	}
	// Ensure doors on corridor that were doors remain open (already handled).
	// Re-assert stairs tiles.
	lvl.Tiles[lvl.StairsUp.Y][lvl.StairsUp.X] = TileStairsUp
	lvl.Tiles[lvl.StairsDown.Y][lvl.StairsDown.X] = TileStairsDown
}

// ensureExitGuarantee loops up to 5 times carving emergency corridors until
// AssertLevelHasExit passes; each iteration alternates orientation and fixes
// stairs tiles. Doors on the carved path are opened.
func ensureExitGuarantee(lvl *Level, rng *rand.Rand) {
	if lvl == nil {
		return
	}
	if rng == nil {
		rng = rand.New(rand.NewPCG(0, 0))
	}
	for i := range 5 {
		if AssertLevelHasExit(lvl) {
			return
		}
		ensureStairsTiles(lvl, rng)
		carveEmergencyCorridor(lvl, rng, i%2 == 0)
		if stairsReachableViaBFS(lvl) {
			if AssertLevelHasExit(lvl) {
				return
			}
		}
	}
}



// spawnLitter places litter without blocking StairsUp->StairsDown BFS.
func spawnLitter(lvl *Level, rng *rand.Rand, biome *Biome) {
	if lvl == nil || rng == nil || biome == nil {
		return
	}
	// Gather candidates walkable not on stairs/enemy/feature.
	enemySet := make(map[Pos]bool, len(lvl.Enemies))
	for _, e := range lvl.Enemies {
		if e != nil {
			enemySet[e.Pos] = true
		}
	}
	featureSet := make(map[Pos]bool, len(lvl.Features))
	for _, f := range lvl.Features {
		featureSet[f.Pos] = true
	}
	var candidates []Pos
	for y := range lvl.H {
		for x := range lvl.W {
			p := Pos{x, y}
			if p == lvl.StairsUp || p == lvl.StairsDown {
				continue
			}
			if enemySet[p] || featureSet[p] {
				continue
			}
			// Must be walkable tile before litter (floor/stairs)
			if lvl.At(p) != TileFloor {
				continue
			}
			// Also must be currently walkable with existing litter
			if !lvl.Walkable(p) {
				continue
			}
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return
	}
	// Shuffle candidates.
	for i := len(candidates) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}
	// Decide count: 6-14 plus floor scaling.
	count := 6 + rng.IntN(9) + lvl.floorForScaling()/2
	if count > len(candidates) {
		count = len(candidates)
	}
	placed := 0
	for _, p := range candidates {
		if placed >= count {
			break
		}
		// Pick category weighted: passable 35%, destructible 30%, impassable 35%
		roll := rng.Float64()
		var category, kind string
		switch {
		case roll < 0.35:
			category = "passable"
			if len(biome.Litter.Passable) == 0 {
				continue
			}
			kind = biome.Litter.Passable[rng.IntN(len(biome.Litter.Passable))]
		case roll < 0.65:
			category = "destructible"
			if len(biome.Litter.Destructible) == 0 {
				continue
			}
			kind = biome.Litter.Destructible[rng.IntN(len(biome.Litter.Destructible))]
		default:
			category = "impassable"
			if len(biome.Litter.Impassable) == 0 {
				continue
			}
			kind = biome.Litter.Impassable[rng.IntN(len(biome.Litter.Impassable))]
		}
		obj := newLitterObj(p, kind, category)
		// Tentatively place and test BFS if this blocks.
		lvl.Litter = append(lvl.Litter, obj)
		if obj.BlocksMovement {
			if !stairsReachableViaBFS(lvl) {
				// Revert: remove last.
				lvl.Litter = lvl.Litter[:len(lvl.Litter)-1]
				continue
			}
		}
		placed++
	}
}

// floorForScaling returns floor index from BiomeID lookup via level's BiomeID.
// Stored separately to avoid adding floor param to Level.
func (l *Level) floorForScaling() int {
	// Infer from BiomeID eligible range midpoint? Better store Floor field.
	// Added Level.Floor field; fallback 0.
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
		biome = GetBiomeForFloor(floor)
	}
	l.BiomeID = biome.ID
	l.Floor = floor
	// Clear any prior state (in case of re-generation).
	l.Enemies = nil
	l.Features = nil
	l.Litter = nil
	l.Items = nil
	// Branch generation.
	switch biome.GenerationMethod {
	case "cavern":
		l.generateCavern(rng, floor)
	default:
		l.generateRooms(rng, floor)
	}
	// Ensure stairs are placed (both generators do, but double-check).
	if !l.InBounds(l.StairsUp) || !l.InBounds(l.StairsDown) {
		// Fallback: place stairs on walkable tiles.
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
		// Carve straight L corridor ignoring litter (litter not yet placed).
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
		// Re-assert stairs tiles.
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
				// Vault door missing — rebuild 7x7 vault around center to ensure walls/door exist.
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
			// Find outer by scanning from center to wall/door
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
				// Fallback to 7x7 around door
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
		fmt.Printf("WARN: floor %d exit not reachable after guarantee (biome %s up %v down %v)\n", floor, biome.ID, l.StairsUp, l.StairsDown)
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
	// Debug helper
	_ = fmt.Sprintf("biome %s floor %d", biome.ID, floor)
}

// isWallForDoor reports whether pos is a wall or out of bounds (treated as wall for door width checks).
func isWallForDoor(l *Level, p Pos) bool {
	if !l.InBounds(p) {
		return true
	}
	return l.At(p) == TileWall
}

// isFloorLikeForDoor reports whether pos is floor-like (floor or stairs) for corridor width checks.
func isFloorLikeForDoor(l *Level, p Pos) bool {
	if !l.InBounds(p) {
		return false
	}
	t := l.At(p)
	return t == TileFloor || t == TileStairsUp || t == TileStairsDown
}

// isSingleWideDoor reports whether a door at p is in a single-wide corridor segment.
// It requires opposite walls (N-S or E-W) at p and that the adjacent corridor segment
// is also 1-wide (orthogonal walls on the corridor side). This prevents doors in
// double-wide (2-tile) hallways where the second lane would be floor instead of wall.
func isSingleWideDoor(l *Level, p Pos) bool {
	x, y := p.X, p.Y
	nWall := isWallForDoor(l, Pos{x, y - 1})
	sWall := isWallForDoor(l, Pos{x, y + 1})
	wWall := isWallForDoor(l, Pos{x - 1, y})
	eWall := isWallForDoor(l, Pos{x + 1, y})
	hasNS := nWall && sWall
	hasWE := wWall && eWall
	if !hasNS && !hasWE {
		return false
	}
	if hasNS && !hasWE {
		eastFloor := isFloorLikeForDoor(l, Pos{x + 1, y})
		westFloor := isFloorLikeForDoor(l, Pos{x - 1, y})
		if eastFloor || westFloor {
			narrowEast := isWallForDoor(l, Pos{x + 1, y - 1}) && isWallForDoor(l, Pos{x + 1, y + 1})
			narrowWest := isWallForDoor(l, Pos{x - 1, y - 1}) && isWallForDoor(l, Pos{x - 1, y + 1})
			eastOK := eastFloor && narrowEast
			westOK := westFloor && narrowWest
			if !eastOK && !westOK {
				return false
			}
		}
		return true
	}
	if hasWE && !hasNS {
		northFloor := isFloorLikeForDoor(l, Pos{x, y - 1})
		southFloor := isFloorLikeForDoor(l, Pos{x, y + 1})
		if northFloor || southFloor {
			narrowNorth := isWallForDoor(l, Pos{x - 1, y - 1}) && isWallForDoor(l, Pos{x + 1, y - 1})
			narrowSouth := isWallForDoor(l, Pos{x - 1, y + 1}) && isWallForDoor(l, Pos{x + 1, y + 1})
			northOK := northFloor && narrowNorth
			southOK := southFloor && narrowSouth
			if !northOK && !southOK {
				return false
			}
		}
		return true
	}
	eastFloor := isFloorLikeForDoor(l, Pos{x + 1, y})
	westFloor := isFloorLikeForDoor(l, Pos{x - 1, y})
	northFloor := isFloorLikeForDoor(l, Pos{x, y - 1})
	southFloor := isFloorLikeForDoor(l, Pos{x, y + 1})
	horizOK := true
	vertOK := true
	if eastFloor || westFloor {
		narrowEast := isWallForDoor(l, Pos{x + 1, y - 1}) && isWallForDoor(l, Pos{x + 1, y + 1})
		narrowWest := isWallForDoor(l, Pos{x - 1, y - 1}) && isWallForDoor(l, Pos{x - 1, y + 1})
		if !(eastFloor && narrowEast) && !(westFloor && narrowWest) {
			horizOK = false
		}
	}
	if northFloor || southFloor {
		narrowNorth := isWallForDoor(l, Pos{x - 1, y - 1}) && isWallForDoor(l, Pos{x + 1, y - 1})
		narrowSouth := isWallForDoor(l, Pos{x - 1, y + 1}) && isWallForDoor(l, Pos{x + 1, y + 1})
		if !(northFloor && narrowNorth) && !(southFloor && narrowSouth) {
			vertOK = false
		}
	}
	return horizOK && vertOK
}

// generateRooms is extracted rooms+corridors generator (logic from original Generate).
func (l *Level) generateRooms(rng *rand.Rand, floor int) {
	type rect struct{ x, y, w, h int }
	var rooms []rect
	var vaultOuters []rect
	var vaultDoors []Pos
	attempts := 50
	for range attempts {
		w := 5 + rng.IntN(7) // 5-11
		h := 4 + rng.IntN(5) // 4-8
		x := 1 + rng.IntN(l.W-w-2)
		y := 1 + rng.IntN(l.H-h-2)
		r := rect{x, y, w, h}
		overlap := false
		for _, o := range rooms {
			if r.x < o.x+o.w+1 && r.x+r.w+1 > o.x && r.y < o.y+o.h+1 && r.y+r.h+1 > o.y {
				overlap = true
				break
			}
		}
		if overlap {
			continue
		}
		for yy := r.y; yy < r.y+r.h; yy++ {
			for xx := r.x; xx < r.x+r.w; xx++ {
				l.Tiles[yy][xx] = TileFloor
			}
		}
		rooms = append(rooms, r)
		if len(rooms) >= 10 {
			break
		}
	}
	if len(rooms) == 0 {
		for y := 1; y < l.H-1; y++ {
			for x := 1; x < l.W-1; x++ {
				l.Tiles[y][x] = TileFloor
			}
		}
		rooms = append(rooms, rect{1, 1, l.W - 2, l.H - 2})
	}
	for i := 1; i < len(rooms); i++ {
		a := rooms[i-1]
		b := rooms[i]
		ax := a.x + a.w/2
		ay := a.y + a.h/2
		bx := b.x + b.w/2
		by := b.y + b.h/2
		if rng.IntN(2) == 0 {
			for x := min(ax, bx); x <= max(ax, bx); x++ {
				l.Tiles[ay][x] = TileFloor
			}
			for y := min(ay, by); y <= max(ay, by); y++ {
				l.Tiles[y][bx] = TileFloor
			}
		} else {
			for y := min(ay, by); y <= max(ay, by); y++ {
				l.Tiles[y][ax] = TileFloor
			}
			for x := min(ax, bx); x <= max(ax, bx); x++ {
				l.Tiles[by][x] = TileFloor
			}
		}
	}
	// --- Special rooms: vault (>=5x5) and merchant (3x3..4x4) with locked doors ---
	if l.Doors == nil {
		l.Doors = make(map[Pos]bool)
	}
	overlaps := func(r rect) bool {
		for _, o := range rooms {
			if r.x < o.x+o.w+1 && r.x+r.w+1 > o.x && r.y < o.y+o.h+1 && r.y+r.h+1 > o.y {
				return true
			}
		}
		return false
	}
	trySpecialRoom := func(w, h int) (rect, Pos, bool) {
		for range 40 {
			x := 1 + rng.IntN(l.W-w-2)
			y := 1 + rng.IntN(l.H-h-2)
			r := rect{x, y, w, h}
			if overlaps(r) {
				continue
			}
			hitStairs := false
			for yy := r.y - 1; yy <= r.y+r.h; yy++ {
				for xx := r.x - 1; xx <= r.x+r.w; xx++ {
					pp := Pos{xx, yy}
					if pp == l.StairsUp || pp == l.StairsDown {
						hitStairs = true
						break
					}
				}
				if hitStairs {
					break
				}
			}
			if hitStairs {
				continue
			}
			for yy := r.y; yy < r.y+r.h; yy++ {
				for xx := r.x; xx < r.x+r.w; xx++ {
					l.Tiles[yy][xx] = TileFloor
				}
			}
			side := rng.IntN(4)
			var door Pos
			switch side {
			case 0:
				door = Pos{r.x + rng.IntN(r.w), r.y - 1}
				if r.w > 2 && door.X == r.x {
					door.X++
				}
				if r.w > 2 && door.X == r.x+r.w-1 {
					door.X--
				}
			case 1:
				door = Pos{r.x + rng.IntN(r.w), r.y + r.h}
			case 2:
				door = Pos{r.x - 1, r.y + rng.IntN(r.h)}
			case 3:
				door = Pos{r.x + r.w, r.y + rng.IntN(r.h)}
			}
			if !l.InBounds(door) {
				for yy := r.y; yy < r.y+r.h; yy++ {
					for xx := r.x; xx < r.x+r.w; xx++ {
						l.Tiles[yy][xx] = TileWall
					}
				}
				continue
			}
			l.Tiles[door.Y][door.X] = TileDoor
			l.Doors[door] = false
			var outside Pos
			switch side {
			case 0:
				outside = Pos{door.X, door.Y - 1}
			case 1:
				outside = Pos{door.X, door.Y + 1}
			case 2:
				outside = Pos{door.X - 1, door.Y}
			case 3:
				outside = Pos{door.X + 1, door.Y}
			}
			if l.InBounds(outside) && l.At(outside) == TileWall {
				l.Tiles[outside.Y][outside.X] = TileFloor
				for range 3 {
					nxt := Pos{outside.X, outside.Y}
					switch side {
					case 0:
						nxt.Y--
					case 1:
						nxt.Y++
					case 2:
						nxt.X--
					case 3:
						nxt.X++
					}
					if !l.InBounds(nxt) {
						break
					}
					if l.At(nxt) == TileFloor || l.At(nxt) == TileStairsDown || l.At(nxt) == TileStairsUp {
						break
					}
					if l.At(nxt) == TileWall {
						l.Tiles[nxt.Y][nxt.X] = TileFloor
					}
					outside = nxt
					found := false
					for _, d := range []Dir{DirN, DirS, DirE, DirW} {
						if l.InBounds(outside.Add(d)) && l.At(outside.Add(d)) == TileFloor {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
			}
			rooms = append(rooms, r)
			return r, door, true
		}
		return rect{}, Pos{}, false
	}
	vw := 5 + rng.IntN(3)
	vh := 5 + rng.IntN(3)
	vaultPlaced := false
	for attempt := 0; attempt < 80 && !vaultPlaced; attempt++ {
		ow := vw + 2
		oh := vh + 2
		maxOx := l.W - ow - 1
		maxOy := l.H - oh - 1
		if maxOx < 1 || maxOy < 1 {
			break
		}
		ox := 1 + rng.IntN(maxOx)
		oy := 1 + rng.IntN(maxOy)
		outer := rect{ox, oy, ow, oh}
		if overlaps(outer) {
			continue
		}
		hitStairs := false
		for yy := oy - 1; yy <= oy+oh; yy++ {
			for xx := ox - 1; xx <= ox+ow; xx++ {
				pp := Pos{xx, yy}
				if pp == l.StairsUp || pp == l.StairsDown {
					hitStairs = true
					break
				}
			}
			if hitStairs {
				break
			}
		}
		if hitStairs {
			continue
		}
		for yy := oy; yy < oy+oh; yy++ {
			for xx := ox; xx < ox+ow; xx++ {
				isPerim := xx == ox || xx == ox+ow-1 || yy == oy || yy == oy+oh-1
				if isPerim {
					l.Tiles[yy][xx] = TileWall
				} else {
					l.Tiles[yy][xx] = TileFloor
				}
			}
		}
		side := rng.IntN(4)
		var door Pos
		switch side {
		case 0:
			door = Pos{ox + ow/2, oy}
		case 1:
			door = Pos{ox + ow/2, oy + oh - 1}
		case 2:
			door = Pos{ox, oy + oh/2}
		case 3:
			door = Pos{ox + ow - 1, oy + oh/2}
		}
		l.Tiles[door.Y][door.X] = TileDoor
		l.Doors[door] = false
		var outside Pos
		var dir Dir
		switch side {
		case 0:
			outside = Pos{door.X, door.Y - 1}
			dir = DirN
		case 1:
			outside = Pos{door.X, door.Y + 1}
			dir = DirS
		case 2:
			outside = Pos{door.X - 1, door.Y}
			dir = DirW
		case 3:
			outside = Pos{door.X + 1, door.Y}
			dir = DirE
		}
		if l.InBounds(outside) {
			if l.At(outside) == TileWall {
				l.Tiles[outside.Y][outside.X] = TileFloor
			}
			cur := outside
			for range 5 {
				nxt := cur.Add(dir)
				if !l.InBounds(nxt) {
					break
				}
				if l.At(nxt) == TileFloor || l.At(nxt) == TileStairsDown || l.At(nxt) == TileStairsUp {
					break
				}
				if l.At(nxt) == TileWall {
					l.Tiles[nxt.Y][nxt.X] = TileFloor
				}
				cur = nxt
				found := false
				for _, d := range []Dir{DirN, DirS, DirE, DirW} {
					adj := cur.Add(d)
					if !l.InBounds(adj) {
						continue
					}
					if l.At(adj) != TileFloor {
						continue
					}
					if adj == door {
						continue
					}
					inside := adj.X > ox && adj.X < ox+ow-1 && adj.Y > oy && adj.Y < oy+oh-1
					if inside {
						continue
					}
					found = true
					break
				}
				if found {
					break
				}
			}
		}
		interior := rect{ox + 1, oy + 1, vw, vh}
		rooms = append(rooms, interior)
		center := Pos{interior.x + interior.w/2, interior.y + interior.h/2}
		l.Tiles[center.Y][center.X] = TileFloor
		l.Features = append(l.Features, Feature{Pos: center, Type: FeatureVault, Locked: true, Treasure: 25 + rng.IntN(56), Trapped: rng.Float64() < 0.2})
		vaultDoors = append(vaultDoors, door)
		vaultPlaced = true
	}
	if !vaultPlaced && len(rooms) > 0 {
		indices := make([]int, len(rooms))
		for i := range indices {
			indices[i] = i
		}
		for i := len(indices) - 1; i > 0; i-- {
			j := rng.IntN(i + 1)
			indices[i], indices[j] = indices[j], indices[i]
		}
		for _, idx := range indices {
			fvw := 5 + rng.IntN(3)
			fvh := 5 + rng.IntN(3)
			r := rooms[idx]
			cx := r.x + r.w/2
			cy := r.y + r.h/2
			vx := cx - fvw/2
			vy := cy - fvh/2
			if vx < 2 {
				vx = 2
			}
			if vy < 2 {
				vy = 2
			}
			if vx+fvw+2 >= l.W {
				vx = l.W - fvw - 3
			}
			if vy+fvh+2 >= l.H {
				vy = l.H - fvh - 3
			}
			ox := vx - 1
			oy := vy - 1
			ow := fvw + 2
			oh := fvh + 2
			if ox < 1 || oy < 1 || ox+ow >= l.W || oy+oh >= l.H {
				continue
			}
			outer := rect{ox, oy, ow, oh}
			overlapOther := false
			for j, o := range rooms {
				if j == idx {
					continue
				}
				if outer.x < o.x+o.w+1 && outer.x+outer.w+1 > o.x && outer.y < o.y+o.h+1 && outer.y+outer.h+1 > o.y {
					overlapOther = true
					break
				}
			}
			if overlapOther {
				continue
			}
			for yy := oy; yy < oy+oh; yy++ {
				for xx := ox; xx < ox+ow; xx++ {
					isPerim := xx == ox || xx == ox+ow-1 || yy == oy || yy == oy+oh-1
					if isPerim {
						l.Tiles[yy][xx] = TileWall
					} else {
						l.Tiles[yy][xx] = TileFloor
					}
				}
			}
			side := rng.IntN(4)
			var door Pos
			switch side {
			case 0:
				door = Pos{ox + ow/2, oy}
			case 1:
				door = Pos{ox + ow/2, oy + oh - 1}
			case 2:
				door = Pos{ox, oy + oh/2}
			case 3:
				door = Pos{ox + ow - 1, oy + oh/2}
			}
			l.Tiles[door.Y][door.X] = TileDoor
			if l.Doors == nil {
				l.Doors = make(map[Pos]bool)
			}
			l.Doors[door] = false
			var outside Pos
			switch side {
			case 0:
				outside = Pos{door.X, door.Y - 1}
			case 1:
				outside = Pos{door.X, door.Y + 1}
			case 2:
				outside = Pos{door.X - 1, door.Y}
			case 3:
				outside = Pos{door.X + 1, door.Y}
			}
			if l.InBounds(outside) && l.At(outside) == TileWall {
				l.Tiles[outside.Y][outside.X] = TileFloor
			}
			rooms[idx] = rect{vx, vy, fvw, fvh}
			center := Pos{vx + fvw/2, vy + fvh/2}
			l.Tiles[center.Y][center.X] = TileFloor
			l.Features = append(l.Features, Feature{Pos: center, Type: FeatureVault, Locked: true, Treasure: 25 + rng.IntN(56), Trapped: rng.Float64() < 0.2})
			vaultDoors = append(vaultDoors, door)
			vaultPlaced = true
			break
		}
	}
	if !vaultPlaced {
		vw, vh := 5, 5
		ox, oy := 5, 5
		ow, oh := 7, 7
		if ox+ow < l.W && oy+oh < l.H {
			for yy := oy; yy < oy+oh; yy++ {
				for xx := ox; xx < ox+ow; xx++ {
					isPerim := xx == ox || xx == ox+ow-1 || yy == oy || yy == oy+oh-1
					if isPerim {
						l.Tiles[yy][xx] = TileWall
					} else {
						l.Tiles[yy][xx] = TileFloor
					}
				}
			}
			door := Pos{ox + ow/2, oy + oh - 1}
			l.Tiles[door.Y][door.X] = TileDoor
			if l.Doors == nil {
				l.Doors = make(map[Pos]bool)
			}
			l.Doors[door] = false
			outside := Pos{door.X, door.Y + 1}
			if l.InBounds(outside) && l.At(outside) == TileWall {
				l.Tiles[outside.Y][outside.X] = TileFloor
			}
			interior := rect{ox + 1, oy + 1, vw, vh}
			rooms = append(rooms, interior)
			center := Pos{interior.x + interior.w/2, interior.y + interior.h/2}
			l.Tiles[center.Y][center.X] = TileFloor
			l.Features = append(l.Features, Feature{Pos: center, Type: FeatureVault, Locked: true, Treasure: 25 + rng.IntN(56), Trapped: rng.Float64() < 0.2})
			vaultDoors = append(vaultDoors, door)
		}
	}
	mw := 3 + rng.IntN(2)
	mh := 3 + rng.IntN(2)
	if mr, mdoor, ok := trySpecialRoom(mw, mh); ok {
		center := Pos{mr.x + mr.w/2, mr.y + mr.h/2}
		wares := merchantWares(rng)
		l.Features = append(l.Features, Feature{Pos: center, Type: FeatureMerchant, Wares: wares})
		_ = mdoor
	} else if len(rooms) > 1 {
		idx := rng.IntN(len(rooms))
		r := rooms[idx]
		// avoid picking same room as vault if possible
		if len(l.Features) > 0 {
			for _, f := range l.Features {
				if f.IsVault() && f.Pos == (Pos{r.x + r.w/2, r.y + r.h/2}) {
					idx = (idx + 1) % len(rooms)
					r = rooms[idx]
					break
				}
			}
		}
		center := Pos{r.x + r.w/2, r.y + r.h/2}
		// ensure merchant room 3x3..4x4 by maybe shrinking? just use center
		side := rng.IntN(4)
		var door Pos
		switch side {
		case 0:
			door = Pos{r.x + r.w/2, r.y - 1}
		case 1:
			door = Pos{r.x + r.w/2, r.y + r.h}
		case 2:
			door = Pos{r.x - 1, r.y + r.h/2}
		case 3:
			door = Pos{r.x + r.w, r.y + r.h/2}
		}
		if l.InBounds(door) && (l.At(door) == TileWall || l.At(door) == TileFloor) {
			l.Tiles[door.Y][door.X] = TileDoor
			if l.Doors == nil {
				l.Doors = make(map[Pos]bool)
			}
			l.Doors[door] = false
		}
		// avoid duplicate merchant feature if already present at center
		has := false
		for _, f := range l.Features {
			if f.Pos == center && f.IsMerchant() {
				has = true
				break
			}
		}
		if !has {
			wares := merchantWares(rng)
			l.Features = append(l.Features, Feature{Pos: center, Type: FeatureMerchant, Wares: wares})
		}
	}
	// Hallway doors at corridor ends (rooms biomes)
	for i := 1; i < len(rooms); i++ {
		if rng.Float64() > 0.55 {
			continue
		}
		a := rooms[i-1]
		b := rooms[i]
		ay := a.y + a.h/2
		by := b.y + b.h/2
		candidates := []Pos{}
		if a.x+a.w < l.W-1 {
			candidates = append(candidates, Pos{a.x + a.w, ay})
		}
		if a.x > 1 {
			candidates = append(candidates, Pos{a.x - 1, ay})
		}
		if a.y+a.h < l.H-1 {
			candidates = append(candidates, Pos{a.x + a.w/2, a.y + a.h})
		}
		if a.y > 1 {
			candidates = append(candidates, Pos{a.x + a.w/2, a.y - 1})
		}
		if b.x+b.w < l.W-1 {
			candidates = append(candidates, Pos{b.x + b.w, by})
		}
		if b.x > 1 {
			candidates = append(candidates, Pos{b.x - 1, by})
		}
		if b.y+b.h < l.H-1 {
			candidates = append(candidates, Pos{b.x + b.w/2, b.y + b.h})
		}
		if b.y > 1 {
			candidates = append(candidates, Pos{b.x + b.w/2, b.y - 1})
		}
		bx := b.x + b.w/2
		ax := a.x + a.w/2
		candidates = append(candidates, Pos{bx, ay}, Pos{ax, by})
		var viable []Pos
		for _, p := range candidates {
			if !l.InBounds(p) {
				continue
			}
			if l.At(p) != TileFloor {
				continue
			}
			if p == l.StairsUp || p == l.StairsDown {
				continue
			}
			if l.IsDoor(p) {
				continue
			}
			if !isSingleWideDoor(l, p) {
				continue
			}
			viable = append(viable, p)
		}
		if len(viable) > 0 {
			chosen := viable[rng.IntN(len(viable))]
			l.Tiles[chosen.Y][chosen.X] = TileDoor
			l.Doors[chosen] = false
		}
	}
	// Post-pass: remove doors without opposite walls and not single-wide.
	// Ensures doors are in 1-tile corridors: opposite walls (N-S or E-W) and
	// adjacent corridor segment width is 1 (orthogonal walls on corridor side).
	for y := range l.H {
		for x := range l.W {
			p := Pos{x, y}
			if l.At(p) != TileDoor {
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
	// Re-enforce vault outer walls (protect against overwrites).
	for idx, outer := range vaultOuters {
		var door Pos
		if idx < len(vaultDoors) {
			door = vaultDoors[idx]
		}
		for yy := outer.y; yy < outer.y+outer.h; yy++ {
			for xx := outer.x; xx < outer.x+outer.w; xx++ {
				isPerim := xx == outer.x || xx == outer.x+outer.w-1 || yy == outer.y || yy == outer.y+outer.h-1
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
		for yy := outer.y + 1; yy < outer.y+outer.h-1; yy++ {
			for xx := outer.x + 1; xx < outer.x+outer.w-1; xx++ {
				if l.At(Pos{xx, yy}) != TileFloor {
					l.Tiles[yy][xx] = TileFloor
				}
			}
		}
	}
	// Scan-based vault re-enforce (fallback for any broken vault)
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
	if len(rooms) > 0 {
		vaultSet := make(map[Pos]bool, len(l.Features))
		for _, f := range l.Features {
			if f.IsVault() {
				vaultSet[f.Pos] = true
			}
		}
		isVaultRoom := func(r rect) bool {
			c := Pos{r.x + r.w/2, r.y + r.h/2}
			return vaultSet[c]
		}
		upIdx := 0
		for upIdx < len(rooms) && isVaultRoom(rooms[upIdx]) {
			upIdx++
		}
		if upIdx >= len(rooms) {
			upIdx = 0
		}
		downIdx := len(rooms) - 1
		for downIdx > 0 && isVaultRoom(rooms[downIdx]) {
			downIdx--
		}
		if downIdx < 0 {
			downIdx = len(rooms) - 1
		}
		if upIdx == downIdx && isVaultRoom(rooms[upIdx]) && len(rooms) > 1 {
			for i, r := range rooms {
				if !isVaultRoom(r) {
					downIdx = i
					break
				}
			}
		}
		r := rooms[upIdx]
		l.StairsUp = Pos{r.x + r.w/2, r.y + r.h/2}
		l.Tiles[l.StairsUp.Y][l.StairsUp.X] = TileStairsUp
		r2 := rooms[downIdx]
		l.StairsDown = Pos{r2.x + r2.w/2, r2.y + r2.h/2}
		if l.StairsDown != l.StairsUp {
			l.Tiles[l.StairsDown.Y][l.StairsDown.X] = TileStairsDown
		} else {
			l.StairsDown = Pos{r2.x + r2.w/2 + 1, r2.y + r2.h/2}
			if l.InBounds(l.StairsDown) {
				l.Tiles[l.StairsDown.Y][l.StairsDown.X] = TileStairsDown
			}
		}
	}
	_ = vaultOuters
	_ = vaultDoors
}
// generateCavern uses cellular automata (and drunkard walk fallback) for organic caves.
func (l *Level) generateCavern(rng *rand.Rand, floor int) {
	_ = floor
	W, H := l.W, l.H
	// Start with solid walls.
	for y := range H {
		for x := range W {
			l.Tiles[y][x] = TileWall
		}
	}
	// Random fill interior 45%.
	for y := 1; y < H-1; y++ {
		for x := 1; x < W-1; x++ {
			if rng.Float64() < 0.45 {
				l.Tiles[y][x] = TileFloor
			} else {
				l.Tiles[y][x] = TileWall
			}
		}
	}
	// Cellular automata iterations.
	for iter := range 4 {
		_ = iter
		next := make([][]Tile, H)
		for y := range H {
			next[y] = make([]Tile, W)
			copy(next[y], l.Tiles[y])
		}
		for y := 1; y < H-1; y++ {
			for x := 1; x < W-1; x++ {
				walls := 0
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if dx == 0 && dy == 0 {
							continue
						}
						if l.Tiles[y+dy][x+dx] == TileWall {
							walls++
						}
					}
				}
				if walls >= 5 {
					next[y][x] = TileWall
				} else if walls <= 2 {
					next[y][x] = TileFloor
				}
			}
		}
		l.Tiles = next
	}
	// Keep largest connected floor component; wall the rest.
	visited := make([][]bool, H)
	for y := range H {
		visited[y] = make([]bool, W)
	}
	var components [][]Pos
	for y := 1; y < H-1; y++ {
		for x := 1; x < W-1; x++ {
			if l.Tiles[y][x] != TileFloor || visited[y][x] {
				continue
			}
			// Flood fill
			var comp []Pos
			queue := []Pos{{x, y}}
			visited[y][x] = true
			for len(queue) > 0 {
				cur := queue[0]
				queue = queue[1:]
				comp = append(comp, cur)
				for _, d := range AllDirs[:4] { // cardinal for connectivity
					np := Pos{cur.X + d.DX, cur.Y + d.DY}
					if np.X < 1 || np.X >= W-1 || np.Y < 1 || np.Y >= H-1 {
						continue
					}
					if visited[np.Y][np.X] {
						continue
					}
					if l.Tiles[np.Y][np.X] != TileFloor {
						continue
					}
					// Also consider diagonal? keep cardinal to avoid thin diagonal bridges.
					visited[np.Y][np.X] = true
					queue = append(queue, np)
				}
			}
			components = append(components, comp)
		}
	}
	if len(components) == 0 {
		// Degenerate: fallback to rooms.
		for y := range H {
			for x := range W {
				l.Tiles[y][x] = TileWall
			}
		}
		l.generateRooms(rng, floor)
		return
	}
	// Find largest.
	largest := components[0]
	for _, c := range components[1:] {
		if len(c) > len(largest) {
			largest = c
		}
	}
	if len(largest) < 80 {
		// Too small: expand via drunkard walk from center to ensure playability.
		largest = append(largest, l.drunkardWalk(rng, 300)...)
		// Deduplicate via set.
		seen := make(map[Pos]bool)
		var filtered []Pos
		for _, p := range largest {
			if !seen[p] && l.InBounds(p) {
				seen[p] = true
				filtered = append(filtered, p)
			}
		}
		largest = filtered
		for _, p := range largest {
			l.Tiles[p.Y][p.X] = TileFloor
		}
	} else {
		// Wall off smaller components.
		largestSet := make(map[Pos]bool, len(largest))
		for _, p := range largest {
			largestSet[p] = true
		}
		for _, comp := range components {
			if len(comp) == len(largest) {
				// Could be same largest (first match) — skip if same set size and overlap check via first element.
				if comp[0] == largest[0] {
					continue
				}
			}
			// Check if this component is the largest via set inclusion: if any point in largestSet, it's largest.
			isLargest := false
			for _, p := range comp {
				if largestSet[p] {
					isLargest = true
					break
				}
			}
			if isLargest {
				continue
			}
			for _, p := range comp {
				l.Tiles[p.Y][p.X] = TileWall
			}
		}
	}
	// Ensure largest still floors (in case drunkard added).
	for _, p := range largest {
		l.Tiles[p.Y][p.X] = TileFloor
	}
	// Place stairs at two distant walkable positions.
	var walks []Pos
	for _, p := range largest {
		if l.At(p) == TileFloor {
			walks = append(walks, p)
		}
	}
	// Also scan full map for floors not in largest after drunkard.
	if len(walks) < 2 {
		for y := 1; y < H-1; y++ {
			for x := 1; x < W-1; x++ {
				if l.Tiles[y][x] == TileFloor {
					walks = append(walks, Pos{x, y})
				}
			}
		}
	}
	if len(walks) >= 2 {
		// Pick up at minimal sum (top-leftish) and down at maximal sum (bottom-rightish) for distance.
		upIdx, downIdx := 0, 0
		minSum, maxSum := walks[0].X+walks[0].Y, walks[0].X+walks[0].Y
		for i, p := range walks {
			s := p.X + p.Y
			if s < minSum {
				minSum = s
				upIdx = i
			}
			if s > maxSum {
				maxSum = s
				downIdx = i
			}
		}
		// Add jitter: pick random among nearby candidates to avoid determinism of corners.
		// Shift by a small random offset among walks sorted by distance.
		if rng.Float64() < 0.5 {
			upIdx = rng.IntN(len(walks))
			for attempts := 0; attempts < 5; attempts++ {
				cand := rng.IntN(len(walks))
				if walks[cand].X+walks[cand].Y < walks[upIdx].X+walks[upIdx].Y {
					upIdx = cand
				}
			}
			downIdx = rng.IntN(len(walks))
			for attempts := 0; attempts < 5; attempts++ {
				cand := rng.IntN(len(walks))
				if walks[cand].X+walks[cand].Y > walks[downIdx].X+walks[downIdx].Y {
					downIdx = cand
				}
			}
		}
		l.StairsUp = walks[upIdx]
		l.StairsDown = walks[downIdx]
		if l.StairsDown == l.StairsUp && len(walks) > 1 {
			// Nudge down one tile if same.
			for _, p := range walks {
				if p != l.StairsUp {
					l.StairsDown = p
					break
				}
			}
		}
		l.Tiles[l.StairsUp.Y][l.StairsUp.X] = TileStairsUp
		l.Tiles[l.StairsDown.Y][l.StairsDown.X] = TileStairsDown
	} else if len(walks) == 1 {
		l.StairsUp = walks[0]
		l.Tiles[l.StairsUp.Y][l.StairsUp.X] = TileStairsUp
		// Place down adjacent.
		for _, d := range AllDirs {
			np := walks[0].Add(d)
			if l.InBounds(np) {
				l.Tiles[np.Y][np.X] = TileFloor
				l.StairsDown = np
				l.Tiles[np.Y][np.X] = TileStairsDown
				break
			}
		}
	}
}

func (l *Level) drunkardWalk(rng *rand.Rand, steps int) []Pos {
	p := Pos{l.W / 2, l.H / 2}
	// Find starting floor if center is wall, wander to floor.
	for tries := 0; tries < 100; tries++ {
		if l.At(p) == TileFloor {
			break
		}
		p = Pos{1 + rng.IntN(l.W-2), 1 + rng.IntN(l.H-2)}
	}
	var out []Pos
	for range steps {
		if l.InBounds(p) {
			out = append(out, p)
			l.Tiles[p.Y][p.X] = TileFloor
		}
		dir := AllDirs[rng.IntN(len(AllDirs))]
		np := p.Add(dir)
		if np.X < 1 || np.X >= l.W-1 || np.Y < 1 || np.Y >= l.H-1 {
			continue
		}
		p = np
	}
	return out
}

// spawnEnemiesWithBiome spawns depth-appropriate plus biome special enemies.
func (l *Level) spawnEnemiesWithBiome(rng *rand.Rand, floor int, biome *Biome) {
	// Collect walkable floor positions for placement.
	// Must include hallway tiles (TileFloor) and exclude stairs/enemy/feature/door
	// but not over-exclude vault interior. Called BEFORE litter so Walkable
	// does not reject floor due to impassable litter; doors are TileDoor and
	// already excluded via At check. Exclude existing Features (vault/merchant
	// centers from generateRooms) so enemies don't stack on features.
	featureSet := make(map[Pos]bool, len(l.Features))
	for _, f := range l.Features {
		featureSet[f.Pos] = true
	}
	var candidates []Pos
	for y := range l.H {
		for x := range l.W {
			p := Pos{x, y}
			if p == l.StairsUp || p == l.StairsDown {
				continue
			}
			if featureSet[p] {
				continue
			}
			if l.IsDoor(p) {
				continue
			}
			if l.At(p) != TileFloor {
				continue
			}
			if !l.Walkable(p) {
				continue
			}
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return
	}
	partyCount := 3 + floor*2 + rng.IntN(3)
	for range partyCount {
		var p Pos
		for tries := 0; tries < 100; tries++ {
			p = candidates[rng.IntN(len(candidates))]
			occupied := false
			for _, e := range l.Enemies {
				if e.Pos == p {
					occupied = true
					break
				}
			}
			if !occupied {
				break
			}
		}
		ep := l.createPartyForFloor(rng, floor, p)
		l.Enemies = append(l.Enemies, ep)
	}
	// Add special enemies in addition.
	if biome != nil && len(biome.SpecialEnemies) > 0 {
		specialCount := 1
		if len(biome.SpecialEnemies) > 1 && rng.Float64() < 0.5 {
			specialCount = 2
		}
		// Deeper floors slightly more special.
		if floor >= 4 && rng.Float64() < 0.4 {
			specialCount++
		}
		if specialCount > 3 {
			specialCount = 3
		}
		for range specialCount {
			var p Pos
			for tries := 0; tries < 100; tries++ {
				p = candidates[rng.IntN(len(candidates))]
				occupied := false
				for _, e := range l.Enemies {
					if e.Pos == p {
						occupied = true
						break
					}
				}
				if !occupied {
					break
				}
			}
			// Pick special id.
			sID := biome.SpecialEnemies[rng.IntN(len(biome.SpecialEnemies))]
			ep := l.createPartyWithEnemyID(rng, floor, p, sID)
			l.Enemies = append(l.Enemies, ep)
		}
	}
}

func (l *Level) createPartyForFloor(rng *rand.Rand, floor int, pos Pos) *EnemyParty {
	partySize := 1
	if floor >= 1 && rng.IntN(3) == 0 {
		partySize++
	}
	if floor >= 3 && rng.IntN(2) == 0 {
		partySize++
	}
	if floor >= 5 && rng.IntN(2) == 0 {
		partySize++
	}
	if partySize > 4 {
		partySize = 4
	}
	if floor >= 2 && rng.IntN(4) == 0 {
		partySize = 1 + rng.IntN(2)
	}
	ep := &EnemyParty{Pos: pos, Active: 0}
	for range partySize {
		entry := pickEnemyForFloor(rng, floor)
		mem := buildMemberFromEntry(entry, rng, floor)
		ep.Members = append(ep.Members, mem)
	}
	return ep
}

func (l *Level) createPartyWithEnemyID(rng *rand.Rand, floor int, pos Pos, id string) *EnemyParty {
	entries := loadEnemies()
	var entry enemyEntry
	found := false
	for _, e := range entries {
		if e.ID == id {
			entry = e
			found = true
			break
		}
	}
	if !found {
		// Fallback synthetic entry.
		entry = enemyEntry{ID: id, Name: id, Glyph: "x", Color: "#6a7a7a", DamageType: "physical", Effect: "hex", EffectChance: 0.08, XP: 12, TalentChance: 0.08, AffixChance: 0.04}
		// Themed defaults.
		switch id {
		case "vine_horror":
			entry.Name = "Vine Horror"
			entry.Glyph = "v"
			entry.Color = "#4a6a4a"
			entry.DamageType = "physical"
			entry.Effect = "entangle"
		case "spore_mother":
			entry.Name = "Spore Mother"
			entry.Glyph = "s"
			entry.Color = "#6a8a6a"
			entry.DamageType = "magic"
			entry.Effect = "spore"
			entry.EffectChance = 0.15
			entry.Regen = true
		}
	}
	partySize := 1
	if rng.Float64() < 0.3 {
		partySize = 2
	}
	if floor >= 5 && rng.Float64() < 0.2 {
		partySize = 3
	}
	ep := &EnemyParty{Pos: pos, Active: 0}
	for range partySize {
		mem := buildMemberFromEntry(entry, rng, floor)
		ep.Members = append(ep.Members, mem)
	}
	return ep
}

func buildMemberFromEntry(entry enemyEntry, rng *rand.Rand, floor int) *Member {
	hp := 6 + floor*2 + rng.IntN(4)
	if entry.Regen {
		hp += 4
	}
	atkMin := 2 + floor
	atkMax := atkMin + 2 + rng.IntN(2)
	if entry.DamageType == "magic" && floor > 2 {
		atkMin++
		atkMax++
	}
	def := 0
	mdef := 0
	if floor >= 2 {
		def = floor / 3
		mdef = floor / 4
	}
	if entry.ID == "orc" {
		def++
	}
	if entry.ID == "kobold" {
		mdef++
	}
	if entry.ID == "troll" {
		def++
		mdef++
	}
	mem := &Member{
		Name: entry.Name, Class: entry.ID,
		HP: hp, MaxHP: hp,
		ATK: [2]int{atkMin, atkMax},
		DEF: def, MDEF: mdef,
		Alive: true, DamageType: entry.DamageType,
		Effect: entry.Effect, EffectChance: entry.EffectChance,
		Regen: entry.Regen, XP: entry.XP, Color: entry.Color,
	}
	if mem.EffectChance < 0 {
		mem.EffectChance = 0
	}
	if mem.EffectChance > 0.3 {
		mem.EffectChance = 0.3
	}
	if floor >= 3 {
		talentChance := entry.TalentChance
		affixChance := entry.AffixChance
		if floor >= 5 {
			talentChance += 0.05
			affixChance += 0.03
		}
		if talentChance > 0 && rng.Float64() < talentChance {
			opts := GetTalentOptions(rng, mem.Class, 1)
			if len(opts) > 0 {
				chosen := opts[0]
				mem.Talents = append(mem.Talents, chosen)
			}
		}
		if affixChance > 0 && rng.Float64() < affixChance {
			aff := GetRandomAffix(rng)
			mem.Affixes = append(mem.Affixes, aff)
		}
	}
	return mem
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
		biome = GetBiomeForFloor(g.Floor)
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
	return GetBiomeForFloor(g.Floor)
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

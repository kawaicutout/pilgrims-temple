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

// LitterObj is a placed litter instance on a level.
type LitterObj struct {
	Pos            Pos    `json:"pos"`
	Kind           string `json:"kind"`
	Category       string `json:"category"` // destructible | passable | impassable
	Glyph          rune   `json:"glyph"`
	BlocksMovement bool   `json:"blocksMovement"`
	BlocksFOV      bool   `json:"blocksFOV"`
}

type biomesFile struct {
	Biomes []Biome `json:"biomes"`
	Notes  string  `json:"notes"`
}

var (
	biomesOnce  sync.Once
	biomesCache []Biome
)

func fallbackBiomes() []Biome {
	return []Biome{
		{
			ID: "crypt", Name: "Crypt", EligibleLevelRange: [2]int{1, 3},
			ColorPalette: ColorPalette{Primary: "#6a7a7a", Secondary: "#5a6a6a", Accent: "#8a7a7a"}, Color: "#6a7a7a", Tint: "#6a7a7a",
			Litter:           LitterDef{Destructible: []string{"barrel", "crate", "urn"}, Passable: []string{"rubble", "dust", "puddle"}, Impassable: []string{"column", "altar", "sarcophagus"}},
			GenerationMethod: "rooms", SpecialEnemies: []string{"rat", "kobold"},
			Ambience: []string{"Cold drafts curl through the crypt aisles.", "Distant stone lids grind against their sarcophagi.", "A faint incense of old myrrh hangs in the air.", "Footsteps echo too long in the vaulted dark."},
		},
		{
			ID: "ossuary", Name: "Ossuary", EligibleLevelRange: [2]int{2, 4},
			ColorPalette: ColorPalette{Primary: "#8a7a6a", Secondary: "#7a6a5a", Accent: "#9a8a7a"}, Color: "#8a7a6a", Tint: "#8a7a6a",
			Litter:           LitterDef{Destructible: []string{"bone_pile", "crate", "barrel"}, Passable: []string{"rubble", "bone_dust", "puddle"}, Impassable: []string{"bone_column", "pit", "rubble_wall"}},
			GenerationMethod: "rooms", SpecialEnemies: []string{"kobold", "orc"},
			Ambience: []string{"Bones whisper as dust shifts across the ossuary.", "A hollow clatter rolls from stacked skulls.", "Chalky air catches in your throat.", "Shadows pool between leaning bone-columns."},
		},
		{
			ID: "fungal", Name: "Fungal Grove", EligibleLevelRange: [2]int{3, 5},
			ColorPalette: ColorPalette{Primary: "#5a7a5a", Secondary: "#4a6a4a", Accent: "#6a8a6a"}, Color: "#5a7a5a", Tint: "#5a7a5a",
			Litter:           LitterDef{Destructible: []string{"mushroom_cap", "spore_pod", "crate"}, Passable: []string{"moss", "slime", "rubble"}, Impassable: []string{"fungal_column", "pit", "thicket"}},
			GenerationMethod: "cavern", SpecialEnemies: []string{"spore_mother", "rat"},
			Ambience: []string{"Spores drift like pale snow through fungal gloom.", "Mushroom caps pulse with faint light.", "The air is thick, sweet, and slightly sour.", "Soft caps sigh as you brush past."},
		},
		{
			ID: "jungle", Name: "Jungle Overgrowth", EligibleLevelRange: [2]int{4, 6},
			ColorPalette: ColorPalette{Primary: "#5a6a4a", Secondary: "#4a5a4a", Accent: "#6a7a4a"}, Color: "#5a6a4a", Tint: "#5a6a4a",
			Litter:           LitterDef{Destructible: []string{"vine_cluster", "crate", "barrel"}, Passable: []string{"moss", "rubble", "puddle"}, Impassable: []string{"vine_wall", "pit", "column"}},
			GenerationMethod: "cavern", SpecialEnemies: []string{"vine_horror", "kobold"},
			Ambience: []string{"Vines tighten overhead with a soft creak.", "Humid air beads on cold stone.", "Leaves rustle where no wind should reach.", "A distant vine snaps taut, then stills."},
		},
		{
			ID: "cinder", Name: "Cinder Chapel", EligibleLevelRange: [2]int{5, 8},
			ColorPalette: ColorPalette{Primary: "#7a6a5a", Secondary: "#6a5a4a", Accent: "#8a7a5a"}, Color: "#7a6a5a", Tint: "#7a6a5a",
			Litter:           LitterDef{Destructible: []string{"ash_barrel", "crate", "cinder_block"}, Passable: []string{"ash", "rubble", "puddle"}, Impassable: []string{"cinder_column", "lava_pit", "rubble_wall"}},
			GenerationMethod: "cavern", SpecialEnemies: []string{"troll", "orc"},
			Ambience: []string{"Ash drifts on heat that has no source.", "Cinder clicks underfoot, cooling and cracking.", "A low draft carries the tang of soot.", "Embers blink in the dark like tired eyes."},
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

// ---------------------------------------------------------------------------
// Litter helpers
// ---------------------------------------------------------------------------

func litterGlyph(kind string) rune {
	switch kind {
	case "barrel", "ash_barrel":
		return ']'
	case "crate", "cinder_block":
		return '='
	case "urn", "bone_pile":
		return 'U'
	case "mushroom_cap":
		return 'o'
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
	case "column", "altar", "sarcophagus", "bone_column", "fungal_column", "vine_wall", "cinder_column":
		return '#'
	case "pit", "lava_pit":
		return '0'
	case "thicket":
		return '#'
	default:
		return '.'
	}
}

func litterBlocks(kind, category string) (blocksMove, blocksFOV bool) {
	switch category {
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
	return LitterObj{Pos: pos, Kind: kind, Category: category, Glyph: litterGlyph(kind), BlocksMovement: bm, BlocksFOV: bf}
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
	}
	// Spawn enemies (depth-appropriate plus special).
	l.spawnEnemiesWithBiome(rng, floor, biome)
	// Spawn litter with BFS guard.
	spawnLitter(l, rng, biome)
	// Spawn level features (vault/forge/den/pitfall + merchant/fountain/shrine) via features.go.
	// Centralized here so both rooms and cavern paths populate features; Level.Generate delegates here.
	l.Features = MaybeSpawnFeatures(l, floor, rng)
	// Debug helper
	_ = fmt.Sprintf("biome %s floor %d", biome.ID, floor)
}

// generateRooms is extracted rooms+corridors generator (logic from original Generate).
func (l *Level) generateRooms(rng *rand.Rand, floor int) {
	type rect struct{ x, y, w, h int }
	var rooms []rect
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
	if len(rooms) > 0 {
		r := rooms[0]
		l.StairsUp = Pos{r.x + r.w/2, r.y + r.h/2}
		l.Tiles[l.StairsUp.Y][l.StairsUp.X] = TileStairsUp
		r2 := rooms[len(rooms)-1]
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
	// Collect walkable room/cavern positions for placement.
	var candidates []Pos
	// Use largest floor set: scan all floors.
	for y := range l.H {
		for x := range l.W {
			p := Pos{x, y}
			if p == l.StairsUp || p == l.StairsDown {
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

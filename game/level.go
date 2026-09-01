package game

import (
	"math/rand/v2"
)

// Level holds one floor.
type Level struct {
	W, H   int
	Tiles  [][]Tile
	Seen   [][]bool // explored
	Visible [][]bool

	StairsUp   Pos
	StairsDown Pos

	Enemies []*EnemyParty
}

func NewLevel(w, h int) *Level {
	tiles := make([][]Tile, h)
	seen := make([][]bool, h)
	vis := make([][]bool, h)
	for y := range h {
		tiles[y] = make([]Tile, w)
		seen[y] = make([]bool, w)
		vis[y] = make([]bool, w)
		for x := range w {
			tiles[y][x] = TileWall
		}
	}
	return &Level{W: w, H: h, Tiles: tiles, Seen: seen, Visible: vis}
}

func (l *Level) InBounds(p Pos) bool { return p.X >= 0 && p.X < l.W && p.Y >= 0 && p.Y < l.H }
func (l *Level) At(p Pos) Tile {
	if !l.InBounds(p) {
		return TileWall
	}
	return l.Tiles[p.Y][p.X]
}
func (l *Level) Set(p Pos, t Tile) {
	if l.InBounds(p) {
		l.Tiles[p.Y][p.X] = t
	}
}
func (l *Level) Walkable(p Pos) bool { return l.At(p).Walkable() }
func (l *Level) BlocksFOV(p Pos) bool { return l.At(p).BlocksFOV() }

// EnemyParty is M1 minimal enemy (party of one for now).
type EnemyParty struct {
	Pos    Pos
	Glyph  rune
	Name   string
	HP     int
	MaxHP  int
	ATK    [2]int // min,max inclusive
	Alive  bool
	Active int // for M1 always 0
}

func (e *EnemyParty) IsAlive() bool { return e.Alive && e.HP > 0 }

// Generate fills a level with rooms+corridors and stairs. Deterministic from rng.
func (l *Level) Generate(rng *rand.Rand, floor int) {
	// Simple room generator: carve random rooms, connect via L-shaped corridors.
	type rect struct{ x, y, w, h int }
	var rooms []rect
	attempts := 50
	for range attempts {
		w := 5 + rng.IntN(7)  // 5-11
		h := 4 + rng.IntN(5)  // 4-8
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
		// Carve room
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
	// Ensure at least one room
	if len(rooms) == 0 {
		for y := 1; y < l.H-1; y++ {
			for x := 1; x < l.W-1; x++ {
				l.Tiles[y][x] = TileFloor
			}
		}
		rooms = append(rooms, rect{1, 1, l.W - 2, l.H - 2})
	}
	// Connect rooms sequentially via L-corridors
	for i := 1; i < len(rooms); i++ {
		a := rooms[i-1]
		b := rooms[i]
		ax := a.x + a.w/2
		ay := a.y + a.h/2
		bx := b.x + b.w/2
		by := b.y + b.h/2
		// Randomize L bend order
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
	// Place stairs at center of first/last room
	if len(rooms) > 0 {
		r := rooms[0]
		l.StairsUp = Pos{r.x + r.w/2, r.y + r.h/2}
		l.Tiles[l.StairsUp.Y][l.StairsUp.X] = TileStairsUp
		r2 := rooms[len(rooms)-1]
		l.StairsDown = Pos{r2.x + r2.w/2, r2.y + r2.h/2}
		// Avoid overwriting up stairs if single room
		if l.StairsDown != l.StairsUp {
			l.Tiles[l.StairsDown.Y][l.StairsDown.X] = TileStairsDown
		} else {
			// Offset down stairs
			l.StairsDown = Pos{r2.x + r2.w/2 + 1, r2.y + r2.h/2}
			if l.InBounds(l.StairsDown) {
				l.Tiles[l.StairsDown.Y][l.StairsDown.X] = TileStairsDown
			}
		}
	}
	// Spawn enemies: deeper floors -> more
	enemyCount := 3 + floor*2 + rng.IntN(3)
	glyphs := []rune{'g', 'o', 'k', 'r'}
	names := []string{"goblin", "orc", "kobold", "rat"}
	for i := 0; i < enemyCount; i++ {
		// Pick random floor tile not on stairs
		var p Pos
		for tries := 0; tries < 100; tries++ {
			rr := rooms[rng.IntN(len(rooms))]
			p = Pos{rr.x + rng.IntN(rr.w), rr.y + rng.IntN(rr.h)}
			if p == l.StairsUp || p == l.StairsDown {
				continue
			}
			if !l.Walkable(p) {
				continue
			}
			// Avoid stacking
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
		idx := rng.IntN(len(glyphs))
		hp := 6 + floor*2 + rng.IntN(4)
		atkMin := 2 + floor
		atkMax := atkMin + 2 + rng.IntN(2)
		l.Enemies = append(l.Enemies, &EnemyParty{
			Pos:   p,
			Glyph: glyphs[idx],
			Name:  names[idx],
			HP:    hp, MaxHP: hp,
			ATK:   [2]int{atkMin, atkMax},
			Alive: true,
		})
	}
}

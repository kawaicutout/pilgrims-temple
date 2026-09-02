package game

import (
	"encoding/json"
	"fmt"
)

// Pos is a grid coordinate.
type Pos struct{ X, Y int }

// MarshalJSON ensures Pos marshals as object {"X":1,"Y":2} for struct fields.
func (p Pos) MarshalJSON() ([]byte, error) {
	type posAlias Pos
	return json.Marshal(posAlias(p))
}

func (p *Pos) UnmarshalJSON(data []byte) error {
	type posAlias Pos
	var a posAlias
	if err := json.Unmarshal(data, &a); err != nil {
		// fallback for string form "x,y" (used as map key text)
		var s string
		if err2 := json.Unmarshal(data, &s); err2 == nil {
			_, err2 = fmt.Sscanf(s, "%d,%d", &p.X, &p.Y)
			return err2
		}
		return err
	}
	*p = Pos(a)
	return nil
}

// MarshalText implements encoding.TextMarshaler for map keys (e.g., Doors).
func (p Pos) MarshalText() ([]byte, error) { return []byte(fmt.Sprintf("%d,%d", p.X, p.Y)), nil }

// UnmarshalText implements encoding.TextUnmarshaler for map keys.
func (p *Pos) UnmarshalText(text []byte) error {
	_, err := fmt.Sscanf(string(text), "%d,%d", &p.X, &p.Y)
	return err
}

func (p Pos) Add(d Dir) Pos { return Pos{p.X + d.DX, p.Y + d.DY} }

// Dir is a movement direction.
type Dir struct{ DX, DY int }

var (
	DirN    = Dir{0, -1}
	DirS    = Dir{0, 1}
	DirW    = Dir{-1, 0}
	DirE    = Dir{1, 0}
	DirNW   = Dir{-1, -1}
	DirNE   = Dir{1, -1}
	DirSW   = Dir{-1, 1}
	DirSE   = Dir{1, 1}
	DirNone = Dir{0, 0}
)

var AllDirs = []Dir{DirN, DirS, DirW, DirE, DirNW, DirNE, DirSW, DirSE}

// Tile kind.
type Tile int

const (
	TileWall Tile = iota
	TileFloor
	TileStairsDown
	TileStairsUp
	TileDoor
)

func (t Tile) Walkable() bool {
	switch t {
	case TileFloor, TileStairsDown, TileStairsUp:
		return true
	case TileDoor:
		// Door walkability is level-state dependent (open vs closed).
		// Base tile is not walkable; Level.Walkable handles open doors.
		return false
	default:
		return false
	}
}

func (t Tile) Glyph() rune {
	switch t {
	case TileWall:
		return '#'
	case TileFloor:
		return '.'
	case TileStairsDown:
		return '>'
	case TileStairsUp:
		return '<'
	case TileDoor:
		return '+'
	default:
		return ' '
	}
}

func (t Tile) BlocksFOV() bool { return t == TileWall }

// Cell is a rendered map cell.
type Cell struct {
	Glyph rune
	FG    string // token name for styling (unused in core, consumed by frontend)
	BG    string
}

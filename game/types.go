package game

// Pos is a grid coordinate.
type Pos struct{ X, Y int }

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
)

func (t Tile) Walkable() bool {
	switch t {
	case TileFloor, TileStairsDown, TileStairsUp:
		return true
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

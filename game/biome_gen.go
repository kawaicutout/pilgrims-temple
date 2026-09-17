package game

import (
	"fmt"
	"math/rand/v2"
	"os"
)

// debugGenLogf prints world-gen diagnostics only when PILGRIMS_DEBUG_GEN
// is set. Unconditional prints corrupt the terminal UI, which shares
// stdout with the game.
func debugGenLogf(format string, args ...any) {
	if os.Getenv("PILGRIMS_DEBUG_GEN") == "" {
		return
	}
	fmt.Printf(format, args...)
}

// dumpLevelGeometry prints level geometry for debugging walkability failures.
func dumpLevelGeometry(l *Level) {
	if l == nil {
		debugGenLogf("dumpLevelGeometry: nil level\n")
		return
	}
	debugGenLogf("=== Level dump W=%d H=%d StairsUp=%v StairsDown=%v Biome=%s Floor=%d ===\n", l.W, l.H, l.StairsUp, l.StairsDown, l.BiomeID, l.Floor)
	counts := map[Tile]int{}
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			counts[l.Tiles[y][x]]++
		}
	}
	debugGenLogf("Tile counts: wall=%d floor=%d door=%d up=%d down=%d total=%d\n", counts[TileWall], counts[TileFloor], counts[TileDoor], counts[TileStairsUp], counts[TileStairsDown], l.W*l.H)
	debugGenLogf("Features (%d):\n", len(l.Features))
	for i, f := range l.Features {
		debugGenLogf("  %d: type=%s pos=%v locked=%v treasure=%d trapped=%v\n", i, f.Type, f.Pos, f.Locked, f.Treasure, f.Trapped)
	}
	debugGenLogf("Doors (%d):\n", len(l.Doors))
	for p, open := range l.Doors {
		debugGenLogf("  %v open=%v tile=%v\n", p, open, l.At(p))
	}
	vaultSet := map[Pos]bool{}
	for _, f := range l.Features {
		if f.IsVault() {
			vaultSet[f.Pos] = true
		}
	}
	for y := 0; y < l.H; y++ {
		row := make([]rune, l.W)
		for x := 0; x < l.W; x++ {
			p := Pos{x, y}
			if vaultSet[p] {
				row[x] = '?'
				continue
			}
			t := l.Tiles[y][x]
			switch t {
			case TileWall:
				row[x] = '#'
			case TileFloor:
				row[x] = '.'
			case TileDoor:
				row[x] = '+'
			case TileStairsUp:
				row[x] = '<'
			case TileStairsDown:
				row[x] = '>'
			default:
				row[x] = ' '
			}
		}
		debugGenLogf("%s\n", string(row))
	}
}

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
		if l.W-w-2 <= 0 || l.H-h-2 <= 0 {
			continue
		}
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
				p := Pos{x, ay}
				if !l.InBounds(p) {
					continue
				}
				l.Tiles[ay][x] = TileFloor
			}
			for y := min(ay, by); y <= max(ay, by); y++ {
				p := Pos{bx, y}
				if !l.InBounds(p) {
					continue
				}
				l.Tiles[y][bx] = TileFloor
			}
		} else {
			for y := min(ay, by); y <= max(ay, by); y++ {
				p := Pos{ax, y}
				if !l.InBounds(p) {
					continue
				}
				l.Tiles[y][ax] = TileFloor
			}
			for x := min(ax, bx); x <= max(ax, bx); x++ {
				p := Pos{x, by}
				if !l.InBounds(p) {
					continue
				}
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
			if l.W-w-2 <= 0 || l.H-h-2 <= 0 {
				return rect{}, Pos{}, false
			}
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
		// Clamp to [1, W-ow-1] and [1, H-oh-1]
		if ox < 1 {
			ox = 1
		}
		if oy < 1 {
			oy = 1
		}
		if ox+ow >= l.W {
			ox = l.W - ow - 1
			if ox < 1 {
				ox = 1
			}
		}
		if oy+oh >= l.H {
			oy = l.H - oh - 1
			if oy < 1 {
				oy = 1
			}
		}
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
		// Clamp ox,oy to [1, W-ow-1] and [1, H-oh-1] with max(1,min) logic
		if l.W > ow+1 && l.H > oh+1 {
			if ox < 1 {
				ox = 1
			}
			if oy < 1 {
				oy = 1
			}
			if ox+ow >= l.W {
				ox = l.W - ow - 1
				if ox < 1 {
					ox = 1
				}
			}
			if oy+oh >= l.H {
				oy = l.H - oh - 1
				if oy < 1 {
					oy = 1
				}
			}
		}
		if ox+ow < l.W && oy+oh < l.H && ox >= 1 && oy >= 1 {
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
	if mr, _, ok := trySpecialRoom(mw, mh); ok {
		center := Pos{mr.x + mr.w/2, mr.y + mr.h/2}
		wares := merchantWares(rng)
		l.Features = append(l.Features, Feature{Pos: center, Type: FeatureMerchant, Wares: wares})
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
}

// generateCavern uses cellular automata (and drunkard walk fallback) for organic caves.
func (l *Level) generateCavern(rng *rand.Rand, floor int) {
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
	for range 4 {
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

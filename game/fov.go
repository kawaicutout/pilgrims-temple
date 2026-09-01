package game

// ComputeFOV fills lvl.Visible from origin with radius via shadowcast (octants).
func ComputeFOV(lvl *Level, origin Pos, radius int) {
	// Clear visible
	for y := range lvl.H {
		for x := range lvl.W {
			lvl.Visible[y][x] = false
		}
	}
	if !lvl.InBounds(origin) {
		return
	}
	lvl.Visible[origin.Y][origin.X] = true
	lvl.Seen[origin.Y][origin.X] = true
	for oct := range 8 {
		castOctant(lvl, origin, radius, oct)
	}
}

func castOctant(lvl *Level, origin Pos, radius, oct int) {
	// Recursive shadowcast using integer slopes.
	var cast func(row, start, end int)
	cast = func(row, startSlopeNum, startSlopeDen int) {}
	// Iterative slope-scan implementation (simpler, no recursion depth)
	_ = cast
	// For M1, we use a simpler perm-sight: ray to every cell within radius, bresenham LOS.
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy > radius*radius+radius {
				continue
			}
			target := Pos{origin.X + dx, origin.Y + dy}
			if !lvl.InBounds(target) {
				continue
			}
			if hasLOS(lvl, origin, target) {
				lvl.Visible[target.Y][target.X] = true
				lvl.Seen[target.Y][target.X] = true
			}
		}
	}
	// Ensure parameter used
	_ = oct
}

func hasLOS(lvl *Level, from, to Pos) bool {
	// Bresenham
	x0, y0 := from.X, from.Y
	x1, y1 := to.X, to.Y
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy
	for {
		if !(x0 == from.X && y0 == from.Y) && !(x0 == to.X && y0 == to.Y) {
			if lvl.BlocksFOV(Pos{x0, y0}) {
				return false
			}
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
	return true
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

package game

import (
	"strings"
	"testing"
)

// Fireball must only burn: an old code path also revealed the whole floor
// and logged the mapping line.
func TestFireballDoesNotMap(t *testing.T) {
	w, h := 10, 8
	seen := make([][]bool, h)
	for y := range seen {
		seen[y] = make([]bool, w)
	}
	g := &Game{
		Levels: []*Level{{W: w, H: h, Seen: seen}},
		Floor:  0,
		Party:  &Party{Pos: Pos{X: 5, Y: 4}},
	}
	g.applyScrollEffect("fireball", true, nil, Pos{}, nil)
	for y := range seen {
		for x := range seen[y] {
			if seen[y][x] {
				t.Fatalf("fireball revealed tile %d,%d", x, y)
			}
		}
	}
	for _, line := range g.Log {
		if strings.Contains(line, "Mapping") {
			t.Fatalf("fireball logged mapping: %q", line)
		}
	}
}

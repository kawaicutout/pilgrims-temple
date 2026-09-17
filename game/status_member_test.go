package game

import (
	"math/rand/v2"
	"strings"
	"testing"
)

func memberGame() *Game {
	w, h := 5, 5
	tiles := make([][]Tile, h)
	seen := make([][]bool, h)
	vis := make([][]bool, h)
	for y := range tiles {
		tiles[y] = make([]Tile, w)
		seen[y] = make([]bool, w)
		vis[y] = make([]bool, w)
		for x := range tiles[y] {
			tiles[y][x] = TileFloor
		}
	}
	a := &Member{Name: "A", Class: "fighter", HP: 20, MaxHP: 20, ATK: [2]int{5, 5}, Light: 6, Alive: true}
	b := &Member{Name: "B", Class: "rogue", HP: 20, MaxHP: 20, ATK: [2]int{5, 5}, Light: 6, Alive: true}
	g := &Game{
		Seed:      3,
		RNG:       rand.New(rand.NewPCG(3, 3)),
		Levels:    []*Level{{W: w, H: h, Tiles: tiles, Seen: seen, Visible: vis}},
		Floor:     0,
		Party:     &Party{Members: []*Member{a, b}, Pos: Pos{2, 2}},
		Food:      500,
		FoodFloat: 500,
	}
	g.Tuning.Layout.LogLines = 8
	return g
}

// Confusion pins to one member: the confused actor stumbles, and acting
// with a different member moves exactly (design §6.5 counterplay).
func TestConfusionIsPerMember(t *testing.T) {
	g := memberGame()
	g.Party.Members[0].ApplyStatus(StatusConfusion, 5)
	g.TryMove(DirN)
	stumbled := false
	for _, line := range g.Log {
		if strings.Contains(line, "stumble") {
			stumbled = true
		}
	}
	if !stumbled {
		t.Fatal("confused actor did not stumble")
	}
	g.Party.Selected = 1
	before := g.Party.Pos
	g.TryMove(DirS)
	want := Pos{before.X, before.Y + 1}
	if g.Party.Pos != want {
		t.Fatalf("clean member moved to %+v, want %+v", g.Party.Pos, want)
	}
	n := 0
	for _, line := range g.Log {
		if strings.Contains(line, "stumble") {
			// only the first move may stumble; count occurrences
			n++
		}
	}
	if n != 1 {
		t.Fatalf("stumble logged %d times, want 1", n)
	}
}

// Poison ticks on the poisoned member only.
func TestPoisonIsPerMember(t *testing.T) {
	g := memberGame()
	g.Party.Members[0].ApplyStatus(StatusPoison, 5)
	g.EndPlayerTurn("")
	if got := g.Party.Members[0].HP; got != 19 {
		t.Fatalf("poisoned member HP = %d, want 19", got)
	}
	if got := g.Party.Members[1].HP; got != 20 {
		t.Fatalf("clean member HP = %d, want 20", got)
	}
}

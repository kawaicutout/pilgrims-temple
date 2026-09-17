package game

import (
	"math/rand/v2"
	"strings"
	"testing"
)

// A killing blow that triggers a level-up must complete the same full
// world turn as any other kill: enemies still act. A prior special case
// ticked food without enemy actions, so the picker chose in a world that
// had half-advanced.
func TestLevelUpKillCompletesFullTurn(t *testing.T) {
	mk := func(xp int) *Game {
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
		hero := &Member{Name: "Hero", Class: "fighter", HP: 30, MaxHP: 30, ATK: [2]int{10, 10}, Light: 6, Alive: true}
		rat := &Member{Name: "Rat", Class: "rat", HP: 1, MaxHP: 1, ATK: [2]int{1, 1}, Alive: true}
		brute := &Member{Name: "Brute", Class: "orc", HP: 10, MaxHP: 10, ATK: [2]int{3, 3}, Alive: true}
		g := &Game{
			Seed:      1,
			RNG:       rand.New(rand.NewPCG(7, 7)),
			Levels:    []*Level{{W: w, H: h, Tiles: tiles, Seen: seen, Visible: vis, Enemies: []*EnemyParty{{Pos: Pos{2, 3}, Members: []*Member{rat}}, {Pos: Pos{2, 1}, Members: []*Member{brute}}}}},
			Floor:     0,
			Party:     &Party{Members: []*Member{hero}, Pos: Pos{2, 2}},
			XP:        xp,
			XPToNext:  100,
			Food:      100,
			FoodFloat: 100,
		}
		g.Tuning.Layout.LogLines = 8
		g.Tuning.LevelUp.TalentChance = 1
		return g
	}
	pending := mk(90) // 90 + 20 kill XP crosses 100
	clean := mk(0)
	pending.TryMove(DirS)
	clean.TryMove(DirS)
	if pending.LevelUpPending == nil {
		t.Logf("pending-twin log:")
		for _, line := range pending.Log {
			t.Logf("  %s", line)
		}
		t.Logf("clean-twin log:")
		for _, line := range clean.Log {
			t.Logf("  %s", line)
		}
		t.Fatal("expected level-up pending after killing blow")
	}
	if clean.LevelUpPending != nil {
		t.Fatal("clean twin must not level up")
	}
	acted := func(g *Game) bool {
		for _, line := range g.Log {
			if strings.Contains(line, "Orc hits") {
				return true
			}
		}
		return false
	}
	if !acted(clean) {
		t.Fatal("fixture broken: second enemy never acted")
	}
	if !acted(pending) {
		t.Fatal("enemies did not act on the level-up turn")
	}
	if pending.Turn != clean.Turn {
		t.Fatalf("turn diverged: pending %d clean %d", pending.Turn, clean.Turn)
	}
	if pending.Food != clean.Food {
		t.Fatalf("food diverged: pending %d clean %d", pending.Food, clean.Food)
	}
}

package main

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font/sfnt"
	"partyrogue/game"
)

// Host-side checks: fonts, colors, input mapping. No display needed.
func TestRendererUnits(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	if fonts.CellW != 10 || fonts.CellH != 18 {
		t.Fatalf("integer metrics = %v/%v", fonts.CellW, fonts.CellH)
	}
	if fonts.Face == nil {
		t.Fatal("face missing")
	}
	want := map[rune]float64{'█': 1, '▓': 0.75, '▒': 0.5, '░': 0.25}
	for r, d := range want {
		if got, ok := fillDensity(r); !ok || got != d {
			t.Fatalf("fill %q = %v,%v", r, got, ok)
		}
	}
	for _, r := range []rune{'g', '@', '#', '—', '·', ' ', '.'} {
		if _, ok := fillDensity(r); ok {
			t.Fatalf("%q misrouted to vector fill", r)
		}
	}
	cases := map[ebiten.Key]game.Key{
		ebiten.KeyArrowUp: game.KeyUp,
		ebiten.KeyNumpad5: game.KeyWait,
		ebiten.KeyEscape:  game.KeyQuit,
		ebiten.KeyEnter:   game.KeyEnter,
	}
	for ek, want := range cases {
		var found bool
		for _, sk := range specialKeys {
			if sk.eb == ek {
				if normalizeEvent(keyEvent{key: sk.key, code: sk.code}) != want {
					t.Fatalf("key %v resolves wrong", ek)
				}
				found = true
			}
		}
		if !found {
			t.Fatalf("key %v unmapped", ek)
		}
	}
	// Movement keys repeat; Enter does not.
	if !repeatKeys[game.KeyUp] || repeatKeys[game.KeyEnter] {
		t.Fatal("repeat set wrong")
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := string(truncateRunes([]rune("hello"), -1)); got != "" {
		t.Fatalf("negative width = %q", got)
	}
	if got := string(truncateRunes([]rune("hello"), 0)); got != "" {
		t.Fatalf("zero width = %q", got)
	}
	if got := string(truncateRunes([]rune("hello"), 1)); got != "h" {
		t.Fatalf("width 1 = %q", got)
	}
	if got := string(truncateRunes([]rune("hello"), 4)); got != "hel…" {
		t.Fatalf("width 4 = %q", got)
	}
	if got := string(truncateRunes([]rune("hi"), 10)); got != "hi" {
		t.Fatalf("wide = %q", got)
	}
}

func TestGlyphCoverage(t *testing.T) {
	raw, err := os.ReadFile("fonts/libertinus-mono.ttf")
	if err != nil {
		t.Fatal(err)
	}
	f, err := sfnt.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	var buf sfnt.Buffer
	has := func(r rune) bool {
		idx, err := f.GlyphIndex(&buf, r)
		return err == nil && idx != 0
	}
	// Every non-ASCII rune in game data renders through this font unless
	// it has a vector fill. A miss here is tofu on screen.
	entries, err := os.ReadDir("../../game/data")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := os.ReadFile("../../game/data/" + e.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range string(b) {
			if r < 128 {
				continue
			}
			if _, ok := fillDensity(r); ok {
				continue
			}
			if !has(r) {
				t.Errorf("%s: %q (U+%04X) has no fill and no glyph", e.Name(), r, r)
			}
		}
	}
	// UI punctuation from code strings.
	for _, r := range []rune{'—'} {
		if !has(r) {
			t.Errorf("UI %q (U+%04X) has no glyph", r, r)
		}
	}
}

func TestFitSize(t *testing.T) {
	advR, emR := 0.6, 1.125 // Libertinus-class ratios, not asserts
	if got := fitSize(110*10, 34*18, 110, 34, advR, emR); got < 15.9 || got > 16.7 {
		t.Fatalf("base window fits ~16, got %v", got)
	}
	if got := fitSize(8000, 6000, 110, 34, advR, emR); got != maxGlyphSize {
		t.Fatalf("huge window unclamped: %v", got)
	}
	if got := fitSize(200, 100, 110, 34, advR, emR); got != minGlyphSize {
		t.Fatalf("tiny window unclamped: %v", got)
	}
	if got := fitSize(0, 600, 110, 34, advR, emR); got != baseFontSize {
		t.Fatalf("zero width = %v", got)
	}
	if got := fitSize(800, 600, 110, 34, 0, emR); got != baseFontSize {
		t.Fatalf("zero ratio = %v", got)
	}
}

func TestLayoutBox(t *testing.T) {
	// Web factor on a normal window.
	if w, h := layoutBox(1568, 827, 0.8); w != 1254 || h != 662 {
		t.Fatalf("80%% box = %dx%d", w, h)
	}
	// Truncation must never reach zero: Ebiten panics otherwise.
	if w, h := layoutBox(1, 1, 0.8); w < 1 || h < 1 {
		t.Fatalf("tiny box = %dx%d", w, h)
	}
	if w, h := layoutBox(0, 0, 0.8); w != 1 || h != 1 {
		t.Fatalf("zero box = %dx%d", w, h)
	}
	if w, h := layoutBox(1100, 612, 1); w != 1100 || h != 612 {
		t.Fatalf("desktop passthrough = %dx%d", w, h)
	}
}

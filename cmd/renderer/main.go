package main

import (
	"log"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"partyrogue/game"
)

// Glyph-size bounds for dynamic refits. fitSize clamps into this range;
// applySize additionally measure-verifies the grid against the window.
const (
	minGlyphSize = 8
	maxGlyphSize = 96
)

// appState mirrors cmd/terminal; stateOver replaces the blocking wait loops.
type appState int

const (
	stateMenu appState = iota
	stateMenuHelp
	stateScores
	stateSeed
	stateCreationClass
	stateCreationRace
	stateCreationName
	stateReview
	statePlaying
	stateUseInventory
	stateUseMember
	stateUseTarget
	stateThrowMenu
	stateThrowCursor
	stateMerchant
	stateShrine
	stateWizard
	stateWizardAddMember
	stateWizardRemoveMember
	stateWizardResurrectMember
	stateOver
)

// App is the Ebiten game: turn-based core, re-render on keypress only.
type App struct {
	tuning              game.Tuning
	fonts               *GridFonts
	cache               *FrameCache
	state               appState
	menu                *game.MainMenuState
	scoresCursor        int
	seedState           *game.SeedEntryState
	classPick           *game.ClassPickState
	racePick            *game.RacePickState
	slotRace            string
	slotClass           string
	nameEntry           *game.NameEntryState
	drafts              []game.DraftMember
	reviewCursor        int
	g                   *game.Game
	quit                bool
	pendingSize         float64
	winW, winH          int
	last                game.Frame
	wizardState         *game.WizardState
	wizardAddCS         *game.CharSelectState
	wizardRemoveIdx     int
	useSelected         int
	useMemberSelected   int
	useMemberAppearance string
	useMemberTitle      string
	useMemberMode       string
	throwSelected       int
}

func newApp(tuning game.Tuning, fonts *GridFonts) *App {
	cols, rows := tuning.Layout.MinCols, tuning.Layout.MinRows
	return &App{
		tuning:      tuning,
		fonts:       fonts,
		cache:       NewFrameCache(fonts, cols, rows),
		menu:        &game.MainMenuState{},
		seedState:   &game.SeedEntryState{},
		wizardState: &game.WizardState{},
	}
}

// show renders one frame into the cache, retaining it for repaints after
// a glyph-size refit.
func (a *App) show(f game.Frame) {
	a.last = f
	a.cache.Render(f)
}

// toMenu saves a live run and returns to the menu.
func (a *App) toMenu() {
	if a.g != nil && !a.g.Over {
		_ = game.Save(a.g)
	}
	a.g = nil
	a.state = stateMenu
	a.show(game.RenderMainMenu(a.tuning, a.menu.Selected))
}

// overToMenu deletes the finished save and returns to the menu.
func (a *App) overToMenu() {
	_ = game.DeleteSave()
	a.g = nil
	a.state = stateMenu
	a.show(game.RenderMainMenu(a.tuning, a.menu.Selected))
}
func (a *App) Update() error {
	if a.quit {
		return errQuit
	}
	if a.pendingSize > 0 {
		a.applySize(a.pendingSize)
		a.pendingSize = 0
	}
	evs, runes := pollKeys()
	evs = append(evs, heldRepeat()...)
	for _, r := range runes {
		evs = append(evs, keyEvent{key: string(r)})
	}
	for _, ev := range evs {
		a.handleKey(ev)
		if a.quit {
			return errQuit
		}
	}
	return nil
}

// applySize rebuilds faces at size and repaints the current frame. Both
// fills and glyphs scale: fills are vector geometry on the new cell pitch,
// glyphs re-rasterize natively. The candidate is measure-verified against
// the window, corrected once, and falls back to base size — an oversize
// grid cannot survive this function.
func (a *App) applySize(size float64) {
	cols, rows := a.tuning.Layout.MinCols, a.tuning.Layout.MinRows
	if f, err := sizedFonts(size); err == nil && a.fits(f) {
		a.installSize(f, cols, rows)
		return
	} else if err == nil {
		if k := a.fitRatio(f); k > 0 {
			if f2, err := sizedFonts(math.Max(minGlyphSize, size*k*0.99)); err == nil && a.fits(f2) {
				a.installSize(f2, cols, rows)
				return
			}
		}
	}
	if fb, err := sizedFonts(baseFontSize); err == nil {
		a.installSize(fb, cols, rows)
	}
}

// fits reports whether the grid at f's cells fits the last seen window.
// Unknown window (zero) counts as fitting: nothing to verify against.
func (a *App) fits(f *GridFonts) bool {
	if a.winW <= 0 || a.winH <= 0 {
		return true
	}
	cols, rows := a.tuning.Layout.MinCols, a.tuning.Layout.MinRows
	return float64(cols)*f.CellW <= float64(a.winW) && float64(rows)*f.CellH <= float64(a.winH)
}

// installSize swaps in rebuilt faces, a matching cache, and repaints. The
// old cache image is disposed: without this every resize refit leaks a
// full-grid GPU texture until GC, which ANGLE reports as oversize
// allocation churn.
func (a *App) installSize(f *GridFonts, cols, rows int) {
	a.fonts = f
	if a.cache != nil {
		a.cache.Image().Dispose()
	}
	a.cache = NewFrameCache(f, cols, rows)
	a.show(a.last)
}

// fitRatio returns the shrink factor that would fit f's grid, or -1 when
// the window is unknown.
func (a *App) fitRatio(f *GridFonts) float64 {
	if a.winW <= 0 || a.winH <= 0 {
		return -1
	}
	cols, rows := a.tuning.Layout.MinCols, a.tuning.Layout.MinRows
	gw, gh := float64(cols)*f.CellW, float64(rows)*f.CellH
	if gw <= 0 || gh <= 0 {
		return -1
	}
	return math.Min(float64(a.winW)/gw, float64(a.winH)/gh)
}

// fitSize picks the glyph size fitting ow×oh window pixels around a
// cols×rows grid. Ratios come from measured faces; the result is clamped
// so layout math can never request an absurd size.
func fitSize(ow, oh, cols, rows int, advR, emR float64) float64 {
	if ow <= 0 || oh <= 0 || cols <= 0 || rows <= 0 || advR <= 0 || emR <= 0 {
		return baseFontSize
	}
	size := math.Min(float64(ow)/(float64(cols)*advR), float64(oh)/(float64(rows)*emR))
	return math.Min(maxGlyphSize, math.Max(minGlyphSize, size))
}

// Draw blits the cached frame centered; rounding slack between the grid
// and the window shows as background margin, never stretch.
func (a *App) Draw(screen *ebiten.Image) {
	screen.Fill(tokenColor("bg"))
	img := a.cache.Image()
	b := img.Bounds()
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	if sw <= 0 || sh <= 0 {
		sw, sh = b.Dx(), b.Dy()
	}
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(float64(sw-b.Dx())/2, float64(sh-b.Dy())/2)
	screen.DrawImage(img, &op)
}

// layoutBox scales the window by the platform fit factor and guarantees a
// positive box. Truncation must never yield 0: Ebiten panics on
// non-positive Layout sizes (seen live at tiny/zero window sizes).
func layoutBox(outsideW, outsideH int, scale float64) (int, int) {
	w, h := outsideW, outsideH
	if scale < 1 && w > 0 && h > 0 {
		w, h = int(float64(w)*scale+0.5), int(float64(h)*scale+0.5)
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

// Layout records the window size and queues a glyph-size refit when the
// window would fit a different size. The grid targets viewportFit() of the
// window (full size on desktop, 80% on web); both fills and glyphs
// re-resolve at the new size, nothing is ever bitmap-scaled. Degenerate
// boxes (minimized/docked windows) keep the current faces: no refit storm.
func (a *App) Layout(outsideW, outsideH int) (int, int) {
	w, h := layoutBox(outsideW, outsideH, viewportFit())
	a.winW, a.winH = w, h
	if a.fonts != nil && w >= 64 && h >= 64 {
		cols, rows := a.tuning.Layout.MinCols, a.tuning.Layout.MinRows
		size := fitSize(w, h, cols, rows, a.fonts.AdvR, a.fonts.EmR)
		if d := size - a.fonts.Size; d > 0.75 || d < -0.75 {
			a.pendingSize = size
		}
	}
	return w, h
}

func main() {
	tuning, err := game.LoadTuning()
	if err != nil {
		log.Fatal(err)
	}
	fonts, err := loadFonts()
	if err != nil {
		log.Fatal(err)
	}
	app := newApp(tuning, fonts)
	app.show(game.RenderMainMenu(tuning, app.menu.Selected))
	armSaveHook(func() *game.Game { return app.g })
	setupShell()
	ebiten.SetWindowTitle("Pilgrims' Temple")
	ebiten.SetWindowSize(app.cache.Image().Bounds().Dx(), app.cache.Image().Bounds().Dy())
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(app); err != nil && err != errQuit {
		log.Fatal(err)
	}
}

// seedNow resolves blank seeds to wall-clock randomness.
func seedNow() int64 {
	return time.Now().UnixNano()
}

package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"partyrogue/game"
)

// FrameCache holds the last rendered frame as an offscreen image.
// Turn-based game: re-render only on state change; Draw blits one image.
type FrameCache struct {
	fonts *GridFonts
	img   *ebiten.Image
	cols  int
	rows  int
}

// NewFrameCache sizes the cache for cols x rows cells.
func NewFrameCache(fonts *GridFonts, cols, rows int) *FrameCache {
	w := int(fonts.CellW*float64(cols)) + 1
	h := int(fonts.CellH*float64(rows)) + 1
	return &FrameCache{fonts: fonts, img: ebiten.NewImage(w, h), cols: cols, rows: rows}
}

// drawText paints a string at cell coordinates in the single grid face.
// Block/shade runes bypass the rasterizer (drawFill); x advances by
// measured width for text, full cells for fills.
func (c *FrameCache) drawText(s string, col, row int, clr color.Color) {
	if s == "" {
		return
	}
	rs := []rune(s)
	x := float64(col) * c.fonts.CellW
	y := float64(row) * c.fonts.CellH
	i := 0
	for i < len(rs) {
		if d, ok := fillDensity(rs[i]); ok {
			c.drawFill(x, y, clr, d)
			x += c.fonts.CellW
			i++
			continue
		}
		j := i + 1
		for j < len(rs) {
			if _, ok := fillDensity(rs[j]); ok {
				break
			}
			j++
		}
		seg := string(rs[i:j])
		c.drawRun(seg, x, y, clr)
		w, _ := text.Measure(seg, c.fonts.Face, 0)
		x += w
		i = j
	}
}

// drawRun paints one text run at pixel coordinates.
func (c *FrameCache) drawRun(s string, x, y float64, clr color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	r, g, b, a := clr.RGBA()
	var cs ebiten.ColorScale
	cs.SetR(float32(r) / 0xffff)
	cs.SetG(float32(g) / 0xffff)
	cs.SetB(float32(b) / 0xffff)
	cs.SetA(float32(a) / 0xffff)
	op.ColorScale.ScaleWithColorScale(cs)
	text.Draw(c.img, s, c.fonts.Face, op)
}

// fillDensity maps block/shade runes to coverage. This table is the
// configuration point: a rune listed here renders as vector geometry
// instead of font ink. Coverage becomes the box alpha, so partial shades
// read as the foreground color at reduced strength.
func fillDensity(r rune) (float64, bool) {
	switch r {
	case '█':
		return 1, true
	case '▓':
		return 0.75, true
	case '▒':
		return 0.5, true
	case '░':
		return 0.25, true
	}
	return 0, false
}

// drawFill paints one cell as a full vector box: solid at full coverage,
// alpha-shaded otherwise. The shade keeps the foreground hue and lets the
// background through, so partial shades work on any background with one
// rect, no dither geometry, no font ink.
func (c *FrameCache) drawFill(x, y float64, clr color.Color, density float64) {
	r, g, b, _ := clr.RGBA()
	fill := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(density * 0xff)}
	vector.DrawFilledRect(c.img, float32(x), float32(y), float32(c.fonts.CellW), float32(c.fonts.CellH), fill, false)
}

// drawCell paints one map cell: optional cursor block, then the glyph.
// Block/shade runes draw as vector fills, everything else as text.
func (c *FrameCache) drawCell(x, y int, ch rune, fg, bg string) {
	px, py := float64(x)*c.fonts.CellW, float64(y)*c.fonts.CellH
	if bg == "cursor" {
		vector.DrawFilledRect(c.img,
			float32(px), float32(py),
			float32(c.fonts.CellW), float32(c.fonts.CellH),
			tokenColor("gold-bright"), false)
		if d, ok := fillDensity(ch); ok {
			c.drawFill(px, py, tokenColor("bg"), d)
		} else if ch != ' ' && ch != 0 {
			c.drawRun(string(ch), px, py, tokenColor("bg"))
		}
		return
	}
	if ch == ' ' || ch == 0 {
		return
	}
	if d, ok := fillDensity(ch); ok {
		c.drawFill(px, py, tokenColor(fg), d)
		return
	}
	c.drawRun(string(ch), px, py, tokenColor(fg))
}

// Render repaints the whole frame: map cells, side panel, status, log, hints.
// Mirrors the terminal composition in cmd/terminal drawFrame.
func (c *FrameCache) Render(f game.Frame) {
	bg := tokenColor("bg")
	c.img.Fill(bg)
	// Map cells.
	for y := 0; y < f.H && y < c.rows; y++ {
		for x := 0; x < f.W && x < c.cols; x++ {
			cell := f.Cells[y][x]
			c.drawCell(x, y, cell.Glyph, cell.FG, cell.BG)
		}
	}
	// Side panel right of the map.
	panelX := f.W + 1
	for i, line := range f.Panel {
		if i >= f.H {
			break
		}
		maxLen := c.cols - panelX
		if maxLen <= 0 {
			break
		}
		fg := "gray-1"
		if i < len(f.PanelFG) && f.PanelFG[i] != "" {
			fg = f.PanelFG[i]
		} else if len(line) > 0 && line[0] == '>' {

			fg = "gold-bright"
		}
		runes := []rune(line)
		runes = truncateRunes(runes, maxLen)
		c.drawText(string(runes), panelX, i, tokenColor(fg))
	}
	// Status, log, hints below the map.
	c.drawText(f.Status, 0, f.H, tokenColor("gold"))
	for i, line := range f.Log {
		c.drawText(line, 0, f.H+1+i, tokenColor("gray-1"))
	}
	if c.rows-1 > f.H+1+len(f.Log) {
		c.drawText(f.Hints, 0, c.rows-1, tokenColor("gray-2"))
	}
}

// truncateRunes fits a panel line into maxLen columns, never panicking on
// narrow or negative widths (menu frames have no side room).
func truncateRunes(runes []rune, maxLen int) []rune {
	if len(runes) > maxLen {
		if maxLen > 1 {
			return append(runes[:maxLen-1], '…')
		} else if maxLen == 1 {
			return runes[:1]
		}
		return runes[:0]
	}
	return runes
}

// Image returns the cached frame.
func (c *FrameCache) Image() *ebiten.Image {
	return c.img
}

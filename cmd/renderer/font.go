package main

import (
	"bytes"
	"embed"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

//go:embed fonts/*.ttf
var fontFS embed.FS

// GridFonts holds the text faces and integer cell metrics. Fractional
// origins snap rows and cause seams; advances are measured per size,
// glyphs re-rasterized, never scaled.
type GridFonts struct {
	Face  text.Face
	Size  float64
	CellW float64
	CellH float64
	// AdvR/EmR are the measured advance and line box in ems. Layout
	// divides window pixels by them to pick a size; every rebuild
	// remeasures, so stale ratios cannot persist.
	AdvR float64
	EmR  float64
}

const baseFontSize = 16

var fontLibSrc *text.GoTextFaceSource

// loadFonts embeds Libertinus Mono, the only grid face. Block and shade
// runes never reach the rasterizer (vector fills in draw.go), so no
// fallback face is needed.
func loadFonts() (*GridFonts, error) {
	libBytes, err := fontFS.ReadFile("fonts/libertinus-mono.ttf")
	if err != nil {
		return nil, err
	}
	fontLibSrc, err = text.NewGoTextFaceSource(bytes.NewReader(libBytes))
	if err != nil {
		return nil, err
	}
	return sizedFonts(baseFontSize)
}

// sizedFonts builds the Libertinus face and integer cell metrics at size.
// Cell pitch rounds the measured advance and line box; fills tile on this
// pitch as exact vector geometry.
func sizedFonts(size float64) (*GridFonts, error) {
	face := &text.GoTextFace{Source: fontLibSrc, Size: size}
	w, _ := text.Measure(strings.Repeat("0", 16), face, 0)
	m := face.Metrics()
	h := m.HAscent + m.HDescent
	return &GridFonts{
		Face:  face,
		Size:  size,
		CellW: math.Round(w / 16),
		CellH: math.Round(h),
		AdvR:  (w / 16) / size,
		EmR:   h / size,
	}, nil
}

// tokenColor maps game FG tokens and #hex to colors, mirroring the terminal.
func tokenColor(tok string) color.Color {
	switch tok {
	case "", "bg":
		return color.RGBA{0x14, 0x12, 0x10, 0xff}
	case "gold":
		return color.RGBA{0xb8, 0x97, 0x5a, 0xff}
	case "gold-bright", "player":
		return color.RGBA{0xd3, 0xad, 0x6b, 0xff}
	case "red-bright", "enemy":
		return color.RGBA{0xc9, 0x6a, 0x5a, 0xff}
	case "red":
		return color.RGBA{0xa8, 0x56, 0x4a, 0xff}
	case "slate":
		return color.RGBA{0x6e, 0x8f, 0xb5, 0xff}
	case "wall":
		return color.RGBA{0x6b, 0x64, 0x5c, 0xff}
	case "floor":
		return color.RGBA{0x4a, 0x46, 0x42, 0xff}
	case "gray-1":
		return color.RGBA{0xb5, 0xae, 0xa5, 0xff}
	case "gray-2", "gray-3":
		return color.RGBA{0x8a, 0x85, 0x7e, 0xff}
	case "gray-4":
		return color.RGBA{0x2c, 0x29, 0x27, 0xff}
	case "fg":
		return color.RGBA{0xe6, 0xe0, 0xd8, 0xff}
	}
	if strings.HasPrefix(tok, "#") {
		if c, ok := parseHexColor(tok); ok {
			return c
		}
		return color.RGBA{0xb5, 0xae, 0xa5, 0xff}
	}
	return color.RGBA{0xb5, 0xae, 0xa5, 0xff}
}

// parseHexColor decodes #rrggbb.
func parseHexColor(s string) (color.Color, bool) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return nil, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return nil, false
	}
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}, true
}

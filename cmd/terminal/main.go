package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gdamore/tcell/v2"
	"partyrogue/game"
)

func main() {
	tuning, err := game.LoadTuning()
	if err != nil {
		log.Fatal(err)
	}
	seed := time.Now().UnixNano()
	g := game.NewGame(seed, tuning)

	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatal(err)
	}
	if err := s.Init(); err != nil {
		log.Fatal(err)
	}
	defer s.Fini()
	// Mouse not needed for roguelike

	bg := tcell.NewRGBColor(20, 18, 16)
	fg := tcell.NewRGBColor(230, 224, 216)
	gold := tcell.NewRGBColor(184, 151, 90)
	goldBr := tcell.NewRGBColor(211, 173, 107)
	redBr := tcell.NewRGBColor(201, 106, 90)
	wallCol := tcell.NewRGBColor(107, 100, 92)
	floorCol := tcell.NewRGBColor(74, 70, 66)
	gray1 := tcell.NewRGBColor(181, 174, 165)
	gray2 := tcell.NewRGBColor(138, 133, 126)

	styleBG := tcell.StyleDefault.Foreground(fg).Background(bg)
	styleGold := styleBG.Foreground(gold)
	styleGoldBr := styleBG.Foreground(goldBr)
	styleRedBr := styleBG.Foreground(redBr)
	styleWall := styleBG.Foreground(wallCol)
	styleFloor := styleBG.Foreground(floorCol)
	styleGray1 := styleBG.Foreground(gray1)
	styleGray2 := styleBG.Foreground(gray2)
	_ = styleGold

	draw := func() {
		w, h := s.Size()
		minW := g.Tuning.Layout.MinCols
		minH := g.Tuning.Layout.MinRows
		for y := range h {
			for x := range w {
				s.SetContent(x, y, ' ', nil, styleBG)
			}
		}
		if w < minW || h < minH {
			msg := fmt.Sprintf(" Pilgrim's Temple — resize to %dx%d (you have %dx%d) ", minW, minH, w, h)
			for i, ch := range msg {
				if i < w {
					s.SetContent(i, 0, ch, nil, styleGoldBr)
				}
			}
			hint := " Enlarge terminal, or run ./run.sh which requests 110x34. "
			for i, ch := range hint {
				if i < w && 1 < h {
					s.SetContent(i, 1, ch, nil, styleGray2)
				}
			}
			s.Show()
			return
		}
		frame := g.Render()
		for y := range frame.H {
			for x := range frame.W {
				if y >= h || x >= w {
					continue
				}
				cell := frame.Cells[y][x]
				var st tcell.Style
				switch cell.FG {
				case "player":
					st = styleGoldBr
				case "enemy":
					st = styleRedBr
				case "wall":
					st = styleWall
				case "floor":
					st = styleFloor
				case "gold":
					st = styleGold
				case "gray-3":
					st = styleGray2
				default:
					st = styleBG
				}
				s.SetContent(x, y, cell.Glyph, nil, st)
			}
		}
		panelX := frame.W + 1
		if panelX < w {
			for i, line := range frame.Panel {
				y := i
				if y >= frame.H {
					break
				}
				maxLen := w - panelX
				if maxLen <= 0 {
					break
				}
				runes := []rune(line)
				if len(runes) > maxLen {
					if maxLen > 1 {
						runes = runes[:maxLen-1]
						runes = append(runes, '…')
					} else {
						runes = runes[:maxLen]
					}
				}
				style := styleGray1
				if len(line) > 0 && line[0] == '>' {
					style = styleGoldBr
				}
				for j, ch := range runes {
					s.SetContent(panelX+j, y, ch, nil, style)
				}
			}
		}
		statusY := frame.H
		if statusY < h {
			for i, ch := range frame.Status {
				if i >= w {
					break
				}
				s.SetContent(i, statusY, ch, nil, styleGold)
			}
		}
		logY := frame.H + 1
		for i, line := range frame.Log {
			y := logY + i
			if y >= h {
				break
			}
			for j, ch := range line {
				if j >= w {
					break
				}
				s.SetContent(j, y, ch, nil, styleGray1)
			}
		}
		hintY := h - 1
		if hintY > logY+len(frame.Log) {
			for i, ch := range frame.Hints {
				if i >= w {
					break
				}
				s.SetContent(i, hintY, ch, nil, styleGray2)
			}
		}
		s.Show()
	}

	draw()
	for {
		ev := s.PollEvent()
		switch e := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
			draw()
		case *tcell.EventKey:
			key, code := tcellKeyToRaw(e)
			k := game.NormalizeKey(key, code)
			if k == game.KeyQuit {
				return
			}
			g.HandleKey(k)
			draw()
			if g.Over {
				// Wait for quit after game over
				for {
					ev2 := s.PollEvent()
					if ke, ok := ev2.(*tcell.EventKey); ok {
						key2, code2 := tcellKeyToRaw(ke)
						k2 := game.NormalizeKey(key2, code2)
						if k2 == game.KeyQuit {
							return
						}
					} else if _, ok := ev2.(*tcell.EventResize); ok {
						s.Sync()
						draw()
					}
				}
			}
		}
	}
}

func tcellKeyToRaw(e *tcell.EventKey) (key string, code string) {
	switch e.Key() {
	case tcell.KeyUp:
		return "ArrowUp", "ArrowUp"
	case tcell.KeyDown:
		return "ArrowDown", "ArrowDown"
	case tcell.KeyLeft:
		return "ArrowLeft", "ArrowLeft"
	case tcell.KeyRight:
		return "ArrowRight", "ArrowRight"
	case tcell.KeyEscape:
		return "Escape", "Escape"
	case tcell.KeyEnter:
		return "Enter", "Enter"
	}
	if e.Key() == tcell.KeyRune {
		return string(e.Rune()), ""
	}
	return "", ""
}

package game

type Key string

const (
	KeyUp          Key = "up"
	KeyDown        Key = "down"
	KeyLeft        Key = "left"
	KeyRight       Key = "right"
	KeyUpLeft      Key = "upleft"
	KeyUpRight     Key = "upright"
	KeyDownLeft    Key = "downleft"
	KeyDownRight   Key = "downright"
	KeyWait        Key = "wait"
	KeyStairsDown  Key = "stairs_down"
	KeyStairsUp    Key = "stairs_up"
	KeyQuit        Key = "quit"
	KeyHelp        Key = "help"
	KeySelect1     Key = "select1"
	KeySelect2     Key = "select2"
	KeySelect3     Key = "select3"
	KeySelect4     Key = "select4"
)

func KeyToDir(k Key) (Dir, bool) {
	switch k {
	case KeyUp:
		return DirN, true
	case KeyDown:
		return DirS, true
	case KeyLeft:
		return DirW, true
	case KeyRight:
		return DirE, true
	case KeyUpLeft:
		return DirNW, true
	case KeyUpRight:
		return DirNE, true
	case KeyDownLeft:
		return DirSW, true
	case KeyDownRight:
		return DirSE, true
	case KeyWait:
		return DirNone, true
	default:
		return DirNone, false
	}
}

func (g *Game) HandleKey(k Key) bool {
	switch k {
	case KeySelect1, KeySelect2, KeySelect3, KeySelect4:
		idx := int(k[len(k)-1] - '1')
		if idx >= 0 && idx < len(g.Party.Members) && g.Party.Members[idx].IsAlive() {
			g.Party.Selected = idx
		}
		return false
	case KeyHelp:
		g.Logf("Seed %d | Help: q/w/e/r select, move numpad/arrows/hjkl, 5/. wait, >/ < stairs, Esc quit.", g.Seed)
		g.Logf("Food %d (%s) | Turn %d | Floor %d/%d", g.Food, g.HungerState(), g.Turn, g.Floor+1, g.Tuning.Floors)
		return false
	case KeyQuit:
		g.Quit = true
		g.Logf("Quit to menu. Seed %d saved.", g.Seed)
		return false
	case KeyStairsDown:
		g.TryStairsDown()
		return true
	case KeyStairsUp:
		g.TryStairsUp()
		return true
	default:
		if dir, ok := KeyToDir(k); ok {
			g.TryMove(dir)
			return true
		}
	}
	return false
}

func NormalizeKey(raw string, code string) Key {
	switch code {
	case "Numpad8", "Digit8", "ArrowUp":
		return KeyUp
	case "Numpad2", "Digit2", "ArrowDown":
		return KeyDown
	case "Numpad4", "Digit4", "ArrowLeft":
		return KeyLeft
	case "Numpad6", "Digit6", "ArrowRight":
		return KeyRight
	case "Numpad7", "Digit7":
		return KeyUpLeft
	case "Numpad9", "Digit9":
		return KeyUpRight
	case "Numpad1", "Digit1":
		return KeyDownLeft
	case "Numpad3", "Digit3":
		return KeyDownRight
	case "Numpad5", "Digit5":
		return KeyWait
	case "Escape":
		return KeyQuit
	}
	switch raw {
	case "8", "k", "K", "ArrowUp":
		return KeyUp
	case "2", "j", "J", "ArrowDown":
		return KeyDown
	case "4", "h", "H", "ArrowLeft":
		return KeyLeft
	case "6", "l", "L", "ArrowRight":
		return KeyRight
	case "7", "y", "Y":
		return KeyUpLeft
	case "9", "u", "U":
		return KeyUpRight
	case "1", "b", "B":
		return KeyDownLeft
	case "3", "n", "N":
		return KeyDownRight
	case "5", ".", " ", "Space":
		return KeyWait
	case ">":
		return KeyStairsDown
	case "<":
		return KeyStairsUp
	case "q":
		return KeySelect1
	case "Q":
		return KeyQuit
	case "w", "W":
		return KeySelect2
	case "e", "E":
		return KeySelect3
	case "r", "R":
		return KeySelect4
	case "?":
		return KeyHelp
	case "Escape":
		return KeyQuit
	}
	return Key(raw)
}

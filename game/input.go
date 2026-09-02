package game

type Key string

const (
	KeyUp         Key = "up"
	KeyDown       Key = "down"
	KeyLeft       Key = "left"
	KeyRight      Key = "right"
	KeyUpLeft     Key = "upleft"
	KeyUpRight    Key = "upright"
	KeyDownLeft   Key = "downleft"
	KeyDownRight  Key = "downright"
	KeyWait       Key = "wait"
	KeyRest       Key = "rest"
	KeyStairsDown Key = "stairs_down"
	KeyStairsUp   Key = "stairs_up"
	KeyQuit       Key = "quit"
	KeyHelp       Key = "help"
	KeyEnter      Key = "enter"
	KeyLook       Key = "look"
	KeyPickup     Key = "pickup"
	KeyUse        Key = "use"
	KeySelect1    Key = "select1"
	KeySelect2    Key = "select2"
	KeySelect3    Key = "select3"
	KeySelect4    Key = "select4"
	KeyThrow      Key = "throw"
	KeyWizard     Key = "wizard"
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
	// Help can be invoked even in look mode.
	if k == KeyHelp {
		g.HelpActive = true
		return false
	}
	// Look mode has priority: when active, movement moves cursor, v/Enter/Esc exits and examines
	if g.Look != nil && g.Look.Active {
		switch k {
		case KeyLook, KeyEnter, KeyQuit:
			// Examine current cursor tile before exiting
			desc := Examine(g, g.Look.Cursor)
			g.Logf("%s", desc)
			g.Look.Active = false
			return false
		default:
			if dir, ok := KeyToDir(k); ok {
				next := g.Look.Cursor.Add(dir)
				if g.CurLevel().InBounds(next) {
					g.Look.Cursor = next
					desc := Examine(g, next)
					g.Logf("%s", desc)
				}
				return false
			}
			// Other keys (help, select) still work in look mode? For now ignore
			return false
		}
	}
	// Throw mode has next priority: when pending, movement moves cursor, Enter confirms, Esc cancels.
	if g.ThrowPending.Active {
		switch k {
		case KeyQuit:
			g.CancelThrow()
			g.Logf("Cancelled throw.")
			return false
		case KeyEnter:
			// Confirm throw at cursor Pos via ThrowAt (which calls TryThrowAppearance)
			g.ThrowAt(g.ThrowPending.Cursor)
			return true
		default:
			if dir, ok := KeyToDir(k); ok {
				if dir == DirNone {
					g.CancelThrow()
					g.Logf("Cancelled throw.")
					return false
				}
				next := g.ThrowPending.Cursor.Add(dir)
				lvl := g.CurLevel()
				if lvl == nil || !lvl.InBounds(next) {
					return false
				}
				// Chebyshev distance 5 from party pos
				dx := next.X - g.Party.Pos.X
				if dx < 0 {
					dx = -dx
				}
				dy := next.Y - g.Party.Pos.Y
				if dy < 0 {
					dy = -dy
				}
				if dx > 5 || dy > 5 {
					return false
				}
				// Must be within FOV Visible, clamped to level bounds (already checked)
				if next.Y < 0 || next.Y >= len(lvl.Visible) || next.X < 0 || next.X >= len(lvl.Visible[0]) {
					return false
				}
				if !lvl.Visible[next.Y][next.X] {
					return false
				}
				g.ThrowPending.Cursor = next
				return false
			}
			return false
		}
	}
	switch k {
	case KeySelect1, KeySelect2, KeySelect3, KeySelect4:
		idx := int(k[len(k)-1] - '1')
		if idx >= 0 && idx < len(g.Party.Members) && g.Party.Members[idx].IsAlive() {
			g.Party.Selected = idx
		}
		return false
	case KeyHelp:
		g.HelpActive = true
		return false
	case KeyLook:
		g.Look = &LookState{Cursor: g.Party.Pos, Active: true}
		g.Logf("Look mode: use hjkl/arrows to move, v/Enter/Esc to examine and exit.")
		desc := Examine(g, g.Look.Cursor)
		g.Logf("%s", desc)
		return false
	case KeyRest:
		RestBatch(g)
		return true
	case KeyPickup:
		if g.TryCloseDoor() {
			g.EndPlayerTurn("")
			return true
		}
		if g.TryUseFountain() {
			g.EndPlayerTurn("")
			return true
		}
		if g.TryUseMerchant() {
			return false
		}
		if g.TryUseForge() {
			g.EndPlayerTurn("")
			return true
		}
		if g.TryUseShrine() {
			return false
		}
		g.TryPickup()
		g.EndPlayerTurn("")
		return true
	case KeyUse:
		if g.TryUseForge() {
			g.EndPlayerTurn("")
			return true
		}
		if g.Party == nil || len(g.Party.Inventory) == 0 {
			g.Logf("No potions or scrolls to use.")
			return false
		}
		// Do not auto-consume; frontend opens usage menu (select via InventoryUseEntries + TryUseAppearance).
		// Keep TryUseItem as fallback but not auto-called on u/U.
		return false
	case KeyThrow:
		hasPotion := false
		if g.Party != nil {
			for _, it := range g.Party.Inventory {
				if it.Kind == "potion" {
					hasPotion = true
					break
				}
			}
		}
		if !hasPotion {
			g.Logf("No potions to throw.")
			return false
		}
		// Do not auto-pick first potion; frontend opens throw menu and calls StartThrow(appearance).
		g.Logf("Throw potion: select potion to throw.")
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
	case "KeyZ":
		return KeyRest
	case "KeyT":
		return KeyThrow
	case "Escape":
		return KeyQuit
	case "Enter":
		return KeyEnter
	case "BracketRight":
		return KeyWizard
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
	case "9":
		return KeyUpRight
	case "u", "U":
		return KeyUse
	case "1", "b", "B":
		return KeyDownLeft
	case "3", "n", "N":
		return KeyDownRight
	case "5", ".", " ", "Space":
		return KeyWait
	case "z", "Z":
		return KeyRest
	case ">":
		return KeyStairsDown
	case "<":
		return KeyStairsUp
	case "Enter":
		return KeyEnter
	case "q", "Q":
		return KeySelect1
	case "w", "W":
		return KeySelect2
	case "e", "E":
		return KeySelect3
	case "r", "R":
		return KeySelect4
	case "v", "V":
		return KeyLook
	case "g", "G":
		return KeyPickup
	case "t", "T":
		return KeyThrow
	case "?":
		return KeyHelp
	case "Escape":
		return KeyQuit
	case "]", "}":
		return KeyWizard
	}
	return Key(raw)
}

//go:build js

package main

import (
	"syscall/js"

	"partyrogue/game"
)

// armSaveHook saves a live run when the tab closes.
func armSaveHook(getGame func() *game.Game) {
	js.Global().Get("window").Call("addEventListener", "beforeunload", js.FuncOf(func(this js.Value, args []js.Value) any {
		if g := getGame(); g != nil && !g.Over {
			_ = game.Save(g)
		}
		return nil
	}))
}

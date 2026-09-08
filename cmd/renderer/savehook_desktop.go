//go:build !js

package main

import (
	"partyrogue/game"
)

// armSaveHook is a no-op on desktop; quit paths save explicitly.
func armSaveHook(_ func() *game.Game) {}

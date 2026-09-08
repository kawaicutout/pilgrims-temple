//go:build !js

package main

// setupShell is a no-op on desktop.
func setupShell() {}

// viewportFit is 1 on desktop: the window opens at grid size.
func viewportFit() float64 { return 1 }

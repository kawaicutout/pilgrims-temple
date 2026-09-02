//go:build !js

package game

func overlayJSON(name string) ([]byte, bool) { return nil, false }

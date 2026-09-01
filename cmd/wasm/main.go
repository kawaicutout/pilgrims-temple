//go:build js && wasm

package main

import (
	"fmt"
	"partyrogue/game"
)

func main() {
	tuning, err := game.LoadTuning()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Pilgrim's Temple (WASM) — %dx%d map, %dx%d min\n",
		tuning.Map.Width, tuning.Map.Height,
		tuning.Layout.MinCols, tuning.Layout.MinRows)
	// Block forever; real main loop lands in M1.
	select {}
}

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
	fmt.Printf("Pilgrim's Temple — %dx%d map, %dx%d min window, %d floors\n",
		tuning.Map.Width, tuning.Map.Height,
		tuning.Layout.MinCols, tuning.Layout.MinRows,
		tuning.Floors)
	fmt.Println("M0 scaffold. M1 starts here.")
}

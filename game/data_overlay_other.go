//go:build !js

package game

import (
	"os"
)

func overlayJSON(name string) ([]byte, bool) {
	// Check filesystem overlay before embedded FS — allows desktop data mods
	// to be detected at runtime without rebuild, and for HasModifiedData to be true.
	for _, p := range []string{"game/data/" + name, "data/" + name} {
		if b, err := os.ReadFile(p); err == nil {
			return b, true
		}
	}
	return nil, false
}

func HasModifiedData() bool {
	// Consider modded if any known data file has a filesystem overlay that differs from embedded.
	names := []string{
		"tuning.json", "classes.json", "races.json", "affixes.json", "talents.json",
		"enemies.json", "potions.json", "scrolls.json", "biomes.json", "floorThemes.json",
		"litter.json", "features.json", "fountains.json", "shrines.json", "merchants.json",
		"world.json", "statuses.json", "title.txt",
	}
	for _, name := range names {
		if b, ok := overlayJSON(name); ok {
			if emb, err := dataFS.ReadFile("data/" + name); err == nil {
				if string(b) != string(emb) {
					return true
				}
			} else {
				return true
			}
		}
	}
	return false
}

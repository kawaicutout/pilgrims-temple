package game

import (
	"encoding/json"
	"testing"
)

// A quit-save must never brick Load game: Quit is session intent, and a
// persisted true used to bounce the first post-load action back to menu.
func TestQuitNotPersisted(t *testing.T) {
	g := &Game{Seed: 1, Quit: true}
	b, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	var h Game
	if err := json.Unmarshal(b, &h); err != nil {
		t.Fatal(err)
	}
	if h.Quit {
		t.Fatal("fresh save round-trip restored Quit")
	}
	// Legacy poisoned payload (pre-fix saves stored quit:true).
	var legacy Game
	if err := json.Unmarshal([]byte(`{"seed":1,"quit":true}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.Quit {
		t.Fatal("legacy quit:true save restored Quit")
	}
}

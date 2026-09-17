package game

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
)

// Gold purse on Game is stored externally to keep this file vet-clean
// before orchestrator adds Game.Gold field. When field exists, these
// helpers will be migrated to use it; external store keeps compile green.
var goldStore = map[*Game]int{}

func (g *Game) SetGold(n int) {
	if n < 0 {
		n = 0
	}
	// keep both for migration
	g.Gold = n
	goldStore[g] = n
}

// AddGold adds n to purse.
func (g *Game) AddGold(n int) {
	if n <= 0 {
		return
	}
	// migrate if needed
	if v, ok := goldStore[g]; ok {
		g.Gold = v
		delete(goldStore, g)
	}
	g.Gold += n
}

// SpendGold deducts n if affordable.
func (g *Game) SpendGold(n int) bool {
	if n <= 0 {
		return true
	}
	if v, ok := goldStore[g]; ok {
		g.Gold = v
		delete(goldStore, g)
	}
	if g.Gold < n {
		return false
	}
	g.Gold -= n
	return true
}

// Merchant is a scarce level feature that converts gold into items.
type Merchant struct {
	Pos    Pos    `json:"pos"`
	Wares  []Ware `json:"wares"`
	Scarce bool   `json:"scarce"`
}

// Ware is one merchant offering.
type Ware struct {
	ID    string `json:"id"`
	Price int    `json:"price"`
	Name  string `json:"name,omitempty"`
}

type merchantsFile struct {
	Wares []Ware `json:"wares"`
}

var merchantsCache []Ware

func loadMerchants() []Ware {
	if merchantsCache != nil {
		return merchantsCache
	}
	b, err := RawJSON("merchants.json")
	if err != nil {
		panic("merchants.json missing — single source required: " + err.Error())
	}
	var mf merchantsFile
	if err := json.Unmarshal(b, &mf); err != nil || len(mf.Wares) == 0 {
		preview := b
		if len(preview) > 200 {
			preview = preview[:200]
		}
		panic("merchants.json invalid or empty — single source required: " + string(preview))
	}
	merchantsCache = mf.Wares
	return merchantsCache
}

// GetMerchantWares returns copy of wares data.
func GetMerchantWares() []Ware {
	src := loadMerchants()
	out := make([]Ware, len(src))
	copy(out, src)
	return out
}

// merchantWares returns 1-2 random wares without replacement, using rng.
func merchantWares(rng *rand.Rand) []Ware {
	wares := loadMerchants()
	if len(wares) == 0 {
		return nil
	}
	idx := make([]int, len(wares))
	for i := range idx {
		idx[i] = i
	}
	for i := len(idx) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		idx[i], idx[j] = idx[j], idx[i]
	}
	n := 1 + rng.IntN(2) // 1-2
	if n > len(wares) {
		n = len(wares)
	}
	picked := make([]Ware, n)
	for i := range n {
		picked[i] = wares[idx[i]]
	}
	return picked
}

// SpawnMerchant creates a merchant on lvl at pos with random wares.
// Scarce: 1-2 wares per merchant; pick without replacement.
func SpawnMerchant(rng *rand.Rand, lvl *Level, pos Pos) *Merchant {
	picked := merchantWares(rng)
	if len(picked) == 0 {
		return &Merchant{Pos: pos, Scarce: true}
	}
	return &Merchant{Pos: pos, Wares: picked, Scarce: true}
}

// MaybeSpawnMerchant chance is scarce (~15% per floor). Returns nil if none.
func MaybeSpawnMerchant(rng *rand.Rand, _ int) bool {
	return rng.Float64() < 0.15
}

// MerchantPrice returns price for ware id or 0.
func MerchantPrice(m *Merchant, wareID string) (int, bool) {
	for _, w := range m.Wares {
		if w.ID == wareID {
			return w.Price, true
		}
	}
	return 0, false
}

// BuyWare attempts purchase; returns error if unaffordable/missing.
func (g *Game) BuyWare(m *Merchant, wareID string) error {
	price, ok := MerchantPrice(m, wareID)
	if !ok {
		return fmt.Errorf("ware %s not found", wareID)
	}
	if !g.SpendGold(price) {
		return fmt.Errorf("need %d gold", price)
	}
	// Ware delivery is caller-handled (give item/ration/upgrade).
	return nil
}

// rollKillDrop rolls loot for one slain enemy: gold (45%) and food (30%)
// arrive instantly; potions (6%) and scrolls (6%) drop on the slain party's
// tile for pickup. Otherwise nothing.
func (g *Game) rollKillDrop(ep *EnemyParty) {
	if g == nil || g.RNG == nil {
		return
	}
	r := g.RNG.Float64()
	switch {
	case r < 0.45:
		amt := 3 + g.RNG.IntN(6) + g.Floor
		g.AddGold(amt)
		g.Logf("Looted %d gold.", amt)
	case r < 0.75:
		amt := 8 + g.RNG.IntN(8)
		g.AddFood(amt)
		g.Logf("Looted %d food.", amt)
	case r < 0.81:
		g.dropKillItem(ep, "potion")
	case r < 0.87:
		g.dropKillItem(ep, "scroll")
	}
}

// dropKillItem places a random potion/scroll of the given kind on the slain
// party's tile. Falls back to nothing when the tile is unavailable.
func (g *Game) dropKillItem(ep *EnemyParty, kind string) {
	lvl := g.CurLevel()
	if ep == nil || lvl == nil {
		return
	}
	var t struct {
		ID  string
		App string
	}
	if kind == "potion" {
		_, types := loadPotionData()
		if len(types) == 0 {
			return
		}
		pick := types[g.RNG.IntN(len(types))]
		t.ID = pick.ID
		t.App = AppearanceForType(pick.ID)
	} else {
		_, types := loadScrollData()
		if len(types) == 0 {
			return
		}
		pick := types[g.RNG.IntN(len(types))]
		t.ID = pick.ID
		t.App = AppearanceForType(pick.ID)
	}
	if t.App == "" {
		t.App = t.ID
	}
	lvl.Items = append(lvl.Items, GroundItem{Kind: kind, ID: t.ID, Name: t.App + " " + kind, Amount: 1, Pos: ep.Pos})
	g.Logf("The slain foe dropped a %s %s!", t.App, kind)
}

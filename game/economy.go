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
	b, err := dataFS.ReadFile("data/merchants.json")
	if err != nil {
		merchantsCache = []Ware{
			{ID: "ration", Price: 25, Name: "Ration"},
			{ID: "potion_heal", Price: 40, Name: "Healing Draught"},
			{ID: "scroll_upgrade", Price: 75, Name: "Scroll of Might"},
		}
		return merchantsCache
	}
	var mf merchantsFile
	if err := json.Unmarshal(b, &mf); err != nil || len(mf.Wares) == 0 {
		merchantsCache = []Ware{
			{ID: "ration", Price: 25, Name: "Ration"},
			{ID: "potion_heal", Price: 40, Name: "Healing Draught"},
			{ID: "scroll_upgrade", Price: 75, Name: "Scroll of Might"},
		}
		return merchantsCache
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
func MaybeSpawnMerchant(rng *rand.Rand, floor int) bool {
	_ = floor
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

package game

import (
	"encoding/json"
	"math/rand/v2"
)

// Knowledge is per-run identified appearance -> typeID.
// Unidentified appearances are absent from the map. Nothing persists between runs.
var Knowledge = map[string]string{}

// hidden mapping built at generation (run-random appearance drawn on generation).
// appearance -> typeID for every appearance, whether identified or not.
var appearanceToType = map[string]string{}

// reverse mapping typeID -> appearance (bijection per run).
var typeToAppearance = map[string]string{}

// ---------------------------------------------------------------------------
// Data loading
// ---------------------------------------------------------------------------

type potionFile struct {
	Appearances []string `json:"appearances"`
	Types       []itemType `json:"types"`
}

type scrollFile struct {
	Appearances []string `json:"appearances"`
	Types       []itemType `json:"types"`
}

type itemType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// fallback appearances — used if JSON load fails; kept for robustness.
// Must match the 8-entry pool described in DESIGN 7.2.
var fallbackPotionAppearances = []string{
	"Azure",
	"Crimson",
	"Mottled Jade",
	"Vivid Yellow",
	"Smoky Grey",
	"Pale Violet",
	"Deep Amber",
	"Murky Brown",
}

var fallbackScrollAppearances = []string{
	"Than of the Moth",
	"Kel of Shadows",
	"Vor of Flames",
	"Zin of Binding",
	"Eld of Whispers",
	"Mor of Unmaking",
	"Ir of Light",
	"Sha of Warding",
}

var fallbackPotionTypes = []itemType{
	{ID: "healing", Name: "Healing"},
	{ID: "poison", Name: "Poison"},
	{ID: "strength", Name: "Strength"},
	{ID: "invisibility", Name: "Invisibility"},
	{ID: "fire_resist", Name: "Fire Resistance"},
	{ID: "paralysis", Name: "Paralysis"},
	{ID: "levitation", Name: "Levitation"},
	{ID: "enlightenment", Name: "Enlightenment"},
}

var fallbackScrollTypes = []itemType{
	{ID: "identify", Name: "Identify"},
	{ID: "teleport", Name: "Teleport"},
	{ID: "fireball", Name: "Fireball"},
	{ID: "enchant", Name: "Enchant"},
	{ID: "mapping", Name: "Mapping"},
	{ID: "confusion", Name: "Confusion"},
	{ID: "healing", Name: "Greater Healing"},
	{ID: "summon", Name: "Summon Aid"},
}

func loadPotionData() (appearances []string, types []itemType) {
	b, err := dataFS.ReadFile("data/potions.json")
	if err != nil {
		return append([]string(nil), fallbackPotionAppearances...), append([]itemType(nil), fallbackPotionTypes...)
	}
	var pf potionFile
	if err := json.Unmarshal(b, &pf); err != nil {
		return append([]string(nil), fallbackPotionAppearances...), append([]itemType(nil), fallbackPotionTypes...)
	}
	if len(pf.Appearances) == 0 {
		pf.Appearances = append([]string(nil), fallbackPotionAppearances...)
	}
	if len(pf.Types) == 0 {
		pf.Types = append([]itemType(nil), fallbackPotionTypes...)
	}
	return pf.Appearances, pf.Types
}

func loadScrollData() (appearances []string, types []itemType) {
	b, err := dataFS.ReadFile("data/scrolls.json")
	if err != nil {
		return append([]string(nil), fallbackScrollAppearances...), append([]itemType(nil), fallbackScrollTypes...)
	}
	var sf scrollFile
	if err := json.Unmarshal(b, &sf); err != nil {
		return append([]string(nil), fallbackScrollAppearances...), append([]itemType(nil), fallbackScrollTypes...)
	}
	if len(sf.Appearances) == 0 {
		sf.Appearances = append([]string(nil), fallbackScrollAppearances...)
	}
	if len(sf.Types) == 0 {
		sf.Types = append([]itemType(nil), fallbackScrollTypes...)
	}
	return sf.Appearances, sf.Types
}

// ---------------------------------------------------------------------------
// Knowledge helpers
// ---------------------------------------------------------------------------

// IsIdentified reports whether appearance has been identified this run.
func IsIdentified(appearance string) bool {
	_, ok := Knowledge[appearance]
	return ok
}

// Identify marks appearance as identified as typeID.
// It is idempotent — identifying the same appearance twice keeps the first type.
func Identify(appearance, typeID string) {
	if appearance == "" || typeID == "" {
		return
	}
	if _, ok := Knowledge[appearance]; ok {
		return
	}
	Knowledge[appearance] = typeID
}

// AppearanceForType returns the run-random appearance for a given typeID,
// or "" if the type is unknown (before InitIdentification or invalid id).
func AppearanceForType(typeID string) string {
	return typeToAppearance[typeID]
}

// TypeForAppearance returns the hidden typeID for an appearance,
// or "" if unknown. This is the unidentified truth; callers should
// normally check IsIdentified first and only reveal via IdentifyOnUse.
func TypeForAppearance(appearance string) string {
	return appearanceToType[appearance]
}

// IdentifyOnUse reveals the type of the used appearance and returns
// whether this was the first time it became identified.
// Every vial/scroll of the same appearance is then considered identified
// (DESIGN 11.1: per-appearance identification).
func IdentifyOnUse(appearance string) bool {
	if IsIdentified(appearance) {
		return false
	}
	typeID := appearanceToType[appearance]
	if typeID == "" {
		return false
	}
	Identify(appearance, typeID)
	return true
}

// ShouldAutoID reports whether the wizard passive should auto-identify
// one held appearance on this turn. Per DESIGN 5.5 and tuning: every 50 turns.
// The caller checks party has a living wizard and picks which held appearance to reveal.
func ShouldAutoID(turn int) bool {
	return turn > 0 && turn%50 == 0
}

// WizardAutoID is an alias for ShouldAutoID for callers that prefer the
// wizard-specific name. Every 50 turns (turn > 0) returns true.
func WizardAutoID(turn int) bool {
	return ShouldAutoID(turn)
}

// WizardShouldAutoID is a second alias covering the alternative naming
// some callers may expect.
func WizardShouldAutoID(turn int) bool {
	return ShouldAutoID(turn)
}

// ResetIdentification clears per-run knowledge without rebuilding the
// appearance shuffle. Useful for tests.
func ResetIdentification() {
	Knowledge = map[string]string{}
}

// InitIdentification builds the run-random bijection for this run and
// clears Knowledge (unidentified until first use). Must be called once
// per run with the run RNG (e.g., from NewGame). It is safe to call
// multiple times — each call re-rolls the shuffle.
func InitIdentification(rng *rand.Rand) {
	pApps, pTypes := loadPotionData()
	sApps, sTypes := loadScrollData()

	// Copy so shuffling does not mutate the source slices.
	potions := append([]string(nil), pApps...)
	scrolls := append([]string(nil), sApps...)

	// Shuffle appearances onto types: each type gets a random appearance.
	// Use rng.Shuffle via the generic helper shuffleStrings.
	shuffleStrings(rng, potions)
	shuffleStrings(rng, scrolls)

	appearanceToType = map[string]string{}
	typeToAppearance = map[string]string{}
	Knowledge = map[string]string{}

	// Potions bijection
	for i := range pTypes {
		if i >= len(potions) {
			break
		}
		app := potions[i]
		tid := pTypes[i].ID
		appearanceToType[app] = tid
		typeToAppearance[tid] = app
	}
	// Scrolls bijection (type IDs may overlap potion ids like "healing";
	// appearances are distinct tokens so no collision in appearanceToType.
	// For typeToAppearance a scroll "healing" would overwrite potion "healing"
	// — disambiguate by namespacing if needed. In practice ids are distinct
	// except "healing"; keep the last write or namespace with prefix.
	// To preserve both, store scroll types with a prefix when colliding.
	for i := range sTypes {
		if i >= len(scrolls) {
			break
		}
		app := scrolls[i]
		tid := sTypes[i].ID
		// If this typeID already exists (potion/scroll overlap), keep both
		// accessible via bare id (last wins) but ensure the appearance still
		// maps correctly. Overlap is intentional minimal; no prefix needed
		// because appearances never collide — the only ambiguity is
		// AppearanceForType("healing") returning the scroll appearance if
		// scroll was processed last. To keep determinism, prefer potion
		// mapping for overlapping ids by not overwriting.
		if _, exists := typeToAppearance[tid]; exists {
			// Keep first (potion) mapping; still register appearance->type
			// so IdentifyOnUse works for the scroll appearance.
			appearanceToType[app] = tid
			continue
		}
		appearanceToType[app] = tid
		typeToAppearance[tid] = app
	}
}

// InitIdentificationSeed is a convenience for callers that have a seed
// instead of an RNG. It creates a temporary RNG from the seed.
func InitIdentificationSeed(seed int64) {
	rng := rand.New(rand.NewPCG(uint64(seed), 0x9e3779b97f4a7c15))
	InitIdentification(rng)
}

// shuffleStrings shuffles s in place using rng.
func shuffleStrings(rng *rand.Rand, s []string) {
	n := len(s)
	for i := n - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		s[i], s[j] = s[j], s[i]
	}
}

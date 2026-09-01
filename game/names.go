package game

import (
	"encoding/json"
	"math/rand/v2"
	"strings"
	"sync"
)

// fallback data — used if JSON load fails; kept for robustness.
var fallbackRealNames = []string{
	"Ari", "Bren", "Cael", "Dara", "Emri", "Fenn", "Garr", "Hale",
	"Joren", "Kael", "Lira", "Mira", "Nara", "Oren", "Selene", "Thalia",
	"Kaito", "Yuki", "Hana", "Rin", "Akira", "Sora", "Kenji", "Elena",
	"Marco", "Sofia", "Lars", "Ingrid", "Bjorn", "Anika", "Sven", "Freya",
	"Amara", "Jabari", "Zuri", "Kofi", "Aisha", "Hassan", "Leila", "Omar",
	"Priya", "Arjun", "Mei", "Chen", "Yara", "Samir", "Nadia", "Rafael",
}

var fallbackFantasyNames = []string{
	"Eldrin", "Thalor", "Bryn", "Aelwen", "Cedric", "Lyra", "Orion", "Sable",
	"Wren", "Ash", "Ember", "Vale", "Jasper", "Luna", "Silas", "Rowan",
	"Aeris", "Zephyr", "Nyx", "Draven", "Seraph", "Kaelen", "Lyris", "Torin",
	"Elyndra", "Merric", "Sorin", "Yvaine", "Caelum", "Isolde", "Thrain", "Elowen",
	"Bramble", "Corin", "Edda", "Galen", "Hawke", "Ilyra", "Jorah", "Kestrel",
}

var fallbackOnsets = []string{"k", "t", "s", "m", "n", "l", "r", "th", "sh", "kh", "br", "dr", "gr", "st", "ae", "io", "el", "or"}
var fallbackNuclei = []string{"a", "e", "i", "o", "u", "ae", "ei", "ou", "ia", "an", "en", "or"}
var fallbackCodas = []string{"", "", "", "n", "r", "l", "s", "th", "en", "ar", "is", "or"}

// active slices populated from JSON on first use; fallback if load fails.
var (
	realNames     []string
	fantasyNames  []string
	conlangOnsets []string
	conlangNuclei []string
	conlangCodas  []string
	onsetWeights  []float64
	nucleiWeights []float64
	codaWeights   []float64
	namesOnce     sync.Once
)

type namesFile struct {
	Real    []string `json:"real"`
	Fantasy []string `json:"fantasy"`
}

type conlangFile struct {
	Onsets        []string  `json:"onsets"`
	Nuclei        []string  `json:"nuclei"`
	Codas         []string  `json:"codas"`
	OnsetWeights  []float64 `json:"onsetWeights"`
	NucleiWeights []float64 `json:"nucleiWeights"`
	CodaWeights   []float64 `json:"codaWeights"`
}

func ensureNamesLoaded() {
	namesOnce.Do(func() {
		// names.json
		if b, err := RawJSON("names.json"); err == nil {
			var nf namesFile
			if json.Unmarshal(b, &nf) == nil && len(nf.Real) > 0 && len(nf.Fantasy) > 0 {
				realNames = nf.Real
				fantasyNames = nf.Fantasy
			} else {
				realNames = fallbackRealNames
				fantasyNames = fallbackFantasyNames
			}
		} else {
			realNames = fallbackRealNames
			fantasyNames = fallbackFantasyNames
		}
		// conlang.json
		if b, err := RawJSON("conlang.json"); err == nil {
			var cf conlangFile
			if json.Unmarshal(b, &cf) == nil && len(cf.Onsets) > 0 && len(cf.Nuclei) > 0 && len(cf.Codas) > 0 {
				conlangOnsets = cf.Onsets
				conlangNuclei = cf.Nuclei
				conlangCodas = cf.Codas
				onsetWeights = cf.OnsetWeights
				nucleiWeights = cf.NucleiWeights
				codaWeights = cf.CodaWeights
			} else {
				conlangOnsets = fallbackOnsets
				conlangNuclei = fallbackNuclei
				conlangCodas = fallbackCodas
			}
		} else {
			conlangOnsets = fallbackOnsets
			conlangNuclei = fallbackNuclei
			conlangCodas = fallbackCodas
		}
		// normalize weights: must match length, else discard (uniform)
		if len(onsetWeights) != len(conlangOnsets) {
			onsetWeights = nil
		}
		if len(nucleiWeights) != len(conlangNuclei) {
			nucleiWeights = nil
		}
		if len(codaWeights) != len(conlangCodas) {
			codaWeights = nil
		}
	})
}

func pickWeighted(rng *rand.Rand, items []string, weights []float64) string {
	if len(items) == 0 {
		return ""
	}
	if len(weights) != len(items) {
		return items[rng.IntN(len(items))]
	}
	var total float64
	for _, w := range weights {
		if w > 0 {
			total += w
		}
	}
	if total <= 0 {
		return items[rng.IntN(len(items))]
	}
	r := rng.Float64() * total
	var cum float64
	for i, w := range weights {
		if w <= 0 {
			continue
		}
		cum += w
		if r < cum {
			return items[i]
		}
	}
	return items[len(items)-1]
}

func conlangName(rng *rand.Rand) string {
	ensureNamesLoaded()
	syllables := 2 + rng.IntN(2) // 2 or 3
	var b strings.Builder
	for i := range syllables {
		on := pickWeighted(rng, conlangOnsets, onsetWeights)
		nu := pickWeighted(rng, conlangNuclei, nucleiWeights)
		co := pickWeighted(rng, conlangCodas, codaWeights)
		syl := on + nu + co
		if i == 0 {
			if len(syl) > 0 {
				syl = strings.ToUpper(syl[:1]) + syl[1:]
			}
		}
		b.WriteString(syl)
		if b.Len() > 9 {
			break
		}
	}
	s := b.String()
	if len(s) < 3 {
		s += pickWeighted(rng, conlangNuclei, nucleiWeights)
	}
	if len(s) > 10 {
		s = s[:10]
	}
	return s
}

// GenerateName picks a name from real (40%), fantasy (40%), conlang (20%), avoiding duplicates in party if provided.
func GenerateName(rng *rand.Rand, used map[string]bool) string {
	ensureNamesLoaded()
	for range 20 {
		var name string
		r := rng.Float64()
		switch {
		case r < 0.4:
			name = realNames[rng.IntN(len(realNames))]
		case r < 0.8:
			name = fantasyNames[rng.IntN(len(fantasyNames))]
		default:
			name = conlangName(rng)
		}
		if !used[name] {
			return name
		}
	}
	base := conlangName(rng)
	for i := 2; i < 100; i++ {
		cand := base
		if i < 10 {
			cand = base + string(rune('0'+i))
		} else {
			cand = base + "-" + string(rune('0'+i%10))
		}
		if !used[cand] {
			return cand
		}
	}
	return conlangName(rng)
}

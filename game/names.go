package game

import (
	"math/rand/v2"
	"strings"
)

// Real names: diverse, grounded.
var realNames = []string{
	"Ari", "Bren", "Cael", "Dara", "Emri", "Fenn", "Garr", "Hale",
	"Joren", "Kael", "Lira", "Mira", "Nara", "Oren", "Selene", "Thalia",
	"Kaito", "Yuki", "Hana", "Rin", "Akira", "Sora", "Kenji", "Elena",
	"Marco", "Sofia", "Lars", "Ingrid", "Bjorn", "Anika", "Sven", "Freya",
	"Amara", "Jabari", "Zuri", "Kofi", "Aisha", "Hassan", "Leila", "Omar",
	"Priya", "Arjun", "Mei", "Chen", "Yara", "Samir", "Nadia", "Rafael",
}

// Fantasy names: slightly wilder, temple-appropriate.
var fantasyNames = []string{
	"Eldrin", "Thalor", "Bryn", "Aelwen", "Cedric", "Lyra", "Orion", "Sable",
	"Wren", "Ash", "Ember", "Vale", "Jasper", "Luna", "Silas", "Rowan",
	"Aeris", "Zephyr", "Nyx", "Draven", "Seraph", "Kaelen", "Lyris", "Torin",
	"Elyndra", "Merric", "Sorin", "Yvaine", "Caelum", "Isolde", "Thrain", "Elowen",
	"Bramble", "Corin", "Edda", "Galen", "Hawke", "Ilyra", "Jorah", "Kestrel",
}

// Conlang syllables for backup generation.
var conlangOnsets = []string{"k", "t", "s", "m", "n", "l", "r", "th", "sh", "kh", "br", "dr", "gr", "st", "ae", "io", "el", "or"}
var conlangNuclei = []string{"a", "e", "i", "o", "u", "ae", "ei", "ou", "ia", "an", "en", "or"}
var conlangCodas = []string{"", "", "", "n", "r", "l", "s", "th", "en", "ar", "is", "or"}

func conlangName(rng *rand.Rand) string {
	syllables := 2 + rng.IntN(2) // 2 or 3
	var b strings.Builder
	for i := range syllables {
		on := conlangOnsets[rng.IntN(len(conlangOnsets))]
		nu := conlangNuclei[rng.IntN(len(conlangNuclei))]
		co := conlangCodas[rng.IntN(len(conlangCodas))]
		syl := on + nu + co
		if i == 0 {
			// Capitalize first syllable
			if len(syl) > 0 {
				syl = strings.ToUpper(syl[:1]) + syl[1:]
			}
		}
		b.WriteString(syl)
		// Avoid overly long
		if b.Len() > 9 {
			break
		}
	}
	s := b.String()
	if len(s) < 3 {
		s += conlangNuclei[rng.IntN(len(conlangNuclei))]
	}
	// Ensure not too long, truncate to 10
	if len(s) > 10 {
		s = s[:10]
	}
	return s
}

// GenerateName picks a name from real (40%), fantasy (40%), conlang (20%), avoiding duplicates in party if provided.
func GenerateName(rng *rand.Rand, used map[string]bool) string {
	for tries := 0; tries < 20; tries++ {
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
	// Fallback: conlang with suffix to ensure unique
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

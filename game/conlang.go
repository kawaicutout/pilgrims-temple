package game

import (
	"encoding/json"
	"math/rand/v2"
	"strings"
	"sync"
)

// conlangData mirrors game/data/conlang.json.
type conlangData struct {
	Onsets       []string `json:"onsets"`
	Nuclei       []string `json:"nuclei"`
	Codas        []string `json:"codas"`
	OnsetWeights []int    `json:"onsetWeights"`
	NucleiWeights []int   `json:"nucleiWeights"`
	CodaWeights  []int    `json:"codaWeights"`
}

var (
	conlangOnce sync.Once
	cachedConlang *conlangData
)

func getConlangData() *conlangData {
	conlangOnce.Do(func() {
		b, err := dataFS.ReadFile("data/conlang.json")
		if err != nil {
			// fallback minimal table if JSON missing
			cachedConlang = &conlangData{
				Onsets: []string{"k", "t", "s", "m", "n", "l", "r", "th", "sh", "kh", "br", "dr", "gr", "st"},
				Nuclei: []string{"a", "e", "i", "o", "u", "ae", "ei", "ou", "ia", "an", "en", "or"},
				Codas:  []string{"", "", "", "n", "r", "l", "s", "th", "en", "ar", "is", "or"},
			}
			return
		}
		var cd conlangData
		if err := json.Unmarshal(b, &cd); err != nil {
			cachedConlang = &conlangData{
				Onsets: []string{"k", "t", "s", "m", "n", "l", "r", "th", "sh", "kh", "br", "dr", "gr", "st"},
				Nuclei: []string{"a", "e", "i", "o", "u", "ae", "ei", "ou", "ia", "an", "en", "or"},
				Codas:  []string{"", "", "", "n", "r", "l", "s", "th", "en", "ar", "is", "or"},
			}
			return
		}
		// normalize to lower and ensure weights length matches
		for i, s := range cd.Onsets {
			cd.Onsets[i] = strings.ToLower(strings.TrimSpace(s))
		}
		for i, s := range cd.Nuclei {
			cd.Nuclei[i] = strings.ToLower(strings.TrimSpace(s))
		}
		for i, s := range cd.Codas {
			cd.Codas[i] = strings.ToLower(strings.TrimSpace(s))
		}
		cachedConlang = &cd
	})
	return cachedConlang
}

// weightedPick returns a weighted random element from items using weights.
// If weights length mismatches or total <=0, falls back to uniform.
func weightedPick(rng *rand.Rand, items []string, weights []int) string {
	if len(items) == 0 {
		return ""
	}
	if len(weights) != len(items) {
		return items[rng.IntN(len(items))]
	}
	total := 0
	for _, w := range weights {
		if w > 0 {
			total += w
		}
	}
	if total <= 0 {
		return items[rng.IntN(len(items))]
	}
	r := rng.IntN(total)
	cum := 0
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

// GenerateConlangWord generates a 1-2 word conlang token using weighted
// onsets/nuclei/codas from conlang.json. Output is lowercase and may contain
// a single space, " of " separator, or apostrophe. Examples: "thae khor",
// "branel", "zae of orin".
func GenerateConlangWord(rng *rand.Rand) string {
	if rng == nil {
		rng = rand.New(rand.NewPCG(0, 1))
	}
	cd := getConlangData()
	if cd == nil || len(cd.Onsets) == 0 || len(cd.Nuclei) == 0 {
		// absolute fallback
		letters := []string{"a", "e", "i", "o", "u", "th", "kh", "br", "sh"}
		return letters[rng.IntN(len(letters))] + letters[rng.IntN(len(letters))]
	}
	makeWord := func() string {
		o := weightedPick(rng, cd.Onsets, cd.OnsetWeights)
		n := weightedPick(rng, cd.Nuclei, cd.NucleiWeights)
		c := weightedPick(rng, cd.Codas, cd.CodaWeights)
		w := o + n + c
		w = strings.ToLower(strings.TrimSpace(w))
		if w == "" {
			w = n
		}
		// collapse any accidental spaces inside components (should not happen)
		w = strings.ReplaceAll(w, " ", "")
		w = strings.ReplaceAll(w, "'", "")
		if w == "" {
			w = "ae"
		}
		return w
	}
	word := makeWord()
	// 40% single word, 60% two words
	if rng.IntN(100) < 40 {
		return word
	}
	word2 := makeWord()
	tries := 0
	for word2 == word && tries < 5 {
		word2 = makeWord()
		tries++
	}
	roll := rng.IntN(100)
	if roll < 55 {
		return word + " " + word2
	} else if roll < 80 {
		return word + " of " + word2
	}
	return word + "'" + word2
}

// generateConlangScrollAppearances returns 8 unique conlang tokens for scrolls
// using rng. Ensures no collision with potion appearance namespace (case-insensitive).
// Returns nil if rng is nil or cannot generate 8 unique tokens within budget.
func generateConlangScrollAppearances(rng *rand.Rand) []string {
	if rng == nil {
		return nil
	}
	// Build potion namespace set (lowercase) to keep distinct.
	potionSet := map[string]struct{}{}
	if pApps, _ := loadPotionData(); len(pApps) > 0 {
		for _, a := range pApps {
			potionSet[strings.ToLower(strings.TrimSpace(a))] = struct{}{}
		}
	}
	for _, a := range fallbackPotionAppearances {
		potionSet[strings.ToLower(strings.TrimSpace(a))] = struct{}{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, 8)
	attempts := 0
	for len(out) < 8 && attempts < 300 {
		attempts++
		w := GenerateConlangWord(rng)
		w = strings.ToLower(strings.TrimSpace(w))
		if w == "" {
			continue
		}
		// validate charset: only a-z, space, apostrophe
		valid := true
		for _, r := range w {
			if (r >= 'a' && r <= 'z') || r == ' ' || r == '\'' {
				continue
			}
			valid = false
			break
		}
		if !valid {
			continue
		}
		// ensure not empty after trim and not duplicate
		if _, ok := potionSet[w]; ok {
			continue
		}
		if _, ok := seen[w]; ok {
			continue
		}
		seen[w] = struct{}{}
		out = append(out, w)
	}
	if len(out) != 8 {
		return nil
	}
	return out
}

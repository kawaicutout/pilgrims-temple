package game

import (
	"encoding/json"
	"math/rand/v2"
)

type talentData struct {
	Generic []struct {
		ID string `json:"id"`
	} `json:"generic"`
	Tagged map[string][]struct {
		ID string `json:"id"`
	} `json:"tagged"`
	PerClass map[string]json.RawMessage `json:"perClass"`
}

type classData struct {
	Classes []struct {
		ID   string   `json:"id"`
		Tags []string `json:"tags"`
	} `json:"classes"`
}

var talentCache *talentData
var classCache *classData

func loadTalents() (*talentData, error) {
	if talentCache != nil {
		return talentCache, nil
	}
	b, err := RawJSON("talents.json")
	if err != nil {
		return nil, err
	}
	var td talentData
	if err := json.Unmarshal(b, &td); err != nil {
		return nil, err
	}
	talentCache = &td
	return talentCache, nil
}

func loadClassesForTalents() (*classData, error) {
	if classCache != nil {
		return classCache, nil
	}
	b, err := RawJSON("classes.json")
	if err != nil {
		return nil, err
	}
	var cd classData
	if err := json.Unmarshal(b, &cd); err != nil {
		return nil, err
	}
	classCache = &cd
	return classCache, nil
}

func GetEligibleTalents(class string) []string {
	td, err := loadTalents()
	if err != nil {
		return []string{"tough", "keen"}
	}
	cd, err := loadClassesForTalents()
	if err != nil {
		var out []string
		for _, g := range td.Generic {
			out = append(out, g.ID)
		}
		return out
	}
	var tags []string
	for _, c := range cd.Classes {
		if c.ID == class {
			tags = c.Tags
			break
		}
	}
	seen := map[string]bool{}
	var out []string
	add := func(id string) {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	// Per-class: unmarshal RawMessage that is a slice, ignore "notes" string
	if raw, ok := td.PerClass[class]; ok {
		// Try to unmarshal as slice; if it's a string (notes), skip
		if len(raw) > 0 && raw[0] == '[' {
			var per []struct{ID string `json:"id"`}
			if err := json.Unmarshal(raw, &per); err == nil {
				for _, p := range per {
					add(p.ID)
				}
			}
		}
	}
	for _, t := range tags {
		if lst, ok := td.Tagged[t]; ok {
			for _, p := range lst {
				add(p.ID)
			}
		}
	}
	for _, g := range td.Generic {
		add(g.ID)
	}
	return out
}

type affixData struct {
	Prefixes []struct {
		ID string `json:"id"`
	} `json:"prefixes"`
	Suffixes []struct {
		ID string `json:"id"`
	} `json:"suffixes"`
}

func GetRandomAffix(rng *rand.Rand) string {
	if rng == nil {
		return "veteran"
	}
	b, err := RawJSON("affixes.json")
	if err != nil {
		return "veteran"
	}
	var ad affixData
	if err := json.Unmarshal(b, &ad); err != nil {
		return "veteran"
	}
	var all []string
	for _, p := range ad.Prefixes {
		all = append(all, p.ID)
	}
	for _, s := range ad.Suffixes {
		all = append(all, s.ID)
	}
	if len(all) == 0 {
		return "veteran"
	}
	return all[rng.IntN(len(all))]
}

func GetTalentOptions(rng *rand.Rand, class string, count int) []string {
	eligible := GetEligibleTalents(class)
	if len(eligible) == 0 {
		return []string{"tough"}
	}
	for i := range eligible {
		j := rng.IntN(len(eligible))
		eligible[i], eligible[j] = eligible[j], eligible[i]
	}
	if count > len(eligible) {
		count = len(eligible)
	}
	return eligible[:count]
}

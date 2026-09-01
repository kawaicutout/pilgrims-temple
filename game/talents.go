package game

import (
	"encoding/json"
	"fmt"
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

// --- Talent / affix description helpers ---

type descEntry struct {
	Name string
	Desc string
}

var talentDescMap map[string]descEntry
var affixDescMap map[string]descEntry

func ensureTalentDescCache() {
	if talentDescMap != nil {
		return
	}
	talentDescMap = make(map[string]descEntry)
	if b, err := RawJSON("talents.json"); err == nil {
		var raw struct {
			Generic []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Desc string `json:"desc"`
			} `json:"generic"`
			Tagged map[string][]struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Desc string `json:"desc"`
			} `json:"tagged"`
		}
		if err := json.Unmarshal(b, &raw); err == nil {
			for _, g := range raw.Generic {
				talentDescMap[g.ID] = descEntry{Name: g.Name, Desc: g.Desc}
			}
			for _, lst := range raw.Tagged {
				for _, e := range lst {
					talentDescMap[e.ID] = descEntry{Name: e.Name, Desc: e.Desc}
				}
			}
		}
	}
	if b, err := RawJSON("classes.json"); err == nil {
		var raw struct {
			Classes []struct {
				BuffA struct {
					ID   string `json:"id"`
					Name string `json:"name"`
					Desc string `json:"desc"`
				} `json:"buffA"`
				BuffB struct {
					ID   string `json:"id"`
					Name string `json:"name"`
					Desc string `json:"desc"`
				} `json:"buffB"`
				Active struct {
					ID   string `json:"id"`
					Name string `json:"name"`
					Desc string `json:"desc"`
				} `json:"active"`
			} `json:"classes"`
		}
		if err := json.Unmarshal(b, &raw); err == nil {
			for _, c := range raw.Classes {
				if c.BuffA.ID != "" {
					if _, ok := talentDescMap[c.BuffA.ID]; !ok {
						talentDescMap[c.BuffA.ID] = descEntry{Name: c.BuffA.Name, Desc: c.BuffA.Desc}
					}
				}
				if c.BuffB.ID != "" {
					talentDescMap[c.BuffB.ID] = descEntry{Name: c.BuffB.Name, Desc: c.BuffB.Desc}
				}
				if c.Active.ID != "" {
					talentDescMap[c.Active.ID] = descEntry{Name: c.Active.Name, Desc: c.Active.Desc}
				}
			}
		}
	}
}

func ensureAffixDescCache() {
	if affixDescMap != nil {
		return
	}
	affixDescMap = make(map[string]descEntry)
	if b, err := RawJSON("affixes.json"); err == nil {
		var raw struct {
			Prefixes []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Desc string `json:"desc"`
			} `json:"prefixes"`
			Suffixes []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Desc string `json:"desc"`
			} `json:"suffixes"`
		}
		if err := json.Unmarshal(b, &raw); err == nil {
			for _, p := range raw.Prefixes {
				affixDescMap[p.ID] = descEntry{Name: p.Name, Desc: p.Desc}
			}
			for _, s := range raw.Suffixes {
				affixDescMap[s.ID] = descEntry{Name: s.Name, Desc: s.Desc}
			}
		}
	}
}

// GetTalentDesc returns "Name - Desc" for a talent id, falling back to id.
func GetTalentDesc(id string) string {
	ensureTalentDescCache()
	if e, ok := talentDescMap[id]; ok {
		switch {
		case e.Name != "" && e.Desc != "":
			return fmt.Sprintf("%s - %s", e.Name, e.Desc)
		case e.Name != "":
			return e.Name
		case e.Desc != "":
			return fmt.Sprintf("%s - %s", id, e.Desc)
		}
	}
	return id
}

// GetAffixDesc returns "Name - Desc" for an affix id, falling back to id.
func GetAffixDesc(id string) string {
	ensureAffixDescCache()
	if e, ok := affixDescMap[id]; ok {
		switch {
		case e.Name != "" && e.Desc != "":
			return fmt.Sprintf("%s - %s", e.Name, e.Desc)
		case e.Name != "":
			return e.Name
		case e.Desc != "":
			return fmt.Sprintf("%s - %s", id, e.Desc)
		}
	}
	return id
}

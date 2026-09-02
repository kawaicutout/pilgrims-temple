package game

import "strings"

// FriendlyID converts internal IDs like "fire_resist", "of_wrath",
// "mushroom_cap", "veteran" into user-facing "Fire Resist", "Of Wrath",
// "Mushroom Cap", "Veteran". It replaces "_" and "-" with spaces and
// Title-cases each word. Internal IDs remain unchanged; only display uses this.
func FriendlyID(id string) string {
	if id == "" {
		return ""
	}
	// Normalize separators.
	s := strings.ReplaceAll(id, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	parts := strings.Fields(s)
	for i, w := range parts {
		if len(w) == 0 {
			continue
		}
		// Title case: first rune upper, rest lower.
		// Handle small words? Keep simple Title.
		parts[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
	}
	return strings.Join(parts, " ")
}

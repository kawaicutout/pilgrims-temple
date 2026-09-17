package game

import "math/rand/v2"

// stairsReachableViaBFS checks StairsUp -> StairsDown over walkable tiles plus litter blocks.
func stairsReachableViaBFS(lvl *Level) bool {
	if lvl == nil {
		return false
	}
	start := lvl.StairsUp
	goal := lvl.StairsDown
	// Floor 0 may have stairs up at same as start? Still need path to down.
	if !lvl.InBounds(start) || !lvl.InBounds(goal) {
		return false
	}
	// BFS
	visited := make(map[Pos]bool, lvl.W*lvl.H)
	queue := []Pos{start}
	visited[start] = true
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == goal {
			return true
		}
		for _, d := range AllDirs {
			np := Pos{cur.X + d.DX, cur.Y + d.DY}
			if !lvl.InBounds(np) {
				continue
			}
			if visited[np] {
				continue
			}
			// Walkable check includes litter via lvl.Walkable
			if !lvl.Walkable(np) {
				continue
			}
			visited[np] = true
			queue = append(queue, np)
		}
	}
	// Also allow case where stairs are same tile (single room) -> visited already
	return visited[goal]
}

// AssertLevelHasExit checks that a level has a reachable exit stairs.
// For non-final floors (Floor != Tuning.Floors-1), both StairsUp and StairsDown
// must be InBounds, on TileStairsUp/TileStairsDown, Walkable (not blocked by
// litter/closed door), not coincident, and BFS-reachable via Walkable tiles.
// For the final floor (relic floor), StairsUp must be present and Walkable and
// reachable to the relic (which sits at StairsDown); if StairsDown is missing,
// StairsUp alone suffices but if present it must also be walkable and reachable.
func AssertLevelHasExit(lvl *Level) bool {
	if lvl == nil {
		return false
	}
	if !lvl.InBounds(lvl.StairsUp) {
		return false
	}
	if lvl.At(lvl.StairsUp) != TileStairsUp {
		return false
	}
	if !lvl.Walkable(lvl.StairsUp) {
		return false
	}
	for _, f := range lvl.Features {
		if f.Pos == lvl.StairsUp {
			return false
		}
	}
	// Determine final floor via tuning; fallback to 8.
	floors := 8
	if t, err := LoadTuning(); err == nil && t.Floors > 0 {
		floors = t.Floors
	}
	isFinal := lvl.Floor == floors-1
	if isFinal {
		if !lvl.InBounds(lvl.StairsDown) {
			// No stairs down required on relic floor; stairs up alone is enough.
			return true
		}
		if at := lvl.At(lvl.StairsDown); at != TileStairsDown && at != TileRelic {
			return false
		}
		if !lvl.Walkable(lvl.StairsDown) {
			return false
		}
		for _, f := range lvl.Features {
			if f.Pos == lvl.StairsDown {
				return false
			}
		}
		return stairsReachableViaBFS(lvl)
	}
	if !lvl.InBounds(lvl.StairsDown) {
		return false
	}
	if lvl.At(lvl.StairsDown) != TileStairsDown {
		return false
	}
	if !lvl.Walkable(lvl.StairsDown) {
		return false
	}
	if lvl.StairsUp == lvl.StairsDown {
		return false
	}
	for _, f := range lvl.Features {
		if f.Pos == lvl.StairsDown {
			return false
		}
	}
	return stairsReachableViaBFS(lvl)
}

// ensureStairsTiles fixes missing or blocked stairs by relocating them to
// walkable TileFloor positions that are not blocked by litter/features/doors.
func ensureStairsTiles(lvl *Level, rng *rand.Rand) {
	if lvl == nil {
		return
	}
	// Helper to clean litter/door at p for stairs.
	cleanPos := func(p Pos) {
		for i := 0; i < len(lvl.Litter); {
			if lvl.Litter[i].Pos == p {
				lvl.Litter = append(lvl.Litter[:i], lvl.Litter[i+1:]...)
			} else {
				i++
			}
		}
		if lvl.Doors != nil {
			delete(lvl.Doors, p)
		}
	}
	needsUp := !lvl.InBounds(lvl.StairsUp) || lvl.At(lvl.StairsUp) != TileStairsUp || !lvl.Walkable(lvl.StairsUp)
	if !needsUp {
		for _, f := range lvl.Features {
			if f.Pos == lvl.StairsUp {
				needsUp = true
				break
			}
		}
	}
	needsDown := false
	floors := 8
	if t, err := LoadTuning(); err == nil && t.Floors > 0 {
		floors = t.Floors
	}
	isFinal := lvl.Floor == floors-1
	if !isFinal {
		needsDown = !lvl.InBounds(lvl.StairsDown) || lvl.At(lvl.StairsDown) != TileStairsDown || !lvl.Walkable(lvl.StairsDown) || lvl.StairsUp == lvl.StairsDown
		if !needsDown {
			for _, f := range lvl.Features {
				if f.Pos == lvl.StairsDown {
					needsDown = true
					break
				}
			}
		}
	} else {
		if lvl.InBounds(lvl.StairsDown) {
			if at := lvl.At(lvl.StairsDown); !lvl.Walkable(lvl.StairsDown) || (at != TileStairsDown && at != TileRelic) {
				needsDown = true
			}
			for _, f := range lvl.Features {
				if f.Pos == lvl.StairsDown {
					needsDown = true
					break
				}
			}
		}
	}
	if !needsUp && !needsDown {
		// Still ensure no blocking litter directly on stairs.
		cleanPos(lvl.StairsUp)
		if lvl.InBounds(lvl.StairsDown) {
			cleanPos(lvl.StairsDown)
		}
		// Re-assert tiles.
		if lvl.InBounds(lvl.StairsUp) {
			lvl.Tiles[lvl.StairsUp.Y][lvl.StairsUp.X] = TileStairsUp
		}
		if lvl.InBounds(lvl.StairsDown) && !(isFinal && lvl.At(lvl.StairsDown) == TileRelic) {
			lvl.Tiles[lvl.StairsDown.Y][lvl.StairsDown.X] = TileStairsDown
		}
		return
	}
	// Collect walkable floor candidates not blocked.
	var candidates []Pos
	for y := 0; y < lvl.H; y++ {
		for x := 0; x < lvl.W; x++ {
			p := Pos{x, y}
			if lvl.At(p) != TileFloor {
				continue
			}
			if !lvl.Walkable(p) {
				continue
			}
			blocked := false
			for _, f := range lvl.Features {
				if f.Pos == p {
					blocked = true
					break
				}
			}
			if blocked {
				continue
			}
			if p == lvl.StairsUp || p == lvl.StairsDown {
				continue
			}
			candidates = append(candidates, p)
		}
	}
	// Also include current stairs positions if they are floor-like fallback.
	if len(candidates) == 0 {
		for y := 0; y < lvl.H; y++ {
			for x := 0; x < lvl.W; x++ {
				p := Pos{x, y}
				if lvl.At(p) == TileFloor {
					candidates = append(candidates, p)
				}
			}
		}
	}
	if len(candidates) == 0 {
		return
	}
	// Shuffle with rng if available.
	if rng != nil {
		for i := len(candidates) - 1; i > 0; i-- {
			j := rng.IntN(i + 1)
			candidates[i], candidates[j] = candidates[j], candidates[i]
		}
	}
	if needsUp {
		// Pick far from down if down valid.
		pick := candidates[0]
		if lvl.InBounds(lvl.StairsDown) && len(candidates) > 1 {
			best := pick
			bestDist := abs(best.X-lvl.StairsDown.X) + abs(best.Y-lvl.StairsDown.Y)
			for _, c := range candidates[1:] {
				d := abs(c.X-lvl.StairsDown.X) + abs(c.Y-lvl.StairsDown.Y)
				if d > bestDist {
					bestDist = d
					best = c
				}
			}
			pick = best
		}
		cleanPos(pick)
		// Clear previous stairs tile to floor if it was stairs.
		if lvl.InBounds(lvl.StairsUp) && lvl.At(lvl.StairsUp) == TileStairsUp {
			lvl.Tiles[lvl.StairsUp.Y][lvl.StairsUp.X] = TileFloor
		}
		lvl.StairsUp = pick
		lvl.Tiles[pick.Y][pick.X] = TileStairsUp
	}
	if needsDown {
		// Refresh candidates excluding new up.
		var filtered []Pos
		for _, c := range candidates {
			if c == lvl.StairsUp {
				continue
			}
			filtered = append(filtered, c)
		}
		if len(filtered) > 0 {
			candidates = filtered
		}
		pick := candidates[0]
		best := pick
		bestDist := abs(best.X-lvl.StairsUp.X) + abs(best.Y-lvl.StairsUp.Y)
		for _, c := range candidates[1:] {
			d := abs(c.X-lvl.StairsUp.X) + abs(c.Y-lvl.StairsUp.Y)
			if d > bestDist {
				bestDist = d
				best = c
			}
		}
		pick = best
		cleanPos(pick)
		if lvl.InBounds(lvl.StairsDown) && lvl.At(lvl.StairsDown) == TileStairsDown {
			lvl.Tiles[lvl.StairsDown.Y][lvl.StairsDown.X] = TileFloor
		}
		lvl.StairsDown = pick
		lvl.Tiles[pick.Y][pick.X] = TileStairsDown
	}
	// Final clean after relocation.
	cleanPos(lvl.StairsUp)
	if lvl.InBounds(lvl.StairsDown) {
		cleanPos(lvl.StairsDown)
	}
	lvl.Tiles[lvl.StairsUp.Y][lvl.StairsUp.X] = TileStairsUp
	if lvl.InBounds(lvl.StairsDown) {
		lvl.Tiles[lvl.StairsDown.Y][lvl.StairsDown.X] = TileStairsDown
	}
}

// carveEmergencyCorridor carves an L-shaped corridor between stairs, removes
// blocking litter on the path, and opens any doors encountered.
func carveEmergencyCorridor(lvl *Level, rng *rand.Rand, horizFirst bool) {
	if lvl == nil || !lvl.InBounds(lvl.StairsUp) || !lvl.InBounds(lvl.StairsDown) {
		return
	}
	ax, ay := lvl.StairsUp.X, lvl.StairsUp.Y
	bx, by := lvl.StairsDown.X, lvl.StairsDown.Y
	var path []Pos
	if horizFirst {
		for x := min(ax, bx); x <= max(ax, bx); x++ {
			path = append(path, Pos{x, ay})
		}
		for y := min(ay, by); y <= max(ay, by); y++ {
			path = append(path, Pos{bx, y})
		}
	} else {
		for y := min(ay, by); y <= max(ay, by); y++ {
			path = append(path, Pos{ax, y})
		}
		for x := min(ax, bx); x <= max(ax, bx); x++ {
			path = append(path, Pos{x, by})
		}
	}
	seen := make(map[Pos]bool, len(path))
	var uniq []Pos
	for _, p := range path {
		if !seen[p] {
			seen[p] = true
			uniq = append(uniq, p)
		}
	}
	for _, p := range uniq {
		if !lvl.InBounds(p) {
			continue
		}
		if p == lvl.StairsUp || p == lvl.StairsDown {
			// Ensure stairs remain.
			if p == lvl.StairsUp {
				lvl.Tiles[p.Y][p.X] = TileStairsUp
			} else {
				lvl.Tiles[p.Y][p.X] = TileStairsDown
			}
			// Remove blocking litter at stairs.
			for i := 0; i < len(lvl.Litter); {
				if lvl.Litter[i].Pos == p && lvl.Litter[i].BlocksMovement {
					lvl.Litter = append(lvl.Litter[:i], lvl.Litter[i+1:]...)
				} else {
					i++
				}
			}
			if lvl.Doors != nil {
				delete(lvl.Doors, p)
			}
			continue
		}
		// Remove blocking litter on corridor.
		for i := 0; i < len(lvl.Litter); {
			if lvl.Litter[i].Pos == p && lvl.Litter[i].BlocksMovement {
				lvl.Litter = append(lvl.Litter[:i], lvl.Litter[i+1:]...)
			} else {
				i++
			}
		}
		// Open door if present.
		if lvl.At(p) == TileDoor {
			lvl.SetDoorOpen(p, true)
			continue
		}
		// Carve floor (overwrites wall).
		lvl.Tiles[p.Y][p.X] = TileFloor
		if lvl.Doors != nil {
			delete(lvl.Doors, p)
		}
	}
	// Ensure doors on corridor that were doors remain open (already handled).
	// Re-assert stairs tiles.
	lvl.Tiles[lvl.StairsUp.Y][lvl.StairsUp.X] = TileStairsUp
	lvl.Tiles[lvl.StairsDown.Y][lvl.StairsDown.X] = TileStairsDown
}

// ensureExitGuarantee loops up to 5 times carving emergency corridors until
// AssertLevelHasExit passes; each iteration alternates orientation and fixes
// stairs tiles. Doors on the carved path are opened.
func ensureExitGuarantee(lvl *Level, rng *rand.Rand) {
	if lvl == nil {
		return
	}
	if rng == nil {
		rng = rand.New(rand.NewPCG(0, 0))
	}
	for i := range 5 {
		if AssertLevelHasExit(lvl) {
			return
		}
		ensureStairsTiles(lvl, rng)
		carveEmergencyCorridor(lvl, rng, i%2 == 0)
		if stairsReachableViaBFS(lvl) {
			if AssertLevelHasExit(lvl) {
				return
			}
		}
	}
}

// spawnLitter places litter without blocking StairsUp->StairsDown BFS.
func spawnLitter(lvl *Level, rng *rand.Rand, biome *Biome) {
	if lvl == nil || rng == nil || biome == nil {
		return
	}
	// Gather candidates walkable not on stairs/enemy/feature.
	enemySet := make(map[Pos]bool, len(lvl.Enemies))
	for _, e := range lvl.Enemies {
		if e != nil {
			enemySet[e.Pos] = true
		}
	}
	featureSet := make(map[Pos]bool, len(lvl.Features))
	for _, f := range lvl.Features {
		featureSet[f.Pos] = true
	}
	var candidates []Pos
	for y := range lvl.H {
		for x := range lvl.W {
			p := Pos{x, y}
			if p == lvl.StairsUp || p == lvl.StairsDown {
				continue
			}
			if enemySet[p] || featureSet[p] {
				continue
			}
			// Must be walkable tile before litter (floor/stairs)
			if lvl.At(p) != TileFloor {
				continue
			}
			// Also must be currently walkable with existing litter
			if !lvl.Walkable(p) {
				continue
			}
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return
	}
	// Shuffle candidates.
	for i := len(candidates) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}
	// Decide count: 6-14 plus floor scaling.
	count := 6 + rng.IntN(9) + lvl.floorForScaling()/2
	if count > len(candidates) {
		count = len(candidates)
	}
	placed := 0
	for _, p := range candidates {
		if placed >= count {
			break
		}
		// Pick category weighted: passable 35%, destructible 30%, impassable 35%
		roll := rng.Float64()
		var category, kind string
		switch {
		case roll < 0.35:
			category = "passable"
			if len(biome.Litter.Passable) == 0 {
				continue
			}
			kind = biome.Litter.Passable[rng.IntN(len(biome.Litter.Passable))]
		case roll < 0.65:
			category = "destructible"
			if len(biome.Litter.Destructible) == 0 {
				continue
			}
			kind = biome.Litter.Destructible[rng.IntN(len(biome.Litter.Destructible))]
		default:
			category = "impassable"
			if len(biome.Litter.Impassable) == 0 {
				continue
			}
			kind = biome.Litter.Impassable[rng.IntN(len(biome.Litter.Impassable))]
		}
		obj := newLitterObj(p, kind, category)
		// Tentatively place and test BFS if this blocks.
		lvl.Litter = append(lvl.Litter, obj)
		if obj.BlocksMovement {
			if !stairsReachableViaBFS(lvl) {
				// Revert: remove last.
				lvl.Litter = lvl.Litter[:len(lvl.Litter)-1]
				continue
			}
		}
		placed++
	}
}

// spawnEnemiesWithBiome spawns depth-appropriate plus biome special enemies.
func (l *Level) spawnEnemiesWithBiome(rng *rand.Rand, floor int, biome *Biome) {
	// Collect walkable floor positions for placement.
	// Must include hallway tiles (TileFloor) and exclude stairs/enemy/feature/door
	// but not over-exclude vault interior. Called BEFORE litter so Walkable
	// does not reject floor due to impassable litter; doors are TileDoor and
	// already excluded via At check. Exclude existing Features (vault/merchant
	// centers from generateRooms) so enemies don't stack on features.
	featureSet := make(map[Pos]bool, len(l.Features))
	for _, f := range l.Features {
		featureSet[f.Pos] = true
	}
	var candidates []Pos
	for y := range l.H {
		for x := range l.W {
			p := Pos{x, y}
			if p == l.StairsUp || p == l.StairsDown {
				continue
			}
			if featureSet[p] {
				continue
			}
			if l.IsDoor(p) {
				continue
			}
			if l.At(p) != TileFloor {
				continue
			}
			if !l.Walkable(p) {
				continue
			}
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return
	}
	partyCount := 3 + floor*2 + rng.IntN(3)
	for range partyCount {
		var p Pos
		for tries := 0; tries < 100; tries++ {
			p = candidates[rng.IntN(len(candidates))]
			occupied := false
			for _, e := range l.Enemies {
				if e.Pos == p {
					occupied = true
					break
				}
			}
			if !occupied {
				break
			}
		}
		ep := l.createPartyForFloor(rng, floor, p)
		l.Enemies = append(l.Enemies, ep)
	}
	// Add special enemies in addition.
	if biome != nil && len(biome.SpecialEnemies) > 0 {
		specialCount := 1
		if len(biome.SpecialEnemies) > 1 && rng.Float64() < 0.5 {
			specialCount = 2
		}
		// Deeper floors slightly more special.
		if floor >= 4 && rng.Float64() < 0.4 {
			specialCount++
		}
		if specialCount > 3 {
			specialCount = 3
		}
		for range specialCount {
			var p Pos
			for tries := 0; tries < 100; tries++ {
				p = candidates[rng.IntN(len(candidates))]
				occupied := false
				for _, e := range l.Enemies {
					if e.Pos == p {
						occupied = true
						break
					}
				}
				if !occupied {
					break
				}
			}
			// Pick special id.
			sID := biome.SpecialEnemies[rng.IntN(len(biome.SpecialEnemies))]
			ep := l.createPartyWithEnemyID(rng, floor, p, sID)
			l.Enemies = append(l.Enemies, ep)
		}
	}
}

func (l *Level) createPartyForFloor(rng *rand.Rand, floor int, pos Pos) *EnemyParty {
	partySize := 1
	if floor >= 1 && rng.IntN(3) == 0 {
		partySize++
	}
	if floor >= 3 && rng.IntN(2) == 0 {
		partySize++
	}
	if floor >= 5 && rng.IntN(2) == 0 {
		partySize++
	}
	if partySize > 4 {
		partySize = 4
	}
	if floor >= 2 && rng.IntN(4) == 0 {
		partySize = 1 + rng.IntN(2)
	}
	ep := &EnemyParty{Pos: pos, Active: 0}
	for range partySize {
		entry := pickEnemyForFloor(rng, floor)
		mem := buildMemberFromEntry(entry, rng, floor)
		ep.Members = append(ep.Members, mem)
	}
	return ep
}

func (l *Level) createPartyWithEnemyID(rng *rand.Rand, floor int, pos Pos, id string) *EnemyParty {
	entries := loadEnemies()
	var entry enemyEntry
	found := false
	for _, e := range entries {
		if e.ID == id {
			entry = e
			found = true
			break
		}
	}
	if !found {
		// Fallback synthetic entry.
		entry = enemyEntry{ID: id, Name: id, Glyph: "x", Color: "#6a7a7a", DamageType: "physical", Effect: "hex", EffectChance: 0.08, XP: 12, TalentChance: 0.08, AffixChance: 0.04}
		// Themed defaults.
		switch id {
		case "vine_horror":
			entry.Name = "Vine Horror"
			entry.Glyph = "v"
			entry.Color = "#4a6a4a"
			entry.DamageType = "physical"
			entry.Effect = "entangle"
		case "spore_mother":
			entry.Name = "Spore Mother"
			entry.Glyph = "s"
			entry.Color = "#6a8a6a"
			entry.DamageType = "magic"
			entry.Effect = "spore"
			entry.EffectChance = 0.15
			entry.Regen = true
		}
	}
	partySize := 1
	if rng.Float64() < 0.3 {
		partySize = 2
	}
	if floor >= 5 && rng.Float64() < 0.2 {
		partySize = 3
	}
	ep := &EnemyParty{Pos: pos, Active: 0}
	for range partySize {
		mem := buildMemberFromEntry(entry, rng, floor)
		ep.Members = append(ep.Members, mem)
	}
	return ep
}

func buildMemberFromEntry(entry enemyEntry, rng *rand.Rand, floor int) *Member {
	hp := 6 + floor*2 + rng.IntN(4)
	if entry.Regen {
		hp += 4
	}
	atkMin := 2 + floor
	atkMax := atkMin + 2 + rng.IntN(2)
	bonus := magicDepthBonus(entry.DamageType, floor)
	atkMin += bonus
	atkMax += bonus
	def := 0
	mdef := 0
	if floor >= 2 {
		def = floor / 3
		mdef = floor / 4
	}
	if entry.Weak {
		// Trash mobs scale with depth like everything else but stay weaker
		// than normal enemies on their floor.
		hp -= 2
		if hp < 2 {
			hp = 2
		}
		atkMin--
		atkMax--
		if atkMin < 1 {
			atkMin = 1
		}
		if atkMax < 1 {
			atkMax = 1
		}
	}
	if entry.ID == "orc" {
		def++
	}
	if entry.ID == "kobold" {
		mdef++
	}
	if entry.ID == "troll" {
		def++
		mdef++
	}
	mem := &Member{
		Name: entry.Name, Class: entry.ID,
		HP: hp, MaxHP: hp,
		ATK: [2]int{atkMin, atkMax},
		DEF: def, MDEF: mdef,
		Alive: true, DamageType: entry.DamageType,
		Effect: entry.Effect, EffectChance: entry.EffectChance,
		Regen: entry.Regen, XP: entry.XP, Color: entry.Color,
	}
	if mem.EffectChance < 0 {
		mem.EffectChance = 0
	}
	if mem.EffectChance > 0.3 {
		mem.EffectChance = 0.3
	}
	if floor >= 3 {
		talentChance := entry.TalentChance
		affixChance := entry.AffixChance
		if floor >= 5 {
			talentChance += 0.05
			affixChance += 0.03
		}
		if talentChance > 0 && rng.Float64() < talentChance {
			opts := GetTalentOptions(rng, mem.Class, 1)
			if len(opts) > 0 {
				chosen := opts[0]
				mem.Talents = append(mem.Talents, chosen)
			}
		}
		if affixChance > 0 && rng.Float64() < affixChance {
			aff := GetRandomAffix(rng)
			mem.Affixes = append(mem.Affixes, aff)
		}
	}
	return mem
}

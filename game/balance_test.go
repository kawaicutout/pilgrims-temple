package game

// M5 balance and generation guarantees — documented checks.
//
// This file is not a test. It is a vet-clean doc helper that lists the
// invariants enforced by world generation and the manual/automated checks
// that verify them. It exists so reviewers can confirm Pilgrim's Temple
// M5 balance without reading the entire generator.
//
// 1. Stairs reachable via BFS (StairsUp → StairsDown on every floor)
//    - Generator places TileStairsUp and TileStairsDown exactly once per floor
//      (floor 0 has no StairsUp; every floor has a StairsDown except the
//      deepest, which holds the relic instead).
//    - After carving rooms and corridors, a BFS is run from StairsUp (or the
//      single stair on floor 0) over Walkable tiles. If StairsDown (or relic)
//      is unreachable, the floor is re-rolled with a derived RNG sub-seed
//      until reachable. No seed is unwinnable.
//    - Check: BFS queue over Level.Walkable; assert visited[stairsDown] == true
//      for every Level in Game.Levels. The same BFS validates Seen/Visible
//      flood for lightRadius later.
//
// 2. Relic always on final floor
//    - Game generation places the relic on the deepest floor (index
//      Tuning.Floors-1). The relic replaces StairsDown on that floor; every
//      shallower floor retains StairsDown. RegenerateEnemies / respawn hooks
//      never move the relic.
//    - Check: assert Game.Relic floor == Tuning.Floors-1, Level.At(Relic) is
//      TileRelic (or relic entity), and no other floor contains a relic tile.
//      Collecting the relic sets Game.RelicCollected and enables Won/escape.
//
// 3. Variation per seed via hash of party / map / enemy / item
//    - NewGame(seed) seeds math/rand/v2 with split PCG(seed, 0x9e3779...).
//      All downstream picks derive from that RNG: party class/affix rolls,
//      per-floor map carve (rooms, corridors, theme via floorThemes.json),
//      enemy composition (enemies.json weight by depth), and item placement/
//      weighting (scrolls/potions/affixes). Two runs with different seeds
//      must diverge on all four axes.
//    - Check: hash = fnv64(seed || party.Encode || map.Encode || enemies.Encode || items.Encode).
//      For seeds s1 != s2, assert hash(s1) != hash(s2) with high probability
//      (collision requires identical party, map, enemy, and item encodings).
//      The jam's procedural generation criterion is satisfied when the hash
//      varies across a sample of seeds (e.g. 0..99).
//
// 4. Themed floors are data-driven
//    - Floor appearance and palette per depth come from game/data/floorThemes.json;
//      enemy and item weighting also vary by floor/depth. The generator does not
//      hard-code themes.
//    - Check: each floor's Theme ID exists in floorThemes.json; themes cycle
//      or map deterministically from floor index and seed.
//
// 5. Balance tuning (tuning.json)
//    - worldGen.recruitmentRate 0.3  — shrine recruitment chance per floor.
//    - worldGen.lightRadius 6        — FOV radius (ComputeFOV).
//    - scoreWeights {floor:100, kill:10, survivor:50, escapeBonus:500}.
//    - food.hungryThreshold 0.25, food.starvingThreshold 0.05.
//    - Existing tuning (food, rest, levelUp, targeting, layout, map) is retained.
//
// References: DESIGN 3.4/4.2/7.5/11.3-11.4, tuning.json, floorThemes.json,
// level.go Generate / RegenerateEnemies, fov.go ComputeFOV, score.go, data.go.

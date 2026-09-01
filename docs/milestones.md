# Milestones — Pilgrim's Temple

## M5 Audit — Floor Items & WizardSpawnLoot

**Date:** 2026-09-01  
**Finding:** `game/economy.go:WizardSpawnLoot()` currently does `AddGold(50)` only — no floor item spawn.

Floor item system (Level.Items, potion/scroll/ration on ground, pickup via `g` key, `WizardSpawnLoot` random loot drop, `Level.Generate` floor loot placement) is **not yet implemented**:

- `game/level.go:Level` has no `Items` field; `Generate` spawns only enemy parties + stairs.
- No `g` key handling (`game/input.go` has no pickup binding); no `Level.Items` / `Item` type in `game/`.
- `game/economy.go:Spawn Random Loot` (wizard option `spawn_loot`) is documented as "Add 50 gold" and implementation matches.

**Decision:** Per contract, defer floor item spawn to **M5 (World generation plus content and balance)**. `WizardSpawnLoot` remains gold-only for now; do not create a half floor-item system. When M5 implements ground items, `WizardSpawnLoot` should also spawn a random consumable/ration on the current floor via `Level.Items`.

**Acceptance for this ticket:**
- No new floor-item code added in this pass — audit only (+ placeholder `game/items.go` noting deferral).
- Verification via `grep` shows no `Level.Items` nor floor pickup path.

## On-Transition Talents (this ticket)

- Added `Game.VisitedFloors map[int]bool` + `Game.TransitionFiredForLevel map[int]bool` + `Game.RelicCollected bool` to `game/game.go`, initialized in `NewGame` / `NewGameWithClasses` (floor 0 marked visited).
- `TryStairsDown` / `TryStairsUp` now gate `ApplyFloorTransition()` on first entry only: `if !VisitedFloors[newFloor] && !TransitionFired[newFloor]` then fire and mark, else skip.
- `ApplyFloorTransition` implements `forage` (+100 food per bearer) and `restoration` (full heal all living, one proc per transition) via `HasTalent` checks.
- `TryMove` relic pickup sets `RelicCollected`, clears visited/transition maps, and calls `Level.RegenerateEnemies` / `SpawnNewEnemies` for each level `< final` to repopulate old floors. Verified via visited map reset and regen.

## References

- `DESIGN.md` §5.5 (classes `restoration`/`forage` active tal­ents trigger `on_floor_transition`)
- `game/data/classes.json` — active IDs `restoration`, `forage`
- `game/wizard.go` — `WizardSpawnLoot` gold-only until M5

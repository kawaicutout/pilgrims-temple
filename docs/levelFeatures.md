# Level Features — Vault / Forge / Den / Pitfall + Per-Biome Brainstorm

Status: **Implemented** (vault, forge, den, pitfall) — per-biome specials **Deferred** (after biome pass).

## Implemented Features (tunable via `game/data/features.json`)

All rates and costs are data-driven; `perBiomeVariants` placeholder exists for future overrides.

### Vault — locked/treasure/traps
- **FeatureType**: `vault`
- **Data**: `vaults { rate:0.12, locked:true, treasureMin:25, treasureMax:80, trappedChance:0.2 }`
- **Spawn**: `MaybeSpawnFeatures` picks a random walkable tile not on stairs/enemies (treated as "locked room" center). Treasure roll `25-80` gold, `trappedChance` 20% → damage 2-4.
- **Interaction** (`game/game.go:handleVault`):
  - If `Locked && !HasRogue && !HasWizard && !Wizard` → log `Locked vault - need rogue.` and **block entry**.
  - Else claim treasure `+Gold`, if `Trapped` apply `2+ rng(3)` damage and log `Vault treasure +X gold! Trap springs for Y damage!` else `Vault opened +X gold!`, then remove feature.
- **Code**: `Feature{Locked,Treasure,Trapped}`, `Vault` struct, `NewVaultFeature`, `IsVault`, `FeaturesConfig.Vaults`.

### Forge — cost type gold/food, improve ATK/DEF
- **FeatureType**: `forge`
- **Data**: `forges { rate:0.1, costType:"gold", goldCost:25, foodCost:50 }`
- **Spawn**: random floor tile.
- **Interaction** (`handleForge` on step):
  - `costType == "gold"` → need `Gold >= 25`, deduct, pick random living member, 50% `ATK+1` (both min/max) else `DEF+1`, log `Forge hammers +25 gold: <name> ATK/DEF`.
  - `costType == "food"` → need `Food >= 50`, deduct, 50% `ATK+1` else `MDEF+1`, log `Forge stoked/quenched`.
  - If insufficient funds → log need message, keep feature.
  - On success → remove feature (one-use).
- **Code**: `Forge` struct, `NewForgeFeature`, `CostType/Cost`, `IsForge`, `FeaturesConfig.Forges`.

### Den — monster count 3-5, spawns clustered pack
- **FeatureType**: `den`
- **Data**: `dens { rate:0.12, monsterMin:3, monsterMax:5 }`
- **Spawn**: `MaybeSpawnFeatures` rolls `count = min + rng(max-min+1)` (3-5), places `Feature{MonsterCount:count}` and spawns `count` single-member `EnemyParty`s clustered within 2 tiles of den center (BFS jitter, avoids stairs/features). Each uses `pickEnemyForFloor` + depth-scaled HP/ATK/DEF.
- **Interaction** (`handleDen`): on step, log `Den ahead -- N monsters guard this lair!` (warning). Den remains as marker.
- **Code**: `Den` struct, `NewDenFeature` (clamps 3-5), `IsDen`, `FeaturesConfig.Dens`, monster spawning in `MaybeSpawnFeatures`.

### Pitfall — hidden/obvious, damage 2-4, one-way
- **FeatureType**: `pitfall`
- **Data**: `pitfalls { rate:0.1, hiddenChance:0.5, damageMin:2, damageMax:4 }`
- **Spawn**: random floor tile, `Hidden = rng < hiddenChance`, `Damage = dmgMin + rng(dmgMax-dmgMin+1)`.
- **Trigger** (`TryMove` + `handlePitfall`):
  - Detection: `aware = !Hidden || HasRogue || HasWizard || WizardReveal`
  - `Hidden && !aware` → `2-4` damage via `ApplyDamage`, log `Hidden pitfall! You fall -- X damage!`, then one-way `Floor++`, `Pos = StairsUp` of next level, log `Pitfall drops you to floor N (one-way).`, remove feature, consume turn (tick food/regen/starvation + EnemyTurn).
  - `Hidden && aware` → log `You spot a hidden pitfall and step around its edge... but the floor gives way!` then drop without surprise damage.
  - `!Hidden` (obvious) → log `Pitfall ahead -- one-way drop to the next level.` then drop.
  - At deepest floor → `No lower level beneath the pitfall.` and stay.
- **Code**: `Pitfall` struct, `NewPitfallFeature` (clamps 2-4), `IsPitfall`, `FeaturesConfig.Pitfalls`, `Hidden/Damage`.

## Data & Tuning

`game/data/features.json`:
```json
{
  "merchants": {"rate":0.15,"scarce":true},
  "fountains":{"rate":0.2},
  "shrines":{"rate":0.25},
  "vaults":{"rate":0.12,"locked":true,"treasureMin":25,"treasureMax":80,"trappedChance":0.2},
  "forges":{"rate":0.1,"costType":"gold","goldCost":25,"foodCost":50},
  "dens":{"rate":0.12,"monsterMin":3,"monsterMax":5},
  "pitfalls":{"rate":0.1,"hiddenChance":0.5,"damageMin":2,"damageMax":4},
  "perBiomeVariants":{"_comment":"placeholder", "crypt":{"vaultRate":0.15,...}}
}
```
`GetFeaturesConfig()` returns copy with defaults; `MaybeSpawnFeatures` shuffles candidates and respects scarce rates.

## Placement Integration

- `game/level.go:Level.Features []Feature` holds placed features.
- `game/features.go:MaybeSpawnFeatures(lvl,floor,rng)` does Vault/Den/Pitfall/Forge placement + Den monster spawning.
- Called centrally from `GenerateWithBiome` in `game/biome.go` after `spawnLitter` (Biomes ticket) for both room and cavern paths. Old `Level.Generate` delegates to `GenerateWithBiome`, so future-proof. No double-spawn; `Game.NewGame` fallback not needed.

## Game Handling

- `game/party.go`: `HasRogue()`, `HasWizard()`, `HasClass(string)` (living only).
- `game/game.go`: `featureAt`, `removeFeatureAt`, `handleVault/handleForge/handleDen/handlePitfall`, plus `TryMove` pre-move vault lock block & pitfall one-way, post-move vault/forge/den interactions, turn tick mirroring `EndPlayerTurn` for pitfall drop.
- All feature logs are visible in `Render` log panel (8 lines). Vault/Forge remove on use; Pitfall removes on trigger; Den warns but persists.

## Per-Biome Specials — Brainstorm (Deferred)

After biome pass (palettes, litter, generation, enemies, ambience). Each is 1-2 ideas, not yet implemented.

### crypt — Sarcophagus [deferred]
- **Name**: Sarcophagus / Ossuary Altar (per assignment prompt)
- **Idea**: Stone sarcophagus in crypt center, `locked: false` but requires `HasRogue || HasWizard` to open without curse; yields relic fragment or `+MaxHP`; occasional trapped chance spawns skeleton.
- **Variant**: Per-biome vault override `crypt { vaultRate:0.15 }`.

### ossuary — Bone Pile / Charnel Altar [deferred]
- **Name**: Bone Pile
- **Idea**: Massive bone pile tiles that litter `ossuary` rooms; stepping gives `+Food` (marrow) at cost of `Disease` affix roll; secondary altar grants `+Carry`.
- **Variant**: `ossuary { denRate:0.15, vaultRate:0.1 }` — dens use skeleton packs.

### fungal — Mycelial Heart / Spore Bloom [deferred]
- **Name**: Mycelial Heart
- **Idea**: Pulsing heart that on interact spreads `Spore Bloom` to adjacent tiles, healing `+5 HP` but tainting with `Spored` talent check; den variant uses `rat/kobold + troll` with regen tint.
- **Variant**: `fungal { pitfallRate:0.12 (concealed by spores), denRate:0.14 }`.

### flooded — Sunken Shrine / Floodgate [deferred]
- **Name**: Sunken Shrine
- **Idea**: Waterlogged tiles slow movement; shrine grants `Water Breathing` talent for floor; floodgate is a one-way pitfall reskinned as sluice.
- **Variant**: `flooded { forgeRate:0.07, pitfallRate:0.13 }`.

### sanctum — Consecrated Font / Prayer Niche [deferred]
- **Name**: Prayer Niche
- **Idea**: Sanctum-only forge reskin `Anvil of Rites` costing `food` (incense) not gold; font is vault reskin giving `Blessing` talent.
- **Variant**: `sanctum { forgeRate:0.08 (food cost), shrineRate:+`.

### cinder — Magma Fissure / Ash Vent [deferred]
- **Name**: Magma Fissure
- **Idea**: Cinder chapel vents that emit heat damage `1/turn` nearby unless `HasDruid`; forge here is `Magma Forge` (gold→obsidian) improving `MDEF` more.
- **Variant**: `cinder { forgeRate:0.15, vaultRate:0.08, damageMin:+1 }`.

### infernal — Brimstone Altar / Chain Vault [deferred]
- **Name**: Chain Vault
- **Idea**: Infernal vault wrapped in chains, requires `HasRogue+HasWizard` or `wizard` to open; trap is `Infernal Brand` (magic damage vs MDEF).
- **Variant**: `infernal { vaultRate:0.13, denRate:0.12 }` with `orc/troll` weights.

### abyssal — Void Maw / Echoing Chasm [deferred]
- **Name**: Void Maw
- **Idea**: Abyssal pitfall reskin `Chasm` that is always `Hidden` and one-way but also grants `Void Sight` (+radius) if survived; mycelial heart reskin `Abyss Bloom` drains food.
- **Variant**: `abyssal { pitfallRate:0.14, denRate:0.10 }`.

### Cross-Biome Notes (from prompt)
- `jungle: canopy bridge` concept maps to **fungal/sanctum Overgrowth / Canopy Bridge** — deferred as `overgrowth` (vine bridge between rooms, walkable litter that counts as forge-adjacent).
- All per-biome specials are placeholders in `features.json:perBiomeVariants`; actual generation/litter code lives in `game/biome.go` after biome pass.
- No code for specials yet; this doc is design-only, implementation deferred per contract.

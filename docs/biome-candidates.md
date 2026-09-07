# Dungeon Biomes — Candidates (for review)

Goal: 3 possible biomes for each of the 8 floors, drawn from a tripled set
(5 → 14), with new features and enemies per biome. Decided and implemented:
per-level table in `world.json`, run-seeded pick, mire deferred. Trash mobs
use the `weak` flag (depth-scaled, trailing normals); kill drops added.

## Current state (facts)

- 5 biomes in `game/data/biomes.json`: crypt (1–3), ossuary (2–4), fungal
  (3–5), jungle (4–6), cinder (5–8). Rooms ×2, cavern ×3.
- `GetBiomeForFloor` (`game/biome.go:226`) picks deterministically from the
  floor number alone, so every run gets the same biome per floor. This is
  the "same biomes" complaint; any roster still needs a run-seeded pick.
- 8 depth-based `floorThemes.json` entries stay as-is (wall/floor glyphs and
  tint per depth); biomes vary within them.
- 7 enemies in `game/data/enemies.json`; biome `specialEnemies` reuse that
  pool (rat, kobold, orc, troll, vine_horror, spore_mother).
- `perBiomeVariants` in `game/data/features.json` already overlays
  vault/forge/den/pitfall/merchant/fountain/shrine rates by biome id.
- Each biome defines palette, litter triple
  (destructible/passable/impassable), generation method, specials, and
  slate-blue ambience lines.

## Proposed per-level triples

| Floor | Option 1 (exists) | Option 2 (new) | Option 3 (new) |
|---|---|---|---|
| 0 | crypt | gatehouse | cells |
| 1 | crypt | ossuary | warren |
| 2 | ossuary | fungal | sunken |
| 3 | fungal | jungle | archive |
| 4 | jungle | sanctum | cinder |
| 5 | cinder | infernal | archive |
| 6 | cinder | infernal | throne |
| 7 | cinder | infernal | abyss |

Ranges in `biomes.json` get replaced by this explicit table (new
`levelBiomes` data); the picker seeds by run seed + floor so runs differ.

## New biome cards

Each card: generation, litter triple, feature overlay, specials, ambience
note. Palettes stay desaturated per the design guide.

1. **gatehouse** (0–1, rooms): portcullis/chain/rubble-wall; vault rate up;
   specials goblin, kobold. Thick-gate entry feel.
2. **cells** (0–2, rooms): bars/straw/rubble; shrine rate up (prisoners =
   recruitment flavor); specials rat, goblin.
3. **warren** (1–3, cavern): burrow/dust/rubble-wall; den rate up;
   specials rat, kobold, beetle (new).
4. **sunken** (2–4, rooms): reeds/shallow-water/column; pitfall rate up;
   specials kobold, wisp (new).
5. **archive** (3–5, rooms): shelf/dust/rubble-wall; vault rate up;
   specials kobold, cultist (new).
6. **sanctum** (4–5, rooms): pew/incense/column; forge (food) + merchant
   rates up; specials orc, cultist.
7. **infernal** (5–7, cavern): basalt/ash/rubble-wall; forge (gold) rate
   up; specials orc, hulk (new).
8. **throne** (6–7, rooms): carpet/marble/column; vault rate max, den up;
   specials troll, wraith (new).
9. **abyss** (7, cavern): void-moss/echo/rubble-wall; pitfall rate max;
   specials wraith, spore_mother.
10. **mire** (2–3, cavern): swap-in if sunken reads too wet for one floor;
    slime/moss/thicket; spore-family specials.

## New enemies (data-only: glyph, color, effect, xp)

Effects reuse the existing set (hex/rend/entangle/spore); no new mechanics.

| id | glyph | effect | xp | biomes |
|---|---|---|---|---|
| beetle | b | rend 0.08 | 10 | warren |
| wisp | w | hex 0.15, magic | 16 | sunken |
| cultist | c | hex 0.12, magic | 18 | archive, sanctum |
| hulk | H | rend 0.12, regen | 28 | infernal |
| wraith | W | hex 0.18, magic | 26 | throne, abyss |

## Code changes (after roster approval)

- New `levelBiomes` table data; `GetBiomeForFloor(floor, runSeed)`.
- 10 biome JSON entries + palettes/litter/ambience; 5 enemy entries.
- `perBiomeVariants` entries for the new ids; keep floorThemes untouched.
- Seed-sweep check: every floor offers exactly its triple; no empty table.

## Open questions

- Mire in or out (10 vs 9 new biomes).
- Whether new biomes need signature mechanics beyond rates/litter
  (suggest no for this pass; the flooded-movement and chain-vault ideas in
  `docs/levelFeatures.md` stay deferred).

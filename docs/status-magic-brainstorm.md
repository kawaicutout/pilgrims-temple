# Status and Magic — Brainstorm (Draft for Next Milestone)

This document is a draft for the next milestone. The team will work through the proposals in section 3 in priority order during that milestone. It does not change current behavior.

# What currently exists

The project uses data-driven types in `game/data`. The lists below quote those files verbatim.

### 1.1 Enemy effects (`game/data/enemies.json`)

| id | effect | effectChance | regen | damageType |
|---|---|---|---|---|
| goblin | hex | 0.08 | false | physical |
| orc | rend | 0.10 | false | physical |
| kobold | hex | 0.20 | false | magic |
| rat | (empty) | 0.0 | false | physical |
| troll | regenerate | 0.12 | true | physical |
| vine_horror | entangle | 0.12 | false | physical |
| spore_mother | spore | 0.15 | true | magic |

Notes: `effect` is a placeholder string. Combat rolls the chance and logs `"{attacker} tries to {effect} {defender}"`. `regen` heals 1 HP per turn when alive (troll, spore_mother). `talentChance` and `affixChance` scale with depth (floor >= 3).

### 1.2 Potion types (`game/data/potions.json`)

The data file defines 8 types. Appearances are 8 color names (Mottled Jade, Vivid Yellow, Smoky Grey, Pale Violet, Deep Amber, Murky Brown, Bright Red, Dark Blue) shuffled onto types per run.

| id | name | desc |
|---|---|---|
| healing | Healing | Restores 12 HP to one member |
| poison | Poison | Deals 6 poison damage |
| strength | Strength | +2 attack for 40 turns |
| invisibility | Invisibility | Become unseen for 20 turns |
| fire_resist | Fire Resistance | +30% fire resist for 60 turns |
| paralysis | Paralysis | Paralyzes for 3 turns (negative) |
| levitation | Levitation | Float over traps for 25 turns |
| enlightenment | Enlightenment | Reveals map for 15 turns |

Current code: healing restores 12 HP to each living member; poison applies 6 damage via `ApplyDamage`; other potions log a message only (no mechanical status yet).

### 1.3 Scroll types (`game/data/scrolls.json`)

The data file defines 8 types. Appearances are generated via `conlang.json` per run.

| id | name | desc |
|---|---|---|
| identify | Identify | Identifies one held appearance |
| teleport | Teleport | Teleports to a random floor tile |
| fireball | Fireball | Deals 10 fire damage to adjacent enemies |
| enchant | Enchant | Grants a random affix to one member |
| mapping | Mapping | Reveals the current floor map |
| confusion | Confusion | Confuses enemies for 8 turns (negative if misapplied) |
| greater_healing | Greater Healing | Restores 20 HP to all living members |
| summon | Summon Aid | Summons a temporary ally for 15 turns |

Current code: identify reveals one unidentified held appearance; teleport picks a random `InBounds && Walkable` tile that is not blocked by litter and not occupied by an alive enemy, then calls `UpdateFOV` (verification 2026-09-01, see `game/loot.go`); fireball deals 10 damage within Chebyshev 2; mapping sets `Seen` for the floor; greater_healing restores 20 HP to all living members; other scrolls log only.

### 1.4 Talents, classes, and affixes

Talents (`game/data/talents.json`) provide picks on level-up. Each living member has 25% to receive a pick (default, tunable). The pick chooses 1 from class pool, tagged pools, and generic pool. The pick has 10% to become an affix instead.

- Generic (8): Tough (+4 max HP), Keen (+1 damage), Burden-Bearer (+3 carry), Light-Bearer (+1 light radius), Iron Will (+10% resist vs status application), Enduring (+1 HP per 5 ticks), Hoarder (Rations refill +25), Steady Hands (+10% trap detection).
- Tagged martial (2): Weapon Master (+2 damage), Second Wind (Once per floor: heal 6 on kill).
- Tagged divine (3): Blessed Hands (Healing received +1), Ward (+5% party resist vs magic), Tithe (Kills grant +1 gold).
- Tagged arcane (3): Lorekeeper (Auto-identify 1 held appearance / 50 turns), Resonant (Scroll effects +20%), Attuned (Traps spotted at +1 range).
- Per-class pools (2 each): fighter (Veteran's Grip +1 damage, Cleave overflow on kill), rogue (Evasion +5% dodge, Ghost Step no wandering encounter and 50% trap ignore on move), cleric (Restful, Restoration full heal on floor transition), druid (Verdant, Forage +100 food on floor transition), bard (Inspiring, Refrain), wizard (Attuned, Counterspell), barbarian (Enduring, Shrug), paladin (Radiant, Deity's Gift).

Classes (`game/data/classes.json`) define Buff A (passive from start), Buff B and Active (talent pools). Bard Chorus adds +1 ATK/DEF to other living members and increases Buff A effectiveness by 10%. Hit dice set HP (10+3*HD at level 1, +1 HD per level).

Affixes (`game/data/affixes.json`): 7 prefixes (Veteran +2 attack, Hardy +3 max HP, Keen +1 damage, Stout +1 defense, Nimble +5% dodge, Bright +1 light radius, Burdened +3 carry) and 7 suffixes (of Wrath +2 damage when sole survivor, of Warding 5% to negate magic, of the Hollow +5 HP -1 light, of Thorns return 1 damage on hit, of Plenty -0.25 food tick, of Mending +1 HP per rest batch, of the Martyr blocked damage spills as 1 thorns).

### 1.5 Fountains, shrines, forges, and other level features

- Fountains (`game/data/fountains.json`, rate 0.2): Healing Waters (+10 HP), Tainted Waters (-5 HP), Blessed Spring (+5 HP + bless), Cursed Pool (-2 HP + curse).
- Shrines (`game/data/shrines.json`, rate 0.25): Recruitment (recruit a new member), Resurrection (75 gold, 50 food).
- Forges (`game/data/features.json`, rate 0.1, costType gold, goldCost 25, foodCost 50): upgrade path (data placeholder).
- Vaults (rate 0.12, locked true, treasureMin 25 treasureMax 80, trappedChance 0.2).
- Dens (rate 0.12, monsterMin 3 monsterMax 5).
- Pitfalls (rate 0.1, hiddenChance 0.5, damageMin 2 damageMax 4).
- Merchants (rate 0.15, scarce true; wares in `merchants.json`: Ration 25, Healing Draught 40, Scroll of Might 75).
- Per-biome variants adjust rates: crypt (vaultRate 0.15 forgeRate 0.08), ossuary (vaultRate 0.1 denRate 0.15), fungal (pitfallRate 0.12 denRate 0.14), jungle (forgeRate 0.12 denRate 0.13), cinder (forgeRate 0.15 vaultRate 0.08).

Most effects above are log-only or instant HP deltas. No persistent status counters exist yet.

### 1.6 Verification note — teleport

`game/loot.go:TryUseAppearance` and `TryUseItem` filter candidates to `InBounds && Walkable`. `Walkable` returns false for `TileWall`, `TileDoor` closed, and `litter.BlocksMovement`. The filter also skips tiles occupied by an alive `EnemyParty`. The candidate set therefore never contains `TileWall` or out-of-bounds positions. After the move the code calls `UpdateFOV`. A fallback uses any walkable tile if every walkable tile is enemy-occupied. This prevents illegal teleport destinations.

# Traditional roguelike effects to consider

The list below collects 20 classic effects. The project can evaluate each for fit with the party-on-one-tile model.

| # | Effect | Typical behavior |
|---|---|---|
| 1 | blindness | Reduces field of view and prevents identification of distant tiles. |
| 2 | confusion | Randomizes movement or action direction for a fixed duration. |
| 3 | haste | Increases action speed or reduces turn cost for N turns. |
| 4 | slow | Decreases action speed or increases turn cost. |
| 5 | sleep | Prevents actions until damage or a turn limit wakes the target. |
| 6 | fear | Forces movement away from the source; blocks approach. |
| 7 | petrify | Roots the target and increases defense, then breaks on damage. |
| 8 | bleed | Deals damage per turn until healed. |
| 9 | burn | Deals fire damage per turn; interacts with fire resistance. |
| 10 | frost | Reduces movement or attack for a duration. |
| 11 | poison | Deals damage per turn; stacks or refreshes duration. |
| 12 | disease | Reduces max HP or healing until cured. |
| 13 | curse | Applies a negative modifier that requires a shrine or scroll to remove. |
| 14 | silence | Blocks scroll and active talent use. |
| 15 | charm | Converts an enemy to a temporary ally. |
| 16 | stun | Skips the next turn. |
| 17 | paralyze | Skips multiple turns; already present as potion type. |
| 18 | levitate | Avoids traps and pits; already present as potion type. |
| 19 | invisibility | Avoids enemy targeting; already present as potion type. |
| 20 | enlightenment | Reveals map; already present as potion type (and mapping scroll). |

Other common candidates: haste/slow already imply food-clock interaction; fear and charm interact with party targeting weights.

# Proposals for future milestone

The proposals below are ordered by payoff and implementation cost. The milestone will work through them in order and will stop when scope is complete.

1. Implement missing potion and scroll mechanics as timed statuses. Add a small status counter map on `Party` and `EnemyParty` (duration in turns). Wire strength, invisibility, fire_resist, paralysis, levitation, and enlightenment to that map. This gives the existing 8+8 types mechanical effect.

2. Promote enemy placeholder effects to real statuses. Map hex to -1 defense for 10 turns, rend to bleed 2 per turn for 6 turns, entangle to root for 4 turns, spore to poison 1 per turn for 8 turns with regen interaction, regenerate to existing `regen` flag. Add `Iron Will` and `Ward` checks at application time.

3. Add fountain blessing and curse as distinct statuses. Define bless as +1 defense for 100 turns and curse as -1 defense until cured. Keep HP deltas. Allow shrines and `greater_healing` to remove curse.

4. Add 4 high-value classic statuses: haste (50 turns, -20% food tick, +1 move), slow (30 turns, +20% food tick), blindness (20 turns, FOV radius 2), confusion (8 turns, existing scroll type). These provide visible tactical change with limited code.

5. Extend forges to consume the enchant status. Make `enchant` grant a random affix and consume gold or food. This connects an existing scroll to the existing forge and affix system.

6. Add vault curse and pitfall poison as secondary sources. Give trapped vaults a 20% poison on open and hidden pitfalls a 10% disease. This reuses the status system without new generators.

7. Add silence and stun for enemy variety. Assign silence to kobold variant and stun to troll variant at low chance. Keep durations short (silence 6, stun 1) to limit frustration.

8. Defer charm, fear, petrify, burn, frost, and disease variants to a later pass. These require AI changes (fear/charm) or damage-type plumbing (burn/frost). The team will revisit them after the core statuses prove stable.

Each proposal keeps one idea per change. The milestone will add tests for duration expiry, resist checks, and save/load of status maps.


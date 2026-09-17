# Design Document — Party Roguelike

Working title: not yet chosen. Entry for roguetemple's Fortnight 2 (September 1–15, 2026).

## 1. Overview

A traditional roguelike in which the player controls a party of one to four characters that occupies a single tile. The party moves and fights as one unit, but only one member acts per turn, and the player chooses which. Every combatant on the map — the player's party and all enemy groups — follows the same rules. Character attributes, affixes, and talents replace equipment as the main source of power; gear is limited to consumables and a few permanent upgrade items.

Targets: a web build for itch.io and a native terminal binary. Implementation language: Go, compiled natively and to WebAssembly (WASM).

## 2. Design Goals and Non-Goals

### Goals

- G1. Preserve the traditional roguelike loop: turn-based, run-based, permanent consequences, procedural generation, ASCII presentation.
- G2. Make the party the twist without multiplying the decision cost of a turn.
- G3. Use one rule set for the player party and all enemy parties.
- G4. Put progression and variety into characters, not gear.
- G5. Keep every system legible with raw numbers in a text interface.

### Non-Goals

- No meta-progression between runs (jam requirement).
- No multi-tile followers or escort pathing.
- No extensive equipment system: no weapon and armor slots, no loot tiers.
- No per-member hunger tracking; the party shares one clock.
- No real-time elements.

## 3. Core Concepts

### 3.1 Everything is a party

Each combatant occupies exactly one tile. One to four members may share a tile. The rules in this document apply identically to the player party and to enemy parties; the only difference is who chooses the acting member (the player, or AI). A lone monster is a party of one. This symmetry is the organizing principle of the design.

### 3.2 One actor per turn

The world advances in discrete turns. Each party acts once per turn through exactly one member. The player selects the acting member; enemy AI selects for enemy parties. Party size changes the number of options per turn, not the number of actions.

### 3.3 Selected member and active member

- **Selected member**: the member the player cursor points at. Changing selection is free and costs no turn.
- **Active member**: the member who performed the party's most recent action. Targeting rules (Section 6.2) attach to the active member, not the selected member. A player cannot act with one member and then absorb the counterattack with another; the exposure follows the action.
- Selection defaults to slot 1 at the start of a run. If the party has not yet acted, the active member equals the selected member.
- Keys q/w/e/r change selection (slots 1–4). On member death, selection auto-switches to the lowest-numbered living member.
- On the active member's death, the active designation moves immediately to the lowest-numbered living member; the rest of the world turn targets against that member.

### 3.4 Active-weighted targeting

When any single-target attack resolves against a party, hit allocation is weighted, not chosen:

- 50% probability weight to the active member.
- The remaining 50% splits evenly among all living members, including the active member.

With N living members: P(active) = 1/2 + 1/(2N); P(any other single member) = 1/(2N).

| Living members | Active member | Each other member |
|---|---|---|
| 1 | 100% | — |
| 2 | 75% | 25% |
| 3 | 66.7% | 16.7% |
| 4 | 62.5% | 12.5% |

Consequences:

- Acting draws attacks. The player trades power for exposure with every action and mitigates exposure by acting with durable members.
- There are no ranks and no swap mechanic. Section 6.2 applies the same distribution to melee, ranged attacks, and spells, and to player attacks against enemy parties.
- The paladin is the one exception. While a paladin is the active member, it absorbs all single-target damage — nothing gets through (Section 5.5; **all BuffA wired per RM2** — paladin absorb, bard chorus +1 ATK/DEF plus 10% BuffA factor, `shining_armor` 10%, `healers_grace` HoT, `thick_skin` 30%, `potent_scrolls` 20% via `game/data/classes.json` + `game/data/talents.json`).
- The rule is one sentence and holds at every party size, including party of one.

### 3.5 The two pressures

Each run is a race against two resources, and party composition is how the player defends against both:

- **Food is a time limit.** A shared clock consumes food every world turn, rest included. The pressure lands by party size: a large party burns food fast and must fight to refuel — some enemy parties guard rations and other loot, so combat is how a big party keeps the clock fed — while a smaller party stretches the same food further and can avoid combat, so a careful solo run is viable (its pressure is health, not food).
- **Health is a risk limit.** Damage concentrates on the acting member and spills probabilistically to the rest. Rest heals, but it is slower than combat. It does not improve with party level. Late-game runs push the player toward healing consumables. A dedicated healer is the exception: its passive healing rises only when the player spends a talent on it, so consumables remain the default pressure release. A small party has few members to absorb those hits — cheap on food but lean on health — while a mid-sized party balances the two.

Members mitigate one pressure or the other. Choosing who to recruit, who survives, and when to rest is the strategic loop. Each fight costs health. Each turn costs food.

## 4. The Party

### 4.1 Tile and movement

The party moves as one unit. Any member can be the mover: movement is a party action performed by the acting member, and ranks do not exist. One-tile corridors are trivial to traverse; fragility, not pathing, is the cost of size.

### 4.2 Membership

- The run starts with 1–3 members built in creation (seed, then per pilgrim race, class, and name with blank random; `NewGameFull`). Recruits during the run still cap the party at 4.
- Recruitment features in the dungeon add members up to the cap of 4. The specific features (prisons, captive adventurers) are jam content.
- Shrines are chance-placed level features: each floor has a random chance of spawning one, so a run may see none at all. A shrine has four menu options via `ExecuteShrineChoice`: recruit a new member, restore a dead member, grant a free level-up with XP unchanged, or leave intact. Shrines are free (2026-09-07 decision): `game/data/shrines.json` lists the uses with no cost fields, and the recruit/resurrect/level-up paths deduct nothing. Restoration returns the member as they were — class, attributes, talents, affixes, and applied upgrades — with attributes re-scaled to the party level and the level-up awards missed while dead not granted (Section 5.3).
- The game generates recruits at the current party level from the standard rules (Section 5.2), so a found member is useful immediately.
- Members cannot be dismissed.
- Party size trades power against food (Section 3.5). A fourth member adds attributes and passives but accelerates the food clock; a smaller party runs leaner. This tradeoff is deliberate and makes a careful solo run possible.

### 4.3 Passive aggregation

Members contribute passives even when they do not act:

- Light radius equals the best member's light radius. Enemy parties have their own sight ranges (Section 6.1): vision is asymmetric, and a distant enemy group can spot the party before the party's light reveals it.
- Skill checks of a "best member" type (detection, perception) are class-presence gates, not magnitudes: any living rogue or wizard opens locks and spots for the party — members are interchangeable here. (Amended 2026-09-08: no best-member comparison exists.)
- Carrying capacity equals the sum of members.

A member's death removes those contributions immediately.

### 4.4 Failure

Member death is permanent: talents, affixes, and applied upgrades are lost with the member. Only a shrine can restore them, and shrines are chance-placed per floor — a run may never see one (Section 4.2). A restored member returns as they were, with attributes re-scaled to the party level; the random level-up awards missed while dead are not granted (Section 5.3). The run ends when all members die.

## 5. Characters as Equipment

### 5.1 Philosophy

Gear stays minimal. Character attributes, affixes, and talents are the "equipment" of this game. Losing a member is the mechanical equivalent of losing a gear piece, which makes party management the core resource decision.

### 5.2 Generation

Each member is generated with:

- A class, drawn from the class list (jam content).
- Rolled attributes within a class-specific range.
- Zero or more affixes: a **prefix** applies a flat stat modifier; a **suffix** applies a conditional behavior modifier. Illustrative pair from the design notes, not a content commitment: "Veteran" (stat prefix), "of Wrath" (conditional suffix).

Class rosters, attribute ranges, and affix lists are jam content, built inside the jam window.

### 5.3 Progression

- Experience is shared; the party has one level and one XP pool. Kills feed the pool.
- On level up (defaults):
  - The level-up interrupts play immediately: the world pauses when XP crosses the threshold, and the player resolves the level-up before play resumes.
  - Each living member receives a small random attribute boost, rolled per member.
  - Each living member has a 25% chance to receive a talent pick, rolled independently per member. For each member who gets a pick, the player chooses one talent from that member's class talent pool plus the generic talent pool.
  - Occasionally a talent pick is replaced by an affix gain for that member; cadence is tunable (Open Questions).
- Level-up awards apply to living members only: a dead member accrues nothing, and resurrection re-scales attributes to the party level without granting the missed awards (Section 4.2).
- Recruits are generated at the current party level (Section 4.2).

### 5.4 Enemy generation

Enemy parties scale with depth on their own tables: floor-driven HP and
attack baselines (`game/biome.go`) with talent/affix chances from
`game/data/enemies.json`, budgeted to track player power across floors
without sharing the player generator. (Amended 2026-09-08: the
asymmetry is deliberate — players build up through classes and gear;
dungeons scale by depth. The old "same rules" sentence was wrong.) An
enemy party is one entity for targeting, area effects, and turn economy.

### 5.5 Classes and party composition

A member's class assigns its unique ability and its attribute profile, and is the primary lever against the two pressures (Section 3.5). Combat classes shift the health curve; utility classes shift the food and dungeon-resource curve.

The eight classes below are illustrative of the system, now **shipped via `game/data/classes.json`** with BuffA/BuffB/Active per class. Exact numbers are data, not prose — `bardRule` chorus +1 ATK/DEF plus 10% BuffA factor (non-stacking) is in `game/data/classes.json:3-5`. The role each class plays is the design commitment; all BuffA are **wired per RM2** (paladin absorb, bard chorus, `shining_armor` 10% resist, `healers_grace` HoT, `thick_skin` 30% `potent_scrolls` 20% etc.):

| Class | Role | Illustrative ability (BuffA wired) | Pressure it mitigates |
|---|---|---|---|
| Fighter | Front-line durability | Shining Armor: 10% resist (11% with bard) | Health |
| Rogue | Utility, resources | Nimble Fingers: open locks; +10% gold (11% with bard) | Food and dungeon resources |
| Cleric | Healing throughput | Healers Grace: HoT; better with rest | Health |
| Druid | Food economy | Verdant: reduces food consumption | Food |
| Bard | Meta-class, force multiplier | Chorus: +1 ATK/DEF to others, +10% to all BuffA | Neither directly |
| Wizard | Item economy | Potent Scrolls: +20% scroll effects; identifies over time; opens locks | Food and dungeon resources |
| Barbarian | Misuse safety | Thick Skin: 30% resist to negative effects | Health and item resources |
| Paladin | Absolute front-line | Vow of Protection: while active, absorbs all single-target damage | Health |
The paladin is the one exception to active-weighted targeting (Section 3.4): while active it absorbs all single-target damage, so shielding the party costs the paladin's action every turn (wired per RM2).

Passive effects stack between members of the same or different classes. Most classes mitigate one pressure directly; the bard is a meta-class that mitigates neither directly but multiplies the other members' contribution to both. This makes party composition the core build decision: the player spends limited party slots and a shared food clock on a mix that survives the run's health pressure within its food budget.

The barbarian makes unidentified-item misuse survivable — a counterweight to the identification systems of Section 11.1. The paladin is the one exception to active-weighted targeting (Section 3.4): while active it absorbs all single-target damage, so shielding the party costs the paladin's action every turn (wired per RM2).

### 5.6 Races

Each pilgrim has a race chosen at creation (CHOOSE RACES). Race is a
flat buff plus a synergy rule; all numbers are data
(`game/data/races.json`), mechanics in `game/race.go`:
| Race | Personal buff | Party / synergy | Pressure |
|---|---|---|---|
| Human | +1 HP, +1 HP/level | +5% XP per living human, stacking | XP engine |
| Elf | +1 DEF | +2 light, non-stacking; identify tick every 250 turns, −50 per extra elf, min 50 | Dungeon resources |
| Dwarf | +4 HP, +1 DEF, +2 damage strictly below half HP | Tremorsense: trap awareness at 4 tiles +1 per extra dwarf | Health |
| Halfling | +1 HP | 5% chance to fully avoid damage/negative effects; 10% extra item on pickup | Item resources |
| Gnome | +2 MDEF | 10% instant-identify of first-of-kind on pickup; 5% per gnome to not consume a scroll | Item economy |
| Half-orc | +2 ATK | +1 ATK per other half-orc (2+ to activate) | Damage |
| Troll | +4 HP, +2 ATK, regen 1 HP every 3rd turn | Double food cost while strictly above half HP | Power at food cost |

Synergy buffs fire at member-count thresholds (usually 1–2 of the
race). Specified 2026-09-08 from shipped behavior; prior doc carried
no race section.

## 6. Combat

### 6.1 Turn structure

Per world turn: the player party acts (one member action), then every other party acts (one member action each). All parties have fixed speed.

Valid actions for the acting member: move (party moves), melee attack (bump), use ability, use item, pick up, interact, wait.

### 6.2 Targeting

Single-target attacks — melee, ranged, spells, from either side — resolve against the defending party with the active-weighted distribution of Section 3.4, except when the defending party's active member is a paladin, which absorbs all single-target damage while active (Section 5.5). The active member of an enemy party is its most recent actor; a party that has not yet acted targets with its first member. If the active member dies, the designation moves immediately to the lowest-numbered living member (Section 3.3). The player does not choose which member of an enemy party absorbs a hit; the distribution does.

### 6.3 Area effects

Breaths, explosions, and traps target a single tile and affect every member of any party on it: "area" means the whole party on the tile, not a spatial radius. Multi-tile shapes, if the jam adds them, apply the same rule per tile. Per-member resistances and defense still apply. Area damage is the counterplay against large parties, for both sides.

### 6.4 Damage resolution

Damage and defense use deterministic ranges shown as raw numbers in the interface. The player can compute outcomes before committing, consistent with goal G5.

### 6.5 Statuses

Statuses attach to individual members and expire per member. Party-level effects enter only through the acting member: a confused actor moves erratically, and the counterplay is to act with a different member. Status effects on non-acting members remain dormant.

### 6.6 Member death

On death, the member leaves the tile roster, selection auto-switches (Section 3.3), passive contributions vanish (Section 4.3), and applied upgrades are lost (Section 7.3). The shared pack persists.

### 6.7 Ranged attacks and area effects

Some attacks work at range (specified 2026-09-08; §15 no
longer defers ranged combat):

- **Thrown potions** (`t`/`T`): inventory menu plus tile cursor.
  Damage resolves as a single-target attack against the defending
  party, favoring its active unit (active-weighted per §6.2).
- **Area scrolls** (fireball: 10 fire damage, Chebyshev radius 2,
  30% reduction on fire-resistant targets): every member of every
  party on each in-radius tile takes the hit, per §6.3. Fireball
  reveals nothing — the old whole-map reveal was a preserved data
  bug, removed 2026-09-08.

**Haste and slow** (model decided 2026-09-08; code follows in MS3).
Haste grants the acting member a 50% chance of an immediate second
strike against the same target — exposure is unchanged, the party
already acted. Slow applies a flat outgoing-damage penalty ([proposed
−25%, for review]). Both keep one action per party per turn (§3.2,
§6.1): haste never adds member actions, slow never removes them. The
pace layer stays: haste ×0.8 / slow ×1.2 food upkeep.


## 7. Items and Economy

### 7.1 Shared pack

One inventory serves the whole party. Capacity equals the sum of member capacities (Section 4.3).

### 7.2 Consumables

Potions, scrolls, and similar single-use items. Effects are unidentified until first use (Section 11.1).

Each kind has a run-random appearance drawn on generation. A potion's appearance is an "{optional adjective} {color}" such as "Azure" or "vivid Yellow"; a scroll's appearance is a "{verb} {noun}" token such as "Than of the Moth". Appearance is consistent across items of the same kind: every vial of the same color is the same potion, which the player learns by use. The inventory view shows each kind's appearance and held count ("Potions: Yellow (x3)"), or the identified type once learned. The appearance pools are jam content, but the pattern is part of the design.

### 7.3 Persistent upgrades

Single-use application items that permanently improve one member, for example a scroll that raises a member's attack. One illustrative item type appears in the design notes; the item table is jam content. Applied upgrades belong to the member and are lost on that member's death.

### 7.4 Rations

The party shares one hunger clock and one ration pool. The clock counts turns of food remaining: every world turn (Section 6.1) it ticks down by a per-member upkeep, and resting spends the same per-turn upkeep for its rest turns, with no surcharge (Section 11.3). The clock is a float under fractional modifiers (verdant, frugal, of_plenty, haste/slow pace) and truncates for display, so the shown number is approximate once modifiers apply (2026-09-08). The hunger state is a party attribute derived from the clock level (Section 11.2). Consuming a ration from the pack refills the clock.

The food clock is the run's time limit; a larger party is stronger per turn but spends the clock faster.

### 7.5 Gold and merchants

Gold is the run's currency. It appears wherever loot appears: slain enemies roll drops (usually gold or food, rarely a potion or scroll on their tile; `rollKillDrop`), and random dungeon finds (like consumables, sometimes placed by level generation) have a chance to include it. Gold has one use — merchants, scarce level features that turn gold into items (consumables, rations, and the permanent upgrades of Section 7.3). Wares and prices are jam content; that gold comes from loot and converts into items is the design commitment.

### 7.6 Summoning (exception)

The Summon Aid scroll is the one recruitment vector outside Section
4.2 (specified 2026-09-08): a random class generated at the party
level joins with a "Summoned" name for 15 turns, then departs
(expiry strips the summoned member). It fizzles at the cap of 4 and
works enemy-side symmetrically — with one known gap: enemy-side
expiry has no removal pass yet (flagged for MS3). No feature, no cost
beyond the scroll.

## 8. Run Structure

- The run descends through a fixed number of floors (default 8, `game/data/tuning.json:3` `floors:8`).
- The final floor has no stairs down: generation places `TileRelic` (`*`) where they would be. Stepping onto it claims the relic (`RelicCollected`): no inventory slot is used, the side panel shows `Relic: return to surface`, and the log says `You got the relic. Return to the surface.` The run is won by climbing back to floor 0 `StairsUp` with the relic, which sets `Escaped` plus the escape bonus (`tuning.json:36` `escapeBonus:500`).
- Death of all members ends the run. The death screen shows the seed and the score.
- A run can be suspended and resumed: save lib `game/save*.go` (`game/save.go:14` `SaveSlot`, `game/save_desktop.go:86` `HasSave`, `game/save_js.go:86` `HasSave`) — **wired per RM3 (Wave 2)**. One slot. The game saves on exit, and loading consumes the save (`game/save_desktop.go:33` `LoadFromFile` deletes file). A finished run cannot be reloaded (no save-scumming). Saving and score records are not meta-progression: a save suspends one run, and scores are records; neither affects the next run's gameplay (Section 2).
- Hit points regenerate slowly per member, once per world turn (Section 6.1). The rest command accelerates regeneration at the risk of encounters (Section 11.3). Rest is slower than combat and its healing does not improve with party level, so its value fades as the run deepens (Section 3.5).
- The score formula rewards floors reached, kills, and surviving members, plus a bonus for escaping back to the surface; the relative weights are tuning (`game/data/tuning.json:32-36` `scoreWeights: floor 100 / kill 10 / survivor 50 / escapeBonus 500`; `game/score.go:106` `scoreForPure`).
- The score table is per device — a local file on desktop, browser storage on web — and lists the best runs (score, seed, floor reached). It records; it does not progress. Scores are trust-based: the editable data folder (Section 13.1) makes desktop tampering trivial, so they are personal records, not competition.

## 9. World Generation

- Base layout: rooms and corridors, with themed floors defined as jam content.
- Recruitment features are placed per floor at a tunable rate.
- Enemy parties scale to the player's party level.
- Enemy party composition scales with depth: shallow floors lean toward lone monsters, deeper floors toward larger and role-mixed parties.
- Item placement is weighted by depth.
- Level features are placed per floor at tunable rates; the committed examples are merchants (gold into items, Section 7.5), fountains (a risky drink with a good or bad outcome), and shrines (chance-placed: recruitment and resurrection, Section 4.2).
- Four more features shipped with §9 rates in `game/data/features.json` (specified 2026-09-08; per-biome overlays retune them):
  - **Vaults** (0.12, locked, treasure 25–80, 20% trapped for 2–4 + possible poison): rogue or wizard to open (§4.3); consumed on loot.
  - **Forges** (0.10; 25 gold or 50 food): a random living member gains +1 attack range or +1 defense — permanent upgrades (§7.3). Insufficient funds is a no-op.
  - **Dens** (0.12; 3–5 monsters): a marker that spawns 1–2 nearby monsters per tick until spent.
  - **Pitfalls** (0.10; hidden half the time, 2–4 damage): levitation floats over; obvious or trap-aware parties (rogue/wizard, dwarf tremorsense, attuned +1 range, steady hands 10%) drop through unhurt, unaware parties take the hit plus possible poison. Either way the fall is one-way, down one floor.
- Generation guarantees: stairs are always present and reachable on every floor, and the relic is always on the final floor; no seed is unwinnable.
- Generation goals: party composition, map layout, enemy composition, and the item set all vary meaningfully per seed, satisfying the jam's procedural generation criterion.

## 10. User Interface

### 10.1 Main view

The map uses a single character cell per tile. The party renders as `@`; an enemy party renders as one glyph representing the party. The examine command reports the composition of a party on any visible tile (within the party's light radius, Section 4.3).

### 10.2 Landscape layout

The interface is landscape: a wide map on the left with the party state panel
beside it, then a full-width status bar, the combat log, and control hints. The
grid is the interface: no box chrome separates the regions in the terminal
build, and the web build keeps separation to plain native HTML (a simple panel
border is fine; nothing fancier — see design-guide Section 5). The layout is
larger than a standard terminal: an 80×24 map, a side panel wide enough for
long affix strings, and an eight-line combat log — roughly 110×34 total in
both builds (the exact minimum stays derived from live UI data, Section 13.3).

```
+--------------------------------------------------------------------------------+  <name> · <affixes>
|  <80 x 24 map, floor/walls>                                                    |  <Class> 34/34
|              @                                                                 |  <blank>
|                                                                                |  <name> · <affixes>
|                                                                                |  <Class> 12/28
|                                                                                |  Potions:
|                                                                                |    <appearance> (xN)
+--------------------------------------------------------------------------------+  Scrolls:
                                                                                    <appearance> (xN)
XP CUR <n> | XP TO NEXT LEVEL <n> | FOOD <n> <state>
> <log line 1>
> <log line 2>
> <log line 3>
> <log line 4>
> <log line 5>
> <log line 6>
> <log line 7>
> <log line 8>
<control hints>
```

Member blocks pair a name/affixes line with a class line that carries the
health numbers; empty slots show only their number. The status bar runs the
run-level counters (XP, food) in one row; food shows the turns remaining with
the hunger state word (`Ok` → `Hungry` → `Starving`) appended when it
changes.

The combat log holds eight lines; on the web it scrolls once full, and in the terminal the oldest line drops off. Control hints occupy the bottom
line. Names, affixes, class words, enemies, and consumable appearances are
placeholders; the layout is the commitment.

### 10.3 Key map

Key map principles:

- Movement is numpad-style on both the numpad and the number row: 7/8/9 are the
  up column, 4/5/6 the middle, 1/2/3 the down column, and 5 waits. The number
  row replaces the old QWE/ASD/ZXC block, freeing q/w/e/r for member selection.
- Arrow keys and hjkl are cardinal movement aliases for players who prefer them.
- q/w/e/r select the acting member (slots 1–4); selection is free and never
  spends a turn.

| Key | Action |
|---|---|
| Numpad / number row 1–9 | Move, numpad layout: 7 NW · 8 N · 9 NE / 4 W · 5 wait · 6 E / 1 SW · 2 S · 3 SE (`game/input.go:264-283` `NormalizeKey` Numpad*/Digit*) |
| Arrows / hjkl | Move (cardinal aliases; `h`/`j`/`k`/`l` per `game/input.go:295-303`) |
| `y` / `b` / `n` + `7`/`9`/`1`/`3` | Diagonals: `y`/`7` NW, `9` NE, `b`/`1` SW, `n`/`3` SE — `u` is **Use**, not NE (`game/input.go:304-313` `u/U→KeyUse`; `game/input.go:308` `KeyUpRight` only on `9`+Numpad9) |
| `.` / `5` / `Space` | Wait 1 turn (alias for 5; `game/input.go:314` `5/./ /Space→KeyWait`; `game/render.go:961` `5 / . / Space - wait`) |
| `z` / `Z` | Rest: 10-turn batch, 15 HP, ends early on hostile/hunger (`game/input.go:316-317` `z/Z→KeyRest`; `game/render.go:962` `z / Z - rest`) |
| `q`/`w`/`e`/`r` (`Q`/`W`/`E`/`R` also) | Select member 1–4 (free; `game/input.go:324-331` `q/Q→Select1` etc.; `R` is **select 4**, not Rest) |
| `g` / `G` | Contextual: pickup (`game/loot.go:61` `TryPickup` at `g.Party.Pos`) or on-feature — fountain/merchant/forge/shrine + close door (`game/input.go:198-216` `KeyPickup` tries door→fountain→merchant→forge→shrine→pickup; `game/render.go:963` `g - contextual use`) |
| `u` / `U` | Use item: opens inventory selector over held potions/scrolls (no auto-consume; `game/input.go:220-230` `KeyUse` opens menu via `InventoryUseEntries`). Potions drink to one chosen member; enchant picks a member and greater_healing picks a member or the party (`TryUseAppearanceOnMember`); other scrolls keep tile targeting |
| `t` / `T` | Throw potion (menu + cursor; `game/input.go:232-245` `KeyThrow`). Thrown damage resolves as a single-target attack favoring the active enemy unit (`pickEnemyTarget`) |
| `v` / `V` | Examine / look mode (`game/input.go:189` `KeyLook`) |
| `>` / `<` | Descend / ascend stairs (`game/input.go:318-321` `>→StairsDown` `<→StairsUp`; ascent from floor 0 blocked per `game/game.go:1943`) |
| `?` | Help overlay (`game/render.go:956-969`) |
| `Esc` | Quit to menu (`game/input.go:288,340` `Escape→KeyQuit`) |
| `]` / `}` / `BracketRight` | Wizard menu (8 entries; `game/wizard.go:15`) |

The use command is a single selector over the held potions and scrolls; examine lives at `v`.

### 10.4 Surface background

The game paints its own surface background; nothing visible is left to the
host default. A terminal only shows a background color where a cell was
explicitly drawn with that style: cells reset by a screen clear fall back to
the terminal's own background, which may be a different color or transparent,
so the palette would appear only behind drawn glyphs. The terminal draw
therefore fills every cell with a styled blank before drawing anything
(`screen.Fill`, not `screen.Clear`), making the whole window carry
the near-black backing.

The web build meets the same rule through the page chrome: the `body` element
carries `--bg` from `web/tokens.css` across the entire viewport, and the grid
paints on top of it.
### 10.5 Main menu and shell screens

The game opens on a main menu (4 entries post-Wave 2): **New Game**, **Load** (gated on `HasSave` — `game/save_desktop.go:86` `HasSave` / `game/save_js.go:86` `HasSave` / `game/save.go:14` `SaveSlot`), **Scores**, and **Exit**. Load is enabled only when a save exists (a `save.json` file on desktop, `localStorage` key `pilgrims_save` on web per `game/save*.go:8-9`). Exit applies to the desktop build only; web players close the tab. Menu navigation is arrows or j/k to move, Enter to select, and Escape to go back.

Death and victory screens show the seed and score and return to the main menu. Every shell screen follows the surface rule (Section 10.4) and the palette rules of the design guide. The web build renders shell controls (start, load, scores, data upload and reset) as plain native HTML.

## 11. Traditional Systems

### 11.1 Identification

Potions and scrolls are unidentified until first use. Knowledge is party-wide and persists for the duration of the run. Nothing carries over between runs (Section 2).

Identification is per appearance. Using one item reveals the type of every item that shares its appearance (Section 7.2), so all vials of a learned color show their real type from then on. The wizard's passive (Section 5.5) auto-identifies held appearances over time — a second, slower path; using an item remains the universal method. Appearance is re-rolled each run, which prevents cross-run meta-gaming of colors.

Five paths identify, not two (specified 2026-09-08): **use** (universal,
immediate); **wizard attunement** (1 held appearance per 50 turns,
45 with a living bard, ticking at turn end and during rest);
**elven senses** (interval 250 turns, −50 per extra elf, floor 50);
**gnome curiosity** (10% instant identify of first-of-kind on pickup,
plus 5% per gnome to not consume a scroll); **lorekeeper study** (1
held appearance per 50 turns during rest, 45 with a living bard,
sharing one ticker with attunement so they never double-fire). The
old `ShouldAutoID` turn-%-50 predicate is uncalled dead code (see
audit S7). These paths are part of the design.

### 11.2 Hunger

The party shares one hunger clock and one ration pool (Section 7.4). Every world turn advances the clock by the per-member upkeep (Section 6.1); resting spends the same per-turn upkeep for each rest turn, with no surcharge (Section 11.3). The hunger state derives from the clock level: as food remaining falls, the party progresses from Ok through Hungry to Starving. Hungry and Starving carry no attribute penalties. Damage starts at zero food: each world turn deals 1 HP to every living member. (Amended 2026-09-08: the old "penalties at the extremes" sentence was wrong.) Rations are items in the shared pack; consuming one refills the clock.

The clock is the run's time limit (Section 3.5). A careful player with few members can stretch the same food across more turns, which is the mechanical basis for solo runs.

### 11.3 Rest

The rest command performs one batch of 10 wait turns at the accelerated rest rate. A full uninterrupted rest heals each living member for 15 HP (default, tunable) — 1.5× the natural per-world-turn regeneration (Section 8), where the same turns would restore 10 HP naturally (natural regen default: 1 HP per member per world turn). The world keeps advancing while the party rests: enemy parties act each rest turn, and wandering encounters are the risk. The batch ends early only when a hostile appears or hunger advances; an interrupted rest credits exactly the completed turns (8 turns rested heals 12 HP). Rest spends the normal per-turn food upkeep for each rest turn, with no surcharge (Section 7.4).

Rest heals slowly and does not scale with party level. In the late game the party's damage per encounter outpaces rest, so the player is nudged toward healing consumables or a dedicated healer (Section 5.5). Rest converts food into health at a poor exchange rate the deeper the run goes.

### 11.4 Seeds and score

Each run starts from a shown seed. The death and victory screens display the seed and the final score, enabling shareable challenge runs.

## 12. Jam Compliance

| Jam criterion | Design response |
|---|---|
| Turn-based | Section 3.2: discrete turns, one action per party per turn. |
| Run-based, no meta-progression | Sections 2 and 8: nothing that affects gameplay persists between runs — identification knowledge, levels, items. A suspended run (save) and local score records are the only persisted data, and neither changes the next run. |
| Permanent consequences | Member death is permanent; upgrades and talents die with the member. The sole exception is a chance-placed free shrine resurrection that restores the member fully (Section 4.2). |
| Meaningful PCG | Section 9: party composition, maps, enemies, and items vary per seed. |
| Single character or small party | The party is the core system, not decoration. |

**Pre-jam boundary.** This document describes systems only. Class rosters, talent lists, affix lists, monster sets, item tables, and floor themes are content and will be produced inside the jam window as data files (Section 13.1), per the jam rules.

## 13. Technical Plan

### 13.1 Architecture

- `game` package: pure Go core, no I/O, deterministic given a seed.
- Terminal frontend: `tcell` TUI, compiled to a single static binary per platform (Linux, macOS, Windows).
- Web frontend: the same core compiled to WASM with `syscall/js`, rendered into an HTML shell; deployed as an itch.io HTML5 upload.
- Web builds bake the same data files into the WASM at compile time (`go:embed`). Players can also upload a data zip to replace the data in-browser; a reset button restores the embedded defaults. Uploaded data persists in browser storage until reset. Both builds consume identical data; only the loading path differs (`archive/zip` inside WASM + `localStorage` + reset in `cmd/wasm/main.go`, overlay in `game/data.go`).
- A simple data editor is a stretch goal: a small UI that edits the data files so content can be tuned without touching code; the web shell exposes a minimal JSON editor over the same overlay. Full editor UI remains follow-up.
### 13.2 Milestone 0 — prototype gate (passed pre-jam)

The pre-jam prototype rendered a room and moved an `@` in both frontends from one core, validating the Go-to-WASM layering and the dual distribution path. It contained no game systems and has been cleared; the implementation notes it produced are in Section 13.3.

### 13.3 Pre-jam implementation notes

- **Surface background.** Every painted screen fills the full surface with the
  styled background before drawing (Section 10.4), including fog-of-war
  cells; no screen falls back to the host default.
- **Web export and compression.** The itch.io upload is a zip of the `web/`
  directory: `index.html`, `tokens.css`, `wasm_exec.js`, `main.wasm.br`, and
  `main.wasm` as an uncompressed fallback. Compress with `brotli -q 11`
  after `-ldflags="-s -w"` and `wasm-opt -O3` (if binaryen is available).
  The loader fetches `main.wasm.br` and decompresses in JavaScript
  (`DecompressionStream("br")`) before compiling, because the host serves
  files without a `Content-Encoding: br` header; compile from bytes
  (`WebAssembly.instantiate`) rather than `instantiateStreaming` so a wrong
  MIME type cannot break the load. `DecompressionStream` needs a modern
  browser (Chrome 80+, Firefox 113+, Safari 16.4+); older browsers fall back
  to `main.wasm` automatically.
- **Terminal size derivation.** The minimum window size stays derived from the
  live UI data — the committed layout is roughly 110×34, larger than a
  standard terminal (Section 10.2) — so panel and log growth cannot drift
  from the launcher scripts or the web page. A terminal smaller than the
  minimum shows a resize prompt with the derived W×H instead of a clipped
  layout.
- **Numpad movement (Section 10.3).** Both frontends map the numpad and the
  number row to the same movement directions. In the terminal the numpad
  arrives as distinct escape sequences and NumLock state can change them;
  on the web `Numpad1` is a different key code than `Digit1` — accept both,
  and treat 5 (either) as wait.
- **Data loading (Section 13.1).** `go:embed` cannot reach files outside the
  embedding package's directory, so the shared data directory must live where
  the core can embed it; the desktop build reads the same directory from
  `data/` beside the executable. The web upload path parses the zip with Go's
  `archive/zip` inside WASM, persists the extracted files in browser storage,
  and the reset button clears that storage and falls back to the embedded
  files.

## 14. Scope and Milestones — **re-dated 2026-09-02** (supersedes pre-jam cut list)

| Milestone | Content | Window |
|---|---|---|
| M0 | Prototype gate: layering validated pre-jam; source cleared | Pre-jam (done) |
| M1 | Solo core loop: movement, bump combat, FOV, stairs | Days 1–3 |
| M2 | Party systems: selection, targeting weights, passives, death and auto-switch | Days 4–6 |
| M3 | Characters: generation, affixes, party level, talents | Days 7–9 |
| M4 | Items and systems: identification, hunger, rest, gold, seeds, score, save/load | Days 10–11 |
| M5 | World generation plus content and balance (jam content) | Days 12–13 |
| M6 | UI polish, main menu and shell screens, web build, itch page, screenshots | Day 14 |

Cut lines (original order, now **re-dated 2026-09-02** — most **delivered**):
- Affix gain on level up — **delivered** (`game/data/affixes.json` 7+7; `game/talents.go:1044` `AffixReplaceChance` 0.10).
- Escape bonus — **delivered** (`tuning.json:36` `escapeBonus:500`; `game/score.go:138` `escaped:=Escaped` per RM3a).
- Floor count (8) — **delivered** (`tuning.json:3` `floors:8`).
## 15. Open Questions — settled 2026-09-08 except class balance

- Affix gain cadence on level ups — **settled: 0.10**
  (`tuning.json` `affixReplaceChance`; Section 5.3).
- XP curve — **settled: base 100 × factor 1.5** (`tuning.json`
  `xpBase`/`xpFactor`).
- Score weighting — **settled: floor 100 / kill 10 / survivor 50 /
  escape 500** (`tuning.json` `scoreWeights`; Section 8).
- Food tuning — **settled: upkeep 1/member/turn, refill 250, clock
  2000, Hungry 0.25, Starving 0.05** (`tuning.json`; Sections 3.5,
  7.4, 11.2).
- Rest exchange rate — **settled: 10-turn batches, 15 HP, 1 HP
  natural regen** (`tuning.json`; Section 11.3).
- Class balance — **held for playtesting** (relic reachable without
  a combat class? druid economy too cheap?). The one live question.
- Shrine spawn rate — **settled: 0.25** (`features.json`; Sections
  4.2, 4.4).
- Light radius, recruitment rate — **settled: 6, 0.3** (`tuning.json`).

**Deferred, not planned (post-jam candidates).** Ritual spellcasting and audio are outside the jam scope — noted so they are not forgotten, not planned. (Ranged combat left this list 2026-09-08: throw-potion cursor mode and area scrolls shipped, now specified in Section 6.7.) **Races shipped** per RM8a: `game/data/races.json` + `game/race.go` + `game/menu.go:616` “CHOOSE RACES” (7 races: human/elf/dwarf/halfling/gnome/half_orc/troll; charBuff/partyBuff/synergyBuff data-driven).

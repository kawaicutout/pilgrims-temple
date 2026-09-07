# Run Start — UI Candidates (for review)

Goal: start a run with a seed entry plus 1–3 player-built characters. Each
character: class, race, name (blank = random). The player can start with one,
two, or three characters, or discard drafted characters. Decided: Option A
(guided pipeline), implemented in `game/menu.go` creation states plus both
frontends; seed blank means random, duplicate classes allowed.

## Current flow (facts)

- Main menu → `CharSelectState` (`game/menu.go:64`): pick exactly 2 distinct
  classes, Enter starts when 2 are picked, Esc steps back one pick.
- Then `RaceSelectState` (`game/menu.go:110`): one race per class slot, in
  order. Then `NewGameWithClassesAndRaces` with a random seed.
- Names come from `game/names.go` pools. No seed entry, no naming, fixed
  count of 2, duplicate classes refused. Both frontends (`cmd/terminal`,
  `cmd/wasm`) implement the same screens separately from shared
  `game.Render*` frames.

## Requirements for all candidates

- Seed step: numeric entry, blank = random. New input mode in both frontends.
- Name step: short text entry, blank = random from `names.go`. Single-line
  raw text is legitimate here (a name is genuinely free text).
- Roster holds 0–3 drafts; Start needs 1+ complete drafts; discard clears a
  draft. Duplicate classes: allowed (current ban was for a fixed pair).
- Core API: `NewGameWithSeedClassesRacesNames(seed, classes, races, names)`.
  `Member.Name` already persists, so saves need no schema change.
- Both frontends stay in lockstep; frames come from `game/render.go`.

## Candidate A — Guided pipeline (recommended)

Seed → per slot: Class → Race → Name → Review list → Start. Esc steps back
one screen. A `Discard` option on the review screen drops the current draft.

- Pros: reuses the existing list-menu pattern three times plus one text
  entry; smallest new render code; matches the current state-machine style,
  so both frontends change symmetrically; each screen fits the 110×34 grid.
- Cons: slowest for three characters (up to 10 screens); text entry must be
  built once and reused for seed and name.
- States: `SeedEntry`, `ClassPick`, `RacePick`, `NameEntry`,
  `ReviewRoster` (Start / Add another / Discard / Back).

## Candidate B — Roster table

One screen, three rows. Each row: class, race, name columns. Arrows move,
Enter cycles class/race or opens name entry, Del clears the row, Start is
enabled with 1+ complete rows.

- Pros: full overview; fastest edits; discard is one key; no screen stack.
- Cons: most new render and navigation code; three-column layout is tight
  beside the 80-column map; text entry still needed; hardest to keep the
  two frontends identical.

## Candidate C — Extend current screens

Seed screen first; class screen allows 1–3 picks with duplicates; race
screen unchanged; name screen walks the picks (blank = random, Esc skips);
done screen offers Start or Drop-last.

- Pros: least new code (two added screens, three reused); familiar flow.
- Cons: flow spreads across five screens with no overview; discard is
  last-in-first-out only; the class screen's "need exactly 2" logic inverts.

## Open questions

- Max name length and charset (suggest 12 runes, letters/spaces/apostrophe).
- Seed format: decimal int64 as shown on the death screen.
- Whether drafted-but-undismissed state needs a save (suggest no: roster
  dies with the menu session).

## Mock-ups

Conventions: centered title, `> ` cursor, dim hints. Truncated to the
content rows; real frames keep the 110x34 grid with panel, status, hints.

### A — Guided pipeline

Seed screen (blank = random):

```
NEW EXPEDITION
Enter seed (blank for random)

> 1788326937246103290_

Enter: continue  Esc: back
```

Class screen, slot 2 of up to 3:

```
CHOOSE PILGRIM 2 (1-3, Enter: done)

> Fighter      sturdy front line
  Rogue        locks and loot
  Cleric       healing over time
  Druid        food economy

Roster: Mara the Fighter
Enter: pick  Del: drop last  Esc: back
```

Race, then name, then review:

```
CHOOSE RACE — Rogue

> Human     +1 HP per level
  Elf       far sight
  ...

RACE FOR ROGUE: Elf_

NAME FOR ELF ROGUE (blank = random)

> Kessa_

REVIEW ROSTER

> Mara — Fighter (Human)
  Kessa — Rogue (Elf)
  [ Add pilgrim ]   (2/3)
  [ Begin descent ]
  [ Discard last ]

Enter: choose  Esc: back
```

### B — Roster table

```
BUILD ROSTER (1-3 pilgrims)

> [1] Fighter  Human   Mara      _
  [2] Rogue    Elf     Kessa
  [3] ---      ---     ---

Arrows: move  Enter: cycle/edit  Del: clear row
Enter on empty class: Start (needs 1+ complete rows)
```

Editing a name cell opens a one-line entry row:

```
> NAME [2]: Kessa_
  (blank = random, max 12 letters)
```

### C — Extended current screens

Seed first, then the current class screen with a count:

```
SEED (blank = random)

> _

CHOOSE PILGRIMS (1-3, dupes allowed)

> Fighter [x] [x]
  Cleric  [x]

Picked: 3/3 — Enter: races  Esc: back
```

Name walk after races (blank skips to random):

```
NAME PILGRIM 2/3 — Cleric (Dwarf)

> _

Enter: keep  Esc: skip all naming
```

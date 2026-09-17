# Pilgrims' Temple

A turn-based party roguelike for roguetemple's Fortnight 2 (September 1–15,
2026). The player builds 1–3 pilgrims and descends 8 floors to claim the
relic, then climbs back to the surface. The party shares one tile and moves
as one unit; one member acts per turn, attacks hit the last actor hardest,
and a shared food clock limits the run. Written in Go, shipped as a
native terminal binary and as WebAssembly (WASM) for itch.io.

## How to play

Build a roster (seed, then class, race, and name per pilgrim; blank means
random), then descend. Each turn: select a member with q/w/e/r (free), then
act — move, bump to attack, use (`u`), throw (`t`), pick up (`g`), rest
(`z`), or wait (`5`). Acting draws attacks toward the actor, so use durable
members on risky turns.

Two clocks limit the run. Food drains every turn for every living member;
rations, forage, and lean parties stretch it. Health drains in fights;
rest heals slowly, potions and scrolls heal fast. Potions and scrolls start
unidentified: using one identifies every item sharing its look. Death is
permanent for members; losing all four ends the run. The run suspends on quit
and the save is consumed on load. Scores stay on the device.

| Key | Action |
|---|---|
| Numpad / number row | Move (7/8/9 up, 4/5/6 mid, 1/2/3 down) |
| Arrows / hjkl | Move (cardinal) |
| q / w / e / r | Select member 1–4 (free) |
| 5 / . / Space | Wait |
| z | Rest (10 turns) |
| g | Contextual: pick up, fountain, merchant, forge, shrine, door |
| u | Use item (choose a member for potions) |
| t | Throw potion (single target, favors the active enemy) |
| v | Examine |
| > / < | Stairs down / up |
| ? | Help |
| Esc | Quit to menu (saves the run) |

## Run and build

- Run: `./run.sh` (opens a terminal at the data-driven size, default
  110×34) or `./make_dist.sh` for the itch.io zips.
- Build: `make bin` (desktop), `make web` (WASM + brotli), `make zip`.
- Check: `make vet`, `make test`.

## Documentation layout

- `DESIGN.md` — systems, rules, key map, milestones.
- `docs/gameplay-loop.md` — the loops the player navigates.
- `docs/design-guide.md` — visual identity for both builds.
- `docs/levelFeatures.md` — vault, forge, den, pitfall, per-biome notes.
- `docs/status-magic-brainstorm.md` — status and consumable reference.
- `docs/milestones.md` — milestone history.
- `docs/run-start-candidates.md`, `docs/biome-candidates.md` — decided
  plans for creation and biomes.
- `Sept7Milestones.md` — committed plan for remaining fixes.
- `game/data/*.json` — all tuning: floors, food, XP, enemies, biomes,
  items, classes, races.

# Milestones — September 7, 2026

Scope: implement the features the code still references but does not perform, as of September 7. Food balance, key map, and the design guide are working as intended and have no milestones here. `localdoc/` stays local working notes; this file is the committed plan.

## M1 — Terminal save/load parity (remaining gap)

Goal: desktop terminal matches the web build: quit saves, death/win deletes, menu offers Load when a save exists.

- Web already does this (`cmd/wasm/main.go` `beforeunload` + `Save`/`DeleteSave`, `game/menu.go:37` `GetMainMenuOptions`).
- Terminal does not: `cmd/terminal/main.go:334` hardcodes New/Scores/Exit and never calls `Save`/`DeleteSave`/`HasSave`.
- Acceptance: quit mid-run then relaunch offers Load with the same floor/turn/food; death removes the save; `grep Save cmd/terminal` finds call sites.

## M2 — Shrine costs removed (decided September 7)

Goal: shrines are free. No cost data anywhere.

- Done in this pass: `game/data/shrines.json` carries id/name/desc only; `game/features.go` `ShrineUse` has no cost fields; both shrine paths (`ExecuteShrineChoice`, step-on `handleShrine`) resurrect and recruit free; `DESIGN.md` §4.2 says free.
- Acceptance: `grep -rn GoldCost shrines` finds nothing; stepping on a shrine with a dead member resurrects with gold at 0.

## M3 — Dead-code and dormant-branch cleanup

Goal: no defined-but-unreachable game rule.

- `StatusRegenerate` (`game/status.go:18`) has no apply path; troll regen uses the `Regen` flag instead. Wire it or delete the constant. (User note: Wire; potions/scrolls should offer regen)
- Half-orc ATK immunity (`game/race.go:518`, `game/party.go:310`) is dormant; no ATK-reduction source exists. Wire a source or delete the branch and helper.
- `game/wizard.go:15` references a `wizard.json` override with no loader. Delete the comment or add the loader. (User note: probably remove?)
- Acceptance: each ID above either has a runtime caller or no longer exists in code or data.

## M4 — Main-doc trust repair (no gameplay change)

Goal: committed docs describe the shipped build.

- `DESIGN.md` §13.1/§14: web upload/reset wording drops the anticipatory note; the WASM shell already parses zips via `archive/zip` and persists to browser storage (`cmd/wasm/main.go:44`, `game/data.go:129`).
- `game/data/features.json` `perBiomeVariants` comment drops "deferred"; the overlay is wired (`game/features.go:478`).
- `docs/status-magic-brainstorm.md` §1.5 shrine line matches free shrines.
- Acceptance: no committed doc claims a shipped feature is planned, or a planned feature is shipped.

## M5 — Repo hygiene (decided September 7)

Goal: generated and personal files stay out of version control.

- `scores.json` and `bin/` are ignored and untracked; builds come from `make terminal` / `make web`.
- Acceptance: `git ls-files` shows neither `scores.json` nor `bin/`; `.gitignore` lists both.

## Out of scope (deferred, not planned)

Ranged weapon combat, ritual spellcasting, audio, per-biome special features (crypt sarcophagus, bone pile, mycelial heart, and the rest in `docs/levelFeatures.md`). Noted so they are not forgotten, not scheduled.

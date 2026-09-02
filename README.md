# Party Roguelike

A turn-based party roguelike for roguetemple's Fortnight 2 (September 1–15,
2026). The player controls a party of one to four characters that shares a
single tile and moves as one unit; one member acts per turn, hits are weighted
at the active member, and a shared food clock is the run's time limit. The
game is written in Go, compiled to a native terminal binary and to WebAssembly
for the itch.io web build.

## Design docs

- `DESIGN.md` — the game design document: systems, rules, and open questions.
- `docs/design-guide.md` — visual identity for the terminal and web builds.
- `docs/gameplay-loop.md` — the loops the player actually navigates.

## Status

**Pre-jam M0 cleared** — the room-renderer mock from before the jam was cleared for a clean slate (see `DESIGN.md` §13.2). Jam window Sept 1–15, 2026.

- Run: `./run.sh` — opens the default terminal at the data-driven size (`game/data/tuning.json` `layout.minCols`/`minRows`, default 110×34; `run.sh:13-28` reads via `jq` then falls back). Or run the binary directly after `make terminal` (`Makefile:23` `bin/pilgrims-temple`) / `make wasm` / `make web` (brotli).
- Build: `make terminal` (native), `make wasm`, `make wasm-br`, `make web`, `make zip` (itch upload per `DESIGN.md` §13.3). `make vet` / `make test` for checks.
- Tokens: `web/tokens.css` is the canonical design tokens mirroring `docs/design-guide.md`.

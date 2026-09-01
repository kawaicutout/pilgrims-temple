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

The pre-jam M0 prototype (a room renderer and mock interface in both
frontends) was cleared to leave a clean slate for the jam window. The
implementation starts fresh on September 1, 2026; build and run instructions
will land with the first milestone. `web/tokens.css` is kept as the canonical
design tokens mirroring the design guide.

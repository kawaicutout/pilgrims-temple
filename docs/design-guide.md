# Design Guide — Party Roguelike

Scope: visual identity for the terminal build and the web (WASM) build. The two
builds share one look. The rules follow the project style guide
(`.ai/style-guide.md`) and design tokens live in `web/tokens.css`.

The rules bind the interface — the panels, log, status bar, and the numbers
and warnings that report game state. Scene content (enemy and item glyphs,
terrain, effects) is game content, not design: it is exempt from the accent
and contrast rules and may use the full palette.

## 1. Identity statement

A muted, desaturated take on the classic ASCII roguelike, set in a dark
underground temple. Full-color game space, but every color is deliberately
drained of saturation so the game reads as one step past standard
full-brightness roguelikes. Text stays near-white on the near-black surface; the
gold and red accents appear only where the game needs to say something
(hits, warnings, party stats), and slate blue marks the world's ambience.

> One line: **A desaturated temple lit by torch-gold, where the numbers are the color.**

## 2. Pillars applied

The identity follows the integrity guidelines, in order.

### Invisibility

No invented texture or ornament in the interface. Interface colors describe
state, not atmosphere; slate blue is the one deliberate atmosphere channel,
scoped to sensory and ambience entries that still carry their meaning in text
first (Section 3). If an interface color does not carry information (gold
for a hit, red for damage, accents for party stats), it does not appear. The desaturation is the single deliberate
departure from the field's bright conventions, and it is a functional one: it
keeps the map readable as a field of glyphs rather than a wall of color.

### Intelligibility

The interface stays legible first:

- Text is the smallest size that remains readable at terminal distances.
- Anything colored against the map is also separated by shape or position
  (a color never carries its meaning alone).
- Accessibility floor: see Section 6. No informational glyph falls below 3:1
  contrast; small informational text stays at or above AA (4.5:1).
- The map is a field of ASCII glyphs, one cell per tile. This layout is
  already committed in DESIGN.md Section 10.

### Integration

The interface palette is a core-plus-accents system:

- Core: near-black background, near-white foreground, and the gray ramp.
- Gold: desaturated. Positive states, warnings, and the party side panel.
- Red: desaturated. Damage, death, and negative states.
- Slate blue: desaturated. Atmosphere and sensory entries — currently
  confined to atmospheric log lines and scene ambience (Section 3).

Gold and red are tinted off the same warm, low-saturation family so they relate
to each other and to the temple setting; slate blue is the cool counterpart,
which marks the world's ambience rather than mechanical state.

### Identity

The distinction from a classic bright roguelike is the desaturation. It is
visible in a thumbnail and it survives both builds.

## 3. Palette

### Core (game space)

| Token | Role | Value |
|---|---|---|
| background | Map and panel backing | near-black, slightly warm: `#141210` |
| foreground | Default text and glyphs | near-white: `#e6e0d8` |
| floor | Open floor glyph | low-contrast gray: `#4a4642` |
| wall | Wall and pillar glyph | mid gray, stepped from floor: `#6b645c` |
| gray-1 | Read-only labels, dim text, near-white | `#b5aea5` |
| gray-2 | Read-only labels, dim text | `#8a857e` |
| gray-3 | Faint chrome, panel borders | `#3d3936` |
| gray-4 | Faintest chrome, decorative | `#2c2927` |

### Accents

| Token | Role | Value |
|---|---|---|
| gold | Positive, warnings, numbers, panel headers | desaturated gold: `#b8975a` |
| gold-bright | Emphasized positive, cursor | brighter gold: `#d3ad6b` |
| red | Damage, death, negative | desaturated red: `#a8564a` |
| red-bright | Emphasized damage, critical | brighter red: `#c96a5a` |
| slate | Atmosphere, sensory log entries, level feel | desaturated cool slate-blue: `#6e8fb5` (tentative, tune at render) |

Every interface element that reports game state — damage numbers, warnings,
party stats, item names in the inventory, whatever the jam adds to the UI —
draws from the gold and red families. Saturation stays low; lightness stays
high enough to hold contrast. Slate blue is the third accent, and it carries
atmosphere, not state: it is used for atmospheric log entries (sensory hints
from distant enemy groups, hidden special features, biome-specific callouts)
and for scene ambience and level feel. There is no audio in the jam scope; all
sensory presentation is textual — log entries, glyphs, and ambience colors. It
never carries mechanical numbers or state, which keep to gold, red, and the
gray ramp. Scene content is exempt:
enemy and item glyphs, terrain, and effects are game content whose colors are
decided in the jam window. Staying inside the desaturated families there is a
recommendation for identity continuity, not a rule.

Rendering: in the terminal build, tcell maps the tokens to truecolor where the
terminal supports it and to the nearest ANSI color otherwise; no separate color
profile is needed. Verification task (not a design decision): confirm the ANSI
fallback preserves the intended hierarchy in non-truecolor terminals.

## 4. Glyphs

The map uses classic ASCII, one character per tile (CP437-like set):

| Entity | Glyph |
|---|---|
| Player party | `@` |
| Enemy party | one glyph per party: the group's letter, case-coded by type (e.g. `g` goblin vs `G` giant); mixed groups use the first living member's type |
| Open floor | `.` |
| Wall and pillar | `#` |
| Stairs down / up | `>` / `<` |
| Items | `?` unidentified, letter after identification |

The player party is always `@`. This is a design commitment in DESIGN.md
Section 10.1 and is not changed by styling.

Enemy parties share a glyph with their group by design: the letter names the
group, not the instance, so two parties of the same group look identical.
Case codes the group type, and a mixed group renders its first living
member's type. Glyph brightness may vary subtly with party size (a party of
four reads brighter than a party of one); brightness only reinforces size —
the letter still carries identity, so the size signal is never color alone.
Enemy glyph colors are scene content and free: the jam can color-code
variants however it likes (a green `g` vs a blue `g`) without touching the
interface palette. Only the UI that reports on enemies — the log, the examine
panel — stays within the accent families.

## 5. Layout and panels

The interface is landscape: a wide map on the left, the party state panel
beside it, then a full-width status bar, the combat log, and control hints. The
grid is the interface: no chrome separates the regions in the terminal build,
and the web build keeps separation to plain native HTML (a simple panel border
is fine; nothing fancier). The layout is larger than a standard terminal: an
80×24 map, a side panel wide enough for long affix strings, and an eight-line
combat log — roughly 110×34 total in both builds (DESIGN.md Section 10.2).
The layout diagram is canonical in DESIGN.md Section 10.2 and is not
duplicated here.

Rules:

- Member blocks pair a name/affixes line with a class line carrying the health
  numbers. The selected member's name line renders in gold-bright.
- Numbers (HP, XP, counts) use gold; the prompts and names that introduce them
  stay in the gray ramp. The distinction is shape and position plus color.
- The combat log uses gold for player-initiated hits and red-bright for hits
  taken (small damage text uses `--red-bright`, never `--red`, per Section 6).
  Atmospheric entries (distant noises, biome callouts, hidden-feature hints)
  use slate; they report the world's feel, not mechanical state. Text alone
  carries the meaning; color reinforces it.

Both builds draw these regions from the same panel data, so the web page and the terminal grid
show the same layout.

## 6. Accessibility

The desaturation is a contrast budget, not a removal. The floors below bind
interface text and informational glyphs — panel numbers, log lines, the `@`,
warnings. Scene glyphs (enemies, items, terrain) are content and exempt;
staying above the floor is a readability recommendation for them, not a rule.
Contrast against the near-black background (`#141210`), measured with WCAG
ratios:

| Color | Ratio vs background | Fitness |
|---|---|---|
| foreground `#e6e0d8` | ~14.3:1 | Small text |
| gold-bright `#d3ad6b` | ~8.9:1 | Small text, emphasis |
| gray-1 `#b5aea5` | ~8.5:1 | Dim read-only labels, small text |
| gold `#b8975a` | ~6.8:1 | Small text, numbers |
| gray-2 `#8a857e` | ~5.1:1 | Dim read-only labels, small text |
| red-bright `#c96a5a` | ~5.1:1 | Small text (damage) |
| red `#a8564a` | ~3.6:1 | Large or graphical damage; not small text |
| slate `#6e8fb5` | ~5.6:1 | Atmospheric entries (ambience, not required state); verify at render |
| wall `#6b645c` | ~3.2:1 | Map wall glyph, graphical emphasis only |
| floor `#4a4642` | ~2.0:1 | Decorative floor, never information |

Rules that follow from the table:

- Nothing in the interface below 3:1 against its background carries required
  information. Everything below 4.5:1 is either decorative or reserved for
  graphical emphasis only.
- The paler floor gray (`#4a4642`) is decorative floor, never information.
  Informational glyphs keep the foreground or an accent color. The base red
  (`--red`) is used at large or graphical weight only; small damage text uses
  `--red-bright`.
- Color is never the only channel. Confirm below:
  - Gold vs red is reinforced by position (panel vs message) and by the message
    text itself.
  - The selected member is identified by the row position and glyph, not color
    alone.
  - Enemy group and party size: the glyph letter names the group; brightness
    only reinforces size, so neither signal depends on color alone.
  - Atmospheric entries are distinguished by their message text; slate only
    carries ambience, never a needed state.
- Avoid pure black `#000000` and pure white `#ffffff`. Both are off relative
  to the warm tint so adjacent colors do not fight.

## 7. Type

Both builds use a monospace face with a serif texture at the terminal scale.

| Build | Typeface | Fallback |
|---|---|---|
| Web | Libertinus Mono (Google Fonts, loaded via `@import` in `web/tokens.css`) | `ui-monospace`, monospace |
| Terminal | system terminal font | — |

Rules:

- The web build sets `font-family` from the tokens and falls back to the system
  monospace if the font does not load.
- The terminal build uses the running terminal's font unchanged. No styling is
  possible or needed there.
- Letter spacing stays at the face default; the terminal grid is
  character-per-cell.

## 8. Web build chrome

The web build renders the same content as the terminal build. The only addition
is a page chrome that leaves the grid untouched:

- The map and panel sit on the shared near-black background, and the
  background covers the entire surface, not just drawn cells. On the web the
  `body` element carries `--bg` for the whole viewport. A styled cell only
  exists where it is drawn: the terminal build fills every cell with a styled
  blank before drawing (see DESIGN.md Section 10.4) so untouched cells never
  fall back to the terminal's own background.
- Page margins and the status line (boot, error, key hint) use the gray ramp.
- No shadows, gradients, rounded corners, glows, or other effects. The web
  build may use plain native HTML panel borders (the palette's gray-3 role is
  "faint chrome, panel borders"); the terminal build stays borderless. The grid
  is the interface.
- The page centers horizontally, matches the terminal column count, and shows a
  single column on small screens.
- Shell controls (new game, load, scores, data upload and reset) are plain
  native HTML buttons — the same allowance as panel borders (DESIGN.md
  Sections 10.5 and 13.1).

## 9. Applying the guide

- Do not add an interface color for decoration. An interface color earns its
  place by carrying information — the one atmosphere channel is slate blue,
  scoped to sensory and ambience entries (Section 3) that always carry their
  meaning in text first.
- Do not brighten the interface beyond the token palette. Full-brightness
  interface colors are the explicit non-goal; the scene is free, and
  desaturation remains the identity by convention.
- When a jam system needs an interface color (a damage type in the log, a
  warning, an affix marker in the panel), pick from the gold or red family at
  a matching saturation, or fall to the gray ramp. The scene element it
  reports on (the enemy, the item) is free.
- When in doubt about whether a new surface is a new design, add to this guide
  only with confirmation.

## 10. Open questions

- Interface third hue: yes. Slate blue is a companion accent — the palette is
  gold, red, slate blue, and gray — but for now it is confined to atmospheric
  log entries and scene ambience (Section 3). It does not carry mechanical
  state; widening its interface use happens only on confirmation.
- Exact gold/red saturation tuning once rendered at terminal scale in both
  builds.
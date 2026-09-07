# Webfont attribution

## Libertinus Mono (`libertinus-mono-*.woff2`)

Copyright (c) 2012–2024, The Libertinus Project Authors.
Licensed under the SIL Open Font License, Version 1.1.
Source: https://fonts.google.com/specimen/Libertinus+Mono

## DejaVu Sans Mono (`dejavu-sans-mono.woff2`)

Copyright (c) 2003–2024, Bitstream, Inc. and the DejaVu contributors
(Arev Fonts). Converted to woff2 from the system package for size; glyphs
unchanged. Licensed under the Bitstream Vera license with the Arev Fonts
addition (permissive; redistribution allowed with this notice retained).
Source: https://dejavu-fonts.github.io/

## Why two fonts

Libertinus Mono lacks block elements and several symbols the game uses
(verified per-glyph: U+2588 █, U+2592 ▒, U+2593 ▓, U+22C5 ⋅, U+2663 ♣,
U+2248 ≈, U+2192 →). DejaVu Sans Mono covers the full game set and sits
second in the `--font-monospace` stack as the symbol fallback.

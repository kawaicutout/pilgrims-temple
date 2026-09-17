# Webfont attribution

## Libertinus Mono (`libertinus-mono-*.woff2`)

Copyright (c) 2012–2024, The Libertinus Project Authors.
Licensed under the SIL Open Font License, Version 1.1.
Source: https://fonts.google.com/specimen/Libertinus+Mono

## Why one font

Block and shade runes (U+2588 █, U+2591 ░, U+2592 ▒, U+2593 ▓) never
render as DOM text: the canvas paints them as vector rect fills. Every
other rune the game shows (verified: U+00B7 ·, U+00D7 ×, U+2014 —,
U+2212 −) sits inside the Libertinus latin ranges above, so no symbol
fallback is needed.

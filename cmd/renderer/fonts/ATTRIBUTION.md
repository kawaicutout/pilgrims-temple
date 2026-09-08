# Font attribution (embedded renderer + web fallback)

## Libertinus Mono (`libertinus-mono.ttf`)

Copyright (c) 2012–2024, The Libertinus Project Authors.
Licensed under the SIL Open Font License, Version 1.1.

The only grid face. Cell metrics derive from its measured advance and
line box. Block and shade runes (U+2588 █, U+2591 ░, U+2592 ▒, U+2593 ▓)
never reach the rasterizer: they render as full-cell vector boxes in
`cmd/renderer/draw.go` (solid, or alpha-shaded for partial shades), so
fills tile seamlessly at any cell pitch.

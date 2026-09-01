#!/usr/bin/env bash
# Pilgrim's Temple — launch at 110×34 (DESIGN 10.2).
# Tries known terminals; falls back to running inline with a size warning.
set -e
BIN="./pilgrims-temple"
COLS=110
ROWS=34
if [ ! -x "$BIN" ]; then
  echo "Building $BIN..."
  make terminal
fi
# Derive size from game/data/tuning.json if jq is available (keeps launcher in sync).
if command -v jq >/dev/null 2>&1 && [ -f game/data/tuning.json ]; then
  COLS=$(jq -r '.layout.minCols' game/data/tuning.json)
  ROWS=$(jq -r '.layout.minRows' game/data/tuning.json)
fi
run_inline() {
  cols=$(tput cols 2>/dev/null || echo 0)
  lines=$(tput lines 2>/dev/null || echo 0)
  if [ "$cols" -lt "$COLS" ] || [ "$lines" -lt "$ROWS" ]; then
    echo "Warning: terminal is ${cols}x${lines}, game wants ${COLS}x${ROWS}."
    echo "Resize your terminal or run via a launcher (xterm/gnome-terminal)."
  fi
  exec "$BIN" "$@"
}
if command -v gnome-terminal >/dev/null 2>&1; then
  exec gnome-terminal --geometry="${COLS}x${ROWS}" -- "$BIN" "$@"
elif command -v xterm >/dev/null 2>&1; then
  exec xterm -geometry "${COLS}x${ROWS}" -e "$BIN" "$@"
elif command -v konsole >/dev/null 2>&1; then
  exec konsole --geometry "${COLS}x${ROWS}" -e "$BIN" "$@"
elif command -v alacritty >/dev/null 2>&1; then
  exec alacritty -o window.dimensions.columns="$COLS" -o window.dimensions.lines="$ROWS" -e "$BIN" "$@"
else
  run_inline "$@"
fi

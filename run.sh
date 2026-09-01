#!/usr/bin/env bash
# Pilgrim's Temple - minimal launcher, opens default terminal at size from tuning.json
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN="$SCRIPT_DIR/bin/pilgrims-temple"
[ -x "$BIN" ] || BIN="$SCRIPT_DIR/pilgrims-temple"
[ -x "$BIN" ] || BIN="./pilgrims-temple"
if [ ! -x "$BIN" ]; then
  echo "Building $BIN..."
  make -C "$SCRIPT_DIR" terminal 2>/dev/null || make terminal
  [ -x "$BIN" ] || BIN="./pilgrims-temple"
fi
COLS=110
ROWS=34
# Data-driven size from game/data/tuning.json (also embedded in WASM JS)
if command -v jq >/dev/null 2>&1; then
  for p in "$SCRIPT_DIR/game/data/tuning.json" "game/data/tuning.json"; do
    if [ -f "$p" ]; then
      c=$(jq -r '.layout.minCols // empty' "$p" 2>/dev/null || true)
      r=$(jq -r '.layout.minRows // empty' "$p" 2>/dev/null || true)
      if [ -n "$c" ] && [ -n "$r" ] && [ "$c" != "null" ] && [ "$r" != "null" ]; then
        COLS=$c
        ROWS=$r
        break
      fi
    fi
  done
fi
# Open default terminal at COLS×ROWS
if [ -n "${TERMINAL:-}" ] && command -v "${TERMINAL%% *}" >/dev/null 2>&1; then
  case "$(basename "${TERMINAL%% *}")" in
    konsole) exec $TERMINAL -p TerminalColumns="$COLS" -p TerminalRows="$ROWS" -e "$BIN" "$@" ;;
    gnome-terminal*) exec $TERMINAL --geometry="${COLS}x${ROWS}" -- "$BIN" "$@" ;;
    *) exec $TERMINAL -e "$BIN" "$@" ;;
  esac
fi
if command -v xdg-terminal-exec >/dev/null 2>&1; then
  exec xdg-terminal-exec --geometry="${COLS}x${ROWS}" -e "$BIN" "$@" 2>/dev/null || exec xdg-terminal-exec -e "$BIN" "$@" 2>/dev/null || true
fi
if command -v x-terminal-emulator >/dev/null 2>&1; then
  exec x-terminal-emulator -geometry "${COLS}x${ROWS}" -e "$BIN" "$@" 2>/dev/null || exec x-terminal-emulator -e "$BIN" "$@"
fi
case "${XDG_CURRENT_DESKTOP:-}" in
  *KDE*|*Plasma*) command -v konsole >/dev/null 2>&1 && exec konsole -p TerminalColumns="$COLS" -p TerminalRows="$ROWS" -e "$BIN" "$@" ;;
  *GNOME*) command -v gnome-terminal >/dev/null 2>&1 && exec gnome-terminal --geometry="${COLS}x${ROWS}" -- "$BIN" "$@" ;;
esac
for cand in konsole gnome-terminal alacritty kitty foot xterm; do
  command -v "$cand" >/dev/null 2>&1 || continue
  case "$cand" in
    konsole) exec konsole -p TerminalColumns="$COLS" -p TerminalRows="$ROWS" -e "$BIN" "$@" ;;
    gnome-terminal) exec gnome-terminal --geometry="${COLS}x${ROWS}" -- "$BIN" "$@" ;;
    alacritty) exec alacritty -o window.dimensions.columns="$COLS" -o window.dimensions.lines="$ROWS" -e "$BIN" "$@" ;;
    kitty) exec kitty -o initial_window_width="${COLS}c" -o initial_window_height="${ROWS}c" "$BIN" "$@" ;;
    foot) exec foot -W "${COLS}x${ROWS}" -e "$BIN" "$@" ;;
    xterm) exec xterm -geometry "${COLS}x${ROWS}" -e "$BIN" "$@" ;;
  esac
done
exec "$BIN" "$@"

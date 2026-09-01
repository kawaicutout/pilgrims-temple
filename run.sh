#!/usr/bin/env bash
# Pilgrim's Temple — open default terminal at 110×34
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
if command -v jq >/dev/null 2>&1; then
  for p in "$SCRIPT_DIR/game/data/tuning.json" "game/data/tuning.json"; do
    if [ -f "$p" ]; then
      c=$(jq -r '.layout.minCols // empty' "$p" 2>/dev/null || true)
      r=$(jq -r '.layout.minRows // empty' "$p" 2>/dev/null || true)
      [ -n "$c" ] && [ -n "$r" ] && [ "$c" != "null" ] && [ "$r" != "null" ] && COLS=$c && ROWS=$r && break
    fi
  done
fi

# Simple: open system default terminal at COLS×ROWS, no inline resize hacks
if [ -n "${TERMINAL:-}" ] && command -v "${TERMINAL%% *}" >/dev/null 2>&1; then
  # shellcheck disable=SC2086
  case "$(basename "${TERMINAL%% *}")" in
    konsole) exec $TERMINAL -p TerminalColumns="$COLS" -p TerminalRows="$ROWS" -e "$BIN" "$@" ;;
    gnome-terminal*) exec $TERMINAL --geometry="${COLS}x${ROWS}" -- "$BIN" "$@" ;;
    alacritty) exec $TERMINAL -o window.dimensions.columns="$COLS" -o window.dimensions.lines="$ROWS" -e "$BIN" "$@" ;;
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
# Fallback: no GUI terminal found, run inline
exec "$BIN" "$@"

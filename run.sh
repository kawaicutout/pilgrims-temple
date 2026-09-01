#!/usr/bin/env bash
# Pilgrim's Temple — launch at 110×34 (DESIGN 10.2).
# Prefers the system default terminal, then falls back to known terminals,
# finally runs inline with a size warning.
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

# Try to launch $BIN in a terminal with requested geometry.
try_term() {
  local term="$1"
  local base
  base=$(basename "$term" 2>/dev/null || echo "$term")
  case "$base" in
    gnome-terminal|gnome-terminal.wrapper)
      exec "$term" --geometry="${COLS}x${ROWS}" -- "$BIN" "$@"
      ;;
    konsole)
      exec "$term" --qwindowgeometry "${COLS}x${ROWS}" -e "$BIN" "$@"
      ;;
    xfce4-terminal)
      exec "$term" --geometry="${COLS}x${ROWS}" -e "$BIN" "$@"
      ;;
    mate-terminal)
      exec "$term" --geometry="${COLS}x${ROWS}" -e "$BIN" "$@"
      ;;
    alacritty)
      exec "$term" -o window.dimensions.columns="$COLS" -o window.dimensions.lines="$ROWS" -e "$BIN" "$@"
      ;;
    kitty)
      exec "$term" -o remember_window_size=no -o initial_window_width="${COLS}c" -o initial_window_height="${ROWS}c" "$BIN" "$@"
      ;;
    foot)
      exec "$term" -W "${COLS}x${ROWS}" -e "$BIN" "$@"
      ;;
    xterm|uxterm|koi8rxterm)
      exec "$term" -geometry "${COLS}x${ROWS}" -e "$BIN" "$@"
      ;;
    *)
      if "$term" -e true 2>/dev/null; then
        exec "$term" -e "$BIN" "$@"
      elif "$term" -- true 2>/dev/null; then
        exec "$term" -- "$BIN" "$@"
      else
        exec "$term" "$BIN" "$@"
      fi
      ;;
  esac
}

# 1. Respect $TERMINAL (user override)
if [ -n "${TERMINAL:-}" ] && command -v "${TERMINAL%% *}" >/dev/null 2>&1; then
  # shellcheck disable=SC2086
  try_term $TERMINAL
fi

# 2. Freedesktop / Debian system defaults
if command -v xdg-terminal-exec >/dev/null 2>&1; then
  exec xdg-terminal-exec -e "$BIN" "$@" 2>/dev/null || true
fi
if command -v x-terminal-emulator >/dev/null 2>&1; then
  try_term x-terminal-emulator
fi

# 3. Desktop-specific default
case "${XDG_CURRENT_DESKTOP:-}" in
  *KDE*|*Plasma*)
    command -v konsole >/dev/null 2>&1 && try_term konsole
    ;;
  *GNOME*)
    command -v gnome-terminal >/dev/null 2>&1 && try_term gnome-terminal
    ;;
  *XFCE*)
    command -v xfce4-terminal >/dev/null 2>&1 && try_term xfce4-terminal
    ;;
  *MATE*)
    command -v mate-terminal >/dev/null 2>&1 && try_term mate-terminal
    ;;
esac

# 4. Ordered fallback — covers most distros without xdg-terminal-exec
for cand in konsole gnome-terminal xfce4-terminal mate-terminal alacritty kitty foot xterm uxterm; do
  if command -v "$cand" >/dev/null 2>&1; then
    try_term "$cand"
  fi
done

# 5. No GUI terminal found — run inline
run_inline "$@"

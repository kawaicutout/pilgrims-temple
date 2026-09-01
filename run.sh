#!/usr/bin/env bash
# Pilgrim's Temple — launch at 110×34 (DESIGN 10.2).
# Prefers the system default terminal, then falls back to known terminals,
# finally runs inline. Tries to auto-resize via window escape sequence.
set -e
BIN="./pilgrims-temple"
COLS=110
ROWS=34
if [ ! -x "$BIN" ]; then
  echo "Building $BIN..."
  make terminal
fi
if command -v jq >/dev/null 2>&1 && [ -f game/data/tuning.json ]; then
  COLS=$(jq -r '.layout.minCols' game/data/tuning.json)
  ROWS=$(jq -r '.layout.minRows' game/data/tuning.json)
fi

# Try to resize the current terminal via xterm escape \e[8;rows;cols t
# Works in xterm, konsole, alacritty, gnome-terminal, foot (when supported).
try_resize_current() {
  if [ -t 1 ] && [ -t 2 ]; then
    printf '\033[8;%d;%dt' "$ROWS" "$COLS" 2>/dev/null || true
    # Give compositor a moment to apply
    sleep 0.15 2>/dev/null || true
  fi
}

run_inline() {
  # First try to auto-resize the current window (fixes manual terminal case)
  try_resize_current
  cols=$(tput cols 2>/dev/null || echo 0)
  lines=$(tput lines 2>/dev/null || echo 0)
  if [ "$cols" -lt "$COLS" ] || [ "$lines" -lt "$ROWS" ]; then
    echo "Warning: terminal is ${cols}x${lines}, game wants ${COLS}x${ROWS}."
    echo "If auto-resize failed, resize manually or run via a GUI launcher."
    echo "Tip: drag window larger or run: printf '\\033[8;${ROWS};${COLS}t'"
  fi
  exec "$BIN" "$@"
}

# Build a shell snippet that resizes the new window then execs the game.
# Keeps numpad intact by preserving TERM and not touching keypad mode.
resize_and_exec_snippet() {
  # Use $BIN and pass through args; escape sequence is sent from inside the new terminal
  printf 'printf \033[8;%d;%dt 2>/dev/null; sleep 0.05 2>/dev/null; exec "%s" "$@"' "$ROWS" "$COLS" "$BIN"
}

try_term() {
  local term="$1"
  local base
  base=$(basename "$term" 2>/dev/null || echo "$term")
  local inner
  inner=$(resize_and_exec_snippet)
  case "$base" in
    gnome-terminal|gnome-terminal.wrapper)
      # --geometry is honored, but also send escape as fallback
      exec "$term" --geometry="${COLS}x${ROWS}" -- sh -c "$inner" sh "$BIN" "$@"
      ;;
    konsole)
      exec "$term" --qwindowgeometry "${COLS}x${ROWS}" -e sh -c "$inner" sh "$BIN" "$@"
      ;;
    xfce4-terminal)
      exec "$term" --geometry="${COLS}x${ROWS}" -e sh -c "$inner" sh "$BIN" "$@"
      ;;
    mate-terminal)
      exec "$term" --geometry="${COLS}x${ROWS}" -e sh -c "$inner" sh "$BIN" "$@"
      ;;
    alacritty)
      exec "$term" -o window.dimensions.columns="$COLS" -o window.dimensions.lines="$ROWS" -e sh -c "$inner" sh "$BIN" "$@"
      ;;
    kitty)
      exec "$term" -o remember_window_size=no -o initial_window_width="${COLS}c" -o initial_window_height="${ROWS}c" sh -c "$inner" sh "$BIN" "$@"
      ;;
    foot)
      exec "$term" -W "${COLS}x${ROWS}" -e sh -c "$inner" sh "$BIN" "$@"
      ;;
    xterm|uxterm|koi8rxterm)
      exec "$term" -geometry "${COLS}x${ROWS}" -e sh -c "$inner" sh "$BIN" "$@"
      ;;
    *)
      if "$term" -e true 2>/dev/null; then
        exec "$term" -e sh -c "$inner" sh "$BIN" "$@"
      elif "$term" -- true 2>/dev/null; then
        exec "$term" -- sh -c "$inner" sh "$BIN" "$@"
      else
        exec "$term" sh -c "$inner" sh "$BIN" "$@"
      fi
      ;;
  esac
}

# 1. Respect $TERMINAL
if [ -n "${TERMINAL:-}" ] && command -v "${TERMINAL%% *}" >/dev/null 2>&1; then
  # shellcheck disable=SC2086
  try_term $TERMINAL
fi

# 2. Freedesktop / Debian defaults
if command -v xdg-terminal-exec >/dev/null 2>&1; then
  inner=$(resize_and_exec_snippet)
  exec xdg-terminal-exec -e sh -c "$inner" sh "$BIN" "$@" 2>/dev/null || true
fi
if command -v x-terminal-emulator >/dev/null 2>&1; then
  try_term x-terminal-emulator
fi

# 3. Desktop-specific
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

# 4. Ordered fallback
for cand in konsole gnome-terminal xfce4-terminal mate-terminal alacritty kitty foot xterm uxterm; do
  if command -v "$cand" >/dev/null 2>&1; then
    try_term "$cand"
  fi
done

# 5. No GUI terminal — resize current and run inline (preserves numpad)
run_inline "$@"

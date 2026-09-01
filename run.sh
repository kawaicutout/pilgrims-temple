#!/usr/bin/env bash
# Pilgrim's Temple — launch at 110×34 (DESIGN 10.2).
# Tries to auto-resize the current terminal in-place; only spawns a new
# window when not in a TTY or when resize fails. Preserves numpad.
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

# Try to resize the current terminal window via escape sequence.
# Works in xterm, konsole, alacritty, gnome-terminal, foot when
# window operations are enabled. Also try wmctrl/xdotool as fallback.
try_resize_current() {
  if [ ! -t 1 ] || [ ! -t 2 ]; then
    return 1
  fi
  # 1) xterm escape \e[8;rows;cols t
  printf '\033[8;%d;%dt' "$ROWS" "$COLS" > /dev/tty 2>/dev/null || true
  # 2) konsole/gnome may need flush
  sleep 0.12 2>/dev/null || true
  # 3) fallback via wmctrl (pixel approximation: assume 9x19 cell)
  if command -v wmctrl >/dev/null 2>&1; then
    # wmctrl expects pixels; approximate
    px_w=$(( COLS * 9 + 16 ))
    px_h=$(( ROWS * 19 + 48 ))
    wmctrl -r :ACTIVE: -e 0,-1,-1,"$px_w","$px_h" 2>/dev/null || true
    sleep 0.12 2>/dev/null || true
  fi
  # 4) stty as last resort (sets pty, not window, but helps tcell Size)
  stty rows "$ROWS" cols "$COLS" 2>/dev/null || true
}

run_inline() {
  try_resize_current
  cols=$(tput cols 2>/dev/null || echo 0)
  lines=$(tput lines 2>/dev/null || echo 0)
  if [ "$cols" -lt "$COLS" ] || [ "$lines" -lt "$ROWS" ]; then
    echo "Note: terminal is ${cols}x${lines}, game wants ${COLS}x${ROWS}."
    echo "If auto-resize failed, drag the window larger or press F11/maximize."
    echo "You can also set TERM size manually: printf '\033[8;${ROWS};${COLS}t'"
  else
    echo "Resized to ${cols}x${lines}."
  fi
  exec "$BIN" "$@"
}

# If we're already in an interactive terminal, prefer to resize in-place
# (preserves numpad, profile, and avoids spawning a second window).
if [ -t 0 ] && [ -t 1 ]; then
  # Check if current size already sufficient — if so, run inline directly
  cur_cols=$(tput cols 2>/dev/null || echo 0)
  cur_lines=$(tput lines 2>/dev/null || echo 0)
  if [ "$cur_cols" -ge "$COLS" ] && [ "$cur_lines" -ge "$ROWS" ]; then
    exec "$BIN" "$@"
  fi
  # Try to resize current window; if it succeeds, run inline
  try_resize_current
  cur_cols2=$(tput cols 2>/dev/null || echo 0)
  cur_lines2=$(tput lines 2>/dev/null || echo 0)
  if [ "$cur_cols2" -ge "$COLS" ] && [ "$cur_lines2" -ge "$ROWS" ]; then
    exec "$BIN" "$@"
  fi
  # If resize didn't reach target but we're still in a TTY, run inline anyway
  # with a warning — the game's own resize prompt will show. This preserves
  # numpad (which breaks when spawning a new konsole with wrong flags).
  # Only spawn a new window if explicitly requested via --new-window or if
  # caller is not interactive (e.g., double-click from Dolphin).
  case "${1:-}" in
    --new-window|-n)
      shift
      ;;
    *)
      # No --new-window and we're in a TTY: stay inline (user said manual exec works)
      # but still offer to spawn if size is way too small
      if [ "$cur_cols2" -ge 80 ] && [ "$cur_lines2" -ge 24 ]; then
        echo "Running in current terminal (${cur_cols2}x${cur_lines2}); use --new-window to force a new window."
        exec "$BIN" "$@"
      fi
      ;;
  esac
fi

# Not in a TTY or current window too small and user asked for new window:
# spawn the system default terminal with correct geometry.

resize_and_exec_snippet() {
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
      exec "$term" --geometry="${COLS}x${ROWS}" -- sh -c "$inner" sh "$BIN" "$@"
      ;;
    konsole)
      # Use profile properties for character size, plus escape fallback
      exec "$term" -p TerminalColumns="$COLS" -p TerminalRows="$ROWS" --qwindowgeometry "${COLS}x${ROWS}" -e sh -c "$inner" sh "$BIN" "$@"
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

if [ -n "${TERMINAL:-}" ] && command -v "${TERMINAL%% *}" >/dev/null 2>&1; then
  # shellcheck disable=SC2086
  try_term $TERMINAL
fi
if command -v xdg-terminal-exec >/dev/null 2>&1; then
  inner=$(resize_and_exec_snippet)
  exec xdg-terminal-exec -e sh -c "$inner" sh "$BIN" "$@" 2>/dev/null || true
fi
if command -v x-terminal-emulator >/dev/null 2>&1; then
  try_term x-terminal-emulator
fi
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
for cand in konsole gnome-terminal xfce4-terminal mate-terminal alacritty kitty foot xterm uxterm; do
  if command -v "$cand" >/dev/null 2>&1; then
    try_term "$cand"
  fi
done

# Last resort: try inline even if not a TTY (will show warning)
run_inline "$@"

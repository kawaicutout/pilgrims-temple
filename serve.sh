#!/usr/bin/env bash
# serve.sh — build latest web build and serve locally on a free port
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "Building web (make web)..."
if ! make web; then
  echo "make web failed" >&2
  exit 1
fi

# Find a free port starting at 8000
find_port() {
  local start=${1:-8000}
  local p=$start
  local max=$((start + 200))
  while [ "$p" -le "$max" ]; do
    # python bind is the most reliable cross-platform probe
    if python3 -c "import socket; s=socket.socket(); s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1); s.bind(('127.0.0.1', $p)); s.close()" 2>/dev/null; then
      # double-check with ss/lsof if available, but don't fail if missing
      if command -v ss >/dev/null 2>&1; then
        if ss -ltn 2>/dev/null | grep -qE ":${p}\b"; then p=$((p+1)); continue; fi
      fi
      if command -v lsof >/dev/null 2>&1; then
        if lsof -i :"$p" -sTCP:LISTEN >/dev/null 2>&1; then p=$((p+1)); continue; fi
      fi
      echo "$p"
      return 0
    fi
    p=$((p+1))
  done
  # Fallback: ask OS for any free port
  python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1])'
}

PORT=$(find_port 8000)
if [ -z "$PORT" ] || [ "$PORT" = "0" ]; then
  PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1])')
fi

echo ""
echo "Serving web/ at http://127.0.0.1:${PORT}/  (also http://localhost:${PORT}/)"
echo "Document root: $SCRIPT_DIR/web"
echo "Press Ctrl+C to stop."
echo ""

# Try to open browser if possible (non-fatal)
if command -v xdg-open >/dev/null 2>&1; then
  (xdg-open "http://127.0.0.1:${PORT}/" >/dev/null 2>&1 & disown) || true
elif command -v open >/dev/null 2>&1; then
  (open "http://127.0.0.1:${PORT}/" >/dev/null 2>&1 & disown) || true
fi

# Serve — prefer python3 --directory (3.7+), fallback to cd web
if python3 -m http.server --help 2>&1 | grep -q -- "--directory"; then
  exec python3 -m http.server "$PORT" --directory web --bind 127.0.0.1
else
  cd web
  exec python3 -m http.server "$PORT" --bind 127.0.0.1
fi

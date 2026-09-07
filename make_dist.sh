#!/usr/bin/env bash
# make_dist.sh — one-click itch.io release.
# Builds windows.zip (exe + data + launcher), linux.zip (binary + data +
# launcher), and web.zip (itch HTML5 upload) into dist/.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"
DIST="$SCRIPT_DIR/dist"

command -v go >/dev/null 2>&1 || { echo "missing: go" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "missing: python3" >&2; exit 1; }

echo "Building desktop binaries (make bin)..."
make bin

echo "Building web (make web)..."
make web
for f in web/index.html web/tokens.css web/wasm_exec.js web/main.wasm.br; do
  [ -f "$f" ] || { echo "missing build artifact: $f" >&2; exit 1; }
done

rm -rf "$DIST"
mkdir -p "$DIST/stage/windows" "$DIST/stage/linux" "$DIST/stage/web"

cp bin/pilgrims-temple.exe run.bat "$DIST/stage/windows/"
cp bin/pilgrims-temple run.sh "$DIST/stage/linux/"
chmod +x "$DIST/stage/linux/run.sh"
cp -r game/data "$DIST/stage/windows/data"
cp -r game/data "$DIST/stage/linux/data"
cp web/index.html web/tokens.css web/wasm_exec.js web/main.wasm.br "$DIST/stage/web/"

python3 - "$DIST" <<'EOF'
import sys, zipfile
from pathlib import Path
dist = Path(sys.argv[1])
stage = dist / "stage"
plans = {
    "windows.zip": ["pilgrims-temple.exe", "run.bat", "data"],
    "linux.zip": ["pilgrims-temple", "run.sh", "data"],
    "web.zip": ["index.html", "tokens.css", "wasm_exec.js", "main.wasm.br"],
}
roots = {"windows.zip": "windows", "linux.zip": "linux", "web.zip": "web"}
for archive, names in plans.items():
    root = stage / roots[archive]
    out = dist / archive
    if out.exists():
        out.unlink()
    with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED, compresslevel=9) as z:
        for name in names:
            p = root / name
            if p.is_dir():
                for f in sorted(p.rglob("*")):
                    if f.is_file():
                        z.write(f, (Path(roots[archive]) / name / f.relative_to(p)).as_posix())
            else:
                z.write(p, (Path(roots[archive]) / name).as_posix())
    print(f"{archive}: {out.stat().st_size} bytes")
EOF

echo ""
echo "dist/ contents:"
ls -lh "$DIST"/*.zip

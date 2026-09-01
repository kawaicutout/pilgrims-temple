#!/usr/bin/env bash
# Pilgrim's Temple - minimal launcher
BIN="./pilgrims-temple"
[ -x "$BIN" ] || BIN="./bin/pilgrims-temple"
exec "$BIN" "$@"

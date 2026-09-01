@echo off
REM Pilgrim's Temple — launch at 110x34 (DESIGN 10.2).
REM Requires modern Windows Terminal or conhost. Uses `mode` to request size.
setlocal
set COLS=110
set ROWS=34
if not exist pilgrims-temple.exe (
  echo Building pilgrims-temple.exe...
  go build -ldflags="-s -w" -o pilgrims-temple.exe ./cmd/terminal
)
REM Try to resize console buffer/window. Silently ignore if unsupported (e.g. Windows Terminal).
mode con: cols=%COLS% lines=%ROWS% >nul 2>&1
pilgrims-temple.exe %*

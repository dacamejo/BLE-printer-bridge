@echo off
setlocal enabledelayedexpansion

REM === Settings (match your config.toml) ===
set HOST=127.0.0.1
set PORT=17800

REM === Start bridge in a new window ===
echo Starting BLE Printer Bridge...
for %%I in ("%~dp0..\..") do set "ROOT_DIR=%%~fI"
start "ble-printer-bridge" cmd /k ^
  "cd /d "%ROOT_DIR%" && go run .\cmd\agent"

REM === Wait for server to come up ===
echo Waiting for bridge to start...
timeout /t 2 /nobreak >nul

REM === Health check (optional) ===
echo Checking /health...
powershell -NoProfile -Command ^
  "try { irm http://%HOST%:%PORT%/health | out-null; 'OK' } catch { 'NOT OK' }"

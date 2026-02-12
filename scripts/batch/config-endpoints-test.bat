@echo off
setlocal enabledelayedexpansion

REM === Settings (match your config.toml) ===
set HOST=127.0.0.1
set PORT=17800
set API_KEY=pos-local-print-2026-demo-key
set BLE_ADDR=66:22:B6:5C:5C:3C

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

REM === Connect to printer ===
echo.
echo Connecting to %BLE_ADDR% ...
powershell -NoProfile -Command ^
  "$h=@{ 'x-api-key'='%API_KEY%' }; $b=@{ address = '%BLE_ADDR%' } | ConvertTo-Json; irm http://%HOST%:%PORT%/ble/connect -Method POST -Headers $h -Body $b -ContentType 'application/json'"

REM === Reading current config ===
echo.
echo Reading current config...
powershell -NoProfile -Command ^
  "$h=@{ 'x-api-key'='%API_KEY%' }; irm http://%HOST%:%PORT%/config -Headers $h | ConvertTo-Json -Depth 8"

REM === Updating current config ===
echo.
echo Updating config with the same values...
powershell -NoProfile -Command ^
  "$h=@{ 'x-api-key'='%API_KEY%' }; " ^
  "$cfg=(irm http://%HOST%:%PORT%/config -Headers $h).config; " ^
  "$body=$cfg | ConvertTo-Json -Depth 8; " ^
  "irm http://%HOST%:%PORT%/config -Method POST -Headers $h -Body $body -ContentType 'application/json' | ConvertTo-Json -Depth 8"

echo.
echo Config read and update complete.
pause

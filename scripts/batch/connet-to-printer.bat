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

REM echo.
REM echo Scanning BLE devices (8s)...
REM powershell -NoProfile -Command ^
REM   "$h=@{ 'x-api-key'='%API_KEY%' }; $b=@{ seconds = 8 } | ConvertTo-Json; irm http://%HOST%:%PORT%/ble/scan -Method POST -Headers $h -Body $b -ContentType 'application/json' | ConvertTo-Json -Depth 8"

REM echo.
REM set /p BLE_ADDR=Enter printer BLE address (e.g. 66:22:B6:5C:5C:3C): 

echo.
echo Connecting to %BLE_ADDR% ...
powershell -NoProfile -Command ^
  "$h=@{ 'x-api-key'='%API_KEY%' }; $b=@{ address = '%BLE_ADDR%' } | ConvertTo-Json; irm http://%HOST%:%PORT%/ble/connect -Method POST -Headers $h -Body $b -ContentType 'application/json'"

echo.
echo Printing test...
powershell -NoProfile -Command ^
  "$nl = [Environment]::NewLine; " ^
  "$h=@{ 'x-api-key'='%API_KEY%' }; " ^
  "$b=@{ text = ('Printer ready' + $nl + $nl + $nl + $nl) } | ConvertTo-Json; " ^
  "irm http://%HOST%:%PORT%/print/text -Method POST -Headers $h -Body $b -ContentType 'application/json'"

echo.
echo Printer ready
pause

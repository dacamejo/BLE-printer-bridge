# BLE Printer Bridge

Lightweight local HTTP bridge for sending ESC/POS print data to BLE receipt printers.

**Suggested GitHub repository description:**

`Local Go HTTP bridge that connects web apps to BLE receipt printers (ESC/POS) with API-key auth, CORS controls, and runtime config updates.`

## What it does

- Exposes a local API (default `127.0.0.1:17800`).
- Scans, connects, disconnects, and inspects BLE devices.
- Sends receipt data as:
  - structured text (`/print/text`), or
  - raw bytes via base64 (`/print/raw`).
- Persists runtime configuration updates to `config.toml`.
- Writes logs to file and optionally to console.

## Requirements

- Windows 10/11 (recommended for BLE flow)
- Go 1.21+
- Bluetooth-enabled machine
- BLE printer advertising

## Run the bridge

### Option 1: Start directly (PowerShell)

```powershell
go run .\cmd\agent
```

The bridge reads `config.toml` from the repository root.

For a fresh setup:

```powershell
Copy-Item .\config.example.toml .\config.toml
```

Then replace placeholder values in `config.toml` with your local BLE and API key settings.

### Option 2: Start from batch script (recommended for Windows operators)

Use `scripts/batch/turn-on-agent.bat`.

What it does:

- Resolves the repository root from the script location.
- Starts the bridge in a separate terminal window.
- Waits briefly for startup.
- Runs a quick `/health` check using PowerShell.

```powershell
.\scripts\batch\turn-on-agent.bat
```

Related helper scripts:

- `scripts/batch/connet-to-printer.bat` → starts the bridge, connects to configured BLE address, and prints a quick test line.
- `scripts/batch/config-endpoints-test.bat` → starts the bridge, exercises `/config` read/update endpoints.

## Configuration

Main settings are in `config.toml`:

- `server.host`, `server.port`
- `auth.api_key` (set this to a strong local secret)
- `ble.device_name_contains`
- `ble.printer_address`
- `ble.service_uuid`
- `ble.write_characteristic_uuid`
- `ble.chunk_size`
- `ble.write_with_response`
- `logging.file_path`
- `logging.console_verbose`
- `cors.allow_origins`
- `cors.allow_origin_patterns`

Environment variable overrides supported:

- `BRIDGE_CORS_ALLOW_ORIGINS`
- `BRIDGE_CORS_ALLOW_ORIGIN_PATTERNS`
- `AGENT_CORS_ALLOW_ORIGINS`
- `AGENT_CORS_ALLOW_ORIGIN_PATTERNS`

`AGENT_*` variables are still accepted for backward compatibility.

## Auth

All endpoints except `/health` require:

```http
x-api-key: <your_api_key>
```

## API

### Health

- `GET /health` (no auth)

### BLE

- `POST /ble/scan`
  - body: `{ "seconds": 8 }` (optional; defaults to 8)
- `POST /ble/connect`
  - body: `{ "address": "AA:BB:CC:DD:EE:FF" }`
- `POST /ble/disconnect`
- `GET /ble/status`
- `POST /ble/describe`

### Print

- `POST /print/text`
  - body: `{ "text": "Hello" }`
  - behavior: wraps text with ESC/POS init + newline (if missing) + partial cut
- `POST /print/raw`
  - body: `{ "base64": "..." }`

### Config

- `GET /config`
- `POST /config`
  - body: full config object
  - saves updated config to `config.toml`

## Quick examples (PowerShell)

### Health

```powershell
Invoke-RestMethod http://127.0.0.1:17800/health
```

### Scan

```powershell
Invoke-RestMethod http://127.0.0.1:17800/ble/scan `
  -Method POST `
  -Headers @{ "x-api-key"="pos-local-print-2026-demo-key" } `
  -Body (@{ seconds = 8 } | ConvertTo-Json) `
  -ContentType "application/json"
```

### Connect

```powershell
Invoke-RestMethod http://127.0.0.1:17800/ble/connect `
  -Method POST `
  -Headers @{ "x-api-key"="pos-local-print-2026-demo-key" } `
  -Body (@{ address = "66:22:B6:5C:5C:3C" } | ConvertTo-Json) `
  -ContentType "application/json"
```

### Print text

```powershell
Invoke-RestMethod http://127.0.0.1:17800/print/text `
  -Method POST `
  -Headers @{ "x-api-key"="pos-local-print-2026-demo-key" } `
  -Body (@{ text = "Test print" } | ConvertTo-Json) `
  -ContentType "application/json"
```

## Notes

- `config.example.toml` is safe to commit and share; keep your real `config.toml` values private.

- If `logging.file_path` points to a non-existing directory, it is created automatically.
- BLE scan is single-flight; concurrent scans are rejected.

## Web app integration impact

- No API path changes were made (`/health`, `/ble/*`, `/print/*`, `/config` are unchanged).
- No request/response contract changes were made.
- Existing clients can keep using `x-api-key` and the same bridge host/port settings.
- If your deployment tooling used `AGENT_CORS_*` environment variables, they still work; `BRIDGE_CORS_*` is now also supported.

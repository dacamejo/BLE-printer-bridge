# BLE Printer Bridge

**BLE Printer Bridge is a local Go service that lets web or desktop apps print ESC/POS receipts to Bluetooth Low Energy (BLE) printers through a simple HTTP API, solving the browser/OS limitations of direct BLE printing.**

BLE printer support from browsers and cross-platform app runtimes is often inconsistent or restricted. This project provides a stable localhost bridge so your app can print receipts without implementing low-level BLE logic in every client.

## Who this is for

This project is for developers building POS, kiosk, order, queue, or receipt workflows who need reliable local printing from:

- Web apps running in a browser
- Electron or desktop helper apps
- Local line-of-business tools that can call HTTP APIs

Use it when you want one small local service that handles BLE discovery/connection and ESC/POS writes, while your app stays focused on business logic.

## Quick start

1. **Set up config**

```powershell
Copy-Item .\config.example.toml .\config.toml
```

2. **Edit `config.toml`**
   - Set `auth.api_key`
   - Set BLE printer details (`printer_address`, service/characteristic UUIDs) as needed

3. **Run the bridge**

```powershell
go run .\cmd\agent
```

4. **Verify service health**

```powershell
Invoke-RestMethod http://127.0.0.1:17800/health
```

5. **Send a test print**

```powershell
Invoke-RestMethod http://127.0.0.1:17800/print/text `
  -Method POST `
  -Headers @{ "x-api-key"="your-local-api-key" } `
  -Body (@{ text = "Hello from BLE Printer Bridge" } | ConvertTo-Json) `
  -ContentType "application/json"
```

## Features

- Localhost HTTP API for BLE printer workflows
- BLE scan, connect, disconnect, and status endpoints
- ESC/POS text print endpoint and raw-byte print endpoint (base64)
- API-key protection for non-health endpoints
- Config file + runtime config update endpoint
- Configurable CORS allowlists/patterns
- File logging with optional console verbosity
- Backward-compatible CORS environment variable names

## When to use this vs alternatives

### Use BLE Printer Bridge when

- You need to print from a browser or local app to **BLE-only ESC/POS printers**
- You want a **self-hosted local bridge** instead of cloud relay services
- You need a simple API contract between application and printer transport

### Consider alternatives when

- Your printers are network/LAN printers (TCP/9100 may be simpler)
- Your environment already has a managed print spooler or printer gateway
- You need vendor SDK features beyond standard ESC/POS operations

## Why this exists

Many teams can generate receipt data but get blocked on the last mile: secure, reliable communication with local BLE receipt printers from modern app runtimes. This bridge isolates that hardware complexity into one local process.

## When not to use it

- If you cannot run local companion software on the client machine
- If your printers are not BLE-capable
- If your use case requires cloud-only, zero-local-install architecture

## Requirements

- Windows 10/11 (recommended for BLE flow)
- Go 1.21+
- Bluetooth-enabled machine
- BLE printer advertising

## Configuration

Primary settings live in `config.toml`:

- `server.host`, `server.port`
- `auth.api_key`
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

Environment variable overrides:

- `BRIDGE_CORS_ALLOW_ORIGINS`
- `BRIDGE_CORS_ALLOW_ORIGIN_PATTERNS`
- `AGENT_CORS_ALLOW_ORIGINS` (backward compatibility)
- `AGENT_CORS_ALLOW_ORIGIN_PATTERNS` (backward compatibility)

## API overview

All endpoints except `/health` require:

```http
x-api-key: <your_api_key>
```

### Health

- `GET /health`

### BLE

- `POST /ble/scan`
- `POST /ble/connect`
- `POST /ble/disconnect`
- `GET /ble/status`
- `POST /ble/describe`

### Print

- `POST /print/text`
- `POST /print/raw`

### Config

- `GET /config`
- `POST /config`

## Notes

- Keep real `config.toml` values private.
- If `logging.file_path` points to a missing directory, it is created automatically.
- BLE scan is single-flight; concurrent scans are rejected.

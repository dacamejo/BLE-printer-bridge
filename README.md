# BLE Printer Bridge

BLE Printer Bridge is a local Go service for developers who need reliable ESC/POS receipt printing to BLE printers from web or desktop apps.
It solves the last-mile problem where browsers and cross-platform runtimes can generate receipt data but cannot safely/reliably talk to local BLE hardware.
Your app sends HTTP requests to `http://127.0.0.1:<PORT>`, and this bridge handles BLE scan/connect/write operations on the same machine.
Use it for POS, kiosk, order, queue, and other local receipt workflows where a lightweight self-hosted bridge is acceptable.

## 30-second quick start (curl)

```bash
cp config.example.toml config.toml
# Edit config.toml and set placeholders like:
# - auth.api_key="<YOUR_LOCAL_API_KEY>"
# - ble.printer_address="<BLE_PRINTER_MAC>"
# - cors.allow_origins="<YOUR_WEB_APP_URL>"
go run ./cmd/agent
curl -sS http://127.0.0.1:17800/health
curl -sS -X POST "http://127.0.0.1:17800/print/text" \
  -H "x-api-key: <YOUR_LOCAL_API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{"text":"Test receipt from BLE Printer Bridge"}'
```

## Who this is for

This project is for developers building POS, kiosk, order, queue, or receipt workflows who need reliable local printing from:

- Web apps running in a browser
- Electron or desktop helper apps
- Local line-of-business tools that can call HTTP APIs

Use it when you want one local service to handle BLE discovery/connection and ESC/POS writes, while your app stays focused on business logic.

## Why this exists

Browsers and many desktop runtimes have intentional limits around direct Bluetooth access, and BLE printer behavior varies by OS and hardware stack. In practice, this makes direct in-app BLE printing brittle or unavailable. This bridge isolates BLE complexity into one localhost process with a stable HTTP contract.

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

## When NOT to use this

- You cannot install/run local companion software on the client machine
- You require a cloud-only architecture with no localhost component
- Your printers are not BLE-capable or do not use ESC/POS-compatible commands

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
x-api-key: <YOUR_LOCAL_API_KEY>
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

## Repository hygiene

- Treat `config.toml` as machine-local and sensitive; do not commit it.
- Keep `config.example.toml` as the only tracked template.
- Keep real API keys, local BLE addresses, and environment-specific origins out of version control.

## Notes

- If `logging.file_path` points to a missing directory, it is created automatically.
- BLE scan is single-flight; concurrent scans are rejected.

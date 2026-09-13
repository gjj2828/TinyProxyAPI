# TinyProxyAPI

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/gjj2828/TinyProxyAPI)](https://goreportcard.com/report/github.com/gjj2828/TinyProxyAPI)
[![CI](https://github.com/gjj2828/TinyProxyAPI/actions/workflows/ci.yml/badge.svg)](https://github.com/gjj2828/TinyProxyAPI/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/gjj2828/TinyProxyAPI)](https://github.com/gjj2828/TinyProxyAPI/releases)

[中文](./README.md) | Ultra-lightweight API routing proxy that forwards requests to the corresponding platform based on the model name in the request.

## Background

Different API proxy providers offer different model routing options at different rates: some offer `luna` forwarding but at higher multipliers, while others only have `terra`/`sol`/`astra` at lower rates. Requests routed through CPA proxies are often blocked or flagged by these services. TinyProxyAPI solves this by acting as a **pass-through** proxy — besides the Host and Authorization headers that must be replaced for routing, all other headers and the entire request body are forwarded unchanged, making forwarded requests appear nearly identical to direct ones and avoiding anti-fraud detection on the proxy services. You can aggregate multiple proxy providers and route different models to the most cost-effective provider by editing `config.toml`, with no changes needed on the Codex client side.

## Features

- **Pass-Through** - Uses `httputil.ReverseProxy`, forwarding Body unchanged while only modifying routing-required headers (Host and Authorization), with no content inspection or modification
- **Flexible Routing** - Full model name matching via TOML configuration
- **Platform Abstraction** - Multiple models on the same platform share base_url and api_key
- **Hot Reload** - Changes to `config.toml` take effect on save (via cross-platform fsnotify); Linux/Mac can also trigger with SIGHUP
- **Auto-Initialization** - Automatically creates a default `config.toml` if missing and opens it in your editor
- **System Tray** - On Windows, hides the console window and manages the program via tray icon (show/hide log window, open config file, exit)
- **Log Window** - View program runtime logs, with optional auto-show on startup

> **Currently only `/v1/responses` endpoint is supported** (OpenAI Responses API). Other endpoints like `/v1/chat/completions` are not yet implemented.

## Quick Start

### Configuration File

```toml
# config.toml
allowed_keys = ["your-client-key-here"]

[server]
addr = ":3178"

[default]
platform = "openai"

[platforms]
"openai" = { base_url = "https://api.openai.com", api_key = "sk-your-key" }
"anthropic" = { base_url = "https://api.anthropic.com", api_key = "sk-ant-your-key" }

[models]
"gpt-4o" = "openai"
"gpt-4o-mini" = "openai"
"claude-sonnet-4-20250514" = "anthropic"
```

> ⚠️ **TOML Gotcha**: `allowed_keys` is a top-level key and must be placed **before all `[table]` sections**.
> If placed after `[default]`, it gets grouped into the `[default]` table, but since the `DefaultConfig` struct has no `AllowedKeys` field, the value is **silently ignored**, causing all client keys to be rejected by the whitelist and the proxy returns 401 `invalid api key`.

### Running

```powershell
# Windows build (hidden console window)
.\build.ps1

# Run
.\TinyProxyAPI.exe -config config.toml
```

> Or build manually: `go build -ldflags="-H windowsgui" -o TinyProxyAPI.exe .`
> On Windows, `go build -o name .` does not automatically append `.exe`, you must specify it explicitly.

### Request Examples

**List Models:**
```bash
curl http://localhost:3178/v1/models
```

**Responses (OpenAI Responses API):**
```bash
curl -X POST http://localhost:3178/v1/responses \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4o", "input": "Hello"}'
```

## Configuration Hot Reload

**Auto-applies on save** - The program monitors the config file via fsnotify, cross-platform (Windows / macOS / Linux).

To trigger manually (Linux/Mac only):

```bash
kill -HUP $(pidof TinyProxyAPI)
```

## System Tray (Windows)

After startup, an icon appears in the system tray. Right-click menu provides:

| Menu Item | Function |
|-----------|----------|
| Show Log | Check to show the log viewer window |
| Show log on startup | Check to auto-show log window on next startup |
| Open Config File | Opens `config.toml` using the associated system editor (fallback to Notepad) |
| Exit | Close the program |

- Left-click the tray icon to toggle log window visibility (replaces the old double-click behavior)
- Clicking the minimize button hides the window to the tray

### Screenshots

Right-click the system tray icon to show the log window, configure whether it is shown at startup, or exit the program:

![TinyProxyAPI tray menu](./docs/screenshots/tray-menu.png)

The log window displays server startup messages, configuration reloads, and runtime errors:

![TinyProxyAPI log window](./docs/screenshots/log-window.png)

### UI State

The "Show log on startup" setting is **not managed by `config.toml`**, but controlled via the tray menu and saved in `~/.tinyproxyapi/state.json` in the user's home directory, defaulting to show on startup.

`config.toml` is always read-only — the program never writes back to it, your comments and formatting are preserved.

### Regenerating the Icon

The tray icon is generated by `tools/genicon` and embedded into `rsrc.syso`. Single-file distribution, no external icon files needed:

```bash
go run ./tools/genicon
rsrc -manifest TinyProxyAPI.exe.manifest -ico icon.ico -o rsrc.syso
```

Both `icon.ico` and `rsrc.syso` need to be committed.

> **Note**: The GROUP_ICON resource ID assigned by `rsrc` depends on the number of image entries in the ICO file, not a fixed value. If you modify `tools/genicon/` and the entry count changes, you must update the `iconResourceID` constant in `tray/tray.go` — after building the exe, run `go run ./tools/resid TinyProxyAPI.exe` to read the actual ID and verify.

## Project Structure

```
TinyProxyAPI/
├── main.go            # Program entry (tray orchestration, state management)
├── config/
│   ├── loader.go      # Config loading (read-only, never writes back to config.toml)
│   └── template.go    # Default config template and auto-initialization
├── state/
│   └── state.go       # UI state read/write (state.json)
├── handler/
│   └── proxy.go       # Request handling
├── matcher/
│   └── matcher.go     # Route matching
├── signal/
│   ├── sighup.go      # SIGHUP signal handling (Linux/Mac)
│   └── watcher.go     # fsnotify file watching (cross-platform)
├── tray/
│   ├── tray.go        # System tray management
│   └── logwindow.go   # Log window
├── tools/
│   ├── genicon/       # Generate icon.ico
│   └── resid/         # Read icon resource ID from exe
├── config.toml        # Config file (contains secrets, not committed)
├── config.toml.example  # Config template
├── icon.ico           # Icon resource, embedded in rsrc.syso
├── rsrc.syso          # Embedded resources (icon + manifest), linked at build time
├── build.ps1          # Windows build script
└── TinyProxyAPI.exe.manifest # Common Controls manifest
```

## Dependencies

- Go 1.21+
- github.com/pelletier/go-toml/v2
- github.com/fsnotify/fsnotify
- github.com/lxn/walk (system tray)

## License

This project is licensed under the [MIT License](./LICENSE).

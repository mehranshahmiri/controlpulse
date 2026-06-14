# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ControlP v3 is a web-based server control panel written in Go. It manages Nginx sites, MySQL databases, SSL certificates, firewall rules (UFW), Fail2Ban, cron jobs, a file manager, a WebSocket terminal, and system monitoring — a self-hosted alternative to cPanel/Plesk.

## Build & Run

```bash
# Build
go build -o controlp main.go

# Run (listens on :8888)
./controlp

# Or run directly
go run main.go
```

Default credentials (override via env vars `CONTROLP_USER` / `CONTROLP_PASS`): `admin` / `password123`

No test suite exists currently.

## Architecture

**Entry point:** `main.go` — sets up Fiber, embeds `views/` and `public/` into the binary (`//go:embed`), defines all routes, and wires session/auth middleware.

**Internal packages** (`internal/`): each feature is an isolated package with no cross-package dependencies. All packages interact with the OS via `exec.Command()`.

| Package | Responsibility |
|---|---|
| `auth` | Session management, login/logout |
| `nginx` | Nginx site configs (sites-available/enabled) |
| `db` | MySQL database management, phpMyAdmin integration |
| `system` | CPU/RAM/disk stats, systemctl service control, updates |
| `cron` | Crontab parsing and job management |
| `firewall` | UFW rule management |
| `ssl` | Certbot certificate provisioning |
| `security` | Fail2Ban jail management |
| `terminal` | WebSocket PTY (bash via `creack/pty`) |
| `fs` | File browser, editor, upload/delete |
| `blueprints` | App installer (WordPress, Laravel) |
| `logs` | Log file reader |

**Frontend:** HTML templates in `views/` rendered by Fiber's template engine. `views/layouts/main.html` is the primary shell; `views/layouts/empty.html` is used for login and API responses. Partials in `views/partials/` support HTMX-style partial refreshes.

**Exception — Environment Manager:** The `/env` route (`views/env.html`) is a standalone SPA served by reading the embed FS directly (`embedDir.ReadFile()`), bypassing Fiber's template engine and layouts entirely. It is not listed in the package table above because it has no `internal/` package — it's a self-contained frontend page with no Go backend logic.

## macOS vs Linux

Every package detects the OS at init and switches to a mock/local mode on macOS:

```go
if runtime.GOOS == "darwin" {
    // use ./mock_nginx, ./www, ./mock_cron, etc.
} else {
    // use /etc/nginx, /var/www/html, etc.
}
```

This means local development on macOS works without real system daemons. Key macOS path mappings:
- `internal/nginx`: uses `./mock_nginx/sites-available` and `./mock_nginx/sites-enabled`
- `internal/fs`: uses `./mock_nginx/html` as the file manager root
- `internal/blueprints`: uses `./www` as the web root for app installs
- `internal/cron`: uses `./mock_cron` as the crontab file

## Key Patterns

- **OS commands:** All system interactions use `exec.Command()`. Privileged commands pipe a sudo password. Service name whitelisting in `internal/system/actions.go` prevents command injection.
- **Path safety:** `internal/fs` uses `getSafePath()` to prevent directory traversal.
- **Domain sanitization:** `internal/nginx` sanitizes domain names before writing config files.
- **Routes:** All routes except `/login`, `/logout` are protected by session middleware in `main.go`. API responses use the empty layout; UI pages use main layout.
- **Sessions:** 24-hour Fiber session store. Auth state stored in `sess.Get("authenticated")`.
- **WebSocket terminal:** `/ws/terminal` upgrades to WebSocket; `internal/terminal` allocates a PTY and bridges browser ↔ bash bidirectionally.

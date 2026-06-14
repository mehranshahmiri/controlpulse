<div align="center">
  <img src="public/controlpulse-logo.png" alt="ControlPulse" height="80">
  <h1>ControlPulse</h1>
  <p>A self-hosted server control panel built in Go.</p>
  <p>
    <a href="https://mehranshahmiri.com">Mehran Shahmiri</a> &nbsp;·&nbsp;
    <a href="https://openloop.in">OpenLoop</a>
  </p>
  <p>
    <img src="https://img.shields.io/badge/license-GPL--3.0-blue" alt="License">
    <img src="https://img.shields.io/badge/language-Go-00ADD8" alt="Go">
    <img src="https://img.shields.io/badge/framework-Fiber-00ACD7" alt="Fiber">
  </p>
</div>

---

> **Note:** This project was vibe-coded and may contain bugs. Use in production at your own risk. Contributions and bug reports are welcome.

ControlPulse is an open-source web-based control panel written in Go. It gives you a clean UI to manage Nginx sites, MySQL databases, SSL certificates, firewall rules, cron jobs, files, system logs, and a live terminal — all from a browser. A self-hosted alternative to cPanel and Plesk.

## Features

- **Nginx site management** — create, enable/disable, and delete virtual hosts
- **SSL certificates** — provision Let's Encrypt certs via Certbot with one click
- **MySQL databases** — create databases and users with generated passwords
- **phpMyAdmin** — install and launch phpMyAdmin from the panel
- **App blueprints** — one-click WordPress and Laravel deployment
- **File manager** — browse, edit, upload, and delete files with a Monaco editor
- **Environment manager** — manage `.env` files across projects in a visual UI
- **Firewall** — manage UFW rules (add/delete ports and protocols)
- **Fail2Ban** — view jails, see banned IPs, and unban with one click
- **Cron jobs** — add and remove crontab entries via the UI
- **System updates** — list and apply `apt` package upgrades
- **Service control** — restart Nginx, PHP-FPM, and MySQL from the settings page
- **Live terminal** — full in-browser PTY terminal over WebSocket
- **System dashboard** — real-time CPU, RAM, and disk usage

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go, [Fiber v2](https://gofiber.io) |
| System stats | [gopsutil](https://github.com/shirou/gopsutil) |
| Terminal | [creack/pty](https://github.com/creack/pty) + WebSocket |
| Database driver | [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) |
| Frontend | HTML templates, HTMX, Alpine.js, Tailwind CSS |
| Env Manager | React (via CDN, no build step) |
| Code editor | Monaco Editor |

## Installation

### One-line install (Ubuntu / Debian)

```bash
curl -sSL https://controlpulse.in/install.sh | sudo bash
```

The script installs dependencies (Nginx, MySQL, PHP 8.3, UFW, Fail2Ban, Certbot), downloads the ControlPulse binary, sets up a systemd service, opens port 8888 in UFW, and prints your auto-generated login credentials.

### Uninstall

```bash
curl -sSL https://controlpulse.in/uninstall.sh | sudo bash
```

### Manual install (build from source)

```bash
git clone https://github.com/mehranshahmiri/controlpulse.git
cd controlpulse
go build -o controlp main.go
./controlp
```

The panel starts on port **8888**. Open `http://your-server-ip:8888` in your browser.

### macOS (development)

```bash
go run main.go
```

On macOS, all system operations run in mock mode. No real Nginx, MySQL, or cron is touched. Mock files are created under `./mock_nginx/` and `./www/`.

### Requirements

- **Linux install:** Ubuntu 22.04+ or Debian 11+ (script handles dependencies automatically)
- **Manual build:** Go 1.21+
- **macOS dev:** Go 1.21+ (no other dependencies)

## Configuration

All configuration is done via environment variables. There are no config files.

| Variable | Default | Description |
|---|---|---|
| `CONTROLP_USER` | `admin` | Login username |
| `CONTROLP_PASS` | `password123` | Login password — **change this** |

> Set these in your systemd service file or shell before running. Never run in production with the default password.

## Architecture

```
controlpulse/
├── main.go                  # Fiber app, all routes
├── internal/
│   ├── auth/                # Session management, login/logout
│   ├── nginx/               # Nginx site configs (sites-available/enabled)
│   ├── db/                  # MySQL management, phpMyAdmin
│   ├── system/              # CPU/RAM/disk stats, service control, updates
│   ├── cron/                # Crontab parsing and job management
│   ├── firewall/            # UFW rule management
│   ├── ssl/                 # Certbot certificate provisioning
│   ├── security/            # Fail2Ban jail management
│   ├── terminal/            # WebSocket PTY (bash)
│   ├── fs/                  # File browser, editor, upload/delete
│   ├── blueprints/          # App installer (WordPress, Laravel)
│   └── logs/                # Log file reader
├── views/                   # HTML templates (Fiber template engine)
│   ├── layouts/             # main.html shell, empty.html for API/login
│   └── partials/            # HTMX partial fragments
└── public/                  # Static assets (logo, JS)
```

Each `internal/` package is isolated — no cross-package imports. All system interactions go through `exec.Command()`. On macOS, each package detects `runtime.GOOS == "darwin"` and switches to a local mock path instead of real system paths.

## Security Notes

- All routes except `/login` and `/logout` are session-protected
- Service names are whitelisted before being passed to `systemctl` (prevents command injection)
- File manager uses `getSafePath()` to block directory traversal
- Domain names are sanitized before being written to Nginx configs
- Sessions expire after 24 hours

## Contributing

Pull requests are welcome. For larger changes, open an issue first to discuss what you'd like to change.

```bash
# Fork the repo, then:
git clone https://github.com/your-username/controlpulse.git
cd controlpulse
go run main.go   # starts in macOS mock mode
```

## License

GNU General Public License v3.0 — see [GPL-3.0](https://www.gnu.org/licenses/gpl-3.0.html).

## Author

Made by [Mehran Shahmiri](https://mehranshahmiri.com), Powered by [OpenLoop](https://openloop.in)

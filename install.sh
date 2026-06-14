#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────────────────────────────────────
# ControlPulse — Installer
# https://github.com/mehranshahmiri/controlpulse
#
# Usage:
#   curl -sSL https://controlpulse.in/install.sh | sudo bash
# ──────────────────────────────────────────────────────────────────────────────

REPO="mehranshahmiri/controlpulse"
BINARY_NAME="controlp"
INSTALL_DIR="/opt/controlp"
SERVICE_FILE="/etc/systemd/system/controlp.service"
ENV_FILE="/etc/controlp/controlp.env"
PORT=8888

# ── Colors ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'
ORANGE='\033[0;33m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
RESET='\033[0m'

info()    { echo -e "${CYAN}${BOLD}→${RESET} $*"; }
success() { echo -e "${GREEN}${BOLD}✓${RESET} $*"; }
warn()    { echo -e "${ORANGE}${BOLD}!${RESET} $*"; }
fatal()   { echo -e "${RED}${BOLD}✗ ERROR:${RESET} $*" >&2; exit 1; }

banner() {
    echo ""
    echo -e "${ORANGE}${BOLD}  ControlPulse — Self-Hosted Server Control Panel${RESET}"
    echo -e "${DIM}  https://github.com/mehranshahmiri/controlpulse${RESET}"
    echo ""
}

generate_password() {
    tr -dc 'A-Za-z0-9!@#%^&*' </dev/urandom | head -c 20
}

check_root() {
    [[ $EUID -eq 0 ]] || fatal "Run as root: sudo bash"
}

check_os() {
    [[ -f /etc/debian_version ]] || fatal "Only Ubuntu/Debian is supported."
    OS_NAME=$(grep '^PRETTY_NAME' /etc/os-release | cut -d= -f2 | tr -d '"')
    info "OS: ${OS_NAME}"
}

check_arch() {
    case "$(uname -m)" in
        x86_64)  ASSET="controlp-linux-amd64" ;;
        aarch64) ASSET="controlp-linux-arm64" ;;
        *)        fatal "Unsupported architecture: $(uname -m)" ;;
    esac
}

install_deps() {
    info "Updating package index..."
    apt-get update -qq

    PACKAGES=()
    for pkg in nginx mysql-server php8.3-fpm php8.3-mysql ufw fail2ban certbot python3-certbot-nginx curl wget; do
        dpkg -s "$pkg" &>/dev/null || PACKAGES+=("$pkg")
    done

    if [[ ${#PACKAGES[@]} -gt 0 ]]; then
        info "Installing: ${PACKAGES[*]}"
        DEBIAN_FRONTEND=noninteractive apt-get install -y -qq "${PACKAGES[@]}"
        success "Dependencies installed."
    else
        success "All dependencies already present."
    fi

    for svc in nginx mysql php8.3-fpm ufw fail2ban; do
        systemctl enable --now "$svc" &>/dev/null || true
    done
}

download_binary() {
    info "Downloading ControlPulse binary..."
    mkdir -p "$INSTALL_DIR"

    URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"
    if ! curl -fsSL "$URL" -o "${INSTALL_DIR}/${BINARY_NAME}"; then
        fatal "Could not download binary.\nCheck releases at: https://github.com/${REPO}/releases"
    fi
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    success "Binary installed to ${INSTALL_DIR}/${BINARY_NAME}"
}

write_env() {
    mkdir -p /etc/controlp

    if [[ -f "$ENV_FILE" ]]; then
        warn "Existing config found at ${ENV_FILE} — keeping credentials."
        return
    fi

    ADMIN_PASS=$(generate_password)
    cat > "$ENV_FILE" <<EOF
CONTROLP_USER=admin
CONTROLP_PASS=${ADMIN_PASS}
EOF
    chmod 600 "$ENV_FILE"
    export _CONTROLP_PASS="$ADMIN_PASS"
    success "Credentials saved to ${ENV_FILE}"
}

write_service() {
    cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=ControlPulse Server Panel
Documentation=https://github.com/mehranshahmiri/controlpulse
After=network.target mysql.service nginx.service

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/${BINARY_NAME}
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=${ENV_FILE}
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=controlp

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable controlp
    systemctl restart controlp
    success "systemd service enabled and started."
}

configure_firewall() {
    if systemctl is-active --quiet ufw; then
        ufw status | grep -q "${PORT}" || ufw allow "${PORT}/tcp" comment "ControlPulse Panel" >/dev/null
        success "UFW: port ${PORT} open."
    else
        warn "UFW not active — ensure port ${PORT} is reachable."
    fi
}

# ── Main ──────────────────────────────────────────────────────────────────────
banner
info "Installing ControlPulse v1.0.0..."
echo ""

check_root
check_os
check_arch
install_deps
download_binary
write_env
write_service
configure_firewall

FINAL_PASS="${_CONTROLP_PASS:-$(grep CONTROLP_PASS "$ENV_FILE" | cut -d= -f2)}"
SERVER_IP=$(curl -fsSL https://api.ipify.org 2>/dev/null || hostname -I | awk '{print $1}')

echo ""
echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
echo -e "${GREEN}${BOLD}  ControlPulse installed successfully!${RESET}"
echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
echo ""
echo -e "  ${BOLD}Panel:${RESET}     http://${SERVER_IP}:${PORT}"
echo -e "  ${BOLD}Username:${RESET}  admin"
echo -e "  ${BOLD}Password:${RESET}  ${ORANGE}${BOLD}${FINAL_PASS}${RESET}"
echo ""
echo -e "  ${DIM}Config:  ${ENV_FILE}${RESET}"
echo -e "  ${DIM}Logs:    journalctl -u controlp -f${RESET}"
echo ""
echo -e "  ${RED}${BOLD}Change your password after first login!${RESET}"
echo ""
echo -e "  To uninstall:"
echo -e "  ${DIM}curl -sSL https://controlpulse.in/uninstall.sh | sudo bash${RESET}"
echo ""

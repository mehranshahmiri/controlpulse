#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────────────────────────────────────
# ControlPulse — Uninstaller
# https://github.com/mehranshahmiri/controlpulse
#
# Usage:
#   curl -sSL https://controlpulse.in/uninstall.sh | sudo bash
# ──────────────────────────────────────────────────────────────────────────────

INSTALL_DIR="/opt/controlp"
SERVICE_FILE="/etc/systemd/system/controlp.service"
ENV_DIR="/etc/controlp"
PORT=8888

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

[[ $EUID -eq 0 ]] || fatal "Run as root: sudo bash"

echo ""
echo -e "${RED}${BOLD}  ControlPulse — Uninstaller${RESET}"
echo -e "${DIM}  https://github.com/mehranshahmiri/controlpulse${RESET}"
echo ""

# Stop and disable service
if systemctl is-active --quiet controlp 2>/dev/null; then
    systemctl stop controlp
    success "Service stopped."
fi
if systemctl is-enabled --quiet controlp 2>/dev/null; then
    systemctl disable controlp
    success "Service disabled."
fi

# Remove service file
if [[ -f "$SERVICE_FILE" ]]; then
    rm -f "$SERVICE_FILE"
    systemctl daemon-reload
    success "Service file removed."
fi

# Remove binary and install directory
if [[ -d "$INSTALL_DIR" ]]; then
    rm -rf "$INSTALL_DIR"
    success "Removed ${INSTALL_DIR}"
fi

# Close firewall port
if systemctl is-active --quiet ufw 2>/dev/null; then
    ufw delete allow "${PORT}/tcp" >/dev/null 2>&1 || true
    success "UFW: port ${PORT} closed."
fi

# Ask about credentials/config
echo ""
read -rp "  Remove credentials and config (${ENV_DIR})? [y/N] " REMOVE_ENV
if [[ "${REMOVE_ENV,,}" == "y" ]]; then
    rm -rf "$ENV_DIR"
    success "Config removed."
else
    info "Config kept at ${ENV_DIR}"
fi

echo ""
success "ControlPulse has been uninstalled."
echo ""

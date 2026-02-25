#!/bin/bash
set -e

PKG_DIR="${1:-cosmic-packages}"

check_apt() {
    if ! command -v apt-get > /dev/null 2>&1 || ! command -v dpkg > /dev/null 2>&1; then
        echo "ERROR: This tool requires a Debian-style system with APT and dpkg." >&2
        exit 1
    fi
}

detect_distro() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        echo "$ID $VERSION_CODENAME"
    else
        echo "unknown unknown"
    fi
}

install_runtime_deps() {
    echo "Installing runtime dependencies..."
    apt-get update -qq
    apt-get install -y --no-install-recommends \
        accountsservice dbus iso-codes libdbus-1-3 libdisplay-info1 \
        libflatpak0 libfontconfig1 libgbm1 libgstreamer-plugins-base1.0-0 \
        libgstreamer1.0-0 libinput10 libpam0g libpipewire-0.3-0 \
        libpixman-1-0 libpulse0 libseat1 libssl3 libwayland-client0 \
        libwayland-server0 libxkbcommon0 network-manager udev
}

check_apt
read -r DISTRO_ID CODENAME <<< "$(detect_distro)"

if [ ! -d "$PKG_DIR" ]; then
    echo "ERROR: Package directory '$PKG_DIR' does not exist." >&2
    exit 1
fi

DEBS=( "$PKG_DIR"/*.deb )
if [ "${#DEBS[@]}" -eq 0 ] || [ ! -f "${DEBS[0]}" ]; then
    echo "ERROR: No .deb packages found in '$PKG_DIR'." >&2
    exit 1
fi

install_runtime_deps

echo "Installing COSMIC packages from '$PKG_DIR'..."
dpkg -i "${DEBS[@]}" || apt-get install -f -y

if command -v systemctl > /dev/null 2>&1; then
    systemctl daemon-reload || true
fi

echo "Installation complete."
echo "Please log out and select the COSMIC session from your display manager."

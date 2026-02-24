#!/bin/bash
set -e

PKG_DIR="${1:-cosmic-packages}"

check_apt() {
    if ! command -v apt-get > /dev/null 2>&1; then
        echo "ERROR: This tool requires an APT-based distribution (Debian or Ubuntu)." >&2
        exit 1
    fi
}

detect_distro() {
    . /etc/os-release
    echo "$ID $VERSION_CODENAME"
}

check_min_version() {
    local id="$1"
    local codename="$2"

    declare -A debian_supported=( [bookworm]=1 [trixie]=1 [forky]=1 [sid]=1 [unstable]=1 [testing]=1 )
    declare -A ubuntu_supported=( [jammy]=1 [noble]=1 [resolute]=1 [devel]=1 )

    if [ "$id" = "debian" ]; then
        if [ -z "${debian_supported[$codename]}" ]; then
            echo "ERROR: Debian release '$codename' is not supported. The minimum supported release is bookworm (12)." >&2
            exit 1
        fi
    elif [ "$id" = "ubuntu" ]; then
        if [ -z "${ubuntu_supported[$codename]}" ]; then
            echo "ERROR: Ubuntu release '$codename' is not supported. Only LTS and devel releases are supported (e.g., jammy, noble, or resolute)." >&2
            exit 1
        fi
    else
        echo "ERROR: Distribution '$id' is not supported. Only Debian and Ubuntu are supported." >&2
        exit 1
    fi
}

install_runtime_deps() {
    echo "Installing runtime dependencies..."
    apt-get update -qq
    apt-get install -y --no-install-recommends \
        accountsservice \
        dbus \
        iso-codes \
        libdbus-1-3 \
        libdisplay-info1 \
        libflatpak0 \
        libfontconfig1 \
        libgbm1 \
        libgstreamer-plugins-base1.0-0 \
        libgstreamer1.0-0 \
        libinput10 \
        libpam0g \
        libpipewire-0.3-0 \
        libpixman-1-0 \
        libpulse0 \
        libseat1 \
        libssl3 \
        libwayland-client0 \
        libwayland-server0 \
        libxkbcommon0 \
        network-manager \
        udev
}

check_apt

read -r DISTRO_ID CODENAME <<< "$(detect_distro)"
check_min_version "$DISTRO_ID" "$CODENAME"

if [ ! -d "$PKG_DIR" ]; then
    echo "ERROR: The specified package directory '$PKG_DIR' does not exist." >&2
    exit 1
fi

DEBS=( "$PKG_DIR"/*.deb )
if [ ${#DEBS[@]} -eq 0 ] || [ ! -f "${DEBS[0]}" ]; then
    echo "ERROR: No .deb packages were found in '$PKG_DIR'." >&2
    exit 1
fi

install_runtime_deps

echo "Installing COSMIC packages from '$PKG_DIR'..."
dpkg -i "${DEBS[@]}" || apt-get install -f -y

if command -v systemctl > /dev/null 2>&1; then
    systemctl daemon-reload || true
fi

echo "Installation complete."
echo "Please log out and select the COSMIC session from your display manager to commence use."

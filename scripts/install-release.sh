#!/bin/bash
set -e

REPO_OWNER="jimed-rand"
REPO_NAME="cosmic-deb"
RELEASE_TAG="${1:-latest}"
TMP_DIR="$(mktemp -d)"

cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

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

ensure_tool() {
    if ! command -v "$1" > /dev/null 2>&1; then
        echo "Required tool '$1' not found; installing..."
        apt-get install -y "$1"
    fi
}

install_runtime_deps() {
    echo "Installing runtime dependencies..."
    apt-get update -qq
    apt-get install -y --no-install-recommends \
        accountsservice \
        curl \
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
ensure_tool curl

if [ "$RELEASE_TAG" = "latest" ]; then
    API_URL="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
else
    API_URL="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/tags/${RELEASE_TAG}"
fi

echo "Fetching release metadata for tag: $RELEASE_TAG"
ASSETS_JSON=$(curl -fsSL "$API_URL")
ARCH=$(dpkg --print-architecture)

DOWNLOAD_URLS=$(echo "$ASSETS_JSON" | grep -oP '"browser_download_url":\s*"\K[^"]+' | grep "_${ARCH}\.deb" || true)

if [ -z "$DOWNLOAD_URLS" ]; then
    echo "ERROR: No .deb packages were found for architecture '$ARCH' in release '$RELEASE_TAG'." >&2
    exit 1
fi

echo "Downloading packages to temporary directory..."
while IFS= read -r url; do
    filename=$(basename "$url")
    echo "  Downloading: $filename"
    curl -fsSL -o "$TMP_DIR/$filename" "$url"
done <<< "$DOWNLOAD_URLS"

install_runtime_deps

echo "Installing COSMIC packages..."
dpkg -i "$TMP_DIR"/*.deb || apt-get install -f -y

if command -v systemctl > /dev/null 2>&1; then
    systemctl daemon-reload || true
fi

echo "Installation complete."
echo "Please log out and select the COSMIC session from your display manager to commence use."

#!/bin/bash
set -e

COSMIC_PACKAGES=(
    cosmic-app-library cosmic-applets cosmic-bg cosmic-comp cosmic-desktop
    cosmic-edit cosmic-files cosmic-greeter cosmic-icons cosmic-idle
    cosmic-initial-setup cosmic-launcher cosmic-notifications cosmic-osd
    cosmic-panel cosmic-player cosmic-randr cosmic-screenshot cosmic-session
    cosmic-settings cosmic-settings-daemon cosmic-store cosmic-term
    cosmic-wallpapers cosmic-workspaces pop-launcher xdg-desktop-portal-cosmic
)

check_apt() {
    if ! command -v apt-get > /dev/null 2>&1 || ! command -v dpkg > /dev/null 2>&1; then
        echo "ERROR: This tool requires a Debian-style system with APT and dpkg." >&2
        exit 1
    fi
}

check_apt

echo "The following COSMIC packages are scheduled for removal:"
for pkg in "${COSMIC_PACKAGES[@]}"; do
    if dpkg -l "$pkg" > /dev/null 2>&1; then
        echo "  $pkg"
    fi
done

printf "\nProceed with uninstallation? [y/N] "
read -r CONFIRM
if [ "$CONFIRM" != "y" ] && [ "$CONFIRM" != "Y" ]; then
    echo "Uninstallation cancelled."
    exit 0
fi

INSTALLED=()
for pkg in "${COSMIC_PACKAGES[@]}"; do
    if dpkg -l "$pkg" > /dev/null 2>&1; then
        INSTALLED+=("$pkg")
    fi
done

if [ "${#INSTALLED[@]}" -eq 0 ]; then
    echo "No installed COSMIC packages found."
    exit 0
fi

echo "Removing installed COSMIC packages..."
apt-get remove -y "${INSTALLED[@]}" || true
apt-get autoremove -y || true

if command -v systemctl > /dev/null 2>&1; then
    systemctl daemon-reload || true
fi

echo "COSMIC Desktop Environment successfully removed."
echo "A display manager restart or reboot may be required."

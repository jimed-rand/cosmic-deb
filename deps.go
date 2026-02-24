package main

func hasDisplayInfoDev(distroID, codename string) bool {
	switch distroID {
	case "debian":
		switch codename {
		case "trixie", "forky", "sid", "unstable", "testing":
			return true
		}
	case "ubuntu":
		switch codename {
		case "noble", "resolute", "devel":
			return true
		}
	}
	return false
}

func hasRustAll(distroID, codename string) bool {
	switch distroID {
	case "debian":
		return true
	case "ubuntu":
		switch codename {
		case "jammy", "noble", "resolute", "devel":
			return true
		}
	}
	return false
}

func hasDhCargo(distroID, codename string) bool {
	switch distroID {
	case "debian":
		return true
	case "ubuntu":
		switch codename {
		case "jammy", "noble", "resolute", "devel":
			return true
		}
	}
	return false
}

func hasJustInApt(distroID, codename string) bool {
	switch distroID {
	case "debian":
		switch codename {
		case "trixie", "forky", "sid", "unstable", "testing":
			return true
		}
	}
	return false
}

func resolveGlobalBuildDeps(distroID, codename string) []string {
	deps := []string{
		"build-essential",
		"cargo",
		"clang",
		"cmake",
		"curl",
		"debhelper",
		"desktop-file-utils",
		"devscripts",
		"git",
		"imagemagick",
		"intltool",
		"iso-codes",
		"libclang-dev",
		"libdbus-1-dev",
		"libegl-dev",
		"libegl1-mesa-dev",
		"libexpat1-dev",
		"libflatpak-dev",
		"libfontconfig-dev",
		"libfreetype-dev",
		"libgbm-dev",
		"libglib2.0-dev",
		"libgstreamer-plugins-base1.0-dev",
		"libgstreamer1.0-dev",
		"libinput-dev",
		"libnm-dev",
		"libpam0g-dev",
		"libpipewire-0.3-dev",
		"libpixman-1-dev",
		"libpulse-dev",
		"libseat-dev",
		"libssl-dev",
		"libsystemd-dev",
		"libudev-dev",
		"libwayland-dev",
		"libxcb-render0-dev",
		"libxcb-shape0-dev",
		"libxcb-xfixes0-dev",
		"libxcb1-dev",
		"libxkbcommon-dev",
		"lld",
		"mold",
		"nasm",
		"pkg-config",
		"rustc",
		"fakeroot",
		"ninja-build",
		"meson",
		"sassc",
		"quilt",
		"libfile-fcntllock-perl",
		"dh-make",
		"dpkg-dev",
		"libglib2.0-dev-bin",
		"libwayland-bin",
		"libxml2-utils",
		"libglib2.0-bin",
		"gettext",
		"itstool",
		"wayland-protocols",
		"libgdk-pixbuf-2.0-dev",
		"dh-exec",
	}

	if hasDisplayInfoDev(distroID, codename) {
		deps = append(deps, "libdisplay-info-dev")
	}
	if hasRustAll(distroID, codename) {
		deps = append(deps, "rust-all")
	}
	if hasDhCargo(distroID, codename) {
		deps = append(deps, "dh-cargo")
	}
	if hasJustInApt(distroID, codename) {
		deps = append(deps, "just")
	}

	return deps
}

func resolvePerComponentBuildDeps(distroID, codename string) map[string][]string {
	displayInfo := hasDisplayInfoDev(distroID, codename)

	cosmicComp := []string{
		"cargo", "cmake", "debhelper", "libegl1-mesa-dev",
		"libfontconfig-dev", "libgbm-dev", "libinput-dev",
		"libpixman-1-dev", "libseat-dev", "libsystemd-dev",
		"libudev-dev", "libwayland-dev", "libxcb1-dev",
		"libxkbcommon-dev", "rustc",
	}
	if displayInfo {
		cosmicComp = append(cosmicComp, "libdisplay-info-dev")
	}

	cosmicSettings := []string{
		"debhelper", "cmake", "just", "libclang-dev",
		"libexpat1-dev", "libfontconfig-dev", "libfreetype-dev",
		"libinput-dev", "libpipewire-0.3-dev", "libudev-dev",
		"libwayland-dev", "libxkbcommon-dev", "mold", "pkg-config",
	}
	if displayInfo {
		cosmicSettings = append(cosmicSettings, "libdisplay-info-dev")
	}

	return map[string][]string{
		"cosmic-comp":     cosmicComp,
		"cosmic-settings": cosmicSettings,
		"cosmic-session": {
			"debhelper", "cargo", "just",
		},
		"cosmic-files": {
			"debhelper", "git", "just", "libclang-dev",
			"libglib2.0-dev", "libxkbcommon-dev", "pkg-config",
		},
		"cosmic-applets": {
			"debhelper", "rustc", "cargo", "libclang-dev",
			"libdbus-1-dev", "libegl-dev", "libpulse-dev",
			"libpipewire-0.3-dev", "libudev-dev", "libxkbcommon-dev",
			"libwayland-dev", "libinput-dev", "just", "pkg-config",
		},
		"cosmic-edit": {
			"debhelper", "git", "just", "pkg-config",
			"libglib2.0-dev", "libxkbcommon-dev",
		},
		"cosmic-store": {
			"debhelper", "git", "just", "libflatpak-dev",
			"libssl-dev", "libxkbcommon-dev", "pkg-config",
		},
		"cosmic-bg": {
			"debhelper", "just", "libwayland-dev",
			"libxkbcommon-dev", "mold", "nasm", "pkg-config",
		},
		"cosmic-greeter": {
			"debhelper", "git", "just", "libclang-dev",
			"libinput-dev", "libpam0g-dev", "libwayland-dev",
			"libxkbcommon-dev", "pkg-config",
		},
		"cosmic-settings-daemon": {
			"debhelper", "cargo", "libudev-dev", "libinput-dev",
			"libssl-dev", "libxkbcommon-dev", "pulseaudio-utils",
			"pkg-config",
		},
		"xdg-desktop-portal-cosmic": {
			"debhelper", "cargo", "libclang-dev", "libglib2.0-dev",
			"libegl-dev", "libgbm-dev", "libpipewire-0.3-dev",
			"libwayland-dev", "libxkbcommon-dev", "pkg-config",
		},
		"cosmic-app-library": {
			"debhelper", "just", "pkg-config",
			"libxkbcommon-dev", "libwayland-dev",
		},
		"cosmic-icons": {
			"debhelper", "just",
		},
		"cosmic-panel": {
			"debhelper", "just", "cargo", "libwayland-dev",
			"libxkbcommon-dev", "pkg-config", "desktop-file-utils",
		},
		"cosmic-notifications": {
			"debhelper", "rustc", "cargo", "just", "intltool",
			"libxkbcommon-dev", "libwayland-dev", "pkg-config",
		},
		"cosmic-osd": {
			"debhelper", "cargo", "just", "libclang-dev",
			"libinput-dev", "libpulse-dev", "libudev-dev",
			"libpipewire-0.3-dev", "libxkbcommon-dev",
			"libwayland-dev", "pkg-config",
		},
		"cosmic-launcher": {
			"debhelper", "rustc", "cargo", "just", "intltool",
			"libxkbcommon-dev", "libwayland-dev", "pkg-config",
		},
		"cosmic-screenshot": {
			"debhelper", "just",
		},
		"cosmic-idle": {
			"debhelper", "cargo", "just", "libxkbcommon-dev",
			"libwayland-dev", "pkg-config",
		},
		"cosmic-randr": {
			"cargo", "debhelper", "just", "libwayland-dev",
			"pkg-config", "rustc",
		},
		"cosmic-wallpapers": {
			"debhelper", "imagemagick",
		},
		"cosmic-workspaces": {
			"debhelper", "cargo", "libegl1-mesa-dev", "libgbm-dev",
			"libinput-dev", "libudev-dev", "libxkbcommon-dev",
			"libwayland-dev", "pkg-config",
		},
		"cosmic-initial-setup": {
			"debhelper", "git", "just", "libflatpak-dev",
			"libinput-dev", "libssl-dev", "libudev-dev",
			"libxkbcommon-dev", "pkg-config",
		},
		"pop-launcher": {
			"cargo", "debhelper", "just", "pkg-config",
			"rustc", "libxkbcommon-dev", "libegl-dev",
		},
		"cosmic-term": {
			"debhelper", "git", "just", "pkg-config",
			"libxkbcommon-dev",
		},
		"cosmic-player": {
			"clang", "debhelper", "just",
			"libgstreamer1.0-dev", "libgstreamer-plugins-base1.0-dev",
			"libxkbcommon-dev", "pkg-config",
		},
	}
}

package debian

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var RuntimeDeps = map[string][]string{
	"cosmic-comp": {"libegl1", "libwayland-server0"},
	"cosmic-session": {
		"cosmic-app-library", "cosmic-applets", "cosmic-bg", "cosmic-comp", "cosmic-files",
		"cosmic-greeter", "cosmic-icons", "cosmic-idle", "cosmic-launcher", "cosmic-notifications",
		"cosmic-osd", "cosmic-panel", "cosmic-randr", "cosmic-screenshot", "cosmic-settings",
		"cosmic-settings-daemon", "cosmic-workspaces", "fonts-open-sans", "gnome-keyring",
		"libsecret-1-0", "switcheroo-control", "xdg-desktop-portal-cosmic", "xwayland",
	},
	"cosmic-files":           {"xdg-utils"},
	"cosmic-applets":         {"cosmic-icons"},
	"cosmic-greeter":         {"adduser", "cosmic-comp", "cosmic-greeter-daemon", "cosmic-randr", "dbus"},
	"cosmic-settings":        {"accountsservice", "cosmic-randr", "gettext", "iso-codes", "network-manager-gnome", "network-manager-openvpn", "network-manager-openvpn-gnome", "xkb-data"},
	"cosmic-settings-daemon": {"acpid"},
	"cosmic-osd":             {"pulseaudio-utils"},
	"cosmic-launcher":        {"pop-launcher"},
	"cosmic-icons":           {"pop-icon-theme"},
	"cosmic-initial-setup":   {"cosmic-icons"},
	"cosmic-store":           {"cosmic-icons"},
	"cosmic-player":          {"gstreamer1.0-plugins-base", "gstreamer1.0-plugins-good"},
	"pop-launcher":           {"qalc", "fd-find"},
}

var Recommends = map[string][]string{
	"cosmic-comp":            {"cosmic-session", "libgl1-mesa-dri"},
	"cosmic-session":         {"cosmic-edit", "cosmic-player", "cosmic-store", "cosmic-term", "cosmic-wallpapers", "orca", "system-config-printer"},
	"cosmic-applets":         {"pipewire-pulse"},
	"cosmic-settings":        {"adw-gtk3"},
	"cosmic-settings-daemon": {"playerctl"},
	"cosmic-greeter":         {"xinit"},
}

var adminSection = map[string]bool{
	"cosmic-session": true, "cosmic-files": true, "cosmic-applets": true, "cosmic-edit": true,
	"cosmic-store": true, "cosmic-bg": true, "cosmic-greeter": true, "cosmic-icons": true,
	"cosmic-osd": true, "cosmic-notifications": true, "cosmic-panel": true, "cosmic-launcher": true,
	"cosmic-screenshot": true, "cosmic-idle": true, "cosmic-workspaces": true, "cosmic-initial-setup": true,
	"cosmic-term": true, "cosmic-player": true, "cosmic-app-library": true,
	"cosmic-settings-daemon": true, "xdg-desktop-portal-cosmic": true, "pop-launcher": true,
}

var utilsSection = map[string]bool{
	"cosmic-settings": true, "cosmic-randr": true, "pop-launcher": true,
}

func sectionFor(pkgName string) string {
	if utilsSection[pkgName] {
		return "utils"
	}
	if adminSection[pkgName] {
		return "admin"
	}
	return "x11"
}

func archString() string {
	if runtime.GOARCH == "arm64" {
		return "arm64"
	}
	return "amd64"
}

func runDpkg(args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func StagingHasContent(stageDir string) bool {
	entries, err := os.ReadDir(stageDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Name() != "DEBIAN" {
			return true
		}
	}
	return false
}

func fileVersion(version, distroCodename string) string {
	if distroCodename != "" {
		return version + "~" + distroCodename
	}
	return version
}

func BuildPackage(stageDir, outDir, pkgName, version, distroCodename, maintainerName, maintainerEmail string) error {
	debianDir := filepath.Join(stageDir, "DEBIAN")
	if err := os.MkdirAll(debianDir, 0755); err != nil {
		return err
	}
	arch := archString()
	fv := fileVersion(version, distroCodename)

	depEntries := []string{"${shlibs:Depends}"}
	if deps, ok := RuntimeDeps[pkgName]; ok {
		depEntries = append(depEntries, deps...)
	}

	control := fmt.Sprintf("Package: %s\nVersion: %s\nSection: %s\nPriority: optional\nArchitecture: %s\nDepends: %s\n",
		pkgName, fv, sectionFor(pkgName), arch, strings.Join(depEntries, ", "))

	if recs, ok := Recommends[pkgName]; ok && len(recs) > 0 {
		control += fmt.Sprintf("Recommends: %s\n", strings.Join(recs, ", "))
	}

	control += fmt.Sprintf("Maintainer: %s <%s>\nDescription: COSMIC Desktop Environment component — %s\n Built from upstream source via the cosmic-deb build tool.\n",
		maintainerName, maintainerEmail, pkgName)

	if err := os.WriteFile(filepath.Join(debianDir, "control"), []byte(control), 0644); err != nil {
		return err
	}

	pkgFile := filepath.Join(outDir, fmt.Sprintf("%s_%s_%s.deb", pkgName, fv, arch))
	return runDpkg("fakeroot", "dpkg-deb", "--build", stageDir, pkgFile)
}

func BuildMetaPackage(workDir, outDir, version, distroCodename, maintainerName, maintainerEmail string, builtRepos []string) error {
	const metaPkg = "cosmic-desktop"
	arch := archString()
	fv := fileVersion(version, distroCodename)
	stageDir := filepath.Join(workDir, metaPkg+"-stage")
	if err := os.MkdirAll(filepath.Join(stageDir, "DEBIAN"), 0755); err != nil {
		return err
	}

	control := fmt.Sprintf("Package: %s\nVersion: %s\nSection: x11\nPriority: optional\nArchitecture: %s\nDepends: %s\nMaintainer: %s <%s>\nDescription: COSMIC Desktop Environment meta package\n This meta package installs the complete COSMIC Desktop Environment\n by declaring dependencies on all COSMIC component packages built\n by the cosmic-deb build tool.\n",
		metaPkg, fv, arch, strings.Join(builtRepos, ", "), maintainerName, maintainerEmail)

	if err := os.WriteFile(filepath.Join(stageDir, "DEBIAN", "control"), []byte(control), 0644); err != nil {
		return err
	}
	pkgFile := filepath.Join(outDir, fmt.Sprintf("%s_%s_%s.deb", metaPkg, fv, arch))
	return runDpkg("fakeroot", "dpkg-deb", "--build", stageDir, pkgFile)
}

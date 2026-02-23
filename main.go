package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/jimed-rand/cosmic-deb/tui"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	outputPkgDir     = "cosmic-packages"
	metaPkgName      = "cosmic-desktop"
	defaultReposFile = "built-in"
	rustLinkerFlags  = "-C link-arg=-fuse-ld=lld"
)

type RepoEntry struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Tag    string `json:"tag"`
	Branch string `json:"branch,omitempty"`
}

type ReposConfig struct {
	GeneratedAt string      `json:"generated_at"`
	EpochLatest string      `json:"epoch_latest"`
	Repos       []RepoEntry `json:"repos"`
}

var globalBuildDeps = []string{
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
	"libdisplay-info-dev",
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
	"libpam-dev",
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
	"dh-cargo",
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
	"libgdk-pixbuf2.0-dev",
	"dh-exec",
	"just",
	"rust-all",
}

var perComponentBuildDeps = map[string][]string{
	"cosmic-comp": {
		"cargo", "cmake", "debhelper", "libegl1-mesa-dev",
		"libfontconfig-dev", "libgbm-dev", "libinput-dev",
		"libpixman-1-dev", "libseat-dev", "libsystemd-dev",
		"libudev-dev", "libwayland-dev", "libxcb1-dev",
		"libxkbcommon-dev", "libdisplay-info-dev", "rustc",
	},
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
		"libinput-dev", "libpam-dev", "libwayland-dev",
		"libxkbcommon-dev", "pkg-config",
	},
	"cosmic-settings": {
		"debhelper", "cmake", "just", "libclang-dev",
		"libexpat1-dev", "libfontconfig-dev", "libfreetype-dev",
		"libinput-dev", "libpipewire-0.3-dev", "libudev-dev",
		"libwayland-dev", "libxkbcommon-dev", "mold", "pkg-config",
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

var perComponentRuntimeDeps = map[string][]string{
	"cosmic-comp": {
		"libegl1", "libwayland-server0",
	},
	"cosmic-session": {
		"cosmic-app-library", "cosmic-applets", "cosmic-bg",
		"cosmic-comp", "cosmic-files", "cosmic-greeter",
		"cosmic-icons", "cosmic-idle", "cosmic-launcher",
		"cosmic-notifications", "cosmic-osd", "cosmic-panel",
		"cosmic-randr", "cosmic-screenshot", "cosmic-settings",
		"cosmic-settings-daemon", "cosmic-workspaces",
		"fonts-open-sans", "gnome-keyring", "libsecret-1-0",
		"switcheroo-control", "xdg-desktop-portal-cosmic", "xwayland",
	},
	"cosmic-files": {
		"xdg-utils",
	},
	"cosmic-applets": {
		"cosmic-icons",
	},
	"cosmic-greeter": {
		"adduser", "cosmic-comp", "cosmic-greeter-daemon",
		"cosmic-randr", "dbus",
	},
	"cosmic-settings": {
		"accountsservice", "cosmic-randr", "gettext",
		"iso-codes", "network-manager-gnome",
		"network-manager-openvpn", "network-manager-openvpn-gnome",
		"xkb-data",
	},
	"cosmic-settings-daemon": {
		"acpid",
	},
	"cosmic-osd": {
		"pulseaudio-utils",
	},
	"cosmic-launcher": {
		"pop-launcher",
	},
	"cosmic-icons": {
		"pop-icon-theme",
	},
	"cosmic-initial-setup": {
		"cosmic-icons",
	},
	"cosmic-store": {
		"cosmic-icons",
	},
	"cosmic-player": {
		"gstreamer1.0-plugins-base", "gstreamer1.0-plugins-good",
	},
	"pop-launcher": {
		"qalc", "fd-find",
	},
}

var perComponentRecommends = map[string][]string{
	"cosmic-comp": {
		"cosmic-session", "libgl1-mesa-dri",
	},
	"cosmic-session": {
		"cosmic-edit", "cosmic-player", "cosmic-store",
		"cosmic-term", "cosmic-wallpapers", "orca",
		"system-config-printer",
	},
	"cosmic-applets": {
		"pipewire-pulse",
	},
	"cosmic-settings": {
		"adw-gtk3",
	},
	"cosmic-settings-daemon": {
		"playerctl",
	},
	"cosmic-greeter": {
		"xinit",
	},
}

type Config struct {
	globalTag       string
	reposFile       string
	workDir         string
	outDir          string
	jobs            int
	skipDeps        bool
	only            string
	updateRepos     bool
	genConfig       bool
	devFinder       bool
	useBranch       bool
	maintainerName  string
	maintainerEmail string
}

var tuiProg *tea.Program

func log(format string, args ...any) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	if tuiProg != nil {
		tuiProg.Send(tui.LogMsg(msg))
	} else {
		fmt.Printf("[%s] %s\n", ts, msg)
	}
}

func die(format string, args ...any) {
	if tuiProg != nil {
		tuiProg.Send(tui.DoneMsg{Err: fmt.Errorf(format, args...)})
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", fmt.Sprintf(format, args...))
	os.Exit(1)
}

func run(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	if tuiProg == nil {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

func runEnv(dir string, extraEnv []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)
	if tuiProg == nil {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

func checkAptBased() {
	if _, err := exec.LookPath("apt-get"); err != nil {
		die("This tool requires an APT-based distribution (Debian or Ubuntu)")
	}
}

func detectDistro() (string, string) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "unknown", "unknown"
	}
	lines := strings.Split(string(data), "\n")
	vals := make(map[string]string)
	for _, l := range lines {
		parts := strings.SplitN(l, "=", 2)
		if len(parts) == 2 {
			vals[parts[0]] = strings.Trim(parts[1], `"`)
		}
	}
	return vals["ID"], vals["VERSION_CODENAME"]
}

func checkMinVersion(id, codename string) {
	debianMinimum := map[string]bool{
		"bookworm": true,
		"trixie":   true,
		"forky":    true,
		"sid":      true,
		"unstable": true,
		"testing":  true,
	}
	ubuntuMinimum := map[string]bool{
		"jammy":  true,
		"noble":  true,
		"plucky": true,
		"devel":  true,
	}
	switch id {
	case "debian":
		if !debianMinimum[codename] {
			die("Debian release '%s' is not supported. Minimum supported release is bookworm (12)", codename)
		}
	case "ubuntu":
		if !ubuntuMinimum[codename] {
			die("Ubuntu release '%s' is not supported. Only LTS and devel releases are supported (e.g., jammy, noble, or plucky)", codename)
		}
	default:
		die("Distribution '%s' is not supported. Only Debian and Ubuntu are supported by this tool", id)
	}
}

func cargoBinDir() string {
	cargoHome := os.Getenv("CARGO_HOME")
	if cargoHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "/root"
		}
		cargoHome = filepath.Join(home, ".cargo")
	}
	return filepath.Join(cargoHome, "bin")
}

func ensureCargoBinInPath() {
	binDir := cargoBinDir()
	current := os.Getenv("PATH")
	if !strings.Contains(current, binDir) {
		os.Setenv("PATH", binDir+":"+current)
	}
}

func collectAllBuildDeps() []string {
	seen := make(map[string]bool)
	var result []string
	for _, dep := range globalBuildDeps {
		if !seen[dep] {
			seen[dep] = true
			result = append(result, dep)
		}
	}
	for _, deps := range perComponentBuildDeps {
		for _, dep := range deps {
			if !seen[dep] {
				seen[dep] = true
				result = append(result, dep)
			}
		}
	}
	return result
}

func installBuildDeps() {
	log("Installing build dependencies (may require sudo password)")
	allDeps := collectAllBuildDeps()
	args := append([]string{"install", "-y", "--no-install-recommends"}, allDeps...)
	executable := "apt-get"
	execArgs := args
	if os.Geteuid() != 0 {
		executable = "sudo"
		execArgs = append([]string{"apt-get"}, args...)
	}
	if err := run("", executable, execArgs...); err != nil {
		die("Failed to install build dependencies: %v", err)
	}
	ensureCargoBinInPath()
	if _, err := exec.LookPath("rustup"); err != nil {
		log("rustup not found; installing via sh.rustup.rs")
		if err := run("", "sh", "-c", "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y"); err != nil {
			die("Failed to install rustup: %v", err)
		}
	}
	log("Configuring Rust stable toolchain via rustup")
	if err := run("", "rustup", "default", "stable"); err != nil {
		die("Failed to set rustup default to stable: %v", err)
	}
	ensureCargoBinInPath()
	if _, err := exec.LookPath("just"); err != nil {
		log("'just' not found; installing via cargo")
		if err := run("", "cargo", "install", "just"); err != nil {
			die("Failed to install just via cargo: %v", err)
		}
		ensureCargoBinInPath()
	}
}

func loadReposConfig(path string) (*ReposConfig, string) {
	if path == "built-in" {
		return getFinderRepos(), "built-in"
	}
	paths := []string{path}
	if !filepath.IsAbs(path) {
		if exe, err := os.Executable(); err == nil {
			paths = append(paths, filepath.Join(filepath.Dir(exe), path))
		}
		paths = append(paths, filepath.Join("/usr/share/cosmic-deb", path))
	}
	var data []byte
	var err error
	var foundPath string
	for _, p := range paths {
		data, err = os.ReadFile(p)
		if err == nil {
			foundPath = p
			break
		}
	}
	if err != nil {
		die("Failed to read repos config at '%s' (searched in %v). Run with -update-repos to generate it.", path, paths)
	}
	var cfg ReposConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		die("Failed to parse repos config '%s': %v", foundPath, err)
	}
	if len(cfg.Repos) == 0 {
		die("Repos config '%s' contains no repositories", foundPath)
	}
	return &cfg, foundPath
}

func latestEpochTag(repoURL string) string {
	cloneURL := repoURL
	if !strings.HasSuffix(cloneURL, ".git") {
		cloneURL += ".git"
	}
	cmd := exec.Command("git", "ls-remote", "--tags", "--sort=-version:refname", cloneURL)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		ref := parts[1]
		if strings.HasSuffix(ref, "^{}") {
			continue
		}
		tag := strings.TrimPrefix(ref, "refs/tags/")
		if strings.HasPrefix(tag, "epoch-") {
			return tag
		}
	}
	return ""
}

func updateReposConfig(path string, existing *ReposConfig) *ReposConfig {
	log("Updating repos config from upstream tags...")
	updated := &ReposConfig{
		GeneratedAt: time.Now().Format("2006-01-02"),
		Repos:       make([]RepoEntry, 0, len(existing.Repos)),
	}
	latestEpoch := ""
	for _, repo := range existing.Repos {
		entry := RepoEntry{
			Name:   repo.Name,
			URL:    repo.URL,
			Branch: repo.Branch,
		}
		tag := latestEpochTag(repo.URL)
		if tag != "" {
			entry.Tag = tag
			if latestEpoch == "" {
				latestEpoch = tag
			}
			log("  %-40s %s", repo.Name, tag)
		} else if repo.Branch != "" {
			entry.Tag = ""
			log("  %-40s (no epoch tag, using branch: %s)", repo.Name, repo.Branch)
		} else {
			entry.Tag = repo.Tag
			log("  %-40s (unchanged: %s)", repo.Name, repo.Tag)
		}
		updated.Repos = append(updated.Repos, entry)
	}
	if latestEpoch != "" {
		updated.EpochLatest = latestEpoch
	} else {
		updated.EpochLatest = existing.EpochLatest
	}

	if path == "built-in" {
		path = "repos.json"
	}

	data, err := json.MarshalIndent(updated, "", "  ")
	if err != nil {
		die("Failed to serialise updated repos config: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		die("Failed to write updated repos config to '%s': %v", path, err)
	}
	log("Repos config written to: %s (epoch_latest: %s)", path, updated.EpochLatest)

	return updated
}

func devUpdateFinder(cfg *ReposConfig) {
	var sb strings.Builder
	sb.WriteString("package main\n\nfunc getFinderRepos() *ReposConfig {\n")
	sb.WriteString("\treturn &ReposConfig{\n")
	sb.WriteString(fmt.Sprintf("\t\tGeneratedAt: %q,\n", cfg.GeneratedAt))
	sb.WriteString(fmt.Sprintf("\t\tEpochLatest: %q,\n", cfg.EpochLatest))
	sb.WriteString("\t\tRepos: []RepoEntry{\n")
	for _, r := range cfg.Repos {
		if r.Branch != "" {
			sb.WriteString(fmt.Sprintf("\t\t\t{Name: %q, URL: %q, Branch: %q},\n", r.Name, r.URL, r.Branch))
		} else {
			sb.WriteString(fmt.Sprintf("\t\t\t{Name: %q, URL: %q, Tag: %q},\n", r.Name, r.URL, r.Tag))
		}
	}
	sb.WriteString("\t\t},\n\t}\n}\n")
	if err := os.WriteFile("finder.go", []byte(sb.String()), 0644); err != nil {
		die("Failed to write to finder.go: %v", err)
	}
	log("Successfully rebuilt finder.go for contributors.")
}

func effectiveTag(repo RepoEntry, globalTag string) string {
	if globalTag != "" {
		return globalTag
	}
	if repo.Tag != "" {
		return repo.Tag
	}
	return ""
}

func archiveURL(repo RepoEntry, tag string) string {
	if tag != "" {
		return fmt.Sprintf("%s/archive/refs/tags/%s.tar.gz", repo.URL, tag)
	}
	branch := repo.Branch
	if branch == "" {
		branch = defaultBranch(repo.URL)
	}
	return fmt.Sprintf("%s/archive/refs/heads/%s.tar.gz", repo.URL, branch)
}

func detectExtractedDir(workDir, tarPath string) (string, error) {
	cmd := exec.Command("tar", "-tzf", tarPath)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list tar contents: %v", err)
	}
	lines := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)
	if len(lines) == 0 || lines[0] == "" {
		return "", fmt.Errorf("tar archive appears to be empty")
	}
	topDir := strings.SplitN(lines[0], "/", 2)[0]
	return filepath.Join(workDir, topDir), nil
}

func defaultBranch(repoURL string) string {
	cloneURL := repoURL
	if !strings.HasSuffix(cloneURL, ".git") {
		cloneURL += ".git"
	}
	cmd := exec.Command("git", "ls-remote", "--symref", cloneURL, "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "main"
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "ref: refs/heads/") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				return strings.TrimPrefix(parts[0], "ref: refs/heads/")
			}
		}
	}
	return "main"
}

func gitCloneSource(workDir string, repo RepoEntry, tag, dest string) string {
	cloneURL := repo.URL
	if !strings.HasSuffix(cloneURL, ".git") {
		cloneURL += ".git"
	}
	args := []string{"clone", "--depth", "1"}
	if tag != "" {
		args = append(args, "--branch", tag)
	} else if repo.Branch != "" {
		args = append(args, "--branch", repo.Branch)
	} else {
		branch := defaultBranch(repo.URL)
		log("Detected default branch for %s: %s", repo.Name, branch)
		args = append(args, "--branch", branch)
	}
	args = append(args, cloneURL, dest)
	log("Cloning %s from %s", repo.Name, cloneURL)
	if err := run(workDir, "git", args...); err != nil {
		die("Failed to git clone %s: %v", repo.Name, err)
	}
	return dest
}

func downloadSource(workDir string, repo RepoEntry, tag string) string {
	dest := filepath.Join(workDir, repo.Name)
	if _, err := os.Stat(dest); err == nil {
		log("Source already present: %s", repo.Name)
		return dest
	}
	url := archiveURL(repo, tag)
	tarName := repo.Name + ".tar.gz"
	tarPath := filepath.Join(workDir, tarName)
	log("Downloading source archive: %s", repo.Name)
	err := run("", "curl", "-fSL", "-o", tarPath, url)
	if err != nil {
		_ = os.Remove(tarPath)
		log("Tarball download failed for %s, falling back to git clone", repo.Name)
		return gitCloneSource(workDir, repo, tag, dest)
	}
	extractedDir, err := detectExtractedDir(workDir, tarPath)
	if err != nil {
		_ = os.Remove(tarPath)
		log("Failed to inspect archive for %s: %v, falling back to git clone", repo.Name, err)
		return gitCloneSource(workDir, repo, tag, dest)
	}
	if err := run(workDir, "tar", "-xzf", tarName); err != nil {
		_ = os.Remove(tarPath)
		log("Failed to extract source for %s: %v, falling back to git clone", repo.Name, err)
		return gitCloneSource(workDir, repo, tag, dest)
	}
	if _, err := os.Stat(extractedDir); err != nil {
		_ = os.Remove(tarPath)
		log("Expected extracted directory not found for %s, falling back to git clone", repo.Name)
		return gitCloneSource(workDir, repo, tag, dest)
	}
	if err := os.Rename(extractedDir, dest); err != nil {
		die("Failed to rename extracted directory for %s: %v", repo.Name, err)
	}
	_ = os.Remove(tarPath)
	return dest
}

func getVersionFromChangelog(repoDir string) string {
	changelogPath := filepath.Join(repoDir, "debian", "changelog")
	if _, err := os.Stat(changelogPath); err != nil {
		return ""
	}
	cmd := exec.Command("dpkg-parsechangelog", "--file="+changelogPath, "--show-field", "Version")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	v := strings.TrimSpace(string(out))
	if idx := strings.Index(v, "-"); idx > 0 {
		v = v[:idx]
	}
	return v
}

func getVersion(repoDir, fallbackTag string) string {
	if v := getVersionFromChangelog(repoDir); v != "" {
		return v
	}
	cargoPath := filepath.Join(repoDir, "Cargo.toml")
	if _, err := os.Stat(cargoPath); err == nil {
		if data, err := os.ReadFile(cargoPath); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "version") && strings.Contains(line, "=") {
					parts := strings.SplitN(line, "=", 2)
					v := strings.Trim(parts[1], ` "'`)
					if v != "" {
						return v
					}
				}
			}
		}
	}
	if fallbackTag != "" {
		v := strings.TrimPrefix(fallbackTag, "epoch-")
		v = strings.TrimPrefix(v, "v")
		return v
	}
	return "0.1.0"
}

func buildEnv() []string {
	existing := os.Getenv("RUSTFLAGS")
	var rfVal string
	if existing != "" {
		rfVal = existing + " " + rustLinkerFlags
	} else {
		rfVal = rustLinkerFlags
	}
	return []string{"RUSTFLAGS=" + rfVal}
}

func runVendor(repoDir string) {
	justfile := filepath.Join(repoDir, "justfile")
	justfileUpper := filepath.Join(repoDir, "Justfile")
	hasJustfile := false
	if _, err := os.Stat(justfile); err == nil {
		hasJustfile = true
	} else if _, err := os.Stat(justfileUpper); err == nil {
		hasJustfile = true
	}
	if hasJustfile {
		log("Running 'just vendor' for dependency vendoring")
		_ = run(repoDir, "just", "vendor")
	}
}

func buildRepo(repoDir, repoName string, jobs int) error {
	log("Building component: %s", repoName)
	env := buildEnv()

	cargoToml := filepath.Join(repoDir, "Cargo.toml")
	if _, err := os.Stat(cargoToml); err == nil {
		_ = run(repoDir, "sed", "-i", "s/lto = \"fat\"/lto = \"thin\"/", "Cargo.toml")
	}

	justfile := filepath.Join(repoDir, "justfile")
	justfileUpper := filepath.Join(repoDir, "Justfile")
	makefile := filepath.Join(repoDir, "Makefile")

	hasJustfile := false
	justfilePath := justfile
	if _, err := os.Stat(justfile); err == nil {
		hasJustfile = true
	} else if _, err := os.Stat(justfileUpper); err == nil {
		hasJustfile = true
		justfilePath = justfileUpper
	}
	_ = justfilePath

	if hasJustfile {
		vendorTar := filepath.Join(repoDir, "vendor.tar")
		if _, err := os.Stat(vendorTar); err == nil {
			return runEnv(repoDir, env, "just", "build-vendored")
		}
		err := runEnv(repoDir, env, "just", "build-release", "--frozen")
		if err != nil {
			return runEnv(repoDir, env, "just", "build-release")
		}
		return nil
	}
	if _, err := os.Stat(makefile); err == nil {
		return runEnv(repoDir, env, "make",
			fmt.Sprintf("-j%d", jobs),
			"ARGS=--frozen --release",
		)
	}
	if _, err := os.Stat(cargoToml); err == nil {
		return runEnv(repoDir, env, "cargo", "build", "--release", "--frozen",
			fmt.Sprintf("--jobs=%d", jobs),
		)
	}
	return fmt.Errorf("no recognised build system found in %s", repoDir)
}

func installToStage(repoDir, stageDir string) error {
	justfile := filepath.Join(repoDir, "justfile")
	justfileUpper := filepath.Join(repoDir, "Justfile")
	makefile := filepath.Join(repoDir, "Makefile")

	hasJustfile := false
	if _, err := os.Stat(justfile); err == nil {
		hasJustfile = true
	} else if _, err := os.Stat(justfileUpper); err == nil {
		hasJustfile = true
	}

	if hasJustfile {
		return run(repoDir, "just", "rootdir="+stageDir, "DESTDIR="+stageDir, "install")
	}
	if _, err := os.Stat(makefile); err == nil {
		return run(repoDir, "make",
			"prefix=/usr",
			"libexecdir=/usr/lib",
			"DESTDIR="+stageDir,
			"install",
		)
	}
	return fmt.Errorf("no install target found for component %s", repoDir)
}

func buildWithDebian(repoDir, outDir string) error {
	log("Using existing debian/ directory to build package...")
	runVendor(repoDir)
	if err := run(repoDir, "dpkg-buildpackage", "-us", "-uc", "-b"); err != nil {
		return err
	}
	parent := filepath.Dir(repoDir)
	files, err := os.ReadDir(parent)
	if err != nil {
		return err
	}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".deb") {
			oldPath := filepath.Join(parent, f.Name())
			newPath := filepath.Join(outDir, f.Name())
			if err := os.Rename(oldPath, newPath); err != nil {
				log("Warning: Failed to move .deb to output directory: %v", err)
			}
		}
	}
	return nil
}

func runtimeDepsForComponent(name string) []string {
	if deps, ok := perComponentRuntimeDeps[name]; ok {
		return deps
	}
	return nil
}

func buildDebianPackage(cfg *Config, stageDir, outDir, pkgName, version string) error {
	// Sanity check: Ensure staging directory actually contains files (post-compile artifacts)
	files, err := os.ReadDir(stageDir)
	if err != nil {
		return err
	}
	hasContent := false
	for _, f := range files {
		if f.Name() != "DEBIAN" {
			hasContent = true
			break
		}
	}
	if !hasContent {
		return fmt.Errorf("staging directory for %s is empty; compilation or installation might have failed to produce artifacts", pkgName)
	}

	debianDir := filepath.Join(stageDir, "DEBIAN")
	if err := os.MkdirAll(debianDir, 0755); err != nil {
		return err
	}
	arch := "amd64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}

	var depEntries []string
	depEntries = append(depEntries, "${shlibs:Depends}")
	componentDeps := runtimeDepsForComponent(pkgName)
	depEntries = append(depEntries, componentDeps...)
	depStr := strings.Join(depEntries, ", ")

	var recommendEntries []string
	if recs, ok := perComponentRecommends[pkgName]; ok {
		recommendEntries = append(recommendEntries, recs...)
	}

	maintainer := fmt.Sprintf("%s <%s>", cfg.maintainerName, cfg.maintainerEmail)

	section := "x11"
	if _, ok := map[string]bool{
		"cosmic-session": true, "cosmic-files": true,
		"cosmic-applets": true, "cosmic-edit": true,
		"cosmic-store": true, "cosmic-bg": true,
		"cosmic-greeter": true, "cosmic-icons": true,
		"cosmic-osd": true, "cosmic-notifications": true,
		"cosmic-panel": true, "cosmic-launcher": true,
		"cosmic-screenshot": true, "cosmic-idle": true,
		"cosmic-workspaces": true, "cosmic-initial-setup": true,
		"cosmic-term": true, "cosmic-player": true,
		"cosmic-app-library":        true,
		"cosmic-settings-daemon":    true,
		"xdg-desktop-portal-cosmic": true,
		"pop-launcher":              true,
	}[pkgName]; ok {
		section = "admin"
	}
	if pkgName == "cosmic-settings" || pkgName == "cosmic-randr" || pkgName == "pop-launcher" {
		section = "utils"
	}

	control := fmt.Sprintf(`Package: %s
Version: %s
Section: %s
Priority: optional
Architecture: %s
Depends: %s
`, pkgName, version, section, arch, depStr)

	if len(recommendEntries) > 0 {
		control += fmt.Sprintf("Recommends: %s\n", strings.Join(recommendEntries, ", "))
	}

	control += fmt.Sprintf(`Maintainer: %s
Description: COSMIC Desktop Environment component — %s
 Built from upstream source via the cosmic-deb build tool.
`, maintainer, pkgName)

	if err := os.WriteFile(filepath.Join(debianDir, "control"), []byte(control), 0644); err != nil {
		return err
	}
	pkgFile := filepath.Join(outDir, fmt.Sprintf("%s_%s_%s.deb", pkgName, version, arch))
	return run("", "fakeroot", "dpkg-deb", "--build", stageDir, pkgFile)
}

func buildMetaPackage(cfg *Config, outDir, version string, builtRepos []string) error {
	arch := "amd64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}
	stageDir := filepath.Join(cfg.workDir, metaPkgName+"-stage")
	if err := os.MkdirAll(filepath.Join(stageDir, "DEBIAN"), 0755); err != nil {
		return err
	}
	var deps []string
	deps = append(deps, builtRepos...)
	depStr := strings.Join(deps, ", ")
	maintainer := fmt.Sprintf("%s <%s>", cfg.maintainerName, cfg.maintainerEmail)
	control := fmt.Sprintf(`Package: %s
Version: %s
Section: x11
Priority: optional
Architecture: %s
Depends: %s
Maintainer: %s
Description: COSMIC Desktop Environment meta package
 This meta package installs the complete COSMIC Desktop Environment
 by declaring dependencies on all COSMIC component packages built
 by the cosmic-deb build tool.
`, metaPkgName, version, arch, depStr, maintainer)
	if err := os.WriteFile(filepath.Join(stageDir, "DEBIAN", "control"), []byte(control), 0644); err != nil {
		return err
	}
	pkgFile := filepath.Join(outDir, fmt.Sprintf("%s_%s_%s.deb", metaPkgName, version, arch))
	log("Building meta package: %s", metaPkgName)
	return run("", "fakeroot", "dpkg-deb", "--build", stageDir, pkgFile)
}

func resolveTag(cfg *Config, repo RepoEntry) string {
	if cfg.useBranch {
		return ""
	}
	return effectiveTag(repo, cfg.globalTag)
}

func processRepo(cfg *Config, repo RepoEntry) bool {
	tag := resolveTag(cfg, repo)
	sourceLabel := tag
	if sourceLabel == "" {
		sourceLabel = "branch:" + func() string {
			if repo.Branch != "" {
				return repo.Branch
			}
			return "main"
		}()
	}
	log("Processing component: %s (%s)", repo.Name, sourceLabel)
	repoDir := downloadSource(cfg.workDir, repo, tag)
	version := getVersion(repoDir, tag)

	if _, err := os.Stat(filepath.Join(repoDir, "debian")); err == nil {
		if err := buildWithDebian(repoDir, cfg.outDir); err == nil {
			log("Successfully packaged %s using debian/ sources", repo.Name)
			return true
		} else {
			log("Warning: debian/ build failed for %s: %v. Falling back to manual build.", repo.Name, err)
		}
	}

	runVendor(repoDir)

	if err := buildRepo(repoDir, repo.Name, cfg.jobs); err != nil {
		log("Warning: Build failed for %s: %v", repo.Name, err)
		return false
	}
	stageDir := filepath.Join(cfg.workDir, repo.Name+"-stage")
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		die("Failed to create staging directory: %v", err)
	}
	if err := installToStage(repoDir, stageDir); err != nil {
		log("Warning: Installation to staging directory failed for %s: %v", repo.Name, err)
		return false
	}
	if err := buildDebianPackage(cfg, stageDir, cfg.outDir, repo.Name, version); err != nil {
		log("Warning: Debian package build failed for %s: %v", repo.Name, err)
		return false
	}
	log("Successfully packaged %s at version %s", repo.Name, version)
	return true
}

func main() {
	cfg := &Config{}
	useTui := flag.Bool("tui", false, "Launch interactive TUI wizard")
	flag.StringVar(&cfg.globalTag, "tag", "", "Override tag for all repos")
	flag.StringVar(&cfg.reposFile, "repos", defaultReposFile, "Path to repos JSON config file")
	flag.BoolVar(&cfg.updateRepos, "update-repos", false, "Fetch latest epoch tags and exit")
	flag.BoolVar(&cfg.genConfig, "gen-config", false, "Generate repos.json and exit")
	flag.BoolVar(&cfg.devFinder, "dev-finder", false, "Update finder.go and exit")
	flag.StringVar(&cfg.workDir, "workdir", "cosmic-work", "Working directory")
	flag.StringVar(&cfg.outDir, "outdir", outputPkgDir, "Output directory")
	flag.IntVar(&cfg.jobs, "jobs", runtime.NumCPU(), "Parallel jobs")
	flag.BoolVar(&cfg.skipDeps, "skip-deps", false, "Skip dependency installation")
	flag.StringVar(&cfg.only, "only", "", "Restrict build to a single component")
	flag.BoolVar(&cfg.useBranch, "use-branch", false, "Build from main branch HEAD instead of epoch tags")
	flag.Parse()

	ensureCargoBinInPath()
	checkAptBased()
	distroID, codename := detectDistro()
	checkMinVersion(distroID, codename)
	log("Detected distribution: %s %s", distroID, codename)

	reposCfg, actualReposFile := loadReposConfig(cfg.reposFile)

	if cfg.updateRepos {
		updateReposConfig(actualReposFile, reposCfg)
		os.Exit(0)
	}

	if cfg.genConfig {
		path := actualReposFile
		if path == "built-in" {
			path = "repos.json"
		}
		data, err := json.MarshalIndent(reposCfg, "", "  ")
		if err != nil {
			die("Failed to serialise config: %v", err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			die("Failed to write config to '%s': %v", path, err)
		}
		log("Config successfully generated at %s", path)
		os.Exit(0)
	}

	if cfg.devFinder {
		devUpdateFinder(reposCfg)
		os.Exit(0)
	}

	log("Loaded %d repos from %s (epoch_latest: %s)", len(reposCfg.Repos), actualReposFile, reposCfg.EpochLatest)

	var epochTags []string
	seen := make(map[string]bool)
	for _, r := range reposCfg.Repos {
		if r.Tag != "" && !seen[r.Tag] {
			seen[r.Tag] = true
			epochTags = append(epochTags, r.Tag)
		}
	}

	if *useTui {
		choices, confirmed, err := tui.RunWizard(distroID, codename, epochTags)
		if err != nil {
			die("TUI failure: %v", err)
		}
		if !confirmed {
			os.Exit(0)
		}
		cfg.maintainerName = choices["maintainer_name"]
		cfg.maintainerEmail = choices["maintainer_email"]
		if choices["release"] == "branch" {
			cfg.useBranch = true
			cfg.globalTag = ""
		} else if cfg.globalTag == "" {
			cfg.globalTag = choices["release"]
		}
		cfg.workDir = choices["workdir"]
		cfg.outDir = choices["outdir"]
		cfg.only = choices["only"]
		if v, ok := choices["jobs"]; ok {
			cfg.jobs, _ = strconv.Atoi(v)
		}
		cfg.skipDeps = choices["skip_deps"] == "y"
	} else {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter Maintainer Name [cosmic-deb]: ")
		name, _ := reader.ReadString('\n')
		cfg.maintainerName = strings.TrimSpace(name)
		fmt.Print("Enter Maintainer Email [cosmic-deb@example.com]: ")
		email, _ := reader.ReadString('\n')
		cfg.maintainerEmail = strings.TrimSpace(email)
		if cfg.maintainerName == "" {
			cfg.maintainerName = "cosmic-deb"
		}
		if cfg.maintainerEmail == "" {
			cfg.maintainerEmail = "cosmic-deb@example.com"
		}

		if !cfg.useBranch && cfg.globalTag == "" {
			fmt.Println("Select source mode:")
			fmt.Printf("  [b] Latest (main branch HEAD)\n")
			if len(epochTags) > 0 {
				limit := len(epochTags)
				if limit > 10 {
					limit = 10
				}
				for i := 0; i < limit; i++ {
					fmt.Printf("  [%d] %s\n", i, epochTags[i])
				}
			}
			fmt.Printf("  [*] Use per-repo tags from repos config\n")
			fmt.Print("Select option (b / index / Enter for per-repo tags): ")
			idxStr, _ := reader.ReadString('\n')
			idxStr = strings.TrimSpace(idxStr)
			if idxStr == "b" {
				cfg.useBranch = true
			} else if idxStr != "" {
				idx, err := strconv.Atoi(idxStr)
				if err == nil && idx >= 0 && idx < len(epochTags) {
					cfg.globalTag = epochTags[idx]
				}
			}
		}
	}

	if cfg.useBranch {
		log("Source mode: main branch HEAD (latest, untagged)")
	} else if cfg.globalTag != "" {
		log("Global tag override: %s (applied to all repos)", cfg.globalTag)
	} else {
		log("Using per-repo tags from: %s", actualReposFile)
	}

	if !cfg.skipDeps {
		installBuildDeps()
	} else {
		ensureCargoBinInPath()
	}

	buildFunc := func() {
		if err := os.MkdirAll(cfg.workDir, 0755); err != nil {
			die("Failed to create working directory: %v", err)
		}
		if err := os.MkdirAll(cfg.outDir, 0755); err != nil {
			die("Failed to create output directory: %v", err)
		}

		repos := reposCfg.Repos
		if cfg.only != "" {
			var found bool
			for _, r := range reposCfg.Repos {
				if r.Name == cfg.only {
					repos = []RepoEntry{r}
					found = true
					break
				}
			}
			if !found {
				die("Component '%s' not found in repos config", cfg.only)
			}
		}

		total := len(repos)
		var builtRepos []string
		for i, repo := range repos {
			if tuiProg != nil {
				tuiProg.Send(tui.ProgressMsg{Step: i + 1, Total: total, Name: repo.Name})
			}
			if processRepo(cfg, repo) {
				builtRepos = append(builtRepos, repo.Name)
			}
		}

		if cfg.only == "" && len(builtRepos) > 0 {
			metaVersion := reposCfg.EpochLatest
			if cfg.globalTag != "" {
				metaVersion = cfg.globalTag
			}
			metaVersion = strings.TrimPrefix(metaVersion, "epoch-")
			metaVersion = strings.TrimPrefix(metaVersion, "v")
			if cfg.useBranch {
				metaVersion = "0.0.0+main"
			}
			if err := buildMetaPackage(cfg, cfg.outDir, metaVersion, builtRepos); err != nil {
				log("Warning: Meta package build failed: %v", err)
			}
		}

		log("All packages have been written to: %s", cfg.outDir)
		if tuiProg != nil {
			tuiProg.Send(tui.DoneMsg{})
		}
	}

	if *useTui {
		tuiProg = tea.NewProgram(tui.MonitorModel{TotalSteps: len(reposCfg.Repos)}, tea.WithAltScreen())
		go buildFunc()
		if _, err := tuiProg.Run(); err != nil {
			die("Monitor failure: %v", err)
		}
	} else {
		buildFunc()
		log("To install all COSMIC components at once, run: sudo bash scripts/install-local.sh %s", cfg.outDir)
	}
}

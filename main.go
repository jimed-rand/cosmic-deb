package main

import (
	"bufio"
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
	githubOrg    = "pop-os"
	epochRepo    = "cosmic-epoch"
	outputPkgDir = "cosmic-packages"
	metaPkgName  = "cosmic-desktop"
)

var cosmicRepos = []string{
	"cosmic-applets",
	"cosmic-applibrary",
	"cosmic-bg",
	"cosmic-comp",
	"cosmic-edit",
	"cosmic-files",
	"cosmic-greeter",
	"cosmic-icons",
	"cosmic-idle",
	"cosmic-launcher",
	"cosmic-notifications",
	"cosmic-osd",
	"cosmic-panel",
	"cosmic-player",
	"cosmic-randr",
	"cosmic-screenshot",
	"cosmic-session",
	"cosmic-settings",
	"cosmic-settings-daemon",
	"cosmic-store",
	"cosmic-term",
	"cosmic-theme-extra",
	"cosmic-wallpapers",
	"cosmic-workspaces-epoch",
	"xdg-desktop-portal-cosmic",
}

var buildDeps = []string{
	"build-essential",
	"curl",
	"git",
	"libdbus-1-dev",
	"libdisplay-info-dev",
	"libflatpak-dev",
	"libglvnd-dev",
	"libgstreamer-plugins-base1.0-dev",
	"libgstreamer1.0-dev",
	"libinput-dev",
	"libpam0g-dev",
	"libpixman-1-dev",
	"libseat-dev",
	"libssl-dev",
	"libwayland-dev",
	"libxkbcommon-dev",
	"lld",
	"pkg-config",
	"rustup",
}

var runtimeDeps = []string{
	"dbus",
	"libdbus-1-3",
	"libdisplay-info1",
	"libflatpak0",
	"libgstreamer-plugins-base1.0-0",
	"libgstreamer1.0-0",
	"libinput10",
	"libpam0g",
	"libpixman-1-0",
	"libseat1",
	"libssl3",
	"libwayland-client0",
	"libwayland-server0",
	"libxkbcommon0",
	"udev",
}

type Config struct {
	tag             string
	workDir         string
	outDir          string
	jobs            int
	skipDeps        bool
	only            string
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

func installBuildDeps() {
	log("Installing build dependencies via apt-get")
	args := append([]string{"install", "-y", "--no-install-recommends"}, buildDeps...)
	if err := run("", "apt-get", args...); err != nil {
		die("Failed to install build dependencies: %v", err)
	}
	log("Configuring Rust stable toolchain via rustup")
	if err := run("", "rustup", "default", "stable"); err != nil {
		die("Failed to set rustup default to stable: %v", err)
	}
	if _, err := exec.LookPath("just"); err != nil {
		log("'just' not found; installing via cargo")
		if err := run("", "cargo", "install", "just"); err != nil {
			die("Failed to install just via cargo: %v", err)
		}
	}
}

func getEpochTags() []string {
	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", githubOrg, epochRepo)
	cmd := exec.Command("git", "ls-remote", "--tags", "--sort=-version:refname", repoURL)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	seen := make(map[string]bool)
	var tags []string
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
		if !strings.HasPrefix(tag, "epoch-") {
			continue
		}
		if !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
	}
	return tags
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

func downloadSource(workDir, repo, tag string) string {
	dest := filepath.Join(workDir, repo)
	if _, err := os.Stat(dest); err == nil {
		log("Source already present: %s", repo)
		return dest
	}
	url := fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/%s.tar.gz", githubOrg, repo, tag)
	tarPath := filepath.Join(workDir, repo+".tar.gz")
	log("Downloading source archive: %s (%s)", repo, tag)
	if err := run(workDir, "curl", "-fSL", "-o", tarPath, url); err != nil {
		die("Failed to download source for %s: %v", repo, err)
	}
	extractedDir, err := detectExtractedDir(workDir, tarPath)
	if err != nil {
		die("Failed to detect extracted directory for %s: %v", repo, err)
	}
	if err := run(workDir, "tar", "-xzf", tarPath); err != nil {
		die("Failed to extract source for %s: %v", repo, err)
	}
	if _, err := os.Stat(extractedDir); err != nil {
		die("Expected extracted directory not found for %s: %s", repo, extractedDir)
	}
	if err := os.Rename(extractedDir, dest); err != nil {
		die("Failed to rename extracted directory for %s: %v", repo, err)
	}
	_ = os.Remove(tarPath)
	return dest
}

func getVersion(repoDir, fallbackBase string) string {
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
	if fallbackBase != "" {
		v := strings.TrimPrefix(fallbackBase, "epoch-")
		v = strings.TrimPrefix(v, "v")
		return v
	}
	return "0.1.0"
}

func buildRepo(repoDir, repo string, jobs int) error {
	log("Building component: %s", repo)
	if _, err := os.Stat(filepath.Join(repoDir, "justfile")); err == nil {
		return run(repoDir, "just", fmt.Sprintf("-j%d", jobs), "build")
	}
	if _, err := os.Stat(filepath.Join(repoDir, "Makefile")); err == nil {
		return run(repoDir, "make", fmt.Sprintf("-j%d", jobs))
	}
	if _, err := os.Stat(filepath.Join(repoDir, "Cargo.toml")); err == nil {
		return run(repoDir, "cargo", "build", "--release",
			fmt.Sprintf("--jobs=%d", jobs))
	}
	return fmt.Errorf("no recognised build system found in %s", repoDir)
}

func installToStage(repoDir, stageDir string) error {
	if _, err := os.Stat(filepath.Join(repoDir, "justfile")); err == nil {
		return run(repoDir, "just", "rootdir="+stageDir, "install")
	}
	if _, err := os.Stat(filepath.Join(repoDir, "Makefile")); err == nil {
		return run(repoDir, "make", "DESTDIR="+stageDir, "install")
	}
	return fmt.Errorf("no install target found for component %s", repoDir)
}

func buildDebianPackage(cfg *Config, stageDir, outDir, pkgName, version string, deps []string) error {
	debianDir := filepath.Join(stageDir, "DEBIAN")
	if err := os.MkdirAll(debianDir, 0755); err != nil {
		return err
	}
	arch := "amd64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}
	depStr := strings.Join(deps, ", ")
	maintainer := fmt.Sprintf("%s <%s>", cfg.maintainerName, cfg.maintainerEmail)
	control := fmt.Sprintf(`Package: %s
Version: %s
Section: x11
Priority: optional
Architecture: %s
Depends: %s
Maintainer: %s
Description: COSMIC Desktop Environment component — %s
 Built from upstream source via the cosmic-deb build tool.
`, pkgName, version, arch, depStr, maintainer, pkgName)
	if err := os.WriteFile(filepath.Join(debianDir, "control"), []byte(control), 0644); err != nil {
		return err
	}
	pkgFile := filepath.Join(outDir, fmt.Sprintf("%s_%s_%s.deb", pkgName, version, arch))
	return run("", "dpkg-deb", "--build", stageDir, pkgFile)
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
	deps = append(deps, runtimeDeps...)
	for _, repo := range builtRepos {
		deps = append(deps, repo)
	}
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
	return run("", "dpkg-deb", "--build", stageDir, pkgFile)
}

func processRepo(cfg *Config, repo string) bool {
	log("Processing component: %s", repo)
	repoDir := downloadSource(cfg.workDir, repo, cfg.tag)
	version := getVersion(repoDir, cfg.tag)
	if err := buildRepo(repoDir, repo, cfg.jobs); err != nil {
		log("Warning: Build failed for %s: %v", repo, err)
		return false
	}
	stageDir := filepath.Join(cfg.workDir, repo+"-stage")
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		die("Failed to create staging directory: %v", err)
	}
	if err := installToStage(repoDir, stageDir); err != nil {
		log("Warning: Installation to staging directory failed for %s: %v", repo, err)
		return false
	}
	if err := buildDebianPackage(cfg, stageDir, cfg.outDir, repo, version, runtimeDeps); err != nil {
		log("Warning: Debian package build failed for %s: %v", repo, err)
		return false
	}
	log("Successfully packaged %s at version %s", repo, version)
	return true
}

func main() {
	cfg := &Config{}
	useTui := flag.Bool("tui", false, "Launch interactive TUI wizard")
	flag.StringVar(&cfg.tag, "tag", "", "Upstream COSMIC epoch release tag (e.g. epoch-1.0.7). When omitted, available tags are fetched and presented interactively.")
	flag.StringVar(&cfg.workDir, "workdir", "cosmic-work", "Working directory for source checkout and compilation")
	flag.StringVar(&cfg.outDir, "outdir", outputPkgDir, "Output directory for produced .deb packages")
	flag.IntVar(&cfg.jobs, "jobs", runtime.NumCPU(), "Number of parallel compilation jobs")
	flag.BoolVar(&cfg.skipDeps, "skip-deps", false, "Skip automatic installation of build dependencies")
	flag.StringVar(&cfg.only, "only", "", "Restrict the build to a single named cosmic-* component")
	flag.Parse()

	checkAptBased()
	distroID, codename := detectDistro()
	checkMinVersion(distroID, codename)
	log("Detected distribution: %s %s", distroID, codename)

	var tags []string
	if cfg.tag == "" {
		log("Fetching available epoch tags via git ls-remote...")
		tags = getEpochTags()
		if len(tags) == 0 {
			die("Failed to retrieve epoch tags from %s/%s. Ensure git is installed and the host has network access to github.com. Alternatively, pass -tag <tag> directly.", githubOrg, epochRepo)
		}
		log("Found %d epoch tag(s)", len(tags))
	}

	if *useTui {
		choices, confirmed, err := tui.RunWizard(distroID, codename, tags)
		if err != nil {
			die("TUI failure: %v", err)
		}
		if !confirmed {
			os.Exit(0)
		}
		cfg.maintainerName = choices["maintainer_name"]
		cfg.maintainerEmail = choices["maintainer_email"]
		if cfg.tag == "" {
			cfg.tag = choices["release"]
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

		if cfg.tag == "" {
			fmt.Println("Available epoch tags:")
			limit := len(tags)
			if limit > 10 {
				limit = 10
			}
			for i := 0; i < limit; i++ {
				fmt.Printf("  [%d] %s\n", i, tags[i])
			}
			fmt.Print("Select tag index [0]: ")
			idxStr, _ := reader.ReadString('\n')
			idxStr = strings.TrimSpace(idxStr)
			idx := 0
			if idxStr != "" {
				idx, _ = strconv.Atoi(idxStr)
			}
			if idx < 0 || idx >= len(tags) {
				idx = 0
			}
			cfg.tag = tags[idx]
		}
	}

	log("Selected release tag: %s", cfg.tag)

	buildFunc := func() {
		if !cfg.skipDeps {
			installBuildDeps()
		}
		if err := os.MkdirAll(cfg.workDir, 0755); err != nil {
			die("Failed to create working directory: %v", err)
		}
		if err := os.MkdirAll(cfg.outDir, 0755); err != nil {
			die("Failed to create output directory: %v", err)
		}

		repos := cosmicRepos
		if cfg.only != "" {
			repos = []string{cfg.only}
		}

		total := len(repos)
		var builtRepos []string
		for i, repo := range repos {
			if tuiProg != nil {
				tuiProg.Send(tui.ProgressMsg{Step: i + 1, Total: total, Name: repo})
			}
			if processRepo(cfg, repo) {
				builtRepos = append(builtRepos, repo)
			}
		}

		if cfg.only == "" && len(builtRepos) > 0 {
			metaVersion := strings.TrimPrefix(cfg.tag, "epoch-")
			metaVersion = strings.TrimPrefix(metaVersion, "v")
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
		tuiProg = tea.NewProgram(tui.MonitorModel{TotalSteps: len(cosmicRepos)}, tea.WithAltScreen())
		go buildFunc()
		if _, err := tuiProg.Run(); err != nil {
			die("Monitor failure: %v", err)
		}
	} else {
		buildFunc()
		log("To install all COSMIC components at once, run: sudo bash scripts/install-local.sh %s", cfg.outDir)
	}
}

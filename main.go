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
	"unzip",
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
			die("Debian release '%s' is not supported. The minimum supported release is bookworm (12)", codename)
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
		log("'Just' not found; installing via cargo")
		if err := run("", "cargo", "install", "just"); err != nil {
			die("Failed to install just via cargo: %v", err)
		}
	}
}

func getReleases() []string {
	cmd := exec.Command("curl", "-s", "https://api.github.com/repos/"+githubOrg+"/"+epochRepo+"/releases")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var releases []string
	s := string(out)
	for {
		idx := strings.Index(s, "\"tag_name\":")
		if idx == -1 {
			break
		}
		s = s[idx+len("\"tag_name\":"):]
		start := strings.Index(s, "\"")
		if start == -1 {
			break
		}
		s = s[start+1:]
		end := strings.Index(s, "\"")
		if end == -1 {
			break
		}
		releases = append(releases, s[:end])
		s = s[end+1:]
	}
	return releases
}

func downloadSource(workDir, repo, tag string) string {
	dest := filepath.Join(workDir, repo)
	if _, err := os.Stat(dest); err == nil {
		log("Source already present: %s", repo)
		return dest
	}
	url := fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/%s.zip", githubOrg, repo, tag)
	zipPath := filepath.Join(workDir, repo+".zip")
	log("Downloading release source: %s (%s)", repo, tag)
	if err := run(workDir, "curl", "-fSL", "-o", zipPath, url); err != nil {
		die("Failed to download source for %s: %v", repo, err)
	}
	if err := run(workDir, "unzip", "-q", "-o", zipPath); err != nil {
		die("Failed to extract source for %s: %v", repo, err)
	}
	extractedDir := filepath.Join(workDir, fmt.Sprintf("%s-%s", repo, strings.TrimPrefix(tag, "v")))
	if strings.HasPrefix(tag, "epoch-") {
		extractedDir = filepath.Join(workDir, fmt.Sprintf("%s-%s", repo, tag))
	}
	if _, err := os.Stat(extractedDir); err != nil {
		entries, _ := filepath.Glob(filepath.Join(workDir, repo+"-*"))
		for _, e := range entries {
			info, _ := os.Stat(e)
			if info != nil && info.IsDir() {
				extractedDir = e
				break
			}
		}
	}
	if err := os.Rename(extractedDir, dest); err != nil {
		die("Failed to rename extracted directory for %s: %v", repo, err)
	}
	_ = os.Remove(zipPath)
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
					v := strings.Trim(parts[1], " \"'")
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

func processRepo(cfg *Config, repo string) {
	log("Processing component: %s", repo)
	repoDir := downloadSource(cfg.workDir, repo, cfg.tag)
	version := getVersion(repoDir, cfg.tag)
	if err := buildRepo(repoDir, repo, cfg.jobs); err != nil {
		log("Warning: Build failed for %s: %v", repo, err)
		return
	}
	stageDir := filepath.Join(cfg.workDir, repo+"-stage")
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		die("Failed to create staging directory: %v", err)
	}
	if err := installToStage(repoDir, stageDir); err != nil {
		log("Warning: Installation to staging directory failed for %s: %v", repo, err)
		return
	}
	if err := buildDebianPackage(cfg, stageDir, cfg.outDir, repo, version, runtimeDeps); err != nil {
		log("Warning: Debian package build failed for %s: %v", repo, err)
		return
	}
	log("Successfully packaged %s at version %s", repo, version)
}

func main() {
	cfg := &Config{}
	useTui := flag.Bool("tui", false, "Launch interactive TUI wizard")
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

	log("Fetching releases from GitHub...")
	releases := getReleases()
	if len(releases) == 0 {
		die("Failed to fetch releases from GitHub")
	}

	if *useTui {
		choices, confirmed, err := tui.RunWizard(distroID, codename, releases)
		if err != nil {
			die("TUI failure: %v", err)
		}
		if !confirmed {
			os.Exit(0)
		}
		cfg.maintainerName = choices["maintainer_name"]
		cfg.maintainerEmail = choices["maintainer_email"]
		cfg.tag = choices["release"]
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

		fmt.Printf("Available releases:\n")
		for i, r := range releases {
			fmt.Printf("[%d] %s\n", i, r)
			if i >= 4 {
				break
			}
		}
		fmt.Printf("Select release index [0]: ")
		idxStr, _ := reader.ReadString('\n')
		idxStr = strings.TrimSpace(idxStr)
		idx := 0
		if idxStr != "" {
			idx, _ = strconv.Atoi(idxStr)
		}
		if idx < 0 || idx >= len(releases) {
			idx = 0
		}
		cfg.tag = releases[idx]
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
		for i, repo := range repos {
			if tuiProg != nil {
				tuiProg.Send(tui.ProgressMsg{Step: i + 1, Total: total, Name: repo})
			}
			processRepo(cfg, repo)
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
		log("To install, run: sudo bash scripts/install-local.sh %s", cfg.outDir)
	}
}

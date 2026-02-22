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
	"libpulse-dev",
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
	"libpulse0",
	"libseat1",
	"libssl3",
	"libwayland-client0",
	"libwayland-server0",
	"libxkbcommon0",
	"udev",
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

func installBuildDeps() {
	log("Installing build dependencies via apt-get")
	args := append([]string{"install", "-y", "--no-install-recommends"}, buildDeps...)
	if err := run("", "apt-get", args...); err != nil {
		die("Failed to install build dependencies: %v", err)
	}
	ensureCargoBinInPath()
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
	cmd := exec.Command("git", "ls-remote", "--tags", "--sort=-version:refname", repoURL)
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
		tag := latestEpochTag(repo.URL + ".git")
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
		branch = "master"
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

func getVersion(repoDir, fallbackTag string) string {
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

func buildRepo(repoDir, repoName string, jobs int) error {
	log("Building component: %s", repoName)
	if _, err := os.Stat(filepath.Join(repoDir, "justfile")); err == nil {
		return run(repoDir, "just", "build")
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
	return run("", "dpkg-deb", "--build", stageDir, pkgFile)
}

func processRepo(cfg *Config, repo RepoEntry) bool {
	tag := effectiveTag(repo, cfg.globalTag)
	log("Processing component: %s (tag: %s)", repo.Name, func() string {
		if tag == "" {
			return "branch:" + repo.Branch
		}
		return tag
	}())
	repoDir := downloadSource(cfg.workDir, repo, tag)
	version := getVersion(repoDir, tag)
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
	if err := buildDebianPackage(cfg, stageDir, cfg.outDir, repo.Name, version, runtimeDeps); err != nil {
		log("Warning: Debian package build failed for %s: %v", repo.Name, err)
		return false
	}
	log("Successfully packaged %s at version %s", repo.Name, version)
	return true
}

func main() {
	cfg := &Config{}
	useTui := flag.Bool("tui", false, "Launch interactive TUI wizard")
	flag.StringVar(&cfg.globalTag, "tag", "", "Override tag for all repos (e.g. epoch-1.0.7). When omitted, per-repo tags from repos.json are used.")
	flag.StringVar(&cfg.reposFile, "repos", defaultReposFile, "Path to repos JSON config file")
	flag.BoolVar(&cfg.updateRepos, "update-repos", false, "Fetch latest epoch tags from upstream and generate repos.json, then exit")
	flag.BoolVar(&cfg.genConfig, "gen-config", false, "Generate repos.json from the current configuration without fetching updates, then exit")
	flag.BoolVar(&cfg.devFinder, "dev-finder", false, "Developer: Update finder.go from the loaded repos config, then exit")
	flag.StringVar(&cfg.workDir, "workdir", "cosmic-work", "Working directory for source checkout and compilation")
	flag.StringVar(&cfg.outDir, "outdir", outputPkgDir, "Output directory for produced .deb packages")
	flag.IntVar(&cfg.jobs, "jobs", runtime.NumCPU(), "Number of parallel compilation jobs")
	flag.BoolVar(&cfg.skipDeps, "skip-deps", false, "Skip automatic installation of build dependencies")
	flag.StringVar(&cfg.only, "only", "", "Restrict the build to a single named cosmic-* component")
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
		if cfg.globalTag == "" {
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

		if cfg.globalTag == "" && len(epochTags) > 1 {
			fmt.Println("Available epoch tags (from repos.json):")
			limit := len(epochTags)
			if limit > 10 {
				limit = 10
			}
			for i := 0; i < limit; i++ {
				fmt.Printf("  [%d] %s\n", i, epochTags[i])
			}
			fmt.Printf("  [*] Use per-repo tags from repos.json\n")
			fmt.Print("Select tag index (or press Enter to use per-repo tags): ")
			idxStr, _ := reader.ReadString('\n')
			idxStr = strings.TrimSpace(idxStr)
			if idxStr != "" {
				idx, err := strconv.Atoi(idxStr)
				if err == nil && idx >= 0 && idx < len(epochTags) {
					cfg.globalTag = epochTags[idx]
				}
			}
		}
	}

	if cfg.globalTag != "" {
		log("Global tag override: %s (applied to all repos)", cfg.globalTag)
	} else {
		log("Using per-repo tags from: %s", actualReposFile)
	}

	buildFunc := func() {
		if !cfg.skipDeps {
			installBuildDeps()
		} else {
			ensureCargoBinInPath()
		}
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

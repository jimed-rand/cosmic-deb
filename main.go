package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jimed-rand/cosmic-deb/pkg/build"
	"github.com/jimed-rand/cosmic-deb/pkg/debian"
	"github.com/jimed-rand/cosmic-deb/pkg/distro"
	"github.com/jimed-rand/cosmic-deb/pkg/repos"
	"github.com/jimed-rand/cosmic-deb/pkg/tui"
)

var (
	flagRepos       = flag.String("repos", "built-in", "Path to repos.json or 'built-in'")
	flagTag         = flag.String("tag", "", "Override epoch tag for all repositories")
	flagUseBranch   = flag.Bool("use-branch", false, "Build from main branch HEAD instead of epoch tag")
	flagWorkDir     = flag.String("workdir", "cosmic-work", "Staging directory for source code")
	flagOutDir      = flag.String("outdir", "cosmic-packages", "Output directory for .deb files")
	flagJobs        = flag.Int("jobs", 0, "Number of parallel compilation jobs (0 = nproc)")
	flagSkipDeps    = flag.Bool("skip-deps", false, "Skip build dependency installation")
	flagOnly        = flag.String("only", "", "Build a single named component")
	flagTUI         = flag.Bool("tui", false, "Launch the TUI configuration wizard")
	flagUpdateRepos = flag.Bool("update-repos", false, "Fetch latest epoch tags and overwrite repos config")
	flagGenConfig   = flag.Bool("gen-config", false, "Export built-in config to repos.json")
	flagDevFinder   = flag.Bool("dev-finder", false, "Regenerate pkg/repos/finder.go from active schema")
	flagVerbose     = flag.Bool("verbose", false, "Enable verbose build output")
)

func log(format string, args ...any) {
	ts := time.Now().Format("15:04:05")
	fmt.Printf("[%s] %s\n", ts, fmt.Sprintf(format, args...))
}

func logVerbose(verbose bool, format string, args ...any) {
	if verbose {
		ts := time.Now().Format("15:04:05")
		fmt.Printf("[%s] [VERBOSE] %s\n", ts, fmt.Sprintf(format, args...))
	}
}

func execShell(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func nproc() int {
	cmd := exec.Command("nproc")
	out, err := cmd.Output()
	if err != nil {
		return 4
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || n < 1 {
		return 4
	}
	return n
}

func main() {
	flag.Parse()

	verbose := *flagVerbose

	log("cosmic-deb starting up")
	logVerbose(verbose, "Parsed flags: repos=%s tag=%s use-branch=%v workdir=%s outdir=%s jobs=%d skip-deps=%v only=%s tui=%v verbose=%v",
		*flagRepos, *flagTag, *flagUseBranch, *flagWorkDir, *flagOutDir, *flagJobs, *flagSkipDeps, *flagOnly, *flagTUI, verbose)

	cfg, cfgPath := repos.Load(*flagRepos)
	if cfg == nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to load repos config from '%s'\n", *flagRepos)
		os.Exit(1)
	}
	log("Loaded repos config: %s (%d repositories)", cfgPath, len(cfg.Repos))

	if *flagGenConfig {
		logVerbose(verbose, "Exporting built-in config to repos.json")
		data, err := repos.MarshalConfig(repos.BuiltIn())
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile("repos.json", data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}
		log("Built-in config exported to repos.json")
		return
	}

	if *flagDevFinder {
		logVerbose(verbose, "Regenerating pkg/repos/finder.go")
		content, err := repos.GenerateFinderGo(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(filepath.Join("pkg", "repos", "finder.go"), []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}
		log("pkg/repos/finder.go regenerated")
		return
	}

	if *flagUpdateRepos {
		log("Updating repos config with latest epoch tags")
		repos.Update(*flagRepos, cfg, func(f string, a ...any) { log(f, a...) })
		return
	}

	di := distro.Detect()
	log("Detected distribution: %s %s", di.ID, di.Codename)

	if ok, reason := distro.CheckSupported(di.ID, di.Codename); !ok {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", reason)
		os.Exit(1)
	}

	jobs := *flagJobs
	if jobs <= 0 {
		jobs = nproc()
		logVerbose(verbose, "Detected %d logical CPUs via nproc", jobs)
	}

	workDir := *flagWorkDir
	outDir := *flagOutDir
	globalTag := *flagTag
	skipDeps := *flagSkipDeps
	onlyComp := *flagOnly

	if *flagTUI {
		log("Launching TUI wizard")
		epochTags := repos.EpochTags(cfg)
		mname, _ := repos.MaintainerFromUpstream()
		choices, confirmed, err := tui.RunWizard(di.ID, di.Codename, mname, epochTags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: TUI failed: %v\n", err)
			os.Exit(1)
		}
		if !confirmed {
			log("Build cancelled via TUI")
			return
		}
		if v, ok := choices["release"]; ok && v != "branch" {
			globalTag = v
		} else if v == "branch" {
			*flagUseBranch = true
		}
		if v, ok := choices["workdir"]; ok && v != "" {
			workDir = v
		}
		if v, ok := choices["outdir"]; ok && v != "" {
			outDir = v
		}
		if v, ok := choices["jobs"]; ok && v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				jobs = n
			}
		}
		if v, ok := choices["skip_deps"]; ok {
			skipDeps = v == "y"
		}
		if v, ok := choices["only"]; ok {
			onlyComp = v
		}
	} else if globalTag == "" && !*flagUseBranch {
		logVerbose(verbose, "No tag or branch flag specified; entering interactive source selection")
		globalTag = interactiveSelectTag(cfg, verbose)
	}

	if *flagUseBranch {
		globalTag = ""
		log("Source mode: main branch HEAD (latest commits)")
	} else if globalTag != "" {
		log("Source mode: epoch tag %s", globalTag)
	}

	logVerbose(verbose, "Resolved configuration: workdir=%s outdir=%s jobs=%d skip-deps=%v only=%s",
		workDir, outDir, jobs, skipDeps, onlyComp)

	if err := os.MkdirAll(workDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Cannot create workdir '%s': %v\n", workDir, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Cannot create outdir '%s': %v\n", outDir, err)
		os.Exit(1)
	}
	logVerbose(verbose, "Working directories created: %s, %s", workDir, outDir)

	if !skipDeps {
		log("Checking build dependencies")
		if !build.CheckAptBased() {
			fmt.Fprintf(os.Stderr, "ERROR: APT not found; only Debian and Ubuntu are supported\n")
			os.Exit(1)
		}
		allDeps := distro.CollectAllBuildDeps(di.ID, di.Codename)
		logVerbose(verbose, "Total build dependency list: %d packages", len(allDeps))
		missing := build.CheckPackagesInstalled(allDeps)
		if len(missing) > 0 {
			log("Installing %d missing packages: %s", len(missing), strings.Join(missing, ", "))
			if err := build.InstallPackages(missing, func(f string, a ...any) { log(f, a...) }); err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: Package installation failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			log("All build dependencies are satisfied")
		}
		if err := build.EnsureRustToolchain(workDir, func(f string, a ...any) { log(f, a...) }); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Rust toolchain setup failed: %v\n", err)
			os.Exit(1)
		}
		if err := build.EnsureJust(workDir, func(f string, a ...any) { log(f, a...) }); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: 'just' installation failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		log("Skipping dependency installation (-skip-deps)")
		build.ApplyIsolatedRustEnv(workDir)
	}

	defer func() {
		if !skipDeps {
			build.PurgeIsolatedRustEnv(workDir, func(f string, a ...any) { log(f, a...) })
		}
	}()

	targetRepos := cfg.Repos
	if onlyComp != "" {
		logVerbose(verbose, "Filtering to single component: %s", onlyComp)
		var filtered []repos.Entry
		for _, r := range cfg.Repos {
			if r.Name == onlyComp {
				filtered = append(filtered, r)
			}
		}
		if len(filtered) == 0 {
			fmt.Fprintf(os.Stderr, "ERROR: Component '%s' not found in repos config\n", onlyComp)
			os.Exit(1)
		}
		targetRepos = filtered
	}

	sort.Slice(targetRepos, func(i, j int) bool {
		return targetRepos[i].Name < targetRepos[j].Name
	})
	log("Build order (%d components): %s", len(targetRepos), func() string {
		var names []string
		for _, r := range targetRepos {
			names = append(names, r.Name)
		}
		return strings.Join(names, ", ")
	}())

	maintainerName, maintainerEmail := repos.MaintainerFromUpstream()
	log("Package maintainer: %s <%s>", maintainerName, maintainerEmail)

	var builtPkgs []string
	total := len(targetRepos)

	var monitorCh chan tui.ProgressMsg
	var logCh chan tui.LogMsg
	var doneCh chan tui.DoneMsg

	useTUIMonitor := *flagTUI
	if useTUIMonitor {
		monitorCh = make(chan tui.ProgressMsg, 32)
		logCh = make(chan tui.LogMsg, 256)
		doneCh = make(chan tui.DoneMsg, 1)
		go runMonitor(monitorCh, logCh, doneCh)
	}

	logFn := func(format string, args ...any) {
		log(format, args...)
		if useTUIMonitor {
			logCh <- tui.LogMsg(fmt.Sprintf(format, args...))
		}
	}

	var buildErr error
	for i, repo := range targetRepos {
		effectiveTag := repos.EffectiveTag(repo, globalTag)
		if *flagUseBranch {
			effectiveTag = ""
		}

		log("[%d/%d] Processing: %s (tag=%q)", i+1, total, repo.Name, effectiveTag)
		if useTUIMonitor {
			monitorCh <- tui.ProgressMsg{Step: i + 1, Total: total, Name: repo.Name}
		}

		repoDir := build.DownloadSource(workDir, repo, effectiveTag, logFn)
		stageDir := filepath.Join(workDir, repo.Name+"-stage")
		logVerbose(verbose, "Source directory: %s", repoDir)

		debianSubdir := filepath.Join(repoDir, "debian")
		if info, err := os.Stat(debianSubdir); err == nil && info.IsDir() {
			logVerbose(verbose, "Found debian/ subdirectory in %s; using dpkg-buildpackage path", repo.Name)
			if err := build.BuildWithDebianDir(repoDir, outDir, workDir, logFn); err != nil {
				log("WARNING: debian/ build failed for %s: %v; attempting manual path", repo.Name, err)
			} else {
				builtPkgs = append(builtPkgs, repo.Name)
				logVerbose(verbose, "Component %s built via debian/ path successfully", repo.Name)
				build.CleanSource(repoDir, stageDir, logFn)
				logVerbose(verbose, "Cleaned source and staging for %s", repo.Name)
				continue
			}
		}

		build.RunVendor(repoDir, workDir, logFn)

		if err := build.Compile(repoDir, repo.Name, workDir, jobs, logFn); err != nil {
			log("ERROR: Compilation failed for %s: %v", repo.Name, err)
			buildErr = err
			build.CleanSource(repoDir, stageDir, logFn)
			continue
		}
		logVerbose(verbose, "Compilation succeeded for %s", repo.Name)

		if !build.ValidateBuildOutput(repoDir) {
			log("WARNING: Build output validation failed for %s; skipping packaging", repo.Name)
			build.CleanSource(repoDir, stageDir, logFn)
			continue
		}
		logVerbose(verbose, "Build output validated for %s", repo.Name)

		version := build.GetVersion(repoDir, effectiveTag)
		logVerbose(verbose, "Resolved version for %s: %s", repo.Name, version)

		if err := os.MkdirAll(stageDir, 0755); err != nil {
			log("ERROR: Cannot create staging dir for %s: %v", repo.Name, err)
			build.CleanSource(repoDir, stageDir, logFn)
			continue
		}

		if err := build.InstallToStage(repoDir, stageDir, workDir); err != nil {
			log("WARNING: Staging install failed for %s: %v", repo.Name, err)
		}

		if debian.StagingHasContent(stageDir) {
			logVerbose(verbose, "Staging directory has content; building .deb for %s", repo.Name)
			if err := debian.BuildPackage(stageDir, outDir, repo.Name, version, di.Codename, maintainerName, maintainerEmail); err != nil {
				log("ERROR: .deb assembly failed for %s: %v", repo.Name, err)
				build.CleanSource(repoDir, stageDir, logFn)
				continue
			}
			builtPkgs = append(builtPkgs, repo.Name)
			log("[%d/%d] Packaged: %s %s~%s", i+1, total, repo.Name, version, di.Codename)
		} else {
			log("WARNING: Empty staging directory for %s; skipping .deb assembly", repo.Name)
		}

		build.CleanSource(repoDir, stageDir, logFn)
		logVerbose(verbose, "Cleaned source and staging for %s", repo.Name)
	}

	if len(builtPkgs) > 0 {
		metaVersion := "1.0.0"
		if globalTag != "" {
			metaVersion = strings.TrimPrefix(globalTag, "epoch-")
			metaVersion = strings.TrimPrefix(metaVersion, "v")
		}
		logVerbose(verbose, "Building cosmic-desktop meta-package (version=%s, deps=%d)", metaVersion, len(builtPkgs))
		if err := debian.BuildMetaPackage(workDir, outDir, metaVersion, di.Codename, maintainerName, maintainerEmail, builtPkgs); err != nil {
			log("WARNING: Meta-package assembly failed: %v", err)
		} else {
			log("Meta-package cosmic-desktop built successfully")
		}
	}

	log("Build summary: %d/%d components packaged successfully", len(builtPkgs), total)
	if len(builtPkgs) > 0 {
		log("Output directory: %s", outDir)
		logVerbose(verbose, "Built packages: %s", strings.Join(builtPkgs, ", "))
	}

	if buildErr != nil {
		log("Build completed with errors")
	}

	if !distro.IsContainer() && len(builtPkgs) > 0 {
		fmt.Printf("\nInstall the built packages now? [y/N] ")
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(answer) == "y" {
			log("Running install-local.sh")
			if err := execShell("bash", filepath.Join("scripts", "install-local.sh"), outDir); err != nil {
				log("WARNING: Installation script failed: %v", err)
			}
		}
	}

	if useTUIMonitor {
		doneCh <- tui.DoneMsg{Err: buildErr}
	}
}

func interactiveSelectTag(cfg *repos.Config, verbose bool) string {
	tags := repos.EpochTags(cfg)
	fmt.Println("Select build source:")
	fmt.Println("  0) Latest (main branch HEAD)")
	for i, t := range tags {
		fmt.Printf("  %d) %s\n", i+1, t)
	}
	fmt.Print("Enter choice [0]: ")
	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(input)
	if input == "" || input == "0" {
		logVerbose(verbose, "User selected: main branch HEAD")
		return ""
	}
	n, err := strconv.Atoi(input)
	if err != nil || n < 1 || n > len(tags) {
		logVerbose(verbose, "Invalid input '%s'; defaulting to main branch HEAD", input)
		return ""
	}
	logVerbose(verbose, "User selected tag: %s", tags[n-1])
	return tags[n-1]
}

func runMonitor(progress <-chan tui.ProgressMsg, logs <-chan tui.LogMsg, done <-chan tui.DoneMsg) {
	model := tui.MonitorModel{}
	p := tea.NewProgram(model, tea.WithAltScreen())
	go func() {
		for {
			select {
			case msg := <-progress:
				p.Send(msg)
			case msg := <-logs:
				p.Send(msg)
			case msg := <-done:
				p.Send(msg)
				return
			}
		}
	}()
	p.Run()
}

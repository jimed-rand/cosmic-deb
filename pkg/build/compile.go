package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const rustLinkerFlags = "-C link-arg=-fuse-ld=lld"

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

func hasJustfile(dir string) (bool, string) {
	lower := filepath.Join(dir, "justfile")
	upper := filepath.Join(dir, "Justfile")
	if _, err := os.Stat(lower); err == nil {
		return true, lower
	}
	if _, err := os.Stat(upper); err == nil {
		return true, upper
	}
	return false, ""
}

func RunVendor(repoDir string, logFn func(string, ...any)) {
	if ok, _ := hasJustfile(repoDir); ok {
		logFn("Running 'just vendor' for %s", filepath.Base(repoDir))
		cmd := exec.Command("just", "vendor")
		cmd.Dir = repoDir
		cmd.Env = os.Environ()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	}
}

func Compile(repoDir, repoName string, jobs int, logFn func(string, ...any)) error {
	logFn("Compiling component: %s", repoName)
	env := buildEnv()

	cargoToml := filepath.Join(repoDir, "Cargo.toml")
	if _, err := os.Stat(cargoToml); err == nil {
		sedCmd := exec.Command("sed", "-i", `s/lto = "fat"/lto = "thin"/`, "Cargo.toml")
		sedCmd.Dir = repoDir
		_ = sedCmd.Run()
	}

	hasJust, _ := hasJustfile(repoDir)
	makefile := filepath.Join(repoDir, "Makefile")

	if hasJust {
		vendorTar := filepath.Join(repoDir, "vendor.tar")
		if _, err := os.Stat(vendorTar); err == nil {
			return runWithEnv(repoDir, env, "just", "build-vendored")
		}
		if err := runWithEnv(repoDir, env, "just", "build-release", "--frozen"); err != nil {
			return runWithEnv(repoDir, env, "just", "build-release")
		}
		return nil
	}
	if _, err := os.Stat(makefile); err == nil {
		return runWithEnv(repoDir, env, "make",
			fmt.Sprintf("-j%d", jobs),
			"ARGS=--frozen --release",
		)
	}
	if _, err := os.Stat(cargoToml); err == nil {
		return runWithEnv(repoDir, env, "cargo", "build", "--release", "--frozen",
			fmt.Sprintf("--jobs=%d", jobs),
		)
	}
	return fmt.Errorf("No recognised build system in %s", repoDir)
}

func ValidateBuildOutput(repoDir string) bool {
	targetRelease := filepath.Join(repoDir, "target", "release")
	if _, err := os.Stat(targetRelease); err == nil {
		entries, _ := os.ReadDir(targetRelease)
		for _, e := range entries {
			if !e.IsDir() {
				info, err := e.Info()
				if err == nil && info.Mode()&0111 != 0 {
					return true
				}
			}
		}
	}
	buildDir := filepath.Join(repoDir, "build")
	if _, err := os.Stat(buildDir); err == nil {
		return true
	}
	return false
}

func InstallToStage(repoDir, stageDir string) error {
	hasJust, _ := hasJustfile(repoDir)
	makefile := filepath.Join(repoDir, "Makefile")

	if hasJust {
		return runCmd(repoDir, "just", "rootdir="+stageDir, "DESTDIR="+stageDir, "install")
	}
	if _, err := os.Stat(makefile); err == nil {
		return runCmd(repoDir, "make",
			"prefix=/usr",
			"libexecdir=/usr/lib",
			"DESTDIR="+stageDir,
			"install",
		)
	}
	return fmt.Errorf("No install target found in %s", repoDir)
}

func BuildWithDebianDir(repoDir, outDir string, logFn func(string, ...any)) error {
	logFn("Using debian/ directory for %s", filepath.Base(repoDir))
	RunVendor(repoDir, logFn)
	if err := runCmd(repoDir, "dpkg-buildpackage", "-us", "-uc", "-b"); err != nil {
		return err
	}
	parent := filepath.Dir(repoDir)
	files, err := os.ReadDir(parent)
	if err != nil {
		return err
	}
	for _, f := range files {
		if len(f.Name()) > 4 && f.Name()[len(f.Name())-4:] == ".deb" {
			oldPath := filepath.Join(parent, f.Name())
			newPath := filepath.Join(outDir, f.Name())
			if err := os.Rename(oldPath, newPath); err != nil {
				logFn("Warning: Failed to move .deb to output directory: %v", err)
			}
		}
	}
	return nil
}

func runWithEnv(dir string, extraEnv []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

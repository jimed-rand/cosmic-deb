package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func CargoBinDir(workDir string) string {
	if workDir != "" {
		return filepath.Join(workDir, ".cargo", "bin")
	}
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

func EnsureCargoBinInPath(workDir string) {
	binDir := CargoBinDir(workDir)
	current := os.Getenv("PATH")
	if !strings.Contains(current, binDir) {
		os.Setenv("PATH", binDir+":"+current)
	}
}

func ApplyIsolatedRustEnv(workDir string) {
	cargoHome := filepath.Join(workDir, ".cargo")
	rustupHome := filepath.Join(workDir, ".rustup")
	os.Setenv("CARGO_HOME", cargoHome)
	os.Setenv("RUSTUP_HOME", rustupHome)
	EnsureCargoBinInPath(workDir)
}

func PurgeIsolatedRustEnv(workDir string, logFn func(string, ...any)) {
	logFn("Purging isolated Rust environment in %s", workDir)
	os.RemoveAll(filepath.Join(workDir, ".cargo"))
	os.RemoveAll(filepath.Join(workDir, ".rustup"))
}

func CheckAptBased() bool {
	_, err := exec.LookPath("apt-get")
	return err == nil
}

func CheckPackagesInstalled(pkgs []string) (missing []string) {
	for _, pkg := range pkgs {
		cmd := exec.Command("dpkg-query", "-W", "-f=${Status}", pkg)
		out, err := cmd.Output()
		if err != nil || !strings.Contains(string(out), "Installed") {
			missing = append(missing, pkg)
		}
	}
	return missing
}

func InstallPackages(pkgs []string, logFn func(string, ...any)) error {
	args := append([]string{"install", "-y", "--no-install-recommends"}, pkgs...)
	executable := "apt-get"
	execArgs := args
	if os.Geteuid() != 0 {
		executable = "sudo"
		execArgs = append([]string{"apt-get"}, args...)
	}
	return runCmd("", executable, execArgs...)
}

func EnsureRustToolchain(workDir string, logFn func(string, ...any)) error {
	ApplyIsolatedRustEnv(workDir)
	if _, err := exec.LookPath("rustup"); err != nil {
		logFn("The rustup binary was not found in PATH; Installing via sh.rustup.rs")
		if err := runCmd("", "sh", "-c", "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --no-modify-path"); err != nil {
			return err
		}
		EnsureCargoBinInPath(workDir)
	}
	logFn("Configuring Rust stable toolchain via rustup")
	if err := runCmd("", "rustup", "default", "stable"); err != nil {
		return err
	}
	EnsureCargoBinInPath(workDir)
	return nil
}

func EnsureJust(workDir string, logFn func(string, ...any)) error {
	EnsureCargoBinInPath(workDir)
	if _, err := exec.LookPath("just"); err != nil {
		logFn("The 'just' binary was not found in PATH; installing via cargo")
		if err := runCmd("", "cargo", "install", "just"); err != nil {
			return err
		}
		EnsureCargoBinInPath(workDir)
	}
	return nil
}

func runCmd(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

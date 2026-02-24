package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func GetVersion(repoDir, fallbackTag string) string {
	if v := versionFromChangelog(repoDir); v != "" {
		return v
	}
	if v := versionFromCargoToml(repoDir); v != "" {
		return v
	}
	if fallbackTag != "" {
		v := strings.TrimPrefix(fallbackTag, "epoch-")
		v = strings.TrimPrefix(v, "v")
		return v
	}
	return "0.1.0"
}

func versionFromChangelog(repoDir string) string {
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

func versionFromCargoToml(repoDir string) string {
	cargoPath := filepath.Join(repoDir, "Cargo.toml")
	if _, err := os.Stat(cargoPath); err != nil {
		return ""
	}
	data, err := os.ReadFile(cargoPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "version") && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			v := strings.Trim(parts[1], ` "'`)
			if v != "" {
				return v
			}
		}
	}
	return ""
}

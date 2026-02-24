package distro

import (
	"os"
	"strings"
)

type Info struct {
	ID       string
	Codename string
}

func Detect() Info {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return Info{"unknown", "unknown"}
	}
	vals := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			vals[parts[0]] = strings.Trim(parts[1], `"`)
		}
	}
	return Info{
		ID:       vals["ID"],
		Codename: vals["VERSION_CODENAME"],
	}
}

var debianSupported = map[string]bool{
	"bookworm": true,
	"trixie":   true,
	"forky":    true,
	"sid":      true,
	"unstable": true,
	"testing":  true,
}

var ubuntuSupported = map[string]bool{
	"jammy":    true,
	"noble":    true,
	"resolute": true,
	"devel":    true,
}

func CheckSupported(id, codename string) (bool, string) {
	switch id {
	case "debian":
		if !debianSupported[codename] {
			return false, "Debian release '" + codename + "' is not supported. Minimum supported release is bookworm (12)."
		}
	case "ubuntu":
		if !ubuntuSupported[codename] {
			return false, "Ubuntu release '" + codename + "' is not supported. Only LTS and devel releases are supported (e.g., jammy, noble, or resolute)."
		}
	default:
		return false, "Distribution '" + id + "' is not supported. Only Debian and Ubuntu are supported."
	}
	return true, ""
}

func IsContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	data, _ := os.ReadFile("/proc/1/cgroup")
	if strings.Contains(string(data), "docker") || strings.Contains(string(data), "lxc") || strings.Contains(string(data), "containerd") {
		return true
	}
	data2, _ := os.ReadFile("/proc/self/mountinfo")
	if strings.Contains(string(data2), "/docker/") {
		return true
	}
	return false
}

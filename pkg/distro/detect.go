package distro

import (
	"os"
	"os/exec"
	"strings"
)

type Info struct {
	ID       string
	Codename string
}

func Detect() Info {
	info := Info{ID: "unknown", Codename: "unknown"}
	data, err := os.ReadFile("/etc/os-release")
	if err == nil {
		vals := make(map[string]string)
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				vals[parts[0]] = strings.Trim(parts[1], `"`)
			}
		}
		info.ID = vals["ID"]
		info.Codename = vals["VERSION_CODENAME"]
		if info.Codename == "" {
			info.Codename = vals["VERSION_ID"]
		}
	}
	if info.ID == "" || info.ID == "unknown" {
		if IsAptBased() {
			info.ID = "debian"
		}
	}
	if info.Codename == "" || info.Codename == "unknown" {
		if info.ID == "debian" || info.ID == "ubuntu" {
			info.Codename = "sid"
		}
	}
	return info
}

func IsAptBased() bool {
	_, errApt := exec.LookPath("apt")
	if errApt != nil {
		_, errApt = exec.LookPath("apt-get")
	}
	if errApt != nil {
		return false
	}
	_, errDpkg := exec.LookPath("dpkg")
	return errDpkg == nil
}

func CheckSupported(id, codename string) (bool, string) {
	if IsAptBased() {
		return true, ""
	}
	if id == "debian" || id == "ubuntu" {
		return true, ""
	}
	return false, "Distribution detection failed or unsupported. This program requires an APT-based system with 'dpkg' (Debian/Ubuntu style)."
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

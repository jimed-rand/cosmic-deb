package repos

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func Load(path string) (*Config, string) {
	if path == "built-in" {
		return BuiltIn(), "built-in"
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
		return nil, ""
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, foundPath
	}
	if len(cfg.Repos) == 0 {
		return nil, foundPath
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

func Update(path string, existing *Config, logFn func(string, ...any)) *Config {
	logFn("Fetching latest epoch tags from upstream repositories...")
	updated := &Config{
		GeneratedAt: time.Now().Format("2006-01-02"),
		Repos:       make([]Entry, 0, len(existing.Repos)),
	}
	latestEpoch := ""
	for _, repo := range existing.Repos {
		entry := Entry{
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
			logFn("  %-40s %s", repo.Name, tag)
		} else if repo.Branch != "" {
			entry.Tag = ""
			logFn("  %-40s (no epoch tag found; using branch: %s)", repo.Name, repo.Branch)
		} else {
			entry.Tag = repo.Tag
			logFn("  %-40s (unchanged: %s)", repo.Name, repo.Tag)
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
		logFn("ERROR: Failed to serialise repos config: %v", err)
		return updated
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		logFn("ERROR: Failed to write repos config to '%s': %v", path, err)
		return updated
	}
	logFn("Repos config written to: %s (epoch_latest: %s)", path, updated.EpochLatest)
	return updated
}

func EpochTags(cfg *Config) []string {
	var tags []string
	seen := make(map[string]bool)
	for _, r := range cfg.Repos {
		if r.Tag != "" && !seen[r.Tag] {
			seen[r.Tag] = true
			tags = append(tags, r.Tag)
		}
	}
	return tags
}

func EffectiveTag(repo Entry, globalTag string) string {
	if globalTag != "" {
		return globalTag
	}
	return repo.Tag
}

func DefaultBranch(repoURL string) string {
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

func GenerateFinderGo(cfg *Config) (string, error) {
	var sb strings.Builder
	sb.WriteString("package repos\n\nfunc BuiltIn() *Config {\n")
	sb.WriteString("\treturn &Config{\n")
	sb.WriteString(fmt.Sprintf("\t\tGeneratedAt: %q,\n", cfg.GeneratedAt))
	sb.WriteString(fmt.Sprintf("\t\tEpochLatest: %q,\n", cfg.EpochLatest))
	sb.WriteString("\t\tRepos: []Entry{\n")
	for _, r := range cfg.Repos {
		if r.Branch != "" {
			sb.WriteString(fmt.Sprintf("\t\t\t{Name: %q, URL: %q, Branch: %q},\n", r.Name, r.URL, r.Branch))
		} else {
			sb.WriteString(fmt.Sprintf("\t\t\t{Name: %q, URL: %q, Tag: %q},\n", r.Name, r.URL, r.Tag))
		}
	}
	sb.WriteString("\t\t},\n\t}\n}\n")
	return sb.String(), nil
}
func MarshalConfig(cfg *Config) ([]byte, error) {
	return json.MarshalIndent(cfg, "", "  ")
}

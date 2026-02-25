package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jimed-rand/cosmic-deb/pkg/repos"
)

func ArchiveURL(repo repos.Entry, tag string) string {
	if tag != "" {
		return fmt.Sprintf("%s/archive/refs/tags/%s.tar.gz", repo.URL, tag)
	}
	branch := repo.Branch
	if branch == "" {
		branch = repos.DefaultBranch(repo.URL)
	}
	return fmt.Sprintf("%s/archive/refs/heads/%s.tar.gz", repo.URL, branch)
}

func detectExtractedDir(workDir, tarPath string) (string, error) {
	cmd := exec.Command("tar", "-tzf", tarPath)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list archive contents: %v", err)
	}
	lines := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)
	if len(lines) == 0 || lines[0] == "" {
		return "", fmt.Errorf("archive appears to be empty")
	}
	topDir := strings.SplitN(lines[0], "/", 2)[0]
	return filepath.Join(workDir, topDir), nil
}

func GitClone(workDir string, repo repos.Entry, tag, dest string, logFn func(string, ...any)) string {
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
		branch := repos.DefaultBranch(repo.URL)
		logFn("Detected default branch for %s: %s", repo.Name, branch)
		args = append(args, "--branch", branch)
	}
	args = append(args, cloneURL, dest)
	logFn("Cloning %s from %s", repo.Name, cloneURL)
	cmd := exec.Command("git", args...)
	cmd.Dir = workDir
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logFn("ERROR: Failed to clone %s: %v", repo.Name, err)
		os.Exit(1)
	}
	return dest
}

func DownloadSource(workDir string, repo repos.Entry, tag string, logFn func(string, ...any)) string {
	dest := filepath.Join(workDir, repo.Name)
	if _, err := os.Stat(dest); err == nil {
		logFn("Source already present: %s", repo.Name)
		return dest
	}
	url := ArchiveURL(repo, tag)
	tarName := repo.Name + ".tar.gz"
	tarPath := filepath.Join(workDir, tarName)
	logFn("Downloading source archive: %s", repo.Name)
	dlCmd := exec.Command("curl", "-fSL", "-o", tarPath, url)
	dlCmd.Stdout = os.Stdout
	dlCmd.Stderr = os.Stderr
	if err := dlCmd.Run(); err != nil {
		_ = os.Remove(tarPath)
		logFn("Tarball download failed for %s; falling back to git clone", repo.Name)
		return GitClone(workDir, repo, tag, dest, logFn)
	}
	extractedDir, err := detectExtractedDir(workDir, tarPath)
	if err != nil {
		_ = os.Remove(tarPath)
		logFn("Failed to inspect archive for %s: %v; falling back to git clone", repo.Name, err)
		return GitClone(workDir, repo, tag, dest, logFn)
	}
	tarCmd := exec.Command("tar", "-xzf", tarName)
	tarCmd.Dir = workDir
	tarCmd.Stdout = os.Stdout
	tarCmd.Stderr = os.Stderr
	if err := tarCmd.Run(); err != nil {
		_ = os.Remove(tarPath)
		logFn("Failed to extract source for %s: %v; falling back to git clone", repo.Name, err)
		return GitClone(workDir, repo, tag, dest, logFn)
	}
	if _, err := os.Stat(extractedDir); err != nil {
		_ = os.Remove(tarPath)
		logFn("Expected extracted directory not found for %s; falling back to git clone", repo.Name)
		return GitClone(workDir, repo, tag, dest, logFn)
	}
	if extractedDir != dest {
		if err := os.Rename(extractedDir, dest); err != nil {
			logFn("ERROR: Failed to rename extracted directory for %s: %v", repo.Name, err)
			os.Exit(1)
		}
	}
	_ = os.Remove(tarPath)
	return dest
}

func CleanSource(repoDir, stageDir string, logFn func(string, ...any)) {
	logFn("Cleaning up source directory: %s", repoDir)
	os.RemoveAll(repoDir)
	logFn("Cleaning up staging directory: %s", stageDir)
	os.RemoveAll(stageDir)
}

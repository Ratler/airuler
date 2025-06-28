package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Repository struct {
	URL       string
	LocalPath string
}

func NewRepository(url, localPath string) *Repository {
	return &Repository{
		URL:       url,
		LocalPath: localPath,
	}
}

func (r *Repository) Clone() error {
	// Ensure parent directory exists
	parentDir := filepath.Dir(r.LocalPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Clone repository
	cmd := exec.Command("git", "clone", r.URL, r.LocalPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (r *Repository) Pull() error {
	if !r.Exists() {
		return fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}

	cmd := exec.Command("git", "-C", r.LocalPath, "pull")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull repository: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (r *Repository) GetCurrentCommit() (string, error) {
	if !r.Exists() {
		return "", fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}

	cmd := exec.Command("git", "-C", r.LocalPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current commit: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func (r *Repository) GetRemoteCommit() (string, error) {
	if !r.Exists() {
		return "", fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}

	// Fetch latest from remote
	cmd := exec.Command("git", "-C", r.LocalPath, "fetch")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to fetch from remote: %w", err)
	}

	// Get remote HEAD commit
	cmd = exec.Command("git", "-C", r.LocalPath, "rev-parse", "origin/HEAD")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to origin/main or origin/master
		cmd = exec.Command("git", "-C", r.LocalPath, "rev-parse", "origin/main")
		output, err = cmd.Output()
		if err != nil {
			cmd = exec.Command("git", "-C", r.LocalPath, "rev-parse", "origin/master")
			output, err = cmd.Output()
			if err != nil {
				return "", fmt.Errorf("failed to get remote commit: %w", err)
			}
		}
	}

	return strings.TrimSpace(string(output)), nil
}

func (r *Repository) HasUpdates() (bool, error) {
	current, err := r.GetCurrentCommit()
	if err != nil {
		return false, err
	}

	remote, err := r.GetRemoteCommit()
	if err != nil {
		return false, err
	}

	return current != remote, nil
}

func (r *Repository) Exists() bool {
	gitDir := filepath.Join(r.LocalPath, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}

func (r *Repository) Remove() error {
	return os.RemoveAll(r.LocalPath)
}

func (r *Repository) CheckoutCommit(commit string) error {
	if !r.Exists() {
		return fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}

	cmd := exec.Command("git", "-C", r.LocalPath, "checkout", commit)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to checkout commit %s: %w\nOutput: %s", commit, err, string(output))
	}

	return nil
}

func (r *Repository) CheckoutMainBranch() error {
	if !r.Exists() {
		return fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}

	// Try to checkout main branch, fallback to master if main doesn't exist
	cmd := exec.Command("git", "-C", r.LocalPath, "checkout", "main")
	err := cmd.Run()
	if err != nil {
		// Fallback to master
		cmd = exec.Command("git", "-C", r.LocalPath, "checkout", "master")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to checkout main/master branch: %w\nOutput: %s", err, string(output))
		}
	}

	return nil
}

func (r *Repository) ResetToCommit(commit string) error {
	if !r.Exists() {
		return fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}

	cmd := exec.Command("git", "-C", r.LocalPath, "reset", "--hard", commit)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reset to commit %s: %w\nOutput: %s", commit, err, string(output))
	}

	return nil
}

func URLToDirectoryName(url string) string {
	// Convert git URL to directory name
	// https://github.com/user/repo -> github.com-user-repo
	// git@github.com:user/repo.git -> github.com-user-repo

	url = strings.TrimSuffix(url, ".git")

	if strings.HasPrefix(url, "git@") {
		// git@github.com:user/repo -> github.com/user/repo
		parts := strings.SplitN(url, ":", 2)
		if len(parts) == 2 {
			host := strings.TrimPrefix(parts[0], "git@")
			url = "https://" + host + "/" + parts[1]
		}
	}

	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		url = strings.TrimPrefix(url, "https://")
		url = strings.TrimPrefix(url, "http://")
	}

	// Replace / and . with -
	url = strings.ReplaceAll(url, "/", "-")
	url = strings.ReplaceAll(url, ".", "-")

	return url
}

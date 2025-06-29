// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
)

// GoGitRepository implements Repository interface using go-git library
type GoGitRepository struct {
	URL       string
	LocalPath string
}

// GoGitRepositoryFactory creates repositories using go-git library
type GoGitRepositoryFactory struct{}

// NewGoGitRepositoryFactory creates a new factory for go-git based operations
func NewGoGitRepositoryFactory() *GoGitRepositoryFactory {
	return &GoGitRepositoryFactory{}
}

// NewRepository creates a new repository instance using go-git
func (f *GoGitRepositoryFactory) NewRepository(url, localPath string) Repository {
	return &GoGitRepository{
		URL:       url,
		LocalPath: localPath,
	}
}

// Clone clones the repository to the local path
func (r *GoGitRepository) Clone() error {
	// Ensure parent directory exists
	parentDir := filepath.Dir(r.LocalPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Clone repository using go-git
	_, err := gogit.PlainClone(r.LocalPath, &gogit.CloneOptions{
		URL: r.URL,
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// Pull updates the repository from remote
func (r *GoGitRepository) Pull() error {
	if !r.Exists() {
		return fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}

	// Open existing repository
	repo, err := gogit.PlainOpen(r.LocalPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Get working tree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Pull changes
	err = worktree.Pull(&gogit.PullOptions{})
	if err != nil && err != gogit.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull repository: %w", err)
	}

	return nil
}

// GetCurrentCommit returns the current commit hash
func (r *GoGitRepository) GetCurrentCommit() (string, error) {
	if !r.Exists() {
		return "", fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}

	// Open repository
	repo, err := gogit.PlainOpen(r.LocalPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	// Get HEAD reference
	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	return head.Hash().String(), nil
}

// GetRemoteCommit returns the remote commit hash
func (r *GoGitRepository) GetRemoteCommit() (string, error) {
	if !r.Exists() {
		return "", fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}

	// Open repository
	repo, err := gogit.PlainOpen(r.LocalPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	// Fetch latest from remote
	err = repo.Fetch(&gogit.FetchOptions{})
	if err != nil && err != gogit.NoErrAlreadyUpToDate {
		return "", fmt.Errorf("failed to fetch from remote: %w", err)
	}

	// Try to get remote HEAD, with fallbacks to main/master
	remoteRefs := []string{"origin/HEAD", "origin/main", "origin/master"}

	for _, remoteRef := range remoteRefs {
		ref, err := repo.Reference(plumbing.NewRemoteReferenceName("origin", strings.TrimPrefix(remoteRef, "origin/")), true)
		if err == nil {
			return ref.Hash().String(), nil
		}
	}

	return "", fmt.Errorf("failed to get remote commit: no valid remote reference found")
}

// HasUpdates checks if there are updates available from remote
func (r *GoGitRepository) HasUpdates() (bool, error) {
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

// Exists checks if the repository exists locally
func (r *GoGitRepository) Exists() bool {
	gitDir := filepath.Join(r.LocalPath, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}

// Remove removes the local repository directory
func (r *GoGitRepository) Remove() error {
	return os.RemoveAll(r.LocalPath)
}

// CheckoutCommit checks out a specific commit
func (r *GoGitRepository) CheckoutCommit(commit string) error {
	if !r.Exists() {
		return fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}

	// Open repository
	repo, err := gogit.PlainOpen(r.LocalPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Get working tree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Parse commit hash
	hash := plumbing.NewHash(commit)

	// Checkout specific commit
	err = worktree.Checkout(&gogit.CheckoutOptions{
		Hash: hash,
	})
	if err != nil {
		return fmt.Errorf("failed to checkout commit %s: %w", commit, err)
	}

	return nil
}

// CheckoutMainBranch checks out the main/master branch
func (r *GoGitRepository) CheckoutMainBranch() error {
	if !r.Exists() {
		return fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}

	// Open repository
	repo, err := gogit.PlainOpen(r.LocalPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Get working tree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Try to checkout main branch, fallback to master if main doesn't exist
	branches := []string{"main", "master"}

	for _, branch := range branches {
		err = worktree.Checkout(&gogit.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(branch),
		})
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("failed to checkout main/master branch: %w", err)
}

// ResetToCommit resets the repository to a specific commit
func (r *GoGitRepository) ResetToCommit(commit string) error {
	if !r.Exists() {
		return fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}

	// Open repository
	repo, err := gogit.PlainOpen(r.LocalPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Get working tree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Parse commit hash
	hash := plumbing.NewHash(commit)

	// Reset to specific commit (hard reset)
	err = worktree.Reset(&gogit.ResetOptions{
		Commit: hash,
		Mode:   gogit.HardReset,
	})
	if err != nil {
		return fmt.Errorf("failed to reset to commit %s: %w", commit, err)
	}

	return nil
}

// URLToDirectoryName converts a git URL to a directory name
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

// Ensure GoGitRepository implements Repository interface
var _ Repository = (*GoGitRepository)(nil)
var _ RepositoryFactory = (*GoGitRepositoryFactory)(nil)

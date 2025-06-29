package git

import (
	"fmt"
	"os"
	"path/filepath"
)

// MockRepository implements Repository interface for testing
type MockRepository struct {
	URL       string
	LocalPath string

	// Test configuration
	ShouldFailClone      bool
	ShouldFailPull       bool
	ShouldFailCommits    bool
	ShouldExist          bool
	MockCurrentCommit    string
	MockRemoteCommit     string
	CloneCalled          bool
	PullCalled           bool
	RemoveCalled         bool
	CheckoutCommitCalled bool
	ResetCalled          bool
}

// MockRepositoryFactory creates mock repositories for testing
type MockRepositoryFactory struct {
	Repositories map[string]*MockRepository
}

// NewMockGitRepositoryFactory creates a new mock factory
func NewMockGitRepositoryFactory() *MockRepositoryFactory {
	return &MockRepositoryFactory{
		Repositories: make(map[string]*MockRepository),
	}
}

// NewRepository creates a new mock repository instance
func (f *MockRepositoryFactory) NewRepository(url, localPath string) Repository {
	key := fmt.Sprintf("%s:%s", url, localPath)
	if repo, exists := f.Repositories[key]; exists {
		return repo
	}

	// Create new mock repository with default settings
	repo := &MockRepository{
		URL:               url,
		LocalPath:         localPath,
		MockCurrentCommit: "abc123def456",
		MockRemoteCommit:  "def456abc123",
	}
	f.Repositories[key] = repo
	return repo
}

// ConfigureRepository allows test setup of mock behavior
func (f *MockRepositoryFactory) ConfigureRepository(url, localPath string, config func(*MockRepository)) {
	key := fmt.Sprintf("%s:%s", url, localPath)
	repo := &MockRepository{
		URL:               url,
		LocalPath:         localPath,
		MockCurrentCommit: "abc123def456",
		MockRemoteCommit:  "def456abc123",
	}
	config(repo)
	f.Repositories[key] = repo
}

// Clone implementation for mock
func (r *MockRepository) Clone() error {
	r.CloneCalled = true
	if r.ShouldFailClone {
		return fmt.Errorf("mock clone failed for %s", r.URL)
	}

	// Create directory structure to simulate successful clone
	if err := os.MkdirAll(filepath.Join(r.LocalPath, ".git"), 0755); err != nil {
		return err
	}

	return nil
}

// Pull implementation for mock
func (r *MockRepository) Pull() error {
	r.PullCalled = true
	if r.ShouldFailPull {
		return fmt.Errorf("mock pull failed")
	}
	if !r.Exists() {
		return fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}
	return nil
}

// GetCurrentCommit implementation for mock
func (r *MockRepository) GetCurrentCommit() (string, error) {
	if r.ShouldFailCommits {
		return "", fmt.Errorf("mock commit fetch failed")
	}
	if !r.Exists() {
		return "", fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}
	return r.MockCurrentCommit, nil
}

// GetRemoteCommit implementation for mock
func (r *MockRepository) GetRemoteCommit() (string, error) {
	if r.ShouldFailCommits {
		return "", fmt.Errorf("mock remote commit fetch failed")
	}
	if !r.Exists() {
		return "", fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}
	return r.MockRemoteCommit, nil
}

// HasUpdates implementation for mock
func (r *MockRepository) HasUpdates() (bool, error) {
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

// Exists implementation for mock
func (r *MockRepository) Exists() bool {
	if r.ShouldExist {
		return true
	}

	// Check if .git directory exists (same as real implementation)
	gitDir := filepath.Join(r.LocalPath, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}

// Remove implementation for mock
func (r *MockRepository) Remove() error {
	r.RemoveCalled = true
	return os.RemoveAll(r.LocalPath)
}

// CheckoutCommit implementation for mock
func (r *MockRepository) CheckoutCommit(_ string) error {
	r.CheckoutCommitCalled = true
	if !r.Exists() {
		return fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}
	return nil
}

// CheckoutMainBranch implementation for mock
func (r *MockRepository) CheckoutMainBranch() error {
	if !r.Exists() {
		return fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}
	return nil
}

// ResetToCommit implementation for mock
func (r *MockRepository) ResetToCommit(_ string) error {
	r.ResetCalled = true
	if !r.Exists() {
		return fmt.Errorf("repository does not exist at %s", r.LocalPath)
	}
	return nil
}

// Ensure MockRepository implements Repository interface
var _ Repository = (*MockRepository)(nil)
var _ RepositoryFactory = (*MockRepositoryFactory)(nil)

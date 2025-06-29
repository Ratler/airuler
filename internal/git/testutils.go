// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestRepository represents a test git repository for testing purposes
type TestRepository struct {
	Path   string
	Remote string
	t      *testing.T
}

// CreateTestRepository creates a minimal git repository for testing
func CreateTestRepository(t *testing.T) *TestRepository {
	t.Helper()

	if !isGitAvailable() {
		t.Skip("git is not available, skipping test repository creation")
	}

	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	// Create directory
	err := os.MkdirAll(repoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository directory: %v", err)
	}

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user name: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(repoPath, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repository\n"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add test file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	return &TestRepository{
		Path: repoPath,
		t:    t,
	}
}

// CreateTestRepositoryWithRemote creates a test repository with a bare remote
func CreateTestRepositoryWithRemote(t *testing.T) *TestRepository {
	t.Helper()

	if !isGitAvailable() {
		t.Skip("git is not available, skipping test repository creation")
	}

	tempDir := t.TempDir()

	// Create bare remote repository
	remotePath := filepath.Join(tempDir, "remote.git")
	err := os.MkdirAll(remotePath, 0755)
	if err != nil {
		t.Fatalf("Failed to create remote repository directory: %v", err)
	}

	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remotePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize bare repository: %v", err)
	}

	// Create local repository
	localPath := filepath.Join(tempDir, "local-repo")
	cmd = exec.Command("git", "clone", remotePath, localPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user name: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(localPath, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repository\n"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add test file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	cmd = exec.Command("git", "push", "origin", "main")
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		// Try master if main fails
		cmd = exec.Command("git", "push", "origin", "master")
		cmd.Dir = localPath
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to push initial commit: %v", err)
		}
	}

	return &TestRepository{
		Path:   localPath,
		Remote: remotePath,
		t:      t,
	}
}

// AddCommit adds a new commit to the test repository
func (tr *TestRepository) AddCommit(message string) string {
	tr.t.Helper()

	// Create a new test file
	testFile := filepath.Join(tr.Path, "file-"+message+".txt")
	err := os.WriteFile(testFile, []byte("Content for "+message), 0600)
	if err != nil {
		tr.t.Fatalf("Failed to create test file: %v", err)
	}

	// Add and commit the file
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = tr.Path
	if err := cmd.Run(); err != nil {
		tr.t.Fatalf("Failed to add files: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = tr.Path
	if err := cmd.Run(); err != nil {
		tr.t.Fatalf("Failed to commit: %v", err)
	}

	// Get the commit hash
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tr.Path
	output, err := cmd.Output()
	if err != nil {
		tr.t.Fatalf("Failed to get commit hash: %v", err)
	}

	return string(output[:7]) // Return short hash
}

// PushToRemote pushes changes to the remote repository
func (tr *TestRepository) PushToRemote() {
	tr.t.Helper()

	if tr.Remote == "" {
		tr.t.Fatal("No remote configured for test repository")
	}

	cmd := exec.Command("git", "push", "origin", "main")
	cmd.Dir = tr.Path
	if err := cmd.Run(); err != nil {
		// Try master if main fails
		cmd = exec.Command("git", "push", "origin", "master")
		cmd.Dir = tr.Path
		if err := cmd.Run(); err != nil {
			tr.t.Fatalf("Failed to push to remote: %v", err)
		}
	}
}

// GetCurrentCommit returns the current commit hash
func (tr *TestRepository) GetCurrentCommit() string {
	tr.t.Helper()

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tr.Path
	output, err := cmd.Output()
	if err != nil {
		tr.t.Fatalf("Failed to get current commit: %v", err)
	}

	return string(output[:40]) // Return full hash
}

// CreateBranch creates a new branch in the test repository
func (tr *TestRepository) CreateBranch(branchName string) {
	tr.t.Helper()

	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = tr.Path
	if err := cmd.Run(); err != nil {
		tr.t.Fatalf("Failed to create branch %s: %v", branchName, err)
	}
}

// CheckoutBranch checks out an existing branch
func (tr *TestRepository) CheckoutBranch(branchName string) {
	tr.t.Helper()

	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = tr.Path
	if err := cmd.Run(); err != nil {
		tr.t.Fatalf("Failed to checkout branch %s: %v", branchName, err)
	}
}

// MockGitCommand represents a mock git command for testing without actual git
type MockGitCommand struct {
	Commands []MockCommand
	Index    int
}

// MockCommand represents an expected git command and its response
type MockCommand struct {
	Args     []string
	Output   string
	Error    error
	ExitCode int
}

// NewMockGitCommand creates a new mock git command handler
func NewMockGitCommand() *MockGitCommand {
	return &MockGitCommand{
		Commands: make([]MockCommand, 0),
		Index:    0,
	}
}

// ExpectCommand adds an expected command to the mock
func (m *MockGitCommand) ExpectCommand(args []string, output string, err error) *MockGitCommand {
	m.Commands = append(m.Commands, MockCommand{
		Args:   args,
		Output: output,
		Error:  err,
	})
	return m
}

// VerifyAllCommandsCalled checks that all expected commands were called
func (m *MockGitCommand) VerifyAllCommandsCalled(t *testing.T) {
	t.Helper()
	if m.Index != len(m.Commands) {
		t.Errorf("Expected %d commands to be called, but only %d were called", len(m.Commands), m.Index)
	}
}

// isGitAvailable checks if git is available on the system
func isGitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

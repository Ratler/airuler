// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestRepository creates a new repository for testing using the go-git implementation
func newTestRepository(url, localPath string) Repository {
	factory := NewGoGitRepositoryFactory()
	return factory.NewRepository(url, localPath)
}

func TestNewRepository(t *testing.T) {
	url := "https://github.com/user/repo"
	localPath := "/tmp/test-repo"

	repo := newTestRepository(url, localPath)

	if repo == nil {
		t.Fatal("NewRepository() returned nil")
	}

	// Test that it implements the interface correctly
	// We can verify the interface implementation at compile time
	if repo == nil {
		t.Error("Repository is nil")
	}
}

func TestRepository_Exists(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		setup    func(string)
		expected bool
	}{
		{
			name:     "repository exists with .git directory",
			setup:    func(path string) { os.MkdirAll(filepath.Join(path, ".git"), 0755) },
			expected: true,
		},
		{
			name:     "repository does not exist",
			setup:    func(_ string) {}, // no setup
			expected: false,
		},
		{
			name:     "directory exists but no .git",
			setup:    func(path string) { os.MkdirAll(path, 0755) },
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath := filepath.Join(tempDir, tt.name)
			tt.setup(repoPath)

			repo := newTestRepository("https://github.com/test/repo", repoPath)
			result := repo.Exists()

			if result != tt.expected {
				t.Errorf("Repository.Exists() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestRepository_Remove(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	// Create a directory structure to remove
	err := os.MkdirAll(filepath.Join(repoPath, "subdir"), 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	repo := newTestRepository("https://github.com/test/repo", repoPath)

	// Verify directory exists before removal
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		t.Fatal("Test directory should exist before removal")
	}

	// Remove the repository
	err = repo.Remove()
	if err != nil {
		t.Errorf("Repository.Remove() failed: %v", err)
	}

	// Verify directory no longer exists
	if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
		t.Error("Repository directory should not exist after removal")
	}
}

func TestURLToDirectoryName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "HTTPS URL with .git suffix",
			url:      "https://github.com/user/repo.git",
			expected: "github-com-user-repo",
		},
		{
			name:     "HTTPS URL without .git suffix",
			url:      "https://github.com/user/repo",
			expected: "github-com-user-repo",
		},
		{
			name:     "SSH URL with .git suffix",
			url:      "git@github.com:user/repo.git",
			expected: "github-com-user-repo",
		},
		{
			name:     "SSH URL without .git suffix",
			url:      "git@github.com:user/repo",
			expected: "github-com-user-repo",
		},
		{
			name:     "HTTP URL",
			url:      "http://gitlab.com/user/repo",
			expected: "gitlab-com-user-repo",
		},
		{
			name:     "URL with subdomain",
			url:      "https://git.example.com/user/repo",
			expected: "git-example-com-user-repo",
		},
		{
			name:     "URL with organization/group",
			url:      "https://github.com/org/group/repo",
			expected: "github-com-org-group-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := URLToDirectoryName(tt.url)
			if result != tt.expected {
				t.Errorf("URLToDirectoryName(%s) = %s, expected %s", tt.url, result, tt.expected)
			}
		})
	}
}

// TestRepository_Clone tests cloning operations that require actual git
func TestRepository_Clone(t *testing.T) {
	if !isGitAvailable() {
		t.Skip("git is not available, skipping clone tests")
	}

	tempDir := t.TempDir()

	tests := []struct {
		name      string
		url       string
		expectErr bool
	}{
		{
			name:      "invalid URL",
			url:       "https://github.com/nonexistent/nonexistent-repo-12345",
			expectErr: true,
		},
		{
			name:      "invalid git URL format",
			url:       "not-a-url",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRepoPath := filepath.Join(tempDir, tt.name)
			repo := newTestRepository(tt.url, testRepoPath)

			err := repo.Clone()

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestRepository_Pull tests pull operations that require existing repositories
func TestRepository_Pull(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		setup     func(string)
		expectErr bool
	}{
		{
			name:      "repository does not exist",
			setup:     func(_ string) {}, // no setup
			expectErr: true,
		},
		{
			name: "directory exists but no .git",
			setup: func(path string) {
				os.MkdirAll(path, 0755)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRepoPath := filepath.Join(tempDir, tt.name)
			tt.setup(testRepoPath)

			repo := newTestRepository("https://github.com/test/repo", testRepoPath)
			err := repo.Pull()

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectErr && err != nil {
				expectedMsg := "repository does not exist"
				if !strings.Contains(err.Error(), expectedMsg) {
					t.Errorf("Expected error to contain %q, got %q", expectedMsg, err.Error())
				}
			}
		})
	}
}

// TestRepository_GetCurrentCommit tests getting current commit from repositories
func TestRepository_GetCurrentCommit(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		setup     func(string)
		expectErr bool
	}{
		{
			name:      "repository does not exist",
			setup:     func(_ string) {}, // no setup
			expectErr: true,
		},
		{
			name: "directory exists but no .git",
			setup: func(path string) {
				os.MkdirAll(path, 0755)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRepoPath := filepath.Join(tempDir, tt.name)
			tt.setup(testRepoPath)

			repo := newTestRepository("https://github.com/test/repo", testRepoPath)
			commit, err := repo.GetCurrentCommit()

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectErr && err != nil {
				expectedMsg := "repository does not exist"
				if !strings.Contains(err.Error(), expectedMsg) {
					t.Errorf("Expected error to contain %q, got %q", expectedMsg, err.Error())
				}
			}

			if !tt.expectErr && commit == "" {
				t.Error("Expected non-empty commit hash")
			}
		})
	}
}

// TestRepository_GetRemoteCommit tests getting remote commit from repositories
func TestRepository_GetRemoteCommit(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		setup     func(string)
		expectErr bool
	}{
		{
			name:      "repository does not exist",
			setup:     func(_ string) {}, // no setup
			expectErr: true,
		},
		{
			name: "directory exists but no .git",
			setup: func(path string) {
				os.MkdirAll(path, 0755)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRepoPath := filepath.Join(tempDir, tt.name)
			tt.setup(testRepoPath)

			repo := newTestRepository("https://github.com/test/repo", testRepoPath)
			commit, err := repo.GetRemoteCommit()

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectErr && err != nil {
				expectedMsg := "repository does not exist"
				if !strings.Contains(err.Error(), expectedMsg) {
					t.Errorf("Expected error to contain %q, got %q", expectedMsg, err.Error())
				}
			}

			if !tt.expectErr && commit == "" {
				t.Error("Expected non-empty commit hash")
			}
		})
	}
}

// TestRepository_HasUpdates tests checking for updates
func TestRepository_HasUpdates(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	repo := newTestRepository("https://github.com/test/repo", repoPath)

	// Test with non-existing repository
	hasUpdates, err := repo.HasUpdates()
	if err == nil {
		t.Error("Expected error for non-existing repository")
	}

	if hasUpdates {
		t.Error("Expected false for non-existing repository")
	}

	expectedMsg := "repository does not exist"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error to contain %q, got %q", expectedMsg, err.Error())
	}
}

// TestRepository_CheckoutCommit tests checking out specific commits
func TestRepository_CheckoutCommit(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		commit    string
		setup     func(string)
		expectErr bool
	}{
		{
			name:      "repository does not exist",
			commit:    "abc123",
			setup:     func(_ string) {}, // no setup
			expectErr: true,
		},
		{
			name:   "directory exists but no .git",
			commit: "abc123",
			setup: func(path string) {
				os.MkdirAll(path, 0755)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRepoPath := filepath.Join(tempDir, tt.name)
			tt.setup(testRepoPath)

			repo := newTestRepository("https://github.com/test/repo", testRepoPath)
			err := repo.CheckoutCommit(tt.commit)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectErr && err != nil {
				expectedMsg := "repository does not exist"
				if !strings.Contains(err.Error(), expectedMsg) {
					t.Errorf("Expected error to contain %q, got %q", expectedMsg, err.Error())
				}
			}
		})
	}
}

// TestRepository_CheckoutMainBranch tests checking out main/master branch
func TestRepository_CheckoutMainBranch(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		setup     func(string)
		expectErr bool
	}{
		{
			name:      "repository does not exist",
			setup:     func(_ string) {}, // no setup
			expectErr: true,
		},
		{
			name: "directory exists but no .git",
			setup: func(path string) {
				os.MkdirAll(path, 0755)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRepoPath := filepath.Join(tempDir, tt.name)
			tt.setup(testRepoPath)

			repo := newTestRepository("https://github.com/test/repo", testRepoPath)
			err := repo.CheckoutMainBranch()

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectErr && err != nil {
				expectedMsg := "repository does not exist"
				if !strings.Contains(err.Error(), expectedMsg) {
					t.Errorf("Expected error to contain %q, got %q", expectedMsg, err.Error())
				}
			}
		})
	}
}

// TestRepository_ResetToCommit tests resetting to specific commits
func TestRepository_ResetToCommit(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		commit    string
		setup     func(string)
		expectErr bool
	}{
		{
			name:      "repository does not exist",
			commit:    "abc123",
			setup:     func(_ string) {}, // no setup
			expectErr: true,
		},
		{
			name:   "directory exists but no .git",
			commit: "abc123",
			setup: func(path string) {
				os.MkdirAll(path, 0755)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRepoPath := filepath.Join(tempDir, tt.name)
			tt.setup(testRepoPath)

			repo := newTestRepository("https://github.com/test/repo", testRepoPath)
			err := repo.ResetToCommit(tt.commit)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectErr && err != nil {
				expectedMsg := "repository does not exist"
				if !strings.Contains(err.Error(), expectedMsg) {
					t.Errorf("Expected error to contain %q, got %q", expectedMsg, err.Error())
				}
			}
		})
	}
}

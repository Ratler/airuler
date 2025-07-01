// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package vendor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ratler/airuler/internal/config"
	"github.com/ratler/airuler/internal/git"
)

func TestNewManager(t *testing.T) {
	cfg := config.NewDefaultConfig()
	manager := NewManager(cfg)

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.config != cfg {
		t.Error("Manager config not set correctly")
	}

	if manager.lockFile == nil {
		t.Error("Manager lockFile not initialized")
	}

	if manager.lockFile.Vendors == nil {
		t.Error("Manager lockFile.Vendors not initialized")
	}
}

func TestManager_LoadLockFile(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	cfg := config.NewDefaultConfig()
	manager := NewManager(cfg)

	t.Run("no lock file exists", func(t *testing.T) {
		err := manager.LoadLockFile()
		if err != nil {
			t.Errorf("LoadLockFile() failed when no lock file exists: %v", err)
		}
	})

	t.Run("load existing lock file", func(t *testing.T) {
		// Create a test lock file
		lockContent := `vendors:
  test-vendor:
    url: "https://github.com/user/repo"
    commit: "abc123def456"
    fetchedat: "2023-01-01T00:00:00Z"
`
		err := os.WriteFile("airuler.lock", []byte(lockContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test lock file: %v", err)
		}

		err = manager.LoadLockFile()
		if err != nil {
			t.Errorf("LoadLockFile() failed: %v", err)
		}

		if len(manager.lockFile.Vendors) != 1 {
			t.Errorf("Expected 1 vendor, got %d", len(manager.lockFile.Vendors))
		}

		vendor, exists := manager.lockFile.Vendors["test-vendor"]
		if !exists {
			t.Error("Expected test-vendor to exist in lock file")
		}

		if vendor.URL != "https://github.com/user/repo" {
			t.Errorf("Expected URL https://github.com/user/repo, got %s", vendor.URL)
		}

		if vendor.Commit != "abc123def456" {
			t.Errorf("Expected commit abc123def456, got %s", vendor.Commit)
		}
	})

	t.Run("invalid lock file format", func(t *testing.T) {
		// Create invalid YAML
		err := os.WriteFile("airuler.lock", []byte("invalid: yaml: content:"), 0644)
		if err != nil {
			t.Fatalf("Failed to create invalid lock file: %v", err)
		}

		err = manager.LoadLockFile()
		if err == nil {
			t.Error("Expected error for invalid YAML")
		}
	})
}

func TestManager_SaveLockFile(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	cfg := config.NewDefaultConfig()
	manager := NewManager(cfg)

	// Add test vendor to lock file
	testTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	manager.lockFile.Vendors["test-vendor"] = config.VendorLock{
		URL:       "https://github.com/user/repo",
		Commit:    "abc123def456",
		FetchedAt: testTime,
	}

	err = manager.SaveLockFile()
	if err != nil {
		t.Errorf("SaveLockFile() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat("airuler.lock"); os.IsNotExist(err) {
		t.Error("Lock file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile("airuler.lock")
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	contentStr := string(content)
	expectedParts := []string{
		"vendors:",
		"test-vendor:",
		"url: https://github.com/user/repo",
		"commit: abc123def456",
	}

	for _, part := range expectedParts {
		if !strings.Contains(contentStr, part) {
			t.Errorf("Lock file content missing expected part: %s", part)
		}
	}
}

func TestManager_List(t *testing.T) {
	cfg := config.NewDefaultConfig()
	manager := NewManager(cfg)

	t.Run("no vendors", func(t *testing.T) {
		err := manager.List()
		if err != nil {
			t.Errorf("List() failed with no vendors: %v", err)
		}
	})

	t.Run("with vendors", func(t *testing.T) {
		testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		manager.lockFile.Vendors["test-vendor"] = config.VendorLock{
			URL:       "https://github.com/user/repo",
			Commit:    "abc123def456",
			FetchedAt: testTime,
		}

		err := manager.List()
		if err != nil {
			t.Errorf("List() failed with vendors: %v", err)
		}
	})
}

func TestManager_Remove(t *testing.T) {
	if !isGitAvailable() {
		t.Skip("git is not available, skipping vendor remove tests")
	}

	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	cfg := config.NewDefaultConfig()
	manager := NewManager(cfg)

	t.Run("vendor not found", func(t *testing.T) {
		err := manager.Remove("nonexistent-vendor")
		if err == nil {
			t.Error("Expected error for nonexistent vendor")
		}

		expectedMsg := "vendor nonexistent-vendor not found"
		if !strings.Contains(err.Error(), expectedMsg) {
			t.Errorf("Expected error to contain %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("remove existing vendor", func(t *testing.T) {
		// Add vendor to lock file
		manager.lockFile.Vendors["test-vendor"] = config.VendorLock{
			URL:       "https://github.com/user/repo",
			Commit:    "abc123",
			FetchedAt: time.Now(),
		}

		// Create vendor directory
		vendorPath := filepath.Join("vendors", "test-vendor")
		err := os.MkdirAll(vendorPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create vendor directory: %v", err)
		}

		// Create a test file in vendor directory
		testFile := filepath.Join(vendorPath, "test.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Remove vendor
		err = manager.Remove("test-vendor")
		if err != nil {
			t.Errorf("Remove() failed: %v", err)
		}

		// Verify vendor removed from lock file
		if _, exists := manager.lockFile.Vendors["test-vendor"]; exists {
			t.Error("Vendor should be removed from lock file")
		}

		// Verify directory removed
		if _, err := os.Stat(vendorPath); !os.IsNotExist(err) {
			t.Error("Vendor directory should be removed")
		}
	})
}

func TestManager_updateVendor(t *testing.T) {
	cfg := config.NewDefaultConfig()
	manager := NewManager(cfg)

	t.Run("vendor not in lock file", func(t *testing.T) {
		err := manager.updateVendor("nonexistent-vendor")
		if err == nil {
			t.Error("Expected error for vendor not in lock file")
		}

		expectedMsg := "vendor nonexistent-vendor not found in lock file"
		if !strings.Contains(err.Error(), expectedMsg) {
			t.Errorf("Expected error to contain %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("vendor directory does not exist", func(t *testing.T) {
		// Add vendor to lock file but don't create directory
		manager.lockFile.Vendors["test-vendor"] = config.VendorLock{
			URL:    "https://github.com/user/repo",
			Commit: "abc123",
		}

		err := manager.updateVendor("test-vendor")
		if err == nil {
			t.Error("Expected error for missing vendor directory")
		}

		expectedMsg := "vendor directory does not exist"
		if !strings.Contains(err.Error(), expectedMsg) {
			t.Errorf("Expected error to contain %q, got %q", expectedMsg, err.Error())
		}
	})
}

func TestManager_Fetch(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	cfg := config.NewDefaultConfig()
	mockFactory := git.NewMockGitRepositoryFactory()
	manager := NewManagerWithGitFactory(cfg, mockFactory)

	t.Run("fetch new vendor", func(t *testing.T) {
		testURL := "https://github.com/user/repo"

		err := manager.Fetch(testURL, "", false)
		if err != nil {
			t.Errorf("Fetch() failed: %v", err)
		}

		// Verify vendor added to lock file
		dirName := git.URLToDirectoryName(testURL)
		if _, exists := manager.lockFile.Vendors[dirName]; !exists {
			t.Error("Vendor should be added to lock file")
		}

		// Verify lock file entry
		vendorLock := manager.lockFile.Vendors[dirName]
		if vendorLock.URL != testURL {
			t.Errorf("Vendor URL = %q, want %q", vendorLock.URL, testURL)
		}
		if vendorLock.Commit == "" {
			t.Error("Vendor commit should not be empty")
		}
		if vendorLock.FetchedAt.IsZero() {
			t.Error("Vendor FetchedAt should be set")
		}
	})

	t.Run("fetch vendor with alias", func(t *testing.T) {
		testURL := "https://github.com/user/another-repo"
		alias := "custom-alias"

		err := manager.Fetch(testURL, alias, false)
		if err != nil {
			t.Errorf("Fetch() with alias failed: %v", err)
		}

		// Verify vendor added with alias as key
		if _, exists := manager.lockFile.Vendors[alias]; !exists {
			t.Error("Vendor should be added to lock file with alias")
		}
	})

	t.Run("fetch existing vendor without update flag", func(t *testing.T) {
		testURL := "https://github.com/user/existing-repo"
		dirName := git.URLToDirectoryName(testURL)

		// Set up mock to say repo exists
		mockRepo := &git.MockRepository{
			URL:         testURL,
			LocalPath:   filepath.Join("vendors", dirName),
			ShouldExist: true,
		}
		mockFactory.Repositories[fmt.Sprintf("%s:%s", testURL, filepath.Join("vendors", dirName))] = mockRepo

		err := manager.Fetch(testURL, "", false)
		if err == nil {
			t.Error("Expected error when fetching existing vendor without update flag")
		}
		if !strings.Contains(err.Error(), "vendor already exists") {
			t.Errorf("Expected 'vendor already exists' error, got: %v", err)
		}
	})

	t.Run("update existing vendor", func(t *testing.T) {
		testURL := "https://github.com/user/update-repo"
		dirName := git.URLToDirectoryName(testURL)

		// Set up mock to say repo exists and can be updated
		mockRepo := &git.MockRepository{
			URL:               testURL,
			LocalPath:         filepath.Join("vendors", dirName),
			ShouldExist:       true,
			MockCurrentCommit: "updated123",
		}
		mockFactory.Repositories[fmt.Sprintf("%s:%s", testURL, filepath.Join("vendors", dirName))] = mockRepo

		err := manager.Fetch(testURL, "", true)
		if err != nil {
			t.Errorf("Fetch() with update failed: %v", err)
		}

		if !mockRepo.PullCalled {
			t.Error("Pull should be called when updating existing vendor")
		}
	})

	t.Run("fetch fails on clone error", func(t *testing.T) {
		testURL := "https://github.com/user/fail-repo"
		dirName := git.URLToDirectoryName(testURL)

		// Set up mock to fail on clone
		mockRepo := &git.MockRepository{
			URL:             testURL,
			LocalPath:       filepath.Join("vendors", dirName),
			ShouldExist:     false,
			ShouldFailClone: true,
		}
		mockFactory.Repositories[fmt.Sprintf("%s:%s", testURL, filepath.Join("vendors", dirName))] = mockRepo

		err := manager.Fetch(testURL, "", false)
		if err == nil {
			t.Error("Expected error when clone fails")
		}
		if !strings.Contains(err.Error(), "failed to clone vendor") {
			t.Errorf("Expected 'failed to clone vendor' error, got: %v", err)
		}
	})
}

func TestManager_Update(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	cfg := config.NewDefaultConfig()
	mockFactory := git.NewMockGitRepositoryFactory()
	manager := NewManagerWithGitFactory(cfg, mockFactory)

	// Set up test vendors in lock file
	manager.lockFile.Vendors["vendor1"] = config.VendorLock{
		URL:       "https://github.com/user/repo1",
		Commit:    "old123",
		FetchedAt: time.Now().Add(-time.Hour),
	}
	manager.lockFile.Vendors["vendor2"] = config.VendorLock{
		URL:       "https://github.com/user/repo2",
		Commit:    "old456",
		FetchedAt: time.Now().Add(-time.Hour),
	}

	t.Run("update all vendors", func(t *testing.T) {
		// Set up mocks for both vendors
		for vendor, lock := range manager.lockFile.Vendors {
			mockRepo := &git.MockRepository{
				URL:               lock.URL,
				LocalPath:         filepath.Join("vendors", vendor),
				ShouldExist:       true,
				MockCurrentCommit: "new" + lock.Commit,
			}
			mockFactory.Repositories[fmt.Sprintf("%s:%s", lock.URL, filepath.Join("vendors", vendor))] = mockRepo
		}

		err := manager.Update([]string{})
		if err != nil {
			t.Errorf("Update() all vendors failed: %v", err)
		}

		// Verify lock file was saved (commits should be updated)
		for vendor, lock := range manager.lockFile.Vendors {
			if !strings.HasPrefix(lock.Commit, "new") {
				t.Errorf("Vendor %s commit should be updated, got: %s", vendor, lock.Commit)
			}
		}
	})

	t.Run("update specific vendors", func(t *testing.T) {
		// Reset commits
		manager.lockFile.Vendors["vendor1"] = config.VendorLock{
			URL:       "https://github.com/user/repo1",
			Commit:    "reset123",
			FetchedAt: time.Now().Add(-time.Hour),
		}

		mockRepo := &git.MockRepository{
			URL:               "https://github.com/user/repo1",
			LocalPath:         filepath.Join("vendors", "vendor1"),
			ShouldExist:       true,
			MockCurrentCommit: "specific123",
		}
		mockFactory.Repositories[fmt.Sprintf("%s:%s", "https://github.com/user/repo1", filepath.Join("vendors", "vendor1"))] = mockRepo

		err := manager.Update([]string{"vendor1"})
		if err != nil {
			t.Errorf("Update() specific vendor failed: %v", err)
		}

		// Verify only vendor1 was updated
		if !strings.Contains(manager.lockFile.Vendors["vendor1"].Commit, "specific") {
			t.Error("vendor1 should be updated")
		}
	})

	t.Run("update non-existent vendor", func(t *testing.T) {
		err := manager.Update([]string{"nonexistent"})
		if err == nil {
			t.Error("Expected error when updating non-existent vendor")
		}
		if !strings.Contains(err.Error(), "not found in lock file") {
			t.Errorf("Expected 'not found in lock file' error, got: %v", err)
		}
	})
}

func TestManager_Status(t *testing.T) {
	cfg := config.NewDefaultConfig()
	mockFactory := git.NewMockGitRepositoryFactory()
	manager := NewManagerWithGitFactory(cfg, mockFactory)

	t.Run("no vendors", func(t *testing.T) {
		err := manager.Status()
		if err != nil {
			t.Errorf("Status() with no vendors failed: %v", err)
		}
	})

	t.Run("vendors with different statuses", func(t *testing.T) {
		// Set up test vendors
		manager.lockFile.Vendors["missing-vendor"] = config.VendorLock{
			URL:    "https://github.com/user/missing",
			Commit: "abc123",
		}
		manager.lockFile.Vendors["uptodate-vendor"] = config.VendorLock{
			URL:    "https://github.com/user/uptodate",
			Commit: "def456",
		}
		manager.lockFile.Vendors["update-available"] = config.VendorLock{
			URL:    "https://github.com/user/outdated",
			Commit: "ghi789",
		}

		// Set up mocks
		// missing-vendor: doesn't exist
		missingRepo := &git.MockRepository{
			URL:         "https://github.com/user/missing",
			LocalPath:   filepath.Join("vendors", "missing-vendor"),
			ShouldExist: false,
		}
		mockFactory.Repositories[fmt.Sprintf("%s:%s", "https://github.com/user/missing", filepath.Join("vendors", "missing-vendor"))] = missingRepo

		// uptodate-vendor: exists and up to date
		uptodateRepo := &git.MockRepository{
			URL:               "https://github.com/user/uptodate",
			LocalPath:         filepath.Join("vendors", "uptodate-vendor"),
			ShouldExist:       true,
			MockCurrentCommit: "def456",
			MockRemoteCommit:  "def456",
		}
		mockFactory.Repositories[fmt.Sprintf("%s:%s", "https://github.com/user/uptodate", filepath.Join("vendors", "uptodate-vendor"))] = uptodateRepo

		// update-available: exists but has updates
		outdatedRepo := &git.MockRepository{
			URL:               "https://github.com/user/outdated",
			LocalPath:         filepath.Join("vendors", "update-available"),
			ShouldExist:       true,
			MockCurrentCommit: "ghi789",
			MockRemoteCommit:  "newer123",
		}
		mockFactory.Repositories[fmt.Sprintf("%s:%s", "https://github.com/user/outdated", filepath.Join("vendors", "update-available"))] = outdatedRepo

		err := manager.Status()
		if err != nil {
			t.Errorf("Status() failed: %v", err)
		}
	})
}

func TestManager_RestoreMissingVendors(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	cfg := config.NewDefaultConfig()
	manager := NewManager(cfg)

	t.Run("no vendors in lock file", func(t *testing.T) {
		err := manager.RestoreMissingVendors()
		if err != nil {
			t.Errorf("RestoreMissingVendors() failed with no vendors: %v", err)
		}
	})

	t.Run("all vendors present", func(t *testing.T) {
		// Add vendor to lock file
		manager.lockFile.Vendors["test-vendor"] = config.VendorLock{
			URL:    "https://github.com/user/repo",
			Commit: "abc123",
		}

		// Create vendor directory with .git
		vendorPath := filepath.Join("vendors", "test-vendor")
		gitPath := filepath.Join(vendorPath, ".git")
		err := os.MkdirAll(gitPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create vendor .git directory: %v", err)
		}

		err = manager.RestoreMissingVendors()
		if err != nil {
			t.Errorf("RestoreMissingVendors() failed: %v", err)
		}
	})

	t.Run("vendor missing but exists in lock file", func(t *testing.T) {
		// Create a manager with mock git factory to avoid network calls
		mockFactory := git.NewMockGitRepositoryFactory()
		mockManager := NewManagerWithGitFactory(cfg, mockFactory)
		mockManager.lockFile = &config.LockFile{Vendors: make(map[string]config.VendorLock)}

		// Configure mock to fail clone (simulating non-existent repo)
		mockFactory.ConfigureRepository("https://github.com/user/missing-repo", "vendors/missing-vendor", func(repo *git.MockRepository) {
			repo.ShouldFailClone = true
		})

		// Add vendor to lock file but don't create directory
		mockManager.lockFile.Vendors["missing-vendor"] = config.VendorLock{
			URL:    "https://github.com/user/missing-repo",
			Commit: "def456",
		}

		// This should not fail the test, just warn about failed clones
		err = mockManager.RestoreMissingVendors()
		if err != nil {
			t.Errorf("RestoreMissingVendors() failed: %v", err)
		}
	})
}

// Helper function to check if git is available on the system
func isGitAvailable() bool {
	// Just check if we can import the git package
	return true // Since we have git operations in our codebase, this should always be true
}

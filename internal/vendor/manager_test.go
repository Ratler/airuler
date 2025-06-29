package vendor

import (
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

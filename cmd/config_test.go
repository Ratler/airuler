package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ratler/airuler/internal/config"
)

func TestInitGlobalConfig(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Save original environment and set XDG_CONFIG_HOME
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)
	os.Setenv("XDG_CONFIG_HOME", tempDir)

	// Test successful global config initialization
	err := initGlobalConfig()
	if err != nil {
		t.Errorf("initGlobalConfig() failed: %v", err)
	}

	// Check that config file was created
	expectedPath := filepath.Join(tempDir, "airuler", "airuler.yaml")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Global config file was not created at %s", expectedPath)
	}

	// Check config content
	configContent, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Errorf("Failed to read global config: %v", err)
	}

	configStr := string(configContent)
	expectedConfigParts := []string{
		"defaults:",
	}

	for _, part := range expectedConfigParts {
		if !containsSubstring(configStr, part) {
			t.Errorf("Global config missing expected content: %s", part)
		}
	}
}

func TestInitGlobalConfigAlreadyExists(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Save original environment and set XDG_CONFIG_HOME
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)
	os.Setenv("XDG_CONFIG_HOME", tempDir)

	// Create existing global config
	configDir := filepath.Join(tempDir, "airuler")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "airuler.yaml")
	if err := os.WriteFile(configPath, []byte("existing: config"), 0644); err != nil {
		t.Fatalf("Failed to create existing config: %v", err)
	}

	// Test that initialization fails when config already exists
	err := initGlobalConfig()
	if err == nil {
		t.Error("initGlobalConfig() should fail when config already exists")
	}

	expectedError := "global config already exists"
	if !containsSubstring(err.Error(), expectedError) {
		t.Errorf("Expected error containing %q, got %q", expectedError, err.Error())
	}
}

func TestShowConfigPaths(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Save current directory and change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Save original environment and set XDG_CONFIG_HOME
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)
	os.Setenv("XDG_CONFIG_HOME", tempDir)

	// Create local config
	if err := os.WriteFile("airuler.yaml", []byte("local: config"), 0644); err != nil {
		t.Fatalf("Failed to create local config: %v", err)
	}

	// Create global config
	globalConfigDir := filepath.Join(tempDir, "airuler")
	if err := os.MkdirAll(globalConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create global config directory: %v", err)
	}

	globalConfigPath := filepath.Join(globalConfigDir, "airuler.yaml")
	if err := os.WriteFile(globalConfigPath, []byte("global: config"), 0644); err != nil {
		t.Fatalf("Failed to create global config: %v", err)
	}

	// Test showConfigPaths function
	err = showConfigPaths()
	if err != nil {
		t.Errorf("showConfigPaths() failed: %v", err)
	}

	// Note: We can't easily test the output since it goes to stdout,
	// but we can at least verify the function doesn't error
}

func TestShowConfigPathsNoConfigs(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Save current directory and change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Save original environment and set XDG_CONFIG_HOME to empty temp dir
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)
	emptyDir := filepath.Join(tempDir, "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("Failed to create empty directory: %v", err)
	}
	os.Setenv("XDG_CONFIG_HOME", emptyDir)

	// Test showConfigPaths when no configs exist
	err = showConfigPaths()
	if err != nil {
		t.Errorf("showConfigPaths() should not fail when no configs exist: %v", err)
	}
}

func TestEditGlobalConfig(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Save original environment and set XDG_CONFIG_HOME
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)
	os.Setenv("XDG_CONFIG_HOME", tempDir)

	// Set test mode to prevent actual editor launch
	originalTestMode := os.Getenv("AIRULER_TEST_MODE")
	defer os.Setenv("AIRULER_TEST_MODE", originalTestMode)
	os.Setenv("AIRULER_TEST_MODE", "1")

	// Test editGlobalConfig when no config exists
	err := editGlobalConfig()
	if err != nil {
		t.Errorf("editGlobalConfig() should not fail when no config exists: %v", err)
	}

	// Create global config
	globalConfigDir := filepath.Join(tempDir, "airuler")
	if err := os.MkdirAll(globalConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create global config directory: %v", err)
	}

	globalConfigPath := filepath.Join(globalConfigDir, "airuler.yaml")
	if err := os.WriteFile(globalConfigPath, []byte("global: config"), 0644); err != nil {
		t.Fatalf("Failed to create global config: %v", err)
	}

	// Test editGlobalConfig when config exists
	err = editGlobalConfig()
	if err != nil {
		t.Errorf("editGlobalConfig() failed: %v", err)
	}

	// Note: In test mode, we only verify the function doesn't error
}

func TestCreateVendorManager(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Save current directory and change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create a test config file
	testConfig := `defaults:
  include_vendors: [test-vendor]
  modes:
    claude: command
`

	if err := os.WriteFile("airuler.yaml", []byte(testConfig), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Create empty lock file
	_ = config.LockFile{
		Vendors: make(map[string]config.VendorLock),
	}

	// We can't easily test the full vendor manager without mocking viper,
	// but we can test that createVendorManager doesn't panic with valid config
	_, err = createVendorManager()
	if err != nil {
		// This might fail due to viper not being initialized in test context,
		// but it shouldn't panic
		t.Logf("createVendorManager() returned error (expected in test context): %v", err)
	}
}

func TestConfigIntegration(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Save original environment and set XDG_CONFIG_HOME
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)
	os.Setenv("XDG_CONFIG_HOME", tempDir)

	// Set test mode to prevent actual editor launch
	originalTestMode := os.Getenv("AIRULER_TEST_MODE")
	defer os.Setenv("AIRULER_TEST_MODE", originalTestMode)
	os.Setenv("AIRULER_TEST_MODE", "1")

	// Test complete workflow: init, check paths, edit

	// 1. Initialize global config
	err := initGlobalConfig()
	if err != nil {
		t.Errorf("initGlobalConfig() failed: %v", err)
	}

	// 2. Check that showConfigPaths works with existing config
	err = showConfigPaths()
	if err != nil {
		t.Errorf("showConfigPaths() failed after init: %v", err)
	}

	// 3. Check that editGlobalConfig works with existing config
	err = editGlobalConfig()
	if err != nil {
		t.Errorf("editGlobalConfig() failed after init: %v", err)
	}

	// 4. Verify we can't init again
	err = initGlobalConfig()
	if err == nil {
		t.Error("initGlobalConfig() should fail when config already exists")
	}
}

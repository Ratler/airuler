package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	// Save original environment
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	originalAppData := os.Getenv("APPDATA")

	defer func() {
		os.Setenv("XDG_CONFIG_HOME", originalXDG)
		os.Setenv("APPDATA", originalAppData)
	}()

	tests := []struct {
		name        string
		setupEnv    func()
		checkResult func(string) bool
	}{
		{
			name: "default config dir",
			setupEnv: func() {
				os.Unsetenv("XDG_CONFIG_HOME")
				os.Unsetenv("APPDATA")
			},
			checkResult: func(dir string) bool {
				switch runtime.GOOS {
				case "windows":
					return strings.Contains(dir, "AppData") && strings.HasSuffix(dir, "airuler")
				case "darwin", "linux":
					return strings.Contains(dir, ".config") && strings.HasSuffix(dir, "airuler")
				default:
					return strings.Contains(dir, ".config") && strings.HasSuffix(dir, "airuler")
				}
			},
		},
	}

	// Only test XDG_CONFIG_HOME on Unix-like systems
	if runtime.GOOS != "windows" {
		tests = append(tests, struct {
			name        string
			setupEnv    func()
			checkResult func(string) bool
		}{
			name: "XDG_CONFIG_HOME set",
			setupEnv: func() {
				os.Setenv("XDG_CONFIG_HOME", "/custom/config")
			},
			checkResult: func(dir string) bool {
				return dir == "/custom/config/airuler"
			},
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()

			dir, err := GetConfigDir()
			if err != nil {
				t.Errorf("GetConfigDir() unexpected error: %v", err)
				return
			}

			if !tt.checkResult(dir) {
				t.Errorf("GetConfigDir() = %v, failed validation", dir)
			}
		})
	}
}

func TestGetConfigFile(t *testing.T) {
	configFile, err := GetConfigFile()
	if err != nil {
		t.Errorf("GetConfigFile() unexpected error: %v", err)
		return
	}

	if !strings.HasSuffix(configFile, "airuler.yaml") {
		t.Errorf("GetConfigFile() = %v, expected to end with airuler.yaml", configFile)
	}

	// Should be an absolute path
	if !filepath.IsAbs(configFile) {
		t.Errorf("GetConfigFile() = %v, expected absolute path", configFile)
	}
}

func TestGetGlobalConfigPath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Save original environment
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	// Set XDG_CONFIG_HOME to our temp directory
	os.Setenv("XDG_CONFIG_HOME", tempDir)

	configPath, err := GetGlobalConfigPath()
	if err != nil {
		t.Errorf("GetGlobalConfigPath() unexpected error: %v", err)
		return
	}

	expectedPath := filepath.Join(tempDir, "airuler", "airuler.yaml")
	if configPath != expectedPath {
		t.Errorf("GetGlobalConfigPath() = %v, expected %v", configPath, expectedPath)
	}

	// Check that the directory was created
	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Errorf("GetGlobalConfigPath() did not create config directory: %v", configDir)
	}
}

func TestHasLocalConfig(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Create temporary directory
	tempDir := t.TempDir()
	os.Chdir(tempDir)

	// Initially should not have local config
	if HasLocalConfig() {
		t.Error("HasLocalConfig() returned true when no config exists")
	}

	// Create local config file
	configFile := "airuler.yaml"
	if err := os.WriteFile(configFile, []byte("test: config"), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Now should have local config
	if !HasLocalConfig() {
		t.Error("HasLocalConfig() returned false when config exists")
	}
}

func TestHasGlobalConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Save original environment
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	// Set XDG_CONFIG_HOME to our temp directory
	os.Setenv("XDG_CONFIG_HOME", tempDir)

	// Initially should not have global config
	if HasGlobalConfig() {
		t.Error("HasGlobalConfig() returned true when no config exists")
	}

	// Create config directory and file
	configDir := filepath.Join(tempDir, "airuler")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	configFile := filepath.Join(configDir, "airuler.yaml")
	if err := os.WriteFile(configFile, []byte("test: config"), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Now should have global config
	if !HasGlobalConfig() {
		t.Error("HasGlobalConfig() returned false when config exists")
	}
}

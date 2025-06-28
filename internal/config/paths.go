package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetConfigDir returns the appropriate configuration directory for the current platform
func GetConfigDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		// Windows: %APPDATA%\airuler
		appData := os.Getenv("APPDATA")
		if appData == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		return filepath.Join(appData, "airuler"), nil

	case "darwin":
		// macOS: ~/Library/Application Support/airuler (but we'll use ~/.config for consistency)
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		// Check for XDG_CONFIG_HOME first
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			return filepath.Join(xdgConfig, "airuler"), nil
		}
		return filepath.Join(homeDir, ".config", "airuler"), nil

	default:
		// Linux and other Unix-like systems: ~/.config/airuler
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		// Check for XDG_CONFIG_HOME first
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			return filepath.Join(xdgConfig, "airuler"), nil
		}
		return filepath.Join(homeDir, ".config", "airuler"), nil
	}
}

// GetConfigFile returns the full path to the config file
func GetConfigFile() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "airuler.yaml"), nil
}

// GetGlobalConfigPath returns the global config path, creating directories if needed
func GetGlobalConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "airuler.yaml"), nil
}

// HasLocalConfig checks if there's a local airuler.yaml in current directory
func HasLocalConfig() bool {
	_, err := os.Stat("airuler.yaml")
	return err == nil
}

// HasGlobalConfig checks if there's a global config file
func HasGlobalConfig() bool {
	configFile, err := GetConfigFile()
	if err != nil {
		return false
	}
	_, err = os.Stat(configFile)
	return err == nil
}

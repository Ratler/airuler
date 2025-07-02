// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
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

// IsTemplateDirectory checks if the given path is an airuler template directory
// by verifying the existence of both templates/ directory and airuler.lock file
func IsTemplateDirectory(path string) bool {
	templatesDir := filepath.Join(path, "templates")
	lockFile := filepath.Join(path, "airuler.lock")

	// Check if templates directory exists
	if stat, err := os.Stat(templatesDir); err != nil || !stat.IsDir() {
		return false
	}

	// Check if airuler.lock file exists
	if _, err := os.Stat(lockFile); err != nil {
		return false
	}

	return true
}

// UpdateLastTemplateDir updates the last_template_dir field in the global configuration
func UpdateLastTemplateDir(templateDir string) error {
	// Get absolute path to ensure consistency
	absPath, err := filepath.Abs(templateDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Verify it's a valid template directory
	if !IsTemplateDirectory(absPath) {
		return fmt.Errorf("directory '%s' is not a valid airuler template directory", absPath)
	}

	// Get global config path
	configPath, err := GetGlobalConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get global config path: %w", err)
	}

	// Read existing config or create new one
	var cfg Config
	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("failed to unmarshal existing config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read config file: %w", err)
	} else {
		// Create new config if it doesn't exist
		cfg = *NewDefaultConfig()
	}

	// Update the last template directory
	cfg.Defaults.LastTemplateDir = absPath

	// Marshal and write back to file
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetLastTemplateDir retrieves the last template directory from global configuration
func GetLastTemplateDir() (string, error) {
	configPath, err := GetGlobalConfigPath()
	if err != nil {
		return "", fmt.Errorf("failed to get global config path: %w", err)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", nil // No config file exists, return empty string
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg.Defaults.LastTemplateDir, nil
}

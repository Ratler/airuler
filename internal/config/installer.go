package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	yaml "gopkg.in/yaml.v3"
)

const installTrackerFileName = "airuler.installs"

// LoadInstallationTracker loads the installation tracker from the given directory
func LoadInstallationTracker(dir string) (*InstallationTracker, error) {
	tracker := &InstallationTracker{Installations: []InstallationRecord{}}

	if dir == "" {
		return tracker, nil
	}

	trackerPath := filepath.Join(dir, installTrackerFileName)

	if _, err := os.Stat(trackerPath); os.IsNotExist(err) {
		return tracker, nil // File doesn't exist yet
	}

	data, err := os.ReadFile(trackerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read installation tracker: %w", err)
	}

	if err := yaml.Unmarshal(data, tracker); err != nil {
		return nil, fmt.Errorf("failed to parse installation tracker: %w", err)
	}

	return tracker, nil
}

// SaveInstallationTracker saves the installation tracker to the given directory
func SaveInstallationTracker(dir string, tracker *InstallationTracker) error {
	if dir == "" {
		return fmt.Errorf("directory cannot be empty")
	}

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := yaml.Marshal(tracker)
	if err != nil {
		return fmt.Errorf("failed to marshal installation tracker: %w", err)
	}

	trackerPath := filepath.Join(dir, installTrackerFileName)
	return os.WriteFile(trackerPath, data, 0600)
}

// AddInstallation adds a new installation record to the tracker
func (t *InstallationTracker) AddInstallation(record InstallationRecord) {
	// Set timestamp if not already set
	if record.InstalledAt.IsZero() {
		record.InstalledAt = time.Now()
	}

	// Remove any existing record with the same target, rule, and location
	t.RemoveInstallation(record.Target, record.Rule, record.Global, record.ProjectPath, record.Mode)

	// Add the new record
	t.Installations = append(t.Installations, record)
}

// RemoveInstallation removes installation records matching the given criteria
func (t *InstallationTracker) RemoveInstallation(target, rule string, global bool, projectPath, mode string) {
	filtered := make([]InstallationRecord, 0, len(t.Installations))

	for _, install := range t.Installations {
		// Keep record if it doesn't match the criteria
		if install.Target != target ||
			install.Rule != rule ||
			install.Global != global ||
			install.ProjectPath != projectPath ||
			install.Mode != mode {
			filtered = append(filtered, install)
		}
	}

	t.Installations = filtered
}

// GetInstallations returns all installation records, optionally filtered by criteria
func (t *InstallationTracker) GetInstallations(target, rule string) []InstallationRecord {
	if target == "" && rule == "" {
		return t.Installations
	}

	var filtered []InstallationRecord
	for _, install := range t.Installations {
		if (target == "" || install.Target == target) &&
			(rule == "" || install.Rule == rule || install.Rule == "*") {
			filtered = append(filtered, install)
		}
	}

	return filtered
}

// LoadGlobalInstallationTracker loads the global installation tracker
func LoadGlobalInstallationTracker() (*InstallationTracker, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	return LoadInstallationTracker(configDir)
}

// SaveGlobalInstallationTracker saves the global installation tracker
func SaveGlobalInstallationTracker(tracker *InstallationTracker) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	return SaveInstallationTracker(configDir, tracker)
}

// LoadProjectInstallationTracker loads the installation tracker (same global location)
func LoadProjectInstallationTracker() (*InstallationTracker, error) {
	return LoadGlobalInstallationTracker()
}

// SaveProjectInstallationTracker saves the installation tracker (same global location)
func SaveProjectInstallationTracker(tracker *InstallationTracker) error {
	return SaveGlobalInstallationTracker(tracker)
}

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	yaml "gopkg.in/yaml.v3"
)

func TestLoadInstallationTracker(t *testing.T) {
	t.Run("empty directory returns empty tracker", func(t *testing.T) {
		tracker, err := LoadInstallationTracker("")
		if err != nil {
			t.Errorf("LoadInstallationTracker(\"\") error = %v, want nil", err)
		}
		if tracker == nil {
			t.Error("LoadInstallationTracker(\"\") returned nil tracker")
		}
		if len(tracker.Installations) != 0 {
			t.Errorf("LoadInstallationTracker(\"\") returned %d installations, want 0", len(tracker.Installations))
		}
	})

	t.Run("non-existent file returns empty tracker", func(t *testing.T) {
		tempDir := t.TempDir()
		tracker, err := LoadInstallationTracker(tempDir)
		if err != nil {
			t.Errorf("LoadInstallationTracker(%q) error = %v, want nil", tempDir, err)
		}
		if tracker == nil {
			t.Error("LoadInstallationTracker returned nil tracker")
		}
		if len(tracker.Installations) != 0 {
			t.Errorf("LoadInstallationTracker returned %d installations, want 0", len(tracker.Installations))
		}
	})

	t.Run("loads existing valid tracker file", func(t *testing.T) {
		tempDir := t.TempDir()
		trackerPath := filepath.Join(tempDir, installTrackerFileName)

		// Create sample tracker data
		sampleTracker := &InstallationTracker{
			Installations: []InstallationRecord{
				{
					Target:      "cursor",
					Rule:        "test-rule",
					Global:      true,
					ProjectPath: "/test/project",
					Mode:        "normal",
					InstalledAt: time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC),
					FilePath:    "/test/file.mdc",
				},
			},
		}

		data, err := yaml.Marshal(sampleTracker)
		if err != nil {
			t.Fatalf("Failed to marshal sample data: %v", err)
		}

		err = os.WriteFile(trackerPath, data, 0600)
		if err != nil {
			t.Fatalf("Failed to write sample file: %v", err)
		}

		// Load the tracker
		tracker, err := LoadInstallationTracker(tempDir)
		if err != nil {
			t.Errorf("LoadInstallationTracker(%q) error = %v, want nil", tempDir, err)
		}

		if len(tracker.Installations) != 1 {
			t.Errorf("LoadInstallationTracker returned %d installations, want 1", len(tracker.Installations))
		}

		install := tracker.Installations[0]
		if install.Target != "cursor" {
			t.Errorf("Installation.Target = %q, want %q", install.Target, "cursor")
		}
		if install.Rule != "test-rule" {
			t.Errorf("Installation.Rule = %q, want %q", install.Rule, "test-rule")
		}
		if !install.Global {
			t.Error("Installation.Global = false, want true")
		}
	})

	t.Run("handles invalid YAML file", func(t *testing.T) {
		tempDir := t.TempDir()
		trackerPath := filepath.Join(tempDir, installTrackerFileName)

		// Write invalid YAML
		err := os.WriteFile(trackerPath, []byte("invalid: yaml: content: ["), 0600)
		if err != nil {
			t.Fatalf("Failed to write invalid file: %v", err)
		}

		_, err = LoadInstallationTracker(tempDir)
		if err == nil {
			t.Error("LoadInstallationTracker with invalid YAML should return error")
		}
	})

	t.Run("handles read permission error", func(t *testing.T) {
		tempDir := t.TempDir()
		trackerPath := filepath.Join(tempDir, installTrackerFileName)

		// Create file with no read permissions
		err := os.WriteFile(trackerPath, []byte("installations: []"), 0000)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		_, err = LoadInstallationTracker(tempDir)
		if err == nil {
			t.Error("LoadInstallationTracker with unreadable file should return error")
		}

		// Cleanup - restore permissions so temp dir can be removed
		os.Chmod(trackerPath, 0600)
	})
}

func TestSaveInstallationTracker(t *testing.T) {
	t.Run("saves tracker to new directory", func(t *testing.T) {
		tempDir := t.TempDir()
		subDir := filepath.Join(tempDir, "subdir")

		tracker := &InstallationTracker{
			Installations: []InstallationRecord{
				{
					Target:      "claude",
					Rule:        "test-rule",
					Global:      false,
					ProjectPath: "/test/project",
					Mode:        "memory",
					InstalledAt: time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC),
					FilePath:    "/test/file.md",
				},
			},
		}

		err := SaveInstallationTracker(subDir, tracker)
		if err != nil {
			t.Errorf("SaveInstallationTracker error = %v, want nil", err)
		}

		// Verify file was created
		trackerPath := filepath.Join(subDir, installTrackerFileName)
		if _, err := os.Stat(trackerPath); os.IsNotExist(err) {
			t.Error("SaveInstallationTracker did not create tracker file")
		}

		// Verify content
		data, err := os.ReadFile(trackerPath)
		if err != nil {
			t.Fatalf("Failed to read saved file: %v", err)
		}

		var loadedTracker InstallationTracker
		err = yaml.Unmarshal(data, &loadedTracker)
		if err != nil {
			t.Fatalf("Failed to unmarshal saved data: %v", err)
		}

		if len(loadedTracker.Installations) != 1 {
			t.Errorf("Saved tracker has %d installations, want 1", len(loadedTracker.Installations))
		}

		install := loadedTracker.Installations[0]
		if install.Target != "claude" {
			t.Errorf("Saved installation.Target = %q, want %q", install.Target, "claude")
		}
	})

	t.Run("returns error for empty directory", func(t *testing.T) {
		tracker := &InstallationTracker{}
		err := SaveInstallationTracker("", tracker)
		if err == nil {
			t.Error("SaveInstallationTracker with empty directory should return error")
		}
	})

	t.Run("handles write permission error", func(t *testing.T) {
		tempDir := t.TempDir()

		// Make directory read-only
		err := os.Chmod(tempDir, 0444)
		if err != nil {
			t.Fatalf("Failed to change directory permissions: %v", err)
		}

		tracker := &InstallationTracker{}
		err = SaveInstallationTracker(tempDir, tracker)
		if err == nil {
			t.Error("SaveInstallationTracker with read-only directory should return error")
		}

		// Cleanup - restore permissions
		os.Chmod(tempDir, 0755)
	})
}

func TestInstallationTracker_AddInstallation(t *testing.T) {
	t.Run("adds new installation", func(t *testing.T) {
		tracker := &InstallationTracker{}

		record := InstallationRecord{
			Target:      "cursor",
			Rule:        "test-rule",
			Global:      true,
			ProjectPath: "/test/project",
			Mode:        "normal",
			FilePath:    "/test/file.mdc",
		}

		tracker.AddInstallation(record)

		if len(tracker.Installations) != 1 {
			t.Errorf("AddInstallation resulted in %d installations, want 1", len(tracker.Installations))
		}

		install := tracker.Installations[0]
		if install.Target != "cursor" {
			t.Errorf("Installation.Target = %q, want %q", install.Target, "cursor")
		}
		if install.InstalledAt.IsZero() {
			t.Error("Installation.InstalledAt should be set automatically")
		}
	})

	t.Run("replaces existing installation with same criteria", func(t *testing.T) {
		tracker := &InstallationTracker{}

		// Add first installation
		record1 := InstallationRecord{
			Target:      "cursor",
			Rule:        "test-rule",
			Global:      true,
			ProjectPath: "/test/project",
			Mode:        "normal",
			FilePath:    "/old/file.mdc",
			InstalledAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		}
		tracker.AddInstallation(record1)

		// Add second installation with same criteria but different file path
		record2 := InstallationRecord{
			Target:      "cursor",
			Rule:        "test-rule",
			Global:      true,
			ProjectPath: "/test/project",
			Mode:        "normal",
			FilePath:    "/new/file.mdc",
		}
		tracker.AddInstallation(record2)

		if len(tracker.Installations) != 1 {
			t.Errorf("AddInstallation resulted in %d installations, want 1", len(tracker.Installations))
		}

		install := tracker.Installations[0]
		if install.FilePath != "/new/file.mdc" {
			t.Errorf("Installation.FilePath = %q, want %q", install.FilePath, "/new/file.mdc")
		}
	})

	t.Run("preserves provided timestamp", func(t *testing.T) {
		tracker := &InstallationTracker{}

		fixedTime := time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC)
		record := InstallationRecord{
			Target:      "claude",
			Rule:        "test-rule",
			Global:      false,
			ProjectPath: "/test/project",
			Mode:        "memory",
			FilePath:    "/test/file.md",
			InstalledAt: fixedTime,
		}

		tracker.AddInstallation(record)

		install := tracker.Installations[0]
		if !install.InstalledAt.Equal(fixedTime) {
			t.Errorf("Installation.InstalledAt = %v, want %v", install.InstalledAt, fixedTime)
		}
	})
}

func TestInstallationTracker_RemoveInstallation(t *testing.T) {
	t.Run("removes matching installation", func(t *testing.T) {
		tracker := &InstallationTracker{
			Installations: []InstallationRecord{
				{
					Target:      "cursor",
					Rule:        "rule1",
					Global:      true,
					ProjectPath: "/project1",
					Mode:        "normal",
					FilePath:    "/file1.mdc",
				},
				{
					Target:      "claude",
					Rule:        "rule2",
					Global:      false,
					ProjectPath: "/project2",
					Mode:        "memory",
					FilePath:    "/file2.md",
				},
			},
		}

		tracker.RemoveInstallation("cursor", "rule1", true, "/project1", "normal")

		if len(tracker.Installations) != 1 {
			t.Errorf("RemoveInstallation resulted in %d installations, want 1", len(tracker.Installations))
		}

		remaining := tracker.Installations[0]
		if remaining.Target != "claude" {
			t.Errorf("Remaining installation.Target = %q, want %q", remaining.Target, "claude")
		}
	})

	t.Run("does not remove non-matching installation", func(t *testing.T) {
		tracker := &InstallationTracker{
			Installations: []InstallationRecord{
				{
					Target:      "cursor",
					Rule:        "rule1",
					Global:      true,
					ProjectPath: "/project1",
					Mode:        "normal",
					FilePath:    "/file1.mdc",
				},
			},
		}

		// Try to remove with different criteria
		tracker.RemoveInstallation("claude", "rule1", true, "/project1", "normal")

		if len(tracker.Installations) != 1 {
			t.Errorf("RemoveInstallation resulted in %d installations, want 1", len(tracker.Installations))
		}
	})

	t.Run("handles empty tracker", func(t *testing.T) {
		tracker := &InstallationTracker{}
		tracker.RemoveInstallation("cursor", "rule1", true, "/project1", "normal")

		if len(tracker.Installations) != 0 {
			t.Errorf("RemoveInstallation on empty tracker resulted in %d installations, want 0", len(tracker.Installations))
		}
	})
}

func TestInstallationTracker_GetInstallations(t *testing.T) {
	tracker := &InstallationTracker{
		Installations: []InstallationRecord{
			{
				Target:      "cursor",
				Rule:        "rule1",
				Global:      true,
				ProjectPath: "/project1",
				Mode:        "normal",
				FilePath:    "/file1.mdc",
			},
			{
				Target:      "claude",
				Rule:        "rule2",
				Global:      false,
				ProjectPath: "/project2",
				Mode:        "memory",
				FilePath:    "/file2.md",
			},
			{
				Target:      "cursor",
				Rule:        "*",
				Global:      true,
				ProjectPath: "/project3",
				Mode:        "normal",
				FilePath:    "/file3.mdc",
			},
		},
	}

	t.Run("returns all installations when no filter", func(t *testing.T) {
		results := tracker.GetInstallations("", "")
		if len(results) != 3 {
			t.Errorf("GetInstallations(\"\", \"\") returned %d installations, want 3", len(results))
		}
	})

	t.Run("filters by target", func(t *testing.T) {
		results := tracker.GetInstallations("cursor", "")
		if len(results) != 2 {
			t.Errorf("GetInstallations(\"cursor\", \"\") returned %d installations, want 2", len(results))
		}

		for _, install := range results {
			if install.Target != "cursor" {
				t.Errorf("Filtered installation has Target = %q, want %q", install.Target, "cursor")
			}
		}
	})

	t.Run("filters by rule", func(t *testing.T) {
		results := tracker.GetInstallations("", "rule1")
		if len(results) != 2 { // rule1 + wildcard *
			t.Errorf("GetInstallations(\"\", \"rule1\") returned %d installations, want 2", len(results))
		}
	})

	t.Run("filters by both target and rule", func(t *testing.T) {
		results := tracker.GetInstallations("claude", "rule2")
		if len(results) != 1 {
			t.Errorf("GetInstallations(\"claude\", \"rule2\") returned %d installations, want 1", len(results))
		}

		if results[0].Target != "claude" || results[0].Rule != "rule2" {
			t.Errorf("Filtered installation = %+v, want Target=claude, Rule=rule2", results[0])
		}
	})

	t.Run("handles wildcard rule matching", func(t *testing.T) {
		results := tracker.GetInstallations("cursor", "anything")
		if len(results) != 1 { // Only the wildcard should match
			t.Errorf("GetInstallations(\"cursor\", \"anything\") returned %d installations, want 1", len(results))
		}

		if results[0].Rule != "*" {
			t.Errorf("Wildcard match returned Rule = %q, want %q", results[0].Rule, "*")
		}
	})
}

func TestGlobalInstallationTrackerFunctions(t *testing.T) {
	t.Run("LoadGlobalInstallationTracker", func(t *testing.T) {
		tracker, err := LoadGlobalInstallationTracker()
		if err != nil {
			t.Errorf("LoadGlobalInstallationTracker() error = %v, want nil", err)
		}
		if tracker == nil {
			t.Error("LoadGlobalInstallationTracker() returned nil tracker")
		}
	})

	t.Run("SaveGlobalInstallationTracker", func(t *testing.T) {
		tracker := &InstallationTracker{
			Installations: []InstallationRecord{
				{
					Target:      "test",
					Rule:        "test-rule",
					Global:      true,
					ProjectPath: "",
					Mode:        "normal",
					FilePath:    "/test/file",
					InstalledAt: time.Now(),
				},
			},
		}

		err := SaveGlobalInstallationTracker(tracker)
		if err != nil {
			t.Errorf("SaveGlobalInstallationTracker() error = %v, want nil", err)
		}

		// Verify we can load it back
		loadedTracker, err := LoadGlobalInstallationTracker()
		if err != nil {
			t.Errorf("LoadGlobalInstallationTracker() after save error = %v, want nil", err)
		}

		if len(loadedTracker.Installations) != 1 {
			t.Errorf("Loaded tracker has %d installations, want 1", len(loadedTracker.Installations))
		}
	})
}

func TestProjectInstallationTrackerFunctions(t *testing.T) {
	t.Run("LoadProjectInstallationTracker", func(t *testing.T) {
		tracker, err := LoadProjectInstallationTracker()
		if err != nil {
			t.Errorf("LoadProjectInstallationTracker() error = %v, want nil", err)
		}
		if tracker == nil {
			t.Error("LoadProjectInstallationTracker() returned nil tracker")
		}
	})

	t.Run("SaveProjectInstallationTracker", func(t *testing.T) {
		tracker := &InstallationTracker{
			Installations: []InstallationRecord{
				{
					Target:      "project-test",
					Rule:        "project-rule",
					Global:      false,
					ProjectPath: "/test/project",
					Mode:        "normal",
					FilePath:    "/test/project/file",
					InstalledAt: time.Now(),
				},
			},
		}

		err := SaveProjectInstallationTracker(tracker)
		if err != nil {
			t.Errorf("SaveProjectInstallationTracker() error = %v, want nil", err)
		}
	})
}

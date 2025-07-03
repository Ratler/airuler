// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ratler/airuler/internal/config"
)

func TestLoadInstallations(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Save original global variables
	originalTarget := uninstallTarget
	originalRule := uninstallRule
	originalGlobal := uninstallGlobal
	originalProject := uninstallProject
	defer func() {
		uninstallTarget = originalTarget
		uninstallRule = originalRule
		uninstallGlobal = originalGlobal
		uninstallProject = originalProject
	}()

	// Create test installation tracker with sample data
	tracker := &config.InstallationTracker{
		Installations: []config.InstallationRecord{
			{
				Target:      "cursor",
				Rule:        "test-rule",
				Global:      true,
				ProjectPath: "",
				Mode:        "normal",
				InstalledAt: time.Now().Add(-time.Hour),
				FilePath:    "/global/cursor/test-rule.mdc",
			},
			{
				Target:      "claude",
				Rule:        "project-rule",
				Global:      false,
				ProjectPath: "/test/project",
				Mode:        "command",
				InstalledAt: time.Now().Add(-time.Minute),
				FilePath:    "/test/project/.claude/commands/project-rule.md",
			},
			{
				Target:      "cursor",
				Rule:        "another-rule",
				Global:      false,
				ProjectPath: "/test/project",
				Mode:        "normal",
				InstalledAt: time.Now(),
				FilePath:    "/test/project/.cursor/rules/another-rule.mdc",
			},
		},
	}

	err = config.SaveGlobalInstallationTracker(tracker)
	if err != nil {
		t.Fatalf("Failed to save test tracker: %v", err)
	}

	t.Run("load all installations", func(t *testing.T) {
		uninstallTarget = ""
		uninstallRule = ""
		uninstallGlobal = false
		uninstallProject = false

		installations, err := loadInstallations()
		if err != nil {
			t.Errorf("loadInstallations() failed: %v", err)
		}

		if len(installations) != 3 {
			t.Errorf("Expected 3 installations, got %d", len(installations))
		}
	})

	t.Run("filter by target", func(t *testing.T) {
		uninstallTarget = "cursor"
		uninstallRule = ""
		uninstallGlobal = false
		uninstallProject = false

		installations, err := loadInstallations()
		if err != nil {
			t.Errorf("loadInstallations() failed: %v", err)
		}

		if len(installations) != 2 {
			t.Errorf("Expected 2 cursor installations, got %d", len(installations))
		}

		for _, install := range installations {
			if install.Target != "cursor" {
				t.Errorf("Expected cursor target, got %s", install.Target)
			}
		}
	})

	t.Run("filter by rule", func(t *testing.T) {
		uninstallTarget = ""
		uninstallRule = "test-rule"
		uninstallGlobal = false
		uninstallProject = false

		installations, err := loadInstallations()
		if err != nil {
			t.Errorf("loadInstallations() failed: %v", err)
		}

		if len(installations) != 1 {
			t.Errorf("Expected 1 test-rule installation, got %d", len(installations))
		}

		if installations[0].Rule != "test-rule" {
			t.Errorf("Expected test-rule, got %s", installations[0].Rule)
		}
	})

	t.Run("filter by global only", func(t *testing.T) {
		uninstallTarget = ""
		uninstallRule = ""
		uninstallGlobal = true
		uninstallProject = false

		installations, err := loadInstallations()
		if err != nil {
			t.Errorf("loadInstallations() failed: %v", err)
		}

		if len(installations) != 1 {
			t.Errorf("Expected 1 global installation, got %d", len(installations))
		}

		if !installations[0].Global {
			t.Error("Expected global installation")
		}
	})

	t.Run("filter by project only", func(t *testing.T) {
		uninstallTarget = ""
		uninstallRule = ""
		uninstallGlobal = false
		uninstallProject = true

		installations, err := loadInstallations()
		if err != nil {
			t.Errorf("loadInstallations() failed: %v", err)
		}

		if len(installations) != 2 {
			t.Errorf("Expected 2 project installations, got %d", len(installations))
		}

		for _, install := range installations {
			if install.Global {
				t.Error("Expected project installation, got global")
			}
		}
	})
}

func TestUninstallRules(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Save original global variables
	originalTarget := uninstallTarget
	originalRule := uninstallRule
	originalGlobal := uninstallGlobal
	originalProject := uninstallProject
	originalForce := uninstallForce
	originalInteractive := uninstallInteractive
	defer func() {
		uninstallTarget = originalTarget
		uninstallRule = originalRule
		uninstallGlobal = originalGlobal
		uninstallProject = originalProject
		uninstallForce = originalForce
		uninstallInteractive = originalInteractive
	}()

	t.Run("no installations found", func(t *testing.T) {
		// Empty tracker
		tracker := &config.InstallationTracker{Installations: []config.InstallationRecord{}}
		err = config.SaveGlobalInstallationTracker(tracker)
		if err != nil {
			t.Fatalf("Failed to save empty tracker: %v", err)
		}

		uninstallTarget = ""
		uninstallRule = ""
		uninstallGlobal = false
		uninstallProject = false
		uninstallForce = true
		uninstallInteractive = false

		uninstallErr := uninstallRules()
		if uninstallErr != nil {
			t.Errorf("uninstallRules() with no installations failed: %v", uninstallErr)
		}
	})

	t.Run("force mode uninstall", func(t *testing.T) {
		// Create test file to uninstall
		testFile := filepath.Join(tempDir, "test-rule.md")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create tracker with test installation
		tracker := &config.InstallationTracker{
			Installations: []config.InstallationRecord{
				{
					Target:      "claude",
					Rule:        "test-rule",
					Global:      false,
					ProjectPath: tempDir,
					Mode:        "command",
					InstalledAt: time.Now(),
					FilePath:    testFile,
				},
			},
		}
		err = config.SaveGlobalInstallationTracker(tracker)
		if err != nil {
			t.Fatalf("Failed to save test tracker: %v", err)
		}

		uninstallTarget = ""
		uninstallRule = ""
		uninstallGlobal = false
		uninstallProject = false
		uninstallForce = true
		uninstallInteractive = false

		uninstallErr2 := uninstallRules()
		if uninstallErr2 != nil {
			t.Errorf("uninstallRules() in force mode failed: %v", uninstallErr2)
		}

		// Verify file was deleted
		if _, err := os.Stat(testFile); !os.IsNotExist(err) {
			t.Error("Test file should have been deleted")
		}

		// Verify installation was removed from tracker
		updatedTracker, err := config.LoadGlobalInstallationTracker()
		if err != nil {
			t.Errorf("Failed to load updated tracker: %v", err)
		}

		if len(updatedTracker.Installations) != 0 {
			t.Errorf("Expected 0 installations after uninstall, got %d", len(updatedTracker.Installations))
		}
	})
}

func TestPerformUninstallation(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Save original global variables
	originalForce := uninstallForce
	originalInteractive := uninstallInteractive
	defer func() {
		uninstallForce = originalForce
		uninstallInteractive = originalInteractive
	}()

	// Create test files
	testFile1 := filepath.Join(tempDir, "test-rule1.md")
	testFile2 := filepath.Join(tempDir, "test-rule2.md")
	err = os.WriteFile(testFile1, []byte("test content 1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}
	err = os.WriteFile(testFile2, []byte("test content 2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	// Create initial tracker
	installations := []config.InstallationRecord{
		{
			Target:      "claude",
			Rule:        "test-rule1",
			Global:      false,
			ProjectPath: tempDir,
			Mode:        "command",
			InstalledAt: time.Now(),
			FilePath:    testFile1,
		},
		{
			Target:      "cursor",
			Rule:        "test-rule2",
			Global:      false,
			ProjectPath: tempDir,
			Mode:        "normal",
			InstalledAt: time.Now(),
			FilePath:    testFile2,
		},
	}

	tracker := &config.InstallationTracker{Installations: installations}
	err = config.SaveGlobalInstallationTracker(tracker)
	if err != nil {
		t.Fatalf("Failed to save test tracker: %v", err)
	}

	uninstallForce = true
	uninstallInteractive = false

	err = performUninstallation(installations)
	if err != nil {
		t.Errorf("performUninstallation() failed: %v", err)
	}

	// Verify files were deleted
	if _, err := os.Stat(testFile1); !os.IsNotExist(err) {
		t.Error("Test file 1 should have been deleted")
	}
	if _, err := os.Stat(testFile2); !os.IsNotExist(err) {
		t.Error("Test file 2 should have been deleted")
	}

	// Verify tracker was updated
	updatedTracker, err := config.LoadGlobalInstallationTracker()
	if err != nil {
		t.Errorf("Failed to load updated tracker: %v", err)
	}

	if len(updatedTracker.Installations) != 0 {
		t.Errorf("Expected 0 installations after uninstall, got %d", len(updatedTracker.Installations))
	}
}

func TestUninstallSingle(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test-rule.md")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	installation := config.InstallationRecord{
		Target:      "claude",
		Rule:        "test-rule",
		Global:      false,
		ProjectPath: tempDir,
		Mode:        "command",
		InstalledAt: time.Now(),
		FilePath:    testFile,
	}

	tracker := &config.InstallationTracker{
		Installations: []config.InstallationRecord{installation},
	}

	t.Run("successful uninstall", func(t *testing.T) {
		err := uninstallSingle(installation, tracker)
		if err != nil {
			t.Errorf("uninstallSingle() failed: %v", err)
		}

		// Verify file was deleted
		if _, err := os.Stat(testFile); !os.IsNotExist(err) {
			t.Error("Test file should have been deleted")
		}

		// Verify installation was removed from tracker
		if len(tracker.Installations) != 0 {
			t.Errorf("Expected 0 installations in tracker, got %d", len(tracker.Installations))
		}
	})

	t.Run("uninstall non-existent file", func(t *testing.T) {
		nonExistentInstallation := config.InstallationRecord{
			Target:      "claude",
			Rule:        "non-existent",
			Global:      false,
			ProjectPath: tempDir,
			Mode:        "command",
			InstalledAt: time.Now(),
			FilePath:    filepath.Join(tempDir, "non-existent.md"),
		}

		// Add to tracker for removal test
		tracker.Installations = []config.InstallationRecord{nonExistentInstallation}

		err := uninstallSingle(nonExistentInstallation, tracker)
		if err != nil {
			t.Errorf("uninstallSingle() with non-existent file should not fail: %v", err)
		}

		// Should still remove from tracker
		if len(tracker.Installations) != 0 {
			t.Errorf("Expected 0 installations in tracker, got %d", len(tracker.Installations))
		}
	})

	t.Run("uninstall gemini target routes to special handler", func(t *testing.T) {
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)

		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to change to temp directory: %v", err)
		}

		// Create a test home directory to avoid interfering with real home directory
		testHomeDir := filepath.Join(tempDir, "test-home-single")
		err = os.MkdirAll(testHomeDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test home directory: %v", err)
		}

		// Save original HOME env var and restore after test
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", testHomeDir)
		defer os.Setenv("HOME", originalHome)

		// Create .gemini directory and GEMINI.md file
		geminiDir := filepath.Join(testHomeDir, ".gemini")
		err = os.MkdirAll(geminiDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create .gemini directory: %v", err)
		}

		geminiFile := filepath.Join(geminiDir, "GEMINI.md")
		err = os.WriteFile(geminiFile, []byte("# Test Content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create gemini file: %v", err)
		}

		geminiInstallation := config.InstallationRecord{
			Target:      "gemini",
			Rule:        "test-rule",
			Global:      true,
			ProjectPath: "",
			Mode:        "",
			InstalledAt: time.Now(),
			FilePath:    geminiFile,
		}

		tracker := &config.InstallationTracker{
			Installations: []config.InstallationRecord{geminiInstallation},
		}

		err = uninstallSingle(geminiInstallation, tracker)
		if err != nil {
			t.Errorf("uninstallSingle() for gemini target failed: %v", err)
		}

		// Verify gemini file was deleted (since it was the last rule)
		if _, err := os.Stat(geminiFile); !os.IsNotExist(err) {
			t.Error("Gemini file should be deleted when last rule is uninstalled")
		}

		// Verify installation was removed from tracker
		if len(tracker.Installations) != 0 {
			t.Errorf("Expected 0 installations in tracker, got %d", len(tracker.Installations))
		}
	})
}

func TestUninstallCopilotRule(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .github directory and copilot instructions file
	githubDir := filepath.Join(tempDir, ".github")
	err = os.MkdirAll(githubDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .github directory: %v", err)
	}

	copilotFile := filepath.Join(githubDir, "copilot-instructions.md")
	err = os.WriteFile(copilotFile, []byte("# Combined Copilot Instructions"), 0644)
	if err != nil {
		t.Fatalf("Failed to create copilot file: %v", err)
	}

	// Create compiled source files for testing reinstall
	compiledDir := filepath.Join("compiled", "copilot")
	err = os.MkdirAll(compiledDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create compiled directory: %v", err)
	}

	rule1Source := filepath.Join(compiledDir, "rule1.copilot-instructions.md")
	rule2Source := filepath.Join(compiledDir, "rule2.copilot-instructions.md")
	err = os.WriteFile(rule1Source, []byte("# Rule 1\nContent for rule 1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create rule1 source: %v", err)
	}
	err = os.WriteFile(rule2Source, []byte("# Rule 2\nContent for rule 2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create rule2 source: %v", err)
	}

	// Create tracker with multiple copilot rules
	installations := []config.InstallationRecord{
		{
			Target:      "copilot",
			Rule:        "rule1",
			Global:      false,
			ProjectPath: tempDir,
			Mode:        "",
			InstalledAt: time.Now(),
			FilePath:    copilotFile,
		},
		{
			Target:      "copilot",
			Rule:        "rule2",
			Global:      false,
			ProjectPath: tempDir,
			Mode:        "",
			InstalledAt: time.Now(),
			FilePath:    copilotFile,
		},
	}

	tracker := &config.InstallationTracker{Installations: installations}

	t.Run("uninstall one copilot rule", func(t *testing.T) {
		// Uninstall rule1, should reinstall with only rule2
		err := uninstallCopilotRule(installations[0], tracker)
		if err != nil {
			t.Errorf("uninstallCopilotRule() failed: %v", err)
		}

		// Verify rule1 was removed from tracker
		remainingRules := tracker.GetInstallations("copilot", "")
		if len(remainingRules) != 1 {
			t.Errorf("Expected 1 remaining copilot rule, got %d", len(remainingRules))
		}
		if remainingRules[0].Rule != "rule2" {
			t.Errorf("Expected remaining rule to be rule2, got %s", remainingRules[0].Rule)
		}

		// Verify copilot file still exists (reinstalled with rule2)
		if _, err := os.Stat(copilotFile); os.IsNotExist(err) {
			t.Error("Copilot file should still exist after partial uninstall")
		}

		// Verify content contains rule2
		content, err := os.ReadFile(copilotFile)
		if err != nil {
			t.Errorf("Failed to read copilot file: %v", err)
		}
		if !strings.Contains(string(content), "Content for rule 2") {
			t.Error("Copilot file should contain rule2 content")
		}
		if strings.Contains(string(content), "Content for rule 1") {
			t.Error("Copilot file should not contain rule1 content")
		}
	})

	t.Run("uninstall last copilot rule", func(t *testing.T) {
		// Uninstall rule2, should delete the file completely
		remainingInstallations := tracker.GetInstallations("copilot", "")
		if len(remainingInstallations) == 0 {
			t.Skip("No remaining installations to test")
		}

		err := uninstallCopilotRule(remainingInstallations[0], tracker)
		if err != nil {
			t.Errorf("uninstallCopilotRule() for last rule failed: %v", err)
		}

		// Verify no copilot rules remain in tracker
		remainingRules := tracker.GetInstallations("copilot", "")
		if len(remainingRules) != 0 {
			t.Errorf("Expected 0 remaining copilot rules, got %d", len(remainingRules))
		}

		// Verify copilot file was deleted
		if _, err := os.Stat(copilotFile); !os.IsNotExist(err) {
			t.Error("Copilot file should be deleted when last rule is uninstalled")
		}
	})
}

func TestReinstallCopilotRules(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create compiled source files
	compiledDir := filepath.Join("compiled", "copilot")
	err = os.MkdirAll(compiledDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create compiled directory: %v", err)
	}

	rule1Source := filepath.Join(compiledDir, "rule1.copilot-instructions.md")
	rule2Source := filepath.Join(compiledDir, "rule2.copilot-instructions.md")
	err = os.WriteFile(rule1Source, []byte("# Rule 1\nContent for rule 1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create rule1 source: %v", err)
	}
	err = os.WriteFile(rule2Source, []byte("# Rule 2\nContent for rule 2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create rule2 source: %v", err)
	}

	rules := []config.InstallationRecord{
		{
			Target:      "copilot",
			Rule:        "rule1",
			Global:      false,
			ProjectPath: tempDir,
			Mode:        "",
			InstalledAt: time.Now(),
			FilePath:    filepath.Join(tempDir, ".github", "copilot-instructions.md"),
		},
		{
			Target:      "copilot",
			Rule:        "rule2",
			Global:      false,
			ProjectPath: tempDir,
			Mode:        "",
			InstalledAt: time.Now(),
			FilePath:    filepath.Join(tempDir, ".github", "copilot-instructions.md"),
		},
	}

	t.Run("reinstall multiple rules", func(t *testing.T) {
		err := reinstallCopilotRules(rules, tempDir)
		if err != nil {
			t.Errorf("reinstallCopilotRules() failed: %v", err)
		}

		// Verify file was created
		copilotFile := filepath.Join(tempDir, ".github", "copilot-instructions.md")
		if _, err := os.Stat(copilotFile); os.IsNotExist(err) {
			t.Error("Copilot file should have been created")
		}

		// Verify content contains both rules
		content, err := os.ReadFile(copilotFile)
		if err != nil {
			t.Errorf("Failed to read copilot file: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "Content for rule 1") {
			t.Error("Copilot file should contain rule1 content")
		}
		if !strings.Contains(contentStr, "Content for rule 2") {
			t.Error("Copilot file should contain rule2 content")
		}
		if !strings.Contains(contentStr, "## rule1") {
			t.Error("Copilot file should contain rule1 header")
		}
		if !strings.Contains(contentStr, "## rule2") {
			t.Error("Copilot file should contain rule2 header")
		}
	})

	t.Run("reinstall with no rules", func(t *testing.T) {
		err := reinstallCopilotRules([]config.InstallationRecord{}, tempDir)
		if err != nil {
			t.Errorf("reinstallCopilotRules() with empty rules failed: %v", err)
		}
	})

	t.Run("reinstall with missing project path", func(t *testing.T) {
		err := reinstallCopilotRules(rules, "")
		if err == nil {
			t.Error("reinstallCopilotRules() without project path should fail")
		}
		if !strings.Contains(err.Error(), "copilot rules require project path") {
			t.Errorf("Expected 'require project path' error, got: %v", err)
		}
	})
}

func TestPrepareSelectionItems(t *testing.T) {
	installations := []config.InstallationRecord{
		{
			Target:      "cursor",
			Rule:        "rule1",
			Global:      true,
			ProjectPath: "",
			Mode:        "normal",
			InstalledAt: time.Now(),
			FilePath:    "/global/cursor/rule1.mdc",
		},
		{
			Target:      "claude",
			Rule:        "rule2",
			Global:      false,
			ProjectPath: "/test/project",
			Mode:        "command",
			InstalledAt: time.Now(),
			FilePath:    "/test/project/.claude/commands/rule2.md",
		},
	}

	items := prepareSelectionItems(installations)

	// Should have group headers + installations
	if len(items) < 2 {
		t.Errorf("Expected at least 2 items (installations), got %d", len(items))
	}

	// Verify installations are included
	installationCount := 0
	for _, item := range items {
		if !strings.HasPrefix(item.displayText, "GROUP_HEADER:") {
			installationCount++
		}
	}

	if installationCount != 2 {
		t.Errorf("Expected 2 installation items, got %d", installationCount)
	}
}

func TestParseDisplayText(t *testing.T) {
	installation := config.InstallationRecord{
		Target:      "cursor",
		Rule:        "test-rule",
		Global:      true,
		ProjectPath: "",
		Mode:        "normal",
		InstalledAt: time.Now().Add(-time.Hour),
		FilePath:    "/global/cursor/test-rule.mdc",
	}

	target, rule, mode, fileName, timeAgo := parseDisplayText("", installation)

	if target != "cursor" {
		t.Errorf("Expected target 'cursor', got '%s'", target)
	}
	if rule != "test-rule" {
		t.Errorf("Expected rule 'test-rule', got '%s'", rule)
	}
	if mode != "normal" {
		t.Errorf("Expected mode 'normal', got '%s'", mode)
	}
	if fileName != "test-rule.mdc" {
		t.Errorf("Expected fileName 'test-rule.mdc', got '%s'", fileName)
	}
	if timeAgo == "" {
		t.Error("Expected timeAgo to be set")
	}
}

func TestUninstallGeminiRule(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	t.Run("global gemini uninstall", func(t *testing.T) {
		// Create a test home directory to avoid interfering with real home directory
		testHomeDir := filepath.Join(tempDir, "test-home")
		err := os.MkdirAll(testHomeDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test home directory: %v", err)
		}

		// Save original HOME env var and restore after test
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", testHomeDir)
		defer os.Setenv("HOME", originalHome)

		// Create .gemini directory and GEMINI.md file
		geminiDir := filepath.Join(testHomeDir, ".gemini")
		err = os.MkdirAll(geminiDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create .gemini directory: %v", err)
		}

		geminiFile := filepath.Join(geminiDir, "GEMINI.md")
		err = os.WriteFile(geminiFile, []byte("# Combined Gemini Instructions"), 0644)
		if err != nil {
			t.Fatalf("Failed to create gemini file: %v", err)
		}

		// Create compiled source files for testing reinstall
		compiledDir := filepath.Join("compiled", "gemini")
		err = os.MkdirAll(compiledDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create compiled directory: %v", err)
		}

		rule1Source := filepath.Join(compiledDir, "rule1.md")
		rule2Source := filepath.Join(compiledDir, "rule2.md")
		err = os.WriteFile(rule1Source, []byte("# Rule 1\nContent for rule 1"), 0644)
		if err != nil {
			t.Fatalf("Failed to create rule1 source: %v", err)
		}
		err = os.WriteFile(rule2Source, []byte("# Rule 2\nContent for rule 2"), 0644)
		if err != nil {
			t.Fatalf("Failed to create rule2 source: %v", err)
		}

		// Create tracker with multiple global gemini rules
		installations := []config.InstallationRecord{
			{
				Target:      "gemini",
				Rule:        "rule1",
				Global:      true,
				ProjectPath: "",
				Mode:        "",
				InstalledAt: time.Now(),
				FilePath:    geminiFile,
			},
			{
				Target:      "gemini",
				Rule:        "rule2",
				Global:      true,
				ProjectPath: "",
				Mode:        "",
				InstalledAt: time.Now(),
				FilePath:    geminiFile,
			},
		}

		tracker := &config.InstallationTracker{Installations: installations}

		// Uninstall rule1, should reinstall with only rule2
		err = uninstallGeminiRule(installations[0], tracker)
		if err != nil {
			t.Errorf("uninstallGeminiRule() failed: %v", err)
		}

		// Verify rule1 was removed from tracker
		remainingRules := tracker.GetInstallations("gemini", "")
		if len(remainingRules) != 1 {
			t.Errorf("Expected 1 remaining gemini rule, got %d", len(remainingRules))
		}
		if remainingRules[0].Rule != "rule2" {
			t.Errorf("Expected remaining rule to be rule2, got %s", remainingRules[0].Rule)
		}

		// Verify gemini file still exists (reinstalled with rule2)
		if _, err := os.Stat(geminiFile); os.IsNotExist(err) {
			t.Error("Gemini file should still exist after partial uninstall")
		}

		// Verify content contains rule2
		content, err := os.ReadFile(geminiFile)
		if err != nil {
			t.Errorf("Failed to read gemini file: %v", err)
		}
		if !strings.Contains(string(content), "Content for rule 2") {
			t.Error("Gemini file should contain rule2 content")
		}
		if strings.Contains(string(content), "Content for rule 1") {
			t.Error("Gemini file should not contain rule1 content")
		}

		// Uninstall rule2, should delete the file completely
		remainingInstallations := tracker.GetInstallations("gemini", "")
		if len(remainingInstallations) > 0 {
			err = uninstallGeminiRule(remainingInstallations[0], tracker)
			if err != nil {
				t.Errorf("uninstallGeminiRule() for last rule failed: %v", err)
			}

			// Verify no gemini rules remain in tracker
			remainingRules = tracker.GetInstallations("gemini", "")
			if len(remainingRules) != 0 {
				t.Errorf("Expected 0 remaining gemini rules, got %d", len(remainingRules))
			}

			// Verify gemini file was deleted
			if _, err := os.Stat(geminiFile); !os.IsNotExist(err) {
				t.Error("Gemini file should be deleted when last rule is uninstalled")
			}
		}
	})

	t.Run("project gemini uninstall", func(t *testing.T) {
		projectDir := filepath.Join(tempDir, "test-project")
		err := os.MkdirAll(projectDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create project directory: %v", err)
		}

		geminiFile := filepath.Join(projectDir, "GEMINI.md")
		err = os.WriteFile(geminiFile, []byte("# Project Gemini Instructions"), 0644)
		if err != nil {
			t.Fatalf("Failed to create project gemini file: %v", err)
		}

		// Create compiled source files
		compiledDir := filepath.Join("compiled", "gemini")
		err = os.MkdirAll(compiledDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create compiled directory: %v", err)
		}

		ruleSource := filepath.Join(compiledDir, "project-rule.md")
		err = os.WriteFile(ruleSource, []byte("# Project Rule\nProject content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create rule source: %v", err)
		}

		// Create tracker with project gemini rule
		installation := config.InstallationRecord{
			Target:      "gemini",
			Rule:        "project-rule",
			Global:      false,
			ProjectPath: projectDir,
			Mode:        "",
			InstalledAt: time.Now(),
			FilePath:    geminiFile,
		}

		tracker := &config.InstallationTracker{Installations: []config.InstallationRecord{installation}}

		// Uninstall the project rule
		err = uninstallGeminiRule(installation, tracker)
		if err != nil {
			t.Errorf("uninstallGeminiRule() for project rule failed: %v", err)
		}

		// Verify rule was removed from tracker
		remainingRules := tracker.GetInstallations("gemini", "")
		if len(remainingRules) != 0 {
			t.Errorf("Expected 0 remaining gemini rules, got %d", len(remainingRules))
		}

		// Verify project gemini file was deleted
		if _, err := os.Stat(geminiFile); !os.IsNotExist(err) {
			t.Error("Project gemini file should be deleted when rule is uninstalled")
		}
	})
}

func TestReinstallGeminiRules(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create compiled source files
	compiledDir := filepath.Join("compiled", "gemini")
	err = os.MkdirAll(compiledDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create compiled directory: %v", err)
	}

	rule1Source := filepath.Join(compiledDir, "rule1.md")
	rule2Source := filepath.Join(compiledDir, "rule2.md")
	err = os.WriteFile(rule1Source, []byte("# Rule 1\nContent for rule 1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create rule1 source: %v", err)
	}
	err = os.WriteFile(rule2Source, []byte("# Rule 2\nContent for rule 2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create rule2 source: %v", err)
	}

	t.Run("reinstall global gemini rules", func(t *testing.T) {
		// Create a test home directory to avoid interfering with real home directory
		testHomeDir := filepath.Join(tempDir, "test-home-reinstall")
		err := os.MkdirAll(testHomeDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test home directory: %v", err)
		}

		// Save original HOME env var and restore after test
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", testHomeDir)
		defer os.Setenv("HOME", originalHome)

		rules := []config.InstallationRecord{
			{
				Target:      "gemini",
				Rule:        "rule1",
				Global:      true,
				ProjectPath: "",
				Mode:        "",
				InstalledAt: time.Now(),
				FilePath:    filepath.Join(testHomeDir, ".gemini", "GEMINI.md"),
			},
			{
				Target:      "gemini",
				Rule:        "rule2",
				Global:      true,
				ProjectPath: "",
				Mode:        "",
				InstalledAt: time.Now(),
				FilePath:    filepath.Join(testHomeDir, ".gemini", "GEMINI.md"),
			},
		}

		err = reinstallGeminiRules(rules, "", true)
		if err != nil {
			t.Errorf("reinstallGeminiRules() for global rules failed: %v", err)
		}

		// Verify file was created
		geminiFile := filepath.Join(testHomeDir, ".gemini", "GEMINI.md")
		if _, err := os.Stat(geminiFile); os.IsNotExist(err) {
			t.Error("Global gemini file should have been created")
		}

		// Verify content contains both rules
		content, err := os.ReadFile(geminiFile)
		if err != nil {
			t.Errorf("Failed to read gemini file: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "Content for rule 1") {
			t.Error("Gemini file should contain rule1 content")
		}
		if !strings.Contains(contentStr, "Content for rule 2") {
			t.Error("Gemini file should contain rule2 content")
		}
		if !strings.Contains(contentStr, "## rule1") {
			t.Error("Gemini file should contain rule1 header")
		}
		if !strings.Contains(contentStr, "## rule2") {
			t.Error("Gemini file should contain rule2 header")
		}
		if !strings.Contains(contentStr, "This file contains custom instructions for Gemini CLI") {
			t.Error("Gemini file should contain Gemini CLI header")
		}
	})

	t.Run("reinstall project gemini rules", func(t *testing.T) {
		projectDir := filepath.Join(tempDir, "test-project")
		err := os.MkdirAll(projectDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create project directory: %v", err)
		}

		rules := []config.InstallationRecord{
			{
				Target:      "gemini",
				Rule:        "rule1",
				Global:      false,
				ProjectPath: projectDir,
				Mode:        "",
				InstalledAt: time.Now(),
				FilePath:    filepath.Join(projectDir, "GEMINI.md"),
			},
		}

		err = reinstallGeminiRules(rules, projectDir, false)
		if err != nil {
			t.Errorf("reinstallGeminiRules() for project rules failed: %v", err)
		}

		// Verify file was created
		geminiFile := filepath.Join(projectDir, "GEMINI.md")
		if _, err := os.Stat(geminiFile); os.IsNotExist(err) {
			t.Error("Project gemini file should have been created")
		}

		// Verify content contains rule
		content, err := os.ReadFile(geminiFile)
		if err != nil {
			t.Errorf("Failed to read project gemini file: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "Content for rule 1") {
			t.Error("Project gemini file should contain rule1 content")
		}
	})

	t.Run("reinstall with no rules", func(t *testing.T) {
		err := reinstallGeminiRules([]config.InstallationRecord{}, tempDir, false)
		if err != nil {
			t.Errorf("reinstallGeminiRules() with empty rules failed: %v", err)
		}
	})

	t.Run("reinstall project rules with missing project path", func(t *testing.T) {
		rules := []config.InstallationRecord{
			{
				Target:      "gemini",
				Rule:        "rule1",
				Global:      false,
				ProjectPath: "",
				Mode:        "",
				InstalledAt: time.Now(),
				FilePath:    "/some/path/GEMINI.md",
			},
		}

		err := reinstallGeminiRules(rules, "", false)
		if err == nil {
			t.Error("reinstallGeminiRules() without project path should fail")
		}
		if !strings.Contains(err.Error(), "gemini project rules require project path") {
			t.Errorf("Expected 'require project path' error, got: %v", err)
		}
	})

	t.Run("reinstall with missing compiled files", func(t *testing.T) {
		// Create a separate test directory to avoid interfering with real home directory
		testHomeDir := filepath.Join(tempDir, "test-home")
		err := os.MkdirAll(testHomeDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test home directory: %v", err)
		}

		// Save original HOME env var and restore after test
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", testHomeDir)
		defer os.Setenv("HOME", originalHome)

		rules := []config.InstallationRecord{
			{
				Target:      "gemini",
				Rule:        "missing-rule",
				Global:      true,
				ProjectPath: "",
				Mode:        "",
				InstalledAt: time.Now(),
				FilePath:    filepath.Join(testHomeDir, ".gemini", "GEMINI.md"),
			},
		}

		err = reinstallGeminiRules(rules, "", true)
		if err != nil {
			t.Errorf("reinstallGeminiRules() with missing compiled files failed: %v", err)
		}

		// Should not create file if no content found
		geminiFile := filepath.Join(testHomeDir, ".gemini", "GEMINI.md")
		if _, err := os.Stat(geminiFile); !os.IsNotExist(err) {
			t.Error("Gemini file should not be created when no compiled content is found")
		}
	})
}

func TestShowUninstallPreviewAndConfirm(t *testing.T) {
	installations := []config.InstallationRecord{
		{
			Target:      "cursor",
			Rule:        "test-rule",
			Global:      true,
			ProjectPath: "",
			Mode:        "normal",
			InstalledAt: time.Now(),
			FilePath:    "/global/cursor/test-rule.mdc",
		},
	}

	// This function requires user input, so we can only test it doesn't panic
	// In a real test environment, you might want to mock the input
	t.Run("displays preview", func(t *testing.T) {
		// Just verify the function doesn't panic when called
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("showUninstallPreviewAndConfirm() panicked: %v", r)
			}
		}()

		// Since this function requires user input via Scanln, we can't easily test it
		// without mocking stdin. For now, we just verify the helper functions work.
		displayUninstallTable(installations)
	})
}

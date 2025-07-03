// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ratler/airuler/internal/compiler"
)

func TestCopyFile(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create source file
	sourceContent := "test content for copy"
	sourcePath := filepath.Join(tempDir, "source.txt")
	if err := os.WriteFile(sourcePath, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Test copy
	destPath := filepath.Join(tempDir, "dest.txt")
	err := copyFile(sourcePath, destPath)
	if err != nil {
		t.Errorf("copyFile() failed: %v", err)
	}

	// Check that destination file exists and has correct content
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Error("Destination file was not created")
	}

	destContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Errorf("Failed to read destination file: %v", err)
	}

	if string(destContent) != sourceContent {
		t.Errorf("Destination content = %q, expected %q", string(destContent), sourceContent)
	}
}

func TestCopyFileNonExistentSource(t *testing.T) {
	tempDir := t.TempDir()

	sourcePath := filepath.Join(tempDir, "nonexistent.txt")
	destPath := filepath.Join(tempDir, "dest.txt")

	err := copyFile(sourcePath, destPath)
	if err == nil {
		t.Error("copyFile() should fail with non-existent source file")
	}
}

func TestGetTargetInstallDir(t *testing.T) {
	// Save original values
	originalProject := installProject
	defer func() { installProject = originalProject }()

	// Test with project path
	installProject = "/test/project"

	tests := []struct {
		target   compiler.Target
		expected string
	}{
		{compiler.TargetCursor, "/test/project/.cursor/rules"},
		{compiler.TargetClaude, "/test/project/.claude/commands"},
		{compiler.TargetCline, "/test/project/.clinerules"},
		{compiler.TargetCopilot, "/test/project/.github"},
		{compiler.TargetGemini, "/test/project"},
		{compiler.TargetRoo, "/test/project/.roo/rules"},
	}

	for _, tt := range tests {
		t.Run(string(tt.target), func(t *testing.T) {
			result, err := getTargetInstallDir(tt.target)
			if err != nil {
				t.Errorf("getTargetInstallDir() failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("getTargetInstallDir() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestGetProjectInstallDir(t *testing.T) {
	tests := []struct {
		target      compiler.Target
		projectPath string
		expected    string
	}{
		{compiler.TargetCursor, "/project", "/project/.cursor/rules"},
		{compiler.TargetClaude, "/project", "/project/.claude/commands"},
		{compiler.TargetCline, "/project", "/project/.clinerules"},
		{compiler.TargetCopilot, "/project", "/project/.github"},
		{compiler.TargetGemini, "/project", "/project"},
		{compiler.TargetRoo, "/project", "/project/.roo/rules"},
	}

	for _, tt := range tests {
		t.Run(string(tt.target), func(t *testing.T) {
			result, err := getProjectInstallDir(tt.target, tt.projectPath)
			if err != nil {
				t.Errorf("getProjectInstallDir() failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("getProjectInstallDir() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestGetProjectInstallDirRelativePath(t *testing.T) {
	// Test with relative path
	result, err := getProjectInstallDir(compiler.TargetClaude, "./relative/path")
	if err != nil {
		t.Errorf("getProjectInstallDir() with relative path failed: %v", err)
	}

	// Should return an absolute path
	if !filepath.IsAbs(result) {
		t.Errorf("getProjectInstallDir() with relative path should return absolute path, got %q", result)
	}

	// Should end with the expected suffix
	if !strings.HasSuffix(result, "/.claude/commands") {
		t.Errorf("getProjectInstallDir() result should end with /.claude/commands, got %q", result)
	}
}

func TestGetGlobalInstallDir(t *testing.T) {
	// This test checks the logic but can't verify actual system paths
	// since they depend on the user's home directory and OS

	tests := []compiler.Target{
		compiler.TargetCursor,
		compiler.TargetClaude,
		compiler.TargetCline,
		compiler.TargetCopilot,
		compiler.TargetGemini,
		compiler.TargetRoo,
	}

	for _, target := range tests {
		t.Run(string(target), func(t *testing.T) {
			result, err := getGlobalInstallDir(target)

			// Copilot should fail for global installation
			if target == compiler.TargetCopilot {
				if err == nil {
					t.Errorf("getGlobalInstallDir() should fail for copilot, but got result: %q", result)
				}
				return
			}

			if err != nil {
				t.Errorf("getGlobalInstallDir() failed for %s: %v", target, err)
			}

			// Should return an absolute path
			if !filepath.IsAbs(result) {
				t.Errorf("getGlobalInstallDir() should return absolute path, got %q", result)
			}

			// Check OS-specific expectations
			switch target {
			case compiler.TargetCursor:
				switch runtime.GOOS {
				case "darwin":
					if !containsSubstring(result, "Library/Application Support") {
						t.Errorf("macOS Cursor path should contain Library/Application Support, got %q", result)
					}
				case "windows":
					if !containsSubstring(result, "AppData") {
						t.Errorf("Windows Cursor path should contain AppData, got %q", result)
					}
				default:
					if !containsSubstring(result, ".config") {
						t.Errorf("Linux Cursor path should contain .config, got %q", result)
					}
				}
			case compiler.TargetClaude:
				if !containsSubstring(result, ".claude/commands") {
					t.Errorf("Claude path should contain .claude/commands, got %q", result)
				}
			case compiler.TargetCline:
				if !containsSubstring(result, ".clinerules") {
					t.Errorf("Cline path should contain .clinerules, got %q", result)
				}
			}
		})
	}
}

func TestInstallFileIntegration(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create source file
	sourceContent := "test rule content"
	sourcePath := filepath.Join(tempDir, "source.md")
	if err := os.WriteFile(sourcePath, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Test install without existing target
	targetPath := filepath.Join(tempDir, "target.md")

	// Save original installForce flag
	originalForce := installForce
	defer func() { installForce = originalForce }()
	installForce = false

	err := installFile(sourcePath, targetPath, compiler.TargetClaude)
	if err != nil {
		t.Errorf("installFile() failed: %v", err)
	}

	// Check that target file was created
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Error("Target file was not created")
	}

	// Check content
	targetContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Errorf("Failed to read target file: %v", err)
	}

	if string(targetContent) != sourceContent {
		t.Errorf("Target content = %q, expected %q", string(targetContent), sourceContent)
	}
}

func TestInstallFileWithBackup(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create source file
	sourceContent := "new rule content"
	sourcePath := filepath.Join(tempDir, "source.md")
	if err := os.WriteFile(sourcePath, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create existing target file
	existingContent := "existing rule content"
	targetPath := filepath.Join(tempDir, "target.md")
	if err := os.WriteFile(targetPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create existing target file: %v", err)
	}

	// Save original installForce flag
	originalForce := installForce
	defer func() { installForce = originalForce }()
	installForce = false

	err := installFile(sourcePath, targetPath, compiler.TargetClaude)
	if err != nil {
		t.Errorf("installFile() failed: %v", err)
	}

	// Check that target file was updated
	targetContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Errorf("Failed to read target file: %v", err)
	}

	if string(targetContent) != sourceContent {
		t.Errorf("Target content = %q, expected %q", string(targetContent), sourceContent)
	}

	// Check that backup was created
	backupFiles := []string{}
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp directory: %v", err)
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "target.md.backup.") {
			backupFiles = append(backupFiles, file.Name())
		}
	}

	if len(backupFiles) != 1 {
		t.Errorf("Expected 1 backup file, found %d: %v", len(backupFiles), backupFiles)
	}

	if len(backupFiles) > 0 {
		// Check backup content
		backupPath := filepath.Join(tempDir, backupFiles[0])
		backupContent, err := os.ReadFile(backupPath)
		if err != nil {
			t.Errorf("Failed to read backup file: %v", err)
		}

		if string(backupContent) != existingContent {
			t.Errorf("Backup content = %q, expected %q", string(backupContent), existingContent)
		}
	}
}

func TestInstallFileWithForce(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create source file
	sourceContent := "new rule content"
	sourcePath := filepath.Join(tempDir, "source.md")
	if err := os.WriteFile(sourcePath, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create existing target file
	existingContent := "existing rule content"
	targetPath := filepath.Join(tempDir, "target.md")
	if err := os.WriteFile(targetPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create existing target file: %v", err)
	}

	// Save original installForce flag
	originalForce := installForce
	defer func() { installForce = originalForce }()
	installForce = true

	err := installFile(sourcePath, targetPath, compiler.TargetClaude)
	if err != nil {
		t.Errorf("installFile() with force failed: %v", err)
	}

	// Check that target file was updated
	targetContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Errorf("Failed to read target file: %v", err)
	}

	if string(targetContent) != sourceContent {
		t.Errorf("Target content = %q, expected %q", string(targetContent), sourceContent)
	}

	// With force flag, no backup should be created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp directory: %v", err)
	}

	backupCount := 0
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "target.md.backup.") {
			backupCount++
		}
	}

	if backupCount != 0 {
		t.Errorf("Expected no backup files with force flag, found %d", backupCount)
	}
}

func TestInstallForTarget(t *testing.T) {
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
	originalTarget := installTarget
	originalRule := installRule
	originalProject := installProject
	defer func() {
		installTarget = originalTarget
		installRule = originalRule
		installProject = originalProject
	}()

	t.Run("no compiled directory", func(t *testing.T) {
		count, err := installForTarget(compiler.TargetCursor)
		if err == nil {
			t.Error("Expected error when compiled directory doesn't exist")
		}
		if count != 0 {
			t.Errorf("Expected count 0, got %d", count)
		}
		if !strings.Contains(err.Error(), "no compiled rules found") {
			t.Errorf("Expected 'no compiled rules found' error, got: %v", err)
		}
	})

	t.Run("install cursor rules", func(t *testing.T) {
		// Create compiled directory and files
		compiledDir := filepath.Join("compiled", "cursor")
		err := os.MkdirAll(compiledDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create compiled directory: %v", err)
		}

		// Create sample rule files
		ruleContent := "# Test Rule\nThis is a test rule."
		ruleFiles := []string{"test-rule.mdc", "another-rule.mdc"}
		for _, fileName := range ruleFiles {
			err := os.WriteFile(filepath.Join(compiledDir, fileName), []byte(ruleContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create rule file %s: %v", fileName, err)
			}
		}

		// Set global install directory (we can't test global install easily, so use project)
		installProject = tempDir

		count, err := installForTarget(compiler.TargetCursor)
		if err != nil {
			t.Errorf("installForTarget() failed: %v", err)
		}
		if count != 2 {
			t.Errorf("Expected count 2, got %d", count)
		}

		// Verify files were installed
		targetDir := filepath.Join(tempDir, ".cursor", "rules")
		for _, fileName := range ruleFiles {
			targetPath := filepath.Join(targetDir, fileName)
			if _, err := os.Stat(targetPath); os.IsNotExist(err) {
				t.Errorf("Rule file %s was not installed", fileName)
			}
		}
	})

	t.Run("install with rule filter", func(t *testing.T) {
		// Clean up previous test
		err := os.RemoveAll(filepath.Join(tempDir, ".cursor"))
		if err != nil {
			t.Fatalf("Failed to clean up: %v", err)
		}

		installRule = "test-rule"
		installProject = tempDir

		count, err := installForTarget(compiler.TargetCursor)
		if err != nil {
			t.Errorf("installForTarget() with filter failed: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected count 1 with filter, got %d", count)
		}

		// Verify only filtered file was installed
		targetDir := filepath.Join(tempDir, ".cursor", "rules")
		if _, err := os.Stat(filepath.Join(targetDir, "test-rule.mdc")); os.IsNotExist(err) {
			t.Error("Filtered rule file was not installed")
		}
		if _, err := os.Stat(filepath.Join(targetDir, "another-rule.mdc")); !os.IsNotExist(err) {
			t.Error("Non-filtered rule file should not be installed")
		}
	})

	t.Run("install claude rules with memory mode", func(t *testing.T) {
		// Create compiled directory for Claude
		compiledDir := filepath.Join("compiled", "claude")
		err := os.MkdirAll(compiledDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create compiled directory: %v", err)
		}

		// Create Claude-specific files
		commandContent := "# Command Rule\nThis is a command rule."
		memoryContent := "# Memory Rule\nThis is a memory rule."

		err = os.WriteFile(filepath.Join(compiledDir, "command-rule.md"), []byte(commandContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create command rule: %v", err)
		}

		err = os.WriteFile(filepath.Join(compiledDir, "CLAUDE.md"), []byte(memoryContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create memory rule: %v", err)
		}

		installRule = "" // install all
		installProject = tempDir

		count, err := installForTarget(compiler.TargetClaude)
		if err != nil {
			t.Errorf("installForTarget() for Claude failed: %v", err)
		}
		if count != 2 {
			t.Errorf("Expected count 2 for Claude rules, got %d", count)
		}

		// Verify files were installed in correct locations
		commandPath := filepath.Join(tempDir, ".claude", "commands", "command-rule.md")
		if _, err := os.Stat(commandPath); os.IsNotExist(err) {
			t.Error("Command rule was not installed in commands directory")
		}

		// CLAUDE.md goes to project root for memory mode
		claudePath := filepath.Join(tempDir, "CLAUDE.md")
		if _, err := os.Stat(claudePath); os.IsNotExist(err) {
			t.Error("CLAUDE.md was not installed in project root for memory mode")
		}
	})
}

func TestInstallRules(t *testing.T) {
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
	originalTarget := installTarget
	originalRule := installRule
	originalProject := installProject
	originalInteractive := installInteractive
	defer func() {
		installTarget = originalTarget
		installRule = originalRule
		installProject = originalProject
		installInteractive = originalInteractive
	}()

	// Setup compiled directories and files for multiple targets
	targets := []compiler.Target{compiler.TargetCursor, compiler.TargetClaude}
	for _, target := range targets {
		compiledDir := filepath.Join("compiled", string(target))
		err := os.MkdirAll(compiledDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create compiled directory for %s: %v", target, err)
		}

		// Create sample rule file
		var fileName string
		if target == compiler.TargetCursor {
			fileName = "test-rule.mdc"
		} else {
			fileName = "test-rule.md"
		}

		err = os.WriteFile(filepath.Join(compiledDir, fileName), []byte("# Test Rule"), 0644)
		if err != nil {
			t.Fatalf("Failed to create rule file for %s: %v", target, err)
		}
	}

	t.Run("install all targets", func(t *testing.T) {
		installTarget = ""
		installRule = ""
		installProject = tempDir
		installInteractive = false

		installErr := installRules()
		if installErr != nil {
			t.Errorf("installRules() failed: %v", installErr)
		}

		// Verify files were installed for all targets
		cursorPath := filepath.Join(tempDir, ".cursor", "rules", "test-rule.mdc")
		if _, err := os.Stat(cursorPath); os.IsNotExist(err) {
			t.Error("Cursor rule was not installed")
		}

		claudePath := filepath.Join(tempDir, ".claude", "commands", "test-rule.md")
		if _, err := os.Stat(claudePath); os.IsNotExist(err) {
			t.Error("Claude rule was not installed")
		}
	})

	t.Run("install specific target", func(t *testing.T) {
		// Clean up previous test
		err := os.RemoveAll(filepath.Join(tempDir, ".cursor"))
		if err != nil {
			t.Fatalf("Failed to clean up: %v", err)
		}
		err = os.RemoveAll(filepath.Join(tempDir, ".claude"))
		if err != nil {
			t.Fatalf("Failed to clean up: %v", err)
		}

		installTarget = "cursor"
		installRule = ""
		installProject = tempDir
		installInteractive = false

		installErr2 := installRules()
		if installErr2 != nil {
			t.Errorf("installRules() for specific target failed: %v", installErr2)
		}

		// Verify only cursor rule was installed
		cursorPath := filepath.Join(tempDir, ".cursor", "rules", "test-rule.mdc")
		if _, err := os.Stat(cursorPath); os.IsNotExist(err) {
			t.Error("Cursor rule was not installed")
		}

		claudePath := filepath.Join(tempDir, ".claude", "commands", "test-rule.md")
		if _, err := os.Stat(claudePath); !os.IsNotExist(err) {
			t.Error("Claude rule should not be installed when targeting cursor only")
		}
	})

	t.Run("invalid target", func(t *testing.T) {
		installTarget = "invalid-target"
		installRule = ""
		installProject = tempDir
		installInteractive = false

		err := installRules()
		if err == nil {
			t.Error("Expected error for invalid target")
		}
		if !strings.Contains(err.Error(), "invalid target") {
			t.Errorf("Expected 'invalid target' error, got: %v", err)
		}
	})
}

func TestRecordInstallation(t *testing.T) {
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
	originalProject := installProject
	defer func() {
		installProject = originalProject
	}()

	t.Run("record global installation", func(t *testing.T) {
		installProject = ""

		err := recordInstallation(compiler.TargetCursor, "test-rule", "/global/path/test-rule.mdc", "normal")
		if err != nil {
			t.Errorf("recordInstallation() failed: %v", err)
		}
	})

	t.Run("record project installation", func(t *testing.T) {
		installProject = tempDir

		err := recordInstallation(compiler.TargetClaude, "project-rule", filepath.Join(tempDir, "project-rule.md"), "command")
		if err != nil {
			t.Errorf("recordInstallation() for project failed: %v", err)
		}
	})
}

func TestInstallFileWithMode(t *testing.T) {
	tempDir := t.TempDir()

	// Create source file
	sourceContent := "# Test Rule\nThis is a test rule with $ARGUMENTS placeholder."
	sourcePath := filepath.Join(tempDir, "source.md")
	if err := os.WriteFile(sourcePath, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	t.Run("install file with command mode", func(t *testing.T) {
		targetPath := filepath.Join(tempDir, "command.md")

		err := installFileWithMode(sourcePath, targetPath, compiler.TargetClaude, "command")
		if err != nil {
			t.Errorf("installFileWithMode() failed: %v", err)
		}

		// Check that file was created
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			t.Error("Target file was not created")
		}

		// Check content contains $ARGUMENTS
		content, err := os.ReadFile(targetPath)
		if err != nil {
			t.Errorf("Failed to read target file: %v", err)
		}

		if !strings.Contains(string(content), "$ARGUMENTS") {
			t.Error("Command mode file should contain $ARGUMENTS placeholder")
		}
	})

	t.Run("install file with memory mode", func(t *testing.T) {
		targetPath := filepath.Join(tempDir, "memory.md")

		err := installFileWithMode(sourcePath, targetPath, compiler.TargetClaude, "memory")
		if err != nil {
			t.Errorf("installFileWithMode() for memory mode failed: %v", err)
		}

		// Memory mode should append to CLAUDE.md rather than replace
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			t.Error("Target file was not created")
		}
	})

	t.Run("install file with normal mode", func(t *testing.T) {
		targetPath := filepath.Join(tempDir, "normal.md")

		err := installFileWithMode(sourcePath, targetPath, compiler.TargetCursor, "")
		if err != nil {
			t.Errorf("installFileWithMode() for normal mode failed: %v", err)
		}

		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			t.Error("Target file was not created")
		}
	})
}

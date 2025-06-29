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

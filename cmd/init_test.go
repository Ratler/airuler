// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"os"
	"testing"
)

func TestInitProject(t *testing.T) {
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

	// Set test mode to skip interactive prompts
	originalTestMode := os.Getenv("AIRULER_TEST_MODE")
	defer os.Setenv("AIRULER_TEST_MODE", originalTestMode)
	os.Setenv("AIRULER_TEST_MODE", "1")

	// Test successful initialization
	err = initProject(".")
	if err != nil {
		t.Errorf("initProject failed: %v", err)
	}

	// Check that required files and directories were created
	expectedPaths := []string{
		"templates/partials",
		"templates/examples",
		"vendors",
		"compiled/cursor",
		"compiled/claude",
		"compiled/cline",
		"compiled/copilot",
		"airuler.yaml",
		"airuler.lock",
		".gitignore",
		"templates/examples/example.tmpl",
	}

	for _, path := range expectedPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected path %s was not created", path)
		}
	}

	// Check airuler.yaml content
	configContent, err := os.ReadFile("airuler.yaml")
	if err != nil {
		t.Errorf("Failed to read airuler.yaml: %v", err)
	}

	configStr := string(configContent)
	expectedConfigParts := []string{
		"defaults:",
	}

	for _, part := range expectedConfigParts {
		if !contains(configStr, part) {
			t.Errorf("airuler.yaml missing expected content: %s", part)
		}
	}

	// Check .gitignore content
	gitignoreContent, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Errorf("Failed to read .gitignore: %v", err)
	}

	gitignoreStr := string(gitignoreContent)
	expectedGitignoreParts := []string{
		"vendors/",
		"*.backup.*",
		".DS_Store",
		".vscode/",
		"*.swp",
	}

	for _, part := range expectedGitignoreParts {
		if !contains(gitignoreStr, part) {
			t.Errorf(".gitignore missing expected content: %s", part)
		}
	}
}

func TestInitProjectAlreadyExists(t *testing.T) {
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

	// Set test mode to skip interactive prompts
	originalTestMode := os.Getenv("AIRULER_TEST_MODE")
	defer os.Setenv("AIRULER_TEST_MODE", originalTestMode)
	os.Setenv("AIRULER_TEST_MODE", "1")

	// Create existing airuler.yaml
	if err := os.WriteFile("airuler.yaml", []byte("existing: config"), 0644); err != nil {
		t.Fatalf("Failed to create existing config: %v", err)
	}

	// Test that initialization fails when config already exists
	err = initProject(".")
	if err == nil {
		t.Error("initProject should fail when airuler.yaml already exists")
	}

	expectedError := "airuler.yaml already exists"
	if !contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing %q, got %q", expectedError, err.Error())
	}
}

func TestInitProjectFilePermissions(t *testing.T) {
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

	// Set test mode to skip interactive prompts
	originalTestMode := os.Getenv("AIRULER_TEST_MODE")
	defer os.Setenv("AIRULER_TEST_MODE", originalTestMode)
	os.Setenv("AIRULER_TEST_MODE", "1")

	// Test successful initialization
	err = initProject(".")
	if err != nil {
		t.Errorf("initProject failed: %v", err)
	}

	// Check file permissions
	fileTests := []struct {
		path         string
		expectedMode os.FileMode
		isDir        bool
	}{
		{"templates", 0755, true},
		{"vendors", 0755, true},
		{"compiled", 0755, true},
		{"airuler.yaml", 0600, false},
		{"airuler.lock", 0600, false},
		{"templates/examples/example.tmpl", 0600, false},
	}

	for _, test := range fileTests {
		info, err := os.Stat(test.path)
		if err != nil {
			t.Errorf("Failed to stat %s: %v", test.path, err)
			continue
		}

		if test.isDir && !info.IsDir() {
			t.Errorf("%s should be a directory", test.path)
		}

		if !test.isDir && info.IsDir() {
			t.Errorf("%s should not be a directory", test.path)
		}

		// Check permissions (mask with 0777 to ignore type bits)
		actualMode := info.Mode() & 0777
		if actualMode != test.expectedMode {
			t.Errorf("%s has mode %o, expected %o", test.path, actualMode, test.expectedMode)
		}
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

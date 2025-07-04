// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
)

func TestSyncCommand(t *testing.T) {
	// Test that the sync command is properly configured
	if syncCmd == nil {
		t.Fatal("syncCmd should not be nil")
	}

	// Test help output
	t.Run("sync help", func(t *testing.T) {
		// Create a new root command for testing
		rootCmd := &cobra.Command{
			Use: "airuler",
		}
		rootCmd.AddCommand(syncCmd)

		// Capture output
		var output bytes.Buffer
		rootCmd.SetOutput(&output)
		rootCmd.SetArgs([]string{"sync", "--help"})

		// Execute command
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Check output
		outputStr := output.String()
		if !strings.Contains(outputStr, "Sync performs the complete airuler workflow") {
			t.Errorf("Expected help output to contain workflow description, got: %s", outputStr)
		}
		if !strings.Contains(outputStr, "--no-git-pull") {
			t.Errorf("Expected help output to contain --no-git-pull flag, got: %s", outputStr)
		}
	})
}

func TestSyncFlags(t *testing.T) {
	// Test flag parsing
	syncCmd.ParseFlags([]string{
		"--no-update",
		"--no-compile",
		"--no-deploy",
		"--no-git-pull",
		"--scope", "global",
		"--targets", "cursor,claude",
		"--dry-run",
		"--force",
	})

	// Verify flags are set correctly
	if !syncNoUpdate {
		t.Error("Expected syncNoUpdate to be true")
	}
	if !syncNoCompile {
		t.Error("Expected syncNoCompile to be true")
	}
	if !syncNoDeploy {
		t.Error("Expected syncNoDeploy to be true")
	}
	if !syncNoGitPull {
		t.Error("Expected syncNoGitPull to be true")
	}
	if syncScope != "global" {
		t.Errorf("Expected syncScope to be 'global', got '%s'", syncScope)
	}
	if syncTargets != "cursor,claude" {
		t.Errorf("Expected syncTargets to be 'cursor,claude', got '%s'", syncTargets)
	}
	if !syncDryRun {
		t.Error("Expected syncDryRun to be true")
	}
	if !syncForce {
		t.Error("Expected syncForce to be true")
	}

	// Reset flags for other tests
	syncNoUpdate = false
	syncNoCompile = false
	syncNoDeploy = false
	syncNoGitPull = false
	syncScope = "all"
	syncTargets = ""
	syncDryRun = false
	syncForce = false
}

func TestRunSyncAllStepsDisabled(t *testing.T) {
	// Set all skip flags to true
	syncNoUpdate = true
	syncNoCompile = true
	syncNoDeploy = true
	syncNoGitPull = true

	defer func() {
		// Reset flags
		syncNoUpdate = false
		syncNoCompile = false
		syncNoDeploy = false
		syncNoGitPull = false
	}()

	err := runSync("")
	if err == nil {
		t.Error("Expected error when all steps are disabled")
	}
	if !strings.Contains(err.Error(), "all steps disabled") {
		t.Errorf("Expected 'all steps disabled' error, got: %v", err)
	}
}

func TestRunSyncGitPull(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "airuler-sync-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save original working directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	t.Run("non-git directory", func(t *testing.T) {
		// Change to non-git directory
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to change to temp dir: %v", err)
		}

		// Should silently continue without error
		err := runSyncGitPull()
		if err != nil {
			t.Errorf("Expected no error for non-git directory, got: %v", err)
		}
	})

	t.Run("git repository with clean working tree", func(t *testing.T) {
		gitDir := filepath.Join(tempDir, "git-test")
		if err := os.MkdirAll(gitDir, 0755); err != nil {
			t.Fatalf("Failed to create git test dir: %v", err)
		}

		// Initialize git repository
		repo, err := gogit.PlainInit(gitDir, false)
		if err != nil {
			t.Fatalf("Failed to init git repo: %v", err)
		}

		// Create initial commit
		worktree, err := repo.Worktree()
		if err != nil {
			t.Fatalf("Failed to get worktree: %v", err)
		}

		// Create a test file
		testFile := filepath.Join(gitDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Add and commit the file
		_, err = worktree.Add("test.txt")
		if err != nil {
			t.Fatalf("Failed to add file: %v", err)
		}

		_, err = worktree.Commit("Initial commit", &gogit.CommitOptions{
			Author: &object.Signature{
				Name:  "Test User",
				Email: "test@example.com",
			},
		})
		if err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		// Change to git directory
		if err := os.Chdir(gitDir); err != nil {
			t.Fatalf("Failed to change to git dir: %v", err)
		}

		// Set force flag to avoid user prompts
		originalForce := syncForce
		syncForce = true
		defer func() { syncForce = originalForce }()

		// Capture output to check behavior with clean repo
		output := captureOutput(func() {
			err = runSyncGitPull()
		})

		// Should handle the case gracefully (git pull will fail but continue with force flag)
		if err != nil {
			t.Errorf("Expected no error for clean git repo with force flag, got: %v", err)
		}

		// Check that it attempted to pull
		if !strings.Contains(output, "Pulling template repository") {
			t.Errorf("Expected git pull attempt message, got: %s", output)
		}
	})

	t.Run("git repository with dirty working tree", func(t *testing.T) {
		gitDir := filepath.Join(tempDir, "git-dirty-test")
		if err := os.MkdirAll(gitDir, 0755); err != nil {
			t.Fatalf("Failed to create git test dir: %v", err)
		}

		// Initialize git repository
		repo, err := gogit.PlainInit(gitDir, false)
		if err != nil {
			t.Fatalf("Failed to init git repo: %v", err)
		}

		// Create initial commit
		worktree, err := repo.Worktree()
		if err != nil {
			t.Fatalf("Failed to get worktree: %v", err)
		}

		// Create and commit a test file
		testFile := filepath.Join(gitDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("initial content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, err = worktree.Add("test.txt")
		if err != nil {
			t.Fatalf("Failed to add file: %v", err)
		}

		_, err = worktree.Commit("Initial commit", &gogit.CommitOptions{
			Author: &object.Signature{
				Name:  "Test User",
				Email: "test@example.com",
			},
		})
		if err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		// Make the working tree dirty
		if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
			t.Fatalf("Failed to modify test file: %v", err)
		}

		// Change to git directory
		if err := os.Chdir(gitDir); err != nil {
			t.Fatalf("Failed to change to git dir: %v", err)
		}

		// Capture output to verify warning message
		output := captureOutput(func() {
			err = runSyncGitPull()
		})

		if err != nil {
			t.Errorf("Expected no error for dirty git repo, got: %v", err)
		}

		// Should skip git pull and show warning
		if !strings.Contains(output, "uncommitted changes") {
			t.Errorf("Expected warning about uncommitted changes, got output: %s", output)
		}
		if !strings.Contains(output, "skipping git pull") {
			t.Errorf("Expected message about skipping git pull, got output: %s", output)
		}
	})
}

func TestRunSyncDryRun(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "airuler-sync-dry-run-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save original working directory and change to temp dir
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Test dry run with default settings
	output := captureOutput(func() {
		_ = runSyncDryRun("")
	})

	// Check that dry run output contains expected elements
	if !strings.Contains(output, "Dry run mode") {
		t.Errorf("Expected dry run mode message, got: %s", output)
	}
	if !strings.Contains(output, "Steps that would run") {
		t.Errorf("Expected steps information, got: %s", output)
	}
	if !strings.Contains(output, "Run without --dry-run") {
		t.Errorf("Expected instruction to run without dry-run, got: %s", output)
	}

	// Test dry run with all steps disabled
	syncNoUpdate = true
	syncNoCompile = true
	syncNoDeploy = true
	defer func() {
		syncNoUpdate = false
		syncNoCompile = false
		syncNoDeploy = false
	}()

	output = captureOutput(func() {
		_ = runSyncDryRun("")
	})

	if !strings.Contains(output, "All steps disabled") {
		t.Errorf("Expected message about all steps disabled, got: %s", output)
	}
}

func TestSyncStepConditions(t *testing.T) {
	tests := []struct {
		name          string
		noUpdate      bool
		noGitPull     bool
		noCompile     bool
		noDeploy      bool
		expectedSteps []string
		expectError   bool
	}{
		{
			name:          "all steps enabled",
			expectedSteps: []string{"pull template repository", "update vendors", "compile templates", "update installations"},
		},
		{
			name:          "no update flag set",
			noUpdate:      true,
			expectedSteps: []string{"compile templates", "update installations"},
		},
		{
			name:          "no git pull flag set",
			noGitPull:     true,
			expectedSteps: []string{"update vendors", "compile templates", "update installations"},
		},
		{
			name:          "no compile flag set",
			noCompile:     true,
			expectedSteps: []string{"pull template repository", "update vendors", "update installations"},
		},
		{
			name:          "no deploy flag set",
			noDeploy:      true,
			expectedSteps: []string{"pull template repository", "update vendors", "compile templates"},
		},
		{
			name:        "all flags set",
			noUpdate:    true,
			noGitPull:   true,
			noCompile:   true,
			noDeploy:    true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set flags
			syncNoUpdate = tt.noUpdate
			syncNoGitPull = tt.noGitPull
			syncNoCompile = tt.noCompile
			syncNoDeploy = tt.noDeploy

			// Reset flags after test
			defer func() {
				syncNoUpdate = false
				syncNoGitPull = false
				syncNoCompile = false
				syncNoDeploy = false
			}()

			// Build expected steps
			var steps []string
			if !syncNoUpdate && !syncNoGitPull {
				steps = append(steps, "pull template repository")
			}
			if !syncNoUpdate {
				steps = append(steps, "update vendors")
			}
			if !syncNoCompile {
				steps = append(steps, "compile templates")
			}
			if !syncNoDeploy {
				steps = append(steps, "update installations")
			}

			// Verify step conditions match expected
			if tt.expectError {
				if len(steps) != 0 {
					t.Errorf("Expected no steps when all disabled, got: %v", steps)
				}
			} else {
				if len(steps) != len(tt.expectedSteps) {
					t.Errorf("Expected %d steps, got %d: %v", len(tt.expectedSteps), len(steps), steps)
				}
				for i, expected := range tt.expectedSteps {
					if i >= len(steps) || steps[i] != expected {
						t.Errorf("Expected step %d to be '%s', got '%s'", i, expected, steps[i])
					}
				}
			}
		})
	}
}

// captureOutput captures stdout during function execution
func captureOutput(fn func()) string {
	// Save original stdout
	originalStdout := os.Stdout

	// Create a pipe to capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run function
	fn()

	// Restore stdout
	w.Close()
	os.Stdout = originalStdout

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestSyncCommandIntegration(t *testing.T) {
	// This test verifies that the sync command is properly registered
	// and has the expected structure

	if syncCmd == nil {
		t.Fatal("syncCmd should not be nil")
	}

	if syncCmd.Use != "sync [target]" {
		t.Errorf("Expected Use to be 'sync [target]', got '%s'", syncCmd.Use)
	}

	if syncCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if syncCmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	// Check that all expected flags are present
	expectedFlags := []string{
		"no-update", "no-compile", "no-deploy", "no-git-pull",
		"scope", "targets", "dry-run", "force",
	}

	for _, flagName := range expectedFlags {
		flag := syncCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag '%s' to exist", flagName)
		}
	}

	// Verify help text includes git pull information
	helpText := syncCmd.Long
	if !strings.Contains(helpText, "Pull template repository") {
		t.Error("Help text should mention git pull functionality")
	}
	if !strings.Contains(helpText, "--no-git-pull") {
		t.Error("Help text should mention --no-git-pull flag")
	}
}

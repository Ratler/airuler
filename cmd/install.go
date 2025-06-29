// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ratler/airuler/internal/compiler"
	"github.com/ratler/airuler/internal/config"
	"github.com/spf13/cobra"
)

var (
	installTarget  string
	installRule    string
	installGlobal  bool
	installProject string
	installForce   bool
)

var installCmd = &cobra.Command{
	Use:   "install [target] [rule]",
	Short: "Install compiled rules to AI coding assistants",
	Long: `Install compiled rules to AI coding assistants.

By default, installs to global configuration directories.
Use --project to install to a specific project directory.

Examples:
  airuler install                           # Install all rules for all targets
  airuler install cursor                    # Install all Cursor rules
  airuler install cursor my-rule            # Install specific Cursor rule
  airuler install --project ./my-project    # Install to project directory`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) >= 1 {
			installTarget = args[0]
		}
		if len(args) >= 2 {
			installRule = args[1]
		}

		return installRules()
	},
}

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().BoolVar(&installGlobal, "global", true, "install to global configuration (default)")
	installCmd.Flags().StringVar(&installProject, "project", "", "install to specific project directory")
	installCmd.Flags().BoolVar(&installForce, "force", false, "overwrite without confirmation")
}

func installRules() error {
	var targets []compiler.Target

	if installTarget != "" {
		target := compiler.Target(installTarget)
		if !isValidTarget(target) {
			return fmt.Errorf("invalid target: %s", installTarget)
		}
		targets = []compiler.Target{target}
	} else {
		targets = compiler.AllTargets
	}

	installed := 0
	for _, target := range targets {
		count, err := installForTarget(target)
		if err != nil {
			fmt.Printf("Warning: failed to install for %s: %v\n", target, err)
			continue
		}
		installed += count
	}

	if installed > 0 {
		fmt.Printf("\n🎉 Successfully installed %d rules\n", installed)
	} else {
		fmt.Println("No rules were installed")
	}

	return nil
}

func installForTarget(target compiler.Target) (int, error) {
	compiledDir := filepath.Join("compiled", string(target))

	if _, err := os.Stat(compiledDir); os.IsNotExist(err) {
		return 0, fmt.Errorf("no compiled rules found for %s. Run 'airuler compile' first", target)
	}

	fmt.Printf("Installing %s rules...\n", target)

	// Find compiled rules
	files, err := os.ReadDir(compiledDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read compiled directory: %w", err)
	}

	// Special handling for Copilot - merge all rules into single file
	if target == compiler.TargetCopilot {
		return installCopilotRules(compiledDir, files)
	}

	installed := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Filter by rule if specified
		if installRule != "" && !strings.Contains(file.Name(), installRule) {
			continue
		}

		sourcePath := filepath.Join(compiledDir, file.Name())

		// Determine mode from filename for Claude target
		mode := "command" // default
		if target == compiler.TargetClaude && file.Name() == "CLAUDE.md" {
			mode = "memory"
		}

		// Get target directory based on mode
		var targetDir string
		var err error
		if installProject != "" {
			targetDir, err = getProjectInstallDirForMode(target, installProject, mode)
		} else {
			targetDir, err = getGlobalInstallDirForMode(target, mode)
		}
		if err != nil {
			fmt.Printf("  ⚠️  Failed to get install directory for %s: %v\n", file.Name(), err)
			continue
		}

		// Ensure target directory exists
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			fmt.Printf("  ⚠️  Failed to create target directory %s: %v\n", targetDir, err)
			continue
		}

		targetPath := filepath.Join(targetDir, file.Name())

		if err := installFileWithMode(sourcePath, targetPath, target, mode); err != nil {
			fmt.Printf("  ⚠️  Failed to install %s: %v\n", file.Name(), err)
			continue
		}

		// Record the installation
		ruleName := installRule
		if ruleName == "" {
			ruleName = "*" // Indicates all rules were installed
		}
		if err := recordInstallation(target, ruleName, targetPath, mode); err != nil {
			fmt.Printf("  ⚠️  Failed to record installation: %v\n", err)
		}

		fmt.Printf("  ✅ %s -> %s\n", file.Name(), targetDir)
		installed++
	}

	return installed, nil
}

func installCopilotRules(compiledDir string, files []os.DirEntry) (int, error) {
	// GitHub Copilot only supports project-level installation
	if installProject == "" {
		return 0, fmt.Errorf("copilot rules can only be installed to projects (use --project flag). Global copilot installation is not supported")
	}

	var ruleContents []string
	var ruleNames []string

	// Collect all copilot rule files
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Filter by rule if specified
		if installRule != "" && !strings.Contains(file.Name(), installRule) {
			continue
		}

		if strings.HasSuffix(file.Name(), ".copilot-instructions.md") {
			sourcePath := filepath.Join(compiledDir, file.Name())
			content, err := os.ReadFile(sourcePath)
			if err != nil {
				fmt.Printf("  ⚠️  Failed to read %s: %v\n", file.Name(), err)
				continue
			}

			ruleContents = append(ruleContents, strings.TrimSpace(string(content)))
			ruleNames = append(ruleNames, strings.TrimSuffix(file.Name(), ".copilot-instructions.md"))
		}
	}

	if len(ruleContents) == 0 {
		return 0, nil
	}

	// Get project directory
	absPath, err := filepath.Abs(installProject)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve project path: %w", err)
	}

	targetDir := filepath.Join(absPath, ".github")
	targetPath := filepath.Join(targetDir, "copilot-instructions.md")

	// Ensure .github directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create .github directory: %w", err)
	}

	// Combine all rules into single content
	var combinedContent strings.Builder
	combinedContent.WriteString("# AI Coding Instructions\n\n")
	combinedContent.WriteString("This file contains custom instructions for GitHub Copilot.\n\n")

	for i, content := range ruleContents {
		if i > 0 {
			combinedContent.WriteString("\n---\n\n")
		}
		if len(ruleNames) > 1 {
			combinedContent.WriteString(fmt.Sprintf("## %s\n\n", ruleNames[i]))
		}
		combinedContent.WriteString(content)
		combinedContent.WriteString("\n")
	}

	// Handle existing file
	if _, err := os.Stat(targetPath); err == nil && !installForce {
		// Create backup
		backupPath := targetPath + ".backup." + time.Now().Format("20060102-150405")
		if err := copyFile(targetPath, backupPath); err != nil {
			return 0, fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("    📋 Backed up existing file to %s\n", filepath.Base(backupPath))
	}

	// Write combined content
	if err := os.WriteFile(targetPath, []byte(combinedContent.String()), 0600); err != nil {
		return 0, fmt.Errorf("failed to write copilot instructions: %w", err)
	}

	// Record installation
	ruleName := installRule
	if ruleName == "" {
		ruleName = "*"
	}
	if err := recordInstallation(compiler.TargetCopilot, ruleName, targetPath, ""); err != nil {
		fmt.Printf("  ⚠️  Failed to record installation: %v\n", err)
	}

	fmt.Printf("  ✅ Combined %d rules -> %s\n", len(ruleContents), targetDir)
	return 1, nil
}

func installFile(source, target string, _ compiler.Target) error {
	// Check if target exists and create backup
	if _, err := os.Stat(target); err == nil && !installForce {
		// Create backup
		backupPath := target + ".backup." + time.Now().Format("20060102-150405")
		if err := copyFile(target, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("    📋 Backed up existing file to %s\n", filepath.Base(backupPath))
	}

	// Copy file
	return copyFile(source, target)
}

func copyFile(source, dest string) error {
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, content, 0600)
}

func getTargetInstallDir(target compiler.Target) (string, error) {
	if installProject != "" {
		return getProjectInstallDir(target, installProject)
	}
	return getGlobalInstallDir(target)
}

func getGlobalInstallDir(target compiler.Target) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	switch target {
	case compiler.TargetCursor:
		switch runtime.GOOS {
		case "darwin":
			return filepath.Join(homeDir, "Library", "Application Support", "Cursor", "User", "globalStorage", "cursor.rules"), nil
		case "windows":
			return filepath.Join(homeDir, "AppData", "Roaming", "Cursor", "User", "globalStorage", "cursor.rules"), nil
		default:
			return filepath.Join(homeDir, ".config", "Cursor", "User", "globalStorage", "cursor.rules"), nil
		}
	case compiler.TargetClaude:
		return filepath.Join(homeDir, ".claude", "commands"), nil
	case compiler.TargetCline:
		return filepath.Join(homeDir, ".clinerules"), nil
	case compiler.TargetCopilot:
		return "", fmt.Errorf("copilot does not support global installation (use --project flag)")
	default:
		return "", fmt.Errorf("unsupported target: %s", target)
	}
}

func getProjectInstallDir(target compiler.Target, projectPath string) (string, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return "", err
	}

	switch target {
	case compiler.TargetCursor:
		return filepath.Join(absPath, ".cursor", "rules"), nil
	case compiler.TargetClaude:
		return filepath.Join(absPath, ".claude", "commands"), nil
	case compiler.TargetCline:
		return filepath.Join(absPath, ".clinerules"), nil
	case compiler.TargetCopilot:
		return filepath.Join(absPath, ".github"), nil
	default:
		return "", fmt.Errorf("unsupported target: %s", target)
	}
}

func getProjectInstallDirForMode(target compiler.Target, projectPath, mode string) (string, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return "", err
	}

	switch target {
	case compiler.TargetClaude:
		if mode == "memory" {
			// For memory mode, install to project root (for CLAUDE.md)
			return absPath, nil
		}
		// For command mode, use .claude/commands/
		return filepath.Join(absPath, ".claude", "commands"), nil
	default:
		// For other targets, mode doesn't matter
		return getProjectInstallDir(target, projectPath)
	}
}

func getGlobalInstallDirForMode(target compiler.Target, mode string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	switch target {
	case compiler.TargetClaude:
		if mode == "memory" {
			// For memory mode, install to home directory (for global CLAUDE.md)
			return homeDir, nil
		}
		// For command mode, use .claude/commands/
		return filepath.Join(homeDir, ".claude", "commands"), nil
	default:
		// For other targets, mode doesn't matter
		return getGlobalInstallDir(target)
	}
}

func installFileWithMode(source, target string, targetType compiler.Target, mode string) error {
	// For memory mode (CLAUDE.md), we need special handling
	if targetType == compiler.TargetClaude && mode == "memory" {
		return installMemoryFile(source, target)
	}

	// For command mode, use regular installation
	return installFile(source, target, targetType)
}

func installMemoryFile(source, target string) error {
	// Read the new content
	newContent, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Check if target file exists
	if _, err := os.Stat(target); err == nil {
		// File exists - append content
		if !installForce {
			// Create backup
			backupPath := target + ".backup." + time.Now().Format("20060102-150405")
			if err := copyFile(target, backupPath); err != nil {
				return fmt.Errorf("failed to create backup: %w", err)
			}
			fmt.Printf("    📋 Backed up existing file to %s\n", filepath.Base(backupPath))
		}

		// Read existing content
		existingContent, err := os.ReadFile(target)
		if err != nil {
			return fmt.Errorf("failed to read existing file: %w", err)
		}

		// Combine content with separator
		combinedContent := strings.TrimSpace(string(existingContent)) + "\n\n" +
			"<!-- Added by airuler -->\n" +
			strings.TrimSpace(string(newContent)) + "\n"

		// Write combined content
		return os.WriteFile(target, []byte(combinedContent), 0600)
	}
	// File doesn't exist - create new
	return os.WriteFile(target, newContent, 0600)
}

func recordInstallation(target compiler.Target, rule, filePath, mode string) error {
	record := config.InstallationRecord{
		Target:      string(target),
		Rule:        rule,
		Global:      installProject == "",
		ProjectPath: installProject,
		Mode:        mode,
		FilePath:    filePath,
		InstalledAt: time.Now(),
	}

	var tracker *config.InstallationTracker
	var err error

	if installProject == "" {
		// Global installation
		tracker, err = config.LoadGlobalInstallationTracker()
		if err != nil {
			return fmt.Errorf("failed to load global installation tracker: %w", err)
		}

		tracker.AddInstallation(record)

		if err := config.SaveGlobalInstallationTracker(tracker); err != nil {
			return fmt.Errorf("failed to save global installation tracker: %w", err)
		}
	} else {
		// Project installation
		tracker, err = config.LoadProjectInstallationTracker()
		if err != nil {
			return fmt.Errorf("failed to load project installation tracker: %w", err)
		}

		tracker.AddInstallation(record)

		if err := config.SaveProjectInstallationTracker(tracker); err != nil {
			return fmt.Errorf("failed to save project installation tracker: %w", err)
		}
	}

	return nil
}

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/ratler/airuler/internal/compiler"
	"github.com/ratler/airuler/internal/config"
	"github.com/ratler/airuler/internal/ui"
	"github.com/ratler/airuler/internal/utils"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Installation flags and variables used by deploy command
var (
	installTarget      string
	installRule        string
	installGlobal      bool
	installProject     string
	installForce       bool
	installInteractive bool
	listFilter         string
)

// Uninstall flags and variables used by manage command
var (
	uninstallTarget      string
	uninstallRule        string
	uninstallGlobal      bool
	uninstallProject     bool
	uninstallForce       bool
	uninstallInteractive bool
)

// Type alias to work around Go compiler parsing issue
type InstallRecord = config.InstallationRecord

// resolveProjectPath resolves the project path relative to the original working directory
// This is needed because setupWorkingDirectory may have changed the current directory
func resolveProjectPath(projectPath string) (string, error) {
	if projectPath == "" {
		return "", nil
	}

	if filepath.IsAbs(projectPath) {
		return projectPath, nil
	}

	// For relative paths, resolve them relative to the original working directory
	originalDir := GetOriginalWorkingDir()
	absPath := filepath.Join(originalDir, projectPath)
	return filepath.Abs(absPath)
}

// installRules is the main installation logic used by deploy command
func installRules() error {
	if installInteractive {
		return runInteractiveInstall()
	}

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
		return 0, fmt.Errorf("no compiled rules found for %s. Run 'airuler sync' or 'airuler deploy' first", target)
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

	// Special handling for Gemini - merge all rules into single file
	if target == compiler.TargetGemini {
		return installGeminiRules(compiledDir, files)
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

		// Determine mode from filename for Claude target only
		mode := "" // default for non-Claude targets
		if target == compiler.TargetClaude {
			mode = "command" // default for Claude
			if file.Name() == "CLAUDE.md" {
				mode = "memory"
			}
		}

		// Get target directory based on mode
		var targetDir string
		var err error
		if installProject != "" {
			resolvedPath, resolveErr := resolveProjectPath(installProject)
			if resolveErr != nil {
				fmt.Printf("  ⚠️  Failed to resolve project path for %s: %v\n", file.Name(), resolveErr)
				continue
			}
			targetDir, err = getProjectInstallDirForMode(target, resolvedPath, mode)
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
			// When installing all templates, use the actual template name from filename
			// Remove the target-specific extension to get the base template name
			baseName := strings.TrimSuffix(file.Name(), ".md")
			baseName = strings.TrimSuffix(baseName, ".mdc")
			ruleName = baseName
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

	// Get project directory
	absPath, err := resolveProjectPath(installProject)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve project path: %w", err)
	}

	targetDir := filepath.Join(absPath, ".github")
	targetPath := filepath.Join(targetDir, "copilot-instructions.md")

	// Collect new rules being installed
	var newRuleContents []string
	var newRuleNames []string

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

			newRuleContents = append(newRuleContents, strings.TrimSpace(string(content)))
			newRuleNames = append(newRuleNames, strings.TrimSuffix(file.Name(), ".copilot-instructions.md"))
		}
	}

	if len(newRuleContents) == 0 {
		return 0, nil
	}

	// Get existing rules from installation tracker
	tracker, err := config.LoadGlobalInstallationTracker()
	if err != nil {
		return 0, fmt.Errorf("failed to load installation tracker: %w", err)
	}

	existingInstalls := tracker.GetInstallations("copilot", "")
	var existingRuleNames []string

	// Filter to only rules for this project
	for _, install := range existingInstalls {
		if !install.Global && install.ProjectPath == absPath {
			existingRuleNames = append(existingRuleNames, install.Rule)
		}
	}

	// Combine existing and new rules, avoiding duplicates
	allRuleNames := make([]string, 0, len(existingRuleNames)+len(newRuleNames))
	allRuleContents := make([]string, 0, len(existingRuleNames)+len(newRuleNames))
	ruleSet := make(map[string]bool)

	// Add existing rules first
	for _, ruleName := range existingRuleNames {
		if !ruleSet[ruleName] {
			// Try to read content from compiled directory
			sourcePath := filepath.Join(compiledDir, ruleName+".copilot-instructions.md")
			content, err := os.ReadFile(sourcePath)
			if err != nil {
				// If we can't find the compiled file, skip this rule
				// This handles cases where the rule was installed but compiled files were cleaned
				continue
			}

			allRuleNames = append(allRuleNames, ruleName)
			allRuleContents = append(allRuleContents, strings.TrimSpace(string(content)))
			ruleSet[ruleName] = true
		}
	}

	// Add new rules, skipping duplicates
	for i, ruleName := range newRuleNames {
		if !ruleSet[ruleName] {
			allRuleNames = append(allRuleNames, ruleName)
			allRuleContents = append(allRuleContents, newRuleContents[i])
			ruleSet[ruleName] = true
		}
	}

	// Ensure .github directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create .github directory: %w", err)
	}

	// Handle existing file backup
	if _, err := os.Stat(targetPath); err == nil && !installForce {
		// Create backup
		backupPath := targetPath + ".backup." + time.Now().Format("20060102-150405")
		if err := copyFile(targetPath, backupPath); err != nil {
			return 0, fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("    📋 Backed up existing file to %s\n", filepath.Base(backupPath))
	}

	// Combine all rules into single content
	var combinedContent strings.Builder
	combinedContent.WriteString("# AI Coding Instructions\n\n")
	combinedContent.WriteString("This file contains custom instructions for GitHub Copilot.\n\n")

	for i, content := range allRuleContents {
		if i > 0 {
			combinedContent.WriteString("\n---\n\n")
		}
		if len(allRuleContents) > 1 {
			combinedContent.WriteString(fmt.Sprintf("## %s\n\n", allRuleNames[i]))
		}
		combinedContent.WriteString(content)
		combinedContent.WriteString("\n")
	}

	// Write combined content
	if err := os.WriteFile(targetPath, []byte(combinedContent.String()), 0600); err != nil {
		return 0, fmt.Errorf("failed to write copilot instructions: %w", err)
	}

	// Record installation for each NEW template that was added
	var newlyInstalledCount int
	if installRule != "" {
		// If specific rule was requested, check if it's actually new
		wasExisting := slices.Contains(existingRuleNames, installRule)

		if !wasExisting {
			if err := recordInstallation(compiler.TargetCopilot, installRule, targetPath, ""); err != nil {
				fmt.Printf("  ⚠️  Failed to record installation: %v\n", err)
			} else {
				newlyInstalledCount = 1
			}
		}
	} else {
		// Record each new template that was added
		for _, ruleName := range newRuleNames {
			// Only record if this rule wasn't already installed
			wasExisting := slices.Contains(existingRuleNames, ruleName)

			if !wasExisting {
				if err := recordInstallation(compiler.TargetCopilot, ruleName, targetPath, ""); err != nil {
					fmt.Printf("  ⚠️  Failed to record installation: %v\n", err)
				} else {
					newlyInstalledCount++
				}
			}
		}
	}

	if newlyInstalledCount > 0 {
		fmt.Printf("  ✅ Combined %d new + %d existing rules -> %s\n", newlyInstalledCount, len(existingRuleNames), targetDir)
	} else {
		fmt.Printf("  ✅ No new rules to install (all %d rules already present) -> %s\n", len(allRuleNames), targetDir)
	}

	return 1, nil
}

func installGeminiRules(compiledDir string, files []os.DirEntry) (int, error) {
	// Get target directory (global or project)
	targetDir, err := getTargetInstallDir(compiler.TargetGemini)
	if err != nil {
		return 0, fmt.Errorf("failed to get Gemini install directory: %w", err)
	}

	targetPath := filepath.Join(targetDir, "GEMINI.md")

	// Collect new rules being installed
	var newRuleContents []string
	var newRuleNames []string

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Filter by rule if specified
		if installRule != "" && !strings.Contains(file.Name(), installRule) {
			continue
		}

		if strings.HasSuffix(file.Name(), ".md") {
			sourcePath := filepath.Join(compiledDir, file.Name())
			content, err := os.ReadFile(sourcePath)
			if err != nil {
				fmt.Printf("  ⚠️  Failed to read %s: %v\n", file.Name(), err)
				continue
			}

			newRuleContents = append(newRuleContents, strings.TrimSpace(string(content)))
			newRuleNames = append(newRuleNames, strings.TrimSuffix(file.Name(), ".md"))
		}
	}

	if len(newRuleContents) == 0 {
		return 0, nil
	}

	// Get existing rules from installation tracker
	tracker, err := config.LoadGlobalInstallationTracker()
	if err != nil {
		return 0, fmt.Errorf("failed to load installation tracker: %w", err)
	}

	var projectPath string
	isGlobal := installProject == ""
	if !isGlobal {
		projectPath, err = resolveProjectPath(installProject)
		if err != nil {
			return 0, fmt.Errorf("failed to resolve project path: %w", err)
		}
	}

	existingInstalls := tracker.GetInstallations("gemini", "")
	var existingRuleNames []string

	// Filter to only rules for this installation context (global vs project)
	for _, install := range existingInstalls {
		if isGlobal && install.Global {
			existingRuleNames = append(existingRuleNames, install.Rule)
		} else if !isGlobal && !install.Global && install.ProjectPath == projectPath {
			existingRuleNames = append(existingRuleNames, install.Rule)
		}
	}

	// Combine existing and new rules, avoiding duplicates
	allRuleNames := make([]string, 0, len(existingRuleNames)+len(newRuleNames))
	allRuleContents := make([]string, 0, len(existingRuleNames)+len(newRuleNames))
	ruleSet := make(map[string]bool)

	// Add existing rules first
	for _, ruleName := range existingRuleNames {
		if !ruleSet[ruleName] {
			// Try to read content from compiled directory
			sourcePath := filepath.Join(compiledDir, ruleName+".md")
			content, err := os.ReadFile(sourcePath)
			if err != nil {
				// If we can't find the compiled file, skip this rule
				// This handles cases where the rule was installed but compiled files were cleaned
				continue
			}

			allRuleNames = append(allRuleNames, ruleName)
			allRuleContents = append(allRuleContents, strings.TrimSpace(string(content)))
			ruleSet[ruleName] = true
		}
	}

	// Add new rules, skipping duplicates
	for i, ruleName := range newRuleNames {
		if !ruleSet[ruleName] {
			allRuleNames = append(allRuleNames, ruleName)
			allRuleContents = append(allRuleContents, newRuleContents[i])
			ruleSet[ruleName] = true
		}
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create Gemini directory: %w", err)
	}

	// Handle existing file backup
	if _, err := os.Stat(targetPath); err == nil && !installForce {
		// Create backup
		backupPath := targetPath + ".backup." + time.Now().Format("20060102-150405")
		if err := copyFile(targetPath, backupPath); err != nil {
			return 0, fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("    📋 Backed up existing file to %s\n", filepath.Base(backupPath))
	}

	// Combine all rules into single content
	var combinedContent strings.Builder
	combinedContent.WriteString("# AI Coding Instructions\n\n")
	combinedContent.WriteString("This file contains custom instructions for Gemini CLI.\n\n")

	for i, content := range allRuleContents {
		if i > 0 {
			combinedContent.WriteString("\n---\n\n")
		}
		if len(allRuleContents) > 1 {
			combinedContent.WriteString(fmt.Sprintf("## %s\n\n", allRuleNames[i]))
		}
		combinedContent.WriteString(content)
		combinedContent.WriteString("\n")
	}

	// Write combined content
	if err := os.WriteFile(targetPath, []byte(combinedContent.String()), 0600); err != nil {
		return 0, fmt.Errorf("failed to write Gemini instructions: %w", err)
	}

	// Record installation for each NEW template that was added
	var newlyInstalledCount int
	if installRule != "" {
		// If specific rule was requested, check if it's actually new
		wasExisting := slices.Contains(existingRuleNames, installRule)

		if !wasExisting {
			mode := ""
			if !isGlobal {
				mode = projectPath
			}
			if err := recordInstallation(compiler.TargetGemini, installRule, targetPath, mode); err != nil {
				fmt.Printf("  ⚠️  Failed to record installation: %v\n", err)
			} else {
				newlyInstalledCount = 1
			}
		}
	} else {
		// Record each new template that was added
		for _, ruleName := range newRuleNames {
			// Only record if this rule wasn't already installed
			wasExisting := slices.Contains(existingRuleNames, ruleName)

			if !wasExisting {
				mode := ""
				if !isGlobal {
					mode = projectPath
				}
				if err := recordInstallation(compiler.TargetGemini, ruleName, targetPath, mode); err != nil {
					fmt.Printf("  ⚠️  Failed to record installation: %v\n", err)
				} else {
					newlyInstalledCount++
				}
			}
		}
	}

	if newlyInstalledCount > 0 {
		fmt.Printf("  ✅ Combined %d new + %d existing rules -> %s\n", newlyInstalledCount, len(existingRuleNames), targetDir)
	} else {
		fmt.Printf("  ✅ No new rules to install (all %d rules already present) -> %s\n", len(allRuleNames), targetDir)
	}

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
		resolvedPath, err := resolveProjectPath(installProject)
		if err != nil {
			return "", err
		}
		return getProjectInstallDir(target, resolvedPath)
	}
	return getGlobalInstallDir(target)
}

func getRooGlobalPath() string {
	homeDir, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		return filepath.Join(homeDir, ".roo", "rules")
	}
	return filepath.Join(homeDir, ".roo", "rules")
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
	case compiler.TargetGemini:
		return filepath.Join(homeDir, ".gemini"), nil
	case compiler.TargetRoo:
		return getRooGlobalPath(), nil
	default:
		return "", fmt.Errorf("unsupported target: %s", target)
	}
}

func getProjectInstallDir(target compiler.Target, projectPath string) (string, error) {
	// Handle both absolute and relative paths correctly
	var absPath string
	if filepath.IsAbs(projectPath) {
		absPath = projectPath
	} else {
		// For relative paths, resolve them relative to the original working directory
		// This handles cases where installation records contain relative paths
		originalDir := GetOriginalWorkingDir()
		resolvedPath := filepath.Join(originalDir, projectPath)
		var err error
		absPath, err = filepath.Abs(resolvedPath)
		if err != nil {
			return "", err
		}
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
	case compiler.TargetGemini:
		return absPath, nil
	case compiler.TargetRoo:
		return filepath.Join(absPath, ".roo", "rules"), nil
	default:
		return "", fmt.Errorf("unsupported target: %s", target)
	}
}

func getProjectInstallDirForMode(target compiler.Target, projectPath, mode string) (string, error) {
	// Handle both absolute and relative paths correctly
	var absPath string
	if filepath.IsAbs(projectPath) {
		absPath = projectPath
	} else {
		// For relative paths, resolve them relative to the original working directory
		// This handles cases where installation records contain relative paths
		originalDir := GetOriginalWorkingDir()
		resolvedPath := filepath.Join(originalDir, projectPath)
		var err error
		absPath, err = filepath.Abs(resolvedPath)
		if err != nil {
			return "", err
		}
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
	// Convert project path to absolute path if it's a project installation
	var projectPath string
	if installProject != "" {
		absPath, err := resolveProjectPath(installProject)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for project: %w", err)
		}
		projectPath = absPath
	}

	record := config.InstallationRecord{
		Target:      string(target),
		Rule:        rule,
		Global:      installProject == "",
		ProjectPath: projectPath,
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

// installSelectionItem represents a template available for installation
type installSelectionItem struct {
	displayText string
	target      compiler.Target
	rule        string
	sourcePath  string
	mode        string // For Claude templates
	isInstalled bool
}

func runInteractiveInstall() error {
	// Load all available templates
	installItems, err := loadAvailableTemplates()
	if err != nil {
		return err
	}

	if len(installItems) == 0 {
		if installProject != "" {
			projectName := filepath.Base(installProject)
			fmt.Printf("📋 No compiled templates found for project installation (%s). Run 'airuler sync' or 'airuler deploy' first.\n", projectName)
		} else {
			fmt.Println("📋 No compiled templates found for global installation. Run 'airuler sync' or 'airuler deploy' first.")
		}
		return nil
	}

	// Convert to generic interactive items
	var items []ui.InteractiveItem
	for _, item := range installItems {
		items = append(items, ui.InteractiveItem{
			DisplayText: item.displayText,
			ID:          fmt.Sprintf("%s:%s", item.target, item.rule),
			Data:        item,
			IsInstalled: item.isInstalled,
		})
	}

	// Determine installation scope for title
	var scopeText string
	if installProject != "" {
		projectName := filepath.Base(installProject)
		scopeText = fmt.Sprintf(" (installing to project: %s)", projectName)
	} else {
		scopeText = " (installing globally)"
	}

	// Custom formatter for install items
	formatter := func(item ui.InteractiveItem, cursor, checkbox string) string {
		installItem := item.Data.(installSelectionItem)

		// Format row with proper column widths
		target := string(installItem.target)
		rule := installItem.rule
		mode := installItem.mode
		if mode == "" {
			mode = "-"
		}
		status := ""
		if installItem.isInstalled {
			status = "installed"
		}

		// Truncate long strings
		if len(rule) > 25 {
			rule = rule[:22] + "..."
		}

		return fmt.Sprintf("%s %s %-8s %-25s %-8s %-10s",
			cursor, checkbox, target, rule, mode, status)
	}

	// Create interactive config
	config := ui.InteractiveConfig{
		Title:        "Select templates to install:" + scopeText,
		Instructions: "↑/↓: navigate • space: toggle • enter: confirm • q: quit",
		Items:        items,
		Formatter:    formatter,
	}

	// Run interactive selection
	selectedItems, cancelled, err := ui.RunInteractiveSelection(config)
	if err != nil {
		return fmt.Errorf("interactive selection failed: %w", err)
	}

	if cancelled {
		if installProject != "" {
			projectName := filepath.Base(installProject)
			fmt.Printf("Installation cancelled (project: %s)\n", projectName)
		} else {
			fmt.Println("Installation cancelled (global)")
		}
		return nil
	}

	if len(selectedItems) == 0 {
		if installProject != "" {
			projectName := filepath.Base(installProject)
			fmt.Printf("No templates selected for installation (project: %s)\n", projectName)
		} else {
			fmt.Println("No templates selected for installation (global)")
		}
		return nil
	}

	// Convert back to install selection items
	var installSelectionItems []installSelectionItem
	for _, item := range selectedItems {
		if !item.IsInstalled {
			installSelectionItems = append(installSelectionItems, item.Data.(installSelectionItem))
		}
	}

	// Perform installations
	return performInteractiveInstallations(installSelectionItems)
}

func loadAvailableTemplates() ([]installSelectionItem, error) {
	var items []installSelectionItem

	// Load current installations to check what's already installed
	tracker, _ := config.LoadGlobalInstallationTracker()
	installations := tracker.GetInstallations("", "")

	// Create a map for quick lookup of installed templates
	installedMap := make(map[string]bool)
	for _, install := range installations {
		key := fmt.Sprintf("%s:%s:%t:%s", install.Target, install.Rule, install.Global, install.ProjectPath)
		installedMap[key] = true
	}

	// Group templates by target
	groups := make(map[compiler.Target][]installSelectionItem)

	// Process each target
	targets := compiler.AllTargets
	if installTarget != "" {
		target := compiler.Target(installTarget)
		if isValidTarget(target) {
			targets = []compiler.Target{target}
		}
	}

	for _, target := range targets {
		compiledDir := filepath.Join("compiled", string(target))

		// Skip if directory doesn't exist
		if _, err := os.Stat(compiledDir); os.IsNotExist(err) {
			continue
		}

		files, err := os.ReadDir(compiledDir)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			// Filter by rule if specified
			if installRule != "" && !strings.Contains(file.Name(), installRule) {
				continue
			}

			// Extract rule name
			ruleName := strings.TrimSuffix(file.Name(), ".md")
			ruleName = strings.TrimSuffix(ruleName, ".mdc")
			ruleName = strings.TrimSuffix(ruleName, ".copilot-instructions")

			// Determine mode for Claude
			mode := ""
			if target == compiler.TargetClaude {
				mode = "command"
				if file.Name() == "CLAUDE.md" {
					mode = "memory"
				}
			}

			// Check if already installed
			var projectPath string
			if installProject != "" {
				absPath, _ := resolveProjectPath(installProject)
				projectPath = absPath
			}
			installKey := fmt.Sprintf("%s:%s:%t:%s", target, ruleName, installProject == "", projectPath)
			isInstalled := installedMap[installKey]

			item := installSelectionItem{
				target:      target,
				rule:        ruleName,
				sourcePath:  filepath.Join(compiledDir, file.Name()),
				mode:        mode,
				isInstalled: isInstalled,
			}

			groups[target] = append(groups[target], item)
		}
	}

	// Sort targets for consistent display
	var sortedTargets []compiler.Target
	for target := range groups {
		sortedTargets = append(sortedTargets, target)
	}
	sort.Slice(sortedTargets, func(i, j int) bool {
		return string(sortedTargets[i]) < string(sortedTargets[j])
	})

	// Build final item list with group headers
	for _, target := range sortedTargets {
		targetItems := groups[target]

		// Sort items within target
		sort.Slice(targetItems, func(i, j int) bool {
			return targetItems[i].rule < targetItems[j].rule
		})

		// Add group header for target
		items = append(items, installSelectionItem{
			displayText: fmt.Sprintf("GROUP_HEADER:📦 %s", cases.Title(language.English).String(string(target))),
		})

		items = append(items, targetItems...)
	}

	return items, nil
}

func performInteractiveInstallations(selectedItems []installSelectionItem) error {
	if installProject != "" {
		projectName := filepath.Base(installProject)
		fmt.Printf("\n🚀 Installing selected templates to project: %s...\n", projectName)
	} else {
		fmt.Println("\n🚀 Installing selected templates globally...")
	}

	// Group by target for Copilot special handling
	targetGroups := make(map[compiler.Target][]installSelectionItem)
	for _, item := range selectedItems {
		targetGroups[item.target] = append(targetGroups[item.target], item)
	}

	installed := 0
	failed := 0

	// Handle Copilot specially (needs to merge files)
	if copilotItems, ok := targetGroups[compiler.TargetCopilot]; ok {
		// Copilot requires project installation
		if installProject == "" {
			fmt.Printf("  ⚠️  Copilot templates can only be installed to projects (use --project flag)\n")
			failed += len(copilotItems)
		} else {
			// Prepare files for Copilot installation
			var files []os.DirEntry
			for _, item := range copilotItems {
				// Create a fake DirEntry for the file
				info, err := os.Stat(item.sourcePath)
				if err != nil {
					fmt.Printf("  ⚠️  Failed to stat %s: %v\n", item.rule, err)
					failed++
					continue
				}
				files = append(files, fakeFileInfo{name: filepath.Base(item.sourcePath), FileInfo: info})
			}

			compiledDir := filepath.Join("compiled", string(compiler.TargetCopilot))
			count, err := installCopilotRules(compiledDir, files)
			if err != nil {
				fmt.Printf("  ⚠️  Failed to install Copilot templates: %v\n", err)
				failed += len(copilotItems)
			} else {
				installed += count
			}
		}
		delete(targetGroups, compiler.TargetCopilot)
	}

	// Handle Gemini specially (needs to merge files)
	if geminiItems, ok := targetGroups[compiler.TargetGemini]; ok {
		// Prepare files for Gemini installation
		var files []os.DirEntry
		for _, item := range geminiItems {
			// Create a fake DirEntry for the file
			info, err := os.Stat(item.sourcePath)
			if err != nil {
				fmt.Printf("  ⚠️  Failed to stat %s: %v\n", item.rule, err)
				failed++
				continue
			}
			files = append(files, fakeFileInfo{name: filepath.Base(item.sourcePath), FileInfo: info})
		}

		compiledDir := filepath.Join("compiled", string(compiler.TargetGemini))
		count, err := installGeminiRules(compiledDir, files)
		if err != nil {
			fmt.Printf("  ⚠️  Failed to install Gemini templates: %v\n", err)
			failed += len(geminiItems)
		} else {
			installed += count
		}
		delete(targetGroups, compiler.TargetGemini)
	}

	// Handle other targets
	for target, items := range targetGroups {
		for _, item := range items {
			// Get target directory based on mode
			var targetDir string
			var err error
			if installProject != "" {
				resolvedPath, resolveErr := resolveProjectPath(installProject)
				if resolveErr != nil {
					fmt.Printf("  ⚠️  Failed to resolve project path for %s: %v\n", item.rule, resolveErr)
					failed++
					continue
				}
				targetDir, err = getProjectInstallDirForMode(target, resolvedPath, item.mode)
			} else {
				targetDir, err = getGlobalInstallDirForMode(target, item.mode)
			}
			if err != nil {
				fmt.Printf("  ⚠️  Failed to get install directory for %s: %v\n", item.rule, err)
				failed++
				continue
			}

			// Ensure target directory exists
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				fmt.Printf("  ⚠️  Failed to create directory %s: %v\n", targetDir, err)
				failed++
				continue
			}

			targetPath := filepath.Join(targetDir, filepath.Base(item.sourcePath))

			if err := installFileWithMode(item.sourcePath, targetPath, target, item.mode); err != nil {
				fmt.Printf("  ⚠️  Failed to install %s: %v\n", item.rule, err)
				failed++
				continue
			}

			// Record the installation
			if err := recordInstallation(target, item.rule, targetPath, item.mode); err != nil {
				fmt.Printf("  ⚠️  Failed to record installation: %v\n", err)
			}

			fmt.Printf("  ✅ %s %s -> %s\n", target, item.rule, targetDir)
			installed++
		}
	}

	if installProject != "" {
		projectName := filepath.Base(installProject)
		fmt.Printf("\n🎉 Installed %d templates to project: %s", installed, projectName)
	} else {
		fmt.Printf("\n🎉 Installed %d templates globally", installed)
	}
	if failed > 0 {
		fmt.Printf(", %d failed", failed)
	}
	fmt.Println()

	return nil
}

// fakeFileInfo implements os.DirEntry for interactive mode
type fakeFileInfo struct {
	name string
	os.FileInfo
}

func (f fakeFileInfo) Name() string               { return f.name }
func (f fakeFileInfo) IsDir() bool                { return false }
func (f fakeFileInfo) Type() os.FileMode          { return f.Mode() }
func (f fakeFileInfo) Info() (os.FileInfo, error) { return f.FileInfo, nil }

// updateSingleInstallationWithStatus updates a single installation and returns status
func updateSingleInstallationWithStatus(installation config.InstallationRecord) (string, error) {
	target := compiler.Target(installation.Target)

	// Validate target
	if !isValidTarget(target) {
		return "failed", fmt.Errorf("invalid target: %s", installation.Target)
	}

	// Find the compiled rule files
	compiledDir := filepath.Join("compiled", string(target))
	files, err := os.ReadDir(compiledDir)
	if err != nil {
		return "failed", fmt.Errorf("failed to read compiled directory: %w", err)
	}

	var sourceFiles []string
	if installation.Rule == "*" {
		// For wildcard installations, install all files for this target
		for _, file := range files {
			if !file.IsDir() {
				sourceFiles = append(sourceFiles, filepath.Join(compiledDir, file.Name()))
			}
		}
	} else {
		// Find the specific compiled rule file
		for _, file := range files {
			if strings.Contains(file.Name(), installation.Rule) {
				sourceFiles = append(sourceFiles, filepath.Join(compiledDir, file.Name()))
				break
			}
		}
	}

	if len(sourceFiles) == 0 {
		return "failed", fmt.Errorf("no compiled rules found for %s", installation.Rule)
	}

	// Determine the target directory based on the original installation
	var targetDir string

	if installation.Global {
		targetDir, err = getGlobalInstallDirForMode(target, installation.Mode)
	} else {
		if installation.ProjectPath == "" {
			return "failed", fmt.Errorf("project path not specified for project installation")
		}
		targetDir, err = getProjectInstallDirForMode(target, installation.ProjectPath, installation.Mode)
	}

	if err != nil {
		return "failed", fmt.Errorf("failed to get target directory: %w", err)
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "failed", fmt.Errorf("failed to create target directory: %w", err)
	}

	// For update-installed, we always force overwrite since we're updating
	originalForce := installForce
	installForce = true
	defer func() { installForce = originalForce }()

	// Install all the files (only if they have changed)
	filesChanged := false
	filesInstalled := false
	for _, sourceFile := range sourceFiles {
		targetPath := filepath.Join(targetDir, filepath.Base(sourceFile))

		// Check if target file exists
		_, err := os.Stat(targetPath)
		fileExists := !os.IsNotExist(err)

		// Check if file has changed before replacing
		if hasFileChanged, err := hasFileContentChanged(sourceFile, targetPath); err != nil {
			return "failed", fmt.Errorf("failed to check file changes for %s: %w", filepath.Base(sourceFile), err)
		} else if hasFileChanged {
			if err := installFileWithMode(sourceFile, targetPath, target, installation.Mode); err != nil {
				return "failed", fmt.Errorf("failed to install file %s: %w", filepath.Base(sourceFile), err)
			}
			if !fileExists {
				filesInstalled = true
			} else {
				filesChanged = true
			}
		}
	}

	// Update the installation record timestamp and return appropriate status
	if filesInstalled {
		// Update timestamp to current time for new installations
		installation.InstalledAt = time.Now()
		if err := updateInstallationRecord(installation); err != nil {
			// Don't fail the whole operation for this, just warn
			fmt.Printf("    Warning: failed to update installation record: %v\n", err)
		}
		return "installed", nil
	} else if filesChanged {
		// Update timestamp to current time for changed files
		installation.InstalledAt = time.Now()
		if err := updateInstallationRecord(installation); err != nil {
			// Don't fail the whole operation for this, just warn
			fmt.Printf("    Warning: failed to update installation record: %v\n", err)
		}
		return "updated", nil
	}

	return "unchanged", nil
}

func updateInstallationRecord(installation config.InstallationRecord) error {
	var tracker *config.InstallationTracker
	var err error

	if installation.Global {
		tracker, err = config.LoadGlobalInstallationTracker()
		if err != nil {
			return err
		}
	} else {
		tracker, err = config.LoadProjectInstallationTracker()
		if err != nil {
			return err
		}
	}

	// Update the installation record
	tracker.AddInstallation(installation) // This will replace the existing record

	if installation.Global {
		return config.SaveGlobalInstallationTracker(tracker)
	}
	return config.SaveProjectInstallationTracker(tracker)
}

// hasFileContentChanged compares the SHA256 hash of source and target files
// Returns true if files are different or target doesn't exist
func hasFileContentChanged(sourceFile, targetFile string) (bool, error) {
	// If target doesn't exist, it's considered changed
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		return true, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to stat target file: %w", err)
	}

	// Calculate hash of source file
	sourceHash, err := calculateFileHash(sourceFile)
	if err != nil {
		return false, fmt.Errorf("failed to calculate source file hash: %w", err)
	}

	// Calculate hash of target file
	targetHash, err := calculateFileHash(targetFile)
	if err != nil {
		return false, fmt.Errorf("failed to calculate target file hash: %w", err)
	}

	// Compare hashes
	return sourceHash != targetHash, nil
}

// calculateFileHash computes SHA256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// runListInstalled displays all installed templates (used by manage command)
func runListInstalled() error {
	// Load global installation tracker
	globalTracker, err := config.LoadGlobalInstallationTracker()
	if err != nil {
		return fmt.Errorf("failed to load global installation tracker: %w", err)
	}

	// Load project installation tracker if in a project
	var projectTracker *config.InstallationTracker
	projectTracker, _ = config.LoadProjectInstallationTracker()

	// Collect and deduplicate installations
	uniqueMap := make(map[string]uniqueInstall)

	// Process global installations
	for _, record := range globalTracker.Installations {
		if shouldIncludeRecord(record, listFilter) {
			key := fmt.Sprintf("%s-%s-%s-%s-global", record.Target, record.Rule, record.Mode, record.FilePath)
			if existing, exists := uniqueMap[key]; !exists || record.InstalledAt.After(existing.InstalledAt) {
				uniqueMap[key] = uniqueInstall{
					Target:      record.Target,
					Rule:        record.Rule,
					Mode:        record.Mode,
					FilePath:    record.FilePath,
					Global:      true,
					InstalledAt: record.InstalledAt,
				}
			}
		}
	}

	// Process project installations
	if projectTracker != nil {
		for _, record := range projectTracker.Installations {
			if shouldIncludeRecord(record, listFilter) {
				key := fmt.Sprintf(
					"%s-%s-%s-%s-%s",
					record.Target,
					record.Rule,
					record.Mode,
					record.FilePath,
					record.ProjectPath,
				)
				if existing, exists := uniqueMap[key]; !exists || record.InstalledAt.After(existing.InstalledAt) {
					uniqueMap[key] = uniqueInstall{
						Target:      record.Target,
						Rule:        record.Rule,
						Mode:        record.Mode,
						FilePath:    record.FilePath,
						Global:      false,
						ProjectPath: record.ProjectPath,
						InstalledAt: record.InstalledAt,
					}
				}
			}
		}
	}

	// Convert map to slice
	var allInstalls []uniqueInstall
	for _, install := range uniqueMap {
		allInstalls = append(allInstalls, install)
	}

	// Check if no templates are installed
	if len(allInstalls) == 0 {
		if listFilter != "" {
			fmt.Println("🔍 No installed templates found matching filter:", listFilter)
		} else {
			fmt.Println("📋 No templates are currently installed")
		}
		return nil
	}

	// Group installations by scope (global vs project)
	var globalInstalls []uniqueInstall
	projectInstalls := make(map[string][]uniqueInstall)

	for _, install := range allInstalls {
		if install.Global {
			globalInstalls = append(globalInstalls, install)
		} else {
			projectInstalls[install.ProjectPath] = append(projectInstalls[install.ProjectPath], install)
		}
	}

	// Sort installations
	sortInstalls := func(installs []uniqueInstall) {
		sort.Slice(installs, func(i, j int) bool {
			if installs[i].Target != installs[j].Target {
				return installs[i].Target < installs[j].Target
			}
			if installs[i].Rule != installs[j].Rule {
				return installs[i].Rule < installs[j].Rule
			}
			return installs[i].Mode < installs[j].Mode
		})
	}

	// Check for missing files
	var missingFiles int
	for i := range allInstalls {
		if _, err := os.Stat(allInstalls[i].FilePath); os.IsNotExist(err) {
			missingFiles++
		}
	}

	// Display header
	fmt.Println("📋 Installed Templates")
	if listFilter != "" {
		fmt.Printf("🔍 Filter: \"%s\"\n", listFilter)
	}
	if missingFiles > 0 {
		fmt.Printf("⚠️  Warning: %d template file(s) are missing\n", missingFiles)
	}
	fmt.Println()

	// Display global installations
	if len(globalInstalls) > 0 {
		fmt.Println("🌍 Global Installations")
		fmt.Println(strings.Repeat("=", 78))
		sortInstalls(globalInstalls)
		displayTable(globalInstalls)
		fmt.Println()
	}

	// Display project installations
	if len(projectInstalls) > 0 {
		// Sort project paths for consistent output
		var projectPaths []string
		for path := range projectInstalls {
			projectPaths = append(projectPaths, path)
		}
		sort.Strings(projectPaths)

		for _, projPath := range projectPaths {
			// Skip empty project paths
			if projPath == "" {
				continue
			}
			// Display only the project name (last directory) instead of full path
			projectName := filepath.Base(projPath)
			fmt.Printf("📁 Project: %s\n", projectName)
			fmt.Println(strings.Repeat("=", 78))
			installs := projectInstalls[projPath]
			sortInstalls(installs)
			displayTable(installs)
			fmt.Println()
		}
	}

	// Display summary
	fmt.Printf("Total: %d template(s) installed\n", len(allInstalls))

	return nil
}

type uniqueInstall struct {
	Target      string
	Rule        string
	Mode        string
	FilePath    string
	Global      bool
	ProjectPath string
	InstalledAt time.Time
}

func displayTable(installs []uniqueInstall) {
	// Print table header with wider columns
	fmt.Printf("%-8s %-20s %-8s %-25s %-15s\n", "Target", "Rule", "Mode", "File", "Installed")
	fmt.Println(strings.Repeat("-", 78))

	// Print each row
	for _, install := range installs {
		target := install.Target
		rule := install.Rule
		if rule == "*" {
			rule = "all templates"
		}

		mode := install.Mode
		if mode == "" {
			mode = "-"
		}

		fileName := filepath.Base(install.FilePath)

		// Check if file exists and add indicator
		if _, err := os.Stat(install.FilePath); os.IsNotExist(err) {
			fileName = fileName + " ⚠️"
		}

		timeAgo := utils.FormatTimeAgo(install.InstalledAt)

		// Truncate long strings
		if len(rule) > 20 {
			rule = rule[:17] + "..."
		}
		if len(fileName) > 25 {
			fileName = fileName[:22] + "..."
		}

		fmt.Printf("%-8s %-20s %-8s %-25s %-15s\n", target, rule, mode, fileName, timeAgo)
	}
}

func shouldIncludeRecord(record config.InstallationRecord, filter string) bool {
	if filter == "" {
		return true
	}

	// Case-insensitive search
	filter = strings.ToLower(filter)

	// Check rule name
	if strings.Contains(strings.ToLower(record.Rule), filter) {
		return true
	}

	// Check target
	if strings.Contains(strings.ToLower(record.Target), filter) {
		return true
	}

	// Check file path
	if strings.Contains(strings.ToLower(record.FilePath), filter) {
		return true
	}

	// Check mode
	if record.Mode != "" && strings.Contains(strings.ToLower(record.Mode), filter) {
		return true
	}

	return false
}

// ============================================================================
// UNINSTALL FUNCTIONS
// ============================================================================

type selectionItem struct {
	displayText  string
	installation config.InstallationRecord
}

func uninstallRules() error {
	// Load all installations
	allInstallations, err := loadInstallations()
	if err != nil {
		return err
	}

	if len(allInstallations) == 0 {
		fmt.Println("📋 No tracked installations found to uninstall")
		return nil
	}

	// Choose mode based on flags
	var selectedInstallations []config.InstallationRecord

	if uninstallInteractive {
		// Interactive mode: Use checkbox selection
		selectedInstallations, err = runInteractiveSelection(allInstallations)
		if err != nil {
			return err
		}
	} else {
		// Default or force mode: Use all filtered installations
		selectedInstallations = allInstallations

		if !uninstallForce {
			// Default mode: Show files and confirm
			if !showUninstallPreviewAndConfirm(selectedInstallations) {
				fmt.Println("Uninstallation cancelled")
				return nil
			}
		}
	}

	if len(selectedInstallations) == 0 {
		fmt.Println("No installations selected for removal")
		return nil
	}

	// Perform the uninstallation
	return performUninstallation(selectedInstallations)
}

func loadInstallations() ([]config.InstallationRecord, error) {
	var allInstallations []config.InstallationRecord

	// Load installation tracker
	tracker, err := config.LoadGlobalInstallationTracker()
	if err != nil {
		return nil, fmt.Errorf("failed to load installation tracker: %w", err)
	}

	installations := tracker.GetInstallations(uninstallTarget, uninstallRule)

	// Filter by installation type if specified
	if uninstallGlobal {
		// Only include global installations
		for _, install := range installations {
			if install.Global {
				allInstallations = append(allInstallations, install)
			}
		}
	} else if uninstallProject {
		// Only include project installations
		for _, install := range installations {
			if !install.Global {
				allInstallations = append(allInstallations, install)
			}
		}
	} else {
		// Include all installations
		allInstallations = installations
	}

	return allInstallations, nil
}

// Helper function to parse display text back into components for table display
func parseDisplayText(_ string, installation config.InstallationRecord) (target, rule, mode, fileName, timeAgo string) {
	target = installation.Target
	rule = installation.Rule
	if rule == "*" {
		rule = "all templates"
	}

	mode = installation.Mode
	if mode == "" {
		mode = "-"
	}

	fileName = filepath.Base(installation.FilePath)
	timeAgo = utils.FormatTimeAgo(installation.InstalledAt)

	// Truncate long strings to fit table columns
	if len(rule) > 20 {
		rule = rule[:17] + "..."
	}
	if len(fileName) > 25 {
		fileName = fileName[:22] + "..."
	}

	return target, rule, mode, fileName, timeAgo
}

func runInteractiveSelection(installations []config.InstallationRecord) ([]config.InstallationRecord, error) {
	// Convert installations to selection items
	selectionItems := prepareSelectionItems(installations)

	if len(selectionItems) == 0 {
		return nil, nil
	}

	// Convert to generic interactive items
	var items []ui.InteractiveItem
	for _, item := range selectionItems {
		items = append(items, ui.InteractiveItem{
			DisplayText: item.displayText,
			ID:          fmt.Sprintf("%s:%s", item.installation.Target, item.installation.Rule),
			Data:        item.installation,
			IsInstalled: false, // For uninstall, all items are "installed" but we treat them as uninstallable
		})
	}

	// Custom formatter for uninstall items
	formatter := func(item ui.InteractiveItem, cursor, checkbox string) string {
		installation := item.Data.(config.InstallationRecord)
		target, rule, mode, fileName, timeAgo := parseDisplayText("", installation)

		return fmt.Sprintf("%s %s %-8s %-20s %-8s %-25s %-15s",
			cursor, checkbox, target, rule, mode, fileName, timeAgo)
	}

	// Create interactive config
	config := ui.InteractiveConfig{
		Title:        "Select templates to uninstall:",
		Instructions: "↑/↓: navigate • space: toggle • enter: confirm • q: quit",
		Items:        items,
		Formatter:    formatter,
	}

	// Run interactive selection
	selectedItems, cancelled, err := ui.RunInteractiveSelection(config)
	if err != nil {
		return nil, fmt.Errorf("interactive selection failed: %w", err)
	}

	if cancelled {
		return nil, nil
	}

	// Convert back to installation records
	result := make([]InstallRecord, 0, len(selectedItems))
	for _, item := range selectedItems {
		result = append(result, item.Data.(InstallRecord))
	}

	return result, nil
}

func prepareSelectionItems(installations []config.InstallationRecord) []selectionItem {
	// Group installations for better display
	groups := make(map[string][]config.InstallationRecord)

	for _, install := range installations {
		var groupKey string
		if install.Global {
			groupKey = "🌍 Global"
		} else {
			if install.ProjectPath != "" {
				projectName := filepath.Base(install.ProjectPath)
				groupKey = fmt.Sprintf("📁 Project: %s", projectName)
			} else {
				groupKey = "📁 Project"
			}
		}
		groups[groupKey] = append(groups[groupKey], install)
	}

	// Sort group names
	var groupNames []string
	for name := range groups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)

	// Create selection items with group headers
	var items []selectionItem

	for _, groupName := range groupNames {
		installs := groups[groupName]

		// Sort installations within group
		sort.Slice(installs, func(i, j int) bool {
			if installs[i].Target != installs[j].Target {
				return installs[i].Target < installs[j].Target
			}
			return installs[i].Rule < installs[j].Rule
		})

		// Add group header
		items = append(items, selectionItem{
			displayText:  fmt.Sprintf("GROUP_HEADER:%s", groupName),
			installation: config.InstallationRecord{}, // Empty record for headers
		})

		// Add installations
		for _, install := range installs {
			items = append(items, selectionItem{
				displayText:  "", // Will be generated in View()
				installation: install,
			})
		}
	}

	return items
}

func showUninstallPreviewAndConfirm(installations []config.InstallationRecord) bool {
	// Display what will be uninstalled using table format
	fmt.Println("🗑️  Files to be deleted:")
	fmt.Println()
	displayUninstallTable(installations)

	// Ask for confirmation
	fmt.Print("\nProceed with uninstallation? [y/N]: ")
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		// If input fails, default to cancel for safety
		fmt.Println("\nInput error, uninstallation cancelled")
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func displayUninstallTable(installations []config.InstallationRecord) {
	// Group installations by scope (global vs project)
	var globalInstalls []config.InstallationRecord
	projectInstalls := make(map[string][]config.InstallationRecord)

	for _, install := range installations {
		if install.Global {
			globalInstalls = append(globalInstalls, install)
		} else {
			projectInstalls[install.ProjectPath] = append(projectInstalls[install.ProjectPath], install)
		}
	}

	// Sort installations
	sortInstalls := func(installs []config.InstallationRecord) {
		sort.Slice(installs, func(i, j int) bool {
			if installs[i].Target != installs[j].Target {
				return installs[i].Target < installs[j].Target
			}
			if installs[i].Rule != installs[j].Rule {
				return installs[i].Rule < installs[j].Rule
			}
			return installs[i].Mode < installs[j].Mode
		})
	}

	// Display global installations
	if len(globalInstalls) > 0 {
		fmt.Println("🌍 Global Installations")
		fmt.Println(strings.Repeat("=", 78))
		sortInstalls(globalInstalls)
		displayUninstallTableSection(globalInstalls)
		fmt.Println()
	}

	// Display project installations
	if len(projectInstalls) > 0 {
		// Sort project paths for consistent output
		var projectPaths []string
		for path := range projectInstalls {
			projectPaths = append(projectPaths, path)
		}
		sort.Strings(projectPaths)

		for _, projPath := range projectPaths {
			// Skip empty project paths
			if projPath == "" {
				continue
			}
			// Display only the project name (last directory) instead of full path
			projectName := filepath.Base(projPath)
			fmt.Printf("📁 Project: %s\n", projectName)
			fmt.Println(strings.Repeat("=", 78))
			installs := projectInstalls[projPath]
			sortInstalls(installs)
			displayUninstallTableSection(installs)
			fmt.Println()
		}
	}
}

func displayUninstallTableSection(installations []config.InstallationRecord) {
	// Print table header
	fmt.Printf("%-8s %-20s %-8s %-25s %-15s\n", "Target", "Rule", "Mode", "File", "Installed")
	fmt.Println(strings.Repeat("-", 78))

	// Print each row
	for _, install := range installations {
		target := install.Target
		rule := install.Rule
		if rule == "*" {
			rule = "all templates"
		}

		mode := install.Mode
		if mode == "" {
			mode = "-"
		}

		fileName := filepath.Base(install.FilePath)
		timeAgo := utils.FormatTimeAgo(install.InstalledAt)

		// Truncate long strings
		if len(rule) > 20 {
			rule = rule[:17] + "..."
		}
		if len(fileName) > 25 {
			fileName = fileName[:22] + "..."
		}

		fmt.Printf("%-8s %-20s %-8s %-25s %-15s\n", target, rule, mode, fileName, timeAgo)
	}
}

func performUninstallation(installations []config.InstallationRecord) error {
	// Load tracker for removal
	tracker, err := config.LoadGlobalInstallationTracker()
	if err != nil {
		return fmt.Errorf("failed to load installation tracker: %w", err)
	}

	if !uninstallForce && !uninstallInteractive {
		fmt.Println()
	}

	uninstalled := 0
	failed := 0

	for _, installation := range installations {
		if err := uninstallSingle(installation, tracker); err != nil {
			fmt.Printf("  ⚠️  Failed to uninstall %s %s: %v\n", installation.Target, installation.Rule, err)
			failed++
		} else {
			fmt.Printf("  ✅ Uninstalled %s %s (%s)\n", installation.Target, installation.Rule, installation.Mode)
			uninstalled++
		}
	}

	// Save the updated tracker
	if err := config.SaveGlobalInstallationTracker(tracker); err != nil {
		fmt.Printf("Warning: failed to save installation tracker: %v\n", err)
	}

	fmt.Printf("\n🎉 Uninstalled %d installations", uninstalled)
	if failed > 0 {
		fmt.Printf(", %d failed", failed)
	}
	fmt.Println()

	return nil
}

func uninstallSingle(installation config.InstallationRecord, tracker *config.InstallationTracker) error {
	// Special handling for copilot target
	if installation.Target == "copilot" {
		return uninstallCopilotRule(installation, tracker)
	}

	// Special handling for gemini target
	if installation.Target == "gemini" {
		return uninstallGeminiRule(installation, tracker)
	}

	// Standard handling for other targets
	// Remove the actual file
	if _, err := os.Stat(installation.FilePath); err == nil {
		if err := os.Remove(installation.FilePath); err != nil {
			return fmt.Errorf("failed to remove file %s: %w", installation.FilePath, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check file %s: %w", installation.FilePath, err)
	}
	// If file doesn't exist, we continue silently (already uninstalled)

	// Remove from tracking
	tracker.RemoveInstallation(
		installation.Target,
		installation.Rule,
		installation.Global,
		installation.ProjectPath,
		installation.Mode,
	)

	return nil
}

// uninstallCopilotRule handles the special case of uninstalling copilot rules
// Since copilot rules are merged into a single file, we use a reinstall strategy:
// 1. Remove the rule from tracking
// 2. Get all remaining copilot rules for this project
// 3. Delete the current combined file
// 4. Reinstall all remaining rules (if any)
func uninstallCopilotRule(installation config.InstallationRecord, tracker *config.InstallationTracker) error {
	// First, remove this rule from tracking
	tracker.RemoveInstallation(
		installation.Target,
		installation.Rule,
		installation.Global,
		installation.ProjectPath,
		installation.Mode,
	)

	// Get all remaining copilot rules for this project/scope
	remainingRules := tracker.GetInstallations("copilot", "")
	var remainingForThisScope []config.InstallationRecord

	// Filter to only rules that match this installation's scope (global vs project)
	for _, rule := range remainingRules {
		if rule.Global == installation.Global && rule.ProjectPath == installation.ProjectPath {
			remainingForThisScope = append(remainingForThisScope, rule)
		}
	}

	// Delete the current combined file if it exists
	if _, err := os.Stat(installation.FilePath); err == nil {
		if err := os.Remove(installation.FilePath); err != nil {
			return fmt.Errorf("failed to remove copilot file %s: %w", installation.FilePath, err)
		}
	}

	// If there are remaining rules, reinstall them
	if len(remainingForThisScope) > 0 {
		return reinstallCopilotRules(remainingForThisScope, installation.ProjectPath)
	}

	return nil
}

// reinstallCopilotRules recreates the copilot-instructions.md file with only the specified rules
func reinstallCopilotRules(rules []config.InstallationRecord, projectPath string) error {
	if len(rules) == 0 {
		return nil
	}

	// Copilot only supports project installation
	if projectPath == "" {
		return fmt.Errorf("copilot rules require project path")
	}

	// Get absolute project path, handling both absolute and relative paths correctly
	var absPath string
	if filepath.IsAbs(projectPath) {
		absPath = projectPath
	} else {
		// For relative paths, resolve them relative to the original working directory
		// This handles cases where installation records contain relative paths
		originalDir := GetOriginalWorkingDir()
		resolvedPath := filepath.Join(originalDir, projectPath)
		var err error
		absPath, err = filepath.Abs(resolvedPath)
		if err != nil {
			return fmt.Errorf("failed to resolve project path: %w", err)
		}
	}

	targetDir := filepath.Join(absPath, ".github")
	targetPath := filepath.Join(targetDir, "copilot-instructions.md")

	// Ensure .github directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create .github directory: %w", err)
	}

	// Collect content for each remaining rule
	var ruleContents []string
	var ruleNames []string

	for _, rule := range rules {
		// Find the compiled source file for this rule
		compiledDir := filepath.Join("compiled", "copilot")
		sourcePath := filepath.Join(compiledDir, rule.Rule+".copilot-instructions.md")

		// Read the source content
		content, err := os.ReadFile(sourcePath)
		if err != nil {
			// If we can't find the source file, skip this rule but don't fail
			// This handles cases where the compiled files may have been cleaned up
			continue
		}

		ruleContents = append(ruleContents, strings.TrimSpace(string(content)))
		ruleNames = append(ruleNames, rule.Rule)
	}

	if len(ruleContents) == 0 {
		// No content found to reinstall, just leave the file deleted
		return nil
	}

	// Combine all rules into single content (same logic as installCopilotRules)
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

	// Write the combined content
	if err := os.WriteFile(targetPath, []byte(combinedContent.String()), 0600); err != nil {
		return fmt.Errorf("failed to write reinstalled copilot instructions: %w", err)
	}

	return nil
}

// uninstallGeminiRule handles the special case of uninstalling gemini rules
// Since gemini rules are merged into a single file, we use a reinstall strategy:
// 1. Remove the rule from tracking
// 2. Get all remaining gemini rules for this scope (global/project)
// 3. Delete the current combined file
// 4. Reinstall all remaining rules (if any)
func uninstallGeminiRule(installation config.InstallationRecord, tracker *config.InstallationTracker) error {
	// First, remove this rule from tracking
	tracker.RemoveInstallation(
		installation.Target,
		installation.Rule,
		installation.Global,
		installation.ProjectPath,
		installation.Mode,
	)

	// Get all remaining gemini rules for this project/scope
	remainingRules := tracker.GetInstallations("gemini", "")
	var remainingForThisScope []config.InstallationRecord

	// Filter to only rules that match this installation's scope (global vs project)
	for _, rule := range remainingRules {
		if rule.Global == installation.Global && rule.ProjectPath == installation.ProjectPath {
			remainingForThisScope = append(remainingForThisScope, rule)
		}
	}

	// Delete the current combined file if it exists
	if _, err := os.Stat(installation.FilePath); err == nil {
		if err := os.Remove(installation.FilePath); err != nil {
			return fmt.Errorf("failed to remove gemini file %s: %w", installation.FilePath, err)
		}
	}

	// If there are remaining rules, reinstall them
	if len(remainingForThisScope) > 0 {
		return reinstallGeminiRules(remainingForThisScope, installation.ProjectPath, installation.Global)
	}

	return nil
}

// reinstallGeminiRules recreates the GEMINI.md file with only the specified rules
func reinstallGeminiRules(rules []config.InstallationRecord, projectPath string, isGlobal bool) error {
	if len(rules) == 0 {
		return nil
	}

	// Determine target directory based on global vs project installation
	var targetDir string
	var targetPath string

	if isGlobal {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		targetDir = filepath.Join(homeDir, ".gemini")
		targetPath = filepath.Join(targetDir, "GEMINI.md")
	} else {
		if projectPath == "" {
			return fmt.Errorf("gemini project rules require project path")
		}

		// Get absolute project path, handling both absolute and relative paths correctly
		var absPath string
		if filepath.IsAbs(projectPath) {
			absPath = projectPath
		} else {
			// For relative paths, resolve them relative to the original working directory
			// This handles cases where installation records contain relative paths
			originalDir := GetOriginalWorkingDir()
			resolvedPath := filepath.Join(originalDir, projectPath)
			var err error
			absPath, err = filepath.Abs(resolvedPath)
			if err != nil {
				return fmt.Errorf("failed to resolve project path: %w", err)
			}
		}
		targetDir = absPath
		targetPath = filepath.Join(targetDir, "GEMINI.md")
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create gemini directory: %w", err)
	}

	// Collect content for each remaining rule
	var ruleContents []string
	var ruleNames []string

	for _, rule := range rules {
		// Find the compiled source file for this rule
		compiledDir := filepath.Join("compiled", "gemini")
		sourcePath := filepath.Join(compiledDir, rule.Rule+".md")

		// Read the source content
		content, err := os.ReadFile(sourcePath)
		if err != nil {
			// If we can't find the source file, skip this rule but don't fail
			// This handles cases where the compiled files may have been cleaned up
			continue
		}

		ruleContents = append(ruleContents, strings.TrimSpace(string(content)))
		ruleNames = append(ruleNames, rule.Rule)
	}

	if len(ruleContents) == 0 {
		// No content found to reinstall, just leave the file deleted
		return nil
	}

	// Combine all rules into single content (same logic as installGeminiRules)
	var combinedContent strings.Builder
	combinedContent.WriteString("# AI Coding Instructions\n\n")
	combinedContent.WriteString("This file contains custom instructions for Gemini CLI.\n\n")

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

	// Write the combined content
	if err := os.WriteFile(targetPath, []byte(combinedContent.String()), 0600); err != nil {
		return fmt.Errorf("failed to write reinstalled gemini instructions: %w", err)
	}

	return nil
}

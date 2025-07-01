// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/ratler/airuler/internal/compiler"
	"github.com/ratler/airuler/internal/config"
	"github.com/ratler/airuler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	installTarget      string
	installRule        string
	installGlobal      bool
	installProject     string
	installForce       bool
	installInteractive bool
)

var installCmd = &cobra.Command{
	Use:   "install [target] [rule]",
	Short: "Install compiled rules to AI coding assistants",
	Long: `Install compiled rules to AI coding assistants.

By default, installs to global configuration directories.
Use --project to install to a specific project directory.

Modes:
  Default: Install all or specified templates
  Interactive (--interactive): Select templates with checkbox interface

Examples:
  airuler install                           # Install all rules for all targets
  airuler install cursor                    # Install all Cursor rules
  airuler install cursor my-rule            # Install specific Cursor rule
  airuler install --project ./my-project    # Install to project directory
  airuler install --interactive             # Interactive selection mode
  airuler install claude --interactive      # Interactive mode for Claude only`,
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
	installCmd.Flags().BoolVar(&installInteractive, "interactive", false, "use interactive checkbox selection")

	// Make --force and --interactive mutually exclusive
	installCmd.MarkFlagsMutuallyExclusive("force", "interactive")
}

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
	absPath, err := filepath.Abs(installProject)
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
		wasExisting := false
		for _, existingName := range existingRuleNames {
			if existingName == installRule {
				wasExisting = true
				break
			}
		}

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
			wasExisting := false
			for _, existingName := range existingRuleNames {
				if existingName == ruleName {
					wasExisting = true
					break
				}
			}

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
	case compiler.TargetRoo:
		return getRooGlobalPath(), nil
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
	case compiler.TargetRoo:
		return filepath.Join(absPath, ".roo", "rules"), nil
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
	// Convert project path to absolute path if it's a project installation
	var projectPath string
	if installProject != "" {
		absPath, err := filepath.Abs(installProject)
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
			fmt.Printf("📋 No compiled templates found for project installation (%s). Run 'airuler compile' first.\n", projectName)
		} else {
			fmt.Println("📋 No compiled templates found for global installation. Run 'airuler compile' first.")
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
				absPath, _ := filepath.Abs(installProject)
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

		// Add group header if we have multiple targets
		if len(groups) > 1 {
			items = append(items, installSelectionItem{
				displayText: fmt.Sprintf("GROUP_HEADER:📦 %s", strings.Title(string(target))),
			})
		}

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

	// Handle other targets
	for target, items := range targetGroups {
		for _, item := range items {
			// Get target directory based on mode
			var targetDir string
			var err error
			if installProject != "" {
				targetDir, err = getProjectInstallDirForMode(target, installProject, item.mode)
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
func (f fakeFileInfo) Type() os.FileMode          { return f.FileInfo.Mode() }
func (f fakeFileInfo) Info() (os.FileInfo, error) { return f.FileInfo, nil }

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ratler/airuler/internal/config"
	"github.com/ratler/airuler/internal/ui"
	"github.com/ratler/airuler/internal/utils"
)

// Type alias to work around Go compiler parsing issue
type InstallRecord = config.InstallationRecord

var (
	uninstallTarget      string
	uninstallRule        string
	uninstallGlobal      bool
	uninstallProject     bool
	uninstallForce       bool
	uninstallInteractive bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [target] [rule]",
	Short: "Uninstall previously installed rules",
	Long: `Uninstall previously installed rules based on installation tracking metadata.

This command provides three modes of operation:

Default Mode (Non-interactive):
  Shows files to be deleted and prompts for confirmation (y/N).
  Good for automation and scripting.

Interactive Mode (--interactive):
  Provides a modern checkbox interface for selecting specific templates to uninstall.
  Use ↑/↓ or j/k to navigate, space to toggle selection, enter to confirm, q to quit.

Force Mode (--force):
  Skips all prompts and immediately deletes selected templates.
  Ideal for automation scenarios.

Examples:
  airuler uninstall                         # Default mode: show files + confirm
  airuler uninstall cursor                  # Uninstall only Cursor installations
  airuler uninstall cursor my-rule          # Uninstall specific rule
  airuler uninstall --interactive           # Interactive checkbox selection
  airuler uninstall --force                 # Skip all confirmations
  airuler uninstall --global               # Uninstall only global installations
  airuler uninstall --project              # Uninstall only project installations`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) >= 1 {
			uninstallTarget = args[0]
		}
		if len(args) >= 2 {
			uninstallRule = args[1]
		}

		return uninstallRules()
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)

	uninstallCmd.Flags().BoolVarP(&uninstallGlobal, "global", "g", false, "uninstall only global installations")
	uninstallCmd.Flags().BoolVarP(&uninstallProject, "project", "p", false, "uninstall only project installations")
	uninstallCmd.Flags().BoolVarP(&uninstallForce, "force", "f", false, "skip confirmation prompts")
	uninstallCmd.Flags().BoolVarP(&uninstallInteractive, "interactive", "i", false, "use interactive checkbox selection")

	// Make --force and --interactive mutually exclusive
	uninstallCmd.MarkFlagsMutuallyExclusive("force", "interactive")
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

type selectionItem struct {
	displayText  string
	installation config.InstallationRecord
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

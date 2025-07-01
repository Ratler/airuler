// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ratler/airuler/internal/config"
	"github.com/spf13/cobra"
)

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

	uninstallCmd.Flags().BoolVar(&uninstallGlobal, "global", false, "uninstall only global installations")
	uninstallCmd.Flags().BoolVar(&uninstallProject, "project", false, "uninstall only project installations")
	uninstallCmd.Flags().BoolVar(&uninstallForce, "force", false, "skip confirmation prompts")
	uninstallCmd.Flags().BoolVar(&uninstallInteractive, "interactive", false, "use interactive checkbox selection")

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

// BubbleTea model for interactive selection
type selectionModel struct {
	items        []selectionItem
	selected     map[int]bool
	cursor       int
	done         bool
	cancelled    bool
	instructions string
}

type selectionItem struct {
	displayText  string
	installation config.InstallationRecord
}

func (m selectionModel) Init() tea.Cmd {
	return nil
}

func (m selectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.cancelled = true
			return m, tea.Quit
		case "up", "k":
			m.cursor = m.findPrevSelectableItem(m.cursor)
		case "down", "j":
			m.cursor = m.findNextSelectableItem(m.cursor)
		case " ":
			// Toggle selection only if not a group header
			if !m.isGroupHeader(m.cursor) {
				if m.selected[m.cursor] {
					delete(m.selected, m.cursor)
				} else {
					m.selected[m.cursor] = true
				}
			}
		case "enter":
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// Helper functions for navigation
func (m selectionModel) isGroupHeader(index int) bool {
	if index < 0 || index >= len(m.items) {
		return false
	}
	return strings.HasPrefix(m.items[index].displayText, "GROUP_HEADER:")
}

func (m selectionModel) findNextSelectableItem(current int) int {
	for i := current + 1; i < len(m.items); i++ {
		if !m.isGroupHeader(i) {
			return i
		}
	}
	return current // Stay at current if no next selectable item
}

func (m selectionModel) findPrevSelectableItem(current int) int {
	for i := current - 1; i >= 0; i-- {
		if !m.isGroupHeader(i) {
			return i
		}
	}
	return current // Stay at current if no previous selectable item
}

func (m selectionModel) View() string {
	var s strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")). // White
		MarginBottom(1)

	s.WriteString(titleStyle.Render("Select templates to uninstall:"))
	s.WriteString("\n\n")

	// Table header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")). // White text
		Background(lipgloss.Color("238"))  // Gray background

	s.WriteString(headerStyle.Render(fmt.Sprintf("   %-3s %-8s %-20s %-8s %-25s %-15s", "SEL", "TARGET", "RULE", "MODE", "FILE", "INSTALLED")))
	s.WriteString("\n")

	// Separator line
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244")) // Medium gray
	s.WriteString(separatorStyle.Render(strings.Repeat("─", 82)))
	s.WriteString("\n")

	// Table rows
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("238"))               // White on gray
	unselectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))                                               // Light gray
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)                                        // White
	groupHeaderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true).Background(lipgloss.Color("236")) // White on dark gray

	for i, item := range m.items {
		// Handle group headers
		if strings.HasPrefix(item.displayText, "GROUP_HEADER:") {
			groupName := strings.TrimPrefix(item.displayText, "GROUP_HEADER:")
			s.WriteString("\n")
			s.WriteString(groupHeaderStyle.Render(fmt.Sprintf("   %s", groupName)))
			s.WriteString("\n")
			continue
		}

		// Parse display text to extract components
		target, rule, mode, fileName, timeAgo := parseDisplayText(item.displayText, item.installation)

		cursor := " "
		if i == m.cursor {
			cursor = cursorStyle.Render("►")
		}

		checkbox := "☐"
		style := unselectedStyle
		if m.selected[i] {
			checkbox = "☑"
			style = selectedStyle
		}

		// Format row with proper column widths
		row := fmt.Sprintf("%s %s %-8s %-20s %-8s %-25s %-15s",
			cursor, checkbox, target, rule, mode, fileName, timeAgo)

		s.WriteString(style.Render(row))
		s.WriteString("\n")
	}

	// Instructions
	s.WriteString("\n")
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("248")). // Light gray
		Italic(true)
	s.WriteString(instructionStyle.Render(m.instructions))

	// Selection counter
	s.WriteString("\n")
	counterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")). // White
		Bold(true)
	selectedCount := len(m.selected)
	// Count only selectable items (exclude group headers)
	selectableCount := 0
	for i := range m.items {
		if !m.isGroupHeader(i) {
			selectableCount++
		}
	}
	s.WriteString(counterStyle.Render(fmt.Sprintf("Selected: %d of %d", selectedCount, selectableCount)))

	return s.String()
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
	timeAgo = formatTimeAgo(installation.InstalledAt)

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
	items := prepareSelectionItems(installations)

	if len(items) == 0 {
		return nil, nil
	}

	// Create BubbleTea model
	model := selectionModel{
		items:        items,
		selected:     make(map[int]bool),
		cursor:       0,
		done:         false,
		cancelled:    false,
		instructions: "↑/↓: navigate • space: toggle • enter: confirm • q: quit",
	}

	// Set cursor to first selectable item
	model.cursor = model.findNextSelectableItem(-1)

	// Run the interactive program
	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("interactive selection failed: %w", err)
	}

	// Extract results
	final := finalModel.(selectionModel)
	if final.cancelled {
		return nil, nil
	}

	var result []config.InstallationRecord
	for i := range final.selected {
		// Skip group headers
		if !final.isGroupHeader(i) {
			result = append(result, final.items[i].installation)
		}
	}

	return result, nil
}

func prepareSelectionItems(installations []config.InstallationRecord) []selectionItem {
	// Group installations for better display
	groups := make(map[string][]config.InstallationRecord)

	for _, install := range installations {
		groupKey := "🌍 Global"
		if !install.Global {
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

		// Add group header if multiple groups
		if len(groups) > 1 {
			items = append(items, selectionItem{
				displayText:  fmt.Sprintf("GROUP_HEADER:%s", groupName),
				installation: config.InstallationRecord{}, // Empty record for headers
			})
		}

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
		timeAgo := formatTimeAgo(install.InstalledAt)

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

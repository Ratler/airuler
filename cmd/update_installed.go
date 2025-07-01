// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ratler/airuler/internal/compiler"
	"github.com/ratler/airuler/internal/config"
	"github.com/spf13/cobra"
)

var (
	updateInstalledTarget  string
	updateInstalledRule    string
	updateInstalledGlobal  bool
	updateInstalledProject bool
)

type updateResult struct {
	Target      string
	Rule        string
	Mode        string
	FilePath    string
	Global      bool
	ProjectPath string
	Status      string // "updated", "unchanged", "installed"
}

var updateInstalledCmd = &cobra.Command{
	Use:   "update-installed [target] [rule]",
	Short: "Update all previously installed rules",
	Long: `Update all previously installed rules based on installation tracking metadata.

This command reads the installation history and reinstalls all previously installed
rules with their original target, mode, and location settings.

Examples:
  airuler update-installed                    # Update all tracked installations
  airuler update-installed cursor            # Update only Cursor installations  
  airuler update-installed cursor my-rule    # Update specific Cursor rule installations
  airuler update-installed --global          # Update only global installations
  airuler update-installed --project         # Update only project installations`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) >= 1 {
			updateInstalledTarget = args[0]
		}
		if len(args) >= 2 {
			updateInstalledRule = args[1]
		}

		return updateInstalledRules()
	},
}

func init() {
	rootCmd.AddCommand(updateInstalledCmd)

	updateInstalledCmd.Flags().BoolVarP(&updateInstalledGlobal, "global", "g", false, "update only global installations")
	updateInstalledCmd.Flags().BoolVarP(&updateInstalledProject, "project", "p", false, "update only project installations")
}

func updateInstalledRules() error {
	var allInstallations []config.InstallationRecord

	// Load installation tracker (global and project are now stored in same location)
	tracker, err := config.LoadGlobalInstallationTracker()
	if err != nil {
		fmt.Printf("Warning: failed to load installation tracker: %v\n", err)
	} else {
		installations := tracker.GetInstallations(updateInstalledTarget, updateInstalledRule)

		// Filter by installation type if specified
		if updateInstalledGlobal {
			// Only include global installations
			for _, install := range installations {
				if install.Global {
					allInstallations = append(allInstallations, install)
				}
			}
		} else if updateInstalledProject {
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
	}

	if len(allInstallations) == 0 {
		fmt.Println("No tracked installations found")
		return nil
	}

	// Process all installations and collect results
	var results []updateResult

	for _, installation := range allInstallations {
		status, err := updateSingleInstallationWithStatus(installation)
		if err != nil {
			status = "failed"
		}

		results = append(results, updateResult{
			Target:      installation.Target,
			Rule:        installation.Rule,
			Mode:        installation.Mode,
			FilePath:    installation.FilePath,
			Global:      installation.Global,
			ProjectPath: installation.ProjectPath,
			Status:      status,
		})
	}

	// Display results in table format
	displayUpdateResults(results)

	return nil
}

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

func displayUpdateResults(results []updateResult) {
	// Sort results for consistent output
	sort.Slice(results, func(i, j int) bool {
		if results[i].Target != results[j].Target {
			return results[i].Target < results[j].Target
		}
		if results[i].Rule != results[j].Rule {
			return results[i].Rule < results[j].Rule
		}
		return results[i].Mode < results[j].Mode
	})

	// Group results by scope (global vs project)
	var globalResults []updateResult
	projectResults := make(map[string][]updateResult)

	for _, result := range results {
		if result.Global {
			globalResults = append(globalResults, result)
		} else {
			projectResults[result.ProjectPath] = append(projectResults[result.ProjectPath], result)
		}
	}

	// Display header
	fmt.Println("🔄 Update Results")
	fmt.Println()

	// Display global updates
	if len(globalResults) > 0 {
		fmt.Println("🌍 Global Installations")
		fmt.Println(strings.Repeat("=", 78))
		displayUpdateTable(globalResults)
		fmt.Println()
	}

	// Display project updates
	if len(projectResults) > 0 {
		// Sort project paths for consistent output
		var projectPaths []string
		for path := range projectResults {
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
			displayUpdateTable(projectResults[projPath])
			fmt.Println()
		}
	}

	// Display summary with status counts
	statusCounts := make(map[string]int)
	for _, result := range results {
		statusCounts[result.Status]++
	}

	fmt.Printf("Total: %d template(s) processed", len(results))
	if len(statusCounts) > 1 {
		var parts []string
		if count := statusCounts["updated"]; count > 0 {
			parts = append(parts, fmt.Sprintf("%d updated", count))
		}
		if count := statusCounts["installed"]; count > 0 {
			parts = append(parts, fmt.Sprintf("%d installed", count))
		}
		if count := statusCounts["unchanged"]; count > 0 {
			parts = append(parts, fmt.Sprintf("%d unchanged", count))
		}
		if count := statusCounts["failed"]; count > 0 {
			parts = append(parts, fmt.Sprintf("%d failed", count))
		}
		if len(parts) > 0 {
			fmt.Printf(" (%s)", strings.Join(parts, ", "))
		}
	}
	fmt.Println()
}

func displayUpdateTable(results []updateResult) {
	// Print table header with Status column instead of Installed
	fmt.Printf("%-8s %-20s %-8s %-25s %-15s\n", "Target", "Rule", "Mode", "File", "Status")
	fmt.Println(strings.Repeat("-", 78))

	// Print each row
	for _, result := range results {
		target := result.Target
		rule := result.Rule
		if rule == "*" {
			rule = "all templates"
		}

		mode := result.Mode
		if mode == "" {
			mode = "-"
		}

		fileName := filepath.Base(result.FilePath)

		// Format status with emoji indicators
		status := result.Status
		switch status {
		case "updated":
			status = "✅ updated"
		case "installed":
			status = "🆕 installed"
		case "unchanged":
			status = "⏸️ unchanged"
		case "failed":
			status = "❌ failed"
		}

		// Truncate long strings
		if len(rule) > 20 {
			rule = rule[:17] + "..."
		}
		if len(fileName) > 25 {
			fileName = fileName[:22] + "..."
		}

		fmt.Printf("%-8s %-20s %-8s %-25s %-15s\n", target, rule, mode, fileName, status)
	}
}

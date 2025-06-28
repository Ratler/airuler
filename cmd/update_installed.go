package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	RunE: func(cmd *cobra.Command, args []string) error {
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

	updateInstalledCmd.Flags().BoolVar(&updateInstalledGlobal, "global", false, "update only global installations")
	updateInstalledCmd.Flags().BoolVar(&updateInstalledProject, "project", false, "update only project installations")
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

	fmt.Printf("Found %d tracked installations to update\n\n", len(allInstallations))

	updated := 0
	failed := 0

	for _, installation := range allInstallations {
		installType := "global"
		if !installation.Global {
			installType = fmt.Sprintf("project: %s", installation.ProjectPath)
		}

		if err := updateSingleInstallation(installation); err != nil {
			fmt.Printf("  ⚠️  Failed to update %s %s (%s) - %s: %v\n", installation.Target, installation.Rule, installation.Mode, installType, err)
			failed++
		} else {
			fmt.Printf("  ✅ Updated %s %s (%s) - %s\n", installation.Target, installation.Rule, installation.Mode, installType)
			updated++
		}
	}

	fmt.Printf("\n🎉 Updated %d installations", updated)
	if failed > 0 {
		fmt.Printf(", %d failed", failed)
	}
	fmt.Println()

	return nil
}

func updateSingleInstallation(installation config.InstallationRecord) error {
	target := compiler.Target(installation.Target)

	// Validate target
	if !isValidTarget(target) {
		return fmt.Errorf("invalid target: %s", installation.Target)
	}

	// Find the compiled rule files
	compiledDir := filepath.Join("compiled", string(target))
	files, err := os.ReadDir(compiledDir)
	if err != nil {
		return fmt.Errorf("failed to read compiled directory: %w", err)
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
		return fmt.Errorf("no compiled rules found for %s", installation.Rule)
	}

	// Determine the target directory based on the original installation
	var targetDir string

	if installation.Global {
		targetDir, err = getGlobalInstallDirForMode(target, installation.Mode)
	} else {
		if installation.ProjectPath == "" {
			return fmt.Errorf("project path not specified for project installation")
		}
		targetDir, err = getProjectInstallDirForMode(target, installation.ProjectPath, installation.Mode)
	}

	if err != nil {
		return fmt.Errorf("failed to get target directory: %w", err)
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// For update-installed, we always force overwrite since we're updating
	originalForce := installForce
	installForce = true
	defer func() { installForce = originalForce }()

	// Install all the files
	for _, sourceFile := range sourceFiles {
		targetPath := filepath.Join(targetDir, filepath.Base(sourceFile))

		if err := installFileWithMode(sourceFile, targetPath, target, installation.Mode); err != nil {
			return fmt.Errorf("failed to install file %s: %w", filepath.Base(sourceFile), err)
		}
	}

	// Update the installation record timestamp
	if err := updateInstallationRecord(installation); err != nil {
		// Don't fail the whole operation for this, just warn
		fmt.Printf("    Warning: failed to update installation record: %v\n", err)
	}

	return nil
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
	} else {
		return config.SaveProjectInstallationTracker(tracker)
	}
}

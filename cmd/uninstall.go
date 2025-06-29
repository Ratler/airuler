package cmd

import (
	"fmt"
	"os"

	"github.com/ratler/airuler/internal/config"
	"github.com/spf13/cobra"
)

var (
	uninstallTarget  string
	uninstallRule    string
	uninstallGlobal  bool
	uninstallProject bool
	uninstallForce   bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [target] [rule]",
	Short: "Uninstall previously installed rules",
	Long: `Uninstall previously installed rules based on installation tracking metadata.

This command reads the installation history and removes previously installed
rules from their target locations, then removes them from the tracking system.

Examples:
  airuler uninstall                         # Uninstall all tracked installations
  airuler uninstall cursor                  # Uninstall only Cursor installations
  airuler uninstall cursor my-rule          # Uninstall specific Cursor rule installations
  airuler uninstall --global               # Uninstall only global installations
  airuler uninstall --project              # Uninstall only project installations
  airuler uninstall --force                # Skip confirmation prompts`,
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
}

func uninstallRules() error {
	var allInstallations []config.InstallationRecord

	// Load installation tracker
	tracker, err := config.LoadGlobalInstallationTracker()
	if err != nil {
		return fmt.Errorf("failed to load installation tracker: %w", err)
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

	if len(allInstallations) == 0 {
		fmt.Println("No tracked installations found to uninstall")
		return nil
	}

	// Show what will be uninstalled
	fmt.Printf("Found %d installations to uninstall:\n\n", len(allInstallations))
	for _, installation := range allInstallations {
		installType := "global"
		if !installation.Global {
			installType = "project"
		}
		fmt.Printf("  • %s %s (%s) - %s\n", installation.Target, installation.Rule, installation.Mode, installType)
		fmt.Printf("    File: %s\n", installation.FilePath)
	}

	// Confirm unless --force is used
	if !uninstallForce {
		fmt.Print("\nProceed with uninstallation? [y/N]: ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// If input fails, default to cancel for safety
			fmt.Println("\nInput error, uninstallation cancelled")
			return err
		}
		if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
			fmt.Println("Uninstallation cancelled")
			return nil
		}
	}

	fmt.Println()

	uninstalled := 0
	failed := 0

	for _, installation := range allInstallations {
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

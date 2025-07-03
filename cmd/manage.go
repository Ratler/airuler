// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ratler/airuler/internal/config"
	"github.com/spf13/cobra"
)

var (
	manageClean        bool
	manageUninstallAll bool
)

var manageCmd = &cobra.Command{
	Use:   "manage [subcommand]",
	Short: "Interactive management hub for airuler",
	Long: `Manage provides an interactive interface for common airuler operations:

- Overview of installations, vendors, and project status
- Interactive management of vendors and installations  
- Clean and rebuild operations
- Uninstall installed rules

Subcommands:
  vendors        Interactive vendor management
  installations  Interactive installation management
  uninstall      Interactive uninstallation of rules

Examples:
  airuler manage                    # Main management interface
  airuler manage vendors            # Vendor-specific management
  airuler manage installations      # Installation-specific management
  airuler manage uninstall          # Interactive uninstallation
  airuler manage uninstall --all    # Uninstall all installations without prompts
  airuler manage --clean            # Clean and rebuild everything`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if manageClean {
			return runManageClean()
		}

		var subcommand string
		if len(args) > 0 {
			subcommand = args[0]
		}

		return runManage(subcommand)
	},
}

func init() {
	rootCmd.AddCommand(manageCmd)

	manageCmd.Flags().BoolVarP(&manageClean, "clean", "c", false, "clean and rebuild everything")
	manageCmd.Flags().BoolVarP(&manageUninstallAll, "all", "a", false, "uninstall all installations without interactive prompts (use with 'uninstall' subcommand)")
}

func runManage(subcommand string) error {
	switch subcommand {
	case "vendors":
		return runManageVendors()
	case "installations":
		return runManageInstallations()
	case "uninstall":
		return runManageUninstall()
	case "":
		return runManageMain()
	default:
		return fmt.Errorf("unknown subcommand: %s", subcommand)
	}
}

func runManageMain() error {
	fmt.Println("🎛️  airuler Management Hub")
	fmt.Println("==========================")
	fmt.Println()

	// Show project status
	if err := showProjectStatus(); err != nil {
		fmt.Printf("Warning: could not determine project status: %v\n", err)
	}

	// Show vendor status
	fmt.Println("\n📦 Vendor Status")
	fmt.Println("================")
	if err := showVendorStatus(); err != nil {
		fmt.Printf("Warning: could not load vendor status: %v\n", err)
	}

	// Show installation status
	fmt.Println("\n🚀 Installation Status")
	fmt.Println("======================")
	if err := showInstallationSummary(); err != nil {
		fmt.Printf("Warning: could not load installation status: %v\n", err)
	}

	// Show available actions
	fmt.Println("\n🎯 Available Actions")
	fmt.Println("====================")
	fmt.Println("  airuler sync                     # Update vendors → compile → deploy")
	fmt.Println("  airuler deploy                   # Compile → install fresh")
	fmt.Println("  airuler manage vendors           # Interactive vendor management")
	fmt.Println("  airuler manage installations     # Interactive installation management")
	fmt.Println("  airuler manage uninstall         # Interactive uninstallation")
	fmt.Println("  airuler manage --clean           # Clean and rebuild everything")
	fmt.Println()

	return nil
}

func runManageVendors() error {
	fmt.Println("📦 Vendor Management")
	fmt.Println("====================")
	fmt.Println()

	// Show current vendor status
	if err := showVendorStatus(); err != nil {
		return fmt.Errorf("failed to load vendor status: %w", err)
	}

	fmt.Println("\n🎯 Vendor Management Commands")
	fmt.Println("==============================")
	fmt.Println("  airuler vendors list             # List all vendors")
	fmt.Println("  airuler vendors add <url>        # Add new vendor repository")
	fmt.Println("  airuler vendors update [name]    # Update vendor repositories")
	fmt.Println("  airuler vendors remove <name>    # Remove vendor")
	fmt.Println("  airuler vendors config [name]    # View vendor configurations")
	fmt.Println("  airuler sync --no-compile --no-deploy  # Update vendors only")
	fmt.Println()

	return nil
}

func runManageInstallations() error {
	fmt.Println("🚀 Installation Management")
	fmt.Println("===========================")
	fmt.Println()

	// Show current installations
	fmt.Println("📋 Current Installations")
	fmt.Println("=========================")

	// Use the existing list-installed functionality
	originalListFilter := listFilter
	listFilter = ""
	defer func() { listFilter = originalListFilter }()

	if err := runListInstalled(); err != nil {
		return fmt.Errorf("failed to load installations: %w", err)
	}

	fmt.Println("\n🎯 Installation Management Commands")
	fmt.Println("===================================")
	fmt.Println("  airuler deploy --interactive      # Interactive template installation")
	fmt.Println("  airuler manage uninstall          # Interactive template uninstallation")
	fmt.Println("  airuler sync --no-update          # Update existing installations")
	fmt.Println("  airuler list-installed            # List all installed templates")
	fmt.Println()

	return nil
}

func runManageUninstall() error {
	fmt.Println("🗑️  Installation Uninstall")
	fmt.Println("==========================")
	fmt.Println()

	// Save original uninstall settings
	originalUninstallInteractive := uninstallInteractive
	originalUninstallTarget := uninstallTarget
	originalUninstallRule := uninstallRule
	originalUninstallGlobal := uninstallGlobal
	originalUninstallProject := uninstallProject
	originalUninstallForce := uninstallForce

	defer func() {
		uninstallInteractive = originalUninstallInteractive
		uninstallTarget = originalUninstallTarget
		uninstallRule = originalUninstallRule
		uninstallGlobal = originalUninstallGlobal
		uninstallProject = originalUninstallProject
		uninstallForce = originalUninstallForce
	}()

	if manageUninstallAll {
		// Uninstall all without interactive prompts
		return runUninstallAll()
	}

	// Set up for interactive uninstallation
	uninstallInteractive = true
	uninstallTarget = ""
	uninstallRule = ""
	uninstallGlobal = false
	uninstallProject = false
	uninstallForce = false

	// Run the uninstall operation
	return uninstallRules()
}

func runManageClean() error {
	fmt.Println("🧹 Clean and Rebuild")
	fmt.Println("====================")
	fmt.Println()

	fmt.Println("This will:")
	fmt.Println("  1. 🗑️  Clean compiled directory")
	fmt.Println("  2. 🔄 Update vendor repositories")
	fmt.Println("  3. ⚙️  Recompile all templates")
	fmt.Println("  4. 🚀 Update all existing installations")
	fmt.Println()

	// Ask for confirmation
	fmt.Print("Continue with clean and rebuild? [y/N]: ")
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		fmt.Println("Operation cancelled")
		return err
	}

	if response != "y" && response != "Y" && response != "yes" {
		fmt.Println("Operation cancelled")
		return nil
	}

	fmt.Println()

	// Step 1: Clean compiled directory
	fmt.Println("🗑️  Cleaning compiled directory...")
	compiledDir := "compiled"
	if _, err := os.Stat(compiledDir); err == nil {
		if err := os.RemoveAll(compiledDir); err != nil {
			return fmt.Errorf("failed to clean compiled directory: %w", err)
		}
		fmt.Println("  ✅ Compiled directory cleaned")
	} else {
		fmt.Println("  📋 No compiled directory to clean")
	}

	// Step 2-4: Run full sync
	fmt.Println("\n🔄 Running full sync workflow...")

	// Set sync flags for full operation
	originalSyncNoUpdate := syncNoUpdate
	originalSyncNoCompile := syncNoCompile
	originalSyncNoDeploy := syncNoDeploy

	syncNoUpdate = false
	syncNoCompile = false
	syncNoDeploy = false

	defer func() {
		syncNoUpdate = originalSyncNoUpdate
		syncNoCompile = originalSyncNoCompile
		syncNoDeploy = originalSyncNoDeploy
	}()

	if err := runSync(""); err != nil {
		return fmt.Errorf("sync failed during clean and rebuild: %w", err)
	}

	fmt.Printf("\n🎉 Clean and rebuild completed successfully\n")
	return nil
}

func showProjectStatus() error {
	fmt.Println("📁 Project Status")
	fmt.Println("=================")

	// Check if we're in a template directory
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	if config.IsTemplateDirectory(currentDir) {
		fmt.Printf("  📂 Current directory: %s (airuler project)\n", filepath.Base(currentDir))

		// Check for templates
		if _, err := os.Stat("templates"); err == nil {
			templateFiles, err := filepath.Glob("templates/*.tmpl")
			if err == nil {
				fmt.Printf("  📄 Templates: %d template file(s)\n", len(templateFiles))
			}
		}

		// Check for compiled rules
		if _, err := os.Stat("compiled"); err == nil {
			compiledDirs, err := os.ReadDir("compiled")
			if err == nil {
				fmt.Printf("  ⚙️  Compiled: %d target(s)\n", len(compiledDirs))
			}
		}

		// Check for vendors
		if _, err := os.Stat("vendors"); err == nil {
			vendorDirs, err := os.ReadDir("vendors")
			if err == nil {
				count := 0
				for _, dir := range vendorDirs {
					if dir.IsDir() {
						count++
					}
				}
				fmt.Printf("  📦 Vendors: %d vendor(s)\n", count)
			}
		}
	} else {
		fmt.Printf("  📂 Current directory: %s (not an airuler project)\n", filepath.Base(currentDir))
		fmt.Println("  💡 Run 'airuler init' to initialize a new project")
	}

	return nil
}

func showInstallationSummary() error {
	// Load installation tracker
	tracker, err := config.LoadGlobalInstallationTracker()
	if err != nil {
		return err
	}

	installations := tracker.GetInstallations("", "")
	if len(installations) == 0 {
		fmt.Println("  📋 No installations found")
		return nil
	}

	// Group by target and scope
	globalCount := 0
	projectCount := 0
	targetCounts := make(map[string]int)

	for _, install := range installations {
		if install.Global {
			globalCount++
		} else {
			projectCount++
		}
		targetCounts[install.Target]++
	}

	fmt.Printf("  📊 Total installations: %d\n", len(installations))
	fmt.Printf("  🌍 Global: %d, 📁 Project: %d\n", globalCount, projectCount)

	fmt.Println("  🎯 By target:")
	for target, count := range targetCounts {
		fmt.Printf("    - %s: %d\n", target, count)
	}

	return nil
}

func runUninstallAll() error {
	// Load installation tracker
	tracker, err := config.LoadGlobalInstallationTracker()
	if err != nil {
		return fmt.Errorf("failed to load installation tracker: %w", err)
	}

	// Get all installations
	installations := tracker.GetInstallations("", "")
	if len(installations) == 0 {
		fmt.Println("📋 No installations found to uninstall")
		return nil
	}

	fmt.Printf("📋 Found %d installation(s) to uninstall\n", len(installations))

	// Ask for confirmation unless --force is used
	if !uninstallForce {
		fmt.Printf("\nThis will uninstall ALL %d installation(s). Continue? [y/N]: ", len(installations))
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			fmt.Println("Operation cancelled")
			return err
		}

		if response != "y" && response != "Y" && response != "yes" {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	fmt.Println()

	// Set up uninstall flags for non-interactive bulk uninstall
	uninstallInteractive = false
	uninstallTarget = ""    // All targets
	uninstallRule = ""      // All rules
	uninstallGlobal = false // Will handle both global and project
	uninstallProject = false
	uninstallForce = true // Skip individual confirmations

	// Uninstall everything
	removed := 0
	failed := 0

	for _, installation := range installations {
		if err := uninstallSingle(installation, tracker); err != nil {
			fmt.Printf("    ⚠️  Failed to uninstall %s %s: %v\n", installation.Target, installation.Rule, err)
			failed++
		} else {
			fmt.Printf("    ✅ Uninstalled %s %s\n", installation.Target, installation.Rule)
			removed++
		}
	}

	// Save the updated tracker
	if err := config.SaveGlobalInstallationTracker(tracker); err != nil {
		fmt.Printf("Warning: failed to save installation tracker: %v\n", err)
	}

	// Summary
	fmt.Println()
	if removed > 0 {
		fmt.Printf("🎉 Successfully uninstalled %d installation(s)", removed)
		if failed > 0 {
			fmt.Printf(", %d failed", failed)
		}
		fmt.Println()
	} else if failed > 0 {
		fmt.Printf("⚠️  %d installation(s) failed to uninstall\n", failed)
	}

	return nil
}

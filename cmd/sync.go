// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"fmt"
	"os"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/ratler/airuler/internal/compiler"
	"github.com/ratler/airuler/internal/config"
	"github.com/ratler/airuler/internal/vendor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	syncNoUpdate          bool
	syncNoCompile         bool
	syncNoDeploy          bool
	syncNoGitPull         bool
	syncScope             string
	syncTargets           string
	syncDryRun            bool
	syncForce             bool
	updateInstalledGlobal bool
)

var syncCmd = &cobra.Command{
	Use:   "sync [target]",
	Short: "Sync vendors, compile templates, and update installations",
	Long: `Sync performs the complete airuler workflow in one command:
1. Pull template repository (unless --no-update or --no-git-pull)
2. Update vendor repositories (unless --no-update)
3. Compile templates (unless --no-compile)  
4. Update existing installations (unless --no-deploy)

This replaces the common workflow: git pull → update → compile → update-installed

Examples:
  airuler sync                      # Full sync: pull → update vendors → compile → deploy
  airuler sync cursor               # Sync only for Cursor target
  airuler sync --no-update          # Skip git pull and vendor updates (compile → deploy only)
  airuler sync --no-git-pull        # Skip git pull only (update vendors → compile → deploy)
  airuler sync --no-compile         # Skip compilation (pull → update vendors → deploy existing)
  airuler sync --no-deploy          # Skip deployment (pull → update vendors → compile only)
  airuler sync --scope project      # Sync only project installations
  airuler sync --targets cursor,claude  # Sync only specific targets
  airuler sync --dry-run            # Show what would happen without doing it`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		var targetFilter string
		if len(args) > 0 {
			targetFilter = args[0]
		}

		return runSync(targetFilter)
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.Flags().BoolVar(&syncNoUpdate, "no-update", false, "skip vendor updates")
	syncCmd.Flags().BoolVar(&syncNoCompile, "no-compile", false, "skip template compilation")
	syncCmd.Flags().BoolVar(&syncNoDeploy, "no-deploy", false, "skip deployment to installations")
	syncCmd.Flags().BoolVar(&syncNoGitPull, "no-git-pull", false, "skip git pull of template repository")
	syncCmd.Flags().StringVarP(&syncScope, "scope", "s", "all", "installation scope: global, project, or all")
	syncCmd.Flags().StringVarP(&syncTargets, "targets", "t", "", "comma-separated list of targets (e.g., cursor,claude)")
	syncCmd.Flags().BoolVarP(&syncDryRun, "dry-run", "n", false, "show what would happen without executing")
	syncCmd.Flags().BoolVarP(&syncForce, "force", "f", false, "skip confirmation prompts")
}

func runSync(targetFilter string) error {
	if syncDryRun {
		return runSyncDryRun(targetFilter)
	}

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

	if len(steps) == 0 {
		return fmt.Errorf("all steps disabled - nothing to do")
	}

	fmt.Printf("🔄 Sync workflow: %s\n", strings.Join(steps, " → "))
	if targetFilter != "" {
		fmt.Printf("📊 Target filter: %s\n", targetFilter)
	}
	if syncTargets != "" {
		fmt.Printf("🎯 Targets: %s\n", syncTargets)
	}
	if syncScope != "all" {
		fmt.Printf("🌍 Scope: %s\n", syncScope)
	}
	fmt.Println()

	// Step 0: Git pull template repository
	if !syncNoUpdate && !syncNoGitPull {
		if err := runSyncGitPull(); err != nil {
			return fmt.Errorf("git pull failed: %w", err)
		}
	}

	// Step 1: Update vendors
	if !syncNoUpdate {
		if err := runSyncUpdateVendors(); err != nil {
			return fmt.Errorf("vendor update failed: %w", err)
		}
	}

	// Step 2: Compile templates
	if !syncNoCompile {
		if err := runSyncCompile(targetFilter); err != nil {
			return fmt.Errorf("compilation failed: %w", err)
		}
	}

	// Step 3: Update installations
	if !syncNoDeploy {
		if err := runSyncDeploy(targetFilter); err != nil {
			return fmt.Errorf("deployment failed: %w", err)
		}
	}

	fmt.Printf("\n🎉 Sync completed successfully\n")
	return nil
}

func runSyncDryRun(targetFilter string) error {
	fmt.Println("🔍 Dry run mode - showing what would happen:")
	fmt.Println()

	// Show what steps would run
	var steps []string
	if !syncNoUpdate {
		steps = append(steps, "📥 Update vendor repositories")
	}
	if !syncNoCompile {
		steps = append(steps, "⚙️  Compile templates")
	}
	if !syncNoDeploy {
		steps = append(steps, "🚀 Update existing installations")
	}

	if len(steps) == 0 {
		fmt.Println("❌ All steps disabled - nothing would happen")
		return nil
	}

	fmt.Println("📋 Steps that would run:")
	for i, step := range steps {
		fmt.Printf("  %d. %s\n", i+1, step)
	}
	fmt.Println()

	// Show target information
	if targetFilter != "" {
		fmt.Printf("📊 Target filter: %s\n", targetFilter)
	}
	if syncTargets != "" {
		fmt.Printf("🎯 Targets: %s\n", syncTargets)
	}
	if syncScope != "all" {
		fmt.Printf("🌍 Scope: %s\n", syncScope)
	}

	// Show current state
	if !syncNoUpdate {
		fmt.Println("\n📥 Vendor repositories that would be updated:")
		if err := showVendorStatus(); err != nil {
			fmt.Printf("  Warning: could not check vendor status: %v\n", err)
		}
	}

	if !syncNoDeploy {
		fmt.Println("\n🚀 Installations that would be updated:")
		if err := showInstallationStatus(targetFilter); err != nil {
			fmt.Printf("  Warning: could not check installation status: %v\n", err)
		}
	}

	fmt.Println("\n💡 Run without --dry-run to execute these changes")
	return nil
}

func runSyncUpdateVendors() error {
	fmt.Println("📥 Updating vendor repositories...")

	// Load config
	cfg := config.NewDefaultConfig()
	if viper.ConfigFileUsed() != "" {
		if err := viper.Unmarshal(cfg); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Create vendor manager
	manager := vendor.NewManager(cfg)
	if err := manager.LoadLockFile(); err != nil {
		return fmt.Errorf("failed to load lock file: %w", err)
	}

	// Update all vendors
	if err := manager.Update(nil); err != nil {
		return fmt.Errorf("failed to update vendors: %w", err)
	}

	fmt.Println("  ✅ Vendor update completed")
	return nil
}

func runSyncCompile(targetFilter string) error {
	fmt.Println("⚙️  Compiling templates...")

	// Parse target filter
	var targets []compiler.Target
	if targetFilter != "" {
		target := compiler.Target(targetFilter)
		if !isValidTarget(target) {
			return fmt.Errorf("invalid target: %s", targetFilter)
		}
		targets = []compiler.Target{target}
	} else if syncTargets != "" {
		targetNames := strings.Split(syncTargets, ",")
		for _, name := range targetNames {
			target := compiler.Target(strings.TrimSpace(name))
			if !isValidTarget(target) {
				return fmt.Errorf("invalid target: %s", target)
			}
			targets = append(targets, target)
		}
	} else {
		targets = compiler.AllTargets
	}

	// Run compilation
	if err := compileTemplates(targets); err != nil {
		return fmt.Errorf("failed to compile templates: %w", err)
	}

	fmt.Println("  ✅ Compilation completed")
	return nil
}

func runSyncDeploy(targetFilter string) error {
	fmt.Println("🚀 Updating existing installations...")

	// Load installation tracker
	tracker, err := config.LoadGlobalInstallationTracker()
	if err != nil {
		return fmt.Errorf("failed to load installation tracker: %w", err)
	}

	// Get existing installations
	installations := tracker.GetInstallations(targetFilter, "")

	// Filter by scope if specified
	if syncScope != "all" {
		var filteredInstallations []config.InstallationRecord
		for _, install := range installations {
			if syncScope == "global" && install.Global {
				filteredInstallations = append(filteredInstallations, install)
			} else if syncScope == "project" && !install.Global {
				filteredInstallations = append(filteredInstallations, install)
			}
		}
		installations = filteredInstallations
	}

	// Filter by targets if specified
	if syncTargets != "" {
		targetList := strings.Split(syncTargets, ",")
		targetMap := make(map[string]bool)
		for _, target := range targetList {
			targetMap[strings.TrimSpace(target)] = true
		}

		var filteredInstallations []config.InstallationRecord
		for _, install := range installations {
			if targetMap[install.Target] {
				filteredInstallations = append(filteredInstallations, install)
			}
		}
		installations = filteredInstallations
	}

	if len(installations) == 0 {
		fmt.Println("  📋 No existing installations found to update")
		return nil
	}

	// Temporarily set force flag for update operations
	originalForce := updateInstalledGlobal
	if syncForce {
		updateInstalledGlobal = true
	}
	defer func() { updateInstalledGlobal = originalForce }()

	// Check which installations actually need updating and update them
	updated := 0
	failed := 0
	unchanged := 0

	for _, installation := range installations {
		status, err := updateSingleInstallationWithStatus(installation)
		if err != nil {
			fmt.Printf("    ⚠️  Failed to update %s %s: %v\n", installation.Target, installation.Rule, err)
			failed++
		} else {
			switch status {
			case "updated":
				if updated == 0 && unchanged == 0 && failed == 0 {
					// First update - show header
					fmt.Printf("  📋 Updating installation(s):\n")
				}
				fmt.Printf("    ✅ Updated %s %s\n", installation.Target, installation.Rule)
				updated++
			case "unchanged":
				unchanged++
				if viper.GetBool("verbose") {
					fmt.Printf("    ⏸️  Unchanged %s %s\n", installation.Target, installation.Rule)
				}
			case "installed":
				if updated == 0 && unchanged == 0 && failed == 0 {
					// First update - show header
					fmt.Printf("  📋 Updating installation(s):\n")
				}
				fmt.Printf("    🆕 Installed %s %s\n", installation.Target, installation.Rule)
				updated++
			}
		}
	}

	if updated > 0 {
		fmt.Printf("  ✅ Updated %d installation(s)", updated)
		if failed > 0 {
			fmt.Printf(", %d failed", failed)
		}
		fmt.Println()
	} else if failed > 0 {
		fmt.Printf("  ⚠️  %d installation(s) failed to update\n", failed)
	} else {
		fmt.Println("  ⏸️  All installations are up to date")
	}

	return nil
}

func showVendorStatus() error {
	// Load config
	cfg := config.NewDefaultConfig()
	if viper.ConfigFileUsed() != "" {
		if err := viper.Unmarshal(cfg); err != nil {
			return err
		}
	}

	// Create vendor manager
	manager := vendor.NewManager(cfg)
	if err := manager.LoadLockFile(); err != nil {
		return err
	}

	// Show vendor status
	return manager.Status()
}

func showInstallationStatus(targetFilter string) error {
	// Load installation tracker
	tracker, err := config.LoadGlobalInstallationTracker()
	if err != nil {
		return err
	}

	// Get installations
	installations := tracker.GetInstallations(targetFilter, "")

	if len(installations) == 0 {
		fmt.Println("  📋 No installations found")
		return nil
	}

	// Group by target for display
	targetGroups := make(map[string][]config.InstallationRecord)
	for _, install := range installations {
		targetGroups[install.Target] = append(targetGroups[install.Target], install)
	}

	for target, installs := range targetGroups {
		fmt.Printf("  📊 %s: %d installation(s)\n", target, len(installs))
	}

	return nil
}

func runSyncGitPull() error {
	fmt.Println("📥 Pulling template repository...")

	// Get current working directory (already template dir due to root.go)
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Try to open as git repository
	repo, err := gogit.PlainOpen(currentDir)
	if err != nil {
		// Not a git repo - silently continue
		return nil //nolint:nilerr // Intentional: we want to continue gracefully when not in a git repo
	}

	// Get worktree to check status
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Check if repository is dirty
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get repository status: %w", err)
	}

	if !status.IsClean() {
		fmt.Println("  ⚠️  Template repository has uncommitted changes, skipping git pull:")
		// List changed files
		for file, stat := range status {
			if stat.Worktree != gogit.Unmodified {
				var statusChar string
				switch stat.Worktree {
				case gogit.Modified:
					statusChar = "M"
				case gogit.Added:
					statusChar = "A"
				case gogit.Deleted:
					statusChar = "D"
				case gogit.Renamed:
					statusChar = "R"
				case gogit.Copied:
					statusChar = "C"
				case gogit.UpdatedButUnmerged:
					statusChar = "U"
				default:
					statusChar = "?"
				}
				fmt.Printf("      %s %s\n", statusChar, file)
			}
		}
		return nil
	}

	// Get current commit before pull
	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}
	oldCommit := head.Hash()

	// Perform pull
	err = worktree.Pull(&gogit.PullOptions{})
	if err != nil {
		if err == gogit.NoErrAlreadyUpToDate {
			fmt.Println("  ✅ Template repository is already up to date")
			return nil
		}

		// Pull failed - ask user if they want to continue
		fmt.Printf("  ❌ Git pull failed: %v\n", err)
		if !syncForce {
			fmt.Print("  Continue anyway? [y/N]: ")
			var response string
			if _, err := fmt.Scanln(&response); err != nil {
				return fmt.Errorf("sync cancelled by user")
			}
			if response != "y" && response != "Y" && response != "yes" {
				return fmt.Errorf("sync cancelled by user")
			}
		}
		return nil
	}

	// Get new commit after pull
	newHead, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get new HEAD: %w", err)
	}
	newCommit := newHead.Hash()

	// Show summary if commits changed
	if oldCommit != newCommit {
		fmt.Printf("  ✅ Pulled changes from %s to %s\n", oldCommit.String()[:8], newCommit.String()[:8])

		// Get the commit objects to show changed files
		oldCommitObj, err := repo.CommitObject(oldCommit)
		if err != nil {
			return nil //nolint:nilerr // Intentional: skip showing changes if we can't get commit objects
		}
		newCommitObj, err := repo.CommitObject(newCommit)
		if err != nil {
			return nil //nolint:nilerr // Intentional: skip showing changes if we can't get commit objects
		}

		// Get the trees for both commits
		oldTree, err := oldCommitObj.Tree()
		if err != nil {
			return nil //nolint:nilerr // Intentional: skip showing changes if we can't get trees
		}
		newTree, err := newCommitObj.Tree()
		if err != nil {
			return nil //nolint:nilerr // Intentional: skip showing changes if we can't get trees
		}

		// Calculate diff
		changes, err := oldTree.Diff(newTree)
		if err != nil {
			return nil //nolint:nilerr // Intentional: skip showing changes if diff calculation fails
		}

		// Show condensed file list
		if len(changes) > 0 {
			fmt.Println("  📝 Changed files:")
			for _, change := range changes {
				from, to, err := change.Files()
				if err != nil {
					continue // Skip files we can't process
				}
				if from == nil && to != nil {
					// Added file
					fmt.Printf("      + %s\n", to.Name)
				} else if from != nil && to == nil {
					// Deleted file
					fmt.Printf("      - %s\n", from.Name)
				} else if from != nil && to != nil {
					// Modified file
					fmt.Printf("      M %s\n", to.Name)
				}
			}
		}
	}

	return nil
}

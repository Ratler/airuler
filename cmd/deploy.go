// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"fmt"
	"strings"

	"github.com/ratler/airuler/internal/compiler"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	deployNoCompile   bool
	deployProject     string
	deployTargets     string
	deployInteractive bool
	deployForce       bool
	deployDryRun      bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy [target] [rule]",
	Short: "Compile templates and install to new locations",
	Long: `Deploy performs fresh installations of AI rules:
1. Compile templates (unless --no-compile)
2. Install to new locations (not update existing)

This replaces the workflow: compile → install

Scope is determined automatically:
- Global scope: when no --project flag is provided
- Project scope: when --project flag is provided

Examples:
  airuler deploy                         # Compile and install globally for all targets
  airuler deploy cursor                  # Deploy only for Cursor target globally
  airuler deploy cursor my-rule          # Deploy specific rule for Cursor globally
  airuler deploy --project ./my-app      # Deploy to specific project directory
  airuler deploy --no-compile            # Install existing compiled rules only
  airuler deploy --interactive           # Interactive template selection
  airuler deploy --targets cursor,claude # Deploy only to specific targets
  airuler deploy --dry-run               # Show what would be deployed`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		var targetFilter, ruleFilter string
		if len(args) >= 1 {
			targetFilter = args[0]
		}
		if len(args) >= 2 {
			ruleFilter = args[1]
		}

		return runDeploy(targetFilter, ruleFilter)
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().BoolVar(&deployNoCompile, "no-compile", false, "skip template compilation, use existing compiled rules")
	deployCmd.Flags().StringVarP(&deployProject, "project", "p", "", "deploy to specific project directory (sets scope to project)")
	deployCmd.Flags().StringVarP(&deployTargets, "targets", "t", "", "comma-separated list of targets (e.g., cursor,claude)")
	deployCmd.Flags().BoolVarP(&deployInteractive, "interactive", "i", false, "interactive template selection")
	deployCmd.Flags().BoolVarP(&deployForce, "force", "f", false, "overwrite existing files without confirmation")
	deployCmd.Flags().BoolVarP(&deployDryRun, "dry-run", "n", false, "show what would be deployed without executing")
}

func runDeploy(targetFilter, ruleFilter string) error {
	if deployDryRun {
		return runDeployDryRun(targetFilter, ruleFilter)
	}

	// Scope is determined automatically: global if no --project, project if --project is specified

	// Skip all output if in interactive mode
	if !deployInteractive {
		var steps []string
		if !deployNoCompile {
			steps = append(steps, "compile templates")
		}
		steps = append(steps, "install to new locations")

		fmt.Printf("🚀 Deploy workflow: %s\n", strings.Join(steps, " → "))
		if targetFilter != "" {
			fmt.Printf("📊 Target filter: %s\n", targetFilter)
		}
		if ruleFilter != "" {
			fmt.Printf("📋 Rule filter: %s\n", ruleFilter)
		}
		if deployTargets != "" {
			fmt.Printf("🎯 Targets: %s\n", deployTargets)
		}
		if deployProject != "" {
			fmt.Printf("📁 Project: %s\n", deployProject)
		} else {
			fmt.Printf("🌍 Scope: global\n")
		}
		fmt.Println()
	}

	// Step 1: Compile templates (if not skipped)
	if !deployNoCompile {
		if err := runDeployCompile(targetFilter); err != nil {
			return fmt.Errorf("compilation failed: %w", err)
		}
	}

	// Step 2: Install templates
	if err := runDeployInstall(targetFilter, ruleFilter); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	if !deployInteractive {
		fmt.Printf("\n🎉 Deploy completed successfully\n")
	}
	return nil
}

func runDeployDryRun(targetFilter, ruleFilter string) error {
	fmt.Println("🔍 Dry run mode - showing what would be deployed:")
	fmt.Println()

	// Show what steps would run
	var steps []string
	if !deployNoCompile {
		steps = append(steps, "⚙️  Compile templates")
	}
	steps = append(steps, "🚀 Install to new locations")

	fmt.Println("📋 Steps that would run:")
	for i, step := range steps {
		fmt.Printf("  %d. %s\n", i+1, step)
	}
	fmt.Println()

	// Show target and rule information
	if targetFilter != "" {
		fmt.Printf("📊 Target filter: %s\n", targetFilter)
	}
	if ruleFilter != "" {
		fmt.Printf("📋 Rule filter: %s\n", ruleFilter)
	}
	if deployTargets != "" {
		fmt.Printf("🎯 Targets: %s\n", deployTargets)
	}

	// Show installation scope
	if deployProject != "" {
		fmt.Printf("📁 Would deploy to project: %s\n", deployProject)
	} else {
		fmt.Printf("🌍 Would deploy globally\n")
	}

	// Show what templates would be compiled/installed
	if err := showDeployTargets(targetFilter, ruleFilter); err != nil {
		fmt.Printf("Warning: could not determine deployment targets: %v\n", err)
	}

	fmt.Println("\n💡 Run without --dry-run to execute these changes")
	return nil
}

func runDeployCompile(targetFilter string) error {
	if !deployInteractive {
		fmt.Println("⚙️  Compiling templates...")
	}

	// Parse target filter
	var targets []compiler.Target
	if targetFilter != "" {
		target := compiler.Target(targetFilter)
		if !isValidTarget(target) {
			return fmt.Errorf("invalid target: %s", targetFilter)
		}
		targets = []compiler.Target{target}
	} else if deployTargets != "" {
		targetNames := strings.Split(deployTargets, ",")
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
	if err := compileTemplatesWithOutput(targets, !deployInteractive); err != nil {
		return fmt.Errorf("failed to compile templates: %w", err)
	}

	if !deployInteractive {
		fmt.Println("  ✅ Compilation completed")
	}
	return nil
}

func runDeployInstall(targetFilter, ruleFilter string) error {
	if deployInteractive {
		return runDeployInteractive(targetFilter, ruleFilter)
	}

	fmt.Println("🚀 Installing templates...")

	// Set up install command flags based on deploy options
	originalInstallTarget := installTarget
	originalInstallRule := installRule
	originalInstallGlobal := installGlobal
	originalInstallProject := installProject
	originalInstallForce := installForce
	originalInstallInteractive := installInteractive

	// Configure install command
	installTarget = targetFilter
	installRule = ruleFilter
	installGlobal = deployProject == ""
	installProject = deployProject
	installForce = deployForce
	installInteractive = false

	// Restore original values after installation
	defer func() {
		installTarget = originalInstallTarget
		installRule = originalInstallRule
		installGlobal = originalInstallGlobal
		installProject = originalInstallProject
		installForce = originalInstallForce
		installInteractive = originalInstallInteractive
	}()

	// Run installation
	if err := installRules(); err != nil {
		return fmt.Errorf("failed to install rules: %w", err)
	}

	fmt.Println("  ✅ Installation completed")
	return nil
}

func runDeployInteractive(targetFilter, ruleFilter string) error {
	// Set up install command flags for interactive mode
	originalInstallTarget := installTarget
	originalInstallRule := installRule
	originalInstallGlobal := installGlobal
	originalInstallProject := installProject
	originalInstallForce := installForce
	originalInstallInteractive := installInteractive

	// Configure install command for interactive mode
	installTarget = targetFilter
	installRule = ruleFilter
	installGlobal = deployProject == ""
	installProject = deployProject
	installForce = deployForce
	installInteractive = true

	// Restore original values after installation
	defer func() {
		installTarget = originalInstallTarget
		installRule = originalInstallRule
		installGlobal = originalInstallGlobal
		installProject = originalInstallProject
		installForce = originalInstallForce
		installInteractive = originalInstallInteractive
	}()

	// Run interactive installation
	if err := runInteractiveInstall(); err != nil {
		return fmt.Errorf("interactive installation failed: %w", err)
	}

	return nil
}

func showDeployTargets(targetFilter, ruleFilter string) error {
	// Parse targets
	var targets []compiler.Target
	if targetFilter != "" {
		target := compiler.Target(targetFilter)
		if !isValidTarget(target) {
			return fmt.Errorf("invalid target: %s", targetFilter)
		}
		targets = []compiler.Target{target}
	} else if deployTargets != "" {
		targetNames := strings.Split(deployTargets, ",")
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

	// Show templates that would be compiled/installed
	if !deployNoCompile {
		fmt.Println("\n⚙️  Templates that would be compiled:")

		// Load templates and show what would be compiled
		templateDirs := []string{"templates"}
		vendorDirs := getVendorTemplateDirs()
		templateDirs = append(templateDirs, vendorDirs...)

		templates, _, err := loadTemplatesFromDirs(templateDirs)
		if err != nil {
			return fmt.Errorf("failed to load templates: %w", err)
		}

		// Filter templates by rule if specified
		if ruleFilter != "" {
			filtered := make(map[string]TemplateSource)
			for name, templateSource := range templates {
				if strings.Contains(name, ruleFilter) {
					filtered[name] = templateSource
				}
			}
			templates = filtered
		}

		if len(templates) == 0 {
			fmt.Println("    📋 No templates found")
		} else {
			// Group by source
			sourceGroups := make(map[string][]string)
			for name, templateSource := range templates {
				sourceGroups[templateSource.SourceType] = append(sourceGroups[templateSource.SourceType], name)
			}

			for source, templateNames := range sourceGroups {
				fmt.Printf("    📦 %s: %d template(s)\n", source, len(templateNames))
				if viper.GetBool("verbose") {
					for _, name := range templateNames {
						fmt.Printf("      - %s\n", name)
					}
				}
			}
		}
	}

	fmt.Println("\n🚀 Installation targets:")
	for _, target := range targets {
		fmt.Printf("    📊 %s\n", target)
	}

	return nil
}

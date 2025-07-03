// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ratler/airuler/internal/config"
	"github.com/ratler/airuler/internal/vendor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v3"
)

var vendorsCmd = &cobra.Command{
	Use:   "vendors",
	Short: "Manage vendor repositories",
	Long: `Manage vendor repositories.

Use subcommands to list, check status, or remove vendors.`,
}

var vendorsListCmd = &cobra.Command{
	Use:   "list [vendor]",
	Short: "List vendors with repository and configuration details",
	Long: `List vendor repositories with their details and configurations.

When no vendor is specified, shows all vendors with basic info and config summaries.
When a specific vendor is provided, shows detailed configuration for that vendor.

Examples:
  airuler vendors list              # List all vendors with summaries
  airuler vendors list my-rules     # Show detailed config for my-rules vendor`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return showCombinedVendorList()
		}
		return showDetailedVendorConfig(args[0])
	},
}

var vendorsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all vendors",
	Long:  `Show the update status of all vendor repositories.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		manager, err := createVendorManager()
		if err != nil {
			return err
		}
		return manager.Status()
	},
}

var vendorsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for updates without fetching",
	Long:  `Check for updates in vendor repositories without fetching them.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		manager, err := createVendorManager()
		if err != nil {
			return err
		}
		return manager.Status()
	},
}

var vendorsRemoveCmd = &cobra.Command{
	Use:   "remove <vendor>",
	Short: "Remove a vendor",
	Long:  `Remove a vendor repository from the vendors directory.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		manager, err := createVendorManager()
		if err != nil {
			return err
		}
		return manager.Remove(args[0])
	},
}

var vendorsIncludeCmd = &cobra.Command{
	Use:   "include <vendor>",
	Short: "Include a vendor in compilation",
	Long:  `Add a vendor to the include_vendors list in configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return modifyIncludeVendors(args[0], true)
	},
}

var vendorsExcludeCmd = &cobra.Command{
	Use:   "exclude <vendor>",
	Short: "Exclude a vendor from compilation",
	Long:  `Remove a vendor from the include_vendors list in configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return modifyIncludeVendors(args[0], false)
	},
}

var vendorsIncludeAllCmd = &cobra.Command{
	Use:   "include-all",
	Short: "Include all vendors in compilation",
	Long:  `Set include_vendors to ["*"] to include all vendors.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		return setIncludeVendorsAll(true)
	},
}

var vendorsExcludeAllCmd = &cobra.Command{
	Use:   "exclude-all",
	Short: "Exclude all vendors from compilation",
	Long:  `Set include_vendors to [] to exclude all vendors (local templates only).`,
	RunE: func(_ *cobra.Command, _ []string) error {
		return setIncludeVendorsAll(false)
	},
}

var (
	fetchAlias  string
	fetchUpdate bool
)

var vendorsAddCmd = &cobra.Command{
	Use:   "add <git-url>",
	Short: "Add a new vendor repository",
	Long: `Add a new vendor repository from a Git URL.

Examples:
  airuler vendors add https://github.com/user/rules-repo
  airuler vendors add https://github.com/user/rules-repo --as my-rules
  airuler vendors add https://github.com/user/rules-repo --update`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		// Reuse fetch command logic
		url := args[0]

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

		return manager.Fetch(url, fetchAlias, fetchUpdate)
	},
}

var vendorsUpdateCmd = &cobra.Command{
	Use:   "update [vendor...]",
	Short: "Update vendor repositories",
	Long: `Update vendor repositories to their latest versions.

If no vendors are specified, all vendors will be updated.

Examples:
  airuler vendors update              # Update all vendors
  airuler vendors update my-rules     # Update specific vendor
  airuler vendors update frontend,backend # Update multiple vendors`,
	RunE: func(_ *cobra.Command, args []string) error {
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

		// Parse vendor names
		var vendorNames []string
		if len(args) > 0 {
			for _, arg := range args {
				names := strings.Split(arg, ",")
				for _, name := range names {
					vendorNames = append(vendorNames, strings.TrimSpace(name))
				}
			}
		}

		// Update vendors
		return manager.Update(vendorNames)
	},
}

func init() {
	rootCmd.AddCommand(vendorsCmd)

	vendorsCmd.AddCommand(vendorsListCmd)
	vendorsCmd.AddCommand(vendorsAddCmd)
	vendorsCmd.AddCommand(vendorsUpdateCmd)
	vendorsCmd.AddCommand(vendorsStatusCmd)
	vendorsCmd.AddCommand(vendorsCheckCmd)
	vendorsCmd.AddCommand(vendorsRemoveCmd)
	vendorsCmd.AddCommand(vendorsIncludeCmd)
	vendorsCmd.AddCommand(vendorsExcludeCmd)
	vendorsCmd.AddCommand(vendorsIncludeAllCmd)
	vendorsCmd.AddCommand(vendorsExcludeAllCmd)

	// Add flags for the add command (reuse fetch flags)
	vendorsAddCmd.Flags().StringVarP(&fetchAlias, "as", "a", "", "alias for the vendor")
	vendorsAddCmd.Flags().BoolVarP(&fetchUpdate, "update", "u", false, "update if vendor already exists")
}

func createVendorManager() (*vendor.Manager, error) {
	// Load config
	cfg := config.NewDefaultConfig()
	if viper.ConfigFileUsed() != "" {
		if err := viper.Unmarshal(cfg); err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Create vendor manager
	manager := vendor.NewManager(cfg)
	if err := manager.LoadLockFile(); err != nil {
		return nil, fmt.Errorf("failed to load lock file: %w", err)
	}

	return manager, nil
}

func loadProjectConfig() (*config.Config, error) {
	cfg := config.NewDefaultConfig()
	if viper.ConfigFileUsed() != "" {
		if err := viper.Unmarshal(cfg); err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}
	return cfg, nil
}

func saveProjectConfig(cfg *config.Config) error {
	configPath := "airuler.yaml"
	if viper.ConfigFileUsed() != "" {
		configPath = viper.ConfigFileUsed()
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, data, 0600)
}

func modifyIncludeVendors(vendorName string, include bool) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	includeVendors := cfg.Defaults.IncludeVendors

	if include {
		// Add vendor if not already included
		found := false
		for _, v := range includeVendors {
			if v == vendorName || v == "*" {
				found = true
				break
			}
		}
		if !found {
			cfg.Defaults.IncludeVendors = append(includeVendors, vendorName)
			fmt.Printf("Added '%s' to include_vendors\n", vendorName)
		} else {
			fmt.Printf("Vendor '%s' is already included\n", vendorName)
		}
	} else {
		// Remove vendor from include list
		newIncludeVendors := []string{}
		found := false
		for _, v := range includeVendors {
			if v != vendorName {
				newIncludeVendors = append(newIncludeVendors, v)
			} else {
				found = true
			}
		}
		if found {
			cfg.Defaults.IncludeVendors = newIncludeVendors
			fmt.Printf("Removed '%s' from include_vendors\n", vendorName)
		} else {
			fmt.Printf("Vendor '%s' was not in include_vendors\n", vendorName)
		}
	}

	return saveProjectConfig(cfg)
}

func setIncludeVendorsAll(includeAll bool) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	if includeAll {
		cfg.Defaults.IncludeVendors = []string{"*"}
		fmt.Println("Set include_vendors to include all vendors")
	} else {
		cfg.Defaults.IncludeVendors = []string{}
		fmt.Println("Set include_vendors to exclude all vendors (local templates only)")
	}

	return saveProjectConfig(cfg)
}

func getStringOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func getVendorTargetNames(targets map[string]config.TargetConfig) []string {
	var names []string
	for target := range targets {
		names = append(names, target)
	}
	return names
}

// showCombinedVendorList displays both repository info and config summaries for all vendors
func showCombinedVendorList() error {
	// Get vendor manager for repository info
	manager, err := createVendorManager()
	if err != nil {
		return err
	}

	// Get current working directory for config loading
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load project configuration
	projectConfig, err := loadProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to load project config: %w", err)
	}

	// Load vendor configurations
	vendorConfigs, err := config.LoadVendorConfigs(currentDir, projectConfig)
	if err != nil {
		return fmt.Errorf("failed to load vendor configurations: %w", err)
	}

	// Get vendor repository info
	lockFile := manager.GetLockFile()
	if len(lockFile.Vendors) == 0 {
		fmt.Println("No vendors found")
		return nil
	}

	fmt.Println("📦 Vendors")
	fmt.Println("==========")

	for vendorName, vendorData := range lockFile.Vendors {
		fmt.Printf("\n🏷️  %s\n", vendorName)

		// Repository info
		fmt.Printf("   %-20s %s\n", "URL:", vendorData.URL)
		fmt.Printf("   %-20s %s\n", "Commit:", vendorData.Commit)
		fmt.Printf("   %-20s %s\n", "Fetched:", vendorData.FetchedAt.Format("2006-01-02 15:04:05"))

		// Configuration info (if available)
		if vendorConfig, exists := vendorConfigs.VendorConfigs[vendorName]; exists {
			if vendorConfig.Vendor.Name != "" && vendorConfig.Vendor.Name != vendorName {
				fmt.Printf("   %-20s %s\n", "Name:", vendorConfig.Vendor.Name)
			}
			if vendorConfig.Vendor.Description != "" {
				fmt.Printf("   %-20s %s\n", "Description:", vendorConfig.Vendor.Description)
			}
			if vendorConfig.Vendor.Version != "" {
				fmt.Printf("   %-20s %s\n", "Version:", vendorConfig.Vendor.Version)
			}
			if len(vendorConfig.TemplateDefaults) > 0 {
				fmt.Printf("   %-20s %d defaults\n", "Template Defaults:", len(vendorConfig.TemplateDefaults))
			}
			if len(vendorConfig.Variables) > 0 {
				fmt.Printf("   %-20s %d variables\n", "Variables:", len(vendorConfig.Variables))
			}
			if len(vendorConfig.Targets) > 0 {
				fmt.Printf("   %-20s %v\n", "Target Configs:", getVendorTargetNames(vendorConfig.Targets))
			}
		} else {
			fmt.Printf("   %-20s %s\n", "Config:", "No configuration found")
		}
	}

	return nil
}

// showDetailedVendorConfig displays detailed configuration for a specific vendor
func showDetailedVendorConfig(vendorName string) error {
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load project configuration
	projectConfig, err := loadProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to load project config: %w", err)
	}

	// Load vendor configurations
	vendorConfigs, err := config.LoadVendorConfigs(currentDir, projectConfig)
	if err != nil {
		return fmt.Errorf("failed to load vendor configurations: %w", err)
	}

	// Check if vendor exists in repository
	manager, err := createVendorManager()
	if err != nil {
		return err
	}

	lockFile := manager.GetLockFile()
	vendorData, repoExists := lockFile.Vendors[vendorName]

	// Check if vendor config exists
	vendorConfig, configExists := vendorConfigs.VendorConfigs[vendorName]

	if !repoExists && !configExists {
		return fmt.Errorf("vendor '%s' not found", vendorName)
	}

	fmt.Printf("📦 Vendor: %s\n", vendorName)
	fmt.Println("=" + strings.Repeat("=", len(vendorName)+9))

	// Repository information
	if repoExists {
		fmt.Println("\n📂 Repository Information:")
		fmt.Printf("   URL:     %s\n", vendorData.URL)
		fmt.Printf("   Commit:  %s\n", vendorData.Commit)
		fmt.Printf("   Fetched: %s\n", vendorData.FetchedAt.Format("2006-01-02 15:04:05"))
	}

	// Configuration information
	if configExists {
		fmt.Println("\n🏷️  Vendor Information:")
		fmt.Printf("   Name:        %s\n", getStringOrDefault(vendorConfig.Vendor.Name, vendorName))
		fmt.Printf("   Description: %s\n", getStringOrDefault(vendorConfig.Vendor.Description, "No description"))
		fmt.Printf("   Version:     %s\n", getStringOrDefault(vendorConfig.Vendor.Version, "Unknown"))
		fmt.Printf("   Author:      %s\n", getStringOrDefault(vendorConfig.Vendor.Author, "Unknown"))
		fmt.Printf("   Homepage:    %s\n", getStringOrDefault(vendorConfig.Vendor.Homepage, "None"))

		// Template defaults
		if len(vendorConfig.TemplateDefaults) > 0 {
			fmt.Println("\n⚙️  Template Defaults:")
			for key, value := range vendorConfig.TemplateDefaults {
				fmt.Printf("   %-15s %v\n", key+":", value)
			}
		}

		// Variables
		if len(vendorConfig.Variables) > 0 {
			fmt.Println("\n🔧 Variables:")
			for key, value := range vendorConfig.Variables {
				fmt.Printf("   %-15s %v\n", key+":", value)
			}
		}

		// Target configurations
		if len(vendorConfig.Targets) > 0 {
			fmt.Println("\n🎯 Target Configurations:")
			for target, targetConfig := range vendorConfig.Targets {
				fmt.Printf("   %s:\n", target)
				if targetConfig.DefaultMode != "" {
					fmt.Printf("     %-18s %s\n", "Default Mode:", targetConfig.DefaultMode)
				}
			}
		}
	} else {
		fmt.Println("\n🏷️  Configuration: No vendor configuration found")
	}

	return nil
}

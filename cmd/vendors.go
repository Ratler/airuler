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
	Use:   "list",
	Short: "List all vendors",
	Long:  `List all vendor repositories with their details.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		manager, err := createVendorManager()
		if err != nil {
			return err
		}
		return manager.List()
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

var vendorsConfigCmd = &cobra.Command{
	Use:   "config [vendor]",
	Short: "View vendor configurations",
	Long:  `View vendor configurations. If no vendor is specified, shows all vendor configs.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return showVendorConfigs(args)
	},
}

func init() {
	rootCmd.AddCommand(vendorsCmd)

	vendorsCmd.AddCommand(vendorsListCmd)
	vendorsCmd.AddCommand(vendorsStatusCmd)
	vendorsCmd.AddCommand(vendorsCheckCmd)
	vendorsCmd.AddCommand(vendorsRemoveCmd)
	vendorsCmd.AddCommand(vendorsIncludeCmd)
	vendorsCmd.AddCommand(vendorsExcludeCmd)
	vendorsCmd.AddCommand(vendorsIncludeAllCmd)
	vendorsCmd.AddCommand(vendorsExcludeAllCmd)
	vendorsCmd.AddCommand(vendorsConfigCmd)
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

// showVendorConfigs displays vendor configurations
func showVendorConfigs(args []string) error {
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

	if len(args) == 0 {
		// Show all vendor configurations
		fmt.Println("📦 Vendor Configurations")
		fmt.Println("========================")

		if len(vendorConfigs.VendorConfigs) == 0 {
			fmt.Println("No vendor configurations found.")
			return nil
		}

		for vendorName, vendorConfig := range vendorConfigs.VendorConfigs {
			fmt.Printf("\n🏷️  %s\n", vendorName)
			fmt.Printf("   %-20s %s\n", "Name:", getStringOrDefault(vendorConfig.Vendor.Name, vendorName))
			fmt.Printf("   %-20s %s\n", "Description:", getStringOrDefault(vendorConfig.Vendor.Description, "No description"))
			fmt.Printf("   %-20s %s\n", "Version:", getStringOrDefault(vendorConfig.Vendor.Version, "Unknown"))

			if len(vendorConfig.TemplateDefaults) > 0 {
				fmt.Printf("   %-20s %d defaults\n", "Template Defaults:", len(vendorConfig.TemplateDefaults))
			}
			if len(vendorConfig.Variables) > 0 {
				fmt.Printf("   %-20s %d variables\n", "Variables:", len(vendorConfig.Variables))
			}
			if len(vendorConfig.Targets) > 0 {
				fmt.Printf("   %-20s %v\n", "Target Configs:", getVendorTargetNames(vendorConfig.Targets))
			}
		}
	} else {
		// Show specific vendor configuration
		vendorName := args[0]
		vendorConfig, exists := vendorConfigs.VendorConfigs[vendorName]
		if !exists {
			return fmt.Errorf("vendor '%s' not found", vendorName)
		}

		fmt.Printf("📦 Vendor Configuration: %s\n", vendorName)
		fmt.Println("=" + strings.Repeat("=", len(vendorName)+25))

		// Vendor manifest
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

		// Compilation settings section removed - no active compilation config fields
	}

	return nil
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

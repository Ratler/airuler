package cmd

import (
	"fmt"
	"os"

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

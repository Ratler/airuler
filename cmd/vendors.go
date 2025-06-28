package cmd

import (
	"fmt"

	"github.com/ratler/airuler/internal/config"
	"github.com/ratler/airuler/internal/vendor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	RunE: func(cmd *cobra.Command, args []string) error {
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
	RunE: func(cmd *cobra.Command, args []string) error {
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
	RunE: func(cmd *cobra.Command, args []string) error {
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
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := createVendorManager()
		if err != nil {
			return err
		}
		return manager.Remove(args[0])
	},
}

func init() {
	rootCmd.AddCommand(vendorsCmd)

	vendorsCmd.AddCommand(vendorsListCmd)
	vendorsCmd.AddCommand(vendorsStatusCmd)
	vendorsCmd.AddCommand(vendorsCheckCmd)
	vendorsCmd.AddCommand(vendorsRemoveCmd)
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

package cmd

import (
	"fmt"

	"github.com/ratler/airuler/internal/config"
	"github.com/ratler/airuler/internal/vendor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	fetchAlias  string
	fetchUpdate bool
)

var fetchCmd = &cobra.Command{
	Use:   "fetch [git-url]",
	Short: "Fetch rules from a Git repository or restore missing vendors",
	Long: `Fetch rules from a Git repository and add as a vendor, or restore missing vendors from lock file.

If a git-url is provided, the repository will be cloned to the vendors/ directory.
If no arguments are provided, missing vendors from the lock file will be restored.

Examples:
  airuler fetch                                      # Restore missing vendors from lock file
  airuler fetch https://github.com/user/rules-repo  # Fetch new vendor
  airuler fetch https://github.com/user/rules-repo --as my-rules
  airuler fetch https://github.com/user/rules-repo --update`,
	Args: cobra.MaximumNArgs(1),
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

		if len(args) == 0 {
			// Restore missing vendors from lock file
			return manager.RestoreMissingVendors()
		}

		// Fetch new vendor
		url := args[0]
		return manager.Fetch(url, fetchAlias, fetchUpdate)
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)

	fetchCmd.Flags().StringVar(&fetchAlias, "as", "", "alias for the vendor")
	fetchCmd.Flags().BoolVar(&fetchUpdate, "update", false, "update if vendor already exists")
}

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
	Use:   "fetch <git-url>",
	Short: "Fetch rules from a Git repository",
	Long: `Fetch rules from a Git repository and add as a vendor.

The repository will be cloned to the vendors/ directory and can be used
in compilation and installation commands.

Examples:
  airuler fetch https://github.com/user/rules-repo
  airuler fetch https://github.com/user/rules-repo --as my-rules
  airuler fetch https://github.com/user/rules-repo --update`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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

		// Fetch vendor
		return manager.Fetch(url, fetchAlias, fetchUpdate)
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)

	fetchCmd.Flags().StringVar(&fetchAlias, "as", "", "alias for the vendor")
	fetchCmd.Flags().BoolVar(&fetchUpdate, "update", false, "update if vendor already exists")
}

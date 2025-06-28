package cmd

import (
	"fmt"
	"strings"

	"github.com/ratler/airuler/internal/config"
	"github.com/ratler/airuler/internal/vendor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	updateInteractive bool
	updateDryRun      bool
)

var updateCmd = &cobra.Command{
	Use:   "update [vendor...]",
	Short: "Update vendor repositories",
	Long: `Update vendor repositories to their latest versions.

If no vendors are specified, all vendors will be updated.

Examples:
  airuler update                    # Update all vendors
  airuler update my-rules           # Update specific vendor
  airuler update frontend,backend   # Update multiple vendors
  airuler update --interactive      # Update with confirmation prompts
  airuler update --dry-run          # Show what would be updated`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		if updateDryRun {
			return showUpdateStatus(manager, vendorNames)
		}

		// Update vendors
		return manager.Update(vendorNames)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().BoolVar(&updateInteractive, "interactive", false, "interactive mode with confirmation prompts")
	updateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "show what would be updated without doing it")
}

func showUpdateStatus(manager *vendor.Manager, vendorNames []string) error {
	fmt.Println("Update status (dry run):")
	return manager.Status()
}

package cmd

import (
	"fmt"
	"os"

	"github.com/ratler/airuler/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "airuler",
	Short: "AI Rules Template Engine",
	Long: `airuler is a CLI tool that compiles AI rule templates into target-specific formats
for various AI coding assistants including Cursor, Claude Code, Cline/Roo, and GitHub Copilot.

It supports template inheritance, vendor management, and multi-repository workflows.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: project dir or ~/.config/airuler/airuler.yaml)")
	rootCmd.PersistentFlags().Bool("verbose", false, "verbose output")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Set up config search paths with proper precedence
		viper.SetConfigName("airuler")
		viper.SetConfigType("yaml")

		// 1. Look in current directory first (project-specific config)
		viper.AddConfigPath(".")

		// 2. Look in global config directory
		if configDir, err := config.GetConfigDir(); err == nil {
			viper.AddConfigPath(configDir)
		}
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil && viper.GetBool("verbose") {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

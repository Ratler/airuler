// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"fmt"
	"os"

	"github.com/ratler/airuler/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile            string
	originalWorkingDir string
)

var rootCmd = &cobra.Command{
	Use:   "airuler",
	Short: "AI Rules Template Engine",
	Long: `airuler is a CLI tool that compiles AI rule templates into target-specific formats
for various AI coding assistants including Cursor, Claude Code, Cline, and GitHub Copilot.

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
	cobra.OnInitialize(setupWorkingDirectory)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: project dir or ~/.config/airuler/airuler.yaml)")
	rootCmd.PersistentFlags().Bool("verbose", false, "verbose output")
	if err := viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose")); err != nil {
		// This should never happen with a valid flag, but handle it gracefully
		panic(fmt.Sprintf("failed to bind verbose flag: %v", err))
	}
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

func setupWorkingDirectory() {
	// Get current working directory and store as original
	currentDir, err := os.Getwd()
	if err != nil {
		// If we can't get current dir, continue without switching
		return
	}

	// Store the original working directory for commands that need it
	originalWorkingDir = currentDir

	// Check if current directory is a template directory
	if config.IsTemplateDirectory(currentDir) {
		// Update the last template directory
		if err := config.UpdateLastTemplateDir(currentDir); err != nil && viper.GetBool("verbose") {
			fmt.Printf("Warning: Failed to update last template directory: %v\n", err)
		}
		return // We're already in a template directory, no need to switch
	}

	// We're not in a template directory, check if we have a last template directory
	lastTemplateDir, err := config.GetLastTemplateDir()
	if err != nil {
		if viper.GetBool("verbose") {
			fmt.Printf("Warning: Failed to get last template directory: %v\n", err)
		}
		return
	}

	// If no last template directory is set, continue normally
	if lastTemplateDir == "" {
		return
	}

	// Verify that the last template directory still exists and is valid
	if !config.IsTemplateDirectory(lastTemplateDir) {
		fmt.Fprintf(os.Stderr, "Error: Last template directory '%s' is no longer a valid airuler template directory\n", lastTemplateDir)
		fmt.Fprintf(os.Stderr, "Please run 'airuler config set-template-dir <path>' to set a new template directory\n")
		os.Exit(1)
	}

	// Switch to the last template directory
	if err := os.Chdir(lastTemplateDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to change to template directory '%s': %v\n", lastTemplateDir, err)
		os.Exit(1)
	}

	// Inform user that we're using the template directory
	fmt.Printf("Using template directory: %s\n", lastTemplateDir)
}

// GetOriginalWorkingDir returns the working directory that was active when airuler started,
// before any automatic directory switching occurred. This is useful for resolving relative
// paths that should be relative to where the user ran the command from.
func GetOriginalWorkingDir() string {
	if originalWorkingDir == "" {
		// Fallback to current directory if not set
		if wd, err := os.Getwd(); err == nil {
			return wd
		}
		return "."
	}
	return originalWorkingDir
}

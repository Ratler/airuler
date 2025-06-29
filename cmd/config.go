// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/ratler/airuler/internal/config"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage global configuration",
	Long: `Manage global airuler configuration.

Global config is stored in:
  - Linux/macOS: ~/.config/airuler/airuler.yaml
  - Windows: %APPDATA%\airuler\airuler.yaml

Project-specific config (./airuler.yaml) takes precedence over global config.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize global configuration",
	Long:  `Create a global configuration file with default settings.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		return initGlobalConfig()
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file paths",
	Long:  `Show the paths where airuler looks for configuration files.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		return showConfigPaths()
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open global config for editing",
	Long:  `Open the global configuration file in the default editor.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		return editGlobalConfig()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configEditCmd)
}

func initGlobalConfig() error {
	globalConfigPath, err := config.GetGlobalConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get global config path: %w", err)
	}

	// Check if global config already exists
	if _, err := os.Stat(globalConfigPath); err == nil {
		return fmt.Errorf("global config already exists at %s", globalConfigPath)
	}

	// Create default config
	cfg := config.NewDefaultConfig()
	cfgData, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(globalConfigPath, cfgData, 0600); err != nil {
		return fmt.Errorf("failed to write global config: %w", err)
	}

	fmt.Printf("✅ Global configuration initialized at: %s\n", globalConfigPath)
	return nil
}

func showConfigPaths() error {
	fmt.Println("Configuration file locations (in order of precedence):")

	// 1. Command line flag
	fmt.Println("  1. --config flag (if specified)")

	// 2. Current directory
	if config.HasLocalConfig() {
		fmt.Println("  2. ./airuler.yaml (✅ found)")
	} else {
		fmt.Println("  2. ./airuler.yaml (not found)")
	}

	// 3. Global config
	globalConfigPath, err := config.GetConfigFile()
	if err != nil {
		fmt.Printf("  3. Global config (error: %v)\n", err)
	} else {
		if config.HasGlobalConfig() {
			fmt.Printf("  3. %s (✅ found)\n", globalConfigPath)
		} else {
			fmt.Printf("  3. %s (not found)\n", globalConfigPath)
		}
	}

	// Show which config is currently being used
	fmt.Println("\nTo create global config:")
	fmt.Println("  airuler config init")

	return nil
}

func editGlobalConfig() error {
	globalConfigPath, err := config.GetGlobalConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get global config path: %w", err)
	}

	// Check if config exists
	if !config.HasGlobalConfig() {
		fmt.Printf("Global config does not exist. Create it first with:\n")
		fmt.Printf("  airuler config init\n")
		return nil
	}

	// Check if we're in test mode (don't actually launch editor)
	if os.Getenv("AIRULER_TEST_MODE") != "" {
		fmt.Printf("Opening %s with nvim...\n", globalConfigPath)
		fmt.Printf("(If this doesn't work, manually edit: %s)\n", globalConfigPath)
		return nil
	}

	// Get editor preference in order of precedence
	editor := getEditor()
	if editor == "" {
		fmt.Printf("No editor found. Please manually edit: %s\n", globalConfigPath)
		return nil
	}

	fmt.Printf("Opening %s with %s...\n", globalConfigPath, editor)
	fmt.Printf("(If this doesn't work, manually edit: %s)\n", globalConfigPath)

	// Launch the editor
	cmd := exec.Command(editor, globalConfigPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// getEditor returns the preferred editor in order of precedence
func getEditor() string {
	// Try environment variables first
	if editor := os.Getenv("EDITOR"); editor != "" {
		if _, err := exec.LookPath(editor); err == nil {
			return editor
		}
	}

	if visual := os.Getenv("VISUAL"); visual != "" {
		if _, err := exec.LookPath(visual); err == nil {
			return visual
		}
	}

	// Platform-specific defaults
	var defaults []string
	if runtime.GOOS == "windows" {
		defaults = []string{"notepad", "code", "vim"}
	} else {
		defaults = []string{"vim", "vi", "nano", "code"}
	}

	// Try common editors
	for _, editor := range defaults {
		if _, err := exec.LookPath(editor); err == nil {
			return editor
		}
	}

	return ""
}

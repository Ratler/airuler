package cmd

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestRootCommand(t *testing.T) {
	// Test that root command exists and has basic properties
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	if rootCmd.Use != "airuler" {
		t.Errorf("rootCmd.Use = %q, expected %q", rootCmd.Use, "airuler")
	}

	if rootCmd.Short != "AI Rules Template Engine" {
		t.Errorf("rootCmd.Short = %q, expected %q", rootCmd.Short, "AI Rules Template Engine")
	}

	// Check that Long description contains key information
	expectedLongParts := []string{
		"airuler",
		"AI rule templates",
		"Cursor",
		"Claude Code",
		"Cline/Roo",
		"GitHub Copilot",
	}

	for _, part := range expectedLongParts {
		if !containsSubstring(rootCmd.Long, part) {
			t.Errorf("rootCmd.Long missing expected part: %s", part)
		}
	}
}

func TestRootCommandFlags(t *testing.T) {
	// Check that expected flags exist
	flags := rootCmd.PersistentFlags()

	// Check config flag
	configFlag := flags.Lookup("config")
	if configFlag == nil {
		t.Error("config flag not found")
	} else {
		if configFlag.Usage != "config file (default: project dir or ~/.config/airuler/airuler.yaml)" {
			t.Errorf("config flag usage = %q, expected proper usage text", configFlag.Usage)
		}
	}

	// Check verbose flag
	verboseFlag := flags.Lookup("verbose")
	if verboseFlag == nil {
		t.Error("verbose flag not found")
	} else {
		if verboseFlag.Usage != "verbose output" {
			t.Errorf("verbose flag usage = %q, expected %q", verboseFlag.Usage, "verbose output")
		}
	}
}

func TestRootCommandSubcommands(t *testing.T) {
	// Check that expected subcommands exist
	expectedCommands := []string{
		"init",
		"compile",
		"install",
		"config",
		"fetch",
		"update",
		"vendors",
		"watch",
	}

	commands := rootCmd.Commands()
	commandMap := make(map[string]*cobra.Command)
	for _, cmd := range commands {
		commandMap[cmd.Name()] = cmd
	}

	for _, expectedName := range expectedCommands {
		if _, exists := commandMap[expectedName]; !exists {
			t.Errorf("Expected subcommand %q not found", expectedName)
		}
	}

	// Check that we have at least the expected number of commands
	// (completion and help may or may not be present depending on cobra version)
	if len(commands) < len(expectedCommands) {
		t.Errorf("Expected at least %d commands, found %d", len(expectedCommands), len(commands))
	}
}

func TestInitConfig(_ *testing.T) {
	// Save original environment
	originalConfigFile := cfgFile
	defer func() { cfgFile = originalConfigFile }()

	// Test with explicit config file
	cfgFile = "/path/to/custom/config.yaml"

	// Reset viper state
	viper.Reset()

	// Call initConfig
	initConfig()

	// Check that viper was configured with the custom config file
	// Note: We can't easily test the full viper functionality without
	// actually creating config files, but we can test that it doesn't panic

	// Reset for next test
	cfgFile = ""
	viper.Reset()

	// Test with default config search paths
	initConfig()

	// Again, mainly testing that it doesn't panic with default settings
}

func TestInitConfigWithEnvironment(_ *testing.T) {
	// Save original environment
	originalConfigFile := cfgFile
	defer func() { cfgFile = originalConfigFile }()

	// Set environment variable
	os.Setenv("VERBOSE", "true")
	defer os.Unsetenv("VERBOSE")

	// Reset viper state
	viper.Reset()
	cfgFile = ""

	// Call initConfig
	initConfig()

	// Test passes if no panic occurs
	// In a real scenario, we'd check that viper picked up the environment variable
}

func TestExecuteFunction(t *testing.T) {
	// We can't easily test Execute() because it calls os.Exit on error
	// But we can test that the function exists and is callable

	// Save original args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Set args to just show help (which shouldn't exit with error)
	os.Args = []string{"airuler", "--help"}

	// Note: Execute() will call os.Exit, so we can't test it directly
	// in a unit test. In integration tests, we'd run the binary separately.

	// Instead, we'll test that the command structure is valid
	if err := rootCmd.ValidateArgs([]string{"--help"}); err != nil {
		// This is expected to not validate since --help is handled by cobra
		// The test passes if we get here without panicking
		t.Logf("Expected validation error for --help flag: %v", err)
	}
}

func TestCommandStructure(t *testing.T) {
	// Test that all commands have proper structure
	commands := rootCmd.Commands()

	for _, cmd := range commands {
		// Check that each command has a Use field
		if cmd.Use == "" {
			t.Errorf("Command has empty Use field: %+v", cmd)
		}

		// Check that each command has a Short description
		if cmd.Short == "" {
			t.Errorf("Command %q has empty Short description", cmd.Use)
		}

		// Commands should either have a Run function or subcommands
		if cmd.RunE == nil && cmd.Run == nil && len(cmd.Commands()) == 0 {
			// Skip completion and help commands which are auto-generated
			if cmd.Name() != "completion" && cmd.Name() != "help" {
				t.Errorf("Command %q has no Run function and no subcommands", cmd.Use)
			}
		}
	}
}

func TestFlagBinding(t *testing.T) {
	// Reset viper to clean state
	viper.Reset()

	// Test that flags are properly bound to viper
	// This is mainly testing that the binding doesn't panic

	// Set a flag value
	err := rootCmd.PersistentFlags().Set("verbose", "true")
	if err != nil {
		t.Errorf("Failed to set verbose flag: %v", err)
	}

	// The actual binding is tested by checking that viper can read the value
	// but this requires the flag to be parsed, which happens during command execution
}

func TestConfigFileDetection(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Save current directory and change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create local config file
	localConfig := "test: local"
	if err := os.WriteFile("airuler.yaml", []byte(localConfig), 0644); err != nil {
		t.Fatalf("Failed to create local config: %v", err)
	}

	// Reset viper and cfgFile
	viper.Reset()
	cfgFile = ""

	// Call initConfig
	initConfig()

	// Test passes if no panic occurs
	// The actual config loading is tested in other functions
}

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ratler/airuler/internal/config"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new airuler project",
	Long: `Initialize a new airuler project with the standard directory structure:

rules/
├── templates/
│   ├── base/
│   ├── partials/
│   └── examples/
├── vendors/
├── compiled/
│   ├── cursor/
│   ├── claude/
│   ├── cline/
│   └── copilot/
├── airuler.yaml
└── airuler.lock`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return initProject()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func initProject() error {
	// Check if airuler.yaml already exists
	if _, err := os.Stat("airuler.yaml"); err == nil {
		return fmt.Errorf("airuler.yaml already exists. Project appears to be already initialized")
	}

	// Create directory structure
	dirs := []string{
		"templates/partials",
		"templates/examples",
		"vendors",
		"compiled/cursor",
		"compiled/claude",
		"compiled/cline",
		"compiled/copilot",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create default config file
	cfg := config.NewDefaultConfig()
	cfgData, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile("airuler.yaml", cfgData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Create empty lock file
	lockFile := &config.LockFile{
		Vendors: make(map[string]config.VendorLock),
	}
	lockData, err := yaml.Marshal(lockFile)
	if err != nil {
		return fmt.Errorf("failed to marshal lock file: %w", err)
	}

	if err := os.WriteFile("airuler.lock", lockData, 0644); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	// Create .gitignore file
	gitignoreContent := `# Compiled rules (uncomment if you don't want to commit generated files)
# compiled/

# Vendor dependencies
vendors/

# Backup files
*.backup.*

# OS files
.DS_Store
Thumbs.db

# Editor files
.vscode/
.idea/
*.swp
*.swo
*~
`
	if err := os.WriteFile(".gitignore", []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to write .gitignore file: %w", err)
	}

	// Create example template
	exampleTemplate := `# {{.Name}} Rule

{{if eq .Target "cursor"}}---
description: {{.Description}}
globs: {{.Globs}}
alwaysApply: true
---
{{end}}

This is an example rule template for {{.Target}}.

{{if eq .Target "claude"}}Arguments: $ARGUMENTS{{end}}

## Guidelines

- Follow coding best practices
- Write clean, readable code
- Include proper error handling`

	examplePath := filepath.Join("templates", "examples", "example.tmpl")
	if err := os.WriteFile(examplePath, []byte(exampleTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write example template: %w", err)
	}

	// Ask user if they want to initialize git repository (skip in test mode)
	if os.Getenv("AIRULER_TEST_MODE") == "" {
		initGit := askYesNo("Initialize git repository? (y/n)")
		if initGit {
			if err := initializeGitRepo(); err != nil {
				fmt.Printf("⚠️  Warning: Failed to initialize git repository: %v\n", err)
			}
		}
	}

	fmt.Println("✅ airuler project initialized successfully!")
	fmt.Println("\nCreated:")
	for _, dir := range dirs {
		fmt.Printf("  📁 %s/\n", dir)
	}
	fmt.Println("  📄 airuler.yaml")
	fmt.Println("  📄 airuler.lock")
	fmt.Println("  📄 .gitignore")
	fmt.Printf("  📄 %s\n", examplePath)

	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add your templates to templates/")
	fmt.Println("  2. Run 'airuler compile' to generate rules")
	fmt.Println("  3. Run 'airuler install' to install rules")

	return nil
}

// askYesNo prompts the user for a yes/no question and returns true for yes
func askYesNo(prompt string) bool {
	fmt.Print(prompt + " ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// initializeGitRepo initializes a git repository and creates an initial commit
func initializeGitRepo() error {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH")
	}

	// Check if already in a git repository
	if _, err := os.Stat(".git"); err == nil {
		return fmt.Errorf("directory is already a git repository")
	}

	// Initialize git repository
	cmd := exec.Command("git", "init")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run 'git init': %w", err)
	}

	// Add all files
	cmd = exec.Command("git", "add", ".")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run 'git add .': %w", err)
	}

	// Create initial commit
	cmd = exec.Command("git", "commit", "-m", "Initial airuler project setup")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	fmt.Println("📦 Git repository initialized with initial commit")
	return nil
}

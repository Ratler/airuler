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
	Use:   "init [path]",
	Short: "Initialize a new airuler project",
	Long: `Initialize a new airuler project with the standard directory structure.

If no path is provided, initializes in the current directory.
If a path is provided, creates the directory and initializes the project there.

Project Structure:
├── templates/          # Your rule templates
│   ├── partials/      # Reusable template components
│   └── examples/      # Example templates  
├── vendors/           # External rule repositories
├── compiled/          # Generated rules for each target
│   ├── cursor/       # Cursor IDE rules
│   ├── claude/       # Claude Code rules
│   ├── cline/        # Cline/Roo rules
│   └── copilot/      # GitHub Copilot rules
├── airuler.yaml       # Project configuration
├── airuler.lock       # Vendor dependency locks
└── README.md          # Project documentation

Examples:
  airuler init                    # Initialize in current directory
  airuler init my-rules-project   # Create and initialize new directory
  airuler init ../other-project   # Initialize in relative path`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var targetPath string
		if len(args) > 0 {
			targetPath = args[0]
		} else {
			targetPath = "."
		}
		return initProject(targetPath)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func initProject(targetPath string) error {
	var originalDir string
	var err error

	// If a path is provided, create the directory and change to it
	if targetPath != "." {
		// Get current directory to restore later if needed
		originalDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Create the target directory (including parent directories)
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
		}

		// Change to the target directory
		if err := os.Chdir(targetPath); err != nil {
			return fmt.Errorf("failed to change to directory %s: %w", targetPath, err)
		}

		// Set up cleanup in case of error
		defer func() {
			if originalDir != "" {
				os.Chdir(originalDir)
			}
		}()
	}

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

	// Create README.md file
	readmeContent := `# AI Rules Project

This is an airuler project for managing AI coding assistant rules and templates.

## About airuler

airuler is a CLI tool that compiles AI rule templates into target-specific formats for various AI coding assistants including Cursor, Claude Code, Cline/Roo, and GitHub Copilot. It supports template inheritance, vendor management, and multi-repository workflows.

For more information, documentation, and source code, visit: https://github.com/Ratler/airuler/

## Project Structure

` + "```" + `
.
├── templates/          # Your rule templates
│   ├── partials/      # Reusable template components
│   └── examples/      # Example templates
├── vendors/           # External rule repositories (git-ignored)
├── compiled/          # Generated rules for each target
│   ├── cursor/       # Cursor IDE rules (.mdc files)
│   ├── claude/       # Claude Code rules (.md files)
│   ├── cline/        # Cline/Roo rules (.md files)
│   └── copilot/      # GitHub Copilot rules (.instructions.md files)
├── airuler.yaml       # Project configuration
├── airuler.lock       # Vendor dependency locks
└── README.md          # This file
` + "```" + `

## Getting Started

### 1. Create Templates

Add your rule templates to the ` + "`templates/`" + ` directory. Templates use Go's text/template syntax with YAML front matter:

` + "```" + `yaml
---
claude_mode: command
description: "Code review guidelines"
globs: "**/*.{js,ts,py,go}"
---

# Code Review Guidelines

When reviewing code, focus on:

{{if eq .Target "claude"}}
Arguments: $ARGUMENTS
{{end}}

- Code clarity and readability
- Performance considerations  
- Security best practices
- Test coverage
` + "```" + `

### 2. Compile Rules

Generate target-specific rules:

` + "```" + `bash
airuler compile              # Compile for all targets
airuler compile claude       # Compile for specific target
` + "```" + `

### 3. Install Rules

Install compiled rules to your AI tools:

` + "```" + `bash
airuler install claude --global    # Install globally
airuler install claude --project . # Install for current project
` + "```" + `

### 4. Manage Vendors

Fetch and manage external rule repositories:

` + "```" + `bash
airuler fetch https://github.com/user/rules-repo  # Add vendor
airuler update                                    # Update vendors
` + "```" + `

## Available Commands

- ` + "`airuler compile`" + ` - Compile templates into target-specific rules
- ` + "`airuler install`" + ` - Install compiled rules to AI tools  
- ` + "`airuler fetch`" + ` - Fetch external rule repositories
- ` + "`airuler update`" + ` - Update vendor repositories
- ` + "`airuler vendors`" + ` - Manage vendor inclusion/exclusion

## Configuration

Edit ` + "`airuler.yaml`" + ` to configure which vendors to include:

` + "```" + `yaml
defaults:
  include_vendors:
    - vendor-name    # Include specific vendor
    - "*"           # Include all vendors
` + "```" + `

For more detailed documentation, visit the [airuler repository](https://github.com/Ratler/airuler/).
`
	if err := os.WriteFile("README.md", []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to write README.md file: %w", err)
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

	if targetPath == "." {
		fmt.Println("✅ airuler project initialized successfully!")
	} else {
		fmt.Printf("✅ airuler project initialized successfully in %s!\n", targetPath)
	}

	fmt.Println("\nCreated:")
	for _, dir := range dirs {
		fmt.Printf("  📁 %s/\n", dir)
	}
	fmt.Println("  📄 airuler.yaml")
	fmt.Println("  📄 airuler.lock")
	fmt.Println("  📄 .gitignore")
	fmt.Println("  📄 README.md")
	fmt.Printf("  📄 %s\n", examplePath)

	fmt.Println("\nNext steps:")
	if targetPath != "." {
		fmt.Printf("  1. cd %s\n", targetPath)
		fmt.Println("  2. Add your templates to templates/")
		fmt.Println("  3. Run 'airuler compile' to generate rules")
		fmt.Println("  4. Run 'airuler install' to install rules")
	} else {
		fmt.Println("  1. Add your templates to templates/")
		fmt.Println("  2. Run 'airuler compile' to generate rules")
		fmt.Println("  3. Run 'airuler install' to install rules")
	}

	// Restore original directory if we changed it
	if originalDir != "" {
		os.Chdir(originalDir)
	}

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

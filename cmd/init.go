// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	airulerconfig "github.com/ratler/airuler/internal/config"
	"github.com/ratler/airuler/internal/git"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize a new airuler project",
	Long: `Initialize a new airuler project with the modern directory structure.

If no path is provided, initializes in the current directory.
If a path is provided, creates the directory and initializes the project there.

Project Structure:
├── templates/         # Your rule templates (.tmpl)
│   ├── components/    # Reusable components (.ptmpl)
│   └── examples/      # Example templates  
├── vendors/           # External rule repositories
├── compiled/          # Generated rules for each target
│   ├── cursor/        # Cursor IDE rules
│   ├── claude/        # Claude Code rules
│   ├── cline/         # Cline rules
│   ├── copilot/       # GitHub Copilot rules
│   └── roo/           # Roo Code rules
├── airuler.yaml       # Project configuration with vendor settings
├── airuler.lock       # Vendor dependency locks
└── README.md          # Project documentation

Examples:
  airuler init                    # Initialize in current directory
  airuler init my-rules-project   # Create and initialize new directory
  airuler init ../other-project   # Initialize in relative path`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
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
				if err := os.Chdir(originalDir); err != nil {
					// Log warning but don't fail the overall operation
					fmt.Printf("Warning: failed to restore original directory %s: %v\n", originalDir, err)
				}
			}
		}()
	}

	// Check if airuler.yaml already exists
	if _, err := os.Stat("airuler.yaml"); err == nil {
		return fmt.Errorf("airuler.yaml already exists. Project appears to be already initialized")
	}

	// Create directory structure
	dirs := []string{
		"templates/components",
		"templates/examples",
		"vendors",
		"compiled/cursor",
		"compiled/claude",
		"compiled/cline",
		"compiled/copilot",
		"compiled/gemini",
		"compiled/roo",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create default config file with modern structure

	// Create a more comprehensive configuration with comments
	modernConfigContent := `# airuler project configuration
defaults:
  # Vendors to include in compilation
  # Use ["*"] to include all vendors, or specify specific vendors by name
  include_vendors: ["*"]

# Vendor metadata - describes this vendor/project
vendor:
  name: "My AI Rules"
  description: "Custom AI coding assistant rules for my project"
  version: "1.0.0"
  author: "Your Name"
  # homepage: "https://github.com/your-username/your-rules"
`

	if err := os.WriteFile("airuler.yaml", []byte(modernConfigContent), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Create empty lock file
	lockFile := &airulerconfig.LockFile{
		Vendors: make(map[string]airulerconfig.VendorLock),
	}
	lockData, err := yaml.Marshal(lockFile)
	if err != nil {
		return fmt.Errorf("failed to marshal lock file: %w", err)
	}

	if err := os.WriteFile("airuler.lock", lockData, 0600); err != nil {
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
	if err := os.WriteFile(".gitignore", []byte(gitignoreContent), 0600); err != nil {
		return fmt.Errorf("failed to write .gitignore file: %w", err)
	}

	// Create README.md file
	readmeContent := `# AI Rules Project

This is an airuler project for managing AI coding assistant rules and templates.

## About airuler

airuler is a CLI tool that compiles AI rule templates into target-specific formats for various AI coding assistants including Cursor, Claude Code, Cline, and GitHub Copilot. It supports template inheritance, vendor management, and multi-repository workflows.

For more information, documentation, and source code, visit: https://github.com/Ratler/airuler/

## Project Structure

` + "```" + `
.
├── templates/        # Your rule templates
│   ├── components/   # Reusable components (.ptmpl)
│   └── examples/     # Example templates
├── vendors/          # External rule repositories (git-ignored)
├── compiled/         # Generated rules for each target
│   ├── cursor/       # Cursor IDE rules (.mdc files)
│   ├── claude/       # Claude Code rules (.md files)
│   ├── cline/        # Cline rules (.md files)
│   ├── copilot/      # GitHub Copilot rules (.instructions.md files)
│   └── roo/          # Roo Code rules (.md files)
├── airuler.yaml      # Project configuration
├── airuler.lock      # Vendor dependency locks
└── README.md         # This file
` + "```" + `

## Getting Started

### 1. Create Templates

Add your rule templates to the ` + "`templates/`" + ` directory. Templates support rich YAML front matter and reusable components:

` + "```" + `yaml
---
claude_mode: both
description: "Modern coding standards"
globs: "**/*.{js,ts,py,go}"
language: "typescript"
framework: "react"
project_type: "web-application"
---
{{template "components/header" .}}

# {{.Language}} {{.Framework}} Standards

{{template "components/guidelines" .}}

{{if eq .Target "claude"}}
When reviewing code, focus on:
- Type safety and interfaces
- Performance considerations  
- Security best practices
{{end}}
` + "```" + `

### 2. Sync Everything

Use the streamlined sync workflow for daily development:

` + "```" + `bash
airuler sync                 # Full workflow: update vendors → compile → deploy
airuler sync claude          # Sync only for Claude target
airuler sync --no-update     # Skip vendor updates
` + "```" + `

### 3. Deploy Fresh Installations

Deploy compiled rules to new locations:

` + "```" + `bash
airuler deploy --global      # Deploy globally for all targets
airuler deploy --project .   # Deploy for current project
airuler deploy --interactive # Interactive template selection
` + "```" + `

### 4. Manage Vendors

Fetch and manage external rule repositories:

` + "```" + `bash
airuler vendors add https://github.com/user/rules-repo  # Add vendor
airuler vendors update                                  # Update vendors
` + "```" + `

## Available Commands

- ` + "`airuler sync`" + ` - Main workflow: update vendors → compile → deploy
- ` + "`airuler deploy`" + ` - Compile templates and install to new locations
- ` + "`airuler manage`" + ` - Interactive management hub for all operations
- ` + "`airuler vendors`" + ` - Manage vendor repositories (add, update, list, config)
- ` + "`airuler init`" + ` - Initialize new airuler projects
- ` + "`airuler watch`" + ` - Watch templates and auto-compile on changes

## Configuration

Edit ` + "`airuler.yaml`" + ` to customize your rules project:

` + "```" + `yaml
# Project metadata
vendor:
  name: "My AI Rules"
  description: "Custom AI rules for my project"
  version: "1.0.0"
  author: "Your Name"

# Template defaults available to all templates
template_defaults:
  language: "typescript"
  framework: "react"
  project_type: "web-application"

# Target-specific configuration
targets:
  claude:
    default_mode: "both"  # memory, command, or both
  cursor:
    always_apply: true

# Global variables for templates
variables:
  company_name: "Your Company"

# Include external vendors
defaults:
  include_vendors: ["*"]  # Include all vendors

# Override external vendor settings (optional)
vendor_overrides:
  vendor-name:
    template_defaults:
      project_type: "mobile-app"
    targets:
      claude:
        default_mode: "command"
` + "```" + `

For more detailed documentation, visit the [airuler repository](https://github.com/Ratler/airuler/).
`
	if err := os.WriteFile("README.md", []byte(readmeContent), 0600); err != nil {
		return fmt.Errorf("failed to write README.md file: %w", err)
	}

	// Create modern example template
	exampleTemplate := `---
claude_mode: both
description: "Modern coding standards with reusable components"
globs: "**/*.{js,ts,jsx,tsx,py,go}"
language: "typescript"
framework: "react"
project_type: "web-application"
tags: ["frontend", "backend", "standards"]
custom:
  min_version: "18.0.0"
  build_tool: "vite"
---
{{template "components/header" .}}

# {{.Language}} {{.Framework}} Coding Standards

This template demonstrates modern airuler features including:
- Vendor configuration defaults
- Reusable components with .ptmpl files
- Rich YAML front matter
- Target-specific compilation

{{template "components/guidelines" .}}

{{if eq .Target "claude"}}
## Code Review Checklist
When reviewing {{.Language}} code:
1. ✅ Check type safety and interfaces
2. ✅ Verify error handling patterns
3. ✅ Ensure performance considerations
4. ✅ Validate security practices
{{end}}

{{template "components/footer" .}}`

	examplePath := filepath.Join("templates", "examples", "modern-example.tmpl")
	if err := os.WriteFile(examplePath, []byte(exampleTemplate), 0600); err != nil {
		return fmt.Errorf("failed to write example template: %w", err)
	}

	// Create example component templates (.ptmpl)
	headerComponent := `---
description: "Standard header component"
---
## {{.Name}} - {{.Target}} Target

**Project**: {{.ProjectType}} | **Language**: {{.Language}} | **Framework**: {{.Framework}}
{{if .Custom.build_tool}}**Build Tool**: {{.Custom.build_tool}}{{end}}

Generated for {{.Target}} on {{/* Date would go here */}}

---`

	headerPath := filepath.Join("templates", "components", "header.ptmpl")
	if err := os.WriteFile(headerPath, []byte(headerComponent), 0600); err != nil {
		return fmt.Errorf("failed to write header component: %w", err)
	}

	guidelinesComponent := `---
description: "Reusable coding guidelines component"
---
## Core Guidelines

### Code Quality
- Write clean, readable code
- Use meaningful variable and function names
- Follow consistent formatting and style
- Implement proper error handling

### {{.Language}} Specific
{{if eq .Language "typescript"}}
- Use strict TypeScript configuration
- Define interfaces for all object shapes
- Avoid \"any\" type - use proper typing
- Implement proper error boundaries
{{else if eq .Language "python"}}
- Follow PEP 8 style guidelines
- Use type hints for function signatures
- Write docstrings for all functions
- Use virtual environments
{{else}}
- Follow language-specific best practices
- Use established conventions and patterns
{{end}}

### Testing
- Write unit tests for all business logic
- Aim for high test coverage (>80%)
- Include integration tests for critical paths
- Test edge cases and error conditions

{{if contains .Tags "frontend"}}
### Frontend Specific
- Ensure accessibility (WCAG compliance)
- Optimize for performance (Core Web Vitals)
- Implement responsive design
- Handle loading and error states
{{end}}`

	guidelinesPath := filepath.Join("templates", "components", "guidelines.ptmpl")
	if err := os.WriteFile(guidelinesPath, []byte(guidelinesComponent), 0600); err != nil {
		return fmt.Errorf("failed to write guidelines component: %w", err)
	}

	footerComponent := `---
description: "Standard footer component"
---

---

## Additional Resources

{{if .Custom.style_guide_url}}
- [Style Guide]({{.Custom.style_guide_url}})
{{end}}
{{if .Documentation}}
- [Documentation]({{.Documentation}})
{{end}}
{{if .Custom.support_email}}
- Support: {{.Custom.support_email}}
{{end}}

*This rule was generated by airuler for {{.Target}}*`

	footerPath := filepath.Join("templates", "components", "footer.ptmpl")
	if err := os.WriteFile(footerPath, []byte(footerComponent), 0600); err != nil {
		return fmt.Errorf("failed to write footer component: %w", err)
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
	fmt.Printf("  📄 %s\n", headerPath)
	fmt.Printf("  📄 %s\n", guidelinesPath)
	fmt.Printf("  📄 %s\n", footerPath)

	fmt.Println("\nNext steps:")
	if targetPath != "." {
		fmt.Printf("  1. cd %s\n", targetPath)
		fmt.Println("  2. Add your templates to templates/")
		fmt.Println("  3. Run 'airuler sync' for the complete workflow")
		fmt.Println("  4. Use 'airuler deploy --interactive' for guided setup")
	} else {
		fmt.Println("  1. Add your templates to templates/")
		fmt.Println("  2. Run 'airuler sync' for the complete workflow")
		fmt.Println("  3. Use 'airuler deploy --interactive' for guided setup")
	}

	// Restore original directory if we changed it
	if originalDir != "" {
		if err := os.Chdir(originalDir); err != nil {
			// Log warning but don't fail the overall operation
			fmt.Printf("Warning: failed to restore original directory %s: %v\n", originalDir, err)
		}
	}

	return nil
}

// askYesNo prompts the user for a yes/no question and returns true for yes
// If stdin is not available for interactive input (common in automated environments),
// it defaults to 'no' to avoid hanging the program
func askYesNo(prompt string) bool {
	// Check if we have a proper terminal for interactive input
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		fmt.Printf("Warning: Cannot check stdin availability: %v. Defaulting to 'no'.\n", err)
		return false
	}

	// If stdin is not a character device (e.g., piped input), handle it differently
	if (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		// We have piped input, try to read it
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(prompt + " ")
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Warning: Could not read piped input: %v. Defaulting to 'no'.\n", err)
			return false
		}
		response = strings.TrimSpace(strings.ToLower(response))
		return response == "y" || response == "yes"
	}

	// We have a terminal, try interactive input
	fmt.Print(prompt + " ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		// Handle EOF or other stdin errors gracefully
		fmt.Printf("\nWarning: No input available (stdin closed). Defaulting to 'no'.\n")
		fmt.Println("Hint: Run 'airuler init' in a proper terminal for interactive prompts.")
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// askString prompts the user for a string input with an optional default value
func askString(prompt, defaultValue string) string {
	// Check if we have a proper terminal for interactive input
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		fmt.Printf("Warning: Cannot check stdin availability: %v. Using default value.\n", err)
		return defaultValue
	}

	// If stdin is not a character device (e.g., piped input), handle it differently
	if (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		// We have piped input, try to read it
		reader := bufio.NewReader(os.Stdin)
		if defaultValue != "" {
			fmt.Printf("%s [%s]: ", prompt, defaultValue)
		} else {
			fmt.Printf("%s: ", prompt)
		}
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Warning: Could not read piped input: %v. Using default value.\n", err)
			return defaultValue
		}
		response = strings.TrimSpace(response)
		if response == "" {
			return defaultValue
		}
		return response
	}

	// We have a terminal, try interactive input
	for {
		if defaultValue != "" {
			fmt.Printf("%s [%s]: ", prompt, defaultValue)
		} else {
			fmt.Printf("%s: ", prompt)
		}

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			// Handle EOF or other stdin errors gracefully
			fmt.Printf("\nWarning: No input available (stdin closed). Using default value.\n")
			fmt.Println("Hint: Run 'airuler init' in a proper terminal for interactive prompts.")
			return defaultValue
		}

		response = strings.TrimSpace(response)
		if response == "" {
			if defaultValue != "" {
				return defaultValue
			}
			fmt.Println("This field is required. Please enter a value.")
			continue
		}
		return response
	}
}

// promptForUserInfo prompts the user for missing git user name and/or email
func promptForUserInfo(existingUser *git.User) (*git.User, error) {
	// Start with existing values or empty strings
	var name, email string
	if existingUser != nil {
		name = existingUser.Name
		email = existingUser.Email
	}

	// Check what's missing or invalid
	needsName := !git.IsValidName(name)
	needsEmail := !git.IsValidEmail(email)

	// Only show prompt if we actually need to ask for something
	if needsName || needsEmail {
		fmt.Println("\n🔧 Git user configuration needed for commits:")

		// Show what we found and what's missing
		if existingUser != nil && existingUser.Name != "" && !needsName {
			fmt.Printf("✓ Using existing name: %s\n", existingUser.Name)
		}
		if existingUser != nil && existingUser.Email != "" && !needsEmail {
			fmt.Printf("✓ Using existing email: %s\n", existingUser.Email)
		}
	}

	// Only prompt for name if it's missing or invalid
	if needsName {
		name = askString("Git user name", name)
		if !git.IsValidName(name) {
			return nil, fmt.Errorf("invalid name: must be at least 2 characters long")
		}
	}

	// Only prompt for email if it's missing or invalid
	if needsEmail {
		email = askString("Git user email", email)
		if !git.IsValidEmail(email) {
			return nil, fmt.Errorf("invalid email format")
		}
	}

	return &git.User{
		Name:  name,
		Email: email,
	}, nil
}

// initializeGitRepo initializes a git repository and creates an initial commit using go-git
func initializeGitRepo() error {
	// Check if already in a git repository
	if _, err := os.Stat(".git"); err == nil {
		return fmt.Errorf("directory is already a git repository")
	}

	// Try to get user information from global git config
	var user *git.User
	var err error

	// Skip user prompting in test mode
	if os.Getenv("AIRULER_TEST_MODE") != "" {
		user = &git.User{
			Name:  "Test User",
			Email: "test@example.com",
		}
	} else {
		// Try to read from global git config first
		globalUser, err := git.GetGlobalGitUser()
		if err != nil {
			fmt.Printf("ℹ️  Could not read git user from ~/.gitconfig: %v\n", err)
		}

		// Check if we have complete user info
		needsUserInfo := globalUser == nil ||
			globalUser.Name == "" ||
			globalUser.Email == "" ||
			!git.IsValidName(globalUser.Name) ||
			!git.IsValidEmail(globalUser.Email)

		if needsUserInfo {
			// Prompt for missing or invalid user information
			user, err = promptForUserInfo(globalUser)
			if err != nil {
				return fmt.Errorf("failed to get user information: %w", err)
			}
		} else {
			user = globalUser
			fmt.Printf("✅ Using git user: %s <%s>\n", user.Name, user.Email)
		}
	}

	// Initialize git repository with go-git
	repo, err := gogit.PlainInit(".", false)
	if err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Set default branch to "main"
	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main"))
	if err := repo.Storer.SetReference(headRef); err != nil {
		return fmt.Errorf("failed to set default branch to main: %w", err)
	}

	// Get repository config and set user information locally
	cfg, err := repo.Config()
	if err != nil {
		return fmt.Errorf("failed to get repository config: %w", err)
	}

	// Set default branch and user info in local repository config
	cfg.Init.DefaultBranch = "main"
	cfg.User.Name = user.Name
	cfg.User.Email = user.Email

	if err := repo.SetConfig(cfg); err != nil {
		return fmt.Errorf("failed to set repository config: %w", err)
	}

	// Get working tree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add all files
	if err := worktree.AddWithOptions(&gogit.AddOptions{All: true}); err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}

	// Create initial commit with proper user information
	commit, err := worktree.Commit("Initial airuler project setup", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  user.Name,
			Email: user.Email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	// Update main branch to point to the new commit
	mainRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName("main"), commit)
	if err := repo.Storer.SetReference(mainRef); err != nil {
		return fmt.Errorf("failed to update main branch reference: %w", err)
	}

	fmt.Printf(
		"📦 Git repository initialized with initial commit on main branch (author: %s <%s>)\n",
		user.Name,
		user.Email,
	)
	return nil
}

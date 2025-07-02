// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

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

# Vendor-specific overrides (optional)
# Use this to customize vendor settings without modifying vendor repositories
# vendor_overrides:
#   vendor-name:
#     template_defaults:
#       project_type: "custom-type"
#       language: "custom-language"
#     targets:
#       claude:
#         default_mode: "command"  # Override vendor's default mode
`

	if err := os.WriteFile("airuler.yaml", []byte(modernConfigContent), 0600); err != nil {
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
├── templates/          # Your rule templates
│   ├── components/    # Reusable components (.ptmpl)
│   └── examples/      # Example templates
├── vendors/           # External rule repositories (git-ignored)
├── compiled/          # Generated rules for each target
│   ├── cursor/       # Cursor IDE rules (.mdc files)
│   ├── claude/       # Claude Code rules (.md files)
│   ├── cline/        # Cline rules (.md files)
│   ├── copilot/      # GitHub Copilot rules (.instructions.md files)
│   └── roo/          # Roo Code rules (.md files)
├── airuler.yaml       # Project configuration
├── airuler.lock       # Vendor dependency locks
└── README.md          # This file
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
- ` + "`airuler vendors list`" + ` - List available vendors
- ` + "`airuler vendors config`" + ` - View vendor configurations

## Configuration

Edit ` + "`airuler.yaml`" + ` to configure vendors and overrides:

` + "```" + `yaml
defaults:
  include_vendors: ["*"]  # Include all vendors

# Override vendor settings (optional)
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
		fmt.Println("  3. Run 'airuler compile' to generate rules")
		fmt.Println("  4. Run 'airuler install' to install rules")
	} else {
		fmt.Println("  1. Add your templates to templates/")
		fmt.Println("  2. Run 'airuler compile' to generate rules")
		fmt.Println("  3. Run 'airuler install' to install rules")
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

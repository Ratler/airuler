package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ratler/airuler/internal/compiler"
	"github.com/ratler/airuler/internal/config"
	"github.com/ratler/airuler/internal/template"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	targetFlag  string
	vendorFlag  string
	vendorsFlag string
	ruleFlag    string
)

type TemplateFrontMatter struct {
	ClaudeMode  string `yaml:"claude_mode"`
	Description string `yaml:"description"`
	Globs       string `yaml:"globs"`
}

var compileCmd = &cobra.Command{
	Use:   "compile [target]",
	Short: "Compile templates into target-specific rules",
	Long: `Compile templates into target-specific rules for AI coding assistants.

Available targets: cursor, claude, cline, copilot

Examples:
  airuler compile                    # Compile for all targets
  airuler compile cursor             # Compile only for Cursor
  airuler compile --vendor frontend  # Compile from specific vendor
  airuler compile --rule my-rule     # Compile specific rule`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var targets []compiler.Target

		if len(args) > 0 {
			target := compiler.Target(args[0])
			if !isValidTarget(target) {
				return fmt.Errorf("invalid target: %s. Valid targets: %s",
					target, strings.Join(getTargetNames(), ", "))
			}
			targets = []compiler.Target{target}
		} else {
			targets = compiler.AllTargets
		}

		return compileTemplates(targets)
	},
}

func init() {
	rootCmd.AddCommand(compileCmd)

	compileCmd.Flags().StringVar(&vendorFlag, "vendor", "", "compile from specific vendor")
	compileCmd.Flags().StringVar(&vendorsFlag, "vendors", "", "compile from specific vendors (comma-separated)")
	compileCmd.Flags().StringVar(&ruleFlag, "rule", "", "compile specific rule")
}

func compileTemplates(targets []compiler.Target) error {

	// Load templates
	templateDirs := []string{"templates"}

	// Add vendor directories based on flags or configuration
	if vendorFlag != "" {
		vendorDir := filepath.Join("vendors", vendorFlag, "templates")
		if _, err := os.Stat(vendorDir); err == nil {
			templateDirs = append(templateDirs, vendorDir)
		} else {
			return fmt.Errorf("vendor directory not found: %s", vendorDir)
		}
	} else if vendorsFlag != "" {
		vendors := strings.Split(vendorsFlag, ",")
		for _, vendor := range vendors {
			vendor = strings.TrimSpace(vendor)
			vendorDir := filepath.Join("vendors", vendor, "templates")
			if _, err := os.Stat(vendorDir); err == nil {
				templateDirs = append(templateDirs, vendorDir)
			} else {
				fmt.Printf("Warning: vendor directory not found: %s\n", vendorDir)
			}
		}
	} else {
		// Auto-include vendors from configuration and lock file
		vendorDirs := getVendorTemplateDirs()
		templateDirs = append(templateDirs, vendorDirs...)
	}

	// Load templates and partials from all directories
	templates, partials, err := loadTemplatesFromDirs(templateDirs)
	if err != nil {
		return err
	}

	if len(templates) == 0 {
		return fmt.Errorf("no templates found in %s", strings.Join(templateDirs, ", "))
	}

	// Filter templates by rule if specified
	if ruleFlag != "" {
		filtered := make(map[string]string)
		for name, content := range templates {
			if strings.Contains(name, ruleFlag) {
				filtered[name] = content
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("no templates found matching rule: %s", ruleFlag)
		}
		templates = filtered
	}

	// Templates will be loaded individually during compilation with front matter stripped

	// Compile for each target
	compiled := 0
	for _, target := range targets {
		fmt.Printf("Compiling for %s...\n", target)

		targetDir := filepath.Join("compiled", string(target))
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create target directory %s: %w", targetDir, err)
		}

		// Create a fresh compiler for each target to avoid template conflicts
		targetComp := compiler.NewCompiler()

		// First, load all partials into the compiler so they're available for inclusion
		for partialName, partialContent := range partials {
			// Strip front matter from partial content before loading
			cleanPartialContent := stripTemplateFrontMatter(partialContent)
			if err := targetComp.LoadTemplate(partialName, cleanPartialContent); err != nil {
				fmt.Printf("Warning: failed to load partial %s: %v\n", partialName, err)
			}
		}

		// Collect memory mode content to handle appending to CLAUDE.md
		memoryModeContent := []string{}

		// Now compile main templates (partials are available for inclusion)
		for templateName, templateContent := range templates {
			// Parse front matter to get template metadata
			frontMatter, err := parseTemplateFrontMatter(templateContent)
			if err != nil {
				fmt.Printf("Warning: failed to parse front matter for %s: %v\n", templateName, err)
			}

			// Strip front matter from template content before loading
			cleanTemplateContent := stripTemplateFrontMatter(templateContent)

			data := template.TemplateData{
				Name:        templateName,
				Description: getValueOrDefault(frontMatter.Description, fmt.Sprintf("AI coding rules for %s", templateName)),
				Globs:       getValueOrDefault(frontMatter.Globs, "**/*"),
				Mode:        frontMatter.ClaudeMode,
			}

			// Load the clean template content (without front matter)
			if err := targetComp.LoadTemplate(templateName, cleanTemplateContent); err != nil {
				fmt.Printf("Warning: failed to load template %s: %v\n", templateName, err)
				continue
			}

			rules, err := targetComp.CompileTemplateWithModes(templateName, target, data)
			if err != nil {
				fmt.Printf("Warning: failed to compile %s for %s: %v\n", templateName, target, err)
				continue
			}

			for _, rule := range rules {
				// Special handling for Claude memory mode
				if target == compiler.TargetClaude && rule.Mode == "memory" {
					memoryModeContent = append(memoryModeContent, rule.Content)
					compiled++
					fmt.Printf("  ✅ %s (memory) -> CLAUDE.md (queued)\n", templateName)
				} else {
					// Regular file writing for non-memory mode
					outputPath := targetComp.GetOutputPath(target, rule.Filename)
					if err := os.WriteFile(outputPath, []byte(rule.Content), 0644); err != nil {
						return fmt.Errorf("failed to write %s: %w", outputPath, err)
					}

					compiled++
					modeDesc := ""
					if rule.Mode != "" && rule.Mode != "command" {
						modeDesc = fmt.Sprintf(" (%s)", rule.Mode)
					}
					fmt.Printf("  ✅ %s%s -> %s\n", templateName, modeDesc, outputPath)
				}
			}
		}

		// Write all collected memory mode content to CLAUDE.md
		if target == compiler.TargetClaude && len(memoryModeContent) > 0 {
			claudeMdPath := targetComp.GetOutputPath(target, "CLAUDE.md")
			// Use clear section separators that Claude will understand
			separator := "\n\n<!-- ==================== NEXT RULE SECTION ==================== -->\n\n"
			combinedContent := strings.Join(memoryModeContent, separator)
			if err := os.WriteFile(claudeMdPath, []byte(combinedContent), 0644); err != nil {
				return fmt.Errorf("failed to write CLAUDE.md: %w", err)
			}
			fmt.Printf("  ✅ Combined %d memory templates -> %s\n", len(memoryModeContent), claudeMdPath)
		}
	}

	fmt.Printf("\n🎉 Successfully compiled %d rules for %d targets\n", len(templates), len(targets))
	return nil
}

func loadTemplatesFromDirs(dirs []string) (map[string]string, map[string]string, error) {
	templates := make(map[string]string) // Main templates to compile individually
	partials := make(map[string]string)  // Partials to load for inclusion only

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if filepath.Ext(path) != ".tmpl" {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// Use relative path from templates dir as template name
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}

			name := strings.TrimSuffix(relPath, ".tmpl")

			// Check if this is a partial (in partials/ directory)
			pathParts := strings.Split(filepath.ToSlash(relPath), "/")
			isPartial := false
			for _, part := range pathParts {
				if part == "partials" {
					isPartial = true
					break
				}
			}

			if isPartial {
				partials[name] = string(content)
			} else {
				templates[name] = string(content)
			}

			return nil
		})

		if err != nil {
			return nil, nil, fmt.Errorf("failed to walk directory %s: %w", dir, err)
		}
	}

	return templates, partials, nil
}

func isValidTarget(target compiler.Target) bool {
	for _, t := range compiler.AllTargets {
		if t == target {
			return true
		}
	}
	return false
}

func getTargetNames() []string {
	var names []string
	for _, target := range compiler.AllTargets {
		names = append(names, string(target))
	}
	return names
}

func parseTemplateFrontMatter(content string) (*TemplateFrontMatter, error) {
	frontMatter := &TemplateFrontMatter{}

	// Check if content starts with YAML front matter
	if !strings.HasPrefix(content, "---") {
		return frontMatter, nil // No front matter, return empty struct
	}

	// Split content by "---" to extract front matter
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return frontMatter, nil // Invalid front matter format
	}

	yamlContent := strings.TrimSpace(parts[1])
	if yamlContent == "" {
		return frontMatter, nil // Empty front matter
	}

	err := yaml.Unmarshal([]byte(yamlContent), frontMatter)
	if err != nil {
		return frontMatter, fmt.Errorf("failed to parse YAML front matter: %w", err)
	}

	return frontMatter, nil
}

func getValueOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func stripTemplateFrontMatter(content string) string {
	// Check if content starts with YAML front matter
	if !strings.HasPrefix(content, "---") {
		return content
	}

	// Split content by "---" to extract front matter
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return content // Invalid front matter format, return as-is
	}

	// Return content without front matter, trimming leading whitespace
	return strings.TrimSpace(parts[2])
}

func getVendorTemplateDirs() []string {
	var vendorDirs []string

	// Load lock file to see what vendors are available
	lockFile := &config.LockFile{Vendors: make(map[string]config.VendorLock)}
	if _, err := os.Stat("airuler.lock"); err == nil {
		data, err := os.ReadFile("airuler.lock")
		if err == nil {
			yaml.Unmarshal(data, lockFile)
		}
	}

	// For now, include all vendors from the lock file
	// TODO: In the future, this should respect the configuration's include_vendors setting
	for vendorName := range lockFile.Vendors {
		vendorDir := filepath.Join("vendors", vendorName, "templates")
		if _, err := os.Stat(vendorDir); err == nil {
			vendorDirs = append(vendorDirs, vendorDir)
		}
	}

	return vendorDirs
}

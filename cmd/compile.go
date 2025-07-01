// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ratler/airuler/internal/compiler"
	"github.com/ratler/airuler/internal/config"
	"github.com/ratler/airuler/internal/template"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v3"
)

var (
	vendorFlag  string
	vendorsFlag string
	ruleFlag    string
)

type TemplateFrontMatter struct {
	ClaudeMode  string  `yaml:"claude_mode"`
	Description string  `yaml:"description"`
	Globs       *string `yaml:"globs"` // Use pointer to detect if field was set

	// Extended fields for advanced templates
	ProjectType   string                 `yaml:"project_type"`
	Language      string                 `yaml:"language"`
	Framework     string                 `yaml:"framework"`
	Tags          []string               `yaml:"tags"`
	AlwaysApply   string                 `yaml:"always_apply"`
	Documentation string                 `yaml:"documentation"`
	StyleGuide    string                 `yaml:"style_guide"`
	Examples      string                 `yaml:"examples"`
	Custom        map[string]interface{} `yaml:"custom"`
}

type TemplateSource struct {
	Content    string
	SourceType string // "local" or vendor name
	SourcePath string // full file path
}

var compileCmd = &cobra.Command{
	Use:   "compile [target]",
	Short: "Compile templates into target-specific rules",
	Long: fmt.Sprintf(`Compile templates into target-specific rules for AI coding assistants.

Available targets: %s

Examples:
  airuler compile                    # Compile for all targets
  airuler compile cursor             # Compile only for Cursor
  airuler compile --vendor frontend  # Compile from specific vendor
  airuler compile --rule my-rule     # Compile specific rule`, strings.Join(getTargetNames(), ", ")),
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
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

	compileCmd.Flags().StringVarP(&vendorFlag, "vendor", "v", "", "compile from specific vendor")
	compileCmd.Flags().StringVar(&vendorsFlag, "vendors", "", "compile from specific vendors (comma-separated)")
	compileCmd.Flags().StringVarP(&ruleFlag, "rule", "r", "", "compile specific rule")
}

func compileTemplates(targets []compiler.Target) error {
	// Clean the compiled directory first to ensure a fresh start
	compiledDir := "compiled"
	if _, err := os.Stat(compiledDir); err == nil {
		fmt.Printf("Cleaning compiled directory...\n")
		if err := os.RemoveAll(compiledDir); err != nil {
			return fmt.Errorf("failed to clean compiled directory: %w", err)
		}
	}

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
		filtered := make(map[string]TemplateSource)
		for name, templateSource := range templates {
			if strings.Contains(name, ruleFlag) {
				filtered[name] = templateSource
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
		for templateName, templateSource := range templates {
			// Parse front matter to get template metadata
			frontMatter, err := parseTemplateFrontMatter(templateSource.Content)
			if err != nil {
				fmt.Printf("Warning: failed to parse front matter for %s: %v\n", templateName, err)
			}

			// Strip front matter from template content before loading
			cleanTemplateContent := stripTemplateFrontMatter(templateSource.Content)

			// Ensure Custom map is initialized
			if frontMatter.Custom == nil {
				frontMatter.Custom = make(map[string]interface{})
			}

			data := template.Data{
				Name: templateName,
				Description: getValueOrDefault(
					frontMatter.Description,
					fmt.Sprintf("AI coding rules for %s", templateName),
				),
				Globs: getGlobsValue(frontMatter.Globs),
				Mode:  frontMatter.ClaudeMode,

				// Extended fields from template front matter
				ProjectType:   frontMatter.ProjectType,
				Language:      frontMatter.Language,
				Framework:     frontMatter.Framework,
				Tags:          frontMatter.Tags,
				AlwaysApply:   frontMatter.AlwaysApply,
				Documentation: frontMatter.Documentation,
				StyleGuide:    frontMatter.StyleGuide,
				Examples:      frontMatter.Examples,
				Custom:        frontMatter.Custom,
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
				// Create display name with source information
				displayName := fmt.Sprintf("%s/%s", templateSource.SourceType, templateName)

				// Special handling for Claude memory mode
				if target == compiler.TargetClaude && rule.Mode == "memory" {
					memoryModeContent = append(memoryModeContent, rule.Content)
					compiled++
					fmt.Printf("  ✅ %s (memory) -> CLAUDE.md (queued)\n", displayName)
				} else {
					// Regular file writing for non-memory mode
					outputPath := targetComp.GetOutputPath(target, rule.Filename)
					if err := os.WriteFile(outputPath, []byte(rule.Content), 0600); err != nil {
						return fmt.Errorf("failed to write %s: %w", outputPath, err)
					}

					compiled++
					modeDesc := ""
					if rule.Mode != "" && rule.Mode != "command" {
						modeDesc = fmt.Sprintf(" (%s)", rule.Mode)
					}
					fmt.Printf("  ✅ %s%s -> %s\n", displayName, modeDesc, outputPath)
				}
			}
		}

		// Write all collected memory mode content to CLAUDE.md
		if target == compiler.TargetClaude && len(memoryModeContent) > 0 {
			claudeMdPath := targetComp.GetOutputPath(target, "CLAUDE.md")
			// Use clear section separators that Claude will understand
			separator := "\n\n<!-- ==================== NEXT RULE SECTION ==================== -->\n\n"
			combinedContent := strings.Join(memoryModeContent, separator)
			if err := os.WriteFile(claudeMdPath, []byte(combinedContent), 0600); err != nil {
				return fmt.Errorf("failed to write CLAUDE.md: %w", err)
			}
			fmt.Printf("  ✅ Combined %d memory templates -> %s\n", len(memoryModeContent), claudeMdPath)
		}
	}

	fmt.Printf("\n🎉 Successfully compiled %d rules for %d targets\n", len(templates), len(targets))
	return nil
}

func loadTemplatesFromDirs(dirs []string) (map[string]TemplateSource, map[string]string, error) {
	templates := make(map[string]TemplateSource)   // Main templates to compile individually
	partials := make(map[string]string)            // Partials to load for inclusion only
	conflicts := make(map[string][]TemplateSource) // Track conflicts for reporting

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		// Determine source type from directory
		sourceType := "local"
		if strings.Contains(dir, "vendors/") {
			// Extract vendor name from path like "vendors/vendor-name/templates"
			parts := strings.Split(dir, "/")
			for i, part := range parts {
				if part == "vendors" && i+1 < len(parts) {
					sourceType = parts[i+1]
					break
				}
			}
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
			isPartial := slices.Contains(pathParts, "partials")

			if isPartial {
				partials[name] = string(content)
			} else {
				// Check for conflicts and prioritize local templates
				if existing, exists := templates[name]; exists {
					// Track conflicts for later reporting
					if _, hasConflict := conflicts[name]; !hasConflict {
						conflicts[name] = []TemplateSource{existing}
					}
					conflicts[name] = append(conflicts[name], TemplateSource{
						Content:    string(content),
						SourceType: sourceType,
						SourcePath: path,
					})

					// If existing is local and new is vendor, keep existing local
					if existing.SourceType == "local" && sourceType != "local" {
						return nil // Skip adding the vendor template
					}
					// If new template is local and existing is vendor, replace with local
					// (or same precedence level - later one wins)
				}

				templates[name] = TemplateSource{
					Content:    string(content),
					SourceType: sourceType,
					SourcePath: path,
				}
			}

			return nil
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to walk directory %s: %w", dir, err)
		}
	}

	// Report conflicts in a consolidated manner
	for templateName, conflictingSources := range conflicts {
		if len(conflictingSources) > 1 {
			// Find which template won (the one in templates map)
			finalTemplate := templates[templateName]

			// Separate into used and ignored
			var used TemplateSource
			var ignored []TemplateSource

			for _, source := range conflictingSources {
				if source.SourceType == finalTemplate.SourceType && source.SourcePath == finalTemplate.SourcePath {
					used = source
				} else {
					ignored = append(ignored, source)
				}
			}

			// Display conflict info
			fmt.Printf("⚠️  Template '%s' found in multiple sources\n", templateName)

			// Show what's being used
			icon := "✅"
			if used.SourceType == "local" {
				icon = "🏠"
			}
			fmt.Printf("  %s Using: %s (%s)\n", icon, used.SourceType, used.SourcePath)

			// Show what's being ignored
			for _, ignoredSource := range ignored {
				fmt.Printf("  ❌ Ignoring: %s (%s)\n", ignoredSource.SourceType, ignoredSource.SourcePath)
			}
		}
	}

	return templates, partials, nil
}

func isValidTarget(target compiler.Target) bool {
	return slices.Contains(compiler.AllTargets, target)
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

func getGlobsValue(globs *string) string {
	if globs == nil {
		return "**/*"
	}
	return *globs
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
			if err := yaml.Unmarshal(data, lockFile); err != nil {
				// Log warning but continue - we can still use other template sources
				if viper.GetBool("verbose") {
					fmt.Printf("Warning: failed to parse airuler.lock: %v\n", err)
				}
			}
		}
	}

	// Load configuration to check include_vendors setting
	includeVendors := viper.GetStringSlice("defaults.include_vendors")

	// Debug output for troubleshooting
	if viper.GetBool("verbose") {
		fmt.Printf("Debug: include_vendors config: %v\n", includeVendors)
		fmt.Printf("Debug: available vendors in lock file: %v\n", func() []string {
			var names []string
			for name := range lockFile.Vendors {
				names = append(names, name)
			}
			return names
		}())
	}

	// If no include_vendors config is set, include all vendors (backward compatibility)
	if len(includeVendors) == 0 {
		for vendorName := range lockFile.Vendors {
			vendorDir := filepath.Join("vendors", vendorName, "templates")
			if _, err := os.Stat(vendorDir); err == nil {
				vendorDirs = append(vendorDirs, vendorDir)
			}
		}
		return vendorDirs
	}

	// Check if "*" is in include_vendors (include all)
	includeAll := slices.Contains(includeVendors, "*")

	if includeAll {
		// Include all vendors
		for vendorName := range lockFile.Vendors {
			vendorDir := filepath.Join("vendors", vendorName, "templates")
			if _, err := os.Stat(vendorDir); err == nil {
				vendorDirs = append(vendorDirs, vendorDir)
			}
		}
	} else {
		// Include only specified vendors
		for _, includeVendor := range includeVendors {
			if _, exists := lockFile.Vendors[includeVendor]; exists {
				vendorDir := filepath.Join("vendors", includeVendor, "templates")
				if _, err := os.Stat(vendorDir); err == nil {
					vendorDirs = append(vendorDirs, vendorDir)
				}
			} else {
				fmt.Printf("Warning: configured vendor '%s' not found in lock file\n", includeVendor)
			}
		}
	}

	return vendorDirs
}

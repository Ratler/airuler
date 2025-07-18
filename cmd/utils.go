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
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v3"
)

// TemplateFrontMatter represents the YAML front matter in template files
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

// TemplateSource represents a template with its source information
type TemplateSource struct {
	Content    string
	SourceType string // "local" or vendor name
	SourcePath string // full file path
}

// compileTemplates compiles templates for the given targets
func compileTemplates(targets []compiler.Target) error {
	return compileTemplatesWithOutput(targets, true)
}

// compileTemplatesWithOutput compiles templates with optional output suppression
func compileTemplatesWithOutput(targets []compiler.Target, showOutput bool) error {
	// Clean the compiled directory first to ensure a fresh start
	compiledDir := "compiled"
	if _, err := os.Stat(compiledDir); err == nil {
		if showOutput {
			fmt.Printf("Cleaning compiled directory...\n")
		}
		if err := os.RemoveAll(compiledDir); err != nil {
			return fmt.Errorf("failed to clean compiled directory: %w", err)
		}
	}

	// Get current working directory for vendor config loading
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load project configuration
	var projectConfig *config.Config
	if viper.ConfigFileUsed() != "" {
		projectConfig = &config.Config{
			Defaults: config.DefaultConfig{
				IncludeVendors: viper.GetStringSlice("defaults.include_vendors"),
			},
			VendorOverrides: make(map[string]config.VendorConfig),
		}
		// Load vendor overrides from viper if they exist
		if viper.IsSet("vendor_overrides") {
			overrides := viper.GetStringMap("vendor_overrides")
			for vendorName := range overrides {
				// Convert the interface{} to VendorConfig - simplified for now
				projectConfig.VendorOverrides[vendorName] = config.NewDefaultVendorConfig()
			}
		}
	} else {
		projectConfig = config.NewDefaultConfig()
	}

	// Load vendor configurations
	vendorConfigs, err := config.LoadVendorConfigs(currentDir, projectConfig)
	if err != nil {
		return fmt.Errorf("failed to load vendor configurations: %w", err)
	}

	// Validate vendor configurations
	if validationErrors := vendorConfigs.ValidateVendorConfigs(); len(validationErrors) > 0 && showOutput {
		fmt.Printf("Warning: Vendor configuration validation errors:\n")
		for _, err := range validationErrors {
			fmt.Printf("  - %v\n", err)
		}
	}

	// Load templates
	templateDirs := []string{"templates"}

	// Add vendor directories
	vendorDirs := getVendorTemplateDirs()
	templateDirs = append(templateDirs, vendorDirs...)

	// Load templates and partials from all directories
	templates, partialsBySource, err := loadTemplatesFromDirsWithOutput(templateDirs, showOutput)
	if err != nil {
		return err
	}

	if len(templates) == 0 {
		return fmt.Errorf("no templates found in %s", strings.Join(templateDirs, ", "))
	}

	// Compile for each target
	compiled := 0
	for _, target := range targets {
		if showOutput {
			fmt.Printf("Compiling for %s...\n", target)
		}

		targetDir := filepath.Join("compiled", string(target))
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create target directory %s: %w", targetDir, err)
		}

		// Collect memory mode content to handle appending to CLAUDE.md
		memoryModeContent := []string{}

		// Now compile main templates (load source-specific partials for each template)
		for templateName, templateSource := range templates {
			// Create a fresh compiler for each template to ensure isolation
			templateComp := compiler.NewCompiler()

			// Load only partials from the same source as this template
			if sourcePartials, exists := partialsBySource[templateSource.SourceType]; exists {
				if viper.GetBool("verbose") && len(sourcePartials) > 0 && showOutput {
					fmt.Printf(
						"Loading %d partials for %s template %s...\n",
						len(sourcePartials),
						templateSource.SourceType,
						templateName,
					)
				}
				for partialName, partialContent := range sourcePartials {
					// Strip front matter from partial content before loading
					cleanPartialContent := stripTemplateFrontMatter(partialContent)
					if err := templateComp.LoadTemplate(partialName, cleanPartialContent); err != nil {
						if showOutput {
							fmt.Printf("Warning: failed to load partial %s: %v\n", partialName, err)
						}
					} else if viper.GetBool("verbose") && showOutput {
						fmt.Printf("  ✓ Loaded partial: %s\n", partialName)
					}
				}
			}
			// Parse front matter to get template metadata
			frontMatter, err := parseTemplateFrontMatter(templateSource.Content)
			if err != nil && showOutput {
				fmt.Printf("Warning: failed to parse front matter for %s: %v\n", templateName, err)
			}

			// Strip front matter from template content before loading
			cleanTemplateContent := stripTemplateFrontMatter(templateSource.Content)

			// Ensure Custom map is initialized
			if frontMatter.Custom == nil {
				frontMatter.Custom = make(map[string]interface{})
			}

			// Resolve vendor configuration context for this template
			templateContext := vendorConfigs.ResolveTemplateContext(templateSource.SourceType, string(target))

			// Apply vendor defaults and then override with front matter
			data := createTemplateData(templateName, *frontMatter, templateContext, string(target))

			// Load the clean template content (without front matter)
			if err := templateComp.LoadTemplate(templateName, cleanTemplateContent); err != nil {
				if showOutput {
					fmt.Printf("Warning: failed to load template %s: %v\n", templateName, err)
				}
				continue
			}

			rules, err := templateComp.CompileTemplateWithModes(templateName, target, data)
			if err != nil {
				if showOutput {
					fmt.Printf("Warning: failed to compile %s for %s: %v\n", templateName, target, err)
				}
				continue
			}

			for _, rule := range rules {
				// Create display name with source information
				displayName := fmt.Sprintf("%s/%s", templateSource.SourceType, templateName)

				// Special handling for Claude memory mode
				if target == compiler.TargetClaude && rule.Mode == "memory" {
					memoryModeContent = append(memoryModeContent, rule.Content)
					compiled++
					if showOutput {
						fmt.Printf("  ✅ %s (memory) -> CLAUDE.md (queued)\n", displayName)
					}
				} else {
					// Regular file writing for non-memory mode
					outputPath := templateComp.GetOutputPath(target, rule.Filename)
					if err := os.WriteFile(outputPath, []byte(rule.Content), 0600); err != nil {
						return fmt.Errorf("failed to write %s: %w", outputPath, err)
					}

					compiled++
					modeDesc := ""
					if rule.Mode != "" && rule.Mode != "command" {
						modeDesc = fmt.Sprintf(" (%s)", rule.Mode)
					}
					if showOutput {
						fmt.Printf("  ✅ %s%s -> %s\n", displayName, modeDesc, outputPath)
					}
				}
			}
		}

		// Write all collected memory mode content to CLAUDE.md
		if target == compiler.TargetClaude && len(memoryModeContent) > 0 {
			// Create a compiler instance just for getting the output path
			outputComp := compiler.NewCompiler()
			claudeMdPath := outputComp.GetOutputPath(target, "CLAUDE.md")
			// Use clear section separators that Claude will understand
			separator := "\n\n---\n\n"
			combinedContent := strings.Join(memoryModeContent, separator)
			if err := os.WriteFile(claudeMdPath, []byte(combinedContent), 0600); err != nil {
				return fmt.Errorf("failed to write CLAUDE.md: %w", err)
			}
			if showOutput {
				fmt.Printf("  ✅ Combined %d memory templates -> %s\n", len(memoryModeContent), claudeMdPath)
			}
		}
	}

	if showOutput {
		fmt.Printf("\n🎉 Successfully compiled %d rules for %d targets\n", len(templates), len(targets))
	}

	// Update last template directory after successful compilation
	if err == nil && config.IsTemplateDirectory(currentDir) {
		if err := config.UpdateLastTemplateDir(currentDir); err != nil && viper.GetBool("verbose") && showOutput {
			fmt.Printf("Warning: Failed to update last template directory: %v\n", err)
		}
	}

	return nil
}

// loadTemplatesFromDirs loads templates and partials from multiple directories
func loadTemplatesFromDirs(dirs []string) (map[string]TemplateSource, map[string]map[string]string, error) {
	return loadTemplatesFromDirsWithOutput(dirs, true)
}

// loadTemplatesFromDirsWithOutput loads templates and partials with optional output suppression
func loadTemplatesFromDirsWithOutput(dirs []string, showOutput bool) (map[string]TemplateSource, map[string]map[string]string, error) {
	templates := make(map[string]TemplateSource)           // Main templates to compile individually
	partialsBySource := make(map[string]map[string]string) // Partials organized by source
	conflicts := make(map[string][]TemplateSource)         // Track conflicts for reporting

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

			ext := filepath.Ext(path)
			if ext != ".tmpl" && ext != ".ptmpl" {
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

			// Remove extension to get the template name
			name := strings.TrimSuffix(relPath, ext)

			// Check if this is a partial:
			// 1. Files with .ptmpl extension are always partials
			// 2. Files in partials/ directory are partials (backward compatibility)
			pathParts := strings.Split(filepath.ToSlash(relPath), "/")
			isPartial := ext == ".ptmpl" || slices.Contains(pathParts, "partials")

			if isPartial {
				// Initialize partials map for this source if not exists
				if partialsBySource[sourceType] == nil {
					partialsBySource[sourceType] = make(map[string]string)
				}
				partialsBySource[sourceType][name] = string(content)
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
			if showOutput {
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
	}

	return templates, partialsBySource, nil
}

// isValidTarget checks if a target is valid
func isValidTarget(target compiler.Target) bool {
	return slices.Contains(compiler.AllTargets, target)
}

// parseTemplateFrontMatter parses YAML front matter from template content
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

// stripTemplateFrontMatter removes YAML front matter from template content
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

// getVendorTemplateDirs returns vendor template directories based on configuration
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

// createTemplateData merges vendor configuration with front matter to create template data
func createTemplateData(
	templateName string,
	frontMatter TemplateFrontMatter,
	context config.ResolvedTemplateContext,
	target string,
) template.Data {
	// Start with vendor defaults
	data := template.Data{
		Name:   templateName,
		Target: target,
	}

	// Apply vendor template defaults first
	applyVendorDefaults(&data, context.TemplateDefaults)

	// Apply vendor variables
	if data.Custom == nil {
		data.Custom = make(map[string]interface{})
	}
	for key, value := range context.Variables {
		data.Custom[key] = value
	}

	// Override with front matter (front matter takes precedence)
	data.Description = getValueOrDefault(
		frontMatter.Description,
		getStringFromVendorDefaults(
			context.TemplateDefaults,
			"description",
			fmt.Sprintf("AI coding rules for %s", templateName),
		),
	)
	data.Globs = getGlobsValue(frontMatter.Globs)

	// Determine Claude mode from front matter, vendor config, or default
	data.Mode = frontMatter.ClaudeMode
	if data.Mode == "" && target == "claude" {
		data.Mode = context.TargetConfig.DefaultMode
	}

	// Override with front matter fields (front matter always wins)
	if frontMatter.ProjectType != "" {
		data.ProjectType = frontMatter.ProjectType
	}
	if frontMatter.Language != "" {
		data.Language = frontMatter.Language
	}
	if frontMatter.Framework != "" {
		data.Framework = frontMatter.Framework
	}
	if frontMatter.Tags != nil {
		data.Tags = frontMatter.Tags
	}
	if frontMatter.AlwaysApply != "" {
		data.AlwaysApply = frontMatter.AlwaysApply
	}
	if frontMatter.Documentation != "" {
		data.Documentation = frontMatter.Documentation
	}
	if frontMatter.StyleGuide != "" {
		data.StyleGuide = frontMatter.StyleGuide
	}
	if frontMatter.Examples != "" {
		data.Examples = frontMatter.Examples
	}

	// Merge custom fields (front matter overrides vendor)
	for key, value := range frontMatter.Custom {
		data.Custom[key] = value
	}

	return data
}

// Helper functions
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

// applyVendorDefaults applies vendor default values to template data
func applyVendorDefaults(data *template.Data, defaults map[string]interface{}) {
	if projectType, ok := defaults["project_type"].(string); ok && data.ProjectType == "" {
		data.ProjectType = projectType
	}
	if language, ok := defaults["language"].(string); ok && data.Language == "" {
		data.Language = language
	}
	if framework, ok := defaults["framework"].(string); ok && data.Framework == "" {
		data.Framework = framework
	}
	if tags, ok := defaults["tags"].([]interface{}); ok && data.Tags == nil {
		var stringTags []string
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				stringTags = append(stringTags, tagStr)
			}
		}
		data.Tags = stringTags
	}
	if alwaysApply, ok := defaults["always_apply"].(string); ok && data.AlwaysApply == "" {
		data.AlwaysApply = alwaysApply
	}
	if documentation, ok := defaults["documentation"].(string); ok && data.Documentation == "" {
		data.Documentation = documentation
	}
	if styleGuide, ok := defaults["style_guide"].(string); ok && data.StyleGuide == "" {
		data.StyleGuide = styleGuide
	}
	if examples, ok := defaults["examples"].(string); ok && data.Examples == "" {
		data.Examples = examples
	}
	if custom, ok := defaults["custom"].(map[string]interface{}); ok {
		if data.Custom == nil {
			data.Custom = make(map[string]interface{})
		}
		for key, value := range custom {
			if _, exists := data.Custom[key]; !exists {
				data.Custom[key] = value
			}
		}
	}
}

// getStringFromVendorDefaults safely gets a string value from vendor defaults
func getStringFromVendorDefaults(defaults map[string]interface{}, key, fallback string) string {
	if value, ok := defaults[key].(string); ok && value != "" {
		return value
	}
	return fallback
}

// getAllTargets returns all available targets
func getAllTargets() []compiler.Target {
	return compiler.AllTargets
}

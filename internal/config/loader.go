// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadVendorConfigs loads and merges vendor configurations according to the hierarchy:
// CLI flags > Project config > Vendor configs > Global config
func LoadVendorConfigs(templateDir string, projectConfig *Config) (*MergedVendorConfigs, error) {
	vendorConfigs := make(map[string]VendorConfig)

	// Get vendor directories
	vendorsDir := filepath.Join(templateDir, "vendors")
	if _, err := os.Stat(vendorsDir); os.IsNotExist(err) {
		// No vendors directory, return empty config
		return &MergedVendorConfigs{
			VendorConfigs: vendorConfigs,
			ProjectConfig: projectConfig,
		}, nil
	}

	// Read vendor directories
	vendorDirs, err := os.ReadDir(vendorsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read vendors directory: %w", err)
	}

	// Load configuration from each vendor
	for _, vendorDir := range vendorDirs {
		if !vendorDir.IsDir() {
			continue
		}

		vendorName := vendorDir.Name()
		vendorConfigPath := filepath.Join(vendorsDir, vendorName, "airuler.yaml")

		// Check if vendor config exists
		if _, err := os.Stat(vendorConfigPath); os.IsNotExist(err) {
			// No config file, use empty config
			vendorConfigs[vendorName] = NewDefaultVendorConfig()
			continue
		}

		// Load vendor configuration
		vendorConfig, err := LoadVendorConfig(vendorConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load vendor config for %s: %w", vendorName, err)
		}

		vendorConfigs[vendorName] = vendorConfig
	}

	// Apply project-level overrides
	mergedConfigs := applyProjectOverrides(vendorConfigs, projectConfig)

	return &MergedVendorConfigs{
		VendorConfigs: mergedConfigs,
		ProjectConfig: projectConfig,
	}, nil
}

// LoadVendorConfig loads a single vendor configuration file
func LoadVendorConfig(configPath string) (VendorConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return VendorConfig{}, fmt.Errorf("failed to read vendor config file: %w", err)
	}

	var vendorConfig VendorConfig
	if err := yaml.Unmarshal(data, &vendorConfig); err != nil {
		return VendorConfig{}, fmt.Errorf("failed to parse vendor config YAML: %w", err)
	}

	// Initialize nil maps
	if vendorConfig.TemplateDefaults == nil {
		vendorConfig.TemplateDefaults = make(map[string]interface{})
	}
	if vendorConfig.Targets == nil {
		vendorConfig.Targets = make(map[string]TargetConfig)
	}
	if vendorConfig.Variables == nil {
		vendorConfig.Variables = make(map[string]interface{})
	}

	return vendorConfig, nil
}

// applyProjectOverrides applies project-level vendor overrides to vendor configurations
func applyProjectOverrides(vendorConfigs map[string]VendorConfig, projectConfig *Config) map[string]VendorConfig {
	if projectConfig == nil || projectConfig.VendorOverrides == nil {
		return vendorConfigs
	}

	mergedConfigs := make(map[string]VendorConfig)

	// Copy all vendor configs first
	for vendorName, vendorConfig := range vendorConfigs {
		mergedConfigs[vendorName] = vendorConfig
	}

	// Apply overrides
	for vendorName, override := range projectConfig.VendorOverrides {
		baseConfig, exists := mergedConfigs[vendorName]
		if !exists {
			// Create new config if vendor doesn't exist yet
			baseConfig = NewDefaultVendorConfig()
		}

		mergedConfig := mergeVendorConfig(baseConfig, override)
		mergedConfigs[vendorName] = mergedConfig
	}

	return mergedConfigs
}

// mergeVendorConfig merges an override config into a base vendor config
func mergeVendorConfig(base, override VendorConfig) VendorConfig {
	merged := VendorConfig{
		Vendor:           base.Vendor, // Vendor manifest cannot be overridden
		TemplateDefaults: mergeStringInterfaceMap(base.TemplateDefaults, override.TemplateDefaults),
		Variables:        mergeStringInterfaceMap(base.Variables, override.Variables),
		Targets:          mergeTargetConfigs(base.Targets, override.Targets),
		Compilation:      mergeCompilationConfig(base.Compilation, override.Compilation),
	}

	return merged
}

// mergeStringInterfaceMap merges two string->interface{} maps, with override taking precedence
func mergeStringInterfaceMap(base, override map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})

	// Copy base
	for k, v := range base {
		merged[k] = v
	}

	// Apply overrides
	for k, v := range override {
		merged[k] = v
	}

	return merged
}

// mergeTargetConfigs merges target configurations
func mergeTargetConfigs(base, override map[string]TargetConfig) map[string]TargetConfig {
	merged := make(map[string]TargetConfig)

	// Copy base
	for target, config := range base {
		merged[target] = config
	}

	// Apply overrides
	for target, overrideConfig := range override {
		baseConfig, exists := merged[target]
		if !exists {
			merged[target] = overrideConfig
		} else {
			merged[target] = mergeTargetConfig(baseConfig, overrideConfig)
		}
	}

	return merged
}

// mergeTargetConfig merges a single target configuration
func mergeTargetConfig(base, override TargetConfig) TargetConfig {
	merged := base

	// Override non-zero values
	if override.DefaultMode != "" {
		merged.DefaultMode = override.DefaultMode
	}

	return merged
}

// mergeCompilationConfig merges compilation configurations
func mergeCompilationConfig(base, _ CompilationConfig) CompilationConfig {
	// Currently no fields to merge, but keeping function for future extensibility
	return base
}

// ResolveTemplateContext resolves the configuration context for a specific template
func (m *MergedVendorConfigs) ResolveTemplateContext(sourceType, target string) ResolvedTemplateContext {
	var vendorConfig VendorConfig

	if sourceType == "local" {
		// Use empty config for local templates
		vendorConfig = NewDefaultVendorConfig()
	} else {
		// Use vendor config if available
		if config, exists := m.VendorConfigs[sourceType]; exists {
			vendorConfig = config
		} else {
			vendorConfig = NewDefaultVendorConfig()
		}
	}

	// Get target-specific config
	var targetConfig TargetConfig
	if config, exists := vendorConfig.Targets[target]; exists {
		targetConfig = config
	}

	return ResolvedTemplateContext{
		SourceType:        sourceType,
		TemplateDefaults:  vendorConfig.TemplateDefaults,
		Variables:         vendorConfig.Variables,
		TargetConfig:      targetConfig,
		CompilationConfig: vendorConfig.Compilation,
	}
}

// GetVendorManifest returns the vendor manifest for a given vendor
func (m *MergedVendorConfigs) GetVendorManifest(vendorName string) (VendorManifest, bool) {
	if config, exists := m.VendorConfigs[vendorName]; exists {
		return config.Vendor, true
	}
	return VendorManifest{}, false
}

// ValidateVendorConfigs validates all vendor configurations for common issues
func (m *MergedVendorConfigs) ValidateVendorConfigs() []error {
	var errors []error

	for vendorName, config := range m.VendorConfigs {
		// Validate vendor manifest
		if config.Vendor.Name == "" && len(config.TemplateDefaults) > 0 {
			errors = append(errors, fmt.Errorf("vendor %s has configuration but no name in manifest", vendorName))
		}

		// Validate target configurations
		for target, targetConfig := range config.Targets {
			if targetConfig.DefaultMode != "" {
				validModes := []string{"memory", "command", "both"}
				isValid := false
				for _, mode := range validModes {
					if targetConfig.DefaultMode == mode {
						isValid = true
						break
					}
				}
				if !isValid {
					errors = append(errors, fmt.Errorf("vendor %s has invalid default_mode '%s' for target %s", vendorName, targetConfig.DefaultMode, target))
				}
			}
		}

		// Validate template defaults don't contain reserved keys
		reservedKeys := []string{"Target", "Name"}
		for _, key := range reservedKeys {
			if _, exists := config.TemplateDefaults[key]; exists {
				errors = append(errors, fmt.Errorf("vendor %s uses reserved template variable name '%s'", vendorName, key))
			}
		}
	}

	return errors
}

// GetIncludedVendors returns the list of vendors that should be included based on project configuration
func GetIncludedVendors(projectConfig *Config, availableVendors []string) []string {
	if projectConfig == nil || len(projectConfig.Defaults.IncludeVendors) == 0 {
		// Default behavior: include all vendors
		return availableVendors
	}

	includeVendors := projectConfig.Defaults.IncludeVendors

	// Handle wildcard
	for _, vendor := range includeVendors {
		if vendor == "*" {
			return availableVendors
		}
	}

	// Filter available vendors by include list
	var included []string
	for _, vendor := range includeVendors {
		for _, available := range availableVendors {
			if vendor == available {
				included = append(included, vendor)
				break
			}
		}
	}

	return included
}

// GetAvailableVendors returns a list of vendor names found in the vendors directory
func GetAvailableVendors(templateDir string) ([]string, error) {
	vendorsDir := filepath.Join(templateDir, "vendors")
	if _, err := os.Stat(vendorsDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	vendorDirs, err := os.ReadDir(vendorsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read vendors directory: %w", err)
	}

	var vendors []string
	for _, vendorDir := range vendorDirs {
		if vendorDir.IsDir() {
			vendors = append(vendors, vendorDir.Name())
		}
	}

	return vendors, nil
}

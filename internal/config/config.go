// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package config

import (
	"time"
)

type Config struct {
	Defaults        DefaultConfig           `yaml:"defaults"`
	VendorOverrides map[string]VendorConfig `yaml:"vendor_overrides,omitempty"`
}

type DefaultConfig struct {
	IncludeVendors  []string `yaml:"include_vendors"`
	LastTemplateDir string   `yaml:"last_template_dir,omitempty"`
}

// VendorConfig represents configuration that can be defined by vendors
// and optionally overridden by template projects
type VendorConfig struct {
	Vendor           VendorManifest          `yaml:"vendor,omitempty"`
	TemplateDefaults map[string]interface{}  `yaml:"template_defaults,omitempty"`
	Targets          map[string]TargetConfig `yaml:"targets,omitempty"`
	Variables        map[string]interface{}  `yaml:"variables,omitempty"`
	Compilation      CompilationConfig       `yaml:"compilation,omitempty"`
}

// VendorManifest contains metadata about the vendor
type VendorManifest struct {
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`
	Version     string `yaml:"version,omitempty"`
	Author      string `yaml:"author,omitempty"`
	Homepage    string `yaml:"homepage,omitempty"`
}

// TargetConfig contains target-specific configuration
type TargetConfig struct {
	DefaultMode string `yaml:"default_mode,omitempty"`
	// Future fields can be added here as needed
}

// CompilationConfig contains compilation behavior settings
// Currently unused but kept for future extensibility
type CompilationConfig struct {
	// Reserved for future use
}

type LockFile struct {
	Vendors map[string]VendorLock `yaml:"vendors"`
}

type VendorLock struct {
	URL       string    `yaml:"url"`
	Commit    string    `yaml:"commit"`
	FetchedAt time.Time `yaml:"fetched_at"`
}

type InstallationRecord struct {
	Target      string    `yaml:"target"`
	Rule        string    `yaml:"rule"`
	Global      bool      `yaml:"global"`
	ProjectPath string    `yaml:"project_path,omitempty"`
	Mode        string    `yaml:"mode"`
	InstalledAt time.Time `yaml:"installed_at"`
	FilePath    string    `yaml:"file_path"`
}

type InstallationTracker struct {
	Installations []InstallationRecord `yaml:"installations"`
}

// MergedVendorConfigs represents the final merged configuration after
// applying hierarchy: CLI flags > Project config > Vendor configs > Global config
type MergedVendorConfigs struct {
	VendorConfigs map[string]VendorConfig // Keyed by vendor name
	ProjectConfig *Config                 // Project-level configuration
}

// ResolvedTemplateContext contains all configuration data available to a template
type ResolvedTemplateContext struct {
	SourceType        string                 // "local" or vendor name
	TemplateDefaults  map[string]interface{} // Merged defaults for this template's source
	Variables         map[string]interface{} // Merged variables for this template's source
	TargetConfig      TargetConfig           // Target-specific config for current compilation
	CompilationConfig CompilationConfig      // Compilation behavior for this template's source
}

func NewDefaultConfig() *Config {
	return &Config{
		Defaults: DefaultConfig{
			IncludeVendors: []string{},
		},
		VendorOverrides: make(map[string]VendorConfig),
	}
}

// NewDefaultVendorConfig creates a new VendorConfig with sensible defaults
func NewDefaultVendorConfig() VendorConfig {
	return VendorConfig{
		TemplateDefaults: make(map[string]interface{}),
		Targets:          make(map[string]TargetConfig),
		Variables:        make(map[string]interface{}),
		Compilation:      CompilationConfig{},
	}
}

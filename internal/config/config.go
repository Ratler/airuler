package config

import (
	"time"
)

type Config struct {
	Defaults DefaultConfig `yaml:"defaults"`
}

type DefaultConfig struct {
	IncludeVendors []string `yaml:"include_vendors"`
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

func NewDefaultConfig() *Config {
	return &Config{
		Defaults: DefaultConfig{
			IncludeVendors: []string{},
		},
	}
}

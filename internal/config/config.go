package config

import (
	"time"
)

type Config struct {
	Vendors  []Vendor      `yaml:"vendors"`
	Defaults DefaultConfig `yaml:"defaults"`
}

type Vendor struct {
	URL        string `yaml:"url"`
	Alias      string `yaml:"alias"`
	Version    string `yaml:"version"`
	Enabled    bool   `yaml:"enabled"`
	AutoUpdate bool   `yaml:"auto_update"`
}

type DefaultConfig struct {
	IncludeVendors []string          `yaml:"include_vendors"`
	Modes          map[string]string `yaml:"modes"` // target -> mode mapping
}

type LockFile struct {
	Vendors map[string]VendorLock `yaml:"vendors"`
}

type VendorLock struct {
	URL       string    `yaml:"url"`
	Commit    string    `yaml:"commit"`
	FetchedAt time.Time `yaml:"fetched_at"`
}

func NewDefaultConfig() *Config {
	return &Config{
		Vendors: []Vendor{},
		Defaults: DefaultConfig{
			IncludeVendors: []string{},
			Modes: map[string]string{
				"claude": "command", // default mode for Claude
			},
		},
	}
}

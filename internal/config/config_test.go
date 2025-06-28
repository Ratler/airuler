package config

import (
	"testing"
	"time"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg == nil {
		t.Fatal("NewDefaultConfig() returned nil")
	}

	if cfg.Vendors == nil {
		t.Error("NewDefaultConfig() did not initialize Vendors slice")
	}

	if len(cfg.Vendors) != 0 {
		t.Errorf("NewDefaultConfig() Vendors length = %d, expected 0", len(cfg.Vendors))
	}

	if cfg.Defaults.IncludeVendors == nil {
		t.Error("NewDefaultConfig() did not initialize IncludeVendors slice")
	}

	if len(cfg.Defaults.IncludeVendors) != 0 {
		t.Errorf("NewDefaultConfig() IncludeVendors length = %d, expected 0", len(cfg.Defaults.IncludeVendors))
	}

}

func TestVendorStruct(t *testing.T) {
	vendor := Vendor{
		URL:        "https://github.com/user/repo",
		Alias:      "test-vendor",
		Version:    "main",
		Enabled:    true,
		AutoUpdate: false,
	}

	if vendor.URL != "https://github.com/user/repo" {
		t.Errorf("Vendor.URL = %s, expected https://github.com/user/repo", vendor.URL)
	}

	if vendor.Alias != "test-vendor" {
		t.Errorf("Vendor.Alias = %s, expected test-vendor", vendor.Alias)
	}

	if vendor.Version != "main" {
		t.Errorf("Vendor.Version = %s, expected main", vendor.Version)
	}

	if !vendor.Enabled {
		t.Error("Vendor.Enabled should be true")
	}

	if vendor.AutoUpdate {
		t.Error("Vendor.AutoUpdate should be false")
	}
}

func TestLockFileStruct(t *testing.T) {
	now := time.Now()
	lockFile := LockFile{
		Vendors: map[string]VendorLock{
			"test-vendor": {
				URL:       "https://github.com/user/repo",
				Commit:    "abc123",
				FetchedAt: now,
			},
		},
	}

	if len(lockFile.Vendors) != 1 {
		t.Errorf("LockFile.Vendors length = %d, expected 1", len(lockFile.Vendors))
	}

	vendor, exists := lockFile.Vendors["test-vendor"]
	if !exists {
		t.Error("LockFile.Vendors missing test-vendor")
	}

	if vendor.URL != "https://github.com/user/repo" {
		t.Errorf("VendorLock.URL = %s, expected https://github.com/user/repo", vendor.URL)
	}

	if vendor.Commit != "abc123" {
		t.Errorf("VendorLock.Commit = %s, expected abc123", vendor.Commit)
	}

	if !vendor.FetchedAt.Equal(now) {
		t.Errorf("VendorLock.FetchedAt = %v, expected %v", vendor.FetchedAt, now)
	}
}

func TestConfigWithVendors(t *testing.T) {
	cfg := &Config{
		Vendors: []Vendor{
			{
				URL:        "https://github.com/vendor1/repo",
				Alias:      "vendor1",
				Version:    "main",
				Enabled:    true,
				AutoUpdate: true,
			},
			{
				URL:        "https://github.com/vendor2/repo",
				Alias:      "vendor2",
				Version:    "v1.0.0",
				Enabled:    false,
				AutoUpdate: false,
			},
		},
		Defaults: DefaultConfig{
			IncludeVendors: []string{"vendor1"},
		},
	}

	if len(cfg.Vendors) != 2 {
		t.Errorf("Config.Vendors length = %d, expected 2", len(cfg.Vendors))
	}

	// Check first vendor
	v1 := cfg.Vendors[0]
	if v1.Alias != "vendor1" || !v1.Enabled || !v1.AutoUpdate {
		t.Errorf("First vendor configuration incorrect: %+v", v1)
	}

	// Check second vendor
	v2 := cfg.Vendors[1]
	if v2.Alias != "vendor2" || v2.Enabled || v2.AutoUpdate {
		t.Errorf("Second vendor configuration incorrect: %+v", v2)
	}

	// Check defaults
	if len(cfg.Defaults.IncludeVendors) != 1 || cfg.Defaults.IncludeVendors[0] != "vendor1" {
		t.Errorf("Config.Defaults.IncludeVendors = %v, expected [vendor1]", cfg.Defaults.IncludeVendors)
	}

}

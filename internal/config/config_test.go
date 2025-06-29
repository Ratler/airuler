// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

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

	if cfg.Defaults.IncludeVendors == nil {
		t.Error("NewDefaultConfig() did not initialize IncludeVendors slice")
	}

	if len(cfg.Defaults.IncludeVendors) != 0 {
		t.Errorf("NewDefaultConfig() IncludeVendors length = %d, expected 0", len(cfg.Defaults.IncludeVendors))
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

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package git

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5/config"
)

// User represents git user configuration
type User struct {
	Name  string
	Email string
}

// GetGlobalGitUser reads user.name and user.email from global git config using go-git
func GetGlobalGitUser() (*User, error) {
	cfg, err := config.LoadConfig(config.GlobalScope)
	if err != nil {
		return nil, fmt.Errorf("failed to load global git config: %w", err)
	}

	// Check if user info is available
	if cfg.User.Name == "" && cfg.User.Email == "" {
		return nil, fmt.Errorf("no user configuration found in global git config")
	}

	return &User{
		Name:  cfg.User.Name,
		Email: cfg.User.Email,
	}, nil
}

// IsValidEmail performs basic email validation
func IsValidEmail(email string) bool {
	if email == "" {
		return false
	}

	// Basic email regex - not RFC compliant but good enough for git
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// IsValidName performs basic name validation
func IsValidName(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && len(name) >= 2
}

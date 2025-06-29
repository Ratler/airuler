// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package git

import (
	"os"
)

// DefaultGitRepositoryFactory returns the appropriate git factory based on configuration
func DefaultGitRepositoryFactory() RepositoryFactory {
	// Check if we should use mock for testing (highest priority)
	if os.Getenv("AIRULER_USE_MOCK_GIT") == "1" {
		return NewMockGitRepositoryFactory()
	}

	// Use go-git as default (pure Go implementation, no system dependencies)
	return NewGoGitRepositoryFactory()
}

// NewGitRepository creates a new git repository using the default factory
func NewGitRepository(url, localPath string) Repository {
	factory := DefaultGitRepositoryFactory()
	return factory.NewRepository(url, localPath)
}

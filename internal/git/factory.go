package git

import (
	"os"
)

// DefaultGitRepositoryFactory returns the appropriate git factory based on configuration
func DefaultGitRepositoryFactory() GitRepositoryFactory {
	// Check if we should use mock for testing (highest priority)
	if os.Getenv("AIRULER_USE_MOCK_GIT") == "1" {
		return NewMockGitRepositoryFactory()
	}

	// Use go-git as default (pure Go implementation, no system dependencies)
	return NewGoGitRepositoryFactory()
}


// NewGitRepository creates a new git repository using the default factory
func NewGitRepository(url, localPath string) GitRepository {
	factory := DefaultGitRepositoryFactory()
	return factory.NewRepository(url, localPath)
}

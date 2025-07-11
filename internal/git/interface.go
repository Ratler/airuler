// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package git

// Repository defines the interface for git repository operations
type Repository interface {
	// Clone clones the repository to the local path
	Clone() error

	// Pull updates the repository from remote
	Pull() error

	// GetCurrentCommit returns the current commit hash
	GetCurrentCommit() (string, error)

	// GetRemoteCommit returns the remote commit hash
	GetRemoteCommit() (string, error)

	// HasUpdates checks if there are updates available from remote
	HasUpdates() (bool, error)

	// Exists checks if the repository exists locally
	Exists() bool

	// Remove removes the local repository directory
	Remove() error

	// CheckoutCommit checks out a specific commit
	CheckoutCommit(commit string) error

	// CheckoutMainBranch checks out the main/master branch
	CheckoutMainBranch() error

	// ResetToCommit resets the repository to a specific commit
	ResetToCommit(commit string) error
}

// RepositoryFactory creates git repository instances
type RepositoryFactory interface {
	// NewRepository creates a new git repository instance
	NewRepository(url, localPath string) Repository
}

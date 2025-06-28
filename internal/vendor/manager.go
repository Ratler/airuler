package vendor

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ratler/airuler/internal/config"
	"github.com/ratler/airuler/internal/git"
	"gopkg.in/yaml.v3"
)

type Manager struct {
	config   *config.Config
	lockFile *config.LockFile
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config:   cfg,
		lockFile: &config.LockFile{Vendors: make(map[string]config.VendorLock)},
	}
}

func (m *Manager) LoadLockFile() error {
	if _, err := os.Stat("airuler.lock"); os.IsNotExist(err) {
		return nil // Lock file doesn't exist yet
	}

	data, err := os.ReadFile("airuler.lock")
	if err != nil {
		return fmt.Errorf("failed to read lock file: %w", err)
	}

	return yaml.Unmarshal(data, m.lockFile)
}

func (m *Manager) SaveLockFile() error {
	data, err := yaml.Marshal(m.lockFile)
	if err != nil {
		return fmt.Errorf("failed to marshal lock file: %w", err)
	}

	return os.WriteFile("airuler.lock", data, 0644)
}

func (m *Manager) Fetch(url, alias string, update bool) error {
	dirName := git.URLToDirectoryName(url)
	if alias != "" {
		dirName = alias
	}

	vendorPath := filepath.Join("vendors", dirName)
	repo := git.NewRepository(url, vendorPath)

	// Check if vendor already exists
	if repo.Exists() {
		if !update {
			return fmt.Errorf("vendor already exists at %s. Use --update to update", vendorPath)
		}

		// Update existing repository
		if err := repo.Pull(); err != nil {
			return fmt.Errorf("failed to update vendor: %w", err)
		}

		fmt.Printf("Updated vendor: %s\n", dirName)
	} else {
		// Clone new repository
		if err := repo.Clone(); err != nil {
			return fmt.Errorf("failed to clone vendor: %w", err)
		}

		fmt.Printf("Fetched vendor: %s -> %s\n", url, vendorPath)
	}

	// Update lock file
	commit, err := repo.GetCurrentCommit()
	if err != nil {
		return fmt.Errorf("failed to get commit hash: %w", err)
	}

	m.lockFile.Vendors[dirName] = config.VendorLock{
		URL:       url,
		Commit:    commit,
		FetchedAt: time.Now(),
	}

	return m.SaveLockFile()
}

func (m *Manager) Update(vendorNames []string) error {
	if len(vendorNames) == 0 {
		// Update all vendors
		for dirName := range m.lockFile.Vendors {
			if err := m.updateVendor(dirName); err != nil {
				fmt.Printf("Warning: failed to update %s: %v\n", dirName, err)
			}
		}
	} else {
		// Update specific vendors
		for _, name := range vendorNames {
			if err := m.updateVendor(name); err != nil {
				return fmt.Errorf("failed to update %s: %w", name, err)
			}
		}
	}

	return m.SaveLockFile()
}

func (m *Manager) updateVendor(dirName string) error {
	lock, exists := m.lockFile.Vendors[dirName]
	if !exists {
		return fmt.Errorf("vendor %s not found in lock file", dirName)
	}

	vendorPath := filepath.Join("vendors", dirName)
	repo := git.NewRepository(lock.URL, vendorPath)

	if !repo.Exists() {
		return fmt.Errorf("vendor directory does not exist: %s (use 'airuler fetch' to clone missing vendors)", vendorPath)
	}

	hasUpdates, err := repo.HasUpdates()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !hasUpdates {
		fmt.Printf("%s is already up to date\n", dirName)
		return nil
	}

	if err := repo.Pull(); err != nil {
		return fmt.Errorf("failed to pull updates: %w", err)
	}

	// Update lock file entry
	commit, err := repo.GetCurrentCommit()
	if err != nil {
		return fmt.Errorf("failed to get commit hash: %w", err)
	}

	lock.Commit = commit
	lock.FetchedAt = time.Now()
	m.lockFile.Vendors[dirName] = lock

	fmt.Printf("Updated %s to %s\n", dirName, commit[:8])
	return nil
}

func (m *Manager) List() error {
	if len(m.lockFile.Vendors) == 0 {
		fmt.Println("No vendors found")
		return nil
	}

	fmt.Println("Vendors:")
	for dirName, lock := range m.lockFile.Vendors {
		fmt.Printf("  %s\n", dirName)
		fmt.Printf("    URL: %s\n", lock.URL)
		fmt.Printf("    Commit: %s\n", lock.Commit)
		fmt.Printf("    Fetched: %s\n", lock.FetchedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func (m *Manager) Status() error {
	if len(m.lockFile.Vendors) == 0 {
		fmt.Println("No vendors found")
		return nil
	}

	fmt.Println("Vendor Status:")
	for dirName, lock := range m.lockFile.Vendors {
		vendorPath := filepath.Join("vendors", dirName)
		repo := git.NewRepository(lock.URL, vendorPath)

		if !repo.Exists() {
			fmt.Printf("  %s: MISSING\n", dirName)
			continue
		}

		hasUpdates, err := repo.HasUpdates()
		if err != nil {
			fmt.Printf("  %s: ERROR (%v)\n", dirName, err)
			continue
		}

		if hasUpdates {
			fmt.Printf("  %s: UPDATE AVAILABLE\n", dirName)
		} else {
			fmt.Printf("  %s: UP TO DATE\n", dirName)
		}
	}

	return nil
}

func (m *Manager) Remove(vendorName string) error {
	lock, exists := m.lockFile.Vendors[vendorName]
	if !exists {
		return fmt.Errorf("vendor %s not found", vendorName)
	}

	vendorPath := filepath.Join("vendors", vendorName)
	repo := git.NewRepository(lock.URL, vendorPath)

	if err := repo.Remove(); err != nil {
		return fmt.Errorf("failed to remove vendor directory: %w", err)
	}

	delete(m.lockFile.Vendors, vendorName)

	if err := m.SaveLockFile(); err != nil {
		return fmt.Errorf("failed to update lock file: %w", err)
	}

	fmt.Printf("Removed vendor: %s\n", vendorName)
	return nil
}

func (m *Manager) RestoreMissingVendors() error {
	if len(m.lockFile.Vendors) == 0 {
		fmt.Println("No vendors found in lock file")
		return nil
	}

	var missingVendors []string
	var restoredCount int

	// Check which vendors are missing
	for dirName := range m.lockFile.Vendors {
		vendorPath := filepath.Join("vendors", dirName)
		repo := git.NewRepository("", vendorPath) // URL not needed for Exists() check
		if !repo.Exists() {
			missingVendors = append(missingVendors, dirName)
		}
	}

	if len(missingVendors) == 0 {
		fmt.Println("All vendors are present")
		return nil
	}

	fmt.Printf("Found %d missing vendor(s), restoring...\n", len(missingVendors))

	// Restore missing vendors
	for _, dirName := range missingVendors {
		lock := m.lockFile.Vendors[dirName]
		vendorPath := filepath.Join("vendors", dirName)
		repo := git.NewRepository(lock.URL, vendorPath)

		fmt.Printf("Cloning %s...\n", dirName)
		if err := repo.Clone(); err != nil {
			fmt.Printf("Warning: failed to clone %s: %v\n", dirName, err)
			continue
		}

		// Ensure we're on the main branch and at the correct commit
		if err := repo.CheckoutMainBranch(); err != nil {
			fmt.Printf("Warning: failed to checkout main branch for %s: %v\n", dirName, err)
			continue
		}

		// Reset to the specific commit from lock file (this maintains branch state)
		if err := repo.ResetToCommit(lock.Commit); err != nil {
			fmt.Printf("Warning: failed to reset to commit %s for %s: %v\n", lock.Commit, dirName, err)
			continue
		}

		fmt.Printf("✅ Restored %s at %s\n", dirName, lock.Commit[:8])
		restoredCount++
	}

	if restoredCount == len(missingVendors) {
		fmt.Printf("\n🎉 Successfully restored %d vendor(s)\n", restoredCount)
	} else {
		fmt.Printf("\n⚠️  Restored %d of %d vendor(s)\n", restoredCount, len(missingVendors))
	}

	return nil
}

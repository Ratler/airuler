package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ratler/airuler/internal/config"
)

var listFilter string

type uniqueInstall struct {
	Target      string
	Rule        string
	Mode        string
	FilePath    string
	Global      bool
	ProjectPath string
	InstalledAt time.Time
}

var listInstalledCmd = &cobra.Command{
	Use:   "list-installed",
	Short: "List all installed templates",
	Long: `List all installed templates with details including target AI tool, 
installation location, and installation time.

Templates are grouped by project and global installations for better organization.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		return runListInstalled()
	},
}

func init() {
	rootCmd.AddCommand(listInstalledCmd)
	listInstalledCmd.Flags().
		StringVarP(&listFilter, "filter", "f", "", "Filter templates by keyword (case-insensitive)")
}

func runListInstalled() error {
	// Load global installation tracker
	globalTracker, err := config.LoadGlobalInstallationTracker()
	if err != nil {
		return fmt.Errorf("failed to load global installation tracker: %w", err)
	}

	// Load project installation tracker if in a project
	var projectTracker *config.InstallationTracker
	projectTracker, _ = config.LoadProjectInstallationTracker()

	// Collect and deduplicate installations
	uniqueMap := make(map[string]uniqueInstall)

	// Process global installations
	for _, record := range globalTracker.Installations {
		if shouldIncludeRecord(record, listFilter) {
			key := fmt.Sprintf("%s-%s-%s-%s-global", record.Target, record.Rule, record.Mode, record.FilePath)
			if existing, exists := uniqueMap[key]; !exists || record.InstalledAt.After(existing.InstalledAt) {
				uniqueMap[key] = uniqueInstall{
					Target:      record.Target,
					Rule:        record.Rule,
					Mode:        record.Mode,
					FilePath:    record.FilePath,
					Global:      true,
					InstalledAt: record.InstalledAt,
				}
			}
		}
	}

	// Process project installations
	if projectTracker != nil {
		for _, record := range projectTracker.Installations {
			if shouldIncludeRecord(record, listFilter) {
				key := fmt.Sprintf(
					"%s-%s-%s-%s-%s",
					record.Target,
					record.Rule,
					record.Mode,
					record.FilePath,
					record.ProjectPath,
				)
				if existing, exists := uniqueMap[key]; !exists || record.InstalledAt.After(existing.InstalledAt) {
					uniqueMap[key] = uniqueInstall{
						Target:      record.Target,
						Rule:        record.Rule,
						Mode:        record.Mode,
						FilePath:    record.FilePath,
						Global:      false,
						ProjectPath: record.ProjectPath,
						InstalledAt: record.InstalledAt,
					}
				}
			}
		}
	}

	// Convert map to slice
	var allInstalls []uniqueInstall
	for _, install := range uniqueMap {
		allInstalls = append(allInstalls, install)
	}

	// Check if no templates are installed
	if len(allInstalls) == 0 {
		if listFilter != "" {
			fmt.Println("🔍 No installed templates found matching filter:", listFilter)
		} else {
			fmt.Println("📋 No templates are currently installed")
		}
		return nil
	}

	// Group installations by scope (global vs project)
	var globalInstalls []uniqueInstall
	projectInstalls := make(map[string][]uniqueInstall)

	for _, install := range allInstalls {
		if install.Global {
			globalInstalls = append(globalInstalls, install)
		} else {
			projectInstalls[install.ProjectPath] = append(projectInstalls[install.ProjectPath], install)
		}
	}

	// Sort installations
	sortInstalls := func(installs []uniqueInstall) {
		sort.Slice(installs, func(i, j int) bool {
			if installs[i].Target != installs[j].Target {
				return installs[i].Target < installs[j].Target
			}
			if installs[i].Rule != installs[j].Rule {
				return installs[i].Rule < installs[j].Rule
			}
			return installs[i].Mode < installs[j].Mode
		})
	}

	// Display header
	fmt.Println("📋 Installed Templates")
	if listFilter != "" {
		fmt.Printf("🔍 Filter: \"%s\"\n", listFilter)
	}
	fmt.Println()

	// Display global installations
	if len(globalInstalls) > 0 {
		fmt.Println("🌍 Global Installations")
		fmt.Println(strings.Repeat("=", 78))
		sortInstalls(globalInstalls)
		displayTable(globalInstalls)
		fmt.Println()
	}

	// Display project installations
	if len(projectInstalls) > 0 {
		// Sort project paths for consistent output
		var projectPaths []string
		for path := range projectInstalls {
			projectPaths = append(projectPaths, path)
		}
		sort.Strings(projectPaths)

		for _, projPath := range projectPaths {
			// Skip empty project paths
			if projPath == "" {
				continue
			}
			// Display only the project name (last directory) instead of full path
			projectName := filepath.Base(projPath)
			fmt.Printf("📁 Project: %s\n", projectName)
			fmt.Println(strings.Repeat("=", 78))
			installs := projectInstalls[projPath]
			sortInstalls(installs)
			displayTable(installs)
			fmt.Println()
		}
	}

	// Display summary
	fmt.Printf("Total: %d template(s) installed\n", len(allInstalls))

	return nil
}

func displayTable(installs []uniqueInstall) {
	// Print table header with wider columns
	fmt.Printf("%-8s %-20s %-8s %-25s %-15s\n", "Target", "Rule", "Mode", "File", "Installed")
	fmt.Println(strings.Repeat("-", 78))

	// Print each row
	for _, install := range installs {
		target := install.Target
		rule := install.Rule
		if rule == "*" {
			rule = "all templates"
		}

		mode := install.Mode
		if mode == "" {
			mode = "-"
		}

		fileName := filepath.Base(install.FilePath)
		timeAgo := formatTimeAgo(install.InstalledAt)

		// Truncate long strings
		if len(rule) > 20 {
			rule = rule[:17] + "..."
		}
		if len(fileName) > 25 {
			fileName = fileName[:22] + "..."
		}

		fmt.Printf("%-8s %-20s %-8s %-25s %-15s\n", target, rule, mode, fileName, timeAgo)
	}
}

func shouldIncludeRecord(record config.InstallationRecord, filter string) bool {
	if filter == "" {
		return true
	}

	// Case-insensitive search
	filter = strings.ToLower(filter)

	// Check rule name
	if strings.Contains(strings.ToLower(record.Rule), filter) {
		return true
	}

	// Check target
	if strings.Contains(strings.ToLower(record.Target), filter) {
		return true
	}

	// Check file path
	if strings.Contains(strings.ToLower(record.FilePath), filter) {
		return true
	}

	// Check mode
	if record.Mode != "" && strings.Contains(strings.ToLower(record.Mode), filter) {
		return true
	}

	return false
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d min ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 30*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if duration < 365*24*time.Hour {
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
	years := int(duration.Hours() / 24 / 365)
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

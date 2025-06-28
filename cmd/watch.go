package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ratler/airuler/internal/compiler"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch templates and auto-compile on changes",
	Long: `Watch template files for changes and automatically recompile when they change.

This is useful during development to get immediate feedback when editing templates.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🔍 Watching templates for changes... (Press Ctrl+C to stop)")
		fmt.Println("Note: This is a basic implementation. For production use, consider using external tools like 'watchexec'.")

		// Simple polling-based watch implementation
		lastModTime, err := getLastModTime()
		if err != nil {
			return fmt.Errorf("failed to get initial modification time: %w", err)
		}

		for {
			time.Sleep(2 * time.Second)

			currentModTime, err := getLastModTime()
			if err != nil {
				fmt.Printf("Warning: failed to check modification time: %v\n", err)
				continue
			}

			if currentModTime.After(lastModTime) {
				fmt.Printf("📝 Changes detected at %s, recompiling...\n", currentModTime.Format("15:04:05"))

				// Run compile command
				if err := compileTemplates(getAllTargets()); err != nil {
					fmt.Printf("❌ Compilation failed: %v\n", err)
				} else {
					fmt.Printf("✅ Compilation successful at %s\n", time.Now().Format("15:04:05"))
				}

				lastModTime = currentModTime
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
}

func getLastModTime() (time.Time, error) {
	var latest time.Time

	// Check templates directory
	err := filepath.Walk("templates", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if !info.IsDir() && filepath.Ext(path) == ".tmpl" {
			if info.ModTime().After(latest) {
				latest = info.ModTime()
			}
		}
		return nil
	})

	return latest, err
}

func getAllTargets() []compiler.Target {
	return compiler.AllTargets
}

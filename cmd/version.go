// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version     string
	buildCommit string
	buildDate   string
)

// SetVersionInfo sets the version information from main
func SetVersionInfo(v, commit, date string) {
	version = v
	buildCommit = commit
	buildDate = date
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display airuler version, build commit, and build date information.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("airuler version %s\n", version)
		fmt.Printf("Build commit: %s\n", buildCommit)
		fmt.Printf("Build date: %s\n", buildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
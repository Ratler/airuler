// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package main

import (
	"github.com/ratler/airuler/cmd"
)

// Version information injected by goreleaser
var (
	Version     = "dev"
	BuildCommit = "unknown"
	BuildDate   = "unknown"
)

func main() {
	cmd.SetVersionInfo(Version, BuildCommit, BuildDate)
	cmd.Execute()
}

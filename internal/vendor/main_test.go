// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package vendor

import (
	"os"
	"testing"
)

// TestMain sets up the test environment for the vendor package
func TestMain(m *testing.M) {
	// Ensure we use mock git for all tests in this package
	os.Setenv("AIRULER_USE_MOCK_GIT", "1")

	// Run tests
	code := m.Run()

	// Clean up (optional)
	os.Unsetenv("AIRULER_USE_MOCK_GIT")

	os.Exit(code)
}

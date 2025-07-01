// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package utils

import (
	"testing"
	"time"
)

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "just now",
			time:     now.Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "1 minute ago",
			time:     now.Add(-1 * time.Minute),
			expected: "1 minute ago",
		},
		{
			name:     "2 minutes ago",
			time:     now.Add(-2 * time.Minute),
			expected: "2 min ago",
		},
		{
			name:     "30 minutes ago",
			time:     now.Add(-30 * time.Minute),
			expected: "30 min ago",
		},
		{
			name:     "1 hour ago",
			time:     now.Add(-1 * time.Hour),
			expected: "1 hour ago",
		},
		{
			name:     "2 hours ago",
			time:     now.Add(-2 * time.Hour),
			expected: "2 hours ago",
		},
		{
			name:     "12 hours ago",
			time:     now.Add(-12 * time.Hour),
			expected: "12 hours ago",
		},
		{
			name:     "1 day ago",
			time:     now.Add(-24 * time.Hour),
			expected: "1 day ago",
		},
		{
			name:     "3 days ago",
			time:     now.Add(-3 * 24 * time.Hour),
			expected: "3 days ago",
		},
		{
			name:     "15 days ago",
			time:     now.Add(-15 * 24 * time.Hour),
			expected: "15 days ago",
		},
		{
			name:     "29 days ago",
			time:     now.Add(-29 * 24 * time.Hour),
			expected: "29 days ago",
		},
		{
			name:     "35 days ago (over a month)",
			time:     now.Add(-35 * 24 * time.Hour),
			expected: now.Add(-35 * 24 * time.Hour).Format("2006-01-02"),
		},
		{
			name:     "1 year ago",
			time:     now.Add(-365 * 24 * time.Hour),
			expected: now.Add(-365 * 24 * time.Hour).Format("2006-01-02"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTimeAgo(tt.time)
			if result != tt.expected {
				t.Errorf("FormatTimeAgo(%v) = %q, want %q", tt.time, result, tt.expected)
			}
		})
	}
}

func TestFormatTimeAgoWithFixedTimes(t *testing.T) {
	// Test with relative durations to avoid time-based flakiness
	now := time.Now()

	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "exactly 1 minute ago",
			duration: 1 * time.Minute,
			expected: "1 minute ago",
		},
		{
			name:     "exactly 1 hour ago",
			duration: 1 * time.Hour,
			expected: "1 hour ago",
		},
		{
			name:     "exactly 1 day ago",
			duration: 24 * time.Hour,
			expected: "1 day ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testTime := now.Add(-tt.duration)
			result := FormatTimeAgo(testTime)

			if result != tt.expected {
				t.Errorf("FormatTimeAgo(%v) = %q, want %q", testTime, result, tt.expected)
			}
		})
	}
}

func TestFormatTimeAgoBoundaryConditions(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "59 seconds (just now)",
			duration: 59 * time.Second,
			expected: "just now",
		},
		{
			name:     "60 seconds (1 minute ago)",
			duration: 60 * time.Second,
			expected: "1 minute ago",
		},
		{
			name:     "61 seconds (1 minute ago)",
			duration: 61 * time.Second,
			expected: "1 minute ago",
		},
		{
			name:     "59 minutes 59 seconds",
			duration: 59*time.Minute + 59*time.Second,
			expected: "59 min ago",
		},
		{
			name:     "60 minutes (1 hour ago)",
			duration: 60 * time.Minute,
			expected: "1 hour ago",
		},
		{
			name:     "23 hours 59 minutes",
			duration: 23*time.Hour + 59*time.Minute,
			expected: "23 hours ago",
		},
		{
			name:     "24 hours (1 day ago)",
			duration: 24 * time.Hour,
			expected: "1 day ago",
		},
		{
			name:     "29 days 23 hours",
			duration: 29*24*time.Hour + 23*time.Hour,
			expected: "29 days ago",
		},
		{
			name:     "30 days (date format)",
			duration: 30 * 24 * time.Hour,
			expected: now.Add(-30 * 24 * time.Hour).Format("2006-01-02"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testTime := now.Add(-tt.duration)
			result := FormatTimeAgo(testTime)
			if result != tt.expected {
				t.Errorf("FormatTimeAgo(now - %v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

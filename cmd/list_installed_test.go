package cmd

import (
	"testing"
	"time"

	"github.com/ratler/airuler/internal/config"
)

func TestShouldIncludeRecord(t *testing.T) {
	tests := []struct {
		name     string
		record   config.InstallationRecord
		filter   string
		expected bool
	}{
		{
			name: "no filter includes all",
			record: config.InstallationRecord{
				Target: "claude",
				Rule:   "test-rule",
			},
			filter:   "",
			expected: true,
		},
		{
			name: "filter matches rule name",
			record: config.InstallationRecord{
				Target: "claude",
				Rule:   "test-rule",
			},
			filter:   "test",
			expected: true,
		},
		{
			name: "filter matches target",
			record: config.InstallationRecord{
				Target: "claude",
				Rule:   "other-rule",
			},
			filter:   "claude",
			expected: true,
		},
		{
			name: "filter matches mode",
			record: config.InstallationRecord{
				Target: "claude",
				Rule:   "test-rule",
				Mode:   "memory",
			},
			filter:   "memory",
			expected: true,
		},
		{
			name: "filter matches filepath",
			record: config.InstallationRecord{
				Target:   "claude",
				Rule:     "test-rule",
				FilePath: "/path/to/command.md",
			},
			filter:   "command",
			expected: true,
		},
		{
			name: "filter case insensitive",
			record: config.InstallationRecord{
				Target: "Claude",
				Rule:   "Test-Rule",
			},
			filter:   "CLAUDE",
			expected: true,
		},
		{
			name: "filter no match",
			record: config.InstallationRecord{
				Target: "cursor",
				Rule:   "test-rule",
			},
			filter:   "claude",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldIncludeRecord(tt.record, tt.filter)
			if result != tt.expected {
				t.Errorf("shouldIncludeRecord() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "just now",
			time:     time.Now().Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "minutes ago",
			time:     time.Now().Add(-5 * time.Minute),
			expected: "5 min ago",
		},
		{
			name:     "1 minute ago",
			time:     time.Now().Add(-1 * time.Minute),
			expected: "1 minute ago",
		},
		{
			name:     "hours ago",
			time:     time.Now().Add(-3 * time.Hour),
			expected: "3 hours ago",
		},
		{
			name:     "1 hour ago",
			time:     time.Now().Add(-1 * time.Hour),
			expected: "1 hour ago",
		},
		{
			name:     "days ago",
			time:     time.Now().Add(-5 * 24 * time.Hour),
			expected: "5 days ago",
		},
		{
			name:     "1 day ago",
			time:     time.Now().Add(-1 * 24 * time.Hour),
			expected: "1 day ago",
		},
		{
			name:     "months ago",
			time:     time.Now().Add(-60 * 24 * time.Hour),
			expected: "2 months ago",
		},
		{
			name:     "1 year ago",
			time:     time.Now().Add(-365 * 24 * time.Hour),
			expected: "1 year ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimeAgo(tt.time)
			if result != tt.expected {
				t.Errorf("formatTimeAgo() = %v, want %v", result, tt.expected)
			}
		})
	}
}

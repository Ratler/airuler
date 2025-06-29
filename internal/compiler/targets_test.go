// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package compiler

import (
	"strings"
	"testing"

	"github.com/ratler/airuler/internal/template"
)

func TestNewCompiler(t *testing.T) {
	compiler := NewCompiler()
	if compiler == nil {
		t.Fatal("NewCompiler() returned nil")
	}

	if compiler.engine == nil {
		t.Error("NewCompiler() did not initialize engine")
	}
}

func TestTargetConstants(t *testing.T) {
	expectedTargets := []Target{TargetCursor, TargetClaude, TargetCline, TargetCopilot, TargetRoo}

	if len(AllTargets) != len(expectedTargets) {
		t.Errorf("AllTargets length = %d, expected %d", len(AllTargets), len(expectedTargets))
	}

	for i, target := range expectedTargets {
		if AllTargets[i] != target {
			t.Errorf("AllTargets[%d] = %v, expected %v", i, AllTargets[i], target)
		}
	}
}

func TestLoadTemplate(t *testing.T) {
	compiler := NewCompiler()

	err := compiler.LoadTemplate("test", "Hello {{.Name}}!")
	if err != nil {
		t.Errorf("LoadTemplate() unexpected error: %v", err)
	}
}

func TestCompileTemplate(t *testing.T) {
	compiler := NewCompiler()

	templateContent := `# {{.Name}} Rule

{{if eq .Target "cursor"}}---
description: {{.Description}}
globs: {{.Globs}}
alwaysApply: true
---
{{end}}

This is a rule for {{.Target}}.

{{if eq .Target "claude"}}Arguments: $ARGUMENTS{{end}}`

	compiler.LoadTemplate("test-rule", templateContent)

	tests := []struct {
		name         string
		target       Target
		data         template.Data
		expectError  bool
		checkContent func(string) bool
		checkFile    func(string) bool
	}{
		{
			name:   "cursor target",
			target: TargetCursor,
			data: template.Data{
				Name:        "test-rule",
				Description: "Test rule",
				Globs:       "**/*.ts",
			},
			expectError: false,
			checkContent: func(content string) bool {
				return strings.Contains(content, "---") &&
					strings.Contains(content, "description: Test rule") &&
					strings.Contains(content, "globs: **/*.ts") &&
					strings.Contains(content, "This is a rule for cursor")
			},
			checkFile: func(filename string) bool {
				return filename == "test-rule.mdc"
			},
		},
		{
			name:   "claude target",
			target: TargetClaude,
			data: template.Data{
				Name:        "test-rule",
				Description: "Test rule",
			},
			expectError: false,
			checkContent: func(content string) bool {
				return !strings.Contains(content, "---") &&
					strings.Contains(content, "This is a rule for claude") &&
					strings.Contains(content, "Arguments: $ARGUMENTS")
			},
			checkFile: func(filename string) bool {
				return filename == "test-rule.md"
			},
		},
		{
			name:   "cline target",
			target: TargetCline,
			data: template.Data{
				Name:        "test-rule",
				Description: "Test rule",
			},
			expectError: false,
			checkContent: func(content string) bool {
				return !strings.Contains(content, "---") &&
					strings.Contains(content, "This is a rule for cline")
			},
			checkFile: func(filename string) bool {
				return filename == "test-rule.md"
			},
		},
		{
			name:   "copilot target",
			target: TargetCopilot,
			data: template.Data{
				Name:        "test-rule",
				Description: "Test rule",
				Globs:       "**/*.ts",
			},
			expectError: false,
			checkContent: func(content string) bool {
				return !strings.Contains(content, "---") &&
					!strings.Contains(content, "description:") &&
					!strings.Contains(content, "applyTo:") &&
					strings.Contains(content, "This is a rule for copilot")
			},
			checkFile: func(filename string) bool {
				return filename == "test-rule.copilot-instructions.md"
			},
		},
		{
			name:   "roo target",
			target: TargetRoo,
			data: template.Data{
				Name:        "test-rule",
				Description: "Test rule",
			},
			expectError: false,
			checkContent: func(content string) bool {
				return !strings.Contains(content, "---") &&
					strings.Contains(content, "This is a rule for roo")
			},
			checkFile: func(filename string) bool {
				return filename == "test-rule.md"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := compiler.CompileTemplate("test-rule", tt.target, tt.data)

			if tt.expectError && err == nil {
				t.Errorf("CompileTemplate() expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("CompileTemplate() unexpected error: %v", err)
				return
			}

			if !tt.expectError {
				if rule.Target != tt.target {
					t.Errorf("CompileTemplate() target = %v, expected %v", rule.Target, tt.target)
				}

				if rule.Name != "test-rule" {
					t.Errorf("CompileTemplate() name = %v, expected %v", rule.Name, "test-rule")
				}

				if !tt.checkContent(rule.Content) {
					t.Errorf("CompileTemplate() content validation failed. Content:\n%s", rule.Content)
				}

				if !tt.checkFile(rule.Filename) {
					t.Errorf("CompileTemplate() filename = %v, expected to pass validation", rule.Filename)
				}
			}
		})
	}
}

func TestProcessors(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name         string
		processor    func(string, string, template.Data) (string, string)
		content      string
		templateName string
		data         template.Data
		expectedExt  string
		checkContent func(string) bool
	}{
		{
			name:         "cursor processor",
			processor:    compiler.processCursor,
			content:      "Simple content",
			templateName: "test",
			data:         template.Data{Description: "Test desc", Globs: "*.ts"},
			expectedExt:  ".mdc",
			checkContent: func(content string) bool {
				return strings.Contains(content, "---") &&
					strings.Contains(content, "description: Test desc") &&
					strings.Contains(content, "globs: *.ts") &&
					strings.Contains(content, "Simple content")
			},
		},
		{
			name:         "cursor processor with existing front matter",
			processor:    compiler.processCursor,
			content:      "---\nexisting: true\n---\nContent",
			templateName: "test",
			data:         template.Data{},
			expectedExt:  ".mdc",
			checkContent: func(content string) bool {
				return strings.Contains(content, "existing: true") &&
					strings.Contains(content, "Content")
			},
		},
		{
			name:         "claude processor",
			processor:    compiler.processClaude,
			content:      "Content with $ARGUMENTS",
			templateName: "test",
			data:         template.Data{},
			expectedExt:  ".md",
			checkContent: func(content string) bool {
				return strings.Contains(content, "Content with $ARGUMENTS")
			},
		},
		{
			name:         "cline processor",
			processor:    compiler.processCline,
			content:      "Content", // Front matter now stripped at template loading stage
			templateName: "test",
			data:         template.Data{},
			expectedExt:  ".md",
			checkContent: func(content string) bool {
				return content == "Content"
			},
		},
		{
			name:         "copilot processor",
			processor:    compiler.processCopilot,
			content:      "Simple content",
			templateName: "test",
			data:         template.Data{Description: "Test desc", Globs: "*.ts"},
			expectedExt:  ".copilot-instructions.md",
			checkContent: func(content string) bool {
				return !strings.Contains(content, "---") &&
					!strings.Contains(content, "description:") &&
					!strings.Contains(content, "applyTo:") &&
					content == "Simple content"
			},
		},
		{
			name:         "copilot processor with front matter removal",
			processor:    compiler.processCopilot,
			content:      "---\ndescription: test\napplyTo: *.ts\n---\n\nSimple content",
			templateName: "test",
			data:         template.Data{Description: "Test desc", Globs: "*.ts"},
			expectedExt:  ".copilot-instructions.md",
			checkContent: func(content string) bool {
				return !strings.Contains(content, "---") &&
					!strings.Contains(content, "description:") &&
					!strings.Contains(content, "applyTo:") &&
					content == "Simple content"
			},
		},
		{
			name:         "roo processor",
			processor:    compiler.processRoo,
			content:      "Content",
			templateName: "test",
			data:         template.Data{},
			expectedExt:  ".md",
			checkContent: func(content string) bool {
				return content == "Content"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, filename := tt.processor(tt.content, tt.templateName, tt.data)

			if !strings.HasSuffix(filename, tt.expectedExt) {
				t.Errorf("Processor filename = %v, expected to end with %v", filename, tt.expectedExt)
			}

			if !tt.checkContent(content) {
				t.Errorf("Processor content validation failed. Content:\n%s", content)
			}
		})
	}
}

func TestGetOutputPath(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		target   Target
		filename string
		expected string
	}{
		{TargetCursor, "test.mdc", "compiled/cursor/test.mdc"},
		{TargetClaude, "test.md", "compiled/claude/test.md"},
		{TargetCline, "test.md", "compiled/cline/test.md"},
		{TargetCopilot, "test.copilot-instructions.md", "compiled/copilot/test.copilot-instructions.md"},
		{TargetRoo, "test.md", "compiled/roo/test.md"},
	}

	for _, tt := range tests {
		t.Run(string(tt.target), func(t *testing.T) {
			result := compiler.GetOutputPath(tt.target, tt.filename)
			if result != tt.expected {
				t.Errorf("GetOutputPath() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		function func(template.Data, string) string
		data     template.Data
		fallback string
		expected string
	}{
		{
			name:     "getDescription with data",
			function: getDescription,
			data:     template.Data{Description: "Custom description"},
			fallback: "fallback",
			expected: "Custom description",
		},
		{
			name:     "getDescription with fallback",
			function: getDescription,
			data:     template.Data{},
			fallback: "fallback",
			expected: "AI coding rules for fallback",
		},
		{
			name:     "getGlobs with data",
			function: func(data template.Data, _ string) string { return getGlobs(data) },
			data:     template.Data{Globs: "*.ts,*.js"},
			fallback: "",
			expected: "*.ts,*.js",
		},
		{
			name:     "getGlobs with empty data",
			function: func(data template.Data, _ string) string { return getGlobs(data) },
			data:     template.Data{},
			fallback: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function(tt.data, tt.fallback)
			if result != tt.expected {
				t.Errorf("Function result = %v, expected %v", result, tt.expected)
			}
		})
	}
}

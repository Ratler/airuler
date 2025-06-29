// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package template

import (
	"strings"
	"testing"
)

func TestPartialsWithLanguageField(t *testing.T) {
	engine := NewEngine()

	// Test 1: Simple partial that uses .Language field
	partialContent := `{{if eq .Language "go"}}Go-specific content{{else}}Generic content{{end}}`
	mainTemplateContent := `Main template using partial: {{template "lang-partial" .}}`

	// Load the partial first
	err := engine.LoadTemplate("lang-partial", partialContent)
	if err != nil {
		t.Fatalf("Failed to load partial template: %v", err)
	}

	// Load the main template
	err = engine.LoadTemplate("main", mainTemplateContent)
	if err != nil {
		t.Fatalf("Failed to load main template: %v", err)
	}

	// Test with Language = "go"
	data := Data{
		Name:     "test",
		Language: "go",
	}

	result, err := engine.Render("main", data)
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	expected := "Main template using partial: Go-specific content"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Test with Language = "python"
	data.Language = "python"
	result, err = engine.Render("main", data)
	if err != nil {
		t.Fatalf("Failed to render template with Language=python: %v", err)
	}

	expected = "Main template using partial: Generic content"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestComplexPartialsWithAllFields(t *testing.T) {
	engine := NewEngine()

	// Create a header partial similar to the one in examples
	headerPartial := `{{if eq .Target "cursor"}}---
description: {{if .Description}}{{.Description}}{{else}}{{title .Name}} coding guidelines{{end}}
globs: {{if .Globs}}{{.Globs}}{{else}}**/*{{end}}
---
{{end}}

# {{title .Name}} Guidelines

{{if .ProjectType}}*Project Type: {{title .ProjectType}}*{{end}}
{{if .Language}}*Language: {{title .Language}}*{{end}}
{{if .Framework}}*Framework: {{title .Framework}}*{{end}}`

	// Create a language-specific guidelines partial
	languageGuidelinesPartial := `## Language-Specific Guidelines

{{if eq .Language "go"}}
### Go Best Practices
- Use gofmt for formatting
- Follow effective Go principles
- Use proper error handling
{{else if eq .Language "python"}}
### Python Best Practices
- Follow PEP 8 style guide
- Use type hints
- Write docstrings
{{else if eq .Language "javascript"}}
### JavaScript Best Practices
- Use ESLint for code quality
- Prefer const/let over var
- Use arrow functions appropriately
{{else}}
### General Guidelines
- Follow language-specific conventions
- Write readable and maintainable code
{{end}}`

	// Main template that uses both partials
	mainTemplate := `{{template "header" .}}

{{template "lang-guidelines" .}}

## Additional Notes
Target: {{.Target}}
Project: {{.Name}}`

	// Load all templates
	tests := []struct {
		name     string
		template string
		content  string
	}{
		{"header", "header", headerPartial},
		{"lang-guidelines", "lang-guidelines", languageGuidelinesPartial},
		{"main", "main", mainTemplate},
	}

	for _, tt := range tests {
		if err := engine.LoadTemplate(tt.template, tt.content); err != nil {
			t.Fatalf("Failed to load template %s: %v", tt.name, err)
		}
	}

	// Test with different language configurations
	testCases := []struct {
		name         string
		data         Data
		expectInText []string
	}{
		{
			name: "Go language test",
			data: Data{
				Target:      "cursor",
				Name:        "my-project",
				Language:    "go",
				ProjectType: "api",
				Framework:   "gin",
				Description: "Go API project",
				Globs:       "**/*.go",
			},
			expectInText: []string{
				"*Language: Go*",
				"### Go Best Practices",
				"Use gofmt for formatting",
				"Follow effective Go principles",
				"description: Go API project",
				"globs: **/*.go",
			},
		},
		{
			name: "Python language test",
			data: Data{
				Target:      "claude",
				Name:        "python-app",
				Language:    "python",
				ProjectType: "web",
				Framework:   "django",
			},
			expectInText: []string{
				"*Language: Python*",
				"### Python Best Practices",
				"Follow PEP 8 style guide",
				"Use type hints",
			},
		},
		{
			name: "JavaScript language test",
			data: Data{
				Target:    "cline",
				Name:      "js-project",
				Language:  "javascript",
				Framework: "react",
			},
			expectInText: []string{
				"*Language: Javascript*", // title case
				"### JavaScript Best Practices",
				"Use ESLint for code quality",
				"Prefer const/let over var",
			},
		},
		{
			name: "Unknown language test",
			data: Data{
				Target:   "claude",
				Name:     "other-project",
				Language: "rust",
			},
			expectInText: []string{
				"*Language: Rust*",
				"### General Guidelines",
				"Follow language-specific conventions",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := engine.Render("main", tc.data)
			if err != nil {
				t.Fatalf("Failed to render template for %s: %v", tc.name, err)
			}

			for _, expected := range tc.expectInText {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected to find %q in result for %s\nGot:\n%s", expected, tc.name, result)
				}
			}
		})
	}
}

func TestNestedPartialsWithLanguage(t *testing.T) {
	engine := NewEngine()

	// Level 3 partial (deepest)
	deepPartial := `Deep content for {{.Language}}`

	// Level 2 partial (uses level 3)
	middlePartial := `Middle: {{template "deep" .}} - Language: {{.Language}}`

	// Level 1 partial (uses level 2)
	topPartial := `Top: {{template "middle" .}} - Also: {{.Language}}`

	// Main template
	mainTemplate := `Main: {{template "top" .}}`

	// Load all templates
	templates := map[string]string{
		"deep":   deepPartial,
		"middle": middlePartial,
		"top":    topPartial,
		"main":   mainTemplate,
	}

	for name, content := range templates {
		if err := engine.LoadTemplate(name, content); err != nil {
			t.Fatalf("Failed to load template %s: %v", name, err)
		}
	}

	// Test nested partial rendering
	data := Data{
		Language: "typescript",
	}

	result, err := engine.Render("main", data)
	if err != nil {
		t.Fatalf("Failed to render nested partials: %v", err)
	}

	// Check that Language field is accessible at all levels
	expectedParts := []string{
		"Deep content for typescript",
		"Language: typescript",
		"Also: typescript",
	}

	for _, expected := range expectedParts {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected to find %q in result\nGot: %s", expected, result)
		}
	}
}

func TestPartialWithAllTemplateFields(t *testing.T) {
	engine := NewEngine()

	// Comprehensive partial that uses all template fields
	comprehensivePartial := `# {{title .Name}} - {{title .Target}}

{{if .Description}}Description: {{.Description}}{{end}}
{{if .Globs}}Globs: {{.Globs}}{{end}}
{{if .ProjectType}}Project Type: {{title .ProjectType}}{{end}}
{{if .Language}}Language: {{title .Language}}{{end}}
{{if .Framework}}Framework: {{title .Framework}}{{end}}
{{if .Tags}}Tags: {{join .Tags ", "}}{{end}}
{{if .AlwaysApply}}Always Apply: {{.AlwaysApply}}{{end}}
{{if .Documentation}}Documentation: {{.Documentation}}{{end}}
{{if .StyleGuide}}Style Guide: {{.StyleGuide}}{{end}}
{{if .Examples}}Examples: {{.Examples}}{{end}}
{{if .Mode}}Mode: {{.Mode}}{{end}}
{{if .Custom}}
Custom Fields:
{{range $key, $value := .Custom}}
- {{$key}}: {{$value}}
{{end}}
{{end}}`

	mainTemplate := `{{template "comprehensive" .}}`

	// Load templates
	if err := engine.LoadTemplate("comprehensive", comprehensivePartial); err != nil {
		t.Fatalf("Failed to load comprehensive partial: %v", err)
	}

	if err := engine.LoadTemplate("main", mainTemplate); err != nil {
		t.Fatalf("Failed to load main template: %v", err)
	}

	// Test with all fields populated
	data := Data{
		Target:        "cursor",
		Name:          "test-project",
		Description:   "Test description",
		Globs:         "**/*.ts",
		ProjectType:   "web",
		Language:      "typescript",
		Framework:     "react",
		Tags:          []string{"frontend", "ui"},
		AlwaysApply:   "true",
		Documentation: "https://docs.example.com",
		StyleGuide:    "https://style.example.com",
		Examples:      "https://examples.example.com",
		Mode:          "command",
		Custom: map[string]interface{}{
			"version": "1.0.0",
			"author":  "test-author",
		},
	}

	result, err := engine.Render("main", data)
	if err != nil {
		t.Fatalf("Failed to render comprehensive template: %v", err)
	}

	// Verify all fields are properly rendered in the partial
	expectedContent := []string{
		"# Test-Project - Cursor",
		"Description: Test description",
		"Globs: **/*.ts",
		"Project Type: Web",
		"Language: Typescript",
		"Framework: React",
		"Tags: frontend, ui",
		"Always Apply: true",
		"Documentation: https://docs.example.com",
		"Style Guide: https://style.example.com",
		"Examples: https://examples.example.com",
		"Mode: command",
		"- version: 1.0.0",
		"- author: test-author",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected to find %q in result\nFull result:\n%s", expected, result)
		}
	}
}

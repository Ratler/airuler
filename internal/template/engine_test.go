package template

import (
	"strings"
	"testing"
)

func TestNewEngine(t *testing.T) {
	engine := NewEngine()
	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}

	if engine.templates == nil {
		t.Error("NewEngine() did not initialize templates map")
	}

	if engine.funcMap == nil {
		t.Error("NewEngine() did not initialize funcMap")
	}
}

func TestLoadTemplate(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name         string
		templateName string
		content      string
		expectError  bool
	}{
		{
			name:         "valid template",
			templateName: "test",
			content:      "Hello {{.Name}}!",
			expectError:  false,
		},
		{
			name:         "template with conditionals",
			templateName: "conditional",
			content:      "{{if eq .Target \"cursor\"}}Cursor{{else}}Other{{end}}",
			expectError:  false,
		},
		{
			name:         "template with functions",
			templateName: "functions",
			content:      "{{upper .Name}} - {{lower .Target}}",
			expectError:  false,
		},
		{
			name:         "invalid template syntax",
			templateName: "invalid",
			content:      "{{.Name",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.LoadTemplate(tt.templateName, tt.content)

			if tt.expectError && err == nil {
				t.Errorf("LoadTemplate() expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("LoadTemplate() unexpected error: %v", err)
			}

			if !tt.expectError {
				if !engine.HasTemplate(tt.templateName) {
					t.Errorf("LoadTemplate() template %s not found after loading", tt.templateName)
				}
			}
		})
	}
}

func TestRender(t *testing.T) {
	engine := NewEngine()

	// Load test templates
	engine.LoadTemplate("simple", "Hello {{.Name}}!")
	engine.LoadTemplate("conditional", "{{if eq .Target \"cursor\"}}Cursor Mode{{else}}Other Mode{{end}}")
	engine.LoadTemplate("functions", "{{upper .Name}} - {{lower .Target}}")

	tests := []struct {
		name         string
		templateName string
		data         TemplateData
		expected     string
		expectError  bool
	}{
		{
			name:         "simple template",
			templateName: "simple",
			data:         TemplateData{Name: "World"},
			expected:     "Hello World!",
			expectError:  false,
		},
		{
			name:         "conditional template - cursor",
			templateName: "conditional",
			data:         TemplateData{Target: "cursor"},
			expected:     "Cursor Mode",
			expectError:  false,
		},
		{
			name:         "conditional template - other",
			templateName: "conditional",
			data:         TemplateData{Target: "claude"},
			expected:     "Other Mode",
			expectError:  false,
		},
		{
			name:         "functions template",
			templateName: "functions",
			data:         TemplateData{Name: "test", Target: "CURSOR"},
			expected:     "TEST - cursor",
			expectError:  false,
		},
		{
			name:         "non-existent template",
			templateName: "nonexistent",
			data:         TemplateData{},
			expected:     "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Render(tt.templateName, tt.data)

			if tt.expectError && err == nil {
				t.Errorf("Render() expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Render() unexpected error: %v", err)
			}

			if !tt.expectError && result != tt.expected {
				t.Errorf("Render() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestHasTemplate(t *testing.T) {
	engine := NewEngine()

	// Initially should not have any templates
	if engine.HasTemplate("test") {
		t.Error("HasTemplate() returned true for non-existent template")
	}

	// Load a template
	engine.LoadTemplate("test", "Hello {{.Name}}!")

	// Now should have the template
	if !engine.HasTemplate("test") {
		t.Error("HasTemplate() returned false for existing template")
	}

	// Should not have other templates
	if engine.HasTemplate("other") {
		t.Error("HasTemplate() returned true for non-existent template 'other'")
	}
}

func TestListTemplates(t *testing.T) {
	engine := NewEngine()

	// Initially should be empty
	templates := engine.ListTemplates()
	if len(templates) != 0 {
		t.Errorf("ListTemplates() returned %d templates, expected 0", len(templates))
	}

	// Load some templates
	templateNames := []string{"template1", "template2", "template3"}
	for _, name := range templateNames {
		engine.LoadTemplate(name, "Content for "+name)
	}

	// Should return all loaded templates
	templates = engine.ListTemplates()
	if len(templates) != len(templateNames) {
		t.Errorf("ListTemplates() returned %d templates, expected %d", len(templates), len(templateNames))
	}

	// Check that all template names are present
	templateMap := make(map[string]bool)
	for _, name := range templates {
		templateMap[name] = true
	}

	for _, expectedName := range templateNames {
		if !templateMap[expectedName] {
			t.Errorf("ListTemplates() missing template: %s", expectedName)
		}
	}
}

func TestTemplateFunctions(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     TemplateData
		expected string
	}{
		{
			name:     "lower function",
			template: "{{lower .Name}}",
			data:     TemplateData{Name: "HELLO"},
			expected: "hello",
		},
		{
			name:     "upper function",
			template: "{{upper .Name}}",
			data:     TemplateData{Name: "hello"},
			expected: "HELLO",
		},
		{
			name:     "title function",
			template: "{{title .Name}}",
			data:     TemplateData{Name: "hello world"},
			expected: "Hello World",
		},
		{
			name:     "contains function",
			template: "{{if contains .Name \"test\"}}Found{{else}}Not found{{end}}",
			data:     TemplateData{Name: "this is a test"},
			expected: "Found",
		},
		{
			name:     "replace function",
			template: "{{replace .Name \"old\" \"new\"}}",
			data:     TemplateData{Name: "old value"},
			expected: "new value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine.LoadTemplate("test", tt.template)
			result, err := engine.Render("test", tt.data)

			if err != nil {
				t.Errorf("Render() unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Render() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestComplexTemplate(t *testing.T) {
	engine := NewEngine()

	complexTemplate := `# {{title .Name}} Rule

{{if eq .Target "cursor"}}---
description: {{.Description}}
globs: {{.Globs}}
alwaysApply: true
---
{{end}}

This is a {{lower .Target}} rule for {{.Name}}.

{{if eq .Target "claude"}}Arguments: $ARGUMENTS{{end}}

## Guidelines
- Use {{upper .Target}} best practices
- {{replace .Description "rule" "guideline"}}`

	data := TemplateData{
		Target:      "cursor",
		Name:        "typescript",
		Description: "TypeScript coding rule",
		Globs:       "**/*.ts",
	}

	engine.LoadTemplate("complex", complexTemplate)
	result, err := engine.Render("complex", data)

	if err != nil {
		t.Fatalf("Render() unexpected error: %v", err)
	}

	// Check that key parts are present
	expectedParts := []string{
		"# Typescript Rule",
		"---",
		"description: TypeScript coding rule",
		"globs: **/*.ts",
		"alwaysApply: true",
		"This is a cursor rule for typescript",
		"Use CURSOR best practices",
		"TypeScript coding guideline",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Render() result missing expected part: %q", part)
		}
	}
}

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package compiler

import (
	"testing"

	"github.com/ratler/airuler/internal/template"
)

func TestClaudeMemoryMode(t *testing.T) {
	compiler := NewCompiler()

	// Load a memory mode template
	templateContent := `---
mode: memory
description: Test memory template
---
# Memory Content
This should go to CLAUDE.md`

	err := compiler.LoadTemplate("test-memory", templateContent)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	data := template.Data{
		Mode: "memory",
	}

	rule, err := compiler.CompileTemplate("test-memory", TargetClaude, data)
	if err != nil {
		t.Fatalf("Failed to compile template: %v", err)
	}

	if rule.Filename != "CLAUDE.md" {
		t.Errorf("Expected filename CLAUDE.md, got %s", rule.Filename)
	}

	if rule.Mode != "memory" {
		t.Errorf("Expected mode memory, got %s", rule.Mode)
	}
}

func TestClaudeCommandMode(t *testing.T) {
	compiler := NewCompiler()

	// Load a command mode template
	templateContent := `---
mode: command
description: Test command template
---
# Command Content
This should go to commands/`

	err := compiler.LoadTemplate("test-command", templateContent)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	data := template.Data{
		Mode: "command",
	}

	rule, err := compiler.CompileTemplate("test-command", TargetClaude, data)
	if err != nil {
		t.Fatalf("Failed to compile template: %v", err)
	}

	if rule.Filename != "test-command.md" {
		t.Errorf("Expected filename test-command.md, got %s", rule.Filename)
	}

	if rule.Mode != "command" {
		t.Errorf("Expected mode command, got %s", rule.Mode)
	}
}

func TestClaudeBothMode(t *testing.T) {
	compiler := NewCompiler()

	// Load a both mode template
	templateContent := `---
mode: both
description: Test both template
---
# Both Content
This should generate both memory and command versions`

	err := compiler.LoadTemplate("test-both", templateContent)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	data := template.Data{
		Mode: "both",
	}

	rules, err := compiler.CompileTemplateWithModes("test-both", TargetClaude, data)
	if err != nil {
		t.Fatalf("Failed to compile template: %v", err)
	}

	if len(rules) != 2 {
		t.Fatalf("Expected 2 rules for both mode, got %d", len(rules))
	}

	// Check that we have both memory and command versions
	foundMemory := false
	foundCommand := false

	for _, rule := range rules {
		if rule.Mode == "memory" && rule.Filename == "CLAUDE.md" {
			foundMemory = true
		}
		if rule.Mode == "command" && rule.Filename == "test-both.md" {
			foundCommand = true
		}
	}

	if !foundMemory {
		t.Error("Expected to find memory mode rule")
	}
	if !foundCommand {
		t.Error("Expected to find command mode rule")
	}
}

func TestClaudeDefaultMode(t *testing.T) {
	compiler := NewCompiler()

	// Load a template with no mode specified
	templateContent := `# Default Content
No mode specified, should default to command`

	err := compiler.LoadTemplate("test-default", templateContent)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	data := template.Data{
		// Mode not set
	}

	rule, err := compiler.CompileTemplate("test-default", TargetClaude, data)
	if err != nil {
		t.Fatalf("Failed to compile template: %v", err)
	}

	if rule.Filename != "test-default.md" {
		t.Errorf("Expected filename test-default.md, got %s", rule.Filename)
	}

	// Mode should default to command
	if rule.Mode != "" && rule.Mode != "command" {
		t.Errorf("Expected mode to be empty or command for default, got %s", rule.Mode)
	}
}

func TestMultipleMemoryModeTemplates(t *testing.T) {
	compiler := NewCompiler()

	// Load multiple memory mode templates
	templates := []struct {
		name    string
		content string
	}{
		{
			name: "standards",
			content: `# Coding Standards
Follow these standards for all code.`,
		},
		{
			name: "architecture",
			content: `# Architecture Guidelines
Use clean architecture principles.`,
		},
		{
			name: "security",
			content: `# Security Best Practices
Always validate input and sanitize output.`,
		},
	}

	var memoryRules []CompiledRule

	for _, tmpl := range templates {
		err := compiler.LoadTemplate(tmpl.name, tmpl.content)
		if err != nil {
			t.Fatalf("Failed to load template %s: %v", tmpl.name, err)
		}

		data := template.Data{
			Mode: "memory",
		}

		rule, err := compiler.CompileTemplate(tmpl.name, TargetClaude, data)
		if err != nil {
			t.Fatalf("Failed to compile template %s: %v", tmpl.name, err)
		}

		if rule.Mode != "memory" {
			t.Errorf("Expected mode memory for %s, got %s", tmpl.name, rule.Mode)
		}

		if rule.Filename != "CLAUDE.md" {
			t.Errorf("Expected filename CLAUDE.md for %s, got %s", tmpl.name, rule.Filename)
		}

		memoryRules = append(memoryRules, rule)
	}

	// Verify all templates compiled to memory mode
	if len(memoryRules) != 3 {
		t.Errorf("Expected 3 memory mode rules, got %d", len(memoryRules))
	}

	// Each rule should have unique content
	contents := make(map[string]bool)
	for _, rule := range memoryRules {
		if contents[rule.Content] {
			t.Error("Duplicate content found in memory rules")
		}
		contents[rule.Content] = true
	}
}

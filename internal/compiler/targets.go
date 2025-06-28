package compiler

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ratler/airuler/internal/template"
)

type Target string

const (
	TargetCursor  Target = "cursor"
	TargetClaude  Target = "claude"
	TargetCline   Target = "cline"
	TargetCopilot Target = "copilot"
)

var AllTargets = []Target{TargetCursor, TargetClaude, TargetCline, TargetCopilot}

type Compiler struct {
	engine *template.Engine
}

func NewCompiler() *Compiler {
	return &Compiler{
		engine: template.NewEngine(),
	}
}

func (c *Compiler) LoadTemplate(name, content string) error {
	return c.engine.LoadTemplate(name, content)
}

func (c *Compiler) CompileTemplate(templateName string, target Target, data template.TemplateData) (CompiledRule, error) {
	// Set target in data
	data.Target = string(target)

	// Render template
	content, err := c.engine.Render(templateName, data)
	if err != nil {
		return CompiledRule{}, err
	}

	// Post-process based on target
	processedContent, filename := c.postProcess(content, templateName, target, data)

	return CompiledRule{
		Target:   target,
		Name:     templateName,
		Filename: filename,
		Content:  processedContent,
		Mode:     data.Mode,
	}, nil
}

func (c *Compiler) CompileTemplateWithModes(templateName string, target Target, data template.TemplateData) ([]CompiledRule, error) {
	// For "both" mode with Claude target, generate both memory and command versions
	if target == TargetClaude && data.Mode == "both" {
		var rules []CompiledRule

		// Generate memory version
		memoryData := data
		memoryData.Mode = "memory"
		memoryRule, err := c.CompileTemplate(templateName, target, memoryData)
		if err != nil {
			return nil, err
		}
		rules = append(rules, memoryRule)

		// Generate command version
		commandData := data
		commandData.Mode = "command"
		commandRule, err := c.CompileTemplate(templateName, target, commandData)
		if err != nil {
			return nil, err
		}
		rules = append(rules, commandRule)

		return rules, nil
	}

	// For all other cases, generate single rule
	rule, err := c.CompileTemplate(templateName, target, data)
	if err != nil {
		return nil, err
	}

	return []CompiledRule{rule}, nil
}

func (c *Compiler) postProcess(content, templateName string, target Target, data template.TemplateData) (string, string) {
	switch target {
	case TargetCursor:
		return c.processCursor(content, templateName, data)
	case TargetClaude:
		return c.processClaude(content, templateName, data)
	case TargetCline:
		return c.processCline(content, templateName, data)
	case TargetCopilot:
		return c.processCopilot(content, templateName, data)
	default:
		return content, templateName + ".txt"
	}
}

func (c *Compiler) processCursor(content, templateName string, data template.TemplateData) (string, string) {
	// Cursor expects .mdc files with YAML front matter
	filename := filepath.Base(templateName) + ".mdc"

	// If content doesn't start with front matter, ensure it has proper structure
	if !strings.HasPrefix(content, "---") {
		frontMatter := fmt.Sprintf(`---
description: %s
globs: %s
alwaysApply: true
---

`, getDescription(data, templateName), getGlobs(data))
		content = frontMatter + content
	}

	return content, filename
}

func (c *Compiler) processClaude(content, templateName string, data template.TemplateData) (string, string) {
	// Determine installation mode (default to "command")
	mode := data.Mode
	if mode == "" {
		mode = "command"
	}

	switch mode {
	case "memory":
		// For memory mode, output as CLAUDE.md
		return content, "CLAUDE.md"
	case "both":
		// For both mode, we'll handle this in the compiler by creating two outputs
		// For now, default to command mode in this function
		fallthrough
	case "command":
		fallthrough
	default:
		// Command mode - individual .md files in .claude/commands/
		filename := filepath.Base(templateName) + ".md"
		return content, filename
	}
}

func (c *Compiler) processCline(content, templateName string, data template.TemplateData) (string, string) {
	// Cline uses .md files in .clinerules/ directory
	filename := filepath.Base(templateName) + ".md"

	return content, filename
}

func (c *Compiler) processCopilot(content, templateName string, data template.TemplateData) (string, string) {
	// GitHub Copilot expects .instructions.md files with optional YAML front matter
	filename := filepath.Base(templateName) + ".instructions.md"

	// Ensure proper front matter format for Copilot
	if !strings.HasPrefix(content, "---") {
		frontMatter := fmt.Sprintf(`---
description: %s
applyTo: %s
---

`, getDescription(data, templateName), getGlobs(data))
		content = frontMatter + content
	}

	return content, filename
}

func getDescription(data template.TemplateData, fallback string) string {
	if data.Description != "" {
		return data.Description
	}
	return fmt.Sprintf("AI coding rules for %s", fallback)
}

func getGlobs(data template.TemplateData) string {
	if data.Globs != "" {
		return data.Globs
	}
	return "**/*"
}

type CompiledRule struct {
	Target   Target
	Name     string
	Filename string
	Content  string
	Mode     string
}

func (c *Compiler) GetOutputPath(target Target, filename string) string {
	return filepath.Join("compiled", string(target), filename)
}

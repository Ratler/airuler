// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package template

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
)

type Engine struct {
	templates map[string]*template.Template
	funcMap   template.FuncMap
}

type Data struct {
	Target      string
	Name        string
	Description string
	Globs       string

	// Extended fields for advanced templates
	ProjectType   string
	Language      string
	Framework     string
	Tags          []string
	AlwaysApply   string
	Documentation string
	StyleGuide    string
	Examples      string

	// Installation mode for Claude Code
	Mode string // "memory", "command", "both"

	// Custom fields map for additional data
	Custom map[string]interface{}
}

// toTitle replaces the deprecated strings.Title function
// It capitalizes the first letter of each word in the string
func toTitle(s string) string {
	prev := ' '
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) && !unicode.IsLetter(prev) {
			prev = r
			return unicode.ToTitle(r)
		}
		prev = r
		return r
	}, s)
}

func NewEngine() *Engine {
	funcMap := template.FuncMap{
		"lower":    strings.ToLower,
		"upper":    strings.ToUpper,
		"title":    toTitle,
		"join":     strings.Join,
		"contains": strings.Contains,
		"replace":  strings.ReplaceAll,
	}

	return &Engine{
		templates: make(map[string]*template.Template),
		funcMap:   funcMap,
	}
}

func (e *Engine) LoadTemplate(name, content string) error {
	// Create a new template with the name
	tmpl := template.New(name).Funcs(e.funcMap)

	// Load all existing templates as associated templates for partials
	for templateName, existingTmpl := range e.templates {
		if templateName != name && existingTmpl.Root != nil {
			// Add existing template as an associated template
			if _, err := tmpl.New(templateName).Parse(existingTmpl.Root.String()); err != nil {
				return fmt.Errorf("failed to parse associated template %s: %w", templateName, err)
			}
		}
	}

	// Parse the main template content
	tmpl, err := tmpl.Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	e.templates[name] = tmpl

	// Update all existing templates to include this new template
	e.updateTemplateReferences()

	return nil
}

func (e *Engine) updateTemplateReferences() {
	// Create a map of all template contents
	templateContents := make(map[string]string)
	for name, tmpl := range e.templates {
		if tmpl.Root != nil {
			templateContents[name] = tmpl.Root.String()
		}
	}

	// Rebuild all templates with all partials available
	newTemplates := make(map[string]*template.Template)
	for name, content := range templateContents {
		tmpl := template.New(name).Funcs(e.funcMap)

		// Add all other templates as associated templates
		for otherName, otherContent := range templateContents {
			if otherName != name {
				if _, err := tmpl.New(otherName).Parse(otherContent); err != nil {
					// Skip this template if it fails to parse, but continue with others
					continue
				}
			}
		}

		// Parse the main template
		if parsedTmpl, err := tmpl.Parse(content); err == nil {
			newTemplates[name] = parsedTmpl
		}
	}

	e.templates = newTemplates
}

func (e *Engine) LoadTemplateFile(path string) error {
	content, err := readFile(path)
	if err != nil {
		return err
	}

	name := filepath.Base(path)
	name = strings.TrimSuffix(name, filepath.Ext(name))

	return e.LoadTemplate(name, content)
}

func (e *Engine) Render(templateName string, data Data) (string, error) {
	tmpl, exists := e.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return buf.String(), nil
}

func (e *Engine) HasTemplate(name string) bool {
	_, exists := e.templates[name]
	return exists
}

func (e *Engine) ListTemplates() []string {
	var names []string
	for name := range e.templates {
		names = append(names, name)
	}
	return names
}

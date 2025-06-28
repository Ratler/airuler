# airuler Template Examples

This directory contains comprehensive examples of airuler templates demonstrating various features and patterns.

## Structure

```
examples/
├── basic/              # Simple template examples
├── advanced/           # Advanced templates with conditionals and partials
├── real-world/         # Real-world rule examples
└── partials/           # Reusable template components
```

## Quick Start

To try these examples:

1. Copy the example templates to your `templates/` directory
2. Run `airuler compile` to generate rules for all targets
3. Check the `compiled/` directory for the generated files

## Template Features Demonstrated

### Basic Templates
- Simple rule structure
- Target-specific content
- Variable substitution

### Advanced Templates  
- Conditional logic based on target
- Partial inclusion and composition
- Template functions and helpers
- Complex front matter handling

### Real-World Examples
- TypeScript/JavaScript coding rules
- Python development guidelines
- React component patterns
- API development standards

### Partials
- Reusable headers and footers
- Common rule sections
- Target-specific components
- Shared configuration blocks

## Usage Patterns

### Simple Rule
```yaml
# Basic rule template
{{template "header" .}}

## Rule Content
Your rule content here.

{{template "footer" .}}
```

### Conditional Content
```yaml
{{if eq .Target "cursor"}}
Cursor-specific content
{{else if eq .Target "claude"}}
Claude-specific content  
{{end}}
```

### Partial Inclusion
```yaml
{{template "shared/common-guidelines" .}}
{{template "shared/error-handling" .}}
```

### Custom Variables
```yaml
Rule: {{.Name}}
Project: {{.ProjectType}}
Language: {{.Language}}
```

For more detailed examples, explore the subdirectories.
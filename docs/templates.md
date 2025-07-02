# Template Syntax

Templates use Go's `text/template` syntax with custom functions and YAML front matter for metadata.

## Template Front Matter

Templates use YAML front matter to define metadata and variables:

```yaml
---
# Core front matter fields (always available)
description: "Project coding standards"     # → {{.Description}}
globs: "**/*.ts,**/*.js"                    # → {{.Globs}}
claude_mode: memory                         # → {{.Mode}} (command/memory/both)

# Extended front matter fields (optional)
project_type: "web-application"             # → {{.ProjectType}}
language: "TypeScript"                      # → {{.Language}}
framework: "React"                          # → {{.Framework}}
tags:                                       # → {{.Tags}} (array)
  - "frontend"
  - "spa"
  - "typescript"
always_apply: "true"                        # → {{.AlwaysApply}}
documentation: "docs/frontend.md"           # → {{.Documentation}}
style_guide: "Airbnb JavaScript"            # → {{.StyleGuide}}
examples: "examples/react/"                 # → {{.Examples}}
custom:                                     # → {{.Custom}} (map)
  build_tool: "Vite"                        # → {{.Custom.build_tool}}
  testing_framework: "Jest"                 # → {{.Custom.testing_framework}}
  version: "18.2.0"                         # → {{.Custom.version}}
---
```

## Template Variables

Variables are populated from three sources:

### 1. System Variables (Always Available)
- `{{.Target}}` - Current compilation target (cursor, claude, cline, copilot, roo)
- `{{.Name}}` - Template filename without extension (e.g., "my-rules" from "my-rules.tmpl")

### 2. Front Matter Variables (From YAML Header)
Basic fields:
- `{{.Description}}` - From `description:` field (defaults to "AI coding rules for {{.Name}}")
- `{{.Globs}}` - From `globs:` field (defaults to "**/*")
- `{{.Mode}}` - From `claude_mode:` field (for Claude Code only)

Extended fields (all optional):
- `{{.ProjectType}}` - From `project_type:` field
- `{{.Language}}` - From `language:` field
- `{{.Framework}}` - From `framework:` field
- `{{.Tags}}` - From `tags:` field (array)
- `{{.AlwaysApply}}` - From `always_apply:` field
- `{{.Documentation}}` - From `documentation:` field
- `{{.StyleGuide}}` - From `style_guide:` field
- `{{.Examples}}` - From `examples:` field
- `{{.Custom}}` - From `custom:` field (map for arbitrary key-value pairs)

### 3. Usage Example

Template with front matter (`templates/react-standards.tmpl`):
```yaml
---
description: "React TypeScript coding standards"
globs: "**/*.{ts,tsx,js,jsx}"
claude_mode: both
project_type: "web-application"
language: "TypeScript"
framework: "React"
tags: ["frontend", "spa", "typescript"]
documentation: "docs/react.md"
custom:
  build_tool: "Vite"
  min_node_version: "18.0.0"
---

# {{.Language}} {{.Framework}} Standards

You're working on a {{.ProjectType}} using {{.Framework}}.

## File Patterns
These rules apply to: {{.Globs}}

## Technology Stack
- Language: {{.Language}}
- Framework: {{.Framework}}
- Build Tool: {{.Custom.build_tool}}
- Min Node Version: {{.Custom.min_node_version}}

## Tags
{{range .Tags}}- {{.}}
{{end}}

{{if .Documentation}}
See full documentation: {{.Documentation}}
{{end}}
```

## Conditionals

Target-specific content:
```go
{{if eq .Target "cursor"}}
Cursor-specific content
{{else if eq .Target "claude"}}
Claude-specific content  
{{end}}

{{if contains .Tags "frontend"}}
Frontend-specific guidelines
{{end}}
```

## Template Functions

- `{{lower .Name}}` - Convert to lowercase
- `{{upper .Name}}` - Convert to uppercase
- `{{title .Name}}` - Convert to title case
- `{{join .Tags ", "}}` - Join array with separator
- `{{contains .Tags "web"}}` - Check if array contains value
- `{{replace .Name "old" "new"}}` - Replace text

## Partials and Template Inheritance

Include reusable components using partials. airuler supports two ways to organize partials:

### Traditional Partials Directory
Place `.tmpl` files in any `partials/` subdirectory:

```
templates/
└── partials/
    ├── header.tmpl
    └── footer.tmpl
```

### Flexible .ptmpl Files
Use `.ptmpl` extension for partials that can be organized anywhere:

```
templates/
├── components/
│   ├── auth.ptmpl
│   └── ui/
│       └── button.ptmpl
├── layouts/
│   └── base.ptmpl
└── shared.ptmpl
```

### Using Partials

Main template (`templates/main.tmpl`):
```go
{{template "partials/header" .}}              <!-- Local traditional partial -->
{{template "components/auth" .}}              <!-- Local .ptmpl from components/ -->
{{template "layouts/base" .}}                 <!-- Local .ptmpl from layouts/ -->

# Main content here

{{template "partials/footer" .}}
```

Partial file example (`templates/components/security.ptmpl`):
```yaml
---
description: "Security checklist component"
---
### Security Checklist
{{if eq .ProjectType "api"}}
- [ ] Authentication implemented
- [ ] Rate limiting configured
- [ ] Input validation on all endpoints
{{else}}
- [ ] General security practices followed
{{end}}
```

### Important Notes

- `.ptmpl` files are always treated as partials and never compiled as main templates
- Partials are referenced by their path relative to templates directory without extension
- **Always include the dot (`.`) parameter** when calling templates: `{{template "components/auth" .}}`
  - The dot passes the current data context (variables like `.Language`, `.Target`, etc.) to the partial
  - Without the dot, partials won't have access to template variables and conditionals will fail
- All template variables from the main template are available in partials when the dot is included
- Both `.tmpl` files in `partials/` directories and `.ptmpl` files anywhere are treated as partials
- Partials can have YAML front matter, but it's stripped during compilation

### Template and Partial Isolation
- **Local templates** can only access **local partials**
- **Vendor templates** can only access **partials from the same vendor**
- This ensures complete isolation and prevents naming conflicts
- Each template compiles independently with its own set of available partials

## Claude Code Installation Modes

Claude Code supports different installation modes to match its dual system:

### Memory Mode (Persistent Instructions)

Memory mode rules are installed as `CLAUDE.md` and automatically loaded as persistent project context:

```yaml
---
claude_mode: memory
description: Project architecture and coding standards
---
# Project Architecture

This is a {{.ProjectType}} project using {{.Framework}}.

## Architecture Overview
- Follow component-based architecture
- Use TypeScript for type safety
- Implement proper error boundaries

## Coding Standards
- All functions must be properly typed
- Use descriptive variable names
- Write unit tests for all business logic

These guidelines apply to all code in this project.
```

**Installation**: Creates/appends to `CLAUDE.md` in project root

### Command Mode (On-Demand Commands)

Command mode rules are installed in `.claude/commands/` and invoked using slash commands:

```yaml
---
claude_mode: command
description: Refactor a function to improve performance
arguments: function_name
---
# Refactor Function

Refactor the function `$ARGUMENTS` with the following approach:

## Analysis Steps
1. **Identify bottlenecks**: Look for inefficient loops, unnecessary operations
2. **Check algorithms**: Consider better algorithmic approaches
3. **Memory usage**: Optimize memory allocations and data structures

## Refactoring Process
1. Show the current function implementation
2. Identify specific performance issues
3. Provide optimized version with explanations
4. Highlight performance improvements made

## Testing
- Ensure functionality remains identical
- Add performance benchmarks if applicable
- Update existing tests as needed
```

**Usage**: Invoke with `/refactor-function myFunctionName`

### Both Mode (Dual Generation)

Both mode generates two versions from a single template:

```yaml
---
claude_mode: both
description: Security best practices for {{.Language}} development
---
# Security Guidelines

## Input Validation
- Sanitize all user inputs
- Use parameterized queries for database operations
- Validate data types and ranges

## Authentication & Authorization
- Implement proper session management
- Use secure password hashing
- Apply principle of least privilege

## Data Protection
- Encrypt sensitive data at rest
- Use HTTPS for all communications
- Implement proper key management

## Error Handling
- Don't leak sensitive information in error messages
- Log security events appropriately
- Implement proper error boundaries

Apply these security practices consistently across the codebase.
```

**Result**: Creates both `CLAUDE.md` (persistent context) and `.claude/commands/security-guidelines.md` (on-demand command)

## Advanced Template Examples

### Multi-Framework Template

```yaml
---
description: "Framework-specific coding standards"
globs: "**/*.{js,ts,jsx,tsx,vue}"
project_type: "web-application"
framework: "react"
language: "javascript"
---
# {{title .Framework}} Coding Standards for {{.ProjectType}}

{{if eq .Framework "react"}}
## React Best Practices
- Use functional components with hooks
- Implement proper prop validation
- Follow React component lifecycle
{{else if eq .Framework "vue"}}
## Vue.js Best Practices
- Use composition API for complex logic
- Implement proper reactive data patterns
- Follow Vue component conventions
{{else if eq .Framework "angular"}}
## Angular Best Practices
- Use Angular CLI for project structure
- Implement proper dependency injection
- Follow Angular style guide
{{end}}

## Universal Principles
- Write semantic, accessible HTML
- Optimize for performance
- Implement proper error handling
```

### Complex Template with Partials

Main template (`templates/comprehensive-guide.tmpl`):
```yaml
---
claude_mode: memory
description: Comprehensive development guide
---
{{template "header" .}}

{{template "coding-standards" .}}

{{template "security-guidelines" .}}

{{template "testing-requirements" .}}

{{template "footer" .}}
```

Header partial (`templates/partials/header.tmpl`):
```yaml
---
description: Common header with project info
---
# {{.Name}} - Development Guide
Project: {{.ProjectType}} | Framework: {{.Framework}} | Target: {{.Target}}

Generated on: {{/* Add date functionality if needed */}}
```

### TypeScript React Template

```yaml
---
claude_mode: both
description: TypeScript React development standards
project_type: web
language: typescript
framework: react
tags: [frontend, web, typescript, react]
globs: "**/*.{ts,tsx,js,jsx}"
---
# TypeScript React Development Standards

## Component Structure
```typescript
// Use functional components with proper typing
interface Props {
  title: string;
  children: React.ReactNode;
  onClick?: () => void;
}

export const MyComponent: React.FC<Props> = ({ title, children, onClick }) => {
  return (
    <div className="my-component">
      <h2>{title}</h2>
      {children}
      {onClick && <button onClick={onClick}>Action</button>}
    </div>
  );
};
```

## State Management
- Use `useState` for local state
- Use `useReducer` for complex state logic
- Implement proper TypeScript interfaces for state

## Testing Requirements
- Write unit tests for all components
- Use React Testing Library
- Test user interactions and edge cases
- Maintain >80% test coverage

{{if eq .Target "claude"}}
When working with this React TypeScript codebase:
1. Always provide proper TypeScript types
2. Suggest performance optimizations when applicable
3. Ensure accessibility best practices
4. Follow React hooks rules and conventions
{{end}}
```
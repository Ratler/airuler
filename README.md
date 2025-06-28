# airuler - AI Rules Template Engine

A Go-based CLI tool that compiles AI rule templates into target-specific formats for various AI coding assistants including Cursor, Claude Code, Cline/Roo, and GitHub Copilot.

## Features

- 🎯 **Multi-target compilation**: Generate rules for Cursor, Claude Code, Cline/Roo, and GitHub Copilot
- 📦 **Vendor management**: Fetch and manage rule templates from Git repositories  
- 🔄 **Template inheritance**: Support for base templates and partials
- 💾 **Safe installation**: Automatic backup of existing rules
- 🔍 **Watch mode**: Auto-compile templates during development
- ⚙️ **Flexible configuration**: YAML-based configuration with lock files
- 🧠 **Claude Code modes**: Memory (persistent) and command (on-demand) installation modes
- 📝 **YAML front matter**: Rich template metadata and configuration

## Quick Start

### Installation

```bash
# Clone and build
git clone https://github.com/ratler/airuler
cd airuler
go build -o airuler
```

### Initialize a new project

```bash
airuler init
```

This creates the following structure and optionally initializes a git repository:
```
rules/
├── templates/          # Your rule templates
│   ├── partials/      # Reusable components
│   └── examples/      # Sample templates
├── vendors/           # External rule repositories  
├── compiled/          # Generated output files
│   ├── cursor/        # Cursor .mdc files
│   ├── claude/        # Claude .md files (memory & commands)
│   ├── cline/         # Cline .md files
│   └── copilot/       # Copilot .instructions.md files
├── airuler.yaml       # Configuration
├── airuler.lock      # Dependency lock file
└── .gitignore         # Git ignore patterns
```

During initialization, airuler will:
- Create the project directory structure
- Generate a default configuration file
- Create a `.gitignore` with sensible defaults
- Optionally initialize a git repository with an initial commit
- Create an example template to get you started

### Create your first template

Add a template file to `templates/my-coding-rules.tmpl`:

```yaml
---
claude_mode: memory
description: Project coding standards for {{.ProjectType}}
globs: "**/*"
---
# {{.Name}} Coding Standards

{{if eq .Target "claude"}}
This document outlines coding standards for this project. These rules are automatically loaded by Claude Code.
{{else}}
This rule applies to {{.Target}} and helps maintain code quality.
{{end}}

## Core Principles
- Write clean, readable code
- Follow language-specific conventions
- Include comprehensive tests
- Document complex logic

{{if eq .Target "cursor"}}
## Cursor Specific
- Use TypeScript for type safety
- Leverage VSCode extensions effectively
{{else if eq .Target "claude"}}
## When reviewing or writing code:
1. Check for adherence to these standards
2. Suggest improvements when standards aren't met
3. Explain the reasoning behind recommendations
{{end}}

## Error Handling
- Always handle errors explicitly
- Use appropriate error types
- Log errors with sufficient context

Remember: These standards apply to all code in this project.
```

### Compile templates

```bash
# Compile for all targets
airuler compile

# Compile for specific target
airuler compile claude

# Compile specific rule
airuler compile --rule my-coding-rules

# Compile from vendor
airuler compile --vendor my-vendor
```

### Install rules

```bash
# Install to global AI agent configs
airuler install

# Install to project directory
airuler install --project ./my-project

# Install specific target
airuler install claude

# Install specific rule
airuler install claude my-coding-rules
```

## Target Support

### Cursor
- **Format**: `.mdc` files with YAML front matter
- **Location**: `.cursor/rules/` (project) or global Cursor config
- **Features**: Supports `description`, `globs`, `alwaysApply`

### Claude Code 🆕
- **Format**: Plain `.md` files with `$ARGUMENTS` placeholder
- **Location**: 
  - **Command mode**: `.claude/commands/` (project) or `~/.claude/commands/` (global)
  - **Memory mode**: `CLAUDE.md` in project root or `~/CLAUDE.md` (global)
- **Features**: 
  - Simple markdown with argument substitution
  - **Installation modes**: `command`, `memory`, or `both`
  - Command mode: On-demand invocation via slash commands
  - Memory mode: Persistent project instructions (automatically loaded)
  - Automatic content appending for memory mode

### Cline/Roo
- **Format**: `.md` files 
- **Location**: `.clinerules/` (project) or `~/.clinerules/`
- **Features**: Plain markdown rules

### GitHub Copilot
- **Format**: `.instructions.md` files with optional YAML front matter
- **Location**: `.github/instructions/` (project) or `~/.github/instructions/`
- **Features**: Supports `description`, `applyTo` globs

## Template Syntax

Templates use Go's `text/template` syntax with custom functions and YAML front matter.

### Template Front Matter 🆕

Templates can include YAML front matter to specify metadata:

```yaml
---
claude_mode: memory       # For Claude Code: "command", "memory", or "both"
description: Custom description
globs: "**/*.ts,**/*.js"
arguments: function_name
projecttype: typescript
language: typescript
framework: react
tags: [frontend, web]
---
```

### Variables
- `{{.Target}}` - Current target (cursor, claude, cline, copilot)
- `{{.Name}}` - Template name
- `{{.Description}}` - Rule description
- `{{.Globs}}` - File glob patterns
- `{{.Arguments}}` - Arguments placeholder
- `{{.Mode}}` - Installation mode (for Claude Code)
- `{{.ProjectType}}` - Project type (from front matter)
- `{{.Language}}` - Programming language
- `{{.Framework}}` - Framework being used
- `{{.Tags}}` - Array of tags

### Conditionals
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

### Functions
- `{{lower .Name}}` - Convert to lowercase
- `{{upper .Name}}` - Convert to uppercase
- `{{title .Name}}` - Convert to title case
- `{{join .Tags ", "}}` - Join array with separator
- `{{contains .Tags "web"}}` - Check if array contains value
- `{{replace .Name "old" "new"}}` - Replace text

### Partials and Template Inheritance

Include reusable components using partials:

```go
{{template "header" .}}

# Main content here

{{template "footer" .}}
```

Create partials in `templates/partials/header.tmpl`:
```yaml
---
description: Common header for all rules
---
# {{.Name}}
Generated for {{.Target}} on {{.ProjectType}} project
```

## Claude Code Installation Modes 🆕

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

## Vendor Management

Fetch and manage external rule repositories to share templates across projects.

### Fetch external rule repositories

```bash
# Fetch from Git repository
airuler fetch https://github.com/company/frontend-rules

# Fetch with custom alias
airuler fetch https://github.com/company/backend-rules --as backend

# Update existing vendor
airuler fetch https://github.com/company/frontend-rules --update
```

### Update vendors

```bash
# Update all vendors
airuler update

# Update specific vendor
airuler update backend

# Check for updates without fetching
airuler update --dry-run
```

### Manage vendors

```bash
# List all vendors
airuler vendors list

# Check vendor status
airuler vendors status

# Remove vendor
airuler vendors remove backend
```

### Compile with vendors

```bash
# Compile including all enabled vendors
airuler compile

# Compile from specific vendor
airuler compile --vendor backend

# Compile from multiple vendors
airuler compile --vendors "backend,frontend"
```

## Configuration

airuler supports both project-specific and global configuration files:

### Configuration Precedence
1. `--config` flag (if specified)
2. `./airuler.yaml` (project-specific config)
3. Global config:
   - Linux/macOS: `~/.config/airuler/airuler.yaml`
   - Windows: `%APPDATA%\airuler\airuler.yaml`

### Managing Global Configuration
```bash
# Initialize global config
airuler config init

# Show config file locations
airuler config path

# Edit global config (uses $EDITOR environment variable)
airuler config edit
```

### airuler.yaml Configuration

```yaml
# Vendor repositories
vendors:
  - url: https://github.com/company/frontend-rules
    alias: frontend
    enabled: true
    auto_update: true
  - url: https://github.com/company/security-rules
    alias: security
    enabled: true
    auto_update: false

# Default settings
defaults:
  include_vendors: [frontend, security]
  modes:
    claude: command  # Default mode for Claude Code

# Update settings
update:
  auto_recompile: true
  check_frequency: daily
  confirm_updates: false
```

## Commands Reference

### Core Commands
```bash
airuler init                          # Initialize project structure
airuler compile [target]              # Compile templates
airuler install [target] [rule]       # Install compiled rules
airuler watch                         # Watch mode for development
```

### Compilation Options
```bash
airuler compile                       # Compile for all targets
airuler compile claude                # Compile for Claude Code only
airuler compile --rule my-rule        # Compile specific rule
airuler compile --vendor frontend     # Compile from vendor
airuler compile --vendors "fe,be"     # Compile from multiple vendors
```

### Installation Options
```bash
airuler install                       # Install all rules globally
airuler install --project ./app       # Install to project directory
airuler install claude                # Install Claude rules only
airuler install claude my-rule        # Install specific Claude rule
airuler install --force               # Overwrite without backup
```

### Configuration Commands
```bash
airuler config init                   # Initialize global configuration
airuler config path                   # Show configuration file paths
airuler config edit                   # Edit global configuration
```

### Vendor Management Commands
```bash
airuler fetch <url>                   # Fetch external repository
airuler fetch <url> --as <alias>      # Fetch with custom alias
airuler fetch <url> --update          # Update existing vendor
airuler update [vendor...]            # Update vendors
airuler update --dry-run              # Check for updates only
airuler vendors list                  # List vendors
airuler vendors status                # Show vendor status
airuler vendors remove <vendor>       # Remove vendor
```

### Global Options
```bash
--config <file>                       # Use specific config file
--verbose                             # Enable verbose output
--help                                # Show help information
```

## Advanced Examples

### Multi-Framework Template

```yaml
---
description: Framework-specific coding standards
projecttype: "{{.ProjectType}}"
framework: "{{.Framework}}"
---
# {{title .Framework}} Coding Standards

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
projecttype: web
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

## Development Workflow

### Local Development

1. **Initialize project**:
   ```bash
   airuler init
   ```

2. **Create templates**:
   - Add templates to `templates/`
   - Use partials for reusable components
   - Include YAML front matter for metadata

3. **Develop with watch mode**:
   ```bash
   airuler watch
   ```

4. **Test compilation**:
   ```bash
   airuler compile claude --rule my-rule
   ```

5. **Install and test**:
   ```bash
   airuler install claude --project ./test-project
   ```

### Team Collaboration

1. **Share vendor repositories**:
   ```bash
   # Team member adds shared rules
   airuler fetch https://github.com/team/coding-standards --as team-standards
   ```

2. **Update configuration**:
   ```yaml
   # airuler.yaml
   defaults:
     include_vendors: [team-standards]
   ```

3. **Sync updates**:
   ```bash
   airuler update
   airuler compile
   ```

### CI/CD Integration

```yaml
# .github/workflows/airuler.yml
name: Update AI Rules
on:
  push:
    paths: ['templates/**']
    
jobs:
  update-rules:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
      - name: Build airuler
        run: go build -o airuler
      - name: Compile rules
        run: ./airuler compile
      - name: Install to project
        run: ./airuler install --project .
```

## Best Practices

### Template Organization
- Use descriptive template names
- Group related templates in subdirectories
- Create reusable partials for common content
- Include comprehensive front matter metadata

### Claude Code Modes
- **Memory mode**: Use for project-wide standards, architecture guidelines, persistent context
- **Command mode**: Use for specific tasks, refactoring commands, analysis tools
- **Both mode**: Use for comprehensive guidelines that need both persistent and on-demand access

### Version Control
- Commit `airuler.yaml` and `airuler.lock` files
- Version control your templates
- Use `.gitignore` for `compiled/` directory if desired
- Document your template structure in project README

### Performance
- Use `--rule` flag to compile specific templates during development
- Leverage watch mode for rapid iteration
- Consider template complexity and compilation time

## Troubleshooting

### Common Issues

**Template compilation errors**:
```bash
# Check template syntax
airuler compile --rule problem-template --verbose
```

**Installation path issues**:
```bash
# Show where rules would be installed
airuler config path
```

**Vendor sync problems**:
```bash
# Check vendor status
airuler vendors status

# Force update
airuler fetch https://github.com/vendor/repo --update
```

**YAML front matter parsing**:
- Ensure proper YAML syntax
- Check indentation (use spaces, not tabs)
- Validate YAML online if needed

## License

MIT License

---

*For more examples and advanced usage, see the `examples/` directory in the repository.*
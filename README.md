# airuler - AI Rules Template Engine

A Go-based CLI tool that compiles AI rule templates into target-specific formats for various AI coding assistants including Cursor, Claude Code, Cline, GitHub Copilot, and Roo Code.

## Features

- 🎯 **Multi-target compilation**: Generate rules for Cursor, Claude Code, Cline, GitHub Copilot, and Roo Code
- 📦 **Vendor management**: Fetch and manage rule templates from Git repositories  
- 🔄 **Template inheritance**: Support for template partials (.tmpl in partials/ dirs and .ptmpl files anywhere)
- 💾 **Safe installation**: Automatic backup of existing rules and installation tracking
- 🔍 **Watch mode**: Auto-compile templates during development
- ⚙️ **Flexible configuration**: YAML-based configuration with lock files
- 🧠 **Claude Code modes**: Memory (persistent) and command (on-demand) installation modes
- 📝 **YAML front matter**: Rich template metadata and configuration
- 🌍 **Global template directory**: Run commands from anywhere - airuler remembers your last template directory

## Installation

### Pre-built Binaries

Download the latest release for your platform from the [GitHub releases page](https://github.com/ratler/airuler/releases).

```bash
# Extract and move to your PATH
tar -xzf airuler_*_linux_amd64.tar.gz
sudo mv airuler /usr/local/bin/
```

### Docker

```bash
# Pull the latest image
docker pull ratler/airuler:latest

# Or run directly
docker run --rm -v $(pwd):/workspace ratler/airuler:latest version
```

### Build from Source

```bash
# Clone and build
git clone https://github.com/ratler/airuler
cd airuler
go build -o airuler
```

## Docker Usage

### Basic Usage

```bash
# Run airuler commands with current directory mounted
docker run --rm -v $(pwd):/workspace ratler/airuler:latest [command]

# Initialize a new project
docker run --rm -v $(pwd):/workspace ratler/airuler:latest init

# Compile templates
docker run --rm -v $(pwd):/workspace ratler/airuler:latest compile

# Install rules to project
docker run --rm -v $(pwd):/workspace ratler/airuler:latest install --project .
```

### Using docker-compose

```bash
# Compile templates
docker-compose run --rm compile

# Watch for changes during development
docker-compose run --rm watch

# Run any airuler command
docker-compose run --rm airuler init
```

## Quick Start

### Initialize a new project

```bash
airuler init
```

This creates the following structure and optionally initializes a git repository:
```
rules/
├── templates/         # Your rule templates
│   ├── partials/      # Reusable components
│   └── examples/      # Sample templates
├── vendors/           # External rule repositories  
├── compiled/          # Generated output files
│   ├── cursor/        # Cursor .mdc files
│   ├── claude/        # Claude .md files (memory & commands)
│   ├── cline/         # Cline .md files
│   ├── copilot/       # Copilot .instructions.md files
│   └── roo/           # Roo Code .md files
├── airuler.yaml       # Configuration
├── airuler.lock       # Dependency lock file
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
description: "Project coding standards"
globs: "**/*"
project_type: "web-application"
language: "TypeScript"
---
# {{.Name}} Coding Standards

This document outlines coding standards for our {{.ProjectType}} project using {{.Language}}.

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

### Cline
- **Format**: `.md` files 
- **Location**: `.clinerules/` (project) or `~/.clinerules/`
- **Features**: Plain markdown rules

### GitHub Copilot
- **Format**: Combined into single `.github/copilot-instructions.md` file
- **Location**: `.github/` (project only - no global installation)
- **Features**: Plain markdown compilation, project-only installation

### Roo Code
- **Format**: Plain `.md` files
- **Location**: `.roo/rules/` (project) or `~/.roo/rules/` (global)
- **Features**: Plain markdown rules, supports directory-based organization

## Template Syntax

Templates use Go's `text/template` syntax with custom functions and YAML front matter.

### Template Front Matter

Templates use YAML front matter to define metadata and variables that can be used in the template content:

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

### Template Variables

Variables are populated from three sources:

#### 1. System Variables (Always Available)
- `{{.Target}}` - Current compilation target (cursor, claude, cline, copilot, roo)
- `{{.Name}}` - Template filename without extension (e.g., "my-rules" from "my-rules.tmpl")

#### 2. Front Matter Variables (From YAML Header)
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

#### 3. Usage Example

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

Include reusable components using partials. Airuler supports two ways to organize partials:

#### Traditional Partials Directory
Place `.tmpl` files in any `partials/` subdirectory:

```
templates/
└── partials/
    ├── header.tmpl
    └── footer.tmpl
```

#### Flexible .ptmpl Files 🆕
Use `.ptmpl` extension for partials that can be organized anywhere in your template structure:

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

#### Using Partials

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

**Important Notes**:
- `.ptmpl` files are always treated as partials and never compiled as main templates
- Partials are referenced by their path relative to templates directory without extension
- **Always include the dot (`.`) parameter** when calling templates: `{{template "components/auth" .}}`
  - The dot passes the current data context (variables like `.Language`, `.Target`, etc.) to the partial
  - Without the dot, partials won't have access to template variables and conditionals will fail
- All template variables from the main template are available in partials when the dot is included
- Both `.tmpl` files in `partials/` directories and `.ptmpl` files anywhere are treated as partials
- Partials can have YAML front matter, but it's stripped during compilation

#### Template and Partial Isolation 🆕
- **Local templates** can only access **local partials**
- **Vendor templates** can only access **partials from the same vendor**
- This ensures complete isolation and prevents naming conflicts
- Each template compiles independently with its own set of available partials

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
# Default settings
defaults:
  include_vendors: ["*"]  # Include all vendors by default
  # Or specify specific vendors:
  # include_vendors: [frontend, security]
  last_template_dir: "/path/to/templates"  # Auto-managed template directory
```

## Global Template Directory 🆕

airuler automatically remembers the last template directory you compiled from, allowing you to run commands from anywhere without needing to be in the template directory.

### How It Works

1. **Automatic Detection**: When you run `airuler compile` (or any command) from a template directory, airuler automatically saves that directory as your default template directory.

2. **Seamless Operation**: When you run airuler commands from outside a template directory, airuler automatically operates as if you're in the last template directory, while keeping you in your current shell location.

3. **User Feedback**: When operating from a remembered template directory, airuler shows: `Using template directory: /path/to/templates`

### Usage Examples

```bash
# Compile from template directory (saves as default)
cd /path/to/my-templates
airuler compile

# Later, run commands from anywhere
cd /completely/different/directory
airuler compile        # Uses /path/to/my-templates internally
airuler install        # Uses /path/to/my-templates internally
airuler watch          # Uses /path/to/my-templates internally
```

### Manual Configuration

Set the template directory manually:

```bash
# Set specific template directory
airuler config set-template-dir /path/to/my-templates

# Set relative to current directory
airuler config set-template-dir ./templates

# The command validates that the path is a valid template directory
# (contains both 'templates/' directory and 'airuler.lock' file)
```

### Template Directory Detection

A directory is considered a valid template directory if it contains:
- `templates/` directory (with your template files)
- `airuler.lock` file (dependency lock file)

### Error Handling

If the remembered template directory is no longer valid:

```bash
$ airuler compile
Error: Last template directory '/old/path/templates' is no longer a valid airuler template directory
Please run 'airuler config set-template-dir <path>' to set a new template directory
```

### Benefits

- **Convenience**: Run airuler commands from anywhere in your filesystem
- **Workflow Integration**: Fits naturally into development workflows where you might be in different directories
- **Project Flexibility**: Switch between different template repositories seamlessly
- **Backward Compatibility**: Existing workflows continue to work unchanged

## Installation Tracking 🛡️

airuler automatically tracks where templates have been installed, enabling safe updates, clean uninstalls, and comprehensive management of your AI rules across different targets and projects.

### How Installation Tracking Works

When you install templates using `airuler install`, airuler automatically:

1. **Records Installation Details**: Tracks the target, rule name, installation location, mode, and timestamp
2. **Maintains Installation Database**: Stores tracking information in global and project-specific databases
3. **Enables Safe Operations**: Allows for clean uninstalls and selective updates

### Installation Database Locations

- **Global installations**: `~/.config/airuler/installations.yaml` (Linux/macOS) or `%APPDATA%\airuler\installations.yaml` (Windows)
- **Project installations**: `./.airuler/installations.yaml` in each project directory

### Key Benefits

- **Clean Uninstalls**: Remove only files that airuler installed, never accidentally delete user files
- **Selective Updates**: Update specific rules or targets without affecting others
- **Installation History**: See what was installed when and where
- **Safe Overwrites**: Automatic backups before overwriting existing files

### Viewing Installed Templates

```bash
# List all installed templates
airuler list-installed

# Filter by keyword
airuler list-installed --filter cursor

# Show global installations only
airuler list-installed --global

# Show project installations only  
airuler list-installed --project
```

Example output:
```
🌍 Global Installations
==============================================================================
Target   Rule                 Mode     File                      Installed      
------------------------------------------------------------------------------
cursor   coding-standards     normal   coding-standards.mdc      2 hours ago
claude   security-guide       memory   CLAUDE.md                 1 day ago
claude   refactor-helper      command  refactor-helper.md        1 day ago

📁 Project Installations (/path/to/project)
==============================================================================
Target   Rule                 Mode     File                      Installed
------------------------------------------------------------------------------
cursor   project-rules        normal   project-rules.mdc         3 hours ago
```

### Updating Installed Templates

```bash
# Update all tracked installations
airuler update-installed

# Update only global installations
airuler update-installed --global

# Update only project installations
airuler update-installed --project
```

The update process:
1. Recompiles templates from source
2. Compares with installed versions
3. Updates only if content has changed
4. Creates backups before overwriting
5. Updates tracking database with new timestamps

### Uninstalling Templates

```bash
# Uninstall all tracked installations (with confirmation)
airuler uninstall

# Uninstall specific target
airuler uninstall claude

# Uninstall specific rule
airuler uninstall claude security-guide

# Interactive selection mode
airuler uninstall --interactive

# Force uninstall without prompts
airuler uninstall --force
```

### Installation Modes and Tracking

airuler tracks different installation types:

- **Global installations**: Rules installed to AI tool global configurations
- **Project installations**: Rules installed to specific project directories  
- **Memory mode (Claude)**: Content appended to CLAUDE.md files
- **Command mode (Claude)**: Individual command files in .claude/commands/

### Manual Installation Management

If you need to manually manage the installation database:

```bash
# View installation tracker files
airuler config path  # Shows config directory locations

# The installation database files are:
# ~/.config/airuler/installations.yaml (global)
# ./.airuler/installations.yaml (project-specific)
```

### Safety Features

- **Backup Creation**: Original files are backed up before overwriting (e.g., `rule.md.backup.20240102-143022`)
- **Collision Detection**: Warns when installing would overwrite non-airuler files
- **Selective Removal**: Only removes files that airuler installed
- **Version Tracking**: Maintains installation history and timestamps

## Commands Reference

### Core Commands
```bash
airuler init [path]                   # Initialize project structure
airuler compile [target]              # Compile templates
airuler install [target] [rule]       # Install compiled rules
airuler update-installed              # Update all tracked installations
airuler uninstall [target] [rule]     # Uninstall tracked installations
airuler watch                         # Watch mode for development
airuler version                       # Show version information
```

### Compilation Options
```bash
airuler compile                       # Compile for all targets
airuler compile claude                # Compile for Claude Code only
airuler compile --rule my-rule        # Compile specific rule (short: -r)
airuler compile --vendor frontend     # Compile from vendor (short: -v)
airuler compile --vendors "fe,be"     # Compile from multiple vendors
```

### Installation Options
```bash
airuler install                       # Install all rules globally
airuler install --project ./app       # Install to project directory (short: -p)
airuler install --global              # Install globally (short: -g, default)
airuler install claude                # Install Claude rules only
airuler install claude my-rule        # Install specific Claude rule
airuler install --force               # Overwrite without backup (short: -f)
airuler install --interactive         # Interactive selection mode (short: -i)
```

### Uninstallation Options
```bash
airuler uninstall                     # Uninstall all tracked installations
airuler uninstall --global            # Uninstall only global installations (short: -g)
airuler uninstall --project           # Uninstall only project installations (short: -p)
airuler uninstall claude              # Uninstall Claude rules only
airuler uninstall claude my-rule      # Uninstall specific Claude rule
airuler uninstall --force             # Skip confirmation prompts (short: -f)
airuler uninstall --interactive       # Interactive selection mode (short: -i)
```

### Configuration Commands
```bash
airuler config init                     # Initialize global configuration
airuler config path                     # Show configuration file paths
airuler config edit                     # Edit global configuration
airuler config set-template-dir <path>  # Set default template directory
```

### Vendor Management Commands
```bash
airuler fetch <url>                   # Fetch external repository
airuler fetch <url> --as <alias>      # Fetch with custom alias (short: -a)
airuler fetch <url> --update          # Update existing vendor (short: -u)
airuler update [vendor...]            # Update vendors
airuler update --dry-run              # Check for updates only (short: -d)
airuler update --interactive          # Interactive update mode (short: -i)
airuler update-installed              # Update all tracked installations
airuler update-installed --global     # Update only global installations (short: -g)
airuler update-installed --project    # Update only project installations (short: -p)
airuler vendors list                  # List vendors
airuler vendors status                # Show vendor status
airuler vendors check                 # Check for vendor updates
airuler vendors remove <vendor>       # Remove vendor
airuler vendors include <vendor>      # Include vendor in compilation
airuler vendors exclude <vendor>      # Exclude vendor from compilation
airuler vendors include-all           # Include all vendors
airuler vendors exclude-all           # Exclude all vendors
```

### Global Options
```bash
--config <file>                       # Use specific config file
--verbose                             # Enable verbose output
--help                                # Show help information
-h                                    # Show help information (short)
```

### Additional Commands
```bash
airuler list-installed                     # List all installed templates
airuler list-installed --filter <keyword>  # Filter templates by keyword (short: -f)
```

## Advanced Examples

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
   airuler update-installed
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

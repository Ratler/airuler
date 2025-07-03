# airuler - AI Rules Template Engine

A Go-based CLI tool that compiles AI rule templates into target-specific formats for various AI coding assistants
including Cursor, Claude Code, Cline, GitHub Copilot, Gemini CLI, and Roo Code.

## The Problem

**Stop duplicating your AI coding rules across multiple tools.**

If you're using Cursor, Claude Code, Cline, GitHub Copilot, Gemini CLI, and Roo Code, you know the pain: maintaining the same coding
standards and project rules across completely different file formats and locations.

airuler solves this by letting you write your rules once as templates, then automatically:

- **Generate** the correct format for each AI assistant
- **Install** rules to the right locations (global configs or project directories)
- **Update** all installations instantly when templates change

## Features

- 🎯 **Multi-target compilation**: Generate rules for Cursor, Claude Code, Cline, GitHub Copilot, Gemini CLI, and Roo Code
- 📦 **Vendor management**: Fetch and manage rule templates from Git repositories
- 🔄 **Template inheritance**: Support for template partials and reusable components
- 💾 **Safe installation**: Automatic backup of existing rules and installation tracking
- 🔍 **Watch mode**: Auto-compile templates during development
- ⚙️ **Flexible configuration**: YAML-based configuration with vendor-specific settings
- 🧠 **Claude Code modes**: Memory (persistent) and command (on-demand) installation modes
- 🎛️ **Vendor configuration**: Per-vendor defaults, variables, and compilation settings

## Installation

### Pre-built Binaries

Download the latest release for your platform from the [GitHub releases page](https://github.com/ratler/airuler/releases).

```bash
# Extract and move to your PATH
tar -xzf airuler_*_linux_amd64.tar.gz
sudo mv airuler_*_linux_adm64/airuler /usr/local/bin/
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
git clone https://github.com/ratler/airuler
cd airuler
go build -o airuler
```

## Quick Start

### 1. Initialize a new project

```bash
airuler init
```

This creates a project structure with `templates/`, `compiled/`, `vendors/` directories and configuration files.

### 2. Create your first template

Create `templates/my-coding-rules.tmpl`:

```yaml
---
claude_mode: memory
description: "Project coding standards"
globs: "**/*"
language: "TypeScript"
---
# {{.Name}} Coding Standards

This document outlines coding standards for our {{.Language}} project.

## Core Principles
- Write clean, readable code
- Follow language-specific conventions
- Include comprehensive tests
- Document complex logic

{{if eq .Target "claude"}}
When reviewing or writing code:
1. Check for adherence to these standards
2. Suggest improvements when standards aren't met
3. Explain the reasoning behind recommendations
{{end}}
```

### 3. Deploy rules (compile + install)

```bash
# Deploy to global AI agent configs
airuler deploy

# Deploy to specific project directory
airuler deploy --project ./my-project

# Deploy for specific target
airuler deploy cursor

# Interactive template selection
airuler deploy --interactive
```

### 4. Sync workflow (update vendors + compile + deploy)

```bash
# Full sync: update vendors → compile → deploy
airuler sync

# Sync for specific target only
airuler sync claude

# Skip vendor updates
airuler sync --no-update
```

## Target Support

| Target             | Format       | Location                             | Features                                            |
| ------------------ | ------------ | ------------------------------------ | --------------------------------------------------- |
| **Cursor**         | `.mdc` files | `.cursor/rules/`                     | YAML front matter, globs, alwaysApply               |
| **Claude Code**    | `.md` files  | `.claude/commands/` or `CLAUDE.md`   | Memory/command modes, `$ARGUMENTS` placeholder      |
| **Cline**          | `.md` files  | `.clinerules/`                       | Plain markdown rules                                |
| **GitHub Copilot** | `.md` files  | `.github/copilot-instructions.md`    | Combined into single file                           |
| **Gemini CLI**     | `.md` files  | `~/.gemini/GEMINI.md` or `GEMINI.md` | Combined into single file, global & project support |
| **Roo Code**       | `.md` files  | `.roo/rules/`                        | Plain markdown rules                                |

## Key Commands

```bash
# Project Setup
airuler init                    # Initialize project
airuler config init             # Initialize global config

# Core Workflow
airuler deploy                  # Compile templates and install fresh
airuler deploy --interactive    # Interactive template selection
airuler sync                    # Update vendors + compile + deploy
airuler watch                   # Development mode with auto-compile

# Management
airuler manage                  # Interactive management hub
airuler manage installations    # View installed templates
airuler manage uninstall        # Remove installed templates
airuler manage uninstall --all  # Remove all installations

# Vendor Management
airuler vendors add <url>       # Add external template repository
airuler vendors list            # List vendors with config details
airuler vendors update          # Update vendor repositories
airuler vendors include <name>  # Include vendor in compilation
airuler vendors remove <name>   # Remove vendor

# Configuration
airuler config path             # Show config locations
airuler config set-template-dir # Set default template directory
```

## Documentation

For detailed information, see:

- **[Command Reference](docs/command-reference.md)** - Complete command and flag reference
- **[Template Syntax](docs/templates.md)** - Template variables, functions, partials, and Claude Code modes
- **[Vendor Management](docs/vendors.md)** - Adding and managing external rule repositories
- **[Configuration](docs/configuration.md)** - YAML configuration, global settings, and template directories
- **[Installation Management](docs/installation.md)** - Installation tracking, updates, and uninstallation
- **[Examples & Best Practices](docs/examples.md)** - Advanced examples and development workflows

## Development

```bash
# Build and test
make build
make test

# Template development with auto-reload
airuler watch

# Format and lint
make fmt
make lint
```

## License

MIT License

______________________________________________________________________

*airuler helps you maintain consistent AI coding assistant rules across different tools and projects through a unified template system.*

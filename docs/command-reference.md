# Command Reference

Complete reference for all airuler commands, flags, and options.

## Global Flags

These flags are available for all commands:

| Flag        | Short | Description      | Default                                         |
| ----------- | ----- | ---------------- | ----------------------------------------------- |
| `--config`  |       | Config file path | Project dir or `~/.config/airuler/airuler.yaml` |
| `--verbose` | `-v`  | Verbose output   | `false`                                         |
| `--help`    | `-h`  | Help for command |                                                 |

## Core Commands

### `airuler init [path]`

Initialize a new airuler project with the modern directory structure.

**Usage:**

```bash
airuler init                    # Initialize in current directory
airuler init my-rules-project   # Create and initialize new directory
airuler init ../other-project   # Initialize in relative path
```

**Arguments:**

- `path` (optional): Directory to initialize (creates if doesn't exist)

**Flags:** None

______________________________________________________________________

### `airuler deploy [target] [rule]`

Compile templates and install to new locations. This replaces the workflow: compile → install.

**Usage:**

```bash
airuler deploy                         # Deploy globally for all targets
airuler deploy cursor                  # Deploy only for Cursor target globally
airuler deploy cursor my-rule          # Deploy specific rule for Cursor globally
airuler deploy --project ./my-app      # Deploy to specific project directory
airuler deploy --interactive           # Interactive template selection
airuler deploy --targets cursor,claude # Deploy only to specific targets
airuler deploy --dry-run               # Show what would be deployed
```

**Arguments:**

- `target` (optional): Specific target to deploy (cursor, claude, cline, copilot, gemini, roo)
- `rule` (optional): Specific rule/template to deploy

**Flags:**

| Flag            | Short | Type   | Description                                                  | Default |
| --------------- | ----- | ------ | ------------------------------------------------------------ | ------- |
| `--no-compile`  |       | bool   | Skip template compilation, use existing compiled rules       | `false` |
| `--project`     | `-p`  | string | Deploy to specific project directory (sets scope to project) |         |
| `--targets`     | `-t`  | string | Comma-separated list of targets (e.g., cursor,claude)        |         |
| `--interactive` | `-i`  | bool   | Interactive template selection                               | `false` |
| `--force`       | `-f`  | bool   | Overwrite existing files without confirmation                | `false` |
| `--dry-run`     | `-n`  | bool   | Show what would be deployed without executing                | `false` |

______________________________________________________________________

### `airuler sync [target]`

Sync template repository, vendors, compile templates, and update installations. This replaces the workflow: git pull → update → compile → update-installed.

**Usage:**

```bash
airuler sync                      # Full sync: git pull → update vendors → compile → deploy
airuler sync cursor               # Sync only for Cursor target
airuler sync --no-update          # Skip git pull and vendor updates (compile → deploy only)
airuler sync --no-git-pull        # Skip git pull only (update vendors → compile → deploy)
airuler sync --no-compile         # Skip compilation (git pull → update vendors → deploy existing)
airuler sync --no-deploy          # Skip deployment (git pull → update vendors → compile only)
airuler sync --scope project      # Sync only project installations
airuler sync --targets cursor,claude  # Sync only specific targets
airuler sync --dry-run            # Show what would happen without doing it
```

**Arguments:**

- `target` (optional): Specific target to sync

**Flags:**

| Flag            | Short | Type   | Description                                           | Default |
| --------------- | ----- | ------ | ----------------------------------------------------- | ------- |
| `--no-update`   |       | bool   | Skip vendor updates (also skips git pull)             | `false` |
| `--no-git-pull` |       | bool   | Skip git pull of template repository                  | `false` |
| `--no-compile`  |       | bool   | Skip template compilation                             | `false` |
| `--no-deploy`   |       | bool   | Skip deployment to installations                      | `false` |
| `--scope`       | `-s`  | string | Installation scope: global, project, or all           | `all`   |
| `--targets`     | `-t`  | string | Comma-separated list of targets (e.g., cursor,claude) |         |
| `--dry-run`     | `-n`  | bool   | Show what would happen without executing              | `false` |
| `--force`       | `-f`  | bool   | Skip confirmation prompts                             | `false` |

______________________________________________________________________

### `airuler watch`

Watch template files for changes and automatically recompile when they change.

**Usage:**

```bash
airuler watch    # Start watching templates directory
```

**Arguments:** None
**Flags:** None

______________________________________________________________________

## Management Commands

### `airuler manage [subcommand]`

Interactive management hub for airuler operations.

**Usage:**

```bash
airuler manage                    # Main management interface
airuler manage vendors            # Vendor-specific management
airuler manage installations      # Installation-specific management
airuler manage uninstall          # Interactive uninstallation
airuler manage uninstall --all    # Uninstall all installations without prompts
airuler manage --clean            # Clean and rebuild everything
```

**Subcommands:**

- `vendors` - Interactive vendor management
- `installations` - Interactive installation management
- `uninstall` - Interactive uninstallation of rules

**Flags:**

| Flag      | Short | Type | Description                                                                               | Default |
| --------- | ----- | ---- | ----------------------------------------------------------------------------------------- | ------- |
| `--clean` | `-c`  | bool | Clean and rebuild everything                                                              | `false` |
| `--all`   | `-a`  | bool | Uninstall all installations without interactive prompts (use with 'uninstall' subcommand) | `false` |

______________________________________________________________________

## Vendor Management Commands

### `airuler vendors [subcommand]`

Manage vendor repositories.

**Subcommands:**

#### `airuler vendors list [vendor]`

List vendors with repository and configuration details.

**Usage:**

```bash
airuler vendors list              # List all vendors with summaries
airuler vendors list my-rules     # Show detailed config for my-rules vendor
```

**Arguments:**

- `vendor` (optional): Specific vendor to show detailed configuration

**Flags:** None

#### `airuler vendors add <git-url>`

Add a new vendor repository from a Git URL.

**Usage:**

```bash
airuler vendors add https://github.com/user/rules-repo
airuler vendors add https://github.com/user/rules-repo --as my-rules
airuler vendors add https://github.com/user/rules-repo --update
```

**Arguments:**

- `git-url` (required): Git repository URL to add

**Flags:**

| Flag       | Short | Type   | Description                     | Default |
| ---------- | ----- | ------ | ------------------------------- | ------- |
| `--as`     | `-a`  | string | Alias for the vendor            |         |
| `--update` | `-u`  | bool   | Update if vendor already exists | `false` |

#### `airuler vendors update [vendor...]`

Update vendor repositories to their latest versions.

**Usage:**

```bash
airuler vendors update              # Update all vendors
airuler vendors update my-rules     # Update specific vendor
airuler vendors update frontend,backend # Update multiple vendors
```

**Arguments:**

- `vendor...` (optional): Specific vendors to update (comma-separated)

**Flags:** None

#### `airuler vendors status`

Show status of all vendors.

**Usage:**

```bash
airuler vendors status
```

**Arguments:** None
**Flags:** None

#### `airuler vendors check`

Check for updates without fetching them.

**Usage:**

```bash
airuler vendors check
```

**Arguments:** None
**Flags:** None

#### `airuler vendors remove <vendor>`

Remove a vendor repository from the vendors directory.

**Usage:**

```bash
airuler vendors remove backend
```

**Arguments:**

- `vendor` (required): Vendor name to remove

**Flags:** None

#### `airuler vendors include <vendor>`

Include a vendor in compilation.

**Usage:**

```bash
airuler vendors include frontend
```

**Arguments:**

- `vendor` (required): Vendor name to include

**Flags:** None

#### `airuler vendors exclude <vendor>`

Exclude a vendor from compilation.

**Usage:**

```bash
airuler vendors exclude backend
```

**Arguments:**

- `vendor` (required): Vendor name to exclude

**Flags:** None

#### `airuler vendors include-all`

Include all vendors in compilation.

**Usage:**

```bash
airuler vendors include-all
```

**Arguments:** None
**Flags:** None

#### `airuler vendors exclude-all`

Exclude all vendors from compilation (local templates only).

**Usage:**

```bash
airuler vendors exclude-all
```

**Arguments:** None
**Flags:** None

______________________________________________________________________

## Configuration Commands

### `airuler config [subcommand]`

Manage global airuler configuration.

**Subcommands:**

#### `airuler config init`

Initialize global configuration.

**Usage:**

```bash
airuler config init
```

**Arguments:** None
**Flags:** None

#### `airuler config path`

Show configuration file paths.

**Usage:**

```bash
airuler config path
```

**Arguments:** None
**Flags:** None

#### `airuler config edit`

Open global config for editing.

**Usage:**

```bash
airuler config edit
```

**Arguments:** None
**Flags:** None

#### `airuler config set-template-dir <path>`

Set the default template directory.

**Usage:**

```bash
airuler config set-template-dir /path/to/templates
```

**Arguments:**

- `path` (required): Path to template directory

**Flags:** None

______________________________________________________________________

## Utility Commands

### `airuler version`

Display airuler version, build commit, and build date information.

**Usage:**

```bash
airuler version
```

**Arguments:** None
**Flags:** None

### `airuler completion`

Generate the autocompletion script for the specified shell.

**Usage:**

```bash
airuler completion bash
airuler completion zsh
airuler completion fish
airuler completion powershell
```

**Arguments:**

- `shell` (required): Shell type (bash, zsh, fish, powershell)

**Flags:** Varies by shell

### `airuler help [command]`

Help about any command.

**Usage:**

```bash
airuler help
airuler help deploy
airuler help vendors add
```

**Arguments:**

- `command` (optional): Command to get help for

**Flags:** None

______________________________________________________________________

## Command Workflows

### Fresh Installation Workflow

```bash
airuler init                    # Initialize project
airuler vendors add <url>       # Add external templates (optional)
airuler deploy                  # Deploy templates to AI tools
```

### Update Workflow

```bash
airuler sync                    # Git pull → update vendors → compile → deploy
```

### Development Workflow

```bash
airuler watch                   # Auto-compile during development
```

### Management Workflow

```bash
airuler manage                  # Interactive management hub
airuler manage installations    # View installed templates
airuler manage uninstall        # Remove unwanted installations
```

### Vendor Management Workflow

```bash
airuler vendors add <url>       # Add vendor
airuler vendors list            # View vendors
airuler vendors update          # Update vendors
airuler vendors remove <name>   # Remove vendor
```

______________________________________________________________________

## Target Support

### Available Targets

| Target    | Description    | Output Format | Output Location                      |
| --------- | -------------- | ------------- | ------------------------------------ |
| `cursor`  | Cursor IDE     | `.mdc` files  | `.cursor/rules/`                     |
| `claude`  | Claude Code    | `.md` files   | `.claude/commands/` or `CLAUDE.md`   |
| `cline`   | Cline          | `.md` files   | `.clinerules/`                       |
| `copilot` | GitHub Copilot | `.md` files   | `.github/copilot-instructions.md`    |
| `gemini`  | Gemini CLI     | `.md` files   | `~/.gemini/GEMINI.md` or `GEMINI.md` |
| `roo`     | Roo Code       | `.md` files   | `.roo/rules/`                        |

### Target-Specific Features

- **Claude Code**: Supports memory/command modes, `$ARGUMENTS` placeholder
- **Copilot**: Merges all rules into single file
- **Gemini**: Merges all rules into single file, supports global & project scope
- **Cursor**: YAML front matter, globs, alwaysApply
- **Cline, Roo**: Plain markdown rules

______________________________________________________________________

## Exit Codes

| Code | Description   |
| ---- | ------------- |
| `0`  | Success       |
| `1`  | General error |

______________________________________________________________________

## See Also

- [Template Syntax](templates.md) - Template variables, functions, and partials
- [Vendor Management](vendors.md) - External template repositories
- [Configuration](configuration.md) - Project and global configuration
- [Installation Management](installation.md) - Installation tracking and management
- [Examples](examples.md) - Usage examples and best practices

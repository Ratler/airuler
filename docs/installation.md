# Installation Management

airuler automatically tracks where templates have been installed, enabling safe updates, clean uninstalls, and comprehensive management of your AI rules across different targets and projects.

## How Installation Tracking Works

When you deploy templates using `airuler deploy`, airuler automatically:

1. **Records Installation Details**: Tracks the target, rule name, installation location, mode, and timestamp
1. **Maintains Installation Database**: Stores tracking information in a single global database
1. **Enables Safe Operations**: Allows for clean uninstalls and selective updates

## Installation Database

### Database Location

- **Installation database**: `~/.config/airuler/airuler.installs` (Linux/macOS) or `%APPDATA%\airuler\airuler.installs` (Windows)

All installations (both global and project-specific) are tracked in this single global database file. Project installations are distinguished by the `global: false` flag and the `project_path` field.

### Database Structure

```yaml
installations:
  - target: "claude"
    rule: "coding-standards"
    global: false
    project_path: "/path/to/project"
    mode: "memory"
    installed_at: "2024-01-15T10:30:00Z"
    file_path: "/path/to/project/CLAUDE.md"
  
  - target: "cursor"
    rule: "security-guide"
    global: true
    project_path: ""
    mode: "normal"
    installed_at: "2024-01-15T11:00:00Z"
    file_path: "/home/user/.cursor/rules/security-guide.mdc"
```

## Key Benefits

- **Clean Uninstalls**: Remove only files that airuler installed, never accidentally delete user files
- **Selective Updates**: Update specific rules or targets without affecting others
- **Installation History**: See what was installed when and where
- **Safe Overwrites**: Automatic backups before overwriting existing files
- **Cross-Platform**: Works consistently across Linux, macOS, and Windows

## Viewing Installed Templates

### Management Commands

```bash
# Interactive management interface with installation overview
airuler manage

# View detailed installation management
airuler manage installations
```

### Example Output

```
🌍 Global Installations
==============================================================================
Target   Rule                 Mode     File                      Installed      
------------------------------------------------------------------------------
cursor   coding-standards     normal   coding-standards.mdc      2 hours ago
claude   security-guide       memory   CLAUDE.md                 1 day ago
claude   refactor-helper      command  refactor-helper.md        1 day ago
gemini   project-standards    -        GEMINI.md                 3 hours ago

📁 Project Installations (/path/to/project)
==============================================================================
Target   Rule                 Mode     File                      Installed
------------------------------------------------------------------------------
cursor   project-rules        normal   project-rules.mdc         3 hours ago
gemini   local-guidelines     -        GEMINI.md                 2 hours ago
```

### Management Features

The `manage installations` command provides:

- List of all installed templates organized by scope (global/project)
- Installation status and file existence verification
- Interactive access to uninstall options

## Updating Installed Templates

### Sync Command

The primary way to update installed templates is through the sync workflow:

```bash
# Full sync: update vendors → compile → update existing installations
airuler sync

# Update only existing installations (skip vendor updates)
airuler sync --no-update

# Update specific target installations only
airuler sync cursor

# Update with specific scope
airuler sync --scope global    # Only global installations
airuler sync --scope project   # Only project installations
```

### Update Process

The update process:

1. **Recompiles templates** from current source
1. **Compares content** with installed versions using hash comparison
1. **Updates only if changed** to avoid unnecessary file modifications
1. **Creates backups** of target files before overwriting (e.g., `rule.md.backup.20240115-143022`)
1. **Updates tracking database** with new timestamps and content hashes

### Sync Options

| Flag           | Short | Description                      | Example                   |
| -------------- | ----- | -------------------------------- | ------------------------- |
| `--no-update`  |       | Skip vendor updates              | `--no-update`             |
| `--no-compile` |       | Skip template compilation        | `--no-compile`            |
| `--no-deploy`  |       | Skip deployment to installations | `--no-deploy`             |
| `--scope`      | `-s`  | Limit to specific scope          | `--scope global`          |
| `--targets`    | `-t`  | Update specific targets only     | `--targets cursor,claude` |
| `--dry-run`    | `-n`  | Show what would happen           | `--dry-run`               |
| `--force`      | `-f`  | Skip confirmation prompts        | `--force`                 |

## Uninstalling Templates

### Uninstall Commands

```bash
# Interactive uninstallation interface
airuler manage uninstall

# Uninstall all installations without prompts
airuler manage uninstall --all
```

### Uninstall Process

1. **Identifies tracked files** from installation database
1. **Prompts for confirmation** (unless using `--force`)
1. **Removes only airuler-installed files** (never removes user files)
1. **Updates installation database** to remove uninstalled entries

Note: Backups of target files created during installation are not automatically restored. They remain in the target directories with `.backup.timestamp` suffixes.

### Uninstall Options

| Flag    | Short | Description                                             |
| ------- | ----- | ------------------------------------------------------- |
| `--all` | `-a`  | Uninstall all installations without interactive prompts |

The `manage uninstall` command provides:

- **Interactive mode** (default): Choose specific installations to remove
- **Bulk mode** (`--all`): Remove all installations with single confirmation
- **Safety checks**: Only removes airuler-tracked installations
- **Confirmation prompts**: Prevents accidental deletions

## Installation Modes and Tracking

airuler tracks different installation types:

### Installation Types

- **Global installations**: Rules installed to AI tool global configurations
- **Project installations**: Rules installed to specific project directories
- **Memory mode (Claude)**: Content appended to CLAUDE.md files
- **Command mode (Claude)**: Individual command files in .claude/commands/
- **Merged files (Copilot, Gemini)**: Multiple rules combined into single files

### Mode-Specific Behavior

**Memory Mode**:

- Tracks content appended to CLAUDE.md
- Updates only the airuler-managed sections
- Preserves user-added content

**Command Mode**:

- Tracks individual command files
- Can update/remove specific commands
- Maintains command isolation

**Normal Mode** (Cursor, Cline, Roo):

- Tracks complete file installations
- Can safely overwrite managed files
- Creates backups before changes

**Merged Mode** (Copilot, Gemini):

- Combines multiple rules into single files (copilot-instructions.md, GEMINI.md)
- Uses reinstall strategy for partial updates
- Maintains rule separation with "---" dividers

## Safety Features

### Backup Creation

- **Automatic backups**: Target rule files are backed up before overwriting (not the installation database)
- **Timestamped names**: `rule.md.backup.20240102-143022`
- **Backup location**: Same directory as the target file
- **Skip with --force**: The `--force` flag skips backup creation

### Collision Detection

```bash
# Warning when installing would overwrite non-airuler files
Warning: File 'CLAUDE.md' exists and was not installed by airuler
Create backup and proceed? [y/N]
```

### Selective Removal

- **Database verification**: Only removes files tracked in installation database
- **Hash verification**: Confirms file content before removal
- **User file protection**: Never removes files not installed by airuler

### Version Tracking

- **Content hashing**: SHA-256 hash of installed content
- **Timestamp tracking**: Installation and update times
- **Change detection**: Updates only when content actually changes

## Manual Installation Management

### Direct Database Access

If you need to manually inspect or modify the installation database:

```bash
# View installation tracker files
airuler config path  # Shows config directory locations

# Database files are located at:
# ~/.config/airuler/airuler.installs (global)
```

### Database Management

The installation database is automatically managed by airuler. If you encounter issues with corrupted or missing installation records, you can:

1. Check the database file location using `airuler config path`
1. Manually inspect the YAML file if needed
1. Remove corrupted entries by editing the file directly
1. Reinstall templates to recreate tracking records

## Advanced Installation Scenarios

### CI/CD Integration

```yaml
# .github/workflows/update-rules.yml
name: Update AI Rules
on:
  schedule:
    - cron: '0 9 * * 1'  # Weekly on Monday

jobs:
  update-rules:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Update installed rules
        run: |
          airuler sync --force
          # Commit updated rules if changed
```

### Multi-Environment Management

```bash
# Development environment
airuler deploy --project ./dev-project

# Staging environment  
airuler deploy --project ./staging-project

# Production environment
airuler deploy --project ./prod-project

# View all environments
airuler manage installations
```

### Batch Operations

```bash
# Update all Claude installations
airuler sync claude

# Sync specific targets only
airuler sync --targets cursor,claude

# Deploy to specific project
airuler deploy --project ./specific-project
```

## Troubleshooting

### Common Issues

**Installation database corruption**:

```bash
# Check database location
airuler config path

# Inspect database file manually
cat ~/.config/airuler/airuler.installs
```

**Missing backup files**:

```bash
# Check backup locations
ls -la *.backup.*

# Reinstall to recreate tracking
airuler deploy --force
```

**Permission issues**:

```bash
# Check file permissions
ls -la ~/.config/airuler/

# Fix permissions
chmod 644 ~/.config/airuler/airuler.installs
chmod 755 ~/.config/airuler/
```

**Path resolution problems**:

```bash
# Debug path resolution
airuler manage installations

# Use absolute paths
airuler deploy --project "$(pwd)/project"
```

### Debugging Commands

```bash
# View installation details
airuler manage installations

# Deploy with dry run to check
airuler deploy --dry-run

# Force reinstall to fix tracking
airuler deploy --force
```

### Recovery Procedures

**Lost installation database**:

1. The database file is not automatically backed up
1. Recreate by reinstalling: `airuler deploy --force`
1. Check config path: `airuler config path`

**Corrupted installations**:

1. Manually backup current database: `cp ~/.config/airuler/airuler.installs ~/.config/airuler/airuler.installs.backup`
1. Edit file manually or remove corrupted entries
1. Verify results: `airuler manage installations`

## Best Practices

### Installation Workflow

1. **Test first**: Use `--project` flag to test in isolated directory
1. **Review changes**: Use `manage installations` to verify installations
1. **Regular updates**: Set up automated sync schedules
1. **Backup management**: Periodically clean old backup files

### Database Maintenance

- **Regular review**: Run `manage installations` to review current installations
- **Cleanup orphans**: Remove entries for deleted projects manually
- **Version control**: Include project `.airuler/` in git (if applicable)
- **Backup database**: Manually backup the installation database periodically

## See Also

- [Configuration](configuration.md) - Global and project configuration
- [Templates](templates.md) - Template structure and compilation
- [Vendors](vendors.md) - Managing external template repositories

# Installation Management

airuler automatically tracks where templates have been installed, enabling safe updates, clean uninstalls, and comprehensive management of your AI rules across different targets and projects.

## How Installation Tracking Works

When you install templates using `airuler install`, airuler automatically:

1. **Records Installation Details**: Tracks the target, rule name, installation location, mode, and timestamp
2. **Maintains Installation Database**: Stores tracking information in a single global database
3. **Enables Safe Operations**: Allows for clean uninstalls and selective updates

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

### List Commands

```bash
# List all installed templates
airuler list-installed

# Filter by keyword
airuler list-installed --filter cursor
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

### Filter Options

| Flag                 | Description                     | Example           |
|----------------------|---------------------------------|-------------------|
| `--filter <keyword>` | Filter by target, rule, or file | `--filter claude` |

## Updating Installed Templates

### Update Commands

```bash
# Update all tracked installations
airuler update-installed

# Update only global installations
airuler update-installed --global

# Update only project installations
airuler update-installed --project

# Update specific target
airuler update-installed claude

# Update specific rule
airuler update-installed claude coding-standards
```

### Update Process

The update process:
1. **Recompiles templates** from current source
2. **Compares content** with installed versions using hash comparison
3. **Updates only if changed** to avoid unnecessary file modifications
4. **Creates backups** of target files before overwriting (e.g., `rule.md.backup.20240115-143022`)
5. **Updates tracking database** with new timestamps and content hashes

### Update Options

| Flag | Description | Example |
|------|-------------|---------|
| `--global` | Update only global installations | `--global` |
| `--project` | Update only project installations | `--project` |

## Uninstalling Templates

### Uninstall Commands

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

# Uninstall only global installations
airuler uninstall --global

# Uninstall only project installations
airuler uninstall --project
```

### Uninstall Process

1. **Identifies tracked files** from installation database
2. **Prompts for confirmation** (unless using `--force`)
3. **Removes only airuler-installed files** (never removes user files)
4. **Updates installation database** to remove uninstalled entries

Note: Backups of target files created during installation are not automatically restored. They remain in the target directories with `.backup.timestamp` suffixes.

### Uninstall Options

| Flag | Description |
|------|-------------|
| `--interactive` | Choose specific installations to remove |
| `--force` | Skip confirmation prompts |
| `--global` | Uninstall only global installations |
| `--project` | Uninstall only project installations |

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
2. Manually inspect the YAML file if needed
3. Remove corrupted entries by editing the file directly
4. Reinstall templates to recreate tracking records

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
          airuler update
          airuler update-installed --force
          # Commit updated rules if changed
```

### Multi-Environment Management

```bash
# Development environment
airuler install --project ./dev-project

# Staging environment  
airuler install --project ./staging-project

# Production environment
airuler install --project ./prod-project

# List all environments
airuler list-installed --filter project
```

### Batch Operations

```bash
# Update all Claude installations
airuler update-installed --target claude

# Uninstall all cursor rules
airuler uninstall cursor --force

# Update specific rule across all projects
airuler update-installed --rule security-standards
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
airuler install
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
airuler list-installed --verbose

# Use absolute paths
airuler install --project "$(pwd)/project"
```

### Debugging Commands

```bash
# View installation details
airuler list-installed

# Install with confirmation prompts
airuler install --interactive

# Force reinstall to fix tracking
airuler install --force
```

### Recovery Procedures

**Lost installation database**:
1. The database file is not automatically backed up
2. Recreate by reinstalling: `airuler install --force`
3. Check config path: `airuler config path`

**Corrupted installations**:
1. Manually backup current database: `cp ~/.config/airuler/airuler.installs ~/.config/airuler/airuler.installs.backup`
2. Edit file manually or remove corrupted entries
3. Verify results: `airuler list-installed`

## Best Practices

### Installation Workflow
1. **Test first**: Use `--project` flag to test in isolated directory
2. **Review changes**: Use `list-installed` to verify installations
3. **Regular updates**: Set up automated update schedules
4. **Backup management**: Periodically clean old backup files

### Database Maintenance
- **Regular review**: Run `list-installed` to review current installations
- **Cleanup orphans**: Remove entries for deleted projects manually
- **Version control**: Include project `.airuler/` in git (if applicable)
- **Backup database**: Manually backup the installation database periodically

## See Also

- [Configuration](configuration.md) - Global and project configuration
- [Templates](templates.md) - Template structure and compilation
- [Vendors](vendors.md) - Managing external template repositories

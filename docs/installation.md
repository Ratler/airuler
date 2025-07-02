# Installation Management

airuler automatically tracks where templates have been installed, enabling safe updates, clean uninstalls, and comprehensive management of your AI rules across different targets and projects.

## How Installation Tracking Works

When you install templates using `airuler install`, airuler automatically:

1. **Records Installation Details**: Tracks the target, rule name, installation location, mode, and timestamp
2. **Maintains Installation Database**: Stores tracking information in global and project-specific databases
3. **Enables Safe Operations**: Allows for clean uninstalls and selective updates

## Installation Database

### Database Locations

- **Global installations**: `~/.config/airuler/installations.yaml` (Linux/macOS) or `%APPDATA%\airuler\installations.yaml` (Windows)
- **Project installations**: `./.airuler/installations.yaml` in each project directory

### Database Structure

```yaml
installations:
  - id: "uuid-1234-5678"
    target: "claude"
    rule: "coding-standards"
    mode: "memory"
    file: "/path/to/project/CLAUDE.md"
    project_path: "/path/to/project"
    installed_at: "2024-01-15T10:30:00Z"
    content_hash: "abc123def456"
    backup_file: "CLAUDE.md.backup.20240115-103000"
  
  - id: "uuid-2345-6789"
    target: "cursor"
    rule: "security-guide"
    mode: "normal"
    file: "/path/to/.cursor/rules/security-guide.mdc"
    project_path: ""  # Empty for global installations
    installed_at: "2024-01-15T11:00:00Z"
    content_hash: "def456ghi789"
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

# Show global installations only
airuler list-installed --global

# Show project installations only  
airuler list-installed --project
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

📁 Project Installations (/path/to/project)
==============================================================================
Target   Rule                 Mode     File                      Installed
------------------------------------------------------------------------------
cursor   project-rules        normal   project-rules.mdc         3 hours ago
```

### Filter Options

| Flag | Description | Example |
|------|-------------|---------|
| `--filter <keyword>` | Filter by target, rule, or file | `--filter claude` |
| `--global` | Show only global installations | `--global` |
| `--project` | Show only project installations | `--project` |

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
airuler update-installed --target claude

# Update specific rule
airuler update-installed --rule coding-standards
```

### Update Process

The update process:
1. **Recompiles templates** from current source
2. **Compares content** with installed versions using hash comparison
3. **Updates only if changed** to avoid unnecessary file modifications
4. **Creates backups** before overwriting (e.g., `rule.md.backup.20240115-143022`)
5. **Updates tracking database** with new timestamps and content hashes

### Update Options

| Flag | Description | Example |
|------|-------------|---------|
| `--global` | Update only global installations | `--global` |
| `--project` | Update only project installations | `--project` |
| `--target <name>` | Update specific target | `--target claude` |
| `--rule <name>` | Update specific rule | `--rule security-guide` |
| `--force` | Update even if content unchanged | `--force` |

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
4. **Restores backups** if available and requested
5. **Updates installation database** to remove uninstalled entries

### Uninstall Options

| Flag | Description |
|------|-------------|
| `--interactive` | Choose specific installations to remove |
| `--force` | Skip confirmation prompts |
| `--global` | Uninstall only global installations |
| `--project` | Uninstall only project installations |
| `--restore-backups` | Restore original files from backups |

## Installation Modes and Tracking

airuler tracks different installation types:

### Installation Types

- **Global installations**: Rules installed to AI tool global configurations
- **Project installations**: Rules installed to specific project directories  
- **Memory mode (Claude)**: Content appended to CLAUDE.md files
- **Command mode (Claude)**: Individual command files in .claude/commands/

### Mode-Specific Behavior

**Memory Mode**:
- Tracks content appended to CLAUDE.md
- Updates only the airuler-managed sections
- Preserves user-added content

**Command Mode**:
- Tracks individual command files
- Can update/remove specific commands
- Maintains command isolation

**Normal Mode** (Cursor, Cline, etc.):
- Tracks complete file installations
- Can safely overwrite managed files
- Creates backups before changes

## Safety Features

### Backup Creation

- **Automatic backups**: Original files are backed up before overwriting
- **Timestamped names**: `rule.md.backup.20240102-143022`
- **Content preservation**: Original content is never lost
- **Restore capability**: Backups can be restored during uninstall

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
# ~/.config/airuler/installations.yaml (global)
# ./.airuler/installations.yaml (project-specific)
```

### Database Repair

```bash
# Validate installation database
airuler validate-installations

# Repair corrupted database
airuler repair-installations

# Clean orphaned entries
airuler clean-installations
```

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
# Repair database
airuler repair-installations

# Rebuild from file system
airuler rebuild-installations
```

**Missing backup files**:
```bash
# Check backup locations
ls -la *.backup.*

# Recreate installation entry
airuler track-existing-installation <file>
```

**Permission issues**:
```bash
# Check file permissions
ls -la ~/.config/airuler/
ls -la ./.airuler/

# Fix permissions
chmod 644 ~/.config/airuler/installations.yaml
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
# Verbose installation output
airuler install --verbose

# Debug installation tracking
airuler list-installed --debug

# Validate installation state
airuler validate-installations --verbose
```

### Recovery Procedures

**Lost installation database**:
1. Check backup locations: `~/.config/airuler/installations.yaml.backup`
2. Rebuild from filesystem: `airuler rebuild-installations`
3. Manually recreate entries: `airuler track-existing-installation`

**Corrupted installations**:
1. Backup current database: `cp installations.yaml installations.yaml.backup`
2. Run repair: `airuler repair-installations`
3. Verify results: `airuler list-installed`

## Best Practices

### Installation Workflow
1. **Test first**: Use `--project` flag to test in isolated directory
2. **Review changes**: Use `list-installed` to verify installations
3. **Regular updates**: Set up automated update schedules
4. **Backup management**: Periodically clean old backup files

### Database Maintenance
- **Regular validation**: Run `validate-installations` monthly
- **Cleanup orphans**: Remove entries for deleted projects
- **Version control**: Include project `.airuler/` in git (if applicable)
- **Backup database**: Keep backups of installation database

### Security Considerations
- **File permissions**: Ensure database files have appropriate permissions
- **Path validation**: Use absolute paths to prevent directory traversal
- **Content verification**: Verify file content before sensitive operations
- **Audit trail**: Maintain logs of installation operations

## See Also

- [Configuration](configuration.md) - Global and project configuration
- [Templates](templates.md) - Template structure and compilation
- [Vendors](vendors.md) - Managing external template repositories
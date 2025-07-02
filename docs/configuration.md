# Configuration

airuler supports both project-specific and global configuration files for flexible template management.

## Configuration Precedence

Configuration is loaded in the following order (highest to lowest priority):

1. `--config` flag (if specified)
2. `./airuler.yaml` (project-specific config)
3. Global config:
   - Linux/macOS: `~/.config/airuler/airuler.yaml`
   - Windows: `%APPDATA%\airuler\airuler.yaml`

## Managing Global Configuration

### Configuration Commands

```bash
# Initialize global config
airuler config init

# Show config file locations
airuler config path

# Edit global config (uses $EDITOR environment variable)
airuler config edit

# Set default template directory
airuler config set-template-dir <path>
```

### Configuration File Locations

Use `airuler config path` to see exact paths for your system:

```bash
$ airuler config path
Global config: ~/.config/airuler/airuler.yaml
Installation tracking: ~/.config/airuler/installations.yaml
```

## airuler.yaml Configuration

### Basic Configuration Structure

```yaml
# Default settings
defaults:
  include_vendors: ["*"]  # Include all vendors by default
  # Or specify specific vendors:
  # include_vendors: [frontend, security]
  last_template_dir: "/path/to/templates"  # Auto-managed template directory
```

### Configuration Options

| Setting | Description | Default | Example |
|---------|-------------|---------|---------|
| `include_vendors` | Vendors to include in compilation | `["*"]` | `["frontend", "security"]` |
| `last_template_dir` | Remembered template directory | auto-detected | `"/home/user/templates"` |

### Project-Specific Configuration

Create `airuler.yaml` in your project root:

```yaml
defaults:
  include_vendors: ["company-standards", "security-rules"]
  
# Project-specific settings can override global ones
compile:
  targets: ["claude", "cursor"]  # Only compile for specific targets
```

### Vendor-Specific Configuration

Each vendor can include its own `airuler.yaml`:

```yaml
# In vendor repository
vendor:
  name: "Frontend Standards"
  description: "React/TypeScript coding standards"
  version: "2.1.0"
  
defaults:
  include_vendors: []  # Vendors don't include other vendors by default
```

## Global Template Directory

airuler automatically remembers the last template directory you compiled from, allowing you to run commands from anywhere.

### How It Works

1. **Automatic Detection**: When you run `airuler compile` from a template directory, airuler saves that directory as your default template directory.

2. **Seamless Operation**: Commands run from outside a template directory automatically operate as if you're in the remembered template directory.

3. **User Feedback**: When operating from a remembered directory, airuler shows: `Using template directory: /path/to/templates`

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
- **Workflow Integration**: Fits naturally into development workflows
- **Project Flexibility**: Switch between different template repositories seamlessly
- **Backward Compatibility**: Existing workflows continue to work unchanged

## Environment Variables

### Configuration Override

```bash
# Override config file location
export AIRULER_CONFIG="/custom/path/to/config.yaml"

# Override template directory
export AIRULER_TEMPLATE_DIR="/custom/templates"

# Enable debug mode
export AIRULER_DEBUG=1

# Disable color output
export NO_COLOR=1
```

### Git Configuration

For vendor management:

```bash
# Git credentials for private repositories
export GIT_USERNAME="your-username"
export GIT_TOKEN="your-personal-access-token"

# SSH key for Git operations
export GIT_SSH_KEY="/path/to/ssh/key"
```

## Lock Files

### airuler.lock Structure

The lock file tracks vendor dependencies and template directory:

```yaml
# Vendor dependencies
vendors:
  frontend:
    url: https://github.com/company/frontend-rules
    revision: abc123def456
    updated: 2024-01-15T10:30:00Z
  backend:
    url: https://github.com/company/backend-rules
    revision: def456ghi789
    updated: 2024-01-15T11:00:00Z

# Configuration metadata
metadata:
  created: 2024-01-01T00:00:00Z
  airuler_version: "1.0.0"
```

### Lock File Management

- **Commit to Version Control**: Always commit `airuler.lock`
- **Automatic Updates**: Lock file updates when vendors change
- **Conflict Resolution**: Run `airuler update` to resolve conflicts
- **Never Edit Manually**: Use airuler commands to modify

## Configuration Examples

### Team Development Setup

Global config (`~/.config/airuler/airuler.yaml`):
```yaml
defaults:
  include_vendors: ["*"]
  
# Personal preferences
editor: "code"
template_dirs:
  - "/home/user/work/templates"
  - "/home/user/personal/templates"
```

Project config (`./airuler.yaml`):
```yaml
defaults:
  include_vendors: ["company-standards", "project-specific"]
  
compile:
  targets: ["claude", "cursor"]
  
install:
  backup: true
  interactive: false
```

### Multi-Project Workspace

```yaml
# Workspace-level configuration
defaults:
  include_vendors: ["workspace-standards"]
  
workspaces:
  frontend:
    path: "./frontend"
    vendors: ["react-standards", "typescript-rules"]
  backend:
    path: "./backend"  
    vendors: ["api-standards", "security-rules"]
  mobile:
    path: "./mobile"
    vendors: ["react-native-standards"]
```

### CI/CD Configuration

```yaml
# Optimized for automated environments
defaults:
  include_vendors: ["*"]
  
ci:
  parallel_compilation: true
  cache_vendors: true
  quiet: true
  
install:
  backup: false
  force: true
```

## Advanced Configuration

### Custom Target Configurations

```yaml
targets:
  claude:
    memory_mode: true
    command_mode: true
    output_format: "markdown"
  cursor:
    always_apply: true
    description_required: true
    output_format: "mdc"
  custom_target:
    enabled: true
    output_dir: "custom/"
    file_extension: ".custom"
```

### Template Processing Options

```yaml
templates:
  strict_mode: true          # Fail on undefined variables
  trim_whitespace: true      # Remove extra whitespace
  include_metadata: false    # Include front matter in output
  
partials:
  auto_discover: true        # Automatically find partials
  validate_references: true  # Check partial references exist
```

### Performance Tuning

```yaml
performance:
  parallel_compilation: true  # Compile templates in parallel
  cache_templates: true       # Cache parsed templates
  max_workers: 4             # Number of parallel workers
  memory_limit: "512MB"      # Memory usage limit
```

## Configuration Validation

### Validate Configuration

```bash
# Check configuration syntax
airuler config validate

# Show resolved configuration
airuler config show

# Debug configuration loading
airuler config debug
```

### Common Configuration Errors

**Invalid YAML syntax**:
```bash
Error: yaml: line 5: found character that cannot start any token
```
Solution: Check YAML formatting, ensure proper indentation

**Missing required fields**:
```bash
Error: include_vendors must be specified
```
Solution: Add required configuration fields

**Invalid paths**:
```bash
Error: template directory '/invalid/path' does not exist
```
Solution: Use valid, absolute paths for directories

## See Also

- [Installation Management](installation.md) - Installation tracking and configuration
- [Vendor Management](vendors.md) - Vendor-specific configuration options  
- [Examples](examples.md) - Real-world configuration examples
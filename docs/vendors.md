# Vendor Management

Fetch and manage external rule repositories to share templates across projects.

## Overview

Vendors allow you to:
- Share templates across multiple projects
- Maintain centralized rule repositories
- Version control shared coding standards
- Collaborate on template development

## Fetching External Repositories

### Basic Fetch Operations

```bash
# Fetch from Git repository
airuler fetch https://github.com/company/frontend-rules

# Fetch with custom alias
airuler fetch https://github.com/company/backend-rules --as backend

# Update existing vendor
airuler fetch https://github.com/company/frontend-rules --update
```

### Fetch Options

| Flag | Description | Example |
|------|-------------|---------|
| `--as <alias>` | Set custom vendor name | `--as backend` |
| `--update` | Update existing vendor repository | `--update` |

## Updating Vendors

### Update Commands

```bash
# Update all vendors
airuler update

# Update specific vendor
airuler update backend

# Check for updates without fetching
airuler update --dry-run

# Interactive update mode
airuler update --interactive
```

### Update Options

| Flag | Description |
|------|-------------|
| `--dry-run` | Check for updates without downloading |
| `--interactive` | Choose which vendors to update |

## Managing Vendors

### List and Status

```bash
# List all vendors
airuler vendors list

# Check vendor status
airuler vendors status

# Check for vendor updates
airuler vendors check
```

### Include/Exclude Vendors

```bash
# Include vendor in compilation
airuler vendors include frontend

# Exclude vendor from compilation
airuler vendors exclude backend

# Include all vendors
airuler vendors include-all

# Exclude all vendors
airuler vendors exclude-all
```

### Remove Vendors

```bash
# Remove vendor completely
airuler vendors remove backend
```

## Compiling with Vendors

### Vendor-Specific Compilation

```bash
# Compile including all enabled vendors
airuler compile

# Compile from specific vendor
airuler compile --vendor backend

# Compile from multiple vendors
airuler compile --vendors "backend,frontend"

# Compile specific rule from vendor
airuler compile --vendor frontend --rule coding-standards
```

### Vendor Compilation Options

| Flag | Description | Example |
|------|-------------|---------|
| `--vendor <name>` | Compile from specific vendor | `--vendor frontend` |
| `--vendors <list>` | Compile from multiple vendors | `--vendors "fe,be"` |

## Vendor Directory Structure

When you fetch a vendor, airuler creates:

```
vendors/
└── <vendor-name>/
    ├── templates/          # Vendor's templates
    │   ├── partials/       # Vendor's partials
    │   └── *.tmpl          # Template files
    ├── .git/              # Git repository data
    └── airuler.yaml       # Vendor's configuration
```

## Vendor Isolation

- **Template Isolation**: Local templates can only access local partials
- **Vendor Isolation**: Vendor templates can only access partials from the same vendor
- **Naming Conflicts**: Vendors are isolated, preventing naming conflicts
- **Independent Compilation**: Each template compiles with its own context

## Configuration Integration

### Project Configuration

In your `airuler.yaml`:

```yaml
defaults:
  include_vendors: ["*"]  # Include all vendors by default
  # Or specify specific vendors:
  # include_vendors: [frontend, security]
```

### Vendor-Specific Configuration

Each vendor can have its own `airuler.yaml` with:
- Default compilation settings
- Vendor-specific metadata
- Template organization preferences

## Lock File Management

The `airuler.lock` file tracks vendor versions:

```yaml
vendors:
  frontend:
    url: https://github.com/company/frontend-rules
    revision: abc123def456
    updated: 2024-01-15T10:30:00Z
  backend:
    url: https://github.com/company/backend-rules
    revision: def456ghi789
    updated: 2024-01-15T11:00:00Z
```

## Team Collaboration Workflow

### Setting Up Shared Vendors

1. **Team Lead**: Create shared repository
   ```bash
   # Initialize vendor repository
   mkdir company-standards
   cd company-standards
   airuler init
   # Add templates and commit
   git init && git add . && git commit -m "Initial standards"
   git remote add origin https://github.com/company/standards
   git push -u origin main
   ```

2. **Team Members**: Fetch shared standards
   ```bash
   airuler fetch https://github.com/company/standards --as company
   ```

3. **Configure Project**: Update `airuler.yaml`
   ```yaml
   defaults:
     include_vendors: [company]
   ```

### Updating Shared Standards

1. **Update Vendor Repository**: Make changes to templates
2. **Team Sync**: Update local copies
   ```bash
   airuler update company
   airuler compile
   airuler update-installed
   ```

## Vendor Best Practices

### Repository Structure
- Follow standard airuler project structure
- Include comprehensive `README.md` for vendor
- Use semantic versioning with Git tags
- Document template purposes and usage

### Template Organization
```
vendor-repo/
├── templates/
│   ├── general/           # General coding standards
│   ├── security/          # Security-focused rules
│   ├── frameworks/        # Framework-specific rules
│   └── partials/          # Shared components
├── README.md             # Vendor documentation
├── CHANGELOG.md          # Version history
└── airuler.yaml          # Vendor configuration
```

### Version Management
- Use Git tags for releases: `v1.0.0`, `v1.1.0`
- Maintain backward compatibility when possible
- Document breaking changes in CHANGELOG
- Test templates before releasing

### Collaboration Guidelines
- Follow consistent naming conventions
- Include template descriptions in front matter
- Use partials for common content
- Test templates across different targets

## Troubleshooting

### Common Issues

**Vendor sync problems**:
```bash
# Check vendor status
airuler vendors status

# Force update
airuler fetch https://github.com/vendor/repo --update
```

**Template conflicts**:
- Vendor templates are isolated by design
- Check template names within vendor
- Verify partial references are correct

**Git authentication**:
- Ensure proper Git credentials for private repositories
- Use SSH keys or personal access tokens as needed
- Check repository permissions

**Lock file issues**:
- Commit `airuler.lock` to version control
- Resolve conflicts by running `airuler update`
- Never manually edit lock file

### Debugging Commands

```bash
# Verbose output for debugging
airuler fetch <url> --verbose
airuler update --verbose
airuler compile --vendor <name> --verbose

# Check vendor configuration
airuler vendors status
airuler config path
```

## See Also

- [Template Syntax](templates.md) - Understanding template structure and partials
- [Configuration](configuration.md) - Project and global configuration options
- [Examples](examples.md) - Real-world vendor usage patterns
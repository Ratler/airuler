# Vendor Management

Fetch and manage external rule repositories to share templates across projects.

## Overview

Vendors allow you to:

- Share templates across multiple projects
- Maintain centralized rule repositories
- Version control shared coding standards
- Collaborate on template development

## Adding External Repositories

### Basic Add Operations

```bash
# Add from Git repository
airuler vendors add https://github.com/company/frontend-rules

# Add with custom alias
airuler vendors add https://github.com/company/backend-rules --as backend

# Update existing vendor during add
airuler vendors add https://github.com/company/frontend-rules --update
```

### Add Options

| Flag           | Short | Description                       | Example        |
| -------------- | ----- | --------------------------------- | -------------- |
| `--as <alias>` | `-a`  | Set custom vendor name            | `--as backend` |
| `--update`     | `-u`  | Update existing vendor repository | `--update`     |

## Updating Vendors

### Update Commands

```bash
# Update all vendors
airuler vendors update

# Update specific vendor
airuler vendors update backend

# Update multiple vendors
airuler vendors update frontend,backend
```

### Update Notes

- Updates pull the latest changes from the vendor's Git repository
- Updates are tracked in the `airuler.lock` file
- Use `airuler vendors status` to check for available updates before updating

## Managing Vendors

### List and Status

```bash
# List all vendors with repository and configuration details
airuler vendors list

# Show detailed configuration for specific vendor
airuler vendors list frontend-vendor

# Check vendor status
airuler vendors status

# Check for vendor updates without fetching
airuler vendors check
```

### Enhanced List Command

The `vendors list` command has been enhanced to combine repository and configuration information:

- **Without arguments**: Shows all vendors with repository info and configuration summaries
- **With vendor name**: Shows detailed configuration for the specific vendor

This replaces the separate `vendors config` command for better usability.

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

## Using Vendors in Workflows

### Vendor Integration in Deploy and Sync

Vendors are automatically included when using the main workflow commands:

```bash
# Deploy with all enabled vendors
airuler deploy

# Deploy for specific targets (includes vendor templates)
airuler deploy --targets cursor,claude

# Sync workflow includes vendor updates and deployment
airuler sync

# Sync without vendor updates
airuler sync --no-update
```

### Vendor Scope Control

Control which vendors are included in compilation through configuration or include/exclude commands:

```bash
# Include specific vendor
airuler vendors include frontend

# Exclude specific vendor
airuler vendors exclude backend

# Include all vendors
airuler vendors include-all

# Exclude all vendors (local templates only)
airuler vendors exclude-all
```

## Vendor Directory Structure

When you add a vendor, airuler creates:

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

Each vendor can have its own `airuler.yaml` with vendor-specific settings that apply to their templates:

```yaml
# vendors/frontend-standards/airuler.yaml
vendor:
  name: "Frontend Standards"
  description: "React/TypeScript coding standards"
  version: "2.1.0"
  author: "Frontend Team"
  homepage: "https://company.com/standards"

template_defaults:
  language: "typescript"
  framework: "react"
  project_type: "web-application"
  custom:
    min_node_version: "18.0.0"
    build_tool: "vite"

targets:
  claude:
    default_mode: "memory"    # Default mode for Claude templates

variables:
  company_name: "Acme Corp"
  style_guide_url: "https://company.com/style-guide"
  support_email: "frontend-team@company.com"
```

**How Vendor Configuration Works:**

- Vendor defaults are applied to templates from that vendor
- Template front matter can override vendor defaults
- Project configuration can override vendor settings via `vendor_overrides`
- Each vendor's templates only use their own configuration

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

1. **Team Members**: Add shared standards

   ```bash
   airuler vendors add https://github.com/company/standards --as company
   ```

1. **Configure Project**: Update `airuler.yaml`

   ```yaml
   defaults:
     include_vendors: [company]
   ```

### Updating Shared Standards

1. **Update Vendor Repository**: Make changes to templates
1. **Team Sync**: Update local copies
   ```bash
   airuler vendors update company
   airuler sync
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
airuler vendors add https://github.com/vendor/repo --update
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
airuler vendors add <url> --verbose
airuler vendors update --verbose
airuler deploy --verbose
airuler sync --verbose

# Check vendor configuration
airuler vendors list
airuler vendors status
airuler config path
```

## See Also

- [Template Syntax](templates.md) - Understanding template structure and partials
- [Configuration](configuration.md) - Project and global configuration options
- [Examples](examples.md) - Real-world vendor usage patterns

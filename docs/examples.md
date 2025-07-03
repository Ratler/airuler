# Examples & Best Practices

Real-world examples and development workflows for airuler template management.

## Development Workflow Examples

### Local Development

#### 1. Initialize and Setup
```bash
# Initialize project
airuler init
cd templates

# Create your first template
cat > coding-standards.tmpl << 'EOF'
---
claude_mode: memory
description: "Project coding standards"
language: "TypeScript"
framework: "React"
---
# {{.Language}} {{.Framework}} Standards

## Code Quality
- Use TypeScript for type safety
- Follow ESLint configuration
- Write unit tests for all components

{{if eq .Target "claude"}}
When reviewing code, ensure:
1. Proper TypeScript types are used
2. Components follow React best practices
3. Tests cover edge cases and error scenarios
{{end}}
EOF
```

#### 2. Development with Watch Mode
```bash
# Start watch mode for rapid iteration
airuler watch

# In another terminal, test compilation
airuler compile claude --rule coding-standards

# Test installation
airuler install claude --project ./test-project
```

#### 3. Test and Validate
```bash
# Compile for all targets to test compatibility
airuler compile

# Install and test with different AI tools
airuler install cursor --project ./cursor-test
airuler install claude --project ./claude-test
airuler install gemini --project ./gemini-test
```

### Team Collaboration

#### 1. Shared Standards Repository
```bash
# Team lead creates shared repository
mkdir company-standards
cd company-standards
airuler init

# Add comprehensive templates
mkdir -p templates/partials
cat > templates/partials/header.ptmpl << 'EOF'
---
description: "Standard header for all rules"
---
# {{.Name}} - {{.Framework}} Development

**Project**: {{.ProjectType}} | **Target**: {{.Target}} | **Language**: {{.Language}}

---
EOF

cat > templates/security-standards.tmpl << 'EOF'
---
claude_mode: both
description: "Security best practices"
language: "typescript"
framework: "react"
project_type: "web-application"
---
{{template "partials/header" .}}

## Authentication & Authorization
- Implement JWT-based authentication
- Use RBAC for authorization
- Validate all user inputs

## Data Protection
- Encrypt sensitive data at rest
- Use HTTPS for all communications
- Implement proper session management

{{if eq .Target "claude"}}
For security reviews:
1. Check for hardcoded secrets
2. Verify input sanitization
3. Confirm proper error handling
{{else if eq .Target "gemini"}}
Security checklist for Gemini CLI:
1. Audit code for hardcoded credentials
2. Validate all user inputs properly
3. Implement comprehensive error handling
{{end}}
EOF

# Commit and push
git add . && git commit -m "Initial security standards"
git remote add origin https://github.com/company/standards
git push -u origin main
```

#### 2. Team Member Setup
```bash
# Clone and fetch shared standards
cd my-project
airuler init
airuler fetch https://github.com/company/standards --as company

# Configure to use company standards
cat >> airuler.yaml << 'EOF'
defaults:
  include_vendors: [company]
EOF

# Compile and install
airuler compile
airuler install --project .
```

#### 3. Syncing Updates
```bash
# Regular sync workflow
airuler update company
airuler compile
airuler update-installed
```

## Template Examples

### Multi-Framework Template

```yaml
---
description: "Framework-agnostic coding standards"
globs: "**/*.{js,ts,jsx,tsx,vue,py}"
project_type: "web-application"
framework: "react"  # Can be overridden
language: "typescript"
tags: ["frontend", "backend", "fullstack"]
---
# {{title .Framework}} Development Standards

{{if contains .Tags "frontend"}}
## Frontend Guidelines
{{if eq .Framework "react"}}
### React Best Practices
- Use functional components with hooks
- Implement proper prop validation with TypeScript
- Follow React component lifecycle best practices

```typescript
interface Props {
  title: string;
  children: React.ReactNode;
  onClick?: () => void;
}

export const Component: React.FC<Props> = ({ title, children, onClick }) => {
  const [isActive, setIsActive] = useState(false);
  
  return (
    <div className={`component ${isActive ? 'active' : ''}`}>
      <h2>{title}</h2>
      {children}
      {onClick && <button onClick={onClick}>Action</button>}
    </div>
  );
};
```
{{else if eq .Framework "vue"}}
### Vue.js Best Practices
- Use Composition API for complex logic
- Implement proper reactive data patterns
- Follow Vue 3 style guide

```vue
<script setup lang="ts">
interface Props {
  title: string
  items: string[]
}

const props = defineProps<Props>()
const isVisible = ref(true)
</script>
```
{{end}}
{{end}}

{{if contains .Tags "backend"}}
## Backend Guidelines
- Implement proper error handling
- Use typed request/response interfaces
- Follow RESTful API conventions
{{end}}

## Universal Principles
- Write clean, readable code
- Include comprehensive documentation
- Implement proper logging
- Follow security best practices

{{if eq .Target "claude"}}
## Code Review Checklist
When reviewing {{.Language}} code:
1. ✅ Verify proper typing and interfaces
2. ✅ Check for potential security vulnerabilities
3. ✅ Ensure error handling is comprehensive
4. ✅ Confirm tests cover edge cases
5. ✅ Validate performance considerations
{{end}}
```

### Complex Template with Partials

Main template (`templates/comprehensive-guide.tmpl`):
```yaml
---
claude_mode: memory
description: "Comprehensive development guide"
project_type: "web-application"
language: "typescript"
framework: "react"
---
{{template "partials/header" .}}

{{template "architecture/overview" .}}

{{template "standards/coding" .}}

{{template "standards/security" .}}

{{template "standards/testing" .}}

{{template "deployment/ci-cd" .}}

{{template "partials/footer" .}}
```

Architecture overview partial (`templates/architecture/overview.ptmpl`):
```yaml
---
description: "Architecture overview section"
---
## Architecture Overview

### Tech Stack
- **Frontend**: {{.Framework}} with {{.Language}}
- **Bundler**: Vite
- **Testing**: Jest + React Testing Library
- **Linting**: ESLint + Prettier

### Project Structure
```
src/
├── components/     # Reusable UI components
├── pages/         # Route components
├── hooks/         # Custom React hooks
├── utils/         # Utility functions
├── types/         # TypeScript type definitions
└── tests/         # Test files
```

### Component Architecture
- **Atomic Design**: Atoms → Molecules → Organisms → Templates → Pages
- **Props Interface**: All components must have TypeScript interfaces
- **Error Boundaries**: Implement error boundaries for component trees
```

Security standards partial (`templates/standards/security.ptmpl`):
```yaml
---
description: "Security standards section"
---
## Security Standards

### Input Validation
```typescript
// Always validate and sanitize user inputs
const validateEmail = (email: string): boolean => {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email) && email.length <= 254;
};

// Use libraries for complex validation
import { z } from 'zod';

const UserSchema = z.object({
  email: z.string().email().max(254),
  age: z.number().min(13).max(120),
});
```

### Authentication
- Use httpOnly cookies for tokens
- Implement proper session management
- Add CSRF protection
- Use secure password hashing (bcrypt)

### Data Protection
- Encrypt sensitive data at rest
- Use HTTPS for all communications
- Implement proper key management
- Never log sensitive information
```

### TypeScript-Specific Template

```yaml
---
claude_mode: both
description: "TypeScript development standards"
language: "typescript"
project_type: "library"
globs: "**/*.{ts,tsx}"
tags: ["typescript", "library", "npm"]
custom:
  min_ts_version: "4.9.0"
  target: "ES2022"
---
# TypeScript Development Standards

## Type Safety Guidelines

### Interface Design
```typescript
// Use interfaces for object shapes
interface User {
  readonly id: string;
  name: string;
  email: string;
  createdAt: Date;
  updatedAt?: Date;
}

// Use type aliases for unions and computed types
type UserRole = 'admin' | 'user' | 'guest';
type UserWithRole = User & { role: UserRole };

// Use generic interfaces for reusable patterns
interface ApiResponse<T> {
  data: T;
  status: 'success' | 'error';
  message?: string;
}
```

### Function Typing
```typescript
// Prefer function declarations with explicit return types
function processUser(user: User): Promise<UserWithRole> {
  // Implementation
}

// Use const assertions for better inference
const STATUSES = ['pending', 'approved', 'rejected'] as const;
type Status = typeof STATUSES[number];
```

### Error Handling
```typescript
// Use Result pattern for error handling
type Result<T, E = Error> = 
  | { success: true; data: T }
  | { success: false; error: E };

async function fetchUser(id: string): Promise<Result<User>> {
  try {
    const user = await userService.findById(id);
    return { success: true, data: user };
  } catch (error) {
    return { success: false, error: error as Error };
  }
}
```

## Build Configuration

### TypeScript Config
```json
{
  "compilerOptions": {
    "target": "{{.Custom.target}}",
    "module": "ESNext",
    "moduleResolution": "node",
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "exactOptionalPropertyTypes": true,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": true
  }
}
```

{{if eq .Target "claude"}}
## TypeScript Review Guidelines

When working with TypeScript code:
1. ✅ Ensure all functions have explicit return types
2. ✅ Use strict type checking (no `any` types)
3. ✅ Implement proper error handling patterns
4. ✅ Use readonly for immutable data
5. ✅ Prefer interfaces over type aliases for object shapes
6. ✅ Add JSDoc comments for public APIs
{{end}}
```

### Gemini CLI Specific Template

```yaml
---
description: "Gemini CLI coding assistant rules"
language: "python"
project_type: "data-science"
globs: "**/*.{py,ipynb}"
tags: ["machine-learning", "data-analysis"]
---
# {{.Language}} Development for {{.ProjectType}}

## Data Science Best Practices

### Code Organization
- Structure notebooks with clear sections
- Extract reusable functions to .py modules
- Use meaningful variable names for data analysis
- Document data sources and transformations

### {{.Language}} Specific Guidelines
```python
# Use type hints for better code clarity
import pandas as pd
from typing import List, Dict, Optional

def analyze_data(df: pd.DataFrame, 
                columns: List[str], 
                threshold: float = 0.5) -> Dict[str, float]:
    """Analyze data with specified threshold."""
    results = {}
    for col in columns:
        if col in df.columns:
            results[col] = df[col].mean()
    return results
```

### Machine Learning Guidelines
- Always split data before any preprocessing
- Use cross-validation for model evaluation
- Document model assumptions and limitations
- Version control datasets and model artifacts

### Jupyter Notebook Standards
- Use descriptive cell headers with markdown
- Clear all outputs before committing
- Include data source documentation
- Add conclusion cells summarizing findings

## Target-Specific Instructions
{{if eq .Target "gemini"}}
When working with this {{.ProjectType}} codebase:
1. 🔍 Focus on data integrity and validation
2. 📊 Suggest appropriate visualizations for data exploration
3. 🧮 Recommend statistical tests when relevant
4. 🔬 Help optimize model performance and interpretability
5. 📝 Ensure reproducibility of analysis results
{{end}}
```

### Docker Integration Template

```yaml
---
claude_mode: command
description: "Review Docker setup for Node.js project"
language: "dockerfile"
project_type: "web-application"
---
# Docker Review: $ARGUMENTS

Review the Docker setup for the {{.ProjectType}} project.

## Dockerfile Analysis

### Multi-stage Build Check
```dockerfile
# Good: Multi-stage build for optimization
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

FROM node:18-alpine AS runtime
WORKDIR /app
COPY --from=builder /app/node_modules ./node_modules
COPY . .
EXPOSE 3000
CMD ["npm", "start"]
```

### Security Best Practices
- ✅ Use specific version tags (not `latest`)
- ✅ Use non-root user
- ✅ Minimize layer count
- ✅ Use .dockerignore file
- ✅ Scan for vulnerabilities

### Performance Optimization
- ✅ Multi-stage builds for smaller images
- ✅ Layer caching optimization
- ✅ Remove dev dependencies in production
- ✅ Use Alpine variants when possible

## Review Steps
1. Check Dockerfile exists and follows best practices
2. Verify .dockerignore includes unnecessary files
3. Test image build and run locally
4. Check for security vulnerabilities
5. Optimize image size if needed
```

## Advanced Workflows

### CI/CD Integration

```yaml
# .github/workflows/airuler.yml
name: Update AI Rules
on:
  push:
    paths: ['templates/**', 'airuler.yaml']
  schedule:
    - cron: '0 9 * * 1'  # Weekly updates

jobs:
  update-rules:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v3
        with:
          go-version: '1.21'
          
      - name: Build airuler
        run: go build -o airuler
        
      - name: Update vendors
        run: ./airuler update
        
      - name: Compile templates
        run: ./airuler compile
        
      - name: Install to projects
        run: |
          ./airuler install --project ./frontend
          ./airuler install --project ./backend
          ./airuler install --project ./mobile
          
      - name: Commit updates
        run: |
          git config user.name "GitHub Actions"
          git config user.email "actions@github.com"
          git add .
          git diff --staged --quiet || git commit -m "Update AI rules [skip ci]"
          git push
```

### Multi-Project Workspace

```bash
# Workspace structure
workspace/
├── shared-templates/          # Central template repository
│   ├── templates/
│   ├── vendors/
│   └── airuler.yaml
├── frontend/                  # React project
├── backend/                   # Node.js API
├── mobile/                    # React Native
└── scripts/
    └── update-all-rules.sh

# Update script
#!/bin/bash
# scripts/update-all-rules.sh

cd shared-templates
airuler update
airuler compile

# Install to all projects
for project in ../frontend ../backend ../mobile; do
  echo "Updating rules for $project"
  airuler install --project "$project"
done

echo "All projects updated with latest rules"
```

### Custom Target Integration

```bash
# Add custom target support
mkdir -p internal/compiler/targets

cat > internal/compiler/targets/custom.go << 'EOF'
package targets

import (
    "path/filepath"
    "github.com/ratler/airuler/internal/template"
)

type CustomTarget struct{}

func (c *CustomTarget) Name() string {
    return "custom"
}

func (c *CustomTarget) FileExtension() string {
    return ".custom"
}

func (c *CustomTarget) CompileTemplate(tmpl *template.Template) (string, error) {
    // Custom compilation logic
    return tmpl.Content, nil
}

func (c *CustomTarget) InstallPath(projectPath string) string {
    if projectPath != "" {
        return filepath.Join(projectPath, ".custom", "rules")
    }
    return filepath.Join(os.Getenv("HOME"), ".custom", "rules")
}
EOF
```

## Best Practices

### Template Organization

#### Directory Structure
```
templates/
├── general/              # General coding standards
│   ├── coding-standards.tmpl
│   └── documentation.tmpl
├── languages/            # Language-specific rules
│   ├── typescript.tmpl
│   ├── python.tmpl
│   └── go.tmpl
├── frameworks/           # Framework-specific rules
│   ├── react.tmpl
│   ├── vue.tmpl
│   └── fastapi.tmpl
├── security/            # Security-focused templates
│   ├── auth.tmpl
│   ├── input-validation.tmpl
│   └── data-protection.tmpl
├── partials/            # Reusable components
│   ├── header.tmpl
│   ├── footer.tmpl
│   └── security-checklist.tmpl
└── examples/            # Example templates
    ├── basic-template.tmpl
    └── advanced-template.tmpl
```

#### Naming Conventions
- Use kebab-case for template files: `coding-standards.tmpl`
- Include target in name if target-specific: `claude-memory-guide.tmpl`
- Use descriptive, specific names: `react-typescript-standards.tmpl`
- Group related templates in subdirectories

### Claude Code Mode Strategy

#### When to Use Each Mode

**Memory Mode**: Use for persistent project context
- Project architecture documentation
- Coding standards that apply to all code
- Technology stack information
- General best practices

**Command Mode**: Use for specific, repeatable tasks
- Code refactoring commands
- Analysis tools
- Template generation
- Specific review checklists

**Both Mode**: Use for comprehensive guidelines
- Security standards (persistent + on-demand checks)
- Testing strategies (general + specific commands)
- Performance guidelines (background + analysis tools)

### Version Control Best Practices

#### What to Commit
```gitignore
# Include in version control
airuler.yaml
airuler.lock
templates/
docs/

# Exclude from version control
compiled/              # Generated files
.airuler/             # Local installation tracking
vendors/              # External repositories
*.backup.*            # Backup files
```

#### Template Versioning
- Use semantic versioning for major template changes
- Tag releases: `git tag v1.2.0`
- Maintain CHANGELOG.md for template updates
- Document breaking changes clearly

### Performance Optimization

#### Compilation Optimization
```bash
# Use specific rules during development
airuler compile --rule my-template

# Parallel compilation for large template sets
airuler compile --parallel

# Cache templates during development
export AIRULER_CACHE_TEMPLATES=1
```

#### Watch Mode Efficiency
```bash
# Watch specific directories
airuler watch --include "templates/react/**"

# Exclude vendor templates from watch
airuler watch --exclude "vendors/**"
```

### Security Considerations

#### Template Security
- Never include secrets in templates
- Use environment variables for sensitive configuration
- Validate template inputs and outputs
- Review vendor templates before use

#### Installation Security
- Use project-specific installations for sensitive projects
- Regular audit of installed templates
- Backup before major updates
- Verify file permissions after installation

## Troubleshooting Workflows

### Common Issues and Solutions

#### Template Compilation Errors
```bash
# Debug template syntax
airuler compile --rule problem-template --verbose

# Validate YAML front matter
yamllint templates/problem-template.tmpl

# Check template function usage
grep -n "{{.*}}" templates/problem-template.tmpl
```

#### Vendor Sync Problems
```bash
# Check vendor status
airuler vendors status

# Force vendor update
airuler fetch https://repo-url --update --verbose

# Reset vendor to clean state
airuler vendors remove vendor-name
airuler fetch https://repo-url --as vendor-name
```

#### Installation Issues
```bash
# Verify installation tracking
airuler list-installed --verbose

# Check file permissions
ls -la ~/.config/airuler/
ls -la ./.airuler/

# Repair installation database
airuler repair-installations
```

### Development Debugging

#### Template Development
```bash
# Test template compilation
airuler compile claude --rule my-template --output /tmp/test

# Verify template variables
airuler compile --rule my-template --debug-vars

# Check partial resolution
airuler compile --rule my-template --debug-partials
```

#### Integration Testing
```bash
# Test full workflow
./scripts/test-integration.sh

#!/bin/bash
# scripts/test-integration.sh
set -e

echo "Testing airuler integration..."

# Setup test environment
TEST_DIR=$(mktemp -d)
cd "$TEST_DIR"

# Initialize and test
airuler init
echo "Test template" > templates/test.tmpl
airuler compile
airuler install --project .

# Verify installation
[ -f .cursor/rules/test.mdc ] || exit 1
[ -f CLAUDE.md ] || exit 1

echo "Integration test passed"
rm -rf "$TEST_DIR"
```

## See Also

- [Template Syntax](templates.md) - Detailed template syntax reference
- [Vendor Management](vendors.md) - Managing external repositories
- [Configuration](configuration.md) - Project and global configuration
- [Installation Management](installation.md) - Installation tracking and updates
# Contributing Guidelines

## Pull Requests

This project uses **semantic pull requests** to maintain a clear and consistent commit history. Pull request titles must follow the conventional commit format:

```
<type>(<scope>): <description>
```

### Examples
- `feat: add new template compilation feature`
- `fix(compiler): resolve template parsing error`  
- `docs: update installation instructions`
- `refactor(config): simplify YAML parsing logic`

### Types
- `feat`: New features
- `fix`: Bug fixes
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring without functionality changes
- `test`: Adding or updating tests
- `chore`: Maintenance tasks, dependency updates

The PR title validation is enforced by our [pr-checks workflow](.github/workflows/pr-checks.yml) using the `amannn/action-semantic-pull-request` action.

## Development Workflow

1. Fork the repository
2. Create a feature branch from `main`
3. Ensure tests pass with `make test`
4. Run code quality checks with `make check`
5. Submit a pull request with a semantic title

## Code Standards

- Follow Go conventions and use `gofmt`
- Run `make lint` to check for style issues
- Include tests for new functionality
- Update documentation as needed

# Contributing to RoboHub Auth Service

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing to the project.

## Development Setup

### Prerequisites

- Go 1.22 or later
- Docker and Docker Compose
- golangci-lint (optional, for local linting)

### Getting Started

1. **Clone the repository**
   ```bash
   git clone https://github.com/YOUR_ORG/auth-service.git
   cd auth-service
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up environment**
   ```bash
   export ROBOHUB_JWT_SECRET="dev-secret-for-local-testing"
   ```

4. **Run tests**
   ```bash
   make test
   ```

5. **Run the service locally**
   ```bash
   make run
   # or
   docker compose up --build
   ```

## Development Workflow

### Before Making Changes

1. Create a new branch from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/your-bug-fix
   ```

2. Make sure tests pass:
   ```bash
   make test
   ```

### Making Changes

1. **Write tests first** (TDD approach recommended)
   - Add test cases for new functionality
   - Ensure tests fail before implementation
   
2. **Implement your changes**
   - Follow Go conventions and best practices
   - Keep functions small and focused
   - Add comments for exported functions/types
   
3. **Run tests frequently**
   ```bash
   go test ./internal/yourpackage -v
   ```

4. **Format your code**
   ```bash
   make fmt
   ```

5. **Run linter**
   ```bash
   make lint
   ```

### Testing Guidelines

- Write unit tests for all new functions
- Aim for >80% code coverage
- Use table-driven tests for multiple scenarios
- Mock external dependencies (see `internal/oidc/fake.go` for example)
- Test error cases, not just happy paths

**Example test structure:**
```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "result", false},
        {"empty input", "", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("expected error=%v, got error=%v", tt.wantErr, err)
            }
            if got != tt.want {
                t.Errorf("expected %v, got %v", tt.want, got)
            }
        })
    }
}
```

### Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting (enforced by CI)
- Use meaningful variable names
- Keep functions under 50 lines when possible
- Prefer explicit error handling over panic
- Use context for cancellation and timeouts

**Naming conventions:**
- Packages: lowercase, single word
- Files: lowercase with underscores (`rate_limiter.go`)
- Functions: CamelCase for exported, camelCase for unexported
- Constants: CamelCase or ALL_CAPS for exported

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `chore`: Maintenance tasks
- `ci`: CI/CD changes

**Examples:**
```
feat(oidc): add support for custom JWKS endpoints

fix(ratelimit): prevent race condition in limiter cache

docs(readme): update deployment instructions

test(policy): add tests for allowlist precedence
```

### Pull Request Process

1. **Update documentation** if needed
   - Update README.md for new features
   - Add/update inline comments
   - Update QUICKSTART.md if commands change

2. **Ensure all checks pass**
   - All tests pass locally: `make test`
   - Linter passes: `make lint`
   - Code is formatted: `make fmt`
   - Coverage is maintained (>80%)

3. **Create Pull Request**
   - Use a clear, descriptive title
   - Reference any related issues
   - Describe what changed and why
   - Include screenshots/examples if relevant
   - Add testing instructions

4. **PR Template:**
   ```markdown
   ## Description
   Brief description of changes
   
   ## Type of Change
   - [ ] Bug fix
   - [ ] New feature
   - [ ] Breaking change
   - [ ] Documentation update
   
   ## Testing
   - [ ] Unit tests added/updated
   - [ ] Integration tests pass
   - [ ] Manual testing completed
   
   ## Checklist
   - [ ] Code follows style guidelines
   - [ ] Self-review completed
   - [ ] Comments added for complex logic
   - [ ] Documentation updated
   - [ ] No new warnings
   - [ ] Tests pass locally
   ```

5. **Address review feedback**
   - Respond to all comments
   - Make requested changes
   - Re-request review when ready

### CI/CD Pipeline

Every pull request triggers the CI pipeline which runs:

1. **Tests** - All unit and integration tests
2. **Linting** - Code quality and style checks
3. **Security** - Gosec security scanner
4. **Build** - Cross-platform binary builds
5. **Coverage** - Minimum 80% required

All checks must pass before merging.

## Project Structure

```
auth-service/
├── cmd/robohub-auth/          # Main application
├── internal/                  # Private packages
│   ├── config/               # Configuration
│   ├── httpapi/              # HTTP handlers
│   ├── oidc/                 # OIDC verification
│   ├── policy/               # Access policies
│   ├── ratelimit/            # Rate limiting
│   ├── token/                # Token operations
│   └── types/                # Shared types
├── examples/                  # Usage examples
├── .github/                   # GitHub Actions
└── [config files]
```

### Adding a New Package

1. Create package directory under `internal/`
2. Add `package_name.go` with implementation
3. Add `package_name_test.go` with tests
4. Export only necessary types/functions
5. Document exported items
6. Update README if user-facing

### Adding a New Endpoint

1. Define request/response types in `internal/types/`
2. Add handler in `internal/httpapi/server.go`
3. Add route in `setupRouter()`
4. Write handler tests in `server_test.go`
5. Update API documentation in README.md

## Common Development Tasks

### Running locally for development
```bash
# With auto-reload (use air or similar)
export ROBOHUB_JWT_SECRET="dev-secret"
go run cmd/robohub-auth/main.go

# With Docker Compose
docker compose up --build
```

### Running specific tests
```bash
# Single package
go test ./internal/policy -v

# Single test
go test ./internal/policy -run TestEnforcer_Evaluate

# With race detection
go test -race ./...
```

### Debugging
```bash
# Run with verbose logging
ROBOHUB_JWT_SECRET=secret go run cmd/robohub-auth/main.go

# Use delve debugger
dlv debug cmd/robohub-auth/main.go
```

### Benchmarking
```bash
# Run benchmarks
go test -bench=. -benchmem ./internal/ratelimit
```

### Updating dependencies
```bash
# Update all dependencies
go get -u ./...
go mod tidy

# Update specific dependency
go get github.com/go-chi/chi/v5@latest
go mod tidy
```

## Security

### Reporting Security Issues

**DO NOT** create public GitHub issues for security vulnerabilities.

Instead, email security@example.com with:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Security Best Practices

- Never commit secrets or credentials
- Use environment variables for configuration
- Validate all user input
- Follow the principle of least privilege
- Keep dependencies updated
- Run security scanner locally: `gosec ./...`

## Getting Help

- **Documentation**: Start with README.md and QUICKSTART.md
- **Issues**: Check existing [GitHub Issues](https://github.com/YOUR_ORG/auth-service/issues)
- **Discussions**: Use [GitHub Discussions](https://github.com/YOUR_ORG/auth-service/discussions)
- **Contact**: Reach out to maintainers

## Code Review Guidelines

### For Authors
- Keep PRs focused and reasonably sized (<500 lines)
- Respond to feedback promptly
- Don't take feedback personally
- Explain your reasoning when disagreeing

### For Reviewers
- Be respectful and constructive
- Explain the "why" behind suggestions
- Distinguish between blocking issues and suggestions
- Approve when ready, even if minor suggestions remain

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

## Questions?

Feel free to open an issue or start a discussion if you have questions about contributing!

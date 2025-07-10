# Contributing to Hexabase KaaS

Thank you for your interest in contributing to Hexabase KaaS! This document provides guidelines and information for contributors.

## 🚀 Getting Started

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- PostgreSQL (for local development)
- Git

### Development Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/hexabase/hexabase-ai.git
   cd hexabase-ai
   ```

2. **Start development environment:**
   ```bash
   make init
   make docker-up
   ```

3. **Verify setup:**
   ```bash
   curl http://localhost:8080/health
   ```

## 🧪 Testing

We follow Test-Driven Development (TDD) principles with comprehensive test coverage.

### Running Tests

```bash
# All tests
make test

# API tests only
make api-test

# Specific test suites
cd api
go test ./internal/auth/... -v
go test ./internal/api/... -v
```

### Test Requirements

- **Write tests first** before implementing features
- Ensure **100% test coverage** for new endpoints
- Include **integration tests** for complete workflows
- Test **error scenarios** and edge cases

### Current Test Coverage

- ✅ **Authentication System**: OAuth/OIDC flows, JWT validation
- ✅ **Organizations API**: CRUD operations with role-based access
- ✅ **Security Features**: CSRF protection, token validation
- ✅ **Database Operations**: GORM models and relationships

## 📋 Development Guidelines

### Code Style

- Follow standard Go conventions
- Use `gofmt` and `golint`
- Write clear, descriptive variable and function names
- Include comprehensive comments for public APIs

### Commit Messages

Use conventional commit format:

```
feat: add workspace creation API
fix: resolve JWT token validation issue
docs: update API documentation
test: add integration tests for organizations
```

### Branch Naming

- `feat/feature-name` - New features
- `fix/bug-description` - Bug fixes
- `docs/documentation-update` - Documentation updates
- `test/test-improvements` - Test additions/improvements

## 🏗️ Architecture

### Project Structure

```
hexabase-ai/
├── api/                    # Go API service
│   ├── cmd/               # Entry points (api, worker)
│   ├── internal/          # Internal packages
│   │   ├── api/          # HTTP handlers & routes
│   │   ├── auth/         # Authentication & OIDC
│   │   ├── config/       # Configuration management
│   │   ├── db/           # Database models & migrations
│   │   ├── service/      # Business logic
│   │   └── k8s/          # vCluster management
│   └── Dockerfile
├── ui/                    # Next.js frontend (planned)
├── deployments/           # Infrastructure as Code
├── docs/                  # Documentation
└── scripts/              # Development scripts
```

### Design Principles

- **Microservices Architecture**: Separate concerns into focused services
- **Clean Architecture**: Domain logic separated from infrastructure
- **Test-Driven Development**: Comprehensive test coverage
- **Security First**: Authentication, authorization, and data protection
- **Cloud Native**: Kubernetes-native design patterns

## 🛠️ Available Commands

```bash
# Development
make dev              # Start development environment
make build           # Build all binaries
make clean           # Clean up containers and data

# Testing
make test            # Run all tests
make api-test        # Run API tests
make test-coverage   # Generate coverage report

# Docker
make docker-up       # Start all services
make docker-down     # Stop all services
make docker-logs     # View service logs

# Database
make db-migrate      # Run database migrations
make db-reset        # Reset database
```

## 📚 Pull Request Process

1. **Fork the repository** and create a feature branch
2. **Write tests** for your changes
3. **Implement** the feature/fix
4. **Ensure all tests pass**: `make test`
5. **Update documentation** if needed
6. **Submit a pull request** with:
   - Clear description of changes
   - Link to related issues
   - Test coverage report
   - Screenshots for UI changes

### Pull Request Checklist

- [ ] Tests added for new functionality
- [ ] All existing tests pass
- [ ] Documentation updated
- [ ] Code follows project conventions
- [ ] No breaking changes (or clearly documented)
- [ ] Security considerations addressed

## 🔒 Security

### Reporting Security Issues

Please report security vulnerabilities to: security@hexabase.com

**Do not** open public issues for security vulnerabilities.

### Security Guidelines

- Never commit secrets, API keys, or passwords
- Use environment variables for configuration
- Follow OAuth 2.0 and OIDC best practices
- Implement proper input validation
- Use HTTPS in production

## 🌟 Areas for Contribution

### High Priority

- **Frontend Development**: Next.js UI implementation
- **vCluster Integration**: Kubernetes workspace management
- **Billing System**: Stripe integration
- **Monitoring**: Observability and metrics

### Documentation

- API documentation improvements
- Tutorial creation
- Architecture diagrams
- Deployment guides

### Testing

- E2E test automation
- Performance testing
- Security testing
- Load testing

## 💬 Community

- **GitHub Issues**: Bug reports and feature requests
- **Discussions**: Design discussions and questions
- **Discord**: Real-time chat (coming soon)

## 📝 License

By contributing, you agree that your contributions will be licensed under the MIT License.

## 🙏 Recognition

Contributors will be recognized in:
- README.md contributors section
- Release notes
- Project documentation

Thank you for helping make Hexabase KaaS better! 🚀
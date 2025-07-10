# Hexabase AI - Project Structure Guide

This guide defines the organizational rules and conventions for the Hexabase AI codebase to ensure consistency, maintainability, and clarity across all repositories.

## Repository Structure Overview

```
hexabase-ai/
├── api/                      # Go API backend
├── ui/                       # Next.js frontend
├── ai-ops/                   # AI operations service
├── deployments/              # Kubernetes & deployment configs
├── ci/                       # CI/CD configurations
├── docs/                     # Project documentation
├── scripts/                  # Utility scripts
├── CLAUDE.md                 # AI assistant instructions
├── STRUCTURE_GUIDE.md        # This file
└── docker-compose.yml        # Local development setup
```

## Directory Organization Rules

### 1. API Directory (`/api`)

```
api/
├── cmd/                      # Application entry points
│   ├── api/                  # Main API server
│   └── worker/               # Background worker
├── internal/                 # Private application code
│   ├── api/                  # HTTP layer
│   │   ├── handlers/         # Request handlers
│   │   ├── middleware/       # HTTP middleware
│   │   └── routes/           # Route definitions
│   ├── domain/               # Business logic layer
│   │   └── [domain]/         # Domain-specific logic
│   │       ├── models.go     # Domain models
│   │       ├── repository.go # Repository interface
│   │       ├── service.go    # Business logic
│   │       ├── provider.go   # Provider interface (if applicable)
│   │       └── types.go      # Common types
│   ├── repository/           # Data access layer
│   │   └── [domain]/         # Domain-specific implementations
│   │       ├── postgres.go   # PostgreSQL implementation
│   │       ├── [provider]/   # Provider implementations (e.g., fission/, knative/)
│   │       │   └── provider.go
│   │       ├── factory.go    # Provider factory (if multi-provider)
│   │       └── config.go     # Configuration management
│   ├── service/              # Service implementations
│   │   └── [domain]/         # Domain-specific services
│   ├── infrastructure/       # Infrastructure code
│   │   └── wire/             # Dependency injection
│   ├── config/               # Configuration management
│   ├── db/                   # Database utilities
│   ├── auth/                 # Authentication/authorization
│   └── [shared]/             # Shared utilities (logging, etc.)
├── testresults/              # ALL test results and reports
│   ├── coverage/             # Coverage reports by timestamp
│   ├── unit/                 # Unit test results
│   ├── logs/                 # Test execution logs
│   ├── summary/              # Summary reports
│   ├── reports/              # Consolidated reports
│   └── COMPREHENSIVE_TEST_REPORT.md
├── tests/                    # Test files
│   ├── integration/          # Integration tests
│   └── e2e/                  # End-to-end tests
└── go.mod                    # Go module definition
```

**Rules for API:**
- All private code must be in `internal/`
- Follow DDD structure: domain → repository → service
- One domain per directory
- Test results MUST go in `/api/testresults/`
- Never place test reports in `/docs/`

### 2. UI Directory (`/ui`)

```
ui/
├── src/
│   ├── app/                  # Next.js app directory
│   ├── components/           # React components
│   ├── hooks/                # Custom React hooks
│   └── lib/                  # Utility functions
├── tests/                    # Playwright tests
├── public/                   # Static assets
└── screenshots/              # Test screenshots
```

**Rules for UI:**
- Components must be self-contained
- Hooks for shared state logic
- Tests alongside implementation
- Screenshots in dedicated directory

### 3. Documentation (`/docs`)

```
docs/
├── README.md                 # Documentation index
├── architecture/             # System design docs
├── api-reference/            # API documentation
├── development/              # Developer guides
├── getting-started/          # User guides
├── implementation-summaries/ # Implementation notes
├── operations/               # Ops guides
├── project-management/       # PM documentation
└── testing/                  # Testing guides ONLY (no results)
```

**Rules for Documentation:**
- NO test results or coverage reports
- Only guides, references, and design docs
- Implementation summaries for completed features
- Clear separation between guides and results

### 4. Deployments (`/deployments`)

```
deployments/
├── helm/                     # Helm charts
├── k8s/                      # Raw Kubernetes manifests
├── monitoring/               # Monitoring configs
├── gitops/                   # GitOps configurations
└── policies/                 # Security policies
```

**Rules for Deployments:**
- Environment-specific values in helm/
- Base manifests in k8s/
- Monitoring separate from application

### 5. CI/CD (`/ci`)

```
ci/
├── github-actions/           # GitHub Actions workflows
├── gitlab-ci/                # GitLab CI configs
└── tekton/                   # Tekton pipelines
```

## File Naming Conventions

### Go Files
- Lowercase with underscores: `auth_handler.go`
- Test files: `auth_handler_test.go`
- Models: `models.go` or `[domain]_models.go`
- Interfaces in: `repository.go` or `service.go`

### Documentation
- Uppercase with hyphens: `API-REFERENCE.md`
- Summaries: `[FEATURE]-SUMMARY.md`
- Guides: `[topic]-guide.md`

### Test Results
- Timestamp format: `YYYYMMDD_HHMMSS`
- Coverage: `[package].coverage`
- Logs: `[package].log`
- JSON results: `[package].json`

## Code Organization Rules

### 1. Domain-Driven Design
- Each domain owns its models, repository interface, and service interface
- Repository implementations separate from domain
- Service implementations separate from domain
- Clear boundaries between domains

### 2. Dependency Direction
```
Handlers → Services → Repositories → Database
    ↓          ↓            ↓
  Domain    Domain       Domain
```
- Dependencies flow inward
- Domain has no external dependencies
- Interfaces defined in domain layer

### 3. Testing Structure
- Unit tests alongside code
- Integration tests in `/tests/integration/`
- E2E tests in `/tests/e2e/` or `/ui/tests/`
- ALL results in `/api/testresults/`

## Migration Rules

### When Moving Files
1. Update all imports
2. Update documentation references
3. Update CI/CD paths
4. Commit with clear message: "refactor: move [what] from [where] to [where]"

### When Consolidating
1. Preserve historical data in timestamped directories
2. Create summary documents
3. Update index/navigation files
4. Remove redundant files after verification

## Prohibited Practices

### Never Do This
1. ❌ Place test results in `/docs/`
2. ❌ Mix test results with test code
3. ❌ Create circular dependencies
4. ❌ Put implementation in interface files
5. ❌ Mix domains in single directory
6. ❌ Use relative imports in Go
7. ❌ Commit generated files (except wire_gen.go)
8. ❌ Store secrets in code

### Always Do This
1. ✅ Keep test results in `/api/testresults/`
2. ✅ Separate concerns by layer
3. ✅ Use interfaces for dependencies
4. ✅ Document public APIs
5. ✅ Follow naming conventions
6. ✅ Update structure guide when adding new patterns

## New Feature Checklist

When adding a new feature:

1. **Domain Layer**
   - [ ] Create domain directory
   - [ ] Define models.go
   - [ ] Define repository.go interface
   - [ ] Define service.go interface

2. **Repository Layer**
   - [ ] Implement repository interface
   - [ ] Add unit tests

3. **Service Layer**
   - [ ] Implement service interface
   - [ ] Add unit tests
   - [ ] Handle business logic

4. **API Layer**
   - [ ] Create handler
   - [ ] Define routes
   - [ ] Add middleware if needed
   - [ ] Add integration tests

5. **Documentation**
   - [ ] Update API reference
   - [ ] Add implementation summary
   - [ ] Update architecture if needed

6. **Testing**
   - [ ] Unit tests >80% coverage
   - [ ] Integration tests for API
   - [ ] E2E tests for critical paths
   - [ ] Results in `/api/testresults/`

## Maintenance Guidelines

### Regular Tasks
- Weekly: Clean up old test results (keep last 10 runs)
- Monthly: Review and update structure guide
- Quarterly: Audit directory structure compliance

### Before Major Releases
- Consolidate test reports
- Update all documentation
- Verify structure compliance
- Clean up deprecated code

## Questions?

If unclear about where something belongs:
1. Check this guide
2. Look for similar existing code
3. Ask in team chat
4. Update guide after decision

---

**Last Updated**: 2025-01-08  
**Version**: 1.0.0  
**Maintainer**: Development Team
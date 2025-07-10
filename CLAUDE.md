# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## ⚠️ CRITICAL DEVELOPMENT REQUIREMENTS

### 1. MANDATORY: Follow STRUCTURE_GUIDE.md

- **ALWAYS** check `/STRUCTURE_GUIDE.md` before creating or organizing files
- **NEVER** place test results in `/docs/` - they MUST go in `/api/testresults/`
- **ALWAYS** follow the domain-driven design structure: domain → repository → service
- **NEVER** mix domains in a single directory

### 2. MANDATORY: Test-Driven Development (TDD)

- **ALWAYS** write tests FIRST before implementing features
- **ALWAYS** check coverage and aim for 90%
- **NEVER** implement features without tests
- **ALWAYS** follow the TDD cycle: Red → Green → Refactor
- See examples in `/api/internal/repository/application/cronjob_test.go` for proper TDD approach

### 3. MANDATORY: Code Quality Checks for Go Development

- **ALWAYS** run `make lint-api` after completing Go code changes in the `/api` directory
- **TIMING**: Run lint checks at these critical points:
  - After implementing a feature (after all tests pass)
  - Before marking any task as completed
  - Before creating any pull request
  - After resolving merge conflicts
- **NEVER** consider Go code work complete without passing lint checks
- **FIX** all linting issues before proceeding to next tasks

## Project Overview

Hexabase AI is a multi-tenant Kubernetes as a Service platform built on K3s and vCluster. The project provides a complete PaaS solution with support for various application types including stateless apps, stateful services, CronJobs, and serverless functions.

## Architecture Context

- check docs under `/docs/architecture`

### Core Components

- **Control Plane**: Go-based API server handling tenant management, OIDC auth, and vCluster orchestration
- **UI**: Next.js frontend with state management and real-time updates
- **Infrastructure**: Host K3s cluster with vCluster for tenant isolation
- **Data Layer**: PostgreSQL (primary), Redis (cache), NATS (message queue)
- **Monitoring**: Prometheus, Grafana, Loki, ClickHouse for metrics storage
- **Security**: Kyverno policies, Trivy scanning, Falco runtime monitoring
- **AI Operations**: Integrated AI agent support with LangFuse tracking and Ollama models
- **Backup System**: Proxmox-integrated backup solution for Dedicated Plan workspaces
- **CI/CD**: Tekton-based pipeline system with GitHub Actions integration

### Key Concepts Mapping

| Hexabase Concept | Kubernetes Equivalent                         | Description                                        |
| ---------------- | --------------------------------------------- | -------------------------------------------------- |
| Organization     | (none)                                        | Billing and organization management unit           |
| Workspace        | vCluster                                      | Tenant isolation boundary (Shared/Dedicated Plans) |
| Project          | Namespace                                     | Resource isolation within Workspace                |
| Application      | Deployment/StatefulSet/CronJob/KnativeService | Deployed workload                                  |
| Workspace Member | OIDC Subject                                  | Technical user with vCluster access                |
| Workspace Group  | OIDC Claim                                    | Permission assignment unit                         |
| Node             | Proxmox VM                                    | Physical compute resource (Dedicated Plan)         |
| Backup Storage   | Proxmox Storage                               | Backup destination (Dedicated Plan)                |
| CronJob          | Kubernetes CronJob                            | Scheduled tasks with execution tracking            |
| Function         | Knative Service                               | Serverless functions with versioning               |

### Application Types Supported

1. **Stateless Applications**: Standard Kubernetes Deployments
2. **Stateful Applications**: StatefulSets with persistent storage
3. **CronJobs**: Scheduled tasks with execution tracking
4. **Serverless Functions**: Knative-based functions with versioning
5. **AI Agents**: Python/Node.js functions with AI model access

## Development Commands

### For Go API Development

```bash
# ALWAYS run tests first (TDD approach)
go test ./internal/domain/[domain] -v      # Test domain logic
go test ./internal/repository/[domain] -v  # Test repository
go test ./internal/service/[domain] -v     # Test service

# Run coverage tests
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Development workflow
go mod download          # Download dependencies
go test ./...           # Run all tests
go run cmd/api/main.go  # Run API server locally
go build -o hexabase-api cmd/api/main.go  # Build binary

# Run tests with coverage reports (output to /api/testresults/)
./run_tests_with_coverage.sh
```

### For Next.js UI Development

```bash
# When working on the UI repository
npm install             # Install dependencies
npm run dev            # Run development server
npm run build          # Build for production
npm test               # Run tests
npm run lint           # Run linting
npx playwright test    # Run E2E tests
```

### For Kubernetes/Helm Development

```bash
# Deploy Hexabase AI using Helm
helm install hexabase-ai ./deployments/helm/hexabase-ai -f values.yaml

# Common kubectl commands for debugging
kubectl get vcluster -A                    # List all vClusters
kubectl get pods -n hexabase-control-plane # List control plane pods
kubectl logs -n hexabase-control-plane <pod-name> # View pod logs
kubectl get cronjobs -n <project-namespace> # List CronJobs
kubectl get jobs -n <project-namespace>     # List Jobs from CronJobs
```

## Implementation Guidelines

### API Development (Go) - TDD Approach

#### 1. Test First (Red Phase)

```go
// Write failing test first
func TestRepository_CreateBackupStorage(t *testing.T) {
    // Setup test database
    gormDB, mock := setupTestDB(t)
    repo := NewPostgresRepository(gormDB)

    // Define expected behavior
    storage := &backup.BackupStorage{
        WorkspaceID: "ws-123",
        Name: "test-backup",
        Type: backup.StorageTypeProxmox,
    }

    // Mock expectations
    mock.ExpectBegin()
    mock.ExpectQuery(`INSERT INTO "backup_storages"`).
        WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("bs-123"))
    mock.ExpectCommit()

    // Execute and assert
    err := repo.CreateBackupStorage(ctx, storage)
    assert.NoError(t, err)
    assert.Equal(t, "bs-123", storage.ID)
}
```

#### 2. Implementation (Green Phase)

```go
// Implement just enough to make test pass
func (r *PostgresRepository) CreateBackupStorage(ctx context.Context, storage *backup.BackupStorage) error {
    dbStorage := r.domainToDBStorage(storage)
    if err := r.db.WithContext(ctx).Create(dbStorage).Error; err != nil {
        return fmt.Errorf("failed to create backup storage: %w", err)
    }
    *storage = *r.dbToDomainStorage(dbStorage)
    return nil
}
```

#### 3. Refactor

- Improve code structure while keeping tests green
- Extract common patterns
- Add comprehensive error handling

### Domain-Driven Design Structure

```
api/internal/
├── domain/           # Business logic layer (interfaces only)
│   └── backup/
│       ├── models.go      # Domain models
│       ├── repository.go  # Repository interface
│       └── service.go     # Service interface
├── repository/       # Data access implementations
│   └── backup/
│       ├── postgres.go    # PostgreSQL implementation
│       └── proxmox.go     # Proxmox API implementation
├── service/          # Business logic implementations
│   └── backup/
│       ├── service.go     # Service implementation
│       └── executor.go    # Backup execution logic
└── api/handlers/     # HTTP handlers
    └── backup.go     # REST API endpoints
```

### UI Development (Next.js)

- Component-based architecture with clear separation of concerns
- Type-safe API client generation from OpenAPI specs
- Real-time updates via WebSocket for status changes
- Implement proper loading states and error boundaries
- Test components with React Testing Library

### vCluster Management

- Use vcluster CLI or Kubernetes API for lifecycle management
- Apply OIDC configuration for each vCluster
- Set ResourceQuotas based on Workspace Plan (Shared/Dedicated)
- Configure HNC (Hierarchical Namespace Controller) for project hierarchy
- Implement proper cleanup on deletion

### CronJob Implementation

- Use Kubernetes batch/v1 CronJob resources
- Track executions in PostgreSQL
- Support template-based CronJobs
- Provide execution history and logs
- Enable/disable scheduling without deletion

### Function Implementation

- Deploy functions as Knative Services
- Version management with atomic deployments
- Support multiple trigger types (HTTP, Event, Schedule)
- Track invocations and performance metrics
- Integrate with CronJob for scheduled functions

### Backup Implementation (Dedicated Plan Only)

- Plan-based feature gating at database level
- Proxmox storage integration for backup volumes
- Support multiple backup types (Full, Incremental, Application)
- Encryption and compression options
- Retention policy management
- Integration with CronJob for scheduled backups

### Async Processing

- Use NATS for task queuing
- Implement retry logic with exponential backoff
- Store task status in PostgreSQL
- Notify completion/failure via NATS
- Use background workers for long-running operations

## Security Considerations

- Never expose or log secrets/API keys
- All external IdP auth via OIDC
- Enforce Kyverno policies for compliance
- Regular vulnerability scanning with Trivy
- Runtime threat detection with Falco
- Encrypt backups with workspace-specific keys
- Validate all user inputs at handler level

## Testing Strategy

### Unit Tests

- Test each component in isolation
- Mock external dependencies
- Aim for >80% code coverage
- Use table-driven tests for multiple scenarios

### Integration Tests

- Test database operations with real transactions
- Test API endpoints with httptest
- Verify Kubernetes resource creation
- Test with real Redis/NATS connections

### E2E Tests

- Test complete user workflows
- Use Playwright for UI testing
- Test vCluster provisioning end-to-end
- Verify backup/restore operations

## Common Patterns

### Error Handling

```go
// Always wrap errors with context
if err := repo.Create(ctx, model); err != nil {
    return fmt.Errorf("failed to create %s: %w", resourceType, err)
}
```

### Logging

```go
// Use structured logging with context
logger.Info("creating backup storage",
    "workspaceID", storage.WorkspaceID,
    "storageType", storage.Type,
    "capacityGB", storage.CapacityGB)
```

### Resource Naming

- Applications: `app-{uuid}`
- CronJobs: Use application ID
- Executions: `cje-{uuid}` (CronJob), `fi-{uuid}` (Function)
- Backups: `bs-{uuid}` (Storage), `bp-{uuid}` (Policy)

## Migration from Legacy System

- Gradual migration from github.com/b-eee/apicore
- Maintain API compatibility during transition
- Document all breaking changes
- Provide migration scripts for data

## Performance Considerations

- Use database indexes for frequently queried fields
- Implement pagination for list operations
- Cache frequently accessed data in Redis
- Use batch operations where possible
- Monitor query performance with explain analyze

## Monitoring and Observability

- Export Prometheus metrics for all operations
- Use OpenTelemetry for distributed tracing
- Log all API requests with correlation IDs
- Track SLIs: latency, error rate, throughput
- Store metrics in ClickHouse for long-term analysis

## Architecture Decision Records (ADRs)

### Creating New ADRs

When creating or updating Architecture Decision Records, follow this standard format:

```markdown
# ADR-XXX: [Title]

**Date**: YYYY-MM-DD  
**Status**: [Proposed|Accepted|Implemented|Deprecated]  
**Authors**: [Team/Person]

## 1. Background

[Context and problem statement - why this decision is needed]

## 2. Status

[Current state of the decision/implementation]

## 3. Other Options Considered

[List and briefly describe alternative solutions]

## 4. What Was Decided

[Clear statement of the chosen solution]

## 5. Why Did You Choose It?

[Rationale and benefits of the chosen solution]

## 6. Why Didn't You Choose the Other Options?

[Specific reasons for rejecting each alternative]

## 7. What Has Not Been Decided

[Open questions and future decisions]

## 8. Considerations

[Implementation details, security, performance, costs, etc.]
```

### ADR Guidelines

1. **Location**: Place ADRs in `/docs/adr/`
2. **Naming**: Use format `XXX-descriptive-name.md` (e.g., `009-caching-strategy.md`)
3. **Updates**: Update status and add notes rather than deleting old ADRs
4. **Linking**: Reference ADRs in code comments where relevant
5. **Review**: All ADRs should be reviewed by at least one other team member

### Current ADRs

See `/docs/adr/README.md` for the complete index of architectural decisions.

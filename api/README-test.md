# API Testing Guide

## Overview

This document describes the testing approach and configuration for the Hexabase AI API. The test suite has been refactored to use TestContainers for test code with real PostgreSQL and Redis instances, following a transaction-based isolation pattern.

## Why This Approach?

### Problems with Traditional Mock-Based Testing
- Divergence between mock behavior and actual database behavior
- Difficulty testing complex transaction processing
- Inability to test database-specific features (constraints, triggers, indexes)
- Unexpected bugs discovered during integration

### Benefits of TestContainers Approach
1. **Real Database Environment**: Uses the same PostgreSQL 15 as production
2. **Fast Execution**: Template databases and transaction rollback for speed
3. **Complete Isolation**: Each test runs in its own environment
4. **Simple Setup**: Containers start once per package in TestMain
5. **Reliable Cleanup**: Automatic container removal and resource cleanup

## Test Environment Configuration

### Testcontainers and Ryuk

The test suite uses [Testcontainers](https://www.testcontainers.org/) for test code with real databases and services. Testcontainers automatically manages Docker containers for testing purposes.

#### TESTCONTAINERS_RYUK_DISABLED

Ryuk is a helper container that Testcontainers uses to clean up test containers automatically. It ensures that all containers created during tests are removed, even if the test process crashes or is forcefully terminated.

**Environment Variable Options:**

- `TESTCONTAINERS_RYUK_DISABLED=false` (default, recommended)
  - Enables Ryuk for automatic container cleanup
  - Provides safety against orphaned containers
  - Recommended for CI/CD environments

- `TESTCONTAINERS_RYUK_DISABLED=true`
  - Disables Ryuk
  - Useful in environments where Ryuk cannot run (e.g., some Docker-in-Docker setups)
  - Requires manual cleanup if tests fail unexpectedly

**Setting the Environment Variable:**

```bash
# Enable Ryuk (recommended)
export TESTCONTAINERS_RYUK_DISABLED=false
go test ./...

# Or inline
TESTCONTAINERS_RYUK_DISABLED=false go test ./...

# Disable Ryuk (only if necessary)
TESTCONTAINERS_RYUK_DISABLED=true go test ./...
```

**Note:** The current implementation reuses containers, so TestMain doesn't explicitly terminate them. Containers will eventually be cleaned up by Docker or Testcontainers.

## Running Tests

### Unit Tests

```bash
# Run all unit tests
go test ./...

# Run tests for a specific package
go test ./internal/auth/service

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...
```

### Test Code

Test code uses Testcontainers and requires Docker to be running:

```bash
# Run test code (skips short tests)
go test ./... -short=false

# Run specific test
go test ./internal/auth/service -run TestService_GetAuthURL -v
```

### Test Coverage

Generate comprehensive test coverage reports:

```bash
# Run the coverage script
./tests/run_tests_with_coverage.sh

# Results are saved in:
# - testresults/unit/TIMESTAMP/          # Unit test results
# - testresults/coverage/TIMESTAMP/      # Coverage reports
# - testresults/logs/TIMESTAMP/          # Test execution logs
# - testresults/summary/                 # Consolidated summaries
```

## Test Architecture

### Test Infrastructure

The latest test infrastructure has the following features:

- **TestMain**: Starts PostgreSQL container once per package and initializes Miniredis
- **WithTestDB**: Provides each test with its own independent database (copied from template DB)
- **setupTestServiceWithDB**: Builds test environment using real repositories and services

### Key Test Patterns

#### Database Isolation
```go
// Each test runs in its own database
withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
    // After test completion, the entire database is dropped
    // Ensures complete isolation
})
```

#### External Black-Box Testing
```go
package service_test // Not package service
// Tests only use public APIs
```

#### Real Service Construction
```go
func setupTestServiceWithDB(t *testing.T, db *gorm.DB, redisClient *internalRedis.Client) (domain.Service, *MockTokenDomainService) {
    // Create real repository components
    postgresRepo := repository.NewPostgresRepository(db)
    redisAuthRepo := repository.NewRedisAuthRepository(redisClient)
    
    // Build real service
    svc := service.NewService(...)
    return svc, tokenDomainService
}
```

### Testing Approach

1. **Real Database Usage**: All tests use real PostgreSQL 15 and Miniredis
2. **Database Isolation**: Each test uses its own database (copied from template DB)
3. **Fast Execution**: Template database means migrations run only once
4. **Minimal Mocking**: Only external services (OAuth, Token Domain Service) are mocked
5. **Real Implementation Testing**: Repository and service layers use actual implementations

### Container Management

Current implementation features:

1. **Container Reuse**: `Reuse: true` option allows container reuse across multiple test runs
2. **No Explicit Termination**: TestMain doesn't terminate containers, delegating management to Docker
3. **Miniredis Cleanup**: FlushAll before each test to reset state
4. **Database Deletion**: Created databases are reliably dropped after each test

## Best Practices

1. **Always run tests with Docker available** for TestContainers
2. **Enable Ryuk** for reliable cleanup (default behavior)
3. **Use `testing.Short()` flag** to skip test code when needed
4. **Follow TDD**: Write tests first, then implementation
5. **Use WithTestDB pattern** for all database tests
6. **Prefer external tests**: Use `package service_test` instead of `package service`
7. **Run lint checks**: Always run `make lint-api` before committing

### ⚠️ Important Note on WithTestDB Usage

**Do not use subtests (t.Run) inside WithTestDB.**

#### Why This Matters
- WithTestDB creates an isolated database for each test, but subtests share the same DB instance as their parent
- Redis state is not cleared between subtests
- Concurrent subtests can cause data conflicts and make tests flaky

#### ❌ Bad Example
```go
func TestExample(t *testing.T) {
    withTestDB(t, func(db *gorm.DB, redisClient *redis.Client) {
        t.Run("subtest1", func(t *testing.T) {
            // Avoid this pattern
        })
        t.Run("subtest2", func(t *testing.T) {
            // Data conflicts may occur
        })
    })
}
```

#### ✅ Good Example
```go
func TestExample_Scenario1(t *testing.T) {
    withTestDB(t, func(db *gorm.DB, redisClient *redis.Client) {
        // Test logic here
    })
}

func TestExample_Scenario2(t *testing.T) {
    withTestDB(t, func(db *gorm.DB, redisClient *redis.Client) {
        // Test logic here
    })
}
```

Each test function should have its own WithTestDB call to ensure complete isolation.

## Troubleshooting

### Containers Not Cleaned Up

If containers remain after tests:

1. Check if Ryuk is enabled: `echo $TESTCONTAINERS_RYUK_DISABLED`
2. Check for postgres containers: `docker ps -a | grep postgres-hexabase-test-`
3. Manually remove containers: `docker ps -a | grep postgres-hexabase-test- | awk '{print $1}' | xargs docker rm -f`
4. Disable container reuse for testing: Run with `Reuse: false` option

### Slow Tests

Tests are optimized for speed through:

1. **Container Reuse**: Same container used across multiple test runs
2. **Template Database**: Migrations run only once
3. **Miniredis**: Fast in-memory Redis implementation
4. **Independent Databases**: Each test uses its own database, enabling parallel execution if needed

To further speed up development:
1. Use `-short` flag to skip test code
2. Run specific tests: `go test ./internal/auth/service -run TestPKCE`
3. Use `TEST_USE_TRANSACTION=true` for transaction mode (faster but unsuitable for tests with DDL operations)

### Performance Metrics

- Container startup: ~1 second initially, instant on reuse
- Database creation: ~0.1 seconds per test
- Test execution: 0.09-0.24 seconds
- Full package: ~2-3 seconds for auth service

## Example: Auth Service Tests

The auth service tests demonstrate the latest testing approach:

```go
// main_test.go - Package-level setup
func TestMain(m *testing.M) {
    // testutil.SetupTestDatabase handles container management
    code := testutil.SetupTestDatabase(m)
    if code != 0 {
        return
    }
}

// service_test.go - Example using WithTestDB
func TestService_HandleCallback(t *testing.T) {
    ctx := context.Background()
    
    withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
        // Setup service with real repositories
        svc, _ := setupTestServiceWithDB(t, db, redisClient)
        
        // Start OAuth flow
        loginReq := &domain.LoginRequest{Provider: "google"}
        authURL, state, err := svc.GetAuthURL(ctx, loginReq)
        require.NoError(t, err)
        
        // Handle callback
        callbackReq := &domain.CallbackRequest{
            Code:  "auth-code-123",
            State: state,
        }
        response, err := svc.HandleCallback(ctx, callbackReq, "192.168.1.1", "Mozilla/5.0")
        require.NoError(t, err)
        
        // Verify user and session in database
        var user domain.User
        err = db.Where("provider = ?", "google").First(&user).Error
        require.NoError(t, err)
        
        // Verify JWT token
        claims, err := svc.ValidateAccessToken(ctx, response.AccessToken)
        require.NoError(t, err)
        assert.NotEmpty(t, claims.SessionID)
    })
    // Entire database is dropped after test
}
```
# Test Utilities Package

This package provides common test utilities for the Hexabase AI project.

## Usage

### In TestMain

```go
package mypackage_test

import (
    "testing"
    "github.com/hexabase/hexabase-ai/api/internal/shared/testutil"
)

func TestMain(m *testing.M) {
    code := testutil.SetupTestDatabase(m)
    os.Exit(code)
}
```

### In Tests

```go
func TestMyFunction(t *testing.T) {
    testutil.WithTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
        // Your test code here
        // Each test gets its own isolated database
    })
}
```

### Environment Variables

- `TEST_USE_TRANSACTION=true`: Use transaction-based isolation (faster but less isolated)

## Test Isolation Methods

### Template Database Method (Default)

- Each test gets its own database created from a template
- Complete isolation between tests
- Slower (150-300ms overhead per test)
- Better for debugging (can inspect database state)

### Transaction Method (Optional)

- All tests share the same database but use transactions
- Faster execution
- Less isolation (DDL changes affect all tests)
- Enable with `TEST_USE_TRANSACTION=true`

## Best Practices

1. Always use `testutil.WithTestDB` for database tests
2. Use template method (default) for development and CI
3. Use transaction method only for quick local testing
4. Clean up resources properly in test cleanup functions

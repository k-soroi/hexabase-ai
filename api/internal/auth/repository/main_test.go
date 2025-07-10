package repository_test

import (
	"testing"

	"gorm.io/gorm"

	internalRedis "github.com/hexabase/hexabase-ai/api/internal/shared/redis"
	"github.com/hexabase/hexabase-ai/api/internal/shared/testutil"
)

// TestMain sets up test infrastructure once per package
func TestMain(m *testing.M) {
	code := testutil.SetupTestDatabase(m)
	if code != 0 {
		return
	}
}

// withTestDB creates a new database from template for each test
func withTestDB(t *testing.T, fn func(db *gorm.DB, redisClient *internalRedis.Client)) {
	t.Helper()

	testutil.WithTestDB(t, fn)
}

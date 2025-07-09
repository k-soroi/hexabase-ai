package testutil

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/hexabase/hexabase-ai/api/internal/shared/config"
	internalRedis "github.com/hexabase/hexabase-ai/api/internal/shared/redis"
)

var ErrNotPostgresContainer = errors.New("found a container but it is not a postgres container")

const (
	postgresImage         = "postgres:15-alpine"
	postgresUser          = "testuser"
	postgresPassword      = "testpass"
	defaultPostgresDB     = "postgres"
	templateDBNameDefault = "template_testdb"
	migrationsPath        = "file://../../shared/db/migrations"
	waitLogOccurrence     = 2
	waitStartupTimeout    = 30 * time.Second
	randomSuffixLength    = 8
	maxTestNameLength     = 20
)

// TestDatabase provides test database infrastructure
type TestDatabase struct {
	Container      *postgrescontainer.PostgresContainer
	ConnString     string
	TemplateDBName string
	Miniredis      *miniredis.Miniredis
}

// Package-level infrastructure
var (
	testDB                *TestDatabase
	setupOnce             sync.Once
	keepAliveDB           *sql.DB
	postgresContainerName string
)

// generateRandomHexString generates a random hex string of a given length
func generateRandomHexString(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random string for container name: %w", err)
	}

	return hex.EncodeToString(b), nil
}

// SetupTestDatabase initializes the test database infrastructure
// This should be called in TestMain
func SetupTestDatabase(m *testing.M) int {
	ctx := context.Background()

	var setupErr error

	setupOnce.Do(func() {
		// Generate unique container name for this test run
		randomSuffix, err := generateRandomHexString(6) //nolint:mnd
		if err != nil {
			setupErr = fmt.Errorf("failed to generate container name: %w", err)
			return
		}

		postgresContainerName = "postgres-hexabase-test-" + randomSuffix

		testDB = &TestDatabase{
			TemplateDBName: templateDBNameDefault,
		}

		// Get a test container by creating a new one with a unique name for this test run.
		pgContainer, connStr, err := createNewPostgresContainer(ctx)
		if err != nil {
			setupErr = fmt.Errorf("failed to get test container: %w", err)
			return
		}

		testDB.Container = pgContainer
		testDB.ConnString = connStr

		// Create template database with migrations
		if err := createTemplateDatabase(ctx, testDB.ConnString, testDB.TemplateDBName); err != nil {
			setupErr = fmt.Errorf("failed to create template database: %w", err)
			return
		}

		// Open and configure a connection to keep the container alive across tests
		keepAliveDB, err = setupKeepAliveConnection(testDB.ConnString)
		if err != nil {
			setupErr = fmt.Errorf("failed to setup keep-alive connection: %w", err)
			return
		}
	})

	if setupErr != nil {
		panic(setupErr)
	}

	// Initialize miniredis
	mr, err := miniredis.Run()
	if err != nil {
		panic(fmt.Sprintf("failed to start miniredis: %v", err))
	}

	testDB.Miniredis = mr

	// Run tests
	code := m.Run()

	// Cleanup miniredis
	if testDB.Miniredis != nil {
		testDB.Miniredis.Close()
	}

	// Close the keep-alive connection
	if keepAliveDB != nil {
		_ = keepAliveDB.Close()
	}

	// Note: We don't terminate the container here to allow reuse
	// Container will be cleaned up by Docker/testcontainers eventually
	return code
}

// createNewPostgresContainer creates a new postgres container instance.
// It uses a unique container name for each test run to avoid conflicts.
func createNewPostgresContainer(ctx context.Context) (*postgrescontainer.PostgresContainer, string, error) {
	pgContainer, err := postgrescontainer.Run(ctx,
		postgresImage,
		postgrescontainer.WithUsername(postgresUser),
		postgrescontainer.WithPassword(postgresPassword),
		postgrescontainer.WithDatabase(defaultPostgresDB),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(waitLogOccurrence).
				WithStartupTimeout(waitStartupTimeout),
		),
		testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Name: postgresContainerName,
			},
			Reuse: true,
		}),
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to start PostgreSQL container: %w", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		// Attempt to terminate the container we just created
		//nolint:contextcheck
		_ = pgContainer.Terminate(context.Background())
		return nil, "", fmt.Errorf("failed to get connection string from new container: %w", err)
	}

	slog.Info("Started new PostgreSQL container", "name", postgresContainerName)

	return pgContainer, connStr, nil
}

// setupKeepAliveConnection establishes and configures a database connection
// to be kept alive for the duration of the test suite.
func setupKeepAliveConnection(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open keep-alive connection: %w", err)
	}

	// Configure the connection pool to prevent idle timeouts
	// that could cause the testcontainer to be reaped.
	db.SetConnMaxLifetime(time.Minute)
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)

	return db, nil
}

// createTemplateDatabase creates a template database with all migrations applied
func createTemplateDatabase(ctx context.Context, connStr, templateDBName string) error {
	// Connect to postgres database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}

	defer func() {
		_ = db.Close()
	}()

	// Drop template database if exists
	_, _ = db.ExecContext(ctx, "DROP DATABASE IF EXISTS "+templateDBName)

	// Create template database
	_, err = db.ExecContext(ctx, "CREATE DATABASE "+templateDBName)
	if err != nil {
		// If it's a duplicate database error, it's okay, another process created it.
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "42P04" { // 42P04 is duplicate_database
			slog.Info("Template database already exists, proceeding.")
		} else {
			return fmt.Errorf("failed to create template database: %w", err)
		}
	}

	// Connect to template database
	// Parse connection string to replace database name
	templateConnStr := strings.Replace(connStr, "dbname=postgres", "dbname="+templateDBName, 1)

	templateDB, err := sql.Open("postgres", templateConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to template database: %w", err)
	}

	defer func() {
		_ = templateDB.Close()
	}()

	// Apply migrations to template database
	driver, err := postgres.WithInstance(templateDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance(
		migrationsPath,
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("Template database created with migrations")

	return nil
}

// generateTestDBName generates a unique database name for the test
func generateTestDBName(t *testing.T) string {
	t.Helper()

	// Generate random suffix
	randomBytes := make([]byte, randomSuffixLength)
	_, err := rand.Read(randomBytes)
	require.NoError(t, err)

	randomHex := hex.EncodeToString(randomBytes)

	// Clean test name
	cleanTestName := strings.ReplaceAll(t.Name(), "/", "_")
	cleanTestName = strings.ReplaceAll(cleanTestName, " ", "_")
	cleanTestName = strings.ToLower(cleanTestName)

	// Truncate to avoid PostgreSQL's 63 character limit
	if len(cleanTestName) > maxTestNameLength {
		cleanTestName = cleanTestName[:maxTestNameLength]
	}

	return fmt.Sprintf("test_%s_%s", cleanTestName, randomHex)
}

// createTestDatabase creates a new test database from template
func createTestDatabase(ctx context.Context, t *testing.T, dbName string) *sql.DB {
	t.Helper()

	adminDB, err := sql.Open("postgres", testDB.ConnString)
	require.NoError(t, err)

	_, err = adminDB.ExecContext(ctx,
		fmt.Sprintf("CREATE DATABASE %s WITH TEMPLATE %s", dbName, testDB.TemplateDBName))
	require.NoError(t, err)

	return adminDB
}

// setupTestDatabaseCleanup registers cleanup functions for the test database
func setupTestDatabaseCleanup(t *testing.T, adminDB *sql.DB, dbName string) {
	t.Helper()

	t.Cleanup(func() {
		// Close the admin connection after all test cleanup is done
		defer func() {
			_ = adminDB.Close()
		}()

		// Create a new connection for cleanup to avoid "database is closed" error
		cleanupDB, err := sql.Open("postgres", testDB.ConnString)
		if err != nil {
			t.Logf("Failed to open connection for cleanup: %v", err)
			return
		}

		defer func() {
			_ = cleanupDB.Close()
		}()

		// Disallow new connections
		_, err = cleanupDB.ExecContext(
			context.Background(),
			fmt.Sprintf("REVOKE CONNECT ON DATABASE %s FROM public", dbName))
		if err != nil {
			t.Logf("Failed to revoke connections on test database %s: %v", dbName, err)
		}

		// Terminate all connections to the test database
		_, err = cleanupDB.ExecContext(context.Background(), fmt.Sprintf(`
			SELECT pg_terminate_backend(pid)
			FROM pg_stat_activity
			WHERE datname = '%s' AND pid <> pg_backend_pid()`, dbName))
		if err != nil {
			t.Logf("Failed to terminate backends for test database %s: %v", dbName, err)
		}

		// Drop the test database
		_, err = cleanupDB.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+dbName)
		if err != nil {
			t.Logf("Failed to drop test database %s: %v", dbName, err)
		}
	})
}

// connectToTestDatabase connects to the test database using GORM
func connectToTestDatabase(t *testing.T, dbName string) *gorm.DB {
	t.Helper()

	testDBConnStr := strings.Replace(testDB.ConnString, "dbname=postgres", "dbname="+dbName, 1)
	gormDB, err := gorm.Open(postgresdriver.Open(testDBConnStr), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	// Close connection when test ends
	sqlDB, err := gormDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return gormDB
}

// createTestRedisClient creates a Redis client for testing
func createTestRedisClient(t *testing.T) *internalRedis.Client {
	t.Helper()

	// Reset miniredis for this test
	testDB.Miniredis.FlushAll()

	// Create Redis client for this test
	redisConfig := &config.RedisConfig{
		Host: testDB.Miniredis.Host(),
		Port: testDB.Miniredis.Port(),
		DB:   0,
	}

	redisClient, err := internalRedis.NewClient(
		redisConfig,
		slog.Default(),
	)
	require.NoError(t, err)

	return redisClient
}

// WithTestDB creates a new database from template for each test
// This is the recommended way to run tests with isolated databases
func WithTestDB(t *testing.T, fn func(db *gorm.DB, redisClient *internalRedis.Client)) {
	t.Helper()

	// Check if we should use transaction mode for speed
	if os.Getenv("TEST_USE_TRANSACTION") == "true" {
		WithTestTransaction(t, fn)
		return
	}

	ctx := context.Background()

	// Ensure test infrastructure is initialized
	if testDB == nil {
		t.Fatal("Test database not initialized. Call SetupTestDatabase in TestMain")
	}

	// Generate unique database name for this test
	dbName := generateTestDBName(t)

	// Create test database
	adminDB := createTestDatabase(ctx, t, dbName)

	// Setup cleanup
	setupTestDatabaseCleanup(t, adminDB, dbName)

	// Connect to test database
	gormDB := connectToTestDatabase(t, dbName)

	// Create Redis client
	redisClient := createTestRedisClient(t)

	// Execute the test function
	fn(gormDB, redisClient)
}

// WithTestTransaction provides transaction-based test isolation (faster but less isolated)
func WithTestTransaction(t *testing.T, fn func(db *gorm.DB, redisClient *internalRedis.Client)) {
	t.Helper()

	// Ensure test infrastructure is initialized
	if testDB == nil {
		t.Fatal("Test database not initialized. Call SetupTestDatabase in TestMain")
	}

	// Connect to the main test database
	testDBConnStr := strings.Replace(testDB.ConnString, "dbname=postgres", "dbname="+testDB.TemplateDBName, 1)
	gormDB, err := gorm.Open(postgresdriver.Open(testDBConnStr), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	// Begin transaction
	tx := gormDB.Begin()
	require.NoError(t, tx.Error)

	// Ensure rollback on cleanup
	t.Cleanup(func() {
		tx.Rollback()
	})

	// Create Redis client
	redisClient := createTestRedisClient(t)

	// Execute the test function
	fn(tx, redisClient)
}

// GetTestRedisClient returns a Redis client for testing
func GetTestRedisClient(t *testing.T) *internalRedis.Client {
	t.Helper()

	// Ensure test infrastructure is initialized
	if testDB == nil || testDB.Miniredis == nil {
		t.Fatal("Test database not initialized. Call SetupTestDatabase in TestMain")
	}

	redisConfig := &config.RedisConfig{
		Host: testDB.Miniredis.Host(),
		Port: testDB.Miniredis.Port(),
		DB:   0,
	}

	redisClient, err := internalRedis.NewClient(
		redisConfig,
		slog.Default(),
	)
	require.NoError(t, err)

	return redisClient
}

// CleanupRedis clears all data from the test Redis instance
func CleanupRedis() {
	if testDB != nil && testDB.Miniredis != nil {
		testDB.Miniredis.FlushAll()
	}
}

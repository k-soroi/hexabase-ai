package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(t, err)

	return gormDB, mock
}

func setupTestRepository(t *testing.T) (*postgresRepository, sqlmock.Sqlmock) {
	gormDB, mock := setupTestDB(t)
	// Repository creates its own infrastructure implementation
	repo := NewPostgresRepository(gormDB)
	return repo, mock
}

func TestPostgresRepository_CreateUser(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepository(t)

	t.Run("successful user creation", func(t *testing.T) {
		user := &domain.User{
			ID:          uuid.New().String(),
			Email:       "test@example.com",
			DisplayName: "Test User",
			Provider:    "google",
			ExternalID:  "google-sub-123",
		}

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO "users"`).
			WithArgs(
				user.ID,
				user.ExternalID,
				user.Provider,
				user.Email,
				user.DisplayName,
				user.AvatarURL,
				sqlmock.AnyArg(), // created_at
				sqlmock.AnyArg(), // updated_at
				sqlmock.AnyArg(), // last_login_at
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.CreateUser(ctx, user)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("duplicate user error", func(t *testing.T) {
		user := &domain.User{
			ID:          uuid.New().String(),
			Email:       "duplicate@example.com",
			DisplayName: "Duplicate User",
			Provider:    "google",
			ExternalID:  "google-sub-456",
		}

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO "users"`).
			WithArgs(
				user.ID,
				user.ExternalID,
				user.Provider,
				user.Email,
				user.DisplayName,
				user.AvatarURL,
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
			).
			WillReturnError(gorm.ErrDuplicatedKey)
		mock.ExpectRollback()

		err := repo.CreateUser(ctx, user)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresRepository_GetUser(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepository(t)

	t.Run("user found", func(t *testing.T) {
		userID := uuid.New().String()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "external_id", "provider", "email", "display_name", "avatar_url",
			"created_at", "updated_at", "last_login_at",
		}).AddRow(
			userID, "google-sub-123", "google", "user@example.com", "Test User", "",
			now, now, now,
		)

		mock.ExpectQuery(`SELECT \* FROM "users" WHERE id = \$1 ORDER BY "users"\."id" LIMIT \$2`).
			WithArgs(userID, 1).
			WillReturnRows(rows)

		user, err := repo.GetUser(ctx, userID)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user not found", func(t *testing.T) {
		userID := uuid.New().String()

		mock.ExpectQuery(`SELECT \* FROM "users" WHERE id = \$1 ORDER BY "users"\."id" LIMIT \$2`).
			WithArgs(userID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		user, err := repo.GetUser(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "user not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresRepository_GetUserByEmail(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepository(t)

	t.Run("user found", func(t *testing.T) {
		email := "user@example.com"
		userID := uuid.New().String()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "external_id", "provider", "email", "display_name", "avatar_url",
			"created_at", "updated_at", "last_login_at",
		}).AddRow(
			userID, "google-sub-123", "google", email, "Test User", "",
			now, now, now,
		)

		mock.ExpectQuery(`SELECT \* FROM "users" WHERE email = \$1 ORDER BY "users"\."id" LIMIT \$2`).
			WithArgs(email, 1).
			WillReturnRows(rows)

		user, err := repo.GetUserByEmail(ctx, email)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, userID, user.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user not found", func(t *testing.T) {
		email := "notfound@example.com"

		mock.ExpectQuery(`SELECT \* FROM "users" WHERE email = \$1 ORDER BY "users"\."id" LIMIT \$2`).
			WithArgs(email, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		user, err := repo.GetUserByEmail(ctx, email)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresRepository_GetUserByExternalID(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepository(t)

	t.Run("user found by external ID", func(t *testing.T) {
		externalID := "google-sub-789"
		provider := "google"
		userID := uuid.New().String()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "external_id", "provider", "email", "display_name", "avatar_url",
			"created_at", "updated_at", "last_login_at",
		}).AddRow(
			userID, externalID, provider, "user@example.com", "Test User", "",
			now, now, now,
		)

		mock.ExpectQuery(`SELECT \* FROM "users" WHERE external_id = \$1 AND provider = \$2 ORDER BY "users"\."id" LIMIT \$3`).
			WithArgs(externalID, provider, 1).
			WillReturnRows(rows)

		user, err := repo.GetUserByExternalID(ctx, externalID, provider)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, externalID, user.ExternalID)
		assert.Equal(t, provider, user.Provider)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresRepository_UpdateUser(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepository(t)

	t.Run("successful user update", func(t *testing.T) {
		user := &domain.User{
			ID:          uuid.New().String(),
			Email:       "updated@example.com",
			DisplayName: "Updated Name",
			Provider:    "google",
			ExternalID:  "google-sub-999",
		}

		mock.ExpectBegin()
		// GORM Save will update all fields
		mock.ExpectExec(`UPDATE "users" SET`).
			WithArgs(
				user.ExternalID,
				user.Provider,
				user.Email,
				user.DisplayName,
				user.AvatarURL,
				sqlmock.AnyArg(), // created_at
				sqlmock.AnyArg(), // updated_at
				sqlmock.AnyArg(), // last_login_at
				user.ID,          // WHERE id = ?
			).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.UpdateUser(ctx, user)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresRepository_UpdateLastLogin(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepository(t)

	t.Run("update last login time", func(t *testing.T) {
		userID := uuid.New().String()

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "users" SET "last_login_at"=\$1,"updated_at"=\$2 WHERE id = \$3`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), userID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.UpdateLastLogin(ctx, userID)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresRepository_CreateSession(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepository(t)

	t.Run("create new session with already-hashed refresh token", func(t *testing.T) {
		plainToken := "refresh-token-123"

		// Simulate service layer hashing the token first
		tokenHashRepo := NewTokenHashRepository()
		hashedToken, salt, err := tokenHashRepo.HashToken(plainToken)
		require.NoError(t, err)

		session := &domain.Session{
			ID:                   uuid.New().String(),
			UserID:               uuid.New().String(),
			RefreshToken:         hashedToken, // Already hashed by service layer
			RefreshTokenSelector: "test-selector-123",
			Salt:                 salt, // Generated by service layer
			DeviceID:             "device-123",
			IPAddress:            "192.168.1.100",
			UserAgent:            "Mozilla/5.0...",
			ExpiresAt:            time.Now().Add(24 * time.Hour),
		}

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO "sessions"`).
			WithArgs(
				session.ID,
				session.UserID,
				session.RefreshToken,         // already hashed token from service layer
				session.RefreshTokenSelector, // selector for O(1) lookup
				session.Salt,                 // salt from service layer
				session.DeviceID,
				session.IPAddress,
				session.UserAgent,
				sqlmock.AnyArg(), // expires_at
				sqlmock.AnyArg(), // created_at
				sqlmock.AnyArg(), // last_used_at
				false,            // revoked (default value)
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err = repo.CreateSession(ctx, session)
		assert.NoError(t, err)

		// Verify that the session has the hashed token and salt (from service layer)
		assert.NotEqual(t, plainToken, session.RefreshToken)
		assert.NotEmpty(t, session.Salt)
		assert.Len(t, session.RefreshToken, 64) // SHA-256 hash as hex = 64 chars
		assert.Len(t, session.Salt, 64)         // 32-byte salt as hex = 64 chars

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresRepository_GetAllActiveSessions(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepository(t)

	t.Run("get all active sessions", func(t *testing.T) {
		sessionID1 := uuid.New().String()
		sessionID2 := uuid.New().String()
		userID1 := uuid.New().String()
		userID2 := uuid.New().String()
		now := time.Now()

		// Mock the query for active sessions
		rows := sqlmock.NewRows([]string{
			"id", "user_id", "refresh_token", "salt", "device_id",
			"ip_address", "user_agent", "expires_at", "created_at", "last_used_at", "revoked",
		}).
			AddRow(sessionID1, userID1, "hash1", "salt1", "device-123",
				"192.168.1.1", "test-agent", now.Add(time.Hour), now, now, false).
			AddRow(sessionID2, userID2, "hash2", "salt2", "device-456",
				"192.168.1.2", "test-agent", now.Add(2*time.Hour), now, now, false)

		mock.ExpectQuery(`SELECT \* FROM "sessions" WHERE revoked = \$1 AND expires_at > \$2`).
			WithArgs(false, sqlmock.AnyArg()).
			WillReturnRows(rows)

		// When
		sessions, err := repo.GetAllActiveSessions(ctx)

		// Then
		require.NoError(t, err)
		assert.Len(t, sessions, 2)
		assert.Equal(t, sessionID1, sessions[0].ID)
		assert.Equal(t, sessionID2, sessions[1].ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresRepository_GetSession(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepository(t)

	t.Run("get session by ID", func(t *testing.T) {
		sessionID := uuid.New().String()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "refresh_token", "device_id", "ip_address",
			"user_agent", "expires_at", "created_at", "last_used_at", "revoked",
		}).AddRow(
			sessionID, "user-123", "refresh-token-123", "device-123", "192.168.1.100",
			"Mozilla/5.0...", now.Add(24*time.Hour), now, now, false,
		)

		mock.ExpectQuery(`SELECT \* FROM "sessions" WHERE id = \$1 ORDER BY "sessions"\."id" LIMIT \$2`).
			WithArgs(sessionID, 1).
			WillReturnRows(rows)

		session, err := repo.GetSession(ctx, sessionID)
		assert.NoError(t, err)
		assert.NotNil(t, session)
		assert.Equal(t, sessionID, session.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresRepository_StoreAuthState(t *testing.T) {
	ctx := context.Background()
	gormDB, mock := setupTestDB(t)
	repo := NewPostgresRepository(gormDB)

	t.Run("successfully stores auth state with code challenge", func(t *testing.T) {
		authState := &domain.AuthState{
			State:         "test-state-123",
			Provider:      "google",
			RedirectURL:   "https://example.com/callback",
			CodeChallenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", // RFC 7636 compliant challenge
			ClientIP:      "192.168.1.1",
			UserAgent:     "Mozilla/5.0",
			ExpiresAt:     time.Now().Add(10 * time.Minute),
			CreatedAt:     time.Now(),
		}

		// Expect the INSERT query with the new code_challenge and is_sign_up columns
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO "auth_states" `+
			`\("state","provider","redirect_url","code_challenge","client_ip","user_agent",`+
			`"is_sign_up","expires_at","created_at"\)`).
			WithArgs(
				authState.State,
				authState.Provider,
				authState.RedirectURL,
				authState.CodeChallenge,
				authState.ClientIP,
				authState.UserAgent,
				authState.IsSignUp,
				sqlmock.AnyArg(), // expires_at
				sqlmock.AnyArg(), // created_at
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.StoreAuthState(ctx, authState)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successfully stores auth state without PKCE", func(t *testing.T) {
		authState := &domain.AuthState{
			State:         "test-state-456",
			Provider:      "github",
			RedirectURL:   "https://example.com/callback",
			CodeChallenge: "", // No PKCE
			ClientIP:      "192.168.1.2",
			UserAgent:     "Chrome/91.0",
			ExpiresAt:     time.Now().Add(10 * time.Minute),
			CreatedAt:     time.Now(),
		}

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO "auth_states"`).
			WithArgs(
				authState.State,
				authState.Provider,
				authState.RedirectURL,
				authState.CodeChallenge,
				authState.ClientIP,
				authState.UserAgent,
				authState.IsSignUp,
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.StoreAuthState(ctx, authState)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresRepository_GetAuthState(t *testing.T) {
	ctx := context.Background()
	gormDB, mock := setupTestDB(t)
	repo := NewPostgresRepository(gormDB)

	t.Run("successfully retrieves auth state with code challenge", func(t *testing.T) {
		stateValue := "test-state-123"
		expectedChallenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
		expiresAt := time.Now().Add(5 * time.Minute)
		createdAt := time.Now().Add(-5 * time.Minute)

		// Expect the SELECT query to include code_challenge column
		rows := sqlmock.NewRows([]string{
			"state", "provider", "redirect_url", "code_challenge",
			"client_ip", "user_agent", "expires_at", "created_at",
		}).AddRow(
			stateValue,
			"google",
			"https://example.com/callback",
			expectedChallenge,
			"192.168.1.1",
			"Mozilla/5.0",
			expiresAt,
			createdAt,
		)

		mock.ExpectQuery(`SELECT \* FROM "auth_states" WHERE state = \$1 AND expires_at > \$2 ORDER BY "auth_states"\."state" LIMIT \$3`).
			WithArgs(stateValue, sqlmock.AnyArg(), 1).
			WillReturnRows(rows)

		authState, err := repo.GetAuthState(ctx, stateValue)
		assert.NoError(t, err)
		assert.NotNil(t, authState)
		assert.Equal(t, stateValue, authState.State)
		assert.Equal(t, expectedChallenge, authState.CodeChallenge)
		assert.Equal(t, "google", authState.Provider)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for expired auth state", func(t *testing.T) {
		stateValue := "expired-state"

		mock.ExpectQuery(`SELECT \* FROM "auth_states" WHERE state = \$1 AND expires_at > \$2 ORDER BY "auth_states"\."state" LIMIT \$3`).
			WithArgs(stateValue, sqlmock.AnyArg(), 1).
			WillReturnError(gorm.ErrRecordNotFound)

		authState, err := repo.GetAuthState(ctx, stateValue)
		assert.Error(t, err)
		assert.Nil(t, authState)
		assert.Contains(t, err.Error(), "auth state not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("handles auth state without PKCE", func(t *testing.T) {
		stateValue := "no-pkce-state"
		expiresAt := time.Now().Add(5 * time.Minute)
		createdAt := time.Now().Add(-5 * time.Minute)

		rows := sqlmock.NewRows([]string{
			"state", "provider", "redirect_url", "code_challenge",
			"client_ip", "user_agent", "expires_at", "created_at",
		}).AddRow(
			stateValue,
			"github",
			"https://example.com/callback",
			"", // Empty code challenge
			"192.168.1.2",
			"Chrome/91.0",
			expiresAt,
			createdAt,
		)

		mock.ExpectQuery(`SELECT \* FROM "auth_states" WHERE state = \$1 AND expires_at > \$2 ORDER BY "auth_states"\."state" LIMIT \$3`).
			WithArgs(stateValue, sqlmock.AnyArg(), 1).
			WillReturnRows(rows)

		authState, err := repo.GetAuthState(ctx, stateValue)
		assert.NoError(t, err)
		assert.NotNil(t, authState)
		assert.Equal(t, stateValue, authState.State)
		assert.Equal(t, "", authState.CodeChallenge)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresRepository_AuthStateColumnCompatibility(t *testing.T) {
	ctx := context.Background()
	gormDB, mock := setupTestDB(t)
	repo := NewPostgresRepository(gormDB)

	t.Run("verifies code_challenge column is used in queries", func(t *testing.T) {
		// This test ensures that the repository is using the new column name
		authState := &domain.AuthState{
			State:         "migration-test",
			Provider:      "google",
			CodeChallenge: "test-challenge-value",
			ExpiresAt:     time.Now().Add(10 * time.Minute),
			CreatedAt:     time.Now(),
		}

		// The INSERT should specifically include code_challenge, not code_verifier
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO "auth_states" \(.*"code_challenge".*\)`).
			WithArgs(
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				authState.CodeChallenge,
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				sqlmock.AnyArg(), // is_sign_up
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.StoreAuthState(ctx, authState)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresRepository_RefreshTokenBlacklist(t *testing.T) {
	ctx := context.Background()
	gormDB, mock := setupTestDB(t)
	repo := NewPostgresRepository(gormDB)

	t.Run("blacklist and check token", func(t *testing.T) {
		token := "test-refresh-token"
		expiresAt := time.Now().Add(1 * time.Hour)

		// 1. BlacklistRefreshToken
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO "refresh_token_blacklists"`).
			WithArgs(sqlmock.AnyArg(), token, expiresAt, sqlmock.AnyArg()). // id, token, expires_at, created_at
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.BlacklistRefreshToken(ctx, token, expiresAt)
		assert.NoError(t, err)

		// 2. IsRefreshTokenBlacklisted - should be true
		rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "refresh_token_blacklists" WHERE token = \$1 AND expires_at > \$2`).
			WithArgs(token, sqlmock.AnyArg()). // token, time.Now()
			WillReturnRows(rows)

		isBlacklisted, err := repo.IsRefreshTokenBlacklisted(ctx, token)
		assert.NoError(t, err)
		assert.True(t, isBlacklisted)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("token not in blacklist", func(t *testing.T) {
		token := "non-blacklisted-token"

		rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "refresh_token_blacklists" WHERE token = \$1 AND expires_at > \$2`).
			WithArgs(token, sqlmock.AnyArg()).
			WillReturnRows(rows)

		isBlacklisted, err := repo.IsRefreshTokenBlacklisted(ctx, token)
		assert.NoError(t, err)
		assert.False(t, isBlacklisted)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("expired token in blacklist", func(t *testing.T) {
		token := "expired-token"

		// IsRefreshTokenBlacklisted checks for "expires_at > ?", so a count of 0 is expected
		rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "refresh_token_blacklists" WHERE token = \$1 AND expires_at > \$2`).
			WithArgs(token, sqlmock.AnyArg()).
			WillReturnRows(rows)

		isBlacklisted, err := repo.IsRefreshTokenBlacklisted(ctx, token)
		assert.NoError(t, err)
		assert.False(t, isBlacklisted)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error on BlacklistRefreshToken", func(t *testing.T) {
		token := "some-token"
		expiresAt := time.Now().Add(1 * time.Hour)

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO "refresh_token_blacklists"`).
			WithArgs(sqlmock.AnyArg(), token, expiresAt, sqlmock.AnyArg()).
			WillReturnError(fmt.Errorf("db error"))
		mock.ExpectRollback()

		err := repo.BlacklistRefreshToken(ctx, token, expiresAt)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to blacklist refresh token")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error on IsRefreshTokenBlacklisted", func(t *testing.T) {
		token := "some-token"

		mock.ExpectQuery(`SELECT count\(\*\) FROM "refresh_token_blacklists"`).
			WithArgs(token, sqlmock.AnyArg()).
			WillReturnError(fmt.Errorf("db error"))

		_, err := repo.IsRefreshTokenBlacklisted(ctx, token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check blacklist")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresRepository_GetSessionByRefreshTokenSelector(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepository(t)

	t.Run("find session by valid selector", func(t *testing.T) {
		selector := "selector-abc123"
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "refresh_token", "refresh_token_selector", "salt",
			"device_id", "ip_address", "user_agent", "expires_at", "created_at", "last_used_at", "revoked",
		}).AddRow(
			"session-123", "user-456", "hashed-verifier-789", selector, "salt-def456",
			"device-xyz", "192.168.1.1", "Mozilla/5.0", now.Add(24*time.Hour), now, now, false,
		)

		mock.ExpectQuery(`SELECT \* FROM "sessions" WHERE refresh_token_selector = \$1 ORDER BY "sessions"\."id" LIMIT \$2`).
			WithArgs(selector, 1).
			WillReturnRows(rows)

		session, err := repo.GetSessionByRefreshTokenSelector(ctx, selector)
		assert.NoError(t, err)
		assert.NotNil(t, session)
		assert.Equal(t, selector, session.RefreshTokenSelector)
		assert.Equal(t, "user-456", session.UserID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("return error for nonexistent selector", func(t *testing.T) {
		selector := "nonexistent-selector"

		mock.ExpectQuery(`SELECT \* FROM "sessions" WHERE refresh_token_selector = \$1 ORDER BY "sessions"\."id" LIMIT \$2`).
			WithArgs(selector, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		_, err := repo.GetSessionByRefreshTokenSelector(ctx, selector)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

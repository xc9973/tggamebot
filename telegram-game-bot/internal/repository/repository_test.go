// Package repository provides data access layer implementations.
// Tests use testcontainers-go to spin up a PostgreSQL container.
// Requirements: 8.1, 8.2 - PostgreSQL database testing
package repository

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"telegram-game-bot/internal/model"
)

// checkDockerAvailable checks if Docker is available and running
func checkDockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	err := cmd.Run()
	return err == nil
}

// setupTestDB creates a PostgreSQL container and returns a connection pool
// Skips the test if Docker is not available
func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	if !checkDockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}

	ctx := context.Background()

	// Create PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create connection pool
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	// Run migrations
	err = runMigrations(ctx, pool)
	require.NoError(t, err)

	// Return cleanup function
	cleanup := func() {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
	}

	return pool, cleanup
}

// runMigrations applies the database schema
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Create users table
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			telegram_id BIGINT PRIMARY KEY,
			username VARCHAR(255) NOT NULL,
			balance BIGINT NOT NULL DEFAULT 1000,
			last_daily_claim BIGINT DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}

	// Create transactions table
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS transactions (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(telegram_id) ON DELETE CASCADE,
			amount BIGINT NOT NULL,
			type VARCHAR(50) NOT NULL,
			description TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}


// ============================================================================
// UserRepository Tests
// ============================================================================

func TestUserRepository_Create(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(pool)
	ctx := context.Background()

	// Test creating a new user
	user, err := repo.Create(ctx, 12345, "testuser")
	require.NoError(t, err)
	assert.Equal(t, int64(12345), user.TelegramID)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, int64(1000), user.Balance) // Initial balance should be 1000
	assert.Equal(t, int64(0), user.LastDailyClaim)
	assert.False(t, user.CreatedAt.IsZero())
}

func TestUserRepository_GetByID(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(pool)
	ctx := context.Background()

	// Create a user first
	_, err := repo.Create(ctx, 12345, "testuser")
	require.NoError(t, err)

	// Test getting the user
	user, err := repo.GetByID(ctx, 12345)
	require.NoError(t, err)
	assert.Equal(t, int64(12345), user.TelegramID)
	assert.Equal(t, "testuser", user.Username)

	// Test getting non-existent user
	_, err = repo.GetByID(ctx, 99999)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserRepository_GetOrCreate(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(pool)
	ctx := context.Background()

	// Test creating new user
	user, created, err := repo.GetOrCreate(ctx, 12345, "testuser")
	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, int64(12345), user.TelegramID)

	// Test getting existing user
	user, created, err = repo.GetOrCreate(ctx, 12345, "testuser")
	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, int64(12345), user.TelegramID)
}

func TestUserRepository_UpdateBalance(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(pool)
	ctx := context.Background()

	// Create a user
	_, err := repo.Create(ctx, 12345, "testuser")
	require.NoError(t, err)

	// Test adding balance
	user, err := repo.UpdateBalance(ctx, 12345, 500)
	require.NoError(t, err)
	assert.Equal(t, int64(1500), user.Balance)

	// Test subtracting balance
	user, err = repo.UpdateBalance(ctx, 12345, -300)
	require.NoError(t, err)
	assert.Equal(t, int64(1200), user.Balance)

	// Test updating non-existent user
	_, err = repo.UpdateBalance(ctx, 99999, 100)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserRepository_SetBalance(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(pool)
	ctx := context.Background()

	// Create a user
	_, err := repo.Create(ctx, 12345, "testuser")
	require.NoError(t, err)

	// Test setting balance
	user, err := repo.SetBalance(ctx, 12345, 5000)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), user.Balance)

	// Test setting non-existent user
	_, err = repo.SetBalance(ctx, 99999, 100)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserRepository_GetTopUsers(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(pool)
	ctx := context.Background()

	// Create users with different balances
	_, _ = repo.Create(ctx, 1, "user1")
	_, _ = repo.Create(ctx, 2, "user2")
	_, _ = repo.Create(ctx, 3, "user3")

	// Set different balances
	_, _ = repo.SetBalance(ctx, 1, 3000)
	_, _ = repo.SetBalance(ctx, 2, 1000)
	_, _ = repo.SetBalance(ctx, 3, 5000)

	// Get top users
	users, err := repo.GetTopUsers(ctx, 10)
	require.NoError(t, err)
	require.Len(t, users, 3)

	// Verify ordering (descending by balance)
	assert.Equal(t, int64(3), users[0].TelegramID) // 5000
	assert.Equal(t, int64(1), users[1].TelegramID) // 3000
	assert.Equal(t, int64(2), users[2].TelegramID) // 1000
}


func TestUserRepository_DailyClaim(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(pool)
	ctx := context.Background()

	// Create a user
	_, err := repo.Create(ctx, 12345, "testuser")
	require.NoError(t, err)

	// Test can claim when never claimed
	canClaim, remaining, err := repo.CanClaimDaily(ctx, 12345, 24)
	require.NoError(t, err)
	assert.True(t, canClaim)
	assert.Equal(t, time.Duration(0), remaining)

	// Update daily claim
	now := time.Now().Unix()
	_, err = repo.UpdateDailyClaim(ctx, 12345, now)
	require.NoError(t, err)

	// Test cannot claim immediately after
	canClaim, remaining, err = repo.CanClaimDaily(ctx, 12345, 24)
	require.NoError(t, err)
	assert.False(t, canClaim)
	assert.True(t, remaining > 0)

	// Test can claim after cooldown (simulate by setting old timestamp)
	oldTime := time.Now().Add(-25 * time.Hour).Unix()
	_, err = repo.UpdateDailyClaim(ctx, 12345, oldTime)
	require.NoError(t, err)

	canClaim, _, err = repo.CanClaimDaily(ctx, 12345, 24)
	require.NoError(t, err)
	assert.True(t, canClaim)
}

func TestUserRepository_UpdateUsername(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(pool)
	ctx := context.Background()

	// Create a user
	_, err := repo.Create(ctx, 12345, "oldname")
	require.NoError(t, err)

	// Update username
	err = repo.UpdateUsername(ctx, 12345, "newname")
	require.NoError(t, err)

	// Verify update
	user, err := repo.GetByID(ctx, 12345)
	require.NoError(t, err)
	assert.Equal(t, "newname", user.Username)

	// Test updating non-existent user
	err = repo.UpdateUsername(ctx, 99999, "name")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserRepository_Exists(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(pool)
	ctx := context.Background()

	// Test non-existent user
	exists, err := repo.Exists(ctx, 12345)
	require.NoError(t, err)
	assert.False(t, exists)

	// Create user
	_, err = repo.Create(ctx, 12345, "testuser")
	require.NoError(t, err)

	// Test existing user
	exists, err = repo.Exists(ctx, 12345)
	require.NoError(t, err)
	assert.True(t, exists)
}

// ============================================================================
// TransactionRepository Tests
// ============================================================================

func TestTransactionRepository_Create(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := NewUserRepository(pool)
	txRepo := NewTransactionRepository(pool)
	ctx := context.Background()

	// Create a user first (foreign key constraint)
	_, err := userRepo.Create(ctx, 12345, "testuser")
	require.NoError(t, err)

	// Create a transaction
	desc := "test transaction"
	tx, err := txRepo.Create(ctx, 12345, 500, model.TxTypeDice, &desc)
	require.NoError(t, err)
	assert.Equal(t, int64(12345), tx.UserID)
	assert.Equal(t, int64(500), tx.Amount)
	assert.Equal(t, model.TxTypeDice, tx.Type)
	assert.NotNil(t, tx.Description)
	assert.Equal(t, "test transaction", *tx.Description)
}

func TestTransactionRepository_GetByUserID(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := NewUserRepository(pool)
	txRepo := NewTransactionRepository(pool)
	ctx := context.Background()

	// Create a user
	_, err := userRepo.Create(ctx, 12345, "testuser")
	require.NoError(t, err)

	// Create multiple transactions
	_, _ = txRepo.Create(ctx, 12345, 100, model.TxTypeDice, nil)
	_, _ = txRepo.Create(ctx, 12345, -50, model.TxTypeSlot, nil)
	_, _ = txRepo.Create(ctx, 12345, 200, model.TxTypeDice, nil)

	// Get transactions
	txs, err := txRepo.GetByUserID(ctx, 12345, 10)
	require.NoError(t, err)
	assert.Len(t, txs, 3)

	// Verify ordering (newest first)
	assert.Equal(t, int64(200), txs[0].Amount)
}

func TestTransactionRepository_GetByUserIDAndType(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := NewUserRepository(pool)
	txRepo := NewTransactionRepository(pool)
	ctx := context.Background()

	// Create a user
	_, err := userRepo.Create(ctx, 12345, "testuser")
	require.NoError(t, err)

	// Create transactions of different types
	_, _ = txRepo.Create(ctx, 12345, 100, model.TxTypeDice, nil)
	_, _ = txRepo.Create(ctx, 12345, -50, model.TxTypeSlot, nil)
	_, _ = txRepo.Create(ctx, 12345, 200, model.TxTypeDice, nil)

	// Get only dice transactions
	txs, err := txRepo.GetByUserIDAndType(ctx, 12345, model.TxTypeDice, 10)
	require.NoError(t, err)
	assert.Len(t, txs, 2)
	for _, tx := range txs {
		assert.Equal(t, model.TxTypeDice, tx.Type)
	}
}


func TestTransactionRepository_GetDailyStats(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := NewUserRepository(pool)
	txRepo := NewTransactionRepository(pool)
	ctx := context.Background()

	// Create users
	_, _ = userRepo.Create(ctx, 1, "user1")
	_, _ = userRepo.Create(ctx, 2, "user2")

	// Create transactions for today
	now := time.Now()
	_, _ = txRepo.CreateWithTime(ctx, 1, 500, model.TxTypeDice, nil, now)
	_, _ = txRepo.CreateWithTime(ctx, 1, -200, model.TxTypeSlot, nil, now)
	_, _ = txRepo.CreateWithTime(ctx, 2, -300, model.TxTypeDice, nil, now)

	// Get daily stats
	stats, err := txRepo.GetDailyStats(ctx, now)
	require.NoError(t, err)
	assert.Len(t, stats, 2)

	// Verify ordering (by net profit descending)
	assert.Equal(t, int64(1), stats[0].UserID)  // 500 - 200 = 300
	assert.Equal(t, int64(300), stats[0].NetProfit)
	assert.Equal(t, int64(2), stats[1].UserID)  // -300
	assert.Equal(t, int64(-300), stats[1].NetProfit)
}

func TestTransactionRepository_GetDailyWinners(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := NewUserRepository(pool)
	txRepo := NewTransactionRepository(pool)
	ctx := context.Background()

	// Create users
	_, _ = userRepo.Create(ctx, 1, "winner1")
	_, _ = userRepo.Create(ctx, 2, "winner2")
	_, _ = userRepo.Create(ctx, 3, "loser1")

	// Create transactions
	now := time.Now()
	_, _ = txRepo.CreateWithTime(ctx, 1, 1000, model.TxTypeDice, nil, now)
	_, _ = txRepo.CreateWithTime(ctx, 2, 500, model.TxTypeSlot, nil, now)
	_, _ = txRepo.CreateWithTime(ctx, 3, -300, model.TxTypeDice, nil, now)

	// Get winners
	winners, err := txRepo.GetDailyWinners(ctx, now, 10)
	require.NoError(t, err)
	assert.Len(t, winners, 2) // Only positive profits

	// Verify ordering
	assert.Equal(t, int64(1), winners[0].UserID)
	assert.Equal(t, int64(1000), winners[0].NetProfit)
	assert.Equal(t, int64(2), winners[1].UserID)
	assert.Equal(t, int64(500), winners[1].NetProfit)
}

func TestTransactionRepository_GetDailyLosers(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := NewUserRepository(pool)
	txRepo := NewTransactionRepository(pool)
	ctx := context.Background()

	// Create users
	_, _ = userRepo.Create(ctx, 1, "winner1")
	_, _ = userRepo.Create(ctx, 2, "loser1")
	_, _ = userRepo.Create(ctx, 3, "loser2")

	// Create transactions
	now := time.Now()
	_, _ = txRepo.CreateWithTime(ctx, 1, 1000, model.TxTypeDice, nil, now)
	_, _ = txRepo.CreateWithTime(ctx, 2, -500, model.TxTypeSlot, nil, now)
	_, _ = txRepo.CreateWithTime(ctx, 3, -800, model.TxTypeDice, nil, now)

	// Get losers
	losers, err := txRepo.GetDailyLosers(ctx, now, 10)
	require.NoError(t, err)
	assert.Len(t, losers, 2) // Only negative profits

	// Verify ordering (most loss first)
	assert.Equal(t, int64(3), losers[0].UserID)
	assert.Equal(t, int64(-800), losers[0].NetProfit)
	assert.Equal(t, int64(2), losers[1].UserID)
	assert.Equal(t, int64(-500), losers[1].NetProfit)
}

func TestTransactionRepository_GetUserDailyProfit(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := NewUserRepository(pool)
	txRepo := NewTransactionRepository(pool)
	ctx := context.Background()

	// Create a user
	_, err := userRepo.Create(ctx, 12345, "testuser")
	require.NoError(t, err)

	// Create transactions
	now := time.Now()
	_, _ = txRepo.CreateWithTime(ctx, 12345, 500, model.TxTypeDice, nil, now)
	_, _ = txRepo.CreateWithTime(ctx, 12345, -200, model.TxTypeSlot, nil, now)
	_, _ = txRepo.CreateWithTime(ctx, 12345, 100, model.TxTypeTransfer, nil, now) // Should not count

	// Get user daily profit
	profit, err := txRepo.GetUserDailyProfit(ctx, 12345, now)
	require.NoError(t, err)
	assert.Equal(t, int64(300), profit) // 500 - 200 = 300 (transfer excluded)
}

func TestTransactionRepository_ExcludesNonGameTransactions(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := NewUserRepository(pool)
	txRepo := NewTransactionRepository(pool)
	ctx := context.Background()

	// Create a user
	_, err := userRepo.Create(ctx, 12345, "testuser")
	require.NoError(t, err)

	// Create various transaction types
	now := time.Now()
	_, _ = txRepo.CreateWithTime(ctx, 12345, 500, model.TxTypeDice, nil, now)
	_, _ = txRepo.CreateWithTime(ctx, 12345, 500, model.TxTypeDaily, nil, now)     // Should not count
	_, _ = txRepo.CreateWithTime(ctx, 12345, 500, model.TxTypeTransfer, nil, now)  // Should not count
	_, _ = txRepo.CreateWithTime(ctx, 12345, 500, model.TxTypeAdminAdd, nil, now)  // Should not count

	// Get daily stats - should only include game transactions
	stats, err := txRepo.GetDailyStats(ctx, now)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	assert.Equal(t, int64(500), stats[0].NetProfit) // Only dice transaction
}

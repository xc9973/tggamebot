// Package repository provides data access layer implementations.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"telegram-game-bot/internal/model"
)

// Common errors for repository operations.
var (
	ErrUserNotFound = errors.New("user not found")
)

// UserRepository handles user data persistence.
// Requirements: 1.1, 1.3, 1.5 - User account management
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create creates a new user with the given Telegram ID and username.
// The user is created with the default initial balance (1000 coins).
// Requirements: 1.1 - Create account with 1000 initial coins
func (r *UserRepository) Create(ctx context.Context, telegramID int64, username string) (*model.User, error) {
	const query = `
		INSERT INTO users (telegram_id, username, balance, last_daily_claim, created_at, updated_at)
		VALUES ($1, $2, 1000, 0, NOW(), NOW())
		RETURNING telegram_id, username, balance, last_daily_claim, created_at, updated_at
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, telegramID, username).Scan(
		&user.TelegramID,
		&user.Username,
		&user.Balance,
		&user.LastDailyClaim,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}


// GetByID retrieves a user by their Telegram ID.
// Returns ErrUserNotFound if the user does not exist.
func (r *UserRepository) GetByID(ctx context.Context, telegramID int64) (*model.User, error) {
	const query = `
		SELECT telegram_id, username, balance, last_daily_claim, created_at, updated_at
		FROM users
		WHERE telegram_id = $1
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, telegramID).Scan(
		&user.TelegramID,
		&user.Username,
		&user.Balance,
		&user.LastDailyClaim,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetOrCreate retrieves a user by Telegram ID, creating one if it doesn't exist.
// This is useful for ensuring a user exists before performing operations.
// Requirements: 1.1 - Create account with 1000 initial coins on first interaction
func (r *UserRepository) GetOrCreate(ctx context.Context, telegramID int64, username string) (*model.User, bool, error) {
	// Try to get existing user first
	user, err := r.GetByID(ctx, telegramID)
	if err == nil {
		return user, false, nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, false, err
	}

	// User doesn't exist, create new one
	user, err = r.Create(ctx, telegramID, username)
	if err != nil {
		// Handle race condition: another request might have created the user
		user, err = r.GetByID(ctx, telegramID)
		if err != nil {
			return nil, false, err
		}
		return user, false, nil
	}

	return user, true, nil
}

// UpdateBalance updates a user's balance by adding the specified amount.
// The amount can be negative to subtract from the balance.
// Returns the updated user.
func (r *UserRepository) UpdateBalance(ctx context.Context, telegramID int64, amount int64) (*model.User, error) {
	const query = `
		UPDATE users
		SET balance = balance + $2, updated_at = NOW()
		WHERE telegram_id = $1
		RETURNING telegram_id, username, balance, last_daily_claim, created_at, updated_at
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, telegramID, amount).Scan(
		&user.TelegramID,
		&user.Username,
		&user.Balance,
		&user.LastDailyClaim,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	return &user, nil
}

// SetBalance sets a user's balance to an exact value.
// Used primarily for admin operations.
func (r *UserRepository) SetBalance(ctx context.Context, telegramID int64, balance int64) (*model.User, error) {
	const query = `
		UPDATE users
		SET balance = $2, updated_at = NOW()
		WHERE telegram_id = $1
		RETURNING telegram_id, username, balance, last_daily_claim, created_at, updated_at
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, telegramID, balance).Scan(
		&user.TelegramID,
		&user.Username,
		&user.Balance,
		&user.LastDailyClaim,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to set balance: %w", err)
	}

	return &user, nil
}


// GetTopUsers retrieves the top N users by balance.
// Requirements: 1.5 - Display top 10 users by balance
func (r *UserRepository) GetTopUsers(ctx context.Context, limit int) ([]*model.User, error) {
	const query = `
		SELECT telegram_id, username, balance, last_daily_claim, created_at, updated_at
		FROM users
		ORDER BY balance DESC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top users: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var user model.User
		err := rows.Scan(
			&user.TelegramID,
			&user.Username,
			&user.Balance,
			&user.LastDailyClaim,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// UpdateDailyClaim updates the user's last daily claim timestamp.
// Requirements: 1.3 - Grant 500 coins if 24 hours passed since last claim
func (r *UserRepository) UpdateDailyClaim(ctx context.Context, telegramID int64, claimTime int64) (*model.User, error) {
	const query = `
		UPDATE users
		SET last_daily_claim = $2, updated_at = NOW()
		WHERE telegram_id = $1
		RETURNING telegram_id, username, balance, last_daily_claim, created_at, updated_at
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, telegramID, claimTime).Scan(
		&user.TelegramID,
		&user.Username,
		&user.Balance,
		&user.LastDailyClaim,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to update daily claim: %w", err)
	}

	return &user, nil
}

// CanClaimDaily checks if a user can claim their daily reward.
// Returns true if 24 hours have passed since the last claim, or if never claimed.
// Also returns the remaining time until next claim if not eligible.
// Requirements: 1.3, 1.4 - Daily claim eligibility check
func (r *UserRepository) CanClaimDaily(ctx context.Context, telegramID int64, cooldownHours int) (bool, time.Duration, error) {
	user, err := r.GetByID(ctx, telegramID)
	if err != nil {
		return false, 0, err
	}

	// If never claimed (last_daily_claim is 0), can claim
	if user.LastDailyClaim == 0 {
		return true, 0, nil
	}

	// Calculate time since last claim
	lastClaim := time.Unix(user.LastDailyClaim, 0)
	cooldown := time.Duration(cooldownHours) * time.Hour
	nextClaimTime := lastClaim.Add(cooldown)
	now := time.Now()

	if now.After(nextClaimTime) || now.Equal(nextClaimTime) {
		return true, 0, nil
	}

	remaining := nextClaimTime.Sub(now)
	return false, remaining, nil
}

// UpdateUsername updates a user's username.
// This is useful when a user changes their Telegram username.
func (r *UserRepository) UpdateUsername(ctx context.Context, telegramID int64, username string) error {
	const query = `
		UPDATE users
		SET username = $2, updated_at = NOW()
		WHERE telegram_id = $1
	`

	result, err := r.pool.Exec(ctx, query, telegramID, username)
	if err != nil {
		return fmt.Errorf("failed to update username: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// Exists checks if a user with the given Telegram ID exists.
func (r *UserRepository) Exists(ctx context.Context, telegramID int64) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM users WHERE telegram_id = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, telegramID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return exists, nil
}

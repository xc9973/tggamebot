package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"telegram-game-bot/internal/model"
)

// TransactionRepository handles transaction data persistence.
// Requirements: 2.5, 11.2 - Transaction history and daily stats
type TransactionRepository struct {
	pool *pgxpool.Pool
}

// NewTransactionRepository creates a new TransactionRepository instance.
func NewTransactionRepository(pool *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{pool: pool}
}

// Create creates a new transaction record.
// Requirements: 2.5 - Record all transfers in transaction history
func (r *TransactionRepository) Create(ctx context.Context, userID int64, amount int64, txType string, description *string) (*model.Transaction, error) {
	const query = `
		INSERT INTO transactions (user_id, amount, type, description, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, user_id, amount, type, description, created_at
	`

	var tx model.Transaction
	err := r.pool.QueryRow(ctx, query, userID, amount, txType, description).Scan(
		&tx.ID,
		&tx.UserID,
		&tx.Amount,
		&tx.Type,
		&tx.Description,
		&tx.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return &tx, nil
}

// CreateWithTime creates a new transaction record with a specific timestamp.
// Useful for testing and data migration.
func (r *TransactionRepository) CreateWithTime(ctx context.Context, userID int64, amount int64, txType string, description *string, createdAt time.Time) (*model.Transaction, error) {
	const query = `
		INSERT INTO transactions (user_id, amount, type, description, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, amount, type, description, created_at
	`

	var tx model.Transaction
	err := r.pool.QueryRow(ctx, query, userID, amount, txType, description, createdAt).Scan(
		&tx.ID,
		&tx.UserID,
		&tx.Amount,
		&tx.Type,
		&tx.Description,
		&tx.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return &tx, nil
}


// GetByUserID retrieves all transactions for a user, ordered by creation time (newest first).
func (r *TransactionRepository) GetByUserID(ctx context.Context, userID int64, limit int) ([]*model.Transaction, error) {
	const query = `
		SELECT id, user_id, amount, type, description, created_at
		FROM transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*model.Transaction
	for rows.Next() {
		var tx model.Transaction
		err := rows.Scan(
			&tx.ID,
			&tx.UserID,
			&tx.Amount,
			&tx.Type,
			&tx.Description,
			&tx.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, &tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

// GetByUserIDAndType retrieves transactions for a user filtered by type.
func (r *TransactionRepository) GetByUserIDAndType(ctx context.Context, userID int64, txType string, limit int) ([]*model.Transaction, error) {
	const query = `
		SELECT id, user_id, amount, type, description, created_at
		FROM transactions
		WHERE user_id = $1 AND type = $2
		ORDER BY created_at DESC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, userID, txType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*model.Transaction
	for rows.Next() {
		var tx model.Transaction
		err := rows.Scan(
			&tx.ID,
			&tx.UserID,
			&tx.Amount,
			&tx.Type,
			&tx.Description,
			&tx.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, &tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}


// GetDailyStats retrieves daily game statistics for ranking.
// Returns users with their net profit/loss for the specified date.
// Requirements: 11.2 - Track daily net profit/loss for each user from game transactions
func (r *TransactionRepository) GetDailyStats(ctx context.Context, date time.Time) ([]*model.DailyRank, error) {
	// Get the start and end of the day
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	const query = `
		SELECT t.user_id, u.username, COALESCE(SUM(t.amount), 0) as net_profit
		FROM transactions t
		JOIN users u ON t.user_id = u.telegram_id
		WHERE t.type IN ('dice', 'slot', 'sicbo_win', 'sicbo_bet')
		  AND t.created_at >= $1
		  AND t.created_at < $2
		GROUP BY t.user_id, u.username
		ORDER BY net_profit DESC
	`

	rows, err := r.pool.Query(ctx, query, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}
	defer rows.Close()

	var stats []*model.DailyRank
	for rows.Next() {
		var rank model.DailyRank
		err := rows.Scan(
			&rank.UserID,
			&rank.Username,
			&rank.NetProfit,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily rank: %w", err)
		}
		stats = append(stats, &rank)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating daily stats: %w", err)
	}

	return stats, nil
}

// GetDailyWinners retrieves the top winners for a specific date.
// Winners are users with positive net profit, sorted by profit descending.
// Requirements: 11.3 - Show top 10 winners (most profit)
func (r *TransactionRepository) GetDailyWinners(ctx context.Context, date time.Time, limit int) ([]*model.DailyRank, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	const query = `
		SELECT t.user_id, u.username, COALESCE(SUM(t.amount), 0) as net_profit
		FROM transactions t
		JOIN users u ON t.user_id = u.telegram_id
		WHERE t.type IN ('dice', 'slot', 'sicbo_win', 'sicbo_bet')
		  AND t.created_at >= $1
		  AND t.created_at < $2
		GROUP BY t.user_id, u.username
		HAVING SUM(t.amount) > 0
		ORDER BY net_profit DESC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, startOfDay, endOfDay, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily winners: %w", err)
	}
	defer rows.Close()

	var winners []*model.DailyRank
	for rows.Next() {
		var rank model.DailyRank
		err := rows.Scan(
			&rank.UserID,
			&rank.Username,
			&rank.NetProfit,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan winner: %w", err)
		}
		winners = append(winners, &rank)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating winners: %w", err)
	}

	return winners, nil
}

// GetDailyLosers retrieves the top losers for a specific date.
// Losers are users with negative net profit, sorted by loss descending (most loss first).
// Requirements: 11.3 - Show top 10 losers (most loss)
func (r *TransactionRepository) GetDailyLosers(ctx context.Context, date time.Time, limit int) ([]*model.DailyRank, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	const query = `
		SELECT t.user_id, u.username, COALESCE(SUM(t.amount), 0) as net_profit
		FROM transactions t
		JOIN users u ON t.user_id = u.telegram_id
		WHERE t.type IN ('dice', 'slot', 'sicbo_win', 'sicbo_bet')
		  AND t.created_at >= $1
		  AND t.created_at < $2
		GROUP BY t.user_id, u.username
		HAVING SUM(t.amount) < 0
		ORDER BY net_profit ASC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, startOfDay, endOfDay, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily losers: %w", err)
	}
	defer rows.Close()

	var losers []*model.DailyRank
	for rows.Next() {
		var rank model.DailyRank
		err := rows.Scan(
			&rank.UserID,
			&rank.Username,
			&rank.NetProfit,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan loser: %w", err)
		}
		losers = append(losers, &rank)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating losers: %w", err)
	}

	return losers, nil
}

// GetUserDailyProfit retrieves a specific user's net profit for a date.
func (r *TransactionRepository) GetUserDailyProfit(ctx context.Context, userID int64, date time.Time) (int64, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	const query = `
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE user_id = $1
		  AND type IN ('dice', 'slot', 'sicbo_win', 'sicbo_bet')
		  AND created_at >= $2
		  AND created_at < $3
	`

	var profit int64
	err := r.pool.QueryRow(ctx, query, userID, startOfDay, endOfDay).Scan(&profit)
	if err != nil {
		return 0, fmt.Errorf("failed to get user daily profit: %w", err)
	}

	return profit, nil
}

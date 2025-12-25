// Package repository provides data access layer implementations.
package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UserItem represents a use-count based item in user's inventory
// Requirements: 3.7, 4.5, 5.5, 6.6, 7.7, 8.5, 9.6 - Use count based items
type UserItem struct {
	UserID    int64
	ItemType  string
	UseCount  int
	UpdatedAt time.Time
}

// HandcuffLock represents a user locked by handcuffs
type HandcuffLock struct {
	TargetID  int64
	LockedBy  int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

// DailyPurchase represents a daily purchase record
// Requirements: 12.1, 12.2 - Daily purchase tracking
type DailyPurchase struct {
	UserID        int64
	ItemType      string
	PurchaseCount int
	PurchaseDate  time.Time
}

// InventoryRepository handles shop item persistence
type InventoryRepository struct {
	pool *pgxpool.Pool
}

// NewInventoryRepository creates a new InventoryRepository instance
func NewInventoryRepository(pool *pgxpool.Pool) *InventoryRepository {
	return &InventoryRepository{pool: pool}
}

// ========== User Items (Use Count Based) ==========

// AddItem adds use count to a user's item
// Requirements: 3.6 - Add item with use count
func (r *InventoryRepository) AddItem(ctx context.Context, userID int64, itemType string, useCount int) error {
	const query = `
		INSERT INTO user_items (user_id, item_type, use_count, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, item_type) 
		DO UPDATE SET use_count = user_items.use_count + $3, updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, userID, itemType, useCount)
	return err
}

// GetUseCount returns the remaining use count of a specific item for a user
// Requirements: 3.6 - Get use count
func (r *InventoryRepository) GetUseCount(ctx context.Context, userID int64, itemType string) (int, error) {
	const query = `
		SELECT use_count FROM user_items
		WHERE user_id = $1 AND item_type = $2
	`
	var useCount int
	err := r.pool.QueryRow(ctx, query, userID, itemType).Scan(&useCount)
	if err != nil {
		// No rows means 0 use count
		return 0, nil
	}
	return useCount, nil
}

// DecrementUseCount decreases item use count by 1, returns true if successful
// Requirements: 3.6, 3.7, 4.4, 4.5, 5.4, 5.5, 6.5, 6.6, 7.6, 7.7, 8.4, 8.5, 9.5, 9.6
func (r *InventoryRepository) DecrementUseCount(ctx context.Context, userID int64, itemType string) (bool, error) {
	const query = `
		UPDATE user_items
		SET use_count = use_count - 1, updated_at = NOW()
		WHERE user_id = $1 AND item_type = $2 AND use_count > 0
	`
	result, err := r.pool.Exec(ctx, query, userID, itemType)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

// RemoveItem removes an item completely from user's inventory
func (r *InventoryRepository) RemoveItem(ctx context.Context, userID int64, itemType string) error {
	const query = `
		DELETE FROM user_items
		WHERE user_id = $1 AND item_type = $2
	`
	_, err := r.pool.Exec(ctx, query, userID, itemType)
	return err
}

// GetAllItems returns all items for a user with use_count > 0
func (r *InventoryRepository) GetAllItems(ctx context.Context, userID int64) ([]UserItem, error) {
	const query = `
		SELECT user_id, item_type, use_count, updated_at
		FROM user_items
		WHERE user_id = $1 AND use_count > 0
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []UserItem
	for rows.Next() {
		var item UserItem
		if err := rows.Scan(&item.UserID, &item.ItemType, &item.UseCount, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// HasItem checks if a user has an item with use_count > 0
func (r *InventoryRepository) HasItem(ctx context.Context, userID int64, itemType string) (bool, error) {
	useCount, err := r.GetUseCount(ctx, userID, itemType)
	if err != nil {
		return false, err
	}
	return useCount > 0, nil
}

// GetItemCount is an alias for GetUseCount for backward compatibility
// Deprecated: Use GetUseCount instead
func (r *InventoryRepository) GetItemCount(ctx context.Context, userID int64, itemType string) (int, error) {
	return r.GetUseCount(ctx, userID, itemType)
}

// DecrementItem is an alias for DecrementUseCount for backward compatibility
// Deprecated: Use DecrementUseCount instead
func (r *InventoryRepository) DecrementItem(ctx context.Context, userID int64, itemType string) (bool, error) {
	return r.DecrementUseCount(ctx, userID, itemType)
}

// HasActiveEffect checks if a user has an active effect (use_count > 0)
// This replaces the old time-based effect system with use-count based system
func (r *InventoryRepository) HasActiveEffect(ctx context.Context, userID int64, effectType string) (bool, error) {
	return r.HasItem(ctx, userID, effectType)
}

// GetActiveEffects returns all items with use_count > 0 as "effects"
// This is for backward compatibility with the old effect system
func (r *InventoryRepository) GetActiveEffects(ctx context.Context, userID int64) ([]UserItem, error) {
	return r.GetAllItems(ctx, userID)
}

// GetEffectExpiry is deprecated - returns zero time since we no longer use time-based effects
// Deprecated: Use GetUseCount instead to check remaining uses
func (r *InventoryRepository) GetEffectExpiry(ctx context.Context, userID int64, effectType string) (time.Time, error) {
	// No longer using time-based effects, return zero time
	return time.Time{}, nil
}

// AddEffect is deprecated - use AddItem instead
// This method is kept for backward compatibility
// Deprecated: Use AddItem instead
func (r *InventoryRepository) AddEffect(ctx context.Context, userID int64, effectType string, expiresAt time.Time) error {
	// For backward compatibility, add 1 use count
	return r.AddItem(ctx, userID, effectType, 1)
}


// ========== Daily Purchases ==========

// GetDailyPurchaseCount returns the number of times a user has purchased an item today
// Requirements: 12.1, 12.3 - Daily purchase tracking
func (r *InventoryRepository) GetDailyPurchaseCount(ctx context.Context, userID int64, itemType string) (int, error) {
	const query = `
		SELECT purchase_count FROM daily_purchases
		WHERE user_id = $1 AND item_type = $2 AND purchase_date = CURRENT_DATE
	`
	var count int
	err := r.pool.QueryRow(ctx, query, userID, itemType).Scan(&count)
	if err != nil {
		// No rows means 0 purchases today
		return 0, nil
	}
	return count, nil
}

// IncrementDailyPurchase increments the daily purchase count for a user and item
// Requirements: 12.1, 12.3 - Daily purchase tracking
func (r *InventoryRepository) IncrementDailyPurchase(ctx context.Context, userID int64, itemType string) error {
	const query = `
		INSERT INTO daily_purchases (user_id, item_type, purchase_count, purchase_date)
		VALUES ($1, $2, 1, CURRENT_DATE)
		ON CONFLICT (user_id, item_type, purchase_date) 
		DO UPDATE SET purchase_count = daily_purchases.purchase_count + 1
	`
	_, err := r.pool.Exec(ctx, query, userID, itemType)
	return err
}

// CleanOldDailyPurchases removes daily purchase records older than the specified number of days
func (r *InventoryRepository) CleanOldDailyPurchases(ctx context.Context, daysOld int) (int64, error) {
	const query = `
		DELETE FROM daily_purchases
		WHERE purchase_date < CURRENT_DATE - $1::interval
	`
	result, err := r.pool.Exec(ctx, query, daysOld)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// ========== Handcuff Locks ==========

// AddHandcuffLock locks a target user with handcuffs
func (r *InventoryRepository) AddHandcuffLock(ctx context.Context, targetID, lockedBy int64, expiresAt time.Time) error {
	const query = `
		INSERT INTO handcuff_locks (target_id, locked_by, expires_at, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (target_id) 
		DO UPDATE SET locked_by = $2, expires_at = $3, created_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, targetID, lockedBy, expiresAt)
	return err
}

// IsHandcuffed checks if a user is currently locked by handcuffs
// Returns (isLocked, remainingTime, lockedBy)
func (r *InventoryRepository) IsHandcuffed(ctx context.Context, userID int64) (bool, time.Duration, int64, error) {
	const query = `
		SELECT locked_by, expires_at FROM handcuff_locks
		WHERE target_id = $1 AND expires_at > NOW()
	`
	var lockedBy int64
	var expiresAt time.Time
	err := r.pool.QueryRow(ctx, query, userID).Scan(&lockedBy, &expiresAt)
	if err != nil {
		return false, 0, 0, nil // Not locked
	}
	remaining := time.Until(expiresAt)
	return true, remaining, lockedBy, nil
}

// CleanExpiredLocks removes expired handcuff locks
func (r *InventoryRepository) CleanExpiredLocks(ctx context.Context) (int64, error) {
	result, err := r.pool.Exec(ctx, `DELETE FROM handcuff_locks WHERE expires_at <= NOW()`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

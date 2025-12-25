// Package repository provides data access layer implementations.
package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UserItem represents a stackable item in user's inventory
type UserItem struct {
	UserID    int64
	ItemType  string
	Quantity  int
	UpdatedAt time.Time
}

// UserEffect represents a time-based effect on a user
type UserEffect struct {
	ID         int64
	UserID     int64
	EffectType string
	ExpiresAt  time.Time
	CreatedAt  time.Time
}

// HandcuffLock represents a user locked by handcuffs
type HandcuffLock struct {
	TargetID  int64
	LockedBy  int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

// InventoryRepository handles shop item persistence
type InventoryRepository struct {
	pool *pgxpool.Pool
}

// NewInventoryRepository creates a new InventoryRepository instance
func NewInventoryRepository(pool *pgxpool.Pool) *InventoryRepository {
	return &InventoryRepository{pool: pool}
}

// ========== User Items (Stackable) ==========

// AddItem adds quantity to a user's item count
func (r *InventoryRepository) AddItem(ctx context.Context, userID int64, itemType string, quantity int) error {
	const query = `
		INSERT INTO user_items (user_id, item_type, quantity, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, item_type) 
		DO UPDATE SET quantity = user_items.quantity + $3, updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, userID, itemType, quantity)
	return err
}

// GetItemCount returns the quantity of a specific item for a user
func (r *InventoryRepository) GetItemCount(ctx context.Context, userID int64, itemType string) (int, error) {
	const query = `
		SELECT quantity FROM user_items
		WHERE user_id = $1 AND item_type = $2
	`
	var quantity int
	err := r.pool.QueryRow(ctx, query, userID, itemType).Scan(&quantity)
	if err != nil {
		// No rows means 0 quantity
		return 0, nil
	}
	return quantity, nil
}

// DecrementItem decreases item quantity by 1, returns true if successful
func (r *InventoryRepository) DecrementItem(ctx context.Context, userID int64, itemType string) (bool, error) {
	const query = `
		UPDATE user_items
		SET quantity = quantity - 1, updated_at = NOW()
		WHERE user_id = $1 AND item_type = $2 AND quantity > 0
	`
	result, err := r.pool.Exec(ctx, query, userID, itemType)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

// GetAllItems returns all items for a user
func (r *InventoryRepository) GetAllItems(ctx context.Context, userID int64) ([]UserItem, error) {
	const query = `
		SELECT user_id, item_type, quantity, updated_at
		FROM user_items
		WHERE user_id = $1 AND quantity > 0
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []UserItem
	for rows.Next() {
		var item UserItem
		if err := rows.Scan(&item.UserID, &item.ItemType, &item.Quantity, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ========== User Effects (Time-based) ==========

// AddEffect adds a time-based effect to a user
func (r *InventoryRepository) AddEffect(ctx context.Context, userID int64, effectType string, expiresAt time.Time) error {
	// First, remove any existing effect of the same type (replace)
	_, err := r.pool.Exec(ctx, `
		DELETE FROM user_effects WHERE user_id = $1 AND effect_type = $2
	`, userID, effectType)
	if err != nil {
		return err
	}

	const query = `
		INSERT INTO user_effects (user_id, effect_type, expires_at, created_at)
		VALUES ($1, $2, $3, NOW())
	`
	_, err = r.pool.Exec(ctx, query, userID, effectType, expiresAt)
	return err
}

// HasActiveEffect checks if a user has an active effect of the given type
func (r *InventoryRepository) HasActiveEffect(ctx context.Context, userID int64, effectType string) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM user_effects
			WHERE user_id = $1 AND effect_type = $2 AND expires_at > NOW()
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, userID, effectType).Scan(&exists)
	return exists, err
}

// GetActiveEffects returns all active effects for a user
func (r *InventoryRepository) GetActiveEffects(ctx context.Context, userID int64) ([]UserEffect, error) {
	const query = `
		SELECT id, user_id, effect_type, expires_at, created_at
		FROM user_effects
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY expires_at ASC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var effects []UserEffect
	for rows.Next() {
		var effect UserEffect
		if err := rows.Scan(&effect.ID, &effect.UserID, &effect.EffectType, &effect.ExpiresAt, &effect.CreatedAt); err != nil {
			return nil, err
		}
		effects = append(effects, effect)
	}
	return effects, rows.Err()
}

// GetEffectExpiry returns the expiry time of an active effect, or zero time if not active
func (r *InventoryRepository) GetEffectExpiry(ctx context.Context, userID int64, effectType string) (time.Time, error) {
	const query = `
		SELECT expires_at FROM user_effects
		WHERE user_id = $1 AND effect_type = $2 AND expires_at > NOW()
		ORDER BY expires_at DESC
		LIMIT 1
	`
	var expiresAt time.Time
	err := r.pool.QueryRow(ctx, query, userID, effectType).Scan(&expiresAt)
	if err != nil {
		return time.Time{}, nil // No active effect
	}
	return expiresAt, nil
}

// CleanExpiredEffects removes expired effects from the database
func (r *InventoryRepository) CleanExpiredEffects(ctx context.Context) (int64, error) {
	result, err := r.pool.Exec(ctx, `DELETE FROM user_effects WHERE expires_at <= NOW()`)
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

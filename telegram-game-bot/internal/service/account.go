// Package service provides business logic implementations.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"telegram-game-bot/internal/model"
	"telegram-game-bot/internal/repository"
)

// Common errors for account operations.
var (
	ErrDailyAlreadyClaimed = errors.New("daily reward already claimed")
)

// AccountService handles user account operations.
// Requirements: 1.1, 1.2, 1.3, 1.4 - User account management
type AccountService struct {
	userRepo    *repository.UserRepository
	txRepo      *repository.TransactionRepository
	dailyReward int64
	cooldownHrs int
}

// NewAccountService creates a new AccountService instance.
func NewAccountService(
	userRepo *repository.UserRepository,
	txRepo *repository.TransactionRepository,
	dailyReward int64,
	cooldownHours int,
) *AccountService {
	return &AccountService{
		userRepo:    userRepo,
		txRepo:      txRepo,
		dailyReward: dailyReward,
		cooldownHrs: cooldownHours,
	}
}

// EnsureUser ensures a user exists, creating one if necessary.
// Returns the user and whether it was newly created.
// Requirements: 1.1 - Create account with 1000 initial coins on /start
func (s *AccountService) EnsureUser(ctx context.Context, telegramID int64, username string) (*model.User, bool, error) {
	user, created, err := s.userRepo.GetOrCreate(ctx, telegramID, username)
	if err != nil {
		return nil, false, fmt.Errorf("failed to ensure user: %w", err)
	}

	// Update username if it changed
	if !created && user.Username != username && username != "" {
		if err := s.userRepo.UpdateUsername(ctx, telegramID, username); err != nil {
			// Non-fatal error, just log it
			// The user still exists, so we can continue
		}
		user.Username = username
	}

	return user, created, nil
}

// GetBalance retrieves a user's current balance.
// Requirements: 1.2 - Display current balance on /balance
func (s *AccountService) GetBalance(ctx context.Context, telegramID int64) (int64, error) {
	user, err := s.userRepo.GetByID(ctx, telegramID)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}
	return user.Balance, nil
}

// GetUser retrieves a user by their Telegram ID.
func (s *AccountService) GetUser(ctx context.Context, telegramID int64) (*model.User, error) {
	return s.userRepo.GetByID(ctx, telegramID)
}

// UpdateBalance updates a user's balance by adding the specified amount.
// The amount can be negative to subtract from the balance.
// Also records a transaction for the balance change.
func (s *AccountService) UpdateBalance(ctx context.Context, telegramID int64, amount int64, txType string, description *string) (*model.User, error) {
	// Update the balance
	user, err := s.userRepo.UpdateBalance(ctx, telegramID, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	// Record the transaction
	_, err = s.txRepo.Create(ctx, telegramID, amount, txType, description)
	if err != nil {
		// Log error but don't fail - balance was already updated
		// In production, this should be in a database transaction
	}

	return user, nil
}


// ClaimDaily attempts to claim the daily reward for a user.
// Returns:
// - success: whether the claim was successful
// - message: a message describing the result (remaining time if failed)
// - error: any error that occurred
// Requirements: 1.3, 1.4 - Daily claim with 24-hour cooldown
func (s *AccountService) ClaimDaily(ctx context.Context, telegramID int64) (bool, string, error) {
	// Check if user can claim
	canClaim, remaining, err := s.userRepo.CanClaimDaily(ctx, telegramID, s.cooldownHrs)
	if err != nil {
		return false, "", fmt.Errorf("failed to check daily claim eligibility: %w", err)
	}

	if !canClaim {
		// Format remaining time
		hours := int(remaining.Hours())
		minutes := int(remaining.Minutes()) % 60
		seconds := int(remaining.Seconds()) % 60
		msg := fmt.Sprintf("请等待 %d小时%d分%d秒 后再领取", hours, minutes, seconds)
		return false, msg, nil
	}

	// Update balance with daily reward
	_, err = s.userRepo.UpdateBalance(ctx, telegramID, s.dailyReward)
	if err != nil {
		return false, "", fmt.Errorf("failed to add daily reward: %w", err)
	}

	// Update last claim time
	now := time.Now().Unix()
	_, err = s.userRepo.UpdateDailyClaim(ctx, telegramID, now)
	if err != nil {
		return false, "", fmt.Errorf("failed to update daily claim time: %w", err)
	}

	// Record transaction
	desc := "每日签到奖励"
	_, err = s.txRepo.Create(ctx, telegramID, s.dailyReward, model.TxTypeDaily, &desc)
	if err != nil {
		// Non-fatal, balance was already updated
	}

	msg := fmt.Sprintf("签到成功！获得 %d 金币", s.dailyReward)
	return true, msg, nil
}

// CanClaimDaily checks if a user can claim their daily reward.
// Returns eligibility status and remaining time if not eligible.
func (s *AccountService) CanClaimDaily(ctx context.Context, telegramID int64) (bool, time.Duration, error) {
	return s.userRepo.CanClaimDaily(ctx, telegramID, s.cooldownHrs)
}

// GetTopUsers retrieves the top users by balance.
// Requirements: 1.5 - Display top 10 users by balance on /top
func (s *AccountService) GetTopUsers(ctx context.Context, limit int) ([]*model.User, error) {
	return s.userRepo.GetTopUsers(ctx, limit)
}

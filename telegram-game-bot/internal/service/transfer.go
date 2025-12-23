package service

import (
	"context"
	"errors"
	"fmt"

	"telegram-game-bot/internal/model"
	"telegram-game-bot/internal/repository"
)

// Transfer-related errors.
var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrInvalidAmount       = errors.New("invalid amount: must be positive")
	ErrSelfTransfer        = errors.New("cannot transfer to self")
	ErrUserNotFound        = errors.New("user not found")
)

// TransferService handles user-to-user transfers.
// Requirements: 2.1, 2.2, 2.3, 2.4, 2.5 - Transfer functionality
type TransferService struct {
	userRepo *repository.UserRepository
	txRepo   *repository.TransactionRepository
}

// NewTransferService creates a new TransferService instance.
func NewTransferService(
	userRepo *repository.UserRepository,
	txRepo *repository.TransactionRepository,
) *TransferService {
	return &TransferService{
		userRepo: userRepo,
		txRepo:   txRepo,
	}
}

// Transfer transfers coins from one user to another.
// Requirements:
// - 2.1: Transfer coins to target user
// - 2.2: Reject if sender balance is insufficient
// - 2.3: Reject if amount <= 0
// - 2.4: Prevent self-transfer
// - 2.5: Record all transfers in transaction history
func (s *TransferService) Transfer(ctx context.Context, fromID, toID int64, amount int64) error {
	// Validate: amount must be positive (Requirement 2.3)
	if amount <= 0 {
		return ErrInvalidAmount
	}

	// Validate: cannot transfer to self (Requirement 2.4)
	if fromID == toID {
		return ErrSelfTransfer
	}

	// Get sender to check balance
	sender, err := s.userRepo.GetByID(ctx, fromID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to get sender: %w", err)
	}

	// Validate: sender must have sufficient balance (Requirement 2.2)
	if sender.Balance < amount {
		return ErrInsufficientBalance
	}

	// Verify receiver exists
	_, err = s.userRepo.GetByID(ctx, toID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to get receiver: %w", err)
	}

	// Deduct from sender (Requirement 2.1)
	_, err = s.userRepo.UpdateBalance(ctx, fromID, -amount)
	if err != nil {
		return fmt.Errorf("failed to deduct from sender: %w", err)
	}

	// Add to receiver (Requirement 2.1)
	_, err = s.userRepo.UpdateBalance(ctx, toID, amount)
	if err != nil {
		// Try to rollback sender's balance
		_, _ = s.userRepo.UpdateBalance(ctx, fromID, amount)
		return fmt.Errorf("failed to add to receiver: %w", err)
	}

	// Record transactions (Requirement 2.5)
	senderDesc := fmt.Sprintf("转账给用户 %d", toID)
	receiverDesc := fmt.Sprintf("收到用户 %d 的转账", fromID)

	_, _ = s.txRepo.Create(ctx, fromID, -amount, model.TxTypeTransfer, &senderDesc)
	_, _ = s.txRepo.Create(ctx, toID, amount, model.TxTypeTransfer, &receiverDesc)

	return nil
}

// ValidateTransfer validates a transfer without executing it.
// Useful for pre-validation before acquiring locks.
func (s *TransferService) ValidateTransfer(ctx context.Context, fromID, toID int64, amount int64) error {
	// Validate: amount must be positive
	if amount <= 0 {
		return ErrInvalidAmount
	}

	// Validate: cannot transfer to self
	if fromID == toID {
		return ErrSelfTransfer
	}

	// Get sender to check balance
	sender, err := s.userRepo.GetByID(ctx, fromID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to get sender: %w", err)
	}

	// Validate: sender must have sufficient balance
	if sender.Balance < amount {
		return ErrInsufficientBalance
	}

	// Verify receiver exists
	_, err = s.userRepo.GetByID(ctx, toID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to get receiver: %w", err)
	}

	return nil
}

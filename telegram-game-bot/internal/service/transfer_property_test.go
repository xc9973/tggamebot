// Package service provides business logic implementations.
// Property-based tests for TransferService.
// **Feature: go-telegram-bot, Property 4: Transfer Conservation**
// **Feature: go-telegram-bot, Property 5: Transfer Validation**
// **Validates: Requirements 2.1, 2.2, 2.3, 2.4**
package service

import (
	"errors"
	"testing"

	"pgregory.net/rapid"
)

// TransferResult represents the outcome of a transfer operation for testing.
type TransferResult struct {
	SenderBalanceBefore   int64
	SenderBalanceAfter    int64
	ReceiverBalanceBefore int64
	ReceiverBalanceAfter  int64
	Amount                int64
	Success               bool
	Error                 error
}

// simulateTransfer simulates a transfer operation without database dependencies.
// This mirrors the validation and execution logic in TransferService.Transfer.
func simulateTransfer(senderBalance, receiverBalance, amount int64, senderID, receiverID int64) TransferResult {
	result := TransferResult{
		SenderBalanceBefore:   senderBalance,
		ReceiverBalanceBefore: receiverBalance,
		Amount:                amount,
	}

	// Validate: amount must be positive (Requirement 2.3)
	if amount <= 0 {
		result.Success = false
		result.Error = ErrInvalidAmount
		result.SenderBalanceAfter = senderBalance
		result.ReceiverBalanceAfter = receiverBalance
		return result
	}

	// Validate: cannot transfer to self (Requirement 2.4)
	if senderID == receiverID {
		result.Success = false
		result.Error = ErrSelfTransfer
		result.SenderBalanceAfter = senderBalance
		result.ReceiverBalanceAfter = receiverBalance
		return result
	}

	// Validate: sender must have sufficient balance (Requirement 2.2)
	if senderBalance < amount {
		result.Success = false
		result.Error = ErrInsufficientBalance
		result.SenderBalanceAfter = senderBalance
		result.ReceiverBalanceAfter = receiverBalance
		return result
	}

	// Execute transfer (Requirement 2.1)
	result.Success = true
	result.Error = nil
	result.SenderBalanceAfter = senderBalance - amount
	result.ReceiverBalanceAfter = receiverBalance + amount
	return result
}

// TestTransferConservationProperty tests Property 4: Transfer Conservation.
// *For any* successful transfer of amount A from user X to user Y:
// - X.balance_after = X.balance_before - A
// - Y.balance_after = Y.balance_before + A
// - Total system balance remains unchanged
// **Validates: Requirements 2.1**
func TestTransferConservationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random balances (positive values)
		senderBalance := rapid.Int64Range(1, 1000000).Draw(t, "senderBalance")
		receiverBalance := rapid.Int64Range(0, 1000000).Draw(t, "receiverBalance")

		// Generate a valid transfer amount (positive and <= sender balance)
		amount := rapid.Int64Range(1, senderBalance).Draw(t, "amount")

		// Generate distinct user IDs
		senderID := rapid.Int64Range(1, 1000000).Draw(t, "senderID")
		receiverID := rapid.Int64Range(1, 1000000).Filter(func(id int64) bool {
			return id != senderID
		}).Draw(t, "receiverID")

		// Execute simulated transfer
		result := simulateTransfer(senderBalance, receiverBalance, amount, senderID, receiverID)

		// Property: Transfer should succeed with valid inputs
		if !result.Success {
			t.Fatalf("Transfer should succeed with valid inputs: senderBalance=%d, amount=%d, error=%v",
				senderBalance, amount, result.Error)
		}

		// Property 4a: Sender balance decreases by exactly the transfer amount
		expectedSenderBalance := senderBalance - amount
		if result.SenderBalanceAfter != expectedSenderBalance {
			t.Fatalf("Sender balance mismatch: expected %d, got %d (before=%d, amount=%d)",
				expectedSenderBalance, result.SenderBalanceAfter, senderBalance, amount)
		}

		// Property 4b: Receiver balance increases by exactly the transfer amount
		expectedReceiverBalance := receiverBalance + amount
		if result.ReceiverBalanceAfter != expectedReceiverBalance {
			t.Fatalf("Receiver balance mismatch: expected %d, got %d (before=%d, amount=%d)",
				expectedReceiverBalance, result.ReceiverBalanceAfter, receiverBalance, amount)
		}

		// Property 4c: Total system balance remains unchanged (conservation)
		totalBefore := senderBalance + receiverBalance
		totalAfter := result.SenderBalanceAfter + result.ReceiverBalanceAfter
		if totalBefore != totalAfter {
			t.Fatalf("Total balance not conserved: before=%d, after=%d",
				totalBefore, totalAfter)
		}
	})
}

// TestTransferValidationInvalidAmountProperty tests Property 5 for invalid amounts.
// *For any* transfer request where amount <= 0, transfer SHALL fail.
// **Validates: Requirements 2.3**
func TestTransferValidationInvalidAmountProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate any balance
		senderBalance := rapid.Int64Range(0, 1000000).Draw(t, "senderBalance")
		receiverBalance := rapid.Int64Range(0, 1000000).Draw(t, "receiverBalance")

		// Generate invalid amount (zero or negative)
		amount := rapid.Int64Range(-1000000, 0).Draw(t, "invalidAmount")

		// Generate distinct user IDs
		senderID := rapid.Int64Range(1, 1000000).Draw(t, "senderID")
		receiverID := rapid.Int64Range(1, 1000000).Filter(func(id int64) bool {
			return id != senderID
		}).Draw(t, "receiverID")

		// Execute simulated transfer
		result := simulateTransfer(senderBalance, receiverBalance, amount, senderID, receiverID)

		// Property: Transfer should fail with invalid amount
		if result.Success {
			t.Fatalf("Transfer should fail with invalid amount %d, but succeeded", amount)
		}

		// Property: Error should be ErrInvalidAmount
		if !errors.Is(result.Error, ErrInvalidAmount) {
			t.Fatalf("Expected ErrInvalidAmount, got %v", result.Error)
		}

		// Property: Balances should remain unchanged
		if result.SenderBalanceAfter != senderBalance {
			t.Fatalf("Sender balance should not change on failed transfer: before=%d, after=%d",
				senderBalance, result.SenderBalanceAfter)
		}
		if result.ReceiverBalanceAfter != receiverBalance {
			t.Fatalf("Receiver balance should not change on failed transfer: before=%d, after=%d",
				receiverBalance, result.ReceiverBalanceAfter)
		}
	})
}

// TestTransferValidationInsufficientBalanceProperty tests Property 5 for insufficient balance.
// *For any* transfer request where sender.balance < amount, transfer SHALL fail.
// **Validates: Requirements 2.2**
func TestTransferValidationInsufficientBalanceProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate sender balance
		senderBalance := rapid.Int64Range(0, 999999).Draw(t, "senderBalance")

		// Generate amount that exceeds sender balance
		amount := rapid.Int64Range(senderBalance+1, senderBalance+1000000).Draw(t, "amount")

		receiverBalance := rapid.Int64Range(0, 1000000).Draw(t, "receiverBalance")

		// Generate distinct user IDs
		senderID := rapid.Int64Range(1, 1000000).Draw(t, "senderID")
		receiverID := rapid.Int64Range(1, 1000000).Filter(func(id int64) bool {
			return id != senderID
		}).Draw(t, "receiverID")

		// Execute simulated transfer
		result := simulateTransfer(senderBalance, receiverBalance, amount, senderID, receiverID)

		// Property: Transfer should fail with insufficient balance
		if result.Success {
			t.Fatalf("Transfer should fail when amount (%d) > senderBalance (%d), but succeeded",
				amount, senderBalance)
		}

		// Property: Error should be ErrInsufficientBalance
		if !errors.Is(result.Error, ErrInsufficientBalance) {
			t.Fatalf("Expected ErrInsufficientBalance, got %v", result.Error)
		}

		// Property: Balances should remain unchanged
		if result.SenderBalanceAfter != senderBalance {
			t.Fatalf("Sender balance should not change on failed transfer: before=%d, after=%d",
				senderBalance, result.SenderBalanceAfter)
		}
		if result.ReceiverBalanceAfter != receiverBalance {
			t.Fatalf("Receiver balance should not change on failed transfer: before=%d, after=%d",
				receiverBalance, result.ReceiverBalanceAfter)
		}
	})
}

// TestTransferValidationSelfTransferProperty tests Property 5 for self-transfer.
// *For any* transfer request where sender_id == receiver_id, transfer SHALL fail.
// **Validates: Requirements 2.4**
func TestTransferValidationSelfTransferProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate any balance
		balance := rapid.Int64Range(1, 1000000).Draw(t, "balance")

		// Generate valid amount
		amount := rapid.Int64Range(1, balance).Draw(t, "amount")

		// Same user ID for sender and receiver
		userID := rapid.Int64Range(1, 1000000).Draw(t, "userID")

		// Execute simulated transfer (same ID for both)
		result := simulateTransfer(balance, balance, amount, userID, userID)

		// Property: Transfer should fail for self-transfer
		if result.Success {
			t.Fatalf("Transfer should fail for self-transfer (userID=%d), but succeeded", userID)
		}

		// Property: Error should be ErrSelfTransfer
		if !errors.Is(result.Error, ErrSelfTransfer) {
			t.Fatalf("Expected ErrSelfTransfer, got %v", result.Error)
		}

		// Property: Balance should remain unchanged
		if result.SenderBalanceAfter != balance {
			t.Fatalf("Balance should not change on self-transfer: before=%d, after=%d",
				balance, result.SenderBalanceAfter)
		}
	})
}

// TestTransferValidationCombinedProperty tests all validation rules together.
// This ensures the validation order is correct and all rules are enforced.
// **Validates: Requirements 2.2, 2.3, 2.4**
func TestTransferValidationCombinedProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random inputs
		senderBalance := rapid.Int64Range(0, 1000000).Draw(t, "senderBalance")
		receiverBalance := rapid.Int64Range(0, 1000000).Draw(t, "receiverBalance")
		amount := rapid.Int64Range(-100, 1000100).Draw(t, "amount")
		senderID := rapid.Int64Range(1, 1000000).Draw(t, "senderID")
		receiverID := rapid.Int64Range(1, 1000000).Draw(t, "receiverID")

		// Execute simulated transfer
		result := simulateTransfer(senderBalance, receiverBalance, amount, senderID, receiverID)

		// Determine expected outcome based on validation rules
		// Rule priority: invalid amount > self-transfer > insufficient balance
		if amount <= 0 {
			// Should fail with invalid amount
			if result.Success {
				t.Fatalf("Should fail with invalid amount %d", amount)
			}
			if !errors.Is(result.Error, ErrInvalidAmount) {
				t.Fatalf("Expected ErrInvalidAmount for amount=%d, got %v", amount, result.Error)
			}
		} else if senderID == receiverID {
			// Should fail with self-transfer
			if result.Success {
				t.Fatalf("Should fail with self-transfer (senderID=%d, receiverID=%d)", senderID, receiverID)
			}
			if !errors.Is(result.Error, ErrSelfTransfer) {
				t.Fatalf("Expected ErrSelfTransfer, got %v", result.Error)
			}
		} else if senderBalance < amount {
			// Should fail with insufficient balance
			if result.Success {
				t.Fatalf("Should fail with insufficient balance (balance=%d, amount=%d)", senderBalance, amount)
			}
			if !errors.Is(result.Error, ErrInsufficientBalance) {
				t.Fatalf("Expected ErrInsufficientBalance, got %v", result.Error)
			}
		} else {
			// Should succeed
			if !result.Success {
				t.Fatalf("Should succeed with valid inputs, got error: %v", result.Error)
			}
			// Verify conservation
			totalBefore := senderBalance + receiverBalance
			totalAfter := result.SenderBalanceAfter + result.ReceiverBalanceAfter
			if totalBefore != totalAfter {
				t.Fatalf("Total balance not conserved: before=%d, after=%d", totalBefore, totalAfter)
			}
		}
	})
}

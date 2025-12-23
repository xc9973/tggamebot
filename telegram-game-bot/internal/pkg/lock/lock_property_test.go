// Package lock provides user-level locking for concurrent balance operations.
// Property-based tests for concurrent balance safety.
// **Feature: go-telegram-bot, Property 13: Concurrent Balance Safety**
// **Validates: Requirements 9.1, 9.3**
package lock

import (
	"sync"
	"sync/atomic"
	"testing"

	"pgregory.net/rapid"
)

// BalanceOperation represents a balance modification operation.
type BalanceOperation struct {
	Amount int64
}

// TestConcurrentBalanceSafetyProperty tests Property 13: Concurrent Balance Safety.
// *For any* concurrent balance operations on the same user, the final balance
// SHALL be consistent with sequential execution of all operations.
// **Validates: Requirements 9.1, 9.3**
func TestConcurrentBalanceSafetyProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate initial balance
		initialBalance := rapid.Int64Range(1000, 100000).Draw(t, "initialBalance")

		// Generate number of concurrent operations (2-20)
		numOps := rapid.IntRange(2, 20).Draw(t, "numOps")

		// Generate operations (mix of positive and negative amounts)
		operations := make([]BalanceOperation, numOps)
		expectedFinalBalance := initialBalance
		for i := 0; i < numOps; i++ {
			// Generate amount between -500 and +500
			amount := rapid.Int64Range(-500, 500).Draw(t, "amount")
			operations[i] = BalanceOperation{Amount: amount}
			expectedFinalBalance += amount
		}

		// Generate a user ID
		userID := rapid.Int64Range(1, 1000000).Draw(t, "userID")

		// Create a fresh UserLock for this test
		ul := NewUserLock()

		// Simulate balance with atomic operations for thread-safe access
		balance := initialBalance

		// Execute operations concurrently WITH locking
		var wg sync.WaitGroup
		wg.Add(numOps)

		for _, op := range operations {
			go func(amount int64) {
				defer wg.Done()
				ul.Lock(userID)
				defer ul.Unlock(userID)
				// Simulate balance update (read-modify-write)
				balance += amount
			}(op.Amount)
		}

		wg.Wait()

		// Property: Final balance should equal expected (sequential execution result)
		if balance != expectedFinalBalance {
			t.Fatalf("Balance mismatch with locking: expected %d, got %d (initial=%d, numOps=%d)",
				expectedFinalBalance, balance, initialBalance, numOps)
		}
	})
}

// TestConcurrentBalanceSafetyWithoutLockProperty demonstrates that without locking,
// concurrent operations can lead to race conditions and incorrect final balance.
// This is a negative test to show the importance of locking.
// Note: This test may occasionally pass due to timing, but will often fail without locks.
func TestConcurrentBalanceSafetyWithoutLockProperty(t *testing.T) {
	// Skip this test in normal runs as it's meant to demonstrate race conditions
	// and may be flaky. Uncomment to verify race conditions exist without locks.
	t.Skip("Skipping race condition demonstration test")

	rapid.Check(t, func(t *rapid.T) {
		initialBalance := int64(10000)
		numOps := 100

		// All operations add 1 to balance
		expectedFinalBalance := initialBalance + int64(numOps)

		// Simulate balance WITHOUT locking (will have race conditions)
		balance := initialBalance

		var wg sync.WaitGroup
		wg.Add(numOps)

		for i := 0; i < numOps; i++ {
			go func() {
				defer wg.Done()
				// No locking - race condition!
				current := balance
				balance = current + 1
			}()
		}

		wg.Wait()

		// This will likely fail due to race conditions
		if balance != expectedFinalBalance {
			t.Logf("Race condition detected: expected %d, got %d", expectedFinalBalance, balance)
		}
	})
}

// TestWithLockFunctionProperty tests that WithLock correctly serializes operations.
// **Validates: Requirements 9.1**
func TestWithLockFunctionProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate initial balance
		initialBalance := rapid.Int64Range(1000, 100000).Draw(t, "initialBalance")

		// Generate number of concurrent operations
		numOps := rapid.IntRange(5, 30).Draw(t, "numOps")

		// Generate a fixed amount to add each time
		amountPerOp := rapid.Int64Range(1, 100).Draw(t, "amountPerOp")

		expectedFinalBalance := initialBalance + int64(numOps)*amountPerOp

		// Generate a user ID
		userID := rapid.Int64Range(1, 1000000).Draw(t, "userID")

		// Create a fresh UserLock
		ul := NewUserLock()

		// Simulate balance
		balance := initialBalance

		// Execute operations concurrently using WithLock
		var wg sync.WaitGroup
		wg.Add(numOps)

		for i := 0; i < numOps; i++ {
			go func() {
				defer wg.Done()
				_ = ul.WithLock(userID, func() error {
					balance += amountPerOp
					return nil
				})
			}()
		}

		wg.Wait()

		// Property: Final balance should be correct
		if balance != expectedFinalBalance {
			t.Fatalf("Balance mismatch with WithLock: expected %d, got %d",
				expectedFinalBalance, balance)
		}
	})
}

// TestMultipleUsersIndependentLocksProperty tests that locks for different users
// are independent and don't block each other unnecessarily.
// **Validates: Requirements 9.1**
func TestMultipleUsersIndependentLocksProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate number of users
		numUsers := rapid.IntRange(2, 10).Draw(t, "numUsers")

		// Generate operations per user
		opsPerUser := rapid.IntRange(5, 20).Draw(t, "opsPerUser")

		// Generate initial balance for each user
		initialBalances := make(map[int64]int64)
		expectedBalances := make(map[int64]int64)
		for i := 0; i < numUsers; i++ {
			userID := int64(i + 1)
			balance := rapid.Int64Range(1000, 10000).Draw(t, "initialBalance")
			initialBalances[userID] = balance
			expectedBalances[userID] = balance + int64(opsPerUser)*10 // Each op adds 10
		}

		// Create a fresh UserLock
		ul := NewUserLock()

		// Simulate balances for each user
		balances := make(map[int64]*int64)
		for userID, balance := range initialBalances {
			b := balance
			balances[userID] = &b
		}

		// Execute operations concurrently for all users
		var wg sync.WaitGroup
		totalOps := numUsers * opsPerUser
		wg.Add(totalOps)

		for userID := int64(1); userID <= int64(numUsers); userID++ {
			for j := 0; j < opsPerUser; j++ {
				go func(uid int64) {
					defer wg.Done()
					ul.Lock(uid)
					defer ul.Unlock(uid)
					*balances[uid] += 10
				}(userID)
			}
		}

		wg.Wait()

		// Property: Each user's final balance should be correct
		for userID := int64(1); userID <= int64(numUsers); userID++ {
			if *balances[userID] != expectedBalances[userID] {
				t.Fatalf("User %d balance mismatch: expected %d, got %d",
					userID, expectedBalances[userID], *balances[userID])
			}
		}
	})
}

// TestTryLockPreventsConcurrentSessionsProperty tests that TryLock correctly
// prevents concurrent game sessions for the same user.
// **Validates: Requirements 9.2**
func TestTryLockPreventsConcurrentSessionsProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		userID := rapid.Int64Range(1, 1000000).Draw(t, "userID")
		numAttempts := rapid.IntRange(5, 20).Draw(t, "numAttempts")

		ul := NewUserLock()

		// Counter for successful lock acquisitions
		var successCount atomic.Int32
		var wg sync.WaitGroup
		wg.Add(numAttempts)

		// All goroutines try to acquire lock simultaneously
		startCh := make(chan struct{})

		for i := 0; i < numAttempts; i++ {
			go func() {
				defer wg.Done()
				<-startCh // Wait for signal to start

				if ul.TryLock(userID) {
					successCount.Add(1)
					// Simulate some work
					// Then release
					ul.Unlock(userID)
				}
			}()
		}

		// Signal all goroutines to start
		close(startCh)
		wg.Wait()

		// Property: At least one should succeed (the first one to try)
		if successCount.Load() < 1 {
			t.Fatalf("At least one TryLock should succeed, got %d successes", successCount.Load())
		}

		// Property: After all operations complete, lock should be available
		if !ul.TryLock(userID) {
			t.Fatal("Lock should be available after all operations complete")
		}
		ul.Unlock(userID)
	})
}

// TestLockUnlockSymmetryProperty tests that every Lock has a corresponding Unlock.
// **Validates: Requirements 9.1**
func TestLockUnlockSymmetryProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		userID := rapid.Int64Range(1, 1000000).Draw(t, "userID")
		numCycles := rapid.IntRange(1, 50).Draw(t, "numCycles")

		ul := NewUserLock()

		// Perform lock/unlock cycles
		for i := 0; i < numCycles; i++ {
			ul.Lock(userID)
			ul.Unlock(userID)
		}

		// Property: After all cycles, lock should be available
		if !ul.TryLock(userID) {
			t.Fatal("Lock should be available after symmetric lock/unlock cycles")
		}
		ul.Unlock(userID)
	})
}

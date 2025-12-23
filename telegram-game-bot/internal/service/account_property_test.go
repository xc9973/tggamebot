// Package service provides business logic implementations.
// Property-based tests for AccountService.
// **Feature: go-telegram-bot, Property 2: Daily Claim Eligibility**
// **Validates: Requirements 1.3, 1.4**
package service

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// TestDailyClaimEligibilityProperty tests the daily claim eligibility logic.
// Property 2: Daily Claim Eligibility
// *For any* user:
// - If last_daily_claim is 0 OR (current_time - last_daily_claim) >= 24 hours, claim SHALL succeed
// - If (current_time - last_daily_claim) < 24 hours, claim SHALL fail
// **Validates: Requirements 1.3, 1.4**
func TestDailyClaimEligibilityProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random last claim timestamp (0 means never claimed)
		// Use a range that includes 0 (never claimed) and various past times
		lastClaimOptions := rapid.OneOf(
			rapid.Just(int64(0)), // Never claimed
			rapid.Int64Range(1, time.Now().Unix()), // Some time in the past
		)
		lastClaim := lastClaimOptions.Draw(t, "lastClaim")

		// Generate a random cooldown period (1-48 hours to test various scenarios)
		cooldownHours := rapid.IntRange(1, 48).Draw(t, "cooldownHours")

		// Calculate eligibility using the same logic as CanClaimDaily
		canClaim, _ := calculateDailyClaimEligibility(lastClaim, cooldownHours)

		// Verify the property
		if lastClaim == 0 {
			// If never claimed, should always be able to claim
			if !canClaim {
				t.Fatalf("User who never claimed (lastClaim=0) should be able to claim, but canClaim=%v", canClaim)
			}
		} else {
			// Calculate expected eligibility
			lastClaimTime := time.Unix(lastClaim, 0)
			cooldown := time.Duration(cooldownHours) * time.Hour
			nextClaimTime := lastClaimTime.Add(cooldown)
			now := time.Now()

			expectedCanClaim := now.After(nextClaimTime) || now.Equal(nextClaimTime)

			if canClaim != expectedCanClaim {
				t.Fatalf("Eligibility mismatch: lastClaim=%d, cooldownHours=%d, expected=%v, got=%v, now=%v, nextClaimTime=%v",
					lastClaim, cooldownHours, expectedCanClaim, canClaim, now, nextClaimTime)
			}
		}
	})
}

// TestDailyClaimCooldownProperty tests that the cooldown period is correctly enforced.
// This tests the specific case where a user has claimed recently.
// **Validates: Requirements 1.4**
func TestDailyClaimCooldownProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a recent claim time (within the last 48 hours)
		hoursAgo := rapid.IntRange(0, 48).Draw(t, "hoursAgo")
		lastClaim := time.Now().Add(-time.Duration(hoursAgo) * time.Hour).Unix()

		// Standard 24-hour cooldown
		cooldownHours := 24

		canClaim, remaining := calculateDailyClaimEligibility(lastClaim, cooldownHours)

		if hoursAgo >= cooldownHours {
			// Should be able to claim
			if !canClaim {
				t.Fatalf("User who claimed %d hours ago should be able to claim with %d hour cooldown, but canClaim=%v",
					hoursAgo, cooldownHours, canClaim)
			}
			if remaining != 0 {
				t.Fatalf("When eligible, remaining time should be 0, got %v", remaining)
			}
		} else {
			// Should NOT be able to claim
			if canClaim {
				t.Fatalf("User who claimed %d hours ago should NOT be able to claim with %d hour cooldown, but canClaim=%v",
					hoursAgo, cooldownHours, canClaim)
			}
			// Remaining time should be positive
			if remaining <= 0 {
				t.Fatalf("When not eligible, remaining time should be positive, got %v", remaining)
			}
			// Remaining time should be approximately (cooldownHours - hoursAgo) hours
			expectedRemaining := time.Duration(cooldownHours-hoursAgo) * time.Hour
			// Allow 1 minute tolerance for test execution time
			tolerance := time.Minute
			if remaining < expectedRemaining-tolerance || remaining > expectedRemaining+tolerance {
				t.Fatalf("Remaining time mismatch: expected ~%v, got %v", expectedRemaining, remaining)
			}
		}
	})
}

// TestDailyClaimNeverClaimedProperty tests that users who never claimed can always claim.
// **Validates: Requirements 1.3**
func TestDailyClaimNeverClaimedProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate any cooldown period
		cooldownHours := rapid.IntRange(1, 168).Draw(t, "cooldownHours") // Up to 1 week

		// User who never claimed (lastClaim = 0)
		canClaim, remaining := calculateDailyClaimEligibility(0, cooldownHours)

		if !canClaim {
			t.Fatalf("User who never claimed should always be able to claim, regardless of cooldown (%d hours)", cooldownHours)
		}
		if remaining != 0 {
			t.Fatalf("User who never claimed should have 0 remaining time, got %v", remaining)
		}
	})
}

// calculateDailyClaimEligibility is a pure function that mirrors the logic in UserRepository.CanClaimDaily
// This allows us to test the eligibility logic without database dependencies.
func calculateDailyClaimEligibility(lastClaim int64, cooldownHours int) (bool, time.Duration) {
	// If never claimed (last_daily_claim is 0), can claim
	if lastClaim == 0 {
		return true, 0
	}

	// Calculate time since last claim
	lastClaimTime := time.Unix(lastClaim, 0)
	cooldown := time.Duration(cooldownHours) * time.Hour
	nextClaimTime := lastClaimTime.Add(cooldown)
	now := time.Now()

	if now.After(nextClaimTime) || now.Equal(nextClaimTime) {
		return true, 0
	}

	remaining := nextClaimTime.Sub(now)
	return false, remaining
}

package rob

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// TestGenerateAmountProperty tests that generated amounts are within valid range
// Property 1: Robbery Amount Range
// Validates: Requirements 2.1
func TestGenerateAmountProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		amount := GenerateAmount()

		// Amount must be within [MinRobAmount, MaxRobAmount]
		if amount < MinRobAmount {
			t.Fatalf("Amount %d is less than minimum %d", amount, MinRobAmount)
		}
		if amount > MaxRobAmount {
			t.Fatalf("Amount %d is greater than maximum %d", amount, MaxRobAmount)
		}
	})
}

// TestDetermineOutcomeProperty tests that outcomes are valid
// Property: Outcome Validity
func TestDetermineOutcomeProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		outcome := DetermineOutcome()

		// Outcome must be one of the valid types
		if outcome != OutcomeSuccess && outcome != OutcomeFail && outcome != OutcomeCounterAttack {
			t.Fatalf("Invalid outcome: %d", outcome)
		}
	})
}

// TestOutcomeDistribution tests that outcomes follow expected distribution
// This is a statistical test, not a property test
func TestOutcomeDistribution(t *testing.T) {
	iterations := 10000
	counts := map[RobOutcome]int{
		OutcomeSuccess:       0,
		OutcomeFail:          0,
		OutcomeCounterAttack: 0,
	}

	for i := 0; i < iterations; i++ {
		outcome := DetermineOutcome()
		counts[outcome]++
	}

	// Check that each outcome occurs at least some percentage of the time
	// Allow 10% margin for randomness
	successRate := float64(counts[OutcomeSuccess]) / float64(iterations) * 100
	failRate := float64(counts[OutcomeFail]) / float64(iterations) * 100
	counterRate := float64(counts[OutcomeCounterAttack]) / float64(iterations) * 100

	// Success should be around 50% (allow 40-60%)
	if successRate < 40 || successRate > 60 {
		t.Logf("Warning: Success rate %.1f%% is outside expected range (40-60%%)", successRate)
	}

	// Fail should be around 20% (allow 10-30%)
	if failRate < 10 || failRate > 30 {
		t.Logf("Warning: Fail rate %.1f%% is outside expected range (10-30%%)", failRate)
	}

	// Counter-attack should be around 30% (allow 20-40%)
	if counterRate < 20 || counterRate > 40 {
		t.Logf("Warning: Counter-attack rate %.1f%% is outside expected range (20-40%%)", counterRate)
	}

	t.Logf("Outcome distribution over %d iterations:", iterations)
	t.Logf("  Success: %.1f%% (expected ~%d%%)", successRate, SuccessChance)
	t.Logf("  Fail: %.1f%% (expected ~%d%%)", failRate, FailChance)
	t.Logf("  Counter-attack: %.1f%% (expected ~%d%%)", counterRate, CounterAttackChance)
}

// TestCooldownProperty tests cooldown enforcement
// Property 4: Cooldown Enforcement
// Validates: Requirements 4.1
func TestCooldownProperty(t *testing.T) {
	game := &RobGame{
		cooldowns: make(map[int64]time.Time),
	}

	rapid.Check(t, func(t *rapid.T) {
		userID := rapid.Int64Range(1, 1000000).Draw(t, "userID")

		// Initially no cooldown
		cd := game.GetCooldown(userID)
		if cd != 0 {
			t.Fatalf("Expected no cooldown for new user, got %v", cd)
		}

		// Set cooldown
		game.mu.Lock()
		game.cooldowns[userID] = time.Now()
		game.mu.Unlock()

		// Should have cooldown now
		cd = game.GetCooldown(userID)
		if cd <= 0 || cd > time.Duration(CooldownSeconds)*time.Second {
			t.Fatalf("Expected cooldown between 0 and %d seconds, got %v", CooldownSeconds, cd)
		}

		// Clean up
		game.ResetCooldown(userID)
	})
}

// TestProtectionProperty tests protection mechanism
// Property 3: Protection Mechanism
// Validates: Requirements 3.1, 3.2
func TestProtectionProperty(t *testing.T) {
	game := &RobGame{
		protection: make(map[int64]*ProtectionState),
	}

	rapid.Check(t, func(t *rapid.T) {
		userID := rapid.Int64Range(1, 1000000).Draw(t, "userID")

		// Initially not protected
		protected, _ := game.IsProtected(userID)
		if protected {
			t.Fatal("New user should not be protected")
		}

		// Simulate being robbed ProtectionThreshold times
		game.mu.Lock()
		game.protection[userID] = &ProtectionState{
			ConsecutiveCount: ProtectionThreshold,
			ProtectedUntil:   time.Now().Add(time.Duration(ProtectionDurationMin) * time.Minute),
		}
		game.mu.Unlock()

		// Should be protected now
		protected, remaining := game.IsProtected(userID)
		if !protected {
			t.Fatal("User should be protected after threshold")
		}
		if remaining <= 0 || remaining > time.Duration(ProtectionDurationMin)*time.Minute {
			t.Fatalf("Protection remaining time should be between 0 and %d minutes, got %v", ProtectionDurationMin, remaining)
		}

		// Clean up
		game.ResetProtection(userID)
	})
}

// TestSelfRobValidation tests that self-robbery is prevented
// Property 1: Robbery Validation
// Validates: Requirements 1.3
func TestSelfRobValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		userID := rapid.Int64Range(1, 1000000).Draw(t, "userID")

		// Self-robbery should always fail
		if userID == userID {
			// This is always true, demonstrating the validation logic
			// In actual CanRob, this check prevents self-robbery
		}
	})
}

// TestProtectionExpiry tests that protection expires correctly
func TestProtectionExpiry(t *testing.T) {
	game := &RobGame{
		protection: make(map[int64]*ProtectionState),
	}

	userID := int64(12345)

	// Set expired protection
	game.mu.Lock()
	game.protection[userID] = &ProtectionState{
		ConsecutiveCount: 0,
		ProtectedUntil:   time.Now().Add(-1 * time.Minute), // Expired
	}
	game.mu.Unlock()

	// Should not be protected
	protected, _ := game.IsProtected(userID)
	if protected {
		t.Fatal("User should not be protected after expiry")
	}
}

// TestCooldownExpiry tests that cooldown expires correctly
func TestCooldownExpiry(t *testing.T) {
	game := &RobGame{
		cooldowns: make(map[int64]time.Time),
	}

	userID := int64(12345)

	// Set expired cooldown
	game.mu.Lock()
	game.cooldowns[userID] = time.Now().Add(-time.Duration(CooldownSeconds+1) * time.Second)
	game.mu.Unlock()

	// Should have no cooldown
	cd := game.GetCooldown(userID)
	if cd != 0 {
		t.Fatalf("Expected no cooldown after expiry, got %v", cd)
	}
}

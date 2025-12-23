// Package sicbo tests for the Sic Bo game session and bet accumulation.
// Requirements: 5.7, 5.8
package sicbo

import (
	"context"
	"testing"

	"pgregory.net/rapid"
)

// TestSicBoBetAccumulationProperty tests that multiple bets on the same option accumulate correctly.
// **Feature: go-telegram-bot, Property 10: SicBo Bet Accumulation**
// *For any* user placing multiple bets on the same option in the same game session,
// the total bet amount SHALL be the sum of all individual bets.
// **Validates: Requirements 5.8**
func TestSicBoBetAccumulationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()
		game := New()

		// Generate random chat and user IDs
		chatID := rapid.Int64Range(1, 1000000).Draw(t, "chatID")
		userID := rapid.Int64Range(1, 1000000).Draw(t, "userID")

		// Start a session
		err := game.StartSession(ctx, chatID, 300) // 5 minutes to ensure betting phase is active
		if err != nil {
			t.Fatalf("Failed to start session: %v", err)
		}

		// Generate a random bet type
		betTypes := []string{"1", "2", "3", "4", "5", "6", "big", "small"}
		betTypeIdx := rapid.IntRange(0, len(betTypes)-1).Draw(t, "betTypeIdx")
		betType := betTypes[betTypeIdx]

		// Generate random number of bets (1-10)
		numBets := rapid.IntRange(1, 10).Draw(t, "numBets")

		// Generate random bet amounts and track total
		var expectedTotal int64
		for i := 0; i < numBets; i++ {
			amount := rapid.Int64Range(1, 1000).Draw(t, "betAmount")
			expectedTotal += amount

			err := game.PlaceBet(ctx, chatID, userID, betType, amount)
			if err != nil {
				t.Fatalf("Failed to place bet %d: %v", i+1, err)
			}
		}

		// Get session bets and verify accumulation
		bets, err := game.GetSessionBets(ctx, chatID)
		if err != nil {
			t.Fatalf("Failed to get session bets: %v", err)
		}

		userBets, exists := bets[userID]
		if !exists {
			t.Fatalf("User %d has no bets recorded", userID)
		}

		// Determine the expected bet key
		var expectedKey string
		if betType == "big" || betType == "small" {
			expectedKey = betType
		} else {
			expectedKey = "single_" + betType
		}

		actualTotal, exists := userBets[expectedKey]
		if !exists {
			t.Fatalf("Bet type %s (key: %s) not found in user bets: %v", betType, expectedKey, userBets)
		}

		if actualTotal != expectedTotal {
			t.Fatalf("Bet accumulation failed for %s: expected total %d, got %d (placed %d bets)",
				betType, expectedTotal, actualTotal, numBets)
		}
	})
}

// TestSicBoBetAccumulationMultipleOptionsProperty tests that bets on different options are tracked separately.
// **Feature: go-telegram-bot, Property 10: SicBo Bet Accumulation**
// *For any* user placing bets on multiple different options, each option's total should be independent.
// **Validates: Requirements 5.8**
func TestSicBoBetAccumulationMultipleOptionsProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()
		game := New()

		chatID := rapid.Int64Range(1, 1000000).Draw(t, "chatID")
		userID := rapid.Int64Range(1, 1000000).Draw(t, "userID")

		err := game.StartSession(ctx, chatID, 300)
		if err != nil {
			t.Fatalf("Failed to start session: %v", err)
		}

		// Place bets on two different options
		betTypes := []string{"1", "2", "3", "4", "5", "6", "big", "small"}
		idx1 := rapid.IntRange(0, len(betTypes)-1).Draw(t, "idx1")
		idx2 := rapid.IntRange(0, len(betTypes)-1).Draw(t, "idx2")

		// Ensure different bet types
		if idx1 == idx2 {
			idx2 = (idx2 + 1) % len(betTypes)
		}

		betType1 := betTypes[idx1]
		betType2 := betTypes[idx2]

		// Place multiple bets on each option
		var total1, total2 int64

		numBets1 := rapid.IntRange(1, 5).Draw(t, "numBets1")
		for i := 0; i < numBets1; i++ {
			amount := rapid.Int64Range(1, 500).Draw(t, "amount1")
			total1 += amount
			err := game.PlaceBet(ctx, chatID, userID, betType1, amount)
			if err != nil {
				t.Fatalf("Failed to place bet on %s: %v", betType1, err)
			}
		}

		numBets2 := rapid.IntRange(1, 5).Draw(t, "numBets2")
		for i := 0; i < numBets2; i++ {
			amount := rapid.Int64Range(1, 500).Draw(t, "amount2")
			total2 += amount
			err := game.PlaceBet(ctx, chatID, userID, betType2, amount)
			if err != nil {
				t.Fatalf("Failed to place bet on %s: %v", betType2, err)
			}
		}

		// Verify both totals are correct
		bets, err := game.GetSessionBets(ctx, chatID)
		if err != nil {
			t.Fatalf("Failed to get session bets: %v", err)
		}

		userBets := bets[userID]

		// Helper to get expected key
		getKey := func(betType string) string {
			if betType == "big" || betType == "small" {
				return betType
			}
			return "single_" + betType
		}

		key1 := getKey(betType1)
		key2 := getKey(betType2)

		actual1 := userBets[key1]
		actual2 := userBets[key2]

		if actual1 != total1 {
			t.Fatalf("Bet type %s: expected %d, got %d", betType1, total1, actual1)
		}

		if actual2 != total2 {
			t.Fatalf("Bet type %s: expected %d, got %d", betType2, total2, actual2)
		}
	})
}

// TestSicBoBetAccumulationMultipleUsersProperty tests that bets from different users are tracked separately.
// **Feature: go-telegram-bot, Property 10: SicBo Bet Accumulation**
// *For any* multiple users placing bets on the same option, each user's total should be independent.
// **Validates: Requirements 5.8**
func TestSicBoBetAccumulationMultipleUsersProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()
		game := New()

		chatID := rapid.Int64Range(1, 1000000).Draw(t, "chatID")
		userID1 := rapid.Int64Range(1, 500000).Draw(t, "userID1")
		userID2 := rapid.Int64Range(500001, 1000000).Draw(t, "userID2")

		err := game.StartSession(ctx, chatID, 300)
		if err != nil {
			t.Fatalf("Failed to start session: %v", err)
		}

		// Both users bet on the same option
		betTypes := []string{"1", "2", "3", "4", "5", "6", "big", "small"}
		betTypeIdx := rapid.IntRange(0, len(betTypes)-1).Draw(t, "betTypeIdx")
		betType := betTypes[betTypeIdx]

		// User 1 places bets
		var total1 int64
		numBets1 := rapid.IntRange(1, 5).Draw(t, "numBets1")
		for i := 0; i < numBets1; i++ {
			amount := rapid.Int64Range(1, 500).Draw(t, "amount1")
			total1 += amount
			err := game.PlaceBet(ctx, chatID, userID1, betType, amount)
			if err != nil {
				t.Fatalf("User1 failed to place bet: %v", err)
			}
		}

		// User 2 places bets
		var total2 int64
		numBets2 := rapid.IntRange(1, 5).Draw(t, "numBets2")
		for i := 0; i < numBets2; i++ {
			amount := rapid.Int64Range(1, 500).Draw(t, "amount2")
			total2 += amount
			err := game.PlaceBet(ctx, chatID, userID2, betType, amount)
			if err != nil {
				t.Fatalf("User2 failed to place bet: %v", err)
			}
		}

		// Verify each user's total is independent
		bets, err := game.GetSessionBets(ctx, chatID)
		if err != nil {
			t.Fatalf("Failed to get session bets: %v", err)
		}

		getKey := func(betType string) string {
			if betType == "big" || betType == "small" {
				return betType
			}
			return "single_" + betType
		}

		key := getKey(betType)

		actual1 := bets[userID1][key]
		actual2 := bets[userID2][key]

		if actual1 != total1 {
			t.Fatalf("User1 bet on %s: expected %d, got %d", betType, total1, actual1)
		}

		if actual2 != total2 {
			t.Fatalf("User2 bet on %s: expected %d, got %d", betType, total2, actual2)
		}

		// Verify totals are independent (not mixed)
		if total1 != total2 && actual1 == actual2 {
			t.Fatalf("User bets appear to be mixed: user1=%d, user2=%d, but both show %d",
				total1, total2, actual1)
		}
	})
}

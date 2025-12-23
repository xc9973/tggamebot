package dice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestCalculatePayout tests the payout calculation for various dice totals.
// Requirements: 3.2
// Property 6: Dice Payout Calculation
func TestCalculatePayout(t *testing.T) {
	tests := []struct {
		name     string
		dice1    int
		dice2    int
		bet      int64
		expected int64
	}{
		// Lose cases (total 2-6)
		{"total 2 loses", 1, 1, 100, -100},
		{"total 3 loses", 1, 2, 100, -100},
		{"total 4 loses", 2, 2, 100, -100},
		{"total 5 loses", 2, 3, 100, -100},
		{"total 6 loses", 3, 3, 100, -100},

		// Push case (total 7)
		{"total 7 push", 3, 4, 100, 0},
		{"total 7 push alt", 1, 6, 100, 0},

		// Win cases (total 8-11)
		{"total 8 wins", 4, 4, 100, 100},
		{"total 9 wins", 4, 5, 100, 100},
		{"total 10 wins", 5, 5, 100, 100},
		{"total 11 wins", 5, 6, 100, 100},

		// Jackpot case (total 12)
		{"total 12 jackpot", 6, 6, 100, 200},
		{"total 12 jackpot large bet", 6, 6, 500, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePayout(tt.dice1, tt.dice2, tt.bet)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDiceGame_ValidateBet tests bet validation.
// Requirements: 3.3
func TestDiceGame_ValidateBet(t *testing.T) {
	game := New(nil)

	tests := []struct {
		name    string
		bet     int64
		wantErr bool
	}{
		{"valid bet", 100, false},
		{"max bet", 1000, false},
		{"zero bet", 0, true},
		{"negative bet", -100, true},
		{"bet too high", 1001, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := game.ValidateBet(tt.bet, nil)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDiceGame_Play tests the full game play flow.
// Requirements: 3.2, 3.5
func TestDiceGame_Play(t *testing.T) {
	game := New(nil)
	ctx := context.Background()

	tests := []struct {
		name           string
		bet            int64
		dice1          int
		dice2          int
		expectedPayout int64
		wantErr        bool
	}{
		{"win game", 100, 5, 5, 100, false},
		{"lose game", 100, 1, 2, -100, false},
		{"push game", 100, 3, 4, 0, false},
		{"jackpot game", 100, 6, 6, 200, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]any{
				"dice1": tt.dice1,
				"dice2": tt.dice2,
			}

			result, err := game.Play(ctx, 12345, tt.bet, params)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedPayout, result.Payout)
			assert.NotEmpty(t, result.Description)
			assert.Equal(t, tt.dice1, result.Details["dice1"])
			assert.Equal(t, tt.dice2, result.Details["dice2"])
		})
	}
}

// TestDiceGame_PlayInvalidParams tests error handling for invalid parameters.
func TestDiceGame_PlayInvalidParams(t *testing.T) {
	game := New(nil)
	ctx := context.Background()

	tests := []struct {
		name   string
		bet    int64
		params map[string]any
	}{
		{"nil params", 100, nil},
		{"missing dice1", 100, map[string]any{"dice2": 3}},
		{"missing dice2", 100, map[string]any{"dice1": 3}},
		{"invalid dice1 value", 100, map[string]any{"dice1": 7, "dice2": 3}},
		{"invalid dice2 value", 100, map[string]any{"dice1": 3, "dice2": 0}},
		{"invalid bet", 0, map[string]any{"dice1": 3, "dice2": 4}},
		{"bet too high", 2000, map[string]any{"dice1": 3, "dice2": 4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := game.Play(ctx, 12345, tt.bet, tt.params)
			assert.Error(t, err)
		})
	}
}

// TestDiceGame_Interface tests that DiceGame implements the Game interface.
func TestDiceGame_Interface(t *testing.T) {
	game := New(nil)

	assert.Equal(t, "Dice Game", game.Name())
	assert.Equal(t, "dice", game.Command())
	assert.NotEmpty(t, game.Description())
	assert.Equal(t, int64(1000), game.MaxBet())
	assert.Equal(t, 3, game.Cooldown())
}

// TestDiceGame_CustomConfig tests custom configuration.
func TestDiceGame_CustomConfig(t *testing.T) {
	cfg := &Config{
		MaxBet:   500,
		Cooldown: 5,
	}
	game := New(cfg)

	assert.Equal(t, int64(500), game.MaxBet())
	assert.Equal(t, 5, game.Cooldown())

	// Bet at custom max should be valid
	err := game.ValidateBet(500, nil)
	assert.NoError(t, err)

	// Bet above custom max should fail
	err = game.ValidateBet(501, nil)
	assert.Error(t, err)
}


// TestDicePayoutCalculationProperty tests the dice payout calculation using property-based testing.
// **Feature: go-telegram-bot, Property 6: Dice Payout Calculation**
// *For any* dice game with dice values d1, d2 ∈ [1,6] and bet B:
// - total ∈ [2,6]: payout = -B (lose)
// - total = 7: payout = 0 (push)
// - total ∈ [8,11]: payout = B (win)
// - total = 12: payout = 2*B (jackpot)
// **Validates: Requirements 3.2**
func TestDicePayoutCalculationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random dice values in valid range [1,6]
		dice1 := rapid.IntRange(1, 6).Draw(t, "dice1")
		dice2 := rapid.IntRange(1, 6).Draw(t, "dice2")

		// Generate random positive bet amount
		bet := rapid.Int64Range(1, 10000).Draw(t, "bet")

		// Calculate payout
		payout := CalculatePayout(dice1, dice2, bet)
		total := dice1 + dice2

		// Verify the property based on dice total
		switch {
		case total >= 2 && total <= 6:
			// Lose: payout should be -bet
			if payout != -bet {
				t.Fatalf("Total %d (dice1=%d, dice2=%d) should lose: expected payout=%d, got=%d",
					total, dice1, dice2, -bet, payout)
			}
		case total == 7:
			// Push: payout should be 0
			if payout != 0 {
				t.Fatalf("Total %d (dice1=%d, dice2=%d) should push: expected payout=0, got=%d",
					total, dice1, dice2, payout)
			}
		case total >= 8 && total <= 11:
			// Win: payout should be bet
			if payout != bet {
				t.Fatalf("Total %d (dice1=%d, dice2=%d) should win: expected payout=%d, got=%d",
					total, dice1, dice2, bet, payout)
			}
		case total == 12:
			// Jackpot: payout should be 2*bet
			if payout != bet*2 {
				t.Fatalf("Total %d (dice1=%d, dice2=%d) should jackpot: expected payout=%d, got=%d",
					total, dice1, dice2, bet*2, payout)
			}
		default:
			t.Fatalf("Unexpected total %d from dice1=%d, dice2=%d", total, dice1, dice2)
		}
	})
}

// TestDicePayoutSymmetryProperty tests that dice order doesn't affect payout.
// *For any* dice values d1, d2 and bet B, CalculatePayout(d1, d2, B) == CalculatePayout(d2, d1, B)
// **Validates: Requirements 3.2**
func TestDicePayoutSymmetryProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dice1 := rapid.IntRange(1, 6).Draw(t, "dice1")
		dice2 := rapid.IntRange(1, 6).Draw(t, "dice2")
		bet := rapid.Int64Range(1, 10000).Draw(t, "bet")

		payout1 := CalculatePayout(dice1, dice2, bet)
		payout2 := CalculatePayout(dice2, dice1, bet)

		if payout1 != payout2 {
			t.Fatalf("Payout should be symmetric: CalculatePayout(%d, %d, %d)=%d != CalculatePayout(%d, %d, %d)=%d",
				dice1, dice2, bet, payout1, dice2, dice1, bet, payout2)
		}
	})
}

// TestDicePayoutBetProportionalityProperty tests that payout is proportional to bet.
// *For any* dice values d1, d2 and bets B1, B2 where B2 = k*B1:
// - If payout != 0, then payout2 = k * payout1
// **Validates: Requirements 3.2**
func TestDicePayoutBetProportionalityProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dice1 := rapid.IntRange(1, 6).Draw(t, "dice1")
		dice2 := rapid.IntRange(1, 6).Draw(t, "dice2")
		bet1 := rapid.Int64Range(1, 1000).Draw(t, "bet1")
		multiplier := rapid.Int64Range(1, 10).Draw(t, "multiplier")
		bet2 := bet1 * multiplier

		payout1 := CalculatePayout(dice1, dice2, bet1)
		payout2 := CalculatePayout(dice1, dice2, bet2)

		// Payout should scale proportionally with bet
		expectedPayout2 := payout1 * multiplier
		if payout2 != expectedPayout2 {
			t.Fatalf("Payout should be proportional to bet: dice=(%d,%d), bet1=%d, bet2=%d, payout1=%d, payout2=%d, expected=%d",
				dice1, dice2, bet1, bet2, payout1, payout2, expectedPayout2)
		}
	})
}

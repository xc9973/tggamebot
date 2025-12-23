// Package sicbo tests for the Sic Bo game calculator.
// Requirements: 5.3, 5.4, 5.5
package sicbo

import (
	"testing"

	"pgregory.net/rapid"
)

// TestIsTriple tests the triple detection function.
func TestIsTriple(t *testing.T) {
	tests := []struct {
		name     string
		dice     [3]int
		expected bool
	}{
		{"triple 1s", [3]int{1, 1, 1}, true},
		{"triple 6s", [3]int{6, 6, 6}, true},
		{"not triple - first different", [3]int{2, 1, 1}, false},
		{"not triple - middle different", [3]int{1, 2, 1}, false},
		{"not triple - last different", [3]int{1, 1, 2}, false},
		{"all different", [3]int{1, 2, 3}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTriple(tt.dice)
			if result != tt.expected {
				t.Errorf("IsTriple(%v) = %v, want %v", tt.dice, result, tt.expected)
			}
		})
	}
}

// TestCalculateSinglePayout tests single number bet payouts.
func TestCalculateSinglePayout(t *testing.T) {
	tests := []struct {
		name       string
		betNumber  int
		dice       [3]int
		betAmount  int64
		expected   int64
	}{
		{"no match", 1, [3]int{2, 3, 4}, 100, -100},
		{"one match", 1, [3]int{1, 2, 3}, 100, 100},
		{"two matches", 1, [3]int{1, 1, 3}, 100, 200},
		{"three matches (triple)", 1, [3]int{1, 1, 1}, 100, 300},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateSinglePayout(tt.betNumber, tt.dice, tt.betAmount)
			if result != tt.expected {
				t.Errorf("CalculateSinglePayout(%d, %v, %d) = %d, want %d",
					tt.betNumber, tt.dice, tt.betAmount, result, tt.expected)
			}
		})
	}
}

// TestCalculateBigSmallPayout tests big/small bet payouts.
func TestCalculateBigSmallPayout(t *testing.T) {
	tests := []struct {
		name      string
		isBig     bool
		dice      [3]int
		betAmount int64
		expected  int64
	}{
		// Big bets (sum 11-17, not triple)
		{"big wins - sum 11", true, [3]int{3, 4, 4}, 100, 100},
		{"big wins - sum 17", true, [3]int{5, 6, 6}, 100, 100},
		{"big loses - sum 10", true, [3]int{2, 4, 4}, 100, -100},
		{"big loses - triple 6", true, [3]int{6, 6, 6}, 100, -100},

		// Small bets (sum 4-10, not triple)
		{"small wins - sum 4", false, [3]int{1, 1, 2}, 100, 100},
		{"small wins - sum 10", false, [3]int{2, 4, 4}, 100, 100},
		{"small loses - sum 11", false, [3]int{3, 4, 4}, 100, -100},
		{"small loses - triple 1", false, [3]int{1, 1, 1}, 100, -100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateBigSmallPayout(tt.isBig, tt.dice, tt.betAmount)
			if result != tt.expected {
				t.Errorf("CalculateBigSmallPayout(%v, %v, %d) = %d, want %d",
					tt.isBig, tt.dice, tt.betAmount, result, tt.expected)
			}
		})
	}
}

// TestSicBoPayoutCalculationProperty tests the SicBo payout calculation using property-based testing.
// **Feature: go-telegram-bot, Property 9: SicBo Payout Calculation**
// *For any* dice result [d1, d2, d3] where each di ∈ [1,6] and fixed bet amount 100:
// - Single number N: payout = 100 * count(N) if count > 0, else -100
// - Big bet: payout = 100 if sum ∈ [11,17] AND not triple, else -100
// - Small bet: payout = 100 if sum ∈ [4,10] AND not triple, else -100
// - Triple detection: is_triple = (d1 == d2 == d3)
// **Validates: Requirements 5.3, 5.4, 5.5**
func TestSicBoPayoutCalculationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random dice values in valid range [1,6]
		d1 := rapid.IntRange(1, 6).Draw(t, "d1")
		d2 := rapid.IntRange(1, 6).Draw(t, "d2")
		d3 := rapid.IntRange(1, 6).Draw(t, "d3")
		dice := [3]int{d1, d2, d3}

		// Fixed bet amount as per requirements
		betAmount := FixedBetAmount // 100

		// Calculate expected values
		sum := d1 + d2 + d3
		isTriple := d1 == d2 && d2 == d3

		// Test triple detection
		if IsTriple(dice) != isTriple {
			t.Fatalf("IsTriple(%v) = %v, expected %v", dice, IsTriple(dice), isTriple)
		}

		// Test single number bets for all numbers 1-6
		for betNumber := 1; betNumber <= 6; betNumber++ {
			matchCount := 0
			for _, d := range dice {
				if d == betNumber {
					matchCount++
				}
			}

			payout := CalculateSinglePayout(betNumber, dice, betAmount)

			var expectedPayout int64
			if matchCount == 0 {
				expectedPayout = -betAmount
			} else {
				expectedPayout = betAmount * int64(matchCount)
			}

			if payout != expectedPayout {
				t.Fatalf("Single bet on %d with dice %v: expected payout %d, got %d (matchCount=%d)",
					betNumber, dice, expectedPayout, payout, matchCount)
			}
		}

		// Test big bet
		bigPayout := CalculateBigSmallPayout(true, dice, betAmount)
		var expectedBigPayout int64
		if isTriple {
			expectedBigPayout = -betAmount // Triple always loses
		} else if sum >= 11 && sum <= 17 {
			expectedBigPayout = betAmount // Big wins
		} else {
			expectedBigPayout = -betAmount // Not in big range
		}

		if bigPayout != expectedBigPayout {
			t.Fatalf("Big bet with dice %v (sum=%d, triple=%v): expected payout %d, got %d",
				dice, sum, isTriple, expectedBigPayout, bigPayout)
		}

		// Test small bet
		smallPayout := CalculateBigSmallPayout(false, dice, betAmount)
		var expectedSmallPayout int64
		if isTriple {
			expectedSmallPayout = -betAmount // Triple always loses
		} else if sum >= 4 && sum <= 10 {
			expectedSmallPayout = betAmount // Small wins
		} else {
			expectedSmallPayout = -betAmount // Not in small range
		}

		if smallPayout != expectedSmallPayout {
			t.Fatalf("Small bet with dice %v (sum=%d, triple=%v): expected payout %d, got %d",
				dice, sum, isTriple, expectedSmallPayout, smallPayout)
		}
	})
}

// TestSicBoTripleDetectionProperty tests that triple detection is correct.
// **Feature: go-telegram-bot, Property 9: SicBo Payout Calculation**
// *For any* dice result [d1, d2, d3], is_triple = (d1 == d2 == d3)
// **Validates: Requirements 5.5**
func TestSicBoTripleDetectionProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d1 := rapid.IntRange(1, 6).Draw(t, "d1")
		d2 := rapid.IntRange(1, 6).Draw(t, "d2")
		d3 := rapid.IntRange(1, 6).Draw(t, "d3")
		dice := [3]int{d1, d2, d3}

		expected := d1 == d2 && d2 == d3
		actual := IsTriple(dice)

		if actual != expected {
			t.Fatalf("IsTriple(%v) = %v, expected %v", dice, actual, expected)
		}
	})
}

// TestSicBoBigSmallMutualExclusionProperty tests that big and small are mutually exclusive (except for triples).
// **Feature: go-telegram-bot, Property 9: SicBo Payout Calculation**
// *For any* non-triple dice result, exactly one of big or small should win.
// **Validates: Requirements 5.4, 5.5**
func TestSicBoBigSmallMutualExclusionProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d1 := rapid.IntRange(1, 6).Draw(t, "d1")
		d2 := rapid.IntRange(1, 6).Draw(t, "d2")
		d3 := rapid.IntRange(1, 6).Draw(t, "d3")
		dice := [3]int{d1, d2, d3}

		betAmount := FixedBetAmount
		sum := d1 + d2 + d3
		isTriple := d1 == d2 && d2 == d3

		bigPayout := CalculateBigSmallPayout(true, dice, betAmount)
		smallPayout := CalculateBigSmallPayout(false, dice, betAmount)

		bigWins := bigPayout > 0
		smallWins := smallPayout > 0

		if isTriple {
			// Both should lose on triple
			if bigWins || smallWins {
				t.Fatalf("Triple %v: big and small should both lose, but bigWins=%v, smallWins=%v",
					dice, bigWins, smallWins)
			}
		} else {
			// For non-triple, exactly one should win (sum 4-10 = small, sum 11-17 = big)
			// Note: sum 3 and sum 18 are only possible with triples
			if bigWins && smallWins {
				t.Fatalf("Non-triple %v (sum=%d): both big and small won, should be mutually exclusive",
					dice, sum)
			}
			if !bigWins && !smallWins {
				t.Fatalf("Non-triple %v (sum=%d): neither big nor small won, one should win",
					dice, sum)
			}
		}
	})
}

// TestSicBoSinglePayoutProportionalProperty tests that single number payout is proportional to match count.
// **Feature: go-telegram-bot, Property 9: SicBo Payout Calculation**
// *For any* single number bet, payout = betAmount * matchCount (or -betAmount if no match)
// **Validates: Requirements 5.3**
func TestSicBoSinglePayoutProportionalProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		betNumber := rapid.IntRange(1, 6).Draw(t, "betNumber")
		d1 := rapid.IntRange(1, 6).Draw(t, "d1")
		d2 := rapid.IntRange(1, 6).Draw(t, "d2")
		d3 := rapid.IntRange(1, 6).Draw(t, "d3")
		dice := [3]int{d1, d2, d3}

		// Test with different bet amounts to verify proportionality
		betAmount := rapid.Int64Range(1, 1000).Draw(t, "betAmount")

		matchCount := 0
		for _, d := range dice {
			if d == betNumber {
				matchCount++
			}
		}

		payout := CalculateSinglePayout(betNumber, dice, betAmount)

		var expectedPayout int64
		if matchCount == 0 {
			expectedPayout = -betAmount
		} else {
			expectedPayout = betAmount * int64(matchCount)
		}

		if payout != expectedPayout {
			t.Fatalf("Single bet %d on dice %v with bet %d: expected %d, got %d (matches=%d)",
				betNumber, dice, betAmount, expectedPayout, payout, matchCount)
		}
	})
}

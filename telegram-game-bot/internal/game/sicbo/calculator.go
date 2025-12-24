// Package sicbo implements the Sic Bo (骰宝) game for the Telegram game bot.
// Requirements: 5.3, 5.4, 5.5
package sicbo

// BetType represents the type of bet in Sic Bo.
type BetType string

const (
	// BetTypeSingle is a bet on a single number (1-6)
	BetTypeSingle BetType = "single"
	// BetTypeBig is a bet on big (sum 11-17, excluding triples)
	BetTypeBig BetType = "big"
	// BetTypeSmall is a bet on small (sum 4-10, excluding triples)
	BetTypeSmall BetType = "small"
)

const (
	// FixedBetAmount is the fixed bet amount per button click
	// Requirements: 5.3
	FixedBetAmount int64 = 1000
)

// IsTriple checks if all three dice show the same value.
// Requirements: 5.5
func IsTriple(dice [3]int) bool {
	return dice[0] == dice[1] && dice[1] == dice[2]
}

// CalculateSinglePayout calculates the payout for a single number bet.
// Rules:
//   - 0 matches: payout = -bet (lose)
//   - 1 match: payout = bet (1:1)
//   - 2 matches: payout = 2*bet (2:1)
//   - 3 matches: payout = 3*bet (3:1)
//
// Requirements: 5.3
func CalculateSinglePayout(betNumber int, dice [3]int, betAmount int64) int64 {
	matchCount := 0
	for _, d := range dice {
		if d == betNumber {
			matchCount++
		}
	}

	if matchCount == 0 {
		return -betAmount
	}
	// 1 match = 1:1, 2 matches = 2:1, 3 matches = 3:1
	return betAmount * int64(matchCount)
}

// CalculateBigSmallPayout calculates the payout for big/small bets.
// Rules:
//   - Triple: payout = -bet (house wins)
//   - Big (sum 11-17) and bet is big: payout = bet (1:1)
//   - Small (sum 4-10) and bet is small: payout = bet (1:1)
//   - Otherwise: payout = -bet (lose)
//
// Requirements: 5.4, 5.5
func CalculateBigSmallPayout(isBig bool, dice [3]int, betAmount int64) int64 {
	// Triple always loses for big/small bets
	if IsTriple(dice) {
		return -betAmount
	}

	total := dice[0] + dice[1] + dice[2]

	if isBig {
		// Big: sum 11-17
		if total >= 11 && total <= 17 {
			return betAmount
		}
	} else {
		// Small: sum 4-10
		if total >= 4 && total <= 10 {
			return betAmount
		}
	}

	return -betAmount
}

// CalculatePayout calculates the payout for any bet type.
// This is the unified entry point for payout calculation.
// Requirements: 5.3, 5.4, 5.5
func CalculatePayout(betType BetType, betNumber int, dice [3]int, betAmount int64) int64 {
	switch betType {
	case BetTypeSingle:
		return CalculateSinglePayout(betNumber, dice, betAmount)
	case BetTypeBig:
		return CalculateBigSmallPayout(true, dice, betAmount)
	case BetTypeSmall:
		return CalculateBigSmallPayout(false, dice, betAmount)
	default:
		return -betAmount
	}
}

// ValidateBetType checks if the bet type and parameters are valid.
func ValidateBetType(betType BetType, betNumber int) bool {
	switch betType {
	case BetTypeSingle:
		return betNumber >= 1 && betNumber <= 6
	case BetTypeBig, BetTypeSmall:
		return true
	default:
		return false
	}
}

// ValidateDice checks if all dice values are valid (1-6).
func ValidateDice(dice [3]int) bool {
	for _, d := range dice {
		if d < 1 || d > 6 {
			return false
		}
	}
	return true
}

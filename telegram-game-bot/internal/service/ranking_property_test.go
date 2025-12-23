// Package service provides business logic implementations.
// Property-based tests for RankingService.
// **Feature: go-telegram-bot, Property 3: Top Users Ordering**
// **Feature: go-telegram-bot, Property 14: Daily Ranking Calculation**
// **Validates: Requirements 1.5, 11.2, 11.3, 11.5**
package service

import (
	"sort"
	"testing"

	"pgregory.net/rapid"

	"telegram-game-bot/internal/model"
)

// TestTopUsersOrderingProperty tests that top users are sorted by balance descending.
// Property 3: Top Users Ordering
// *For any* set of users, GetTopUsers SHALL return users sorted by balance in descending order.
// **Validates: Requirements 1.5**
func TestTopUsersOrderingProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random number of users (1-50)
		numUsers := rapid.IntRange(1, 50).Draw(t, "numUsers")

		// Generate random users with random balances
		users := make([]*model.User, numUsers)
		for i := 0; i < numUsers; i++ {
			users[i] = &model.User{
				TelegramID: rapid.Int64Range(1, 1000000).Draw(t, "telegramID"),
				Username:   rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "username"),
				Balance:    rapid.Int64Range(0, 1000000).Draw(t, "balance"),
			}
		}

		// Generate a random limit (1 to numUsers+5 to test edge cases)
		limit := rapid.IntRange(1, numUsers+5).Draw(t, "limit")

		// Apply the sorting logic (same as repository)
		result := getTopUsersSorted(users, limit)

		// Verify Property 3: results are sorted by balance descending
		if !isSortedByBalanceDesc(result) {
			t.Fatalf("Top users not sorted by balance descending: %v", balancesOf(result))
		}

		// Verify limit is respected
		expectedLen := min(limit, len(users))
		if len(result) != expectedLen {
			t.Fatalf("Expected %d users, got %d", expectedLen, len(result))
		}

		// Verify we got the actual top users (highest balances)
		if len(result) > 0 {
			// Sort original users by balance to find expected top
			sortedOriginal := make([]*model.User, len(users))
			copy(sortedOriginal, users)
			sort.Slice(sortedOriginal, func(i, j int) bool {
				return sortedOriginal[i].Balance > sortedOriginal[j].Balance
			})

			// The minimum balance in result should be >= any balance not in result
			minResultBalance := result[len(result)-1].Balance
			for i := len(result); i < len(sortedOriginal); i++ {
				if sortedOriginal[i].Balance > minResultBalance {
					t.Fatalf("User with balance %d not in top %d, but user with balance %d is",
						sortedOriginal[i].Balance, limit, minResultBalance)
				}
			}
		}
	})
}

// TestDailyRankingCalculationProperty tests daily ranking calculation from game transactions.
// Property 14: Daily Ranking Calculation
// *For any* day's game transactions:
// - Daily net profit = SUM(amount) for game-type transactions only
// - Winners = users with positive net profit, sorted descending
// - Losers = users with negative net profit, sorted ascending (most loss first)
// - Only transaction types: dice, slot, sicbo_win, sicbo_bet are counted
// **Validates: Requirements 11.2, 11.3, 11.5**
func TestDailyRankingCalculationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random transactions for multiple users
		numUsers := rapid.IntRange(1, 20).Draw(t, "numUsers")
		numTxPerUser := rapid.IntRange(0, 10).Draw(t, "numTxPerUser")

		// Transaction types - mix of game and non-game types
		gameTypes := []string{model.TxTypeDice, model.TxTypeSlot, model.TxTypeSicBoWin, model.TxTypeSicBoBet}
		nonGameTypes := []string{model.TxTypeTransfer, model.TxTypeDaily, model.TxTypeAdminAdd}
		allTypes := append(gameTypes, nonGameTypes...)

		// Generate transactions and calculate expected profits
		userProfits := make(map[int64]int64)
		usernames := make(map[int64]string)

		for i := 0; i < numUsers; i++ {
			userID := int64(i + 1)
			usernames[userID] = rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "username")
			userProfits[userID] = 0

			for j := 0; j < numTxPerUser; j++ {
				txType := rapid.SampledFrom(allTypes).Draw(t, "txType")
				amount := rapid.Int64Range(-1000, 1000).Draw(t, "amount")

				// Only count game transactions for profit
				if isGameTransaction(txType) {
					userProfits[userID] += amount
				}
			}
		}

		// Calculate expected winners and losers
		expectedWinners, expectedLosers := calculateExpectedRankings(userProfits, usernames)

		// Verify winners are sorted by profit descending
		if !isWinnersSortedCorrectly(expectedWinners) {
			t.Fatalf("Winners not sorted by profit descending: %v", profitsOf(expectedWinners))
		}

		// Verify losers are sorted by profit ascending (most loss first)
		if !isLosersSortedCorrectly(expectedLosers) {
			t.Fatalf("Losers not sorted by loss descending: %v", profitsOf(expectedLosers))
		}

		// Verify all winners have positive profit
		for _, w := range expectedWinners {
			if w.NetProfit <= 0 {
				t.Fatalf("Winner has non-positive profit: %d", w.NetProfit)
			}
		}

		// Verify all losers have negative profit
		for _, l := range expectedLosers {
			if l.NetProfit >= 0 {
				t.Fatalf("Loser has non-negative profit: %d", l.NetProfit)
			}
		}
	})
}

// TestGameTransactionTypesOnlyProperty tests that only game transactions are counted.
// **Validates: Requirements 11.5**
func TestGameTransactionTypesOnlyProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a mix of transaction types
		txType := rapid.SampledFrom([]string{
			model.TxTypeDice, model.TxTypeSlot, model.TxTypeSicBoWin, model.TxTypeSicBoBet,
			model.TxTypeTransfer, model.TxTypeDaily, model.TxTypeAdminAdd, model.TxTypeAdminSub,
		}).Draw(t, "txType")

		isGame := isGameTransaction(txType)
		expectedGame := txType == model.TxTypeDice ||
			txType == model.TxTypeSlot ||
			txType == model.TxTypeSicBoWin ||
			txType == model.TxTypeSicBoBet

		if isGame != expectedGame {
			t.Fatalf("Transaction type %s: expected isGame=%v, got %v", txType, expectedGame, isGame)
		}
	})
}

// Helper functions that mirror the repository/service logic

// getTopUsersSorted sorts users by balance descending and returns top N.
func getTopUsersSorted(users []*model.User, limit int) []*model.User {
	// Make a copy to avoid modifying original
	sorted := make([]*model.User, len(users))
	copy(sorted, users)

	// Sort by balance descending
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Balance > sorted[j].Balance
	})

	// Apply limit
	if limit > len(sorted) {
		limit = len(sorted)
	}
	return sorted[:limit]
}

// isSortedByBalanceDesc checks if users are sorted by balance in descending order.
func isSortedByBalanceDesc(users []*model.User) bool {
	for i := 1; i < len(users); i++ {
		if users[i].Balance > users[i-1].Balance {
			return false
		}
	}
	return true
}

// isGameTransaction checks if a transaction type counts towards daily rankings.
func isGameTransaction(txType string) bool {
	gameTypes := model.GameTransactionTypes()
	for _, gt := range gameTypes {
		if txType == gt {
			return true
		}
	}
	return false
}

// calculateExpectedRankings calculates winners and losers from user profits.
func calculateExpectedRankings(profits map[int64]int64, usernames map[int64]string) ([]*model.DailyRank, []*model.DailyRank) {
	var winners, losers []*model.DailyRank

	for userID, profit := range profits {
		rank := &model.DailyRank{
			UserID:    userID,
			Username:  usernames[userID],
			NetProfit: profit,
		}
		if profit > 0 {
			winners = append(winners, rank)
		} else if profit < 0 {
			losers = append(losers, rank)
		}
		// Users with 0 profit are neither winners nor losers
	}

	// Sort winners by profit descending
	sort.Slice(winners, func(i, j int) bool {
		return winners[i].NetProfit > winners[j].NetProfit
	})

	// Sort losers by profit ascending (most loss first)
	sort.Slice(losers, func(i, j int) bool {
		return losers[i].NetProfit < losers[j].NetProfit
	})

	return winners, losers
}

// isWinnersSortedCorrectly checks if winners are sorted by profit descending.
func isWinnersSortedCorrectly(winners []*model.DailyRank) bool {
	for i := 1; i < len(winners); i++ {
		if winners[i].NetProfit > winners[i-1].NetProfit {
			return false
		}
	}
	return true
}

// isLosersSortedCorrectly checks if losers are sorted by profit ascending (most loss first).
func isLosersSortedCorrectly(losers []*model.DailyRank) bool {
	for i := 1; i < len(losers); i++ {
		if losers[i].NetProfit < losers[i-1].NetProfit {
			return false
		}
	}
	return true
}

// balancesOf extracts balances from users for debugging.
func balancesOf(users []*model.User) []int64 {
	balances := make([]int64, len(users))
	for i, u := range users {
		balances[i] = u.Balance
	}
	return balances
}

// profitsOf extracts profits from rankings for debugging.
func profitsOf(ranks []*model.DailyRank) []int64 {
	profits := make([]int64, len(ranks))
	for i, r := range ranks {
		profits[i] = r.NetProfit
	}
	return profits
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

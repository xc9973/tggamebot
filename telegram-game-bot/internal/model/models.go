// Package model defines the data models for the Telegram game bot.
package model

import "time"

// User represents a Telegram user account in the game system.
// Requirements: 8.1 - users table with telegram_id, username, balance, last_daily_claim, created_at, updated_at
type User struct {
	TelegramID     int64     `db:"telegram_id"`
	Username       string    `db:"username"`
	Balance        int64     `db:"balance"`
	LastDailyClaim int64     `db:"last_daily_claim"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// Transaction represents a balance change record.
// Requirements: 8.2 - transactions table with id, user_id, amount, type, description, created_at
type Transaction struct {
	ID          int64     `db:"id"`
	UserID      int64     `db:"user_id"`
	Amount      int64     `db:"amount"`
	Type        string    `db:"type"`
	Description *string   `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
}

// DailyRank represents a user's daily game performance for ranking.
// Used by the daily_game_stats view for winner/loser rankings.
type DailyRank struct {
	UserID    int64  `db:"user_id"`
	Username  string `db:"username"`
	NetProfit int64  `db:"net_profit"`
}

// Transaction types for categorizing balance changes.
const (
	TxTypeInitial      = "initial"       // Initial balance on account creation
	TxTypeDaily        = "daily"         // Daily reward claim
	TxTypeTransfer     = "transfer"      // User-to-user transfer
	TxTypeDice         = "dice"          // Dice game result
	TxTypeSlot         = "slot"          // Slot machine result
	TxTypeSicBoBet     = "sicbo_bet"     // SicBo bet placement
	TxTypeSicBoWin     = "sicbo_win"     // SicBo winnings
	TxTypeAdminAdd     = "admin_add"     // Admin added balance
	TxTypeAdminSub     = "admin_sub"     // Admin subtracted balance
	TxTypeAdminSet     = "admin_set"     // Admin set balance
	TxTypeRob          = "rob"           // Robbery - robber gains coins
	TxTypeRobbed       = "robbed"        // Robbery - victim loses coins
	TxTypeShopPurchase = "shop_purchase" // Shop item purchase
)

// GameTransactionTypes returns the transaction types that count towards daily game rankings.
// Requirements: 11.5 - Only count game-related transactions (exclude transfers, daily rewards)
func GameTransactionTypes() []string {
	return []string{TxTypeDice, TxTypeSlot, TxTypeSicBoWin, TxTypeSicBoBet, TxTypeRob, TxTypeRobbed}
}

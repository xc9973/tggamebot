// Package game defines the game interfaces and registry for the Telegram game bot.
// Requirements: 10.1 - Define a common Game interface for all games
// Requirements: 10.3 - Adding a new game only requires implementing the Game interface
package game

import "context"

// GameResult represents the outcome of a game play.
type GameResult struct {
	Payout      int64             // Net payout (positive = win, negative = loss, 0 = push)
	Description string            // Human-readable result description
	Details     map[string]any    // Additional game-specific details
}

// Game defines the interface that all games must implement.
// This enables a plugin-style architecture where new games can be added
// by simply implementing this interface.
// Requirements: 10.1, 10.3, 10.4
type Game interface {
	// Name returns the game's display name (e.g., "Dice Game", "Slot Machine")
	Name() string

	// Command returns the command that triggers this game (e.g., "dice", "slot")
	Command() string

	// Description returns a brief description of the game
	Description() string

	// Play executes the game logic and returns the result.
	// Parameters:
	//   - ctx: context for cancellation and timeouts
	//   - userID: the Telegram user ID playing the game
	//   - bet: the amount being wagered
	//   - params: additional game-specific parameters
	// Returns:
	//   - GameResult containing payout and details
	//   - error if the game cannot be played
	Play(ctx context.Context, userID int64, bet int64, params map[string]any) (*GameResult, error)

	// ValidateBet checks if the bet amount and parameters are valid.
	// Returns nil if valid, or an error describing the validation failure.
	ValidateBet(bet int64, params map[string]any) error

	// MaxBet returns the maximum allowed bet for this game.
	// Returns 0 if there is no maximum.
	MaxBet() int64

	// Cooldown returns the cooldown duration in seconds between plays.
	// Returns 0 if there is no cooldown.
	Cooldown() int
}

// MultiPlayerGame extends Game for games that support multiple players
// in a single session (e.g., SicBo).
// Requirements: 10.1
type MultiPlayerGame interface {
	Game

	// StartSession begins a new multiplayer game session in a chat.
	// Parameters:
	//   - ctx: context for cancellation and timeouts
	//   - chatID: the Telegram chat ID where the session is started
	//   - duration: betting phase duration in seconds
	// Returns:
	//   - error if session cannot be started
	StartSession(ctx context.Context, chatID int64, duration int) error

	// PlaceBet places a bet for a user in an active session.
	// Parameters:
	//   - ctx: context for cancellation and timeouts
	//   - chatID: the Telegram chat ID of the session
	//   - userID: the Telegram user ID placing the bet
	//   - betType: the type of bet (e.g., "big", "small", "1", "2", etc.)
	//   - amount: the bet amount
	// Returns:
	//   - error if bet cannot be placed
	PlaceBet(ctx context.Context, chatID, userID int64, betType string, amount int64) error

	// GetSessionBets returns all bets placed in the current session.
	// Parameters:
	//   - ctx: context for cancellation and timeouts
	//   - chatID: the Telegram chat ID of the session
	// Returns:
	//   - map of userID to their bets (betType -> amount)
	//   - error if session not found
	GetSessionBets(ctx context.Context, chatID int64) (map[int64]map[string]int64, error)

	// Settle ends the session and calculates results for all participants.
	// Parameters:
	//   - ctx: context for cancellation and timeouts
	//   - chatID: the Telegram chat ID of the session
	// Returns:
	//   - map of userID to their net payout
	//   - game result details (e.g., dice values)
	//   - error if settlement fails
	Settle(ctx context.Context, chatID int64) (map[int64]int64, map[string]any, error)

	// IsSessionActive checks if there's an active session in the chat.
	IsSessionActive(chatID int64) bool

	// GetSessionTimeRemaining returns seconds remaining in the betting phase.
	// Returns 0 if no active session or betting phase ended.
	GetSessionTimeRemaining(chatID int64) int
}

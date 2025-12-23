// Package sicbo implements the Sic Bo (骰宝) multiplayer game.
// Requirements: 5.1, 5.2, 5.7, 5.8
package sicbo

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"telegram-game-bot/internal/game"
)

const (
	// DefaultBettingDuration is the default betting phase duration in seconds
	// Requirements: 5.1
	DefaultBettingDuration = 60
)

// Errors for SicBo game
var (
	ErrNoActiveSession    = errors.New("no active session in this chat")
	ErrSessionExists      = errors.New("session already exists in this chat")
	ErrBettingEnded       = errors.New("betting phase has ended")
	ErrInvalidBetType     = errors.New("invalid bet type")
	ErrInvalidBetNumber   = errors.New("bet number must be between 1 and 6")
	ErrInsufficientAmount = errors.New("bet amount must be positive")
)

// Bet represents a single bet placed by a user.
type Bet struct {
	UserID    int64
	BetType   BetType
	BetNumber int   // Only used for single number bets
	Amount    int64 // Accumulated amount for this bet option
}

// Session represents an active SicBo game session.
type Session struct {
	ChatID         int64
	StartTime      time.Time
	BettingEndTime time.Time
	Bets           map[int64]map[string]*Bet // userID -> betKey -> Bet
	DiceResults    [3]int
	Settled        bool
	mu             sync.RWMutex
}

// betKey generates a unique key for a bet option.
func betKey(betType BetType, betNumber int) string {
	if betType == BetTypeSingle {
		return fmt.Sprintf("%s_%d", betType, betNumber)
	}
	return string(betType)
}

// SicBoGame implements the MultiPlayerGame interface for Sic Bo.
// Requirements: 5.1, 5.2, 5.7, 5.8, 10.1
type SicBoGame struct {
	sessions map[int64]*Session // chatID -> Session
	mu       sync.RWMutex
}

// New creates a new SicBoGame instance.
func New() *SicBoGame {
	return &SicBoGame{
		sessions: make(map[int64]*Session),
	}
}

// Name returns the game's display name.
func (g *SicBoGame) Name() string {
	return "Sic Bo"
}

// Command returns the command that triggers this game.
func (g *SicBoGame) Command() string {
	return "sicbo"
}

// Description returns a brief description of the game.
func (g *SicBoGame) Description() string {
	return "Multiplayer dice game! Bet on numbers (1-6), big, or small. Fixed 100 coins per bet."
}

// MaxBet returns the maximum allowed bet (fixed at 100 per click).
func (g *SicBoGame) MaxBet() int64 {
	return FixedBetAmount
}

// Cooldown returns 0 as SicBo is session-based.
func (g *SicBoGame) Cooldown() int {
	return 0
}

// ValidateBet validates the bet parameters.
func (g *SicBoGame) ValidateBet(bet int64, params map[string]any) error {
	if bet <= 0 {
		return ErrInsufficientAmount
	}
	return nil
}


// Play is not used for multiplayer games - use PlaceBet instead.
func (g *SicBoGame) Play(ctx context.Context, userID int64, bet int64, params map[string]any) (*game.GameResult, error) {
	return nil, errors.New("use PlaceBet for multiplayer games")
}

// StartSession begins a new multiplayer game session in a chat.
// Requirements: 5.1
func (g *SicBoGame) StartSession(ctx context.Context, chatID int64, duration int) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Check if session already exists
	if session, exists := g.sessions[chatID]; exists && !session.Settled {
		return ErrSessionExists
	}

	if duration <= 0 {
		duration = DefaultBettingDuration
	}

	now := time.Now()
	g.sessions[chatID] = &Session{
		ChatID:         chatID,
		StartTime:      now,
		BettingEndTime: now.Add(time.Duration(duration) * time.Second),
		Bets:           make(map[int64]map[string]*Bet),
		Settled:        false,
	}

	return nil
}

// PlaceBet places a bet for a user in an active session.
// Supports accumulating bets on the same option (Requirements: 5.8).
// Requirements: 5.2, 5.7, 5.8
func (g *SicBoGame) PlaceBet(ctx context.Context, chatID, userID int64, betTypeStr string, amount int64) error {
	g.mu.RLock()
	session, exists := g.sessions[chatID]
	g.mu.RUnlock()

	if !exists || session.Settled {
		return ErrNoActiveSession
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Check if betting phase has ended
	if time.Now().After(session.BettingEndTime) {
		return ErrBettingEnded
	}

	// Parse bet type
	betType, betNumber, err := parseBetType(betTypeStr)
	if err != nil {
		return err
	}

	// Validate bet
	if !ValidateBetType(betType, betNumber) {
		return ErrInvalidBetType
	}

	if amount <= 0 {
		return ErrInsufficientAmount
	}

	// Initialize user's bet map if needed
	if session.Bets[userID] == nil {
		session.Bets[userID] = make(map[string]*Bet)
	}

	// Get or create bet for this option
	key := betKey(betType, betNumber)
	if existingBet, ok := session.Bets[userID][key]; ok {
		// Accumulate bet amount (Requirements: 5.8)
		existingBet.Amount += amount
	} else {
		// Create new bet
		session.Bets[userID][key] = &Bet{
			UserID:    userID,
			BetType:   betType,
			BetNumber: betNumber,
			Amount:    amount,
		}
	}

	return nil
}

// parseBetType parses a bet type string into BetType and bet number.
// Format: "single_N" for single number, "big", "small" for big/small.
func parseBetType(betTypeStr string) (BetType, int, error) {
	switch betTypeStr {
	case "big":
		return BetTypeBig, 0, nil
	case "small":
		return BetTypeSmall, 0, nil
	case "1", "2", "3", "4", "5", "6":
		var num int
		fmt.Sscanf(betTypeStr, "%d", &num)
		return BetTypeSingle, num, nil
	default:
		// Try parsing as "single_N" format
		var num int
		if _, err := fmt.Sscanf(betTypeStr, "single_%d", &num); err == nil {
			if num >= 1 && num <= 6 {
				return BetTypeSingle, num, nil
			}
		}
		return "", 0, ErrInvalidBetType
	}
}

// GetSessionBets returns all bets placed in the current session.
func (g *SicBoGame) GetSessionBets(ctx context.Context, chatID int64) (map[int64]map[string]int64, error) {
	g.mu.RLock()
	session, exists := g.sessions[chatID]
	g.mu.RUnlock()

	if !exists {
		return nil, ErrNoActiveSession
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	result := make(map[int64]map[string]int64)
	for userID, bets := range session.Bets {
		result[userID] = make(map[string]int64)
		for key, bet := range bets {
			result[userID][key] = bet.Amount
		}
	}

	return result, nil
}


// Settle ends the session and calculates results for all participants.
// Requirements: 5.7
func (g *SicBoGame) Settle(ctx context.Context, chatID int64) (map[int64]int64, map[string]any, error) {
	g.mu.Lock()
	session, exists := g.sessions[chatID]
	if !exists || session.Settled {
		g.mu.Unlock()
		return nil, nil, ErrNoActiveSession
	}
	g.mu.Unlock()

	session.mu.Lock()
	defer session.mu.Unlock()

	// Generate dice results
	session.DiceResults = rollDice()
	session.Settled = true

	// Calculate payouts for each user
	payouts := make(map[int64]int64)
	for userID, bets := range session.Bets {
		var totalPayout int64
		for _, bet := range bets {
			payout := CalculatePayout(bet.BetType, bet.BetNumber, session.DiceResults, bet.Amount)
			totalPayout += payout
		}
		payouts[userID] = totalPayout
	}

	// Build details
	details := map[string]any{
		"dice":      session.DiceResults,
		"total":     session.DiceResults[0] + session.DiceResults[1] + session.DiceResults[2],
		"is_triple": IsTriple(session.DiceResults),
	}

	// Clean up session
	g.mu.Lock()
	delete(g.sessions, chatID)
	g.mu.Unlock()

	return payouts, details, nil
}

// SettleWithDice settles the game with specific dice values (for testing).
func (g *SicBoGame) SettleWithDice(ctx context.Context, chatID int64, dice [3]int) (map[int64]int64, map[string]any, error) {
	g.mu.Lock()
	session, exists := g.sessions[chatID]
	if !exists || session.Settled {
		g.mu.Unlock()
		return nil, nil, ErrNoActiveSession
	}
	g.mu.Unlock()

	session.mu.Lock()
	defer session.mu.Unlock()

	// Use provided dice results
	session.DiceResults = dice
	session.Settled = true

	// Calculate payouts for each user
	payouts := make(map[int64]int64)
	for userID, bets := range session.Bets {
		var totalPayout int64
		for _, bet := range bets {
			payout := CalculatePayout(bet.BetType, bet.BetNumber, session.DiceResults, bet.Amount)
			totalPayout += payout
		}
		payouts[userID] = totalPayout
	}

	// Build details
	details := map[string]any{
		"dice":      session.DiceResults,
		"total":     session.DiceResults[0] + session.DiceResults[1] + session.DiceResults[2],
		"is_triple": IsTriple(session.DiceResults),
	}

	// Clean up session
	g.mu.Lock()
	delete(g.sessions, chatID)
	g.mu.Unlock()

	return payouts, details, nil
}

// IsSessionActive checks if there's an active session in the chat.
func (g *SicBoGame) IsSessionActive(chatID int64) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	session, exists := g.sessions[chatID]
	return exists && !session.Settled
}

// GetSessionTimeRemaining returns seconds remaining in the betting phase.
func (g *SicBoGame) GetSessionTimeRemaining(chatID int64) int {
	g.mu.RLock()
	session, exists := g.sessions[chatID]
	g.mu.RUnlock()

	if !exists || session.Settled {
		return 0
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	remaining := time.Until(session.BettingEndTime)
	if remaining < 0 {
		return 0
	}
	return int(remaining.Seconds())
}

// GetSessionStats returns statistics about the current session.
func (g *SicBoGame) GetSessionStats(chatID int64) (playerCount int, totalBetAmount int64, betCount int) {
	g.mu.RLock()
	session, exists := g.sessions[chatID]
	g.mu.RUnlock()

	if !exists {
		return 0, 0, 0
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	playerCount = len(session.Bets)
	for _, bets := range session.Bets {
		for _, bet := range bets {
			totalBetAmount += bet.Amount
			betCount++
		}
	}

	return playerCount, totalBetAmount, betCount
}

// rollDice generates three random dice values.
func rollDice() [3]int {
	return [3]int{
		rand.Intn(6) + 1,
		rand.Intn(6) + 1,
		rand.Intn(6) + 1,
	}
}

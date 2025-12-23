// Package dice implements the dice game for the Telegram game bot.
// Requirements: 3.2, 3.3, 3.5
package dice

import (
	"context"
	"errors"
	"fmt"

	"telegram-game-bot/internal/game"
)

const (
	// DefaultMaxBet is the maximum allowed bet for dice game
	// Requirements: 3.3
	DefaultMaxBet = 1000

	// DefaultCooldown is the cooldown between dice games in seconds
	// Requirements: 3.4
	DefaultCooldown = 3
)

// Errors for dice game
var (
	ErrInvalidBet      = errors.New("bet amount must be positive")
	ErrBetTooHigh      = errors.New("bet exceeds maximum allowed")
	ErrInvalidDice     = errors.New("dice values must be between 1 and 6")
	ErrMissingDice     = errors.New("dice values are required")
)

// DiceGame implements the Game interface for dice gambling.
// Requirements: 3.2, 3.3, 3.5, 10.1
type DiceGame struct {
	maxBet   int64
	cooldown int
}

// Config holds configuration for the dice game.
type Config struct {
	MaxBet   int64
	Cooldown int
}

// New creates a new DiceGame with the given configuration.
func New(cfg *Config) *DiceGame {
	maxBet := int64(DefaultMaxBet)
	cooldown := DefaultCooldown

	if cfg != nil {
		if cfg.MaxBet > 0 {
			maxBet = cfg.MaxBet
		}
		if cfg.Cooldown > 0 {
			cooldown = cfg.Cooldown
		}
	}

	return &DiceGame{
		maxBet:   maxBet,
		cooldown: cooldown,
	}
}

// Name returns the game's display name.
func (d *DiceGame) Name() string {
	return "Dice Game"
}

// Command returns the command that triggers this game.
func (d *DiceGame) Command() string {
	return "dice"
}

// Description returns a brief description of the game.
func (d *DiceGame) Description() string {
	return "Roll two dice and win based on the total: 2-6 lose, 7 push, 8-11 win, 12 jackpot!"
}

// MaxBet returns the maximum allowed bet.
// Requirements: 3.3
func (d *DiceGame) MaxBet() int64 {
	return d.maxBet
}

// Cooldown returns the cooldown duration in seconds.
// Requirements: 3.4
func (d *DiceGame) Cooldown() int {
	return d.cooldown
}

// ValidateBet checks if the bet amount and parameters are valid.
// Requirements: 3.3
func (d *DiceGame) ValidateBet(bet int64, params map[string]any) error {
	if bet <= 0 {
		return ErrInvalidBet
	}
	if bet > d.maxBet {
		return fmt.Errorf("%w: max bet is %d", ErrBetTooHigh, d.maxBet)
	}
	return nil
}

// Play executes the dice game logic.
// Requirements: 3.2, 3.5
func (d *DiceGame) Play(ctx context.Context, userID int64, bet int64, params map[string]any) (*game.GameResult, error) {
	// Validate bet
	if err := d.ValidateBet(bet, params); err != nil {
		return nil, err
	}

	// Extract dice values from params
	dice1, dice2, err := extractDiceValues(params)
	if err != nil {
		return nil, err
	}

	// Calculate payout
	payout := CalculatePayout(dice1, dice2, bet)
	total := dice1 + dice2

	// Build result description
	var description string
	switch {
	case payout > bet:
		description = fmt.Sprintf("🎲🎲 Dice: %d + %d = %d\n🎊 JACKPOT! You won %d coins!", dice1, dice2, total, payout)
	case payout > 0:
		description = fmt.Sprintf("🎲🎲 Dice: %d + %d = %d\n🎉 You won %d coins!", dice1, dice2, total, payout)
	case payout == 0:
		description = fmt.Sprintf("🎲🎲 Dice: %d + %d = %d\n😐 Push! Your bet is returned.", dice1, dice2, total)
	default:
		description = fmt.Sprintf("🎲🎲 Dice: %d + %d = %d\n😢 You lost %d coins.", dice1, dice2, total, -payout)
	}

	return &game.GameResult{
		Payout:      payout,
		Description: description,
		Details: map[string]any{
			"dice1": dice1,
			"dice2": dice2,
			"total": total,
			"bet":   bet,
		},
	}, nil
}

// CalculatePayout calculates the payout for a dice game.
// Rules (Property 6):
//   - total ∈ [2,6]: payout = -bet (lose)
//   - total = 7: payout = 0 (push)
//   - total ∈ [8,11]: payout = bet (win)
//   - total = 12: payout = 2*bet (jackpot)
//
// Requirements: 3.2
func CalculatePayout(dice1, dice2 int, bet int64) int64 {
	total := dice1 + dice2

	switch {
	case total <= 6:
		return -bet
	case total == 7:
		return 0
	case total <= 11:
		return bet
	default: // total == 12
		return bet * 2
	}
}

// extractDiceValues extracts dice values from params.
func extractDiceValues(params map[string]any) (int, int, error) {
	if params == nil {
		return 0, 0, ErrMissingDice
	}

	dice1, ok1 := extractInt(params, "dice1")
	dice2, ok2 := extractInt(params, "dice2")

	if !ok1 || !ok2 {
		return 0, 0, ErrMissingDice
	}

	if dice1 < 1 || dice1 > 6 || dice2 < 1 || dice2 > 6 {
		return 0, 0, ErrInvalidDice
	}

	return dice1, dice2, nil
}

// extractInt extracts an integer from params map.
func extractInt(params map[string]any, key string) (int, bool) {
	v, ok := params[key]
	if !ok {
		return 0, false
	}

	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	default:
		return 0, false
	}
}

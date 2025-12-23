// Package slot implements the slot machine game for the Telegram game bot.
// Requirements: 4.2, 4.4
package slot

import (
	"context"
	"errors"
	"fmt"

	"telegram-game-bot/internal/game"
)

const (
	// DefaultMaxBet is the maximum allowed bet for slot game
	DefaultMaxBet = 100000

	// DefaultCooldown is the cooldown between slot games in seconds
	// Requirements: 4.3
	DefaultCooldown = 5
)

// Symbol constants for display
const (
	SymbolBAR    = 1
	SymbolGrape  = 2
	SymbolLemon  = 3
	SymbolSeven  = 4
)

// Symbol names for display
var SymbolNames = map[int]string{
	SymbolBAR:   "BAR",
	SymbolGrape: "🍇",
	SymbolLemon: "🍋",
	SymbolSeven: "7️⃣",
}

// Errors for slot game
var (
	ErrInvalidBet       = errors.New("bet amount must be positive")
	ErrBetTooHigh       = errors.New("bet exceeds maximum allowed")
	ErrInvalidSlotValue = errors.New("slot value must be between 1 and 64")
	ErrMissingSlotValue = errors.New("slot value is required")
)

// SlotGame implements the Game interface for slot machine gambling.
// Requirements: 4.2, 4.4, 10.1
type SlotGame struct {
	maxBet   int64
	cooldown int
}

// Config holds configuration for the slot game.
type Config struct {
	MaxBet   int64
	Cooldown int
}

// New creates a new SlotGame with the given configuration.
func New(cfg *Config) *SlotGame {
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

	return &SlotGame{
		maxBet:   maxBet,
		cooldown: cooldown,
	}
}

// Name returns the game's display name.
func (s *SlotGame) Name() string {
	return "Slot Machine"
}

// Command returns the command that triggers this game.
func (s *SlotGame) Command() string {
	return "slot"
}

// Description returns a brief description of the game.
func (s *SlotGame) Description() string {
	return "Spin the slot machine! 3 matches = big win, 2 matches = push, no match = lose"
}

// MaxBet returns the maximum allowed bet.
func (s *SlotGame) MaxBet() int64 {
	return s.maxBet
}

// Cooldown returns the cooldown duration in seconds.
// Requirements: 4.3
func (s *SlotGame) Cooldown() int {
	return s.cooldown
}

// ValidateBet checks if the bet amount and parameters are valid.
func (s *SlotGame) ValidateBet(bet int64, params map[string]any) error {
	if bet <= 0 {
		return ErrInvalidBet
	}
	if bet > s.maxBet {
		return fmt.Errorf("%w: max bet is %d", ErrBetTooHigh, s.maxBet)
	}
	return nil
}

// Play executes the slot game logic.
// Requirements: 4.2, 4.4
func (s *SlotGame) Play(ctx context.Context, userID int64, bet int64, params map[string]any) (*game.GameResult, error) {
	// Validate bet
	if err := s.ValidateBet(bet, params); err != nil {
		return nil, err
	}

	// Extract slot value from params
	slotValue, err := extractSlotValue(params)
	if err != nil {
		return nil, err
	}

	// Decode slot value to symbols
	left, middle, right := DecodeSlot(slotValue)

	// Calculate payout
	payout := CalculatePayout(left, middle, right, bet)

	// Build result description
	slotDisplay := fmt.Sprintf("%s %s %s", SymbolNames[left], SymbolNames[middle], SymbolNames[right])
	var description string
	switch {
	case payout > 0:
		description = fmt.Sprintf("🎰 %s\n🎊 JACKPOT! Three matching symbols! You won %d coins!", slotDisplay, payout)
	case payout == 0:
		description = fmt.Sprintf("🎰 %s\n😐 Two matching symbols. Push! Your bet is returned.", slotDisplay)
	default:
		description = fmt.Sprintf("🎰 %s\n😢 No match. You lost %d coins.", slotDisplay, -payout)
	}

	return &game.GameResult{
		Payout:      payout,
		Description: description,
		Details: map[string]any{
			"slot_value": slotValue,
			"left":       left,
			"middle":     middle,
			"right":      right,
			"bet":        bet,
		},
	}, nil
}

// DecodeSlot decodes a slot value (1-64) into three symbols (1-4 each).
// Formula: value = left + (middle-1)*4 + (right-1)*16
// Property 7: DecodeSlot(V) produces (left, middle, right) where each ∈ [1,4]
// Requirements: 4.4
func DecodeSlot(slotValue int) (left, middle, right int) {
	value := slotValue - 1 // Convert to 0-63
	left = (value % 4) + 1
	middle = ((value / 4) % 4) + 1
	right = (value / 16) + 1
	return left, middle, right
}

// EncodeSlot encodes three symbols (1-4 each) into a slot value (1-64).
// This is the inverse of DecodeSlot for round-trip testing.
// Property 7: EncodeSlot(left, middle, right) = V (round-trip)
// Requirements: 4.4
func EncodeSlot(left, middle, right int) int {
	return left + (middle-1)*4 + (right-1)*16
}

// CalculatePayout calculates the payout for a slot game.
// Rules (Property 8):
//   - If left == middle == right: tiered payout based on bet amount
//   - If exactly 2 symbols match: payout = 0 (push)
//   - If no symbols match: payout = -bet (lose)
//
// Tiered multipliers for 3 matches:
//   - bet <= 1000: 3x
//   - bet 1001-10000: 2x
//   - bet 10001-100000: 1.5x
//   - bet > 100000: 1x
//
// Requirements: 4.2
func CalculatePayout(left, middle, right int, bet int64) int64 {
	// Three matching symbols - jackpot with tiered multiplier
	if left == middle && middle == right {
		var multiplier float64
		switch {
		case bet <= 1000:
			multiplier = 3.0
		case bet <= 10000:
			multiplier = 2.0
		case bet <= 100000:
			multiplier = 1.5
		default:
			multiplier = 1.0
		}
		return int64(float64(bet) * multiplier)
	}

	// Two matching symbols - push
	if left == middle || middle == right || left == right {
		return 0
	}

	// No matching symbols - lose
	return -bet
}

// extractSlotValue extracts the slot value from params.
func extractSlotValue(params map[string]any) (int, error) {
	if params == nil {
		return 0, ErrMissingSlotValue
	}

	v, ok := params["slot_value"]
	if !ok {
		return 0, ErrMissingSlotValue
	}

	var slotValue int
	switch val := v.(type) {
	case int:
		slotValue = val
	case int64:
		slotValue = int(val)
	case float64:
		slotValue = int(val)
	default:
		return 0, ErrMissingSlotValue
	}

	if slotValue < 1 || slotValue > 64 {
		return 0, ErrInvalidSlotValue
	}

	return slotValue, nil
}

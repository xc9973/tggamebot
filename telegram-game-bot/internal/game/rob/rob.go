// Package rob implements the robbery game (打劫游戏).
// Requirements: Rob Game - Allow users to rob coins from other users
package rob

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"telegram-game-bot/internal/pkg/lock"
	"telegram-game-bot/internal/repository"
)

// Constants for rob game configuration
const (
	MinRobAmount          = 10           // Minimum robbery amount
	MaxRobAmount          = 1000         // Maximum robbery amount
	CooldownSeconds       = 21           // Cooldown between robbery attempts
	ProtectionThreshold   = 3            // Consecutive robberies before protection
	ProtectionDurationMin = 30           // Protection duration in minutes
	
	// Outcome chances (must sum to 100)
	SuccessChance       = 50  // 50% chance of successful robbery
	FailChance          = 20  // 20% chance of failed robbery (no transfer)
	CounterAttackChance = 30  // 30% chance of counter-attack (robber loses coins)
)

// RobOutcome represents the outcome type of a robbery attempt
type RobOutcome int

const (
	OutcomeSuccess       RobOutcome = iota // Robber successfully steals coins
	OutcomeFail                            // Robbery failed, no coins transferred
	OutcomeCounterAttack                   // Victim counter-attacks, robber loses coins
)

// Transaction types for robbery
const (
	TxTypeRob           = "rob"           // Robber gains coins
	TxTypeRobbed        = "robbed"        // Victim loses coins
	TxTypeCounterAttack = "counterattack" // Counter-attack (robber loses coins)
)

// Errors for rob game
var (
	ErrSelfRob         = errors.New("不能打劫自己")
	ErrVictimNotFound  = errors.New("目标用户未注册")
	ErrVictimProtected = errors.New("目标用户在保护期")
	ErrCooldown        = errors.New("打劫冷却中")
	ErrNoBalance       = errors.New("目标用户余额为0")
)

// ProtectionState tracks a user's protection status
type ProtectionState struct {
	ConsecutiveCount int       // Number of consecutive times robbed
	ProtectedUntil   time.Time // When protection expires
}

// RobResult contains the result of a robbery attempt
type RobResult struct {
	Success     bool
	Outcome     RobOutcome // The outcome type
	Amount      int64
	RobberName  string
	VictimName  string
	NewBalance  int64  // Robber's new balance
	Message     string // Result message
}

// RobGame manages the robbery game logic
type RobGame struct {
	userRepo *repository.UserRepository
	txRepo   *repository.TransactionRepository
	userLock *lock.UserLock

	// In-memory state (resets on restart)
	protection map[int64]*ProtectionState // victim_id -> state
	cooldowns  map[int64]time.Time        // robber_id -> last_rob_time
	mu         sync.RWMutex
}

// NewRobGame creates a new RobGame instance
func NewRobGame(
	userRepo *repository.UserRepository,
	txRepo *repository.TransactionRepository,
	userLock *lock.UserLock,
) *RobGame {
	return &RobGame{
		userRepo:   userRepo,
		txRepo:     txRepo,
		userLock:   userLock,
		protection: make(map[int64]*ProtectionState),
		cooldowns:  make(map[int64]time.Time),
	}
}

// GenerateAmount generates a random robbery amount between MinRobAmount and MaxRobAmount
func GenerateAmount() int64 {
	return int64(rand.Intn(MaxRobAmount-MinRobAmount+1) + MinRobAmount)
}

// DetermineOutcome randomly determines the outcome of a robbery attempt
// Returns: OutcomeSuccess (50%), OutcomeFail (20%), or OutcomeCounterAttack (30%)
func DetermineOutcome() RobOutcome {
	roll := rand.Intn(100) // 0-99
	if roll < SuccessChance {
		return OutcomeSuccess
	} else if roll < SuccessChance+FailChance {
		return OutcomeFail
	}
	return OutcomeCounterAttack
}

// GetCooldown returns the remaining cooldown time for a robber
func (g *RobGame) GetCooldown(robberID int64) time.Duration {
	g.mu.RLock()
	defer g.mu.RUnlock()

	lastTime, ok := g.cooldowns[robberID]
	if !ok {
		return 0
	}

	elapsed := time.Since(lastTime)
	remaining := time.Duration(CooldownSeconds)*time.Second - elapsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsProtected checks if a user is in protection period
// Returns (isProtected, remainingTime)
func (g *RobGame) IsProtected(userID int64) (bool, time.Duration) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	state, ok := g.protection[userID]
	if !ok {
		return false, 0
	}

	if time.Now().Before(state.ProtectedUntil) {
		return true, time.Until(state.ProtectedUntil)
	}

	return false, 0
}

// CanRob checks if a robbery can be performed
// Returns (canRob, errorMessage)
func (g *RobGame) CanRob(ctx context.Context, robberID, victimID int64) (bool, string) {
	// Check self-robbery
	if robberID == victimID {
		return false, "不能打劫自己"
	}

	// Check if victim exists
	exists, err := g.userRepo.Exists(ctx, victimID)
	if err != nil || !exists {
		return false, "目标用户未注册"
	}

	// Check cooldown
	if remaining := g.GetCooldown(robberID); remaining > 0 {
		secs := int(remaining.Seconds()) + 1
		return false, fmt.Sprintf("打劫冷却中，请等待 %d 秒", secs)
	}

	// Check protection
	if protected, remaining := g.IsProtected(victimID); protected {
		mins := int(remaining.Minutes()) + 1
		return false, fmt.Sprintf("目标用户在保护期，剩余 %d 分钟", mins)
	}

	return true, ""
}


// Rob executes a robbery attempt
func (g *RobGame) Rob(ctx context.Context, robberID, victimID int64, robberName, victimName string) (*RobResult, error) {
	// Validate robbery
	canRob, errMsg := g.CanRob(ctx, robberID, victimID)
	if !canRob {
		return &RobResult{
			Success: false,
			Message: errMsg,
		}, nil
	}

	// Lock both users (always lock in order to prevent deadlock)
	firstID, secondID := robberID, victimID
	if victimID < robberID {
		firstID, secondID = victimID, robberID
	}
	g.userLock.Lock(firstID)
	defer g.userLock.Unlock(firstID)
	g.userLock.Lock(secondID)
	defer g.userLock.Unlock(secondID)

	// Get both users' balances
	victim, err := g.userRepo.GetByID(ctx, victimID)
	if err != nil {
		return nil, fmt.Errorf("获取目标用户失败: %w", err)
	}

	robber, err := g.userRepo.GetByID(ctx, robberID)
	if err != nil {
		return nil, fmt.Errorf("获取打劫者信息失败: %w", err)
	}

	// Update cooldown first (regardless of outcome)
	g.mu.Lock()
	g.cooldowns[robberID] = time.Now()
	g.mu.Unlock()

	// Determine outcome
	outcome := DetermineOutcome()

	switch outcome {
	case OutcomeFail:
		// Robbery failed - no coins transferred
		return &RobResult{
			Success:    false,
			Outcome:    OutcomeFail,
			Amount:     0,
			RobberName: robberName,
			VictimName: victimName,
			NewBalance: robber.Balance,
			Message:    fmt.Sprintf("😅 %s 打劫 %s 失败了！空手而归...", robberName, victimName),
		}, nil

	case OutcomeCounterAttack:
		// Counter-attack - robber loses coins to victim
		amount := GenerateAmount()
		// Cap at robber's balance (can't go negative)
		if amount > robber.Balance {
			amount = robber.Balance
		}
		
		if amount <= 0 {
			return &RobResult{
				Success:    false,
				Outcome:    OutcomeCounterAttack,
				Amount:     0,
				RobberName: robberName,
				VictimName: victimName,
				NewBalance: robber.Balance,
				Message:    fmt.Sprintf("⚔️ %s 被 %s 反击了！但你身无分文，逃过一劫...", robberName, victimName),
			}, nil
		}

		// Transfer coins: deduct from robber
		newRobber, err := g.userRepo.UpdateBalance(ctx, robberID, -amount)
		if err != nil {
			return nil, fmt.Errorf("扣除打劫者余额失败: %w", err)
		}

		// Transfer coins: add to victim
		_, err = g.userRepo.UpdateBalance(ctx, victimID, amount)
		if err != nil {
			// Try to rollback robber's balance
			g.userRepo.UpdateBalance(ctx, robberID, amount)
			return nil, fmt.Errorf("增加目标用户余额失败: %w", err)
		}

		// Record transactions
		counterDesc := fmt.Sprintf("打劫 %s 被反击损失 %d 金币", victimName, amount)
		g.txRepo.Create(ctx, robberID, -amount, TxTypeCounterAttack, &counterDesc)

		victimGainDesc := fmt.Sprintf("反击 %s 获得 %d 金币", robberName, amount)
		g.txRepo.Create(ctx, victimID, amount, TxTypeRob, &victimGainDesc)

		return &RobResult{
			Success:    false,
			Outcome:    OutcomeCounterAttack,
			Amount:     amount,
			RobberName: robberName,
			VictimName: victimName,
			NewBalance: newRobber.Balance,
			Message:    fmt.Sprintf("⚔️ %s 打劫 %s 被反击！损失 %d 金币！", robberName, victimName, amount),
		}, nil

	default: // OutcomeSuccess
		// Successful robbery
		if victim.Balance <= 0 {
			return &RobResult{
				Success: false,
				Outcome: OutcomeFail,
				Message: "目标用户余额为0，无法打劫",
			}, nil
		}

		amount := GenerateAmount()
		// Cap at victim's balance
		if amount > victim.Balance {
			amount = victim.Balance
		}

		// Transfer coins: deduct from victim
		_, err = g.userRepo.UpdateBalance(ctx, victimID, -amount)
		if err != nil {
			return nil, fmt.Errorf("扣除目标用户余额失败: %w", err)
		}

		// Transfer coins: add to robber
		newRobber, err := g.userRepo.UpdateBalance(ctx, robberID, amount)
		if err != nil {
			// Try to rollback victim's balance
			g.userRepo.UpdateBalance(ctx, victimID, amount)
			return nil, fmt.Errorf("增加打劫者余额失败: %w", err)
		}

		// Record transactions
		robDesc := fmt.Sprintf("打劫 %s 获得 %d 金币", victimName, amount)
		g.txRepo.Create(ctx, robberID, amount, TxTypeRob, &robDesc)

		robbedDesc := fmt.Sprintf("被 %s 打劫损失 %d 金币", robberName, amount)
		g.txRepo.Create(ctx, victimID, -amount, TxTypeRobbed, &robbedDesc)

		// Update victim's protection state
		g.mu.Lock()
		state, ok := g.protection[victimID]
		if !ok {
			state = &ProtectionState{}
			g.protection[victimID] = state
		}

		// Check if protection has expired, reset count if so
		if time.Now().After(state.ProtectedUntil) && state.ConsecutiveCount > 0 {
			state.ConsecutiveCount = 0
		}

		state.ConsecutiveCount++

		// Activate protection if threshold reached
		protectionActivated := false
		if state.ConsecutiveCount >= ProtectionThreshold {
			state.ProtectedUntil = time.Now().Add(time.Duration(ProtectionDurationMin) * time.Minute)
			state.ConsecutiveCount = 0 // Reset after protection activates
			protectionActivated = true
		}
		g.mu.Unlock()

		// Build result message
		msg := fmt.Sprintf("🔫 %s 打劫了 %s，获得 %d 金币！", robberName, victimName, amount)
		if protectionActivated {
			msg += fmt.Sprintf("\n🛡️ %s 触发保护期 %d 分钟", victimName, ProtectionDurationMin)
		}

		return &RobResult{
			Success:    true,
			Outcome:    OutcomeSuccess,
			Amount:     amount,
			RobberName: robberName,
			VictimName: victimName,
			NewBalance: newRobber.Balance,
			Message:    msg,
		}, nil
	}
}

// ResetProtection resets a user's protection state (for testing)
func (g *RobGame) ResetProtection(userID int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.protection, userID)
}

// ResetCooldown resets a user's cooldown (for testing)
func (g *RobGame) ResetCooldown(userID int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.cooldowns, userID)
}

// GetProtectionState returns the protection state for a user (for testing)
func (g *RobGame) GetProtectionState(userID int64) *ProtectionState {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.protection[userID]
}

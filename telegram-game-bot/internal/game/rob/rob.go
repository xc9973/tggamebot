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
	
	// Outcome chances (must sum to 100) - default without items
	SuccessChance       = 50  // 50% chance of successful robbery
	FailChance          = 20  // 20% chance of failed robbery (no transfer)
	CounterAttackChance = 30  // 30% chance of counter-attack (robber loses coins)
	
	// Bloodthirst sword success rate
	BloodthirstSuccessChance = 80 // 80% success rate with bloodthirst sword
	
	// Blunt knife amount limits
	// Requirements: 6.5 - Blunt knife limits robbery amount to 1-100
	BluntKnifeMinAmount = 1   // Minimum robbery amount with blunt knife
	BluntKnifeMaxAmount = 100 // Maximum robbery amount with blunt knife
	
	// Great sword critical hit
	// Requirements: 7.6 - Great sword has 0.01% chance to rob 90% of target's coins
	GreatSwordCriticalChance = 1     // 0.01% = 1 in 10000
	GreatSwordCriticalDenom  = 10000 // Denominator for critical chance calculation
	GreatSwordCriticalPercent = 90   // Rob 90% of target's coins on critical hit
)

// ItemEffectChecker interface for checking shop item effects
// This allows the rob game to check item effects without depending on shop service directly
type ItemEffectChecker interface {
	// IsHandcuffed checks if user is locked by handcuffs
	IsHandcuffed(ctx context.Context, userID int64) (bool, time.Duration)
	// HasShield checks if user has active shield
	HasShield(ctx context.Context, userID int64) bool
	// HasThornArmor checks if user has active thorn armor
	HasThornArmor(ctx context.Context, userID int64) bool
	// HasBloodthirstSword checks if user has active bloodthirst sword
	HasBloodthirstSword(ctx context.Context, userID int64) bool
	// HasEmperorClothes checks if user has active emperor clothes (highest priority defense)
	// Emperor clothes immune ALL attacks including bypass defense items (blunt knife, great sword)
	HasEmperorClothes(ctx context.Context, userID int64) bool
	// HasBluntKnife checks if user has active blunt knife
	// Blunt knife bypasses Shield and Thorn Armor but NOT Emperor Clothes
	// Requirements: 6.4 - Bypass defense check
	HasBluntKnife(ctx context.Context, userID int64) bool
	// HasGreatSword checks if user has active great sword
	// Great sword bypasses Shield and Thorn Armor but NOT Emperor Clothes
	// Great sword has 0.01% chance to rob 90% of target's coins
	// Requirements: 7.5, 7.6 - Bypass defense and critical hit
	HasGreatSword(ctx context.Context, userID int64) bool
	// HasGoldenCassock checks if user has active golden cassock
	// Golden cassock removes attacker's defensive items (Shield, Thorn Armor)
	// Requirements: 8.3, 8.4 - Golden cassock defense removal
	HasGoldenCassock(ctx context.Context, userID int64) bool
	// RemoveDefensiveItems removes all defensive items (Shield, Thorn Armor) from a user
	// This is triggered by Golden Cassock effect
	// Requirements: 8.4 - Remove attacker's defensive items
	RemoveDefensiveItems(ctx context.Context, userID int64) error
	// DecrementUseCountByString decreases the use count of an item by 1
	DecrementUseCountByString(ctx context.Context, userID int64, effectType string) error
}

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
	userRepo    *repository.UserRepository
	txRepo      *repository.TransactionRepository
	userLock    *lock.UserLock
	itemChecker ItemEffectChecker // Optional: for shop item effects

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

// SetItemChecker sets the item effect checker (called after shop service is initialized)
func (g *RobGame) SetItemChecker(checker ItemEffectChecker) {
	g.itemChecker = checker
}

// GenerateAmount generates a random robbery amount between MinRobAmount and MaxRobAmount
func GenerateAmount() int64 {
	return int64(rand.Intn(MaxRobAmount-MinRobAmount+1) + MinRobAmount)
}

// GenerateBluntKnifeAmount generates a random robbery amount for blunt knife (1-100)
// Requirements: 6.5 - Blunt knife limits robbery amount to 1-100
func GenerateBluntKnifeAmount() int64 {
	return int64(rand.Intn(BluntKnifeMaxAmount-BluntKnifeMinAmount+1) + BluntKnifeMinAmount)
}

// IsGreatSwordCritical checks if great sword triggers a critical hit (0.01% chance)
// Requirements: 7.6 - Great sword has 0.01% chance to rob 90% of target's coins
func IsGreatSwordCritical() bool {
	return rand.Intn(GreatSwordCriticalDenom) < GreatSwordCriticalChance
}

// CalculateGreatSwordCriticalAmount calculates the amount for a great sword critical hit (90% of target's balance)
// Requirements: 7.6 - Rob 90% of target's coins on critical hit
func CalculateGreatSwordCriticalAmount(targetBalance int64) int64 {
	return targetBalance * GreatSwordCriticalPercent / 100
}

// DetermineOutcome randomly determines the outcome of a robbery attempt
// Returns: OutcomeSuccess (50%), OutcomeFail (20%), or OutcomeCounterAttack (30%)
func DetermineOutcome() RobOutcome {
	return DetermineOutcomeWithRate(SuccessChance)
}

// DetermineOutcomeWithRate determines outcome with custom success rate
func DetermineOutcomeWithRate(successRate int) RobOutcome {
	roll := rand.Intn(100) // 0-99
	if roll < successRate {
		return OutcomeSuccess
	}
	// Distribute remaining chance between fail and counter-attack
	// Keep same ratio: fail 20%, counter 30% -> fail 40%, counter 60% of remaining
	remaining := 100 - successRate
	failThreshold := successRate + (remaining * 40 / 100)
	if roll < failThreshold {
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

	// Check shop item effects
	if g.itemChecker != nil {
		// Check if robber is handcuffed
		if locked, remaining := g.itemChecker.IsHandcuffed(ctx, robberID); locked {
			mins := int(remaining.Minutes()) + 1
			return false, fmt.Sprintf("🔗 你被手铐锁定，无法打劫！剩余 %d 分钟", mins)
		}

		// Check if victim has Emperor Clothes (highest priority defense)
		// Emperor Clothes immune ALL attacks including bypass defense items (blunt knife, great sword)
		// Requirements: 9.4, 9.5 - Emperor clothes prevents ALL robbery attempts
		if g.itemChecker.HasEmperorClothes(ctx, victimID) {
			return false, "👑 目标有皇帝的新衣，无法打劫"
		}

		// Check if victim has Golden Cassock - triggers defense removal on attacker
		// Requirements: 8.4 - Golden cassock removes attacker's defensive items (Shield, Thorn Armor)
		if g.itemChecker.HasGoldenCassock(ctx, victimID) {
			// Remove attacker's defensive items (Shield, Thorn Armor)
			g.itemChecker.RemoveDefensiveItems(ctx, robberID)
			// Decrement golden cassock use count
			g.itemChecker.DecrementUseCountByString(ctx, victimID, "golden_cassock")
		}

		// Check if robber has blunt knife or great sword (bypasses shield and thorn armor)
		// Requirements: 6.4 - Blunt knife ignores Shield and Thorn Armor (but NOT Emperor Clothes)
		// Requirements: 7.5 - Great sword ignores Shield and Thorn Armor (but NOT Emperor Clothes)
		hasBluntKnife := g.itemChecker.HasBluntKnife(ctx, robberID)
		hasGreatSword := g.itemChecker.HasGreatSword(ctx, robberID)
		hasBypassDefense := hasBluntKnife || hasGreatSword

		// Check if victim has shield (can be bypassed by blunt knife/great sword)
		// Requirements: 6.4, 7.5 - Blunt knife and great sword bypass shield
		if g.itemChecker.HasShield(ctx, victimID) && !hasBypassDefense {
			return false, "🛡️ 目标有保护罩，无法打劫"
		}
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
	// Use TryLock to avoid blocking if someone else is using the lock
	firstID, secondID := robberID, victimID
	if victimID < robberID {
		firstID, secondID = victimID, robberID
	}
	
	// Try to acquire first lock
	if !g.userLock.TryLock(firstID) {
		return &RobResult{
			Success: false,
			Message: "系统繁忙，请稍后重试",
		}, nil
	}
	defer g.userLock.Unlock(firstID)
	
	// Try to acquire second lock
	if !g.userLock.TryLock(secondID) {
		return &RobResult{
			Success: false,
			Message: "目标用户正在进行其他操作，请稍后重试",
		}, nil
	}
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

	// Check for bloodthirst sword effect (80% success rate)
	successRate := SuccessChance
	hasBloodthirst := false
	if g.itemChecker != nil && g.itemChecker.HasBloodthirstSword(ctx, robberID) {
		successRate = BloodthirstSuccessChance
		hasBloodthirst = true
	}

	// Determine outcome with appropriate success rate
	outcome := DetermineOutcomeWithRate(successRate)

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

		// Check for blunt knife effect
		// Requirements: 6.4, 6.5 - Blunt knife bypasses defense and limits amount to 1-100
		hasBluntKnife := false
		if g.itemChecker != nil && g.itemChecker.HasBluntKnife(ctx, robberID) {
			hasBluntKnife = true
		}

		// Check for great sword effect
		// Requirements: 7.5, 7.6 - Great sword bypasses defense and has 0.01% critical hit
		hasGreatSword := false
		isGreatSwordCritical := false
		if g.itemChecker != nil && g.itemChecker.HasGreatSword(ctx, robberID) {
			hasGreatSword = true
			// Check for critical hit (0.01% chance)
			isGreatSwordCritical = IsGreatSwordCritical()
		}

		// Generate robbery amount based on weapon
		var amount int64
		if hasBluntKnife {
			// Blunt knife limits amount to 1-100
			// Requirements: 6.5 - Blunt knife limits robbery amount to 1-100
			amount = GenerateBluntKnifeAmount()
		} else if hasGreatSword && isGreatSwordCritical {
			// Great sword critical hit - rob 90% of target's coins
			// Requirements: 7.6 - Great sword has 0.01% chance to rob 90% of target's coins
			amount = CalculateGreatSwordCriticalAmount(victim.Balance)
		} else {
			amount = GenerateAmount()
		}
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

		// Check for thorn armor effect - attacker loses double coins
		// Requirements: 6.4 - Blunt knife bypasses thorn armor
		// Requirements: 7.5 - Great sword bypasses thorn armor
		thornArmorTriggered := false
		thornDamage := int64(0)
		// Blunt knife and great sword bypass thorn armor effect
		hasBypassDefense := hasBluntKnife || hasGreatSword
		if g.itemChecker != nil && g.itemChecker.HasThornArmor(ctx, victimID) && !hasBypassDefense {
			thornDamage = amount * 2
			// Cap at robber's new balance
			if thornDamage > newRobber.Balance {
				thornDamage = newRobber.Balance
			}
			if thornDamage > 0 {
				// Deduct from robber
				newRobber, err = g.userRepo.UpdateBalance(ctx, robberID, -thornDamage)
				if err == nil {
					// Add to victim
					g.userRepo.UpdateBalance(ctx, victimID, thornDamage)
					// Record transactions
					thornDesc := fmt.Sprintf("荆棘刺甲反伤 %d 金币", thornDamage)
					g.txRepo.Create(ctx, robberID, -thornDamage, TxTypeRobbed, &thornDesc)
					thornGainDesc := fmt.Sprintf("荆棘刺甲反伤获得 %d 金币", thornDamage)
					g.txRepo.Create(ctx, victimID, thornDamage, TxTypeRob, &thornGainDesc)
					thornArmorTriggered = true
				}
			}
		}

		// Decrement blunt knife use count after successful use
		// Requirements: 6.5 - Decrement use count by 1 on each use
		if hasBluntKnife && g.itemChecker != nil {
			g.itemChecker.DecrementUseCountByString(ctx, robberID, "blunt_knife")
		}

		// Decrement great sword use count after successful use
		// Requirements: 7.6 - Decrement use count by 1 on each use
		if hasGreatSword && g.itemChecker != nil {
			g.itemChecker.DecrementUseCountByString(ctx, robberID, "great_sword")
		}

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
		if hasBluntKnife {
			msg = fmt.Sprintf("🔪 %s 使用钝刀打劫了 %s，获得 %d 金币！", robberName, victimName, amount)
		} else if hasGreatSword {
			if isGreatSwordCritical {
				// Great sword critical hit message
				// Requirements: 7.6 - Great sword has 0.01% chance to rob 90% of target's coins
				msg = fmt.Sprintf("⚔️💥 %s 使用大宝剑打劫了 %s，触发暴击！获得 %d 金币（90%%）！", robberName, victimName, amount)
			} else {
				msg = fmt.Sprintf("⚔️ %s 使用大宝剑打劫了 %s，获得 %d 金币！", robberName, victimName, amount)
			}
		} else if hasBloodthirst {
			msg = fmt.Sprintf("🗡️ %s 使用饮血剑打劫了 %s，获得 %d 金币！", robberName, victimName, amount)
		}
		if thornArmorTriggered {
			msg += fmt.Sprintf("\n🌵 荆棘刺甲反伤！%s 损失 %d 金币！", robberName, thornDamage)
		}
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

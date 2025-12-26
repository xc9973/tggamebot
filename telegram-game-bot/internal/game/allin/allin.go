// Package allin implements all-in gambling games (梭哈游戏).
package allin

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

// Constants for all-in game configuration
const (
	MinAllInBalance    = 100 // Minimum balance to participate
	AllInRobCooldown   = 60  // Cooldown for all-in rob (seconds)
	AllInDiceCooldown  = 30  // Cooldown for all-in dice (seconds)
	DuelTimeout        = 60  // Duel timeout (seconds)
	AllInSuccessChance = 50  // 50% success rate
	DiceWinThreshold   = 7   // Dice total >= 7 wins
)

// Transaction types
const (
	TxTypeAllInRobWin  = "allin_rob_win"
	TxTypeAllInRobLose = "allin_rob_lose"
	TxTypeDuelWin      = "duel_win"
	TxTypeDuelLose     = "duel_lose"
	TxTypeDiceWin      = "dice_win"
	TxTypeDiceLose     = "dice_lose"
)

// Errors
var (
	ErrInsufficientBalance = errors.New("余额不足100金币，无法参与梭哈")
	ErrSelfAllIn           = errors.New("不能对自己梭哈")
	ErrTargetNotFound      = errors.New("目标用户未注册")
	ErrCooldown            = errors.New("梭哈冷却中")
	ErrEmperorClothes      = errors.New("目标有皇帝的新衣，无法梭哈")
	ErrPendingDuel         = errors.New("你已有待处理的对决")
	ErrNoPendingDuel       = errors.New("没有待处理的对决")
	ErrDuelTimeout         = errors.New("对决已超时")
	ErrNotDuelTarget       = errors.New("这不是你的对决")
)

// ItemEffectChecker interface for checking shop item effects
type ItemEffectChecker interface {
	HasEmperorClothes(ctx context.Context, userID int64) bool
	DecrementUseCountByString(ctx context.Context, userID int64, effectType string) error
}

// DuelRequest represents a pending duel challenge
type DuelRequest struct {
	ChallengerID   int64
	ChallengerName string
	TargetID       int64
	TargetName     string
	Amount         int64
	CreatedAt      time.Time
	MessageID      int
	ChatID         int64
}

// AllInResult represents the result of an all-in rob
type AllInResult struct {
	Success      bool
	Amount       int64
	AttackerName string
	VictimName   string
	NewBalance   int64
	Message      string
}

// DuelResult represents the result of a duel
type DuelResult struct {
	WinnerID   int64
	WinnerName string
	LoserID    int64
	LoserName  string
	Amount     int64
	Message    string
}

// DiceResult represents the result of an all-in dice roll
type DiceResult struct {
	Dice1      int
	Dice2      int
	Total      int
	Won        bool
	OldBalance int64
	NewBalance int64
	Message    string
}

// AllInGame manages all-in gambling games
type AllInGame struct {
	userRepo    *repository.UserRepository
	txRepo      *repository.TransactionRepository
	userLock    *lock.UserLock
	itemChecker ItemEffectChecker

	robCooldowns  map[int64]time.Time
	diceCooldowns map[int64]time.Time
	pendingDuels  map[int64]*DuelRequest // target_id -> request
	
	mu sync.RWMutex
}

// NewAllInGame creates a new AllInGame instance
func NewAllInGame(
	userRepo *repository.UserRepository,
	txRepo *repository.TransactionRepository,
	userLock *lock.UserLock,
) *AllInGame {
	return &AllInGame{
		userRepo:      userRepo,
		txRepo:        txRepo,
		userLock:      userLock,
		robCooldowns:  make(map[int64]time.Time),
		diceCooldowns: make(map[int64]time.Time),
		pendingDuels:  make(map[int64]*DuelRequest),
	}
}

// SetItemChecker sets the item effect checker
func (g *AllInGame) SetItemChecker(checker ItemEffectChecker) {
	g.itemChecker = checker
}


// GetRobCooldown returns remaining cooldown for all-in rob
func (g *AllInGame) GetRobCooldown(userID int64) time.Duration {
	g.mu.RLock()
	defer g.mu.RUnlock()

	lastTime, ok := g.robCooldowns[userID]
	if !ok {
		return 0
	}

	remaining := time.Duration(AllInRobCooldown)*time.Second - time.Since(lastTime)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetDiceCooldown returns remaining cooldown for all-in dice
func (g *AllInGame) GetDiceCooldown(userID int64) time.Duration {
	g.mu.RLock()
	defer g.mu.RUnlock()

	lastTime, ok := g.diceCooldowns[userID]
	if !ok {
		return 0
	}

	remaining := time.Duration(AllInDiceCooldown)*time.Second - time.Since(lastTime)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// AllInRob executes an all-in robbery attempt
func (g *AllInGame) AllInRob(ctx context.Context, robberID, victimID int64, robberName, victimName string) (*AllInResult, error) {
	// Check self-robbery
	if robberID == victimID {
		return nil, ErrSelfAllIn
	}

	// Check if victim exists
	exists, err := g.userRepo.Exists(ctx, victimID)
	if err != nil || !exists {
		return nil, ErrTargetNotFound
	}

	// Check cooldown
	if remaining := g.GetRobCooldown(robberID); remaining > 0 {
		secs := int(remaining.Seconds()) + 1
		return &AllInResult{
			Success: false,
			Message: fmt.Sprintf("梭哈打劫冷却中，请等待 %d 秒", secs),
		}, nil
	}

	// Check emperor clothes
	if g.itemChecker != nil && g.itemChecker.HasEmperorClothes(ctx, victimID) {
		g.itemChecker.DecrementUseCountByString(ctx, victimID, "emperor_clothes")
		return &AllInResult{
			Success: false,
			Message: "👑 目标有皇帝的新衣，无法梭哈打劫",
		}, nil
	}

	// Lock both users
	firstID, secondID := robberID, victimID
	if victimID < robberID {
		firstID, secondID = victimID, robberID
	}

	if !g.userLock.TryLock(firstID) {
		return &AllInResult{
			Success: false,
			Message: "系统繁忙，请稍后重试",
		}, nil
	}
	defer g.userLock.Unlock(firstID)

	if !g.userLock.TryLock(secondID) {
		return &AllInResult{
			Success: false,
			Message: "目标用户正在进行其他操作，请稍后重试",
		}, nil
	}
	defer g.userLock.Unlock(secondID)

	// Get balances
	robber, err := g.userRepo.GetByID(ctx, robberID)
	if err != nil {
		return nil, err
	}

	victim, err := g.userRepo.GetByID(ctx, victimID)
	if err != nil {
		return nil, err
	}

	// Check minimum balance
	if robber.Balance < MinAllInBalance {
		return &AllInResult{
			Success: false,
			Message: fmt.Sprintf("余额不足 %d 金币，无法梭哈打劫", MinAllInBalance),
		}, nil
	}

	// Update cooldown
	g.mu.Lock()
	g.robCooldowns[robberID] = time.Now()
	g.mu.Unlock()

	// Calculate amount (min of both balances)
	amount := robber.Balance
	if victim.Balance < amount {
		amount = victim.Balance
	}

	// 50% success rate
	success := rand.Intn(100) < AllInSuccessChance

	if success {
		// Success: robber wins
		g.userRepo.UpdateBalance(ctx, victimID, -amount)
		newRobber, _ := g.userRepo.UpdateBalance(ctx, robberID, amount)

		// Record transactions
		winDesc := fmt.Sprintf("梭哈打劫 %s 成功，获得 %d 金币", victimName, amount)
		g.txRepo.Create(ctx, robberID, amount, TxTypeAllInRobWin, &winDesc)
		loseDesc := fmt.Sprintf("被 %s 梭哈打劫，损失 %d 金币", robberName, amount)
		g.txRepo.Create(ctx, victimID, -amount, TxTypeAllInRobLose, &loseDesc)

		return &AllInResult{
			Success:      true,
			Amount:       amount,
			AttackerName: robberName,
			VictimName:   victimName,
			NewBalance:   newRobber.Balance,
			Message:      fmt.Sprintf("🎰 梭哈成功！%s 打劫 %s 获得 %d 金币！", robberName, victimName, amount),
		}, nil
	} else {
		// Failure: robber loses all
		loseAmount := robber.Balance
		g.userRepo.UpdateBalance(ctx, robberID, -loseAmount)
		g.userRepo.UpdateBalance(ctx, victimID, loseAmount)

		// Record transactions
		loseDesc := fmt.Sprintf("梭哈打劫 %s 失败，损失 %d 金币", victimName, loseAmount)
		g.txRepo.Create(ctx, robberID, -loseAmount, TxTypeAllInRobLose, &loseDesc)
		winDesc := fmt.Sprintf("被 %s 梭哈打劫失败，获得 %d 金币", robberName, loseAmount)
		g.txRepo.Create(ctx, victimID, loseAmount, TxTypeAllInRobWin, &winDesc)

		return &AllInResult{
			Success:      false,
			Amount:       loseAmount,
			AttackerName: robberName,
			VictimName:   victimName,
			NewBalance:   0,
			Message:      fmt.Sprintf("💀 梭哈失败！%s 打劫 %s 失败，损失全部 %d 金币！", robberName, victimName, loseAmount),
		}, nil
	}
}


// AllInDice plays the all-in dice game
func (g *AllInGame) AllInDice(ctx context.Context, userID int64, userName string) (*DiceResult, error) {
	// Check cooldown
	if remaining := g.GetDiceCooldown(userID); remaining > 0 {
		secs := int(remaining.Seconds()) + 1
		return &DiceResult{
			Won:     false,
			Message: fmt.Sprintf("梭哈骰子冷却中，请等待 %d 秒", secs),
		}, nil
	}

	// Lock user
	g.userLock.Lock(userID)
	defer g.userLock.Unlock(userID)

	// Get balance
	user, err := g.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check minimum balance
	if user.Balance < MinAllInBalance {
		return &DiceResult{
			Won:     false,
			Message: fmt.Sprintf("余额不足 %d 金币，无法梭哈骰子", MinAllInBalance),
		}, nil
	}

	// Update cooldown
	g.mu.Lock()
	g.diceCooldowns[userID] = time.Now()
	g.mu.Unlock()

	oldBalance := user.Balance

	// Roll two dice
	dice1 := rand.Intn(6) + 1
	dice2 := rand.Intn(6) + 1
	total := dice1 + dice2

	if total >= DiceWinThreshold {
		// Win: double balance
		winAmount := oldBalance
		newUser, _ := g.userRepo.UpdateBalance(ctx, userID, winAmount)

		winDesc := fmt.Sprintf("梭哈骰子 %d+%d=%d 赢了，获得 %d 金币", dice1, dice2, total, winAmount)
		g.txRepo.Create(ctx, userID, winAmount, TxTypeDiceWin, &winDesc)

		return &DiceResult{
			Dice1:      dice1,
			Dice2:      dice2,
			Total:      total,
			Won:        true,
			OldBalance: oldBalance,
			NewBalance: newUser.Balance,
			Message:    fmt.Sprintf("🎲 %s 掷出 %d + %d = %d\n🎉 赢了！金币翻倍：%d → %d", userName, dice1, dice2, total, oldBalance, newUser.Balance),
		}, nil
	} else {
		// Lose: balance becomes 0
		g.userRepo.UpdateBalance(ctx, userID, -oldBalance)

		loseDesc := fmt.Sprintf("梭哈骰子 %d+%d=%d 输了，损失 %d 金币", dice1, dice2, total, oldBalance)
		g.txRepo.Create(ctx, userID, -oldBalance, TxTypeDiceLose, &loseDesc)

		return &DiceResult{
			Dice1:      dice1,
			Dice2:      dice2,
			Total:      total,
			Won:        false,
			OldBalance: oldBalance,
			NewBalance: 0,
			Message:    fmt.Sprintf("🎲 %s 掷出 %d + %d = %d\n💀 输了！金币清零：%d → 0", userName, dice1, dice2, total, oldBalance),
		}, nil
	}
}

// CreateDuel creates a duel challenge
func (g *AllInGame) CreateDuel(ctx context.Context, challengerID, targetID int64, challengerName, targetName string, chatID int64) (*DuelRequest, error) {
	// Check self-duel
	if challengerID == targetID {
		return nil, ErrSelfAllIn
	}

	// Check if target exists
	exists, err := g.userRepo.Exists(ctx, targetID)
	if err != nil || !exists {
		return nil, ErrTargetNotFound
	}

	// Check if challenger already has pending duel
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, duel := range g.pendingDuels {
		if duel.ChallengerID == challengerID {
			return nil, ErrPendingDuel
		}
	}

	// Check if target already has pending duel
	if _, exists := g.pendingDuels[targetID]; exists {
		return nil, errors.New("目标已有待处理的对决")
	}

	// Get balances
	challenger, err := g.userRepo.GetByID(ctx, challengerID)
	if err != nil {
		return nil, err
	}

	target, err := g.userRepo.GetByID(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Check minimum balance
	if challenger.Balance < MinAllInBalance {
		return nil, ErrInsufficientBalance
	}
	if target.Balance < MinAllInBalance {
		return nil, errors.New("目标余额不足100金币")
	}

	// Calculate amount
	amount := challenger.Balance
	if target.Balance < amount {
		amount = target.Balance
	}

	// Create duel request
	duel := &DuelRequest{
		ChallengerID:   challengerID,
		ChallengerName: challengerName,
		TargetID:       targetID,
		TargetName:     targetName,
		Amount:         amount,
		CreatedAt:      time.Now(),
		ChatID:         chatID,
	}

	g.pendingDuels[targetID] = duel

	// Start timeout goroutine
	go func() {
		time.Sleep(time.Duration(DuelTimeout) * time.Second)
		g.mu.Lock()
		defer g.mu.Unlock()
		if d, exists := g.pendingDuels[targetID]; exists && d.CreatedAt.Equal(duel.CreatedAt) {
			delete(g.pendingDuels, targetID)
		}
	}()

	return duel, nil
}

// SetDuelMessageID sets the message ID for a pending duel
func (g *AllInGame) SetDuelMessageID(targetID int64, messageID int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if duel, exists := g.pendingDuels[targetID]; exists {
		duel.MessageID = messageID
	}
}

// GetPendingDuel returns the pending duel for a target
func (g *AllInGame) GetPendingDuel(targetID int64) *DuelRequest {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.pendingDuels[targetID]
}

// AcceptDuel accepts and executes a duel
func (g *AllInGame) AcceptDuel(ctx context.Context, targetID int64) (*DuelResult, error) {
	g.mu.Lock()
	duel, exists := g.pendingDuels[targetID]
	if !exists {
		g.mu.Unlock()
		return nil, ErrNoPendingDuel
	}

	// Check timeout
	if time.Since(duel.CreatedAt) > time.Duration(DuelTimeout)*time.Second {
		delete(g.pendingDuels, targetID)
		g.mu.Unlock()
		return nil, ErrDuelTimeout
	}

	delete(g.pendingDuels, targetID)
	g.mu.Unlock()

	// Lock both users
	firstID, secondID := duel.ChallengerID, targetID
	if targetID < duel.ChallengerID {
		firstID, secondID = targetID, duel.ChallengerID
	}

	g.userLock.Lock(firstID)
	defer g.userLock.Unlock(firstID)
	g.userLock.Lock(secondID)
	defer g.userLock.Unlock(secondID)

	// Get current balances
	challenger, err := g.userRepo.GetByID(ctx, duel.ChallengerID)
	if err != nil {
		return nil, err
	}

	target, err := g.userRepo.GetByID(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Recalculate amount based on current balances
	amount := challenger.Balance
	if target.Balance < amount {
		amount = target.Balance
	}

	if amount < MinAllInBalance {
		return nil, ErrInsufficientBalance
	}

	// 50/50 duel
	challengerWins := rand.Intn(100) < 50

	var winnerID, loserID int64
	var winnerName, loserName string

	if challengerWins {
		winnerID, loserID = duel.ChallengerID, targetID
		winnerName, loserName = duel.ChallengerName, duel.TargetName
	} else {
		winnerID, loserID = targetID, duel.ChallengerID
		winnerName, loserName = duel.TargetName, duel.ChallengerName
	}

	// Transfer coins
	g.userRepo.UpdateBalance(ctx, loserID, -amount)
	g.userRepo.UpdateBalance(ctx, winnerID, amount)

	// Record transactions
	winDesc := fmt.Sprintf("对决 %s 获胜，获得 %d 金币", loserName, amount)
	g.txRepo.Create(ctx, winnerID, amount, TxTypeDuelWin, &winDesc)
	loseDesc := fmt.Sprintf("对决 %s 失败，损失 %d 金币", winnerName, amount)
	g.txRepo.Create(ctx, loserID, -amount, TxTypeDuelLose, &loseDesc)

	return &DuelResult{
		WinnerID:   winnerID,
		WinnerName: winnerName,
		LoserID:    loserID,
		LoserName:  loserName,
		Amount:     amount,
		Message:    fmt.Sprintf("⚔️ 对决结果：%s 获胜！\n💰 %s 获得 %d 金币", winnerName, winnerName, amount),
	}, nil
}

// DeclineDuel declines a duel
func (g *AllInGame) DeclineDuel(targetID int64) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.pendingDuels[targetID]; !exists {
		return ErrNoPendingDuel
	}

	delete(g.pendingDuels, targetID)
	return nil
}

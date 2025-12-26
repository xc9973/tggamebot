# Design Document: 梭哈游戏系统

## Overview

梭哈游戏系统为玩家提供三种高风险高回报的赌博玩法：梭哈打劫、梭哈对决和梭哈骰子。所有玩法都需要最低100金币参与。

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Handler Layer                         │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐       │
│  │ /shdj       │ │ /duijue     │ │ /shdice     │       │
│  └──────┬──────┘ └──────┬──────┘ └──────┬──────┘       │
└─────────┼───────────────┼───────────────┼───────────────┘
          │               │               │
┌─────────▼───────────────▼───────────────▼───────────────┐
│                   AllIn Game Service                     │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐       │
│  │ AllInRob    │ │ Duel        │ │ AllInDice   │       │
│  └─────────────┘ └─────────────┘ └─────────────┘       │
│                                                          │
│  ┌─────────────────────────────────────────────┐       │
│  │ Pending Duels Map (in-memory)               │       │
│  └─────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────┘
          │
┌─────────▼───────────────────────────────────────────────┐
│                   Repository Layer                       │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐       │
│  │ UserRepo    │ │ TxRepo      │ │ ShopService │       │
│  └─────────────┘ └─────────────┘ └─────────────┘       │
└─────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### AllInGame struct

```go
type AllInGame struct {
    userRepo     *repository.UserRepository
    txRepo       *repository.TransactionRepository
    userLock     *lock.UserLock
    itemChecker  ItemEffectChecker
    
    // Cooldowns
    robCooldowns  map[int64]time.Time  // user_id -> last_rob_time
    diceCooldowns map[int64]time.Time  // user_id -> last_dice_time
    
    // Pending duels
    pendingDuels map[int64]*DuelRequest  // challenger_id -> request
    
    mu sync.RWMutex
}

type DuelRequest struct {
    ChallengerID   int64
    ChallengerName string
    TargetID       int64
    TargetName     string
    Amount         int64     // min of both balances
    CreatedAt      time.Time
    MessageID      int       // for updating the message
    ChatID         int64
}
```

### Constants

```go
const (
    MinAllInBalance      = 100   // 最低参与金额
    AllInRobCooldown     = 60    // 梭哈打劫冷却（秒）
    AllInDiceCooldown    = 30    // 梭哈骰子冷却（秒）
    DuelTimeout          = 60    // 对决超时（秒）
    AllInSuccessChance   = 50    // 成功率50%
    DiceWinThreshold     = 7     // 骰子>=7赢
)
```

### Methods

```go
// AllInRob executes all-in robbery
func (g *AllInGame) AllInRob(ctx context.Context, robberID, victimID int64, robberName, victimName string) (*AllInResult, error)

// CreateDuel creates a duel challenge
func (g *AllInGame) CreateDuel(ctx context.Context, challengerID, targetID int64, challengerName, targetName string) (*DuelRequest, error)

// AcceptDuel accepts and executes a duel
func (g *AllInGame) AcceptDuel(ctx context.Context, targetID int64) (*DuelResult, error)

// DeclineDuel declines a duel
func (g *AllInGame) DeclineDuel(ctx context.Context, targetID int64) error

// AllInDice plays all-in dice game
func (g *AllInGame) AllInDice(ctx context.Context, userID int64, userName string) (*DiceResult, error)
```

## Data Models

### AllInResult

```go
type AllInResult struct {
    Success      bool
    Amount       int64
    AttackerName string
    VictimName   string
    NewBalance   int64
    Message      string
}
```

### DuelResult

```go
type DuelResult struct {
    WinnerID     int64
    WinnerName   string
    LoserID      int64
    LoserName    string
    Amount       int64
    Message      string
}
```

### DiceResult

```go
type DiceResult struct {
    Dice1      int
    Dice2      int
    Total      int
    Won        bool
    OldBalance int64
    NewBalance int64
    Message    string
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do.*

### Property 1: Minimum Balance Requirement
*For any* user attempting any all-in game (rob, duel, dice), if their balance is less than 100 coins, the system should reject the attempt.
**Validates: Requirements 1.1, 2.5, 3.1**

### Property 2: All-In Rob Success Transfer
*For any* successful all-in rob, the transferred amount should equal min(attacker_balance, victim_balance).
**Validates: Requirements 1.2**

### Property 3: All-In Rob Failure Transfer
*For any* failed all-in rob, the attacker's entire balance should be transferred to the victim.
**Validates: Requirements 1.3**

### Property 4: Cooldown Enforcement
*For any* user, attempting all-in rob within 60 seconds or all-in dice within 30 seconds of the last attempt should be rejected.
**Validates: Requirements 1.4, 3.5**

### Property 5: Emperor Clothes Immunity
*For any* all-in rob attempt against a user with Emperor Clothes, the attempt should be blocked.
**Validates: Requirements 1.5**

### Property 6: Duel Amount Calculation
*For any* executed duel, the transferred amount should equal min(challenger_balance, target_balance).
**Validates: Requirements 2.3**

### Property 7: Dice Roll Range
*For any* dice roll, the total should be between 2 and 12 inclusive.
**Validates: Requirements 3.2**

### Property 8: Dice Win Condition
*For any* dice roll with total >= 7, the user's balance should double.
**Validates: Requirements 3.3**

### Property 9: Dice Lose Condition
*For any* dice roll with total <= 6, the user's balance should become 0.
**Validates: Requirements 3.4**

## Error Handling

| Error | Condition | Message |
|-------|-----------|---------|
| ErrInsufficientBalance | Balance < 100 | "余额不足100金币，无法参与梭哈" |
| ErrSelfAllIn | Target is self | "不能对自己梭哈" |
| ErrCooldown | Within cooldown | "梭哈冷却中，请等待 X 秒" |
| ErrEmperorClothes | Target has emperor clothes | "目标有皇帝的新衣，无法梭哈" |
| ErrPendingDuel | Already has pending duel | "你已有待处理的对决" |
| ErrNoPendingDuel | No duel to accept | "没有待处理的对决" |
| ErrDuelTimeout | Duel expired | "对决已超时" |

## Testing Strategy

### Unit Tests
- Test minimum balance validation
- Test cooldown enforcement
- Test emperor clothes blocking
- Test duel state management

### Property-Based Tests
- Use rapid library for Go
- Minimum 100 iterations per property
- Test transfer amount calculations
- Test dice roll distribution
- Test win/lose conditions

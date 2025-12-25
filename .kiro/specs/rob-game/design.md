# Design Document: Rob Game (打劫游戏)

## Overview

打劫游戏是一个玩家互动游戏，允许用户通过 `/dajie` 命令打劫其他用户的金币。游戏包含保护机制防止玩家被过度打劫。

## Architecture

打劫游戏作为新模块集成到现有 Go Bot 架构中：

```
telegram-game-bot/
├── internal/
│   ├── game/
│   │   └── rob/
│   │       ├── rob.go           # 打劫游戏逻辑
│   │       ├── protection.go    # 保护期管理
│   │       └── rob_test.go      # 测试
│   └── handler/
│       └── game.go              # 添加 /dajie handler
```

## Components and Interfaces

### RobGame 结构

```go
// RobGame 打劫游戏
type RobGame struct {
    userRepo    *repository.UserRepository
    txRepo      *repository.TransactionRepository
    userLock    *lock.UserLock
    
    // 保护期状态 (内存存储)
    protection  map[int64]*ProtectionState  // victim_id -> state
    cooldowns   map[int64]time.Time         // robber_id -> last_rob_time
    mu          sync.RWMutex
}

// ProtectionState 保护期状态
type ProtectionState struct {
    ConsecutiveCount int       // 连续被打劫次数
    ProtectedUntil   time.Time // 保护期结束时间
}

// RobResult 打劫结果
type RobResult struct {
    Success     bool
    Amount      int64
    RobberName  string
    VictimName  string
    Message     string
}
```

### 核心方法

```go
// Rob 执行打劫
func (g *RobGame) Rob(ctx context.Context, robberID, victimID int64) (*RobResult, error)

// CanRob 检查是否可以打劫
func (g *RobGame) CanRob(robberID, victimID int64) (bool, string)

// IsProtected 检查用户是否在保护期
func (g *RobGame) IsProtected(userID int64) (bool, time.Duration)

// GetCooldown 获取冷却剩余时间
func (g *RobGame) GetCooldown(robberID int64) time.Duration

// generateAmount 生成随机打劫金额 (10-1000)
func (g *RobGame) generateAmount() int64
```

## Data Models

### 内存状态（无需数据库表）

保护期和冷却状态存储在内存中，重启后重置。

### 交易类型

```go
const (
    TxTypeRob    = "rob"     // 打劫者获得
    TxTypeRobbed = "robbed"  // 被打劫者损失
)
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system.*

### Property 1: Robbery Validation

*For any* robbery attempt:
- If robber_id == victim_id, robbery SHALL fail
- If victim is not registered, robbery SHALL fail

**Validates: Requirements 1.3, 1.4**

### Property 2: Robbery Amount Calculation

*For any* successful robbery:
- Generated amount SHALL be in range [10, 1000]
- Actual stolen amount = min(generated_amount, victim.balance)
- robber.balance_after = robber.balance_before + actual_amount
- victim.balance_after = victim.balance_before - actual_amount

**Validates: Requirements 2.1, 2.2, 2.3**

### Property 3: Protection Mechanism

*For any* user:
- After being robbed 3 times consecutively, protection SHALL activate for 30 minutes
- While protected, all robbery attempts against them SHALL fail
- After protection expires, consecutive count SHALL reset to 0

**Validates: Requirements 3.1, 3.2, 3.3, 3.5**

### Property 4: Cooldown Enforcement

*For any* robber:
- If last_rob_time < 21 seconds ago, robbery SHALL fail
- Cooldown only applies to robber, not victim
- Being robbed does not affect robber's ability to rob others

**Validates: Requirements 4.1, 4.3**

### Property 5: Transaction Recording

*For any* successful robbery of amount A:
- Transaction with type "rob" and amount +A SHALL exist for robber
- Transaction with type "robbed" and amount -A SHALL exist for victim
- Both transactions SHALL be included in daily rankings

**Validates: Requirements 5.1, 5.2, 5.3**

## Error Handling

```go
var (
    ErrSelfRob           = errors.New("不能打劫自己")
    ErrVictimNotFound    = errors.New("目标用户未注册")
    ErrVictimProtected   = errors.New("目标用户在保护期")
    ErrCooldown          = errors.New("打劫冷却中")
    ErrInsufficientFunds = errors.New("目标用户余额不足")
)
```

## Testing Strategy

### Unit Tests
- 金额生成范围测试
- 保护期激活/过期测试
- 冷却时间测试

### Property-Based Tests
使用 rapid 库：

```go
func TestRobAmountProperty(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        amount := generateAmount()
        assert.GreaterOrEqual(t, amount, int64(10))
        assert.LessOrEqual(t, amount, int64(1000))
    })
}
```

## Configuration

```yaml
games:
  rob:
    min_amount: 10
    max_amount: 1000
    cooldown_seconds: 21
    protection_threshold: 3
    protection_duration_minutes: 30
```

# Design Document: Go Telegram Game Bot

## Overview

使用 Go 语言完全重写 Telegram 游戏机器人，采用模块化、可扩展的架构设计。核心目标是高性能、易维护、易扩展。

### 架构图

```
┌─────────────────────────────────────────────────────────┐
│                    Telegram Bot                          │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │   Handler   │  │   Handler   │  │   Handler   │     │
│  │   (dice)    │  │   (slot)    │  │   (sicbo)   │ ... │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘     │
│         │                │                │             │
│         ▼                ▼                ▼             │
│  ┌─────────────────────────────────────────────────┐   │
│  │              Game Engine (Interface)             │   │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐           │   │
│  │  │  Dice   │ │  Slot   │ │  SicBo  │  + Future │   │
│  │  │ Game    │ │  Game   │ │  Game   │    Games  │   │
│  │  └─────────┘ └─────────┘ └─────────┘           │   │
│  └─────────────────────────────────────────────────┘   │
│                          │                              │
│                          ▼                              │
│  ┌─────────────────────────────────────────────────┐   │
│  │              Service Layer                       │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐        │   │
│  │  │ Account  │ │ Transfer │ │ Ranking  │        │   │
│  │  │ Service  │ │ Service  │ │ Service  │        │   │
│  │  └──────────┘ └──────────┘ └──────────┘        │   │
│  └─────────────────────────────────────────────────┘   │
│                          │                              │
│                          ▼                              │
│  ┌─────────────────────────────────────────────────┐   │
│  │              Repository Layer                    │   │
│  │  ┌──────────┐ ┌──────────┐                      │   │
│  │  │   User   │ │Transaction│                      │   │
│  │  │   Repo   │ │   Repo   │                      │   │
│  │  └──────────┘ └──────────┘                      │   │
│  └─────────────────────────────────────────────────┘   │
│                          │                              │
└──────────────────────────┼──────────────────────────────┘
                           ▼
                  ┌─────────────────┐
                  │   PostgreSQL    │
                  └─────────────────┘
```

## Architecture

### 技术栈

| 组件 | 技术选型 | 理由 |
|------|----------|------|
| Language | Go 1.21+ | 高性能、并发友好 |
| Bot Framework | telebot/v3 | 成熟、API 完整 |
| Database | PostgreSQL 15+ | 高并发、ACID |
| DB Driver | pgx/v5 | 高性能 PostgreSQL 驱动 |
| Migration | golang-migrate | 标准迁移工具 |
| Config | viper | 灵活的配置管理 |
| Logging | zerolog | 高性能结构化日志 |
| Testing | testify + rapid | 单元测试 + PBT |

### 项目结构

```
telegram-game-bot/
├── cmd/
│   └── bot/
│       └── main.go              # 入口
├── internal/
│   ├── config/
│   │   └── config.go            # 配置管理
│   ├── bot/
│   │   ├── bot.go               # Bot 初始化
│   │   └── middleware.go        # 中间件（白名单、权限）
│   ├── handler/
│   │   ├── account.go           # 账户命令
│   │   ├── transfer.go          # 转账命令
│   │   ├── admin.go             # 管理员命令
│   │   ├── ranking.go           # 排行榜命令
│   │   └── game.go              # 游戏命令入口
│   ├── game/
│   │   ├── interface.go         # Game 接口定义
│   │   ├── registry.go          # 游戏注册表
│   │   ├── dice/
│   │   │   └── dice.go          # 骰子游戏
│   │   ├── slot/
│   │   │   └── slot.go          # 老虎机游戏
│   │   └── sicbo/
│   │       ├── sicbo.go         # 骰宝游戏
│   │       ├── calculator.go    # 赔率计算
│   │       └── keyboard.go      # 键盘生成
│   ├── service/
│   │   ├── account.go           # 账户服务
│   │   ├── transfer.go          # 转账服务
│   │   └── ranking.go           # 排行榜服务
│   ├── repository/
│   │   ├── user.go              # 用户仓储
│   │   └── transaction.go       # 交易仓储
│   ├── model/
│   │   └── models.go            # 数据模型
│   └── pkg/
│       ├── lock/
│       │   └── lock.go          # 用户锁
│       └── db/
│           └── postgres.go      # 数据库连接池
├── migrations/
│   ├── 001_create_users.up.sql
│   ├── 001_create_users.down.sql
│   ├── 002_create_transactions.up.sql
│   └── 002_create_transactions.down.sql
├── config/
│   └── config.yaml              # 配置文件
├── go.mod
├── go.sum
├── Dockerfile
└── docker-compose.yml
```

## Components and Interfaces

### Game Interface (核心扩展点)

```go
// Game 定义游戏接口，所有游戏必须实现
type Game interface {
    // Name 返回游戏名称（用于注册和日志）
    Name() string
    
    // Command 返回触发命令（如 "dice", "slot"）
    Command() string
    
    // Description 返回游戏描述
    Description() string
    
    // Play 执行游戏逻辑
    // ctx: 上下文
    // userID: 用户 ID
    // bet: 下注金额
    // params: 额外参数（如骰子点数）
    // 返回: 赔付金额（正数赢，负数输，0平局）
    Play(ctx context.Context, userID int64, bet int64, params map[string]any) (payout int64, err error)
    
    // ValidateBet 验证下注参数
    ValidateBet(bet int64, params map[string]any) error
}

// MultiPlayerGame 多人游戏接口（如骰宝）
type MultiPlayerGame interface {
    Game
    
    // StartSession 开始游戏会话
    StartSession(ctx context.Context, chatID int64) error
    
    // PlaceBet 下注
    PlaceBet(ctx context.Context, chatID, userID int64, betType string, amount int64) error
    
    // Settle 结算
    Settle(ctx context.Context, chatID int64) (results map[int64]int64, err error)
}
```

### 游戏注册表

```go
// Registry 游戏注册表
type Registry struct {
    games map[string]Game
    mu    sync.RWMutex
}

// Register 注册游戏
func (r *Registry) Register(g Game) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.games[g.Command()] = g
}

// Get 获取游戏
func (r *Registry) Get(command string) (Game, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    g, ok := r.games[command]
    return g, ok
}
```

### Service Interfaces

```go
// AccountService 账户服务接口
type AccountService interface {
    EnsureUser(ctx context.Context, telegramID int64, username string) (*model.User, error)
    GetBalance(ctx context.Context, userID int64) (int64, error)
    UpdateBalance(ctx context.Context, userID int64, amount int64) error
    ClaimDaily(ctx context.Context, userID int64) (bool, string, error)
}

// TransferService 转账服务接口
type TransferService interface {
    Transfer(ctx context.Context, fromID, toID int64, amount int64) error
}

// RankingService 排行榜服务接口
type RankingService interface {
    GetTopUsers(ctx context.Context, limit int) ([]*model.User, error)
    GetDailyWinners(ctx context.Context, limit int) ([]*model.DailyRank, error)
    GetDailyLosers(ctx context.Context, limit int) ([]*model.DailyRank, error)
}
```

## Data Models

### PostgreSQL Schema

```sql
-- users 表
CREATE TABLE users (
    telegram_id BIGINT PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    balance BIGINT NOT NULL DEFAULT 1000,
    last_daily_claim BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_balance ON users(balance DESC);

-- transactions 表
CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(telegram_id),
    amount BIGINT NOT NULL,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_user_time ON transactions(user_id, created_at DESC);
CREATE INDEX idx_transactions_type_time ON transactions(type, created_at DESC);

-- 每日统计视图（用于排行榜）
CREATE VIEW daily_game_stats AS
SELECT 
    user_id,
    SUM(amount) as net_profit,
    DATE(created_at) as game_date
FROM transactions
WHERE type IN ('dice', 'slot', 'sicbo_win', 'sicbo_bet')
GROUP BY user_id, DATE(created_at);
```

### Go Models

```go
type User struct {
    TelegramID     int64     `db:"telegram_id"`
    Username       string    `db:"username"`
    Balance        int64     `db:"balance"`
    LastDailyClaim int64     `db:"last_daily_claim"`
    CreatedAt      time.Time `db:"created_at"`
    UpdatedAt      time.Time `db:"updated_at"`
}

type Transaction struct {
    ID          int64     `db:"id"`
    UserID      int64     `db:"user_id"`
    Amount      int64     `db:"amount"`
    Type        string    `db:"type"`
    Description *string   `db:"description"`
    CreatedAt   time.Time `db:"created_at"`
}

type DailyRank struct {
    UserID    int64  `db:"user_id"`
    Username  string `db:"username"`
    NetProfit int64  `db:"net_profit"`
}
```


## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: New User Initial Balance

*For any* new user registration, the created account SHALL have exactly 1000 coins as initial balance.

**Validates: Requirements 1.1**

### Property 2: Daily Claim Eligibility

*For any* user:
- If last_daily_claim is 0 OR (current_time - last_daily_claim) >= 24 hours, claim SHALL succeed and add 500 coins
- If (current_time - last_daily_claim) < 24 hours, claim SHALL fail

**Validates: Requirements 1.3, 1.4**

### Property 3: Top Users Ordering

*For any* set of users, GetTopUsers SHALL return users sorted by balance in descending order.

**Validates: Requirements 1.5**

### Property 4: Transfer Conservation

*For any* successful transfer of amount A from user X to user Y:
- X.balance_after = X.balance_before - A
- Y.balance_after = Y.balance_before + A
- Total system balance remains unchanged

**Validates: Requirements 2.1**

### Property 5: Transfer Validation

*For any* transfer request:
- If amount <= 0, transfer SHALL fail
- If sender.balance < amount, transfer SHALL fail
- If sender_id == receiver_id, transfer SHALL fail

**Validates: Requirements 2.2, 2.3, 2.4**

### Property 6: Dice Payout Calculation

*For any* dice game with dice values d1, d2 ∈ [1,6] and bet B:
- total ∈ [2,6]: payout = -B (lose)
- total = 7: payout = 0 (push)
- total ∈ [8,11]: payout = B (win)
- total = 12: payout = 2*B (jackpot)

**Validates: Requirements 3.2**

### Property 7: Slot Decode Correctness

*For any* slot value V ∈ [1,64]:
- DecodeSlot(V) produces (left, middle, right) where each ∈ [1,4]
- EncodeSlot(left, middle, right) = V (round-trip)

**Validates: Requirements 4.4**

### Property 8: Slot Payout Calculation

*For any* slot result and bet B:
- If left == middle == right: payout > 0 (tiered by bet amount)
- If exactly 2 symbols match: payout = 0
- If no symbols match: payout = -B

**Validates: Requirements 4.2**

### Property 9: SicBo Payout Calculation

*For any* dice result [d1, d2, d3] where each di ∈ [1,6] and fixed bet amount 100:
- Single number N: payout = 100 * count(N) if count > 0, else -100
- Big bet: payout = 100 if sum ∈ [11,17] AND not triple, else -100
- Small bet: payout = 100 if sum ∈ [4,10] AND not triple, else -100
- Triple detection: is_triple = (d1 == d2 == d3)

**Validates: Requirements 5.3, 5.4, 5.5**

### Property 10: SicBo Bet Accumulation

*For any* user placing multiple bets on the same option in the same game session, the total bet amount SHALL be the sum of all individual bets.

**Validates: Requirements 5.7**

### Property 11: Admin Permission Check

*For any* admin command execution:
- If user_id NOT IN admin_ids, command SHALL fail with permission error
- If user_id IN admin_ids, command SHALL execute

**Validates: Requirements 6.4**

### Property 12: Whitelist Enforcement

*For any* command in a group chat:
- If chat_id NOT IN allowed_chats, command SHALL be ignored
- If chat_id IN allowed_chats, command SHALL be processed

**Validates: Requirements 7.1**

### Property 13: Concurrent Balance Safety

*For any* concurrent balance operations on the same user, the final balance SHALL be consistent with sequential execution of all operations.

**Validates: Requirements 9.1, 9.3**

### Property 14: Daily Ranking Calculation

*For any* day's game transactions:
- Daily net profit = SUM(amount) for game-type transactions only
- Winners = users with positive net profit, sorted descending
- Losers = users with negative net profit, sorted ascending (most loss first)
- Only transaction types: dice, slot, sicbo_win, sicbo_bet are counted

**Validates: Requirements 11.2, 11.3, 11.5**

## Error Handling

### Error Types

```go
var (
    ErrInsufficientBalance = errors.New("insufficient balance")
    ErrInvalidAmount       = errors.New("invalid amount")
    ErrSelfTransfer        = errors.New("cannot transfer to self")
    ErrUserNotFound        = errors.New("user not found")
    ErrDailyAlreadyClaimed = errors.New("daily reward already claimed")
    ErrGameInProgress      = errors.New("game already in progress")
    ErrBetTooHigh          = errors.New("bet exceeds maximum")
    ErrNotAdmin            = errors.New("admin permission required")
    ErrChatNotAllowed      = errors.New("chat not in whitelist")
)
```

## Testing Strategy

### Unit Tests
- 每个 calculator 函数的边界值测试
- Service 层业务逻辑测试
- Repository 层 SQL 测试（使用 testcontainers）

### Property-Based Tests
使用 `pgregory.net/rapid` 库实现：

```go
func TestDicePayoutProperty(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        d1 := rapid.IntRange(1, 6).Draw(t, "d1")
        d2 := rapid.IntRange(1, 6).Draw(t, "d2")
        bet := rapid.Int64Range(1, 1000).Draw(t, "bet")
        
        payout := CalculateDicePayout(d1, d2, bet)
        total := d1 + d2
        
        switch {
        case total <= 6:
            assert.Equal(t, -bet, payout)
        case total == 7:
            assert.Equal(t, int64(0), payout)
        case total <= 11:
            assert.Equal(t, bet, payout)
        case total == 12:
            assert.Equal(t, bet*2, payout)
        }
    })
}
```

### Integration Tests
- Bot 命令端到端测试
- 数据库事务测试
- 并发安全测试

## Configuration

```yaml
# config/config.yaml
bot:
  token: "${BOT_TOKEN}"
  
database:
  host: localhost
  port: 5432
  user: gamebot
  password: "${DB_PASSWORD}"
  name: gamebot
  pool_size: 20

admin:
  ids:
    - 123456789
    - 987654321

whitelist:
  chats:
    - -1001234567890

daily:
  reward: 500
  cooldown_hours: 24

games:
  dice:
    max_bet: 1000
    cooldown_seconds: 3
  slot:
    cooldown_seconds: 5
  sicbo:
    betting_duration_seconds: 60
```

# Design Document: Shop System

## Overview

商店系统为打劫游戏提供道具购买功能。玩家通过私聊 bot 访问商店界面，使用按钮交互购买道具。道具分为一次性使用（手铐）和时效性（保护罩、双刃剑、饮血剑）两类，购买后存储在用户背包中，并在打劫游戏中生效。

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Telegram Bot                            │
├─────────────────────────────────────────────────────────────┤
│  /start (private) → ShopHandler                              │
│  /handcuff        → HandcuffHandler                          │
│  /bag             → InventoryHandler                         │
│  Callbacks        → ShopCallbackHandler                      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Shop Service                            │
├─────────────────────────────────────────────────────────────┤
│  - GetShopItems()                                            │
│  - PurchaseItem(userID, itemType)                           │
│  - UseHandcuff(userID, targetID)                            │
│  - GetUserInventory(userID)                                 │
│  - HasActiveEffect(userID, effectType)                      │
│  - GetActiveEffects(userID)                                 │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Inventory Repository                       │
├─────────────────────────────────────────────────────────────┤
│  - user_items table (handcuff count)                        │
│  - user_effects table (time-based effects)                  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Rob Game Integration                    │
├─────────────────────────────────────────────────────────────┤
│  - Check handcuff lock before robbery                       │
│  - Check shield before being robbed                         │
│  - Apply double edge sword effect                           │
│  - Apply bloodthirst sword success rate                     │
└─────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. Item Types

```go
type ItemType string

const (
    ItemHandcuff        ItemType = "handcuff"        // 手铐
    ItemShield          ItemType = "shield"          // 保护罩
    ItemThornArmor      ItemType = "thorn_armor"    // 荆棘刺甲
    ItemBloodthirstSword ItemType = "bloodthirst"    // 饮血剑
)

type ItemConfig struct {
    Type        ItemType
    Name        string  // 显示名称
    Emoji       string  // 图标
    Price       int64   // 价格
    Duration    time.Duration // 时效（0表示一次性）
    Description string  // 描述
}

var ShopItems = map[ItemType]ItemConfig{
    ItemHandcuff: {
        Type:        ItemHandcuff,
        Name:        "手铐",
        Emoji:       "🔗",
        Price:       500,
        Duration:    0, // 一次性
        Description: "锁定目标30分钟，使其无法打劫",
    },
    ItemShield: {
        Type:        ItemShield,
        Name:        "保护罩",
        Emoji:       "🛡️",
        Price:       500,
        Duration:    6 * time.Hour,
        Description: "6小时内无法被打劫",
    },
    ItemThornArmor: {
        Type:        ItemThornArmor,
        Name:        "荆棘刺甲",
        Emoji:       "🌵",
        Price:       500,
        Duration:    3 * time.Hour,
        Description: "3小时内被打劫成功时，攻击方扣双倍",
    },
    ItemBloodthirstSword: {
        Type:        ItemBloodthirstSword,
        Name:        "饮血剑",
        Emoji:       "🗡️",
        Price:       1000,
        Duration:    30 * time.Minute,
        Description: "30分钟内打劫成功率提升到80%",
    },
}
```

### 2. Shop Service Interface

```go
type ShopService interface {
    // 获取商店商品列表
    GetShopItems() []ItemConfig
    
    // 购买道具
    PurchaseItem(ctx context.Context, userID int64, itemType ItemType) error
    
    // 使用手铐
    UseHandcuff(ctx context.Context, userID, targetID int64) error
    
    // 获取用户背包
    GetUserInventory(ctx context.Context, userID int64) (*UserInventory, error)
    
    // 检查用户是否有某个效果
    HasActiveEffect(ctx context.Context, userID int64, effectType ItemType) bool
    
    // 检查用户是否被手铐锁定
    IsHandcuffed(ctx context.Context, userID int64) (bool, time.Duration)
    
    // 检查用户是否有保护罩
    HasShield(ctx context.Context, userID int64) bool
    
    // 检查用户是否有荆棘刺甲
    HasThornArmor(ctx context.Context, userID int64) bool
    
    // 检查用户是否有饮血剑
    HasBloodthirstSword(ctx context.Context, userID int64) bool
}
```

### 3. Shop Handler

```go
// HandleShopStart 处理私聊 /start 显示商店
func (h *ShopHandler) HandleShopStart(c tele.Context) error

// HandleShopCallback 处理商店按钮回调
func (h *ShopHandler) HandleShopCallback(c tele.Context) error

// HandleHandcuff 处理 /handcuff 命令
func (h *ShopHandler) HandleHandcuff(c tele.Context) error

// HandleBag 处理 /bag 命令
func (h *ShopHandler) HandleBag(c tele.Context) error
```

### 4. Keyboard Builder

```go
// BuildShopPanel 构建商店主面板
func BuildShopPanel() *tele.ReplyMarkup

// BuildConfirmPanel 构建购买确认面板
func BuildConfirmPanel(itemType ItemType) *tele.ReplyMarkup
```

## Data Models

### Database Schema

```sql
-- 用户道具表（存储手铐数量）
CREATE TABLE IF NOT EXISTS user_items (
    user_id BIGINT NOT NULL,
    item_type VARCHAR(50) NOT NULL,
    quantity INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, item_type)
);

-- 用户效果表（存储时效性道具）
CREATE TABLE IF NOT EXISTS user_effects (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    effect_type VARCHAR(50) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_user_effects_user ON user_effects(user_id);
CREATE INDEX idx_user_effects_expires ON user_effects(expires_at);

-- 手铐锁定表（存储被锁定的用户）
CREATE TABLE IF NOT EXISTS handcuff_locks (
    target_id BIGINT PRIMARY KEY,
    locked_by BIGINT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_handcuff_locks_expires ON handcuff_locks(expires_at);
```

### Go Models

```go
// UserItem 用户道具（手铐）
type UserItem struct {
    UserID    int64
    ItemType  string
    Quantity  int
    UpdatedAt time.Time
}

// UserEffect 用户效果（时效性道具）
type UserEffect struct {
    ID         int64
    UserID     int64
    EffectType string
    ExpiresAt  time.Time
    CreatedAt  time.Time
}

// HandcuffLock 手铐锁定
type HandcuffLock struct {
    TargetID  int64
    LockedBy  int64
    ExpiresAt time.Time
    CreatedAt time.Time
}

// UserInventory 用户背包
type UserInventory struct {
    HandcuffCount int
    Effects       []UserEffect
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Purchase Transaction Integrity

*For any* item purchase attempt, if the user has sufficient balance, the balance should decrease by exactly the item price and the item should be added to inventory; if insufficient balance, no state change should occur.

**Validates: Requirements 1.4, 1.5, 1.6**

### Property 2: Handcuff Consumption

*For any* successful handcuff use, the user's handcuff count should decrease by exactly 1, and the target should be locked for 30 minutes.

**Validates: Requirements 2.2, 2.7**

### Property 3: Item Activation and Expiration

*For any* time-based item purchase (shield, double edge sword, bloodthirst sword), the effect should be active immediately after purchase and should expire exactly at the configured duration.

**Validates: Requirements 3.3, 3.6, 4.3, 4.5, 5.3, 5.5**

### Property 4: Shield Protection Effect

*For any* robbery attempt against a user with active shield, the robbery should fail with a protection message.

**Validates: Requirements 3.4**

### Property 5: Thorn Armor Effect

*For any* successful robbery against a user with active thorn armor, the attacker should lose exactly double the robbery amount.

**Validates: Requirements 4.4**

### Property 6: Bloodthirst Sword Success Rate

*For any* robbery attempt by a user with active bloodthirst sword, the success rate should be 80% (instead of default 50%).

**Validates: Requirements 5.4**

### Property 7: Handcuff Lock Effect

*For any* robbery attempt by a user who is handcuff-locked, the robbery should fail with a lock message.

**Validates: Requirements 2.4**

### Property 8: Inventory Stacking

*For any* user with multiple items, all items should be stored correctly and all active effects should apply simultaneously.

**Validates: Requirements 6.1, 6.2, 6.3, 6.4**

## Error Handling

| Error | Condition | Response |
|-------|-----------|----------|
| ErrInsufficientBalance | Balance < item price | "❌ 余额不足，需要 X 金币" |
| ErrNoHandcuff | Use /handcuff without item | Silent ignore (no response) |
| ErrTargetNotFound | Handcuff target not found | "❌ 目标用户未找到" |
| ErrSelfHandcuff | Handcuff self | "❌ 不能对自己使用手铐" |
| ErrAlreadyLocked | Target already locked | "❌ 目标已被锁定" |

## Testing Strategy

### Unit Tests
- Item configuration validation
- Price and duration constants
- Keyboard builder output

### Property-Based Tests
- Purchase transaction integrity (balance changes)
- Effect activation and expiration timing
- Handcuff consumption and lock duration
- Effect stacking behavior

### Integration Tests
- Full purchase flow with database
- Rob game integration with item effects
- Concurrent purchase handling

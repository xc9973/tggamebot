# Design Document: Shop System

## Overview

商店系统为打劫游戏提供道具购买功能。玩家通过私聊 bot 访问商店界面，使用按钮交互购买道具。所有道具都是次数限制型（用完即失效），购买后存储在用户背包中，并在打劫游戏中生效。系统支持每日购买限制。

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
│  - DecrementUseCount(userID, effectType)                    │
│  - CheckDailyLimit(userID, itemType)                        │
│  - RemoveDefensiveItems(userID)                             │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Inventory Repository                       │
├─────────────────────────────────────────────────────────────┤
│  - user_items table (item use counts)                       │
│  - daily_purchases table (daily purchase tracking)          │
│  - handcuff_locks table (lock status)                       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Rob Game Integration                    │
├─────────────────────────────────────────────────────────────┤
│  - Check handcuff lock before robbery                       │
│  - Check Emperor_Clothes first (highest priority)           │
│  - Check Shield (can be bypassed by Blunt_Knife/Great_Sword)│
│  - Apply Thorn_Armor effect (can be bypassed)               │
│  - Apply Bloodthirst_Sword success rate                     │
│  - Apply Blunt_Knife limited amount                         │
│  - Apply Great_Sword critical hit                           │
│  - Apply Golden_Cassock defense removal                     │
└─────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. Item Types

```go
type ItemType string

const (
    ItemHandcuff         ItemType = "handcuff"         // 手铐
    ItemShield           ItemType = "shield"           // 保护罩
    ItemThornArmor       ItemType = "thorn_armor"      // 荆棘刺甲
    ItemBloodthirstSword ItemType = "bloodthirst"      // 饮血剑
    ItemBluntKnife       ItemType = "blunt_knife"      // 钝刀
    ItemGreatSword       ItemType = "great_sword"      // 大宝剑
    ItemGoldenCassock    ItemType = "golden_cassock"   // 紫金袈裟
    ItemEmperorClothes   ItemType = "emperor_clothes"  // 皇帝的新衣
)

type ItemConfig struct {
    Type          ItemType     // 道具类型
    Name          string       // 显示名称
    Emoji         string       // 图标
    Price         int64        // 价格
    UseCount      int          // 使用次数
    Description   string       // 描述
    Category      ItemCategory // 分类
    DailyLimit    int          // 每日购买限制（0表示无限制）
    BypassDefense bool         // 是否无视普通防御（保护罩、荆棘刺甲）
    ImmuneBypass  bool         // 是否免疫无视防御攻击
}

var ShopItems = map[ItemType]ItemConfig{
    ItemHandcuff: {
        Type:        ItemHandcuff,
        Name:        "手铐",
        Emoji:       "🔗",
        Price:       500,
        UseCount:    1,
        Description: "锁定目标30分钟，使其无法打劫",
        Category:    CategoryAttack,
        DailyLimit:  5,
    },
    ItemShield: {
        Type:        ItemShield,
        Name:        "保护罩",
        Emoji:       "🛡️",
        Price:       500,
        UseCount:    10,
        Description: "防止被打劫10次",
        Category:    CategoryDefense,
        DailyLimit:  2,
    },
    ItemThornArmor: {
        Type:        ItemThornArmor,
        Name:        "荆棘刺甲",
        Emoji:       "🌵",
        Price:       500,
        UseCount:    5,
        Description: "被打劫成功时攻击方扣双倍（5次）",
        Category:    CategoryPassive,
    },
    ItemBloodthirstSword: {
        Type:        ItemBloodthirstSword,
        Name:        "饮血剑",
        Emoji:       "🗡️",
        Price:       1000,
        UseCount:    10,
        Description: "打劫成功率提升到80%（10次）",
        Category:    CategoryAttack,
    },
    ItemBluntKnife: {
        Type:          ItemBluntKnife,
        Name:          "钝刀",
        Emoji:         "🔪",
        Price:         1000,
        UseCount:      10,
        Description:   "无视防御，打劫1-100随机（10次）",
        Category:      CategoryAttack,
        BypassDefense: true,
    },
    ItemGreatSword: {
        Type:          ItemGreatSword,
        Name:          "大宝剑",
        Emoji:         "⚔️",
        Price:         10000,
        UseCount:      3,
        Description:   "无视防御，0.01%打劫90%（3次）",
        Category:      CategoryAttack,
        DailyLimit:    1,
        BypassDefense: true,
    },
    ItemGoldenCassock: {
        Type:        ItemGoldenCassock,
        Name:        "紫金袈裟",
        Emoji:       "👘",
        Price:       10000,
        UseCount:    3,
        Description: "攻击者失去所有防御道具（3次）",
        Category:    CategoryDefense,
    },
    ItemEmperorClothes: {
        Type:         ItemEmperorClothes,
        Name:         "皇帝的新衣",
        Emoji:        "👑",
        Price:        5000,
        UseCount:     3,
        Description:  "免疫所有攻击（3次）",
        Category:     CategoryDefense,
        ImmuneBypass: true,
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
    
    // 检查每日购买限制
    CheckDailyLimit(ctx context.Context, userID int64, itemType ItemType) (bool, int, error)
    
    // 使用手铐
    UseHandcuff(ctx context.Context, userID, targetID int64) error
    
    // 获取用户背包
    GetUserInventory(ctx context.Context, userID int64) (*UserInventory, error)
    
    // 检查用户是否有某个效果（返回剩余次数）
    GetEffectUseCount(ctx context.Context, userID int64, effectType ItemType) (int, error)
    
    // 减少道具使用次数
    DecrementUseCount(ctx context.Context, userID int64, effectType ItemType) error
    
    // 检查用户是否被手铐锁定
    IsHandcuffed(ctx context.Context, userID int64) (bool, time.Duration)
    
    // 检查用户是否有皇帝的新衣（最高优先级防御）
    HasEmperorClothes(ctx context.Context, userID int64) bool
    
    // 检查用户是否有保护罩
    HasShield(ctx context.Context, userID int64) bool
    
    // 检查用户是否有荆棘刺甲
    HasThornArmor(ctx context.Context, userID int64) bool
    
    // 检查用户是否有饮血剑
    HasBloodthirstSword(ctx context.Context, userID int64) bool
    
    // 检查用户是否有钝刀
    HasBluntKnife(ctx context.Context, userID int64) bool
    
    // 检查用户是否有大宝剑
    HasGreatSword(ctx context.Context, userID int64) bool
    
    // 检查用户是否有紫金袈裟
    HasGoldenCassock(ctx context.Context, userID int64) bool
    
    // 移除用户的防御道具（被紫金袈裟触发）
    RemoveDefensiveItems(ctx context.Context, userID int64) error
}
```

## Data Models

### Database Schema

```sql
-- 用户道具表（存储道具剩余使用次数）
CREATE TABLE IF NOT EXISTS user_items (
    user_id BIGINT NOT NULL,
    item_type VARCHAR(50) NOT NULL,
    use_count INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, item_type)
);

-- 手铐锁定表（存储被锁定的用户）
CREATE TABLE IF NOT EXISTS handcuff_locks (
    target_id BIGINT PRIMARY KEY,
    locked_by BIGINT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_handcuff_locks_expires ON handcuff_locks(expires_at);

-- 每日购买记录表
CREATE TABLE IF NOT EXISTS daily_purchases (
    user_id BIGINT NOT NULL,
    item_type VARCHAR(50) NOT NULL,
    purchase_count INT NOT NULL DEFAULT 0,
    purchase_date DATE NOT NULL DEFAULT CURRENT_DATE,
    PRIMARY KEY (user_id, item_type, purchase_date)
);
CREATE INDEX idx_daily_purchases_date ON daily_purchases(purchase_date);
```

### Go Models

```go
// UserItem 用户道具
type UserItem struct {
    UserID    int64
    ItemType  string
    UseCount  int
    UpdatedAt time.Time
}

// HandcuffLock 手铐锁定
type HandcuffLock struct {
    TargetID  int64
    LockedBy  int64
    ExpiresAt time.Time
    CreatedAt time.Time
}

// DailyPurchase 每日购买记录
type DailyPurchase struct {
    UserID        int64
    ItemType      string
    PurchaseCount int
    PurchaseDate  time.Time
}

// UserInventory 用户背包
type UserInventory struct {
    Items []UserItem
}
```

## Defense Priority

打劫时防御检查顺序（从高到低）：

1. **皇帝的新衣** - 最高优先级，免疫所有攻击（包括钝刀、大宝剑）
2. **保护罩** - 普通防御，可被钝刀、大宝剑无视
3. **荆棘刺甲** - 被动反伤，可被钝刀、大宝剑无视

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system.*

### Property 1: Purchase Transaction Integrity

*For any* item purchase attempt, if the user has sufficient balance and has not exceeded daily limit, the balance should decrease by exactly the item price and the item use count should be added to inventory; otherwise, no state change should occur.

**Validates: Requirements 1.4, 1.5, 1.6**

### Property 2: Daily Purchase Limit Enforcement

*For any* item with a daily limit (handcuff=5, shield=2, great_sword=1), after reaching the limit, all subsequent purchase attempts on the same day should be rejected without state change.

**Validates: Requirements 2.3, 2.9, 3.3, 3.8, 7.3, 7.8, 12.1, 12.3, 12.4**

### Property 3: Use Count Decrement

*For any* item use, the use count should decrease by exactly 1. When use count reaches 0, the item effect should be removed.

**Validates: Requirements 3.6, 3.7, 4.4, 4.5, 5.4, 5.5, 6.5, 6.6, 7.6, 7.7, 8.4, 8.5, 9.5, 9.6**

### Property 4: Emperor Clothes Immunity

*For any* robbery attempt against a user with active Emperor_Clothes, the robbery should fail regardless of attacker's items (including Blunt_Knife and Great_Sword).

**Validates: Requirements 9.4, 9.5**

### Property 5: Defense Bypass Items

*For any* robbery attempt by a user with active Blunt_Knife or Great_Sword against a user WITHOUT Emperor_Clothes, the target's Shield and Thorn_Armor effects should be ignored.

**Validates: Requirements 6.4, 7.5**

### Property 6: Blunt Knife Amount Limit

*For any* robbery with active Blunt_Knife, the robbery amount should be a random value in the range [1, 100] coins.

**Validates: Requirements 6.5**

### Property 7: Great Sword Critical Hit

*For any* robbery with active Great_Sword, there should be a 0.01% probability to rob 90% of target's coins.

**Validates: Requirements 7.6**

### Property 8: Golden Cassock Defense Removal

*For any* robbery attempt against a user with active Golden_Cassock, all defensive items (Shield, Thorn_Armor) should be removed from the attacker.

**Validates: Requirements 8.4**

### Property 9: Thorn Armor Double Damage

*For any* successful robbery against a user with active Thorn_Armor (and attacker has no defense bypass), the attacker should lose exactly double the robbery amount.

**Validates: Requirements 4.4**

### Property 10: Handcuff Lock Effect

*For any* robbery attempt by a user who is handcuff-locked, the robbery should fail with a lock message.

**Validates: Requirements 2.5, 2.6**

## Error Handling

| Error | Condition | Response |
|-------|-----------|----------|
| ErrInsufficientBalance | Balance < item price | "❌ 余额不足，需要 X 金币" |
| ErrDailyLimitReached | Daily purchase limit exceeded | "❌ 今日购买次数已达上限" |
| ErrNoHandcuff | Use /handcuff without item | Silent ignore (no response) |
| ErrTargetNotFound | Handcuff target not found | "❌ 目标用户未找到" |
| ErrSelfHandcuff | Handcuff self | "❌ 不能对自己使用手铐" |
| ErrAlreadyLocked | Target already locked | "❌ 目标已被锁定" |

## Testing Strategy

### Unit Tests
- Item configuration validation (8 items)
- Price and use count constants
- Keyboard builder output (8 item buttons)
- Defense bypass and immunity flags

### Property-Based Tests
- Purchase transaction integrity
- Daily purchase limit enforcement
- Use count decrement behavior
- Emperor clothes immunity
- Defense bypass behavior
- Blunt knife amount range [1, 100]
- Great sword critical hit probability
- Golden cassock defense removal
- Thorn armor double damage
- Handcuff lock effect

### Integration Tests
- Full purchase flow with database
- Rob game integration with all item effects
- Defense priority order
- Concurrent purchase handling
